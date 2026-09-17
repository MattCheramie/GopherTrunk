package receiver

import (
	"math"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
)

// neighbourBurstDibits is one 144-dibit repeater burst: random payload halves
// around the BS-Data sync word, the shape the IPSC idle-beacon trains have.
func neighbourBurstDibits(rng *rand.Rand) []uint8 {
	out := make([]uint8, 144)
	for i := range out {
		out[i] = uint8(rng.Intn(4))
	}
	copy(out[54:78], dmr.BSData.Dibits[:])
	return out
}

// countBSDataSyncs counts BS-Data sync words in a dibit stream (≤ 2 dibits
// off over the 24-dibit word, the usual correlator tolerance).
func countBSDataSyncs(d []uint8) int {
	n := 0
	for i := 0; i+24 <= len(d); i++ {
		miss := 0
		for k, s := range dmr.BSData.Dibits {
			if d[i+k] != s {
				miss++
				if miss > 2 {
					break
				}
			}
		}
		if miss <= 2 {
			n++
			i += 23
		}
	}
	return n
}

// addComplexNoise adds circular AWGN at rmsDb dBFS to iq in place.
func addComplexNoise(rng *rand.Rand, iq []complex64, rmsDb float64) {
	sigma := math.Pow(10, rmsDb/20) / math.Sqrt2
	for i := range iq {
		iq[i] += complex(float32(rng.NormFloat64()*sigma), float32(rng.NormFloat64()*sigma))
	}
}

// neighbourScene models the 15/17 Sep IPSC field condition on the 442.3875 MHz
// wideband tap: the wanted repeater keys in trains with idle gaps, and a
// second carrier 20 kHz away — inside the tap's ±24 kHz but far outside the
// 12.5 kHz channel — sits a few dB below the wanted one the whole time (the
// tap read −52 dBFS in every gap where the capture's in-channel floor was
// −81 dBFS). Returns the IQ for train 1, the gap and train 2, and the
// per-train dibit count fed.
func neighbourScene(rng *rand.Rand) (train1, gap, train2 []complex64) {
	const (
		trainBursts = 60 // ≈1.8 s of back-to-back repeater bursts
		gapSamples  = 48_000 * 3 / 2
	)
	train := func() []complex64 {
		var d []uint8
		for i := 0; i < trainBursts; i++ {
			d = append(d, neighbourBurstDibits(rng)...)
		}
		return makeC4FMIQWithOffset(d, 0)
	}
	train1 = train()
	train2 = train()
	gap = make([]complex64, gapSamples)
	// The neighbour: a continuous 4FSK carrier at −20 kHz, 4 dB BELOW the
	// wanted signal (the tap read it at −52 dBFS against the repeater's
	// −48.5), running through all three phases with a continuous phase so
	// the phases splice cleanly. During a train the FM capture effect lets
	// the wanted signal win; in the gap the neighbour is all there is.
	total := len(train1) + len(gap) + len(train2)
	nd := make([]uint8, total/10+1)
	for i := range nd {
		nd[i] = uint8(rng.Intn(4))
	}
	nb := makeC4FMIQWithOffset(nd, -20_000)
	gainN := float32(math.Pow(10, -4.0/20))
	pos := 0
	for _, buf := range [][]complex64{train1, gap, train2} {
		for i := range buf {
			buf[i] += nb[pos] * complex(gainN, 0)
			pos++
		}
		addComplexNoise(rng, buf, -30)
	}
	return train1, gap, train2
}

// runNeighbourScene feeds the scene through a receiver in RTL-sized chunks
// and returns the BS-Data syncs recovered from each train plus the coarse
// carrier offset the receiver ended on.
func runNeighbourScene(opts Options, train1, gap, train2 []complex64) (syncs1, syncs2 int, offHz float64) {
	var got []uint8
	opts.SampleRateHz = 48_000
	opts.DeviationHz = 1944.0
	opts.DibitSink = func(d []uint8, _ int) { got = append(got, d...) }
	r := New(opts)
	feed := func(iq []complex64) {
		for i := 0; i < len(iq); i += 4096 {
			end := i + 4096
			if end > len(iq) {
				end = len(iq)
			}
			r.Process(iq[i:end])
		}
	}
	feed(train1)
	syncs1 = countBSDataSyncs(got)
	got = got[:0]
	feed(gap)
	got = got[:0]
	feed(train2)
	syncs2 = countBSDataSyncs(got)
	return syncs1, syncs2, r.CoarseCarrierOffsetHz()
}

// TestReceiverIgnoresStrongOffChannelNeighbour pins the channel filter
// against the 15/17 Sep IPSC field condition: a carrier 20 kHz off the
// channel, a few dB stronger than the wanted repeater, must not reach the
// discriminator. Without the filter the neighbour owns the discriminator
// during the wanted repeater's idle gap, the carrier gate reads it as a
// present carrier and the coarse acquirer engages at −20 kHz — after which
// the wanted train is mixed 20 kHz OFF centre and decodes nothing until the
// wideband engine's heal resets the receiver (the field log's
// coarse_offset_hz=−20338 heals). With the filter the acquirer never sees
// it and the second train decodes like the first. Fails against the old
// receiver (no channel filter: second train ≈ 0 syncs, offset ≈ −20 kHz).
func TestReceiverIgnoresStrongOffChannelNeighbour(t *testing.T) {
	rng := rand.New(rand.NewSource(17))
	train1, gap, train2 := neighbourScene(rng)

	s1, s2, off := runNeighbourScene(Options{ChannelFilterHz: DefaultChannelFilterHz}, train1, gap, train2)
	t.Logf("channel filter on:  train1 syncs=%d train2 syncs=%d coarse_offset_hz=%.0f", s1, s2, off)
	u1, u2, uoff := runNeighbourScene(Options{}, train1, gap, train2)
	t.Logf("channel filter off: train1 syncs=%d train2 syncs=%d coarse_offset_hz=%.0f (the field condition)", u1, u2, uoff)

	if s1 < 50 {
		t.Fatalf("first train decoded only %d/60 bursts with the neighbour present", s1)
	}
	if s2 < s1*4/5 {
		t.Fatalf("second train (after an idle gap the neighbour dominated) decoded %d bursts vs %d in the first — the neighbour deafened the receiver", s2, s1)
	}
	if math.Abs(off) > 1000 {
		t.Fatalf("coarse carrier acquirer engaged at %.0f Hz on the off-channel neighbour", off)
	}
}

// TestChannelFilterDesign pins the default filter's shape at the 48 kHz
// channel rate: flat across the DMR signal out to the acquirer's 6 kHz
// mistune reach, ≥ 50 dB down from the 12.5 kHz-spaced adjacent channel's
// far half out to 20 kHz (where the field neighbour sat), and skipped at
// rates too low to fit a passband.
func TestChannelFilterDesign(t *testing.T) {
	taps := channelFilterTaps(48_000, DefaultChannelFilterHz)
	if taps == nil {
		t.Fatal("no taps at 48 kHz")
	}
	resp := func(fHz float64) float64 {
		var re, im float64
		for n, h := range taps {
			th := -2 * math.Pi * fHz * float64(n) / 48_000
			re += float64(h) * math.Cos(th)
			im += float64(h) * math.Sin(th)
		}
		return 20 * math.Log10(math.Hypot(re, im))
	}
	for _, f := range []float64{0, 1944, 3400, 6000} {
		if g := resp(f); math.Abs(g) > 0.5 {
			t.Errorf("passband at %.0f Hz: %.2f dB, want flat within 0.5 dB", f, g)
		}
	}
	for _, f := range []float64{12_500, 15_000, 20_000, 23_000} {
		if g := resp(f); g > -50 {
			t.Errorf("stopband at %.0f Hz: %.1f dB, want ≤ −50 dB", f, g)
		}
	}
	if len(taps) > channelFilterMaxTaps {
		t.Errorf("taps=%d exceeds cap %d", len(taps), channelFilterMaxTaps)
	}
	if got := channelFilterTaps(9_600, DefaultChannelFilterHz); got != nil {
		t.Errorf("filter designed at 9.6 kHz (passband edge past the design limit), want skipped")
	}
}
