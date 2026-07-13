---
slug: scattering
title: Scattering
entry_type: term
category: propagation
description: Scattering is the redirection of radio energy into many directions by rough surfaces, particles, or irregularities smaller than or near the wavelength.
keywords: scattering, radio scattering, Rayleigh scattering, diffuse reflection, rough surface, rain scatter, tropospheric scatter, particle scatter
aka: [scattering, radio scattering]
autolink: true
infobox:
  - { label: Type, value: Multi-directional redirection }
  - { label: Causes, value: Rough surfaces, particles, irregularities }
  - { label: Strong when, value: Scatterer size ≈ wavelength }
see_also: [multipath-propagation, rain-fade, atmospheric-absorption, refraction, radio-propagation]
cite_urls:
  - https://en.wikipedia.org/wiki/Scattering
  - https://en.wikipedia.org/wiki/Rayleigh_scattering
---

**Scattering** is the redirection of radio energy into many directions at once when a wave
meets a surface or object that is rough, irregular, or comparable in size to its
[wavelength](/reference/wavelength/).[^wiki] Unlike a mirror-like reflection off a smooth
plane, scattering spreads a portion of the incident power over a broad angular range. It
is a major contributor to [multipath](/reference/multipath-propagation/) in cluttered
environments and the mechanism behind rain scatter, tropospheric scatter, and radar
returns from precipitation and rough terrain.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="An incoming plane wave strikes a rough, irregular surface and is redirected outward as many rays fanning into different directions, illustrating diffuse scattering." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="45" x2="150" y2="90" stroke="currentColor" stroke-width="1.5" marker-end="url(#scar)"/>
  <text x="70" y="45" font-size="9" fill="currentColor">incident</text>
  <path d="M120 120 L140 100 L160 118 L180 98 L200 120 L220 102 L240 120 L260 100 L280 120" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="200" y="138" text-anchor="middle" font-size="9" fill="currentColor">rough surface</text>
  <line x1="165" y1="95" x2="150" y2="40" stroke="currentColor" stroke-width="1.1" marker-end="url(#scar)"/>
  <line x1="175" y1="95" x2="230" y2="35" stroke="currentColor" stroke-width="1.1" marker-end="url(#scar)"/>
  <line x1="185" y1="95" x2="300" y2="55" stroke="currentColor" stroke-width="1.1" marker-end="url(#scar)"/>
  <line x1="180" y1="95" x2="360" y2="90" stroke="currentColor" stroke-width="1.1" marker-end="url(#scar)"/>
  <text x="300" y="30" font-size="9" fill="currentColor">scattered rays</text>
  <defs><marker id="scar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A rough or wavelength-scale object redirects incident energy into many directions rather than a single specular beam.</figcaption>
</figure>

## How it works

Whether a surface reflects cleanly or scatters is judged against the wavelength by the
**Rayleigh roughness criterion**: irregularities much smaller than λ look smooth and
reflect specularly, while irregularities comparable to or larger than λ scatter diffusely.
The same wave that mirrors off a calm building wall at UHF scatters strongly at millimetre
wave, because the wall's texture is now large relative to the wavelength.

Scattering by small particles has a strong frequency dependence:

- **Rayleigh regime** (particle radius ≪ λ, e.g. drizzle or dust at microwave): scattered
  power rises steeply with frequency (roughly as the fourth power), so higher bands are hit
  hardest. This is the same law that makes the sky blue.
- **Mie regime** (particle size ≈ λ, e.g. raindrops at millimetre wave): scattering is
  strong and less steeply frequency-dependent, and the redirected power that leaves the
  path becomes attenuation — the dominant contribution to [rain fade](/reference/rain-fade/).

## Variants

Several named propagation modes are scattering in disguise:

- **Tropospheric scatter (troposcatter):** turbulent refractive-index blobs high in the
  troposphere scatter a faint fraction of a UHF/SHF beam forward, supporting reliable links
  of hundreds of kilometres well beyond the horizon.
- **Rain scatter:** precipitation redirects microwave energy, both attenuating the intended
  path and creating unintended off-axis paths (exploited by amateurs, and a source of
  interference in satellite bands).
- **Rough-surface scatter:** terrain, sea, and foliage scatter contribute the diffuse rays
  that populate the [multipath](/reference/multipath-propagation/) channel and drive
  [Rayleigh fading](/reference/rayleigh-fading/).

## Relevance to SDR

For land-mobile scanning, the practical face of scattering is **clutter multipath**: the
countless diffuse reflections off buildings, vehicles, and vegetation that combine at the
antenna to produce fading and intersymbol interference. There is no clean specular echo to
equalise away; the channel is a statistical sum of scattered arrivals, which is exactly why
[Rayleigh](/reference/rayleigh-fading/) statistics apply in dense urban settings.

Scattering matters more as frequency rises. At the VHF/UHF bands used by P25, DMR, and
TETRA it is a moderate multipath contributor; at the millimetre-wave bands of 5G and
satellite downlinks it becomes a first-order loss and fade mechanism, driving the need for
higher fade margins and adaptive coding. [GopherTrunk](/reference/software-defined-radio/)
does not model scattering; it is a channel property that shapes the signal arriving at the
receiver, and it appears indirectly as fading and elevated
[EVM](/reference/error-vector-magnitude/) in the decode chain.

## Sources

[^wiki]: [Scattering](https://en.wikipedia.org/wiki/Scattering) — Wikipedia, on the redirection of waves by rough surfaces and particles, and the Rayleigh and Mie regimes.
