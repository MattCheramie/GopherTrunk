package composer

import (
	"bytes"
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
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	p25p1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice/imbe"
)

// discardLogger returns a slog.Logger that drops every record. Used by
// unit tests that exercise warn-log code paths without wanting the
// noise in `go test -v` output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// p25p1VoiceInfo builds a deterministic, unique 88-bit IMBE info frame.
func p25p1VoiceInfo(seed int) []byte {
	bits := make([]byte, imbe.InfoBits)
	x := uint32(seed)*2654435761 + 1
	for i := range bits {
		x = x*1664525 + 1013904223
		bits[i] = byte(x >> 31)
	}
	return bits
}

// buildP25P1VoiceStream assembles a dibit stream of `ldus` P25 Phase 1
// LDU1s preceded by a clock-settling lead-in. want holds every IMBE
// frame it carries, in transmission order.
func buildP25P1VoiceStream(t *testing.T, ldus int) (dibits []uint8, want [][]byte) {
	return buildP25P1VoiceStreamSeed(t, ldus, 0)
}

// buildP25P1VoiceStreamSeed is buildP25P1VoiceStream with a seed offset so a
// test can build two streams with DISTINCT voice frames (e.g. a wanted call
// and an adjacent-channel neighbour) and tell which one the chain decoded.
func buildP25P1VoiceStreamSeed(t *testing.T, ldus, seed0 int) (dibits []uint8, want [][]byte) {
	t.Helper()
	dibits = make([]uint8, 400)
	for i := range dibits {
		dibits[i] = uint8(i % 4)
	}
	frame := seed0
	for l := 0; l < ldus; l++ {
		var voice [phase1.LDUVoiceSubframeCount][]byte
		for s := range voice {
			// Build the real on-air burst (per-vector FEC + §7.4
			// scramble + §7.5 interleave) so the composer's voice
			// chain decodes it through the deinterleave just like a
			// real LDU off the air.
			onAir, err := imbe.EncodeFrameToChannel(p25p1VoiceInfo(frame))
			frame++
			if err != nil {
				t.Fatalf("EncodeFrameToChannel: %v", err)
			}
			voice[s] = onAir
			wf, _, _ := imbe.DecodeChannelToFrame(onAir)
			want = append(want, wf)
		}
		var lces [phase1.LDULCESBlockCount][]byte
		var lsd [phase1.LDULSDBlockCount][]byte
		ldu, err := phase1.AssembleLDU(0x123, phase1.DUIDLogicalLink1, voice, lces, lsd)
		if err != nil {
			t.Fatalf("AssembleLDU: %v", err)
		}
		dibits = append(dibits, framing.BitsToDibits(ldu)...)
	}
	return dibits, want
}

// TestComposerP25Phase1VoiceChainExtractsRawFrames drives the full
// composer Phase 1 voice path — modulated C4FM IQ → Phase 1 receiver →
// LDU assembler → IMBE voice-frame extraction → recorder .raw sidecar —
// and confirms the recovered frames round-trip to the modulated ones.
func TestComposerP25Phase1VoiceChainExtractsRawFrames(t *testing.T) {
	const (
		sampleRate = 48_000.0
		deviation  = 1800.0
		ldus       = 12
	)
	framesPerLDU := phase1.LDUVoiceSubframeCount

	dibits, want := buildP25P1VoiceStream(t, ldus)
	iq := demod.ModulateP25C4FM(dibits, sampleRate, deviation)

	src := newFakeSource()
	bus := events.NewBus(8)
	sink := &recordingSink{}
	eng := &fakeEngine{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          sink,
		Engine:        eng,
		IQSampleRate:  uint32(sampleRate),
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()
	defer bus.Close()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "P25P1Site", Protocol: "p25",
				GroupID: 42, FrequencyHz: 851_000_000,
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})

	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	waitFor(t, 6*time.Second, func() bool {
		return len(sink.rawFrames("VOICE-1")) >= 6*framesPerLDU
	})

	got := sink.rawFrames("VOICE-1")
	for _, f := range got {
		if len(f) != imbe.FrameBytes {
			t.Fatalf("raw frame length = %d, want %d", len(f), imbe.FrameBytes)
		}
	}

	// At least one full LDU's worth of IMBE frames must round-trip to
	// the modulated frames, proving the receiver → LDU assembler →
	// voice-extraction chain is wired correctly through live IQ. (The
	// C4FM demod is not bit-exact, so a stricter all-frames check
	// belongs in the receiver's own unit tests, not this wiring test.)
	matches := bestAlignmentMatches(got, want)
	if matches < framesPerLDU {
		t.Errorf("only %d of %d captured frames round-tripped; want at least one LDU (%d)",
			matches, len(got), framesPerLDU)
	}

	waitFor(t, time.Second, func() bool { return eng.touched.Load() > 0 })
}

// TestComposerP25Phase1VoiceChainDecodesWidebandIQ drives the voice
// chain on the PHYSICAL-SDR path — genuine wideband C4FM IQ that the
// chain must decimate itself before the receiver runs. This is the
// configuration in the field report where a voice grant on a single
// RTL-SDR started dropping live IQ ("consumer can't keep up"): the old
// front end ran an 81-tap FIR over every input sample and threw away all
// but every decim-th output. The chain now decimates with the same
// filter but convolves only at the kept positions. This test guards that
// the change preserves decode through real wideband IQ — a path the rest
// of the suite (all 48 kHz / pass-through) never covers.
func TestComposerP25Phase1VoiceChainDecodesWidebandIQ(t *testing.T) {
	const (
		// A wideband rate that forces a genuine decimation (decim = 5)
		// without the multi-second modulation cost of synthesising the full
		// 2.4 MS/s stream (sps=500). The field report's 2.4 MS/s (decim=50)
		// runs the SAME decimating-FIR path; the byte-for-byte equivalence
		// to the old front end at 2.4 MS/s is pinned separately and cheaply
		// in TestDecimatingFIRMatchesOldFrontEnd.
		sampleRate = 240_000.0
		deviation  = 1800.0
		ldus       = 12
	)
	framesPerLDU := phase1.LDUVoiceSubframeCount

	dibits, want := buildP25P1VoiceStream(t, ldus)
	// Modulate at the wideband rate so the chain sees real C4FM it must
	// decimate itself (not pre-channelised 48 kHz like the other tests).
	iq := demod.ModulateP25C4FM(dibits, sampleRate, deviation)

	src := newFakeSource()
	bus := events.NewBus(8)
	sink := &recordingSink{}
	eng := &fakeEngine{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          sink,
		Engine:        eng,
		IQSampleRate:  uint32(sampleRate),
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()
	defer bus.Close()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "P25P1Site", Protocol: "p25",
				GroupID: 42, FrequencyHz: 851_000_000,
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})

	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	waitFor(t, 8*time.Second, func() bool {
		return len(sink.rawFrames("VOICE-1")) >= 6*framesPerLDU
	})

	got := sink.rawFrames("VOICE-1")
	for _, f := range got {
		if len(f) != imbe.FrameBytes {
			t.Fatalf("raw frame length = %d, want %d", len(f), imbe.FrameBytes)
		}
	}
	// At least one full LDU of IMBE frames must round-trip through the
	// polyphase-decimated wideband path, same bar as the 48 kHz test.
	matches := bestAlignmentMatches(got, want)
	if matches < framesPerLDU {
		t.Errorf("wideband path: only %d of %d captured frames round-tripped; want at least one LDU (%d)",
			matches, len(got), framesPerLDU)
	}
}

// TestP25P1VoiceFrontEndRejectsAdjacentChannel pins the channel selectivity
// of the wideband voice tap's front end: an adjacent P25 channel centred at
// ±12.5 kHz must be strongly rejected before the receiver's FM discriminator.
// The old pass-through front end (decim==1) applied NO filter, so a ±12.5 kHz
// neighbour reached the discriminator at full amplitude and the capture effect
// decoded it as a foreign talkgroup — the root cause of short/garbled wideband
// recordings. Fails first: pass-through leaves the neighbour at ~0 dB.
func TestP25P1VoiceFrontEndRejectsAdjacentChannel(t *testing.T) {
	const (
		fs        = float64(p25p1VoiceIntermediateHz) // 48 kHz wideband tap output → decim==1
		bw        = uint32(12_500)                    // default operator VoiceBandwidthHz
		adjOffset = 12_500.0                          // adjacent P25 channel spacing
		nSamples  = 24_000
	)
	// Amplitude at DC (wanted channel centre) vs at +12.5 kHz (adjacent centre).
	// A fresh front end per tone keeps the filter history from carrying over.
	respAt := func(offsetHz float64) float64 {
		fe := newP25P1VoiceFrontEnd(fs, bw)
		in := make([]complex64, nSamples)
		for i := range in {
			th := 2 * math.Pi * offsetHz * float64(i) / fs
			in[i] = complex(float32(math.Cos(th)), float32(math.Sin(th)))
		}
		out := fe.Process(nil, in)
		// Peak magnitude over the settled tail (skip the filter warm-up).
		var peak float64
		for _, s := range out[len(out)/2:] {
			if m := math.Hypot(float64(real(s)), float64(imag(s))); m > peak {
				peak = m
			}
		}
		return peak
	}

	wanted := respAt(0)
	neighbour := respAt(adjOffset)
	if wanted < 0.5 {
		t.Fatalf("wanted channel (DC) attenuated to %.3f; front end should pass it flat", wanted)
	}
	rejectionDB := 20 * math.Log10(neighbour/wanted)
	if rejectionDB > -40 {
		t.Errorf("adjacent channel (+%.0f Hz) rejection = %.1f dB; want <= -40 dB "+
			"(a ±12.5 kHz neighbour leaks into the voice tap and the FM discriminator "+
			"captures it during the wanted talker's gaps)", adjOffset, rejectionDB)
	}
}

// TestComposerP25Phase1VoiceChainSurvivesAdjacentChannel is the end-to-end
// regression for the wideband truncation/quality report: a wanted call at DC
// with a STRONGER adjacent call at +12.5 kHz present. Without a channel-select
// filter the FM discriminator is captured by the louder neighbour and the
// wanted call's frames are lost (short/garbled recording). With the filter the
// neighbour is rejected and the wanted call decodes cleanly. Fails first.
func TestComposerP25Phase1VoiceChainSurvivesAdjacentChannel(t *testing.T) {
	const (
		sampleRate = float64(p25p1VoiceIntermediateHz) // 48 kHz: wideband tap, decim==1
		deviation  = 1800.0
		ldus       = 12
		adjOffset  = 12_500.0
	)
	framesPerLDU := phase1.LDUVoiceSubframeCount

	// Wanted call (TG 42) at DC.
	dibitsW, wantW := buildP25P1VoiceStreamSeed(t, ldus, 0)
	iq := demod.ModulateP25C4FM(dibitsW, sampleRate, deviation)

	// A DISTINCT, LOUDER neighbour on the adjacent channel (+12.5 kHz). At 2×
	// amplitude it would win the FM capture and be decoded in place of the
	// wanted call if it reached the discriminator unfiltered.
	dibitsN, wantN := buildP25P1VoiceStreamSeed(t, ldus, 500_000)
	iqN := demod.ModulateP25C4FM(dibitsN, sampleRate, deviation)
	for i := range iq {
		if i >= len(iqN) {
			break
		}
		th := 2 * math.Pi * adjOffset * float64(i) / sampleRate
		rot := complex(float32(math.Cos(th)), float32(math.Sin(th)))
		iq[i] += 2 * iqN[i] * rot
	}

	src := newFakeSource()
	bus := events.NewBus(8)
	sink := &recordingSink{}
	eng := &fakeEngine{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          sink,
		Engine:        eng,
		IQSampleRate:  uint32(sampleRate),
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()
	defer bus.Close()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "P25P1Site", Protocol: "p25",
				GroupID: 42, FrequencyHz: 851_000_000,
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})

	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	waitFor(t, 6*time.Second, func() bool {
		return len(sink.rawFrames("VOICE-1")) >= 6*framesPerLDU
	})
	got := sink.rawFrames("VOICE-1")

	matchW := bestAlignmentMatches(got, wantW)
	matchN := bestAlignmentMatches(got, wantN)
	if matchW < framesPerLDU {
		t.Errorf("wanted call recovered only %d frames with a strong +%.0f Hz neighbour present; "+
			"want >= %d (adjacent channel leaked into the tap and captured the demod)",
			matchW, adjOffset, framesPerLDU)
	}
	if matchN >= framesPerLDU {
		t.Errorf("adjacent-channel neighbour leaked: %d of its frames decoded; "+
			"the channel-select filter should reject the +%.0f Hz channel", matchN, adjOffset)
	}
}

// BenchmarkP25P1VoiceFrontEnd2p4MS measures the new decimating-FIR front
// end at the reporter's 2.4 MS/s rate, and BenchmarkP25P1VoiceFrontEndNaive2p4MS
// the old filter-everything-then-stride front end it replaced, so the
// per-chunk cost reduction is visible in one `go test -bench` run (mirrors
// BenchmarkDDCBankProcess10MHz4Taps for issue #764). Both use the same
// 81-tap filter; the naive baseline runs it over every input sample then
// keeps 1 of 50, the new path convolves only at the kept positions.
func BenchmarkP25P1VoiceFrontEnd2p4MS(b *testing.B) {
	const inRate = 2_400_000.0
	fe := newDecimatingFIR(inRate, p25p1VoiceIntermediateHz, 12_500, false)
	chunk := benchTone(8192)
	var dst []complex64
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst = fe.Process(dst, chunk)
	}
}

func BenchmarkP25P1VoiceFrontEndNaive2p4MS(b *testing.B) {
	const inRate = 2_400_000.0
	decim := int(inRate) / p25p1VoiceIntermediateHz
	cutoff := 12_500.0 / inRate
	lpf := filter.NewFIR(filter.LowpassKaiser(81, cutoff, 8.6))
	chunk := benchTone(8192)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = decimateComplex(lpf.Process(nil, chunk), decim)
	}
}

// benchTone builds a deterministic complex tone buffer for the front-end
// benchmarks. Avoids math/rand and Date-dependent state.
func benchTone(n int) []complex64 {
	out := make([]complex64, n)
	for i := range out {
		p := float64(i) * 0.017
		out[i] = complex(float32(math.Cos(p)), float32(math.Sin(p)))
	}
	return out
}

// TestComposerP25Phase1VoiceChainHonoursDemodMode confirms a CallStart
// whose Grant carries P25Phase1DemodMode="cqpsk" routes the voice IQ
// through the linear-CQPSK / LSM path. Before the fix the voice chain
// hardcoded C4FM regardless of the system-level setting, so on a
// simulcast site the control channel decoded fine but every voice
// grant landed in an FM-discriminator that couldn't sync on
// LSM-modulated dibits, the LDU sink never fired, no frames advanced
// the activity counter, and the watchdog reaped the call at
// call_timeout_ms with an empty WAV. Issue #356 follow-up.
func TestComposerP25Phase1VoiceChainHonoursDemodMode(t *testing.T) {
	const (
		sampleRate = 48_000.0
		sps        = 10 // 48 kHz / 4800 baud
		ldus       = 12
	)
	framesPerLDU := phase1.LDUVoiceSubframeCount

	dibits, want := buildP25P1VoiceStream(t, ldus)
	iq := dibitsToLSMIQTest(dibits, sps, p25p1rx.PulseSpanSymbols, p25p1rx.RolloffAlpha)

	src := newFakeSource()
	bus := events.NewBus(8)
	sink := &recordingSink{}
	eng := &fakeEngine{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          sink,
		Engine:        eng,
		IQSampleRate:  uint32(sampleRate),
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()
	defer bus.Close()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "Simulcast-P25", Protocol: "p25",
				GroupID: 42, FrequencyHz: 851_000_000,
				P25Phase1DemodMode: "cqpsk",
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})

	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	waitFor(t, 6*time.Second, func() bool {
		return len(sink.rawFrames("VOICE-1")) >= 6*framesPerLDU
	})

	got := sink.rawFrames("VOICE-1")
	for _, f := range got {
		if len(f) != imbe.FrameBytes {
			t.Fatalf("raw frame length = %d, want %d", len(f), imbe.FrameBytes)
		}
	}

	// Round-trip parity: at least one LDU of IMBE frames recovered from
	// the LSM IQ matches what was modulated — proves the CQPSK path is
	// the one actually decoding, not a coincidental C4FM lock on the
	// LSM signal (which would not produce parity-valid LDUs at all).
	matches := bestAlignmentMatches(got, want)
	if matches < framesPerLDU {
		t.Errorf("cqpsk path: only %d of %d captured frames round-tripped; want at least one LDU (%d)",
			matches, len(got), framesPerLDU)
	}
}

// TestComposerResolveP25Phase1DemodMode is the unit-level guard for
// the wiring: it asserts the composer maps the grant-carried system
// string to the matching p25p1rx.DemodMode and warn-logs on unknown
// values. Combined with the end-to-end smoke test above (which proves
// the resolved mode actually drives the receiver), a regression that
// re-hardcodes C4FM or drops the parse step would break both.
func TestComposerResolveP25Phase1DemodMode(t *testing.T) {
	c := &Composer{log: discardLogger()}
	cases := []struct {
		in   string
		want p25p1rx.DemodMode
	}{
		{"", p25p1rx.DemodC4FM},
		{"c4fm", p25p1rx.DemodC4FM},
		{"C4FM", p25p1rx.DemodC4FM},
		{"fm", p25p1rx.DemodC4FM},
		{"cqpsk", p25p1rx.DemodCQPSK},
		{"CQPSK", p25p1rx.DemodCQPSK},
		{"lsm", p25p1rx.DemodCQPSK},
		{"linear", p25p1rx.DemodCQPSK},
		{"bogus", p25p1rx.DemodC4FM}, // fallback path, warn-logged
	}
	for _, tc := range cases {
		if got := c.resolveP25Phase1DemodMode("VOICE-X", tc.in); got != tc.want {
			t.Errorf("resolveP25Phase1DemodMode(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// dibitsToLSMIQTest synthesises an LSM IQ stream from a canonical-TIA
// dibit sequence — the inverse of the receiver's LSM dibit remap
// (0,1,3,2) feeds a π/4-DQPSK modulator at π/4 rotation. Copied from
// the receiver package's test helper so the composer test doesn't have
// to import internal symbols.
func dibitsToLSMIQTest(dibits []uint8, sps, span int, alpha float64) []complex64 {
	var lsmDibitRemap = [4]uint8{0, 1, 3, 2}
	var inv [4]uint8
	for i, m := range lsmDibitRemap {
		inv[m] = uint8(i)
	}
	pre := make([]uint8, len(dibits))
	for i, d := range dibits {
		pre[i] = inv[d&3]
	}
	return demod.ModulatePiOver4DQPSK(pre, sps, span, alpha, math.Pi/4)
}

// TestComposerP25Phase1TouchGatedOnLDUProgress reproduces the
// regression behind issue #356: once LDU delivery stops (e.g. the
// transmitting unit sent a TDU, the carrier dropped, or the C4FM
// receiver lost lock on simulcast garbage), the chain MUST stop
// touching the engine so the trunking watchdog can fire and release
// the voice SDR. Before the fix the chain emitted a 1 s heartbeat
// unconditionally and the active call lived forever.
func TestComposerP25Phase1TouchGatedOnLDUProgress(t *testing.T) {
	const (
		sampleRate = 48_000.0
		deviation  = 1800.0
		ldus       = 6
	)
	dibits, _ := buildP25P1VoiceStream(t, ldus)
	iq := demod.ModulateP25C4FM(dibits, sampleRate, deviation)

	src := newFakeSource()
	bus := events.NewBus(8)
	sink := &recordingSink{}
	eng := &fakeEngine{}
	c, err := New(Options{
		Bus:           bus,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          sink,
		Engine:        eng,
		IQSampleRate:  uint32(sampleRate),
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()
	defer bus.Close()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "P25P1Site", Protocol: "p25",
				GroupID: 42, FrequencyHz: 851_000_000,
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})
	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })

	// Feed IQ → LDUs decode → frame counter advances → Touch fires.
	src.SendIQ(iq)
	waitFor(t, 6*time.Second, func() bool { return eng.touched.Load() >= 1 })

	// Stop feeding IQ. The chain must stop touching after the last LDU.
	// 10× TouchInterval (300 ms) is well past any in-flight LDU.
	time.Sleep(300 * time.Millisecond)
	baseline := eng.touched.Load()
	time.Sleep(300 * time.Millisecond)
	if got := eng.touched.Load(); got != baseline {
		t.Fatalf("expected no further touches after IQ stopped; baseline=%d got=%d", baseline, got)
	}
}

// TestComposerP25Phase1VoiceChainLogsDemodMode is the operator-visible
// diagnostic guard for issue #356: the CC pipeline logs the demod mode
// it locked the control channel with, but until this log was added the
// voice chain was silent — so on a field report it was impossible to
// tell from logs whether voice grants were running c4fm or cqpsk. The
// chain now emits one Info line at startup naming the resolved mode.
func TestComposerP25Phase1VoiceChainLogsDemodMode(t *testing.T) {
	const sampleRate = 48_000.0
	buf := &syncBuffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	src := newFakeSource()
	bus := events.NewBus(8)
	sink := &recordingSink{}
	eng := &fakeEngine{}
	c, err := New(Options{
		Bus:           bus,
		Log:           logger,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          sink,
		Engine:        eng,
		IQSampleRate:  uint32(sampleRate),
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()
	defer bus.Close()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "Simulcast-P25", Protocol: "p25",
				GroupID: 42, FrequencyHz: 851_000_000,
				P25Phase1DemodMode: "cqpsk",
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})

	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), "composer: p25p1 voice chain started") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	out := buf.String()
	if !strings.Contains(out, "composer: p25p1 voice chain started") {
		t.Fatalf("expected p25p1 startup log; got:\n%s", out)
	}
	if !strings.Contains(out, "demod_mode=cqpsk") {
		t.Errorf("expected demod_mode=cqpsk in log; got:\n%s", out)
	}
}

// TestComposerP25Phase1VoiceChainLogsDecodeQuality drives enough live
// LDUs through the chain to trip the rolling decode-quality summary and
// asserts the line is emitted with the per-call counters. This is the
// metric a field operator watches to tell whether raising the voice SDR
// gain reduces uncorrectable subframes behind garbled audio — issue
// #356 follow-up.
func TestComposerP25Phase1VoiceChainLogsDecodeQuality(t *testing.T) {
	const (
		sampleRate = 48_000.0
		deviation  = 1800.0
		ldus       = 40 // well past the 25-LDU summary threshold
	)
	dibits, _ := buildP25P1VoiceStream(t, ldus)
	iq := demod.ModulateP25C4FM(dibits, sampleRate, deviation)

	buf := &syncBuffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	src := newFakeSource()
	bus := events.NewBus(8)
	sink := &recordingSink{}
	eng := &fakeEngine{}
	c, err := New(Options{
		Bus:           bus,
		Log:           logger,
		Devices:       &fakeDevices{src: map[string]IQSource{"VOICE-1": src}},
		Sink:          sink,
		Engine:        eng,
		IQSampleRate:  uint32(sampleRate),
		PCMSampleRate: 8000,
		TouchInterval: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.Run(ctx)
	defer c.Close()
	defer bus.Close()

	bus.Publish(events.Event{
		Kind: events.KindCallStart,
		Payload: trunking.CallStart{
			Grant: trunking.Grant{
				System: "P25P1Site", Protocol: "p25",
				GroupID: 42, FrequencyHz: 851_000_000,
			},
			DeviceSerial: "VOICE-1",
			StartedAt:    time.Now().UTC(),
		},
	})

	waitFor(t, 2*time.Second, func() bool { return len(c.ActiveChains()) == 1 })
	src.SendIQ(iq)

	waitFor(t, 6*time.Second, func() bool {
		return strings.Contains(buf.String(), "composer: p25p1 decode quality")
	})
	out := buf.String()
	if !strings.Contains(out, "composer: p25p1 decode quality") {
		t.Fatalf("expected p25p1 decode quality log; got:\n%s", out)
	}
	if !strings.Contains(out, "ldus=") || !strings.Contains(out, "uncorrectable_ldus=") {
		t.Errorf("expected ldus= and uncorrectable_ldus= fields in decode quality log; got:\n%s", out)
	}
	// gated_ldus is the truncation metric: how many delivered LDUs the
	// talkgroup gate dropped from the WAV. It must be surfaced so a field
	// operator can see a short recording's cause without re-deriving it.
	if !strings.Contains(out, "gated_ldus=") {
		t.Errorf("expected gated_ldus= field in decode quality log; got:\n%s", out)
	}
}

// syncBuffer is a bytes.Buffer guarded by a mutex so the test
// goroutine can read the captured log output while the composer
// chain goroutine concurrently writes to it via slog's handler.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
