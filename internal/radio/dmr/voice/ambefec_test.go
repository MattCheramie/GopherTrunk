package voice

import (
	"bytes"
	"testing"
)

// mkInfo builds a deterministic, unique 49-bit AMBE payload (one bit
// per byte) from seed via a small LCG.
func mkInfo(seed int) []byte {
	f := make([]byte, ambeInfoBits)
	x := uint32(seed)*2654435761 + 1
	for i := range f {
		x = x*1664525 + 1013904223
		f[i] = byte(x >> 31)
	}
	return f
}

// TestGolaySyndromeMatchesMbelib spot-checks the init-generated
// syndrome table against entries of mbelib's precomputed
// golayMatrix[2048] (ecc_const.h) — confirming the Golay(23,12)
// decode is bit-identical to the reference.
func TestGolaySyndromeMatchesMbelib(t *testing.T) {
	want := map[int]uint16{
		0: 0, 15: 72, 23: 2084, 27: 769, 29: 1024, 30: 144,
		31: 2, 39: 72, 43: 72, 51: 16, 53: 1, 54: 1538, 58: 2048,
	}
	for idx, w := range want {
		if got := golaySyndrome[idx]; got != w {
			t.Errorf("golaySyndrome[%d] = %d, want %d (mbelib golayMatrix)", idx, got, w)
		}
	}
}

func TestGolay2312CorrectsErrors(t *testing.T) {
	for data := uint16(0); data < 4096; data += 137 {
		cw := golayEncode2312(data)
		if d, e := golayDecode2312(cw); d != data || e != 0 {
			t.Fatalf("clean data=%d: got (%d, %d)", data, d, e)
		}
		for _, flips := range [][]int{{3}, {0, 22}, {1, 11, 17}} {
			bad := cw
			for _, b := range flips {
				bad ^= 1 << uint(b)
			}
			if d, e := golayDecode2312(bad); d != data || e < 0 || e > 3 {
				t.Errorf("data=%d flips=%v: got (%d, %d)", data, flips, d, e)
			}
		}
	}
}

func TestAMBEFrameRoundTrip(t *testing.T) {
	for seed := 0; seed < 64; seed++ {
		info := mkInfo(seed)
		frame, err := EncodeAMBEFrame(info)
		if err != nil {
			t.Fatalf("seed %d: encode: %v", seed, err)
		}
		if len(frame) != ambeOnAirBits {
			t.Fatalf("seed %d: on-air frame is %d bits, want %d", seed, len(frame), ambeOnAirBits)
		}
		got, errs, err := DecodeAMBEFrame(frame)
		if err != nil {
			t.Fatalf("seed %d: decode: %v", seed, err)
		}
		if errs != 0 {
			t.Errorf("seed %d: clean frame reported %d corrected errors", seed, errs)
		}
		if !bytes.Equal(got, info) {
			t.Errorf("seed %d: round trip mismatch\n got %v\nwant %v", seed, got, info)
		}
	}
}

func TestAMBEFrameCorrectsErrors(t *testing.T) {
	// Group the 72 on-air frame positions by the C-vector they
	// deinterleave into, so errors can be aimed at C0 / C1 (the only
	// FEC-protected sub-vectors).
	var c0pos, c1pos []int
	for i := 0; i < 36; i++ {
		switch rW[i] {
		case 0:
			c0pos = append(c0pos, 2*i)
		case 1:
			c1pos = append(c1pos, 2*i)
		}
		switch rY[i] {
		case 0:
			c0pos = append(c0pos, 2*i+1)
		case 1:
			c1pos = append(c1pos, 2*i+1)
		}
	}

	info := mkInfo(7)
	for _, tc := range []struct {
		name string
		pos  []int
	}{
		{"C0", c0pos[:3]},
		{"C1", c1pos[:3]},
	} {
		frame, err := EncodeAMBEFrame(info)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range tc.pos {
			frame[p] ^= 1
		}
		got, _, err := DecodeAMBEFrame(frame)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, info) {
			t.Errorf("%s: three bit errors in one sub-vector were not corrected", tc.name)
		}
	}
}

func TestDecodeAMBEFrameRejectsBadLength(t *testing.T) {
	if _, _, err := DecodeAMBEFrame(make([]byte, 71)); err == nil {
		t.Error("expected error for a 71-bit frame")
	}
	if _, err := EncodeAMBEFrame(make([]byte, 48)); err == nil {
		t.Error("expected error for a 48-bit payload")
	}
}
