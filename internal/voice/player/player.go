// Package player is the live-audio sink that turns int16 PCM coming
// out of the per-call composer / conventional scanner into sound out
// of the host's speakers. It plugs into the same composer.PCMSink /
// conventional.Recorder fan-out point the per-call WAV recorder uses,
// so wiring is a one-line change in cmd/gophertrunk/daemon.go.
//
// The default backend is github.com/ebitengine/oto/v3 — routes to
// ALSA on Linux (cgo + libasound2-dev at build time, libasound2 at
// runtime), CoreAudio on macOS, WASAPI on Windows. When audio.enabled
// is false in config, when the backend fails to initialise on a
// headless box, or when audio.device is "null", the daemon silently
// uses a no-op player and the rest of the system runs identically.
//
// Volume and mute are applied at WritePCM time as a software gain
// stage so changes are instant and don't require reopening the
// device.
package player

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// Config controls how the live-audio player initialises.
type Config struct {
	// Enabled gates the real backend. False (or any backend
	// failure) leaves the player as a no-op so headless servers
	// keep working.
	Enabled bool
	// Device is the backend-specific output device name; empty
	// (or "default") = system default. "null" forces the no-op
	// backend even when Enabled=true (useful for tests).
	Device string
	// SampleRate is the host playback rate in Hz. Must match the
	// rate the composer feeds the recorder (typically 8000).
	SampleRate uint32
	// BufferMs is the depth of the playback queue. Larger =
	// more resilient to scheduling jitter, smaller = lower
	// latency. Default 80 ms.
	BufferMs int
	// Volume is the initial software gain (0..1). Default 0.8.
	Volume float32
	// Muted is the initial mute state.
	Muted bool
	// DisableAutoFallback turns OFF the default Linux behaviour of
	// transparently dropping to the direct-ioctl backend when
	// libasound2.so.2 fails to dlopen (typical on distroless /
	// Alpine images). Zero value (false) = auto-fallback enabled.
	// Other platforms ignore the flag.
	DisableAutoFallback bool
}

// Backend is the per-OS audio sink the Player drives. Production
// uses an oto-backed implementation; tests use a loopback that
// captures whatever the player writes.
type Backend interface {
	// Write blocks until the supplied PCM has been queued to the
	// device. samples are 16-bit LE PCM at the configured rate.
	Write(samples []int16) error
	Close() error
}

// Player mixes PCM from one-or-more call serials into a single
// output stream. WritePCM matches composer.PCMSink so the daemon
// can drop the Player into the existing fan-out alongside the
// Recorder.
type Player struct {
	log *slog.Logger

	backend    Backend
	sampleRate uint32

	// Software gain stage. Stored as the float32 bit pattern in a
	// uint32 atomic so SetVolume is lock-free.
	volume atomic.Uint32 // math.Float32bits
	muted  atomic.Bool

	// Single bounded PCM queue. WritePCM does a non-blocking
	// send and drops on full so the composer goroutine never
	// blocks on a slow audio device.
	queue chan []int16

	// drops counts samples lost to a full queue. Surfaced via
	// Stats for the API / TUI.
	drops atomic.Uint64

	stopOnce sync.Once
	stop     chan struct{}
	done     chan struct{}
}

// defaultBackendFactory is the production backend constructor. Set
// by the oto-backed init in player_oto.go. Left nil in builds where
// no backend has been linked in.
var defaultBackendFactory func(cfg Config) (Backend, error)

// New constructs a Player. When cfg.Enabled is false, the device
// is "null", or the backend factory fails, the returned Player is
// a no-op (writes are accepted and dropped) so the daemon keeps
// running on headless boxes.
func New(cfg Config, log *slog.Logger) (*Player, error) {
	if log == nil {
		return nil, errors.New("player: logger is required")
	}
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 8000
	}
	if cfg.BufferMs <= 0 {
		cfg.BufferMs = 80
	}
	if cfg.Volume <= 0 {
		cfg.Volume = 0.8
	}
	if cfg.Volume > 1 {
		cfg.Volume = 1
	}

	p := &Player{
		log:        log,
		sampleRate: cfg.SampleRate,
		queue:      make(chan []int16, 32),
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
	}
	p.setVolume(cfg.Volume)
	p.muted.Store(cfg.Muted)

	if !cfg.Enabled || cfg.Device == "null" || defaultBackendFactory == nil {
		log.Info("player: live audio disabled", "enabled", cfg.Enabled, "device", cfg.Device)
		close(p.done)
		return p, nil
	}

	be, err := defaultBackendFactory(cfg)
	if err != nil {
		log.Warn("player: backend init failed; running headless", "err", err)
		close(p.done)
		return p, nil
	}
	p.backend = be
	go p.run()
	return p, nil
}

// NewWithBackend constructs a Player wrapping an explicit backend.
// Used by tests with a loopback backend.
func NewWithBackend(be Backend, cfg Config, log *slog.Logger) (*Player, error) {
	if log == nil {
		return nil, errors.New("player: logger is required")
	}
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 8000
	}
	if cfg.Volume <= 0 {
		cfg.Volume = 0.8
	}
	if cfg.Volume > 1 {
		cfg.Volume = 1
	}
	p := &Player{
		log:        log,
		sampleRate: cfg.SampleRate,
		queue:      make(chan []int16, 32),
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
		backend:    be,
	}
	p.setVolume(cfg.Volume)
	p.muted.Store(cfg.Muted)
	go p.run()
	return p, nil
}

// WritePCM is the composer / conventional-scanner side of the fan-out.
// Matches the composer.PCMSink and conventional.Recorder interfaces
// so the daemon can drop the player straight into the existing
// fanoutSink without any code changes downstream.
//
// The deviceSerial argument is the same routing key the recorder uses
// (one in-flight call per voice SDR); the player doesn't tee per
// device because the host has one output stream, but it's kept on
// the signature so the interface stays compatible.
func (p *Player) WritePCM(deviceSerial string, samples []int16) error {
	if p == nil || p.backend == nil || len(samples) == 0 {
		return nil
	}
	if p.muted.Load() {
		return nil
	}
	buf := make([]int16, len(samples))
	gain := p.getVolume()
	if gain != 1.0 {
		for i, s := range samples {
			v := float32(s) * gain
			if v > 32767 {
				v = 32767
			} else if v < -32768 {
				v = -32768
			}
			buf[i] = int16(v)
		}
	} else {
		copy(buf, samples)
	}

	select {
	case p.queue <- buf:
	default:
		p.drops.Add(uint64(len(buf)))
	}
	return nil
}

// SetVolume sets the software gain in 0..1.
func (p *Player) SetVolume(v float32) {
	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}
	p.setVolume(v)
}

// Volume returns the current software gain.
func (p *Player) Volume() float32 { return p.getVolume() }

// SetMuted toggles the mute state.
func (p *Player) SetMuted(m bool) { p.muted.Store(m) }

// Muted returns true if currently muted.
func (p *Player) Muted() bool { return p.muted.Load() }

// Stats reports a snapshot of player counters for the API.
type Stats struct {
	Enabled    bool
	SampleRate uint32
	Volume     float32
	Muted      bool
	DropsTotal uint64
}

func (p *Player) Stats() Stats {
	return Stats{
		Enabled:    p.backend != nil,
		SampleRate: p.sampleRate,
		Volume:     p.getVolume(),
		Muted:      p.muted.Load(),
		DropsTotal: p.drops.Load(),
	}
}

// Close stops the player and releases the backend.
func (p *Player) Close() error {
	if p == nil {
		return nil
	}
	var closeErr error
	p.stopOnce.Do(func() {
		close(p.stop)
		<-p.done
		if p.backend != nil {
			closeErr = p.backend.Close()
		}
	})
	return closeErr
}

func (p *Player) setVolume(v float32) {
	p.volume.Store(math.Float32bits(v))
}

func (p *Player) getVolume() float32 {
	return math.Float32frombits(p.volume.Load())
}

func (p *Player) run() {
	defer close(p.done)
	idle := time.NewTicker(20 * time.Millisecond)
	defer idle.Stop()
	for {
		select {
		case <-p.stop:
			return
		case buf := <-p.queue:
			if err := p.backend.Write(buf); err != nil {
				p.log.Warn("player: backend write failed", "err", err)
				return
			}
		case <-idle.C:
			// no-op; keeps the goroutine responsive to stop even if
			// the queue stays empty for long stretches between calls.
		}
	}
}

// ListDevices returns the audio outputs the linked backend can route
// to. On Linux the listing includes "default" (system sink via
// libasound2 or the ioctl fallback), "null" (no-op backend), plus an
// entry per /dev/snd/pcmC*D*p the kernel exposes — the latter are
// what audio.device: "ioctl:hw:C,D" selects. On macOS / Windows the
// list is short by design (oto routes to the OS default sink).
func ListDevices() []string {
	devs := []string{"default (system output)", "null (no-op)"}
	devs = append(devs, listPlatformDevices()...)
	return devs
}

// String returns a human-readable description of the player state,
// suitable for logging and the TUI status strip.
func (p *Player) String() string {
	if p == nil {
		return "player(nil)"
	}
	return fmt.Sprintf("player(enabled=%t volume=%.2f muted=%t)",
		p.backend != nil, p.getVolume(), p.muted.Load())
}
