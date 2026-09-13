package afsk

import (
	"context"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/fleetsync"
)

// The reporter's ground truth from #1184 (Fleet 107, Unit 1772) is the
// vector for every synthetic run, so the harness and these tests assert
// the same thing.
const (
	wantFleet = 107
	wantUnit  = 1772
)

// synthIQ builds an FFSK-modulated IQ stream at rate Hz carrying one
// FleetSync burst: a symbol-clock training run of dotting, the burst
// itself, and a trailing run so the last payload bits clear the filter
// and timing-loop delays. markHz/spaceHz let a test swap the tone sense.
func synthIQ(t *testing.T, rate, baud float64, fs2 bool, markHz, spaceHz float64) []complex64 {
	t.Helper()
	burst, err := fleetsync.SynthBurst(wantFleet, wantUnit, fs2)
	if err != nil {
		t.Fatal(err)
	}
	bits := append(fleetsync.Dotting(96), burst...)
	bits = append(bits, fleetsync.Dotting(96)...)
	return demod.ModulateFFSK(bits, rate, baud, markHz, spaceHz)
}

// runIQ feeds iq through a fresh Receiver in chunks of chunkLen and
// returns every framed message.
func runIQ(t *testing.T, iq []complex64, rate uint32, baud, chunkLen int) []fleetsync.Message {
	t.Helper()
	var got []fleetsync.Message
	r, err := New(Options{
		InputRateHz: rate,
		BaudHz:      baud,
		OnMessage:   func(m fleetsync.Message) { got = append(got, m) },
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(iq); i += chunkLen {
		end := i + chunkLen
		if end > len(iq) {
			end = len(iq)
		}
		r.ProcessIQ(iq[i:end])
	}
	return got
}

func assertANI(t *testing.T, got []fleetsync.Message, fs2 bool, label string) {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("%s: framed %d messages, want 1 (%+v)", label, len(got), got)
	}
	m := got[0]
	if !m.CRCOK || m.Fleet != wantFleet || m.Unit != wantUnit || m.IsFS2 != fs2 {
		t.Fatalf("%s: decoded %+v, want CRCOK fleet %d unit %d IsFS2=%v", label, m, wantFleet, wantUnit, fs2)
	}
}

// TestReceiverDecodesSynthesisedBursts is the DSP → framer → decoder
// end-to-end round trip: FS-I and FS-II bursts, FFSK-modulated onto an
// FM carrier at several capture rates (exercising trivial and non-trivial
// resample ratios), must decode to the ANI they carry.
func TestReceiverDecodesSynthesisedBursts(t *testing.T) {
	for _, rate := range []uint32{9600, 48000, 96000, 250000} {
		for _, fs2 := range []bool{false, true} {
			iq := synthIQ(t, float64(rate), DefaultBaudHz, fs2, MarkHz, SpaceHz)
			got := runIQ(t, iq, rate, DefaultBaudHz, 4096)
			assertANI(t, got, fs2, "rate "+itoa(int(rate))+" fs2="+boolStr(fs2))
		}
	}
}

// TestReceiverDecodesInvertedToneSense: an FM discriminator can present
// the burst with the opposite tone sense (mark on 1800 Hz). The framer
// locks on the complemented sync word and de-inverts the payload, so the
// front end must still deliver the right ANI.
func TestReceiverDecodesInvertedToneSense(t *testing.T) {
	iq := synthIQ(t, 48000, DefaultBaudHz, false, SpaceHz, MarkHz) // tones swapped
	got := runIQ(t, iq, 48000, DefaultBaudHz, 4096)
	assertANI(t, got, false, "inverted tone sense")
}

// TestReceiverIsChunkInvariant: the decode must not depend on how the IQ
// stream is chopped — every stage carries state across calls.
func TestReceiverIsChunkInvariant(t *testing.T) {
	iq := synthIQ(t, 48000, DefaultBaudHz, true, MarkHz, SpaceHz)
	for _, chunkLen := range []int{1, 37, 500, 4096, len(iq)} {
		got := runIQ(t, iq, 48000, DefaultBaudHz, chunkLen)
		assertANI(t, got, true, "chunk "+itoa(chunkLen))
	}
}

// TestReceiverDecodesUnderNoise: a burst buried in additive white noise
// (≈12 dB SNR on the IQ, well below what a usable FM channel delivers)
// still decodes; and FS-II's ECC is exercised through the DSP path, not
// only on ideal bits.
func TestReceiverDecodesUnderNoise(t *testing.T) {
	rng := rand.New(rand.NewSource(1184))
	for _, fs2 := range []bool{false, true} {
		iq := synthIQ(t, 48000, DefaultBaudHz, fs2, MarkHz, SpaceHz)
		const sigma = 0.18 // per-component; unit-magnitude carrier ⇒ ~12 dB
		noisy := make([]complex64, len(iq))
		for i, s := range iq {
			noisy[i] = s + complex(float32(rng.NormFloat64()*sigma), float32(rng.NormFloat64()*sigma))
		}
		got := runIQ(t, noisy, 48000, DefaultBaudHz, 4096)
		assertANI(t, got, fs2, "noisy fs2="+boolStr(fs2))
	}
}

// TestReceiverToleratesCarrierOffset: a few hundred hertz of tuning error
// puts a DC term on the FM discriminator output; the FFSK stage's mix +
// low-pass must remove it so the slicer threshold stays valid.
func TestReceiverToleratesCarrierOffset(t *testing.T) {
	const rate = 48000.0
	iq := synthIQ(t, rate, DefaultBaudHz, false, MarkHz, SpaceHz)
	for _, offHz := range []float64{-600, 400} {
		shifted := make([]complex64, len(iq))
		step := 2 * math.Pi * offHz / rate
		for i, s := range iq {
			c, sn := math.Cos(step*float64(i)), math.Sin(step*float64(i))
			re, im := float64(real(s)), float64(imag(s))
			shifted[i] = complex(float32(re*c-im*sn), float32(re*sn+im*c))
		}
		got := runIQ(t, shifted, uint32(rate), DefaultBaudHz, 4096)
		assertANI(t, got, false, "carrier offset "+itoa(int(offHz))+" Hz")
	}
}

// TestReceiverAudioPathMatchesIQPath: feeding the FM-discriminated audio
// directly (ProcessAudio — a discriminator tap or mono audio capture)
// decodes the same burst as the IQ path.
func TestReceiverAudioPathMatchesIQPath(t *testing.T) {
	iq := synthIQ(t, 48000, DefaultBaudHz, false, MarkHz, SpaceHz)
	fm := demod.NewFM()
	audio := fm.Process(nil, iq)

	var got []fleetsync.Message
	r, err := New(Options{InputRateHz: 48000, OnMessage: func(m fleetsync.Message) { got = append(got, m) }})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(audio); i += 1000 {
		end := i + 1000
		if end > len(audio) {
			end = len(audio)
		}
		r.ProcessAudio(audio[i:end])
	}
	assertANI(t, got, false, "audio path")
	if s := r.Stats(); s.SamplesSeen != uint64(len(audio)) || s.BitsEmitted == 0 {
		t.Errorf("stats = %+v, want SamplesSeen=%d and bits emitted", s, len(audio))
	}
}

// TestReceiverWrongBaudDoesNotFalseDecode: a 1200-baud burst run through a
// 2400-baud receiver must not produce a CRC-valid ANI — the harness's
// baud sweep relies on the CRC being the gate, never on sync locks.
func TestReceiverWrongBaudDoesNotFalseDecode(t *testing.T) {
	iq := synthIQ(t, 48000, DefaultBaudHz, false, MarkHz, SpaceHz)
	got := runIQ(t, iq, 48000, 2400, 4096)
	for _, m := range got {
		if m.CRCOK {
			t.Fatalf("2400-baud receiver CRC-validated a 1200-baud burst: %+v", m)
		}
	}
}

// TestReceiverIgnoresNoiseOnly: seconds of pure noise must not yield a
// CRC-valid burst (false ANI would be worse than none for an operator).
func TestReceiverIgnoresNoiseOnly(t *testing.T) {
	rng := rand.New(rand.NewSource(437))
	noise := make([]complex64, 48000*5)
	for i := range noise {
		noise[i] = complex(float32(rng.NormFloat64()*0.3), float32(rng.NormFloat64()*0.3))
	}
	got := runIQ(t, noise, 48000, DefaultBaudHz, 4096)
	for _, m := range got {
		if m.CRCOK {
			t.Fatalf("noise-only input produced a CRC-valid burst: %+v", m)
		}
	}
}

func TestReceiverNewRejectsBadOptions(t *testing.T) {
	if _, err := New(Options{InputRateHz: 48000}); err == nil {
		t.Error("New without OnMessage: want error")
	}
	if _, err := New(Options{OnMessage: func(fleetsync.Message) {}}); err == nil {
		t.Error("New without InputRateHz: want error")
	}
	if _, err := New(Options{InputRateHz: 48000, BaudHz: -1, OnMessage: func(fleetsync.Message) {}}); err == nil {
		t.Error("New with negative BaudHz: want error")
	}
}

func TestReceiverProcessLifecycle(t *testing.T) {
	r, err := New(Options{InputRateHz: 48000, OnMessage: func(fleetsync.Message) {}})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Process(context.Background(), nil); err == nil {
		t.Error("Process with nil input: want error")
	}

	in := make(chan []complex64)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Process(ctx, in) }()
	cancel()
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("Process err = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Process did not exit after ctx cancel")
	}

	in2 := make(chan []complex64, 1)
	done2 := make(chan error, 1)
	go func() { done2 <- r.Process(context.Background(), in2) }()
	in2 <- make([]complex64, 4800)
	close(in2)
	select {
	case err := <-done2:
		if err != nil {
			t.Errorf("Process on closed input = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Process did not exit after input close")
	}
	if got := r.Stats().SamplesSeen; got != 4800 {
		t.Errorf("SamplesSeen = %d, want 4800", got)
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
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
