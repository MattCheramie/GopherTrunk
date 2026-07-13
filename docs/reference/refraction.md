---
slug: refraction
title: Refraction
entry_type: term
category: propagation
description: Refraction is the bending of radio waves as they pass through a medium of changing refractive index, extending the radio horizon beyond the visual one.
keywords: refraction, atmospheric refraction, refractive index, k-factor, four-thirds earth, radio horizon, ducting, bending of radio waves
aka: [refraction, atmospheric refraction]
autolink: true
infobox:
  - { label: Type, value: Wave-bending phenomenon }
  - { label: Cause, value: Gradient in refractive index }
  - { label: Standard model, value: k = 4/3 effective Earth }
see_also: [radio-horizon, tropospheric-ducting, knife-edge-diffraction, ionospheric-propagation, radio-propagation]
cite_urls:
  - https://en.wikipedia.org/wiki/Refraction
  - https://en.wikipedia.org/wiki/Atmospheric_refraction
---

**Refraction** is the bending of a radio wave's path as it travels through a medium whose
refractive index changes with position.[^wiki] In the atmosphere, air density, pressure,
temperature, and humidity normally decrease with altitude, so the refractive index falls
with height and radio rays curve gently **downward**, following the Earth's curvature and
reaching slightly farther than a straight line would. This is why the
[radio horizon](/reference/radio-horizon/) is a little beyond the visual horizon.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="Over a curved Earth, a straight optical ray leaves the surface and departs into space, while a refracted radio ray curves downward to follow the curvature and reach farther around the horizon." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 150 Q230 100 440 150" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <text x="230" y="145" text-anchor="middle" font-size="9" fill="currentColor">Earth surface</text>
  <line x1="55" y1="118" x2="55" y2="95" stroke="currentColor" stroke-width="2"/><text x="55" y="90" text-anchor="middle" font-size="9" fill="currentColor">TX</text>
  <line x1="62" y1="100" x2="440" y2="35" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3"/>
  <text x="330" y="45" font-size="8" fill="currentColor">straight (optical)</text>
  <path d="M62 100 Q250 78 435 118" fill="none" stroke="currentColor" stroke-width="1.5" marker-end="url(#refar)"/>
  <text x="250" y="72" font-size="8" fill="currentColor">refracted (radio)</text>
  <defs><marker id="refar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The refracted radio ray curves downward, reaching beyond the straight-line optical horizon.</figcaption>
</figure>

## How it works

By Snell's law, a wave crossing between regions of differing refractive index changes
direction; a continuous gradient bends it along a smooth curve. Two ways of quantifying
atmospheric refraction are standard:

- **Refractivity gradient.** The refractive index of air is only slightly above 1, so
  engineers work with *refractivity* `N = (n − 1)·10⁶`. Its rate of decrease with height
  sets how sharply rays bend.
- **The k-factor (effective Earth radius).** Rather than track curved rays over a curved
  Earth, the geometry is flattened by pretending the Earth has an inflated radius `k·R`
  and letting the rays travel straight. Standard atmosphere gives **k = 4/3**, the
  familiar "four-thirds Earth" that extends the horizon by about 15% versus the geometric
  value.

Conditions that steepen the gradient bend rays harder. A strong temperature inversion can
push k very high or even trap rays in a [tropospheric duct](/reference/tropospheric-ducting/),
carrying VHF/UHF signals hundreds of kilometres. A sub-standard gradient (k < 1) bends rays
upward, *shortening* the horizon and even blocking paths that normally close.

## Variants

Refraction operates at every layer. In the troposphere it is the mild, ever-present
bending described above. In the [ionosphere](/reference/ionospheric-propagation/) the
mechanism is far stronger: free electrons make the refractive index depend on frequency,
bending HF waves so sharply they return to Earth — the basis of long-distance shortwave
propagation. The same physics, applied to light, is what makes a spoon look bent in a
glass of water.

## Relevance to SDR

For a scanner, tropospheric refraction is the everyday reason coverage reaches a bit past
line-of-sight. The k = 4/3 model is baked into how coverage maps and
[radio-horizon](/reference/radio-horizon/) calculators estimate range for a given antenna
height, so a [trunking site](/reference/trunking-site/) reliably serves users somewhat
beyond the geometric horizon. When the weather sets up a strong inversion, enhanced
refraction and ducting can briefly deliver distant VHF/UHF systems from far outside the
normal service area — a familiar "band opening" for hobbyists.

Refraction also subtly shifts the apparent elevation of satellites near the horizon, a
correction that precise [GNSS](/reference/gps-gnss/) and satellite-tracking receivers
account for. [GopherTrunk](/reference/software-defined-radio/) does no propagation
modelling itself; refraction is a property of the channel that changes which signals arrive
at the antenna, not something the decoder computes.

## In practice

The single most useful takeaway is the four-thirds-Earth rule: it lets a planner treat
rays as straight while still crediting the extra range refraction provides, and it pairs
with [knife-edge diffraction](/reference/knife-edge-diffraction/) and
[Fresnel-zone](/reference/fresnel-zone/) clearance to predict real coverage over terrain.

## Sources

[^wiki]: [Atmospheric refraction](https://en.wikipedia.org/wiki/Atmospheric_refraction) — Wikipedia, on the downward bending of rays, the refractivity gradient, and the effective-Earth-radius model.
