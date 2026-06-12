package voice

import "github.com/MattCheramie/GopherTrunk/internal/radio/dmr"

// VoiceSuperframe is one decoded DMR voice superframe: the 18 on-air
// AMBE+2 frames carried by bursts A–F. Each frame is 72 bits, one bit
// per byte MSB-first; AMBE forward-error-correction has not yet been
// applied.
type VoiceSuperframe struct {
	// Frames holds the 18 AMBE frames in transmission order — bursts
	// A..F, three frames per burst.
	Frames [FramesPerSuperframe][]byte
	// SyncName is the voice sync word that framed burst A:
	// "BS-Voice", "MS-Voice", "DM-Voice-TS1" or "DM-Voice-TS2".
	SyncName string
	// StartDibit is the absolute dibit index of burst A's first dibit.
	StartDibit int
	// Phase distinguishes the two interleaved calls on a 2-slot TDMA
	// carrier: it is the burst-A start's parity in burst units
	// ((StartDibit / BurstDibits) mod stride), so the two timeslots —
	// whose burst-A anchors sit exactly one physical burst (132 dibits)
	// apart — always carry different, stable phases for the life of a
	// call. It is a relative discriminator, NOT an absolute TS1/TS2
	// label (the BS-sourced sync word is identical on both slots);
	// binding a phase to a talkgroup is the embedded-LC decoder's job.
	// Always 0 for a single-slot (stride 1) Decoder.
	Phase uint8
	// HasLC reports that a Full Link Control was reassembled from the
	// embedded signalling carried by bursts B–E of this superframe and
	// passed its BPTC + CRC check. When true, LC holds the decoded PDU —
	// its destination/source addresses identify the call's talkgroup and
	// radio, which a consumer binds to the superframe's Phase to label
	// the timeslot (the BS-sourced burst-A sync alone cannot).
	HasLC bool
	LC    dmr.FLC
	// HasRC reports that a Reverse Channel word was decoded from a single-
	// fragment (LCSS == LCSSSingle) embedded-signalling field carried by one
	// of bursts B–E; RC then holds the 11-bit RC information. This is in-call
	// signalling independent of HasLC/LC — see dmr/rc.go, including the
	// capture-pending FEC posture.
	HasRC bool
	RC    dmr.ReverseChannel
}

// voiceSyncs are the sync words that frame burst A of a voice
// superframe. Bursts B–F carry embedded signalling instead, so they
// produce no sync match and are located by TDMA cadence.
var voiceSyncs = []dmr.SyncPattern{
	dmr.BSVoice, dmr.MSVoice, dmr.DMVoice1, dmr.DMVoice2,
}

// burstLookback is the dibit distance from a sync match (the index of
// the last sync dibit) back to the burst's first dibit: 54 payload
// dibits + 24 sync dibits − 1.
const burstLookback = VoiceHalfDibits + 24 - 1 // 77

// superframeDibits is the dibit span of a full A–F superframe on a
// single-slot (stride 1) stream — six contiguous 132-dibit bursts.
const superframeDibits = dmr.BurstDibits * BurstsPerSuperframe // 792

// Decoder extracts DMR voice superframes from a dibit stream. It is
// the voice-burst counterpart of the tier2 / tier3 control-channel
// Process adapters: those slice a burst on every sync match, but
// voice bursts B–F carry no sync, so the Decoder locks onto burst A
// via its voice sync word and slices B–F at a fixed TDMA cadence.
//
// stride is the number of physical 132-dibit bursts between two
// consecutive bursts of the SAME logical call:
//
//   - stride 1 — a single-slot / direct stream where a call's bursts
//     A–F are contiguous (132-dibit cadence). This is NewDecoder and
//     matches synthetic single-slot vectors.
//   - stride 2 — a real 2-slot TDMA carrier (NewInterleavedDecoder),
//     where the other timeslot's burst sits between each of a call's
//     bursts, so same-slot bursts are 264 dibits apart. One such
//     Decoder transparently emits superframes for BOTH slots: it locks
//     each slot's burst A on its own voice sync and gathers that slot's
//     B–F by striding over the interleaved burst. The two slots' output
//     is told apart by VoiceSuperframe.Phase.
//
// A Decoder is stateful and not safe for concurrent use; construct
// one per voice-call decode chain.
type Decoder struct {
	det      *dmr.SyncDetector
	buf      []uint8
	bufStart int // absolute dibit index of buf[0]
	pending  []dmr.Match
	stride   int // physical bursts between consecutive same-slot bursts
	span     int // dibit span of a full A–F superframe at this stride
	bufKeep  int // dibits retained so two in-flight anchors can complete
}

func newDecoder(stride int) *Decoder {
	// Bursts sit at start + b*stride*BurstDibits for b in 0..5, so the
	// span runs from burst A's first dibit to burst F's last.
	span := ((BurstsPerSuperframe-1)*stride + 1) * dmr.BurstDibits
	return &Decoder{
		det:    dmr.NewSyncDetector(voiceSyncs, 2),
		stride: stride,
		span:   span,
		// Retain two full superframe spans plus a burst so the two
		// interleaved slots' anchors can each complete without the
		// trailing bursts being trimmed out from under them.
		bufKeep: 2*span + dmr.BurstDibits,
	}
}

// NewDecoder returns a single-slot Decoder (stride 1): a call's bursts
// A–F are expected back-to-back at the 132-dibit cadence.
func NewDecoder() *Decoder { return newDecoder(1) }

// NewInterleavedDecoder returns a Decoder for a real 2-slot DMR TDMA
// carrier (stride 2): a call's bursts are 264 dibits apart because the
// other timeslot's burst is interleaved between them. It emits
// superframes for both slots, tagged by VoiceSuperframe.Phase.
//
// NOTE: stride 2 assumes the two slots' bursts are adjacent 132-dibit
// units with no inter-burst CACH / guard dibits in the demodulated
// stream. The exact same-slot cadence on live BS-sourced outbound air
// (where a CACH may precede each burst) should be confirmed against a
// real IQ capture before this replaces NewDecoder on the production
// voice path; see docs/status.md.
func NewInterleavedDecoder() *Decoder { return newDecoder(2) }

// Reset clears all buffered state. Call on a stream re-sync so a stale
// burst-A anchor does not slice across the discontinuity. The stride
// the Decoder was constructed with is preserved.
func (d *Decoder) Reset() {
	d.buf = d.buf[:0]
	d.bufStart = 0
	d.pending = d.pending[:0]
	d.det = dmr.NewSyncDetector(voiceSyncs, 2)
}

// Process consumes a window of dibits and returns every voice
// superframe that completed within it. baseIdx is the absolute dibit
// index of dibits[0]; it must be monotonically non-decreasing across
// calls. Superframes are returned in stream order.
func (d *Decoder) Process(dibits []uint8, baseIdx int) []VoiceSuperframe {
	if len(d.buf) == 0 {
		d.bufStart = baseIdx
	}
	d.buf = append(d.buf, dibits...)

	matches, _ := d.det.Process(nil, dibits, baseIdx)
	d.pending = append(d.pending, matches...)

	var out []VoiceSuperframe
	bufEnd := d.bufStart + len(d.buf)
	keep := d.pending[:0]
	for _, m := range d.pending {
		start := m.Index - burstLookback
		if start+d.span > bufEnd {
			keep = append(keep, m) // trailing bursts not buffered yet
			continue
		}
		if start < d.bufStart {
			continue // anchor fell off the front of the buffer
		}
		out = append(out, d.sliceSuperframe(start, m.Pattern.Name))
	}
	d.pending = keep

	if len(d.buf) > d.bufKeep {
		drop := len(d.buf) - d.bufKeep
		copy(d.buf, d.buf[drop:])
		d.buf = d.buf[:d.bufKeep]
		d.bufStart += drop
	}
	return out
}

// sliceSuperframe cuts the six 132-dibit bursts of one logical call
// starting at absolute dibit index start and extracts their 18 AMBE
// frames. Consecutive bursts are stride*132 dibits apart, so on a
// 2-slot stream the interleaved other-slot burst is skipped. The caller
// has already confirmed the full span is buffered.
func (d *Decoder) sliceSuperframe(start int, syncName string) VoiceSuperframe {
	sf := VoiceSuperframe{
		SyncName:   syncName,
		StartDibit: start,
		Phase:      uint8((start / dmr.BurstDibits) % d.stride),
	}
	off := start - d.bufStart
	step := d.stride * dmr.BurstDibits
	frame := 0
	// fragIdx maps the four embedded-LC-bearing bursts B,C,D,E (burst
	// indices 1..4) to their fragment slot 0..3; A and F carry none.
	var frags [4][]byte
	for b := 0; b < BurstsPerSuperframe; b++ {
		var burst dmr.Burst
		copy(burst.Dibits[:], d.buf[off+b*step:off+b*step+dmr.BurstDibits])
		for _, f := range AMBEFrames(&burst) {
			sf.Frames[frame] = f
			frame++
		}
		if b >= 1 && b <= 4 {
			emb, frag := dmr.SplitEmbeddedField(dibitsToBits(burst.Sync()))
			frags[b-1] = frag
			// A single-fragment (LCSS==Single) embedded field is not part of
			// the four-fragment Full LC: it carries either a Reverse Channel
			// word or the null idle. Decode the first RC seen in the
			// superframe; the LC reassembly below is gated by its own
			// BPTC+CRC, so leaving the fragment in frags is harmless.
			if emb.LCSS == dmr.LCSSSingle && !sf.HasRC {
				if rc, ok := dmr.DecodeReverseChannel(frag); ok {
					sf.HasRC = true
					sf.RC = rc
				}
			}
		}
	}
	if lc, ok := dmr.ReassembleEmbeddedLC(frags); ok {
		sf.HasLC = true
		sf.LC = lc
	}
	return sf
}

// dibitsToBits expands dibits to bits, two per dibit, MSB-first.
func dibitsToBits(dibits []uint8) []byte {
	out := make([]byte, 0, len(dibits)*2)
	for _, d := range dibits {
		out = append(out, (d>>1)&1, d&1)
	}
	return out
}
