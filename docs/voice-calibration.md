---
layout: page
title: Voice calibration
description: Tuning AGC TargetPeak so pure-Go IMBE / AMBE+2 loudness matches DSD-FME and OP25
nav_group: Reference
---

# Voice decoder calibration

GopherTrunk's pure-Go IMBE (P25 P1) and AMBE+2 (DMR, NXDN)
decoders produce intelligible end-to-end audio. The remaining
polish work is **absolute-level calibration**: tune the AGC `TargetPeak`
in [`internal/voice/mbe/agc.go`](https://github.com/MattCheramie/GopherTrunk/blob/main/internal/voice/mbe/agc.go) so the
in-tree decoders' loudness matches the reference output from
DSD-FME or OP25. This document is the operator-facing recipe.

## What's already shipping

- The comparison harness at
  [`internal/voice/calibrate`](https://github.com/MattCheramie/GopherTrunk/tree/main/internal/voice/calibrate/) reads a
  `.raw` vocoder-frame stream and a reference `.wav` and computes
  RMS-ratio (dB) + best-alignment normalised cross-correlation.
- The RMS + cross-correlation math is exposed separately as
  [`calibrate.CompareSamples([]int16, []int16) Result`](https://github.com/MattCheramie/GopherTrunk/blob/main/internal/voice/calibrate/calibrate.go)
  for callers that already have PCM in memory — and so the
  math is unconditionally test-covered against synthetic
  fixtures (e.g. a +3 dB-louder reference must produce
  `RMSRatioDb = −3.0 ± 0.5`). The skip-gated `Compare` tests
  for the actual vocoders still wait on captured reference
  WAVs landing under `internal/voice/{imbe,ambe2}/testdata/`,
  but a regression in the loudness / similarity math now
  fails CI without that step.
- The vendor-extension hook
  [`ambe2.SetKnoxTone(b1, freqA, freqB)`](https://github.com/MattCheramie/GopherTrunk/blob/main/internal/voice/ambe2/knox.go)
  lets operators register per-vendor knox / call-alert dual-tone
  pairs (b1 ∈ [144, 163]) the public AMBE+2 spec doesn't document.
  For curated tables, the bundle API
  [`ambe2.RegisterPreset(KnoxPreset)`](https://github.com/MattCheramie/GopherTrunk/blob/main/internal/voice/ambe2/knox.go)
  registers many entries at once with a name surfacing through
  `ambe2.ListPresets()` for diagnostics.
- The wrapper CLI [`cmd/voice-calibrate`](https://github.com/MattCheramie/GopherTrunk/blob/main/cmd/voice-calibrate/main.go)
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

Tune the daemon to a P25 P1 system (for IMBE) or a DMR / NXDN
system (for AMBE+2). Record a 5+ second voice call:

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
[`internal/voice/mbe/agc.go`](https://github.com/MattCheramie/GopherTrunk/blob/main/internal/voice/mbe/agc.go) holds
the knob; lowering `TargetPeak` quietens the in-tree decoder
relative to the reference and vice versa.

A failing PeakXcorr (with a clean RMSRatio) means the synthesis
path itself is producing a different waveform. That's deeper than a
gain knob — check the spectral envelope decoder
([`internal/voice/mbe/synth.go`](https://github.com/MattCheramie/GopherTrunk/blob/main/internal/voice/mbe/synth.go))
and the prediction-residual gain path. For DMR (AMBE+2) specifically —
especially rough **female** voices — see
[DMR voice quality](dmr-voice-quality.md), which documents a mbelib
reconciliation review and a §6.2 spectral-enhancement divergence that
this same PeakXcorr A/B is the gate for.

## Reading P25 Phase 1 decode quality

Each P25 voice chain logs a rolling `composer: p25p1 decode quality`
line. Beyond `corrected_bit_errs` (IMBE FEC) it now reports the outer
Reed-Solomon health of the signalling words:

- `lc_rs_uncorrectable` — LDU1 Link Control words the RS(24,12,13) layer
  could not recover. The talkgroup is read from the LC, so a high count
  means talkgroups can't be trusted and the recorder's talkgroup gating
  may drop or split audio.
- `ess_rs_uncorrectable` — LDU2 Encryption Sync words the RS(24,16,9)
  layer could not recover (would otherwise surface garbage algorithm IDs).

Both rising together is the FEC-layer signature of marginal RF: raise or
lower the voice SDR gain and re-check. Note some SDRs (e.g. UHD/SoapyRemote
front-ends) reject `set_rx_agc` — the daemon logs `disable agc not
applied` and the gain must then be set manually rather than relying on AGC.

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
  ([README](https://github.com/MattCheramie/GopherTrunk/blob/main/internal/voice/imbe/testdata/README.md))
- `internal/voice/ambe2/testdata/` — AMBE+2 fixtures
  ([README](https://github.com/MattCheramie/GopherTrunk/blob/main/internal/voice/ambe2/testdata/README.md))

Both READMEs document the file naming the calibrate tests expect.
Tests `t.Skip` when files are absent; CI stays green.
