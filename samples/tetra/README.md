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

### What a second live capture set taught us (468.5 MHz, 1 Msps)

A second batch of live downlink captures (seven `*.cfile` clips,
≈3.3 s each, 1 Msps, centre 468.475 MHz; carrier ≈119 kHz off centre)
was replayed through the production pipeline. They are **not** committed
(per `samples/.gitignore`), but they surfaced two concrete findings:

1. **The front-end was spectrum-inverted (issue #264).** Raw, the
   π/4-DQPSK differential decode is garbled (dibit histogram ≈
   34/29/18/19 %). With the spectrum de-inverted (`gophertrunk replay
   -conjugate …`, or `iq_invert: true` on the device) the dibits snap
   to a uniform 25/25/25/25 % and the **normal** training sequence
   (NTS1) correlates at Hamming distance **0** over hundreds of bursts —
   i.e. the normal-burst demod is provably clean once the spectrum is
   corrected.

2. **A real-air decoder bug: the colour-code recovery was
   rotation-blind.** π/4-DQPSK carries data in the differential phase,
   so a residual carrier offset rotates the whole demodulated dibit
   stream by an unknown 0..3 (these captures sat at rotation 1). The
   synchronisation-training-sequence (STS) correlator that gates BSCH
   recovery matched only the rotation-0 orientation, so on air it never
   fired — which is almost certainly why the reference carrier above
   recovered no colour code either. `RecoverColourCode` and the live
   `processSB` STS detection are now tried under all four rotations
   (regression: `TestRecoverColourCodeUnderRotation`,
   `TestProcessLearnsColourCodeFromSBBurstUnderRotation`).

Even with both addressed these particular clips still do **not** lock:
the synchronisation burst itself never demodulates cleanly (best STS
match ≈5/19 mismatches vs. 0/19 for the normal bursts, and the
near-misses do not sit on the ~18 360-symbol multiframe grid). Each clip
is only ≈3.3 s — a synchronisation burst recurs only once per ≈1.02 s
multiframe — so they span too few SB opportunities, and the SB region
appears further disrupted (likely the frequency-correction field pulling
the carrier/timing loop). A **≥30 s** cleartext capture that includes
several clean frame-18 SB slots is still what closes the lock + Viterbi
histogram criteria.

### What the 15 Jul capture set taught us (469.875 MHz, 2.5 MS/s)

A third contribution — a `gophertrunk`-bundle downlink capture at
469.875 MHz plus SDR++ side recordings — was replayed through the
production pipeline. The set:

- a bundle DDC slice (`.slice.wav`, 144 kHz IQ, 15 s),
- an SDR++ **baseband** WAV (39062.5 Hz IQ = 2.5 MS/s ÷ 64, 90 s),
- an SDR++ **audio** WAV (48 kHz mono — already FM/PSK-demodulated
  audio, not IQ, so not decodable).

None are committed (`samples/.gitignore` excludes `*.wav`), but they
surfaced two findings.

1. **The signal is modulation-degraded — it does not lock.** On the
   144 kHz slice the receiver acquires stable symbol timing (17954 baud)
   and correlates the normal training sequence on the correct 255-dibit
   TDMA grid (NTS1 at Hamming distance 1, ~129 hits in 15 s), but the
   demodulated dibit histogram is skewed (~37/18/17/28 % vs. the ideal
   25/25/25/25), the constellation EVM is ≈22 %, and **every** SCH/HD
   burst fails the §8.3.1 CRC-16. Over the full 90 s baseband capture
   (1189 normal bursts, resampled to 144 kHz + CFO-corrected + spectrum
   de-inverted) the chain recovers effectively **zero** genuine
   CRC-valid frames — a single pass in 1189 is at the ~1.8 % random
   CRC-16 collision rate. This is the same "the degradation is baked
   into the captured samples" signature as issues #764/#771: no receiver
   change recovers it, so **no `cc.locked` is claimed** and no
   `.metadata.json` sidecar ships (the skip-gated realair test stays
   dormant). A cleaner cleartext downlink is still what's needed.

2. **It exposed a real low-rate-replay bug (now fixed).** The 39062.5 Hz
   baseband WAV decoded at **+8.5 % baud (19531 vs. 18000)** with a
   broken TDMA grid, because `ccdecoder.Downconverter` only *decimated*:
   a pre-channelized capture below the 144 kHz TETRA target was passed
   through un-resampled, and the receiver rounded the resulting
   fractional 2.17 samples/symbol to 2. The down-converter now
   **interpolates** a sub-target stream up to the per-protocol channel
   rate so the receiver always gets its designed ~8 sps (see
   `internal/scanner/ccdecoder/ddc.go`; regression tests
   `TestDownconverterUpsample*` and `TestDDCUpsamplesSubTargetTETRAToLock`
   in that package, plus `TestDaemonCCDecodesTETRAAt48kHz`). This is why
   the committed 48 kHz `TETRA IQ.wav` and any SDR++ baseband WAV now
   feed the receiver at the right baud.

On the carrier AFC (`internal/radio/tetra/receiver/afc.go`): a follow-up
investigation with controlled fixtures found it **sound**, not the
weak link the residual originally looked like. Its per-block feed-forward
4·Δφ estimator drives a constant offset to a **~1 Hz** residual on a
clean signal and **~3 Hz at 6 dB SNR**, and still locks under linear
drift (±1.5 kHz across the capture) and per-symbol phase noise
(400 Hz-rms). The earlier "misfire on resampled IQ" was the DDC's short
up-sampling prototype (images collapsing a dibit class), fixed above by
`ddcUpMinTaps` — not the AFC. So the 15 Jul slice's ~700–900 Hz offset
is well inside what the AFC removes, and its non-lock is the baked-in
modulation degradation, consistent with #764 rather than a carrier bug.

The AFC's one real boundary is its **±2250 Hz pull-in ceiling** — the
4·Δφ estimator folds larger offsets into ±π/4 per symbol, so a control
channel sitting more than ~2.25 kHz off centre after tuning does not
lock (verified: a clean 3 kHz offset fails). That gross offset is
nominally the tuner/DDC's job (`-tune-hz` / `-auto-tune` / `iq_invert`),
so it is a documented design boundary, not a defect. Extending it would
need a coarse frequency-acquisition stage ahead of the fine estimator
computed on the pre-matched-filter signal (the matched RRC, centred at
0 Hz, biases any coarse estimate taken after it toward 0) — a
self-contained enhancement worth its own change, but one that would not
have altered the 15 Jul result.

## Recommended sources

- **telive / osmo-tetra** — produces both IQ recordings and a
  decoder log GopherTrunk can cross-check against.
- **A TETRA Direct Mode Operation (DMO)** test transmission from
  a controlled radio — easiest to label.

⚠️  TETRA captures may contain encrypted traffic (TEA1/TEA2/TEA3/
TEA4). Cleartext frames only are needed for the recovery-margin
work; encryption-key recovery is **out of scope**.
