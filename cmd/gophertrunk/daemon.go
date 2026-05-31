package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/api/rigctld"
	"github.com/MattCheramie/GopherTrunk/internal/broadcast"
	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
	"github.com/MattCheramie/GopherTrunk/internal/metrics"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/cchunt"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/conventional"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/widebandt2"

	adsbbeast "github.com/MattCheramie/GopherTrunk/internal/radio/adsb/beast"
	adsbppm "github.com/MattCheramie/GopherTrunk/internal/radio/adsb/ppm"
	aisgmsk "github.com/MattCheramie/GopherTrunk/internal/radio/ais/gmsk"
	aprsafsk "github.com/MattCheramie/GopherTrunk/internal/radio/aprs/afsk"
	dscffsk "github.com/MattCheramie/GopherTrunk/internal/radio/dsc/ffsk"
	mdc1200afsk "github.com/MattCheramie/GopherTrunk/internal/radio/mdc1200/afsk"
	pocsagrx "github.com/MattCheramie/GopherTrunk/internal/radio/pager/pocsag/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/baseband"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtltcp"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/wbvoice"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice"
	"github.com/MattCheramie/GopherTrunk/internal/voice/composer"
	"github.com/MattCheramie/GopherTrunk/internal/voice/player"
	"github.com/MattCheramie/GopherTrunk/internal/voice/toneout"
	gtweb "github.com/MattCheramie/GopherTrunk/web"
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

// gainLooksLikeDBMistake reports whether a successfully-parsed gain value
// is almost certainly a dB figure the operator forgot to express in
// tenths-of-dB. GopherTrunk's gain: string is tenths (so "320" = 32 dB),
// but SDRTrunk / OP25 / gqrx all take whole dB — a habit that lands "32"
// (= 3.2 dB, which SetGain then snaps to the bottom of the ladder, leaving
// the radio effectively deaf) in many first-run configs. A bare integer
// (no decimal point) that parses to a positive value at or below 50 tenths
// (5.0 dB) is the tell: real manual gains run ~100-496 tenths and the
// shipped examples are all three digits, so this never fires on a correct
// config. Decimal forms like "32.0" are already interpreted as whole dB by
// parseGain, so they're taken at face value and skipped.
func gainLooksLikeDBMistake(raw string, tenthDB int) bool {
	raw = strings.TrimSpace(raw)
	if strings.ContainsAny(raw, ".,") {
		return false
	}
	return tenthDB > 0 && tenthDB <= 50
}

// warnGainUnits emits a one-line, actionable WARN when a parsed gain looks
// like a dB figure that should have been tenths-of-dB (see
// gainLooksLikeDBMistake). No-op for plausible gains, so it's safe to call
// unconditionally after every successful parseGain.
func warnGainUnits(log *slog.Logger, serial, raw string, tenthDB int) {
	if !gainLooksLikeDBMistake(raw, tenthDB) {
		return
	}
	log.Warn("daemon: gain looks like dB, not tenths-of-dB — radio may be effectively deaf",
		"serial", serial,
		"configured", raw,
		"parsed_db", float64(tenthDB)/10.0,
		"did_you_mean", strconv.Itoa(tenthDB*10),
		"hint", "gain: is in TENTHS of a dB (\"320\" = 32 dB). SDRTrunk/OP25/gqrx users multiply dB by 10. Use 'gain: auto' or run 'gophertrunk sdr list' for the supported ladder.")
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
	// cfgMu guards cfg against concurrent reload via SIGHUP /
	// PATCH /api/v1/settings vs. other goroutines that snapshot
	// config fields (HTTPListenAddr, startup logging, runtime DTO
	// build). Reload takes the write lock for the whole apply
	// phase; readers use Cfg() to take an RLock'd snapshot.
	cfgMu   sync.RWMutex
	cfg     config.Config
	cfgPath string
	log     *slog.Logger
	writer  *config.Writer // optional; nil when daemon ran without -config

	bus          *events.Bus
	pool         *sdr.Pool
	talkgroups   *trunking.TalkgroupDB
	rids         *trunking.RIDDB
	systems      []trunking.System
	engine       *trunking.Engine
	voicePool    *trunking.VoicePool
	affiliations *trunking.AffiliationTracker
	recorder     *voice.Recorder
	broadcast    *broadcast.Manager
	composer     *composer.Composer
	player       *player.Player
	toneout      *toneout.Detector
	audioPub     *api.AudioPublisher
	db           *storage.DB
	callLog      *storage.CallLog
	locationLog  *storage.LocationLog
	bookmarks    *storage.BookmarkStore
	pagerLog     *storage.PagerLog
	aprsLog      *storage.APRSLog
	vesselLog    *storage.VesselLog
	dscLog       *storage.DSCLog
	aircraftLog  *storage.AircraftLog
	mdc1200Log   *storage.MDC1200Log
	messageLog   *gtlog.MessageLog
	retention    *storage.Retention
	ccCache      *trunking.Cache
	cchuntSup    *cchunt.Supervisor
	ccDecoder    *ccdecoder.Decoder
	// ccDecoderOpts is captured at construction so the spawn closure
	// can rebuild the decoder after an IQ-stream death — Decoder.Run
	// closes its bus subscription on exit, so a fresh instance is
	// needed to restart cleanly (issue #345).
	ccDecoderOpts ccdecoder.Options
	// controlSerial + controlSampleRate let the ccdecoder retry loop
	// ask the pool to re-acquire the control SDR by serial after a
	// USB disconnect/re-enumerate before rebuilding the decoder
	// against a fresh Device handle (issue #345).
	controlSerial     string
	controlSampleRate uint32
	convScan          *conventional.Scanner
	widebandT2        []*widebandt2.Engine
	// virtualVoiceTuners holds one VirtualTuner per voice tap on a
	// wideband dongle. Each implements both trunking.Tuner (the voice
	// pool calls SetCenterFreq on bind) and composer.IQSource (the
	// composer reads decimated 48 kHz IQ during the call). Wired up
	// alongside each wideband Engine in NewDaemon; surfaced into the
	// voice pool via collectVoiceDevices and into the composer via
	// poolDevices.virtualMap. See internal/sdr/wbvoice.
	virtualVoiceTuners []*wbvoice.VirtualTuner
	// pocsagReceivers holds one POCSAG receiver per configured
	// paging.pocsag entry. Each subscribes to the iqtap broker
	// for its assigned SDR and publishes pages onto the events
	// bus on KindPagerMessage. See internal/radio/pager/pocsag/
	// receiver. Pinned (not auto-discovered): the receiver tunes
	// the SDR directly to its paging frequency, so the operator
	// has to dedicate a dongle to it. Multi-channel-from-one-SDR
	// is a planned follow-up.
	pocsagReceivers []*pocsagrx.Receiver
	pocsagSpecs     []pocsagSpec // index-aligned with pocsagReceivers
	// aprsReceivers holds one APRS AFSK receiver per configured
	// aprs.channels entry. Each subscribes to the iqtap broker
	// for its assigned SDR and publishes packets onto the events
	// bus on KindAPRSPacket. See internal/radio/aprs/afsk. Same
	// pinned-channel layout as POCSAG: one SDR per APRS frequency.
	aprsReceivers []*aprsafsk.Receiver
	aprsSpecs     []aprsSpec // index-aligned with aprsReceivers
	// aisReceivers holds one AIS GMSK receiver per configured
	// ais.channels entry. Same shape as the APRS receivers above:
	// each subscribes to its assigned SDR's iqtap broker and
	// publishes decoded vessel messages on KindAISMessage.
	aisReceivers []*aisgmsk.Receiver
	aisSpecs     []aisSpec // index-aligned with aisReceivers
	// dscReceivers holds one DSC FFSK receiver per configured
	// dsc.channels entry. Same shape as the AIS receivers above: each
	// subscribes to its assigned SDR's iqtap broker and publishes
	// decoded DSC sequences on KindDSCMessage.
	dscReceivers []*dscffsk.Receiver
	dscSpecs     []dscSpec // index-aligned with dscReceivers
	// mdc1200Receivers holds one MDC1200 FFSK receiver per configured
	// mdc1200.channels entry. Same shape as the APRS receivers above:
	// each subscribes to its assigned SDR's iqtap broker and publishes
	// decoded signaling bursts on KindMDC1200Message.
	mdc1200Receivers []*mdc1200afsk.Receiver
	mdc1200Specs     []mdc1200Spec // index-aligned with mdc1200Receivers
	// adsbBeastClients consume ADS-B Mode-S frames from external
	// dump1090 / readsb upstreams (BEAST binary protocol). Each
	// client decodes frames, pairs CPR halves via an embedded
	// per-ICAO tracker, and publishes KindAircraftReport.
	adsbBeastClients []*adsbbeast.Client
	adsbBeastNames   []string // index-aligned with adsbBeastClients
	// adsbPPMReceivers hold one native 1090 MHz Mode-S PPM receiver per
	// configured adsb.channels entry. Each subscribes to its assigned
	// SDR's iqtap broker and publishes the same KindAircraftReport the
	// BEAST clients do.
	adsbPPMReceivers []*adsbppm.Receiver
	adsbPPMSpecs     []adsbPPMSpec // index-aligned with adsbPPMReceivers
	// iqBrokers holds an iqtap.Broker per pool entry, keyed by serial.
	// Primary consumers (CC decoder, conventional scanner) stream IQ
	// through the broker so secondary observers (live spectrum,
	// future paging / AIS / ADS-B decoders, rtl_tcp server) can
	// Subscribe without disturbing the primary's StreamIQ contract.
	// Foundation for the trunking-adjacent feature work — see
	// internal/sdr/iqtap. Populated after wrapBasebandRecorders so
	// the broker wraps the recorder when baseband recording is on
	// for the same dongle.
	iqBrokers map[string]*iqtap.Broker
	metrics   *metrics.Metrics
	httpAPI   *api.Server
	grpcAPI   *api.GRPCServer
	rigctld   *rigctld.Server

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
	// RID alias DB — populated below from per-system rid_alias_file
	// entries. Empty when no system has the key set; live observations
	// still flow through the affiliation tracker regardless.
	d.rids = trunking.NewRIDDB()

	// Systems pulled from config; talkgroup CSVs loaded eagerly so the
	// API can serve them immediately.
	for _, sys := range cfg.Trunking.Systems {
		proto, err := trunking.ParseProtocol(sys.Protocol)
		if err != nil {
			return nil, fmt.Errorf("daemon: %w", err)
		}
		var p25BandPlan []trunking.P25BandPlanEntry
		if len(sys.P25BandPlan) > 0 {
			p25BandPlan = make([]trunking.P25BandPlanEntry, 0, len(sys.P25BandPlan))
			for _, e := range sys.P25BandPlan {
				p25BandPlan = append(p25BandPlan, trunking.P25BandPlanEntry{
					ChannelID:   e.ChannelID,
					BaseHz:      e.BaseHz,
					SpacingHz:   e.SpacingHz,
					TxOffsetHz:  e.TxOffsetHz,
					BandwidthHz: e.BandwidthHz,
				})
			}
		}
		s := trunking.System{
			Name:                    sys.Name,
			Protocol:                proto,
			ControlChannels:         sys.ControlChannels,
			P25BandPlan:             p25BandPlan,
			TETRAColourCode:         sys.TETRAColourCode,
			TETRAChannel:            sys.TETRAChannel,
			TETRAChannelCoding:      sys.TETRAChannelCoding,
			TETRAClockMode:          sys.TETRAClockMode,
			LTRFCSMode:              sys.LTRFCSMode,
			LTRManchesterMode:       sys.LTRManchesterMode,
			P25Phase1DemodMode:      sys.P25Phase1DemodMode,
			P25Phase2TrellisMode:    sys.P25Phase2TrellisMode,
			P25Phase2RSMode:         sys.P25Phase2RSMode,
			P25Phase2InterleaveMode: sys.P25Phase2InterleaveMode,
			P25Phase2ScramblerMode:  sys.P25Phase2ScramblerMode,
			P25Phase2ClockMode:      sys.P25Phase2ClockMode,
			NXDNViterbiMode:         sys.NXDNViterbiMode,
			NXDNDeviationHz:         sys.NXDNDeviationHz,
			EDACSBCHMode:            sys.EDACSBCHMode,
			MPT1327BCHMode:          sys.MPT1327BCHMode,
			MPT1327CWSCTolerance:    sys.MPT1327CWSCTolerance,
			MotorolaBCHMode:         sys.MotorolaBCHMode,
			DStarFECMode:            sys.DStarFECMode,
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
		if sys.RIDAliasFile != "" {
			n, err := loadRIDFile(d.rids, sys.RIDAliasFile)
			if err != nil {
				log.Warn("daemon: rid alias load failed",
					"system", sys.Name, "file", sys.RIDAliasFile, "err", err)
				d.addWarning(fmt.Sprintf(
					"rid_alias_file %q for system %q failed to load (%v) — radios on this system will have no operator aliases",
					sys.RIDAliasFile, sys.Name, err))
			} else {
				log.Info("daemon: rids loaded", "system", sys.Name, "count", n)
			}
		}
	}

	// SDR pool — best-effort. Components that need a Voice device
	// fall through gracefully when the pool is empty. The pool is
	// also constructed when only baseband replay recordings are
	// configured, so an offline capture can be decoded with no radio.
	if len(cfg.SDR.Devices) > 0 || len(cfg.Baseband.Replay) > 0 || len(cfg.SDR.RTLTCP) > 0 {
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
					warnGainUnits(log, dev.Serial, dev.Gain, gain)
					h = h.WithGain(gain)
				}
			}
			hints = append(hints, h)
		}
		// Mount baseband replay recordings as virtual tuners. The
		// FileDriver registers globally; the pool's Open then
		// enumerates it alongside any real drivers.
		if len(cfg.Baseband.Replay) > 0 {
			specs := make([]baseband.ReplaySpec, 0, len(cfg.Baseband.Replay))
			for _, r := range cfg.Baseband.Replay {
				loop := true
				if r.Loop != nil {
					loop = *r.Loop
				}
				specs = append(specs, baseband.ReplaySpec{Path: r.File, Serial: r.Serial, Loop: loop})
				if r.Serial != "" {
					hints = append(hints, sdr.Hint{Serial: r.Serial, Role: sdr.ParseRole(r.Role)})
				}
			}
			sdr.Register(baseband.NewFileDriver(specs))
			log.Info("baseband replay mounted", "recordings", len(specs))
		}
		// Mount rtl_tcp endpoints as virtual tuners. Each entry
		// becomes one pool device; the driver dials lazily inside
		// Pool.Open so misconfigured / down hosts surface as
		// "failed to open" warnings rather than blocking daemon
		// startup. Per-endpoint Hint carries role / ppm / gain /
		// bias-tee through the same hint-matcher local USB devices
		// use.
		if len(cfg.SDR.RTLTCP) > 0 {
			rspecs := make([]rtltcp.Spec, 0, len(cfg.SDR.RTLTCP))
			for _, r := range cfg.SDR.RTLTCP {
				if r.Addr == "" {
					log.Warn("daemon: rtl_tcp entry missing addr; skipping")
					continue
				}
				rspecs = append(rspecs, rtltcp.Spec{
					Addr:           r.Addr,
					Serial:         r.Serial,
					Role:           r.Role,
					ConnectTimeout: time.Duration(r.ConnectTimeoutMs) * time.Millisecond,
				})
				if r.Serial != "" {
					h := sdr.Hint{
						Serial:  r.Serial,
						Role:    sdr.ParseRole(r.Role),
						PPM:     r.PPM,
						BiasTee: r.BiasTee,
					}
					if r.Gain != "" {
						gain, ok := parseGain(r.Gain)
						if !ok {
							log.Warn("daemon: ignoring unparseable rtl_tcp gain",
								"serial", r.Serial, "gain", r.Gain)
						} else {
							warnGainUnits(log, r.Serial, r.Gain, gain)
							h = h.WithGain(gain)
						}
					}
					hints = append(hints, h)
				}
			}
			if len(rspecs) > 0 {
				sdr.Register(rtltcp.New(rspecs, log))
				log.Info("rtl_tcp endpoints mounted", "count", len(rspecs))
			}
		}
		if err := d.pool.OpenWith(sdr.PoolOpenOptions{
			SampleRateHz: cfg.SDR.SampleRate,
			Hints:        hints,
			// Strict mode: when the operator has listed devices in
			// sdr.devices, treat that list as an allowlist. Devices
			// that are physically connected but not named are left
			// alone — they may belong to another process on the
			// host (issue #264).
			Strict: len(cfg.SDR.Devices) > 0,
		}); err != nil {
			log.Warn("daemon: SDR pool open failed", "err", err)
			d.addWarning(fmt.Sprintf(
				"SDR pool failed to open (%v) — no radios will demodulate; check device permissions / cabling / kernel modules",
				err))
			d.pool = nil
		} else {
			d.wrapBasebandRecorders(cfg, log)
			d.wrapIQBrokers(cfg, log)
		}
	} else if len(cfg.Trunking.Systems) > 0 {
		// Trunked systems configured but no SDR devices listed —
		// the daemon will look healthy from a logs-only vantage but
		// can't actually decode anything.
		d.addWarning("trunking.systems configured but sdr.devices is empty — daemon has nothing to demodulate; add at least one device")
	}

	// Virtual voice taps on any `role: wideband` dongle, built before the
	// voice pool (below) and the composer's virtualVoiceMap so a wideband-
	// only topology actually has voice devices to follow grants with.
	// Building these after the voice pool was the cause of issue #422
	// (every grant dropped with "no voice SDR").
	if err := d.buildVirtualVoiceTuners(cfg, log); err != nil {
		return nil, err
	}

	// Metrics — constructed early so downstream components (notably
	// the cc decoder's IQ-power gauge) can publish into it. Run +
	// Close are still kicked off later from Daemon.Run / Daemon.Close
	// so the subscription does no work until the daemon is actually
	// running.
	if cfg.Metrics.Enabled {
		var pool metrics.Snapshotter
		if d.pool != nil {
			pool = d.pool
		}
		m, err := metrics.New(d.bus, pool, version)
		if err != nil {
			return nil, fmt.Errorf("daemon: metrics: %w", err)
		}
		d.metrics = m
	}

	// Voice device list from the pool; empty when no SDRs.
	d.voicePool = trunking.NewVoicePool(d.collectVoiceDevices())
	// Wire the voice pool's reacquire hook so a USB disconnect
	// that left a voice dongle's handle stale gets the device
	// re-opened from sdr.Pool on the next Bind, before the call
	// drops. Pool entries share the configured sdr.sample_rate;
	// per-role rate divergence isn't supported today, so the same
	// value used at pool.Open is correct for reacquire. Issue #345.
	if d.pool != nil {
		sampleRate := cfg.SDR.SampleRate
		d.voicePool.SetReacquire(func(serial string) (trunking.Tuner, error) {
			entry, err := d.pool.Reacquire(serial, sampleRate)
			if err != nil {
				return nil, err
			}
			return entry.Device, nil
		})
	}

	engine, err := trunking.NewEngine(trunking.EngineOptions{
		Bus:         d.bus,
		Log:         log,
		VoicePool:   d.voicePool,
		Talkgroups:  d.talkgroups,
		ScanMode:    trunking.ParseScanMode(cfg.Scanner.ScanMode),
		CallTimeout: time.Duration(cfg.Trunking.CallTimeoutMs) * time.Millisecond,
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

	// Outbound call-streaming manager — optional. Subscribes to the
	// bus at construction so calls completed before Run starts are
	// not lost. Built only when at least one feed is enabled.
	{
		bcastRate := int(cfg.Recordings.SampleRate)
		if bcastRate == 0 {
			bcastRate = 8000
		}
		mgr, err := buildBroadcastManager(cfg.Broadcast, bcastRate, d.bus, log)
		if err != nil {
			return nil, fmt.Errorf("daemon: broadcast: %w", err)
		}
		if mgr != nil {
			d.broadcast = mgr
			log.Info("outbound call streaming enabled", "backends", mgr.Backends())
		}
	}

	// Decoded-message log — optional. Subscribes to the bus and writes
	// a human-readable text log of every trunking event.
	if cfg.Log.MessageLog.Enabled && cfg.Log.MessageLog.Path != "" {
		ml, err := gtlog.NewMessageLog(gtlog.MessageLogOptions{
			Bus:       d.bus,
			Path:      cfg.Log.MessageLog.Path,
			MaxSizeMB: cfg.Log.MessageLog.MaxSizeMB,
		})
		if err != nil {
			return nil, fmt.Errorf("daemon: message log: %w", err)
		}
		d.messageLog = ml
	}

	// Affiliation tracker — always on. Subscribes to the bus and
	// maintains the protocol-agnostic unit-activity table surfaced at
	// GET /api/v1/affiliations.
	{
		at, err := trunking.NewAffiliationTracker(trunking.AffiliationTrackerOptions{Bus: d.bus})
		if err != nil {
			return nil, fmt.Errorf("daemon: affiliation tracker: %w", err)
		}
		d.affiliations = at
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
			sinks = append(sinks, playerSink{p: d.player, engine: d.engine})
		}
		sinks = append(sinks, d.audioPub)
		var sink composer.PCMSink = d.recorder
		if len(sinks) > 1 {
			sink = fanoutSink(sinks)
			// Surface the fanout wiring at startup so operators can
			// confirm from logs that the raw-frame path is in place.
			// Before the fanoutSink.WriteRawFrame fix (issue #356), a
			// multi-sink config silently dropped every IMBE / AMBE
			// frame and there was nothing in the logs to tell the
			// operator the audio path was broken.
			log.Info("composer: sink fanout configured",
				"count", len(sinks), "raw_frames_supported", true)
		} else {
			log.Info("composer: sink direct configured")
		}
		comp, err := composer.New(composer.Options{
			Bus:           d.bus,
			Devices:       &poolDevices{pool: d.pool, rateHz: cfg.SDR.SampleRate, virtualMap: d.virtualVoiceMap()},
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
				// Route the supervisor's retunes through the broker
				// (when present) so SetCenterFreq follows the same
				// broker.SetInner swap path the ccdecoder uses after
				// pool.Reacquire — and so retunes during a future
				// spectrum subscription still hit whatever device the
				// pool now considers authoritative.
				var cchTuner cchunt.Tuner = controlEntry.Device
				if br := d.iqBrokers[controlEntry.Info.Serial]; br != nil {
					cchTuner = br
				}
				sup, err := cchunt.New(cchunt.Options{
					Bus:            d.bus,
					Log:            log,
					Tuner:          cchTuner,
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
				// Avoid the typed-nil trap: a nil *metrics.Metrics
				// wrapped in IQPowerObserver still tests as non-nil
				// inside the decoder. Hand the interface only when
				// the concrete pointer is alive.
				var iqObs ccdecoder.IQPowerObserver
				if d.metrics != nil {
					iqObs = d.metrics
				}
				// Route the control SDR's IQ through the iqtap broker
				// so secondary observers (live spectrum, future paging /
				// AIS / ADS-B decoders) can Subscribe without disturbing
				// the CC decoder. Tuner goes through the same broker so
				// SetCenterFreq calls land on whichever inner the broker
				// currently wraps (kept in sync with pool.Reacquire via
				// broker.SetInner below).
				var iqSrc ccdecoder.IQSource = controlEntry.Device
				var tuner ccdecoder.Tuner = controlEntry.Device
				if br := d.iqBrokers[controlEntry.Info.Serial]; br != nil {
					iqSrc = br
					tuner = br
				}
				iqCorrect := false
				for _, dev := range cfg.SDR.Devices {
					if dev.Serial == controlEntry.Info.Serial {
						iqCorrect = dev.IQCorrect
						break
					}
				}
				d.ccDecoderOpts = ccdecoder.Options{
					Bus:          d.bus,
					Log:          log,
					Tuner:        tuner,
					IQ:           iqSrc,
					Systems:      d.systems,
					SampleRateHz: float64(cfg.SDR.SampleRate),
					Metrics:      iqObs,
					IQCorrect:    iqCorrect,
				}
				d.controlSerial = controlEntry.Info.Serial
				d.controlSampleRate = cfg.SDR.SampleRate
				dec, err := ccdecoder.New(d.ccDecoderOpts)
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
				convRec = convFanoutRecorder{d.recorder, playerSink{p: d.player, engine: d.engine}}
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

	// Wide-band DMR Tier II engines — one per dongle the operator
	// pinned to `role: wideband` in config. Each engine fans the
	// dongle's IQ stream out to N independent T2 receivers + state
	// machines via internal/dsp/tuner, letting a single SDR cover
	// every repeater inside its IQ bandwidth (typically a 2.4 MHz
	// cluster around a chosen centre frequency).
	if d.pool != nil {
		for _, devCfg := range cfg.SDR.Devices {
			if devCfg.Role != "wideband" {
				continue
			}
			entry := d.pool.FindBySerial(devCfg.Serial)
			if entry == nil {
				log.Warn("daemon: wideband: dongle with configured serial not in pool",
					"serial", devCfg.Serial)
				continue
			}
			channels := make([]widebandt2.ChannelConfig, 0, len(devCfg.Channels))
			for _, ch := range devCfg.Channels {
				channels = append(channels, widebandt2.ChannelConfig{
					FrequencyHz: ch.FrequencyHz,
					SystemName:  ch.System,
				})
			}
			// Route IQ + tuning through the iqtap broker so the live
			// spectrum view (and any other secondary observer) can
			// Subscribe to chunk copies. Without the broker indirection
			// the engine's StreamIQ would bypass the fan-out goroutine
			// and its SetCenterFreq would skip the broker's centerHz
			// cache, leaving spectrum frames empty and stamped at 0.
			// Mirrors the CC decoder wiring above.
			var iqDev sdr.Device = entry.Device
			if br := d.iqBrokers[entry.Info.Serial]; br != nil {
				iqDev = br
			}
			eng, err := widebandt2.New(widebandt2.Options{
				Log:           log,
				Bus:           d.bus,
				Device:        iqDev,
				SampleRateHz:  cfg.SDR.SampleRate,
				CenterFreqHz:  devCfg.CenterFreqHz,
				TunerStrategy: devCfg.TunerStrategy,
				Channels:      channels,
				Systems:       d.systems,
			})
			if err != nil {
				return nil, fmt.Errorf("daemon: widebandt2 %q: %w", devCfg.Serial, err)
			}
			d.widebandT2 = append(d.widebandT2, eng)
			// Virtual voice taps for this dongle are built earlier by
			// buildVirtualVoiceTuners (before the voice pool and composer
			// are constructed) so trunked grants inside the IQ window have
			// a device to land on. See issue #422.
		}
	}

	// POCSAG paging receivers — one per configured paging.pocsag
	// entry. Constructed here; the run loop spawns them with the
	// iqtap broker subscription. Per-entry validation lives in
	// the receiver.New constructor; entries that fail validation
	// surface as a startup warning and are skipped (their slot
	// is preserved as nil to keep slice indexing simple).
	for _, pc := range cfg.Paging.POCSAG {
		spec := pocsagSpec{serial: pc.Serial, freq: pc.FrequencyHz}
		if pc.Serial == "" || pc.FrequencyHz == 0 {
			d.addWarning(fmt.Sprintf(
				"paging.pocsag: entry missing serial or frequency_hz (serial=%q freq=%d) — skipped",
				pc.Serial, pc.FrequencyHz))
			d.pocsagReceivers = append(d.pocsagReceivers, nil)
			d.pocsagSpecs = append(d.pocsagSpecs, spec)
			continue
		}
		rcv, err := pocsagrx.New(pocsagrx.Options{
			InputRateHz: cfg.SDR.SampleRate,
			BaudHz:      pc.BaudHz,
			SourceName:  pc.Serial,
			Bus:         d.bus,
			Log:         log,
		})
		if err != nil {
			d.addWarning(fmt.Sprintf("paging.pocsag[%s]: %v — skipped", pc.Serial, err))
			d.pocsagReceivers = append(d.pocsagReceivers, nil)
			d.pocsagSpecs = append(d.pocsagSpecs, spec)
			continue
		}
		d.pocsagReceivers = append(d.pocsagReceivers, rcv)
		d.pocsagSpecs = append(d.pocsagSpecs, spec)
	}

	// APRS / AX.25 Bell-202 AFSK receivers — one per configured
	// aprs.channels entry. Same construction shape as POCSAG above:
	// per-entry validation in the receiver, failures surface as a
	// startup warning and skip the entry (nil slot preserved for
	// stable indexing).
	for _, ac := range cfg.APRS.Channels {
		spec := aprsSpec{serial: ac.Serial, freq: ac.FrequencyHz}
		if ac.Serial == "" || ac.FrequencyHz == 0 {
			d.addWarning(fmt.Sprintf(
				"aprs.channels: entry missing serial or frequency_hz (serial=%q freq=%d) — skipped",
				ac.Serial, ac.FrequencyHz))
			d.aprsReceivers = append(d.aprsReceivers, nil)
			d.aprsSpecs = append(d.aprsSpecs, spec)
			continue
		}
		rcv, err := aprsafsk.New(aprsafsk.Options{
			InputRateHz: cfg.SDR.SampleRate,
			SourceName:  ac.Serial,
			Bus:         d.bus,
			DropBadFCS:  ac.DropBadFCS,
			DropNonUI:   ac.DropNonUI,
			Log:         log,
		})
		if err != nil {
			d.addWarning(fmt.Sprintf("aprs.channels[%s]: %v — skipped", ac.Serial, err))
			d.aprsReceivers = append(d.aprsReceivers, nil)
			d.aprsSpecs = append(d.aprsSpecs, spec)
			continue
		}
		d.aprsReceivers = append(d.aprsReceivers, rcv)
		d.aprsSpecs = append(d.aprsSpecs, spec)
	}

	// AIS GMSK receivers — one per configured ais.channels entry.
	// Same construction shape as POCSAG / APRS above: per-entry
	// validation in the receiver, failures surface as a startup
	// warning and skip the entry (nil slot preserved for stable
	// indexing).
	for _, ac := range cfg.AIS.Channels {
		spec := aisSpec{serial: ac.Serial, freq: ac.FrequencyHz}
		if ac.Serial == "" || ac.FrequencyHz == 0 {
			d.addWarning(fmt.Sprintf(
				"ais.channels: entry missing serial or frequency_hz (serial=%q freq=%d) — skipped",
				ac.Serial, ac.FrequencyHz))
			d.aisReceivers = append(d.aisReceivers, nil)
			d.aisSpecs = append(d.aisSpecs, spec)
			continue
		}
		rcv, err := aisgmsk.New(aisgmsk.Options{
			InputRateHz:     cfg.SDR.SampleRate,
			SourceName:      ac.Serial,
			Bus:             d.bus,
			DropBadFCS:      ac.DropBadFCS,
			DropNonPosition: ac.DropNonPosition,
			Log:             log,
		})
		if err != nil {
			d.addWarning(fmt.Sprintf("ais.channels[%s]: %v — skipped", ac.Serial, err))
			d.aisReceivers = append(d.aisReceivers, nil)
			d.aisSpecs = append(d.aisSpecs, spec)
			continue
		}
		d.aisReceivers = append(d.aisReceivers, rcv)
		d.aisSpecs = append(d.aisSpecs, spec)
	}

	// DSC FFSK receivers — one per configured dsc.channels entry.
	// Same construction shape as AIS / MDC1200 above: per-entry
	// validation in the receiver, failures surface as a startup
	// warning and skip the entry (nil slot preserved for stable
	// indexing).
	for _, dc := range cfg.DSC.Channels {
		spec := dscSpec{serial: dc.Serial, freq: dc.FrequencyHz}
		if dc.Serial == "" || dc.FrequencyHz == 0 {
			d.addWarning(fmt.Sprintf(
				"dsc.channels: entry missing serial or frequency_hz (serial=%q freq=%d) — skipped",
				dc.Serial, dc.FrequencyHz))
			d.dscReceivers = append(d.dscReceivers, nil)
			d.dscSpecs = append(d.dscSpecs, spec)
			continue
		}
		rcv, err := dscffsk.New(dscffsk.Options{
			InputRateHz: cfg.SDR.SampleRate,
			SourceName:  dc.Serial,
			Bus:         d.bus,
			DropBadFCS:  dc.DropBadFCS,
			Log:         log,
		})
		if err != nil {
			d.addWarning(fmt.Sprintf("dsc.channels[%s]: %v — skipped", dc.Serial, err))
			d.dscReceivers = append(d.dscReceivers, nil)
			d.dscSpecs = append(d.dscSpecs, spec)
			continue
		}
		d.dscReceivers = append(d.dscReceivers, rcv)
		d.dscSpecs = append(d.dscSpecs, spec)
	}

	// MDC1200 FFSK receivers — one per configured mdc1200.channels
	// entry. Same construction shape as APRS / AIS above: per-entry
	// validation in the receiver, failures surface as a startup
	// warning and skip the entry (nil slot preserved for stable
	// indexing).
	for _, mc := range cfg.MDC1200.Channels {
		spec := mdc1200Spec{serial: mc.Serial, freq: mc.FrequencyHz}
		if mc.Serial == "" || mc.FrequencyHz == 0 {
			d.addWarning(fmt.Sprintf(
				"mdc1200.channels: entry missing serial or frequency_hz (serial=%q freq=%d) — skipped",
				mc.Serial, mc.FrequencyHz))
			d.mdc1200Receivers = append(d.mdc1200Receivers, nil)
			d.mdc1200Specs = append(d.mdc1200Specs, spec)
			continue
		}
		rcv, err := mdc1200afsk.New(mdc1200afsk.Options{
			InputRateHz: cfg.SDR.SampleRate,
			SourceName:  mc.Serial,
			Bus:         d.bus,
			DropBadCRC:  mc.DropBadCRC,
			Log:         log,
		})
		if err != nil {
			d.addWarning(fmt.Sprintf("mdc1200.channels[%s]: %v — skipped", mc.Serial, err))
			d.mdc1200Receivers = append(d.mdc1200Receivers, nil)
			d.mdc1200Specs = append(d.mdc1200Specs, spec)
			continue
		}
		d.mdc1200Receivers = append(d.mdc1200Receivers, rcv)
		d.mdc1200Specs = append(d.mdc1200Specs, spec)
	}

	// ADS-B BEAST upstreams — one client per configured
	// adsb.beast_upstreams entry. Each opens a TCP connection
	// to a dump1090 / readsb BEAST output port, decodes the
	// Mode-S frames, runs them through the CPR pair-tracker,
	// and publishes KindAircraftReport. Validation failures
	// surface as startup warnings.
	for _, bc := range cfg.ADSB.BeastUpstreams {
		if bc.Addr == "" {
			d.addWarning(fmt.Sprintf(
				"adsb.beast_upstreams: entry missing addr (name=%q) — skipped",
				bc.Name))
			continue
		}
		name := bc.Name
		if name == "" {
			name = bc.Addr
		}
		client, err := adsbbeast.New(adsbbeast.Options{
			Addr:       bc.Addr,
			Bus:        d.bus,
			SourceName: name,
			Log:        log,
		})
		if err != nil {
			d.addWarning(fmt.Sprintf("adsb.beast_upstreams[%s]: %v — skipped", name, err))
			continue
		}
		d.adsbBeastClients = append(d.adsbBeastClients, client)
		d.adsbBeastNames = append(d.adsbBeastNames, name)
	}

	// ADS-B native PPM receivers — one per configured adsb.channels
	// entry. Each pins an SDR to 1090 MHz and demodulates Mode-S
	// straight off the air, publishing the same KindAircraftReport the
	// BEAST clients do. Same construction shape as the AIS receivers.
	for _, ac := range cfg.ADSB.Channels {
		freq := ac.FrequencyHz
		if freq == 0 {
			freq = 1_090_000_000 // 1090 MHz default
		}
		spec := adsbPPMSpec{serial: ac.Serial, freq: freq}
		if ac.Serial == "" {
			d.addWarning("adsb.channels: entry missing serial — skipped")
			d.adsbPPMReceivers = append(d.adsbPPMReceivers, nil)
			d.adsbPPMSpecs = append(d.adsbPPMSpecs, spec)
			continue
		}
		rcv, err := adsbppm.New(adsbppm.Options{
			InputRateHz: cfg.SDR.SampleRate,
			SourceName:  ac.Serial,
			Bus:         d.bus,
			Log:         log,
		})
		if err != nil {
			d.addWarning(fmt.Sprintf("adsb.channels[%s]: %v — skipped", ac.Serial, err))
			d.adsbPPMReceivers = append(d.adsbPPMReceivers, nil)
			d.adsbPPMSpecs = append(d.adsbPPMSpecs, spec)
			continue
		}
		d.adsbPPMReceivers = append(d.adsbPPMReceivers, rcv)
		d.adsbPPMSpecs = append(d.adsbPPMSpecs, spec)
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

		ll, err := storage.NewLocationLog(db, d.bus, log)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("daemon: location log: %w", err)
		}
		d.locationLog = ll

		bs, err := storage.NewBookmarkStore(db, d.bus)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("daemon: bookmarks: %w", err)
		}
		d.bookmarks = bs

		pl, err := storage.NewPagerLog(db, d.bus, log)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("daemon: pager log: %w", err)
		}
		d.pagerLog = pl

		al, err := storage.NewAPRSLog(db, d.bus, log)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("daemon: aprs log: %w", err)
		}
		d.aprsLog = al

		vl, err := storage.NewVesselLog(db, d.bus, log)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("daemon: vessel log: %w", err)
		}
		d.vesselLog = vl

		dl, err := storage.NewDSCLog(db, d.bus, log)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("daemon: dsc log: %w", err)
		}
		d.dscLog = dl

		acl, err := storage.NewAircraftLog(db, d.bus, log)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("daemon: aircraft log: %w", err)
		}
		d.aircraftLog = acl

		mdl, err := storage.NewMDC1200Log(db, d.bus, log)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("daemon: mdc1200 log: %w", err)
		}
		d.mdc1200Log = mdl

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
			RIDs:           d.rids,
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
		if len(d.iqBrokers) > 0 {
			opts.Spectrum = newSpectrumProvider(d.pool, d.iqBrokers, log)
			opts.Diag = newDiagProvider(d.pool, d.iqBrokers, cfg.SDR.SampleRate, log)
		}
		if d.bookmarks != nil {
			opts.Bookmarks = bookmarkProvider{store: d.bookmarks}
		}
		if d.pagerLog != nil {
			opts.Pager = pagerProvider{log: d.pagerLog}
		}
		if d.aprsLog != nil {
			opts.APRS = aprsProvider{log: d.aprsLog}
		}
		if d.vesselLog != nil {
			opts.AIS = aisProvider{log: d.vesselLog}
		}
		if d.dscLog != nil {
			opts.DSC = dscProvider{log: d.dscLog}
		}
		if d.aircraftLog != nil {
			opts.ADSB = adsbProvider{log: d.aircraftLog}
		}
		if d.mdc1200Log != nil {
			opts.MDC1200 = mdc1200Provider{log: d.mdc1200Log}
		}
		if d.db != nil {
			opts.History = api.HistoryFromStorage(d.db)
		}
		if d.locationLog != nil {
			opts.Locations = api.LocationsFromStorage(d.locationLog)
		}
		if d.affiliations != nil {
			opts.Affiliations = affiliationProvider{d.affiliations}
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
		if d.broadcast != nil {
			opts.Broadcast = broadcastStatus{d.broadcast}
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
			opts.Importer = newDaemonImporter(d)
		}
		// Embed the SPA when the build linked in real assets
		// (see web/embed.go). HasAssets is false for fresh
		// checkouts without `make web-build` — the launcher falls
		// back to filesystem discovery in that case.
		if gtweb.HasAssets() {
			opts.WebAssets = gtweb.Assets()
		}
		srv, err := api.NewServer(opts)
		if err != nil {
			return nil, fmt.Errorf("daemon: http api: %w", err)
		}
		d.httpAPI = srv
	}

	// rigctld TCP server — optional. Exposes the control SDR's
	// frequency to external Hamlib clients (loggers, sat trackers).
	// Wired only when an SDR is in the pool *and* the operator opted
	// in via api.rigctld in the YAML; otherwise stays off so a daemon
	// without a tuner doesn't pretend to be a controllable rig.
	if cfg.API.Rigctld != "" && d.pool != nil {
		ctrlEntry := d.pool.FirstByRole(sdr.RoleControl)
		if ctrlEntry == nil {
			log.Warn("daemon: rigctld configured but no control SDR in pool; skipping")
		} else {
			var rigCtrl rigctld.Controller
			if br := d.iqBrokers[ctrlEntry.Info.Serial]; br != nil {
				rigCtrl = brokerRigController{serial: ctrlEntry.Info.Serial, broker: br}
			} else {
				rigCtrl = &poolRigController{serial: ctrlEntry.Info.Serial, dev: ctrlEntry.Device}
			}
			rs, err := rigctld.New(cfg.API.Rigctld, rigCtrl, log)
			if err != nil {
				return nil, fmt.Errorf("daemon: rigctld: %w", err)
			}
			d.rigctld = rs
		}
	}

	// gRPC — optional.
	if cfg.API.GRPCAddr != "" {
		grpcOpts := api.GRPCServerOptions{
			Addr:       cfg.API.GRPCAddr,
			Systems:    d.systems,
			Talkgroups: d.talkgroups,
			RIDs:       d.rids,
			Engine:     d.engine,
			Audio:      d.audioPub,
			Log:        log,
			TLSCert:    cfg.API.TLSCert,
			TLSKey:     cfg.API.TLSKey,
		}
		if d.affiliations != nil {
			grpcOpts.Affiliations = affiliationProvider{d.affiliations}
		}
		if d.db != nil {
			grpcOpts.History = api.HistoryFromStorage(d.db)
		}
		gsrv, err := api.NewGRPCServer(grpcOpts)
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
	// Catch the single-SDR-control-only setup early: trunking systems
	// declared but no `role: voice` SDR means every grant will drop at
	// HandleGrant. Warn once at startup so the operator sees it before
	// the first grant. Issue #379.
	if len(d.systems) > 0 && len(d.voicePool.Devices()) == 0 {
		d.log.Warn("no voice SDR configured but trunking systems are defined; voice grants will be dropped — add a role: voice device (see docs/hardware.md)",
			"systems", len(d.systems))
	}

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
	if d.locationLog != nil {
		d.spawn(runCtx, "locationlog", false, func(ctx context.Context) error {
			return d.locationLog.Run(ctx)
		})
	}
	if d.pagerLog != nil {
		d.spawn(runCtx, "pagerlog", false, func(ctx context.Context) error {
			return d.pagerLog.Run(ctx)
		})
	}
	if d.aprsLog != nil {
		d.spawn(runCtx, "aprslog", false, func(ctx context.Context) error {
			return d.aprsLog.Run(ctx)
		})
	}
	if d.vesselLog != nil {
		d.spawn(runCtx, "vessellog", false, func(ctx context.Context) error {
			return d.vesselLog.Run(ctx)
		})
	}
	if d.dscLog != nil {
		d.spawn(runCtx, "dsclog", false, func(ctx context.Context) error {
			return d.dscLog.Run(ctx)
		})
	}
	if d.aircraftLog != nil {
		d.spawn(runCtx, "aircraftlog", false, func(ctx context.Context) error {
			return d.aircraftLog.Run(ctx)
		})
	}
	if d.mdc1200Log != nil {
		d.spawn(runCtx, "mdc1200log", false, func(ctx context.Context) error {
			return d.mdc1200Log.Run(ctx)
		})
	}
	if d.messageLog != nil {
		d.spawn(runCtx, "messagelog", false, func(ctx context.Context) error {
			return d.messageLog.Run(ctx)
		})
	}
	if d.affiliations != nil {
		d.spawn(runCtx, "affiliations", false, func(ctx context.Context) error {
			return d.affiliations.Run(ctx)
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
	if d.broadcast != nil {
		d.spawn(runCtx, "broadcast", false, func(ctx context.Context) error {
			return d.broadcast.Run(ctx)
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
		d.spawn(runCtx, "ccdecoder", false, d.runCCDecoderWithRetry)
	}
	if d.convScan != nil {
		d.spawn(runCtx, "conv-scanner", false, func(ctx context.Context) error {
			return d.convScan.Run(ctx)
		})
	}
	for i, eng := range d.widebandT2 {
		eng := eng
		name := fmt.Sprintf("widebandt2-%d", i)
		d.spawn(runCtx, name, false, func(ctx context.Context) error {
			return eng.Run(ctx)
		})
	}
	// POCSAG paging receivers — one per configured paging.pocsag
	// entry. Each subscribes to its assigned SDR's iqtap broker
	// and runs the FM-demod → bit-slicer → syncer pipeline,
	// publishing pages onto the events bus where the PagerLog
	// subscriber persists them and the web /pagers panel renders
	// them. Non-essential: a misconfigured paging frequency or a
	// missing SDR is logged but doesn't bring down the trunking
	// pipeline.
	for i, rcv := range d.pocsagReceivers {
		if rcv == nil {
			continue // skipped at construction; warning already logged
		}
		rcv := rcv
		spec := d.pocsagSpecs[i]
		name := fmt.Sprintf("pocsag-%s-%d", spec.serial, spec.freq)
		d.spawn(runCtx, name, false, func(ctx context.Context) error {
			br := d.iqBrokers[spec.serial]
			if br == nil {
				d.log.Warn("pocsag: SDR not found, skipping receiver",
					"serial", spec.serial, "freq_hz", spec.freq)
				return nil
			}
			if err := br.SetCenterFreq(spec.freq); err != nil {
				d.log.Warn("pocsag: SetCenterFreq failed",
					"serial", spec.serial, "freq_hz", spec.freq, "err", err)
				return nil
			}
			sub := br.Subscribe()
			defer sub.Close()
			return rcv.Process(ctx, sub.C)
		})
	}
	// APRS receivers — same shape as POCSAG above. Each subscribes
	// to its assigned SDR's iqtap broker and runs the AFSK pipeline
	// (FM demod → FFSK discriminator → symbol-timing recovery →
	// NRZI → HDLC framer → AX.25 + APRS parsing), publishing packets
	// onto the events bus where the APRSLog subscriber persists them
	// and the /aprs panel renders them. Non-essential: a missing SDR
	// or misconfigured frequency is logged but doesn't bring down the
	// trunking pipeline.
	for i, rcv := range d.aprsReceivers {
		if rcv == nil {
			continue // skipped at construction; warning already logged
		}
		rcv := rcv
		spec := d.aprsSpecs[i]
		name := fmt.Sprintf("aprs-%s-%d", spec.serial, spec.freq)
		d.spawn(runCtx, name, false, func(ctx context.Context) error {
			br := d.iqBrokers[spec.serial]
			if br == nil {
				d.log.Warn("aprs: SDR not found, skipping receiver",
					"serial", spec.serial, "freq_hz", spec.freq)
				return nil
			}
			if err := br.SetCenterFreq(spec.freq); err != nil {
				d.log.Warn("aprs: SetCenterFreq failed",
					"serial", spec.serial, "freq_hz", spec.freq, "err", err)
				return nil
			}
			sub := br.Subscribe()
			defer sub.Close()
			return rcv.Process(ctx, sub.C)
		})
	}
	// AIS receivers — same shape as APRS / POCSAG above. Each
	// subscribes to its assigned SDR's iqtap broker and runs the
	// GMSK pipeline (FM demod → GFSK matched filter → symbol-
	// timing recovery → NRZI → HDLC framer → CRC validation →
	// AIS message parser), publishing messages onto the events
	// bus where the VesselLog subscriber persists them and the
	// /ais panel renders them. Non-essential: a missing SDR or
	// misconfigured frequency is logged but doesn't bring down
	// the trunking pipeline.
	for i, rcv := range d.aisReceivers {
		if rcv == nil {
			continue // skipped at construction; warning already logged
		}
		rcv := rcv
		spec := d.aisSpecs[i]
		name := fmt.Sprintf("ais-%s-%d", spec.serial, spec.freq)
		d.spawn(runCtx, name, false, func(ctx context.Context) error {
			br := d.iqBrokers[spec.serial]
			if br == nil {
				d.log.Warn("ais: SDR not found, skipping receiver",
					"serial", spec.serial, "freq_hz", spec.freq)
				return nil
			}
			if err := br.SetCenterFreq(spec.freq); err != nil {
				d.log.Warn("ais: SetCenterFreq failed",
					"serial", spec.serial, "freq_hz", spec.freq, "err", err)
				return nil
			}
			sub := br.Subscribe()
			defer sub.Close()
			return rcv.Process(ctx, sub.C)
		})
	}
	// DSC receivers — same shape as AIS above. Each subscribes to its
	// assigned SDR's iqtap broker and runs the FFSK pipeline (FM demod
	// → FFSK discriminator at 1300/2100 Hz → symbol-timing recovery →
	// direct-FSK slicer → BCH character sync → ITU-R M.493 parser),
	// publishing sequences onto the events bus where the DSCLog
	// subscriber persists them and the /dsc panel renders them.
	// Non-essential: a missing SDR or misconfigured frequency is
	// logged but doesn't bring down the trunking pipeline.
	for i, rcv := range d.dscReceivers {
		if rcv == nil {
			continue // skipped at construction; warning already logged
		}
		rcv := rcv
		spec := d.dscSpecs[i]
		name := fmt.Sprintf("dsc-%s-%d", spec.serial, spec.freq)
		d.spawn(runCtx, name, false, func(ctx context.Context) error {
			br := d.iqBrokers[spec.serial]
			if br == nil {
				d.log.Warn("dsc: SDR not found, skipping receiver",
					"serial", spec.serial, "freq_hz", spec.freq)
				return nil
			}
			if err := br.SetCenterFreq(spec.freq); err != nil {
				d.log.Warn("dsc: SetCenterFreq failed",
					"serial", spec.serial, "freq_hz", spec.freq, "err", err)
				return nil
			}
			sub := br.Subscribe()
			defer sub.Close()
			return rcv.Process(ctx, sub.C)
		})
	}
	// MDC1200 receivers — same shape as APRS / AIS above. Each
	// subscribes to its assigned SDR's iqtap broker and runs the FFSK
	// pipeline (FM demod → FFSK discriminator → symbol-timing recovery
	// → NRZ slicer → sync framer → op/arg/unit-ID parser), publishing
	// bursts onto the events bus where the MDC1200Log subscriber
	// persists them and the /mdc1200 panel renders them. Non-essential:
	// a missing SDR or misconfigured frequency is logged but doesn't
	// bring down the trunking pipeline.
	for i, rcv := range d.mdc1200Receivers {
		if rcv == nil {
			continue // skipped at construction; warning already logged
		}
		rcv := rcv
		spec := d.mdc1200Specs[i]
		name := fmt.Sprintf("mdc1200-%s-%d", spec.serial, spec.freq)
		d.spawn(runCtx, name, false, func(ctx context.Context) error {
			br := d.iqBrokers[spec.serial]
			if br == nil {
				d.log.Warn("mdc1200: SDR not found, skipping receiver",
					"serial", spec.serial, "freq_hz", spec.freq)
				return nil
			}
			if err := br.SetCenterFreq(spec.freq); err != nil {
				d.log.Warn("mdc1200: SetCenterFreq failed",
					"serial", spec.serial, "freq_hz", spec.freq, "err", err)
				return nil
			}
			sub := br.Subscribe()
			defer sub.Close()
			return rcv.Process(ctx, sub.C)
		})
	}
	// ADS-B BEAST upstream clients — each consumes Mode-S
	// frames from a separately-running dump1090 / readsb /
	// commercial hub. Reconnect-with-backoff on disconnects;
	// the embedded CPR tracker pairs even+odd position halves
	// so the resulting AircraftReport rows carry decoded
	// lat/lon.
	for i, client := range d.adsbBeastClients {
		client := client
		name := fmt.Sprintf("adsb-beast-%s", d.adsbBeastNames[i])
		d.spawn(runCtx, name, false, func(ctx context.Context) error {
			return client.Run(ctx)
		})
	}
	// ADS-B native PPM receivers — same shape as the AIS receivers.
	// Each subscribes to its assigned SDR's iqtap broker (pinned to
	// 1090 MHz) and demodulates Mode-S frames off the air, publishing
	// KindAircraftReport. Non-essential: a missing SDR is logged but
	// doesn't bring down the pipeline.
	for i, rcv := range d.adsbPPMReceivers {
		if rcv == nil {
			continue // skipped at construction; warning already logged
		}
		rcv := rcv
		spec := d.adsbPPMSpecs[i]
		name := fmt.Sprintf("adsb-ppm-%s-%d", spec.serial, spec.freq)
		d.spawn(runCtx, name, false, func(ctx context.Context) error {
			br := d.iqBrokers[spec.serial]
			if br == nil {
				d.log.Warn("adsb/ppm: SDR not found, skipping receiver",
					"serial", spec.serial, "freq_hz", spec.freq)
				return nil
			}
			if err := br.SetCenterFreq(spec.freq); err != nil {
				d.log.Warn("adsb/ppm: SetCenterFreq failed",
					"serial", spec.serial, "freq_hz", spec.freq, "err", err)
				return nil
			}
			sub := br.Subscribe()
			defer sub.Close()
			return rcv.Process(ctx, sub.C)
		})
	}
	if d.pool != nil {
		// Periodic USB-disconnect watchdog. Catches dongles that
		// vanish while idle (between calls / hunts) and re-acquires
		// them by serial as soon as they come back, so the next
		// consumer doesn't pay the reacquire round-trip mid-call.
		// Negative interval disables; zero falls back to
		// sdr.DefaultWatchdogInterval (30s). Issue #345.
		interval := watchdogInterval(d.cfg.SDR.WatchdogIntervalMs)
		if interval > 0 {
			sampleRate := d.cfg.SDR.SampleRate
			d.spawn(runCtx, "sdr-watchdog", false, func(ctx context.Context) error {
				return d.pool.RunWatchdog(ctx, interval, sampleRate)
			})
		}
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
	if d.rigctld != nil {
		// Non-essential: external loggers / sat-trackers consume
		// this, but a bind failure (port 4532 already taken by a
		// real Hamlib daemon, for instance) shouldn't bring down
		// the trunking pipeline. Log + continue.
		d.spawn(runCtx, "rigctld", false, func(ctx context.Context) error {
			return d.rigctld.Run(ctx)
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
		if d.rigctld != nil {
			_ = d.rigctld.Close()
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
		if d.broadcast != nil {
			_ = d.broadcast.Close()
		}
		if d.recorder != nil {
			_ = d.recorder.Close()
		}
		if d.callLog != nil {
			_ = d.callLog.Close()
		}
		if d.locationLog != nil {
			_ = d.locationLog.Close()
		}
		if d.pagerLog != nil {
			_ = d.pagerLog.Close()
		}
		if d.aprsLog != nil {
			_ = d.aprsLog.Close()
		}
		if d.vesselLog != nil {
			_ = d.vesselLog.Close()
		}
		if d.dscLog != nil {
			_ = d.dscLog.Close()
		}
		if d.aircraftLog != nil {
			_ = d.aircraftLog.Close()
		}
		if d.mdc1200Log != nil {
			_ = d.mdc1200Log.Close()
		}
		if d.messageLog != nil {
			_ = d.messageLog.Close()
		}
		if d.affiliations != nil {
			_ = d.affiliations.Close()
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

// Cfg returns a snapshot of the daemon's current in-memory config
// under an RLock. Callers that need to observe values that might
// race against a SIGHUP / PATCH reload should use this getter
// instead of reading d.cfg directly.
func (d *Daemon) Cfg() config.Config {
	d.cfgMu.RLock()
	defer d.cfgMu.RUnlock()
	return d.cfg
}

// HTTPListenAddr returns the resolved address of the HTTP API
// (helpful for tests using an ephemeral ":0" port). Returns "" if no
// API server is configured. Prefers the listener's actually-bound
// address once the API server has called net.Listen — important for
// configurations that use ":0" / "127.0.0.1:0" so callers see the
// kernel-assigned port instead of "0".
func (d *Daemon) HTTPListenAddr() string {
	if d.httpAPI == nil {
		return ""
	}
	if bound := d.httpAPI.BoundAddr(); bound != "" {
		return bound
	}
	return d.cfg.API.HTTPAddr
}

// Bus returns the daemon's events bus. Tests use it to inject
// synthetic events.
func (d *Daemon) Bus() *events.Bus { return d.bus }

// IQBroker returns the iqtap.Broker for the SDR with the given serial,
// or nil when no such SDR is in the pool. Used by the --iq-capture
// flag in main.go to attach a raw-IQ file writer to a live control
// SDR (issue #402 diagnostic).
func (d *Daemon) IQBroker(serial string) *iqtap.Broker { return d.iqBrokers[serial] }

// IQBrokerSerials returns the serials of every SDR currently wired
// through an iqtap.Broker. Used by --iq-capture to validate the
// requested serial and to print a friendly hint on a miss.
func (d *Daemon) IQBrokerSerials() []string {
	out := make([]string, 0, len(d.iqBrokers))
	for s := range d.iqBrokers {
		out = append(out, s)
	}
	return out
}

// spawn runs fn in a goroutine. essential=true means a non-context
// error from fn aborts the whole daemon (the launcher / TUI rely on
// these components — silently demoting their failures to log warnings
// produces a half-dead daemon that frustrates the operator).
// essential=false retains the legacy behaviour: warn-and-continue.
// ccDecoderRetryBackoffs is the per-attempt sleep schedule used after
// an IQ-stream death. After the schedule is exhausted, the daemon
// escalates to a fatal error so an external supervisor (systemd,
// docker, the launcher) restarts a clean process. See issue #345.
var ccDecoderRetryBackoffs = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	5 * time.Second,
	10 * time.Second,
}

// ccDecoderHealthyRunDuration is how long a successful Run must last
// before the retry counter resets. Anything shorter is treated as a
// repeated failure (e.g. the device immediately re-dies on reopen).
const ccDecoderHealthyRunDuration = 60 * time.Second

// runCCDecoderWithRetry drives the ccdecoder under a small restart
// loop. On ErrIQStreamClosed (the USB reaper died, the consumer
// channel closed — see issue #345) it sleeps with backoff, rebuilds
// the decoder against the same SDR, and retries. When the backoff
// schedule is exhausted it records a fatal error so the daemon's
// supervisor restarts a clean process. ctx cancel and any other Run
// error end the loop without escalation.
func (d *Daemon) runCCDecoderWithRetry(ctx context.Context) error {
	dec := d.ccDecoder
	var attempt int
	for {
		started := time.Now()
		err := dec.Run(ctx)
		if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if !errors.Is(err, ccdecoder.ErrIQStreamClosed) {
			return err
		}
		// A Run that survived a healthy window means the previous
		// recovery succeeded; reset the attempt counter so a fresh
		// failure stretch gets its own retry budget.
		if time.Since(started) >= ccDecoderHealthyRunDuration {
			attempt = 0
		}
		if attempt >= len(ccDecoderRetryBackoffs) {
			d.log.Error("daemon: ccdecoder: IQ stream died and retries exhausted; escalating to fatal",
				"attempts", attempt, "err", err)
			fatal := fmt.Errorf("ccdecoder: %w", err)
			d.recordFatal(fatal)
			return fatal
		}
		wait := ccDecoderRetryBackoffs[attempt]
		attempt++
		d.log.Warn("daemon: ccdecoder: IQ stream died; retrying",
			"attempt", attempt, "max_attempts", len(ccDecoderRetryBackoffs),
			"backoff", wait, "err", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
		// Attempt to re-acquire the control SDR by serial before
		// rebuilding the decoder. The retry's whole point is to
		// recover from a USB disconnect/re-enumerate; without
		// swapping in a freshly-opened Device, the rebuild will
		// stream against the same dead handle and fail again. If the
		// pool is missing (test scaffolding) or the serial wasn't
		// captured, skip re-acquire and rebuild against whatever the
		// captured opts hold — restores legacy behaviour for callers
		// that drive the loop with a logical IQSource.
		if d.pool != nil && d.controlSerial != "" {
			if newEntry, rerr := d.pool.Reacquire(d.controlSerial, d.controlSampleRate); rerr != nil {
				// Failure to re-acquire is *not* terminal — the
				// next backoff iteration retries. Log and fall
				// through to the rebuild attempt below; the
				// rebuilt decoder against the stale handle will
				// itself return ErrIQStreamClosed on the next
				// Run, queuing another retry.
				d.log.Warn("daemon: ccdecoder: control SDR reacquire failed; will retry",
					"serial", d.controlSerial, "err", rerr)
			} else {
				d.log.Info("daemon: ccdecoder: control SDR reacquired",
					"serial", d.controlSerial)
				// Keep the broker pointed at the fresh handle so
				// secondary subscribers (spectrum, paging, ...)
				// resume streaming on the next StreamIQ. If no
				// broker exists (pool wired without iqBrokers),
				// fall back to the raw new device.
				var iqSrc ccdecoder.IQSource = newEntry.Device
				var tuner ccdecoder.Tuner = newEntry.Device
				if br := d.iqBrokers[d.controlSerial]; br != nil {
					br.SetInner(newEntry.Device)
					iqSrc = br
					tuner = br
				}
				d.ccDecoderOpts.IQ = iqSrc
				d.ccDecoderOpts.Tuner = tuner
				if d.cchuntSup != nil {
					if serr := d.cchuntSup.SwapTuner(tuner); serr != nil {
						d.log.Warn("daemon: cchunt: SwapTuner failed",
							"err", serr)
					}
				}
			}
		}
		// Rebuild the decoder so it gets a fresh bus subscription —
		// Decoder.Run defers Close on its subscription, so reusing
		// the existing instance would see an immediately-closed
		// channel and exit on the first event.
		next, nerr := ccdecoder.New(d.ccDecoderOpts)
		if nerr != nil {
			d.log.Error("daemon: ccdecoder: rebuild failed; escalating to fatal", "err", nerr)
			fatal := fmt.Errorf("ccdecoder rebuild: %w", nerr)
			d.recordFatal(fatal)
			return fatal
		}
		dec = next
		d.mu.Lock()
		d.ccDecoder = dec
		d.mu.Unlock()
	}
}

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

// buildVirtualVoiceTuners spins up one wbvoice.VirtualTuner per voice tap
// on every `role: wideband` dongle. The taps subscribe to the dongle's
// iqtap broker on each StreamIQ call, run a single-tap DDC, and emit
// 48 kHz IQ to the composer. Out-of-window grants surface ErrOutOfBand
// and fall back to a physical voice SDR (when present) via the voice
// pool's bind retry.
//
// This must run before the voice pool (collectVoiceDevices) and the
// composer's virtualVoiceMap are built — otherwise a wideband-only
// topology yields an empty voice pool and every grant is dropped with
// "no voice SDR" (issue #422).
func (d *Daemon) buildVirtualVoiceTuners(cfg config.Config, log *slog.Logger) error {
	if d.pool == nil {
		return nil
	}
	for _, devCfg := range cfg.SDR.Devices {
		if devCfg.Role != "wideband" {
			continue
		}
		entry := d.pool.FindBySerial(devCfg.Serial)
		if entry == nil {
			// The wideband engine loop logs the missing-dongle warning;
			// stay quiet here to avoid duplicating it.
			continue
		}
		taps := devCfg.VoiceTaps
		if taps < 0 {
			taps = 0
		}
		br := d.iqBrokers[entry.Info.Serial]
		if br == nil && taps > 0 {
			log.Warn("daemon: wideband: voice_taps requested but no iqtap broker; virtual voice disabled",
				"serial", devCfg.Serial, "voice_taps", taps)
			taps = 0
		}
		for i := 0; i < taps; i++ {
			vt, err := wbvoice.New(wbvoice.Options{
				Serial:           fmt.Sprintf("wb:%s:tap-%d", entry.Info.Serial, i),
				Broker:           br,
				WidebandCenterHz: devCfg.CenterFreqHz,
				SDRSampleRateHz:  cfg.SDR.SampleRate,
				Log:              log,
			})
			if err != nil {
				return fmt.Errorf("daemon: wbvoice %q tap %d: %w", devCfg.Serial, i, err)
			}
			d.virtualVoiceTuners = append(d.virtualVoiceTuners, vt)
			log.Info("daemon: wideband: virtual voice tap registered",
				"wideband_serial", entry.Info.Serial,
				"tap_serial", vt.Serial())
		}
	}
	return nil
}

func (d *Daemon) collectVoiceDevices() []*trunking.VoiceDevice {
	var voices []*trunking.VoiceDevice
	if d.pool != nil {
		for _, e := range d.pool.AllByRole(sdr.RoleVoice) {
			voices = append(voices, &trunking.VoiceDevice{
				Tuner:  e.Device,
				Serial: e.Info.Serial,
			})
		}
	}
	// Append virtual voice tuners after physical ones so the engine's
	// FindFree prefers a physical SDR when both are free. Out-of-
	// window grants still surface ErrOutOfBand on virtual tuners and
	// fall through to the next free device via the engine's bind
	// retry — see engine.HandleGrant.
	for _, vt := range d.virtualVoiceTuners {
		voices = append(voices, &trunking.VoiceDevice{
			Tuner:  vt,
			Serial: vt.Serial(),
		})
	}
	return voices
}

// virtualVoiceMap builds the serial → IQSource map the composer's
// poolDevices uses to resolve a virtual tuner. Returns nil when no
// virtual tuners are configured so the lookup is a no-op.
func (d *Daemon) virtualVoiceMap() map[string]composer.IQSource {
	if len(d.virtualVoiceTuners) == 0 {
		return nil
	}
	out := make(map[string]composer.IQSource, len(d.virtualVoiceTuners))
	for _, vt := range d.virtualVoiceTuners {
		out[vt.Serial()] = vt
	}
	return out
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

// watchdogInterval maps the YAML knob (sdr.watchdog_interval_ms) onto
// the duration the SDR pool's USB-disconnect watchdog ticks at.
// Negative explicitly disables (returns 0, which the watchdog treats
// as "park on ctx and exit"). Zero (the default) selects the package
// default (sdr.DefaultWatchdogInterval).
func watchdogInterval(ms int) time.Duration {
	if ms < 0 {
		return 0
	}
	if ms == 0 {
		return sdr.DefaultWatchdogInterval
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

// WriteRawFrame fans raw IMBE / AMBE frames out to every contained
// sink that implements the rawFrameSink shape — currently just the
// recorder. Sinks that don't (tone-out, live player, audio publisher)
// are silently skipped. Without this the voice composer chains'
// `rs, _ := c.sink.(rawFrameSink)` type assertion fails against a
// fanoutSink, rs stays nil, and every IMBE / AMBE frame is dropped
// before reaching disk while the activity counter still bumps —
// producing healthy-looking call lifecycle logs alongside 0-byte
// .raw and 44-byte (header-only) .wav files. Issue #356 root cause.
func (f fanoutSink) WriteRawFrame(serial string, frame []byte) error {
	for _, s := range f {
		if rs, ok := s.(interface {
			WriteRawFrame(string, []byte) error
		}); ok {
			_ = rs.WriteRawFrame(serial, frame)
		}
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

// playerSink adapts *player.Player to composer.PCMSink. When engine is
// non-nil it also honours the per-talkgroup Mute flag: PCM for a call
// whose talkgroup is muted is dropped before reaching the speakers
// (the call is still recorded and streamed).
type playerSink struct {
	p      *player.Player
	engine *trunking.Engine
}

func (s playerSink) WritePCM(serial string, samples []int16) error {
	if s.engine != nil {
		if tg := s.engine.TalkgroupForDevice(serial); tg != nil && tg.Mute {
			return nil
		}
	}
	return s.p.WritePCM(serial, samples)
}

// broadcastStatus adapts the broadcast.Manager into the
// api.BroadcastStatusProvider interface, keeping the api package free
// of a compile-time dependency on internal/broadcast.
type broadcastStatus struct{ mgr *broadcast.Manager }

func (b broadcastStatus) BroadcastStats() any { return b.mgr.Stats() }

// affiliationProvider adapts the AffiliationTracker into the
// api.AffiliationProvider interface.
type affiliationProvider struct{ t *trunking.AffiliationTracker }

func (a affiliationProvider) Affiliations() []trunking.UnitActivity { return a.t.Snapshot() }

// wrapBasebandRecorders replaces the Device of every pool entry whose
// serial appears in baseband.record with a RecordingDevice, teeing its
// live IQ to a WAV recording. Runs once at construction, before any
// streaming starts, so mutating the entry's Device pointer is safe.
func (d *Daemon) wrapBasebandRecorders(cfg config.Config, log *slog.Logger) {
	if len(cfg.Baseband.Record) == 0 || d.pool == nil {
		return
	}
	dirBySerial := make(map[string]string, len(cfg.Baseband.Record))
	for _, r := range cfg.Baseband.Record {
		dirBySerial[r.Serial] = r.Dir
	}
	rate := cfg.SDR.SampleRate
	if rate == 0 {
		rate = sdr.DefaultSampleRateHz
	}
	for _, e := range d.pool.Entries() {
		dir, ok := dirBySerial[e.Info.Serial]
		if !ok {
			continue
		}
		rec := baseband.NewRecordingDevice(e.Device, dir, log)
		_ = rec.SetSampleRate(rate)
		e.Device = rec
		log.Info("baseband recording enabled", "serial", e.Info.Serial, "dir", dir)
	}
}

// wrapIQBrokers creates one iqtap.Broker per pool entry, keyed by
// serial, wrapping whatever entry.Device currently points at (the raw
// driver Device, or a baseband.RecordingDevice if wrapBasebandRecorders
// already wrapped it). Brokers live in d.iqBrokers as a parallel map
// to the pool — entry.Device itself is left untouched so the existing
// Pool.Reacquire contract (which mutates entry.Device in place) keeps
// working unchanged. Primary consumers that want fan-out wire
// d.iqBrokers[serial] in as their IQSource + Tuner; the broker
// forwards every Device method to its inner.
//
// After a successful pool.Reacquire, the daemon must call
// broker.SetInner(newEntry.Device) so the next StreamIQ session
// streams from the fresh handle. Note: Reacquire returns the raw new
// device, not a RecordingDevice — baseband recording stops across a
// USB-disconnect cycle, which is an existing behavior the broker
// inherits rather than fixes.
func (d *Daemon) wrapIQBrokers(cfg config.Config, log *slog.Logger) {
	if d.pool == nil {
		return
	}
	rate := cfg.SDR.SampleRate
	if rate == 0 {
		rate = sdr.DefaultSampleRateHz
	}
	d.iqBrokers = make(map[string]*iqtap.Broker, len(d.pool.Entries()))
	for _, e := range d.pool.Entries() {
		br := iqtap.New(e.Device, 0, log)
		// pool.Open already programmed cfg.SDR.SampleRate on the raw
		// device before we wrapped it, so Broker.SetSampleRate's cache
		// path never ran. Seed it directly so spectrum frame stamps
		// (and any other SampleRateHz reader) anchor at the right rate
		// from the first frame instead of 0.
		br.Seed(0, rate)
		d.iqBrokers[e.Info.Serial] = br
	}
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

// poolDevices adapts *sdr.Pool to composer.Devices. The composer
// needs StreamIQ + SampleRateHz; sdr.Device only satisfies the
// former, so physical entries are wrapped in deviceWithRate that
// reports the daemon-wide rate. Virtual voice tuners (wideband-
// derived) already implement both interface methods directly and
// take precedence — that's how a P25 grant on the same SDR as the
// wideband CC tap gets followed without retuning a physical voice
// SDR.
type poolDevices struct {
	pool       *sdr.Pool
	rateHz     uint32
	virtualMap map[string]composer.IQSource
}

func (p *poolDevices) FindBySerial(serial string) composer.IQSource {
	if src, ok := p.virtualMap[serial]; ok {
		return src
	}
	if p.pool == nil {
		return nil
	}
	e := p.pool.FindBySerial(serial)
	if e == nil {
		return nil
	}
	return deviceWithRate{Device: e.Device, rate: p.rateHz}
}

// deviceWithRate makes an sdr.Device satisfy composer.IQSource by
// stamping the daemon-wide configured sample rate onto it. Physical
// SDRs share one rate (cfg.SDR.SampleRate); the composer uses it
// to size each call's decimator.
type deviceWithRate struct {
	sdr.Device
	rate uint32
}

func (d deviceWithRate) SampleRateHz() uint32 { return d.rate }

// brokerRigController adapts an iqtap.Broker into the rigctld
// Controller interface. Routing through the broker means SetFreq
// goes through the same path the ccdecoder uses (so the broker's
// CenterHz / SampleRateHz stay in sync) AND survives pool.Reacquire
// because the broker keeps its inner pointer.
type brokerRigController struct {
	serial string
	broker *iqtap.Broker
}

func (b brokerRigController) Serial() string { return b.serial }
func (b brokerRigController) Freq() (uint32, error) {
	return b.broker.CenterHz(), nil
}
func (b brokerRigController) SetFreq(hz uint32) error {
	return b.broker.SetCenterFreq(hz)
}

// poolRigController is the fallback for tests / scaffolding paths
// where the broker isn't wired. Talks directly to an sdr.Device. The
// device interface doesn't expose a CenterFreq getter, so Freq
// returns the last SetFreq value cached locally (0 before any set).
type poolRigController struct {
	serial string
	dev    sdr.Device

	mu sync.Mutex
	hz uint32
}

func (p *poolRigController) Serial() string { return p.serial }
func (p *poolRigController) Freq() (uint32, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.hz, nil
}
func (p *poolRigController) SetFreq(hz uint32) error {
	if err := p.dev.SetCenterFreq(hz); err != nil {
		return err
	}
	p.mu.Lock()
	p.hz = hz
	p.mu.Unlock()
	return nil
}

// bookmarkProvider adapts the storage.BookmarkStore into the
// api.BookmarkProvider interface, attaching a fresh request-scoped
// context to each call. The handlers don't carry a context all the
// way through today (the existing patterns use background-scoped
// queries with their own timeouts); a 5-second cap keeps a wedged
// DB write from pinning a handler forever.
type bookmarkProvider struct{ store *storage.BookmarkStore }

func (p bookmarkProvider) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func (p bookmarkProvider) ListBookmarks() ([]storage.Bookmark, error) {
	ctx, cancel := p.ctx()
	defer cancel()
	return p.store.List(ctx)
}

func (p bookmarkProvider) GetBookmark(id int64) (storage.Bookmark, error) {
	ctx, cancel := p.ctx()
	defer cancel()
	return p.store.Get(ctx, id)
}

func (p bookmarkProvider) CreateBookmark(b storage.Bookmark) (storage.Bookmark, error) {
	ctx, cancel := p.ctx()
	defer cancel()
	return p.store.Create(ctx, b)
}

func (p bookmarkProvider) UpdateBookmark(b storage.Bookmark) (storage.Bookmark, error) {
	ctx, cancel := p.ctx()
	defer cancel()
	return p.store.Update(ctx, b)
}

func (p bookmarkProvider) DeleteBookmark(id int64) error {
	ctx, cancel := p.ctx()
	defer cancel()
	return p.store.Delete(ctx, id)
}

// pagerProvider adapts storage.PagerLog into the api.PagerProvider
// interface so the api package can stay free of the storage import
// dependency. Read-only — the decoder writes via the events bus.
type pagerProvider struct{ log *storage.PagerLog }

func (p pagerProvider) RecentPagerMessages(limit int) ([]storage.PagerMessage, error) {
	return p.log.Recent(limit)
}

// pocsagSpec captures the broker-side wiring info for one configured
// POCSAG paging channel. Index-aligned with Daemon.pocsagReceivers so
// the Run loop can spawn each receiver without re-walking the YAML.
type pocsagSpec struct {
	serial string
	freq   uint32
}

// aprsProvider adapts storage.APRSLog into api.APRSProvider so the
// api package stays free of the storage import dependency. Read-only
// — the decoder writes via the events bus.
type aprsProvider struct{ log *storage.APRSLog }

func (a aprsProvider) RecentAPRSPackets(limit int) ([]storage.APRSPacket, error) {
	return a.log.Recent(limit)
}

// aisProvider adapts storage.VesselLog into api.AISProvider so the
// api package stays free of the storage import dependency. Read-only
// — the decoder writes via the events bus.
type aisProvider struct{ log *storage.VesselLog }

func (a aisProvider) RecentAISMessages(limit int) ([]storage.AISMessage, error) {
	return a.log.Recent(limit)
}

// dscProvider adapts storage.DSCLog into api.DSCProvider so the
// api package stays free of the storage import dependency. Read-only
// — the decoder writes via the events bus.
type dscProvider struct{ log *storage.DSCLog }

func (d dscProvider) RecentDSCMessages(limit int) ([]storage.DSCMessage, error) {
	return d.log.Recent(limit)
}

// adsbProvider adapts storage.AircraftLog into api.ADSBProvider so
// the api package stays free of the storage import dependency.
// Read-only — the decoder writes via the events bus.
type adsbProvider struct{ log *storage.AircraftLog }

func (a adsbProvider) RecentAircraftReports(limit int) ([]storage.AircraftReport, error) {
	return a.log.Recent(limit)
}

// mdc1200Provider adapts storage.MDC1200Log into api.MDC1200Provider
// so the api package stays free of the storage import dependency.
// Read-only — the decoder writes via the events bus.
type mdc1200Provider struct{ log *storage.MDC1200Log }

func (m mdc1200Provider) RecentMDC1200Messages(limit int) ([]storage.MDC1200Message, error) {
	return m.log.Recent(limit)
}

// aprsSpec captures the broker-side wiring info for one configured
// APRS channel. Index-aligned with Daemon.aprsReceivers so the Run
// loop can spawn each receiver without re-walking the YAML. Mirrors
// pocsagSpec.
type aprsSpec struct {
	serial string
	freq   uint32
}

// aisSpec captures the broker-side wiring info for one configured
// AIS channel. Index-aligned with Daemon.aisReceivers so the Run
// loop can spawn each receiver without re-walking the YAML. Mirrors
// aprsSpec / pocsagSpec.
type aisSpec struct {
	serial string
	freq   uint32
}

// dscSpec captures the broker-side wiring info for one configured DSC
// channel. Index-aligned with Daemon.dscReceivers so the Run loop can
// spawn each receiver without re-walking the YAML. Mirrors aisSpec /
// mdc1200Spec.
type dscSpec struct {
	serial string
	freq   uint32
}

// adsbPPMSpec captures the broker-side wiring info for one configured
// native ADS-B PPM channel. Index-aligned with Daemon.adsbPPMReceivers
// so the Run loop can spawn each receiver without re-walking the YAML.
type adsbPPMSpec struct {
	serial string
	freq   uint32
}

// mdc1200Spec captures the broker-side wiring info for one configured
// MDC1200 channel. Index-aligned with Daemon.mdc1200Receivers so the
// Run loop can spawn each receiver without re-walking the YAML.
// Mirrors aprsSpec / aisSpec.
type mdc1200Spec struct {
	serial string
	freq   uint32
}

// loadRIDFile dispatches a per-system rid_alias_file load to the JSON
// or CSV reader based on extension. JSON if the path ends in ".json"
// (case-insensitive), CSV otherwise — matches the talkgroup loader's
// expectation that the operator already knows the file format.
func loadRIDFile(db *trunking.RIDDB, path string) (int, error) {
	if strings.EqualFold(filepath.Ext(path), ".json") {
		return db.LoadJSONFile(path)
	}
	return db.LoadCSVFile(path)
}
