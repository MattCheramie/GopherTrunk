package conventional

import (
	"context"
	"encoding/binary"
	"math"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/radio/fleetsync"
	fsafsk "github.com/MattCheramie/GopherTrunk/internal/radio/fleetsync/afsk"
)

// fakeDataDecoder records what the scanner feeds it. busy, when set,
// decides Busy() from the number of chunks processed so far.
type fakeDataDecoder struct {
	mu     sync.Mutex
	chunks int
	resets int
	rate   float64
	busy   func(chunks int) bool
}

func (d *fakeDataDecoder) ProcessIQ(_ []complex64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.chunks++
}

func (d *fakeDataDecoder) Busy() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.busy != nil && d.busy(d.chunks)
}

func (d *fakeDataDecoder) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.resets++
}

func (d *fakeDataDecoder) counts() (chunks, resets int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.chunks, d.resets
}

func fakeFactory(d *fakeDataDecoder) DataDecoderFactory {
	return func(_ Channel, _ string, rateHz float64) (DataDecoder, error) {
		d.rate = rateHz
		return d, nil
	}
}

func newDataTestScanner(t *testing.T, ch Channel, dec *fakeDataDecoder, clock func() time.Time) *Scanner {
	t.Helper()
	tuner := &fakeTuner{}
	s, err := New(Options{
		Tuner: tuner, IQ: &fakeIQ{chunks: map[uint32][][]complex64{}, tuner: tuner},
		Engine: &fakeEngine{}, Recorder: fakeRecorder{},
		DeviceSerial: "CONV-DATA", Channels: []Channel{ch},
		MinDwellPerChannel: 30 * time.Millisecond,
		SampleRateHz:       2_400_000,
		DataDecoders:       fakeFactory(dec),
		Now:                clock,
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestPickDataDecimation(t *testing.T) {
	for _, tc := range []struct {
		in    float64
		m     int
		rateH float64
	}{
		{2_400_000, 50, 48_000},       // RTL-SDR default
		{2_048_000, 40, 51_200},       // 2.048 MS/s: 48 761.9 is not an integer rate
		{3_200_000, 64, 50_000},       // L = 24
		{1_000_000, 20, 50_000},       //
		{48_000, 1, 48_000},           // already at the channel rate
		{2_400_000.5, 1, 2_400_000.5}, // non-integer: no decimation (the factory then sees the raw rate)
	} {
		m := pickDataDecimation(tc.in)
		if m != tc.m || tc.in/float64(m) != tc.rateH {
			t.Errorf("pickDataDecimation(%v) = %d (rate %v), want %d (rate %v)", tc.in, m, tc.in/float64(m), tc.m, tc.rateH)
		}
	}
}

// The decoder is handed the decimated channel rate, never the SDR rate —
// the #1184 CTCSS lesson (a rad/sample threshold read 50x off at 2.4 MS/s).
func TestDataDecoderFactoryGetsChannelRate(t *testing.T) {
	dec := &fakeDataDecoder{}
	newDataTestScanner(t, Channel{Label: "A", FrequencyHz: 462_562_500, SquelchDbFS: -10, Decoders: []string{DecoderFleetSync}}, dec, nil)
	if dec.rate != 48_000 {
		t.Fatalf("factory rate = %v, want 48000 (the decimated channel rate)", dec.rate)
	}
}

// A burst sent at key-up must be seen from its first bit: the chunk that
// opens the squelch is fed to the decoders before the dwell starts.
func TestDataDecoderSeesSquelchBreakingChunk(t *testing.T) {
	dec := &fakeDataDecoder{}
	s := newDataTestScanner(t, Channel{Label: "A", FrequencyHz: 462_562_500, SquelchDbFS: -10, Decoders: []string{DecoderFleetSync}}, dec, nil)
	stream := make(chan []complex64, 1)
	stream <- loudChunk(4096)
	if !s.scanWindow(context.Background(), 0, s.channels[0], stream, nil) {
		t.Fatal("squelch did not break on a loud chunk")
	}
	if n, _ := dec.counts(); n != 1 {
		t.Fatalf("decoder saw %d chunks before the dwell, want 1 (the squelch-breaking chunk)", n)
	}
}

// A decoder mid-burst when the scan window expires holds the channel — at
// most dataScanHoldMax — instead of hopping away mid-frame.
func TestDataDecoderBusyExtendsScanWindow(t *testing.T) {
	run := func(busy bool) time.Duration {
		dec := &fakeDataDecoder{busy: func(int) bool { return busy }}
		s := newDataTestScanner(t, Channel{Label: "A", FrequencyHz: 462_562_500, SquelchDbFS: -10, Decoders: []string{DecoderFleetSync}}, dec, nil)
		stream := make(chan []complex64) // never delivers: the window can only time out
		start := time.Now()
		if s.scanWindow(context.Background(), 0, s.channels[0], stream, nil) {
			t.Fatal("squelch broke with no IQ")
		}
		return time.Since(start)
	}
	idle := run(false)
	held := run(true)
	if idle > 30*time.Millisecond+dataScanHoldMax/2 {
		t.Errorf("idle decoder: window took %v, want ~MinDwellPerChannel (30ms)", idle)
	}
	if held < 30*time.Millisecond+dataScanHoldMax-dataScanHoldStep {
		t.Errorf("busy decoder: window took %v, want it extended by ~%v", held, dataScanHoldMax)
	}
	if held > 30*time.Millisecond+dataScanHoldMax+time.Second {
		t.Errorf("busy decoder: window took %v — the hold must be bounded", held)
	}
}

// While dwelling, a decoder mid-burst counts as channel activity, so
// hangtime cannot end the call part-way through a burst (issue #1220). The
// clock advances 10 ms per silent chunk; hangtime is 100 ms.
func TestDataDecoderBusyHoldsHangtime(t *testing.T) {
	endsAfter := func(busyChunks int) int {
		var clock atomic.Int64
		base := time.Unix(1_700_000_000, 0)
		now := func() time.Time { return base.Add(time.Duration(clock.Load())) }
		dec := &fakeDataDecoder{busy: func(n int) bool { return n <= busyChunks }}
		s := newDataTestScanner(t, Channel{
			Label: "A", FrequencyHz: 462_562_500, SquelchDbFS: -10,
			Hangtime: 100 * time.Millisecond, Decoders: []string{DecoderFleetSync},
		}, dec, now)
		stream := make(chan []complex64)
		streamCtx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			s.beginDwell(0, s.channels[0], stream, streamCtx, cancel)
		}()
		sent := 0
		for {
			clock.Add(int64(10 * time.Millisecond))
			select {
			case stream <- make([]complex64, 256): // silence: below squelch
				sent++
			case <-done:
				return sent
			}
			if sent > 1000 {
				t.Fatal("dwell never ended")
			}
		}
	}
	idle := endsAfter(0)
	held := endsAfter(30)
	if idle > 15 {
		t.Errorf("no burst: dwell ended after %d silent chunks, want ~hangtime (10)", idle)
	}
	if held < 30+10 {
		t.Errorf("burst in progress for 30 chunks: dwell ended after %d chunks, want > 40 (busy + hangtime)", held)
	}
}

// Decoder state is reset on every retune so one channel's filter history
// cannot bleed into the next.
func TestDataDecoderResetOnRetune(t *testing.T) {
	dec := &fakeDataDecoder{}
	s := newDataTestScanner(t, Channel{Label: "A", FrequencyHz: 462_562_500, SquelchDbFS: -10, Decoders: []string{DecoderFleetSync}}, dec, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_ = s.Run(ctx)
	if _, resets := dec.counts(); resets < 2 {
		t.Fatalf("decoder reset %d times over several scan passes, want one per tune", resets)
	}
}

func TestDataDecodersRejectBadNames(t *testing.T) {
	for _, names := range [][]string{{"pocsag"}, {DecoderMDC1200, DecoderMDC1200}} {
		_, err := New(Options{
			Tuner: &fakeTuner{}, IQ: &fakeIQ{}, Engine: &fakeEngine{}, Recorder: fakeRecorder{},
			DeviceSerial: "X", Channels: []Channel{{FrequencyHz: 1, Decoders: names}},
		})
		if err == nil {
			t.Errorf("decoders %v accepted", names)
		}
	}
}

// With no factory the channel still scans; the decoders are just absent.
func TestDataDecodersWithoutFactoryAreDisabled(t *testing.T) {
	s, err := New(Options{
		Tuner: &fakeTuner{}, IQ: &fakeIQ{}, Engine: &fakeEngine{}, Recorder: fakeRecorder{},
		DeviceSerial: "X", SampleRateHz: 2_400_000,
		Channels: []Channel{{FrequencyHz: 1, Decoders: []string{DecoderMDC1200}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.dataFor(0) != nil {
		t.Fatal("decoders built without a factory")
	}
	if got := s.Snapshot().Channels[0].Decoders; len(got) != 1 || got[0] != DecoderMDC1200 {
		t.Fatalf("Snapshot decoders = %v", got)
	}
}

// loadFleetSyncAt2400k reads a 48 kHz real-air FleetSync slice (#1184,
// Fleet 107 / Unit 1772) and interpolates it to the 2.4 MS/s the scanner
// actually delivers, then adds white noise across the WHOLE 2.4 MHz band at
// wbSNRdB relative to the signal. The noise in the ~16 kHz channel is
// ~17 dB below the band total, so a band SNR of 0 dB is a comfortable
// in-channel signal — the regime a real scanner SDR sits in.
func loadFleetSyncAt2400k(t *testing.T, name string, wbSNRdB float64) []complex64 {
	t.Helper()
	b, err := os.ReadFile("../../radio/fleetsync/afsk/testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	iq := make([]complex64, len(b)/4)
	for i := range iq {
		iq[i] = complex(
			float32(int16(binary.LittleEndian.Uint16(b[4*i:])))/32768,
			float32(int16(binary.LittleEndian.Uint16(b[4*i+2:])))/32768)
	}
	x := dsp.NewResampler(50, 1, 16, 8.6).Process(nil, iq)
	var p float64
	for _, v := range x {
		p += float64(real(v)*real(v) + imag(v)*imag(v))
	}
	p /= float64(len(x))
	sd := math.Sqrt(p / math.Pow(10, wbSNRdB/10) / 2)
	r := rand.New(rand.NewSource(1))
	for i := range x {
		x[i] += complex(float32(r.NormFloat64()*sd), float32(r.NormFloat64()*sd))
	}
	return x
}

// countFleetSync runs x through fn in RTL-sized chunks and counts CRC-valid
// Fleet 107 / Unit 1772 bursts.
func countFleetSync(t *testing.T, x []complex64, rateHz float64, pre func([]complex64) []complex64) int {
	t.Helper()
	ok := 0
	rcv, err := fsafsk.New(fsafsk.Options{InputRateHz: uint32(rateHz), OnMessage: func(m fleetsync.Message) {
		if m.CRCOK && m.Fleet == 107 && m.Unit == 1772 {
			ok++
		}
	}})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(x); i += 16384 {
		rcv.ProcessIQ(pre(x[i:min(i+16384, len(x))]))
	}
	return ok
}

// TestDataFrontEndDecodesRealAirFleetSyncAtSDRRate is the #1220 rate pin.
// The dedicated-SDR FleetSync receiver discriminates whatever IQ it is
// given; handed the scanner's raw 2.4 MS/s stream with band-wide noise at
// 0 dB, the FM discriminator is swamped by the 2.4 MHz of noise around the
// channel and nothing decodes (measured: 0/2 bursts on both slices). The
// scanner's data front end decimates to 48 kHz behind a channel filter, and
// the same samples decode every burst.
func TestDataFrontEndDecodesRealAirFleetSyncAtSDRRate(t *testing.T) {
	for _, name := range []string{
		"fleetsync1_fleet107_unit1772_48k.cs16",
		"fleetsync2_fleet107_unit1772_48k.cs16",
	} {
		t.Run(name, func(t *testing.T) {
			x := loadFleetSyncAt2400k(t, name, 0)
			if raw := countFleetSync(t, x, 2_400_000, func(c []complex64) []complex64 { return c }); raw != 0 {
				t.Logf("raw 2.4 MS/s decoded %d bursts (the fixture no longer shows the wideband-noise failure)", raw)
			}
			fe := newDataFrontEnd(2_400_000)
			if got := countFleetSync(t, x, fe.rate, fe.process); got < 2 {
				t.Fatalf("through the scanner's data front end: %d CRC-valid bursts, want >= 2", got)
			}
		})
	}
}

// TestScannerDecodesRealAirFleetSyncOnChannel runs the real-air FS-II slice
// through the whole scanner — squelch break, dwell, data front end and a
// real FleetSync receiver built by the factory — at the SDR rate.
func TestScannerDecodesRealAirFleetSyncOnChannel(t *testing.T) {
	const freq = 462_562_500
	x := loadFleetSyncAt2400k(t, "fleetsync2_fleet107_unit1772_48k.cs16", 0)
	var chunks [][]complex64
	for i := 0; i < len(x); i += 16384 {
		chunks = append(chunks, x[i:min(i+16384, len(x))])
	}
	tuner := &fakeTuner{}
	var mu sync.Mutex
	var got []fleetsync.Message
	s, err := New(Options{
		Tuner: tuner, IQ: &fakeIQ{chunks: map[uint32][][]complex64{freq: chunks}, tuner: tuner},
		Engine: &fakeEngine{}, Recorder: fakeRecorder{}, DeviceSerial: "CONV-FS",
		Channels: []Channel{{
			Label: "Kenwood", FrequencyHz: freq, SquelchDbFS: -60,
			Hangtime: 5 * time.Second, Decoders: []string{DecoderFleetSync},
		}},
		SampleRateHz: 2_400_000,
		DataDecoders: func(ch Channel, kind string, rateHz float64) (DataDecoder, error) {
			return fsafsk.New(fsafsk.Options{InputRateHz: uint32(rateHz), OnMessage: func(m fleetsync.Message) {
				mu.Lock()
				got = append(got, m)
				mu.Unlock()
			}})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go func() { _ = s.Run(ctx) }()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		ok := 0
		for _, m := range got {
			if m.CRCOK && m.IsFS2 && m.Fleet == 107 && m.Unit == 1772 {
				ok++
			}
		}
		mu.Unlock()
		if ok >= 2 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	t.Fatalf("scanner decoded %d bursts (%+v), want >= 2 CRC-valid FS-II Fleet 107 / Unit 1772", len(got), got)
}
