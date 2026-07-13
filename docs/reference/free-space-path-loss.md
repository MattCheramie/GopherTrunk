---
slug: free-space-path-loss
title: Free-space path loss (FSPL)
entry_type: term
category: propagation
description: Free-space path loss is the drop in signal power over an unobstructed path, falling as 1/r² and rising with frequency — the baseline term in every RF link budget.
keywords: free space path loss, FSPL, path loss, inverse square law, link budget, decibel, spreading loss, Friis equation, propagation loss
aka: [FSPL, free-space path loss]
autolink: true
infobox:
  - { label: Type, value: Idealised propagation loss }
  - { label: Relation, value: "Power ∝ 1/r² (spreading)" }
  - { label: Formula, value: "FSPL(dB)=20log₁₀d+20log₁₀f+k" }
see_also: [path-loss, friis-transmission-equation, link-budget, decibel, radio-propagation]
cite_urls:
  - https://en.wikipedia.org/wiki/Free-space_path_loss
  - https://en.wikipedia.org/wiki/Friis_transmission_equation
---

**Free-space path loss** (**FSPL**) is the reduction in a signal's power density as it
spreads outward over a clear, unobstructed path between two isotropic antennas.[^wiki]
It is the idealised baseline of [path loss](/reference/path-loss/) — no obstacles, no
reflections, no atmosphere — and it falls off with the square of distance, the plain
geometry of energy spreading over an ever-larger sphere.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="A point source at left with expanding circular wavefronts; the same power is spread over a larger sphere at each radius, and a curve below shows received power dropping steeply with distance." xmlns="http://www.w3.org/2000/svg">
  <circle cx="60" cy="55" r="4" fill="currentColor"/><text x="60" y="30" text-anchor="middle" font-size="8" fill="currentColor">TX</text>
  <path d="M60 55 A30 30 0 0 1 60 55 M75 55 A15 15 0 1 1 45 55 A15 15 0 1 1 75 55" fill="none" stroke="currentColor" stroke-opacity="0.7"/>
  <circle cx="60" cy="55" r="30" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
  <circle cx="60" cy="55" r="48" fill="none" stroke="currentColor" stroke-opacity="0.35"/>
  <circle cx="60" cy="55" r="66" fill="none" stroke="currentColor" stroke-opacity="0.2"/>
  <text x="150" y="30" font-size="8" fill="currentColor" fill-opacity="0.8">power spreads over 4πr²</text>
  <path d="M40 150 L40 105 M40 150 L440 150" stroke="currentColor" stroke-opacity="0.5"/>
  <path d="M45 108 Q120 140 250 148 Q350 150 435 150" fill="none" stroke="currentColor" stroke-width="1.6" marker-end="url(#fsar)"/>
  <text x="440" y="162" text-anchor="end" font-size="8" fill="currentColor">distance →</text><text x="30" y="112" text-anchor="end" font-size="8" fill="currentColor">Prx</text>
  <defs><marker id="fsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The transmitted power spreads over a sphere of area 4πr², so received power falls as 1/r² — a fourfold drop (6 dB) for every doubling of distance.</figcaption>
</figure>

## How it works

Imagine a transmitter radiating equally in all directions. At distance *d* its power is
spread evenly over a sphere of surface area 4π*d*². A receiving antenna captures only the
fraction of that sphere its **effective aperture** intercepts, so received power falls
as 1/*d*². This is pure geometric spreading — no energy is absorbed, it is just diluted.

Two facts about the standard formula surprise newcomers:

- **The frequency term is about the antenna, not the wave.** A wave loses no energy to
  frequency in free space. FSPL rises with frequency because the reference is the
  *isotropic* antenna, whose effective aperture shrinks with wavelength (∝ λ²). Hold the
  physical dish size fixed and higher frequencies actually capture *more* — the loss
  "penalty" is an artifact of the isotropic reference, which the
  [Friis equation](/reference/friis-transmission-equation/) makes explicit.
- **It is a 20 log law.** In [decibels](/reference/decibel/):

  FSPL(dB) = 20 log₁₀(*d*) + 20 log₁₀(*f*) + *k*

  where *k* folds in the constants and units (for *d* in km and *f* in MHz, *k* ≈ 32.4).
  Every doubling of distance or frequency adds 6 dB; every tenfold increase adds 20 dB.

FSPL is not the whole story of real links — [multipath](/reference/multipath-propagation/),
[atmospheric absorption](/reference/atmospheric-absorption/), diffraction, and terrain
all add to it — but it is the floor every other loss builds on, and over a clear
line-of-sight path it is often the dominant term.

## In practice

FSPL is the first line of any [link budget](/reference/link-budget/): start with
transmit power and [antenna gain](/reference/antenna-gain/), subtract FSPL for the path,
subtract extra margins, and compare the result against
[receiver sensitivity](/reference/receiver-sensitivity/) to see whether the link
closes. The [Friis transmission equation](/reference/friis-transmission-equation/) is
FSPL written with real (non-isotropic) antenna gains folded in. Extreme cases make the
20 log law vivid: a satellite or a [Moonbounce](/reference/moonbounce-eme/) path can run
to 200 dB or more, which is why those links demand huge antennas and low-noise front
ends.

## Relevance to SDR

For any SDR user, FSPL is the intuition behind "why can't I hear it?" — doubling the
distance to a transmitter costs 6 dB, and moving up a band costs more against a fixed
whip. For a trunking scanner like **GopherTrunk**, FSPL underlies the coverage picture:
a system's usable range is set by transmit power minus path loss versus the receiver's
sensitivity and noise floor. GopherTrunk does not compute path loss — it decodes
whatever arrives — but the term explains why antenna height, a clear path, and a
low-noise front end so often matter more than raw radio quality, and why a distant
hilltop [site](/reference/trunking-site/) can outperform a closer, obstructed one.

## Sources

[^wiki]: [Free-space path loss](https://en.wikipedia.org/wiki/Free-space_path_loss) — Wikipedia, on inverse-square spreading, the decibel formula, and the frequency term's isotropic-aperture origin.
