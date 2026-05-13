# TETRA captures

Drop **ETSI TETRA TMO** downlink IQ recordings here to profile the
on-air recovery margins of the §8.3.1 channel-coding chain
(descramble + deinterleave + depuncture + Viterbi + CRC-16 verify
+ tail strip) shipping in `internal/radio/tetra/`.

## Capture format

| Property | Expected value |
| --- | --- |
| File format | Complex float32 IQ (`*.cfile`) or complex int16 (`*.bin`) |
| Sample rate | Any rate ≥ 36 kHz; 36 kHz nominal |
| Modulation | π/4-DQPSK at 18 ksym/s |
| Channel width | 25 kHz |
| Centre | Tuned on the BS downlink carrier |
| Duration | ≥ 30 seconds — captures multiple SCH/HD, SCH/F, BSCH frames + an idle window for noise floor profiling |

## Metadata schema

```json
{
  "source": "Live TETRA TMO downlink @ <city, country>",
  "tool_cross_check": "telive 1.5 / osmo-tetra",
  "expected": {
    "mcc": 901,
    "mnc": 16383,
    "colour_code": 53,
    "ts1_mac_resource_pdus": [
      { "address": "ssi=1234", "downlink_assign": "yes" }
    ]
  },
  "snr_estimate_db": 18.5,
  "co_channel_interference": "none",
  "notes": "Co-channel + adjacent-channel interference scenarios welcome — they're what the Viterbi recovery margin profiling needs"
}
```

## Why captures are needed

[`docs/opt-in-features.md`](../../docs/opt-in-features.md) §5 flags
"on-air recovery margins" as the remaining TETRA work. Unit tests
already round-trip clean fixtures end-to-end; what's missing is
**measuring how the §8.3.1 Viterbi decoder behaves under real
co-channel + adjacent-channel interference** — the synthesized
fixtures don't model the burst-error structure live RF produces.

The captures here will feed a future
`internal/radio/tetra/recovery_margin_test.go` that asserts:

- the §8.3.1 chain recovers ≥ X% of frames at Y dB SNR,
- the Viterbi correction-depth metric stays within spec under
  interference conditions Z.

## Recommended sources

- **telive / osmo-tetra** — produces both IQ recordings and a
  decoder log GopherTrunk can cross-check against.
- **A TETRA Direct Mode Operation (DMO)** test transmission from
  a controlled radio — easiest to label.

⚠️  TETRA captures may contain encrypted traffic (TEA1/TEA2/TEA3/
TEA4). Cleartext frames only are needed for the recovery-margin
work; encryption-key recovery is **out of scope**.
