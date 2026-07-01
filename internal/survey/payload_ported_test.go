package survey

import (
	"math"
	"testing"
)

// synthFSKPayload builds a 2-FSK baseband for the given bits: bit 1 → +dev,
// bit 0 → -dev. (Named to avoid colliding with other survey test helpers.)
func synthFSKPayload(bits []int, rate, baud, dev float64) []complex64 {
	sps := int(rate / baud)
	iq := make([]complex64, 0, len(bits)*sps)
	phase := 0.0
	for _, b := range bits {
		f := -dev
		if b == 1 {
			f = dev
		}
		step := 2 * math.Pi * f / rate
		for i := 0; i < sps; i++ {
			iq = append(iq, complex64(complex(math.Cos(phase), math.Sin(phase))))
			phase += step
		}
	}
	return iq
}

func TestPackBits(t *testing.T) {
	t.Parallel()
	got := packBits([]byte{1, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1})
	if len(got) != 2 || got[0] != 0x81 || got[1] != 0x55 {
		t.Fatalf("packBits = %x, want [81 55]", got)
	}
	if packBits([]byte{1, 0, 1}) != nil {
		t.Fatalf("partial byte should pack to nil")
	}
}

func TestSliceSymbolsTwoLevel(t *testing.T) {
	t.Parallel()
	bits := sliceSymbols([]float64{-1, 1, -1, 1, -1, 1}, 2)
	want := []byte{0, 1, 0, 1, 0, 1}
	if len(bits) != len(want) {
		t.Fatalf("len %d != %d", len(bits), len(want))
	}
	for i := range want {
		if bits[i] != want[i] {
			t.Fatalf("bit %d: got %d want %d", i, bits[i], want[i])
		}
	}
}

func TestRecoverPayloadAlternatingFSK(t *testing.T) {
	t.Parallel()
	// 64 alternating bits ⇒ after slicing, an alternating bitstream ⇒ 0x55/0xAA.
	var bits []int
	for i := 0; i < 64; i++ {
		bits = append(bits, i%2)
	}
	iq := synthFSKPayload(bits, 48000, 4800, 1200)
	got := RecoverPayload(iq, 48000, ClassFeatures{BaudHz: 4800, IFModality: 2})
	if len(got) == 0 {
		t.Fatalf("no payload recovered")
	}
	for _, b := range got {
		if b != 0x55 && b != 0xAA {
			t.Fatalf("alternating FSK should pack to 0x55/0xAA, got %x in %x", b, got)
		}
	}
}

func TestRecoverPayloadGuards(t *testing.T) {
	t.Parallel()
	if RecoverPayload(nil, 48000, ClassFeatures{BaudHz: 4800}) != nil {
		t.Fatalf("nil IQ should yield nil")
	}
	if RecoverPayload(make([]complex64, 100), 48000, ClassFeatures{BaudHz: 0}) != nil {
		t.Fatalf("zero baud should yield nil")
	}
}
