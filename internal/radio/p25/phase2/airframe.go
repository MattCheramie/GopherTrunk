package phase2

import (
	"math/bits"

	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// P25 Phase 2 TDMA outbound air format (TIA-102.BBAC), as decoded on live
// air by OP25 (gr-op25_repeater/lib: p25p2_tdma.cc, p25p2_sync.cc,
// p25p2_isch.cc, p25p2_duid.cc, p25p2_vf.cc, apps/tdma/lfsr.py) and SDRTrunk
// (module/decode/p25/phase2: SuperFrameFragment, ISCHDecoder, DataUnitID,
// Voice4Timeslot, ScramblingSequence). Every constant below is one the two
// references agree on; the tests pin them with literal values from both.
//
// This replaces, for the voice path, the working model in superframe.go /
// isch.go / voice.go (a sync word in sub-frame 0 only, a Golay(24,12) ISCH
// carrying a SlotType, voice frames contiguous after it, no scrambling),
// which no reference shares and which never matched real air: the #915
// ground-truth capture failed that ISCH Golay on 46–62 % of sub-frames.
//
// The real outbound channel is a stream of 30 ms timeslots, 12 per 360 ms
// superframe. Each timeslot is one 180-dibit packet:
//
//	dibits   0..19   ISCH (40 bits): an S-ISCH (the sync word
//	                 0x575D57F7FF) before timeslots 2,3, 6,7, 10,11, an
//	                 I-ISCH (a (40,9) codeword naming the timeslot)
//	                 before 0,1, 4,5, 8,9
//	dibits  20..179  the 320-bit burst body, its 8-bit DUID spread over
//	                 body dibits 0, 37, 122, 159
//
// A voice burst (DUID VOICE-4 / VOICE-2) carries four (two) 72-bit AMBE+2
// frames at body dibits 1, 38, 86, 123 (only the first two in VOICE-2),
// XOR-scrambled with the 4320-bit PN44 superframe sequence; timeslot t's
// body uses sequence bits [20+360t, 340+360t).
//
// The two TDMA logical channels interleave by timeslot, EXCEPT that the
// last pair is swapped: timeslots 0,2,4,6,8,11 are channel 0 and
// 1,3,5,7,9,10 channel 1 (OP25 which_slot[], SDRTrunk's final-fragment
// C/D swap). A grant's Timeslot (1-based, from the channel number's
// n mod 2) names that logical channel.
const (
	// AirISCHDibits is the ISCH that opens every timeslot packet.
	AirISCHDibits = 20
	// AirBodyDibits is the burst body after it.
	AirBodyDibits = 160
	// AirPacketDibits is one 30 ms timeslot: ISCH + body.
	AirPacketDibits = AirISCHDibits + AirBodyDibits
	// AirSlotsPerSuperframe is the 12 timeslots of a 360 ms superframe.
	AirSlotsPerSuperframe = 12

	// SISCHSync is the S-ISCH: the 40-bit outbound sync word.
	SISCHSync uint64 = OutboundSyncHex

	// airVoiceFrameDibits is one 72-bit AMBE+2 voice codeword.
	airVoiceFrameDibits = 36
)

// airVoiceFrameBody holds the body-dibit offsets of the four voice frames
// (VOICE-2 uses the first two). OP25 burstp[11/48/96/133] less its
// 10-dibit lead; SDRTrunk Voice4Timeslot bits 2/76/172/246 halved.
var airVoiceFrameBody = [4]int{1, 38, 86, 123}

// airDUIDBody holds the body-dibit offsets of the DUID's four dibits, most
// significant first (OP25 extract_duid; SDRTrunk DATA_UNIT_ID bits
// 0,1,74,75,244,245,318,319).
var airDUIDBody = [4]int{0, 37, 122, 159}

// airSlotLCH maps a timeslot 0..11 to its logical channel (OP25
// which_slot[]).
var airSlotLCH = [AirSlotsPerSuperframe]int{0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 0}

// AirSlotLogicalChannel returns the TDMA logical channel (0 or 1) that
// timeslot 0..11 carries, or -1 for an index outside the superframe.
func AirSlotLogicalChannel(slot int) int {
	if slot < 0 || slot >= AirSlotsPerSuperframe {
		return -1
	}
	return airSlotLCH[slot]
}

// DUID is a Phase 2 burst's Data Unit ID (SDRTrunk DataUnitID).
type DUID int8

const (
	DUIDUnknown        DUID = -1
	DUIDVoice4         DUID = 0
	DUIDSACCHScrambled DUID = 3
	DUIDVoice2         DUID = 6
	DUIDFACCHScrambled DUID = 9
	DUIDDATCHScrambled DUID = 10
	DUIDSACCHClear     DUID = 12
	DUIDLCCHClear      DUID = 13
	DUIDFACCHClear     DUID = 15
)

// IsVoice reports whether the burst carries AMBE+2 voice frames.
func (d DUID) IsVoice() bool { return d == DUIDVoice4 || d == DUIDVoice2 }

// VoiceFrames is how many voice frames the burst carries (4, 2 or 0).
func (d DUID) VoiceFrames() int {
	switch d {
	case DUIDVoice4:
		return 4
	case DUIDVoice2:
		return 2
	}
	return 0
}

func (d DUID) String() string {
	switch d {
	case DUIDVoice4:
		return "voice4"
	case DUIDVoice2:
		return "voice2"
	case DUIDSACCHScrambled:
		return "sacch-s"
	case DUIDFACCHScrambled:
		return "facch-s"
	case DUIDDATCHScrambled:
		return "datch-s"
	case DUIDSACCHClear:
		return "sacch-u"
	case DUIDLCCHClear:
		return "lcch-u"
	case DUIDFACCHClear:
		return "facch-u"
	case DUIDUnknown:
		return "unknown"
	}
	return "reserved"
}

// duidCodewords is the 8-bit codeword of each 4-bit DUID value (SDRTrunk
// DataUnitID; the exact entries of OP25's _duid_lookup).
var duidCodewords = [16]uint8{
	0x00, 0x17, 0x2E, 0x39, 0x4B, 0x5C, 0x65, 0x72,
	0x8D, 0x9A, 0xA3, 0xB4, 0xC6, 0xD1, 0xE8, 0xFF,
}

// DecodeDUID returns the DUID nearest the 8-bit codeword. The code has
// minimum distance 4, so one bit error is corrected and a codeword at
// distance 2 from two values is unknown.
func DecodeDUID(cw uint8) DUID {
	best, bestD, tie := -1, 9, false
	for v, c := range duidCodewords {
		d := bits.OnesCount8(cw ^ c)
		switch {
		case d < bestD:
			best, bestD, tie = v, d, false
		case d == bestD:
			tie = true
		}
	}
	if tie || bestD > 1 {
		return DUIDUnknown
	}
	return DUID(best)
}

// EncodeDUID returns the 8-bit codeword for a DUID value.
func EncodeDUID(d DUID) uint8 { return duidCodewords[uint8(d)&0xF] }

// IISCH is the informational ISCH before timeslots 0,1, 4,5, 8,9.
type IISCH struct {
	// Channel is the logical channel (0 or 1) of the timeslot it opens.
	Channel uint8
	// Location is the ISCH position 0..2 in the superframe; the timeslot
	// index is Location*4 + Channel.
	Location uint8
	// Bit carries the I-ISCH's single flag bit (OP25 "fr").
	Bit uint8
	// Count is the 2-bit superframe sequence counter.
	Count uint8
}

// Slot is the timeslot index 0..11 the I-ISCH opens.
func (i IISCH) Slot() int { return int(i.Location)*4 + int(i.Channel) }

// iischGenerator and iischOffset build the (40,9) I-ISCH codewords
// (SDRTrunk ISCHDecoder). The 7-bit values 0..127 used here are the
// 9-bit message with its two leading bits zero; their codewords equal
// OP25's 128-entry p25p2_isch table exactly.
var iischGenerator = [9]uint64{
	0x8816CE36D7, 0x201DFD4F64, 0x100F4B1758, 0x0C00DED18E, 0x020807F7FF,
	0x09048D9B72, 0x009DA3A171, 0x0058CBAA4E, 0x00343D8597,
}

const iischOffset uint64 = 0x184229d461

var iischCodewords = func() (t [128]uint64) {
	for v := range t {
		var cw uint64
		for b := 0; b < 9; b++ {
			if v>>(8-b)&1 != 0 {
				cw ^= iischGenerator[b]
			}
		}
		t[v] = cw ^ iischOffset
	}
	return t
}()

// iischMaxErrs is the I-ISCH correction radius: the code is (40,9,16).
const iischMaxErrs = 7

func iischValue(i IISCH) int {
	return int(i.Channel&3)<<5 | int(i.Location&3)<<3 | int(i.Bit&1)<<2 | int(i.Count&3)
}

// EncodeIISCH returns the 40-bit I-ISCH codeword.
func EncodeIISCH(i IISCH) uint64 { return iischCodewords[iischValue(i)] }

// DecodeIISCH decodes a 40-bit ISCH as an I-ISCH, nearest codeword within
// iischMaxErrs bit errors. ok is false for anything farther, or a
// location of 3 (no such ISCH position).
func DecodeIISCH(cw uint64) (i IISCH, errs int, ok bool) {
	best, bestD := -1, iischMaxErrs+1
	for v, c := range iischCodewords {
		if d := bits.OnesCount64(cw ^ c); d < bestD {
			best, bestD = v, d
		}
	}
	if best < 0 {
		return IISCH{}, -1, false
	}
	i = IISCH{
		Channel:  uint8(best >> 5 & 3),
		Location: uint8(best >> 3 & 3),
		Bit:      uint8(best >> 2 & 1),
		Count:    uint8(best & 3),
	}
	if i.Location == 3 {
		return IISCH{}, -1, false
	}
	return i, bestD, true
}

// AirScrambleMask returns timeslot slot's body descrambling mask as
// AirBodyDibits dibits: PN44 superframe bits [20+360*slot, 340+360*slot),
// two per dibit, high bit first. seed is the outbound PN44 seed
// (framing.PN44SeedFromIdentity of WACN, System ID and NAC).
func AirScrambleMask(seed uint64, slot int) []uint8 {
	s := framing.NewPN44Scrambler(seed)
	s.Advance(20 + 360*slot)
	mask := make([]uint8, AirBodyDibits)
	for k := range mask {
		mask[k] = s.Next()<<1 | s.Next()
	}
	return mask
}

// AirMasks caches the 12 per-timeslot masks for one seed.
type AirMasks struct {
	seed  uint64
	masks [AirSlotsPerSuperframe][]uint8
}

// NewAirMasks precomputes every timeslot's mask for seed.
func NewAirMasks(seed uint64) *AirMasks {
	m := &AirMasks{seed: seed}
	for s := range m.masks {
		m.masks[s] = AirScrambleMask(seed, s)
	}
	return m
}

// Seed returns the seed the masks were built from.
func (m *AirMasks) Seed() uint64 { return m.seed }

// Slot returns timeslot slot's mask.
func (m *AirMasks) Slot(slot int) []uint8 { return m.masks[slot] }

// AirBurst is one timeslot packet framed off the air.
type AirBurst struct {
	// StartDibit is the absolute stream index of the packet's first
	// (ISCH) dibit.
	StartDibit int
	// Slot is the timeslot index 0..11.
	Slot int
	// LogicalChannel is the TDMA channel (0 or 1) the timeslot carries.
	LogicalChannel int
	// DUID is the burst's data unit ID.
	DUID DUID
	// Packet is the 180-dibit packet, de-rotated, NOT descrambled.
	Packet []uint8
}

// Body returns the 160-dibit burst body.
func (b AirBurst) Body() []uint8 { return b.Packet[AirISCHDibits:] }

// ExtractAirVoiceFrames descrambles a voice burst with its timeslot's
// mask and FEC-decodes each AMBE+2 frame to a 7-byte vocoder frame (the
// 72-bit codeword layout is DMR's, which OP25's p25p2_vf extract_vcw
// matches bit for bit). totalErrs sums the Golay corrections.
func ExtractAirVoiceFrames(b AirBurst, masks *AirMasks) (frames [][]byte, totalErrs int) {
	n := b.DUID.VoiceFrames()
	if n == 0 || masks == nil || len(b.Packet) != AirPacketDibits {
		return nil, 0
	}
	body := b.Body()
	mask := masks.Slot(b.Slot)
	frames = make([][]byte, n)
	cw := make([]uint8, airVoiceFrameDibits)
	for i := 0; i < n; i++ {
		off := airVoiceFrameBody[i]
		for k := range cw {
			cw[k] = body[off+k] ^ mask[off+k]
		}
		info, errs, err := dmrvoice.DecodeAMBEFrame(framing.DibitsToBits(cw))
		totalErrs += errs
		if err != nil {
			frames[i] = make([]byte, VoiceFrameBytes)
			continue
		}
		frames[i] = framing.PackBitsMSB(info)
	}
	return frames, totalErrs
}

func packetDUID(packet []uint8) DUID {
	var cw uint8
	for _, off := range airDUIDBody {
		cw = cw<<2 | packet[AirISCHDibits+off]&3
	}
	return DecodeDUID(cw)
}

// sischMaxErrs bounds the bit errors accepted in an S-ISCH under the
// canonical rotation; the other three rotations accept fewer, so that
// searching four patterns does not quadruple false locks on noise (the
// SuperframeDecoder's rule, issue #915).
const (
	sischMaxErrs        = 4
	sischMaxErrsRotated = 2
)

// airLockLossDibits drops the lock after two superframes without a
// recognised ISCH.
const airLockLossDibits = 2 * AirSlotsPerSuperframe * AirPacketDibits

// AirFramer frames a Phase 2 outbound dibit stream into timeslot bursts.
// It hunts the S-ISCH under all four constant dibit rotations (a residual
// carrier offset near ±1500 Hz rotates the differentially decoded stream,
// issue #915), then cuts 180-dibit packets on that grid, naming each
// packet's timeslot from its I-ISCH (or by counting from the last one,
// across the S-ISCH pairs). Bursts before the first I-ISCH are dropped:
// until then the timeslot, and so the logical channel, is unknown. Not
// safe for concurrent use.
type AirFramer struct {
	buf      []uint8
	bufStart int // absolute index of buf[0]

	locked   bool
	rot      uint8
	next     int // absolute index of the next packet to cut
	slot     int // timeslot of the next packet; -1 = not yet named
	lastGood int // absolute index of the last packet with a recognised ISCH

	sisch [4]uint64 // S-ISCH under each rotation
}

// NewAirFramer returns a framer ready to hunt.
func NewAirFramer() *AirFramer {
	f := &AirFramer{slot: -1}
	for r := 0; r < 4; r++ {
		var w uint64
		for i := 0; i < AirISCHDibits; i++ {
			d := uint8(SISCHSync>>(2*(AirISCHDibits-1-i))) & 3
			w = w<<2 | uint64((d+uint8(r))&3)
		}
		f.sisch[r] = w
	}
	return f
}

// Locked reports whether the framer is on a timeslot grid.
func (f *AirFramer) Locked() bool { return f.locked }

// Process consumes dibits whose first one has absolute index baseIdx
// (monotonic across calls) and returns the bursts completed by them, in
// stream order.
func (f *AirFramer) Process(dibits []uint8, baseIdx int) []AirBurst {
	if len(f.buf) == 0 {
		f.bufStart = baseIdx
	} else if gap := baseIdx - (f.bufStart + len(f.buf)); gap != 0 {
		// A discontinuity (dropped IQ): the grid is no longer valid.
		f.buf, f.bufStart = f.buf[:0], baseIdx
		f.unlock()
	}
	f.buf = append(f.buf, dibits...)
	var out []AirBurst
	for {
		if !f.locked && !f.hunt() {
			break
		}
		off := f.next - f.bufStart
		if off+AirPacketDibits > len(f.buf) {
			break
		}
		if b, ok := f.cut(f.buf[off : off+AirPacketDibits]); ok {
			out = append(out, b)
		}
		f.next += AirPacketDibits
		if f.locked && f.next-f.lastGood > airLockLossDibits {
			f.unlock()
		}
	}
	f.trim()
	return out
}

func (f *AirFramer) unlock() {
	f.locked, f.slot = false, -1
}

// hunt searches the buffer for an S-ISCH and locks the grid on it.
func (f *AirFramer) hunt() bool {
	start := 0
	if f.next > f.bufStart {
		start = f.next - f.bufStart
	}
	for i := start; i+AirISCHDibits <= len(f.buf); i++ {
		w := ischWordAt(f.buf[i:])
		for r := 0; r < 4; r++ {
			limit := sischMaxErrs
			if r != 0 {
				limit = sischMaxErrsRotated
			}
			if bits.OnesCount64(w^f.sisch[r]) <= limit {
				// An S-ISCH opens timeslot 2, 3, 6, 7, 10 or 11, so the
				// packets one and two before it may be I-ISCH ones that
				// name their timeslot: start the grid there rather than
				// lose them.
				back := i
				for k := 1; k <= 2 && i-k*AirPacketDibits >= 0; k++ {
					back = i - k*AirPacketDibits
				}
				f.locked, f.rot = true, uint8(r)
				f.next, f.lastGood, f.slot = f.bufStart+back, f.bufStart+i, -1
				return true
			}
		}
	}
	f.next = f.bufStart + max(start, len(f.buf)-AirISCHDibits+1)
	return false
}

// ischWordAt packs the 20 ISCH dibits at the head of d into 40 bits.
func ischWordAt(d []uint8) uint64 {
	var w uint64
	for _, x := range d[:AirISCHDibits] {
		w = w<<2 | uint64(x&3)
	}
	return w
}

// cut de-rotates one packet, names its timeslot and returns it.
func (f *AirFramer) cut(raw []uint8) (AirBurst, bool) {
	pkt := make([]uint8, AirPacketDibits)
	derot := (4 - f.rot) & 3
	for i, d := range raw {
		pkt[i] = (d + derot) & 3
	}
	start := f.next
	slot := -1
	if f.slot >= 0 {
		slot = f.slot
	}
	w := ischWordAt(pkt)
	if ii, _, ok := DecodeIISCH(w); ok {
		slot = ii.Slot()
		f.lastGood = start
	} else if bits.OnesCount64(w^SISCHSync) <= sischMaxErrs {
		f.lastGood = start
	}
	if slot < 0 {
		return AirBurst{}, false
	}
	f.slot = (slot + 1) % AirSlotsPerSuperframe
	return AirBurst{
		StartDibit:     start,
		Slot:           slot,
		LogicalChannel: airSlotLCH[slot],
		DUID:           packetDUID(pkt),
		Packet:         pkt,
	}, true
}

// trim drops buffered dibits no longer needed.
func (f *AirFramer) trim() {
	keep := f.next - f.bufStart
	if !f.locked {
		// Keep two packets of history for hunt's look-back.
		keep = len(f.buf) - (2*AirPacketDibits + AirISCHDibits - 1)
	}
	if keep <= 0 {
		return
	}
	keep = min(keep, len(f.buf))
	f.buf = append(f.buf[:0], f.buf[keep:]...)
	f.bufStart += keep
}
