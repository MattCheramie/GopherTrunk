package voice

import (
	"bytes"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// lcFragments encodes a Full Link Control into the four 32-bit embedded
// fragments that bursts B–E of a superframe carry.
func lcFragments(t *testing.T, f dmr.FLC) [4][]byte {
	t.Helper()
	info := dmr.AssembleFLC(f)
	lc := make([]byte, framing.EmbLCBits)
	for i := 0; i < framing.EmbLCBits; i++ {
		lc[i] = (info[i/8] >> uint(7-(i%8))) & 1
	}
	ch := framing.EncodeEmbeddedLC(lc)
	var frags [4][]byte
	for i := range frags {
		frags[i] = ch[i*framing.EmbeddedFragmentBits : (i+1)*framing.EmbeddedFragmentBits]
	}
	return frags
}

// embeddedSync packs an EMB header + 32-bit fragment into a burst's
// 24-dibit centre field.
func embeddedSync(emb dmr.EMB, frag []byte) [24]uint8 {
	field := dmr.AssembleEmbeddedField(emb, frag)
	var d [24]uint8
	for i := 0; i < 24; i++ {
		d[i] = field[2*i]<<1 | field[2*i+1]
	}
	return d
}

// burstLCSS gives the LCSS a voice burst carries when a Full LC is
// spread across bursts B–E: B=First, C/D=Continuation, E=Last.
func burstLCSS(b int) dmr.LCSS {
	switch b {
	case 1:
		return dmr.LCSSFirst
	case 4:
		return dmr.LCSSLast
	default:
		return dmr.LCSSCont
	}
}

// interleaveTail is trailing filler appended to a synthetic interleaved
// stream so both slots' burst-A anchors have a full largest-cadence
// superframe span buffered behind them (the auto-detecting decoder waits
// for that before slicing). A live carrier is continuous, so on air this
// trailing data is always present; the fixtures must mimic it.
const interleaveTail = 2 * (dmr.BurstDibits + cachDibits) // 288

// buildInterleavedStreamWithLC is buildInterleavedStream extended so each
// slot's bursts B–E carry that slot's own embedded Link Control, so a
// decoder can recover and label each timeslot's talkgroup. cach dibits of
// filler are inserted before each woven burst, modelling the outbound CACH
// (cach=0 reproduces a CACH-free 264-dibit-cadence stream).
func buildInterleavedStreamWithLC(t *testing.T, aLC, bLC dmr.FLC, cach int) (stream []uint8, aFrames, bFrames [][]byte) {
	t.Helper()
	const bSeedOffset = 10000
	aFrags := lcFragments(t, aLC)
	bFrags := lcFragments(t, bLC)
	for f := 0; f < FramesPerSuperframe; f++ {
		aFrames = append(aFrames, mkFrame(f))
		bFrames = append(bFrames, mkFrame(f+bSeedOffset))
	}
	cachFiller := make([]uint8, cach)
	for b := 0; b < BurstsPerSuperframe; b++ {
		aSync, bSync := dmr.BSData.Dibits, dmr.BSData.Dibits
		switch {
		case b == 0:
			aSync, bSync = dmr.BSVoice.Dibits, dmr.BSVoice.Dibits
		case b >= 1 && b <= 4:
			aSync = embeddedSync(dmr.EMB{ColorCode: 1, LCSS: burstLCSS(b)}, aFrags[b-1])
			bSync = embeddedSync(dmr.EMB{ColorCode: 1, LCSS: burstLCSS(b)}, bFrags[b-1])
		}
		base := b * FramesPerBurst
		stream = append(stream, cachFiller...)
		stream = append(stream, makeVoiceBurst(aFrames[base:base+FramesPerBurst], aSync)...)
		stream = append(stream, cachFiller...)
		stream = append(stream, makeVoiceBurst(bFrames[base:base+FramesPerBurst], bSync)...)
	}
	stream = append(stream, make([]uint8, interleaveTail)...)
	return stream, aFrames, bFrames
}

// buildStream assembles a dibit stream of n voice superframes preceded
// by lead dibits of filler. Frame f of the stream carries mkFrame(f),
// so the caller can verify ordering across the whole stream. Burst A
// of each superframe is framed by the BS-Voice sync; bursts B–F use a
// non-voice sync (BS-Data) standing in for embedded signalling so the
// voice-only SyncDetector ignores them.
func buildStream(t *testing.T, n, lead int) (stream []uint8, frames [][]byte) {
	t.Helper()
	stream = make([]uint8, lead)
	total := n * FramesPerSuperframe
	for f := 0; f < total; f++ {
		frames = append(frames, mkFrame(f))
	}
	for s := 0; s < n; s++ {
		for b := 0; b < BurstsPerSuperframe; b++ {
			sync := dmr.BSData.Dibits
			if b == 0 {
				sync = dmr.BSVoice.Dibits
			}
			base := s*FramesPerSuperframe + b*FramesPerBurst
			stream = append(stream, makeVoiceBurst(frames[base:base+FramesPerBurst], sync)...)
		}
	}
	return stream, frames
}

// buildInterleavedStream assembles a real 2-slot TDMA dibit stream: for
// each of n superframes the six bursts of slot A and slot B are woven
// burst-by-burst (A.b0, B.b0, A.b1, B.b1, … A.b5, B.b5), so each slot's
// own bursts are 264 dibits (two physical bursts) apart. Burst A of each
// slot's superframe carries BS-Voice sync; the rest carry BS-Data
// (standing in for embedded signalling). Slot B's frames seed from a
// large offset so the two slots' AMBE payloads are unmistakably
// distinct. lead prepends filler dibits to exercise non-aligned starts.
func buildInterleavedStream(t *testing.T, n, lead int) (stream []uint8, aFrames, bFrames [][]byte) {
	t.Helper()
	const bSeedOffset = 10000
	stream = make([]uint8, lead)
	total := n * FramesPerSuperframe
	for f := 0; f < total; f++ {
		aFrames = append(aFrames, mkFrame(f))
		bFrames = append(bFrames, mkFrame(f+bSeedOffset))
	}
	for s := 0; s < n; s++ {
		for b := 0; b < BurstsPerSuperframe; b++ {
			sync := dmr.BSData.Dibits
			if b == 0 {
				sync = dmr.BSVoice.Dibits
			}
			base := s*FramesPerSuperframe + b*FramesPerBurst
			stream = append(stream, makeVoiceBurst(aFrames[base:base+FramesPerBurst], sync)...)
			stream = append(stream, makeVoiceBurst(bFrames[base:base+FramesPerBurst], sync)...)
		}
	}
	stream = append(stream, make([]uint8, interleaveTail)...)
	return stream, aFrames, bFrames
}

func checkFrames(t *testing.T, sf VoiceSuperframe, frames [][]byte, firstFrame int) {
	t.Helper()
	for i := 0; i < FramesPerSuperframe; i++ {
		if !bytes.Equal(sf.Frames[i], frames[firstFrame+i]) {
			t.Errorf("frame %d:\n got %v\nwant %v", i, sf.Frames[i], frames[firstFrame+i])
		}
	}
}

func TestDecoderExtractsSuperframe(t *testing.T) {
	stream, frames := buildStream(t, 1, 0)

	d := NewDecoder()
	got := d.Process(stream, 0)

	if len(got) != 1 {
		t.Fatalf("got %d superframes, want 1", len(got))
	}
	if got[0].SyncName != "BS-Voice" {
		t.Errorf("SyncName = %q, want BS-Voice", got[0].SyncName)
	}
	if got[0].StartDibit != 0 {
		t.Errorf("StartDibit = %d, want 0", got[0].StartDibit)
	}
	checkFrames(t, got[0], frames, 0)
}

func TestDecoderTwoSuperframes(t *testing.T) {
	stream, frames := buildStream(t, 2, 0)

	d := NewDecoder()
	got := d.Process(stream, 0)

	if len(got) != 2 {
		t.Fatalf("got %d superframes, want 2", len(got))
	}
	checkFrames(t, got[0], frames, 0)
	checkFrames(t, got[1], frames, FramesPerSuperframe)
	if got[1].StartDibit != superframeDibits {
		t.Errorf("second StartDibit = %d, want %d", got[1].StartDibit, superframeDibits)
	}
}

func TestDecoderLeadingFiller(t *testing.T) {
	const lead = 137
	stream, frames := buildStream(t, 1, lead)

	d := NewDecoder()
	got := d.Process(stream, 0)

	if len(got) != 1 {
		t.Fatalf("got %d superframes, want 1", len(got))
	}
	if got[0].StartDibit != lead {
		t.Errorf("StartDibit = %d, want %d", got[0].StartDibit, lead)
	}
	checkFrames(t, got[0], frames, 0)
}

func TestDecoderChunkedInput(t *testing.T) {
	stream, frames := buildStream(t, 1, 0)

	d := NewDecoder()
	var got []VoiceSuperframe
	const chunk = 100
	for off := 0; off < len(stream); off += chunk {
		end := off + chunk
		if end > len(stream) {
			end = len(stream)
		}
		got = append(got, d.Process(stream[off:end], off)...)
	}

	if len(got) != 1 {
		t.Fatalf("got %d superframes across chunks, want 1", len(got))
	}
	checkFrames(t, got[0], frames, 0)
}

func TestDecoderIncompleteSuperframe(t *testing.T) {
	stream, _ := buildStream(t, 1, 0)
	// Truncate to burst A plus two bursts — not a full superframe.
	truncated := stream[:3*dmr.BurstDibits]

	d := NewDecoder()
	if got := d.Process(truncated, 0); len(got) != 0 {
		t.Fatalf("got %d superframes from a partial superframe, want 0", len(got))
	}
}

func TestDecoderResetClearsState(t *testing.T) {
	stream, _ := buildStream(t, 1, 0)

	d := NewDecoder()
	// Feed burst A only, then reset before the rest arrives.
	d.Process(stream[:dmr.BurstDibits], 0)
	d.Reset()

	// A fresh full stream after reset must still decode cleanly.
	fresh, frames := buildStream(t, 1, 0)
	got := d.Process(fresh, 0)
	if len(got) != 1 {
		t.Fatalf("got %d superframes after reset, want 1", len(got))
	}
	checkFrames(t, got[0], frames, 0)
}

// TestInterleavedDecoderSeparatesSlots is the core 2-slot guard: fed a
// carrier whose two timeslots interleave burst-by-burst, the interleaved
// Decoder must emit one clean superframe per slot — each slice gathering
// only its own slot's bursts (264-dibit stride) and skipping the other
// slot's interleaved burst — tagged with distinct phases.
func TestInterleavedDecoderSeparatesSlots(t *testing.T) {
	stream, aFrames, bFrames := buildInterleavedStream(t, 1, 0)

	d := NewInterleavedDecoder()
	got := d.Process(stream, 0)

	if len(got) != 2 {
		t.Fatalf("got %d superframes, want 2 (one per timeslot)", len(got))
	}
	// Stream order: slot A's burst A (StartDibit 0) is detected before
	// slot B's (StartDibit 132, one physical burst later).
	a, b := got[0], got[1]
	if a.StartDibit != 0 || b.StartDibit != dmr.BurstDibits {
		t.Fatalf("StartDibits = %d, %d; want 0, %d", a.StartDibit, b.StartDibit, dmr.BurstDibits)
	}
	if a.Phase == b.Phase {
		t.Errorf("both slots reported Phase %d; want distinct phases", a.Phase)
	}
	// Each superframe must contain ONLY its own slot's AMBE frames — the
	// whole point of the stride. A single-slot (stride 1) decoder would
	// instead interleave A and B frames here and fail this check.
	checkFrames(t, a, aFrames, 0)
	checkFrames(t, b, bFrames, 0)
}

// TestLegacyDecoderMixesInterleavedSlots documents the bug the
// interleaved decoder fixes: a stride-1 Decoder run over a 2-slot stream
// locks burst A correctly but then grabs the WRONG (other-slot) bursts
// for B–F, so its output does not match either pure slot.
func TestLegacyDecoderMixesInterleavedSlots(t *testing.T) {
	stream, aFrames, bFrames := buildInterleavedStream(t, 1, 0)

	d := NewDecoder() // stride 1 — wrong cadence for an interleaved carrier
	got := d.Process(stream, 0)
	if len(got) == 0 {
		t.Fatal("expected at least one (mis-assembled) superframe")
	}
	// got[0] anchors on slot A's burst A but slices contiguous bursts,
	// so its frames are an A/B interleave — equal to neither pure slot.
	mixed := got[0]
	aClean, bClean := true, true
	for i := 0; i < FramesPerSuperframe; i++ {
		if !bytes.Equal(mixed.Frames[i], aFrames[i]) {
			aClean = false
		}
		if !bytes.Equal(mixed.Frames[i], bFrames[i]) {
			bClean = false
		}
	}
	if aClean || bClean {
		t.Errorf("stride-1 decode unexpectedly matched a pure slot (aClean=%v bClean=%v); the interleaving guard is not exercised", aClean, bClean)
	}
}

// TestInterleavedDecoderPhaseStable confirms that across several
// superframes — and from a non-burst-aligned start offset — every
// superframe of a given slot carries the same Phase, so a downstream
// consumer can group a call's superframes by Phase for the call's
// lifetime.
func TestInterleavedDecoderPhaseStable(t *testing.T) {
	const n, lead = 3, 17
	stream, aFrames, bFrames := buildInterleavedStream(t, n, lead)

	d := NewInterleavedDecoder()
	got := d.Process(stream, 0)
	if len(got) != 2*n {
		t.Fatalf("got %d superframes, want %d (%d per slot)", len(got), 2*n, n)
	}

	// Partition by phase; every superframe sharing slot A's phase must
	// carry slot A's frames, and likewise for B.
	aPhase := got[0].Phase
	var ai, bi int
	for _, sf := range got {
		if sf.Phase == aPhase {
			checkFrames(t, sf, aFrames, ai*FramesPerSuperframe)
			ai++
		} else {
			checkFrames(t, sf, bFrames, bi*FramesPerSuperframe)
			bi++
		}
	}
	if ai != n || bi != n {
		t.Errorf("phase partition = %d/%d superframes, want %d/%d", ai, bi, n, n)
	}
}

// TestInterleavedDecoderChunkedInput feeds the 2-slot stream in small
// chunks to exercise the cross-Process buffering of two simultaneously
// in-flight superframe anchors.
func TestInterleavedDecoderChunkedInput(t *testing.T) {
	stream, aFrames, bFrames := buildInterleavedStream(t, 1, 0)

	d := NewInterleavedDecoder()
	var got []VoiceSuperframe
	const chunk = 137
	for off := 0; off < len(stream); off += chunk {
		end := off + chunk
		if end > len(stream) {
			end = len(stream)
		}
		got = append(got, d.Process(stream[off:end], off)...)
	}
	if len(got) != 2 {
		t.Fatalf("got %d superframes across chunks, want 2", len(got))
	}
	checkFrames(t, got[0], aFrames, 0)
	checkFrames(t, got[1], bFrames, 0)
}

// TestInterleavedDecoderLabelsSlotsByEmbeddedLC is the Phase 4 headline:
// two concurrent calls interleaved on a carrier, each carrying its OWN
// talkgroup's embedded Link Control in bursts B–E. The interleaved
// decoder must hand back one superframe per slot whose recovered LC
// names that slot's talkgroup — the label the identical BS-sourced
// burst-A sync can't provide. This is what lets a consumer bind a
// phase to a grant's talkgroup.
func TestInterleavedDecoderLabelsSlotsByEmbeddedLC(t *testing.T) {
	aLC := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 1001, SrcAddr: 55}
	bLC := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 2002, SrcAddr: 66}
	stream, aFrames, bFrames := buildInterleavedStreamWithLC(t, aLC, bLC, 0)

	d := NewInterleavedDecoder()
	got := d.Process(stream, 0)
	if len(got) != 2 {
		t.Fatalf("got %d superframes, want 2", len(got))
	}
	a, b := got[0], got[1]

	// Each slot's superframe carries its own talkgroup + source.
	if !a.HasLC || a.LC.DstAddr != 1001 || a.LC.SrcAddr != 55 {
		t.Errorf("slot A LC: hasLC=%v dst=%d src=%d, want dst=1001 src=55", a.HasLC, a.LC.DstAddr, a.LC.SrcAddr)
	}
	if !b.HasLC || b.LC.DstAddr != 2002 || b.LC.SrcAddr != 66 {
		t.Errorf("slot B LC: hasLC=%v dst=%d src=%d, want dst=2002 src=66", b.HasLC, b.LC.DstAddr, b.LC.SrcAddr)
	}
	// Phases distinct, and audio still cleanly separated per slot.
	if a.Phase == b.Phase {
		t.Errorf("slots share Phase %d; want distinct", a.Phase)
	}
	checkFrames(t, a, aFrames, 0)
	checkFrames(t, b, bFrames, 0)
}

// TestInterleavedDecoderAutoDetectsCACHCadence is the #644 guard: on a real
// outbound DMR carrier a 12-dibit CACH precedes each burst, so a call's
// same-slot bursts are 288 dibits apart, not 264. Fed such a stream, the
// interleaved decoder must auto-detect the 288 cadence (the one whose bursts
// B–E reassemble a CRC-valid embedded LC), lock it, and still separate the
// two slots cleanly. Slicing at the wrong 264 cadence is exactly what
// produced the garbled, encrypted-sounding audio reported in #644.
func TestInterleavedDecoderAutoDetectsCACHCadence(t *testing.T) {
	aLC := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 1001, SrcAddr: 55}
	bLC := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 2002, SrcAddr: 66}
	stream, aFrames, bFrames := buildInterleavedStreamWithLC(t, aLC, bLC, cachDibits)

	d := NewInterleavedDecoder()
	got := d.Process(stream, 0)
	if len(got) != 2 {
		t.Fatalf("got %d superframes, want 2 (one per timeslot)", len(got))
	}
	if d.lockedStep != 2*(dmr.BurstDibits+cachDibits) {
		t.Errorf("lockedStep = %d, want %d (288, the CACH cadence)", d.lockedStep, 2*(dmr.BurstDibits+cachDibits))
	}
	a, b := got[0], got[1]
	if !a.HasLC || a.LC.DstAddr != 1001 || a.LC.SrcAddr != 55 {
		t.Errorf("slot A LC: hasLC=%v dst=%d src=%d, want dst=1001 src=55", a.HasLC, a.LC.DstAddr, a.LC.SrcAddr)
	}
	if !b.HasLC || b.LC.DstAddr != 2002 || b.LC.SrcAddr != 66 {
		t.Errorf("slot B LC: hasLC=%v dst=%d src=%d, want dst=2002 src=66", b.HasLC, b.LC.DstAddr, b.LC.SrcAddr)
	}
	if a.Phase == b.Phase {
		t.Errorf("slots share Phase %d; want distinct", a.Phase)
	}
	// Audio cleanly separated per slot despite the inter-burst CACH.
	checkFrames(t, a, aFrames, 0)
	checkFrames(t, b, bFrames, 0)
}

// TestInterleavedDecoderAutoDetectsNoCACHCadence is the symmetric case: a
// CACH-free carrier (e.g. a hotspot / replayed stream) has its same-slot
// bursts 264 dibits apart, and the decoder must lock that cadence instead.
func TestInterleavedDecoderAutoDetectsNoCACHCadence(t *testing.T) {
	aLC := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 1001, SrcAddr: 55}
	bLC := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 2002, SrcAddr: 66}
	stream, aFrames, bFrames := buildInterleavedStreamWithLC(t, aLC, bLC, 0)

	d := NewInterleavedDecoder()
	got := d.Process(stream, 0)
	if len(got) != 2 {
		t.Fatalf("got %d superframes, want 2", len(got))
	}
	if d.lockedStep != 2*dmr.BurstDibits {
		t.Errorf("lockedStep = %d, want %d (264, the CACH-free cadence)", d.lockedStep, 2*dmr.BurstDibits)
	}
	checkFrames(t, got[0], aFrames, 0)
	checkFrames(t, got[1], bFrames, 0)
}

// TestDecoderSuperframeNoLCWhenAbsent confirms a superframe whose B–E
// bursts carry no valid embedded LC (the plain single-slot synth) leaves
// HasLC false rather than surfacing a garbage talkgroup.
func TestDecoderSuperframeNoLCWhenAbsent(t *testing.T) {
	stream, _ := buildStream(t, 1, 0) // B–F use BS-Data filler, not an LC
	got := NewDecoder().Process(stream, 0)
	if len(got) != 1 {
		t.Fatalf("got %d superframes, want 1", len(got))
	}
	if got[0].HasLC {
		t.Errorf("HasLC = true on a stream with no embedded LC; want false")
	}
}
