package main

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/widebandt2"
)

// TestWidebandConvSharedFrequencyDaemonWiring pins issue #1288 §5 through
// NewDaemon: two logical channels on one frequency, told apart by CTCSS and
// DCS, share ONE engine tap (the carrier is extracted once) but each get
// their own source, scanner, talkgroup ID and a serial naming their tone,
// while a channel alone on its frequency keeps the plain serial.
func TestWidebandConvSharedFrequencyDaemonWiring(t *testing.T) {
	registerMockDongle(t)
	const shared, alone = 460_100_000, 460_300_000
	cfg := config.Default()
	cfg.SDR.SampleRate = 2_400_000
	cfg.SDR.Devices = []config.DeviceConfig{{
		Serial: "mock-00", Role: "wideband", CenterFreqHz: 460_500_000,
		Conventional: []config.ConvChannelConfig{
			{Label: "Fire North", FrequencyHz: shared, Mode: "nfm", Tone: config.ConvToneConfig{Mode: "ctcss", CTCSSHz: 100}},
			{Label: "Ops", FrequencyHz: alone, Mode: "nfm"},
			{Label: "Fire South", FrequencyHz: shared, Mode: "nfm", Tone: config.ConvToneConfig{Mode: "dcs", DCSCode: "023"}},
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
	d, err := NewDaemon(cfg, "test-wideband-conv-shared", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	t.Cleanup(d.Close)

	if len(d.widebandT2) != 1 {
		t.Fatalf("%d wideband engines, want 1", len(d.widebandT2))
	}
	if tags := d.widebandT2[0].ChannelProtocolTags(); len(tags) != 2 || tags[shared] != "fm-conv" || tags[alone] != "fm-conv" {
		t.Errorf("engine channels %v, want one fm-conv tap per frequency (2)", tags)
	}
	if n := len(d.widebandT2[0].Channels()); n != 2 {
		t.Errorf("engine has %d taps, want 2: the shared carrier must be extracted once", n)
	}
	if len(d.wbConvChannels) != 3 || len(d.wbConvScans) != 3 {
		t.Fatalf("%d sources / %d scanners, want 3 / 3 (one per logical channel)", len(d.wbConvChannels), len(d.wbConvScans))
	}
	want := map[string]uint32{ // serial → talkgroup
		"wb:mock-00:conv:460100000:ctcss100.0": config.WidebandConvPositionalBase,
		"wb:mock-00:conv:460300000":            config.WidebandConvPositionalBase | 1,
		"wb:mock-00:conv:460100000:dcs023n":    config.WidebandConvPositionalBase | 2,
	}
	vm := d.virtualVoiceMap()
	for _, c := range d.wbConvChannels {
		s := c.source.Serial()
		tg, ok := want[s]
		if !ok {
			t.Errorf("unexpected serial %q", s)
			continue
		}
		delete(want, s)
		if c.channel.TalkgroupID != tg {
			t.Errorf("%s: talkgroup %#x, want %#x", s, c.channel.TalkgroupID, tg)
		}
		if vm[s] == nil {
			t.Errorf("%s: missing from the composer's device map", s)
		}
	}
	for s := range want {
		t.Errorf("no channel with serial %q", s)
	}

	tones := map[string]string{}
	for _, ch := range d.newScannerCockpit().Status().Conventional.WidebandChannels {
		tones[ch.Label] = ch.Tone
	}
	if tones["Fire North"] != "CTCSS 100.0 Hz" || tones["Fire South"] != "DCS 023N" || tones["Ops"] != "" {
		t.Errorf("status tones %v", tones)
	}
}

// TestWidebandConvSharedFrequencyToneDiscrimination is issue #1288's test E
// on a shared carrier: one engine tap at one frequency, two logical channels
// gated on CTCSS 100 Hz and 141.3 Hz. A transmission carrying 100 Hz and a
// later one carrying 141.3 Hz must each open exactly one call, on the channel
// whose tone it carries, and each call's voice chain must hear its own whole
// transmission (pre-roll included) and nothing of the other.
func TestWidebandConvSharedFrequencyToneDiscrimination(t *testing.T) {
	const (
		rate     = 960_000.0
		centerHz = 460_500_000
		chunk    = 9_600 // 10 ms
		seconds  = 4.0
	)
	chHz := uint32(centerHz - 200_000)
	a := synthToneGated(rate, seconds, -200_000, 1000, 100, 0.5, 1.5)
	b := synthToneGated(rate, seconds, -200_000, 1000, 141.3, 2.3, 3.3)
	iq := make([]complex64, len(a))
	for i := range iq {
		iq[i] = a[i] + b[i]
	}
	var chunks [][]complex64
	for i := 0; i+chunk <= len(iq); i += chunk {
		chunks = append(chunks, iq[i:i+chunk])
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := events.NewBus(64)
	defer bus.Close()
	eng, err := widebandt2.New(widebandt2.Options{
		Log: log, Bus: bus,
		Device:       &pacedWidebandDevice{chunks: chunks, period: 10 * time.Millisecond},
		SampleRateHz: uint32(rate), CenterFreqHz: centerHz,
		Channels: widebandConvTapChannels([]config.ConvChannelConfig{
			{FrequencyHz: chHz, Mode: "nfm"}, {FrequencyHz: chHz, Mode: "nfm"},
		}),
		Serial: "WB1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := len(eng.Channels()); n != 1 {
		t.Fatalf("engine has %d taps for one shared frequency, want 1", n)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	chains := map[float64]*voiceChainEngine{}
	for i, tone := range []float64{100, 141.3} {
		g := config.ConvToneConfig{Mode: "ctcss", CTCSSHz: tone}
		src := widebandt2.NewChannelVoiceSource(func() *widebandt2.Engine { return eng }, chHz,
			wbConvSharedSerial("WB1", chHz, convToneGate(g)))
		ch := config.ConvChannelConfig{Label: "L", FrequencyHz: chHz, Mode: "nfm", SquelchDbFS: -30, HangtimeMs: 250, Tone: g}
		chain := &voiceChainEngine{src: src}
		chains[tone] = chain
		cs, err := newWidebandConvScanner(src, convChannelFromConfig(ch, config.WidebandConvPositionalBase|uint32(i)),
			chain, discardRecorder{}, nil, log)
		if err != nil {
			t.Fatal(err)
		}
		go func() { _ = cs.Run(ctx) }()
	}
	time.Sleep(50 * time.Millisecond)
	go func() { _ = eng.Run(ctx) }()

	deadline := time.Now().Add(20 * time.Second)
	for {
		done := true
		for _, c := range chains {
			c.mu.Lock()
			done = done && c.ended
			c.mu.Unlock()
		}
		if done || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	// Let any (wrong) second call on either channel show up before checking.
	time.Sleep(300 * time.Millisecond)
	cancel()

	for _, tone := range []float64{100, 141.3} {
		c := chains[tone]
		c.mu.Lock()
		carrier, lead := carrierSpan(c.iq)
		calls, ended := c.calls, c.ended
		c.mu.Unlock()
		t.Logf("CTCSS %.1f Hz channel: %d call(s), voice chain heard %.2f s of carrier after %.3f s of noise", tone, calls, carrier, lead)
		if calls != 1 || !ended {
			t.Errorf("CTCSS %.1f Hz channel: calls=%d ended=%v, want exactly one call that ended", tone, calls, ended)
			continue
		}
		// Each transmission is 1.0 s; the other channel's would add a
		// second.
		if carrier < 0.95 || carrier > 1.1 {
			t.Errorf("CTCSS %.1f Hz channel: voice chain heard %.2f s of carrier, want its own 1.0 s transmission only", tone, carrier)
		}
		if lead > 0.03 {
			t.Errorf("CTCSS %.1f Hz channel: voice chain starts with %.3f s of noise", tone, lead)
		}
	}
}
