package receiver

import (
	"context"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/pager/flex"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// fskOptions shapes a synthetic FLEX transmission.
type fskOptions struct {
	rate      float64 // IQ sample rate
	offsetHz  float64 // carrier offset from the channel centre
	snrDB     float64 // carrier-to-noise over the whole sample rate
	clockPPM  float64 // transmitter symbol clock error vs the receiver's
	inverted  bool    // frequency sense inverted
	leadNoise float64 // seconds of noise before the first frame
}

// modulateFLEX turns back-to-back frames into 4-level FSK IQ: ±4800 Hz outer
// and ±1600 Hz inner deviation, each frame's first 144 symbols at 1600
// symbols/s and the rest at its mode's rate, with the frequency steps
// smoothed like a transmitter's pre-modulation filter.
func modulateFLEX(frames [][]byte, modes []flex.Mode, o fskOptions) []complex64 {
	rng := rand.New(rand.NewSource(7))
	// Symbol schedule: end time and level of every symbol.
	type seg struct {
		end float64
		hz  float64
	}
	var sched []seg
	t := o.leadNoise
	scale := 1 + o.clockPPM*1e-6
	for f, syms := range frames {
		for i, s := range syms {
			baud := float64(flex.SyncSymbolRate)
			if i >= 32+64+48 {
				baud = float64(modes[f].Baud)
			}
			t += scale / baud
			hz := (float64(s)*2 - 3) * 1600
			if o.inverted {
				hz = -hz
			}
			sched = append(sched, seg{t, hz})
		}
	}
	total := int((t + 0.05) * o.rate)
	freq := make([]float64, total)
	k := 0
	for i := range freq {
		ti := float64(i) / o.rate
		for k < len(sched) && ti >= sched[k].end {
			k++
		}
		if ti >= o.leadNoise && k < len(sched) {
			freq[i] = sched[k].hz
		}
	}
	// Pre-modulation smoothing: a short moving average, twice.
	for pass := 0; pass < 2; pass++ {
		sm := make([]float64, total)
		const w = 3
		for i := range sm {
			var s float64
			n := 0
			for j := i - w; j <= i+w; j++ {
				if j >= 0 && j < total {
					s += freq[j]
					n++
				}
			}
			sm[i] = s / float64(n)
		}
		freq = sm
	}
	noise := math.Pow(10, -o.snrDB/20) / math.Sqrt2
	iq := make([]complex64, total)
	var phase float64
	for i := range iq {
		ti := float64(i) / o.rate
		amp := 1.0
		if ti < o.leadNoise || k >= len(sched) && ti > t {
			amp = 0
		}
		phase += 2 * math.Pi * (freq[i] + o.offsetHz) / o.rate
		iq[i] = complex(
			float32(amp*math.Cos(phase)+noise*rng.NormFloat64()),
			float32(amp*math.Sin(phase)+noise*rng.NormFloat64()))
	}
	return iq
}

func phaseFor(t *testing.T, pages ...flex.Page) [flex.PhaseWords]uint32 {
	t.Helper()
	w, ok := flex.PhaseWith(pages...)
	if !ok {
		t.Fatal("pages do not fit a phase")
	}
	return w
}

// modeFrame builds a frame carrying one distinct page per phase.
func modeFrame(t *testing.T, mode flex.Mode, frameNo int) ([]byte, []string) {
	var phases [][flex.PhaseWords]uint32
	var texts []string
	for _, ph := range mode.Phases() {
		text := "Unit " + string(ph) + " respond to 100 Main St, frame " + string(rune('0'+frameNo))
		texts = append(texts, text)
		phases = append(phases, phaseFor(t, flex.Page{Capcode: 1000 + uint32(ph-'A'), Text: text}))
	}
	return flex.FrameSymbols(mode, 1, frameNo, phases...), texts
}

func demodIQ(t *testing.T, iq []complex64, rate uint32) ([]flex.Message, flex.Stats) {
	t.Helper()
	dm, err := NewIQDemod(rate)
	if err != nil {
		t.Fatal(err)
	}
	var got []flex.Message
	for len(iq) > 0 { // odd-sized chunks, as a live stream delivers
		n := 4099
		if n > len(iq) {
			n = len(iq)
		}
		got = append(got, dm.ProcessIQ(iq[:n])...)
		iq = iq[n:]
	}
	return got, dm.Decoder().Stats()
}

// TestDemodDecodesEveryMode: each FLEX mode, as noisy off-centre FSK at the
// channel front end's 48 kHz, in both frequency senses. The 4-level modes are
// the ones a sign slicer cannot read at all.
func TestDemodDecodesEveryMode(t *testing.T) {
	for _, mode := range flex.Modes {
		for _, inverted := range []bool{false, true} {
			frame, texts := modeFrame(t, mode, 5)
			iq := modulateFLEX([][]byte{frame}, []flex.Mode{mode}, fskOptions{
				rate: 48000, offsetHz: 600, snrDB: 18, inverted: inverted, leadNoise: 0.3,
			})
			got, st := demodIQ(t, iq, 48000)
			if len(got) != len(texts) {
				t.Errorf("%d bps (%d/%d) inverted=%v: %d pages, want %d (stats %+v)",
					mode.BitRate(), mode.Baud, mode.Levels, inverted, len(got), len(texts), st)
				continue
			}
			for i, m := range got {
				if m.Text != texts[i] || m.Mode != mode {
					t.Errorf("%d bps inverted=%v page %d = %v, want %q", mode.BitRate(), inverted, i, m, texts[i])
				}
			}
		}
	}
}

// TestDemodFollowsMixedRates: frames of different modes back to back, with a
// 60 ppm clock error between transmitter and receiver (a cheap dongle), each
// frame acquired on its own.
func TestDemodFollowsMixedRates(t *testing.T) {
	order := []flex.Mode{flex.Mode3200x4, flex.Mode1600x2, flex.Mode3200x2, flex.Mode1600x4, flex.Mode3200x4}
	for _, ppm := range []float64{-60, 60} {
		var frames [][]byte
		var want []string
		for i, mode := range order {
			f, texts := modeFrame(t, mode, i)
			frames = append(frames, f)
			want = append(want, texts...)
		}
		iq := modulateFLEX(frames, order, fskOptions{rate: 48000, offsetHz: -400, snrDB: 20, clockPPM: ppm, leadNoise: 0.2})
		got, st := demodIQ(t, iq, 48000)
		if len(got) != len(want) {
			t.Fatalf("%+.0f ppm: %d pages, want %d (stats %+v)", ppm, len(got), len(want), st)
		}
		for i, m := range got {
			if m.Text != want[i] {
				t.Errorf("%+.0f ppm page %d = %v, want %q", ppm, i, m, want[i])
			}
		}
		if st.Frames1600 != 1 || st.Frames3200 != 2 || st.Frames6400 != 2 || st.WordsFailed != 0 {
			t.Errorf("%+.0f ppm stats = %+v, want 1/2/2 frames and no failed words", ppm, st)
		}
	}
}

// TestDemodIgnoresNoise: ten seconds of noise produce no frame.
func TestDemodIgnoresNoise(t *testing.T) {
	iq := modulateFLEX(nil, nil, fskOptions{rate: 48000, snrDB: 0, leadNoise: 10})
	got, st := demodIQ(t, iq, 48000)
	if len(got) != 0 || st.FramesSynced != 0 {
		t.Errorf("noise decoded %d pages, %d frames", len(got), st.FramesSynced)
	}
}

// TestReceiverPublishesPages runs a 6400 bps frame through the live Receiver
// and expects its four pages on the bus.
func TestReceiverPublishesPages(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	r, err := New(Options{InputRateHz: 48000, Bus: bus, SourceName: "test"})
	if err != nil {
		t.Fatal(err)
	}
	frame, texts := modeFrame(t, flex.Mode3200x4, 9)
	iq := modulateFLEX([][]byte{frame}, []flex.Mode{flex.Mode3200x4}, fskOptions{rate: 48000, snrDB: 25, leadNoise: 0.1})
	in := make(chan []complex64, 1)
	in <- iq
	close(in)
	if err := r.Process(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	for i, want := range texts {
		select {
		case ev := <-sub.C:
			msg, ok := ev.Payload.(storage.PagerMessage)
			if !ok || ev.Kind != events.KindPagerMessage {
				t.Fatalf("event %d: %+v", i, ev)
			}
			if msg.Protocol != "flex" || msg.Body != want || msg.RIC != 1000+uint32(i) || msg.Encoding != "alpha" {
				t.Errorf("page %d = %+v, want %q", i, msg, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("page %d never reached the bus", i)
		}
	}
	if st := r.Stats(); st.Frames6400 != 1 || r.PagesEmitted() != 4 {
		t.Errorf("stats = %+v, pages emitted %d", st, r.PagesEmitted())
	}
}

// TestResamplerHandlesAwkwardRates: a front-end rate with no small ratio to
// 25.6 kHz is approximated (the timing loop takes up the few ppm), not refused.
func TestResamplerHandlesAwkwardRates(t *testing.T) {
	for _, rate := range []uint32{48000, 51200, 44100, 22050, 48077, 50001} {
		if _, err := newResampler(rate); err != nil {
			t.Errorf("rate %d Hz: %v", rate, err)
		}
	}
	if _, err := newResampler(0); err == nil {
		t.Error("zero rate accepted")
	}
	// The approximated path still decodes.
	frame, texts := modeFrame(t, flex.Mode3200x4, 2)
	iq := modulateFLEX([][]byte{frame}, []flex.Mode{flex.Mode3200x4}, fskOptions{rate: 48077, snrDB: 20, leadNoise: 0.2})
	got, st := demodIQ(t, iq, 48077)
	if len(got) != len(texts) || st.WordsFailed != 0 {
		t.Errorf("48077 Hz: %d pages, stats %+v", len(got), st)
	}
}
