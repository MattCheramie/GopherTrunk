---
slug: fm-deviation
title: FM deviation
entry_type: term
category: modulation
description: "FM deviation is the peak amount a message shifts an FM carrier's frequency away from its center, setting occupied bandwidth via Carson's rule."
keywords: FM deviation, frequency deviation, peak deviation, Carson's rule, wideband FM, narrowband FM, deviation ratio, 5 kHz deviation
aka: [FM deviation, frequency deviation, peak deviation]
autolink: true
infobox:
  - { label: Symbol, value: "Δf" }
  - { label: Unit, value: "Hertz (kHz typical)" }
  - { label: Relation, value: "Carson: B ≈ 2(Δf + fm)" }
see_also: [frequency-modulation, modulation-index, bandwidth, broadcast-fm, carrier-wave]
cite_urls:
  - https://en.wikipedia.org/wiki/Frequency_deviation
  - https://en.wikipedia.org/wiki/Carson_bandwidth_rule
---

**FM deviation** is the peak amount a modulating signal shifts a
[frequency-modulated](/reference/frequency-modulation/) carrier away from its center
frequency.[^wiki] Written Δf, it is measured in hertz and, for a given audio level, is set by
the transmitter's design; the louder the instantaneous message, the farther the carrier
momentarily strays. Deviation is the single most important number for sizing an FM signal's
[bandwidth](/reference/bandwidth/) and choosing the receiver filter.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A carrier frequency deviating up and down around a center line in step with a message waveform, with the peak excursion labelled delta f." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="75" x2="430" y2="75" stroke="currentColor" stroke-opacity="0.35" stroke-dasharray="4 3"/>
  <text x="34" y="70" font-size="8" fill="currentColor">fc</text>
  <path d="M30 75 q6 -3 12 -3 q6 0 12 3 M66 75 q8 3 16 9 q8 6 16 3 M114 90 q10 -4 20 -12 q10 -8 20 -3 M170 60 q8 5 16 12 q8 7 16 3 M218 82 q12 -6 24 -14 q12 -8 24 0 M290 68 q8 4 16 7 M322 79 q10 -3 20 -6 q10 -3 20 2" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <line x1="250" y1="75" x2="250" y2="47" stroke="currentColor" marker-end="url(#fmdar)"/>
  <line x1="250" y1="75" x2="250" y2="103" stroke="currentColor" marker-end="url(#fmdar)"/>
  <text x="258" y="52" font-size="9" fill="currentColor">+Δf</text>
  <text x="258" y="102" font-size="9" fill="currentColor">−Δf</text>
  <defs><marker id="fmdar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Deviation Δf is the peak frequency swing above and below the carrier as the message rises and falls.</figcaption>
</figure>

## How it works

In FM the instantaneous carrier frequency is f_c + k·m(t), so the frequency wanders in step
with the message and Δf is the largest excursion the loudest allowed signal produces. Deviation
is fixed by the transmitter, not by how far off you tune. It combines with the highest
modulating frequency f_m to give the [modulation index](/reference/modulation-index/) (the
deviation ratio) β = Δf / f_m, and, more usefully, the occupied bandwidth through **Carson's
rule**: B ≈ 2(Δf + f_m). Doubling the deviation roughly doubles the bandwidth but also improves
the demodulated signal-to-noise ratio, because FM's noise-quieting advantage grows with
deviation — this is the classic bandwidth-for-quality bargain that makes wideband FM sound so
clean.

The improvement is not free of a threshold, though. FM's noise advantage only appears once the
carrier is strong enough for the demodulator to stay locked; below the **FM threshold** (very
roughly 10 dB of carrier-to-noise) the discriminator produces sharp clicks and the signal
collapses much faster than an AM signal would. A higher-deviation, wider-bandwidth signal admits
more noise and so reaches that threshold at a higher carrier level — one more reason deviation is
a balance rather than a "more is always better" setting. Related is the **capture effect**: when
two FM signals overlap, the discriminator locks to the stronger one and largely ignores the
weaker, an effect that also strengthens with deviation.

Systems are described as **narrowband** or **wideband** purely by their deviation. A narrowband
land-mobile channel might allow only ±2.5 kHz of deviation; classic wideband voice used ±5 kHz;
[broadcast FM](/reference/broadcast-fm/) uses ±75 kHz. The receiver's channel filter and
discriminator must be wide enough to pass the deviation without clipping the peaks, yet narrow
enough to reject the neighbors — a filter much wider than Carson's rule just lets in extra noise.

## Relevance to SDR

Deviation dictates the channel-filter width GopherTrunk applies before demodulating any
FM-family signal. The C4FM and related four-level modes at the core of P25 and DMR are
frequency-shift schemes with a defined peak deviation (roughly ±1.8 kHz for the outer C4FM
symbols in a 12.5 kHz P25 channel), and the decoder's front-end filtering and FM discriminator
are sized around that figure. Get the assumed deviation wrong — too narrow a filter — and the
symbol peaks are clipped and the eye closes; too wide, and excess noise degrades the
[SNR](/reference/signal-to-noise-ratio/). For analog FM voice, deviation also sets how hot the
recovered audio is, so a mismatch between transmit deviation and receiver expectation shows up
directly as loud or weak audio.

## In practice

Regulators specify a maximum deviation per channel plan, and transmitters use a deviation limiter
to keep peaks inside it. When narrowbanding forced many land-mobile users from 25 kHz to 12.5 kHz
channels, the allowed deviation was cut roughly in half, which is why older wideband radios sound
over-deviated (distorted, loud) when heard on a narrowband receiver, and vice versa.

Deviation is also directly measurable, which makes it a handy diagnostic. On a calibrated
receiver or [spectrum analyzer](/reference/spectrum-analyzer/) an operator can read peak deviation
to confirm a transmitter is neither under-deviated (weak, quiet audio, poor noise performance) nor
over-deviated (distorted, splattering into neighbors). For the digital FSK-family carriers a
scanner decodes, the "deviation" is really the spacing of the discrete frequency states, and it is
fixed by the standard rather than by an audio level — so a symbol-level discriminator expects the
states to land at their nominal offsets, and a transmitter whose deviation has drifted shows up as
a distorted symbol constellation and a rising error rate.

## Sources

[^wiki]: [Frequency deviation](https://en.wikipedia.org/wiki/Frequency_deviation) — Wikipedia, for the peak-excursion definition; Carson bandwidth rule for the bandwidth relation.
