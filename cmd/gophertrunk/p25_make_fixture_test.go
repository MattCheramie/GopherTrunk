//go:build integration

package main

import (
	"os"
	"strconv"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// TestGenerateP25Fixture channelises a wideband real-air P25 Phase 1 capture
// down to the production 48 kHz baseband and writes it as a small GNU Radio
// f32 cfile, so a multi-megabyte 2 MSPS recording becomes a commit-sized
// regression fixture (the shape of testdata/mmr-s9-cc.cfile / mmr-city-cc.cfile).
//
// It is a one-shot operator tool, not a CI gate: it only runs under the
// `integration` build tag AND when P25_GEN_SRC points at the raw capture, so
// the normal suite never touches it. The down-conversion reuses the exact DDC
// the live/replay pipeline uses (ccdecoder.NewDownconverterWithOffset →
// dsp.Resampler, rational L/M to land on 48000 Hz exactly), so the fixture is
// channelised identically to how the receiver would see the signal on the air.
//
// The default output lands in samples/p25/ — which is git-ignored (capture
// binaries are never committed, see samples/.gitignore). So this regenerates
// the local capture the demod-quality harness (TestReplayP25RealCaptureMetrics)
// grades against; the committed record of that grading is the metadata sidecar.
//
// Usage:
//
//	P25_GEN_SRC=/path/to/p25_450500center_2msps.cfile \
//	P25_GEN_OFFSET_HZ=-625000 \
//	go test -tags integration -run TestGenerateP25Fixture ./cmd/gophertrunk/
//
// Defaults: offset -625000 Hz (the 449.875 MHz control channel inside a
// 450.500 MHz-centred 2 MSPS capture), output ../../samples/p25/p25-450875-cc.cfile.
func TestGenerateP25Fixture(t *testing.T) {
	src := os.Getenv("P25_GEN_SRC")
	if src == "" {
		t.Skip("P25_GEN_SRC unset — set it to a raw 2 MSPS P25 .cfile to (re)generate the fixture")
	}
	out := os.Getenv("P25_GEN_OUT")
	if out == "" {
		out = "../../samples/p25/p25-450875-cc.cfile"
	}
	const (
		inRateHz  = 2_000_000.0
		targetHz  = 48_000.0
		defOffset = -625_000.0 // 449.875 MHz CC within a 450.500 MHz-centred capture
	)
	offsetHz := defOffset
	if s := os.Getenv("P25_GEN_OFFSET_HZ"); s != "" {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			t.Fatalf("P25_GEN_OFFSET_HZ=%q: %v", s, err)
		}
		offsetHz = v
	}

	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	pairs := len(raw) / 8
	if pairs == 0 {
		t.Fatalf("%s has no samples", src)
	}
	iq := make([]complex64, pairs)
	dec, _ := siglab.FormatF32.Decoder()
	dec(raw[:pairs*8], iq)

	ddc := ccdecoder.NewDownconverterWithOffset(inRateHz, targetHz, offsetHz)
	var chan48 []complex64
	var scratch []complex64
	const chunk = 65536
	for off := 0; off < len(iq); off += chunk {
		end := off + chunk
		if end > len(iq) {
			end = len(iq)
		}
		scratch = ddc.Process(scratch, iq[off:end])
		chan48 = append(chan48, scratch...) // copy out: scratch is reused next chunk
	}
	if len(chan48) == 0 {
		t.Fatalf("down-conversion produced no samples")
	}

	if err := siglab.WriteCapture(out, chan48, siglab.FormatF32); err != nil {
		t.Fatalf("write %s: %v", out, err)
	}
	t.Logf("wrote %s: %d samples @ %.0f Hz (%.2f s, %d bytes) from %s offset=%.0f Hz",
		out, len(chan48), ddc.OutRateHz(), float64(len(chan48))/ddc.OutRateHz(), len(chan48)*8, src, offsetHz)
}
