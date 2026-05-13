# Voice decoder calibration

GopherTrunk's pure-Go IMBE (P25 P1) and AMBE+2 (DMR, NXDN, dPMR,
D-STAR) decoders produce intelligible end-to-end audio. The remaining
polish work is **absolute-level calibration**: tune the AGC `TargetPeak`
in [`internal/voice/mbe/agc.go`](../internal/voice/mbe/agc.go) so the
in-tree decoders' loudness matches the reference output from
DSD-FME or OP25. This document is the operator-facing recipe.

## What's already shipping

- The comparison harness at
  [`internal/voice/calibrate`](../internal/voice/calibrate/) reads a
  `.raw` vocoder-frame stream and a reference `.wav` and computes
  RMS-ratio (dB) + best-alignment normalised cross-correlation.
- The vendor-extension hook
  [`ambe2.SetKnoxTone(b1, freqA, freqB)`](../internal/voice/ambe2/knox.go)
  lets operators register per-vendor knox / call-alert dual-tone
  pairs (b1 ∈ [144, 163]) the public AMBE+2 spec doesn't document.
- The wrapper CLI [`cmd/voice-calibrate`](../cmd/voice-calibrate/main.go)
  exposes `calibrate.Compare` so a one-off check doesn't require a
  test.

## End-to-end recipe

### 1. Capture a reference call

Edit `config.yaml` to enable raw-frame recording:

```yaml
recordings:
  dir: ./recordings
  write_raw: true
```

Tune the daemon to a P25 P1 system (for IMBE) or a DMR / NXDN / dPMR
/ D-STAR system (for AMBE+2). Record a 5+ second voice call:

```sh
./bin/gophertrunk run -config config.yaml
# wait for a voice call, then Ctrl+C
```

The daemon writes two files under `recordings/<system>/<tg>/`:

- `<UTC>_src<id>.wav` — in-tree decoder output (8 kHz mono 16-bit
  PCM).
- `<UTC>_src<id>.raw` — per-frame compressed vocoder stream
  (11 bytes/frame for IMBE, 7 bytes/frame for AMBE+2).

### 2. Decode through DSD-FME (or OP25)

Run the **same** `.raw` through an external reference decoder:

```sh
# DSD-FME (https://github.com/lwvmobile/dsd-fme)
dsd-fme -r <call>.raw -o reference.wav

# OP25 (https://github.com/osmocom/op25)
# (see OP25's docs for mbe-decode invocation against a .raw)
```

DSD-FME's `-r` mode reads the byte-packed AMBE+2 / IMBE frames
directly and writes 8 kHz mono 16-bit PCM, matching the in-tree
calibrate harness's expected WAV format.

### 3. Run the calibration

Either drop the two files into the testdata directory and run the
unit test, or use the CLI for a one-off check:

```sh
# Option A: in-tree test
cp <call>.raw   internal/voice/imbe/testdata/p25-p1-voice.raw
cp reference.wav internal/voice/imbe/testdata/p25-p1-voice-dsdfme.wav
go test ./internal/voice/calibrate/ -v -run TestCompareIMBE

# Option B: one-off CLI
go run ./cmd/voice-calibrate \
    -raw      <call>.raw \
    -ref-wav  reference.wav \
    -vocoder  imbe
```

The CLI prints the `calibrate.Result` fields (RMSRatioDb, PeakXcorr,
LagSamples, sample counts). Acceptance criteria:

- `|RMSRatioDb| < 3.0` — loudness offset under ±3 dB.
- `PeakXcorr > 0.85` — waveform similarity 85%+ at best lag.

### 4. Tune if the thresholds miss

A failing RMSRatioDb means the in-tree AGC's `TargetPeak` is off.
[`internal/voice/mbe/agc.go`](../internal/voice/mbe/agc.go) holds
the knob; lowering `TargetPeak` quietens the in-tree decoder
relative to the reference and vice versa.

A failing PeakXcorr (with a clean RMSRatio) means the synthesis
path itself is producing a different waveform. That's deeper than a
gain knob — check the spectral envelope decoder
([`internal/voice/mbe/synth.go`](../internal/voice/mbe/synth.go))
and the prediction-residual gain path.

## Knox / call-alert tones

If your captured call contains AMBE+2 knox tones (b1 ∈ [144, 163]),
the in-tree decoder routes those frames through silence by default
because the AMBE+2 spec doesn't document their frequencies publicly.
That's not a calibration failure — it's the documented contract.

Operators with a per-vendor reference (Motorola Trbo, Hytera,
generic) can register the (freqA, freqB) pair via
`ambe2.SetKnoxTone` before running the calibration:

```go
import "github.com/MattCheramie/GopherTrunk/internal/voice/ambe2"

func init() {
    // Example: register a hypothetical Motorola Trbo call-alert
    // tone for b1 = 150.
    _ = ambe2.SetKnoxTone(150, 1100, 1750)
}
```

After registration, the matching tone frames synthesise as
summed-sinewave dual-tones (identical synthesis path to DTMF).

## Where to drop fixtures

Per-vocoder testdata directories:

- `internal/voice/imbe/testdata/` — IMBE fixtures
  ([README](../internal/voice/imbe/testdata/README.md))
- `internal/voice/ambe2/testdata/` — AMBE+2 fixtures
  ([README](../internal/voice/ambe2/testdata/README.md))

Both READMEs document the file naming the calibrate tests expect.
Tests `t.Skip` when files are absent; CI stays green.
