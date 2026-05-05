package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/api"
	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/metrics"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice"
	"github.com/MattCheramie/GopherTrunk/internal/voice/composer"
)

// Daemon owns the lifecycle of every long-lived component the
// gophertrunk binary runs: the events bus, SDR pool, trunking engine,
// per-call recorder, SQLite call log + retention sweeper, Prometheus
// collector, and HTTP/gRPC API servers.
//
// Components are constructed in dependency order in NewDaemon, started
// by Run, and torn down by Close in reverse order. Run blocks until
// ctx cancels; Close is idempotent.
type Daemon struct {
	cfg config.Config
	log *slog.Logger

	bus        *events.Bus
	pool       *sdr.Pool
	talkgroups *trunking.TalkgroupDB
	systems    []trunking.System
	engine     *trunking.Engine
	voicePool  *trunking.VoicePool
	recorder   *voice.Recorder
	composer   *composer.Composer
	db         *storage.DB
	callLog    *storage.CallLog
	retention  *storage.Retention
	metrics    *metrics.Metrics
	httpAPI    *api.Server
	grpcAPI    *api.GRPCServer

	wg        sync.WaitGroup
	closeOnce sync.Once
}

// NewDaemon constructs the daemon from the loaded config. Components
// that are disabled by config (no API addr, no storage path, etc.) are
// simply not constructed and the daemon proceeds with the rest.
//
// version is stamped into Prometheus build_info and the API
// /api/v1/version response.
func NewDaemon(cfg config.Config, version string, log *slog.Logger) (*Daemon, error) {
	if log == nil {
		return nil, errors.New("daemon: logger is required")
	}

	d := &Daemon{cfg: cfg, log: log, bus: events.NewBus(64)}

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
			Name:            sys.Name,
			Protocol:        proto,
			ControlChannels: sys.ControlChannels,
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
			} else {
				log.Info("daemon: talkgroups loaded", "system", sys.Name, "count", n)
			}
		}
	}

	// SDR pool — best-effort. Components that need a Voice device
	// fall through gracefully when the pool is empty.
	if len(cfg.SDR.Devices) > 0 {
		d.pool = sdr.NewPool(log)
		var hints []sdr.Hint
		for _, dev := range cfg.SDR.Devices {
			hints = append(hints, sdr.Hint{Serial: dev.Serial, Role: sdr.ParseRole(dev.Role)})
		}
		if err := d.pool.Open(hints); err != nil {
			log.Warn("daemon: SDR pool open failed", "err", err)
			d.pool = nil
		}
	}

	// Voice device list from the pool; empty when no SDRs.
	d.voicePool = trunking.NewVoicePool(d.collectVoiceDevices())

	engine, err := trunking.NewEngine(trunking.EngineOptions{
		Bus:        d.bus,
		Log:        log,
		VoicePool:  d.voicePool,
		Talkgroups: d.talkgroups,
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

	// Composer is wired when we have a Voice device pool to source IQ
	// from + a recorder to feed PCM into. Without an SDR pool there's
	// nothing to demod, and without a recorder PCM has no destination.
	if d.pool != nil && d.recorder != nil {
		comp, err := composer.New(composer.Options{
			Bus:           d.bus,
			Devices:       &poolDevices{pool: d.pool},
			Sink:          d.recorder,
			Engine:        d.engine,
			Log:           log,
			IQSampleRate:  cfg.SDR.SampleRate,
			PCMSampleRate: cfg.Recordings.SampleRate,
		})
		if err != nil {
			return nil, fmt.Errorf("daemon: composer: %w", err)
		}
		d.composer = comp
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
		opts := api.ServerOptions{
			Addr:       cfg.API.HTTPAddr,
			Bus:        d.bus,
			Engine:     d.engine,
			Talkgroups: d.talkgroups,
			Systems:    d.systems,
			Log:        log,
			Version:    version,
		}
		if d.db != nil {
			opts.History = api.HistoryFromStorage(d.db)
		}
		if d.metrics != nil {
			opts.MetricsHandler = d.metrics.Handler()
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
			Log:        log,
		})
		if err != nil {
			return nil, fmt.Errorf("daemon: grpc api: %w", err)
		}
		d.grpcAPI = gsrv
	}

	return d, nil
}

// Run starts every long-lived goroutine and blocks until ctx cancels.
// Each component returns its own error; the first non-nil non-context
// error short-circuits the rest and triggers Close.
func (d *Daemon) Run(ctx context.Context) error {
	d.log.Info("gophertrunk starting",
		"http_addr", d.cfg.API.HTTPAddr,
		"grpc_addr", d.cfg.API.GRPCAddr,
		"systems", len(d.systems),
		"voice_devices", len(d.voicePool.Devices()),
	)

	d.spawn(ctx, "engine", func(ctx context.Context) error {
		return d.engine.Run(ctx)
	})

	if d.callLog != nil {
		d.spawn(ctx, "calllog", func(ctx context.Context) error {
			return d.callLog.Run(ctx)
		})
	}
	if d.retention != nil {
		d.spawn(ctx, "retention", func(ctx context.Context) error {
			return d.retention.Run(ctx)
		})
	}
	if d.recorder != nil {
		d.spawn(ctx, "recorder", func(ctx context.Context) error {
			return d.recorder.Run(ctx)
		})
	}
	if d.composer != nil {
		d.spawn(ctx, "composer", func(ctx context.Context) error {
			return d.composer.Run(ctx)
		})
	}
	if d.metrics != nil {
		d.spawn(ctx, "metrics", func(ctx context.Context) error {
			return d.metrics.Run(ctx)
		})
	}
	if d.httpAPI != nil {
		d.spawn(ctx, "http", func(ctx context.Context) error {
			return d.httpAPI.Run(ctx)
		})
	}
	if d.grpcAPI != nil {
		d.spawn(ctx, "grpc", func(ctx context.Context) error {
			return d.grpcAPI.Run(ctx)
		})
	}

	<-ctx.Done()
	d.log.Info("gophertrunk shutdown initiated")
	d.Close()
	d.wg.Wait()
	d.log.Info("gophertrunk shutdown complete")
	return ctx.Err()
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
		if d.composer != nil {
			_ = d.composer.Close()
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

func (d *Daemon) spawn(ctx context.Context, name string, fn func(context.Context) error) {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		if err := fn(ctx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			d.log.Warn("daemon: component exited with error", "component", name, "err", err)
		}
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
