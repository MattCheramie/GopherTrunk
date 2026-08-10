package main

import (
	"context"
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// fakeVoiceDevice is a minimal sdr.Device: it records the last SetCenterFreq
// and streams one fixed IQ buffer.
type fakeVoiceDevice struct {
	lastCenter uint32
	stream     []complex64
}

func (f *fakeVoiceDevice) Info() sdr.Info                { return sdr.Info{Serial: "fake"} }
func (f *fakeVoiceDevice) SetCenterFreq(hz uint32) error { f.lastCenter = hz; return nil }
func (f *fakeVoiceDevice) SetSampleRate(uint32) error    { return nil }
func (f *fakeVoiceDevice) SetGain(int) error             { return nil }
func (f *fakeVoiceDevice) SetPPM(int) error              { return nil }
func (f *fakeVoiceDevice) SetBiasTee(bool) error         { return nil }
func (f *fakeVoiceDevice) Close() error                  { return nil }
func (f *fakeVoiceDevice) StreamIQ(context.Context) (<-chan []complex64, error) {
	ch := make(chan []complex64, 1)
	ch <- f.stream
	close(ch)
	return ch, nil
}

type fakeResolver struct{ e *sdr.PoolEntry }

func (r fakeResolver) FindBySerial(string) *sdr.PoolEntry { return r.e }

// TestDCAvoidVoiceTuner verifies the two things the wrapper does: SetCenterFreq
// places the hardware LO offsetHz BELOW the requested channel, and StreamIQ
// mixes a carrier sitting at +offsetHz (where that low LO puts it) back down to
// DC — i.e. the offset tuning is undone digitally so the composer sees an
// on-channel stream.
func TestDCAvoidVoiceTuner(t *testing.T) {
	const rate = 2_400_000
	const offset = rate / 4 // 600 kHz — exact quarter-rate, so the mix is exact

	// A pure carrier at +offset: exactly where a channel lands when the LO is
	// tuned offset below it.
	const n = 4096
	in := make([]complex64, n)
	for i := range in {
		th := 2 * math.Pi * float64(offset) / rate * float64(i)
		in[i] = complex(float32(math.Cos(th)), float32(math.Sin(th)))
	}

	fake := &fakeVoiceDevice{stream: in}
	res := fakeResolver{e: &sdr.PoolEntry{Device: fake, Info: sdr.Info{Serial: "fake"}}}
	w := &dcAvoidVoiceTuner{pool: res, serial: "fake", offsetHz: offset, rateHz: rate}

	if got := w.SampleRateHz(); got != rate {
		t.Errorf("SampleRateHz = %d, want %d", got, rate)
	}

	// SetCenterFreq(channel) must tune the hardware LO to channel-offset.
	const channel = 853_000_000
	if err := w.SetCenterFreq(channel); err != nil {
		t.Fatalf("SetCenterFreq: %v", err)
	}
	if fake.lastCenter != channel-offset {
		t.Errorf("hardware LO = %d, want %d (channel - offset)", fake.lastCenter, channel-offset)
	}

	// StreamIQ must mix the +offset carrier down to DC: every output sample
	// collapses to (nearly) the same constant phasor of unit magnitude. A
	// wrong-sign mix would leave a tone at 2*offset (= rate/2, an alternating
	// +1/-1 sequence), which this spread check rejects.
	ch, err := w.StreamIQ(context.Background())
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	out := <-ch
	if len(out) != n {
		t.Fatalf("output len = %d, want %d", len(out), n)
	}
	ref := complex128(out[0])
	var maxDev float64
	for _, s := range out {
		d := complex128(s) - ref
		if m := math.Hypot(real(d), imag(d)); m > maxDev {
			maxDev = m
		}
	}
	if maxDev > 1e-3 {
		t.Errorf("mixed output is not DC: max deviation %.4g from the first sample (want ~0)", maxDev)
	}
	if mag := math.Hypot(real(ref), imag(ref)); math.Abs(mag-1) > 1e-3 {
		t.Errorf("DC magnitude = %.4g, want ~1 (carrier amplitude preserved)", mag)
	}
}
