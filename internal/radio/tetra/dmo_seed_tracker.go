package tetra

// Per-transmission scramble-seed tracking for TETRA DMO traffic.
//
// What the 12 Sep #1003 capture (three consecutive PTTs on 438.9 MHz, clear
// TEA0, Motorola MTP8500Ex) established, by solving each DNB's scramble seed
// exactly (dmo_seed.go) and lining it up against the DSB's decoded DM-SYNC:
//
//   - the TCH/S scramble seed CHANGES FROM ONE PTT TO THE NEXT (0x012915c0,
//     0x015d9c07, 0x01671384 on the three transmissions) — it is not a network
//     constant, so no configured colour / MNI can describe it and any brute
//     force over "colour codes" was doomed to pick a partial-keystream artifact
//     (colour 36 / 31 / 39 / 3 on the earlier runs: all bogus);
//   - the seed's low 24 bits are, bit for bit, the 24-bit field at SCH/H bits
//     42..65 of that transmission's DM-SYNC PDU — the field immediately before
//     the 24-bit MNI at bits 66..89, which decodes as MCC 250 / MNC 1, exactly
//     the operator's codeplug (so the surrounding layout is the standard
//     DMAC-SYNC SCH/H: … source address type, source address, MNI, message
//     type …, and the field is the transmitting MS's source address);
//   - the seed's top 6 bits (e(1..6)) were 000001 on all three.
//
// Neither reference decoder (osmo-tetra-dmo, TetraDMO-Receiver) carries a DMO
// traffic seed rule at all, so the constants below are capture-pinned rather
// than spec-quoted; they are only ever used as a HINT that traffic must confirm.
// The exact per-burst solve is the authority: it needs no field layout, no
// prefix and no configuration, and a single error-free DNB is enough.

const (
	// DMSCHHSeedFieldOffset / DMSCHHSeedFieldBits locate the 24-bit DM-SYNC
	// SCH/H field the traffic seed's low 24 bits are copied from.
	DMSCHHSeedFieldOffset = 42
	DMSCHHSeedFieldBits   = 24
	// DMScrambleSeedPrefix is e(1..6) of the DMO traffic seed as observed on air
	// (three transmissions, one network). Capture-pinned; see the file comment.
	DMScrambleSeedPrefix uint32 = 1
)

// DMSyncSCHHScrambleSeed derives the traffic scramble seed a transmission
// announces in its DSB SCH/H (the decoded 124 type-1 bits of the DM-SYNC PDU's
// SCH/H half): DMScrambleSeedPrefix || bits 42..65. It is a hint — the
// DMSeedTracker confirms it against the DNB traffic before trusting it.
func DMSyncSCHHScrambleSeed(schh []byte) (seed uint32, ok bool) {
	if len(schh) < DMSCHHSeedFieldOffset+DMSCHHSeedFieldBits {
		return 0, false
	}
	return DMScrambleSeedPrefix<<24 | bitsToUint(schh, DMSCHHSeedFieldOffset, DMSCHHSeedFieldBits), true
}

// dmSeedHintMinCRC is how many CRC-valid TCH/S decodes at a hinted seed confirm
// the hint when no burst solves exactly (a weak transmission). A wrong seed
// passes the 8-bit class-2 CRC ~1/128 of bursts (soft and hard attempts), so
// three passes inside a dmSeedHintWindow-burst window is a ~1e-4 false accept
// per window (two was measurably not enough: a synthetic 90-burst random
// payload confirmed a wrong hint); a correct hint on a real transmission clears
// it in three bursts. The exact solve, which any error-free burst delivers,
// overrides a hint either way.
const (
	dmSeedHintMinCRC = 3
	dmSeedHintWindow = 12
)

// DMSeedTracker learns the scramble seed of the DMO transmission in progress
// from its bursts, per transmission. Feed it every DSB (ObserveDSB: the SCH/H
// hint) and every slot-grid-qualified DNB (ObserveDNB: the exact solve, and the
// decode itself). It is shared by the control pipeline and the voice chain so
// both see the same rule. Not safe for concurrent use.
//
// Precedence: an exact solve (SolveTCHScrambleSeed) adopts immediately — a
// solution is a 128-check-redundant proof that the burst is a TCH/S codeword
// under that seed, so it cannot be a chance hit — and it also preempts a
// previously adopted seed, which is how a new PTT with a new seed takes over.
// The SCH/H hint (or an external hint, e.g. the pipeline's answer polled by the
// voice chain) is adopted only after dmSeedHintMinCRC CRC-valid decodes, and
// only while no exact solve has spoken for the current transmission. A pinned
// seed (configuration) is never changed.
type DMSeedTracker struct {
	seed     uint32
	known    bool
	verified bool // known by exact solve or a CRC-confirmed hint (never the fallback)
	pinned   bool

	hint     uint32
	hintSet  bool
	hintCRC  int
	hintSeen int

	// Counters for the status/ended logs.
	ExactAdopts int // seeds adopted from an exact per-burst solve
	HintAdopts  int // seeds adopted from a CRC-confirmed hint
	Solved      int // bursts that solved exactly
}

// NewDMSeedTracker returns an empty tracker (no seed known).
func NewDMSeedTracker() *DMSeedTracker { return &DMSeedTracker{} }

// Seed returns the seed currently in use and whether one is known at all
// (verified or fallback).
func (t *DMSeedTracker) Seed() (uint32, bool) { return t.seed, t.known }

// Verified reports whether the current seed was confirmed by the traffic (an
// exact solve or a CRC-confirmed hint), as opposed to a fallback guess.
func (t *DMSeedTracker) Verified() bool { return t.known && t.verified }

// Pin fixes the seed from configuration; nothing observed afterwards changes it.
func (t *DMSeedTracker) Pin(seed uint32) {
	t.seed, t.known, t.verified, t.pinned = seed, true, true, true
}

// Adopt installs an externally verified seed (the control pipeline's answer,
// carried on the grant). Ignored when pinned.
func (t *DMSeedTracker) Adopt(seed uint32) {
	if t.pinned {
		return
	}
	t.seed, t.known, t.verified = seed, true, true
	t.hintCRC, t.hintSeen = 0, 0
}

// Fallback installs an unverified seed to decode at when nothing better is
// known (the voice chain's give-up path). A later exact solve replaces it.
func (t *DMSeedTracker) Fallback(seed uint32) {
	if t.pinned || t.known {
		return
	}
	t.seed, t.known, t.verified = seed, true, false
}

// Hint offers a candidate seed from outside the burst stream (the pipeline's
// live colour polled by the voice chain). It is confirmed like an SCH/H hint.
func (t *DMSeedTracker) Hint(seed uint32) {
	if t.pinned || (t.hintSet && t.hint == seed) {
		return
	}
	t.hint, t.hintSet, t.hintCRC, t.hintSeen = seed, true, 0, 0
}

// Reset forgets the transmission's seed and hint (a traffic drought ended it);
// a pinned seed survives.
func (t *DMSeedTracker) Reset() {
	if t.pinned {
		t.hintSet, t.hintCRC, t.hintSeen = false, 0, 0
		return
	}
	*t = DMSeedTracker{ExactAdopts: t.ExactAdopts, HintAdopts: t.HintAdopts, Solved: t.Solved}
}

// ObserveDSB decodes a DSB's SCH/H and records the seed it announces as the
// current hint. Returns the hint and whether the SCH/H decoded CRC-clean.
func (t *DMSeedTracker) ObserveDSB(b DMBurst) (hint uint32, ok bool) {
	schh, ok := DecodeDMSCHH(b)
	if !ok {
		return 0, false
	}
	hint, ok = DMSyncSCHHScrambleSeed(schh)
	if !ok {
		return 0, false
	}
	t.Hint(hint)
	return hint, true
}

// HintKnown returns the pending hint (from a DSB SCH/H or Hint) and whether one
// is set.
func (t *DMSeedTracker) HintKnown() (uint32, bool) { return t.hint, t.hintSet }

// ObserveDNB decodes one DNB at the best seed available, learning the seed on
// the way. frames are the two 137-bit speech frames (nil when the burst yields
// none — noise, an SCH/F DNB, or a seed not yet known), seed is the seed in use
// after the call, and adopted reports that this burst changed it.
//
// qualified says the burst passed the caller's slot-grid gate (tetra.DMSlotGrid).
// The exact solve is allowed on any burst — a solution cannot come from noise —
// but only a qualified burst may CRC-confirm a hint, since a correlator false
// alarm passes the 8-bit CRC at any seed ~1/128 of the time.
func (t *DMSeedTracker) ObserveDNB(b DMBurst, qualified bool) (frames [][]byte, seed uint32, adopted bool) {
	if b.Kind != DMBurstNormal {
		return nil, t.seed, false
	}
	// Solve first, even with a seed in hand: a stale seed from the previous PTT
	// can still CRC-pass an occasional burst of the new one (related seeds share
	// keystream structure — measured ~9% on air for a "close" wrong seed, and
	// the 8-bit CRC alone admits 1/256 at random), which would emit garbage
	// speech and delay the switch. The solve costs microseconds and is exact.
	if s, ok := DMBurstScrambleSeed(b); ok && !t.pinned {
		t.Solved++
		if !t.known || s != t.seed {
			t.seed, t.known, t.verified = s, true, true
			t.ExactAdopts++
			t.hintCRC, t.hintSeen = 0, 0
			adopted = true
		}
		return dmDecodeDNB(b, t.seed), t.seed, adopted
	}
	if t.known {
		if frames = dmDecodeDNB(b, t.seed); frames != nil {
			return frames, t.seed, false
		}
		if t.pinned {
			return nil, t.seed, false
		}
	}
	// No exact solution (bit errors): let a pending hint earn adoption by CRC.
	if qualified && t.hintSet && (!t.known || t.hint != t.seed) {
		if t.hintSeen++; t.hintSeen > dmSeedHintWindow {
			t.hintSeen, t.hintCRC = 1, 0
		}
		if frames = dmDecodeDNB(b, t.hint); frames != nil {
			t.hintCRC++
			if t.hintCRC >= dmSeedHintMinCRC {
				t.seed, t.known, t.verified = t.hint, true, true
				t.HintAdopts++
				t.hintCRC, t.hintSeen = 0, 0
				adopted = true
			}
			return frames, t.seed, adopted
		}
	}
	return nil, t.seed, false
}

// dmDecodeDNB decodes a DNB's TCH/S at seed, soft-decision first with the hard
// fallback the production paths use.
func dmDecodeDNB(b DMBurst, seed uint32) [][]byte {
	if frames := DMBurstTCHSpeechSoft(b, seed); len(frames) == 2 {
		return frames
	}
	if frames := DMBurstTCHSpeech(b, seed); len(frames) == 2 {
		return frames
	}
	return nil
}
