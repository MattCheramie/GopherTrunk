---
slug: antenna-efficiency
title: Antenna efficiency
entry_type: term
category: antennas
description: Antenna efficiency is the fraction of power delivered to an antenna that is actually radiated rather than dissipated as heat, set by the ratio of radiation resistance to total resistance.
keywords: antenna efficiency, radiation efficiency, radiation resistance, loss resistance, ohmic loss, ground loss, gain vs directivity, electrically small antenna
aka: [antenna efficiency, radiation efficiency]
autolink: true
infobox:
  - { label: Type, value: Antenna performance ratio }
  - { label: Formula, value: η = R_rad / (R_rad + R_loss) }
  - { label: Links gain to, value: Directivity (G = η · D) }
see_also: [antenna-gain, feedpoint-impedance, monopole-antenna, radials-counterpoise, decibel]
cite_urls:
  - https://en.wikipedia.org/wiki/Antenna_efficiency
---

**Antenna efficiency** (more precisely, radiation efficiency) is the fraction of the power
delivered to an antenna that leaves it as radio waves, the rest being lost as heat in the
conductors and the surrounding ground.[^wiki] It is written as a ratio η between 0 and 1, or as a
percentage, and it is the factor that separates an antenna's **gain** from its **directivity**:
gain is directivity scaled down by efficiency. A directional antenna can concentrate energy
beautifully yet still be a poor radiator if it wastes most of the power as heat.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Input power entering an antenna splits into a large radiated fraction leaving as radio waves and a small lost fraction dissipated as heat, with efficiency defined as the radiated fraction over the total." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="aear" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="55" y="72" text-anchor="middle" font-size="9" fill="currentColor">P_in</text>
  <line x1="70" y1="75" x2="130" y2="75" stroke="currentColor" stroke-width="1.4" marker-end="url(#aear)"/>
  <rect x="135" y="52" width="70" height="46" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="170" y="79" text-anchor="middle" font-size="9" fill="currentColor">antenna</text>
  <line x1="205" y1="66" x2="320" y2="40" stroke="currentColor" stroke-width="2" marker-end="url(#aear)"/>
  <text x="360" y="42" font-size="9" fill="currentColor">radiated (η)</text>
  <line x1="205" y1="86" x2="320" y2="112" stroke="currentColor" stroke-width="1" stroke-dasharray="3 3" marker-end="url(#aear)"/>
  <text x="360" y="115" font-size="9" fill="currentColor">heat (1 − η)</text>
  <text x="230" y="140" text-anchor="middle" font-size="9" fill="currentColor">η = P_radiated / P_in</text>
</svg>
<figcaption>Efficiency is the split between power that radiates and power that turns into heat in the conductors and ground.</figcaption>
</figure>

## How it works

Model the antenna's [feedpoint](/reference/feedpoint-impedance/) resistance as two series parts:
the **radiation resistance** *R*_rad, which represents power that escapes as radio waves, and the
**loss resistance** *R*_loss, which represents ohmic heating in the metal, dielectric losses, and
— for ground-mounted verticals — currents flowing through lossy soil. Since the same current
flows through both, the efficiency is simply the ratio of the useful resistance to the total:

> η = *R*_rad / (*R*_rad + *R*_loss).

The consequence for [gain](/reference/antenna-gain/) is direct. Directivity *D* describes only
the *shape* of the pattern — how well the antenna concentrates whatever it radiates — while gain
*G* also accounts for the losses:

> *G* = η · *D*,  or in decibels,  *G*(dBi) = *D*(dBi) − 10·log₁₀(1/η).

An antenna that is 50% efficient (η = 0.5) throws away 3 dB relative to its directivity.

## In practice

Efficiency becomes the dominant problem for **electrically small** antennas — those much shorter
than a quarter wavelength, such as a whip on a handheld or a ferrite loop for LF/MF. As an antenna
shrinks, its radiation resistance falls toward a fraction of an ohm while the loss resistance
stays roughly fixed, so η plummets. This is why a mobile "rubber duck" can be tens of decibels
down on a full-size antenna, and why a small transmit loop needs heroic conductor sizing to keep
losses low.

For a ground-mounted [monopole](/reference/monopole-antenna/), the biggest lever is the ground
system. Return currents flow through the earth beneath the antenna, and lossy soil adds directly
to *R*_loss. Laying an extensive set of [radials](/reference/radials-counterpoise/) provides a
low-resistance path for those currents, sometimes lifting efficiency from a dismal 20% over poor
soil to well above 90%. Conductor size, connector and coax losses, and lossy loading coils all
subtract further.

## Relevance to SDR

For receiving, low antenna efficiency is often tolerable in the crowded VHF/UHF bands a scanner
watches, because external and man-made noise usually dominate the receiver's own noise floor — an
inefficient antenna attenuates the wanted signal and the ambient noise together, leaving the
signal-to-noise ratio roughly unchanged. Below about 30 MHz, and at the noise-quiet upper UHF and
microwave bands, efficiency matters more because the receiver's [noise figure](/reference/noise-figure/)
can become the limit, so throwing away signal directly costs sensitivity. **GopherTrunk** cannot
see or correct for antenna efficiency; it acts entirely on the samples the SDR delivers. But
efficiency is one of the physical factors — alongside gain, matching, and feedline loss — that set
how much signal reaches the front end and therefore whether a weak trunking channel decodes.

## Sources

[^wiki]: [Antenna efficiency](https://en.wikipedia.org/wiki/Antenna_efficiency) — Wikipedia, for the radiation/loss resistance definition and the gain-equals-efficiency-times-directivity relation.
