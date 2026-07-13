---
slug: rain-fade
title: Rain fade
entry_type: term
category: propagation
description: Rain fade is the attenuation of microwave and satellite radio signals by rainfall, growing severe above about 10 GHz and driving link fade margins.
keywords: rain fade, rain attenuation, precipitation loss, Ku band, Ka band, satellite outage, ITU-R P.618, microwave link fade
aka: [rain fade, rain attenuation]
autolink: true
infobox:
  - { label: Type, value: Weather-induced attenuation }
  - { label: Onset, value: Noticeable above ~10 GHz }
  - { label: Worst for, value: Ku/Ka-band satellite, mm-wave }
see_also: [atmospheric-absorption, scattering, fade-margin, link-budget, free-space-path-loss]
cite_urls:
  - https://en.wikipedia.org/wiki/Rain_fade
  - https://en.wikipedia.org/wiki/Attenuation
---

**Rain fade** is the attenuation of a radio signal caused by rainfall (and, similarly,
snow, hail, and wet foliage) along the propagation path.[^wiki] It becomes significant at
microwave frequencies above roughly 10 GHz and worsens with frequency, making it the
dominant weather impairment for Ku- and Ka-band satellite links, millimetre-wave backhaul,
and 5G mm-wave cells. Raindrops both absorb energy and [scatter](/reference/scattering/) it
out of the path, so a heavy downpour can drive an otherwise healthy link into outage.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="A satellite downlink beam passing through a rain cell loses strength; a bar to the right shows attenuation rising steeply with frequency from C band through Ku to Ka band." xmlns="http://www.w3.org/2000/svg">
  <circle cx="55" cy="30" r="10" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="55" y="52" text-anchor="middle" font-size="8" fill="currentColor">sat</text>
  <line x1="60" y1="40" x2="150" y2="140" stroke="currentColor" stroke-width="1.4" marker-end="url(#rnar)"/>
  <g stroke="currentColor" stroke-width="1">
    <line x1="95" y1="60" x2="90" y2="72"/><line x1="110" y1="66" x2="105" y2="78"/>
    <line x1="125" y1="72" x2="120" y2="84"/><line x1="105" y1="82" x2="100" y2="94"/>
    <line x1="120" y1="92" x2="115" y2="104"/>
  </g>
  <text x="140" y="70" font-size="8" fill="currentColor">rain cell</text>
  <text x="150" y="153" text-anchor="middle" font-size="8" fill="currentColor">ground RX</text>
  <line x1="255" y1="140" x2="255" y2="20" stroke="currentColor" stroke-width="1.1"/>
  <line x1="255" y1="140" x2="450" y2="140" stroke="currentColor" stroke-width="1.1"/>
  <path d="M255 138 L300 132 L345 112 L390 72 L435 32" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <text x="300" y="128" font-size="8" fill="currentColor">C</text>
  <text x="360" y="95" font-size="8" fill="currentColor">Ku</text>
  <text x="425" y="52" font-size="8" fill="currentColor">Ka</text>
  <text x="350" y="158" text-anchor="middle" font-size="8" fill="currentColor">attenuation vs frequency</text>
  <defs><marker id="rnar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Rain along the path absorbs and scatters energy; the loss climbs steeply with frequency from C band to Ka band and above.</figcaption>
</figure>

## How it works

Raindrops are dielectric spheres roughly comparable in size to microwave wavelengths, so
they interact strongly through two mechanisms:

- **Absorption.** Water is lossy at microwave frequencies; energy driving the polar water
  molecules is dissipated as heat.
- **Scattering.** In the Mie regime, where drop diameter approaches the wavelength, drops
  [scatter](/reference/scattering/) a large fraction of the incident energy out of the
  beam.

Specific attenuation (dB per kilometre) rises with **rain rate** and with **frequency**.
At C band (~4–6 GHz) it is nearly negligible; at Ku band (~12–18 GHz) a heavy storm costs
several dB; at Ka band (~20–30 GHz) and above, the same storm can cost tens of dB —
enough to break the link. Because a satellite's slant path only crosses the rain cell for
a limited height, the *effective path length* through rain, not the whole path, sets the
loss. The ITU-R P.618 model combines rain-rate statistics with path geometry to predict
outage percentages, and rain also *depolarises* the signal, degrading systems that reuse
frequencies on orthogonal polarisations.

## In practice

Because rain is intermittent, links to rain-prone frequencies are designed with a
**[fade margin](/reference/fade-margin/)** sized to a target availability (e.g. 99.9% of
the year). Two active techniques reduce the required margin:

- **Adaptive coding and modulation (ACM):** the system drops to a more robust modulation
  and heavier [FEC](/reference/forward-error-correction/) during a fade, trading data rate
  for a lower usable [SNR](/reference/signal-to-noise-ratio/) so the link stays up.
- **Site diversity:** two ground stations far enough apart are rarely rained on at once,
  so the network switches to whichever has the clear sky.

Uplink power control, where the transmitter raises power during rain, is a further remedy.

## Relevance to SDR

Rain fade is essentially absent from the VHF/UHF land-mobile bands that trunking scanners
monitor — P25, DMR, TETRA, and NXDN sit far below the frequencies where rain matters, so
weather has negligible direct effect on their [link budget](/reference/link-budget/). It
becomes a first-order concern only for the microwave and satellite links an SDR hobbyist
might also receive: Ku/Ka-band satellite downlinks, DVB-S feeds, and mm-wave systems all
show measurable rain fade.

[GopherTrunk](/reference/software-defined-radio/) targets terrestrial land-mobile
protocols, so rain fade is not a factor in its decode path; it is included here as the
canonical weather-driven attenuation mechanism that shapes the design of any higher-band RF
link. Its close cousin is clear-air [atmospheric absorption](/reference/atmospheric-absorption/),
which sets the baseline loss on which rain fade adds.

## Sources

[^wiki]: [Rain fade](https://en.wikipedia.org/wiki/Rain_fade) — Wikipedia, on precipitation attenuation, its frequency dependence, and mitigation by fade margin and diversity.
