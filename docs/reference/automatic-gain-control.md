---
slug: automatic-gain-control
title: Automatic gain control (AGC)
entry_type: term
category: sdr-dsp
description: Automatic gain control adjusts amplification to keep a signal at a usable level; in SDR it can be hardware or software, and is often set manually for stable decoding.
keywords: AGC, automatic gain control, gain, headroom, clipping, pumping, attack time, decay time, gain staging
aka: [automatic gain control, AGC]
autolink: true
infobox:
  - { label: Type, value: Gain-management technique }
  - { label: Goal, value: Keep level usable, avoid clipping }
  - { label: Note, value: Manual gain often best for decoding }
see_also: [dbfs, analog-to-digital-converter, noise-floor, ppm-frequency-correction, squelch, noise-blanker, dynamic-range]
related_lessons:
  - { title: "Gain, AGC & avoiding overload", url: /learn/rf-sdr/gain-and-agc/ }
related_reading:
  - { title: "SDR Internals, Part 4: DSP foundations — filters, NCO & AGC", url: /blog/deep-dives/sdr-internals-04-dsp-foundations-filters-nco-agc/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Automatic_gain_control
  - https://en.wikipedia.org/wiki/Dynamic_range
---

**Automatic gain control** (**AGC**) is a feedback loop that adjusts amplification to keep a
signal at a usable level — high enough above the [noise floor](/reference/noise-floor/) to
resolve fine detail, but below the [ADC](/reference/analog-to-digital-converter/)'s clipping
ceiling (0 [dBFS](/reference/dbfs/)).[^wiki] It measures the output level, compares it against a
target, and drives a variable-gain stage in the opposite direction, so a signal that fades or
surges by tens of decibels still arrives at the detector within a narrow window.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="An input whose amplitude varies wildly, and an output of roughly constant amplitude after AGC." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="22" font-size="9" fill="currentColor">input (varying level)</text>
  <path d="M20 45 q6 -6 12 0 t12 0 q6 -22 12 0 t12 0 q6 -22 12 0 t12 0 q6 -4 12 0 t12 0 q6 -4 12 0 t12 0 q6 -16 12 0 t12 0 q6 -16 12 0 t12 0" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="20" y="95" font-size="9" fill="currentColor">output (levelled)</text>
  <path d="M20 118 q6 -13 12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0" fill="none" stroke="currentColor" stroke-width="1.3"/>
</svg>
<figcaption>AGC continuously adjusts gain so the output level stays roughly constant despite a fading input.</figcaption>
</figure>

## How it works

An AGC loop has three parts: a **level detector** (often an envelope or RMS estimate of the
signal), a **comparison** against a reference set-point, and a **variable-gain element** whose
gain is driven by the error. Because it is a closed loop, its behaviour is governed by two time
constants:

- **Attack time** — how fast the loop reduces gain when the signal suddenly grows. Fast attack
  protects against clipping on a strong burst but can clamp the leading edge of a transmission.
- **Decay (release) time** — how fast the loop restores gain in a quiet interval. Slow decay is
  smoother; fast decay recovers quickly but can "pump," audibly or numerically ramping the noise
  up between symbols.

If the loop responds faster than the modulation itself, it will fight the signal it is trying to
preserve — flattening the amplitude variations of an AM or QAM waveform, or modulating the noise
floor of a bursty digital channel. This is why AGC is tuned to be slow relative to the symbol
rate but fast relative to fading.

## Variants

- **Hardware AGC** lives in the tuner or IF chain (e.g. the [LNA](/reference/low-noise-amplifier/)
  and mixer gain stages of an [RTL-SDR](/reference/rtl-sdr/) or [Airspy](/reference/airspy/)),
  adjusting analog gain before the ADC so the converter's [dynamic range](/reference/dynamic-range/)
  is used well.
- **Software (digital) AGC** operates after digitisation, scaling the [IQ](/reference/iq-data/)
  stream in the DSP chain. It cannot recover headroom already lost to ADC clipping, but it is
  precise and repeatable.
- **Feed-forward vs feedback** — most AGCs are feedback loops; feed-forward designs measure the
  input and pre-scale it, trading loop stability for speed.

## In practice

For monitoring a fixed trunking system, a well-chosen **manual gain** usually beats AGC. AGC
optimises for an average level, but a decoder wants a *consistent* level and clean headroom for
the strongest expected signal. Automatic loops tend to raise gain during silence — lifting the
noise floor and inviting [intermodulation](/reference/intermodulation/) from strong nearby
signals — then clamp hard on a local transmission, producing the "pumping" that disrupts symbol
slicing. The practical routine is to set gain so the loudest signal peaks a few dB below 0 dBFS
and leave it there. AGC is distinct from [squelch](/reference/squelch/) (which mutes weak audio)
and a [noise blanker](/reference/noise-blanker/) (which removes impulse spikes); all three manage
level, but only AGC changes amplification.

## Relevance to SDR

Setting gain correctly is the single setting beginners most often get wrong: too little buries
weak signals in quantisation noise, too much overloads the front end and manufactures spurs.
GopherTrunk normalises channel amplitude in its DSP chain so the symbol slicer sees a stable
level regardless of the captured signal strength, which is a form of software AGC applied per
channel rather than across the whole captured band. See the gain lesson for a practical routine.

## Sources

[^wiki]: [Automatic gain control](https://en.wikipedia.org/wiki/Automatic_gain_control) — Wikipedia, on closed-loop gain adjustment, attack/decay, and headroom.
[^dr]: [Dynamic range](https://en.wikipedia.org/wiki/Dynamic_range) — Wikipedia, on the span between noise floor and clipping that AGC positions a signal within.
