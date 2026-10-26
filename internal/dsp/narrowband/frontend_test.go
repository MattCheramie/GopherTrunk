package narrowband

import (
	"context"
	"math"
	"math/cmplx"
	"testing"
)

func TestPickDecimation(t *testing.T) {
	cases := []struct {
		in   float64
		want int
	}{
		{2_400_000, 50}, // 48 kHz
		{2_048_000, 40}, // 51.2 kHz
		{1_024_000, 20}, // 51.2 kHz
		{3_000_000, 60}, // 50 kHz
		{2_500_000, 50}, // 50 kHz
		{250_000, 5},    // 50 kHz
		{96_000, 2},     // 48 kHz
		{48_000, 1},
		{20_000, 1},
		{48_000.5, 1}, // not an integer rate
	}
	for _, c := range cases {
		if got := PickDecimation(c.in); got != c.want {
			t.Errorf("PickDecimation(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

// tone returns n samples of a unit complex tone at fHz.
func tone(fHz, rateHz float64, n int) []complex64 {
	out := make([]complex64, n)
	for i := range out {
		ph := 2 * math.Pi * fHz * float64(i) / rateHz
		out[i] = complex(float32(math.Cos(ph)), float32(math.Sin(ph)))
	}
	return out
}

func rmsDB(x []complex64, skip int) float64 {
	var p float64
	n := 0
	for _, v := range x[skip:] {
		p += float64(real(v)*real(v) + imag(v)*imag(v))
		n++
	}
	if n == 0 || p == 0 {
		return -200
	}
	return 10 * math.Log10(p/float64(n))
}

// TestFrontEndPassesChannelRejectsNeighbour pins the reason the front end
// exists: a carrier in the channel comes through at unity, one 20 kHz away
// (the next-but-one 12.5 kHz channel, inside the old full-rate
// discriminator's view) is attenuated by more than 50 dB.
func TestFrontEndPassesChannelRejectsNeighbour(t *testing.T) {
	const rate = 2_400_000
	n := 2 * rate / 10 // 200 ms
	skip := 4800       // filter settling at the 48 kHz output

	fe := New(rate, 0, 0)
	if fe.OutRate() != 48_000 {
		t.Fatalf("OutRate = %d, want 48000", fe.OutRate())
	}
	in := rmsDB(fe.Process(tone(2_000, rate, n)), skip)
	fe.Reset()
	out := rmsDB(fe.Process(tone(20_000, rate, n)), skip)
	if in < -1 || in > 1 {
		t.Errorf("in-channel tone level = %.1f dB, want ~0 dB", in)
	}
	if in-out < 50 {
		t.Errorf("neighbour at +20 kHz attenuated only %.1f dB, want > 50 dB", in-out)
	}
	t.Logf("in-channel %.2f dB, +20 kHz %.2f dB", in, out)
}

// TestFrontEndOffsetMixesChannelToDC: with the LO tuned 300 kHz below the
// channel the carrier arrives at +300 kHz; the front end must deliver it
// at 0 Hz at full level and reject a carrier that really is at 0 Hz (the
// tuner's DC spur).
func TestFrontEndOffsetMixesChannelToDC(t *testing.T) {
	const rate = 2_400_000
	const off = 300_000
	n := 2 * rate / 10
	skip := 4800

	fe := New(rate, off, 0)
	y := fe.Process(tone(off+1_000, rate, n))
	lvl := rmsDB(y, skip)
	if lvl < -1 || lvl > 1 {
		t.Errorf("offset channel level = %.1f dB, want ~0 dB", lvl)
	}
	// The output should be a tone at +1 kHz: measure its frequency from
	// the mean phase step at 48 kHz.
	var acc complex128
	for i := skip + 1; i < len(y); i++ {
		acc += complex128(y[i]) * cmplx.Conj(complex128(y[i-1]))
	}
	fHz := cmplx.Phase(acc) / (2 * math.Pi) * fe.OutRateHz()
	if math.Abs(fHz-1_000) > 5 {
		t.Errorf("output tone at %.1f Hz, want 1000 Hz", fHz)
	}
	fe.Reset()
	spur := rmsDB(fe.Process(tone(0, rate, n)), skip)
	if lvl-spur < 50 {
		t.Errorf("DC spur attenuated only %.1f dB, want > 50 dB", lvl-spur)
	}
}

func TestFrontEndNoDecimationBelowChannelRate(t *testing.T) {
	fe := New(48_000, 0, 0)
	if fe.Decimation() != 1 || fe.OutRate() != 48_000 {
		t.Fatalf("48 kHz input: decim=%d out=%d", fe.Decimation(), fe.OutRate())
	}
	y := fe.Process(tone(1_000, 48_000, 9600))
	if len(y) != 9600 {
		t.Errorf("len(out) = %d, want 9600 (no decimation)", len(y))
	}
}

func TestPumpDeliversAndCloses(t *testing.T) {
	fe := New(2_400_000, 0, 0)
	in := make(chan []complex64, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := fe.Pump(ctx, in)
	in <- tone(1_000, 2_400_000, 48_000)
	in <- tone(1_000, 2_400_000, 48_000)
	close(in)
	total := 0
	for chunk := range out {
		total += len(chunk)
	}
	if total != 2*48_000/50 {
		t.Errorf("pumped %d samples, want %d", total, 2*48_000/50)
	}
}
