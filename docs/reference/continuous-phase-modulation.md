---
slug: continuous-phase-modulation
title: Continuous-phase modulation (CPM)
entry_type: technology
category: modulation
description: Continuous-phase modulation (CPM) is a family of digital modulations that keep the carrier phase continuous for a constant envelope and compact spectrum; MSK and GMSK are members.
keywords: CPM, continuous phase modulation, continuous phase FSK, constant envelope, spectral efficiency, MSK, GMSK, CPFSK, modulation index, phase trellis
aka: [continuous-phase modulation, CPM]
autolink: true
infobox:
  - { label: Type, value: Digital modulation family }
  - { label: Idea, value: Phase never jumps -> constant envelope }
  - { label: Members, value: MSK, GMSK, CPFSK }
see_also: [minimum-shift-keying, gmsk, frequency-shift-keying, four-fsk, modulation-index, phase-shift-keying]
cite_urls:
  - https://en.wikipedia.org/wiki/Continuous_phase_modulation
  - https://en.wikipedia.org/wiki/Minimum-shift_keying
---

**Continuous-phase modulation** (**CPM**) is a family of digital modulations in which the
[carrier](/reference/carrier-wave/) phase is constrained to vary *continuously*, never
jumping between symbols.[^wiki] That single rule gives every CPM signal a **constant
envelope** — its amplitude never changes — and a **compact spectrum**, because abrupt
phase discontinuities are exactly what create wide spectral sidelobes. Well-known members
include [minimum-shift keying](/reference/minimum-shift-keying/) and its Gaussian-filtered
refinement [GMSK](/reference/gmsk/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Two phase paths over time: a jagged one with vertical jumps marked as ordinary PSK, and a smooth continuous one marked as CPM." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="120" x2="440" y2="120" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M30 100 L110 100 L110 60 L190 60 L190 90 L270 90 L270 40 L350 40 L350 80 L430 80" fill="none" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.5" stroke-dasharray="4 3"/>
  <path d="M30 100 C 70 100, 90 60, 130 60 S 190 90, 230 90 C 270 90, 290 40, 330 40 S 390 80, 430 80" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="35" y="52" font-size="9" fill="currentColor">smooth = CPM (continuous)</text>
  <text x="250" y="134" font-size="9" fill="currentColor">dashed = PSK with phase jumps</text>
</svg>
<figcaption>CPM forces the phase path to be continuous (solid), avoiding the jumps of ordinary PSK (dashed) and so keeping a constant envelope and narrow spectrum.</figcaption>
</figure>

## How it works

A CPM signal is defined by an accumulated phase: the modulator integrates a data-driven
frequency deviation, so the transmitted phase is the running sum of every past symbol.
Because integration is inherently continuous, the phase cannot jump. Three parameters
characterise a CPM scheme:

- the **[modulation index](/reference/modulation-index/) h**, which sets how far the
  phase advances per symbol (h = 0.5 gives MSK);
- the **pulse shape**, which spreads each symbol's phase contribution over one interval
  (rectangular) or several (partial-response, as in GMSK's Gaussian pulse);
- the **alphabet size**, binary or multi-level.

Constant envelope is the payoff: a CPM signal can be driven through a saturated,
non-linear power amplifier at high efficiency without spectral regrowth, unlike
amplitude- or phase-jump schemes that need linear amplifiers and back-off. The cost is
receiver complexity. Optimal CPM detection tracks the accumulated phase through a
**trellis** with a [Viterbi-style](/reference/viterbi-algorithm/) sequence estimator,
because the phase memory couples successive symbols. Simpler, slightly sub-optimal
receivers use a frequency discriminator and accept a small penalty.

## Variants

The CPM family is a continuum. Full-response schemes (one symbol per pulse) include MSK
and general CPFSK; partial-response schemes spread each pulse over several symbols to
narrow the spectrum further, at the price of controlled intersymbol interference that the
trellis detector unwinds. [GMSK](/reference/gmsk/) is the best-known partial-response
member. Multi-h CPM cycles the modulation index between symbols to improve distance
properties. The [4FSK](/reference/four-fsk/)/C4FM used in land-mobile radio is
continuous-phase FSK and so sits within this same family.

## Relevance to SDR

CPM matters to software radio precisely because of its efficiency: any system that must
squeeze the most range and battery life out of a cheap non-linear amplifier tends toward
CPM. GSM (GMSK), AIS (GMSK), aircraft and satellite telemetry, and the C4FM land-mobile
modes are all continuous-phase. On a waterfall these signals are narrow and flat-topped,
with none of the amplitude blinking of ASK. Choosing the right demodulator — coherent
trellis versus non-coherent discriminator — is the main practical decision.

GopherTrunk lives squarely in this family: the P25 C4FM, DMR, and NXDN modes it decodes
are continuous-phase 4FSK, and its AIS path uses a GMSK demodulator. CPM is therefore
not just background here but the theory underpinning most of what GopherTrunk actually
receives.

## Sources

[^wiki]: [Continuous phase modulation](https://en.wikipedia.org/wiki/Continuous_phase_modulation) — Wikipedia, for the CPM definition, the modulation-index/pulse-shape parameters, and full- versus partial-response schemes.
