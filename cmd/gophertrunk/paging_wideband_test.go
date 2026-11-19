package main

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/pager/flex"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// pagingAirFake is an SDR with one FLEX transmitter on the air at txHz. Like
// a real dongle it delivers IQ only while a StreamIQ session runs, refuses a
// second session while one is live, and what it delivers depends on where it
// was tuned: the transmitter sits at txHz minus the programmed centre.
type pagingAirFake struct {
	serial string
	rateHz float64
	txHz   uint32
	syms   []byte // one FLEX frame, 1600 symbols/s

	// dieAfter, when > 0, ends the FIRST stream session after that many
	// chunks of noise, as a stream that dies under a decoder does.
	dieAfter int

	center  atomic.Uint32
	active  atomic.Bool
	streams atomic.Int32
}

func (f *pagingAirFake) Info() sdr.Info { return sdr.Info{Driver: "fake", Serial: f.serial} }
func (f *pagingAirFake) SetCenterFreq(hz uint32) error {
	f.center.Store(hz)
	return nil
}
func (f *pagingAirFake) SetSampleRate(uint32) error { return nil }
func (f *pagingAirFake) SetGain(int) error          { return nil }
func (f *pagingAirFake) SetPPM(int) error           { return nil }
func (f *pagingAirFake) SetBiasTee(bool) error      { return nil }
func (f *pagingAirFake) Close() error               { return nil }

// pagingFakeChunk is the fake's chunk size: 10 ms at 240 kS/s.
const pagingFakeChunk = 2400

func (f *pagingAirFake) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	if !f.active.CompareAndSwap(false, true) {
		return nil, errors.New("fake: stream already active")
	}
	session := int(f.streams.Add(1))
	offsetHz := float64(f.txHz) - float64(f.center.Load())
	out := make(chan []complex64, 4)
	go func() {
		defer f.active.Store(false)
		defer close(out)
		rng := rand.New(rand.NewSource(int64(session)))
		noise := func(n int) []complex64 {
			c := make([]complex64, n)
			for i := range c {
				c[i] = complex(float32(0.01*rng.NormFloat64()), float32(0.01*rng.NormFloat64()))
			}
			return c
		}
		// Real-time pacing: the group's per-tap feeds drop when a receiver
		// falls behind, as they do on a live dongle, so a flood would lose
		// the frame instead of testing the wiring.
		tick := time.NewTicker(time.Duration(float64(pagingFakeChunk) / f.rateHz * float64(time.Second)))
		defer tick.Stop()
		send := func(c []complex64) bool {
			select {
			case <-ctx.Done():
				return false
			case <-tick.C:
			}
			select {
			case <-ctx.Done():
				return false
			case out <- c:
				return true
			}
		}
		if f.dieAfter > 0 && session == 1 {
			for i := 0; i < f.dieAfter; i++ {
				if !send(noise(pagingFakeChunk)) {
					return
				}
			}
			return // the stream dies under the decoder
		}
		air := flexFSK(f.syms, f.rateHz, offsetHz, 0.1, rng)
		for len(air) > 0 {
			n := pagingFakeChunk
			if n > len(air) {
				n = len(air)
			}
			if !send(air[:n]) {
				return
			}
			air = air[n:]
		}
		for send(noise(pagingFakeChunk)) {
		}
	}()
	return out, nil
}

// flexFSK modulates one 1600 symbols/s FLEX frame as 2-level FSK IQ at
// rateHz: symbol 0 at -4800 Hz and symbol 3 at +4800 Hz around offsetHz,
// after leadSec of noise, with the frequency steps smoothed as a
// transmitter's pre-modulation filter does.
func flexFSK(syms []byte, rateHz, offsetHz, leadSec float64, rng *rand.Rand) []complex64 {
	perSym := rateHz / float64(flex.SyncSymbolRate)
	lead := int(leadSec * rateHz)
	body := int(float64(len(syms)) * perSym)
	total := lead + body + int(0.05*rateHz)
	freq := make([]float64, total)
	for i := 0; i < body; i++ {
		freq[lead+i] = (float64(syms[int(float64(i)/perSym)])*2 - 3) * 1600
	}
	w := int(perSym / 10)
	sm := make([]float64, total)
	var acc float64
	for i := range freq {
		acc += freq[i]
		if i >= w {
			acc -= freq[i-w]
		}
		sm[i] = acc / float64(w)
	}
	iq := make([]complex64, total)
	var phase float64
	for i := range iq {
		amp := 0.0
		if i >= lead && i < lead+body {
			amp = 0.5
		}
		phase += 2 * math.Pi * (sm[i] + offsetHz) / rateHz
		iq[i] = complex(
			float32(amp*math.Cos(phase)+0.01*rng.NormFloat64()),
			float32(amp*math.Sin(phase)+0.01*rng.NormFloat64()))
	}
	return iq
}

const (
	pagingTestSerial = "pager-sdr"
	pagingTestRateHz = 240_000
	pagingTestTxHz   = 929_662_500
	pagingTestText   = "ENGINE 4 RESPOND 100 MAIN ST"
	pagingTestRIC    = 1234567
)

// pagingTestGroup is the Discord reporter's shape: one dedicated dongle, one
// paging.wideband entry listing 929 MHz FLEX channels.
func pagingTestGroup() config.PagingWidebandConfig {
	return config.PagingWidebandConfig{
		Serial: pagingTestSerial,
		Channels: []config.PagingWidebandChannel{
			{Protocol: "flex", FrequencyHz: 929_612_500},
			{Protocol: "flex", FrequencyHz: pagingTestTxHz},
		},
	}
}

// newPagingRunDaemon builds the Daemon runWidebandPagingGroup needs around
// dev: its iqtap broker (seeded at brokerRateHz, as wrapIQBrokers seeds it
// with the rate the pool programmed), the event bus and the primary-stream
// bookkeeping. sdr.sample_rate is cfgRateHz.
func newPagingRunDaemon(t *testing.T, dev sdr.Device, cfgRateHz, brokerRateHz uint32) (*Daemon, *iqtap.Broker) {
	t.Helper()
	br := iqtap.New(dev, 0, nil)
	br.Seed(0, brokerRateHz)
	d := &Daemon{
		log:            slog.New(slog.DiscardHandler),
		bus:            events.NewBus(64),
		iqPrimary:      map[string]bool{},
		scannerBrokers: map[string]*iqtap.Broker{},
		iqBrokers:      map[string]*iqtap.Broker{pagingTestSerial: br},
	}
	d.cfg.SDR.SampleRate = cfgRateHz
	t.Cleanup(d.bus.Close)
	return d, br
}

func pagingTestFrame(t *testing.T) []byte {
	t.Helper()
	phase, ok := flex.PhaseWith(flex.Page{Capcode: pagingTestRIC, Text: pagingTestText})
	if !ok {
		t.Fatal("test page does not fit a FLEX phase")
	}
	return flex.FrameSymbols(flex.Mode1600x2, 0, 1, phase)
}

// startPagingGroup builds the group as the daemon does and runs it as the
// daemon's Run loop spawns it. The returned stop cancels it and waits.
func startPagingGroup(t *testing.T, d *Daemon, sampleRateHz uint32) (pages *events.Subscription, stop func()) {
	t.Helper()
	group := d.buildWidebandPagingGroup(pagingTestGroup(), sampleRateHz, d.log)
	if group == nil {
		t.Fatalf("buildWidebandPagingGroup returned nil; warnings=%v", d.StartupWarnings())
	}
	pages = d.bus.Subscribe()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = d.runWidebandPagingGroup(ctx, *group)
	}()
	var once sync.Once
	stop = func() {
		once.Do(func() {
			cancel()
			wg.Wait()
			pages.Close()
		})
	}
	t.Cleanup(stop)
	return pages, stop
}

// awaitTestPage waits for the fake transmitter's page on the bus.
func awaitTestPage(t *testing.T, pages *events.Subscription, within time.Duration, why string) {
	t.Helper()
	deadline := time.After(within)
	for {
		select {
		case ev, ok := <-pages.C:
			if !ok {
				t.Fatalf("event bus closed before the page arrived (%s)", why)
			}
			if ev.Kind != events.KindPagerMessage {
				continue
			}
			msg, _ := ev.Payload.(storage.PagerMessage)
			if msg.Protocol != "flex" || msg.RIC != pagingTestRIC || msg.Body != pagingTestText {
				t.Fatalf("decoded page = %+v, want flex RIC %d %q", msg, pagingTestRIC, pagingTestText)
			}
			return
		case <-deadline:
			t.Fatalf("no page decoded within %v: %s", within, why)
		}
	}
}

// TestWidebandPagingDecodesWithNoLiveView is the Discord report: a
// paging-only config (one dedicated RTL-SDR, one paging.wideband group of
// 929 MHz FLEX channels) decoded pages only while the Spectrum view was open.
// The group only Subscribed to its SDR's iqtap broker, and a broker fans IQ
// out only while a primary StreamIQ session runs. With no trunking system
// nothing ever started one, so the dongle sat closed until a live view's
// on-demand stream (acquireViewerIQ) happened to feed the group, and it went
// deaf again when the view closed. The group must run its SDR itself.
func TestWidebandPagingDecodesWithNoLiveView(t *testing.T) {
	dev := &pagingAirFake{serial: pagingTestSerial, rateHz: pagingTestRateHz,
		txHz: pagingTestTxHz, syms: pagingTestFrame(t)}
	d, br := newPagingRunDaemon(t, dev, pagingTestRateHz, pagingTestRateHz)
	pages, _ := startPagingGroup(t, d, pagingTestRateHz)

	deadline := time.Now().Add(3 * time.Second)
	for !br.Stats().Streaming {
		if time.Now().After(deadline) {
			t.Fatal("paging.wideband never started its SDR's IQ stream: with no live view open " +
				"the dongle stays closed and no page can be decoded")
		}
		time.Sleep(5 * time.Millisecond)
	}
	awaitTestPage(t, pages, 15*time.Second, "the SDR is streaming but the group decoded nothing")

	// The group owns the stream now, so a live view must ride its fan-out
	// rather than open (and later close) a stream of its own.
	release := d.acquireViewerIQForRole(pagingTestSerial, sdr.RoleDedicated)
	defer release()
	if n := len(d.viewerPumps); n != 0 {
		t.Fatalf("a live view started its own stream on the paging SDR (%d viewer pumps)", n)
	}
	if got := dev.streams.Load(); got != 1 {
		t.Fatalf("StreamIQ sessions = %d, want 1", got)
	}
}

// TestWidebandPagingBankFollowsItsSDRsOwnRate: a dongle a paging section
// pins may run its own sample_rate (#1316), and the group's DDC bank must be
// built for the rate that dongle streams at. It was built for the shared
// sdr.sample_rate, which puts every tap at the wrong offset and the wrong
// output rate on such a dongle: nothing decodes, with no warning.
func TestWidebandPagingBankFollowsItsSDRsOwnRate(t *testing.T) {
	const serial = "pager-own-rate"
	sdr.Register(&fixedSDRDriver{name: "fake-paging-own-rate", serials: []string{serial}})

	cfg := config.Default()
	cfg.API.HTTPAddr = ""
	cfg.Trunking.Systems = nil
	cfg.SDR.SampleRate = 2_400_000
	cfg.SDR.Devices = []config.DeviceConfig{{Serial: serial, Role: "dedicated", SampleRate: 1_200_000}}
	group := pagingTestGroup()
	group.Serial = serial
	cfg.Paging.Wideband = []config.PagingWidebandConfig{group}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config: %v", err)
	}

	d, err := NewDaemon(cfg, "paging-rate-test", slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	t.Cleanup(d.Close)
	if d.pool == nil {
		t.Fatalf("pool did not open: %v", d.StartupWarnings())
	}
	if e := d.pool.FindBySerial(serial); e == nil || e.Role != sdr.RoleDedicated {
		t.Fatalf("paging SDR entry = %+v, want role dedicated", e)
	}
	if len(d.pagingGroups) != 1 {
		t.Fatalf("paging groups = %d, want 1; warnings=%v", len(d.pagingGroups), d.StartupWarnings())
	}
	if got := d.pagingGroups[0].sampleRateHz; got != 1_200_000 {
		t.Fatalf("group DDC bank rate = %d S/s, want the dongle's own 1200000 (sdr.sample_rate is 2400000)", got)
	}
	if got := d.iqBrokers[serial].SampleRateHz(); got != 1_200_000 {
		t.Fatalf("broker rate = %d S/s, want 1200000", got)
	}
}
