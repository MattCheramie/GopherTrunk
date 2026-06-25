package log

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// PowerLog writes a per-channel IQ-signal-level log gated on decode
// activity. It subscribes to the events bus and appends one timestamped
// line per KindChannelPower event — the wideband engine
// (internal/scanner/widebandt2) emits one such event per diagnostics
// window for each channel that decoded at least one frame that window, so
// idle / off-band channels never appear. By default only low-power
// windows are written (the "decoding but signal is weak" diagnostic the
// branch is named for); AllWindows opts into a full per-window power
// time-series. The file rotates to "<path>.1" when it exceeds the
// configured size cap, mirroring MessageLog.
type PowerLog struct {
	bus        *events.Bus
	path       string
	maxBytes   int64
	allWindows bool
	loc        *time.Location
	sub        *events.Subscription
	runDone    chan struct{}
	closeOnce  sync.Once

	mu   sync.Mutex
	f    *os.File
	size int64
}

// PowerLogOptions configure a PowerLog.
type PowerLogOptions struct {
	Bus *events.Bus
	// Path is the log file path. Required.
	Path string
	// MaxSizeMB caps the file size before rotation. Default 16.
	MaxSizeMB int
	// AllWindows, when true, logs every decode-active window. When false
	// (default) only windows whose IQ power is below the low-power
	// threshold are written.
	AllWindows bool
	// Loc is the timezone displayed timestamps are rendered in. Nil means
	// time.Local (the daemon passes cfg.Display.Location()).
	Loc *time.Location
}

// NewPowerLog opens the log file and subscribes to the bus.
func NewPowerLog(opts PowerLogOptions) (*PowerLog, error) {
	if opts.Bus == nil {
		return nil, errors.New("log/powerlog: events.Bus is required")
	}
	if opts.Path == "" {
		return nil, errors.New("log/powerlog: Path is required")
	}
	if opts.MaxSizeMB <= 0 {
		opts.MaxSizeMB = 16
	}
	if err := os.MkdirAll(filepath.Dir(opts.Path), 0o755); err != nil {
		return nil, fmt.Errorf("log/powerlog: mkdir: %w", err)
	}
	f, err := os.OpenFile(opts.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("log/powerlog: open: %w", err)
	}
	loc := opts.Loc
	if loc == nil {
		loc = time.Local
	}
	info, _ := f.Stat()
	pl := &PowerLog{
		bus:        opts.Bus,
		path:       opts.Path,
		maxBytes:   int64(opts.MaxSizeMB) * 1024 * 1024,
		allWindows: opts.AllWindows,
		loc:        loc,
		runDone:    make(chan struct{}),
		f:          f,
	}
	if info != nil {
		pl.size = info.Size()
	}
	pl.sub = opts.Bus.Subscribe()
	return pl, nil
}

// Run drains events until ctx cancels or the bus closes.
func (p *PowerLog) Run(ctx context.Context) error {
	defer close(p.runDone)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-p.sub.C:
			if !ok {
				return nil
			}
			if ev.Kind != events.KindChannelPower {
				continue
			}
			cp, ok := ev.Payload.(events.ChannelPower)
			if !ok {
				continue
			}
			if !p.allWindows && !cp.LowPower {
				continue
			}
			p.write(formatChannelPower(ev.Timestamp, cp, p.loc))
		}
	}
}

func (p *PowerLog) write(line string) {
	if line == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.f == nil {
		return
	}
	if p.maxBytes > 0 && p.size+int64(len(line)) > p.maxBytes {
		p.rotate()
	}
	n, err := p.f.WriteString(line)
	p.size += int64(n)
	_ = err
}

// rotate closes the current file, renames it to "<path>.1", and opens a
// fresh one. Caller holds p.mu.
func (p *PowerLog) rotate() {
	p.f.Close()
	_ = os.Rename(p.path, p.path+".1")
	f, err := os.OpenFile(p.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		p.f = nil
		return
	}
	p.f = f
	p.size = 0
}

// Close releases the bus subscription, waits for Run to drain, and closes
// the file.
func (p *PowerLog) Close() error {
	p.closeOnce.Do(func() {
		p.sub.Close()
		select {
		case <-p.runDone:
		case <-time.After(time.Second):
		}
		p.mu.Lock()
		if p.f != nil {
			p.f.Close()
			p.f = nil
		}
		p.mu.Unlock()
	})
	return nil
}

// formatChannelPower renders one ChannelPower event as a power-log line
// (newline-terminated).
func formatChannelPower(ts time.Time, cp events.ChannelPower, loc *time.Location) string {
	stamp := stampTime(ts, loc)
	return fmt.Sprintf("%s %-12s system=%s proto=%s freq=%d dbfs=%.1f decoded=%d low=%t\n",
		stamp, "POWER", cp.System, cp.Protocol, cp.FreqHz, cp.PowerDbFS, cp.Decoded, cp.LowPower)
}
