package voice

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
)

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
	// carrier: it is the burst-A start's parity over the physical-burst
	// period ((StartDibit / (step/2)) mod 2), so the two timeslots —
	// whose burst-A anchors sit exactly one physical burst apart (132
	// dibits, or 144 with an inter-burst CACH) — always carry different,
	// stable phases for the life of a call. It is a relative discriminator,
	// NOT an absolute TS1/TS2
	// label (the BS-sourced sync word is identical on both slots);
	// binding a phase to a talkgroup is the embedded-LC decoder's job.
	// Always 0 for a single-slot Decoder.
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
	// HasTalkerAlias reports that this superframe's embedded Full LC was a
	// talker-alias header or block (FLCO 0x04-0x07). TalkerAlias then holds
	// the parsed fragment; the voice chain feeds it to a
	// dmr.TalkerAliasAssembler to reassemble the radio's display name across
	// the header + block superframes. Independent of HasLC's group-voice
	// interpretation, which does not apply to an alias LC.
	HasTalkerAlias bool
	TalkerAlias    dmr.TalkerAliasFragment
	// HasGPS reports that this superframe's embedded Full LC was a GPS Info
	// PDU (FLCO 0x08). GPS then holds the decoded position; the voice chain
	// binds it to the call's source radio (the LC carries no address) and
	// publishes it on the location bus.
	HasGPS bool
	GPS    dmr.GPSInfo
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

// cachDibits is the Common Announcement Channel that precedes each burst
// on a DMR outbound (BS-sourced) carrier — 24 bits = 12 dibits (ETSI
// TS 102 361-1 §6.2). It sits between the demodulated bursts, so on
// outbound air a call's same-slot bursts are 2*(132+12)=288 dibits apart
// rather than the 2*132=264 of a CACH-free (e.g. direct-mode replayed)
// stream. The interleaved decoder auto-detects which of the two is in use.
const cachDibits = 12

// Decoder extracts DMR voice superframes from a dibit stream. It is
// the voice-burst counterpart of the tier2 / tier3 control-channel
// Process adapters: those slice a burst on every sync match, but
// voice bursts B–F carry no sync, so the Decoder locks onto burst A
// via its voice sync word and slices B–F at a fixed TDMA cadence — the
// same-slot stride, in dibits, between two consecutive bursts of the
// SAME logical call:
//
//   - single-slot — a direct / single-timeslot stream where a call's
//     bursts A–F are contiguous (132-dibit cadence). This is NewDecoder.
//   - interleaved — a real 2-slot TDMA carrier (NewInterleavedDecoder),
//     where the other timeslot's burst sits between each of a call's
//     bursts. Same-slot bursts are 264 dibits apart with no inter-burst
//     CACH, or 288 when a CACH precedes each burst on outbound air; the
//     decoder auto-detects which is in use per call (see resolveAndSlice)
//     and locks it. One such Decoder transparently emits superframes for
//     BOTH slots and tells them apart by VoiceSuperframe.Phase.
//
// A Decoder is stateful and not safe for concurrent use; construct
// one per voice-call decode chain.
type Decoder struct {
	det      *dmr.SyncDetector
	buf      []uint8
	bufStart int // absolute dibit index of buf[0]
	pending  []dmr.Match
	// cadenceCandidates holds the same-slot burst stride(s) in dibits the
	// decoder may use, smallest first. A single-slot decoder has exactly
	// one (132). An interleaved decoder lists the 2-slot cadences it
	// auto-detects between: 264 (no inter-burst CACH) and 288 (a 12-dibit
	// CACH precedes each burst on live BS-sourced outbound air).
	cadenceCandidates []int
	// lockedStep is the same-slot stride (dibits) the decoder has committed
	// to for the rest of the call, or 0 while still undetected. Detection
	// picks the candidate whose bursts B–E reassemble a CRC-valid embedded
	// Link Control (sliceAt → HasLC); a wrong cadence cannot.
	lockedStep int
	// lockedByLC records HOW lockedStep was set: true once a CRC-valid embedded
	// Link Control confirmed the cadence (authoritative — nothing overrides it),
	// false while the lock is only a provisional AMBE-FEC-quality guess. A
	// provisional lock stays re-checkable every superframe so a wrong early
	// heuristic guess can be corrected by a later CRC-valid LC (or a later clear
	// FEC winner at a different stride) — without this a single confident-but-
	// wrong lock garbles the whole rest of the call and can never self-correct,
	// because a wrong slice never reassembles the LC that would fix it (#644).
	// Set true for a single-cadence (single-slot) decoder: its one stride is not
	// a guess, so its decode path is unchanged.
	lockedByLC bool
	// maxSpan is the dibit span of a full A–F superframe at the LARGEST
	// candidate stride. The Process gate waits for this many dibits past an
	// anchor before slicing, so every candidate is fully buffered when
	// detection runs.
	maxSpan int
	bufKeep int // dibits retained so two in-flight anchors can complete
}

// spanFor is the dibit span from a superframe's burst-A first dibit to its
// burst-F last dibit when consecutive same-slot bursts are step dibits apart.
func spanFor(step int) int { return (BurstsPerSuperframe-1)*step + dmr.BurstDibits }

func newDecoder(candidates []int) *Decoder {
	maxStep := candidates[0]
	for _, c := range candidates {
		if c > maxStep {
			maxStep = c
		}
	}
	maxSpan := spanFor(maxStep)
	d := &Decoder{
		det:               dmr.NewSyncDetector(voiceSyncs, 2),
		cadenceCandidates: candidates,
		maxSpan:           maxSpan,
		// Retain two full (largest-cadence) superframe spans plus a burst
		// so the two interleaved slots' anchors can each complete without
		// the trailing bursts being trimmed out from under them.
		bufKeep: 2*maxSpan + dmr.BurstDibits,
	}
	// A single-cadence decoder has nothing to detect: lock immediately, and
	// authoritatively (its one stride is not a guess) so its decode path is
	// unchanged by the provisional-lock re-checking below.
	if len(candidates) == 1 {
		d.lockedStep = candidates[0]
		d.lockedByLC = true
	}
	return d
}

// NewDecoder returns a single-slot Decoder: a call's bursts A–F are
// expected back-to-back at the 132-dibit cadence.
func NewDecoder() *Decoder { return newDecoder([]int{dmr.BurstDibits}) }

// NewInterleavedDecoder returns a Decoder for a real 2-slot DMR TDMA
// carrier: a call's bursts are interleaved with the other timeslot's, so
// same-slot bursts are 264 dibits apart with no inter-burst CACH, or 288
// when a 12-dibit CACH precedes each burst on live BS-sourced outbound air.
// The decoder auto-detects which cadence is in use per call — the one whose
// bursts B–E reassemble a CRC-valid embedded Link Control — locks it, then
// emits superframes for both slots tagged by VoiceSuperframe.Phase.
func NewInterleavedDecoder() *Decoder {
	return newDecoder([]int{2 * dmr.BurstDibits, 2 * (dmr.BurstDibits + cachDibits)})
}

// Reset clears all buffered state. Call on a stream re-sync so a stale
// burst-A anchor does not slice across the discontinuity. The cadence
// candidates and any already-locked cadence are preserved — the same
// call's same-slot stride does not change across a re-sync.
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
		if start+d.maxSpan > bufEnd {
			keep = append(keep, m) // trailing bursts not buffered yet
			continue
		}
		if start < d.bufStart {
			continue // anchor fell off the front of the buffer
		}
		out = append(out, d.resolveAndSlice(start, m.Pattern.Name))
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

// ambeCadenceLockCeiling and ambeCadenceLockMargin gate locking a cadence by
// AMBE FEC quality (see resolveAndSlice). A superframe sliced at the correct
// same-slot stride decodes its 18 AMBE frames with very few Golay-corrected
// bits; the wrong stride pulls bits from the other timeslot / the inter-burst
// CACH and scores far higher (18 frames × up to 6 corrected bits = 108 max).
//   - ceiling caps the winner's total corrected bits: ~24 ≈ 1.3 bits/frame, a
//     generous weak-signal allowance still well under the ~4–5/frame a random
//     (wrong-cadence) slice averages.
//   - margin requires the runner-up to exceed 2×winner by this floor, so two
//     similarly-bad (both-garbled) candidates never lock the wrong one.
const (
	ambeCadenceLockCeiling = 24
	ambeCadenceLockMargin  = 6
)

// ambeErrorScore sums the Golay(23,12) corrected-bit count across a
// superframe's 18 AMBE frames. It is the cadence-quality metric that lets the
// interleaved decoder detect the same-slot stride WITHOUT the embedded Link
// Control, whose FEC/CRC constants are still unvalidated against live air — the
// coupling that left a 288-cadence (CACH) carrier sounding garbled when its LC
// never decoded (issue #644).
func ambeErrorScore(sf VoiceSuperframe) int {
	score := 0
	for i := range sf.Frames {
		// DecodeAMBEFrame only errors on a wrong-length frame; here every
		// frame is a full 72 bits, so err is always nil and we score the
		// corrected-bit count.
		if _, errs, err := DecodeAMBEFrame(sf.Frames[i]); err == nil {
			score += errs
		}
	}
	return score
}

// resolveAndSlice produces the superframe anchored at start.
//
// A CRC-valid embedded Link Control is authoritative: once one confirms the
// cadence (lockedByLC) the decoder slices at that stride for the rest of the
// call. Absent any authoritative lock — undetected, or only a provisional
// AMBE-FEC-quality guess — it re-scores every candidate this superframe and:
//   - a bursts-B–E CRC-valid LC on any candidate locks it authoritatively;
//   - otherwise the clearly-best candidate by AMBE FEC quality (ambeErrorScore)
//     becomes a PROVISIONAL lock, so a real carrier whose LC never decodes
//     still finds its cadence (issue #644);
//   - with no clear winner it falls back to the smallest candidate.
//
// Re-scoring while the lock is only provisional is what lets a wrong early FEC
// guess be corrected — by a later CRC-valid LC, or by a later clear FEC winner
// at a different stride. Without it a single confident-but-wrong lock would
// garble the whole rest of the call and could never self-correct, since a wrong
// slice never reassembles the LC that would fix it (the reopened #644 garble).
// The caller has guaranteed maxSpan dibits past start are buffered, so every
// candidate is sliceable.
func (d *Decoder) resolveAndSlice(start int, syncName string) VoiceSuperframe {
	// An LC-confirmed cadence is authoritative — slice at it directly.
	if d.lockedStep != 0 && d.lockedByLC {
		return d.sliceAt(start, d.lockedStep, syncName)
	}

	bestIdx, bestScore, secondScore := -1, math.MaxInt, math.MaxInt
	var bestSF VoiceSuperframe
	for i, step := range d.cadenceCandidates {
		sf := d.sliceAt(start, step, syncName)
		if sf.HasLC {
			// Authoritative: a CRC-valid LC overrides any provisional lock.
			d.lockedStep, d.lockedByLC = step, true
			return sf
		}
		switch s := ambeErrorScore(sf); {
		case s < bestScore:
			secondScore = bestScore
			bestScore, bestIdx, bestSF = s, i, sf
		case s < secondScore:
			secondScore = s
		}
	}

	clearWinner := bestIdx >= 0 && bestScore <= ambeCadenceLockCeiling &&
		bestScore*2+ambeCadenceLockMargin <= secondScore

	if d.lockedStep != 0 {
		// Provisionally (FEC-)locked. Keep the current stride unless the data
		// now clearly favours a different one — then move the provisional lock.
		if clearWinner && d.cadenceCandidates[bestIdx] != d.lockedStep {
			d.lockedStep = d.cadenceCandidates[bestIdx]
			return bestSF
		}
		return d.sliceAt(start, d.lockedStep, syncName)
	}

	if clearWinner {
		d.lockedStep = d.cadenceCandidates[bestIdx] // provisional — lockedByLC stays false
		return bestSF
	}
	return d.sliceAt(start, d.cadenceCandidates[0], syncName)
}

// sliceAt cuts the six 132-dibit bursts of one logical call whose burst A
// starts at absolute dibit index start, with consecutive same-slot bursts
// step dibits apart, and extracts their 18 AMBE frames + embedded LC. On a
// 2-slot stream step skips the interleaved other-slot burst (and any CACH).
// The caller has already confirmed the full span is buffered.
func (d *Decoder) sliceAt(start, step int, syncName string) VoiceSuperframe {
	sf := VoiceSuperframe{
		SyncName:   syncName,
		StartDibit: start,
	}
	// Phase distinguishes the two interleaved calls: their burst-A anchors
	// sit one physical-burst period (step/2 dibits) apart, so this parity is
	// stable and distinct per slot. A single-cadence (single-slot) decoder
	// has no second slot, so Phase stays 0.
	if len(d.cadenceCandidates) > 1 {
		sf.Phase = uint8((start / (step / 2)) % 2)
	}
	off := start - d.bufStart
	frame := 0
	// frags maps the four embedded-LC-bearing bursts B,C,D,E (burst
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
	if info, lc, ok := dmr.ReassembleEmbeddedLCInfo(frags); ok {
		sf.HasLC = true
		sf.LC = lc
		// A talker-alias header/block reuses the Full LC framing but its
		// octets 2..8 are alias text, not the group/source addresses the
		// generic LC view reads. Surface the parsed fragment so the voice
		// chain can reassemble the display name across superframes.
		if frag, ok := dmr.ParseTalkerAliasFragment(info); ok {
			sf.HasTalkerAlias = true
			sf.TalkerAlias = frag
		}
		// GPS Info reuses the Full LC framing too — its payload is a position
		// fix, not the group/source addresses the generic LC view reads.
		if gps, ok := dmr.ParseGPSInfo(info); ok {
			sf.HasGPS = true
			sf.GPS = gps
		}
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
