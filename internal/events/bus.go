// Package events implements an in-process pub/sub bus used by the engine to
// publish trunking events. A separate API surface (gRPC, WebSocket) subscribes
// without coupling to the engine.
package events

import (
	"sync"
	"sync/atomic"
	"time"
)

type Kind string

const (
	KindSDRAttached Kind = "sdr.attached"
	KindSDRDetached Kind = "sdr.detached"
	KindCCLocked    Kind = "cc.locked"
	KindCCLost      Kind = "cc.lost"
	KindCallStart   Kind = "call.start"
	KindCallEnd     Kind = "call.end"
	KindGrant       Kind = "grant"
	KindToneAlert   Kind = "tone.alert"
	KindDecodeError Kind = "decode.error"
	KindError       Kind = "error"
	// Scanner subsystem (internal/scanner/cchunt):
	//   KindHuntProgress fires once per CC candidate the hunter
	//     tries — payload identifies which system + frequency +
	//     position in the candidate list, so the TUI can render
	//     "trying 851.012500 MHz (2/3)".
	//   KindHuntFailed fires when a system exhausts its CC list
	//     without locking; payload carries the next backoff window
	//     so operators can see "retry in 5 s".
	KindHuntProgress Kind = "cchunt.progress"
	KindHuntFailed   Kind = "cchunt.failed"
	// KindAffiliation fires when a radio unit affiliates with a
	// talkgroup. P25 control-channel publishes one per Group
	// Affiliation Response TSBK (opcode 0x28); the payload identifies
	// the source unit, the group it's joining, and the response code
	// (accepted / denied / refused / failed) from the system. Useful
	// downstream as a "who is listening where" feed for telemetry
	// dashboards.
	//
	// KindUnitRegistration fires when a radio registers (or
	// deregisters) on a site. P25 control-channel publishes one per
	// Unit Registration Response TSBK (opcode 0x2C); the payload
	// identifies the source unit, the WACN + System ID it's
	// registering on, and the response code. Useful as a "which
	// radios are on which site" feed.
	KindAffiliation      Kind = "affiliation"
	KindUnitRegistration Kind = "registration"
	// KindAudioState fires when an operator changes the live-audio
	// cockpit — volume, mute, or recording-gate. The payload is the
	// new state (the same shape as GET /api/v1/audio). Subscribers
	// can re-render volume sliders / mute indicators instantly
	// instead of waiting for the next 3 s poll tick. Emitted by
	// the HTTP API's PATCH /api/v1/audio handler.
	KindAudioState Kind = "audio.state"
	// KindPatch fires when a trunked system announces (or cancels) a
	// patch / dynamic-regroup — a super-group that merges several
	// talkgroups onto one channel. P25 Phase 2 publishes one per
	// Motorola group-regroup or Harris regroup MAC PDU. The payload
	// (trunking.Patch) carries the super-group, its member talkgroups,
	// and whether the patch is being added or removed.
	KindPatch Kind = "patch"
)

// Stage names a particular FEC / parser checkpoint inside a protocol
// decoder. Stages are used as Prometheus label values, so the wire
// shapes below are part of the Stage's public contract — extend the
// const block, don't rename existing entries.
type Stage string

// Known decode stages. Add new ones here and reference them from the
// publishing protocol package; the events bus itself stays neutral.
const (
	StageNIDBCH          Stage = "nid-bch"          // P25 Phase 1 NID BCH(63,16,11)
	StageTSBKTrellis     Stage = "tsbk-trellis"     // P25 Phase 1 TSBK ½-rate trellis
	StageTSBKCRC         Stage = "tsbk-crc"         // P25 Phase 1 TSBK CRC trailer
	StageNoBandPlan      Stage = "no-bandplan"      // Voice grant arrived for an unknown channel ID / LCN
	StageSlotTypeHamming Stage = "slottype-hamming" // DMR slot-type Hamming(20,8)
	StageVoiceHeaderBPTC Stage = "voiceheader-bptc" // DMR Tier II Voice LC Header BPTC(196,96)
	StageVoiceHeaderRS   Stage = "voiceheader-rs"   // DMR Tier II Voice LC Header RS(12,9,4)
	StageSACCHTrellis    Stage = "sacch-trellis"    // NXDN SACCH ½-rate trellis
)

// DecodeError is the payload published with KindDecodeError. Protocol
// packages publish this when an FEC primitive returns errCount == -1
// (or a parser short-circuits on a structural failure) so the metrics
// collector can increment gophertrunk_decode_errors_total without
// each package having to hold a *metrics.Metrics handle.
type DecodeError struct {
	Protocol string
	Stage    Stage
}

type Event struct {
	Kind      Kind
	Timestamp time.Time
	Payload   any
}

type Bus struct {
	mu     sync.RWMutex
	subs   map[uint64]chan Event
	nextID atomic.Uint64
	buffer int
	closed bool
}

func NewBus(buffer int) *Bus {
	if buffer <= 0 {
		buffer = 64
	}
	return &Bus{subs: make(map[uint64]chan Event), buffer: buffer}
}

type Subscription struct {
	id uint64
	C  <-chan Event
	b  *Bus
}

func (b *Bus) Subscribe() *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		ch := make(chan Event)
		close(ch)
		return &Subscription{C: ch}
	}
	id := b.nextID.Add(1)
	ch := make(chan Event, b.buffer)
	b.subs[id] = ch
	return &Subscription{id: id, C: ch, b: b}
}

func (s *Subscription) Close() {
	if s.b == nil {
		return
	}
	s.b.mu.Lock()
	defer s.b.mu.Unlock()
	if ch, ok := s.b.subs[s.id]; ok {
		delete(s.b.subs, s.id)
		close(ch)
	}
	s.b = nil
}

// Publish delivers e to every subscriber. Slow subscribers drop the event
// rather than blocking the publisher; we count drops via the returned int.
func (b *Bus) Publish(e Event) int {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	dropped := 0
	for _, ch := range b.subs {
		select {
		case ch <- e:
		default:
			dropped++
		}
	}
	return dropped
}

func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for id, ch := range b.subs {
		close(ch)
		delete(b.subs, id)
	}
}
