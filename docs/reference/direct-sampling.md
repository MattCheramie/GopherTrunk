---
slug: direct-sampling
title: Direct sampling (direct RF sampling)
entry_type: term
category: sdr-dsp
description: "Direct sampling digitises the radio-frequency signal at the antenna with no mixer, letting a fast ADC capture whole bands at once."
keywords: direct sampling, direct RF sampling, direct-sampling mode, mixerless receiver, RF ADC, RTL-SDR direct sampling, HF reception, Nyquist zone
aka: [direct RF sampling, direct-sampling mode, RF sampling]
autolink: true
infobox:
  - { label: Type, value: Receiver architecture }
  - { label: Mixer, value: None (ADC at RF) }
  - { label: Key limit, value: ADC sample rate and analog bandwidth }
see_also: [analog-to-digital-converter, bandpass-sampling, rtl-sdr, nyquist-theorem, sample-rate, superheterodyne-receiver]
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
  - https://en.wikipedia.org/wiki/Undersampling
---

**Direct sampling** feeds the radio-frequency signal from the antenna straight into an
[analog-to-digital converter](/reference/analog-to-digital-converter/) with **no mixer
and no down-conversion** — the ADC digitises the RF itself.[^sdr] Everything after the
converter, including tuning and channel selection, happens in software. It is the purest
form of [software-defined radio](/reference/software-defined-radio/): move the ADC as
close to the antenna as the hardware allows and let DSP do the rest.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="An antenna feeds an anti-alias filter into an ADC that samples the radio-frequency signal directly, and the digital output goes to software for tuning and demodulation, with no mixer stage between antenna and ADC." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="dsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <path d="M40 40 l10 -18 l10 18 M50 22 v45" fill="none" stroke="currentColor"/>
  <text x="30" y="82" font-size="8" fill="currentColor">antenna</text>
  <line x1="60" y1="55" x2="100" y2="55" stroke="currentColor" marker-end="url(#dsar)"/>
  <rect x="100" y="40" width="60" height="30" fill="none" stroke="currentColor"/><text x="107" y="59" font-size="8" fill="currentColor">anti-alias</text>
  <line x1="160" y1="55" x2="200" y2="55" stroke="currentColor" marker-end="url(#dsar)"/>
  <rect x="200" y="38" width="54" height="34" fill="none" stroke="currentColor"/><text x="216" y="59" font-size="11" fill="currentColor">ADC</text>
  <line x1="254" y1="55" x2="300" y2="55" stroke="currentColor" marker-end="url(#dsar)"/>
  <rect x="300" y="38" width="80" height="34" fill="none" stroke="currentColor"/><text x="309" y="59" font-size="8" fill="currentColor">DSP / tune</text>
  <line x1="380" y1="55" x2="430" y2="55" stroke="currentColor" marker-end="url(#dsar)"/>
  <text x="118" y="100" font-size="8" fill="currentColor">no mixer — the ADC samples RF directly</text>
</svg>
<figcaption>In direct sampling the ADC digitises the radio-frequency signal itself; tuning and demodulation are done afterwards in software.</figcaption>
</figure>

## How it works

An ordinary receiver mixes RF down to a lower frequency the ADC can handle. A
direct-sampling receiver skips that step and clocks the ADC fast enough to capture the RF
band directly. To recover a signal at frequency *f* without a mixer, the converter must
sample fast enough to satisfy the [Nyquist theorem](/reference/nyquist-theorem/) for the
band of interest. Two regimes exist:

- **Baseband (Nyquist) sampling** captures everything from DC up to half the
  [sample rate](/reference/sample-rate/). To reach, say, 30 MHz directly the ADC runs at
  60+ MS/s, and an anti-alias low-pass filter blocks anything above the first Nyquist
  zone.
- **[Bandpass (under)sampling](/reference/bandpass-sampling/)** deliberately samples
  slower than the signal frequency, relying on controlled
  [aliasing](/reference/aliasing/) to fold a higher Nyquist zone down to baseband. A tight
  bandpass filter must then isolate the one zone you want.

Either way, the whole captured span is available in software at once — you tune by
selecting and down-converting a channel digitally rather than by retuning hardware.

## Variants

**RTL-SDR direct-sampling mode.** The [RTL-SDR](/reference/rtl-sdr/) is normally a
quadrature-sampling receiver: its [R820T tuner](/reference/r820t-tuner/) down-converts RF
before the [RTL2832U](/reference/rtl2832u/) digitises it, and it cannot tune below about
24 MHz. Feeding an HF antenna to one of the RTL2832U's ADC inputs — the "Q-branch" or
"direct-sampling" hack — bypasses the tuner and samples RF directly at 28.8 MS/s. That
opens up shortwave and HF reception below the tuner's floor, at the cost of poor
filtering and images from the first Nyquist zone. A cleaner alternative is an
[upconverter](/reference/upconverter/) that shifts HF up into the tuner's normal range.

**Dedicated RF-sampling SDRs.** Higher-end designs use ADCs fast enough to sample the
whole HF (or VHF) spectrum properly, with real anti-alias filtering, so no mixer is
needed at all.

## Relevance to SDR

Direct sampling is attractive because removing the mixer removes a whole class of
impairments — no [image frequencies](/reference/image-frequency/), no
[local-oscillator](/reference/local-oscillator/) leakage, no mixer
[intermodulation](/reference/intermodulation/). Its limits are the ADC's clock rate
(which caps the top usable frequency) and its bit depth (which caps
[dynamic range](/reference/dynamic-range/) across a wide capture). For very high
frequencies, [superheterodyne](/reference/superheterodyne-receiver/) or direct-conversion
front ends remain necessary because no affordable ADC samples fast enough.

GopherTrunk is a software decoder that works on whatever IQ stream its source device
delivers, so it treats a direct-sampling capture the same as any other. The trunking
systems it targets (P25, DMR, TETRA) live in the VHF/UHF bands above the reach of the
cheap RTL-SDR direct-sampling hack, so those are normally received with the tuner, but
the direct-sampling concept is fundamental to how RF-sampling SDRs present their data.

## Sources

[^sdr]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, on placing the ADC at or near the antenna and doing tuning in software.
