---
slug: spurious-emissions
title: Spurious Emissions
entry_type: term
category: rf-fundamentals
description: Spurious emissions are unwanted transmitter outputs outside the necessary bandwidth — harmonics, mixer products, and spurs — held below regulatory limits by filtering.
keywords: spurious emissions, spurs, out-of-band emission, unwanted emission, spurious emission mask, harmonics, mixer products, FCC limits, ETSI, transmitter cleanliness
aka: [spurious emission, spurs, spurious output, unwanted emissions]
autolink: true
infobox:
  - { label: Type, value: Unwanted out-of-band transmitter output }
  - { label: Sources, value: Harmonics, mixer & IMD products, LO leakage }
  - { label: Limited by, value: Regulators (FCC, ETSI, ITU-R) }
see_also: [harmonics, intermodulation, occupied-bandwidth, rf-filter, fcc, etsi]
cite_urls:
  - https://en.wikipedia.org/wiki/Spurious_emission
  - https://en.wikipedia.org/wiki/Out-of-band_emission
---

**Spurious emissions** are any radio-frequency energy a transmitter radiates *outside*
the bandwidth it needs for its wanted signal, that could be removed without harming the
information being sent.[^wiki] They include [harmonics](/reference/harmonics/), mixer
and [intermodulation](/reference/intermodulation/) products, parasitic oscillations,
and local-oscillator leakage — and because they interfere with other services, national
and international regulators cap them with hard limits.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="A transmit spectrum showing a central wanted channel inside a necessary-bandwidth band, with small spurious spikes scattered across the rest of the spectrum below a regulatory limit line." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="spurar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" fill="currentColor" font-size="10">
    <line x1="40" y1="20" x2="40" y2="145"/>
    <line x1="40" y1="145" x2="440" y2="145" marker-end="url(#spurar)"/>
    <text x="422" y="163">freq</text>
    <line x1="30" y1="55" x2="440" y2="55" stroke-dasharray="4 3" stroke-opacity="0.7"/>
    <text x="300" y="50" fill="currentColor">regulatory limit</text>
    <rect x="150" y="40" width="60" height="105" fill="currentColor" fill-opacity="0.18" stroke="none"/>
    <line x1="180" y1="30" x2="180" y2="145" stroke-width="3"/>
    <text x="150" y="24">wanted channel</text>
    <line x1="90" y1="120" x2="90" y2="145" stroke-width="2"/>
    <line x1="270" y1="110" x2="270" y2="145" stroke-width="2"/>
    <line x1="360" y1="128" x2="360" y2="145" stroke-width="2"/>
    <text x="255" y="104">spurs</text>
  </g>
</svg>
<figcaption>A clean transmitter keeps all energy outside its necessary bandwidth (the shaded channel) below the regulatory spurious-emission limit.</figcaption>
</figure>

## How it works

Emissions are split by frequency offset. **Out-of-band emissions** fall in the shoulders
immediately adjacent to the channel and come mostly from modulation sidebands and
imperfect [pulse shaping](/reference/pulse-shaping/); they are controlled through the
[occupied-bandwidth](/reference/occupied-bandwidth/) and adjacent-channel-power
specifications. **Spurious emissions** proper are the discrete products that land farther
away — at harmonics, at sums and differences of the various oscillators inside the radio,
and at parasitic resonances.

The physical origins mirror those of harmonics: every nonlinear stage
([power amplifier](/reference/power-amplifier/), [mixer](/reference/mixer-rf/), frequency
multiplier) creates new frequencies, and any oscillator that leaks past its intended
stage can radiate directly. A superheterodyne transmitter with several conversion stages
has many possible spur frequencies, computed from integer combinations *m·f₁ ± n·f₂* of
its internal signals.

Because most spurs sit well away from the wanted channel, the primary defence is
**filtering** — low-pass filters after the final amplifier for harmonics, and
band-pass / cavity filters to reject conversion products — combined with good shielding
and careful gain staging so that no stage is driven hard enough to generate strong
products in the first place.

## In practice

Regulators specify a **spurious-emission mask**: a curve of maximum allowed power versus
frequency offset that a transmitter must sit beneath. In the United States the
[FCC](/reference/fcc/) sets these limits in its rules for each radio service; in Europe
[ETSI](/reference/etsi/) harmonised standards do the same, both anchored to
[ITU-R](/reference/itu-r/) recommendations. Limits are typically quoted as an
attenuation relative to the mean transmit power (for example, "−60 dBc or better") or as
an absolute power in a reference bandwidth. Land-mobile equipment must pass type approval
against these masks before it can be sold, and repair or modification that degrades a
transmitter's spurious performance makes it non-compliant.

## Relevance to SDR

Spurious-emission compliance is a transmitter concern, and every P25, DMR, and TETRA
base station or subscriber unit is engineered and tested against the applicable mask.
A poorly maintained transmitter that develops a spur can appear as a phantom carrier on
a monitored band.

**GopherTrunk** does not transmit and therefore has no spurious-emission obligations of
its own. The concept still helps its users interpret a waterfall: SDR receivers generate
internal spurs and images too — a spike that stays put as you retune, or that appears at
a predictable offset, is usually a receiver-generated spur rather than a real signal.
Recognising these avoids chasing spurious "transmissions" that are artefacts of the
front-end, and pairs naturally with an understanding of
[image frequency](/reference/image-frequency/) and receiver overload.

## Sources

[^wiki]: [Spurious emission](https://en.wikipedia.org/wiki/Spurious_emission) — Wikipedia, definition and regulatory framing of unwanted transmitter output.
[^oob]: [Out-of-band emission](https://en.wikipedia.org/wiki/Out-of-band_emission) — Wikipedia, distinction between out-of-band and spurious-domain emissions.
