# IMBE calibration fixtures

This directory is the drop zone for IMBE reference data the
[`internal/voice/calibrate`](../../calibrate/) harness compares the
in-tree `internal/voice/imbe` decoder against. Tests that depend on
these fixtures `t.Skip` when the files aren't present, so CI stays
green until reference data lands.

## Required files

| File                          | Format                                    |
| ----------------------------- | ----------------------------------------- |
| `p25-p1-voice.raw`            | Raw IMBE frames, 11 bytes per frame, packed MSB-first, no header |
| `p25-p1-voice-dsdfme.wav`     | DSD-FME-decoded reference WAV (8 kHz, 16-bit, mono PCM, RIFF/WAVE) |

Both files must be derived from the **same** original P25 Phase 1
call so the calibrate harness's RMS + cross-correlation comparison
is meaningful. Capture both with a recorded radio session:

1. Tune GopherTrunk's daemon to a P25 P1 system with
   `recordings.write_raw: true` set in `config.yaml`.
2. Record a 5+ second voice call; the daemon writes both
   `<call>.wav` (pure-Go IMBE decode) and `<call>.raw` (the
   per-frame compressed IMBE stream) into
   `recordings/<system>/<tg>/`.
3. Run the **same** `.raw` through DSD-FME or OP25's `mbe-decode` to
   produce the reference WAV.
4. Drop the `.raw` into this directory as `p25-p1-voice.raw` and the
   DSD-FME WAV as `p25-p1-voice-dsdfme.wav`.

## Frame layout

IMBE has 144 bits per 20 ms frame. The wire-format spreads those
across 18 channel bits + 56 protection-coded bits + 70 quantised-
parameter bits. GopherTrunk's `.raw` sidecar packs the **pre-error-
correction wire bits** into 11 bytes (88 bits, the closest byte-
aligned envelope; the trailing 56 bits are not currently recorded —
the parameter section is what the decoder consumes). Decoders accept
either the 11-byte sidecar (in-tree) or the 18-byte uncoded
parameter section depending on which mode they target; both
DSD-FME and OP25 read the 11-byte sidecar GopherTrunk produces.

## Validation

Once the files are in place, run:

```sh
go test ./internal/voice/calibrate/ -v -run TestCompareIMBE
```

The test's acceptance criteria match the calibrate package's docs:

- `|Result.RMSRatioDb| < 3.0` — loudness offset under ±3 dB.
- `Result.PeakXcorr > 0.85` — waveform similarity 85%+ at the best
  cross-correlation lag.

Failing these signals the in-tree decoder's AGC `TargetPeak` (in
[`internal/voice/mbe/agc.go`](../../mbe/agc.go)) needs tuning, or
the synthesis path has a level offset the reference doesn't.

## Why a real capture matters

Pure-Go IMBE produces intelligible audio end-to-end, but the AGC
gain and final mixing depend on coefficient values that aren't
fully spec-locked. A reference capture decoded through both DSD-FME
and the in-tree decoder lets `calibrate.Compare` quantify any level
offset so the AGC can be tuned to match what operators expect.
