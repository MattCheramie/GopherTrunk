package main

import (
	"context"
	"io"
	"log/slog"
	"math"
	"math/cmplx"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/widebandt2"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// pacedWidebandDevice streams pre-built wideband IQ chunks at real time, so
// the scanners' wall-clock hangtime sees the same silences the samples hold.
type pacedWidebandDevice struct {
	chunks [][]complex64
	period time.Duration
}

func (d *pacedWidebandDevice) Info() sdr.Info             { return sdr.Info{Driver: "mock", Serial: "WB1"} }
func (d *pacedWidebandDevice) SetCenterFreq(uint32) error { return nil }
func (d *pacedWidebandDevice) SetSampleRate(uint32) error { return nil }
func (d *pacedWidebandDevice) SetGain(int) error          { return nil }
func (d *pacedWidebandDevice) SetPPM(int) error           { return nil }
func (d *pacedWidebandDevice) SetBiasTee(bool) error      { return nil }
func (d *pacedWidebandDevice) Close() error               { return nil }
func (d *pacedWidebandDevice) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	out := make(chan []complex64, 4)
	go func() {
		defer close(out)
		tick := time.NewTicker(d.period)
		defer tick.Stop()
		for _, c := range d.chunks {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
			select {
			case <-ctx.Done():
				return
			case out <- c:
			}
		}
		<-ctx.Done()
	}()
	return out, nil
}

// syntheticCallLog is a conventional.Engine that records every call.
type syntheticCallLog struct {
	mu    sync.Mutex
	start time.Time
	calls map[string]*loggedCall // by device serial
	order []*loggedCall
}

type loggedCall struct {
	serial     string
	freqHz     uint32
	groupID    uint32
	start, end time.Duration
	ended      bool
}

func (l *syntheticCallLog) HandleSyntheticCall(g trunking.Grant, serial string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	c := &loggedCall{serial: serial, freqHz: g.FrequencyHz, groupID: g.GroupID, start: time.Since(l.start)}
	l.calls[serial] = c
	l.order = append(l.order, c)
}

func (l *syntheticCallLog) EndSyntheticCall(serial string, _ trunking.EndReason) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	c, ok := l.calls[serial]
	if !ok {
		return false
	}
	delete(l.calls, serial)
	c.end, c.ended = time.Since(l.start), true
	return true
}

func (l *syntheticCallLog) Touch(string) {}

type discardRecorder struct{}

func (discardRecorder) WritePCM(string, []int16) error { return nil }

// nfmBurst is a transmitter: an FM carrier at offsetHz, modulated by a tone,
// keyed from on to off seconds.
type nfmBurst struct {
	offsetHz, toneHz float64
	on, off          float64
}

// synthWideband sums the transmitters plus a little receiver noise.
func synthWideband(rate float64, seconds float64, txs []nfmBurst) []complex64 {
	const amp, deviationHz = 0.3, 3000.0
	n := int(rate * seconds)
	out := make([]complex64, n)
	rng := rand.New(rand.NewSource(1))
	phases := make([]float64, len(txs))
	for i := range out {
		t := float64(i) / rate
		var s complex128
		for k, tx := range txs {
			inst := tx.offsetHz + deviationHz*math.Sin(2*math.Pi*tx.toneHz*t)
			phases[k] += 2 * math.Pi * inst / rate
			if t >= tx.on && t < tx.off {
				s += cmplx.Rect(amp, phases[k])
			}
		}
		s += complex(rng.NormFloat64()*0.01, rng.NormFloat64()*0.01)
		out[i] = complex64(s)
	}
	return out
}

// toneRatioDB is the FM-demodulated power at want over the power at other,
// in dB, for iq at rate.
func toneRatioDB(iq []complex64, rate, want, other float64) float64 {
	disc := make([]float64, 0, len(iq))
	for i := 1; i < len(iq); i++ {
		disc = append(disc, cmplx.Phase(complex128(iq[i]*complex(real(iq[i-1]), -imag(iq[i-1])))))
	}
	power := func(f float64) float64 {
		var re, im float64
		for i, v := range disc {
			re += v * math.Cos(2*math.Pi*f*float64(i)/rate)
			im += v * math.Sin(2*math.Pi*f*float64(i)/rate)
		}
		return re*re + im*im
	}
	return 10 * math.Log10(power(want)/power(other))
}

// TestWidebandConventionalOverlappingCallsRecordIndependently is issue
// #1288's acceptance test A, on the production wiring: one wideband capture
// holding two overlapping NFM transmissions on channels 550 kHz apart, run
// through a real widebandt2 engine with two tap-only conventional channels
// and one newWidebandConvScanner per channel. A scan-list scanner dwelling
// on channel A would have missed B's whole transmission; here each channel
// opens its own call under its own serial while the other is active, both
// end, and each tap carries only its own transmitter's audio.
func TestWidebandConventionalOverlappingCallsRecordIndependently(t *testing.T) {
	const (
		rate     = 960_000.0
		centerHz = 460_500_000
		chunk    = 9_600 // 10 ms
		seconds  = 2.4
	)
	aHz, bHz := uint32(centerHz-300_000), uint32(centerHz+250_000)
	txA := nfmBurst{offsetHz: -300_000, toneHz: 1000, on: 0.2, off: 1.4}
	txB := nfmBurst{offsetHz: 250_000, toneHz: 1700, on: 0.6, off: 1.9}
	iq := synthWideband(rate, seconds, []nfmBurst{txA, txB})
	var chunks [][]complex64
	for i := 0; i+chunk <= len(iq); i += chunk {
		chunks = append(chunks, iq[i:i+chunk])
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := events.NewBus(64)
	defer bus.Close()
	eng, err := widebandt2.New(widebandt2.Options{
		Log:          log,
		Bus:          bus,
		Device:       &pacedWidebandDevice{chunks: chunks, period: 10 * time.Millisecond},
		SampleRateHz: uint32(rate),
		CenterFreqHz: centerHz,
		Channels: []widebandt2.ChannelConfig{
			{FrequencyHz: aHz, Conventional: true},
			{FrequencyHz: bHz, Conventional: true},
		},
		Serial: "WB1",
	})
	if err != nil {
		t.Fatal(err)
	}
	get := func() *widebandt2.Engine { return eng }

	calls := &syntheticCallLog{start: time.Now(), calls: map[string]*loggedCall{}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Taps for the cross-talk check, subscribed before any IQ flows.
	type tap struct {
		ch    <-chan []complex64
		mu    sync.Mutex
		iq    []complex64
		unsub func() uint64
	}
	taps := map[uint32]*tap{}
	for _, hz := range []uint32{aHz, bHz} {
		ch, unsub, ok := eng.SubscribeChannelIQ(hz)
		if !ok {
			t.Fatalf("no tap on %d Hz", hz)
		}
		tp := &tap{ch: ch, unsub: unsub}
		taps[hz] = tp
		go func() {
			for c := range tp.ch {
				tp.mu.Lock()
				tp.iq = append(tp.iq, c...)
				tp.mu.Unlock()
			}
		}()
	}

	for i, hz := range []uint32{aHz, bHz} {
		cfgCh := config.ConvChannelConfig{Label: string(rune('A' + i)), FrequencyHz: hz, Mode: "nfm", SquelchDbFS: -30, HangtimeMs: 250}
		src := widebandt2.NewChannelVoiceSource(get, hz, wbConvSerial("WB1", hz))
		cs, err := newWidebandConvScanner(src, convChannelFromConfig(cfgCh, config.WidebandConvPositionalBase|uint32(i)),
			calls, discardRecorder{}, nil, log)
		if err != nil {
			t.Fatal(err)
		}
		go func() { _ = cs.Run(ctx) }()
	}
	// Scanners subscribe as soon as they run; give them a moment before the
	// first chunk.
	time.Sleep(50 * time.Millisecond)
	go func() { _ = eng.Run(ctx) }()

	deadline := time.Now().Add(15 * time.Second)
	for {
		calls.mu.Lock()
		done := len(calls.order) >= 2 && calls.order[0].ended && calls.order[1].ended
		calls.mu.Unlock()
		if done || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()

	calls.mu.Lock()
	defer calls.mu.Unlock()
	if len(calls.order) != 2 {
		t.Fatalf("%d calls, want 2 (one per channel): %+v", len(calls.order), calls.order)
	}
	byFreq := map[uint32]*loggedCall{}
	for _, c := range calls.order {
		byFreq[c.freqHz] = c
	}
	a, b := byFreq[aHz], byFreq[bHz]
	if a == nil || b == nil {
		t.Fatalf("calls on %v, want one on each of %d and %d", calls.order, aHz, bHz)
	}
	if a.serial == b.serial || a.groupID == b.groupID {
		t.Errorf("calls share an identity: A %s/%#x, B %s/%#x", a.serial, a.groupID, b.serial, b.groupID)
	}
	if !a.ended || !b.ended {
		t.Fatalf("calls did not end: A %+v, B %+v", a, b)
	}
	if !(b.start < a.end && a.start < b.end) {
		t.Errorf("calls did not overlap: A %v–%v, B %v–%v", a.start, a.end, b.start, b.end)
	}

	// Cross-talk: during the overlap (0.7–1.3 s) each tap carries its own
	// transmitter's tone far above the other's.
	for hz, want := range map[uint32][2]float64{aHz: {txA.toneHz, txB.toneHz}, bHz: {txB.toneHz, txA.toneHz}} {
		tp := taps[hz]
		tp.unsub()
		tp.mu.Lock()
		lo, hi := int(0.7*48_000), int(1.3*48_000)
		if len(tp.iq) < hi {
			tp.mu.Unlock()
			t.Fatalf("tap %d Hz delivered %d samples, want ≥ %d", hz, len(tp.iq), hi)
		}
		r := toneRatioDB(tp.iq[lo:hi], 48_000, want[0], want[1])
		t.Logf("tap %d Hz: own tone %.1f dB above the other", hz, r)
		tp.mu.Unlock()
		if r < 30 {
			t.Errorf("tap %d Hz: own tone only %.1f dB above the other channel's", hz, r)
		}
	}
}

// TestWidebandConvDaemonWiring builds a daemon whose only radio is a
// role: wideband dongle with two conventional channels and no trunking
// system, and checks the pieces the overlapping-calls test does not: config
// validation accepts it, the engine taps both channels as fm-conv, each
// channel gets its own scanner and composer source under its own serial, the
// sources stay out of the trunking voice pool, and positional talkgroup IDs
// sit in the range config validation checks (issue #1288).
func TestWidebandConvDaemonWiring(t *testing.T) {
	registerMockDongle(t)
	cfg := config.Default()
	cfg.SDR.SampleRate = 2_400_000
	cfg.SDR.Devices = []config.DeviceConfig{{
		Serial: "mock-00", Role: "wideband", CenterFreqHz: 460_500_000,
		Conventional: []config.ConvChannelConfig{
			{Label: "Fire", FrequencyHz: 460_100_000, Mode: "nfm"},
			{Label: "EMS", FrequencyHz: 460_900_000, Mode: "nfm", TalkgroupID: 4242,
				Tone: config.ConvToneConfig{Mode: "ctcss", CTCSSHz: 123.0}},
		},
	}}
	cfg.Trunking.Systems = nil
	cfg.API.HTTPAddr = ""
	cfg.Storage.Path = ""
	cfg.Storage.CCCacheFile = ""
	cfg.Recordings.Dir = t.TempDir()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	d, err := NewDaemon(cfg, "test-wideband-conv", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	if len(d.widebandT2) != 1 {
		t.Fatalf("%d wideband engines, want 1", len(d.widebandT2))
	}
	tags := d.widebandT2[0].ChannelProtocolTags()
	if tags[460_100_000] != "fm-conv" || tags[460_900_000] != "fm-conv" {
		t.Errorf("engine channel tags %v, want both fm-conv", tags)
	}
	if len(d.wbConvChannels) != 2 || len(d.wbConvScans) != 2 {
		t.Fatalf("%d sources / %d scanners, want 2 / 2", len(d.wbConvChannels), len(d.wbConvScans))
	}
	if len(d.convScans) != 0 {
		t.Errorf("%d scan-list scanners, want 0 (the wideband channels must not join the cockpit's list)", len(d.convScans))
	}
	vm := d.virtualVoiceMap()
	wantID := map[uint32]uint32{460_100_000: config.WidebandConvPositionalBase, 460_900_000: 4242}
	for _, c := range d.wbConvChannels {
		hz := c.channel.FrequencyHz
		if c.source.Serial() != wbConvSerial("mock-00", hz) {
			t.Errorf("%d Hz: serial %q", hz, c.source.Serial())
		}
		if vm[c.source.Serial()] == nil {
			t.Errorf("%d Hz: source missing from the composer's device map", hz)
		}
		if c.channel.TalkgroupID != wantID[hz] {
			t.Errorf("%d Hz: talkgroup %#x, want %#x", hz, c.channel.TalkgroupID, wantID[hz])
		}
	}
	for _, v := range d.collectVoiceDevices() {
		if strings.Contains(v.Serial, ":conv:") {
			t.Errorf("wideband conventional source %q is in the trunking voice pool", v.Serial)
		}
	}
	if st := d.wbConvScans[1].Snapshot(); len(st.Channels) != 1 || st.Channels[0].FrequencyHz != 460_900_000 {
		t.Errorf("second scanner channels %+v, want only 460.9 MHz", st.Channels)
	}
}
