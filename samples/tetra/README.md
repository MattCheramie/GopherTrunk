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
   the first burst arriving. The wiring ships at
   [`cmd/gophertrunk/integration_cc_tetra_realair_test.go`](../../cmd/gophertrunk/integration_cc_tetra_realair_test.go)
   (build tag `integration`) — a skip-gated sibling of
   `integration_cc_tetra_test.go` that replays the baseband WAV
   natively through the `baseband-replay` SDR driver and asserts the
   lock plus the decoded MCC/MNC (which only descramble under the
   colour code learned from the BSCH, so a match validates the
   #648/#662 recovery on real RF). It runs as soon as a capture +
   `.metadata.json` pair is present and skips otherwise.
2. **Frame recovery rate.** ≥ 90% of bursts that pass the
   §8.3.1 CRC-16 verify when re-encoded round-trip on the in-tree
   chain must also pass when decoded straight from the capture.
3. **Viterbi correction-depth histogram.** The
   `gophertrunk_tetra_viterbi_corrections` Prometheus histogram
   (opt-in via `metrics.detailed_fec: true`; defined in
   [`internal/metrics/prom.go`](../../internal/metrics/prom.go))
   reports, per recovered BSCH / SCH-HD burst, the channel bits the
   FEC chain corrected — measured decoder-independently as the
   Hamming weight between the received bits and the re-encoded
   recovered frame, so it is identical on the hard and soft paths.
   Acceptable margin: p95 ≤ 8 bit errors, p99 ≤ 12. Captures with
   co-channel interference will sit higher; that's the signal the
   calibration work was designed to surface.

The histogram is **wired and unit-tested** against synthetic noisy
bursts (`internal/radio/tetra/fec_observer_test.go`), but it is
opt-in and off by default: the buckets only become interpretable once
a *validating* on-air capture exists to calibrate the p95/p99 margins
against, which is the whole point of dropping one here.

### Status of the committed reference capture

`TETRA IQ.wav` is a 48 kHz, 41.7 s baseband recording that the survey
classifier reads as a strong digital carrier (18 ksym/s π/4-DQPSK,
baud-line prominence ~94). It does **not** yet validate the decode
chain: demodulating the full capture through the production receiver
and scanning for a CRC-valid BSCH across a ±8 kHz frequency sweep (and
channel-filter / loop-gain variations) recovers **no** colour code, so
no `events.KindCCLocked` fires and the histogram never populates. It is
kept as a reference carrier (and as the survey-classifier regression
guard, `internal/survey/classify_realair_test.go`); a TMO downlink
capture whose SYNC bursts decode CRC-clean is still needed to close the
follow-up. Such a capture ships with a `.metadata.json` sidecar (schema
above) and the realair test above starts asserting against it
automatically.

## Recommended sources

- **telive / osmo-tetra** — produces both IQ recordings and a
  decoder log GopherTrunk can cross-check against.
- **A TETRA Direct Mode Operation (DMO)** test transmission from
  a controlled radio — easiest to label.

⚠️  TETRA captures may contain encrypted traffic (TEA1/TEA2/TEA3/
TEA4). Cleartext frames only are needed for the recovery-margin
work; encryption-key recovery is **out of scope**.
