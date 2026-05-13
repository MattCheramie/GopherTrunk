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

## Acceptance criteria

A capture is considered "validating" when:

1. **Lock latency.** Replayed through `newTETRAPipeline`, the daemon
   publishes `events.KindCCLocked` within **5 seconds wall time** of
   the first burst arriving. The exact wiring lives at
   [`cmd/gophertrunk/integration_cc_tetra_test.go`](../../cmd/gophertrunk/integration_cc_tetra_test.go);
   add a `_realair_test.go` sibling that points the mock SDR at the
   capture file and asserts the timeout.
2. **Frame recovery rate.** ≥ 90% of bursts that pass the
   §8.3.1 CRC-16 verify when re-encoded round-trip on the in-tree
   chain must also pass when decoded straight from the capture.
3. **Viterbi correction-depth histogram.** The
   `gophertrunk_tetra_viterbi_corrections` Prometheus histogram
   (gated by `metrics.detailed_fec: true`; new metric in
   `internal/metrics/metrics.go`) reports the bit-error count per
   recovered frame. Acceptable margin: p95 ≤ 8 bit errors per
   116-bit logical channel, p99 ≤ 12. Captures with co-channel
   interference will sit higher; that's the signal the calibration
   work was designed to surface.

The new Prometheus histogram is **not yet wired** — it lands
together with the first real-air capture validating it. Implementing
the histogram before a capture exists would risk emitting metrics
nobody can interpret.

## Recommended sources

- **telive / osmo-tetra** — produces both IQ recordings and a
  decoder log GopherTrunk can cross-check against.
- **A TETRA Direct Mode Operation (DMO)** test transmission from
  a controlled radio — easiest to label.

⚠️  TETRA captures may contain encrypted traffic (TEA1/TEA2/TEA3/
TEA4). Cleartext frames only are needed for the recovery-margin
work; encryption-key recovery is **out of scope**.
