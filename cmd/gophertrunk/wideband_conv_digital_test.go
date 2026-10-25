package main

import (
	"context"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/widebandt2"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// p25LDUDibits is n back-to-back LDUs (LDU1/LDU2 alternating) carrying nac
// with random voice bits: one keyed P25 Phase 1 radio.
func p25LDUDibits(t *testing.T, nac uint16, n int, seed int64) []uint8 {
	t.Helper()
	r := rand.New(rand.NewSource(seed))
	var out []uint8
	for i := 0; i < n; i++ {
		var voice [phase1.LDUVoiceSubframeCount][]byte
		for j := range voice {
			voice[j] = make([]byte, phase1.LDUVoiceSubframeBits)
			for k := range voice[j] {
				voice[j][k] = byte(r.Intn(2))
			}
		}
		duid := phase1.DUIDLogicalLink1
		if i%2 == 1 {
			duid = phase1.DUIDLogicalLink2
		}
		var lces [phase1.LDULCESBlockCount][]byte
		var lsd [phase1.LDULSDBlockCount][]byte
		bits, err := phase1.AssembleLDU(nac, duid, voice, lces, lsd)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, framing.BitsToDibits(bits)...)
	}
	return out
}

// addAt48k up-converts a 48 kHz baseband transmission into wideband IQ at
// rate (an integer multiple of 48 kHz), offsetHz from the centre, starting at
// sample start. Linear interpolation: the engine's channel filter removes the
// images, as a band-limited transmitter's would be.
func addAt48k(out []complex64, tx []complex64, rate, offsetHz float64, start int) {
	up := int(rate / 48_000)
	for i := 0; i+1 < len(tx); i++ {
		a, b := tx[i], tx[i+1]
		for k := 0; k < up; k++ {
			n := start + i*up + k
			if n >= len(out) {
				return
			}
			f := float32(k) / float32(up)
			s := a + complex(f, 0)*(b-a)
			ph := 2 * math.Pi * offsetHz * float64(n) / rate
			out[n] += s * complex64(complex(math.Cos(ph), math.Sin(ph)))
		}
	}
}

// protoChainEngine is voiceChainEngine that also records each call's grant
// protocol.
type protoChainEngine struct {
	*voiceChainEngine
	mu     sync.Mutex
	protos []string
}

func (e *protoChainEngine) HandleSyntheticCall(g trunking.Grant, serial string) {
	e.mu.Lock()
	e.protos = append(e.protos, g.Protocol)
	e.mu.Unlock()
	e.voiceChainEngine.HandleSyntheticCall(g, serial)
}

// TestWidebandConvP25SharedFrequencyByNAC runs issue #1288's digital
// conventional mode end to end through the wideband engine: one tap at one
// frequency, two mode: p25 channels sharing it with NAC 293 and NAC 4A5, a
// NAC 293 transmission and then a NAC 4A5 one. Each channel must open
// exactly one call, with the p25 protocol (so the composer runs the P25
// voice chain), and its voice chain must hear its own transmission from the
// first samples and none of the other's. Before #1288's digital step a p25
// entry on a wideband dongle did not validate at all.
func TestWidebandConvP25SharedFrequencyByNAC(t *testing.T) {
	const (
		rate     = 960_000.0
		centerHz = 460_500_000
		chunk    = 9_600 // 10 ms
		seconds  = 5.0
		offset   = -200_000.0
	)
	chHz := uint32(centerHz + offset)
	r := rand.New(rand.NewSource(5))
	iq := make([]complex64, int(rate*seconds))
	for i := range iq {
		iq[i] = complex(float32(r.NormFloat64()*0.01), float32(r.NormFloat64()*0.01))
	}
	// 8 LDUs = 1.44 s per transmission.
	for _, tx := range []struct {
		nac   uint16
		start float64
	}{{0x293, 0.5}, {0x4A5, 2.8}} {
		bb := demod.ModulateP25C4FM(p25LDUDibits(t, tx.nac, 8, int64(tx.nac)), 48_000, 1800)
		for i := range bb {
			bb[i] *= 0.3
		}
		addAt48k(iq, bb, rate, offset, int(tx.start*rate))
	}
	var chunks [][]complex64
	for i := 0; i+chunk <= len(iq); i += chunk {
		chunks = append(chunks, iq[i:i+chunk])
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := events.NewBus(64)
	defer bus.Close()
	cfgChans := []config.ConvChannelConfig{
		{Label: "Own", FrequencyHz: chHz, Mode: "p25", NAC: "293", SquelchDbFS: -40, HangtimeMs: 300},
		{Label: "Other", FrequencyHz: chHz, Mode: "p25", NAC: "4A5", SquelchDbFS: -40, HangtimeMs: 300},
	}
	eng, err := widebandt2.New(widebandt2.Options{
		Log: log, Bus: bus,
		Device:       &pacedWidebandDevice{chunks: chunks, period: 10 * time.Millisecond},
		SampleRateHz: uint32(rate), CenterFreqHz: centerHz,
		Channels: widebandConvTapChannels(cfgChans),
		Serial:   "WB1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if tags := eng.ChannelProtocolTags(); len(tags) != 1 || tags[chHz] != "p25-conv" {
		t.Fatalf("engine taps %v, want one p25-conv tap", tags)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	chains := make([]*protoChainEngine, len(cfgChans))
	for i, ch := range cfgChans {
		conv := convChannelFromConfig(ch, config.WidebandConvPositionalBase|uint32(i))
		if !conv.NACSet {
			t.Fatalf("%s: NAC did not reach the scanner channel", ch.Label)
		}
		src := widebandt2.NewChannelVoiceSource(func() *widebandt2.Engine { return eng }, chHz,
			wbConvSharedP25Serial("WB1", chHz, conv.NAC))
		chains[i] = &protoChainEngine{voiceChainEngine: &voiceChainEngine{src: src}}
		cs, err := newWidebandConvScanner(src, conv, chains[i], discardRecorder{}, nil, log)
		if err != nil {
			t.Fatal(err)
		}
		go func() { _ = cs.Run(ctx) }()
	}
	time.Sleep(50 * time.Millisecond)
	go func() { _ = eng.Run(ctx) }()

	deadline := time.Now().Add(30 * time.Second)
	for {
		done := true
		for _, c := range chains {
			c.voiceChainEngine.mu.Lock()
			done = done && c.ended
			c.voiceChainEngine.mu.Unlock()
		}
		if done || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond)
	cancel()

	for i, c := range chains {
		c.voiceChainEngine.mu.Lock()
		calls, ended := c.calls, c.ended
		heard := float64(len(c.iq)) / 48_000
		c.voiceChainEngine.mu.Unlock()
		c.mu.Lock()
		protos := append([]string(nil), c.protos...)
		c.mu.Unlock()
		t.Logf("%s (NAC %s): %d call(s) %v, voice chain heard %.2f s", cfgChans[i].Label, cfgChans[i].NAC, calls, protos, heard)
		if calls != 1 || !ended {
			t.Errorf("%s: calls=%d ended=%v, want exactly one call that ended", cfgChans[i].Label, calls, ended)
			continue
		}
		if len(protos) != 1 || protos[0] != "p25" {
			t.Errorf("%s: grant protocols %v, want [p25]", cfgChans[i].Label, protos)
		}
		// Its own 1.44 s transmission plus up to the NAC hold and the
		// hangtime; the other channel's would add another 1.44 s.
		if heard < 1.3 || heard > 2.6 {
			t.Errorf("%s: voice chain heard %.2f s, want its own ~1.44 s transmission (plus tail) only", cfgChans[i].Label, heard)
		}
	}
}

// TestWidebandConvDigitalModesDaemonWiring builds a daemon whose wideband
// dongle carries one channel of every digital conventional mode (issue
// #1288): each gets its own tap tagged "<mode>-conv", source and scanner,
// and a p25 channel's NAC and demod reach its scanner channel.
func TestWidebandConvDigitalModesDaemonWiring(t *testing.T) {
	registerMockDongle(t)
	cfg := config.Default()
	cfg.SDR.SampleRate = 2_400_000
	modes := []struct {
		mode string
		hz   uint32
	}{
		{"p25", 460_100_000}, {"dmr", 460_200_000}, {"nxdn", 460_300_000},
		{"nxdn48", 460_400_000}, {"dstar", 460_600_000}, {"ysf", 460_700_000},
	}
	var chans []config.ConvChannelConfig
	for _, m := range modes {
		ch := config.ConvChannelConfig{Label: m.mode, FrequencyHz: m.hz, Mode: m.mode}
		if m.mode == "p25" {
			ch.NAC, ch.P25Phase1DemodMode = "293", "cqpsk"
		}
		chans = append(chans, ch)
	}
	cfg.SDR.Devices = []config.DeviceConfig{{Serial: "mock-00", Role: "wideband", CenterFreqHz: 460_500_000, Conventional: chans}}
	cfg.Trunking.Systems = nil
	cfg.API.HTTPAddr = ""
	cfg.Storage.Path = ""
	cfg.Storage.CCCacheFile = ""
	cfg.Recordings.Dir = t.TempDir()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	d, err := NewDaemon(cfg, "test-wideband-conv-digital", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	t.Cleanup(d.Close)

	tags := d.widebandT2[0].ChannelProtocolTags()
	for _, m := range modes {
		if tags[m.hz] != m.mode+"-conv" {
			t.Errorf("%s tap tag %q, want %q", m.mode, tags[m.hz], m.mode+"-conv")
		}
	}
	if len(d.wbConvScans) != len(modes) {
		t.Fatalf("%d scanners, want %d", len(d.wbConvScans), len(modes))
	}
	for _, c := range d.wbConvChannels {
		if c.channel.Mode != "p25" {
			continue
		}
		if !c.channel.NACSet || c.channel.NAC != 0x293 || c.channel.P25Phase1DemodMode != "cqpsk" {
			t.Errorf("p25 channel NAC=%#x set=%v demod=%q, want 0x293 / true / cqpsk", c.channel.NAC, c.channel.NACSet, c.channel.P25Phase1DemodMode)
		}
		if c.nac != "293" {
			t.Errorf("p25 channel status nac %q, want 293", c.nac)
		}
	}
}

// TestConventionalChannelMetrics pins issue #1288 §13 through NewDaemon: with
// metrics on, /metrics carries gophertrunk_conventional_channels_configured
// and _active by kind, read from the scan list and the wideband channels.
func TestConventionalChannelMetrics(t *testing.T) {
	registerMockDongle(t)
	cfg := config.Default()
	cfg.SDR.SampleRate = 2_400_000
	cfg.SDR.Devices = []config.DeviceConfig{{
		Serial: "mock-00", Role: "wideband", CenterFreqHz: 460_500_000,
		Conventional: []config.ConvChannelConfig{
			{Label: "Fire", FrequencyHz: 460_100_000, Mode: "nfm"},
			{Label: "P25", FrequencyHz: 460_200_000, Mode: "p25", NAC: "293"},
			{Label: "Air", FrequencyHz: 460_300_000, Mode: "am"},
		},
	}}
	cfg.Trunking.Systems = nil
	cfg.Metrics.Enabled = true
	cfg.API.HTTPAddr = "127.0.0.1:0"
	cfg.Storage.Path = ""
	cfg.Storage.CCCacheFile = ""
	cfg.Recordings.Dir = t.TempDir()
	d, err := NewDaemon(cfg, "test-conv-metrics", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	t.Cleanup(d.Close)
	if d.metrics == nil {
		t.Fatal("metrics not built")
	}
	mfs, err := d.metrics.Registry().Gather()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]float64{}
	for _, mf := range mfs {
		if !strings.HasPrefix(mf.GetName(), "gophertrunk_conventional_channels_") {
			continue
		}
		for _, m := range mf.GetMetric() {
			got[mf.GetName()+"/"+m.GetLabel()[0].GetValue()] = m.GetGauge().GetValue()
		}
	}
	want := map[string]float64{
		"gophertrunk_conventional_channels_configured/wideband": 3,
		"gophertrunk_conventional_channels_active/wideband":     0,
		"gophertrunk_conventional_channels_configured/scan":     0,
		"gophertrunk_conventional_channels_active/scan":         0,
	}
	for k, v := range want {
		if g, ok := got[k]; !ok || g != v {
			t.Errorf("%s = %v (present %v), want %v", k, g, ok, v)
		}
	}
}
