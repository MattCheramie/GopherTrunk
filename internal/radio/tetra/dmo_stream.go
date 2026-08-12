package tetra

import "sort"

// DMO (Direct Mode) streaming burst extraction. ExtractDMBurstsSoft (dmo.go) is
// stateless over a COMPLETE dibit slice — perfect for the offline replay harness,
// which accumulates the whole capture before slicing. The live daemon instead sees
// the demodulated dibit/soft-differential stream arrive in chunks and must decode
// as it goes without buffering the whole (unbounded) transmission. DMStreamExtractor
// is that streaming adapter: it keeps a BOUNDED sliding window of the most recent
// dibits + differentials, re-runs ExtractDMBurstsSoft over the window each time new
// symbols arrive, and emits each newly-completed DSB/DNB exactly once (deduped by its
// absolute training-sequence lead index) in time order.
//
// It is the DMO analog of TrafficExtractor's streaming role for TMO, and is shared by
// both the control-side pipeline (newTETRADMOPipeline: DSB SCH/S → lock/colour) and the
// voice chain (DNB TCH/S → speech). Because it delegates the framing to the same
// ExtractDMBurstsSoft the offline path uses, the two stay bit-identical on the bursts
// they both see.

const (
	// dmStreamKeepTail is how many trailing dibits the window retains after each
	// extraction pass. It must exceed the widest burst span so a burst straddling the
	// trim boundary is not lost: a DNB reaches from L+dmDNBBKN1Start (L-108) to
	// L+dmDNBBKN2Start+dmBlockDibits (L+119), a DSB from L+dmDSBFreqCorrStart (L-100)
	// to L+dmDSBBKN2Start+dmBlockDibits (L+127) — ~235 dibits — plus the 19-dibit sync
	// correlation lead. 384 gives comfortable margin so a burst whose lead sits just
	// inside the retained tail is still fully present (and re-scanned) next pass.
	dmStreamKeepTail = 384
	// dmStreamMinRun is the smallest window (in dibits) worth running an extraction
	// pass over — a little more than dmStreamKeepTail so the first pass has at least
	// one burst-width of fresh symbols beyond the retained tail.
	dmStreamMinRun = dmStreamKeepTail + 256
)

// DMStreamExtractor turns a chunked DMO dibit/soft-differential stream into a
// time-ordered sequence of DMBursts via a bounded sliding window. Not safe for
// concurrent use; drive it from a single goroutine (the receiver's sink callback,
// like TrafficExtractor).
type DMStreamExtractor struct {
	onBurst func(DMBurst)

	dibits  []uint8
	diffs   []complex64
	baseIdx int  // absolute dibit index of dibits[0]
	nextIdx int  // absolute index the next appended dibit should carry (baseIdx+len)
	primed  bool // nextIdx is meaningful once at least one chunk has been appended
	softOK  bool // the diffs buffer is strictly parallel to dibits (no hard-chunk gap)

	// lastLead is the highest absolute training-sequence lead already emitted, so a
	// burst re-seen in a later window (its lead still inside the retained tail) is not
	// emitted twice. Absolute, so it survives window trimming.
	lastLead int
}

// NewDMStreamExtractor builds a streaming DMO extractor that calls onBurst for each
// newly-completed DSB/DNB, in ascending lead order. The burst's slices are freshly
// allocated by ExtractDMBursts, so onBurst may retain them, but treat them as
// read-only.
func NewDMStreamExtractor(onBurst func(DMBurst)) *DMStreamExtractor {
	return &DMStreamExtractor{onBurst: onBurst, lastLead: -1, softOK: true}
}

// Process appends one chunk of dibits and its parallel per-symbol differentials
// (diffs, the receiver's SoftSink output — same length as dibits, or nil for a
// hard-only stream), then extracts and emits any newly-completed bursts. base is the
// absolute dibit index of dibits[0]; a base that is not contiguous with the previous
// chunk (a receiver Reset restarts the index at 0) transparently restarts the window.
func (e *DMStreamExtractor) Process(dibits []uint8, diffs []complex64, base int) {
	if len(dibits) == 0 {
		return
	}
	if len(diffs) != len(dibits) {
		diffs = nil // keep the two strictly parallel; degrade to hard on mismatch
	}
	// A non-contiguous base means the upstream receiver re-anchored (Reset): drop the
	// stale window and restart from the new base so leads stay absolute-consistent.
	if e.primed && base != e.nextIdx {
		e.reset(base)
	}
	if !e.primed {
		e.baseIdx = base
		e.nextIdx = base
		e.primed = true
	}

	e.dibits = append(e.dibits, dibits...)
	if diffs == nil {
		e.softOK = false
	}
	if e.softOK {
		e.diffs = append(e.diffs, diffs...)
	} else {
		e.diffs = e.diffs[:0]
	}
	e.nextIdx = base + len(dibits)

	if len(e.dibits) < dmStreamMinRun {
		return
	}
	e.extractAndEmit()
	e.trim()
}

// Flush runs a final extraction pass over whatever remains in the window (used at
// end-of-stream so the last buffered burst is not stranded below dmStreamMinRun). It
// does not trim, so it is safe to call repeatedly.
func (e *DMStreamExtractor) Flush() {
	if len(e.dibits) == 0 {
		return
	}
	e.extractAndEmit()
}

// Reset drops all buffered state, as if the extractor were freshly constructed. The
// lead-dedup watermark is cleared too, so a stream that restarts from index 0 (a
// receiver Reset) does not suppress its first bursts.
func (e *DMStreamExtractor) Reset() { e.reset(0) }

func (e *DMStreamExtractor) reset(base int) {
	e.dibits = e.dibits[:0]
	e.diffs = e.diffs[:0]
	e.baseIdx = base
	e.nextIdx = base
	e.primed = false
	e.softOK = true
	e.lastLead = -1
}

func (e *DMStreamExtractor) extractAndEmit() {
	var bursts []DMBurst
	if e.softOK && len(e.diffs) == len(e.dibits) {
		bursts = ExtractDMBurstsSoft(e.dibits, e.diffs, e.baseIdx)
	} else {
		bursts = ExtractDMBursts(e.dibits, e.baseIdx)
	}
	// Emit strictly in ascending lead order so downstream (voice frames especially)
	// stays time-ordered, and only bursts past the dedup watermark.
	sort.Slice(bursts, func(i, j int) bool { return bursts[i].Lead < bursts[j].Lead })
	for i := range bursts {
		if bursts[i].Lead <= e.lastLead {
			continue
		}
		e.lastLead = bursts[i].Lead
		if e.onBurst != nil {
			e.onBurst(bursts[i])
		}
	}
}

// trim discards all but the trailing dmStreamKeepTail dibits so the window stays
// bounded. lastLead is absolute and unaffected, so a burst re-seen in the retained
// tail after the trim is still suppressed by the dedup check.
func (e *DMStreamExtractor) trim() {
	over := len(e.dibits) - dmStreamKeepTail
	if over <= 0 {
		return
	}
	e.dibits = append(e.dibits[:0], e.dibits[over:]...)
	if e.softOK {
		e.diffs = append(e.diffs[:0], e.diffs[over:]...)
	}
	e.baseIdx += over
}
