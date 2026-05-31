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
	t.Helper()
	dibits = make([]uint8, 400)
	for i := range dibits {
		dibits[i] = uint8(i % 4)
	}
	frame := 0
	for l := 0; l < ldus; l++ {
		var voice [phase1.LDUVoiceSubframeCount][]byte
		for s := range voice {
			ch, err := imbe.EncodeChannel(p25p1VoiceInfo(frame))
			frame++
			if err != nil {
				t.Fatalf("EncodeChannel: %v", err)
			}
			onAir, err := imbe.Scramble(ch)
			if err != nil {
				t.Fatalf("Scramble: %v", err)
			}
			// Scramble/Descramble mutate in place — keep an independent
			// copy for the LDU so the want computation cannot corrupt it.
			voice[s] = append([]byte(nil), onAir...)
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
