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
	"bytes"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier3"
	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// beaconLogInterval rate-limits the operator-facing "site alive" Info log
// so a repeater whose idle beacon fires every few seconds announces
// itself once, then stays quiet — the Beacons counter keeps the exact
// count. See handleCSBK.
const beaconLogInterval = 30 * time.Second

// csbkFailLogInterval parks the Debug log for CSBK bursts that BPTC-decode
// but fail the CSBK CRC (or are BPTC-uncorrectable). A keyed idle IPSC
// repeater can put such a burst on BOTH timeslots every 30 ms (field
// report: one identical "CSBK CRC mismatch" line per burst for seconds at
// a time), so the first failure logs immediately — with the block's
// opcode/FID/LB/PF and raw info hex, the instrument that lets the next
// report pin what those bursts actually are — and unchanged repeats are
// summarised at this interval with the suppressed count. Mirrors the Tier
// III csbkRepeatLogInterval parking.
const csbkFailLogInterval = 10 * time.Second

// lateEntryConfirm is how many CRC-valid embedded-LC superframes must name
// the same (destination, source) before a transmission whose Voice LC
// Header was never decoded is granted by late entry. The embedded LC repeats
// every superframe (360 ms), so two agreeing copies cost ~720 ms of the
// transmission and rule out a lone miscorrected LC forging a phantom call.
const lateEntryConfirm = 2

// lateEntryWindow bounds how long a single unconfirmed late-entry candidate
// stays valid: a second agreeing LC must land within this much of the first
// or the candidate is restarted.
const lateEntryWindow = 3 * time.Second

// headerRekeyDibits separates the repeated copies of ONE transmission's Voice
// LC Header (a repeater sends 2–3 back-to-back copies at keyup, 60 ms / 288
// dibits apart on a two-slot carrier) from the header of a NEW transmission
// on the same (destination, source). A header for a tracked call that lands
// more than this far past the call's last header/LC is a re-key whose
// terminator went undecoded — the 10 Sep IPSC field report: the composer's
// voice-path detector released the call at PTT release, but the control
// path's copy of the Terminator-with-LC sat BPTC-uncorrectable for 0.8–6.5 s
// on its weaker channelizer tap, so the next transmission's header was
// "deduped" against a call the engine had already ended and the reply was
// never recorded. 0.25 s (1200 dibits): the copies of one header span
// ~120–180 ms, while a re-key's header is always at least one superframe
// (360 ms) past the previous over's last superframe start — the operator's
// 10 Sep capture has one reply keyed up so fast that a 0.5 s rule missed it.
const headerRekeyDibits = 1200

// superframeRekeyGapDibits is the same re-key rule for a transmission whose
// header was ALSO lost: a CRC-valid embedded LC naming a tracked call after
// this much silence (no header / LC for the call) is a new transmission and
// re-grants by late entry. Measured from the START of the last superframe
// that named the call, so 2 s (9600 dibits) is ~1.6 s of actual silence:
// superframes of one transmission are 1728 dibits (360 ms) apart, so a
// four-superframe fade mid-over is tolerated, while a reply that follows a
// typical ≥3 s repeater hang time is caught even when its header is lost.
const superframeRekeyGapDibits = 9600

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
	// Dibits counts demodulated dibits handed to Process — the receiver's
	// output rate (4800/s on a live stream). It separates "the receiver
	// emits nothing" from "the receiver emits junk that never syncs", which
	// sync_hits alone cannot (the 12 Sep IPSC field log: a tap deaf for
	// three-minute stretches at a steady -51 dBFS with sync_hits=0).
	Dibits   uint64
	SyncHits uint64 // FSW matches reported by the burst-sync detector
	Bursts   uint64 // slot-type-parsed bursts handed to IngestBurst
	FECPass  uint64 // Voice LC Header with BPTC + RS both valid
	FECFail  uint64 // Voice LC Header BPTC uncorrectable or RS mismatch
	Locks    uint64 // cc.locked declarations (lifetime)
	Beacons  uint64 // CRC-valid CSBK "site alive" bursts (issue #1036)
	// DroppedOffCC counts bursts discarded because a configured colour-code
	// filter (Options.ColorCodeFilter) did not match the burst's decoded
	// colour code. Zero unless the operator pinned a colour code (the DMR
	// IPSC / linked-repeater profile) to ignore other systems sharing the
	// wideband passband.
	DroppedOffCC uint64
	// LateEntries counts grants raised by late entry — a transmission whose
	// Voice LC Header bursts never decoded (lost to a fade / bin-edge SNR at
	// keyup) but whose embedded Link Control, repeated every voice
	// superframe, named the call. Zero when every transmission's header
	// decodes; a steadily climbing value on a live repeater is the signature
	// of a weak tap (the conversation is being caught mid-transmission).
	LateEntries uint64
	// CSBKCRCFail counts CSBK-typed bursts that passed BPTC(196,96) but failed
	// the CSBK CRC — either between-beacon noise, or a real proprietary CSBK
	// train GT does not yet understand (see handleCSBK's parked log).
	CSBKCRCFail uint64
	// Rekeys counts transmissions granted because a Voice LC Header (or, with
	// the header lost too, a fresh embedded LC after a silence gap) named a
	// call this channel still tracked — i.e. the previous transmission's
	// Terminator-with-LC never decoded here. Each one is a reply that would
	// have been missed before the re-key rule (see headerRekeyDibits).
	Rekeys uint64
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

	// colorFilter, when non-nil, restricts this channel to a single DMR
	// colour code: any burst whose decoded slot-type colour code differs is
	// dropped before it can grant, lock, or raise a decode error. This is the
	// "list of frequencies for one Colour Code" IPSC / linked-repeater profile
	// — it keeps a co-channel system on a different colour code (bleeding into
	// the shared wideband passband) from polluting the call log. nil ⇒ accept
	// every colour code (the historical default).
	colorFilter *uint8

	// interleavedVoice is stamped onto every grant as Grant.DMRInterleavedVoice
	// (see Options.InterleavedVoice) so the voice composer runs the two-slot
	// interleaved decoder rather than slicing both timeslots into one garbled
	// stream.
	interleavedVoice bool

	// proc is the cross-call dibit / sync state the Process adapter
	// uses (see process.go). Lazily constructed on the first
	// Process call.
	proc *processState

	mu     sync.Mutex
	locked bool
	last   LockState

	// calls tracks the in-progress calls on this repeater, keyed by decoded
	// destination (the talkgroup for a group call, the called subscriber for a
	// private call). A conventional DMR repeater carries two independent
	// timeslots (TS1/TS2), each able to hold a separate simultaneous call, so a
	// single scalar "current call" collapsed two concurrent talkgroups into one
	// — the two Voice LC Headers alternate and ping-pong the state, and both
	// grants carried Timeslot 0 so the engine folded them onto one (freq, 0)
	// channel. Keying by destination lets the two calls coexist; each is given a
	// synthetic Timeslot (see assignSlot) so the engine's (freq, timeslot) call
	// identity keeps them apart. The base-station wire format does not label a
	// burst's physical slot (both slots share the BS sync words), and the voice
	// composer routes audio by the embedded-LC talkgroup rather than this field,
	// so the synthetic slot is purely an engine-identity token and never
	// mis-routes audio. Touched only on the single decode goroutine (like the
	// scalar state it replaces), so it needs no lock; Counters() reads only the
	// atomic cnt below.
	calls map[uint32]*convCall

	// voice assembles voice superframes from the same dibit stream Process
	// slices bursts from, so the embedded Link Control (bursts B–E of every
	// superframe) can grant a transmission whose Voice LC Header never
	// decoded — DMR late entry, the mechanism every subscriber radio uses to
	// join a call in progress. Without it the conventional path had exactly
	// one chance per transmission (the header bursts at keyup, the weakest
	// moment of a PTT on a marginal tap): a header lost to a fade dropped the
	// whole over, which an operator with a radio beside the scanner sees as
	// "GT said the call ended but the conversation continued". Lazily built
	// on the first Process call (interleaved when InterleavedVoice is set,
	// single-slot for direct mode); nil until then, so IngestBurst-only
	// callers (tests, the Tier III adapter) are unaffected.
	voice *dmrvoice.Decoder

	// lateEntry holds the unconfirmed late-entry candidates keyed by
	// destination: the (source, first-seen) of an embedded LC that named a
	// call not in calls. A second agreeing LC within lateEntryWindow grants.
	lateEntry map[uint32]*lateEntryCandidate

	// endedAtDibit records, per destination, the absolute dibit index of the
	// Terminator-with-LC burst that last released its call. The voice
	// superframe assembler runs a span behind the burst slicer, so the
	// closing superframes of a transmission can surface AFTER its terminator
	// was processed; their embedded LCs belong to the call that just ended
	// and must not seed a late-entry re-grant of it (measured on air: 7
	// phantom grant/release pairs in a 120 s capture without this gate). An
	// LC whose superframe started before the terminator is ignored.
	endedAtDibit map[uint32]int
	// ingestDibit is the absolute dibit index of the burst IngestBurst is
	// currently handling when driven from Process (−1 otherwise).
	ingestDibit int

	// polarity is the discriminator polarity (0 identity, dmr.PolarityFlip)
	// this stream decodes at, learned from the first FEC-valid burst; −1 while
	// unknown, when both candidates are tried. DMR's data and voice sync
	// words are each other's flip image, so until the polarity is known a
	// voice burst A at the untried polarity looks like a data burst and gets
	// its (AMBE-bit) slot type parsed; once known, only real data-sync bursts
	// reach the slot-type path. A front end's inversion is a fixed property
	// of the stream, so the lock is never dropped.
	polarity int
	// burstValid is set by the FEC-validated decode paths (header BPTC+RS,
	// CSBK CRC, terminator LC) so Process can learn the polarity.
	burstValid bool

	// cnt holds the lock-free decode-activity counters exposed via
	// Counters(). Incremented on the existing hot paths with atomic
	// adds so any goroutine can snapshot them without taking c.mu.
	cnt struct {
		dibits       atomic.Uint64
		syncHits     atomic.Uint64
		bursts       atomic.Uint64
		fecPass      atomic.Uint64
		fecFail      atomic.Uint64
		locks        atomic.Uint64
		beacons      atomic.Uint64
		droppedOffCC atomic.Uint64
		lateEntries  atomic.Uint64
		csbkCRCFail  atomic.Uint64
		rekeys       atomic.Uint64
	}

	// beaconLogAt is the last time handleCSBK emitted an Info "site alive"
	// line; guarded by mu and used only to rate-limit that log.
	beaconLogAt time.Time

	// csbkFailLog parks the CSBK-failure Debug log (see csbkFailLogInterval).
	// Touched only on the decode goroutine.
	csbkFailLog struct {
		at         time.Time
		suppressed int
		lastKey    string
	}
}

// lateEntryCandidate is one unconfirmed late-entry call: the source the
// first embedded LC named for a destination, when it was seen, and how many
// agreeing LCs have accumulated.
type lateEntryCandidate struct {
	src     uint32
	firstAt time.Time
	seen    int
}

// Counters returns a snapshot of this channel's decode-activity
// counters. Safe to call concurrently with the decode path.
func (c *ConventionalChannel) Counters() Counters {
	return Counters{
		Dibits:       c.cnt.dibits.Load(),
		SyncHits:     c.cnt.syncHits.Load(),
		Bursts:       c.cnt.bursts.Load(),
		FECPass:      c.cnt.fecPass.Load(),
		FECFail:      c.cnt.fecFail.Load(),
		Locks:        c.cnt.locks.Load(),
		Beacons:      c.cnt.beacons.Load(),
		DroppedOffCC: c.cnt.droppedOffCC.Load(),
		LateEntries:  c.cnt.lateEntries.Load(),
		CSBKCRCFail:  c.cnt.csbkCRCFail.Load(),
		Rekeys:       c.cnt.rekeys.Load(),
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
	// ColorCodeFilter, when non-nil, pins this channel to a single DMR colour
	// code (0..15): bursts on any other colour code are dropped before grant /
	// lock / decode-error. nil ⇒ accept every colour code. See the colorFilter
	// field on ConventionalChannel.
	ColorCodeFilter *uint8
	// InterleavedVoice stamps Grant.DMRInterleavedVoice so the voice composer
	// runs the two-slot interleaved AMBE decoder instead of the single-slot one.
	// A base-station repeater carries TS1 and TS2 interleaved on one carrier, so
	// the single-slot decoder slices straight through both slots and produces
	// garbled ("DJ scratchy") audio; the interleaved decoder + slotRouter route
	// each slot's superframes by their embedded-LC talkgroup. Resolved per
	// system (true for conventional Tier II, false for Tier I direct mode, which
	// is genuinely single-slot). Mirrors tier3.Options.InterleavedVoice.
	InterleavedVoice bool
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
		ingestDibit:      -1,
		polarity:         -1,
		bus:              opts.Bus,
		log:              log,
		systemName:       opts.SystemName,
		freqHz:           opts.FrequencyHz,
		now:              now,
		protocolTag:      tag,
		syncPatterns:     opts.SyncPatterns,
		colorFilter:      opts.ColorCodeFilter,
		interleavedVoice: opts.InterleavedVoice,
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
	// Colour-code hard filter (IPSC / linked-repeater profile). The slot
	// type's colour code is Hamming(20,8)-corrected, so it is reliable enough
	// to reject a co-channel system on a different colour code before it can
	// grant, lock, or raise a decode error — which is exactly the noise the
	// profile exists to suppress. A rare slot-type FEC miscorrection can flip
	// the colour code on an otherwise-valid burst; dropping that one burst is
	// acceptable for an opt-in filter, and the next header on the same call
	// re-decodes cleanly. The filter is off (nil) by default, so the shared
	// off-air-colour-code path is byte-for-byte unchanged.
	if c.colorFilter != nil && slot.ColorCode != *c.colorFilter {
		c.cnt.droppedOffCC.Add(1)
		return
	}
	switch slot.DataType {
	case dmr.DTVoiceLCHeader:
		c.handleVoiceHeader(b, slot)
	case dmr.DTTerminatorWithLC:
		c.handleTerminator(b, slot)
	case dmr.DTCSBK:
		c.handleCSBK(b, slot)
	case dmr.DTIdle:
		c.handleIdle(b, slot)
	}
}

// IdleInfoPattern is the fixed 96-bit information block of the ETSI DMR Idle
// burst (TS 102 361-1 §9.3.x "Idle") as recovered through BPTC(196,96). It is
// what a Motorola / Hytera IPSC repeater transmits on BOTH timeslots while
// keyed but idle — the periodic ~10 s "beacon" trains between calls — and it
// is pinned two independent ways: the operator's 10 Sep 442.3875 MHz
// beacon-only capture decodes 2402/2416 synced bursts to exactly this block
// at colour code 12, and BPTC-decoding MMDVMHost's DMR_IDLE_DATA burst
// constant (DMRDefines.h) yields the same 12 bytes
// (TestIdleInfoPatternMatchesMMDVMHostConstant).
var IdleInfoPattern = [12]byte{0xff, 0x83, 0xdf, 0x17, 0x32, 0x09, 0x4e, 0xd1, 0xe7, 0xcd, 0x8a, 0x91}

// handleIdle processes an Idle-typed burst. A conventional/IPSC repeater
// that is keyed with nothing to carry fills both slots with Idle bursts, so
// on this profile Idle IS the idle beacon (the operator's "camp / idle
// beacon" request, 10 Sep): before this handler Idle fell through
// IngestBurst's switch, Beacons stayed 0 on a repeater that was audibly
// keyed for ten seconds at a time, the channel never reported locked until
// someone spoke, and the wideband engine's "strong in-channel signal but no
// sync" hint fired on every beacon train.
//
// The gate is content-strict, mirroring handleCSBK: the burst must BPTC-
// decode AND its information block must equal IdleInfoPattern, so a noise
// burst that false-syncs and happens to Golay-decode to (cc, Idle) cannot
// forge a beacon — it would also have to be a BPTC codeword carrying that
// exact 96-bit block. A BPTC-clean Idle with a DIFFERENT block is a vendor
// variant GT has not seen: it is dropped, and its info hex is logged
// (parked) so the next field log pins it. Not a decode error on the bus.
func (c *ConventionalChannel) handleIdle(b *dmr.Burst, slot dmr.SlotType) {
	bits, errs := framing.DecodeBPTC196_96(b.PayloadBits())
	if errs < 0 {
		return
	}
	info := infoBitsToBytes(bits)
	if !bytes.Equal(info, IdleInfoPattern[:]) {
		c.logCSBKFailure("dmr/tier2: idle burst with an unknown information block (vendor idle variant?)", slot, nil, info)
		return
	}
	c.cnt.beacons.Add(1)
	c.burstValid = true
	// A keyed repeater sending the ETSI idle pattern at this colour code is
	// as strong a "this carrier is a DMR repeater" proof as a voice header,
	// so camp on it: the channel reports locked (site alive) while idle.
	c.maybeLock(LockState{FrequencyHz: c.freqHz, ColorCode: slot.ColorCode})
	c.mu.Lock()
	now := c.now()
	due := c.beaconLogAt.IsZero() || now.Sub(c.beaconLogAt) >= beaconLogInterval
	if due {
		c.beaconLogAt = now
	}
	c.mu.Unlock()
	if due {
		c.log.Info("dmr/tier2 site alive (idle beacon)",
			"freq", c.freqHz, "cc", slot.ColorCode, "system", c.systemName)
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
		c.logCSBKFailure("dmr/tier2: CSBK BPTC uncorrectable (between-beacon noise)", slot, nil, nil)
		return
	}
	info := infoBitsToBytes(bits)
	csbk, err := tier3.ParseCSBK(info)
	if err != nil {
		// BPTC(196,96) passed — the burst is well received — but the 16-bit
		// CSBK CRC (ETSI mask, pinned by real Tier III vectors) does not check.
		// On a keyed idle IPSC repeater this can be EVERY burst on both slots
		// for seconds (a vendor-proprietary CSBK train, not noise), so the log
		// is parked and carries the decoded header + raw block so the next
		// field log pins what the train is. Not a decode error on the bus.
		c.cnt.csbkCRCFail.Add(1)
		c.logCSBKFailure("dmr/tier2: CSBK CRC mismatch (between-beacon noise or proprietary CSBK train)", slot, &csbk, info)
		return
	}
	c.cnt.beacons.Add(1)
	c.burstValid = true

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

// logCSBKFailure emits the parked Debug line for a CSBK burst that failed
// BPTC or CRC: the first occurrence of a (message, cc, opcode, fid) key logs
// at once with the block's header fields + raw info hex; repeats within
// csbkFailLogInterval are counted and summarised on the next emit. A change
// of key (different colour code / opcode) logs immediately so a real
// transition is never hidden behind the parking.
func (c *ConventionalChannel) logCSBKFailure(msg string, slot dmr.SlotType, csbk *tier3.CSBK, info []byte) {
	key := msg + "|" + strconv.Itoa(int(slot.ColorCode))
	attrs := []any{"cc", slot.ColorCode}
	if csbk != nil {
		key += "|" + strconv.Itoa(int(csbk.Opcode)) + "|" + strconv.Itoa(int(csbk.FID))
		attrs = append(attrs,
			"csbko", fmt.Sprintf("0x%02x", uint8(csbk.Opcode)), "fid", fmt.Sprintf("0x%02x", csbk.FID),
			"lb", csbk.LB, "pf", csbk.PF, "info_hex", hex.EncodeToString(info))
	} else if info != nil {
		key += "|" + hex.EncodeToString(info)
		attrs = append(attrs, "info_hex", hex.EncodeToString(info))
	}
	now := c.now()
	st := &c.csbkFailLog
	if !st.at.IsZero() && st.lastKey == key && now.Sub(st.at) < csbkFailLogInterval {
		st.suppressed++
		return
	}
	if st.suppressed > 0 {
		attrs = append(attrs, "suppressed_repeats", st.suppressed)
	}
	st.at, st.lastKey, st.suppressed = now, key, 0
	c.log.Debug(msg, attrs...)
}

// ingestVoiceSuperframe is the late-entry path: every CRC-valid embedded
// Link Control that names a group / unit-to-unit voice call either refreshes
// the tracked call it belongs to or, for a destination no Voice LC Header
// ever granted, accumulates towards a late-entry grant (lateEntryConfirm
// agreeing superframes within lateEntryWindow). The optional colour-code
// filter is honoured through the superframe's majority EMB colour code.
func (c *ConventionalChannel) ingestVoiceSuperframe(sf dmrvoice.VoiceSuperframe) {
	if !sf.HasLC {
		return
	}
	if c.colorFilter != nil && (!sf.HasEMB || sf.EMBColorCode != *c.colorFilter) {
		c.cnt.droppedOffCC.Add(1)
		return
	}
	var dest, src uint32
	var prio uint8
	var enc, emer, individual bool
	if gv, ok := sf.LC.AsGroupVoiceUser(); ok {
		dest, src, enc, emer, prio = gv.GroupAddress, gv.SourceID, gv.Encrypted, gv.Emergency, gv.Priority
	} else if uu, ok := sf.LC.AsUnitToUnitVoice(); ok {
		dest, src, enc, emer, individual, prio = uu.DestinationID, uu.SourceID, uu.Encrypted, uu.Emergency, true, uu.Priority
	} else {
		return
	}
	if dest == 0 || src == 0 {
		return
	}
	if end, ok := c.endedAtDibit[dest]; ok && sf.StartDibit < end {
		// Closing superframe of a call whose terminator already released it.
		return
	}
	now := c.now()
	if existing, ok := c.calls[dest]; ok {
		if existing.src == src && existing.lastDibit >= 0 &&
			sf.StartDibit-existing.lastDibit > superframeRekeyGapDibits {
			// The call went silent for longer than any fade a transmission
			// survives and then its (dest, src) is back in a fresh embedded
			// LC: a re-key whose header AND whose predecessor's terminator
			// were both lost. Release the stale call and re-grant by late
			// entry, exactly as if the call had never been tracked.
			c.cnt.rekeys.Add(1)
			c.cnt.lateEntries.Add(1)
			c.log.Debug("dmr/tier2: re-key by late entry — embedded LC for a still-tracked call after a silence gap",
				"dst", dest, "src", src, "gap_dibits", sf.StartDibit-existing.lastDibit)
			c.releaseCall(dest)
			ts := c.assignSlot(now)
			c.calls[dest] = newConvCall(src, ts, now, sf.StartDibit)
			c.publishGrant(dest, src, individual, enc, emer, prio, ts, sf.EMBColorCode, true)
			return
		}
		// The call is already tracked (header-granted or late-entered): the
		// LC is liveness for slot reuse. A source change mid-call is a talker
		// change on the same slot — republish, as the header path does.
		existing.lastAt = now
		existing.touch(sf.StartDibit)
		if existing.src != src {
			existing.src = src
			existing.anchorDibit = sf.StartDibit
			c.publishGrant(dest, src, individual, enc, emer, prio, existing.slot, sf.EMBColorCode, true)
		}
		return
	}
	if c.lateEntry == nil {
		c.lateEntry = make(map[uint32]*lateEntryCandidate)
	}
	cand := c.lateEntry[dest]
	if cand == nil || cand.src != src || now.Sub(cand.firstAt) > lateEntryWindow {
		c.lateEntry[dest] = &lateEntryCandidate{src: src, firstAt: now, seen: 1}
		if lateEntryConfirm > 1 {
			return
		}
		cand = c.lateEntry[dest]
	} else {
		cand.seen++
		if cand.seen < lateEntryConfirm {
			return
		}
	}
	delete(c.lateEntry, dest)
	// Two agreeing CRC-valid embedded LCs are as strong an on-air proof as a
	// BPTC+RS-clean header: declare the lock too, so a camped channel whose
	// first transmission lost its header still reports locked.
	c.cnt.lateEntries.Add(1)
	if c.polarity < 0 {
		// The assembler ran on identity-polarity dibits (see Process), so two
		// CRC-valid LCs also fix the stream's polarity at identity.
		c.polarity = 0
	}
	c.maybeLock(LockState{FrequencyHz: c.freqHz, ColorCode: sf.EMBColorCode})
	if c.calls == nil {
		c.calls = make(map[uint32]*convCall)
	}
	ts := c.assignSlot(now)
	c.calls[dest] = newConvCall(src, ts, now, sf.StartDibit)
	c.publishGrant(dest, src, individual, enc, emer, prio, ts, sf.EMBColorCode, true)
}

// publishGrant publishes one trunking.Grant for this channel and logs it.
// lateEntry marks a grant raised from the embedded LC rather than a header.
func (c *ConventionalChannel) publishGrant(dest, src uint32, individual, enc, emer bool, prio uint8, ts uint8, cc uint8, lateEntry bool) {
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:              c.systemName,
			Protocol:            c.protocolTag,
			GroupID:             dest,
			SourceID:            src,
			Individual:          individual,
			FrequencyHz:         c.freqHz,
			ChannelID:           cc,
			Timeslot:            ts,
			DMRInterleavedVoice: c.interleavedVoice,
			Encrypted:           enc,
			Emergency:           emer,
			Priority:            prio,
			At:                  c.now(),
		},
	})
	c.log.Debug("dmr/tier2: grant",
		"system", c.systemName, "freq_hz", c.freqHz,
		"cc", cc, "dst", dest, "src", src,
		"individual", individual, "enc", enc, "emer", emer,
		"late_entry", lateEntry)
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

// ResetLateEntry drops any unconfirmed late-entry candidates and the voice
// superframe assembler's buffered state, e.g. after a retune.
func (c *ConventionalChannel) ResetLateEntry() {
	c.lateEntry = nil
	c.endedAtDibit = nil
	if c.voice != nil {
		c.voice.Reset()
	}
}

// ResyncReset drops the Process adapter's cross-call dibit buffer, its pending
// sync matches and the late-entry / superframe state so a receiver-side Reset
// (which restarts the dibit index at 0) can reacquire cleanly. Without it the
// adapter's absolute bufStart stays at the pre-reset value while every new sync
// match arrives at a small index, so each is discarded as "lookback already
// trimmed" for ever — the trap tier3.ResyncReset documents. Tracked calls are
// kept (the engine's own timers end them); the learned polarity is kept too, a
// front end's inversion being a fixed property of the stream.
//
// Precondition: called on the same goroutine as Process (from the engine, after
// the receiver's Process returns), so the proc swap never races a Process call.
func (c *ConventionalChannel) ResyncReset() {
	c.proc = nil
	c.ResetLateEntry()
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
	c.burstValid = true
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
	// Assign or refresh this call's synthetic timeslot. Concurrent calls on the
	// repeater's two slots carry distinct destinations, so key by dest: a
	// repeated header for the same (dest, src) is a dedupe; a header for a dest
	// already in call whose source changed is a talker change on the same slot;
	// a new dest claims the lowest free synthetic slot.
	now := c.now()
	if c.calls == nil {
		c.calls = make(map[uint32]*convCall)
	}
	var ts uint8
	if existing, ok := c.calls[dest]; ok {
		if existing.src == src {
			if c.ingestDibit < 0 || existing.anchorDibit < 0 ||
				c.ingestDibit-existing.anchorDibit <= headerRekeyDibits {
				// Same transmission's repeated Voice LC Header copy — or a
				// header that surfaced behind the superframes that already
				// (re-)granted this transmission by late entry. Dedupe.
				existing.lastAt = now
				existing.touch(c.ingestDibit)
				return
			}
			// A Voice LC Header is only ever sent at keyup, so one this far
			// past the tracked call's own header is a NEW transmission whose
			// predecessor's terminator this channel never decoded. Release
			// the stale call (idempotent for an engine the voice path
			// already released) and grant the re-key.
			c.cnt.rekeys.Add(1)
			c.log.Debug("dmr/tier2: re-key — Voice LC Header for a still-tracked call; previous terminator undecoded here",
				"dst", dest, "src", src, "gap_dibits", c.ingestDibit-existing.anchorDibit)
			c.releaseCall(dest)
			ts = c.assignSlot(now)
			c.calls[dest] = newConvCall(src, ts, now, c.ingestDibit)
			c.publishGrant(dest, src, individual, enc, emer, prio, ts, slot.ColorCode, false)
			return
		}
		// Same destination, new talker: keep the slot, republish with the new
		// source so the engine surfaces the talker change.
		existing.lastAt = now
		existing.src = src
		existing.anchorDibit = c.ingestDibit
		existing.touch(c.ingestDibit)
		ts = existing.slot
	} else {
		ts = c.assignSlot(now)
		c.calls[dest] = newConvCall(src, ts, now, c.ingestDibit)
	}
	// A header grant supersedes any pending late-entry candidate for the
	// destination (the header is the stronger evidence and arrives first).
	delete(c.lateEntry, dest)
	c.publishGrant(dest, src, individual, enc, emer, prio, ts, slot.ColorCode, false)
}

func (c *ConventionalChannel) handleTerminator(b *dmr.Burst, slot dmr.SlotType) {
	if len(c.calls) == 0 {
		return
	}
	// Resolve which of the (up to two) concurrent calls this terminator ends
	// from its own Full LC destination, so a TS1 terminator cannot tear down the
	// TS2 call. The Terminator-with-LC carries the same FLC as the Voice LC
	// Header — group talkgroup or unit-to-unit destination — protected by
	// BPTC(196,96) + RS(12,9) with the terminator parity seed.
	dest, ok, bptcOK := c.terminatorDest(b)
	if ok {
		c.burstValid = true
	}
	if !ok {
		// The terminator's Full LC did not decode. With a single call active the
		// terminator is unambiguous, so end that call (preserving the prompt
		// teardown Tier II has always done) — but only if its BPTC(196,96)
		// block decoded (an RS/FLC mismatch on a well-received burst). A burst
		// whose slot type says Terminator but whose payload is not even a BPTC
		// codeword is a forged slot type, not a weak terminator: a repeater
		// repeats the real Terminator-with-LC for its whole hang time (50–170
		// copies on the 9 Sep captures), so a genuine end never depends on one
		// uncorrectable burst, while a false one ends a live call mid-sentence.
		// With two concurrent calls it is ambiguous — releasing by a guess could
		// cross-tear the other slot — so release nothing and let each voice
		// chain's own terminator detector + hangtime end its call.
		if !bptcOK {
			c.log.Debug("dmr/tier2: terminator slot type without a BPTC-valid payload ignored", "cc", slot.ColorCode)
			return
		}
		if len(c.calls) != 1 {
			c.log.Debug("dmr/tier2: ambiguous terminator (LC undecodable, two calls active); leaving to hangtime",
				"cc", slot.ColorCode)
			return
		}
		for d := range c.calls {
			dest = d
		}
	}
	cc, tracked := c.calls[dest]
	if !tracked {
		// Terminator for a call we aren't tracking (already ended, or its header
		// was never decoded) — harmless no-op.
		return
	}
	// A Terminator with LC is the explicit end of the transmission. Publish a
	// call release so the engine ends the call at once, rather than waiting out
	// the composer's hangtime / no-voice timers — the same prompt-teardown path
	// TETRA's D-RELEASE drives.
	c.releaseCall(dest)
	c.log.Debug("dmr/tier2: terminator", "dst", dest, "slot", cc.slot)
}

// releaseCall forgets the tracked call on dest, records where in the stream it
// ended (so the closing superframes that surface behind the slicer cannot
// re-grant it) and publishes one call.release keyed by (System, GroupID) —
// which uniquely names this call among the concurrent slots. The engine's
// release handling is idempotent, so releasing a call the voice path already
// ended is harmless.
func (c *ConventionalChannel) releaseCall(dest uint32) {
	delete(c.calls, dest)
	delete(c.lateEntry, dest)
	if c.ingestDibit >= 0 {
		if c.endedAtDibit == nil {
			c.endedAtDibit = make(map[uint32]int)
		}
		c.endedAtDibit[dest] = c.ingestDibit
	}
	if c.bus != nil && dest != 0 {
		c.bus.Publish(events.Event{
			Kind: events.KindCallRelease,
			Payload: trunking.CallRelease{
				System:  c.systemName,
				GroupID: dest,
				Reason:  trunking.EndReasonReleased,
				At:      c.now(),
			},
		})
	}
}

// terminatorDest decodes a Terminator-with-LC burst's Full LC and returns the
// call destination it names (talkgroup or called subscriber), or ok=false if
// the embedded LC fails FEC. It mirrors handleVoiceHeader's BPTC + RS(12,9)
// pipeline but with the terminator parity seed.
//
// bptcOK reports whether the burst's BPTC(196,96) block decoded at all, so
// the caller can tell a weak-but-real terminator (BPTC ok, RS/FLC mismatch)
// from a forged slot type on a non-BPTC payload.
func (c *ConventionalChannel) terminatorDest(b *dmr.Burst) (dest uint32, ok bool, bptcOK bool) {
	bits, errs := framing.DecodeBPTC196_96(b.PayloadBits())
	if errs < 0 {
		return 0, false, false
	}
	infoBytes := infoBitsToBytes(bits)
	if !framing.VerifyRS12_9(infoBytes, framing.RS129SeedTerminatorLC) {
		return 0, false, true
	}
	flc, err := dmr.ParseFLC(infoBytes)
	if err != nil {
		return 0, false, true
	}
	if gv, ok := flc.AsGroupVoiceUser(); ok {
		return gv.GroupAddress, true, true
	}
	if uu, ok := flc.AsUnitToUnitVoice(); ok {
		return uu.DestinationID, true, true
	}
	return 0, false, true
}

// convCall is one in-progress conventional-DMR call. slot is the synthetic
// timeslot (1 or 2) assigned to keep concurrent calls apart in the engine's
// (frequency, timeslot) call identity; lastAt tracks liveness for slot reuse.
type convCall struct {
	src    uint32
	slot   uint8
	lastAt time.Time
	// anchorDibit is the absolute dibit index of the evidence that most
	// recently STARTED this call's current transmission: the last Voice LC
	// Header copy, or the embedded LC that (re-)granted it by late entry.
	// The header re-key rule measures against it. lastDibit is the index of
	// the last header or CRC-valid embedded LC of any kind that named the
	// call; the silence-gap re-key rule measures against it. Both are −1
	// when unknown (a burst ingested directly rather than through Process).
	// They are stream-time clocks — wall-clock would not survive an offline
	// replay that runs faster than real time — and they are kept separately
	// because Process assembles a chunk's superframes BEFORE it slices the
	// chunk's data bursts, so a new transmission's first superframe can
	// refresh the call ahead of that transmission's own header: measured
	// against lastDibit that header would look like a stale copy.
	anchorDibit int
	lastDibit   int
}

// newConvCall tracks a call granted by evidence at absolute dibit index
// dibit (−1 when unknown).
func newConvCall(src uint32, slot uint8, at time.Time, dibit int) *convCall {
	return &convCall{src: src, slot: slot, lastAt: at, anchorDibit: dibit, lastDibit: dibit}
}

// touch records further evidence of the call at dibit; positions only ever
// move forward, so evidence that surfaces out of stream order (a header
// sliced one chunk behind the superframe that followed it) cannot rewind.
func (cc *convCall) touch(dibit int) {
	if dibit > cc.lastDibit {
		cc.lastDibit = dibit
	}
}

// assignSlot returns a synthetic timeslot (1 then 2) for a new concurrent call.
// A conventional DMR carrier runs at most two calls (one per TDMA slot); if
// both synthetic slots are already in use — only possible when a prior call's
// terminator was missed and its state went stale — the stalest call is evicted
// and its slot reused. The engine's own call timeout reaps the evicted call.
func (c *ConventionalChannel) assignSlot(now time.Time) uint8 {
	var used1, used2 bool
	var stalestDest uint32
	var stalestAt time.Time
	first := true
	for dest, cc := range c.calls {
		switch cc.slot {
		case 1:
			used1 = true
		case 2:
			used2 = true
		}
		if first || cc.lastAt.Before(stalestAt) {
			stalestDest, stalestAt, first = dest, cc.lastAt, false
		}
	}
	if !used1 {
		return 1
	}
	if !used2 {
		return 2
	}
	slot := c.calls[stalestDest].slot
	delete(c.calls, stalestDest)
	return slot
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
