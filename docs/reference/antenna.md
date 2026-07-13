---
slug: antenna
title: Antenna
entry_type: term
category: antennas
description: An antenna is a conductor that converts electrical signals into radio waves and back; its dimensions, sized to the wavelength, set the reception of an SDR system.
keywords: antenna, aerial, radiator, resonance, reception, radiation pattern, feedpoint impedance, gain, polarization, reciprocity
aka: [antenna, aerial]
autolink: true
infobox:
  - { label: Type, value: Transducer (RF ↔ current) }
  - { label: Sized to, value: Wavelength (e.g. λ/4, λ/2) }
  - { label: Key specs, value: Resonance, gain, polarization, SWR }
see_also: [dipole-antenna, monopole-antenna, yagi-uda-antenna, antenna-gain, radiation-pattern, feedpoint-impedance, polarization, standing-wave-ratio, wavelength]
related_lessons:
  - { title: "Antennas 101", url: /learn/rf-sdr/antennas/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Antenna_(radio)
  - https://www.itu.int/rec/R-REC-V.573/en
---

An **antenna** is a conductor that converts electrical signals into
[radio waves](/reference/radio-wave/) and, on receive, converts passing radio waves
back into a tiny current.[^wiki] It sets the ceiling on everything downstream — no receiver
can recover a signal the antenna never captured. Because a metal structure that radiates
efficiently also *receives* efficiently, the same piece of hardware serves both roles;
this symmetry is the **reciprocity** principle, and it means a scanner's receive antenna
can be understood using the same transmit-side theory found in any antenna handbook.[^itu]

<figure class="figure" markdown="0">
<svg viewBox="0 0 300 150" role="img" aria-label="A vertical antenna element with concentric arcs radiating outward to represent transmitted or received waves." xmlns="http://www.w3.org/2000/svg">
  <line x1="150" y1="120" x2="150" y2="40" stroke="currentColor" stroke-width="2.5"/>
  <line x1="120" y1="120" x2="180" y2="120" stroke="currentColor" stroke-width="1.5"/>
  <g fill="none" stroke="currentColor" stroke-opacity="0.5"><path d="M150 80 A 40 40 0 0 1 190 80"/><path d="M150 80 A 70 70 0 0 1 220 80"/><path d="M150 80 A 40 40 0 0 0 110 80"/><path d="M150 80 A 70 70 0 0 0 80 80"/></g>
  <text x="150" y="140" text-anchor="middle" font-size="9" fill="currentColor">converts between waves and current</text>
</svg>
<figcaption>An antenna couples radio waves to and from the receiver; its size follows the wavelength it works at.</figcaption>
</figure>

## How it works

An antenna is a resonant structure: it works best when its physical dimensions are a
specific fraction of the signal's [wavelength](/reference/wavelength/). A quarter-wave
whip is λ/4 tall; a half-wave [dipole](/reference/dipole-antenna/) is λ/2 end to end. At
resonance the current and voltage along the conductor stand in a fixed relationship, the
reactive part of the [feedpoint impedance](/reference/feedpoint-impedance/) cancels out,
and the antenna presents a nearly real resistance — around 73 Ω for a free-space
half-wave dipole, near 36 Ω for a quarter-wave [monopole](/reference/monopole-antenna/)
over a ground plane. A feedline (typically 50 Ω coax) delivers energy to or from that
feedpoint, and how closely the two impedances match is measured by
[SWR](/reference/standing-wave-ratio/) and [return loss](/reference/return-loss/).

The energy an antenna radiates is not spread evenly in all directions. The angular map
of that energy is the [radiation pattern](/reference/radiation-pattern/); its
concentration relative to an isotropic reference is the antenna's
[gain](/reference/antenna-gain/), and the angular width of its main lobe is the
[beamwidth](/reference/beamwidth/). The orientation of the radiated electric field is the
wave's [polarization](/reference/polarization/), which is set by the antenna's geometry
and must be matched at both ends to avoid loss. Together, four properties characterize
almost any antenna:

- **Resonance and [bandwidth](/reference/bandwidth/)** — the frequency where it is
  well-matched, and how wide a range stays usable around it.
- **[Gain](/reference/antenna-gain/) and pattern** — how sharply it concentrates energy,
  and in which directions.
- **[Polarization](/reference/polarization/)** — the field orientation it favours.
- **[Feedpoint impedance](/reference/feedpoint-impedance/)** and the resulting match to
  the feedline.

## Variants

Antennas span a wide family. **Resonant wire** types such as the
[dipole](/reference/dipole-antenna/) and [monopole](/reference/monopole-antenna/) are the
simplest. **Directional arrays** like the [Yagi-Uda](/reference/yagi-uda-antenna/) add
parasitic elements to focus a beam and raise gain at the expense of coverage.
**Broadband** designs (discone, log-periodic) trade peak efficiency for a wide usable
range — attractive for scanning across many bands. **Aperture** antennas (horn,
parabolic dish) dominate at microwave frequencies, where a physically small structure can
be many wavelengths across. The right choice depends on the target band, the directions
of interest, and whether the goal is all-round monitoring or reaching one distant site.

## Relevance to SDR

For a software-defined radio receiver, the antenna is the single most cost-effective
place to improve results. Choosing one cut for the target [band](/reference/frequency-bands/),
matching its [polarization](/reference/polarization/) to the traffic (vertical for most
land-mobile and trunked systems), and placing it high with a clear path usually improves
[SNR](/reference/signal-to-noise-ratio/) more than any setting inside the radio.
GopherTrunk is purely a receive/decode chain — it has no transmit hardware and no
beamforming array — so it treats the antenna as a fixed front end and works with whatever
signal arrives at the [ADC](/reference/analog-to-digital-converter/). Understanding
antenna behaviour tells the operator why a strong nearby signal may still fail to decode
(often [multipath](/reference/multipath-propagation/) or a polarization mismatch) and why
a modest antenna on a rooftop routinely beats a high-gain one indoors.

## Sources

[^wiki]: [Antenna (radio)](https://en.wikipedia.org/wiki/Antenna_(radio)) — Wikipedia, for the definition, reciprocity, and key properties of antennas.
[^itu]: [Recommendation ITU-R V.573: Radiocommunication vocabulary](https://www.itu.int/rec/R-REC-V.573/en) — International Telecommunication Union, for standardized definitions of antenna and radiation terms.
