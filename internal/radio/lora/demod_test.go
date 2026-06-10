package lora

import (
	"math"
	"math/rand"
	"testing"
)

// addNoise adds complex AWGN at the given SNR (dB) relative to unit-power
// signal samples, returning a fresh buffer.
func addNoise(in []complex64, snrDB float64, rng *rand.Rand) []complex64 {
	sigma := math.Sqrt(0.5 * math.Pow(10, -snrDB/10))
	out := make([]complex64, len(in))
	for i, s := range in {
		out[i] = s + complex(
			float32(rng.NormFloat64()*sigma),
			float32(rng.NormFloat64()*sigma),
		)
	}
	return out
}

// applyCFO multiplies in by a complex exponential of cfoBins bins of
// carrier offset, simulating a tuning error.
func applyCFO(in []complex64, cfoBins float64, sf, osf int) []complex64 {
	n := 1 << uint(sf)
	w := 2 * math.Pi * cfoBins / float64(n*osf)
	out := make([]complex64, len(in))
	for k, s := range in {
		ph := w * float64(k)
		out[k] = s * complex(float32(math.Cos(ph)), float32(math.Sin(ph)))
	}
	return out
}

// pad prepends/append silence so the frame does not start at sample 0.
func pad(in []complex64, lead, tail int) []complex64 {
	out := make([]complex64, lead+len(in)+tail)
	copy(out[lead:], in)
	return out
}

func TestModDemodClean(t *testing.T) {
	osf := DefaultOversample
	payloads := [][]byte{
		{0x2A},
		{0xDE, 0xAD, 0xBE, 0xEF},
		[]byte("GopherTrunk"),
	}
	for _, sf := range []int{7, 8, 9, 10, 11, 12} {
		for cr := 1; cr <= 4; cr++ {
			for _, pl := range payloads {
				iq := LoRaModulate(pl, sf, cr, osf, BW125, true, 0x12)
				iq = pad(iq, 1024, 512)
				m := NewDemodulator(sf, osf, BW125)
				f, ok := m.DecodeFirst(iq)
				if !ok {
					t.Fatalf("sf=%d cr=%d: decode failed", sf, cr)
				}
				if !f.CRCOK {
					t.Fatalf("sf=%d cr=%d: CRC failed", sf, cr)
				}
				if f.CR != cr {
					t.Fatalf("sf=%d cr=%d: auto-detected CR=%d", sf, cr, f.CR)
				}
				if string(f.Payload) != string(pl) {
					t.Fatalf("sf=%d cr=%d: payload %x want %x", sf, cr, f.Payload, pl)
				}
			}
		}
	}
}

func TestModDemodNoise(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	osf := DefaultOversample
	pl := []byte("noisy lora frame")
	for _, sf := range []int{7, 9, 12} {
		cr := 4 // strongest FEC
		iq := LoRaModulate(pl, sf, cr, osf, BW125, true, 0x12)
		iq = pad(iq, 800, 400)
		// LoRa processing gain grows with SF; a modest SNR suffices.
		iq = addNoise(iq, 0.0, rng)
		m := NewDemodulator(sf, osf, BW125)
		f, ok := m.DecodeFirst(iq)
		if !ok || !f.CRCOK || string(f.Payload) != string(pl) {
			t.Fatalf("sf=%d: ok=%v crc=%v payload=%x", sf, ok, f.CRCOK, f.Payload)
		}
	}
}

func TestModDemodCFO(t *testing.T) {
	osf := DefaultOversample
	pl := []byte("carrier offset")
	for _, sf := range []int{7, 9, 12} {
		for _, cfo := range []float64{-3, 1.5, 4} {
			iq := LoRaModulate(pl, sf, 4, osf, BW125, true, 0x12)
			iq = pad(iq, 600, 600)
			iq = applyCFO(iq, cfo, sf, osf)
			m := NewDemodulator(sf, osf, BW125)
			f, ok := m.DecodeFirst(iq)
			if !ok || !f.CRCOK || string(f.Payload) != string(pl) {
				t.Fatalf("sf=%d cfo=%.1f: ok=%v crc=%v payload=%x", sf, cfo, ok, f.CRCOK, f.Payload)
			}
		}
	}
}

func TestModDemodSFAutoDetect(t *testing.T) {
	// A demodulator tuned to the wrong SF must NOT lock; the right one must.
	osf := DefaultOversample
	pl := []byte("sf detect")
	iq := LoRaModulate(pl, 9, 4, osf, BW125, true, 0x12)
	iq = pad(iq, 500, 500)

	if f, ok := NewDemodulator(9, osf, BW125).DecodeFirst(iq); !ok || string(f.Payload) != string(pl) {
		t.Fatalf("correct SF9 failed: ok=%v", ok)
	}
	// Wrong SF should not produce this payload (it may fail to lock or fail CRC).
	if f, ok := NewDemodulator(7, osf, BW125).DecodeFirst(iq); ok && f.CRCOK && string(f.Payload) == string(pl) {
		t.Fatal("SF7 demod wrongly decoded an SF9 frame")
	}
}
