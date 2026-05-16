package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/metrics"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/cchunt"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/conventional"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice"
	"github.com/MattCheramie/GopherTrunk/internal/voice/composer"
	"github.com/MattCheramie/GopherTrunk/internal/voice/player"
	"github.com/MattCheramie/GopherTrunk/internal/voice/toneout"
)

// parseGain converts a config.DeviceConfig.Gain value to a tenths-
// of-dB integer suitable for sdr.Device.SetGain. Accepts:
//
//	"auto" / "AUTO" / ""        →  -1 (automatic)
//	"49.6" or "49,6"            →  496
//	"496"                       →  496
//
// Returns ok=false on any other shape so the caller can warn and
// move on without crashing the daemon.
func parseGain(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "auto") {
		return -1, true
	}
	// Accept "49.6" by multiplying. Comma is tolerated for non-US
	// locales pasted out of GUI tools.
	if strings.ContainsAny(s, ".,") {
		s = strings.ReplaceAll(s, ",", ".")
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, false
		}
		return int(f*10 + 0.5), true
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

// Daemon owns the lifecycle of every long-lived component the
// gophertrunk binary runs: the events bus, SDR pool, trunking engine,
// per-call recorder, SQLite call log + retention sweeper, Prometheus
// collector, and HTTP/gRPC API servers.
//
// Components are constructed in dependency order in NewDaemon, started
// by Run, and torn down by Close in reverse order. Run blocks until
// ctx cancels; Close is idempotent.
type Daemon struct {
	cfg      config.Config
	cfgPath  string
	log      *slog.Logger
	writer   *config.Writer // optional; nil when daemon ran without -config

	bus        *events.Bus
	pool       *sdr.Pool
	talkgroups *trunking.TalkgroupDB
	systems    []trunking.System
	engine     *trunking.Engine
	voicePool  *trunking.VoicePool
	recorder   *voice.Recorder
	composer   *composer.Composer
	player     *player.Player
	toneout    *toneout.Detector
	audioPub   *api.AudioPublisher
	db         *storage.DB
	callLog    *storage.CallLog
	retention  *storage.Retention
	ccCache    *trunking.Cache
	cchuntSup  *cchunt.Supervisor
	ccDecoder  *ccdecoder.Decoder
	convScan   *conventional.Scanner
	metrics    *metrics.Metrics
	httpAPI    *api.Server
	grpcAPI    *api.GRPCServer

	// startupWarnings collects non-fatal observations from
	// NewDaemon / preflight (missing talkgroup CSV, SDR enumeration
	// empty, etc.) so the launcher can render them above its menu
	// and the TUI can pin them as a one-shot dashboard banner.
	startupWarnings []string

	// ready is closed by Run once all spawned essential components
	// have started. Consumers use it to gate the launcher prompt so
	// it never lands against a half-dead daemon.
	ready     chan struct{}
	readyOnce sync.Once

	// fatalErr is set by an essential component that fails after
	// Run has begun spawning goroutines. Run's blocking select picks
	// it up and unwinds the daemon. Guarded by mu.
	mu       sync.Mutex
	fatalErr error
	cancel   context.CancelFunc

	wg        sync.WaitGroup
	closeOnce sync.Once
}

// ConfigPath returns the absolute path of the config.yaml backing
// the daemon, or "" when the daemon was started without -config.
// The settings PATCH endpoint reads this to decide whether
// edit-and-write is supported.
func (d *Daemon) ConfigPath() string {
	if d.writer != nil {
		return d.writer.Path()
	}
	return d.cfgPath
}

// Writer returns the live config.yaml writer the daemon installed,
// or nil when no config file backs the daemon. Used by NewDaemon to
// expose the writer through ServerOptions.ConfigWriter.
func (d *Daemon) Writer() *config.Writer { return d.writer }

// Ready returns a channel that closes once every essential component
// has started successfully. Used by the launcher to wait for the HTTP
// API to bind before prompting the operator.
func (d *Daemon) Ready() <-chan struct{} {
	if d.ready == nil {
		// Conservative: a Daemon constructed by code that pre-dates
		// the Ready surface should still satisfy <-d.Ready() with
		// an already-closed channel.
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return d.ready
}

// StartupWarnings returns the non-fatal warnings collected during
// NewDaemon construction. Callers should treat the slice as read-only.
func (d *Daemon) StartupWarnings() []string {
	return append([]string(nil), d.startupWarnings...)
}

// addWarning records a non-fatal startup observation. Threaded into
// the launcher + TUI dashboard so silent degradations get an audible
// surface.
func (d *Daemon) addWarning(msg string) {
	d.startupWarnings = append(d.startupWarnings, msg)
}

// NewDaemon constructs the daemon from the loaded config. Components
// that are disabled by config (no API addr, no storage path, etc.) are
// simply not constructed and the daemon proceeds with the rest.
//
// version is stamped into Prometheus build_info and the API
// /api/v1/version response.
//
// Use NewDaemonWithPath to install a live config.yaml writer for the
// PATCH /api/v1/settings endpoint.
func NewDaemon(cfg config.Config, version string, log *slog.Logger) (*Daemon, error) {
	return NewDaemonWithPath(cfg, "", version, log)
}

// NewDaemonWithPath is the constructor used by main when a config
// file backs the daemon: the path is installed as a config.Writer
// so PATCH /api/v1/settings can re-write the file in place.
func NewDaemonWithPath(cfg config.Config, cfgPath string, version string, log *slog.Logger) (*Daemon, error) {
	if log == nil {
		return nil, errors.New("daemon: logger is required")
	}

	d := &Daemon{
		cfg:     cfg,
		cfgPath: cfgPath,
		log:     log,
		bus:     events.NewBus(64),
		ready:   make(chan struct{}),
	}
	if cfgPath != "" {
		w, err := config.NewWriter(cfgPath)
		if err != nil {
			return nil, fmt.Errorf("daemon: config writer: %w", err)
		}
		d.writer = w
	}

	// Talkgroup DB — populated below from per-system CSVs.
	d.talkgroups = trunking.NewTalkgroupDB()

	// Systems pulled from config; talkgroup CSVs loaded eagerly so the
	// API can serve them immediately.
	for _, sys := range cfg.Trunking.Systems {
		proto, err := trunking.ParseProtocol(sys.Protocol)
		if err != nil {
			return nil, fmt.Errorf("daemon: %w", err)
		}
		s := trunking.System{
			Name:                   sys.Name,
			Protocol:               proto,
			ControlChannels:        sys.ControlChannels,
			TETRAColourCode:        sys.TETRAColourCode,
			TETRAChannel:           sys.TETRAChannel,
			TETRAChannelCoding:     sys.TETRAChannelCoding,
			TETRAClockMode:         sys.TETRAClockMode,
			LTRFCSMode:             sys.LTRFCSMode,
			LTRManchesterMode:      sys.LTRManchesterMode,
			P25Phase2TrellisMode:   sys.P25Phase2TrellisMode,
			P25Phase2RSMode:        sys.P25Phase2RSMode,
			P25Phase2ScramblerMode: sys.P25Phase2ScramblerMode,
			P25Phase2ClockMode:     sys.P25Phase2ClockMode,
			NXDNViterbiMode:        sys.NXDNViterbiMode,
			EDACSBCHMode:           sys.EDACSBCHMode,
			MPT1327BCHMode:         sys.MPT1327BCHMode,
			MPT1327CWSCTolerance:   sys.MPT1327CWSCTolerance,
			MotorolaBCHMode:        sys.MotorolaBCHMode,
			DStarFECMode:           sys.DStarFECMode,
		}
		if err := s.Validate(); err != nil {
			return nil, fmt.Errorf("daemon: system %q: %w", sys.Name, err)
		}
		d.systems = append(d.systems, s)
		if sys.TalkgroupFile != "" {
			n, err := d.talkgroups.LoadCSVFile(sys.TalkgroupFile)
			if err != nil {
				log.Warn("daemon: talkgroup load failed",
					"system", sys.Name, "file", sys.TalkgroupFile, "err", err)
				d.addWarning(fmt.Sprintf(
					"talkgroup_file %q for system %q failed to load (%v) — calls on this system will have no alpha tags",
					sys.TalkgroupFile, sys.Name, err))
			} else {
				log.Info("daemon: talkgroups loaded", "system", sys.Name, "count", n)
			}
		}
	}

	// SDR pool — best-effort. Components that need a Voice device
	// fall through gracefully when the pool is empty.
	if len(cfg.SDR.Devices) > 0 {
		d.pool = sdr.NewPool(log)
		d.pool.SetBus(d.bus)
		var hints []sdr.Hint
		for _, dev := range cfg.SDR.Devices {
			h := sdr.Hint{
				Serial:  dev.Serial,
				Role:    sdr.ParseRole(dev.Role),
				PPM:     dev.PPM,
				BiasTee: dev.BiasTee,
			}
			if dev.Gain != "" {
				gain, ok := parseGain(dev.Gain)
				if !ok {
					log.Warn("daemon: ignoring unparseable gain",
						"serial", dev.Serial, "gain", dev.Gain)
				} else {
					h = h.WithGain(gain)
				}
			}
			hints = append(hints, h)
		}
		if err := d.pool.Open(hints); err != nil {
			log.Warn("daemon: SDR pool open failed", "err", err)
			d.addWarning(fmt.Sprintf(
				"SDR pool failed to open (%v) — no radios will demodulate; check device permissions / cabling / kernel modules",
				err))
			d.pool = nil
		}
	} else if len(cfg.Trunking.Systems) > 0 {
		// Trunked systems configured but no SDR devices listed —
		// the daemon will look healthy from a logs-only vantage but
		// can't actually decode anything.
		d.addWarning("trunking.systems configured but sdr.devices is empty — daemon has nothing to demodulate; add at least one device")
	}

	// Voice device list from the pool; empty when no SDRs.
	d.voicePool = trunking.NewVoicePool(d.collectVoiceDevices())

	engine, err := trunking.NewEngine(trunking.EngineOptions{
		Bus:        d.bus,
		Log:        log,
		VoicePool:  d.voicePool,
		Talkgroups: d.talkgroups,
		ScanMode:   trunking.ParseScanMode(cfg.Scanner.ScanMode),
	})
	if err != nil {
		return nil, fmt.Errorf("daemon: engine: %w", err)
	}
	d.engine = engine

	// Recorder is optional; needs a target directory.
	if cfg.Recordings.Dir != "" {
		rec, err := voice.NewRecorder(voice.RecorderOptions{
			Bus:        d.bus,
			Log:        log,
			OutDir:     cfg.Recordings.Dir,
			SampleRate: cfg.Recordings.SampleRate,
			WriteRaw:   cfg.Recordings.WriteRaw,
		})
		if err != nil {
			return nil, fmt.Errorf("daemon: recorder: %w", err)
		}
		d.recorder = rec
	}

	// Tone-out detector — optional. Built before the composer so it can
	// share the composer's PCM sink via fanoutSink.
	if len(cfg.ToneOut.Profiles) > 0 {
		profiles, err := toneProfilesFromConfig(cfg.ToneOut.Profiles)
		if err != nil {
			return nil, fmt.Errorf("daemon: tone-out: %w", err)
		}
		det, err := toneout.New(toneout.Options{
			Bus:        d.bus,
			Profiles:   profiles,
			SampleRate: cfg.Recordings.SampleRate,
			Log:        log,
		})
		if err != nil {
			return nil, fmt.Errorf("daemon: tone-out: %w", err)
		}
		d.toneout = det
	}

	// Live audio player — optional. Independent of the composer; if
	// enabled, we feed it as one fan-out arm alongside the recorder.
	// Defaults to disabled so headless servers stay quiet.
	playerSampleRate := cfg.Audio.SampleRate
	if playerSampleRate == 0 {
		playerSampleRate = cfg.Recordings.SampleRate
	}
	pl, err := player.New(player.Config{
		Enabled:    cfg.Audio.Enabled,
		Device:     cfg.Audio.Device,
		SampleRate: playerSampleRate,
		BufferMs:   cfg.Audio.BufferMs,
		Volume:     cfg.Audio.Volume,
		Muted:      cfg.Audio.Muted,
	}, log)
	if err != nil {
		return nil, fmt.Errorf("daemon: player: %w", err)
	}
	d.player = pl

	// Audio publisher — fans decoded PCM out to gRPC StreamAudio
	// subscribers. Constructed unconditionally so the gRPC server
	// can register the RPC; if no composer is wired downstream
	// the publisher just sits idle (no producers, no consumers
	// matter). Run is spawned by Daemon.Run.
	audioPub, err := api.NewAudioPublisher(d.bus, log)
	if err != nil {
		return nil, fmt.Errorf("daemon: audio publisher: %w", err)
	}
	d.audioPub = audioPub

	// Composer is wired when we have a Voice device pool to source IQ
	// from + a recorder to feed PCM into. Without an SDR pool there's
	// nothing to demod, and without a recorder PCM has no destination.
	if d.pool != nil && d.recorder != nil {
		// Fan PCM to recorder + tone-out (if configured) + live player + gRPC publisher.
		sinks := []composer.PCMSink{d.recorder}
		if d.toneout != nil {
			sinks = append(sinks, d.toneout)
		}
		if d.player != nil {
			sinks = append(sinks, playerSink{d.player})
		}
		sinks = append(sinks, d.audioPub)
		var sink composer.PCMSink = d.recorder
		if len(sinks) > 1 {
			sink = fanoutSink(sinks)
		}
		comp, err := composer.New(composer.Options{
			Bus:           d.bus,
			Devices:       &poolDevices{pool: d.pool},
			Sink:          sink,
			Engine:        d.engine,
			Log:           log,
			IQSampleRate:  cfg.SDR.SampleRate,
			PCMSampleRate: cfg.Recordings.SampleRate,
			Equalizer: composer.EqualizerConfig{
				Enabled:  cfg.Recordings.Equalizer.Enabled,
				Taps:     cfg.Recordings.Equalizer.Taps,
				StepSize: cfg.Recordings.Equalizer.StepSize,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("daemon: composer: %w", err)
		}
		d.composer = comp
	}

	// CC cache — JSON file used by the hunter to bias retunes toward
	// the last-known good control channel per system. Optional; nil
	// disables persistence (hunts still work, just without the cache
	// bias).
	if cfg.Storage.CCCacheFile != "" {
		cache, err := trunking.OpenCache(cfg.Storage.CCCacheFile)
		if err != nil {
			return nil, fmt.Errorf("daemon: cc cache: %w", err)
		}
		d.ccCache = cache
	}

	// CC hunter supervisor — opt-in via scanner.cc_hunt.enabled OR
	// by default when at least one trunked system is configured.
	// Constructs an IQ → CC decoder connector alongside (below) so
	// the supervisor's retunes drive a live demod pipeline. P25
	// Phase 1 and YSF have wired pipelines today; other protocols
	// stay in `state=hunting` until their per-protocol
	// ControlChannel.Process(stream, baseIdx) adapters ship — see
	// internal/scanner/ccdecoder/pipelines.go for the factory map.
	if d.pool != nil && len(d.systems) > 0 {
		cchEnabled := cfg.Scanner.CCHunt.Enabled || cfg.Scanner.CCHunt == (config.CCHuntConfig{})
		if cchEnabled {
			controlEntry := d.pool.FirstByRole(sdr.RoleControl)
			if controlEntry != nil {
				sup, err := cchunt.New(cchunt.Options{
					Bus:            d.bus,
					Log:            log,
					Tuner:          controlEntry.Device,
					Cache:          d.ccCache,
					Systems:        d.systems,
					Dwell:          msToDuration(cfg.Scanner.CCHunt.DwellMs, 3*time.Second),
					InitialBackoff: msToDuration(cfg.Scanner.CCHunt.BackoffMs, 5*time.Second),
					MaxBackoff:     msToDuration(cfg.Scanner.CCHunt.MaxBackoffMs, 60*time.Second),
				})
				if err != nil {
					return nil, fmt.Errorf("daemon: cchunt: %w", err)
				}
				d.cchuntSup = sup

				// IQ → CC decoder connector. Owns one StreamIQ
				// loop on the control SDR, subscribes to
				// KindHuntProgress so it learns which system /
				// frequency the supervisor is currently attempting,
				// and pumps IQ through the matching per-protocol
				// pipeline so the CC state machine publishes
				// cc.locked / grant events on the bus. Only P25
				// Phase 1 and YSF have wired pipelines today —
				// other protocols log a "no factory" debug message
				// when their HuntProgress lands and the supervisor
				// stays in `state=hunting`. See README for the
				// per-protocol Process(stream, baseIdx) adapter
				// follow-ups.
				dec, err := ccdecoder.New(ccdecoder.Options{
					Bus:          d.bus,
					Log:          log,
					Tuner:        controlEntry.Device,
					IQ:           controlEntry.Device,
					Systems:      d.systems,
					SampleRateHz: float64(cfg.SDR.SampleRate),
				})
				if err != nil {
					return nil, fmt.Errorf("daemon: ccdecoder: %w", err)
				}
				d.ccDecoder = dec
			} else {
				log.Warn("daemon: scanner.cc_hunt enabled but no control SDR in pool; skipping")
			}
		}
	}

	// Conventional FM scanner — constructed when:
	//   (a) scanner.conventional channels are configured, OR
	//   (b) scanner.manual_tune_enabled is explicitly true, OR
	//   (c) auto-detect finds a spare Voice SDR (>= 2 Voice SDRs
	//       in the pool) AND manual_tune_disabled is not set.
	//
	// Requires its own dedicated Voice SDR (carved out of the pool's
	// voice fleet) and a recorder to land WAVs into. The scanner
	// publishes synthetic CallStart/CallEnd events through
	// Engine.HandleSyntheticCall, so the recorder, call log, and
	// /api/v1/calls/active surfaces light up like any other call.
	var voiceEntries []*sdr.PoolEntry
	if d.pool != nil {
		voiceEntries = d.pool.AllByRole(sdr.RoleVoice)
	}
	hasConventional := len(cfg.Scanner.Conventional) > 0
	autoEnable := !cfg.Scanner.ManualTuneDisabled && len(voiceEntries) >= 2
	convWanted := hasConventional || cfg.Scanner.ManualTuneEnabled || autoEnable
	if convWanted && autoEnable && !cfg.Scanner.ManualTuneEnabled && !hasConventional {
		log.Info("daemon: scanner: auto-enabling manual tune (spare Voice SDR detected)",
			"voice_sdrs", len(voiceEntries))
	}
	if d.pool != nil && d.recorder != nil && d.engine != nil && convWanted {
		if len(voiceEntries) == 0 {
			log.Warn("daemon: scanner.conventional / manual_tune_enabled configured but no Voice SDRs in the pool; skipping")
		} else {
			// Use the LAST voice SDR so the trunking engine still
			// has the first N-1 voice radios for normal grants.
			convEntry := voiceEntries[len(voiceEntries)-1]
			channels := make([]conventional.Channel, 0, len(cfg.Scanner.Conventional))
			for _, ch := range cfg.Scanner.Conventional {
				channels = append(channels, conventional.Channel{
					Label:       ch.Label,
					FrequencyHz: ch.FrequencyHz,
					Mode:        ch.Mode,
					SquelchDbFS: ch.SquelchDbFS,
					Hangtime:    msToDuration(ch.HangtimeMs, 1500*time.Millisecond),
					Priority:    ch.Priority,
					Tone: conventional.ToneConfig{
						Mode:    ch.Tone.Mode,
						CTCSSHz: ch.Tone.CTCSSHz,
						DCSCode: ch.Tone.DCSCode,
					},
				})
			}
			var convRec conventional.Recorder = d.recorder
			if d.player != nil {
				convRec = convFanoutRecorder{d.recorder, playerSink{d.player}}
			}
			cs, err := conventional.New(conventional.Options{
				Log:          log,
				Tuner:        convEntry.Device,
				IQ:           convEntry.Device,
				Engine:       d.engine,
				Recorder:     convRec,
				DeviceSerial: convEntry.Info.Serial,
				SystemName:   "scanner",
				Channels:     channels,
				SampleRateHz: float64(cfg.SDR.SampleRate),
			})
			if err != nil {
				return nil, fmt.Errorf("daemon: conv scanner: %w", err)
			}
			d.convScan = cs
		}
	}

	// Storage / call log / retention — optional.
	if cfg.Storage.Path != "" {
		db, err := storage.Open(cfg.Storage.Path)
		if err != nil {
			return nil, fmt.Errorf("daemon: storage: %w", err)
		}
		d.db = db
		cl, err := storage.NewCallLog(db, d.bus, log)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("daemon: call log: %w", err)
		}
		d.callLog = cl

		if cfg.Retention.CallLogDays > 0 || cfg.Retention.FilesDays > 0 {
			interval, err := retentionInterval(cfg.Retention.Interval)
			if err != nil {
				return nil, fmt.Errorf("daemon: retention.interval: %w", err)
			}
			r, err := storage.NewRetention(storage.RetentionOptions{
				DB:            db,
				FilesRoot:     cfg.Recordings.Dir,
				CallRowMaxAge: time.Duration(cfg.Retention.CallLogDays) * 24 * time.Hour,
				FilesMaxAge:   time.Duration(cfg.Retention.FilesDays) * 24 * time.Hour,
				Interval:      interval,
				Log:           log,
			})
			if err != nil {
				return nil, fmt.Errorf("daemon: retention: %w", err)
			}
			d.retention = r
		}
	}

	// Metrics — optional. Subscribes to the bus at construction time.
	if cfg.Metrics.Enabled {
		m, err := metrics.New(d.bus, version)
		if err != nil {
			return nil, fmt.Errorf("daemon: metrics: %w", err)
		}
		d.metrics = m
	}

	// HTTP API — optional.
	if cfg.API.HTTPAddr != "" {
		authMode, ok := api.ParseAuthMode(cfg.API.Auth.Mode)
		if !ok {
			return nil, fmt.Errorf("daemon: api.auth.mode: unrecognised value %q (expected auto / required / disabled)", cfg.API.Auth.Mode)
		}
		opts := api.ServerOptions{
			Addr:           cfg.API.HTTPAddr,
			Bus:            d.bus,
			Engine:         d.engine,
			Mutator:        d.engine,
			Talkgroups:     d.talkgroups,
			Systems:        d.systems,
			Log:            log,
			Version:        version,
			AllowMutations: cfg.API.AllowMutations,
			Auth: api.AuthConfig{
				Mode:            authMode,
				Token:           cfg.API.Auth.Token,
				TokenFile:       cfg.API.Auth.TokenFile,
				TrustedNetworks: cfg.API.Auth.TrustedNetworks,
			},
			CORS: api.CORSConfig{
				AllowedOrigins: cfg.API.CORS.AllowedOrigins,
			},
			AudioPublisher: d.audioPub,
			TLSCert:        cfg.API.TLSCert,
			TLSKey:         cfg.API.TLSKey,
		}
		if d.db != nil {
			opts.History = api.HistoryFromStorage(d.db)
		}
		if d.metrics != nil {
			opts.MetricsHandler = d.metrics.Handler()
		}
		if d.retention != nil {
			opts.Retention = d.retention
		}
		if d.toneout != nil {
			opts.Tones = d.toneout
		}
		if d.pool != nil {
			opts.Devices = d.pool
		}
		opts.Scanner = scannerCockpit{
			cchunt:     d.cchuntSup,
			conv:       d.convScan,
			engine:     d.engine,
			talkgroups: d.talkgroups,
		}
		if d.player != nil || d.recorder != nil {
			opts.Audio = audioCockpit{player: d.player, recorder: d.recorder}
		}
		cfgCopy := cfg
		opts.Runtime = &runtimeSnapshot{
			cfg:     &cfgCopy,
			version: version,
			metrics: cfg.Metrics.Enabled,
			daemon:  d,
		}
		// Live settings editing — only enabled when the daemon was
		// started with a -config (otherwise there's no file to write
		// back to).
		if d.writer != nil {
			opts.ConfigWriter = d.writer
			opts.SettingsApplier = newDaemonSettingsApplier(d, version)
		}
		srv, err := api.NewServer(opts)
		if err != nil {
			return nil, fmt.Errorf("daemon: http api: %w", err)
		}
		d.httpAPI = srv
	}

	// gRPC — optional.
	if cfg.API.GRPCAddr != "" {
		gsrv, err := api.NewGRPCServer(api.GRPCServerOptions{
			Addr:       cfg.API.GRPCAddr,
			Systems:    d.systems,
			Talkgroups: d.talkgroups,
			Engine:     d.engine,
			Audio:      d.audioPub,
			Log:        log,
			TLSCert:    cfg.API.TLSCert,
			TLSKey:     cfg.API.TLSKey,
		})
		if err != nil {
			return nil, fmt.Errorf("daemon: grpc api: %w", err)
		}
		d.grpcAPI = gsrv
	}

	return d, nil
}

// Run starts every long-lived goroutine and blocks until ctx cancels
// (or an essential component fails). Each component returns its own
// error; non-essential errors land on the log, essential errors
// cancel the daemon's internal context so the launcher / TUI doesn't
// keep running against a half-dead daemon.
//
// Run also closes d.Ready() once the spawned components have settled.
// "Settled" today is conservative — a brief 250 ms timer after every
// spawn call has fired — but ensures the HTTP listener bound (so the
// launcher's TUI/web flow can reach it) before the prompt appears.
func (d *Daemon) Run(ctx context.Context) error {
	d.log.Info("gophertrunk starting",
		"http_addr", d.cfg.API.HTTPAddr,
		"grpc_addr", d.cfg.API.GRPCAddr,
		"systems", len(d.systems),
		"voice_devices", len(d.voicePool.Devices()),
	)

	// Wrap ctx so an essential component error can cancel siblings.
	runCtx, cancel := context.WithCancel(ctx)
	d.mu.Lock()
	d.cancel = cancel
	d.mu.Unlock()
	defer cancel()

	d.spawn(runCtx, "engine", true, func(ctx context.Context) error {
		return d.engine.Run(ctx)
	})

	if d.callLog != nil {
		d.spawn(runCtx, "calllog", false, func(ctx context.Context) error {
			return d.callLog.Run(ctx)
		})
	}
	if d.retention != nil {
		d.spawn(runCtx, "retention", false, func(ctx context.Context) error {
			return d.retention.Run(ctx)
		})
	}
	if d.recorder != nil {
		d.spawn(runCtx, "recorder", false, func(ctx context.Context) error {
			return d.recorder.Run(ctx)
		})
	}
	if d.composer != nil {
		d.spawn(runCtx, "composer", false, func(ctx context.Context) error {
			return d.composer.Run(ctx)
		})
	}
	if d.audioPub != nil {
		d.spawn(runCtx, "audio-publisher", false, func(ctx context.Context) error {
			return d.audioPub.Run(ctx)
		})
	}
	if d.metrics != nil {
		d.spawn(runCtx, "metrics", false, func(ctx context.Context) error {
			return d.metrics.Run(ctx)
		})
	}
	if d.cchuntSup != nil {
		d.spawn(runCtx, "cchunt", false, func(ctx context.Context) error {
			return d.cchuntSup.Run(ctx)
		})
	}
	if d.ccDecoder != nil {
		d.spawn(runCtx, "ccdecoder", false, func(ctx context.Context) error {
			return d.ccDecoder.Run(ctx)
		})
	}
	if d.convScan != nil {
		d.spawn(runCtx, "conv-scanner", false, func(ctx context.Context) error {
			return d.convScan.Run(ctx)
		})
	}
	if d.httpAPI != nil {
		// HTTP is essential — the launcher's TUI / web paths need
		// it bound, and bind failures (port in use, permission
		// denied) used to be demoted to a warning while the daemon
		// kept running. Now they abort cleanly.
		d.spawn(runCtx, "http", true, func(ctx context.Context) error {
			return d.httpAPI.Run(ctx)
		})
	}
	if d.grpcAPI != nil {
		d.spawn(runCtx, "grpc", true, func(ctx context.Context) error {
			return d.grpcAPI.Run(ctx)
		})
	}

	// Conservative readiness: give every spawn a brief grace window
	// so the HTTP listener has time to bind. Components without an
	// explicit "ready" hook share this same gate.
	go d.markReadyAfter(runCtx, 250*time.Millisecond)

	<-runCtx.Done()
	d.log.Info("gophertrunk shutdown initiated")
	d.Close()
	d.wg.Wait()
	d.log.Info("gophertrunk shutdown complete")
	if err := d.takeFatal(); err != nil {
		return err
	}
	return ctx.Err()
}

// markReadyAfter waits the supplied grace window and then closes the
// Ready channel. Cancellation aborts the wait but still closes the
// channel so blocked consumers don't deadlock.
func (d *Daemon) markReadyAfter(ctx context.Context, grace time.Duration) {
	t := time.NewTimer(grace)
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
	}
	d.readyOnce.Do(func() { close(d.ready) })
}

// recordFatal stores an essential-component error (first wins) and
// cancels the daemon's run context so siblings unwind. Safe to call
// from any goroutine.
func (d *Daemon) recordFatal(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.fatalErr == nil {
		d.fatalErr = err
	}
	if d.cancel != nil {
		d.cancel()
	}
}

// takeFatal returns the captured essential-component error (if any).
func (d *Daemon) takeFatal() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.fatalErr
}

// Close releases every component. Idempotent and safe to call
// concurrently with Run.
func (d *Daemon) Close() {
	d.closeOnce.Do(func() {
		if d.httpAPI != nil {
			_ = d.httpAPI.Close()
		}
		if d.grpcAPI != nil {
			d.grpcAPI.Stop()
		}
		if d.ccDecoder != nil {
			_ = d.ccDecoder.Close()
		}
		if d.composer != nil {
			_ = d.composer.Close()
		}
		if d.audioPub != nil {
			_ = d.audioPub.Close()
		}
		if d.player != nil {
			_ = d.player.Close()
		}
		if d.recorder != nil {
			_ = d.recorder.Close()
		}
		if d.callLog != nil {
			_ = d.callLog.Close()
		}
		if d.metrics != nil {
			_ = d.metrics.Close()
		}
		if d.engine != nil {
			d.engine.Close()
		}
		if d.db != nil {
			_ = d.db.Close()
		}
		if d.pool != nil {
			_ = d.pool.Close()
		}
		d.bus.Close()
	})
}

// HTTPListenAddr returns the resolved address of the HTTP API
// (helpful for tests using an ephemeral ":0" port). Returns "" if no
// API server is configured.
func (d *Daemon) HTTPListenAddr() string {
	if d.httpAPI == nil {
		return ""
	}
	return d.cfg.API.HTTPAddr
}

// Bus returns the daemon's events bus. Tests use it to inject
// synthetic events.
func (d *Daemon) Bus() *events.Bus { return d.bus }

// spawn runs fn in a goroutine. essential=true means a non-context
// error from fn aborts the whole daemon (the launcher / TUI rely on
// these components — silently demoting their failures to log warnings
// produces a half-dead daemon that frustrates the operator).
// essential=false retains the legacy behaviour: warn-and-continue.
func (d *Daemon) spawn(ctx context.Context, name string, essential bool, fn func(context.Context) error) {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		err := fn(ctx)
		if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		if essential {
			d.log.Error("daemon: essential component failed",
				"component", name, "err", err)
			d.recordFatal(fmt.Errorf("%s: %w", name, err))
			return
		}
		d.log.Warn("daemon: component exited with error", "component", name, "err", err)
	}()
}

func (d *Daemon) collectVoiceDevices() []*trunking.VoiceDevice {
	if d.pool == nil {
		return nil
	}
	var voices []*trunking.VoiceDevice
	for _, e := range d.pool.AllByRole(sdr.RoleVoice) {
		voices = append(voices, &trunking.VoiceDevice{
			Tuner:  e.Device,
			Serial: e.Info.Serial,
		})
	}
	return voices
}

func retentionInterval(s string) (time.Duration, error) {
	if s == "" {
		return time.Hour, nil
	}
	return time.ParseDuration(s)
}

// msToDuration converts a config milliseconds field to a Duration,
// falling back to fallback when the field is zero or negative.
func msToDuration(ms int, fallback time.Duration) time.Duration {
	if ms <= 0 {
		return fallback
	}
	return time.Duration(ms) * time.Millisecond
}

// fanoutSink writes the same PCM frame to several composer.PCMSink
// downstreams. Used to feed both the recorder and the tone-out
// detector from one composer.
type fanoutSink []composer.PCMSink

func (f fanoutSink) WritePCM(serial string, samples []int16) error {
	for _, s := range f {
		_ = s.WritePCM(serial, samples)
	}
	return nil
}

// toneProfilesFromConfig converts the YAML config shape into the
// internal toneout.Profile shape, parsing duration strings.
func toneProfilesFromConfig(in []config.ToneProfileConfig) ([]toneout.Profile, error) {
	out := make([]toneout.Profile, 0, len(in))
	for _, p := range in {
		tones := make([]toneout.Tone, 0, len(p.Tones))
		for ti, t := range p.Tones {
			minD, err := time.ParseDuration(t.MinDuration)
			if err != nil {
				return nil, fmt.Errorf("profile %q tone[%d] min_duration: %w", p.Name, ti, err)
			}
			maxD := time.Duration(0)
			if t.MaxDuration != "" {
				maxD, err = time.ParseDuration(t.MaxDuration)
				if err != nil {
					return nil, fmt.Errorf("profile %q tone[%d] max_duration: %w", p.Name, ti, err)
				}
			}
			tones = append(tones, toneout.Tone{
				FrequencyHz: t.FrequencyHz,
				MinDuration: minD,
				MaxDuration: maxD,
			})
		}
		var maxGap, cooldown time.Duration
		if p.MaxGap != "" {
			d, err := time.ParseDuration(p.MaxGap)
			if err != nil {
				return nil, fmt.Errorf("profile %q max_gap: %w", p.Name, err)
			}
			maxGap = d
		}
		if p.Cooldown != "" {
			d, err := time.ParseDuration(p.Cooldown)
			if err != nil {
				return nil, fmt.Errorf("profile %q cooldown: %w", p.Name, err)
			}
			cooldown = d
		}
		out = append(out, toneout.Profile{
			Name:               p.Name,
			AlphaTag:           p.AlphaTag,
			Tones:              tones,
			ToleranceHz:        p.ToleranceHz,
			MagnitudeThreshold: p.MagnitudeThreshold,
			MaxGap:             maxGap,
			Cooldown:           cooldown,
			System:             p.System,
			GroupID:            p.GroupID,
		})
	}
	return out, nil
}

// playerSink adapts *player.Player to composer.PCMSink. The Player's
// WritePCM signature is identical so the adapter is a zero-cost shim
// that exists only to satisfy Go's nominal-typing for the fan-out
// slice element type.
type playerSink struct{ p *player.Player }

func (s playerSink) WritePCM(serial string, samples []int16) error {
	return s.p.WritePCM(serial, samples)
}

// convFanoutRecorder lets the conventional scanner drive both the
// WAV recorder and the live player. The conventional.Recorder
// interface only requires WritePCM, matching what both downstreams
// implement.
type convFanoutRecorder []conventional.Recorder

func (f convFanoutRecorder) WritePCM(serial string, samples []int16) error {
	for _, r := range f {
		_ = r.WritePCM(serial, samples)
	}
	return nil
}

// audioCockpit aggregates the live-audio Player and the WAV
// Recorder into the api.AudioController interface so GET / PATCH
// /api/v1/audio can drive volume, mute, and recording from one
// place. A nil Player (audio.enabled=false in config) collapses
// the playback side to no-ops while the recorder gate still works.
type audioCockpit struct {
	player   *player.Player
	recorder *voice.Recorder
}

func (a audioCockpit) Volume() float32 {
	if a.player == nil {
		return 0
	}
	return a.player.Volume()
}

func (a audioCockpit) SetVolume(v float32) {
	if a.player == nil {
		return
	}
	a.player.SetVolume(v)
}

func (a audioCockpit) Muted() bool {
	if a.player == nil {
		return true
	}
	return a.player.Muted()
}

func (a audioCockpit) SetMuted(m bool) {
	if a.player == nil {
		return
	}
	a.player.SetMuted(m)
}

func (a audioCockpit) RecordingEnabled() bool {
	if a.recorder == nil {
		return false
	}
	return a.recorder.RecordingEnabled()
}

func (a audioCockpit) SetRecordingEnabled(enabled bool) {
	if a.recorder == nil {
		return
	}
	a.recorder.SetRecordingEnabled(enabled)
}

func (a audioCockpit) DropsTotal() uint64 {
	if a.player == nil {
		return 0
	}
	return a.player.Stats().DropsTotal
}

func (a audioCockpit) SampleRate() uint32 {
	if a.player == nil {
		return 0
	}
	return a.player.Stats().SampleRate
}

func (a audioCockpit) BackendEnabled() bool {
	if a.player == nil {
		return false
	}
	return a.player.Stats().Enabled
}

// poolDevices adapts *sdr.Pool to composer.Devices. The composer only
// needs StreamIQ; sdr.Device satisfies that subset directly.
type poolDevices struct{ pool *sdr.Pool }

func (p *poolDevices) FindBySerial(serial string) composer.IQSource {
	if p.pool == nil {
		return nil
	}
	e := p.pool.FindBySerial(serial)
	if e == nil {
		return nil
	}
	return e.Device
}
