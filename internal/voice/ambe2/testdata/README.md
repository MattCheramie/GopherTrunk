# AMBE+2 calibration fixtures

Drop zone for AMBE+2 reference data the
[`internal/voice/calibrate`](../../calibrate/) harness compares the
in-tree `internal/voice/ambe2` decoder against. Tests that depend on
these fixtures `t.Skip` when the files aren't present, so CI stays
green until reference data lands.

## Required files

| File                          | Format                                    |
| ----------------------------- | ----------------------------------------- |
| `dmr-voice.raw`               | Raw AMBE+2 frames, 7 bytes per frame, packed MSB-first, no header |
| `dmr-voice-dsdfme.wav`        | DSD-FME-decoded reference WAV (8 kHz, 16-bit, mono PCM, RIFF/WAVE) |

Both files must come from the **same** original DMR (or NXDN /
dPMR / D-STAR — anything that uses AMBE+2) call so the calibrate
harness's RMS + cross-correlation comparison is meaningful.

## Capture recipe

See [`docs/voice-calibration.md`](../../../../docs/voice-calibration.md)
for the end-to-end recipe — record with
`recordings.write_raw: true`, then push the `.raw` sidecar through
DSD-FME / OP25 to produce the reference WAV. The shipping
`cmd/voice-calibrate` CLI wraps `calibrate.Compare` so operators
don't have to write a test to drive a one-off check.

## Frame layout

AMBE+2 packs 49 quantised parameter bits per 20 ms frame into 7
bytes (the 8th bit of each byte's MSB-first stream stays zero
padding). The wire layout is the AMBE+2 spec's `info` section —
the per-frame protection bits the spec wraps it in for radio
transmission are stripped by the upstream radio decoder before
the `.raw` sidecar is written.

GopherTrunk's `.raw` sidecar layout is byte-aligned: each 7-byte
record is one AMBE+2 frame in MSB-first packed order. DSD-FME and
OP25 both accept this layout directly.

## Validation

Once the files are in place, run:

```sh
go test ./internal/voice/calibrate/ -v -run TestCompareAMBE2
```

Acceptance criteria match the calibrate package docs:

- `|Result.RMSRatioDb| < 3.0` — loudness offset under ±3 dB.
- `Result.PeakXcorr > 0.85` — waveform similarity 85%+ at the best
  cross-correlation lag.

## Knox / call-alert tones

Knox / call-alert dual-tones (AMBE+2 `b1 ∈ [144, 163]`) are
vendor-specific. The public AMBE+2 spec doesn't document their
frequencies; without registration via
[`ambe2.SetKnoxTone`](../knox.go) those indices decode as silence.

If your calibration fixtures contain knox tones, register the
per-vendor table before running the test:

```go
ambe2.SetKnoxTone(150, 1100, 1750) // example: Motorola Trbo call alert "1"
```

Without overrides, the calibrate test against a fixture that
contains knox tones will show an RMS offset and reduced
cross-correlation in those regions — that's expected behaviour, not
a regression.
