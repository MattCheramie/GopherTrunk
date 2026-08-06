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
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier3"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// beaconLogInterval rate-limits the operator-facing "site alive" Info log
// so a repeater whose idle beacon fires every few seconds announces
// itself once, then stays quiet — the Beacons counter keeps the exact
// count. See handleCSBK.
const beaconLogInterval = 30 * time.Second

// LockState is the payload of cc.locked / cc.lost events emitted by
// the Tier II per-repeater state machine. DMR Tier II is conventional
// (no dedicated control channel), so "locked" here means "we've
// received at least one FEC-validated Voice LC Header (BPTC + RS pass)
// on the tuned frequency" — proof of a real DMR transmission, not just
// a slot-type codeword that a noise burst can forge.
type LockState struct {
	FrequencyHz uint32
	ColorCode   uint8 // from the first valid slot-type decode
}

// Counters is a lock-free snapshot of one channel's decode activity,
// read periodically by an operator-facing supervisor (the wideband
// engine's diagnostics) to tell which stage is failing. All counts are
// monotonic since channel construction. Purely observational — the
// counters never gate decoding.
//
// Reading the four-way story:
//   - SyncHits == 0                  → no DMR sync detected (no signal,
//     mistune, wrong offset, or spectrum-inversion the polarity pass
//     can't recover).
//   - SyncHits > 0, FECPass == 0 and FECFail > 0 → sync seen but every
//     Voice LC Header fails FEC (weak/dirty signal, clipping).
//   - FECPass > 0                    → genuine DMR headers decoded.
type Counters struct {
	SyncHits uint64 // FSW matches reported by the burst-sync detector
	Bursts   uint64 // slot-type-parsed bursts handed to IngestBurst
	FECPass  uint64 // Voice LC Header with BPTC + RS both valid
	FECFail  uint64 // Voice LC Header BPTC uncorrectable or RS mismatch
	Locks    uint64 // cc.locked declarations (lifetime)
	Beacons  uint64 // CRC-valid CSBK "site alive" bursts (issue #1036)
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

	// cnt holds the lock-free decode-activity counters exposed via
	// Counters(). Incremented on the existing hot paths with atomic
	// adds so any goroutine can snapshot them without taking c.mu.
	cnt struct {
		syncHits atomic.Uint64
		bursts   atomic.Uint64
		fecPass  atomic.Uint64
		fecFail  atomic.Uint64
		locks    atomic.Uint64
		beacons  atomic.Uint64
	}

	// beaconLogAt is the last time handleCSBK emitted an Info "site alive"
	// line; guarded by mu and used only to rate-limit that log.
	beaconLogAt time.Time
}

// Counters returns a snapshot of this channel's decode-activity
// counters. Safe to call concurrently with the decode path.
func (c *ConventionalChannel) Counters() Counters {
	return Counters{
		SyncHits: c.cnt.syncHits.Load(),
		Bursts:   c.cnt.bursts.Load(),
		FECPass:  c.cnt.fecPass.Load(),
		FECFail:  c.cnt.fecFail.Load(),
		Locks:    c.cnt.locks.Load(),
		Beacons:  c.cnt.beacons.Load(),
	}
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
// to the state machine. The lock is NOT declared here on the slot type
// alone: the slot-type Hamming(20,8) corrects up to 3 bit errors, so a
// noise burst whose 24-dibit sync false-matched (the detector runs at
// tolerance 2 against 9 patterns) routinely yields a "valid" slot type
// — typically with the minimum-distance color code 0xF — and the old
// unconditional lock here produced the field-reported "instalock cc=15
// then nothing". Instead the lock is gated on a FEC-validated Voice LC
// Header inside handleVoiceHeader (BPTC + RS both pass), mirroring Tier
// III's lock-only-after-CRC discipline. Voice payload bursts (B-F)
// don't carry a fresh FLC, so they fall through untouched. CSBK bursts
// are routed to handleCSBK: on a conventional/IPSC repeater the periodic
// idle "beacon" is a CSBK, and a CRC-valid one is surfaced as a "site
// alive" signal (issue #1036) rather than being ignored.
func (c *ConventionalChannel) IngestBurst(b *dmr.Burst, slot dmr.SlotType) {
	c.cnt.bursts.Add(1)
	switch slot.DataType {
	case dmr.DTVoiceLCHeader:
		c.handleVoiceHeader(b, slot)
	case dmr.DTTerminatorWithLC:
		c.handleTerminator()
	case dmr.DTCSBK:
		c.handleCSBK(b, slot)
	}
}

// handleCSBK processes a Control Signaling Block burst on the parked
// conventional frequency. Issue #1036 asks for conventional DMR / IPSC
// monitoring where the repeater drops to silence between calls and emits
// periodic idle "beacons" — short transmissions that carry valid sync +
// colour code but no voice — so the scanner can log the site as alive
// instead of flagging the gaps as corrupt data. On DMR that beacon is a
// CSBK (typically a Preamble or C_BCAST). Tier III owns CSBK *trunking*
// semantics (grants, aloha); here we only need the keep-alive fact, so a
// CRC-valid CSBK bumps the Beacons counter and (rate-limited) logs "site
// alive".
//
// The gate is CRC-strict, mirroring handleVoiceHeader's FEC discipline:
// a noise burst that false-syncs would have to survive BPTC(196,96)
// correction *and* match the 16-bit CSBK CRC (mask 0x5A5A) to count, so
// it cannot forge a beacon the way a bare slot-type decode could (the
// "instalock cc=15" trap this package already guards against). A
// BPTC-uncorrectable or CRC-failing CSBK is dropped at Debug and,
// deliberately, is NOT published as a KindDecodeError: between-beacon
// noise on a parked conventional channel is expected, and surfacing it as
// a decode error is exactly the "treated as failed/corrupted data"
// symptom #1036 reports.
func (c *ConventionalChannel) handleCSBK(b *dmr.Burst, slot dmr.SlotType) {
	payload := b.PayloadBits()
	bits, errs := framing.DecodeBPTC196_96(payload)
	if errs < 0 {
		c.log.Debug("dmr/tier2: CSBK BPTC uncorrectable (between-beacon noise)", "cc", slot.ColorCode)
		return
	}
	csbk, err := tier3.ParseCSBK(infoBitsToBytes(bits))
	if err != nil {
		c.log.Debug("dmr/tier2: CSBK CRC mismatch (between-beacon noise)", "cc", slot.ColorCode)
		return
	}
	c.cnt.beacons.Add(1)

	// Rate-limit the operator-facing Info line so a fast beacon interval
	// doesn't flood the log; the counter still records every beacon.
	c.mu.Lock()
	now := c.now()
	due := c.beaconLogAt.IsZero() || now.Sub(c.beaconLogAt) >= beaconLogInterval
	if due {
		c.beaconLogAt = now
	}
	c.mu.Unlock()
	if due {
		c.log.Info("dmr/tier2 site alive (beacon)",
			"freq", c.freqHz, "cc", slot.ColorCode,
			"csbk", csbk.Opcode.String(), "system", c.systemName)
	} else {
		c.log.Debug("dmr/tier2: beacon", "cc", slot.ColorCode, "csbk", csbk.Opcode.String())
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
	c.cnt.locks.Add(1)
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
		c.cnt.fecFail.Add(1)
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
		c.cnt.fecFail.Add(1)
		c.log.Debug("dmr/tier2: voice header RS(12,9) parity mismatch",
			"cc", slot.ColorCode,
			"info_hex", hex.EncodeToString(infoBytes))
		c.bus.Publish(events.Event{
			Kind:    events.KindDecodeError,
			Payload: events.DecodeError{Protocol: c.protocolTag, Stage: events.StageVoiceHeaderRS},
		})
		return
	}
	// BPTC + RS both passed: this is a genuine DMR transmission on the
	// tuned frequency. Declare the lock here (not on the slot type alone)
	// so a false sync / miscorrected slot type can't forge it.
	c.cnt.fecPass.Add(1)
	c.maybeLock(LockState{FrequencyHz: c.freqHz, ColorCode: slot.ColorCode})
	flc, err := dmr.ParseFLC(infoBytes)
	if err != nil {
		c.log.Debug("dmr/tier2: FLC parse failed", "err", err)
		return
	}
	// A Voice LC Header names either a group call (destination is a
	// talkgroup) or a private unit-to-unit call (destination is a
	// subscriber). Both are followed on the tuned frequency; the only
	// difference to the engine is Grant.Individual, which keeps a private
	// call's destination RID out of the talkgroup list.
	var dest, src uint32
	var prio uint8
	var enc, emer, individual bool
	if gv, ok := flc.AsGroupVoiceUser(); ok {
		dest, src, enc, emer, prio = gv.GroupAddress, gv.SourceID, gv.Encrypted, gv.Emergency, gv.Priority
	} else if uu, ok := flc.AsUnitToUnitVoice(); ok {
		dest, src, enc, emer, individual, prio = uu.DestinationID, uu.SourceID, uu.Encrypted, uu.Emergency, true, uu.Priority
	} else {
		c.log.Debug("dmr/tier2: non-voice FLCO ignored", "flco", flc.FLCO)
		return
	}
	if c.inCall && c.lastTG == dest && c.lastSrc == src {
		// Same call's repeated Voice LC Header — dedupe.
		return
	}
	c.inCall = true
	c.lastTG = dest
	c.lastSrc = src
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:      c.systemName,
			Protocol:    c.protocolTag,
			GroupID:     dest,
			SourceID:    src,
			Individual:  individual,
			FrequencyHz: c.freqHz,
			ChannelID:   slot.ColorCode,
			Encrypted:   enc,
			Emergency:   emer,
			Priority:    prio,
			At:          c.now(),
		},
	})
	c.log.Debug("dmr/tier2: grant",
		"system", c.systemName, "freq_hz", c.freqHz,
		"cc", slot.ColorCode, "dst", dest, "src", src,
		"individual", individual, "enc", enc, "emer", emer)
}

func (c *ConventionalChannel) handleTerminator() {
	if !c.inCall {
		return
	}
	// A Terminator with LC is the explicit end of the transmission. Publish a
	// call release so the engine ends the call at once, rather than waiting out
	// the composer's hangtime / no-voice timers — the same prompt-teardown path
	// TETRA's D-RELEASE drives. Keyed by (System, GroupID); a no-match release
	// is a harmless no-op.
	if c.bus != nil && c.lastTG != 0 {
		c.bus.Publish(events.Event{
			Kind: events.KindCallRelease,
			Payload: trunking.CallRelease{
				System:  c.systemName,
				GroupID: c.lastTG,
				Reason:  trunking.EndReasonReleased,
				At:      c.now(),
			},
		})
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
