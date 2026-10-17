//go:build integration

package phase2_test

// Does a fixed frequency correction restore lock on the part of the capture
// the receiver loses? A constant dibit rotation in a differentially decoded
// system is the signature of a residual carrier offset (each unit of rotation
// is SymbolRate/4 = 1500 Hz), so if the late stream is merely mistuned, a DDC
// offset should bring the sync back at the canonical rotation and a low
// tolerance. If nothing does, the loss is symbol timing, not carrier.

import (
	"os"
	"testing"

	rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
)

func syncHits(stream []uint8, rot, tol int) int {
	base := syncDibitsLocal()
	pat := make([]uint8, 20)
	for i, p := range base {
		pat[i] = (p + uint8(rot)) & 3
	}
	hits := 0
	for i := 0; i+20 <= len(stream); i++ {
		miss := 0
		for k := 0; k < 20; k++ {
			if stream[i+k] != pat[k] {
				miss++
				if miss > tol {
					break
				}
			}
		}
		if miss <= tol {
			hits++
			i += 19
		}
	}
	return hits
}

func TestACCHFrequencySweep(t *testing.T) {
	path := os.Getenv("GT_P2_WAV")
	if path == "" {
		t.Skip("set GT_P2_WAV")
	}
	iq, rate := readWavIQ(t, path)

	run := func(seg []complex64, offsetHz float64) []uint8 {
		var stream []uint8
		dc := ccdecoder.NewDownconverterWithOffset(rate, 48000, offsetHz)
		r := rx.New(rx.Options{SampleRateHz: dc.OutRateHz(), ClockMode: rx.ClockGardner,
			GardnerGain: 0.005, DibitSink: func(d []uint8, _ int) { stream = append(stream, d...) }})
		var buf []complex64
		const chunk = 8192
		for i := 0; i < len(seg); i += chunk {
			end := i + chunk
			if end > len(seg) {
				end = len(seg)
			}
			buf = dc.Process(buf[:0], seg[i:end])
			r.Process(buf)
		}
		return stream
	}

	segs := map[string][]complex64{
		"first 1.0 s": iq[:int(rate)],
		"1.0-3.5 s":   iq[int(rate):min(len(iq), int(3.5*rate))],
		"3.5 s-end":   iq[min(len(iq), int(3.5*rate)):],
	}
	for _, name := range []string{"first 1.0 s", "1.0-3.5 s", "3.5 s-end"} {
		seg := segs[name]
		best, bestHits := 0.0, -1
		var line string
		for _, off := range []float64{-3000, -2250, -1500, -750, 0, 750, 1500, 2250, 3000} {
			st := run(seg, off)
			h := syncHits(st, 0, 2)
			line += " " + itoa(int(off)) + ":" + itoa(h)
			if h > bestHits {
				best, bestHits = off, h
			}
		}
		t.Logf("%-11s expect≈%d superframes | hits@offset:%s | best %+.0f Hz → %d",
			name, int(float64(len(seg))/rate/0.36), line, best, bestHits)
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b []byte
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}
