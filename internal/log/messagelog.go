package log

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// stampLayout renders a displayed timestamp with milliseconds and an explicit
// numeric offset, so "Z" appears only when the location really is UTC and any
// other zone is unambiguous (e.g. "2026-06-25T14:03:05.250+02:00").
const stampLayout = "2006-01-02T15:04:05.000Z07:00"

// stampTime formats ts in loc using stampLayout, substituting time.Now() for a
// zero timestamp and time.Local for a nil location.
func stampTime(ts time.Time, loc *time.Location) string {
	if ts.IsZero() {
		ts = time.Now()
	}
	if loc == nil {
		loc = time.Local
	}
	return ts.In(loc).Format(stampLayout)
}

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
	loc       *time.Location
	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once

	// lastNeighbors remembers the last logged neighbour set per system (the
	// joined neighbour lines) so a SiteUpdate — which the decoder republishes
	// on every status broadcast — only writes a block when the adjacent-site
	// list actually changes. Accessed only from the Run goroutine.
	lastNeighbors map[string]string

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
	// Loc is the timezone displayed timestamps are rendered in. Nil means
	// time.Local — the operator's wall-clock time (the daemon passes
	// cfg.Display.Location()).
	Loc *time.Location
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
	loc := opts.Loc
	if loc == nil {
		loc = time.Local
	}
	info, _ := f.Stat()
	ml := &MessageLog{
		bus:           opts.Bus,
		path:          opts.Path,
		maxBytes:      int64(opts.MaxSizeMB) * 1024 * 1024,
		loc:           loc,
		runDone:       make(chan struct{}),
		f:             f,
		lastNeighbors: make(map[string]string),
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
			if line := m.formatEvent(ev); line != "" {
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
func (m *MessageLog) formatEvent(ev events.Event) string {
	ts := ev.Timestamp
	stamp := stampTime(ts, m.loc)
	var body string
	switch ev.Kind {
	case events.KindGrant:
		if g, ok := ev.Payload.(trunking.Grant); ok {
			body = fmt.Sprintf("%-12s system=%s proto=%s tg=%d src=%d freq=%d enc=%t emer=%t",
				"GRANT", g.System, g.Protocol, g.GroupID, g.SourceID,
				g.FrequencyHz, g.Encrypted, g.Emergency)
			// ALGID/KID only meaningful when the grant is encrypted
			// AND the in-call signalling has surfaced them (Phase 2
			// at grant time; Phase 1 after the first LDU2).
			if g.Encrypted && (g.AlgorithmID != 0 || g.KeyID != 0) {
				body += fmt.Sprintf(" alg=0x%02X key=0x%04X",
					g.AlgorithmID, g.KeyID)
			}
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
	case events.KindCallEncryption:
		if c, ok := ev.Payload.(trunking.CallEncryption); ok {
			// Only log the engine's enriched republish (System set);
			// the raw composer publish has empty identity fields and
			// would just be noise.
			if c.System != "" {
				body = fmt.Sprintf("%-12s system=%s tg=%d dev=%s alg=0x%02X key=0x%04X",
					"CALL-ENCRYPT", c.System, c.GroupID, c.DeviceSerial,
					c.AlgorithmID, c.KeyID)
			}
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
	case events.KindUnitToUnitRequest:
		if u, ok := ev.Payload.(trunking.UnitToUnitRequest); ok {
			body = fmt.Sprintf("%-12s system=%s proto=%s src=%d target=%d",
				"U2U-REQUEST", u.System, u.Protocol, u.SourceID, u.TargetID)
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
	case events.KindSiteUpdate:
		if su, ok := ev.Payload.(trunking.SiteUpdate); ok {
			return m.formatNeighborBlock(su, stamp)
		}
		return ""
	default:
		return ""
	}
	if body == "" {
		return ""
	}
	return stamp + " " + body + "\n"
}

// formatNeighborBlock renders the camped site's adjacent ("neighbor") sites as
// one log line per neighbour — GopherTrunk's analogue of SDRtrunk's "Neighbor
// Sites" block. A SiteUpdate is republished on every status broadcast, so the
// block is deduplicated per system: it is only emitted when the neighbour set
// changes. Returns "" when there are no neighbours or the set is unchanged.
func (m *MessageLog) formatNeighborBlock(su trunking.SiteUpdate, stamp string) string {
	lines := trunking.RenderNeighborLines(su.Topology)
	if len(lines) == 0 {
		return ""
	}
	joined := strings.Join(lines, "\n")
	if m.lastNeighbors[su.System] == joined {
		return ""
	}
	m.lastNeighbors[su.System] = joined

	var b strings.Builder
	fmt.Fprintf(&b, "%s %-12s system=%s count=%d\n", stamp, "NEIGHBORS", su.System, len(lines))
	for _, ln := range lines {
		fmt.Fprintf(&b, "%s %-12s system=%s %s\n", stamp, "NEIGHBOR", su.System, ln)
	}
	return b.String()
}
