---
slug: friis-transmission-equation
title: Friis transmission equation
entry_type: term
category: rf-fundamentals
description: The Friis transmission equation gives the power a receiving antenna collects from a transmitting antenna over free space, scaling with wavelength squared and the inverse square of distance.
keywords: Friis transmission equation, Friis formula, received power, free-space, antenna gain, wavelength, path loss, Harald Friis, radio link
aka: [Friis transmission equation, Friis formula, Friis equation]
autolink: true
infobox:
  - { label: Type, value: Free-space propagation law }
  - { label: Formula, value: "P_r/P_t = G_t·G_r·(λ/4πd)²" }
  - { label: Named for, value: "Harald T. Friis (1946)" }
see_also: [free-space-path-loss, antenna-gain, path-loss, link-budget, wavelength, effective-aperture]
cite_urls:
  - https://en.wikipedia.org/wiki/Friis_transmission_equation
---

The **Friis transmission equation** predicts the power a receiving antenna extracts from a
transmitting antenna across an unobstructed free-space path: in its ideal form
**P_r / P_t = G_t · G_r · (λ / 4πd)²**, where *G_t* and *G_r* are the antenna gains, *λ* is the
[wavelength](/reference/wavelength/), and *d* is the separation.[^wiki] Published by Harald T. Friis
in 1946, it is the foundation of every free-space [link budget](/reference/link-budget/) and the
origin of the [free-space path loss](/reference/free-space-path-loss/) term.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A transmit antenna radiates over distance d to a receive antenna; received-to-transmitted power ratio equals the product of both antenna gains times wavelength over four pi d, all squared." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="fr-ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="10" fill="currentColor" stroke="none">
    <line x1="40" y1="90" x2="40" y2="50" stroke="currentColor"/>
    <path d="M32 50 L40 42 L48 50" fill="none" stroke="currentColor"/>
    <text x="26" y="105">G_t, P_t</text>
    <path d="M50 66 q 90 -18 180 0" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
    <path d="M50 72 q 90 6 180 0" fill="none" stroke="currentColor" stroke-opacity="0.3"/>
    <line x1="55" y1="69" x2="220" y2="69" stroke="currentColor" stroke-opacity="0.6" marker-end="url(#fr-ar)"/>
    <text x="120" y="60">distance d</text>
    <line x1="240" y1="90" x2="240" y2="50" stroke="currentColor"/>
    <path d="M232 50 L240 42 L248 50" fill="none" stroke="currentColor"/>
    <text x="226" y="105">G_r, P_r</text>
    <text x="300" y="60" font-style="italic">P_r</text>
    <line x1="298" y1="66" x2="330" y2="66" stroke="currentColor"/>
    <text x="300" y="80" font-style="italic">P_t</text>
    <text x="340" y="72">= G_t G_r (λ/4πd)²</text>
  </g>
</svg>
<figcaption>The Friis equation: received power falls as the inverse square of distance and rises with the square of wavelength, scaled by both antenna gains.</figcaption>
</figure>

## How it works

The formula is built from two physical ideas. First, an isotropic source spreads its power over the
surface of an expanding sphere, so the **power density** at distance *d* falls as *1/(4πd²)* — the
inverse-square law. A transmit antenna with gain *G_t* concentrates energy toward the receiver,
multiplying the density in that direction. Second, the receive antenna captures power in proportion
to its **[effective aperture](/reference/effective-aperture/)**, and aperture relates to gain by
*A_e = G_r·λ²/4π*. Multiplying the arriving power density by the effective aperture yields:

**P_r = P_t · G_t · G_r · (λ / 4πd)²**

Two consequences deserve emphasis. The **inverse-square in distance** means doubling range costs 6 dB
of received power. The **wavelength-squared** factor means that, for antennas of *fixed gain*, a
longer wavelength (lower frequency) delivers more received power — the low band travels "better" not
because space treats it differently but because a fixed-gain antenna has a larger physical aperture
at longer wavelengths. Expressed in decibels, the (λ/4πd)² term is exactly the negative of
[free-space path loss](/reference/free-space-path-loss/).

## In practice

Friis is an idealization: it assumes a clear line of sight, far-field distances, matched
polarization, impedance-matched and lossless antennas, and no obstruction or multipath. Real links
add correction terms for feedline loss, polarization mismatch, atmospheric absorption, and fading —
all handled as extra dB in a [link budget](/reference/link-budget/). Even so, the bare equation sets
the ceiling: no terrestrial path beats free space, so Friis is the optimistic bound against which
real measurements are compared.

## Relevance to SDR

For a receive-only [SDR](/reference/software-defined-radio/), Friis quantifies the arriving power from
a known transmitter. Given the transmitter's [EIRP](/reference/erp-eirp/) (which already folds in
*P_t·G_t*), the receive [antenna gain](/reference/antenna-gain/), the frequency, and the distance,
the equation predicts *P_r* — the input level that your front end must lift above its
[noise floor](/reference/noise-floor/) to decode. It also explains recurring field observations: why
the same trunking system is easier to hear on its VHF control channel than an equivalent 800 MHz one
with identical antennas, and why every 6 dB of arriving margin corresponds to a doubling of usable
range.

**GopherTrunk** does not evaluate the Friis equation in its decode chain, but the equation is the
right mental model for predicting whether a distant site delivers enough signal to lock, and how much
receive gain would recover a marginal link.

## Sources

[^wiki]: [Friis transmission equation](https://en.wikipedia.org/wiki/Friis_transmission_equation) — Wikipedia, derivation from aperture and gain, wavelength dependence, and idealizing assumptions.
