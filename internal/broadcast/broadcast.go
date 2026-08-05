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
	"math"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/loudness"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice/mp3"
)

// NormalizeConfig configures optional loudness normalization of the
// distributed MP3 copy. When Enabled, the WAV on disk is left untouched
// and the gain is applied in memory before MP3 encoding. Params are
// assumed to already carry defaults (see loudness.NormalizeParams).
type NormalizeConfig struct {
	Enabled bool
	Params  loudness.NormalizeParams
}

// Call is a completed call queued for outbound streaming. It carries
// the call metadata plus a lazy MP3 accessor — the WAV on AudioPath is
// encoded to MP3 at most once regardless of how many backends consume
// it.
type Call struct {
	System         string
	Protocol       string
	Talkgroup      uint32
	TalkgroupLabel string
	// TalkgroupTag / TalkgroupGroup carry the talkgroup's department/category
	// and top-level group. Aggregators that show per-call tag/group columns
	// (RdioScanner's talkgroupTag / talkgroupGroup) render these; without them
	// the console falls back to its own static per-system config, which is
	// where a stale label like a wrong protocol type comes from. Both empty
	// until the talkgroup is resolved/tagged.
	TalkgroupTag   string
	TalkgroupGroup string
	Source         uint32
	FrequencyHz    uint32
	// ChannelID / RFSS / Site / NAC / Timeslot carry the P25 site
	// identity decoded off the control channel (issue #698). They let a
	// metadata sink (the webhook) label a call by site for downstream
	// dashboards instead of resolving an opaque channel ID by hand. All
	// stay zero on non-P25 calls and until the site's status TSBKs land;
	// audio-upload backends ignore them.
	ChannelID uint8
	RFSS      uint8
	Site      uint8
	NAC       uint16
	Timeslot  uint8
	Encrypted bool
	// AlgorithmID / KeyID surface the P25 encryption parameters when
	// the in-call signalling has revealed them. Zero on clear calls
	// and on encrypted calls whose Encryption Sync never arrived
	// before the call ended. Aggregators that accept the fields can
	// thread them into their API payload; aggregators that don't will
	// just ignore them.
	AlgorithmID uint8
	KeyID       uint16
	// Individual marks a unit-to-unit / individual call, whose Talkgroup
	// field carries a destination radio ID rather than a talkgroup, and
	// DataCall a data (non-voice) grant. They let a metadata sink emit a
	// call_type so a downstream consumer doesn't mistake a unit's target RID
	// for a talkgroup (issue #897). Both false for an ordinary group call.
	Individual    bool
	DataCall      bool
	Emergency     bool
	PatchedGroups []uint32
	StartedAt     time.Time
	EndedAt       time.Time
	AudioPath     string // .wav on disk written by the recorder
	SampleRate    int

	// normalize, when Enabled, loudness-normalizes the audio in memory
	// before MP3 encoding (the on-disk WAV is left pristine). Set by the
	// Manager from its configuration.
	normalize NormalizeConfig

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
	if c.normalize.Enabled {
		c.mp3Data, c.mp3Err = encodeNormalizedMP3(c.AudioPath, c.normalize.Params)
	} else {
		c.mp3Data, _, c.mp3Err = mp3.EncodeWAVFile(c.AudioPath)
	}
	return c.mp3Data, c.mp3Err
}

// encodeNormalizedMP3 reads the WAV at path, applies a single linear
// loudness-normalization gain in memory (leaving the file untouched), and
// MP3-encodes the result. When the audio is too short or quiet to measure,
// it encodes the samples unchanged.
func encodeNormalizedMP3(path string, p loudness.NormalizeParams) ([]byte, error) {
	samples, rate, err := mp3.ReadWAV(path)
	if err != nil {
		return nil, err
	}
	fs := make([]float64, len(samples))
	for i, s := range samples {
		fs[i] = float64(s) / 32768.0
	}
	if gain, ok := loudness.NormalizeGain(fs, rate, p); ok {
		for i, v := range fs {
			samples[i] = clampInt16(v * gain * 32768.0)
		}
	}
	return mp3.Encode(samples, rate)
}

// clampInt16 rounds and saturates a float sample value to int16.
func clampInt16(v float64) int16 {
	r := math.Round(v)
	if r > 32767 {
		return 32767
	}
	if r < -32768 {
		return -32768
	}
	return int16(r)
}

// callFromEvent builds a *Call from a recorder CallComplete payload.
func callFromEvent(cc trunking.CallComplete) *Call {
	c := &Call{
		System:        cc.Grant.System,
		Protocol:      cc.Grant.Protocol,
		Talkgroup:     cc.Grant.GroupID,
		Source:        cc.Grant.SourceID,
		FrequencyHz:   cc.Grant.FrequencyHz,
		ChannelID:     cc.Grant.ChannelID,
		RFSS:          cc.Grant.RFSSID,
		Site:          cc.Grant.SiteID,
		NAC:           cc.Grant.NAC,
		Timeslot:      cc.Grant.Timeslot,
		Encrypted:     cc.Grant.Encrypted,
		AlgorithmID:   cc.Grant.AlgorithmID,
		KeyID:         cc.Grant.KeyID,
		Individual:    cc.Grant.Individual,
		DataCall:      cc.Grant.DataCall,
		Emergency:     cc.Grant.Emergency,
		PatchedGroups: cc.Grant.PatchedGroups,
		StartedAt:     cc.StartedAt,
		EndedAt:       cc.EndedAt,
		AudioPath:     cc.AudioPath,
		SampleRate:    int(cc.SampleRate),
	}
	if cc.Talkgroup != nil {
		c.TalkgroupLabel = cc.Talkgroup.AlphaTag
		c.TalkgroupTag = cc.Talkgroup.Tag
		c.TalkgroupGroup = cc.Talkgroup.Group
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
