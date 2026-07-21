package phase2

// SuperframeDecoder assembles structured P25 Phase 2 superframes from a
// raw dibit stream. It is the TDMA-layer counterpart of the flat
// ControlChannel.Process adapter (process.go): where Process slices a
// MAC PDU window on every outbound-sync match, the SuperframeDecoder
// locks onto the 360 ms superframe boundary via that same 20-dibit
// outbound sync and then slices all 12 sub-frames at the fixed
// DibitsPerSubframe (180-dibit) TDMA cadence.
//
// Sub-frames alternate between the two TDMA timeslots — even Index →
// timeslot 0, odd Index → timeslot 1 — so a caller routing voice or
// MAC sub-frames can demultiplex the two logical channels by Index
// parity without any further symbol-level work.
//
// A SuperframeDecoder is stateful and not safe for concurrent use;
// construct one per traffic-channel decode chain. Buffering spans
// Process calls so a superframe straddling IQ-chunk boundaries is
// still assembled.

// Subframe is one of the 12 sub-frames of a Superframe.
type Subframe struct {
	// Index is the sub-frame position 0..11 within the superframe.
	Index int
	// Timeslot is the TDMA timeslot the sub-frame belongs to, 0 or 1,
	// derived from Index parity.
	Timeslot int
	// SlotType names what the sub-frame carries (voice vs MAC). It is
	// SlotTypeUnknown until the ISCH decode in isch.go fills it.
	SlotType SlotType
	// Dibits holds the DibitsPerSubframe raw dibits of the sub-frame.
	Dibits []uint8
}

// Superframe is one decoded 360 ms P25 Phase 2 superframe.
type Superframe struct {
	// StartDibit is the absolute dibit index of sub-frame 0's first
	// dibit in the source stream.
	StartDibit int
	// Subframes holds the 12 sub-frames in transmission order.
	Subframes [SubframesPerSuperframe]Subframe
}

// syncLookback maps a sync match — reported by SyncDetector as the
// absolute index of the sync word's last dibit — back to the first
// dibit of the sub-frame that carries it.
const syncLookback = SyncDibits - 1

// superframeBufKeep retains enough dibits that a pending anchor near
// the tail of the buffer can still slice its full superframe once the
// trailing sub-frames arrive. Dibits are one byte each, so the cost is
// trivial.
const superframeBufKeep = 2 * DibitsPerSuperframe

// syncAnchor is a candidate superframe start: the absolute index the
// SyncDetector reported, plus the constant dibit rotation under which the
// sync matched. Real air H-DQPSK is differentially decoded, so a residual
// carrier offset near an odd multiple of ±1500 Hz (a quarter of the 6000-baud
// symbol rate) rotates every dibit in the stream by a constant 0..3. The
// SuperframeDecoder detects the outbound sync under all four rotations and
// de-rotates the sliced superframe back to the canonical convention, so the
// ISCH + MAC FEC downstream always sees canonical dibits regardless of the
// receiver's residual carrier phase (issue #915).
type syncAnchor struct {
	idx int
	rot uint8
}

// SuperframeDecoder is the stateful dibit → Superframe assembler.
type SuperframeDecoder struct {
	dets     [4]*SyncDetector // one per constant dibit rotation (issue #915)
	buf      []uint8
	bufStart int          // absolute dibit index of buf[0]
	pending  []syncAnchor // sync anchors awaiting a full superframe
}

// NewSuperframeDecoder returns a SuperframeDecoder ready to consume
// dibits.
func NewSuperframeDecoder() *SuperframeDecoder {
	d := &SuperframeDecoder{}
	d.initDetectors()
	return d
}

// initDetectors builds the four rotation-shifted sync detectors. Detector r
// matches when the incoming stream is rotated by +r from the canonical sync;
// a match therefore means every dibit is canonical+r, so the superframe is
// de-rotated by -r (add 4-r) to recover canonical dibits.
//
// The canonical rotation (r=0) keeps the historical tolerance of 2 dibit
// mismatches. The three added rotations use a stricter tolerance of 1 so that
// searching four patterns instead of one does not multiply the false-lock rate
// on noise (a spurious 18/20 match under any of the extra rotations would
// otherwise let a non-signal masquerade as a locked Phase 2 superframe — see
// hunt.TestSurveyDeepRoutesAnalogToSiglab). A real rotated on-air stream carries
// a clean sync (≥19/20), so tolerance 1 still catches it.
func (d *SuperframeDecoder) initDetectors() {
	base := OutboundSyncDibits()
	for r := 0; r < 4; r++ {
		pat := make([]uint8, len(base))
		for i, p := range base {
			pat[i] = (p + uint8(r)) & 3
		}
		tolerance := 2
		if r != 0 {
			tolerance = 1
		}
		d.dets[r] = NewSyncDetector(pat, tolerance)
	}
}

// Reset clears all buffered state. Call on a stream re-sync so a stale
// anchor does not slice across the discontinuity.
func (d *SuperframeDecoder) Reset() {
	d.buf = d.buf[:0]
	d.bufStart = 0
	d.pending = d.pending[:0]
	d.initDetectors()
}

// Process consumes a window of dibits and returns every superframe that
// completed within it. baseIdx is the absolute dibit index of
// dibits[0]; it must be monotonically non-decreasing across calls.
// Superframes are returned in stream order.
func (d *SuperframeDecoder) Process(dibits []uint8, baseIdx int) []Superframe {
	if len(d.buf) == 0 {
		d.bufStart = baseIdx
	}
	d.buf = append(d.buf, dibits...)

	// Detect the outbound sync under each of the four constant dibit
	// rotations; only the rotation that matches the stream's residual-carrier
	// phase fires (a wrong rotation shifts most of the 20 sync dibits, far
	// exceeding the tolerance). Merge the fresh anchors in stream order.
	var fresh []syncAnchor
	for r := uint8(0); r < 4; r++ {
		matches, _ := d.dets[r].Process(nil, dibits, baseIdx)
		for _, m := range matches {
			fresh = append(fresh, syncAnchor{idx: m, rot: r})
		}
	}
	sortAnchors(fresh)
	d.pending = append(d.pending, fresh...)

	var out []Superframe
	bufEnd := d.bufStart + len(d.buf)
	keep := d.pending[:0]
	for _, a := range d.pending {
		start := a.idx - syncLookback - SyncSubframeIndex*DibitsPerSubframe
		if start+DibitsPerSuperframe > bufEnd {
			keep = append(keep, a) // trailing sub-frames not buffered yet
			continue
		}
		if start < d.bufStart {
			continue // anchor fell off the front of the buffer
		}
		out = append(out, d.sliceSuperframe(start, a.rot))
	}
	d.pending = keep
	d.trim()
	return out
}

// sortAnchors orders anchors by their absolute index (insertion sort — the
// per-call anchor count is tiny: sync is ~one match per 2160-dibit superframe).
func sortAnchors(a []syncAnchor) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1].idx > a[j].idx; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}

// trim bounds the cross-call buffer. A pending anchor whose data is
// dropped is discarded by the start < bufStart guard in Process.
func (d *SuperframeDecoder) trim() {
	if len(d.buf) <= superframeBufKeep {
		return
	}
	drop := len(d.buf) - superframeBufKeep
	copy(d.buf, d.buf[drop:])
	d.buf = d.buf[:superframeBufKeep]
	d.bufStart += drop
}

// sliceSuperframe cuts the 12 fixed-length sub-frames starting at
// absolute dibit index start. The caller has confirmed the full
// superframe span is buffered. rot is the constant dibit rotation under
// which the sync matched; each sub-frame is de-rotated by -rot (add 4-rot)
// to recover the canonical dibit convention before the ISCH + MAC FEC run.
// Each sub-frame's ISCH is decoded so the SlotType is populated; an
// uncorrectable ISCH leaves SlotTypeUnknown.
func (d *SuperframeDecoder) sliceSuperframe(start int, rot uint8) Superframe {
	sf := Superframe{StartDibit: start}
	off := start - d.bufStart
	derot := (4 - rot) & 3
	for i := 0; i < SubframesPerSuperframe; i++ {
		sub := make([]uint8, DibitsPerSubframe)
		copy(sub, d.buf[off+i*DibitsPerSubframe:off+(i+1)*DibitsPerSubframe])
		if derot != 0 {
			for j := range sub {
				sub[j] = (sub[j] + derot) & 3
			}
		}
		slot := SlotTypeUnknown
		if isch, _, err := DecodeISCH(sub[ISCHOffset : ISCHOffset+ISCHDibits]); err == nil {
			slot = isch.SlotType
		}
		sf.Subframes[i] = Subframe{
			Index:    i,
			Timeslot: i & 1,
			SlotType: slot,
			Dibits:   sub,
		}
	}
	return sf
}
