---
slug: effective-aperture
title: Effective aperture
entry_type: term
category: antennas
description: Effective aperture is the equivalent capture area of a receiving antenna, tied directly to its gain by the wavelength-squared relation and central to the Friis link equation.
keywords: effective aperture, effective area, capture area, aperture, antenna gain, Friis equation, wavelength, aperture efficiency, parabolic antenna
aka: [effective aperture, effective area, capture area]
autolink: true
infobox:
  - { label: Type, value: Receiving-antenna property }
  - { label: Unit, value: square metres (m²) }
  - { label: Relation, value: A_e = G · λ² / 4π }
see_also: [antenna-gain, friis-transmission-equation, parabolic-antenna, wavelength, radiation-pattern]
cite_urls:
  - https://en.wikipedia.org/wiki/Antenna_aperture
  - https://en.wikipedia.org/wiki/Friis_transmission_equation
---

**Effective aperture** (or effective area) is the equivalent physical area over which a receiving
antenna "captures" power from a passing radio wave.[^wiki] If a wave carries a power density of
*S* watts per square metre, an antenna with effective aperture *A*_e delivers *P* = *S* · *A*_e
watts to a matched load. Even a thin-wire antenna with almost no physical area has a well-defined
effective aperture, because the figure describes how much power it *extracts* from the field, not
its metal footprint. Effective aperture is tied one-to-one to [antenna gain](/reference/antenna-gain/)
and is the quantity that makes the [Friis transmission equation](/reference/friis-transmission-equation/)
work.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A plane wave with power density S sweeps across a shaded rectangular capture area labelled effective aperture A-e, which funnels the intercepted power into an antenna and its load." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="efap" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="30" y1="30" x2="30" y2="130" stroke="currentColor" stroke-width="1" stroke-dasharray="4 3"/>
  <line x1="55" y1="30" x2="55" y2="130" stroke="currentColor" stroke-width="1" stroke-dasharray="4 3"/>
  <text x="42" y="20" text-anchor="middle" font-size="9" fill="currentColor">S (W/m²)</text>
  <rect x="150" y="45" width="70" height="70" fill="currentColor" fill-opacity="0.13" stroke="currentColor" stroke-width="1.3"/>
  <text x="185" y="84" text-anchor="middle" font-size="9" fill="currentColor">A_e</text>
  <line x1="220" y1="80" x2="330" y2="80" stroke="currentColor" stroke-width="1.4" marker-end="url(#efap)"/>
  <circle cx="345" cy="80" r="8" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="345" y="112" text-anchor="middle" font-size="8.5" fill="currentColor">load</text>
  <text x="410" y="50" font-size="9" fill="currentColor">A_e = Gλ²/4π</text>
  <text x="410" y="66" font-size="8.5" fill="currentColor">P = S·A_e</text>
</svg>
<figcaption>Effective aperture is the equivalent area that intercepts the incoming power density and delivers it to the load.</figcaption>
</figure>

## How it works

For any antenna, effective aperture and gain are two expressions of the same directional
selectivity, related by a compact and universal formula:

> *A*_e = *G* · λ² / (4π),

where *G* is the gain (as a linear ratio, not decibels) and λ is the wavelength. The wavelength
appears because a receiving antenna's ability to gather power scales with the size of the field
region it interacts with, which is set by λ. Two consequences follow immediately:

- **Gain and aperture are interchangeable.** Quoting a gain of 6 dBi (a linear ratio of 4) at a
  given frequency fixes the effective aperture, and vice versa. This equivalence is a direct
  result of reciprocity.
- **At a fixed physical size, higher frequency means higher gain.** Because *A*_e is roughly the
  physical area for a large aperture antenna, *G* = 4π*A*_e / λ² rises as λ shrinks. A dish of a
  given diameter has far more gain at 10 GHz than at 1 GHz.

For aperture-type antennas — dishes, horns, patches — the *effective* aperture is less than the
*physical* aperture *A*_phys by an **aperture efficiency** η_ap (typically 0.5–0.7 for a
[parabolic reflector](/reference/parabolic-antenna/)):

> *A*_e = η_ap · *A*_phys.

The shortfall comes from non-uniform illumination, spillover past the reflector edge, blockage by
the feed, and surface errors.

## In practice

Effective aperture is the natural bridge between an antenna's physics and a link budget. The
[Friis equation](/reference/friis-transmission-equation/) can be written with the receive antenna
as an aperture,

> *P*_r = *P*_t · *G*_t · *A*_e / (4π*d*²),

which reads directly as "transmitted power spread over a sphere of radius *d*, intercepted by an
area *A*_e." A curiosity that surprises newcomers: even a short dipole, whose gain is only 1.64
(2.15 dBi), has an effective aperture of about 0.13 λ² — considerably larger than the wire itself.
Small antennas are better power collectors than their size suggests, which is why a modest whip
still hears distant signals.

## Relevance to SDR

Effective aperture is the concept a scanner user is really invoking when they say a bigger
antenna "hears more." It explains why the same physical dish or Yagi gives more gain on higher
bands, and it underpins any link-budget estimate of whether a distant
[trunking site](/reference/trunking-site/) will be receivable. For the crowded VHF/UHF scanner
bands, where external noise usually dominates, more aperture raises signal and noise together and
the net benefit is smaller than the raw number implies; on quiet microwave bands the aperture gain
translates almost directly into sensitivity. **GopherTrunk** works on the delivered IQ samples and
has no notion of aperture, but effective aperture is one of the terms that sets how much signal
power reaches the SDR, and thus the signal-to-noise ratio the decoder must work with.

## Sources

[^wiki]: [Antenna aperture](https://en.wikipedia.org/wiki/Antenna_aperture) — Wikipedia, for the effective-aperture definition and the A_e = Gλ²/4π relation.
