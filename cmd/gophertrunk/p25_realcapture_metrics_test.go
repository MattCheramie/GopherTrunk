//go:build integration

package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/metrics"
	p25phase1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// p25MetricsExpected is the "expected" payload for samples/p25/*.metadata.json.
// Every bound is optional: a zero/absent field is reported but not asserted, so
// a capture can be dropped in to *measure* the demod before anyone commits to a
// pass/fail threshold. See samples/p25/README.md.
type p25MetricsExpected struct {
	DemodMode     string  `json:"demod_mode"` // "c4fm" (default) or "cqpsk"
	NAC           string  `json:"nac"`        // hex, e.g. "0x167"
	MinNIDTrusted int     `json:"min_nid_trusted"`
	MinTSBK       int     `json:"min_tsbk"`
	MaxEVMPct     float64 `json:"max_evm_pct"`
	MinSNRdB      float64 `json:"min_snr_db"`
	MinSyncMargin int     `json:"min_sync_margin"`
}

// TestReplayP25RealCaptureMetrics is the skip-gated field-truth companion to
// the synthetic sweep (sweep_test.go). When a samples/p25/*.cfile +
// *.metadata.json pair is present it runs the real capture through the
// production receiver, reports the demod-quality metrics the harness
// measures — pre-FEC EVM, estimated SNR, FSW sync-margin, NID/TSBK yields —
// and asserts whichever bounds the metadata declares. With no capture present
// it skips. See samples/p25/README.md for the schema and how to drop a capture.
func TestReplayP25RealCaptureMetrics(t *testing.T) {
	cfilePath, metaPath := findRealairCapture(t, "p25")
	if cfilePath == "" {
		t.Skip("samples/p25/: no *.cfile present — drop a capture + metadata pair to run this test (see samples/p25/README.md)")
	}
	meta := loadRealairMetadata(t, metaPath)

	var exp p25MetricsExpected
	if len(meta.Expected) > 0 {
		if err := json.Unmarshal(meta.Expected, &exp); err != nil {
			t.Fatalf("parse expected payload: %v", err)
		}
	}

	demodMode := p25phase1rx.DemodC4FM
	rotations := p25phase1.RotationsC4FM
	if exp.DemodMode == "cqpsk" || exp.DemodMode == "lsm" {
		demodMode = p25phase1rx.DemodCQPSK
		rotations = p25phase1.RotationsAll
	}

	// Load the GNU Radio cfile (interleaved little-endian float32 IQ).
	raw, err := os.ReadFile(cfilePath)
	if err != nil {
		t.Fatalf("read capture: %v", err)
	}
	pairs := len(raw) / 8
	if pairs == 0 {
		t.Fatalf("capture %s has no samples", cfilePath)
	}
	iq := make([]complex64, pairs)
	dec, _ := siglab.FormatF32.Decoder()
	dec(raw[:pairs*8], iq)

	// Control channel: collect the locked NAC + NID/TSBK yield breakdown.
	bus := events.NewBus(1024)
	sub := bus.Subscribe()
	nacs := make(map[uint16]int)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for ev := range sub.C {
			if ev.Kind != events.KindCCLocked {
				continue
			}
			if ls, ok := ev.Payload.(p25phase1.LockState); ok {
				nacs[ls.NAC]++
			}
		}
	}()
	cc := p25phase1.New(p25phase1.Options{
		Bus:        bus,
		Log:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		SystemName: "p25-metrics-test",
		Rotations:  rotations,
	})

	// Receiver with the metrics taps wired in.
	var recovered []uint8
	var soft []float32
	var syms []complex64
	rx := p25phase1rx.New(p25phase1rx.Options{
		SampleRateHz: float64(meta.SampleRateHz),
		DeviationHz:  1800.0,
		DemodMode:    demodMode,
		SoftSink:     func(s []float32) { soft = append(soft, s...) },
		SymbolSink:   func(c []complex64) { syms = append(syms, c...) },
		DibitSink: func(d []uint8, baseIdx int) {
			recovered = append(recovered, d...)
			cc.Process(d, baseIdx)
		},
	})

	const chunk = 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		rx.Process(iq[i:end])
	}
	sub.Close()
	<-done

	// EVM + SNR from the demod taps (constellation for CQPSK, soft eye for C4FM).
	const warmupSkip = 256
	var evm, snrEst float64
	var modName string
	if demodMode == p25phase1rx.DemodCQPSK {
		modName = "cqpsk"
		if len(syms) > warmupSkip {
			syms = syms[warmupSkip:]
		}
		evm = metrics.EVMConstellation(syms)
		snrEst = metrics.SNRM2M4Constellation(syms)
	} else {
		modName = "c4fm"
		if len(soft) > warmupSkip {
			soft = soft[warmupSkip:]
		}
		outer := metrics.EstimateOuterRailC4FM(soft)
		evm = metrics.EVMC4FM(soft, outer)
		snrEst = metrics.SNRResidualC4FM(soft, outer)
	}

	// FSW sync-margin distribution: run a standalone detector over the
	// recovered dibits and fold the per-hit margins into the collector.
	var coll metrics.Collector
	det := p25phase1.NewSyncDetector(-1) // default tolerance 4
	det.SetRotations(rotations)
	_, _, margins, _ := det.ProcessWithMargin(nil, nil, nil, recovered, 0)
	for _, m := range margins {
		coll.AddSyncMargin(m)
	}
	sum := coll.Summarize()

	stats := cc.Stats()
	t.Logf("p25 real-capture metrics (%s, %s):", cfilePath, modName)
	t.Logf("  EVM=%.1f%%  SNR≈%.1f dB  (soft/sym n=%d)", evm, snrEst, len(soft)+len(syms))
	t.Logf("  sync margin: hits=%d  min=%d  median=%d  mean=%.2f", sum.SyncHits, sum.SyncMarginMin, sum.SyncMarginMedian, sum.SyncMarginMean)
	t.Logf("  NID: trusted=%d marginal=%d failed=%d", stats.NIDTrusted, stats.NIDMarginal, stats.NIDFailed)
	t.Logf("  TSBK: decoded=%d trellis-failed=%d crc-failed=%d", stats.TSBKDecoded, stats.TSBKTrellisFailed, stats.TSBKCRCFailed)
	t.Logf("  locked NACs: %v", nacs)

	// Assert the bounds the metadata declares (all optional).
	if exp.MinNIDTrusted > 0 && stats.NIDTrusted < int64(exp.MinNIDTrusted) {
		t.Errorf("NID trusted = %d, want ≥ %d", stats.NIDTrusted, exp.MinNIDTrusted)
	}
	if exp.MinTSBK > 0 && stats.TSBKDecoded < int64(exp.MinTSBK) {
		t.Errorf("TSBK decoded = %d, want ≥ %d", stats.TSBKDecoded, exp.MinTSBK)
	}
	if exp.MaxEVMPct > 0 && evm > exp.MaxEVMPct {
		t.Errorf("EVM = %.1f%%, want ≤ %.1f%%", evm, exp.MaxEVMPct)
	}
	if exp.MinSNRdB > 0 && snrEst < exp.MinSNRdB {
		t.Errorf("SNR estimate = %.1f dB, want ≥ %.1f dB", snrEst, exp.MinSNRdB)
	}
	if exp.MinSyncMargin > 0 && sum.SyncHits > 0 && sum.SyncMarginMin < exp.MinSyncMargin {
		t.Errorf("min sync margin = %d, want ≥ %d", sum.SyncMarginMin, exp.MinSyncMargin)
	}
	if exp.NAC != "" {
		want, err := parseHex16(exp.NAC)
		if err != nil {
			t.Fatalf("metadata expected.nac=%q: %v", exp.NAC, err)
		}
		if nacs[want] == 0 {
			t.Errorf("expected NAC %#x never locked; saw %v", want, nacs)
		}
	}
}
