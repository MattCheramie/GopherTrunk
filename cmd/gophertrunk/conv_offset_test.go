package main

import (
	"context"
	"io"
	"log/slog"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// clippingFrontEnd simulates an overloaded 8-bit SDR front end receiving an
// analog FM transmitter: the carrier sits at channelHz+carrierOffHz (a real
// transmitter is never exactly on the LO's idea of the channel), the ADC clips
// I and Q INDEPENDENTLY at full scale (overdrive > 1 squares the constellation,
// exactly what the #1184 Kenwood captures show: |IQ| ∈ [1.0, 1.414]), and the
// front end adds its DC spur. SetCenterFreq records where the LO was put; the
// stream is synthesised relative to it, so an offset-tuning front end sees the
// carrier far from DC and an on-channel one sees it within a few hundred Hz.
type clippingFrontEnd struct {
	rateHz       float64
	channelHz    float64
	carrierOffHz float64
	toneHz       float64
	devHz        float64
	overdrive    float64
	dc           complex64
	seconds      float64

	mu   sync.Mutex
	loHz float64
}

func (c *clippingFrontEnd) SetCenterFreq(hz uint32) error {
	c.mu.Lock()
	c.loHz = float64(hz)
	c.mu.Unlock()
	return nil
}

func (c *clippingFrontEnd) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	c.mu.Lock()
	ifHz := c.channelHz + c.carrierOffHz - c.loHz
	c.mu.Unlock()
	out := make(chan []complex64, 4)
	n := int(c.rateHz * c.seconds)
	go func() {
		defer close(out)
		const chunk = 16384
		phase := 0.0
		quant := func(v float64) float32 {
			if v > 1 {
				v = 1
			} else if v < -1 {
				v = -1
			}
			return float32(math.Round(v*127) / 127) // 8-bit ADC
		}
		for start := 0; start < n; start += chunk {
			end := min(start+chunk, n)
			buf := make([]complex64, end-start)
			for i := range buf {
				t := float64(start+i) / c.rateHz
				inst := ifHz + c.devHz*math.Sin(2*math.Pi*c.toneHz*t)
				phase += 2 * math.Pi * inst / c.rateHz
				re := c.overdrive*math.Cos(phase) + float64(real(c.dc))
				im := c.overdrive*math.Sin(phase) + float64(imag(c.dc))
				buf[i] = complex(quant(re), quant(im))
			}
			select {
			case out <- buf:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

// demodChannelAudio is a minimal analog-FM receive chain (channel low-pass,
// decimate to 48 kHz, quadrature discriminator) — the same stages the
// composer's runFMChain runs ahead of its audio filters.
func demodChannelAudio(t *testing.T, in <-chan []complex64, rateHz float64) (audio []float32, audioHz float64) {
	t.Helper()
	var iq []complex64
	for c := range in {
		iq = append(iq, c...)
	}
	decim := int(math.Round(rateHz / 48_000))
	taps := filter.LowpassKaiser(255, 15_000/rateHz, 8.6)
	var bb []complex64
	for o := len(taps); o < len(iq); o += decim {
		var acc complex64
		for k, h := range taps {
			acc += iq[o-k] * complex(h, 0)
		}
		bb = append(bb, acc)
	}
	audio = demod.NewFM().Process(nil, bb)
	return audio[len(audio)/10:], rateHz / float64(decim) // drop filter settle
}

// toneAmplitude is a Goertzel-style single-bin DFT magnitude.
func toneAmplitude(x []float32, fs, f float64) float64 {
	var re, im float64
	w := 2 * math.Pi * f / fs
	for i, v := range x {
		win := 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(len(x)-1))
		re += float64(v) * win * math.Cos(w*float64(i))
		im -= float64(v) * win * math.Sin(w*float64(i))
	}
	return math.Hypot(re, im)
}

// clipSpurDb runs the clipping front end through a convScannerFrontEnd with
// the given scanner.lo_offset_hz and returns the level (dB) of the tone at
// 4×|carrier offset| relative to the wanted 1 kHz modulation tone, plus the
// audio SINAD (1 kHz tone power over everything else).
func clipSpurDb(t *testing.T, loOffsetCfg int) (spurDb, sinad, loOffset float64) {
	t.Helper()
	const rate = 2_400_000
	fe := &clippingFrontEnd{
		rateHz: rate, channelHz: 447_100_000, carrierOffHz: -420,
		toneHz: 1000, devHz: 2500, overdrive: 3, dc: complex(0.02, -0.015),
		seconds: 0.4,
	}
	off, _ := convScannerLOOffsetHz(rate, loOffsetCfg)
	w := newConvScannerFrontEnd(fe, off, rate, "test", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := w.SetCenterFreq(uint32(fe.channelHz)); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := w.StreamIQ(ctx)
	if err != nil {
		t.Fatal(err)
	}
	audio, audioHz := demodChannelAudio(t, stream, rate)
	wanted := toneAmplitude(audio, audioHz, fe.toneHz)
	spur := toneAmplitude(audio, audioHz, 4*math.Abs(fe.carrierOffHz))
	return 20 * math.Log10(spur/wanted), toneSINADDb(audio, audioHz, fe.toneHz), off
}

// toneSINADDb fits the wanted tone (least squares, after removing the mean) and
// returns its power over the residual's.
func toneSINADDb(x []float32, fs, f float64) float64 {
	var mean float64
	for _, v := range x {
		mean += float64(v)
	}
	mean /= float64(len(x))
	var c, s float64
	w := 2 * math.Pi * f / fs
	for i, v := range x {
		c += (float64(v) - mean) * math.Cos(w*float64(i))
		s += (float64(v) - mean) * math.Sin(w*float64(i))
	}
	c *= 2 / float64(len(x))
	s *= 2 / float64(len(x))
	var sig, res float64
	for i, v := range x {
		tone := c*math.Cos(w*float64(i)) + s*math.Sin(w*float64(i))
		e := float64(v) - mean - tone
		sig += tone * tone
		res += e * e
	}
	return 10 * math.Log10(sig/res)
}

// TestConvScannerOffsetTuningRemovesClippingWhistle is the #1184 regression:
// with the LO on-channel (the pre-fix scanner, reproduced here with
// scanner.lo_offset_hz < 0), an overloaded front end puts a tone at 4× the
// carrier offset into the FM audio — the reporter's Kenwood "high-pitched
// noise" (their captures carry it 18–25 dB above the voice floor at exactly
// 4× each file's offset). With the default automatic LO offset the same RF
// demodulates clean.
//
// It also pins why the offset is searched rather than taken as sample_rate/4
// (the dc_avoid default): at fs/4 the 3rd-order clipping product aliases
// exactly onto the channel, and the audio is as wrecked as on-channel.
func TestConvScannerOffsetTuningRemovesClippingWhistle(t *testing.T) {
	onChan, onSINAD, off0 := clipSpurDb(t, -1)
	if off0 != 0 {
		t.Fatalf("lo_offset_hz<0 must tune on-channel, got offset %v", off0)
	}
	offTuned, offSINAD, off := clipSpurDb(t, 0)
	_, quarterSINAD, _ := clipSpurDb(t, 600_000) // fs/4 at 2.4 MS/s
	t.Logf("on-channel: spur %.1f dB, SINAD %.1f dB | fs/4: SINAD %.1f dB | auto offset %.0f Hz: spur %.1f dB, SINAD %.1f dB",
		onChan, onSINAD, quarterSINAD, off, offTuned, offSINAD)
	if off == 0 {
		t.Fatal("default config must enable LO offset tuning at 2.4 MS/s")
	}
	if onChan < -30 || onSINAD > 10 {
		t.Fatalf("fixture no longer reproduces the on-channel clipping damage (spur %.1f dB, SINAD %.1f dB); the test cannot fail first", onChan, onSINAD)
	}
	if offTuned > -45 || offSINAD < 40 {
		t.Fatalf("offset-tuned: spur %.1f dB (want < -45), SINAD %.1f dB (want > 40)", offTuned, offSINAD)
	}
	if quarterSINAD > 10 {
		t.Fatalf("fs/4 offset SINAD %.1f dB: expected the aliased clipping product to wreck it (the reason the picker avoids fs/4)", quarterSINAD)
	}
}

// TestPickClipSafeLOOffset pins the offset picker: every common RTL/Airspy
// rate gets an offset whose DC spur, image and clipping products all clear
// the channel, and sample_rate/4 — the dc_avoid default, whose 3rd-order
// clipping product aliases exactly onto the channel — is never chosen.
func TestPickClipSafeLOOffset(t *testing.T) {
	for _, fs := range []float64{960_000, 1_024_000, 1_800_000, 2_048_000, 2_400_000, 2_560_000, 3_000_000, 3_200_000, 6_000_000, 10_000_000} {
		off := pickClipSafeLOOffsetHz(fs)
		if off < convOffsetMinHz || off > convOffsetMaxFrac*fs {
			t.Errorf("fs=%.0f: offset %.0f outside [%d, %.0f]", fs, off, convOffsetMinHz, convOffsetMaxFrac*fs)
			continue
		}
		if s := loOffsetClearanceHz(off, fs); s <= 0 {
			t.Errorf("fs=%.0f: offset %.0f leaves a product in the channel (clearance %.0f Hz)", fs, off, s)
		}
		if math.Abs(off-fs/4) < 1 {
			t.Errorf("fs=%.0f: picked fs/4, which aliases the clipping product onto the channel", fs)
		}
	}
	if s := loOffsetClearanceHz(600_000, 2_400_000); s > 0 {
		t.Errorf("fs/4 at 2.4 MS/s scored clear (%.0f Hz); the 3rd-order clipping product lands on the channel", s)
	}
	if off := pickClipSafeLOOffsetHz(250_000); off != 0 {
		t.Errorf("250 kS/s has no room for an offset, got %.0f", off)
	}
}

func TestConvScannerLOOffsetConfig(t *testing.T) {
	if off, _ := convScannerLOOffsetHz(2_400_000, -1); off != 0 {
		t.Errorf("negative disables: got %v", off)
	}
	if off, _ := convScannerLOOffsetHz(2_400_000, 250_000); off != 250_000 {
		t.Errorf("explicit offset: got %v", off)
	}
	if off, _ := convScannerLOOffsetHz(2_400_000, 1_000_000); off != 0 {
		t.Errorf("offset beyond 35%% of the rate must be refused, got %v", off)
	}
	if off, _ := convScannerLOOffsetHz(2_400_000, 0); off == 0 {
		t.Error("auto must pick an offset at 2.4 MS/s")
	}
}

// TestConvScannerFrontEndTunesBelowChannel pins the tuning contract the FM
// chain relies on: the LO goes offset BELOW the channel.
func TestConvScannerFrontEndTunesBelowChannel(t *testing.T) {
	fe := &clippingFrontEnd{}
	w := newConvScannerFrontEnd(fe, 200_000, 2_400_000, "s", nil)
	if err := w.SetCenterFreq(447_100_000); err != nil {
		t.Fatal(err)
	}
	if fe.loHz != 446_900_000 {
		t.Fatalf("LO = %.0f, want 446900000", fe.loHz)
	}
}

// TestConvScannerFrontEndWarnsOnOverload: a rail-pinned stream logs the
// overload WARN (once per gap), a clean one does not.
func TestConvScannerFrontEndWarnsOnOverload(t *testing.T) {
	var buf strings.Builder
	var mu sync.Mutex
	log := slog.New(slog.NewTextHandler(lockedWriter{&mu, &buf}, nil))
	w := newConvScannerFrontEnd(&clippingFrontEnd{}, 0, 2_400_000, "RT", log)
	now := time.Unix(0, 0)
	w.now = func() time.Time { return now }
	clean := make([]complex64, 1000)
	for i := range clean {
		clean[i] = complex(0.3, -0.2)
	}
	railed := make([]complex64, 1000)
	for i := range railed {
		railed[i] = complex(1, -1)
	}
	for range 3 {
		w.observeRaw(clean)
		now = now.Add(600 * time.Millisecond)
	}
	mu.Lock()
	if strings.Contains(buf.String(), "overloaded") {
		t.Fatalf("clean stream warned: %s", buf.String())
	}
	mu.Unlock()
	for range 6 {
		w.observeRaw(railed)
		now = now.Add(600 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if got := strings.Count(buf.String(), "front end overloaded"); got != 1 {
		t.Fatalf("want exactly one rate-limited overload WARN, got %d: %s", got, buf.String())
	}
}

type lockedWriter struct {
	mu *sync.Mutex
	w  io.Writer
}

func (l lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}
