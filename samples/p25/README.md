# P25 Phase 1 captures (demod-quality measurement)

Drop **P25 Phase 1** control-channel IQ recordings here to feed the
demodulator measurement harness
([`internal/radio/p25/phase1/metrics`](../../internal/radio/p25/phase1/metrics/)).
The synthetic sweep
([`internal/radio/p25/phase1/receiver/sweep_test.go`](../../internal/radio/p25/phase1/receiver/sweep_test.go))
benchmarks the demod against the theoretical limit; a capture dropped here is
the **field truth** — it runs the real signal through the receiver and reports
the pre-FEC error rate, EVM, estimated SNR, and FSW sync-margin so a weak-decode
report becomes a number you can act on.

**Status:** validated against a live UHF C4FM site (449.875 MHz, NAC 0x2C1) —
see `p25-450875-cc.metadata.json`. The harness graded it at EVM ≈ 12.7%,
SNR ≈ 14.5 dB, sync-margin min=3/median=5, NID trusted=31/failed=0, TSBK
decoded=36/CRC-failed=0. The `.cfile` itself is git-ignored (capture binaries
are never committed); the committed metadata sidecar carries the result, and
the capture is reproducible from the raw 2 MSPS recording via
`TestGenerateP25Fixture` (`cmd/gophertrunk/p25_make_fixture_test.go`).

Two captures are most valuable, and you can drop either or both:

- a **C4FM** site (the FM-discriminator path — the one the harness shows is the
  most noise-fragile), and
- an **LSM / CQPSK simulcast** site (the linear path).

## Weak-signal voice (LDU) captures — the priority ask

The captures above are control-channel recordings. The open **weak-signal voice**
investigation needs the other kind: a **P25 Phase 1 C4FM voice-channel** recording
of a *marginal* call — the scenario where a hardware radio (e.g. an Astro Spectra)
decodes clean audio but GopherTrunk recovers only a handful of IMBE frames on the
same antenna.

Why this specific capture matters: the default C4FM voice receiver has **no channel
equalizer and no soft-decision FEC** (a blind CMA/FSE equalizer exists only on the
opt-in `cqpsk`/LSM path). Those are the two levers that roughly doubled TETRA yield
on ISI-smeared / weak captures. Before porting either onto the C4FM path, we need a
real weak call to (a) measure the baseline pre-FEC EVM / SNR / LDU yield and (b) A/B
the change against — per the project rule that a green synthetic is not proof of an
on-air fix.

What to capture:

- Tune the **voice channel** (the granted frequency), not the control channel, during
  a call that is **weak / multipath-degraded** (the condition that fails today). A
  clean, strong call will not exercise the missing equalizer.
- C4FM (the FM-discriminator path). If the site is LSM/CQPSK, capture that too and set
  `demod_mode: "cqpsk"`.
- Include the same call decoded by the reference radio if possible (note it in
  `tool_cross_check`) so "hardware decodes, GT doesn't" is anchored to one signal.
- Set `"expected": { "demod_mode": "c4fm" }` and leave the quality bounds out at first
  (report-only); the harness logs EVM/SNR/LDU yield as the baseline to improve on.

Drop it here as `*.cfile` + `*.metadata.json` (same format as below); the binary is
git-ignored, the metadata sidecar is committed.

## Capture format

| Property | Expected value |
| --- | --- |
| File format | Complex float32 IQ (`*.cfile`, GNU Radio interleaved little-endian) |
| Sample rate | Any rate ≥ 48 kHz; channelise to 48 kHz nominal so it decodes standalone |
| Modulation | C4FM (4-level FSK, 4800 sym/s) **or** LSM/CQPSK (π/4-DQPSK) |
| Centre | Tuned on the control-channel carrier (small DC offset OK; the AFC re-tunes) |
| Duration | ≥ 1 second — enough for several FSW+NID+TSBK frames |

Keep it small enough to commit (a ~1 s 48 kHz slice is ~400 KB), like the
existing `cmd/gophertrunk/testdata/mmr-s9-cc.cfile` regression fixture.

## Metadata schema

Alongside each `*.cfile`, place a `*.metadata.json`:

```json
{
  "source": "RTL-SDR @ <site>, channelised to 48 kHz",
  "tool_cross_check": "OP25 / DSD-FME (optional, for the cross-check)",
  "sample_rate_hz": 48000,
  "center_freq_hz": 420050000,
  "expected": {
    "demod_mode": "c4fm",
    "nac": "0x167",
    "min_nid_trusted": 8,
    "min_tsbk": 24,
    "max_evm_pct": 30.0,
    "min_snr_db": 12.0,
    "min_sync_margin": 2
  },
  "notes": "Optional: capture conditions, antenna, observed RF SNR, etc."
}
```

- `demod_mode` selects the receiver path: `"c4fm"` (default) or `"cqpsk"`.
- `nac` (hex) is asserted against the locked NAC when present.
- `min_nid_trusted` / `min_tsbk` are decode-yield floors (see the
  `mmr-s9-cc.cfile` regression test for the shape).
- `max_evm_pct` / `min_snr_db` / `min_sync_margin` are the **demod-quality**
  bounds the harness grades — leave them out to only *report* the metrics
  without asserting. As the demod improves, tighten them (they are floors,
  not targets).

The metrics are always logged by
`TestReplayP25RealCaptureMetrics`
([`cmd/gophertrunk/p25_realcapture_metrics_test.go`](../../cmd/gophertrunk/p25_realcapture_metrics_test.go));
the bounds above turn the log lines into a pass/fail gate. With no capture
present the test skips.
