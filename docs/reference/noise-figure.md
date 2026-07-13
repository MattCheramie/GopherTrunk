---
slug: noise-figure
title: Noise figure (NF)
entry_type: term
category: rf-metrics
description: Noise figure is the decibel measure of how much a component degrades signal-to-noise ratio; the first-stage NF dominates a cascade via the Friis formula.
keywords: noise figure, NF, noise factor, F, Friis noise formula, cascaded noise, receiver noise, LNA, low noise amplifier, SNR degradation
aka: [NF, noise factor, F]
autolink: true
infobox:
  - { label: Symbol, value: "NF (dB) / F (ratio)" }
  - { label: Unit, value: Decibels (dB) }
  - { label: Formula, value: "NF = 10·log₁₀(SNR_in / SNR_out)" }
see_also: [thermal-noise, noise-temperature, low-noise-amplifier, receiver-sensitivity, signal-to-noise-ratio, friis-transmission-equation]
cite_urls:
  - https://en.wikipedia.org/wiki/Noise_figure
  - https://en.wikipedia.org/wiki/Friis_formulas_for_noise
---

**Noise figure** (**NF**) is the amount, in [decibels](/reference/decibel/), by which
a component or receiver worsens the
[signal-to-noise ratio](/reference/signal-to-noise-ratio/) of the signal passing
through it — the ratio of SNR at the input to SNR at the output.[^wiki] Its linear
form, the **noise factor** *F*, expresses the same thing as a plain ratio. Because
every real device adds its own [thermal noise](/reference/thermal-noise/) on top of
the noise already present, NF is always ≥ 0 dB (F ≥ 1); an ideal noiseless device
has NF = 0 dB. Noise figure is the headline number that, together with bandwidth and
required SNR, sets a receiver's [sensitivity](/reference/receiver-sensitivity/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="A three-stage cascade of amplifier and mixer blocks showing that the first stage gain and noise figure dominate the total noise figure via the Friis formula." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="nfar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="12" y1="55" x2="58" y2="55" stroke="currentColor" stroke-width="1.4" marker-end="url(#nfar)"/>
  <rect x="60" y="35" width="80" height="40" rx="4" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="100" y="52" text-anchor="middle" font-size="10" fill="currentColor">LNA</text>
  <text x="100" y="66" text-anchor="middle" font-size="9" fill="currentColor">F1, G1</text>
  <line x1="140" y1="55" x2="168" y2="55" stroke="currentColor" stroke-width="1.4" marker-end="url(#nfar)"/>
  <rect x="170" y="35" width="80" height="40" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="210" y="52" text-anchor="middle" font-size="10" fill="currentColor">mixer</text>
  <text x="210" y="66" text-anchor="middle" font-size="9" fill="currentColor">F2, G2</text>
  <line x1="250" y1="55" x2="278" y2="55" stroke="currentColor" stroke-width="1.4" marker-end="url(#nfar)"/>
  <rect x="280" y="35" width="80" height="40" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="320" y="52" text-anchor="middle" font-size="10" fill="currentColor">IF amp</text>
  <text x="320" y="66" text-anchor="middle" font-size="9" fill="currentColor">F3, G3</text>
  <line x1="360" y1="55" x2="406" y2="55" stroke="currentColor" stroke-width="1.4" marker-end="url(#nfar)"/>
  <text x="235" y="135" text-anchor="middle" font-size="11" fill="currentColor">F = F1 + (F2−1)/G1 + (F3−1)/(G1·G2)</text>
  <text x="235" y="155" text-anchor="middle" font-size="9.5" fill="currentColor" fill-opacity="0.8">high G1 shrinks every later term — stage 1 dominates</text>
</svg>
<figcaption>Friis's cascade formula: a high-gain, low-noise first stage divides down every later stage's noise contribution, so the front end sets the system noise figure.</figcaption>
</figure>

## How it works

Feed a device a signal accompanied by exactly the standard thermal noise (kTB at
290 K). The device amplifies both, but it also injects its own noise, so the SNR at
the output is worse than at the input. The **noise factor** captures that loss:

**F = SNR_in / SNR_out**  and  **NF = 10·log₁₀ F**

A perfect amplifier (F = 1, NF = 0 dB) would raise signal and noise equally and
preserve SNR. A device with NF = 3 dB halves the SNR; NF = 10 dB divides it by ten.

The crucial insight for a receiver is that stages combine through the **Friis
formula for noise**:

**F_total = F₁ + (F₂ − 1)/G₁ + (F₃ − 1)/(G₁G₂) + …**

Each later stage's noise contribution is divided by the *gain that precedes it*. If
the first stage has high gain G₁, the (F₂ − 1)/G₁ term shrinks and every stage after
it matters even less. This is why the **first amplifier dominates** the system noise
figure — and why a good [low-noise amplifier](/reference/low-noise-amplifier/) placed
first, before any lossy cable or filter, is the single most effective way to lower a
receiver's noise figure.

## Variants

- **Passive losses.** A lossy component (feedline, attenuator, filter) has a noise
  figure equal to its loss: 3 dB of cable loss ahead of the LNA is 3 dB of noise
  figure that the LNA can no longer undo. Put the amplifier at the antenna, not at
  the radio, whenever possible.
- **Noise temperature.** For very low-noise systems (satellite, radio astronomy),
  engineers use [noise temperature](/reference/noise-temperature/) T_e instead —
  it resolves fractions of a dB better than NF and adds linearly in a cascade.
  They are two ways of writing the same physics: NF = 10·log₁₀(1 + T_e/290).

## Relevance to SDR

Noise figure is why an external LNA transforms a mediocre SDR. RTL-SDR dongles often
have a noise figure of 5–8 dB or worse, degraded further by USB and clock spurs;
placing a low-noise, high-gain preamplifier at the antenna makes the dongle's own
noise figure almost irrelevant, per Friis, and drops the effective system NF close
to that of the LNA. The payoff is real weak-signal margin: lowering system NF by
6 dB is worth the same as a 6 dB better antenna or 4× the transmit power at the far
end.

GopherTrunk itself is downstream of all of this — it processes the samples the
front end delivers and cannot recover SNR the receiver's noise figure has already
thrown away. Its practical relevance to a GopherTrunk user is diagnostic: if a
control channel decodes poorly and the signal is genuinely weak (low SNR at the
ADC), the fix is upstream in the RF chain — a better LNA, shorter or better feedline,
or an LNA moved to the mast — not a software setting.

## Sources

[^wiki]: [Noise figure](https://en.wikipedia.org/wiki/Noise_figure) — Wikipedia, definition of noise factor/figure and the Friis cascade formula.
