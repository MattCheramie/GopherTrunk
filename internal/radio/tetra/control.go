package tetra

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// LockState is the payload of cc.locked / cc.lost events emitted by
// the TETRA TMO control-channel state machine.
type LockState struct {
	FrequencyHz  uint32
	MCC          uint16 // first MLE-SYSINFO MCC, when seen
	MNC          uint16 // first MLE-SYSINFO MNC, when seen
	LocationArea uint16
}

// LockedFrequencyHz / LockedNAC make LockState satisfy
// trunking.LockedPayload so the cchunt supervisor's state machine
// recognises TETRA lock events alongside the protocol-neutral P25 /
// DMR / NXDN payloads. TETRA doesn't have a P25-style NAC; the
// LocationArea is the closest per-cell identifier and gets plumbed
// into the NAC slot. Without these methods, the supervisor's
// type-assertion on cc.locked silently drops the event and
// /api/v1/scanner never surfaces state=locked.
func (s LockState) LockedFrequencyHz() uint32 { return s.FrequencyHz }
func (s LockState) LockedNAC() uint16         { return s.LocationArea }

// ControlChannel ingests TETRA Layer-3 PDUs from a single control
// channel, emits cc.locked the first time a valid MLE-SYSINFO (or
// any non-idle CMCE PDU) arrives on a freshly-tuned device, and
// republishes voice grants as events.KindGrant carrying a
// `trunking.Grant` payload with `Protocol = "tetra"`. Same shape as
// the other trunked-protocol control channels.
type ControlChannel struct {
	bus        *events.Bus
	log        *slog.Logger
	systemName string
	freqHz     uint32
	resolver   Resolver
	now        func() time.Time

	// fecObserver, when non-nil, receives the Viterbi/FEC correction
	// depth (the number of channel bits the §8.3.1 decode chain
	// corrected) each time a signalling burst is recovered CRC-clean.
	// It is the hook the metrics layer wires the
	// gophertrunk_tetra_viterbi_corrections histogram onto; nil ⇒ no
	// computation and zero overhead (the production default unless
	// metrics.detailed_fec is enabled).
	fecObserver func(channel string, corrections int)

	// proc is the cross-call dibit / sync state the Process
	// adapter uses (see process.go). Lazily constructed on the
	// first Process call.
	proc *processState

	// debug is set once in New when the logger is at debug level. It gates
	// the decode-health counter accumulation (bumpStat) so a production
	// (info-level) decode carries no counter overhead — same nil-guard
	// spirit as fecObserver above.
	debug bool
	// statsMu guards stats independently of mu: the counters are bumped
	// from the decode path (dispatchSlice / decodeSB / publishGrant), some
	// of which run outside mu or call methods that take mu, so a separate
	// lock avoids any re-entrancy. DrainStats reads and resets under it.
	statsMu sync.Mutex
	stats   Stats

	mu               sync.Mutex
	locked           bool
	last             LockState
	strictValidation bool
	channelCoding    ChannelCodingMode
	channelType      ChannelType
	colourCode       uint32
	colourLearned    bool
	// colourConfirmed marks the learned colour code as corroborated by more
	// than one BSCH (or set by the operator), after which it is authoritative
	// and a single mis-decoded sync burst can no longer change it. colourTally
	// counts how many BSCHs agreed on each candidate extended colour code until
	// one crosses colourConfirmThreshold. See LearnColourCode.
	colourConfirmed bool
	colourTally     map[uint32]int
	// mainCarrier is the cell's own carrier number, learned from the
	// broadcast SYSINFO. With the tuned control-channel frequency it lets a
	// grant's carrier number resolve to Hz relative to this carrier, without a
	// configured band plan (see publishGrant).
	mainCarrier    uint16
	mainCarrierSet bool

	// sysInfo holds the cell's full SYSINFO frequency parameters (band, offset,
	// duplex spacing, reverse operation) once a SYSINFO broadcast is decoded.
	// With these the cell's carriers resolve to their *absolute* frequency
	// including the ±6.25/12.5 kHz offset field — a carrier a cell broadcasts as
	// 469.875 MHz actually sits at 469.88125 MHz. sysInfoSet gates the absolute
	// path; without it carrierFrequency falls back to 25 kHz spacing relative to
	// the tuned control frequency (see carrierFrequency).
	sysInfo    SysInfo
	sysInfoSet bool

	// individuals is the set of SSIs observed acting as a CMCE calling /
	// transmitting party (msg.PartySSI). A party is always an individual radio, so
	// this lets classifyParties recognise a grant addressed to a subscriber ISSI
	// (e.g. a D-CONNECT to the calling party, which carries no party field of its
	// own) and flag it Individual instead of surfacing it as a talkgroup. Bounded
	// by individualsCap; guarded by mu.
	individuals map[uint32]struct{}

	// callGroups maps a CMCE 14-bit call identifier to the group SSI (GSSI) the
	// call is addressed to, learned from D-SETUP/D-CONNECT. A D-RELEASE /
	// D-TX-GRANTED carries only the call identifier, so this resolves it back to
	// the talkgroup the engine keys active calls by. Lazily built; entries are
	// dropped on release. Guarded by mu.
	callGroups map[uint16]uint32

	// fragTMSDU accumulates a TM-SDU that spans a start-fragment MAC-RESOURCE and
	// one or more following MAC-FRAG / MAC-END PDUs (§21.4.3.3/.4). fragResource
	// is the start fragment's MAC-RESOURCE context (address + channel allocation),
	// needed to act on the reassembled L3 CMCE PDU because MAC-FRAG/MAC-END carry
	// no address of their own. fragActive marks a reassembly in progress. A single
	// TM-SDU is in flight at a time (a new start fragment replaces an incomplete
	// one); bounded by fragMaxBits and cleared on reassembly or resync. Guarded by
	// mu.
	fragTMSDU    []byte
	fragResource MACResource
	fragActive   bool

	// pendingSoft holds the per-symbol complex differential (soft
	// information) for the next Process call, stashed by StashSoft
	// immediately before the matching dibit chunk arrives (see
	// receiver SoftSink). Process consumes it via takeStashSoft.
	pendingSoft     []complex64
	pendingSoftBase int
	pendingSoftSet  bool

	// lastActivityNano is the Unix-nanos timestamp of the most recent
	// successful control-channel decode (a synchronisation burst, SYSINFO,
	// or grant). It is the heartbeat CheckStale watches to declare the lock
	// lost when the carrier goes silent. Atomic so the decode hot path
	// (noteActivity) never contends on mu. Zero until the first decode.
	lastActivityNano atomic.Int64
}

// StashSoft records the soft per-symbol differentials for the dibit
// chunk that arrives next at Process (same baseIdx). The receiver's
// SoftSink calls this just before DibitSink → Process. diffs is copied
// since the caller reuses its buffer.
func (c *ControlChannel) StashSoft(diffs []complex64, baseIdx int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cap(c.pendingSoft) < len(diffs) {
		c.pendingSoft = make([]complex64, len(diffs))
	} else {
		c.pendingSoft = c.pendingSoft[:len(diffs)]
	}
	copy(c.pendingSoft, diffs)
	c.pendingSoftBase = baseIdx
	c.pendingSoftSet = true
}

// takeStashSoft returns the stashed soft differentials if they match the
// chunk at baseIdx with length n, then clears the stash. Returns nil
// when no matching soft data was stashed (hard-only path).
func (c *ControlChannel) takeStashSoft(baseIdx, n int) []complex64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.pendingSoftSet || c.pendingSoftBase != baseIdx || len(c.pendingSoft) != n {
		c.pendingSoftSet = false
		return nil
	}
	c.pendingSoftSet = false
	return c.pendingSoft
}

// SetStrictValidation toggles the strict frame-validity filter on the
// Ingest path. When enabled, PDUs whose (Discriminator, Type) pair is
// not in the documented ETSI EN 300 392-2 set are silently dropped at
// Ingest time. The Process adapter already filters at the framing
// layer; strict-mode tightens it further so PDUs from a
// misaligned-but-passing window still drop out.
func (c *ControlChannel) SetStrictValidation(strict bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.strictValidation = strict
}

// ChannelCodingMode selects how the Process adapter interprets the
// incoming dibit stream:
//
//   - ChannelCodingOff (default): the adapter slices a fixed 48-
//     dibit window after each normal-training-sequence sync and
//     parses the bits straight as a PDU. Works on synthesized
//     test fixtures where the type-2 / type-5 layers aren't
//     present; matches the legacy adapter behaviour.
//
//   - ChannelCodingOn: the adapter slices the channel-appropriate
//     number of dibits per the configured ChannelType, runs the
//     full type-5 → type-1 decode chain (descramble + deinterleave
//
//   - depuncture + Viterbi + CRC-16 verify + tail strip) per
//     ETSI EN 300 392-2 §8.3.1 using the per-channel helpers in
//     channel_coding.go, then parses the recovered info bits as a
//     PDU. Frames whose CRC fails are silently dropped.
//
// Use SetColourCode to seed the scrambler and SetExpectedChannel
// to tell the adapter which logical channel lives in each burst.
type ChannelCodingMode uint8

const (
	ChannelCodingOff ChannelCodingMode = iota
	ChannelCodingOn
)

// ChannelType identifies which TETRA logical channel the Process
// adapter is currently decoding under ChannelCodingOn. The
// connector (or higher-layer caller) sets this per burst /
// per-slot via SetExpectedChannel.
type ChannelType uint8

const (
	// ChannelSCHHD covers SCH/HD, BNCH and STCH — they share the
	// same coding chain per §8.3.1.4.1. 216 type-5 bits / 108
	// dibits per burst, recovering 124 info bits.
	ChannelSCHHD ChannelType = iota
	// ChannelSCHF — full-slot signaling channel. 432 type-5 bits
	// / 216 dibits, recovering 268 info bits.
	ChannelSCHF
	// ChannelSCHHU — half-slot signaling on the uplink. 168
	// type-5 bits / 84 dibits, recovering 92 info bits.
	ChannelSCHHU
	// ChannelBSCH — broadcast synchronisation channel. 120 type-5
	// bits / 60 dibits, recovering 60 info bits. Colour code
	// is implicitly 0 for BSCH regardless of SetColourCode.
	ChannelBSCH
	// ChannelAACH — access-assignment channel (slot header).
	// 30 type-5 bits / 15 dibits, recovering 14 info bits.
	// AACH skips RCPC + interleaving, just RM + scramble.
	ChannelAACH
)

// ParseChannelType maps a config / user-facing string into a
// ChannelType. Recognised values (case-insensitive, "/" optional):
// "sch/hd" | "schhd" | "sch_hd", "sch/f" | "schf", "sch/hu" |
// "schhu", "bsch", "aach". An empty string returns ChannelSCHHD —
// the default ChannelCodingOn channel — and `ok = true` so config
// callers can leave the field blank. Unknown strings return
// ChannelSCHHD with `ok = false` so callers can surface the
// misconfiguration.
func ParseChannelType(s string) (ChannelType, bool) {
	switch strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(s, "/", ""), "_", "")) {
	case "":
		return ChannelSCHHD, true
	case "schhd", "bnch", "stch":
		return ChannelSCHHD, true
	case "schf":
		return ChannelSCHF, true
	case "schhu":
		return ChannelSCHHU, true
	case "bsch":
		return ChannelBSCH, true
	case "aach":
		return ChannelAACH, true
	default:
		return ChannelSCHHD, false
	}
}

// ParseChannelCoding maps a config / user-facing string into a
// ChannelCodingMode. Recognised values (case-insensitive): "" →
// ChannelCodingOn (the new default — full §8.3.1 channel-coding
// chain); "off" / "false" / "0" → ChannelCodingOff (legacy 48-dibit
// raw-PDU path, explicit opt-out for pre-stripped fixtures);
// "on" / "true" / "1" → ChannelCodingOn. Unknown strings return
// ChannelCodingOn with `ok = false` so callers can surface the
// misconfiguration.
func ParseChannelCoding(s string) (ChannelCodingMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return ChannelCodingOn, true
	case "off", "false", "0":
		return ChannelCodingOff, true
	case "on", "true", "1":
		return ChannelCodingOn, true
	default:
		return ChannelCodingOn, false
	}
}

// SetChannelCoding toggles the full EN 300 392-2 §8.3.1 channel
// coding chain on the Process adapter. See ChannelCodingMode for
// the trade-offs.
func (c *ControlChannel) SetChannelCoding(mode ChannelCodingMode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channelCoding = mode
}

// SetExpectedChannel tells the Process adapter which TETRA logical
// channel lives in each burst window. Only consulted when
// ChannelCodingMode is ChannelCodingOn; ignored otherwise. The
// default channel under ChannelCodingOn is ChannelSCHHD (the most
// common signaling carrier for cc.locked / Grant events).
func (c *ControlChannel) SetExpectedChannel(ch ChannelType) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channelType = ch
}

// SetColourCode sets the 30-bit extended colour code the scrambler
// uses under ChannelCodingOn (low 30 bits of colourCode hold
// e(1)..e(30)). BSCH ignores this and uses 0 per §8.2.5.2.
func (c *ControlChannel) SetColourCode(colourCode uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.colourCode = colourCode & 0x3FFFFFFF
	c.colourLearned = true
	// A non-zero operator-configured colour is authoritative: no learned BSCH may
	// override it. A zero value means "unset — auto-learn from the BSCH", so it
	// must NOT confirm/lock (the pipeline seeds SetColourCode(0) when no colour is
	// configured).
	c.colourConfirmed = c.colourCode != 0
}

// colourConfirmThreshold is how many BSCHs must agree on a colour code before
// it is locked in as authoritative. The colour code is constant for a cell and
// the BSCH repeats about once per second, so requiring two agreeing decodes
// filters out a single sync burst whose bit errors slipped through the BSCH FEC
// (a mis-correction to a valid-but-wrong codeword) — which would otherwise lock
// a wrong scrambler for the whole session and silently break every BNCH/SCH/TCH
// descramble (empty or truncated recordings until restart).
const colourConfirmThreshold = 2

// LearnColourCode records a colour code recovered at runtime from a decoded BSCH
// SYNC PDU (see process.go). The first BSCH sets the colour PROVISIONALLY so a
// cold receiver can start descrambling immediately, but the value is not locked
// until a second BSCH corroborates it (colourConfirmThreshold): if the first
// burst mis-decoded the colour, the true colour — which every later BSCH carries
// — overtakes it and corrects the scrambler instead of being ignored. An
// operator-configured colour is authoritative and never overridden. Returns true
// if the stored value changed.
func (c *ControlChannel) LearnColourCode(ext uint32) bool {
	ext &= 0x3FFFFFFF
	c.mu.Lock()
	defer c.mu.Unlock()
	// A confirmed (operator-set, or already-corroborated) colour is authoritative.
	if c.colourConfirmed {
		return false
	}
	if c.colourTally == nil {
		c.colourTally = make(map[uint32]int)
	}
	c.colourTally[ext]++
	changed := false
	// Provisionally adopt the first colour we see so decode can begin at once —
	// but a later, corroborated value may still correct it below. The gate is
	// "no usable colour yet" (colourCode == 0), which also covers the pipeline's
	// SetColourCode(0) "unset" seed; a real cell's colour is never 0 (its MCC/MNC
	// are non-zero).
	if c.colourCode == 0 {
		c.colourCode = ext
		c.colourLearned = true
		changed = true
		c.log.Info("tetra cc provisional colour code from BSCH (unconfirmed)",
			"colour_code", ext&0x3F, "colour_ext", ext, "system", c.systemName)
	}
	if c.colourTally[ext] >= colourConfirmThreshold {
		if c.colourCode != ext {
			c.colourCode = ext
			changed = true
		}
		c.colourConfirmed = true
		// colour_code is the ETSI 6-bit colour code (low 6 bits); colour_ext is
		// the full 30-bit extended value that also seeds the scrambler.
		c.log.Info("tetra cc colour code confirmed from BSCH",
			"colour_code", ext&0x3F, "colour_ext", ext, "system", c.systemName)
	}
	return changed
}

// ChannelCoding returns the current ChannelCodingMode. Mirrors the
// Set* family so callers (and tests) can introspect the configured
// mode without poking at unexported state.
func (c *ControlChannel) ChannelCoding() ChannelCodingMode {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.channelCoding
}

// ExpectedChannel returns the ChannelType the Process adapter
// currently expects under ChannelCodingOn.
func (c *ControlChannel) ExpectedChannel() ChannelType {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.channelType
}

// ColourCode returns the configured 30-bit extended colour code.
func (c *ControlChannel) ColourCode() uint32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.colourCode
}

// TopologyConfig is a snapshot of the TETRA single-cell identity for the hunt
// layer: the MCC/MNC/Location Area learned from MLE-SYSINFO plus the colour
// code. Neighbor-cell broadcasts are not accumulated, so identity only.
type TopologyConfig struct {
	MCC          uint16
	MNC          uint16
	LocationArea uint16
	ColourCode   uint32
	// MainCarrier and the absolute downlink/uplink carrier frequencies derived
	// from SYSINFO (§21.4.4.1) — the offset-corrected true carrier, not the tuned
	// frequency. DownlinkHz/UplinkHz are 0 when not yet known (UplinkHz also 0
	// when the band's duplex spacing is not mapped; see duplexSpacingHz).
	MainCarrier uint16
	DownlinkHz  uint32
	UplinkHz    uint32
}

// Topology returns the accumulated single-cell identity. Safe for concurrent
// use (reads the lock state + colour code under the control-channel mutex).
func (c *ControlChannel) Topology() TopologyConfig {
	dl, ul, _, _ := c.cellFrequencies()
	c.mu.Lock()
	defer c.mu.Unlock()
	return TopologyConfig{
		MCC:          c.last.MCC,
		MNC:          c.last.MNC,
		LocationArea: c.last.LocationArea,
		ColourCode:   c.colourCode,
		MainCarrier:  c.mainCarrier,
		DownlinkHz:   dl,
		UplinkHz:     ul,
	}
}

// Stats is a snapshot of TETRA decode-health counters accumulated since the
// last DrainStats call. It is debug-only telemetry: the counters only move
// when the control channel's logger is at debug level (see New), and feed the
// throttled "tetra: decode status" console line the connector emits. All
// fields are cumulative counts over the drain window except Dibits, which is
// the symbol count used to estimate the effective baud.
type Stats struct {
	Dibits   int64 // symbols (dibits) processed — effective-baud numerator
	SBBursts int64 // synchronisation-burst candidates entering the SB decoder
	BSCHOK   int64 // BSCH blocks recovered CRC-clean
	BSCHFail int64 // SB candidates whose BSCH did not decode under any rotation
	SysInfo  int64 // BNCH SYSINFO PDUs decoded off the synchronisation burst
	SCHPDUs  int64 // signalling PDUs parsed off the normal-sync path
	Grants   int64 // voice grants published
}

// addStat adds n to a decode-health counter when debug telemetry is on. A
// no-op (no lock, no write) otherwise, so production decode is unaffected.
// field must point into c.stats.
func (c *ControlChannel) addStat(field *int64, n int64) {
	if !c.debug {
		return
	}
	c.statsMu.Lock()
	*field += n
	c.statsMu.Unlock()
}

// DrainStats returns the decode-health counters accumulated since the last
// call and resets them to zero. Stays all-zero unless the logger is at debug
// level (see New / addStat). Safe for concurrent use.
func (c *ControlChannel) DrainStats() Stats {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	s := c.stats
	c.stats = Stats{}
	return s
}

// Locked reports whether the control channel has declared lock. Read-only;
// mirrors the Topology accessor so the connector can surface lock state in the
// periodic decode-status line without poking unexported fields.
func (c *ControlChannel) Locked() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.locked
}

// Options configure a ControlChannel.
type Options struct {
	Bus         *events.Bus
	Log         *slog.Logger
	SystemName  string
	FrequencyHz uint32
	Resolver    Resolver
	Now         func() time.Time

	// FECObserver, when non-nil, is invoked with the per-burst FEC
	// correction depth (channel bits the §8.3.1 chain corrected) every
	// time a BSCH or BNCH/SCH-HD burst recovers CRC-clean. Wired by the
	// connector to the metrics layer only when metrics.detailed_fec is
	// enabled. nil ⇒ zero overhead.
	FECObserver func(channel string, corrections int)
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
		bus:         opts.Bus,
		log:         log,
		systemName:  opts.SystemName,
		freqHz:      opts.FrequencyHz,
		resolver:    opts.Resolver,
		now:         now,
		fecObserver: opts.FECObserver,
		debug:       log.Enabled(context.Background(), slog.LevelDebug),
	}
}

// Ingest hands a single decoded PDU to the state machine. Real
// captures arrive via an upstream π/4-DQPSK demod + RCPC/RM FEC;
// tests publish PDUs directly.
func (c *ControlChannel) Ingest(p PDU) {
	c.mu.Lock()
	strict := c.strictValidation
	c.mu.Unlock()
	if strict && !p.IsKnown() {
		return
	}
	if p.IsIdle() {
		return
	}
	if sb, ok := p.AsSystemBroadcast(); ok {
		c.maybeLock(LockState{
			FrequencyHz:  c.freqHz,
			MCC:          sb.MCC,
			MNC:          sb.MNC,
			LocationArea: sb.LocationArea,
		})
	}
	// Voice grants and call teardown are NOT parsed here: on real air they ride
	// in a bit-packed CMCE TM-SDU that the byte-aligned ParsePDU cannot frame
	// (a 3-bit MLE discriminator + 5-bit type, not 4+4). They are decoded
	// bit-accurately from the MAC layer instead — see downlink.go handleCMCE,
	// where the MAC-RESOURCE address already gives the reliable GSSI.
}

// tetraChannelSpacingHz is the TETRA carrier spacing (25 kHz), used to derive a
// grant carrier's frequency relative to the cell's own carrier.
const tetraChannelSpacingHz = 25_000

// maxGrantCarrierOffsetHz bounds how far a grant's resolved traffic frequency may
// sit from the control channel before it is treated as a bit-alignment / false-CRC
// parsing artefact rather than a real allocation. Every traffic carrier a control
// channel grants lives in the same operator allocation as the CC itself — TETRA
// site downlink carriers span at most a few MHz — so a resolved frequency tens of
// MHz away (e.g. carrier 6 → 400 MHz on a 467 MHz cell) can only be a corrupt
// block whose 16-bit SCH CRC happened to pass, or a slipped bit cursor. 10 MHz is
// comfortably wider than any real single-site carrier spread yet rejects such
// foreign-band garbage outright.
const maxGrantCarrierOffsetHz = 10_000_000

// carrierFrequency derives the Hz of a TETRA carrier number for this cell.
//
// When the full SYSINFO frequency parameters are known (frequency band +
// offset), it returns the *absolute* downlink carrier — base(band) +
// carrier*25 kHz + offset — so a grant's frequency lands on the real carrier,
// including the ±6.25/12.5 kHz offset the cell broadcasts. Falling back (only
// the main carrier number learned, e.g. an operator-seeded band plan), it
// derives the carrier at 25 kHz spacing relative to the tuned control frequency.
// Returns false until enough is known to resolve a frequency.
func (c *ControlChannel) carrierFrequency(carrier uint16) (uint32, bool) {
	c.mu.Lock()
	si, siSet := c.sysInfo, c.sysInfoSet
	mc, set := c.mainCarrier, c.mainCarrierSet
	c.mu.Unlock()
	var hz int64
	switch {
	case siSet:
		hz = tetraDLCarrierHz(si.FreqBand, carrier, si.Offset)
	case set && c.freqHz != 0:
		hz = int64(c.freqHz) + (int64(carrier)-int64(mc))*tetraChannelSpacingHz
	default:
		return 0, false
	}
	if hz <= 0 {
		return 0, false
	}
	return uint32(hz), true
}

// learnMainCarrier records the cell's own carrier number, so grant carrier
// numbers can resolve to Hz relative to it at 25 kHz spacing. Used by the
// relative fallback path; learnSysInfo supersedes it with absolute frequencies.
func (c *ControlChannel) learnMainCarrier(carrier uint16) {
	c.mu.Lock()
	c.mainCarrier = carrier
	c.mainCarrierSet = true
	c.mu.Unlock()
}

// learnSysInfo records the cell's full SYSINFO frequency parameters (band,
// offset, duplex spacing, reverse operation) so carrier numbers resolve to their
// absolute downlink frequency including the offset field, and the true downlink
// and uplink carrier frequencies can be reported. The first time it resolves a
// downlink that differs from the tuned frequency it logs the offset-corrected
// carrier — visible in debug.log, where "broadcast 469.875 but real 469.88125"
// finally shows up.
func (c *ControlChannel) learnSysInfo(si SysInfo) {
	c.mu.Lock()
	first := !c.sysInfoSet || c.sysInfo != si
	c.sysInfo = si
	c.sysInfoSet = true
	c.mainCarrier = si.MainCarrier
	c.mainCarrierSet = true
	c.mu.Unlock()
	if first {
		dl := tetraDLCarrierHz(si.FreqBand, si.MainCarrier, si.Offset)
		ul, ulOK := tetraULCarrierHz(dl, si.FreqBand, si.DuplexSpacing, si.ReverseOper)
		c.log.Info("tetra: SYSINFO carrier",
			"system", c.systemName,
			"main_carrier", si.MainCarrier,
			"band", si.FreqBand,
			"offset_field", si.Offset,
			"downlink_hz", dl,
			"uplink_hz", ul,
			"uplink_known", ulOK,
			"tuned_hz", c.freqHz)
	}
}

// cellFrequencies returns the cell's absolute downlink and uplink carrier
// frequencies (Hz) derived from the learned SYSINFO. downlinkOK/uplinkOK report
// whether each is known (uplink needs a mapped duplex spacing for the band).
func (c *ControlChannel) cellFrequencies() (downlinkHz, uplinkHz uint32, downlinkOK, uplinkOK bool) {
	c.mu.Lock()
	si, set := c.sysInfo, c.sysInfoSet
	c.mu.Unlock()
	if !set {
		return 0, 0, false, false
	}
	dl := tetraDLCarrierHz(si.FreqBand, si.MainCarrier, si.Offset)
	if dl <= 0 {
		return 0, 0, false, false
	}
	if ul, ok := tetraULCarrierHz(dl, si.FreqBand, si.DuplexSpacing, si.ReverseOper); ok && ul > 0 {
		return uint32(dl), uint32(ul), true, true
	}
	return uint32(dl), 0, true, false
}

func (c *ControlChannel) publishGrant(g VoiceGrant) {
	if c.bus == nil {
		return
	}
	c.mu.Lock()
	colourExt := c.colourCode
	c.mu.Unlock()
	freq := uint32(0)
	if c.resolver != nil {
		if hz, err := c.resolver.Frequency(g.CarrierNumber); err == nil {
			freq = hz
		} else {
			c.log.Debug("tetra: band-plan resolution failed",
				"carrier", g.CarrierNumber, "err", err)
		}
	} else if hz, ok := c.carrierFrequency(g.CarrierNumber); ok {
		// No configured band plan: derive the grant frequency relative to this
		// cell's own carrier (learned from SYSINFO) at 25 kHz TETRA spacing.
		// Exact for a same-carrier SCBS (carrier == mainCarrier ⇒ the CC freq).
		freq = hz
	}
	// Structural sanity guard against a bit-alignment slip or a false-positive SCH
	// CRC: a grant whose carrier resolves to a band foreign to this cell cannot be
	// real (see maxGrantCarrierOffsetHz). Drop it rather than log a ghost call and
	// spawn a phantom recording on a frequency the site does not operate. Only
	// enforced once both the CC frequency and a resolved grant frequency are known;
	// an offline replay with neither (freq == 0) simply skips the guard.
	if freq != 0 && c.freqHz != 0 {
		off := int64(freq) - int64(c.freqHz)
		if off < 0 {
			off = -off
		}
		if off > maxGrantCarrierOffsetHz {
			c.log.Warn("tetra: dropping grant — carrier resolves to a foreign band (bit-alignment/CRC artefact)",
				"system", c.systemName,
				"src", g.SourceSSI, "dst", g.DestSSI,
				"carrier", g.CarrierNumber, "resolved_hz", freq,
				"cc_hz", c.freqHz, "offset_hz", off)
			return
		}
	}
	c.bus.Publish(events.Event{
		Kind: events.KindGrant,
		Payload: trunking.Grant{
			System:      c.systemName,
			Protocol:    "tetra",
			GroupID:     g.DestSSI,
			SourceID:    g.SourceSSI,
			FrequencyHz: freq,
			ChannelNum:  g.CarrierNumber,
			// TETRA carries a 0-based (0..3) timeslot on the grant; the
			// trunking engine's Timeslot is 1-based with 0 reserved for
			// "not applicable", so map 0..3 → 1..4. The voice tap uses it
			// to pick the granted slot out of the 4-slot TDMA frame.
			Timeslot:         g.Timeslot + 1,
			Encrypted:        g.Encrypted,
			Emergency:        g.Emergency,
			Individual:       g.Individual,
			TETRAColourExt:   colourExt,
			TETRAUsageMarker: g.UsageMarker,
			At:               c.now(),
		},
	})
	c.addStat(&c.stats.Grants, 1)
	c.log.Debug("tetra: grant",
		"system", c.systemName,
		"src", g.SourceSSI, "dst", g.DestSSI,
		"carrier", g.CarrierNumber, "slot", g.Timeslot, "freq_hz", freq,
		"individual", g.Individual, "enc", g.Encrypted, "emer", g.Emergency)
}

// rememberCall records the call-identifier → group-SSI mapping learned from a
// D-SETUP/D-CONNECT, so a later D-RELEASE/D-TX-GRANTED (which carries only the
// call identifier) can be resolved back to the talkgroup. No-op for a zero
// group.
func (c *ControlChannel) rememberCall(callID uint16, gssi uint32) {
	if gssi == 0 {
		return
	}
	c.mu.Lock()
	if c.callGroups == nil {
		c.callGroups = make(map[uint16]uint32)
	}
	c.callGroups[callID] = gssi
	c.mu.Unlock()
}

// resolveGroup maps a CMCE call identifier to its group SSI, preferring the
// learned D-SETUP/D-CONNECT mapping and falling back to the MAC-RESOURCE address
// SSI the PDU was addressed under (which for a group call is the GSSI). Returns
// 0 when neither is known.
func (c *ControlChannel) resolveGroup(callID uint16, addrSSI uint32) uint32 {
	c.mu.Lock()
	gssi := c.callGroups[callID]
	c.mu.Unlock()
	if gssi != 0 {
		return gssi
	}
	return addrSSI
}

// forgetCall drops a call-identifier mapping once its call has been released.
func (c *ControlChannel) forgetCall(callID uint16) {
	c.mu.Lock()
	delete(c.callGroups, callID)
	c.mu.Unlock()
}

// individualsCap bounds the learned-individuals set so a long-running site cannot
// grow it without limit. A cell's active subscriber set is far smaller; once full
// the set stops learning new individuals (existing ones still classify).
const individualsCap = 8192

// noteIndividual records an SSI seen acting as a CMCE calling/transmitting party
// — always an individual radio. No-op for 0 or once the set is at capacity.
func (c *ControlChannel) noteIndividual(ssi uint32) {
	if ssi == 0 {
		return
	}
	c.mu.Lock()
	if c.individuals == nil {
		c.individuals = make(map[uint32]struct{})
	}
	if _, ok := c.individuals[ssi]; !ok && len(c.individuals) < individualsCap {
		c.individuals[ssi] = struct{}{}
	}
	c.mu.Unlock()
}

// isIndividual reports whether an SSI has been seen acting as a party (and is
// therefore an individual radio rather than a talkgroup GSSI).
func (c *ControlChannel) isIndividual(ssi uint32) bool {
	c.mu.Lock()
	_, ok := c.individuals[ssi]
	c.mu.Unlock()
	return ok
}

// fragMaxBits bounds the reassembly buffer (one-bit-per-byte). A control-channel
// TM-SDU is well under this; the cap only stops a runaway MAC-FRAG stream that
// never delivers a MAC-END from growing the buffer without limit.
const fragMaxBits = 2048

// stashFragment records a start-fragment MAC-RESOURCE's partial TM-SDU and its
// resource context so a following MAC-END can reassemble the complete L3 PDU. A
// new start fragment replaces any incomplete one (single TM-SDU in flight).
func (c *ControlChannel) stashFragment(m MACResource, partial []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fragResource = m
	c.fragTMSDU = append(c.fragTMSDU[:0], partial...)
	c.fragActive = true
}

// appendFragment appends a MAC-FRAG continuation to the in-progress TM-SDU. A
// no-op when no start fragment is in progress; a stream that would exceed
// fragMaxBits is dropped rather than accumulated.
func (c *ControlChannel) appendFragment(payload []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.fragActive {
		return
	}
	if len(c.fragTMSDU)+len(payload) > fragMaxBits {
		c.fragActive = false
		c.fragTMSDU = c.fragTMSDU[:0]
		return
	}
	c.fragTMSDU = append(c.fragTMSDU, payload...)
}

// takeFragment appends a MAC-END payload, returns the reassembled TM-SDU (a
// fresh slice) plus the start fragment's MAC-RESOURCE context, and clears the
// buffer. ok is false for a stray MAC-END with no start fragment in progress or
// an over-length reassembly.
func (c *ControlChannel) takeFragment(payload []byte) (MACResource, []byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer func() {
		c.fragActive = false
		c.fragTMSDU = c.fragTMSDU[:0]
		c.fragResource = MACResource{}
	}()
	if !c.fragActive || len(c.fragTMSDU)+len(payload) > fragMaxBits {
		return MACResource{}, nil, false
	}
	sdu := make([]byte, 0, len(c.fragTMSDU)+len(payload))
	sdu = append(sdu, c.fragTMSDU...)
	sdu = append(sdu, payload...)
	return c.fragResource, sdu, true
}

// publishRelease emits a KindCallRelease so the engine ends the active call for
// this talkgroup at once, rather than waiting out the voice hangtime/no-voice
// timers. No-op without a bus or a resolvable group.
func (c *ControlChannel) publishRelease(gssi uint32, cause uint8) {
	if c.bus == nil || gssi == 0 {
		return
	}
	c.bus.Publish(events.Event{
		Kind: events.KindCallRelease,
		Payload: trunking.CallRelease{
			System:  c.systemName,
			GroupID: gssi,
			Reason:  trunking.EndReasonReleased,
			At:      c.now(),
		},
	})
	c.log.Debug("tetra: call release",
		"system", c.systemName, "group", gssi, "cause", cause)
}

// publishTalker emits a KindCallTalker so the engine backfills the current
// transmitting party's SSI onto the active call for this talkgroup.
func (c *ControlChannel) publishTalker(gssi, src uint32) {
	if c.bus == nil || gssi == 0 || src == 0 {
		return
	}
	c.bus.Publish(events.Event{
		Kind: events.KindCallTalker,
		Payload: trunking.CallTalker{
			System:   c.systemName,
			GroupID:  gssi,
			SourceID: src,
			At:       c.now(),
		},
	})
	c.log.Debug("tetra: call talker",
		"system", c.systemName, "group", gssi, "src", src)
}

// noteActivity stamps the control-channel heartbeat with the current time.
// Called from the decode path on every successful CC recovery (sync burst,
// SYSINFO, grant) so CheckStale can tell a live carrier from a silent one.
// Lock-free (atomic) so it adds no contention to the hot path.
func (c *ControlChannel) noteActivity() {
	c.lastActivityNano.Store(c.now().UnixNano())
}

// CheckStale declares the control channel lost (publishes cc.lost via MarkLost)
// when it is locked but has decoded nothing for longer than timeout. This is the
// watchdog the cchunt supervisor needs to leave StateLocked and re-hunt when the
// carrier genuinely goes silent — TETRA's maybeLock is edge-triggered and never
// re-affirms a steady lock, so without this a dead carrier would never surface a
// cc.lost. A no-op until the first decode (lastActivity==0), while unlocked, or
// while activity is fresh, so a healthy carrier (sync burst every ~1 s) never
// trips it.
func (c *ControlChannel) CheckStale(now time.Time, timeout time.Duration) {
	last := c.lastActivityNano.Load()
	if last == 0 {
		return
	}
	if now.Sub(time.Unix(0, last)) <= timeout {
		return
	}
	c.MarkLost()
}

// NeedsResync reports whether the control-channel heartbeat is older than
// timeout, i.e. no sync burst / SYSINFO / grant has decoded for that long. It is
// the trigger the pipeline uses to force a fast DSP re-acquire (reset the symbol
// timing to centre) after a noise burst wanders the loops off-lock — the
// steady-state Gardner loop at the production gain re-converges only very slowly,
// so from-centre reacquisition (~one AFC block) is far faster than waiting it
// out.
//
// Like CheckStale it is a lock-free atomic read of the heartbeat and is a no-op
// before the first decode (lastActivity==0). Deliberately independent of the
// locked flag: MarkLost never clears the heartbeat, so a stale locked OR
// already-lost channel both keep asking for a reacquire until the carrier
// returns — bounding the recovery tail the cchunt same-frequency guard would
// otherwise leave running for tens of seconds.
func (c *ControlChannel) NeedsResync(now time.Time, timeout time.Duration) bool {
	last := c.lastActivityNano.Load()
	if last == 0 {
		return false
	}
	return now.Sub(time.Unix(0, last)) > timeout
}

func (c *ControlChannel) maybeLock(s LockState) {
	c.noteActivity()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.locked && c.last == s {
		return
	}
	// Preserve previously-learned MCC/MNC/LA if the new state has none.
	if c.locked && s.MCC == 0 && c.last.MCC != 0 {
		s.MCC = c.last.MCC
		s.MNC = c.last.MNC
		s.LocationArea = c.last.LocationArea
		if c.last == s {
			return
		}
	}
	c.locked = true
	c.last = s
	c.bus.Publish(events.Event{Kind: events.KindCCLocked, Payload: s})
	c.log.Info("tetra cc locked",
		"freq", s.FrequencyHz, "mcc", s.MCC, "mnc", s.MNC,
		"la", s.LocationArea, "system", c.systemName)
}

// MarkLost publishes cc.lost and resets the locked flag. CheckStale calls this
// when a locked control channel has gone silent past the staleness timeout, so
// the cchunt supervisor leaves StateLocked and re-hunts. Idempotent while
// already unlocked.
func (c *ControlChannel) MarkLost() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.locked {
		return
	}
	c.locked = false
	c.bus.Publish(events.Event{Kind: events.KindCCLost, Payload: c.last})
}

// ResyncReset drops the dibit-domain synchronisation state so the next Process
// call rebuilds it from a fresh baseline. It is called immediately after the
// receiver's Reset (which restarts its dibit index at 0) so the two stay in
// step: processState indexes its rolling SB buffer by ABSOLUTE dibit index
// (bufBase, pendingSTS), so a receiver whose index jumped back to 0 while proc
// held a large bufBase would drive every future SB look-back into the
// "already trimmed" branch and never decode another sync burst. Nil-ing proc
// forces a clean lazy rebuild with the new baseIdx.
//
// Everything that makes re-lock fast is PRESERVED: the learned/confirmed colour
// code, the lock flag and cell identity, the channel configuration, the activity
// heartbeat, and the decode-health stats. Only the throwaway dibit-sync scratch
// (proc) and the soft-differential stash are cleared.
//
// Precondition: called on the same goroutine as Process (from
// tetraPipeline.Process, after rx.Process returns), so the unlocked proc read in
// Process never races this write; the mutex here guards proc and the pendingSoft
// stash against the other lock-holding accessors.
func (c *ControlChannel) ResyncReset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.proc = nil
	c.pendingSoft = c.pendingSoft[:0]
	c.pendingSoftBase = 0
	c.pendingSoftSet = false
	// A stream re-sync abandons any half-reassembled TM-SDU: its continuation
	// belonged to the pre-resync bit index and must not splice onto a fragment
	// decoded after the baseline jump.
	c.fragActive = false
	c.fragTMSDU = c.fragTMSDU[:0]
	c.fragResource = MACResource{}
}
