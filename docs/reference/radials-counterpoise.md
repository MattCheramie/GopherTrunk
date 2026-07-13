---
slug: radials-counterpoise
title: Radials & counterpoise
entry_type: term
category: antennas
description: Radials and a counterpoise form the artificial ground system beneath a vertical or monopole antenna, providing the return path for antenna current and setting its efficiency.
keywords: radials, counterpoise, ground system, ground plane, monopole, vertical antenna, elevated radials, buried radials, radial field, ground loss, quarter wave
aka: [radials, counterpoise, ground radials, radial system]
autolink: true
infobox:
  - { label: Type, value: Antenna ground system }
  - { label: Role, value: Return path for monopole current }
  - { label: Forms, value: Buried, elevated, or a counterpoise }
see_also: [ground-plane-antenna, monopole-antenna, antenna-efficiency, feedpoint-impedance, dipole-antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Counterpoise_(ground_system)
  - https://en.wikipedia.org/wiki/Radial_(radio)
---

**Radials and a counterpoise** are the artificial ground system placed under a vertical
[monopole](/reference/monopole-antenna/) antenna to give its current somewhere to return to.[^wiki]
A monopole is only half an antenna — the "missing" half is supplied by its image in a conducting
ground plane — so it needs a good electrical ground beneath it to work. **Radials** are wires laid
out from the base like spokes; a **counterpoise** is a similar system of conductors that stands in
for a real earth connection, often when a true ground is impractical. Together they set the
antenna's [efficiency](/reference/antenna-efficiency/) and stabilize its feedpoint.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="A vertical monopole element rises from a feedpoint at ground level, with many radial wires spreading outward horizontally from the base to form the ground system." xmlns="http://www.w3.org/2000/svg">
  <line x1="230" y1="30" x2="230" y2="110" stroke="currentColor" stroke-width="2"/>
  <text x="242" y="55" font-size="8.5" fill="currentColor">λ/4 element</text>
  <circle cx="230" cy="110" r="3.5" fill="currentColor"/>
  <text x="230" y="128" text-anchor="middle" font-size="8" fill="currentColor">feedpoint</text>
  <line x1="230" y1="110" x2="70" y2="128" stroke="currentColor" stroke-width="1"/>
  <line x1="230" y1="110" x2="130" y2="145" stroke="currentColor" stroke-width="1"/>
  <line x1="230" y1="110" x2="230" y2="150" stroke="currentColor" stroke-width="1"/>
  <line x1="230" y1="110" x2="330" y2="145" stroke="currentColor" stroke-width="1"/>
  <line x1="230" y1="110" x2="390" y2="128" stroke="currentColor" stroke-width="1"/>
  <line x1="230" y1="110" x2="150" y2="120" stroke="currentColor" stroke-width="1"/>
  <line x1="230" y1="110" x2="310" y2="120" stroke="currentColor" stroke-width="1"/>
  <text x="230" y="162" text-anchor="middle" font-size="8.5" fill="currentColor">radial ground system</text>
</svg>
<figcaption>A quarter-wave monopole with radials: the spoked wires supply the ground-return path that the monopole's image current requires.</figcaption>
</figure>

## How it works

Current fed into a monopole must complete a circuit. In an ideal installation over perfectly
conducting earth, the ground acts as a mirror, reflecting the antenna's image and supplying the
return current with no loss. Real soil is far from a perfect conductor, so the return currents
flow through lossy earth, adding resistance that dissipates power as heat — this **ground loss**
is often the single largest inefficiency of a ground-mounted vertical. Radials intercept those
return currents in low-resistance copper before they reach the lossy soil:

- **Buried (or on-ground) radials** are laid on or just under the surface. Because they work
  *with* the soil rather than resonating, more is better and length is not critical — 30 to 60+
  quarter-wave radials approach the performance of a solid ground plane, while a handful gives
  poor efficiency. This is the classic broadcast-tower ground field.
- **Elevated radials** are raised above ground and *do* resonate. A surprisingly small number —
  as few as two to four tuned quarter-wave radials — can rival dozens of buried ones, because
  they are decoupled from the lossy earth and carry the return current in the air.
- A **counterpoise** is a conductor or set of conductors serving as the ground reference when no
  earth connection is used at all — for example a metal vehicle body under a mobile whip, or a
  wire grid under a portable vertical.

The ground system also fixes the [feedpoint impedance](/reference/feedpoint-impedance/). A good
radial field brings a quarter-wave monopole to its ideal ~36 Ω; a poor one raises the apparent
resistance (extra loss resistance in series) and can shift resonance, degrading the match.

## In practice

The trade-offs are practical rather than exotic. Buried systems are mechanically robust and
invisible but need a lot of wire; elevated systems are efficient with little wire but must be
supported and kept clear of people. Radial length interacts with the design: for elevated systems,
each radial is trimmed to a quarter wavelength; for buried systems, many shorter radials can
outperform a few long ones for the same total copper. A [ground-plane antenna](/reference/ground-plane-antenna/)
is essentially a monopole whose radials have been formalized into a small set of elevated,
often downward-sloped, elements at the feedpoint.

## Relevance to SDR

For **receiving**, the demands on a ground system are gentler than for transmitting: external and
atmospheric noise usually dominate the receiver's own noise floor on the HF and low-VHF bands, so
a lossy ground attenuates signal and noise together and the signal-to-noise ratio suffers less
than raw efficiency figures suggest. Even so, a decent counterpoise stabilizes a vertical's
pattern and impedance and reduces common-mode pickup of local electrical noise, which can be the
real limit on a scanner's usable sensitivity. **GopherTrunk** never sees the ground system — it
processes IQ samples from the SDR — but the radials under a vertical are part of the physical chain
that determines how much clean signal reaches the front end, particularly for lower-frequency
monopole installations.

## Sources

[^wiki]: [Counterpoise (ground system)](https://en.wikipedia.org/wiki/Counterpoise_(ground_system)) — Wikipedia, for the role of radials and counterpoises as the return-current ground for monopole antennas.
