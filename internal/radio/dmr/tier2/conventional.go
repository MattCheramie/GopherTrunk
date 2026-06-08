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
	"encoding/hex"
	"log/slog"
	"strings"
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

	// protocolTag is the grant / decode-error Protocol string. It is
	// "dmr-tier2" for base-station conventional decode and "dmr-tier1" for
	// direct-mode decode — the wire format is identical, so the same state
	// machine serves both, distinguished by tag + sync-word set.
	protocolTag string
	// syncPatterns restricts the burst-sync detector to a subset of the 9
	// ETSI sync words. nil ⇒ all syncs (Tier II default). Tier I passes the
	// direct-mode syncs (DM-Voice/Data) so it doesn't false-lock on
	// base-station traffic.
	syncPatterns []dmr.SyncPattern

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
	// ProtocolTag overrides the grant / decode-error Protocol string.
	// Empty ⇒ "dmr-tier2". Set to "dmr-tier1" for direct-mode decode.
	ProtocolTag string
	// SyncPatterns restricts the burst-sync detector to these sync words.
	// nil ⇒ all 9 ETSI syncs (Tier II). Tier I passes the direct-mode set.
	SyncPatterns []dmr.SyncPattern
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
	tag := opts.ProtocolTag
	if tag == "" {
		tag = "dmr-tier2"
	}
	return &ConventionalChannel{
		bus:          opts.Bus,
		log:          log,
		systemName:   opts.SystemName,
		freqHz:       opts.FrequencyHz,
		now:          now,
		protocolTag:  tag,
		syncPatterns: opts.SyncPatterns,
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
	// The lock is to the repeater *frequency*; the color code is metadata
	// read from the slot-type field. A single Golay(20,8)-miscorrected
	// burst can flip the decoded color code (e.g. CC 0x7 → 0x5) on
	// otherwise-identical traffic, so deduping on the full {freq, CC}
	// LockState let an occasional slot-type FEC miss republish cc.locked
	// on every flip — churning the event bus and the
	// control_channel_transitions metric, and making the Tier II
	// integration test flaky on slow runners (the /metrics scrape lands
	// after several spurious re-locks). Dedup on frequency so a transient
	// color-code flicker leaves the established lock — and the color code
	// it first reported — untouched. A genuine retune to a different
	// frequency still (re)locks.
	if c.locked && c.last.FrequencyHz == s.FrequencyHz {
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
	payload := b.PayloadBits()
	bits, errs := framing.DecodeBPTC196_96(payload)
	if errs < 0 {
		// Dump the exact on-air bits so a single real failing burst can
		// be replayed through a reference decoder (DSD-FME / MMDVMHost)
		// offline. RS(12,9) and BPTC/Hamming match the MMDVM reference,
		// so an off-air-only BPTC failure points at the receiver's dibit
		// recovery or a bit-ordering detail a real capture would expose —
		// see docs/decoder-capture-needs.md. Debug level keeps it
		// opt-in.
		c.log.Debug("dmr/tier2: voice header BPTC uncorrectable",
			"cc", slot.ColorCode,
			"burst_dibits", dibitDigits(b.Dibits[:]),
			"payload_hex", hex.EncodeToString(packBitsMSB(payload)))
		c.bus.Publish(events.Event{
			Kind:    events.KindDecodeError,
			Payload: events.DecodeError{Protocol: c.protocolTag, Stage: events.StageVoiceHeaderBPTC},
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
		// BPTC succeeded but the RS(12,9) parity disagrees: the recovered
		// 12 octets (9 FLC + 3 seeded parity) are dumped so the exact
		// bytes can be checked against a reference decoder. Our RS(12,9)
		// is verified equal to MMDVMHost CRS129 (generator {64,56,14},
		// roots alpha^1..3, seed 0x96) by TestRS129MatchesIndependent-
		// ReferenceEncoder, so a real mismatch here implicates the BPTC
		// info-bit recovery feeding it rather than the RS field itself.
		c.log.Debug("dmr/tier2: voice header RS(12,9) parity mismatch",
			"cc", slot.ColorCode,
			"info_hex", hex.EncodeToString(infoBytes))
		c.bus.Publish(events.Event{
			Kind:    events.KindDecodeError,
			Payload: events.DecodeError{Protocol: c.protocolTag, Stage: events.StageVoiceHeaderRS},
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
			Protocol:    c.protocolTag,
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

// packBitsMSB packs a 0/1 bit slice MSB-first into bytes (the final byte
// is zero-padded if len(bits) isn't a multiple of 8). Used only to render
// a failing burst's payload bits as hex for the diagnostic Debug log.
func packBitsMSB(bits []byte) []byte {
	out := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b&1 != 0 {
			out[i>>3] |= 1 << uint(7-(i&7))
		}
	}
	return out
}

// dibitDigits renders a dibit slice as a compact base-4 digit string
// (e.g. "0312...") so a failing burst's exact 132 symbols can be copied
// out of the Debug log and replayed through a reference decoder.
func dibitDigits(dibits []uint8) string {
	var sb strings.Builder
	sb.Grow(len(dibits))
	for _, d := range dibits {
		sb.WriteByte('0' + (d & 3))
	}
	return sb.String()
}
