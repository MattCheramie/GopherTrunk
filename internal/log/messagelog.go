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
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// MessageLog writes a human-readable, per-event decoded-message log —
// the GopherTrunk analogue of SDRtrunk's per-channel decoded message
// log. It subscribes to the events bus and appends one timestamped
// line per trunking event (grants, control-channel lock/loss,
// affiliations, registrations, patches, talker aliases, locations,
// tone alerts, decode errors). The file rotates to "<path>.1" when it
// exceeds the configured size cap.
type MessageLog struct {
	bus       *events.Bus
	path      string
	maxBytes  int64
	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once

	mu   sync.Mutex
	f    *os.File
	size int64
}

// MessageLogOptions configure a MessageLog.
type MessageLogOptions struct {
	Bus *events.Bus
	// Path is the log file path. Required.
	Path string
	// MaxSizeMB caps the file size before rotation. Default 16.
	MaxSizeMB int
}

// NewMessageLog opens the log file and subscribes to the bus.
func NewMessageLog(opts MessageLogOptions) (*MessageLog, error) {
	if opts.Bus == nil {
		return nil, errors.New("log/messagelog: events.Bus is required")
	}
	if opts.Path == "" {
		return nil, errors.New("log/messagelog: Path is required")
	}
	if opts.MaxSizeMB <= 0 {
		opts.MaxSizeMB = 16
	}
	if err := os.MkdirAll(filepath.Dir(opts.Path), 0o755); err != nil {
		return nil, fmt.Errorf("log/messagelog: mkdir: %w", err)
	}
	f, err := os.OpenFile(opts.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("log/messagelog: open: %w", err)
	}
	info, _ := f.Stat()
	ml := &MessageLog{
		bus:      opts.Bus,
		path:     opts.Path,
		maxBytes: int64(opts.MaxSizeMB) * 1024 * 1024,
		runDone:  make(chan struct{}),
		f:        f,
	}
	if info != nil {
		ml.size = info.Size()
	}
	ml.sub = opts.Bus.Subscribe()
	return ml, nil
}

// Run drains events until ctx cancels or the bus closes.
func (m *MessageLog) Run(ctx context.Context) error {
	defer close(m.runDone)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-m.sub.C:
			if !ok {
				return nil
			}
			if line := formatEvent(ev); line != "" {
				m.write(line)
			}
		}
	}
}

func (m *MessageLog) write(line string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.f == nil {
		return
	}
	if m.maxBytes > 0 && m.size+int64(len(line)) > m.maxBytes {
		m.rotate()
	}
	n, err := m.f.WriteString(line)
	m.size += int64(n)
	_ = err
}

// rotate closes the current file, renames it to "<path>.1", and opens
// a fresh one. Caller holds m.mu.
func (m *MessageLog) rotate() {
	m.f.Close()
	_ = os.Rename(m.path, m.path+".1")
	f, err := os.OpenFile(m.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		m.f = nil
		return
	}
	m.f = f
	m.size = 0
}

// Close releases the bus subscription, waits for Run to drain, and
// closes the file.
func (m *MessageLog) Close() error {
	m.closeOnce.Do(func() {
		m.sub.Close()
		select {
		case <-m.runDone:
		case <-time.After(time.Second):
		}
		m.mu.Lock()
		if m.f != nil {
			m.f.Close()
			m.f = nil
		}
		m.mu.Unlock()
	})
	return nil
}

// formatEvent renders one bus event as a decoded-message log line
// (newline-terminated). Returns "" for events with no useful textual
// form so they are skipped.
func formatEvent(ev events.Event) string {
	ts := ev.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	stamp := ts.UTC().Format("2006-01-02T15:04:05.000Z")
	var body string
	switch ev.Kind {
	case events.KindGrant:
		if g, ok := ev.Payload.(trunking.Grant); ok {
			body = fmt.Sprintf("%-12s system=%s proto=%s tg=%d src=%d freq=%d enc=%t emer=%t",
				"GRANT", g.System, g.Protocol, g.GroupID, g.SourceID,
				g.FrequencyHz, g.Encrypted, g.Emergency)
		}
	case events.KindCallStart:
		if c, ok := ev.Payload.(trunking.CallStart); ok {
			body = fmt.Sprintf("%-12s system=%s tg=%d src=%d dev=%s",
				"CALL-START", c.Grant.System, c.Grant.GroupID, c.Grant.SourceID, c.DeviceSerial)
		}
	case events.KindCallEnd:
		if c, ok := ev.Payload.(trunking.CallEnd); ok {
			body = fmt.Sprintf("%-12s system=%s tg=%d dur=%s reason=%s",
				"CALL-END", c.Grant.System, c.Grant.GroupID,
				c.Duration().Round(time.Millisecond), c.Reason)
		}
	case events.KindAffiliation:
		if a, ok := ev.Payload.(trunking.Affiliation); ok {
			body = fmt.Sprintf("%-12s system=%s proto=%s src=%d tg=%d resp=%s",
				"AFFILIATION", a.System, a.Protocol, a.SourceID, a.GroupID, a.Response)
		}
	case events.KindUnitRegistration:
		if r, ok := ev.Payload.(trunking.UnitRegistration); ok {
			body = fmt.Sprintf("%-12s system=%s proto=%s src=%d wacn=%d sysid=%d resp=%s",
				"REGISTRATION", r.System, r.Protocol, r.SourceID, r.WACN, r.SystemID, r.Response)
		}
	case events.KindPatch:
		body = fmt.Sprintf("%-12s %+v", "PATCH", ev.Payload)
	case events.KindTalkerAlias:
		if a, ok := ev.Payload.(trunking.TalkerAlias); ok {
			body = fmt.Sprintf("%-12s system=%s src=%d alias=%q",
				"TALKER-ALIAS", a.System, a.SourceID, a.Alias)
		}
	case events.KindLocation:
		if l, ok := ev.Payload.(trunking.Location); ok {
			body = fmt.Sprintf("%-12s system=%s proto=%s src=%d lat=%.6f lon=%.6f",
				"LOCATION", l.System, l.Protocol, l.RadioID, l.Latitude, l.Longitude)
		}
	case events.KindDecodeError:
		if d, ok := ev.Payload.(events.DecodeError); ok {
			body = fmt.Sprintf("%-12s proto=%s stage=%s", "DECODE-ERR", d.Protocol, d.Stage)
		}
	case events.KindToneAlert:
		body = fmt.Sprintf("%-12s %+v", "TONE-ALERT", ev.Payload)
	default:
		return ""
	}
	if body == "" {
		return ""
	}
	return stamp + " " + body + "\n"
}
