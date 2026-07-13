---
slug: receiver-sensitivity
title: Receiver sensitivity (MDS)
entry_type: term
category: rf-metrics
description: Receiver sensitivity is the weakest signal a receiver can usefully detect, set by the thermal noise floor plus its noise figure plus the SNR the mode requires.
keywords: receiver sensitivity, MDS, minimum discernible signal, minimum detectable signal, noise floor, noise figure, required SNR, dBm, SINAD, 12 dB SINAD
aka: [MDS, minimum discernible signal, minimum detectable signal, sensitivity]
autolink: true
infobox:
  - { label: Symbol, value: "MDS / S_min" }
  - { label: Unit, value: dBm }
  - { label: Formula, value: "−174 + 10·log₁₀B + NF + SNR_req" }
see_also: [thermal-noise, noise-figure, noise-floor, signal-to-noise-ratio, low-noise-amplifier, sinad]
cite_urls:
  - https://en.wikipedia.org/wiki/Sensitivity_(electronics)
  - https://en.wikipedia.org/wiki/Noise_floor
---

**Receiver sensitivity** — often quoted as the **minimum discernible signal (MDS)** —
is the weakest input power a receiver can detect while still meeting a required
quality threshold.[^wiki] It is not a single mysterious property of the radio but a
sum of three legible terms: the [thermal noise](/reference/thermal-noise/) floor set
by kTB, the receiver's own [noise figure](/reference/noise-figure/), and the
[signal-to-noise ratio](/reference/signal-to-noise-ratio/) the chosen modulation
needs to work. Add them and you have the faintest signal, in dBm, that will decode.
Sensitivity defines the bottom of a receiver's [dynamic range](/reference/dynamic-range/)
and is the number that ultimately decides how far away a transmitter can be heard.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 185" role="img" aria-label="A stacked power budget building up from the minus 174 dBm per hertz thermal density, adding ten log of bandwidth, then noise figure, then the required signal-to-noise ratio, to reach the sensitivity or minimum discernible signal level." xmlns="http://www.w3.org/2000/svg">
  <line x1="60" y1="20" x2="60" y2="160" stroke="currentColor" stroke-width="1.3"/>
  <text x="52" y="18" text-anchor="end" font-size="9" fill="currentColor">dBm</text>
  <rect x="70" y="140" width="120" height="20" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/>
  <text x="200" y="154" font-size="9.5" fill="currentColor">−174 dBm/Hz (kT)</text>
  <rect x="70" y="112" width="120" height="26" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/>
  <text x="200" y="129" font-size="9.5" fill="currentColor">+ 10·log₁₀(B)  → noise floor</text>
  <rect x="70" y="86" width="120" height="24" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.2"/>
  <text x="200" y="102" font-size="9.5" fill="currentColor">+ noise figure (NF)</text>
  <rect x="70" y="56" width="120" height="28" fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="1.2"/>
  <text x="200" y="74" font-size="9.5" fill="currentColor">+ required SNR</text>
  <line x1="70" y1="52" x2="330" y2="52" stroke="currentColor" stroke-width="1.6" stroke-dasharray="1 0"/>
  <text x="200" y="46" font-size="10" fill="currentColor">= sensitivity (MDS)</text>
</svg>
<figcaption>Sensitivity is a stacked budget: the kTB thermal density, widened by bandwidth to the noise floor, raised by the receiver's noise figure, then by the SNR the mode needs to decode.</figcaption>
</figure>

## How it works

The chain from physics to spec is short and exact. Start with thermal noise density,
**−174 dBm/Hz** at 290 K. Spread it over the receiver's bandwidth B by adding
10·log₁₀(B) in [decibels](/reference/decibel/) — that gives the input
[noise floor](/reference/noise-floor/). Raise it by the receiver's
[noise figure](/reference/noise-figure/), the noise the front end adds on top.
Finally add the **required SNR** the demodulator needs for acceptable quality. The
whole thing:

**MDS = −174 dBm/Hz + 10·log₁₀(B) + NF + SNR_required**

A worked example for a 12.5 kHz land-mobile channel with a 6 dB noise figure and a
mode needing 8 dB of SNR:

−174 + 10·log₁₀(12 500) + 6 + 8 ≈ −174 + 41 + 6 + 8 = **−119 dBm**.

Every term is a lever. Narrower bandwidth lowers the noise floor (but must still pass
the signal). A better [low-noise amplifier](/reference/low-noise-amplifier/) first in
the chain lowers NF. A more robust modulation or stronger FEC lowers the required
SNR. Sensitivity improves — the MDS number gets more negative — whenever any of these
drops.

## Variants

- **MDS / MDS at 0 dB SNR.** Purists define minimum *detectable* signal as the level
  where signal power equals the noise floor (SNR = 0 dB); the practical sensitivity
  adds the SNR the mode actually needs.
- **SINAD sensitivity.** Analog and many commercial receivers are specified by the
  input level that yields **12 dB [SINAD](/reference/sinad/)**, a listening-quality
  criterion rather than a raw SNR.
- **BER sensitivity.** Digital modes specify the input that achieves a target
  [bit error rate](/reference/bit-error-rate/) (e.g. 5% BER for P25) — the SNR term is
  effectively "whatever SNR reaches that BER after FEC."

## Relevance to SDR

Sensitivity puts a floor under what any decoder can do, GopherTrunk included. Two
SDRs may claim the same sensitivity on paper yet behave differently in the field
because [dynamic range](/reference/dynamic-range/), not sensitivity, is the limit in a
crowded spectrum — a very sensitive receiver that overloads on a strong neighbour is
useless. Still, when a band is quiet and a distant control channel simply will not
decode, the sensitivity budget tells you where the margin went: usually the noise
figure (add a mast-mounted LNA and shorten feedline) or the required SNR (nothing you
can change about the mode).

GopherTrunk sits at the SNR_required end of this equation: its decode thresholds are
statements about how much SNR each protocol and vocoder needs. It cannot lower a
receiver's MDS — that is fixed by kTB, bandwidth, and the front-end noise figure —
so closing a weak link is an antenna, LNA, and feedline problem first, and a software
problem never.

## Sources

[^wiki]: [Sensitivity (electronics)](https://en.wikipedia.org/wiki/Sensitivity_(electronics)) — Wikipedia, receiver sensitivity, MDS, and the noise-floor-plus-NF-plus-SNR budget.
