// Package tier2 decodes DMR Tier II conventional traffic. Tier II
// runs without a control channel: a repeater carries voice + signaling
// on a fixed frequency, and the start of every transmission is marked
// by a Voice LC Header burst whose 96-bit BPTC info block carries a
// Full Link Control PDU (source, destination, group/private flag).
//
// ConventionalChannel is the per-repeater state machine that watches
// for those headers and republishes them as protocol-agnostic
// trunking.Grant events. Compared to Tier III (internal/radio/dmr/tier3)
// the wire format is identical at the burst + slot-type + BPTC layers
// — only the call-setup mechanism differs (embedded LC vs. CSBK).
package tier2

import (
	"log/slog"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// LockState is the payload of cc.locked / cc.lost events emitted by
// the Tier II per-repeater state machine. DMR Tier II is conventional
// (no dedicated control channel), so "locked" here means "we've
// received at least one burst with a valid slot-type codeword on the
// tuned frequency."
type LockState struct {
	FrequencyHz uint32
	ColorCode   uint8 // from the first valid slot-type decode
}

// LockedFrequencyHz / LockedNAC make LockState satisfy
// trunking.LockedPayload so the cchunt supervisor's state machine
// recognises Tier II lock events alongside the other protocols. DMR
// doesn't have a P25-style NAC; the color code is the closest
// per-site identifier and gets plumbed into the NAC slot.
func (s LockState) LockedFrequencyHz() uint32 { return s.FrequencyHz }
func (s LockState) LockedNAC() uint16         { return uint16(s.ColorCode) }

// ConventionalChannel ingests bursts from one Tier II repeater
// frequency and emits a trunking.Grant the first time a Voice LC
// Header burst announces a new (talkgroup, source) tuple. Subsequent
// header bursts within the same superframe are de-duplicated so a
// long transmission produces exactly one grant. A Terminator with
// Link Control burst clears the state so the next transmission
// triggers a fresh grant.
type ConventionalChannel struct {
	bus        *events.Bus
	log        *slog.Logger
	systemName string
	freqHz     uint32
	now        func() time.Time

	// proc is the cross-call dibit / sync state the Process adapter
	// uses (see process.go). Lazily constructed on the first
	// Process call.
	proc *processState

	mu     sync.Mutex
	locked bool
	last   LockState

	inCall  bool
	lastTG  uint32
	lastSrc uint32
}

// Options configure a ConventionalChannel.
type Options struct {
	Bus         *events.Bus
	Log         *slog.Logger
	SystemName  string
	FrequencyHz uint32
	Now         func() time.Time
}

// New constructs a ConventionalChannel.
func New(opts Options) *ConventionalChannel {
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &ConventionalChannel{
		bus:        opts.Bus,
		log:        log,
		systemName: opts.SystemName,
		freqHz:     opts.FrequencyHz,
		now:        now,
	}
}

// IngestBurst hands one DMR burst (with its already-decoded slot type)
// to the state machine. Any burst with a valid slot-type codeword
// triggers cc.locked the first time it arrives on a freshly-tuned
// device. Bursts whose data type isn't a Voice LC Header or
// Terminator-with-LC otherwise only update the lock state: voice
// payload bursts (B-F) don't carry a fresh FLC, and CSBK bursts
// belong to Tier III.
func (c *ConventionalChannel) IngestBurst(b *dmr.Burst, slot dmr.SlotType) {
	c.maybeLock(LockState{
		FrequencyHz: c.freqHz,
		ColorCode:   slot.ColorCode,
	})
	switch slot.DataType {
	case dmr.DTVoiceLCHeader:
		c.handleVoiceHeader(b, slot)
	case dmr.DTTerminatorWithLC:
		c.handleTerminator()
	}
}

func (c *ConventionalChannel) maybeLock(s LockState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.locked && c.last == s {
		return
	}
	c.locked = true
	c.last = s
	c.bus.Publish(events.Event{Kind: events.KindCCLocked, Payload: s})
	c.log.Info("dmr/tier2 cc locked",
		"freq", s.FrequencyHz, "cc", s.ColorCode, "system", c.systemName)
}

// MarkLost publishes cc.lost and resets the locked flag. The trunking
// engine's hunter calls this when the repeater goes silent.
func (c *ConventionalChannel) MarkLost() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.locked {
		return
	}
	c.locked = false
	c.bus.Publish(events.Event{Kind: events.KindCCLost, Payload: c.last})
}

func (c *ConventionalChannel) handleVoiceHeader(b *dmr.Burst, slot dmr.SlotType) {
	bits, errs := framing.DecodeBPTC196_96(b.PayloadBits())
	if errs < 0 {
		c.log.Debug("dmr/tier2: voice header BPTC uncorrectable")
		c.bus.Publish(events.Event{
			Kind:    events.KindDecodeError,
			Payload: events.DecodeError{Protocol: "dmr-tier2", Stage: events.StageVoiceHeaderBPTC},
		})
		return
	}
	infoBytes := infoBitsToBytes(bits)
	// RS(12,9,4) parity check on the BPTC-recovered info block.
	// BPTC reports its own correction success but doesn't catch
	// systematic FEC misses — the RS layer above gives that
	// confidence. ETSI applies a per-context XOR seed to the parity
	// before transmission; for Voice LC Header it's 0x96 0x96 0x96.
	if !framing.VerifyRS12_9(infoBytes, framing.RS129SeedVoiceLCHeader) {
		c.log.Debug("dmr/tier2: voice header RS(12,9) parity mismatch")
		c.bus.Publish(events.Event{
			Kind:    events.KindDecodeError,
			Payload: events.DecodeError{Protocol: "dmr-tier2", Stage: events.StageVoiceHeaderRS},
		})
		return
	}
	flc, err := dmr.ParseFLC(infoBytes)
	if err != nil {
		c.log.Debug("dmr/tier2: FLC parse failed", "err", err)
		return
	}
	gv, ok := flc.AsGroupVoiceUser()
	if !ok {
		// Unit-to-unit and other opcodes are out of scope for this
		// pass — the engine's grant model is talkgroup-keyed.
		c.log.Debug("dmr/tier2: non-group FLCO ignored", "flco", flc.FLCO)
		return
	}
	if c.inCall && c.lastTG == gv.GroupAddress && c.lastSrc == gv.SourceID {
		// Same call's repeated Voice LC Header — dedupe.
		return
	}
	c.inCall = true
	c.lastTG = gv.GroupAddress
	c.lastSrc = gv.SourceID
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:      c.systemName,
			Protocol:    "dmr-tier2",
			GroupID:     gv.GroupAddress,
			SourceID:    gv.SourceID,
			FrequencyHz: c.freqHz,
			ChannelID:   slot.ColorCode,
			Encrypted:   gv.Encrypted,
			Emergency:   gv.Emergency,
			At:          c.now(),
		},
	})
	c.log.Debug("dmr/tier2: grant",
		"system", c.systemName, "freq_hz", c.freqHz,
		"cc", slot.ColorCode, "tg", gv.GroupAddress, "src", gv.SourceID,
		"enc", gv.Encrypted, "emer", gv.Emergency)
}

func (c *ConventionalChannel) handleTerminator() {
	if !c.inCall {
		return
	}
	c.inCall = false
	c.lastTG = 0
	c.lastSrc = 0
	c.log.Debug("dmr/tier2: terminator")
}

// infoBitsToBytes packs a 96-bit slice (each entry 0/1, MSB-first)
// into 12 bytes — the same shape ParseFLC expects for its leading 9
// octets, with the trailing 3 octets carrying RS(12,9) parity that
// this package intentionally ignores for now.
func infoBitsToBytes(bits []byte) []byte {
	if len(bits) != 96 {
		panic("dmr/tier2: infoBitsToBytes requires 96 bits")
	}
	out := make([]byte, 12)
	for i := 0; i < 96; i++ {
		if bits[i]&1 != 0 {
			out[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	return out
}
