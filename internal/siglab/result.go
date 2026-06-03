package siglab

import (
	"reflect"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Result is the protocol-agnostic structured outcome of one engine run. It
// is the single model every exporter (JSON / JSONL / YAML / CSV) and the
// acceptance harness render — populated identically for all 13 protocols.
type Result struct {
	// Provenance.
	Source         string  `json:"source" yaml:"source"`
	Protocol       string  `json:"protocol" yaml:"protocol"`
	SampleRateHz   float64 `json:"sample_rate_hz" yaml:"sample_rate_hz"`
	PipelineRateHz float64 `json:"pipeline_rate_hz" yaml:"pipeline_rate_hz"`
	TuneHz         float64 `json:"tune_hz" yaml:"tune_hz"`
	DurationSec    float64 `json:"duration_sec" yaml:"duration_sec"`

	// Throughput.
	TotalSamples     int64   `json:"total_samples" yaml:"total_samples"`
	Symbols          int64   `json:"symbols" yaml:"symbols"`
	EffectiveBaud    float64 `json:"effective_baud" yaml:"effective_baud"`
	ExpectedBaud     float64 `json:"expected_baud" yaml:"expected_baud"`
	BaudDeviationPct float64 `json:"baud_deviation_pct" yaml:"baud_deviation_pct"`

	// Lock.
	Locked         bool      `json:"locked" yaml:"locked"`
	LockLatencySec float64   `json:"lock_latency_sec" yaml:"lock_latency_sec"`
	Lock           *LockInfo `json:"lock,omitempty" yaml:"lock,omitempty"`

	// Grants + events.
	Grants       []GrantRecord  `json:"grants" yaml:"grants"`
	Events       []EventRecord  `json:"events" yaml:"events"`
	EventCounts  map[string]int `json:"event_counts" yaml:"event_counts"`
	DecodeErrors map[string]int `json:"decode_errors" yaml:"decode_errors"`

	// Analysis (nil unless CollectIQDiag).
	Signal *SignalQuality `json:"signal,omitempty" yaml:"signal,omitempty"`
	P25P1  *P25P1Detail   `json:"p25p1,omitempty" yaml:"p25p1,omitempty"`

	// Verdict (nil unless Config.Acceptance supplied).
	Verdict *Verdict `json:"verdict,omitempty" yaml:"verdict,omitempty"`
}

// LockInfo is the flattened, protocol-agnostic view of a KindCCLocked
// payload: the frequency every LockState carries, plus a Fields map of the
// remaining protocol-specific fields (NAC/DUID, ColorCode/SystemID, MCC/MNC,
// RAN, …) so one struct represents all 13 protocols' distinct LockState
// types.
type LockInfo struct {
	FrequencyHz uint32         `json:"frequency_hz" yaml:"frequency_hz"`
	Fields      map[string]any `json:"fields" yaml:"fields"`
}

// GrantRecord mirrors trunking.Grant, flattened with the stream-time offset
// at which it was observed.
type GrantRecord struct {
	OffsetSec   float64 `json:"offset_sec" yaml:"offset_sec"`
	GroupID     uint32  `json:"group_id" yaml:"group_id"`
	SourceID    uint32  `json:"source_id" yaml:"source_id"`
	ChannelID   uint8   `json:"channel_id" yaml:"channel_id"`
	ChannelNum  uint16  `json:"channel_num" yaml:"channel_num"`
	Timeslot    uint8   `json:"timeslot" yaml:"timeslot"`
	FrequencyHz uint32  `json:"frequency_hz" yaml:"frequency_hz"`
	Encrypted   bool    `json:"encrypted" yaml:"encrypted"`
	Emergency   bool    `json:"emergency" yaml:"emergency"`
}

// EventRecord is one bus event captured generically — Kind plus the
// reflection-flattened payload — so the JSONL stream and the events CSV
// represent every protocol's events without a typed switch per protocol.
type EventRecord struct {
	Seq       int            `json:"seq" yaml:"seq"`
	OffsetSec float64        `json:"offset_sec" yaml:"offset_sec"`
	Kind      string         `json:"kind" yaml:"kind"`
	Fields    map[string]any `json:"fields" yaml:"fields"`
}

// collector accumulates bus events into the Result-facing slices/maps. It is
// driven by a single goroutine draining one subscription, so it needs no
// locking; the engine reads the assembled fields after the drainer exits.
type collector struct {
	startWall   time.Time
	seq         int
	events      []EventRecord
	grants      []GrantRecord
	eventCounts map[string]int
	decodeErr   map[string]int
	locked      bool
	lockLatency float64
	lock        *LockInfo
	// onEvent, when set, is called for every EventRecord as it is built —
	// the streaming sink behind RunStream / JSONL.
	onEvent func(EventRecord)
}

func newCollector(start time.Time, onEvent func(EventRecord)) *collector {
	return &collector{
		startWall:   start,
		eventCounts: map[string]int{},
		decodeErr:   map[string]int{},
		onEvent:     onEvent,
	}
}

// observe folds one bus event into the running totals.
func (c *collector) observe(ev events.Event) {
	offset := ev.Timestamp.Sub(c.startWall).Seconds()
	if offset < 0 {
		offset = 0
	}
	kind := string(ev.Kind)
	c.eventCounts[kind]++

	rec := EventRecord{Seq: c.seq, OffsetSec: offset, Kind: kind, Fields: flattenPayload(ev.Payload)}
	c.seq++
	c.events = append(c.events, rec)
	if c.onEvent != nil {
		c.onEvent(rec)
	}

	switch ev.Kind {
	case events.KindCCLocked:
		if !c.locked {
			c.locked = true
			c.lockLatency = offset
			c.lock = lockInfoFromFields(rec.Fields)
		}
	case events.KindGrant:
		if g, ok := ev.Payload.(trunking.Grant); ok {
			c.grants = append(c.grants, GrantRecord{
				OffsetSec:   offset,
				GroupID:     g.GroupID,
				SourceID:    g.SourceID,
				ChannelID:   g.ChannelID,
				ChannelNum:  g.ChannelNum,
				Timeslot:    g.Timeslot,
				FrequencyHz: g.FrequencyHz,
				Encrypted:   g.Encrypted,
				Emergency:   g.Emergency,
			})
		}
	case events.KindDecodeError:
		if de, ok := ev.Payload.(events.DecodeError); ok {
			c.decodeErr[string(de.Stage)]++
		}
	}
}

// lockInfoFromFields pulls the common FrequencyHz out of a flattened lock
// payload and keeps the remainder as protocol-specific Fields.
func lockInfoFromFields(fields map[string]any) *LockInfo {
	li := &LockInfo{Fields: map[string]any{}}
	for k, v := range fields {
		if k == "FrequencyHz" {
			li.FrequencyHz = toUint32(v)
			continue
		}
		li.Fields[k] = v
	}
	return li
}

// flattenPayload reflects an event payload into a JSON/YAML-friendly map of
// its exported fields. Pointers are dereferenced; func/chan/unsafe fields
// are skipped. A non-struct payload is wrapped as {"value": payload}. This
// is what lets the collector capture all 13 protocols' distinct LockState
// types (and every other payload) without a per-type switch.
func flattenPayload(v any) map[string]any {
	out := map[string]any{}
	if v == nil {
		return out
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return out
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		out["value"] = v
		return out
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		switch rv.Field(i).Kind() {
		case reflect.Func, reflect.Chan, reflect.UnsafePointer:
			continue
		}
		out[f.Name] = rv.Field(i).Interface()
	}
	return out
}

// toUint32 best-effort coerces a reflected numeric field to uint32.
func toUint32(v any) uint32 {
	switch n := v.(type) {
	case uint32:
		return n
	case uint16:
		return uint32(n)
	case uint8:
		return uint32(n)
	case uint64:
		return uint32(n)
	case uint:
		return uint32(n)
	case int:
		return uint32(n)
	case int32:
		return uint32(n)
	case int64:
		return uint32(n)
	default:
		return 0
	}
}
