// Package broadcast streams completed trunked-radio calls to external
// call aggregators — Broadcastify Calls, RdioScanner, OpenMHz — and to
// live Icecast/ShoutCast audio servers.
//
// The Manager subscribes to events.KindCallComplete (published by the
// recorder once a call's WAV is flushed to disk), resolves which
// backends want the call, encodes the audio to MP3 once, and dispatches
// it to each backend with bounded exponential-backoff retry. A call is
// skipped entirely when its talkgroup is flagged Stream=false or when
// it is shorter than the configured minimum duration.
package broadcast

import (
	"context"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice/mp3"
)

// Call is a completed call queued for outbound streaming. It carries
// the call metadata plus a lazy MP3 accessor — the WAV on AudioPath is
// encoded to MP3 at most once regardless of how many backends consume
// it.
type Call struct {
	System         string
	Protocol       string
	Talkgroup      uint32
	TalkgroupLabel string
	Source         uint32
	FrequencyHz    uint32
	Encrypted      bool
	Emergency      bool
	PatchedGroups  []uint32
	StartedAt      time.Time
	EndedAt        time.Time
	AudioPath      string // .wav on disk written by the recorder
	SampleRate     int

	mu      sync.Mutex
	mp3Data []byte
	mp3Err  error
	mp3Done bool
}

// Duration reports how long the call ran.
func (c *Call) Duration() time.Duration { return c.EndedAt.Sub(c.StartedAt) }

// MP3 returns the call audio encoded as an MP3 byte stream. The encode
// runs once on first use; later calls return the cached result (or the
// cached error). Safe for concurrent use across backend workers.
func (c *Call) MP3() ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.mp3Done {
		return c.mp3Data, c.mp3Err
	}
	c.mp3Done = true
	c.mp3Data, _, c.mp3Err = mp3.EncodeWAVFile(c.AudioPath)
	return c.mp3Data, c.mp3Err
}

// callFromEvent builds a *Call from a recorder CallComplete payload.
func callFromEvent(cc trunking.CallComplete) *Call {
	c := &Call{
		System:        cc.Grant.System,
		Protocol:      cc.Grant.Protocol,
		Talkgroup:     cc.Grant.GroupID,
		Source:        cc.Grant.SourceID,
		FrequencyHz:   cc.Grant.FrequencyHz,
		Encrypted:     cc.Grant.Encrypted,
		Emergency:     cc.Grant.Emergency,
		PatchedGroups: cc.Grant.PatchedGroups,
		StartedAt:     cc.StartedAt,
		EndedAt:       cc.EndedAt,
		AudioPath:     cc.AudioPath,
		SampleRate:    int(cc.SampleRate),
	}
	if cc.Talkgroup != nil {
		c.TalkgroupLabel = cc.Talkgroup.AlphaTag
	}
	return c
}

// Backend is one outbound streaming destination. Implementations must
// be safe for concurrent Send calls — the Manager runs several upload
// workers. Send should treat a retry as a fresh attempt; the Manager
// handles backoff between attempts.
type Backend interface {
	// Name is a short stable identifier used in logs and stats.
	Name() string
	// Accepts reports whether this backend wants calls from the named
	// trunking system. An unfiltered backend accepts every system.
	Accepts(systemName string) bool
	// Send delivers the call. A non-nil error triggers a retry.
	Send(ctx context.Context, c *Call) error
}

// systemFilter restricts a backend to a set of trunking-system names.
// A zero filter (constructed from an empty list) matches every system.
type systemFilter struct {
	systems map[string]bool
}

func newSystemFilter(names []string) systemFilter {
	if len(names) == 0 {
		return systemFilter{}
	}
	m := make(map[string]bool, len(names))
	for _, n := range names {
		if n != "" {
			m[n] = true
		}
	}
	return systemFilter{systems: m}
}

// Accepts reports whether systemName passes the filter.
func (f systemFilter) Accepts(systemName string) bool {
	if f.systems == nil {
		return true
	}
	return f.systems[systemName]
}
