package mpt1327

import (
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// LockState is the payload of cc.locked / cc.lost events emitted by
// the MPT 1327 control-channel state machine.
type LockState struct {
	FrequencyHz uint32
	SystemID    uint16 // first AHYC's Function payload, when seen
	Prefix      uint8  // first valid codeword's prefix
}

// ControlChannel ingests MPT 1327 codewords from a single control
// channel, emits cc.locked the first time a valid Aloha (ALH) or
// AHYC broadcast arrives on a freshly-tuned device, and republishes
// GTC voice grants as events.KindGrant carrying a `trunking.Grant`
// payload with `Protocol = "mpt1327"`. Same shape as the Motorola /
// EDACS / LTR control channels.
type ControlChannel struct {
	bus        *events.Bus
	log        *slog.Logger
	systemName string
	freqHz     uint32
	resolver   Resolver
	now        func() time.Time

	// proc is the cross-call bit / alignment state the Process
	// adapter uses (see process.go). Lazily constructed on the
	// first Process call.
	proc *processState

	mu               sync.Mutex
	locked           bool
	last             LockState
	strictValidation bool
	bchMode          BCHMode
}

// SetStrictValidation toggles the strict frame-validity filter on
// the Ingest path. When enabled, codewords whose Kind falls
// outside the recognised opcode set (Aloha / Ahoy / AhoyChan /
// GoToChan / Ack / Disconnect / Data / Emergency) are silently
// dropped. The Process adapter already filters at the alignment
// layer; strict-mode tightens it further at Ingest time so PDUs
// from a misaligned-but-passing window still drop out.
func (c *ControlChannel) SetStrictValidation(strict bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.strictValidation = strict
}

// BCHMode selects how the Process adapter interprets the incoming
// bit stream:
//
//   - BCHOff (default): the adapter reads 38-bit information
//     windows directly from the wire and treats them as Codewords.
//     Works on synthesized test fixtures whose codewords are
//     pre-stripped of the FEC layer.
//
//   - BCHOn: the adapter reads 64-bit on-wire codewords, runs
//     them through the BCH(64,48,2) check + single-bit correction
//     in internal/radio/framing/bch_mpt1327.go, and extracts the
//     38-bit information field expected by the existing
//     CodewordFromBits parser. The 10-bit Op field that the
//     spec carries between Ident and Function isn't modelled by
//     this package and is dropped during extraction; the Kind
//     extracted from the upper 4 bits of Function (as
//     CodewordKind defines) still drives the state machine.
//
// Under BCHOn, the alignment search picks the first 64-bit
// window that passes BCH (much more selective than the 38-bit
// "recognised opcode" search), so live-air captures whose first
// few codewords carry bit errors still synchronise. Single-bit
// errors per codeword are corrected; uncorrectable codewords
// (≥ 2 bit errors in unfavourable positions) are dropped.
type BCHMode uint8

const (
	BCHOff BCHMode = iota
	BCHOn
)

// SetBCHMode toggles the BCH FEC layer on the Process adapter.
// See BCHMode for the trade-offs. The mode applies to every
// subsequent Process call; the Ingest entry point is unaffected
// (callers that pre-parse codewords don't go through this
// adapter).
func (c *ControlChannel) SetBCHMode(mode BCHMode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bchMode = mode
}

// BCHMode returns the configured BCHMode.
func (c *ControlChannel) BCHMode() BCHMode {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.bchMode
}

// ParseBCHMode maps a config / user-facing string into a BCHMode.
// Recognised values (case-insensitive): "" / "off" / "false" / "0"
// → BCHOff (legacy 38-bit pre-stripped codeword path); "on" /
// "true" / "1" → BCHOn (64-bit on-wire BCH(63, 38) decode). Unknown
// strings return BCHOff with `ok = false`.
func ParseBCHMode(s string) (BCHMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off", "false", "0":
		return BCHOff, true
	case "on", "true", "1":
		return BCHOn, true
	default:
		return BCHOff, false
	}
}

// Options configure a ControlChannel.
type Options struct {
	Bus         *events.Bus
	Log         *slog.Logger
	SystemName  string
	FrequencyHz uint32
	Resolver    Resolver
	Now         func() time.Time
}

// New constructs a ControlChannel.
func New(opts Options) *ControlChannel {
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &ControlChannel{
		bus:        opts.Bus,
		log:        log,
		systemName: opts.SystemName,
		freqHz:     opts.FrequencyHz,
		resolver:   opts.Resolver,
		now:        now,
	}
}

// codewordKindIsRecognised reports whether the Codeword's Kind
// matches one of the trunking-relevant opcodes the state machine
// acts on. Mirrors the Process adapter's alignment filter so
// SetStrictValidation can apply it on the Ingest path too.
func codewordKindIsRecognised(w Codeword) bool {
	if w.Type != TypeAddress {
		return false
	}
	switch w.Kind() {
	case KindAloha, KindAhoy, KindAhoyChan, KindGoToChan,
		KindAck, KindDisconnect, KindData, KindEmergency:
		return true
	}
	return false
}

// Ingest hands a single decoded codeword to the state machine. Real
// captures arrive via an upstream FFSK demod + BCH decoder; tests
// publish codewords directly.
func (c *ControlChannel) Ingest(w Codeword) {
	c.mu.Lock()
	strict := c.strictValidation
	c.mu.Unlock()
	if strict && !codewordKindIsRecognised(w) {
		return
	}
	if w.Type != TypeAddress {
		// Data codewords carry short-message payloads we don't
		// follow at the trunking layer.
		return
	}
	switch w.Kind() {
	case KindAloha:
		c.maybeLock(LockState{
			FrequencyHz: c.freqHz,
			Prefix:      w.Prefix,
		})
	case KindAhoyChan:
		if a, ok := w.AsAhoyChannel(); ok {
			c.maybeLock(LockState{
				FrequencyHz: c.freqHz,
				SystemID:    a.System,
				Prefix:      w.Prefix,
			})
		}
	case KindGoToChan:
		if g, ok := w.AsGoToChannel(); ok {
			c.publishGrant(g)
		}
	}
}

func (c *ControlChannel) publishGrant(g GoToChannel) {
	if c.bus == nil {
		return
	}
	freq := uint32(0)
	if c.resolver != nil {
		if hz, err := c.resolver.Frequency(g.Channel); err == nil {
			freq = hz
		} else {
			c.log.Debug("mpt1327: band-plan resolution failed",
				"channel", g.Channel, "err", err)
		}
	}
	// MPT 1327 doesn't carry a single "talkgroup" the way the central-
	// CC trunked systems do — the called party is identified by
	// (Prefix, Ident). We surface Ident as GroupID so downstream
	// consumers (the engine, the recorder) get something useful, and
	// fold Prefix into the high 16 bits of GroupID for disambiguation
	// when Idents repeat across prefixes.
	groupID := uint32(g.Prefix)<<16 | uint32(g.Ident)
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:      c.systemName,
			Protocol:    "mpt1327",
			GroupID:     groupID,
			FrequencyHz: freq,
			ChannelNum:  g.Channel,
			At:          c.now(),
		},
	})
	c.log.Debug("mpt1327: grant",
		"system", c.systemName,
		"prefix", g.Prefix, "ident", g.Ident,
		"channel", g.Channel, "freq_hz", freq)
}

func (c *ControlChannel) maybeLock(s LockState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.locked && c.last == s {
		return
	}
	c.locked = true
	c.last = s
	c.bus.Publish(events.Event{Kind: events.KindCCLocked, Payload: s})
	c.log.Info("mpt1327 cc locked",
		"freq", s.FrequencyHz, "sys", s.SystemID, "prefix", s.Prefix,
		"system", c.systemName)
}

// MarkLost publishes cc.lost and resets the locked flag. The trunking
// engine's hunter calls this when the control channel goes silent.
func (c *ControlChannel) MarkLost() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.locked {
		return
	}
	c.locked = false
	c.bus.Publish(events.Event{Kind: events.KindCCLost, Payload: c.last})
}
