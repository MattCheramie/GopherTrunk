---
slug: fresnel-zone
title: Fresnel zone
entry_type: term
category: propagation
description: A Fresnel zone is the elliptical region around a radio path that must stay clear of obstructions to avoid diffraction loss on a line-of-sight link.
keywords: Fresnel zone, first Fresnel zone, clearance, path clearance, diffraction loss, line-of-sight, obstruction, ellipsoid
aka: [Fresnel zone, first Fresnel zone]
autolink: true
infobox:
  - { label: Type, value: Path-clearance region }
  - { label: Shape, value: Ellipsoid around the direct ray }
  - { label: Rule of thumb, value: Keep 60% of first zone clear }
see_also: [knife-edge-diffraction, radio-horizon, multipath-propagation, rician-fading, free-space-path-loss]
cite_urls:
  - https://en.wikipedia.org/wiki/Fresnel_zone
  - https://en.wikipedia.org/wiki/Fresnel_diffraction
---

**A Fresnel zone** is one of a family of concentric ellipsoids surrounding the straight
line between a transmitter and receiver; the **first Fresnel zone** is the innermost, and
it must stay largely clear of obstacles for a link to behave as clean line-of-sight.[^wiki]
Even when the direct ray grazes over an obstruction, an object intruding into this zone
diffracts energy and adds loss. Fresnel-zone clearance is the practical criterion that
turns "we can see the far antenna" into "the link will actually close."

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A transmitter and receiver joined by a direct ray, surrounded by an elliptical first Fresnel zone; a hilltop intruding into the ellipse causes diffraction loss even though line of sight is maintained." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="120" x2="40" y2="70" stroke="currentColor" stroke-width="2"/><text x="40" y="135" text-anchor="middle" font-size="9" fill="currentColor">TX</text>
  <line x1="420" y1="120" x2="420" y2="70" stroke="currentColor" stroke-width="2"/><text x="420" y="135" text-anchor="middle" font-size="9" fill="currentColor">RX</text>
  <line x1="40" y1="70" x2="420" y2="70" stroke="currentColor" stroke-width="1.2"/>
  <ellipse cx="230" cy="70" rx="190" ry="42" fill="none" stroke="currentColor" stroke-width="1.2" stroke-dasharray="5 3"/>
  <text x="230" y="40" text-anchor="middle" font-size="9" fill="currentColor">first Fresnel zone</text>
  <path d="M250 120 L285 92 L320 120 Z" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1"/>
  <text x="285" y="135" text-anchor="middle" font-size="8" fill="currentColor">intrusion → loss</text>
  <defs><marker id="frzar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The first Fresnel zone is an ellipsoid around the direct ray; an obstacle poking into it causes diffraction loss even with visual line of sight.</figcaption>
</figure>

## How it works

The zones are defined by path-length difference. The `n`-th Fresnel zone is the set of
points for which a wave scattered from that point travels `n·λ/2` farther than the direct
ray, where `λ` is the [wavelength](/reference/wavelength/). Contributions from within the
first zone (`0` to `λ/2` extra path) arrive roughly in phase with the direct ray and add
constructively; the second zone tends to cancel, and so on. Because the first zone carries
most of the useful energy, it is the one that matters for clearance.

Its radius is largest at the path midpoint:

- `r ≈ √(λ·d₁·d₂ / d)`, where `d₁` and `d₂` are the distances from each endpoint to the
  point of interest and `d = d₁ + d₂` is the total path length.

Two practical facts follow. The zone is **wider at lower frequencies** (longer λ), so VHF
links demand more clearance than microwave ones over the same distance. And engineers
apply the **60% rule**: keeping about 60% of the first zone free of obstruction yields
essentially free-space loss, so links are planned with antenna heights (and terrain
allowance for [refraction](/reference/refraction/)) that meet it.

## Relevance to SDR

Fresnel-zone clearance explains reception that visual line-of-sight alone does not. A
scanner may "see" a distant [trunking site](/reference/trunking-site/) tower yet receive
a weak, fady signal because a ridge or building row cuts into the first Fresnel zone,
adding [knife-edge diffraction](/reference/knife-edge-diffraction/) loss. Raising the
antenna a few metres can lift the zone clear of the obstruction and dramatically improve
the signal — often more than adding gain would. Clearance also preserves a high
[Rician K-factor](/reference/rician-fading/) by keeping the dominant direct ray strong
relative to scatter.

For microwave point-to-point backhaul (the links that connect simulcast and multisite
infrastructure), Fresnel clearance is a hard design requirement: planners survey the
terrain profile and set tower heights so the first zone stays open across the whole hop.
[GopherTrunk](/reference/software-defined-radio/) is a receiver and does no path
planning, but the concept directly informs where to place a scanner antenna for the best
capture.

## In practice

The zone is why "just barely line-of-sight" links are unreliable: at grazing incidence
half the first zone is blocked, costing roughly 6 dB versus free space, and any further
intrusion drops the signal into the diffraction region analysed by the
[knife-edge](/reference/knife-edge-diffraction/) model. Clear the zone and the link
behaves as [free-space path loss](/reference/free-space-path-loss/) predicts.

## Sources

[^wiki]: [Fresnel zone](https://en.wikipedia.org/wiki/Fresnel_zone) — Wikipedia, on the elliptical clearance regions, the radius formula, and the 60% clearance rule.
