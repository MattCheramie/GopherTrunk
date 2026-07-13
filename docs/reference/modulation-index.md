---
slug: modulation-index
title: Modulation index
entry_type: term
category: modulation
description: "Modulation index measures how strongly a message modulates a carrier — the AM depth ratio, or in FM the ratio of peak deviation to modulating frequency."
keywords: modulation index, AM modulation depth, FM modulation index, deviation ratio, overmodulation, beta, Bessel sidebands
aka: [modulation index, modulation depth, deviation ratio]
autolink: true
infobox:
  - { label: Symbol, value: "m (AM), β (FM)" }
  - { label: Unit, value: Dimensionless ratio }
  - { label: Relation, value: "AM: m = Am/Ac; FM: β = Δf/fm" }
see_also: [fm-deviation, amplitude-modulation, frequency-modulation, bandwidth, carrier-wave]
cite_urls:
  - https://en.wikipedia.org/wiki/Modulation_index
  - https://en.wikipedia.org/wiki/Frequency_modulation
---

**Modulation index** is a dimensionless number that says how strongly a message drives a
[carrier](/reference/carrier-wave/) — how far the carrier's amplitude, frequency, or phase
is pushed relative to its unmodulated value.[^wiki] Its exact definition depends on the
modulation type, but in every case a larger index means deeper modulation, more sideband
energy, and wider occupied [bandwidth](/reference/bandwidth/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Two AM waveforms: one lightly modulated with a shallow envelope at low modulation index, and one at 100 percent modulation whose envelope just touches zero." xmlns="http://www.w3.org/2000/svg">
  <text x="105" y="20" text-anchor="middle" font-size="9" fill="currentColor">m ≈ 0.4</text>
  <path d="M20 75 q4 -14 8 0 q4 -18 8 0 q4 -14 8 0 q4 -9 8 0 q4 -7 8 0 q4 -9 8 0 q4 -14 8 0 q4 -18 8 0 q4 -14 8 0 q4 -9 8 0 q4 -7 8 0 q4 -9 8 0 q4 -14 8 0 q4 -18 8 0 q4 -14 8 0 q4 -9 8 0 q4 -7 8 0 q4 -9 8 0" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <path d="M20 75 C 60 55, 100 55, 140 75 S 190 95, 190 75" fill="none" stroke="currentColor" stroke-opacity="0.5" stroke-dasharray="3 2"/>
  <text x="355" y="20" text-anchor="middle" font-size="9" fill="currentColor">m = 1.0 (100%)</text>
  <path d="M270 75 q4 -35 8 0 q4 -45 8 0 q4 -35 8 0 q4 -20 8 0 q4 -5 8 0 q4 -20 8 0 q4 -35 8 0 q4 -45 8 0 q4 -35 8 0 q4 -20 8 0 q4 -5 8 0 q4 -20 8 0 q4 -35 8 0 q4 -45 8 0 q4 -35 8 0 q4 -20 8 0 q4 -5 8 0 q4 -20 8 0" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <path d="M270 75 C 310 30, 350 30, 390 75 S 440 120, 440 75" fill="none" stroke="currentColor" stroke-opacity="0.5" stroke-dasharray="3 2"/>
  <line x1="270" y1="120" x2="440" y2="120" stroke="currentColor" stroke-opacity="0.3"/>
</svg>
<figcaption>In AM the index sets envelope depth: m = 1 means the envelope just reaches zero; m &gt; 1 overmodulates and distorts.</figcaption>
</figure>

## How it works

**In amplitude modulation** the index (often written *m* or *µ*) is the ratio of message
amplitude to carrier amplitude, m = A_m / A_c, usually quoted as a percentage. At m = 0 there
is no modulation; at m = 1 (100%) the envelope swings all the way from twice the carrier down
to zero; beyond m = 1 the carrier "overmodulates," the envelope tries to go negative, the
recovered audio clips, and the transmitter splatters energy into adjacent channels. Higher m
puts more power in the [sidebands](/reference/amplitude-modulation/) and so improves the
recovered signal-to-noise ratio, which is why broadcasters run AM as deep as regulation and
distortion allow.

**In angle modulation** the index measures phase swing in radians. For frequency modulation
it is β = Δf / f_m, the ratio of the peak [frequency deviation](/reference/fm-deviation/) to
the modulating frequency; for phase modulation it is simply the peak phase deviation. Here the
index is not a "depth" you can exceed — it directly sets the number and strength of significant
sidebands. Unlike AM, an FM carrier's amplitude never changes; increasing β spreads energy
into ever more sideband pairs whose amplitudes follow Bessel functions of β. (Curiously, at a
few specific β values the carrier component itself vanishes.)

## Relevance to SDR

The distinction between narrowband (β &lt; ~0.5) and wideband FM is entirely a statement about
modulation index, and it decides how much bandwidth a signal needs and how a demodulator should
be sized. Land-mobile [FM](/reference/frequency-modulation/) voice and the analog outer layer of
digital modes run modest indices in 12.5 kHz channels, whereas [broadcast FM](/reference/broadcast-fm/)
uses a much larger deviation and index for its fidelity. When GopherTrunk demodulates an FM-family
carrier, it does not need to estimate the index explicitly, but the index chosen by the transmitter
is what determines the channel width the receiver must pass and the deviation the discriminator will
see. For AM signals a software receiver can even read the modulation index off the recovered envelope
as a quick check of whether a station is under- or over-modulating.

## In practice

Index directly feeds bandwidth estimates: Carson's rule for FM, B ≈ 2(Δf + f_m) = 2 f_m(β + 1),
is really a statement in terms of the deviation ratio. Choosing the index is therefore a
bandwidth-versus-quality trade the system designer makes before anyone tunes the signal.

## Sources

[^wiki]: [Modulation index](https://en.wikipedia.org/wiki/Modulation_index) — Wikipedia, for the AM depth and FM/PM index definitions and their sideband consequences.
