# YSF (Yaesu System Fusion) captures

Drop **YSF DN-mode** voice/data IQ recordings here to unblock the
FICH interleaver / puncture schedule calibration documented in
[`docs/opt-in-features.md`](../../docs/opt-in-features.md) §5.

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

## Why a capture (not synthesized) is needed

[`internal/radio/ysf/fich_trellis.go`](../../internal/radio/ysf/fich_trellis.go)
ships a K=5 ½-rate trellis encoder + decoder that round-trips
cleanly in unit tests. What's missing is **calibration of the
exact on-air interleave + puncture schedule** — published
references (DSDcc, MMDVMHost, the Yaesu CAI document) all describe
slightly different schedules. A real-air capture pins which schedule
real Yaesu radios actually transmit.

Specific things the capture needs to disambiguate:

- the interleave permutation (`fich_interleave` candidates),
- the puncture mask (some references puncture 1-of-N, others 4-of-32),
- whether the K=5 polynomial is `(0o23, 0o35)` or `(0o31, 0o27)`.

## Recommended sources

- **Pi-Star / MMDVMHost** dashboard with FICH logging enabled.
- **A controlled transmission** from a YSF-capable HT (e.g.,
  FT-70D, FTM-300) with the FICH header fields known in advance.
