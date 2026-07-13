---
slug: knife-edge-diffraction
title: Knife-edge diffraction
entry_type: term
category: propagation
description: Knife-edge diffraction is the bending of radio waves around a sharp obstacle edge, letting signals reach beyond line-of-sight with a predictable loss.
keywords: knife-edge diffraction, diffraction loss, obstruction loss, Fresnel-Kirchhoff parameter, shadow zone, obstacle gain, terrain diffraction
aka: [knife-edge diffraction, edge diffraction]
autolink: true
infobox:
  - { label: Type, value: Obstacle diffraction model }
  - { label: Parameter, value: "Fresnel–Kirchhoff v" }
  - { label: Effect, value: Loss (or slight gain) past an edge }
see_also: [fresnel-zone, radio-horizon, refraction, multipath-propagation, free-space-path-loss]
cite_urls:
  - https://en.wikipedia.org/wiki/Knife-edge_effect
  - https://en.wikipedia.org/wiki/Diffraction
---

**Knife-edge diffraction** describes how a radio wave bends around the sharp top edge of
an obstacle — a ridgeline, a rooftop, a hill — so that some energy reaches into the
geometric shadow behind it.[^wiki] It is the reason signals are receivable just over the
crest of a hill even with no line of sight, and it is modelled by treating the obstacle
as an idealised opaque half-plane with a single sharp edge. The predicted loss depends on
how deeply the edge cuts into the [Fresnel zone](/reference/fresnel-zone/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A transmitter behind a sharp ridge; the direct ray is blocked but energy diffracts over the edge and curves down to a receiver in the shadow region, with reduced strength." xmlns="http://www.w3.org/2000/svg">
  <line x1="35" y1="120" x2="35" y2="80" stroke="currentColor" stroke-width="2"/><text x="35" y="135" text-anchor="middle" font-size="9" fill="currentColor">TX</text>
  <line x1="425" y1="120" x2="425" y2="95" stroke="currentColor" stroke-width="2"/><text x="425" y="135" text-anchor="middle" font-size="9" fill="currentColor">RX</text>
  <path d="M215 120 L245 40 L275 120 Z" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.2"/>
  <text x="245" y="34" text-anchor="middle" font-size="9" fill="currentColor">edge</text>
  <line x1="42" y1="82" x2="242" y2="45" stroke="currentColor" stroke-width="1.3"/>
  <path d="M245 42 Q330 44 420 92" fill="none" stroke="currentColor" stroke-width="1.3" stroke-dasharray="4 3" marker-end="url(#kear)"/>
  <text x="345" y="55" font-size="8" fill="currentColor">diffracted</text>
  <text x="345" y="112" font-size="8" fill="currentColor">shadow zone</text>
  <defs><marker id="kear" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Energy diffracts over the sharp edge and curves into the shadow behind it, reaching a receiver with no direct line of sight.</figcaption>
</figure>

## How it works

The geometry is captured by the dimensionless **Fresnel–Kirchhoff diffraction parameter**

- `v = h · √(2/λ · (1/d₁ + 1/d₂))`,

where `h` is the height of the edge above (positive) or below (negative) the direct ray,
`λ` is the [wavelength](/reference/wavelength/), and `d₁`, `d₂` are the distances from the
edge to each endpoint. The diffraction loss is then a smooth function of `v`:

- **v ≪ 0** (edge well below the ray, first [Fresnel zone](/reference/fresnel-zone/)
  clear): essentially no loss.
- **v = 0** (edge exactly grazing the line of sight): about **6 dB** of loss — half the
  wavefront is blocked.
- **v > 0** (edge above the ray, receiver in shadow): loss grows steadily, roughly
  20·log-scale with `v`, reaching tens of dB deep in the shadow.

A curiosity of the model is the **obstacle gain** or "knife-edge gain": for a narrow band
of slightly negative `v`, the edge can reflect and refocus energy so the received level is
marginally *higher* than the unobstructed free-space value.

## Variants

A single sharp ridge is the ideal case. Real terrain often presents rounded hills or
multiple successive edges, handled by extensions — rounded-obstacle corrections and
multiple-knife-edge methods (Bullington, Epstein–Peterson, Deygout) that cascade several
diffraction events along a path profile. These underpin the terrain-diffraction engines in
propagation planning tools such as ITU-R P.526.

## Relevance to SDR

Knife-edge diffraction is why VHF and UHF coverage extends somewhat beyond the optical
[radio horizon](/reference/radio-horizon/) and why a scanner can hear a repeater whose
tower is hidden behind a hill. The received strength in such a shadow follows the `v`
curve, so a modest change in geometry — moving over the crest, or raising the antenna to
reduce `h` — can swing the signal by many decibels. Because longer wavelengths diffract
more readily, low-VHF signals fill in behind terrain far better than microwave ones, a
reason land-mobile trunking favours VHF/UHF for wide-area coverage.

Combined with [refraction](/reference/refraction/), which slightly extends the horizon,
diffraction explains most "impossible" over-the-hill receptions.
[GopherTrunk](/reference/software-defined-radio/) does no propagation modelling — it simply
decodes whatever reaches the antenna — but the diffraction loss on a shadowed path is
often the difference between a decodable and an undecodable
[trunking site](/reference/trunking-site/), and it shows up directly as reduced
[SNR](/reference/signal-to-noise-ratio/) at the receiver.

## Sources

[^wiki]: [Knife-edge effect](https://en.wikipedia.org/wiki/Knife-edge_effect) — Wikipedia, on obstacle diffraction, the Fresnel–Kirchhoff parameter, and diffraction loss.
