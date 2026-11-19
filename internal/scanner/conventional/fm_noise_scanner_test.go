package conventional

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// The scan-list side of the FM noise squelch (issue #1329): what the
// reporter's PlutoSky run did, replayed from real receiver noise and a real
// NFM transmission at the scanner's own rate.

// scriptedIQ plays one stream through every StreamIQ call in turn (the
// scanner reopens its stream on each visit), one chunk a tick, and replays
// from loopFrom when the script runs out, so the stream never falls silent.
// The channel is unbuffered: taken() is exactly how many chunks the scanner
// has received.
type scriptedIQ struct {
	mu       sync.Mutex
	chunks   [][]complex64
	next     int
	loopFrom int
	sent     int
}

func (f *scriptedIQ) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	out := make(chan []complex64)
	go func() {
		defer close(out)
		tick := time.NewTicker(2 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
			f.mu.Lock()
			if f.next >= len(f.chunks) {
				f.next = f.loopFrom
			}
			chunk := f.chunks[f.next]
			f.mu.Unlock()
			select {
			case <-ctx.Done():
				return
			case out <- chunk:
				f.mu.Lock()
				f.next++
				f.sent++
				f.mu.Unlock()
			}
		}
	}()
	return out, nil
}

func (f *scriptedIQ) taken() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sent
}

// posEngine is a fakeEngine that notes how far into the script each call
// started and ended.
type posEngine struct {
	fakeEngine
	src                *scriptedIQ
	startedAt, endedAt []int
}

func (e *posEngine) HandleSyntheticCall(g trunking.Grant, serial string) {
	at := e.src.taken()
	e.fakeEngine.HandleSyntheticCall(g, serial)
	e.mu.Lock()
	e.startedAt = append(e.startedAt, at)
	e.mu.Unlock()
}

func (e *posEngine) EndSyntheticCall(serial string, reason trunking.EndReason) bool {
	at := e.src.taken()
	e.mu.Lock()
	e.endedAt = append(e.endedAt, at)
	e.mu.Unlock()
	return e.fakeEngine.EndSyntheticCall(serial, reason)
}

func (e *posEngine) marks() (starts, ends []int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]int(nil), e.startedAt...), append([]int(nil), e.endedAt...)
}

const noiseScanChunk = 8192 // 3.4 ms at 2.4 MS/s

func noiseChunks(iq []complex64) [][]complex64 {
	var out [][]complex64
	for s := 0; s+noiseScanChunk <= len(iq); s += noiseScanChunk {
		out = append(out, iq[s:s+noiseScanChunk])
	}
	return out
}

// agcNoise is real RTL-SDR noise (an empty stretch of the 2.4 MS/s capture)
// raised 15.6 dB, to the -37 dBFS in the channel that the reporter's
// PlutoSky held it at under its AGC (-12 dBFS across 4.8 MHz, 17 Nov log):
// 23 dB over the -60 dBFS squelch they had been given.
func agcNoise(t testing.TB, offsetHz float64) []complex64 {
	return shiftHz(loadCF32(t, mmrCapture), offsetHz, mmrRate, 6)
}

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func newNoiseTestScanner(t *testing.T, ch Channel, src *scriptedIQ, eng Engine, logTo *syncBuffer) *Scanner {
	t.Helper()
	s, err := New(Options{
		Log:   slog.New(slog.NewTextHandler(logTo, nil)),
		Tuner: &fakeTuner{}, IQ: src, Engine: eng, Recorder: fakeRecorder{},
		DeviceSerial: "PLUTO", SystemName: "scanner",
		Channels:     []Channel{ch},
		SampleRateHz: mmrRate,
		// Long enough that a slow test machine still gets a measurement
		// window of stream into every visit.
		MinDwellPerChannel: 400 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// TestScannerNoiseAboveSquelchOpensNoCall is the #1329 regression. The
// reporter's scan list (PMR446, squelch_dbfs -60) opened a call 100 ms after
// start-up on an empty channel and recorded noise for 33 and 41 minutes:
// the board's noise sat above the threshold, and the level was the FM
// squelch's only test.
//
// Real receiver noise at that level must open nothing, however long it
// lasts. The scanner says once what it is seeing, and reports the channel
// as noise (quieting about 0 dB) with its level. With the noise squelch
// switched off the same stream opens a call at once: the old behaviour.
func TestScannerNoiseAboveSquelchOpensNoCall(t *testing.T) {
	noise := noiseChunks(agcNoise(t, -500_000))
	pmr := Channel{Label: "PMR446 ch1", FrequencyHz: 446_006_250, Mode: "nfm", SquelchDbFS: -60}

	t.Run("noise squelch", func(t *testing.T) {
		src := &scriptedIQ{chunks: noise}
		eng := &posEngine{src: src}
		var logs syncBuffer
		s := newNoiseTestScanner(t, pmr, src, eng, &logs)
		ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
		defer cancel()
		_ = s.Run(ctx)

		if n := eng.startCount(); n != 0 {
			t.Fatalf("%d calls opened on receiver noise above squelch_dbfs, want none", n)
		}
		// The level alone opens on the first chunks; 50 is the test having
		// run at all, on however slow a machine.
		if got := src.taken(); got < 50 {
			t.Fatalf("the scanner only read %d chunks; the test did not run", got)
		}
		st := s.Snapshot().Channels[0]
		if st.LevelDbFS == nil || *st.LevelDbFS < -40 || *st.LevelDbFS > -34 {
			t.Errorf("level_dbfs = %v, want about -37 (the noise, 23 dB over the squelch)", st.LevelDbFS)
		}
		if st.QuietingDb == nil || *st.QuietingDb < -3 || *st.QuietingDb > 3 {
			t.Errorf("quieting_db = %v, want about 0 on noise", st.QuietingDb)
		}
		out := logs.String()
		if n := strings.Count(out, "above squelch_dbfs with no carrier"); n != 1 {
			t.Errorf("the noise note was logged %d times, want once:\n%s", n, out)
		}
		if !strings.Contains(out, "squelch_dbfs=-60") || !strings.Contains(out, "level_dbfs=-3") {
			t.Errorf("the note does not carry the level and the threshold:\n%s", out)
		}
	})

	t.Run("level squelch only", func(t *testing.T) {
		off := pmr
		off.NoiseSquelchOff = true
		src := &scriptedIQ{chunks: noise}
		eng := &posEngine{src: src}
		var logs syncBuffer
		s := newNoiseTestScanner(t, off, src, eng, &logs)
		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()
		_ = s.Run(ctx)

		starts, _ := eng.marks()
		if len(starts) == 0 || starts[0] > 3 {
			t.Fatalf("with the noise squelch off the level alone must open on this noise at once (the old behaviour); calls started at chunks %v", starts)
		}
		if n := eng.normalEndCount(); n != 0 {
			t.Errorf("the call on noise ended on its own %d times; on the level alone it never should", n)
		}
		if !strings.Contains(logs.String(), "noise_squelch=off") {
			t.Errorf("the open line does not say the noise squelch is off:\n%s", logs.String())
		}
	})
}

// TestScannerNoiseSquelchFollowsARealTransmission plays what an AGC rig
// hears around one transmission: noise above squelch_dbfs, the real NFM
// handheld, the same noise again. The call must open on the transmission
// (not on the noise ahead of it), within the squelch's latency, and end at
// hangtime after the transmitter drops, although the channel never falls
// under squelch_dbfs. On the level alone it opens on the first chunk of
// noise and is still open when the test stops looking.
func TestScannerNoiseSquelchFollowsARealTransmission(t *testing.T) {
	lead := noiseChunks(agcNoise(t, -500_000)[:mmrRate*4/10])
	voice := noiseChunks(loadRealAirAt2400k(t, "ctcss_none_radtel_447100k_48k.cs16"))
	tail := noiseChunks(agcNoise(t, 300_000))
	script := append(append(append([][]complex64(nil), lead...), voice...), tail...)
	txFirst, txEnd := len(lead), len(lead)+len(voice)
	ch := Channel{Label: "PMR446 ch1", FrequencyHz: 446_006_250, Mode: "nfm", SquelchDbFS: -60,
		Hangtime: 300 * time.Millisecond}

	run := func(t *testing.T, ch Channel) (*posEngine, string) {
		src := &scriptedIQ{chunks: script, loopFrom: txEnd}
		eng := &posEngine{src: src}
		var logs syncBuffer
		s := newNoiseTestScanner(t, ch, src, eng, &logs)
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() { _ = s.Run(ctx); close(done) }()
		// Until the call has ended on its own and the noise after it has
		// had a full second to reopen the channel, or 6 s.
		deadline := time.Now().Add(6 * time.Second)
		var endedAt time.Time
		for time.Now().Before(deadline) {
			if eng.normalEndCount() > 0 && endedAt.IsZero() {
				endedAt = time.Now()
			}
			if !endedAt.IsZero() && time.Since(endedAt) > time.Second {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		cancel()
		<-done
		return eng, logs.String()
	}

	t.Run("noise squelch", func(t *testing.T) {
		eng, logs := run(t, ch)
		starts, ends := eng.marks()
		if len(starts) != 1 {
			t.Fatalf("%d calls opened (at chunks %v), want exactly one: the transmission (chunks %d..%d)", len(starts), starts, txFirst, txEnd)
		}
		// The squelch reports the carrier fmNoiseLatency after it arrives:
		// 25 ms is 8 chunks. Twice that, because the scanner may start a
		// new visit (and the meter a new window) just as the carrier comes
		// up, and where that falls depends on how fast this machine runs.
		if at := starts[0]; at <= txFirst || at > txFirst+20 {
			t.Errorf("the call opened at chunk %d, want within 20 chunks after the transmission began at %d", at, txFirst)
		}
		if eng.normalEndCount() != 1 {
			t.Fatalf("the call did not end on its own after the transmitter dropped (end reasons %v)", eng.endReasonsCopy())
		}
		if at := ends[0]; at < txEnd {
			t.Errorf("the call ended at chunk %d, before the transmission did (%d)", at, txEnd)
		}
		if !strings.Contains(logs, "conv: squelch opened") || !strings.Contains(logs, "quieting_db=") {
			t.Errorf("no squelch-opened line with the quieting it measured:\n%s", logs)
		}
	})

	t.Run("level squelch only", func(t *testing.T) {
		off := ch
		off.NoiseSquelchOff = true
		eng, _ := run(t, off)
		starts, _ := eng.marks()
		if len(starts) == 0 || starts[0] >= txFirst {
			t.Fatalf("on the level alone the call must open on the noise ahead of the transmission (the old behaviour); calls started at chunks %v, transmission at %d", starts, txFirst)
		}
		if n := eng.normalEndCount(); n != 0 {
			t.Errorf("on the level alone the call ended on its own %d times; the noise holds it open", n)
		}
	})
}

// A stream that starts late in a short scan window still opens: a channel
// that cleared squelch_dbfs before the noise squelch's first reading gets
// one extension to make it. Here the window is 10 ms, the first chunk
// arrives at once and the rest only after the window has run out; without
// the extension the scanner has left by then and a carrier never opens the
// channel. (A 48 kHz stream, so the arithmetic costs nothing and the timing
// is the test's own.)
func TestScannerNoiseSquelchExtendsAShortWindowOnce(t *testing.T) {
	voice := readCS16Fixture(t, radtelNoTone)
	const chunk = 164
	src := &scriptedIQ{chunks: [][]complex64{voice[:chunk]}}
	var logs syncBuffer
	s, err := New(Options{
		Log:   slog.New(slog.NewTextHandler(&logs, nil)),
		Tuner: &fakeTuner{}, IQ: src, Engine: &fakeEngine{}, Recorder: fakeRecorder{},
		DeviceSerial: "PLUTO", Channels: []Channel{{Label: "A", FrequencyHz: 446_006_250, SquelchDbFS: -60}},
		SampleRateHz:       48_000,
		MinDwellPerChannel: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	stream := make(chan []complex64)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for i := 0; i+chunk <= len(voice); i += chunk {
			select {
			case stream <- voice[i : i+chunk]:
			case <-ctx.Done():
				return
			}
			if i == 0 {
				time.Sleep(15 * time.Millisecond) // past the 10 ms window
			}
		}
	}()
	if !s.scanWindow(ctx, 0, s.channels[0], stream, nil) {
		t.Fatal("a carrier did not open the channel: the 10 ms window ended before the noise squelch's first reading and was not extended")
	}
}

// The onset mark a counted source gets is placed the squelch's latency
// before the chunk that opened it, so the voice chain starts where the
// carrier came up, never before the stream's first sample, and exactly on
// the chunk for a channel without the noise squelch.
func TestStreamPosBackdatesOnsetByTheSquelchLatency(t *testing.T) {
	lead := uint64(fmNoiseLatency.Seconds() * mmrRate)
	p := &streamPos{next: 1_000_000, first: 1_000_000, gap: 240_000, lead: lead}
	p.advance(8192, false)
	p.advance(8192, false) // 16384 samples of noise: less than the lead
	p.advance(8192, true)
	if p.onset != 1_000_000 {
		t.Errorf("onset %d, want the stream's first sample 1000000 (the lead reaches back past it)", p.onset)
	}
	p = &streamPos{next: 1_000_000, first: 1_000_000, gap: 240_000, lead: lead}
	for i := 0; i < 100; i++ {
		p.advance(8192, false)
	}
	p.advance(8192, true)
	if want := uint64(1_000_000+100*8192) - lead; p.onset != want {
		t.Errorf("onset %d, want %d (%d samples before the opening chunk)", p.onset, want, lead)
	}
	p = &streamPos{next: 0, gap: 240_000}
	p.advance(8192, false)
	p.advance(8192, true)
	if p.onset != 8192 {
		t.Errorf("onset %d without a lead, want the opening chunk's start 8192", p.onset)
	}
}

// A manual-tune (temporary) channel gets the noise squelch like a configured
// one, and can ask for the level alone.
func TestAddTemporaryChannelNoiseSquelch(t *testing.T) {
	s, err := New(Options{
		Tuner: &fakeTuner{}, IQ: &scriptedIQ{chunks: [][]complex64{make([]complex64, 16)}},
		Engine: &fakeEngine{}, Recorder: fakeRecorder{},
		DeviceSerial: "PLUTO", SampleRateHz: mmrRate,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		ch   Channel
		want bool
	}{
		{Channel{FrequencyHz: 446_006_250}, true},
		{Channel{FrequencyHz: 446_018_750, Mode: "nfm"}, true},
		{Channel{FrequencyHz: 446_031_250, Mode: "nfm", NoiseSquelchOff: true}, false},
		{Channel{FrequencyHz: 126_400_000, Mode: ModeAM}, false},
	} {
		idx := s.AddTemporaryChannel(tc.ch)
		if got := s.noiseSquelchFor(idx) != nil; got != tc.want {
			t.Errorf("temporary channel %+v: noise squelch = %v, want %v", tc.ch, got, tc.want)
		}
	}
}

// A long scan list under an AGC names the first few channels sitting on
// noise and then says once that there are more, not a line per channel.
func TestNoiseAboveSquelchNoteStopsListing(t *testing.T) {
	var logs syncBuffer
	s := &Scanner{log: slog.New(slog.NewTextHandler(&logs, nil))}
	meters := make([]*fmNoiseMeter, 0, 12)
	for i := 0; i < 12; i++ {
		m := newFMNoiseMeter(48_000)
		meters = append(meters, m)
		ch := Channel{Label: "ch", FrequencyHz: uint32(446_006_250 + 12_500*i), SquelchDbFS: -60}
		s.noteNoiseAboveSquelch(ch, m, -37)
		s.noteNoiseAboveSquelch(ch, m, -37) // a second visit says nothing
	}
	out := logs.String()
	if n := strings.Count(out, "above squelch_dbfs with no carrier on it"); n != noiseNoteChannels {
		t.Errorf("%d channels named, want %d:\n%s", n, noiseNoteChannels, out)
	}
	if n := strings.Count(out, "not naming the rest"); n != 1 {
		t.Errorf("the closing line was logged %d times, want once:\n%s", n, out)
	}
}
