package keystream

import (
	"bytes"
	"testing"
)

// synth builds a frame: ct = pt ^ ks (ks truncated/cycled to len(pt)).
func synth(label string, iv, ks, pt []byte) Frame {
	ct := make([]byte, len(pt))
	for i := range pt {
		ct[i] = pt[i] ^ ks[i%len(ks)]
	}
	return Frame{Label: label, IV: iv, CT: ct}
}

func TestFindReuseGroupsByIV(t *testing.T) {
	t.Parallel()
	ksA := []byte{0x9e, 0x21, 0x55, 0x7c, 0x03, 0xaa, 0x11, 0x42}
	ksB := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	frames := []Frame{
		synth("a1", []byte{1, 2, 3}, ksA, []byte("ATTACK AT DAWN")),
		synth("a2", []byte{1, 2, 3}, ksA, []byte("RETREAT BY SEA")),
		synth("b1", []byte{9, 9, 9}, ksB, []byte("UNRELATED FRAME")),
	}
	groups := FindReuse(frames)
	if len(groups) != 1 {
		t.Fatalf("want 1 reuse group, got %d", len(groups))
	}
	if len(groups[0].Frames) != 2 {
		t.Fatalf("want 2 frames in the reused group, got %d", len(groups[0].Frames))
	}
}

func TestRecoverWithKnownDecryptsGroup(t *testing.T) {
	t.Parallel()
	ks := []byte{0x9e, 0x21, 0x55, 0x7c, 0x03, 0xaa, 0x11, 0x42, 0xfe, 0x10, 0x77, 0x33, 0x88, 0x29}
	pt1 := []byte("ATTACK AT DAWN")
	pt2 := []byte("RETREAT BY SEA")
	iv := []byte{1, 2, 3}
	g := ReuseGroup{IV: iv, Frames: []Frame{
		synth("a1", iv, ks, pt1),
		synth("a2", iv, ks, pt2),
	}}
	recoveredKS, decoded, ok := RecoverWithKnown(g, "a1", pt1)
	if !ok {
		t.Fatal("known frame not found")
	}
	if !bytes.Equal(recoveredKS, ks[:len(pt1)]) {
		t.Fatalf("recovered keystream wrong:\n got %x\nwant %x", recoveredKS, ks[:len(pt1)])
	}
	want := map[string][]byte{"a1": pt1, "a2": pt2}
	for _, d := range decoded {
		if !bytes.Equal(d.Plaintext, want[d.Label]) {
			t.Fatalf("frame %s decrypted to %q, want %q", d.Label, d.Plaintext, want[d.Label])
		}
	}
}

func TestCribDragGroupFindsKnownWord(t *testing.T) {
	t.Parallel()
	ks := bytes.Repeat([]byte{0x5a, 0xc3, 0x07, 0x91, 0x2e, 0x6b}, 16)
	iv := []byte{0x10, 0x20}
	g := ReuseGroup{IV: iv, Frames: []Frame{
		synth("m1", iv, ks, []byte("the dispatch unit is on the way now")),
		synth("m2", iv, ks, []byte("please send the bridge to the depot")),
	}}
	hits := CribDragGroup(g, []byte("the "), 20)
	if len(hits) == 0 {
		t.Fatal("crib-drag found nothing")
	}
	// The top-scoring hit should reveal real English in the partner frame.
	top := hits[0]
	if len(top.Revealed) == 0 {
		t.Fatal("empty revealed fragment")
	}
}
