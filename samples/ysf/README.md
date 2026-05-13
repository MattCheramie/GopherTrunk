# YSF (Yaesu System Fusion) captures

Drop **YSF DN-mode** voice/data IQ recordings here to **validate**
the on-air FICH interleaver / puncture schedule shipped in
[`internal/radio/ysf/fich_trellis.go`](../../internal/radio/ysf/fich_trellis.go)
against a real Yaesu transmission. The spec-level codec
(`EncodeFICHOnAir` / `DecodeFICHOnAir`) now ships per the MMDVMHost
reference; this directory exists so a real capture can confirm the
choice or trigger the alternate-schedule swap below.

> **Note**: audio-only recordings (MP3, post-FM-demod WAV) cannot
> validate the YSF decoder. YSF is 4-level FSK and the
> constellation amplitude doesn't survive a discriminator-output
> recording (let alone MP3 compression). The earlier
> `Yaesu_sys_fusion.wav` upload was removed for this reason. **IQ
> recordings only** (stereo 16-bit WAV with I/Q separated, or
> `.cfile` / `.bin`) are usable.

## Capture format

| Property | Expected value |
| --- | --- |
| File format | Complex float32 IQ (`*.cfile`) or complex int16 (`*.bin`) |
| Sample rate | Any rate ≥ 48 kHz; 48 kHz nominal |
| Modulation | C4FSK at 4800 symbols/s, ±2700 Hz peak deviation |
| Channel width | 12.5 kHz |
| Centre | Tuned on the YSF carrier |
| Duration | ≥ 10 seconds — captures enough FICH cycles to validate the interleave schedule |

## Metadata schema

```json
{
  "source": "MMDVMHost / Pi-Star reflector @ <reflector>",
  "tool_cross_check": "DSDcc 1.9.5 in YSF mode",
  "mode": "DN",
  "expected": {
    "callsign": "N0CALL",
    "destination": "ALL",
    "fich_sequence": [
      { "frame": 0, "ft": 0, "dt": 1, "fn": 0, "ct": 1 },
      { "frame": 1, "ft": 0, "dt": 1, "fn": 1, "ct": 1 },
      { "frame": 2, "ft": 0, "dt": 1, "fn": 2, "ct": 1 }
    ]
  },
  "notes": "FICH fields ft/dt/fn/ct per Yaesu Common Air Interface §3.3"
}
```

## Why a capture is needed

[`internal/radio/ysf/fich_trellis.go`](../../internal/radio/ysf/fich_trellis.go)
ships:

- K=5 ½-rate trellis encoder/decoder — round-trips cleanly in unit
  tests.
- `EncodeFICHOnAir` / `DecodeFICHOnAir` — full on-air codec with
  puncture positions `{0, 1, 102, 103}` and column-major 10×10
  interleave (`out[k] = depunctured[(k%10)*10 + (k/10)]`), per the
  MMDVMHost / DSDcc / Pi-Star reference. `TestFICHOnAirRecoversFromSingleBitFlip`
  exhaustively confirms every one of the 100 on-air bit positions
  is Viterbi-corrected.

What's still pending is **empirical confirmation** that real Yaesu
hardware uses the same schedule. Published references converge on
MMDVMHost's table; if a captured FICH passes through
`DecodeFICHOnAir` and fails `ParseFICH`'s CRC check, swap to the
alternate schedule:

### Alternate schedule (DSDcc fallback)

If MMDVMHost's table doesn't decode the capture, edit
`internal/radio/ysf/fich_trellis.go` and:

1. **Puncture positions** — swap
   `fichPuncturePositions = [4]int{0, 1, 102, 103}` for
   `[4]int{0, 51, 52, 103}` (DSDcc's spread-puncture variant), then
   re-run `TestFICHPuncturePositionsExactly4` to confirm the
   strictly-increasing invariant.
2. **Interleave permutation** — replace the
   `fichInterleavePerm[k] = (k%10)*10 + (k/10)` formula with
   `(k%4)*25 + (k/4)` (4-column variant) and verify
   `TestFICHInterleavePermBijective` still passes.

If neither schedule decodes, the capture is the source of truth —
publish the polynomial choice (whether the K=5 generator pair is
`(0o23, 0o35)` or `(0o31, 0o27)`) in this file's metadata block so
the next iteration knows which variant to start from.

## Recommended sources

- **Pi-Star / MMDVMHost** dashboard with FICH logging enabled.
- **A controlled transmission** from a YSF-capable HT (e.g.,
  FT-70D, FTM-300) with the FICH header fields known in advance.
