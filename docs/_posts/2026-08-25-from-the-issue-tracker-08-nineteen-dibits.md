---
title: "From the Issue Tracker, Part 8: Nineteen Dibits — A Perfect Hypothesis Meets a Rail-Pinned ADC"
description: A meticulously argued bug report proved that GopherTrunk's wideband DDC starved the P25 demodulator with chunks too short to hold a frame sync word — except the sync detector buffers across chunks, the incriminating log line only fires while unlocked, and the raw capture was 50% pinned to the ADC rails. The fix was turning the gain down.
category: solution-postmortem
keywords: p25 wideband, ddc channelizer, frame sync word, dibits, agc saturation, adc clipping, rtl-sdr gain, front-end overload, constant envelope c4fm, gophertrunk debugging
tags: [from-the-issue-tracker, p25, sdr-hardware, wideband, gain, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 8
---

*Part 8 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that
fought back. [Part 7]({{ '/blog/solution-postmortem/from-the-issue-tracker-07-lsm-myth/' | relative_url }})
ended with a site that failed because its front end was starved of gain. This
part — [#881](https://github.com/MattCheramie/GopherTrunk/issues/881), from the
same meticulous reporter — is the exact mirror image: a device that failed
because its front end had far too much. Between the two sits one of the
best-argued wrong hypotheses the tracker has ever seen.*

> **TL;DR:** A wideband RTL-SDR with four P25 control channels clustered in a
> 225 kHz span never decoded a single TSBK, while an identically configured
> device with a 1.75 MHz span decoded thousands. The reporter built a compelling
> case that the DDC delivered chunks of ~19 dibits — too short to contain the
> 24-dibit frame sync word. But the sync detector buffers 24 dibits *across*
> chunks, both devices produce byte-identical chunk sizes, and the damning log
> line only fires while unlocked — the working device simply locked and went
> quiet. A raw capture settled it: ~50% of samples were pinned to the u8 ADC
> rails (24.9% at 0, 24.9% at 255) — AGC-driven front-end saturation. It still
> decoded 3 of 4 control channels offline, because constant-envelope C4FM
> survives hard limiting, which is also why the FFT looked clean. The fix was
> counter-intuitive: a **fixed low gain** of 20 dB.

## The report

As bug reports go, this one was a gift. Two `role: wideband` RTL dongles on the
same host, same build, same 2.048 MS/s sample rate, same four-channel count,
same `auto(ddc)` strategy. The UHF device, its channels spread across 1.75 MHz,
decoded 13,000+ TSBKs. The VHF device, its channels clustered into 225 kHz,
decoded exactly zero — and emitted a steady stream of:

```
p25/phase1: no FSW hits in chunk system=RMR freq_hz=166450000 phase=1 dibits=19
```

The reporter histogrammed the `dibits=` values across a whole run: 18, 19, 20 —
never larger. The P25 frame sync word is 48 bits = **24 dibits**. A chunk of 19
dibits can never contain one. Meanwhile the working device logged almost no
such lines. And GopherTrunk itself had flagged the failing device at startup:

```
widebandt2: capture is oversampled for the channel plan — the carriers span
less than half the captured band ... channel_span_hz=250000 min_sample_rate_hz=277778
```

The hypothesis practically wrote itself: when the channel plan occupies a small
fraction of the captured band, the DDC's per-tap output arrives in chunks
shorter than one sync word, so the correlator can never match and the channel
never locks — independent of signal quality. The report even ruled things out
with metrics: per-channel power at a healthy −41 to −43 dBFS, `iq_clip_ratio 0`,
clean carriers on the spectrum display, frequencies confirmed against the
network's own neighbor broadcasts. Signal present, geometry suspect,
smoking-gun log line attached.

It was coherent, evidence-based, and wrong on every count. What follows is why
each piece of evidence meant something other than what it appeared to mean —
the anatomy of a false confirmation.

## Dismantling the hypothesis

Three facts, all checkable in code and pinned in tests, took the chunk-starvation
theory apart:

**The sync detector buffers across chunks.** `p25phase1.SyncDetector` keeps a
24-dibit circular history that persists between `Process` calls — precisely so
a frame sync word split across chunk boundaries still correlates. That behavior
was added in [#275](https://github.com/MattCheramie/GopherTrunk/issues/275),
and a test already existed that locks a control channel and follows grants
while being fed **19-dibit batches**. A short chunk cannot, by construction,
prevent sync.

**Both devices get the same ~19-dibit chunks.** At 2.048 MS/s the shared
front-end decimator is a no-op for either geometry, and both plans run the
identical per-tap resampler. A test with both channel plans verbatim fed each
one a single 8192-sample RTL USB transfer: both emitted 192 samples per tap ≈
19 dibits, **byte-for-byte identical**. The clustered plan and the wide plan
produce indistinguishable chunking — the working device was decoding 13,000
TSBKs from 19-dibit chunks all along.

**The log line only fires while unlocked.** This was the keystone.
`no FSW hits in chunk` is emitted only in the `!locked` state. The working
device stopped logging it the moment it locked — not because its chunks grew.
"852 short-chunk lines on the failing device, ~none on the working one" is
fully explained by lock state alone. The histogram was real data measuring the
wrong thing: not *why* the device failed, but merely *that* it failed,
repeated 852 times.

One more tell pointed away from the channelizer: the plan's isolated tap at
+100 kHz, with no adjacent channel anywhere near it, failed alongside the
clustered ones. A per-tap adjacency problem doesn't take down a lonely tap.

The startup warning, meanwhile, was a red herring of our own manufacture — an
advisory about wasted capture bandwidth that happened to fire on exactly the
failing device, welding the two facts together in a way no debugger could
resist. (The escape hatch it implied didn't even exist: setting `sample_rate`
near the reported minimum just snaps back up to the RTL wideband floor of
2.048 MS/s.)

## The capture that settled it

Per the repo's discipline — no fix without reproducing the symptom — the next
step was a raw capture: ~10 seconds of the full 2.048 MS/s stream from the
failing dongle. The first thing anyone should do with a suspect capture is not
an FFT. It's a **sample-value histogram**:

- 24.9% of u8 samples exactly `0`
- 24.9% of u8 samples exactly `255`
- raw IQ RMS ≈ **+1.3 dBFS**

Half the samples pinned to the ADC rails. The tuner AGC, wide open on a VHF
whip in an environment thick with strong signals, was slamming the front end
into hard saturation. The "signal" reaching the DSP was a square-ish caricature
of the band.

And here is the twist that explains everything the reporter saw: **the capture
still decoded.** Replayed offline through the exact `role: wideband` code path
— the same DDC bank, the same P25 receiver, the reporter's exact four-channel
clustered plan — three of the four control channels locked and produced
396/396/72 TSBKs. C4FM is a constant-envelope modulation: the information
lives in phase and frequency, which survive hard limiting even when the
amplitude is clipped to death. That's why the FFT looked clean, why the
carriers appeared healthy, and why decode was *possible* — while marginal taps
and live conditions still starved. A regression test
(`TestClusteredOversampledPlanDecodesP25`) now drives the full wideband engine
over the exact clustered plan with a slice of this capture, permanently pinning
"the geometry decodes fine."

(One honest wrinkle: the reporter's earlier gain-sweep observation — input
level frozen at −14.18 dBFS whether gain was set to 14.4 dB, auto, or 40 dB —
was itself a symptom. A front end that doesn't respond to 26 dB of commanded
gain change is already telling you the AGC owns the knob.)

## The counter-intuitive fix

On an overloading front end, **more gain makes it worse** — and so does AGC,
which will happily drive into saturation on a strong-signal band. The remedy is
to back off: fixed low gain, and if needed an inline attenuator or a filter for
the offending strong neighbor.

The reporter switched the device from `gain: auto` to `gain: "200"` — 20 dB
fixed, in GopherTrunk's tenths-of-a-dB convention — and all three strong
control channels locked immediately:

```
tsbk decoded  system=RMR  freq_hz=166450000  phase=1  nac=365
tsbk decoded  system=RMR  freq_hz=166462500  phase=1  nac=365
tsbk decoded  system=RMR  freq_hz=166650000  phase=1  nac=362
```

The fourth channel stayed silent — a genuinely weak neighbor site, which is
what "too weak" is supposed to look like. No production DSP code changed,
because there was nothing in the DSP to fix. GopherTrunk's
`wideband front end overloaded` warning (added in
[#749](https://github.com/MattCheramie/GopherTrunk/issues/749)) exists to catch
exactly this state; when it fires, the gain conversation should start before
any DSP conversation.

| Evidence | Looked like | Actually was |
|---|---|---|
| `dibits=19` on every `no FSW hits` line | chunks too short for the 24-dibit FSW | normal chunk size; detector buffers across chunks |
| 852 log lines vs ~6 on the working device | failing geometry starves sync | line only fires while unlocked; working device locked |
| oversampled-plan startup warning | GT fingering its own channelizer | advisory about wasted bandwidth, coincident not causal |
| clean carriers on the FFT | healthy signal, so DSP must be at fault | constant-envelope C4FM looks fine even hard-limited |
| `iq_clip_ratio 0` at the taps | no overload | measured post-DDC; the raw ADC was 50% rail-pinned |

## What we keep

- **A log line's firing condition is part of its meaning.** `no FSW hits in
  chunk` fires only while unlocked, so comparing its frequency across a locked
  and an unlocked device measures lock state, not chunk behavior. Before
  histogramming a diagnostic, read the guard around it — step one in the
  [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }}).
- **Histogram the raw samples before you trust the spectrum.** Two spikes at
  the ADC rails is front-end saturation, full stop — and an FFT will not show
  it on a constant-envelope signal. The rail-pinned histogram and its cousins
  are catalogued in [signal signatures]({{ '/reference/signal-signatures/' | relative_url }}).
- **Constant-envelope modulations hide overload.** C4FM decoding *survives*
  hard limiting, so "it decodes offline" and "the front end is saturated" are
  not contradictory. Overload shows up first as marginal taps dying, not as
  total silence.
- **On a hot band: fixed low gain beats AGC, and more gain makes it worse.**
  The overload/starvation symmetry — this part and
  [Part 7]({{ '/blog/solution-postmortem/from-the-issue-tracker-07-lsm-myth/' | relative_url }})'s
  — is laid out in [SDR gain and overload]({{ '/reference/sdr-gain-overload/' | relative_url }}).
- **Respect a good wrong hypothesis — then make it pay rent.** The chunk theory
  was falsifiable, and falsifying it (cross-chunk buffering test, byte-identical
  chunk comparison, the isolated-tap tell) produced the regression tests and the
  capture that found the truth. The fastest route out was through the raw IQ:
  when a live device fails but the numbers argue, get the `.cfile`.

## Series navigation

**Part 8** · ←
[Part 7: The LSM Myth]({{ '/blog/solution-postmortem/from-the-issue-tracker-07-lsm-myth/' | relative_url }})
