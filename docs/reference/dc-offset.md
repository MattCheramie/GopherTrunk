---
slug: dc-offset
title: DC offset (DC spike)
entry_type: term
category: sdr-dsp
description: A DC offset is a constant component in an SDR's IQ stream that shows as a spike at the spectrum centre (0 Hz) — a zero-IF receiver artefact, not a real signal.
keywords: DC offset, DC spike, center spike, zero-IF, IQ imbalance, LO leakage, DC blocker, self-mixing
aka: [DC offset, DC spike, "center spike"]
autolink: true
infobox:
  - { label: Type, value: Zero-IF receiver artefact }
  - { label: Shows as, value: Spike at spectrum centre (0 Hz) }
  - { label: Cause, value: LO leakage & converter bias }
see_also: [baseband, iq-data, local-oscillator, fft-and-waterfall, iq-imbalance, direct-conversion-receiver, zero-if]
related_lessons:
  - { title: "The FFT & reading a waterfall", url: /learn/rf-sdr/fft-and-waterfall/ }
cite_urls:
  - https://en.wikipedia.org/wiki/DC_bias
  - https://en.wikipedia.org/wiki/Direct-conversion_receiver
---

A **DC offset** is a constant (zero-frequency) component in the [IQ](/reference/iq-data/)
stream that shows up as a **spike in the exact centre** of the spectrum and waterfall.[^wiki]
It is an artefact of zero-IF/[baseband](/reference/baseband/) receivers — local-oscillator
leakage and converter bias — **not** a real signal on the air.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A spectrum with a sharp narrow spike exactly at the centre frequency, the DC spike, distinct from real signals elsewhere." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="95" x2="430" y2="95" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M30 88 L80 90 L130 86 L180 89 L230 88 L280 90 L330 87 L380 89 L430 88" fill="none" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.5"/>
  <line x1="230" y1="95" x2="230" y2="35" stroke="currentColor" stroke-width="2.2"/><text x="230" y="28" text-anchor="middle" font-size="8.5" fill="currentColor">DC spike (0 Hz)</text>
  <path d="M150 95 L160 60 L170 95 Z" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/><text x="160" y="52" text-anchor="middle" font-size="8" fill="currentColor">real signal</text>
</svg>
<figcaption>The DC spike sits exactly at the tuned centre; it's an artefact to ignore or notch, not a station.</figcaption>
</figure>

## How it works

In a [direct-conversion receiver](/reference/direct-conversion-receiver/) the
[local oscillator](/reference/local-oscillator/) runs at the same frequency as the wanted signal.
Some of that LO energy inevitably leaks back to the mixer input and mixes with itself —
**self-mixing** — producing a constant output that sits at 0 Hz, exactly the tuned centre. Bias
in the ADC and the analog signal path adds more constant offset. Because these are steady, they
appear at DC: a persistent narrow line dead centre in the spectrum that stays put no matter how
you tune, distinguishing it from any real carrier, which moves as you retune.

The DC term carries no information — it is the average of the IQ stream — so removing it costs
nothing but must be done carefully, since an aggressive DC blocker can also attenuate genuine
signal energy that happens to sit near 0 Hz.

## In practice

- **Tune slightly off-centre.** The simplest fix is to offset the tuner so the channel of
  interest does not sit under the spike, then shift it back digitally. Many SDR applications do
  this automatically as an "offset tuning" or "LO offset" mode.
- **DC-blocking filter.** A DC blocker — a high-pass [IIR filter](/reference/iir-filter/) with a
  notch at 0 Hz, or subtracting a running mean — removes the constant component in software.
- **Distinguish from IQ imbalance.** DC offset is a *centre* spike; a separate zero-IF artefact,
  [IQ imbalance](/reference/iq-imbalance/), instead produces *mirror images* of real signals
  reflected across 0 Hz. They have different causes and different corrections, and both are worth
  recognising when reading a [waterfall](/reference/fast-fourier-transform/).

## Relevance to SDR

The DC spike is a common source of "phantom carrier" confusion for newcomers reading a spectrum
display, who mistake the fixed centre line for a real transmission. It is characteristic of the
low-cost zero-IF front ends used in RTL-SDR-class dongles and many SDR tuners. GopherTrunk tunes
each channel with its own digital down-converter, so a channel is generally moved off the raw
centre before demodulation, keeping the wanted signal clear of the hardware DC artefact.

## Sources

[^wiki]: [DC bias](https://en.wikipedia.org/wiki/DC_bias) — Wikipedia, on a constant zero-frequency component in a signal.
[^dcr]: [Direct-conversion receiver](https://en.wikipedia.org/wiki/Direct-conversion_receiver) — Wikipedia, on LO self-mixing as the source of the DC-offset spike in zero-IF designs.
