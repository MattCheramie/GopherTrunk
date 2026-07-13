package widebandt2

import (
	"context"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestClusteredOversampledPlanDecodesP25 is the regression guard for issue
// #881. The report hypothesised that a tightly-clustered / heavily
// oversampled channel plan (4 P25 control channels spanning 225 kHz inside a
// 2.048 MS/s capture) makes the wideband DDC hand the P25 demod undersized
// dibit chunks, so the frame sync word never fits and the control channels
// never lock.
//
// The fixture testdata/rmr-clustered-cc-2048k.u8 is a 0.25 s slice of the
// reporter's own raw capture (er-imagery, issue #881: serial 11257831, a
// NESDR SMArTee v5 on a VHF whip; centre 166.55 MHz, 2.048 MS/s, u8 IQ). It
// is run through the FULL live wideband engine — the same DDCBank tuner bank
// and P25 Phase 1 receiver chain the daemon drives for a role: wideband
// device — using the reporter's exact 4-channel clustered plan:
//
//	166.425000  (-125.0 kHz)
//	166.450000  (-100.0 kHz)   RMR control channel, NAC 365
//	166.462500  ( -87.5 kHz)   RMR control channel, NAC 365
//	166.650000  (+100.0 kHz)   Anakie control channel, NAC 362
//
// Result: the clustered plan decodes. Three of the four control channels lock
// and produce TSBKs through the DDCBank path (the fourth, 166.425, is a
// neighbour CC not strongly present in this window). This refutes the
// "clustered plan starves the demod" hypothesis at the top of the issue:
// chunk size is a function of the per-tap output rate (48 kHz, ~19 dibits per
// 8192-sample USB transfer), the P25 SyncDetector already correlates a frame
// sync word across those short chunks (a persistent 24-dibit history — issue
// #275), and the channel-plan geometry never enters into it.
//
// (The reporter's *live* non-decode tracks the front-end/gain state, not the
// DSP: this very capture is severely front-end-overloaded — ~50% of its u8
// samples are pinned to an ADC rail, wideband RMS ≈ +1.3 dBFS — yet still
// decodes, because C4FM is constant-envelope and its phase survives hard
// limiting. That is an overload / gain-staging condition, class of issue
// #749, not a channelizer defect.)
func TestClusteredOversampledPlanDecodesP25(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "rmr-clustered-cc-2048k.u8"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	n := len(raw) / 2
	iq := make([]complex64, n)
	for i := 0; i < n; i++ {
		ir := float32(raw[2*i]) - 127.5
		qr := float32(raw[2*i+1]) - 127.5
		iq[i] = complex(ir/127.5, qr/127.5)
	}

	// Chunk like an RTL USB bulk transfer (~8192 complex samples). At the
	// 48 kHz per-tap rate this is the ~19-dibit chunk the report flagged; the
	// assertion below proves that chunk size does not prevent sync.
	const chunkSamples = 8192
	var chunks [][]complex64
	for i := 0; i < len(iq); i += chunkSamples {
		end := i + chunkSamples
		if end > len(iq) {
			end = len(iq)
		}
		buf := make([]complex64, end-i)
		copy(buf, iq[i:end])
		chunks = append(chunks, buf)
	}

	const centerHz = 166_550_000
	freqs := []uint32{166_425_000, 166_450_000, 166_462_500, 166_650_000}
	var channels []ChannelConfig
	var systems []trunking.System
	for i, f := range freqs {
		name := "clustered-" + string(rune('A'+i))
		channels = append(channels, ChannelConfig{FrequencyHz: f, SystemName: name})
		systems = append(systems, p25Phase1System(name, f))
	}

	// Sanity-check the fixture really is the heavily-oversampled geometry the
	// report describes: the whole plan spans 225 kHz, well under half the
	// 2.048 MS/s band, so the "capture is oversampled for the channel plan"
	// advisory fires on exactly this plan.
	span := float64(freqs[len(freqs)-1]) - float64(freqs[0])
	if span > 0.5*2_048_000 {
		t.Fatalf("fixture span %.0f Hz is not the clustered/oversampled geometry", span)
	}

	dev := newMockDevice(chunks)
	bus := events.NewBus(4096)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	go func() {
		for range sub.C { //nolint:revive // drain so Publish never blocks the pump
		}
	}()

	e, err := New(Options{
		Log:          slog.New(slog.NewTextHandler(discardWriter{}, nil)),
		Device:       dev,
		Bus:          bus,
		SampleRateHz: 2_048_000,
		CenterFreqHz: centerHz,
		Channels:     channels,
		Systems:      systems,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := e.bank.OutputRateHz(); math.Abs(got-narrowbandRateHz) > 1 {
		t.Fatalf("per-tap output rate = %.1f Hz, want %.1f", got, narrowbandRateHz)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	runDone := make(chan error, 1)
	go func() { runDone <- e.Run(ctx) }()
	// The mock device closes its stream once every chunk is drained, which Run
	// reports as ErrIQStreamClosed — expected here (all fixture IQ has been
	// processed by then), so it is not a failure.
	<-runDone
	cancel()

	decoding := 0
	for _, ec := range e.channels {
		var decoded uint64
		if ec.decoded != nil {
			decoded = ec.decoded()
		}
		t.Logf("channel freq=%d proto=%s TSBK_decoded=%d", ec.freqHz, ec.protoTag, decoded)
		if decoded > 0 {
			decoding++
		}
	}

	// The core of the issue: a clustered/oversampled plan must NOT starve the
	// P25 demod. Three of the four control channels are actively broadcasting
	// in this slice; require at least two to decode through the DDCBank path.
	if decoding < 2 {
		t.Fatalf("clustered plan decoded %d/4 control channels through the wideband DDCBank path; "+
			"the oversampled/clustered geometry is NOT supposed to prevent P25 sync (issue #881)", decoding)
	}
}
