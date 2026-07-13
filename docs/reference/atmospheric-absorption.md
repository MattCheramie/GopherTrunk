---
slug: atmospheric-absorption
title: Atmospheric absorption
entry_type: term
category: propagation
description: Atmospheric absorption is the loss of radio energy to oxygen and water-vapour resonances, creating strong peaks and clear windows at millimetre-wave bands.
keywords: atmospheric absorption, gaseous absorption, oxygen absorption, water vapour absorption, 60 GHz, 22 GHz, millimetre wave window, ITU-R P.676
aka: [atmospheric absorption, gaseous absorption, atmospheric attenuation]
autolink: true
infobox:
  - { label: Type, value: Clear-air molecular loss }
  - { label: Main absorbers, value: "O2 (~60 GHz), H2O (~22, 183 GHz)" }
  - { label: Below ~10 GHz, value: Negligible }
see_also: [rain-fade, scattering, free-space-path-loss, frequency-bands, radio-propagation]
cite_urls:
  - https://en.wikipedia.org/wiki/Radio_propagation
  - https://en.wikipedia.org/wiki/Absorption_(electromagnetic_radiation)
---

**Atmospheric absorption** is the loss of radio energy to the molecules of the atmosphere
itself — chiefly oxygen and water vapour — as a wave travels through clear air.[^wiki]
Unlike [rain fade](/reference/rain-fade/), which depends on weather, this is an
ever-present, frequency-dependent loss that stays negligible below about 10 GHz but rises
into sharp resonance peaks at millimetre-wave frequencies. Those peaks and the low-loss
gaps between them — the atmospheric **windows** — shape which bands are usable for
terrestrial and satellite links.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="A curve of atmospheric attenuation versus frequency from 1 to 300 gigahertz, low at the left, with tall absorption peaks at 22, 60, and 183 gigahertz for water vapour and oxygen, and low-loss window regions between them." xmlns="http://www.w3.org/2000/svg">
  <line x1="35" y1="20" x2="35" y2="135" stroke="currentColor" stroke-width="1.1"/>
  <line x1="35" y1="135" x2="445" y2="135" stroke="currentColor" stroke-width="1.1" marker-end="url(#aaar)"/>
  <text x="240" y="155" text-anchor="middle" font-size="9" fill="currentColor">frequency (GHz, log)</text>
  <text x="24" y="30" font-size="9" fill="currentColor" transform="rotate(-90 24 80)">attenuation</text>
  <path d="M40 128 Q120 122 175 118 Q190 90 205 118 Q250 122 300 128 Q320 40 340 128 Q380 118 405 96 Q418 44 430 108" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="205" y="82" text-anchor="middle" font-size="7" fill="currentColor">H2O 22</text>
  <text x="330" y="34" text-anchor="middle" font-size="7" fill="currentColor">O2 60</text>
  <text x="418" y="38" text-anchor="middle" font-size="7" fill="currentColor">H2O 183</text>
  <text x="120" y="115" font-size="7" fill="currentColor">window</text>
  <text x="270" y="120" font-size="7" fill="currentColor">window</text>
  <defs><marker id="aaar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Clear-air attenuation is low below 10 GHz, then peaks at molecular resonances (H2O ~22/183 GHz, O2 ~60 GHz) with usable windows between.</figcaption>
</figure>

## How it works

Gas molecules have quantised rotational energy states. When the radio frequency matches
the energy of a transition, the molecule absorbs a photon, and the wave loses energy — a
resonance line. Two gases dominate at radio frequencies:

- **Oxygen (O₂).** A dense cluster of magnetic-dipole lines near **60 GHz** produces a
  very strong absorption band (roughly 15 dB/km at sea level), plus an isolated line at
  118 GHz.
- **Water vapour (H₂O).** Electric-dipole lines at about **22 GHz** and a much stronger one
  near **183 GHz**, with absorption scaling with humidity.

Between these lines lie **atmospheric windows** — bands around 35, 94, 140, and 220 GHz
where loss is comparatively low. Absorption also depends on path geometry: a satellite link
at low elevation cuts a long slant through the dense lower atmosphere and suffers far more
than one straight overhead. The ITU-R P.676 model gives the specific attenuation of oxygen
and water vapour line by line for link planning.

## Variants

The 60 GHz oxygen peak is a double-edged tool. Its heavy loss makes it useless for
long-range links, but ideal for **short-range, frequency-reuse-dense** systems: 60 GHz
Wi-Fi (802.11ad/ay) and indoor mm-wave backhaul exploit the fact that signals die out
within a room or a city block, so the same channel can be reused nearby with little
interference and good security. The windows, by contrast, are chosen for
Earth–space links, mm-wave imaging, and radio astronomy.

## Relevance to SDR

For the VHF/UHF land-mobile bands that trunking scanners monitor, atmospheric absorption is
utterly negligible — P25, DMR, TETRA, and NXDN operate three orders of magnitude below the
first significant resonance, so molecular loss never enters their
[link budget](/reference/link-budget/). It becomes relevant only when an SDR user reaches up
into the microwave and millimetre-wave world: high-band satellite downlinks, 5G mm-wave,
and experimental links must respect these peaks and windows in their band planning.

[GopherTrunk](/reference/software-defined-radio/) decodes terrestrial land-mobile signals
and does not model gaseous absorption; it is included here as the fundamental clear-air loss
that, together with [free-space path loss](/reference/free-space-path-loss/) and
[rain fade](/reference/rain-fade/), determines the total attenuation of any high-frequency
link. The key intuition for band planners: absorption sets *which* millimetre-wave
frequencies are even worth using.

## Sources

[^wiki]: [Radio propagation](https://en.wikipedia.org/wiki/Radio_propagation) — Wikipedia, on gaseous absorption by oxygen and water vapour and the resulting atmospheric windows.
