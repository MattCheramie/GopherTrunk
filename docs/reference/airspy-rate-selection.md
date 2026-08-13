---
slug: airspy-rate-selection
title: Airspy sample-rate selection
entry_type: term
category: fn-hardware
description: "Why GopherTrunk recommends 2.5 MS/s on the Airspy R2, how the firmware's IQ-rate table caused a half-rate regression, and a capture-and-compare recipe for qualifying any untested rate."
keywords: airspy, airspy r2, airspy mini, sample rate, 10 msps, 2.5 msps, phase noise, real sampling, hilbert, evm, snr, replay
see_also: [airspy, airspy-r2, airspy-mini, sample-rate, phase-noise, sdr-gain-overload, rtlsdr-usb-recovery, signal-signatures, diagnostic-playbook, c4fm]
---

**Airspy sample-rate selection** matters more than it looks: on the
[Airspy R2](/reference/airspy-r2/), keep `sample_rate: 2500000`. The R2 advertises only
two native rates — 10 MS/s and 2.5 MS/s — and the 2.5 MS/s path is the FPGA's
decimate-by-4 of the same 10 MS/s ADC stream. That decimated path is measurably
*cleaner*: in [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) the same
P25 carrier replayed at demod SNR ≈ 19.7 dB / EVM 7.4% from a 2.5 MS/s capture (locks)
but ≈ 9.5 dB / EVM 22.5% from a 10 MS/s capture (no lock).

## The 10 MS/s deficit is in the hardware, not the DSP

[#764](https://github.com/MattCheramie/GopherTrunk/issues/764) started as a GopherTrunk
bug report, and two real rate-dependent bugs *were* fixed (per-tap full-rate resampling
on one goroutine, and a channelizer bin count hardcoded for 2.4 MS/s). But the residual
~10 dB in-channel deficit at 10 MS/s survived both fixes, and the decisive test proved
it was baked into the captured samples: decimating the 10 MS/s capture 4:1 with an
*independent* resampler (not GopherTrunk's DSP at all) and replaying it through the
proven 2.5 MS/s path reproduced the same ≈ 9.5 dB.

The signature is worth memorizing (see [signal signatures](/reference/signal-signatures/)):

| Symptom | Looks like | Actually | Fix/Check |
|---|---|---|---|
| Wideband FFT carrier SNR *higher* at 10 MS/s, in-channel EVM ~10 dB *worse* | Decoder bug at high rates | Sampling-clock [phase noise](/reference/phase-noise/) / reciprocal mixing at the native ADC rate | Run at 2.5 MS/s; qualify other rates with the recipe below |
| All taps go dark when raising the rate, including previously strong ones | Weak signal | (Historic, fixed) per-tap full-rate resampling overran the USB ring | Upgrade; the shared decimation stage landed with #768 |
| Deficit persists at very different gain settings | Front-end overload / intermod | Not compression — phase noise. In [#771](https://github.com/MattCheramie/GopherTrunk/issues/771), gain 600 and gain 300 gave EVM 22.5% vs 22.7% despite ~6 dB power difference | If EVM tracked gain, suspect [overload](/reference/sdr-gain-overload/) instead |
| `agc_level` ~9–10× above target while failing | AGC failure to converge | Normal operating gain of the symbol AGC (matched-filter gain ≈ samples/symbol) — the *working* capture shows the identical value | Always check an alarming metric in the passing case too |

Neither capture in #764 clipped (both peaked around −48 dBFS), so this is not the
[overload signature](/reference/sdr-gain-overload/) — carrier-clean but
modulation-degraded is a different animal.

## The firmware rate table is IQ rates, not real-sample rates

The Airspy firmware's `GET_SAMPLERATES` table (`{10000000, 2500000}` on the R2) lists
**IQ output rates**. A v0.5.8 regression
([#851](https://github.com/MattCheramie/GopherTrunk/issues/851)) doubled the requested
IQ rate before matching the table, on the assumption it held real-sample rates:
`2500000 × 2 = 5000000` matched nothing, snapped to the nearest entry, and
`ActualSampleRate()` returned 1,250,000 Hz. The hardware still delivered a correct
2.5 MS/s, but the daemon built its DDC at half rate — the symbol clock ran 2× off and
every frame sync was dropped. The tell was the WARN pair
`requested_hz=2500000 actual_hz=1250000` with healthy `iq_power_dbfs` and zero clipping.

Two traps from that thread:

- The reporter's proposed workaround — `sample_rate: 1250000` — would have made things
  worse: it is not a table entry either.
- The reporter's mechanism ("lost DC-spike headroom") was wrong; a plausible mechanism
  attached to a real symptom still needs verifying.

## Airspy R2 and Mini are real-sampling devices

The R2/Mini firmware streams bare real ADC samples (unpacked little-endian uint16,
12-bit, DC at 2048) at **twice** the configured IQ rate; conversion to complex baseband
is the host's job ([#454](https://github.com/MattCheramie/GopherTrunk/issues/454)).
GopherTrunk performs it with a DC blocker, Fs/4 translation, and a half-band Hilbert
pair, stateful across USB packets. An early driver instead paired adjacent real samples
as I and Q — and because neighbouring samples of an oversampled signal are highly
correlated, the result was the textbook mispairing signature: **phase imbalance ≈ +78°
with image rejection ≈ 3.3 dB**. If you ever see numbers like that from any
real-sampling front end, that is the diagnosis.

The same issue (with [#270](https://github.com/MattCheramie/GopherTrunk/issues/270))
also fixed an opcode table that was systematically offset against libairspy's command
enum — the cause of the original `set sample rate ... protocol error` open failures.

## Qualifying an untested rate

The daemon's sample-rate validator historically capped rates well below the R2's native
10 MS/s ([#270](https://github.com/MattCheramie/GopherTrunk/issues/270),
[#550](https://github.com/MattCheramie/GopherTrunk/issues/550)); note that GopherTrunk's
rational L/M resampler is exact at any rational rate, so "the resampler fails at rate X"
claims should be treated skeptically — the #550 report of failures at 5/6.25/16 MHz was
built on UHD's own CIC-rolloff messages, which never touch a host-side resampler.

The trustworthy way to qualify a rate, from
[#771](https://github.com/MattCheramie/GopherTrunk/issues/771):

1. Capture the same signal twice, once at a known-good rate (2.4 or 2.5 MS/s) and once
   at the rate under test.
2. Replay both with `gophertrunk replay -diag`.
3. Compare the `demod (c4fm): EVM=.. SNR≈..` lines. A deficit that survives the rate
   change through an independent decode path is in the captured samples.

This matters beyond the R2: the [Airspy Mini](/reference/airspy-mini/) shares the
R820T2 tuner but samples on a different clock, and its 6 MS/s top rate is the same
untested "native top rate" regime — do not assume it is clean without the comparison.

## Provenance

- [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) — wideband DDC above 2.5 MS/s: two real GopherTrunk bugs fixed, residual ~10 dB deficit proven to be Airspy hardware/firmware phase noise at the native 10 MS/s rate.
- [#771](https://github.com/MattCheramie/GopherTrunk/issues/771) — the "AGC fails to converge" red herring, the gain-independence test, and the capture-and-compare qualification recipe.
- [#851](https://github.com/MattCheramie/GopherTrunk/issues/851) — `ActualSampleRate()` half-rate regression from misreading the firmware's IQ-rate table.
- [#454](https://github.com/MattCheramie/GopherTrunk/issues/454) — R2/Mini are real-sampling; the +78° / 3.3 dB mispairing signature and the host-side Hilbert converter.
- [#270](https://github.com/MattCheramie/GopherTrunk/issues/270) — original Airspy R2 driver bring-up and the sample-rate validator limits.
- [#550](https://github.com/MattCheramie/GopherTrunk/issues/550) — the claimed resampler failures at specific rates that turned out to be UHD messages, not GopherTrunk behavior.
