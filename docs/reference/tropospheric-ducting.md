---
slug: tropospheric-ducting
title: Tropospheric ducting
entry_type: term
category: propagation
description: Tropospheric ducting is a VHF/UHF over-horizon mode where a temperature inversion traps radio waves in a low-atmosphere duct, carrying signals hundreds of kilometres.
keywords: tropospheric ducting, tropo, temperature inversion, atmospheric duct, VHF propagation, UHF propagation, refractive index, over the horizon, band opening
aka: [tropo, tropospheric ducting, atmospheric duct]
autolink: true
infobox:
  - { label: Type, value: VHF/UHF over-horizon mode }
  - { label: Mechanism, value: Trapping in a refractive duct }
  - { label: Cause, value: Temperature/humidity inversion }
see_also: [radio-horizon, radio-propagation, refraction, atmospheric-absorption, broadcast-fm]
cite_urls:
  - https://en.wikipedia.org/wiki/Tropospheric_propagation
  - https://en.wikipedia.org/wiki/Atmospheric_duct
---

**Tropospheric ducting** is a VHF/UHF propagation mode in which a layer of the lower
atmosphere traps a [radio wave](/reference/radio-wave/) and guides it far beyond the
normal [radio horizon](/reference/radio-horizon/), sometimes for many hundreds of
kilometres.[^wiki] It happens when a temperature or humidity inversion bends the wave
back toward the ground faster than the earth curves away, creating a natural waveguide
in the troposphere.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A curved earth with a transmitter, a horizontal inversion layer above the surface, and a radio ray bouncing between the inversion boundary and the ground to reach a receiver far past the horizon." xmlns="http://www.w3.org/2000/svg">
  <path d="M10 150 Q230 118 450 150" fill="none" stroke="currentColor" stroke-opacity="0.4" stroke-width="1.4"/>
  <path d="M10 70 Q230 52 450 70" fill="none" stroke="currentColor" stroke-opacity="0.5" stroke-dasharray="6 4"/><text x="14" y="46" font-size="9" fill="currentColor">inversion layer (duct top)</text>
  <line x1="55" y1="138" x2="55" y2="112" stroke="currentColor" stroke-width="2"/><text x="55" y="153" text-anchor="middle" font-size="8" fill="currentColor">TX</text>
  <line x1="410" y1="140" x2="410" y2="116" stroke="currentColor" stroke-width="2"/><text x="410" y="153" text-anchor="middle" font-size="8" fill="currentColor">RX</text>
  <path d="M55 112 Q150 60 230 108 Q310 60 410 116" fill="none" stroke="currentColor" stroke-width="1.5" marker-end="url(#tdar)"/>
  <text x="150" y="132" font-size="8" fill="currentColor" fill-opacity="0.7">trapped ray hops inside the duct</text>
  <defs><marker id="tdar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>An inversion caps a duct near the surface; the ray refracts down at the boundary and up off the ground, ferrying VHF/UHF signals well past the horizon.</figcaption>
</figure>

## How it works

Radio waves bend according to gradients in the atmosphere's refractive index, which
depends on temperature, pressure, and water-vapour content. Normally the index falls
gently with height, giving the wave a slight downward curve — the reason the true radio
horizon is a bit farther than the visual one. When a **temperature inversion** puts
warm, dry air over cooler, moister air, the index drops sharply at the boundary. A wave
climbing into that gradient is refracted back down strongly enough to be trapped: it
skips along between the inversion "lid" and the ground (or between two elevated layers),
losing little energy per bounce.

Because the duct is only tens to a few hundred metres thick, the effect is strongly
frequency-dependent — the wavelength must be small relative to the duct for the trapping
to hold, so ducting favours VHF, UHF, and microwave rather than the low bands. Common
duct-forming conditions include:

- **Surface radiation inversions** on calm, clear nights as the ground cools.
- **Subsidence inversions** under stable high-pressure systems, especially in summer.
- **Evaporation ducts** over warm seas, where a humid layer clings to the water — the
  reason marine and coastal paths open so often and so far.

Ducted signals can arrive with surprisingly little loss but also with fading and
multipath, since the same signal may travel several duct paths of different length.

## In practice

Ducting produces the "band openings" that let VHF/UHF operators, TV and
[FM broadcast](/reference/broadcast-fm/) DXers, and amateurs work stations hundreds of
kilometres away that are normally over the horizon. Openings track weather rather than
the sun, so they favour stable high-pressure spells and warm coastlines and can last
minutes or days. The flip side for microwave link engineers is interference: a duct can
carry a distant co-channel transmitter into a receiver that would otherwise never hear
it.

## Relevance to SDR

Tropospheric ducting matters directly to VHF/UHF SDR listening because it sits in the
same bands a scanner uses. During an opening, an [RTL-SDR](/reference/rtl-sdr/) or
[Airspy](/reference/airspy/) may suddenly pull in distant [FM](/reference/broadcast-fm/)
stations, pagers, or land-mobile traffic from far outside the normal
[coverage](/reference/radio-horizon/) footprint. For a trunking scanner like
**GopherTrunk**, this is usually a nuisance rather than a feature: a ducted-in distant
site on the same frequency as the local control channel can corrupt decoding or cause
brief co-channel interference. GopherTrunk does not model the troposphere — it simply
decodes whatever reaches the receiver — but recognising a duct explains otherwise
baffling reception of a system that "shouldn't" be audible.

## Sources

[^wiki]: [Tropospheric propagation](https://en.wikipedia.org/wiki/Tropospheric_propagation) — Wikipedia, on refractive-index gradients, temperature inversions, and atmospheric ducts at VHF/UHF.
