package phase2

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

// probeSweepInterval bounds how often the self-aligning ScramblerProbe
// re-runs its full-sequence phase sweep while it has not yet pinned a
// channel's PN44 phase. Sweeping the whole 4320-bit sequence on every MAC
// burst of an undecodable channel is expensive — the FEC-heavy work,
// multiplied across the concurrent voice + signalling follows a busy
// multi-site Phase 2 system runs, starves the live IQ consumer, which is why
// the reporter saw probe collapse superframe sync entirely ("probe →
// superframes=0", issue #915). Once a channel decodes, the phase pins on the
// first RS-valid burst and the sweep never runs again; between misses it is
// amortised to one full sweep per this many MAC bursts. Counted in MAC
// bursts, not superframes.
const probeSweepInterval = 24

// ScramblerPin caches the PN44 channel-bit sequence phase that
// ScramblerProbe discovers for one P25 Phase 2 traffic channel, so the
// self-aligning sweep runs once (then O(1) per burst) instead of on every
// MAC PDU.
//
// Why the pin exists (issue #915, Finding B). The completed-call source RID
// rides an in-call MAC PDU that is PN44-scrambled on the coded channel bits
// (TIA-102.BBAC-1 §7.2.5). Descrambling needs the burst's phase into the
// continuous 4320-bit superframe sequence. GopherTrunk derives that phase
// from its superframe grid (slotChannelPN44Offset = slot·360 + 64, where 64
// is MACPayloadOffset·2), but the intra-slot offset is a working assumption —
// the reference decoders place the MAC payload at slot·360 + 20 — so the
// fixed offset can be wrong by a constant, and then the burst never satisfies
// the outer RS(24,16,9) parity for any seed (the reporter's mac_rs_valid=0).
//
// ScramblerProbe resolves this by trying candidate phases against the RS
// gate: a wrong phase packs into random bytes that satisfy the 8-symbol
// syndrome with probability ≈2^-48, so the first RS-valid phase is the true
// one. The pre-existing probe swept only the 12 slot BASES while holding the
// intra-slot offset fixed at the (possibly wrong) grid assumption, so it
// could never recover a channel whose true intra-slot offset differs — this
// pin sweeps the full sequence phase, which does, and remembers it.
//
// Not safe for concurrent use: construct one per call / follow and drive it
// from the single receiver goroutine (MACDispatcher owns one).
type ScramblerPin struct {
	seq  []byte // precomputed pn44SuperframeBits keystream for seed
	seed uint64

	have  bool
	phase int // pinned sequence-phase offset (0..pn44SuperframeBits-1)

	sweptOnce  bool
	sinceSweep int
}

// keystream returns the cached PN44 keystream for seed, recomputing it (and
// dropping any pinned phase) when the seed changes.
func (p *ScramblerPin) keystream(seed uint64) []byte {
	if p.seq != nil && p.seed == seed {
		return p.seq
	}
	seq := make([]byte, pn44SuperframeBits)
	s := framing.NewPN44Scrambler(seed)
	for i := range seq {
		seq[i] = s.Next()
	}
	p.seq = seq
	p.seed = seed
	p.have = false
	p.sweptOnce = false
	p.sinceSweep = 0
	return seq
}

// descrambleWithSeq XORs bits in place with the precomputed keystream
// starting at offset (folded into the sequence period). It is the O(1)-per-
// bit equivalent of descrambleAtOffset's Advance-then-Apply, so the phase
// sweep does not re-clock the LFSR from zero on every candidate.
func descrambleWithSeq(bits, seq []byte, offset int) {
	n := len(seq)
	offset %= n
	if offset < 0 {
		offset += n
	}
	for i := range bits {
		bits[i] ^= seq[offset]
		offset++
		if offset == n {
			offset = 0
		}
	}
}

// decodeAtPhase descrambles macDibits at the given channel-bit phase using
// the precomputed keystream and runs the RS-gated FEC + parse chain. The RS
// gate is what lets the probe tell the true phase from a wrong one, so it is
// always on here regardless of the channel's RSMode.
func decodeAtPhase(macDibits []uint8, mode TrellisMode, interleave InterleaveMode, seq []byte, phase int) (MACPDU, bool) {
	bits := framing.DibitsToBits(macDibits)
	descrambleWithSeq(bits, seq, phase)
	return decodeMACChannelAndParse(framing.BitsToDibits(bits), mode, interleave, RSOn)
}

// probeSubframe recovers the MAC PDU from one MAC sub-frame under the
// self-aligning ScramblerProbe, using and updating the pin. slotIndex is the
// sub-frame's 0..11 position within the superframe (known once superframe
// sync has locked), so the candidate phase is slotIndex·360 + phase. Because
// the intra-slot offset is constant across the 12 slots, a phase discovered
// on one slot applies to all of them, so the pin generalises after the first
// RS-valid burst.
func (p *ScramblerPin) probeSubframe(macDibits []uint8, mode TrellisMode, interleave InterleaveMode, seed uint64, slotIndex int) (MACPDU, bool) {
	seq := p.keystream(seed)
	base := slotIndex * DibitsPerSubframe * 2 // slot base, in channel bits

	// Fast path: a pinned channel decodes at its known phase directly.
	if p.have {
		if pdu, ok := decodeAtPhase(macDibits, mode, interleave, seq, base+p.phase); ok {
			return pdu, true
		}
		// The pinned phase stopped validating (a re-tune, or a fluke lock) —
		// drop it and re-sweep.
		p.have = false
	}

	// Backoff: while unpinned, sweep the first burst immediately, then at most
	// once per probeSweepInterval bursts, so an undecodable channel can't
	// starve the IQ consumer with a full-sequence sweep on every PDU.
	p.sinceSweep++
	if p.sweptOnce && p.sinceSweep < probeSweepInterval {
		return MACPDU{}, false
	}
	p.sweptOnce = true
	p.sinceSweep = 0

	for c := 0; c < pn44SuperframeBits; c++ {
		if pdu, ok := decodeAtPhase(macDibits, mode, interleave, seq, base+c); ok {
			p.have = true
			p.phase = c
			return pdu, true
		}
	}
	return MACPDU{}, false
}
