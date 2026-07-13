---
slug: feedpoint-impedance
title: Feedpoint impedance
entry_type: term
category: antennas
description: Feedpoint impedance is the complex impedance an antenna presents at its terminals, combining radiation and loss resistance with reactance, and it must be matched to the feedline for efficient power transfer.
keywords: feedpoint impedance, feed impedance, antenna impedance, radiation resistance, reactance, 50 ohm, 75 ohm, matching, resonance, complex impedance
aka: [feedpoint impedance, feed impedance, antenna impedance]
autolink: true
infobox:
  - { label: Type, value: Antenna terminal property }
  - { label: Form, value: R + jX (complex ohms) }
  - { label: Target, value: Match to 50 Ω (or 75 Ω) feedline }
see_also: [impedance, standing-wave-ratio, balun, antenna-tuner, dipole-antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Antenna_(radio)#Impedance
  - https://en.wikipedia.org/wiki/Feed_line
---

**Feedpoint impedance** is the [impedance](/reference/impedance/) an antenna presents at the
terminals where the feedline connects — the load the transmitter or receiver actually "sees."
It is a complex quantity, *Z* = *R* + j*X*, whose real part combines the useful **radiation
resistance** with ohmic **loss resistance**, and whose imaginary part is the **reactance** that
appears when the antenna is not at resonance.[^wiki] Matching this impedance to the feedline —
almost always 50 Ω in radio work — is what allows power to transfer without reflection, and a
mismatch shows up as an elevated [standing-wave ratio](/reference/standing-wave-ratio/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="A coaxial feedline of fifty ohms connects to an antenna's feedpoint, where the impedance is shown as radiation resistance plus loss resistance plus reactance, with a note that matching to fifty ohms transfers maximum power." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="fpiar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="30" y1="85" x2="185" y2="85" stroke="currentColor" stroke-width="2.4"/>
  <text x="100" y="76" text-anchor="middle" font-size="9" fill="currentColor">50 Ω feedline</text>
  <line x1="185" y1="85" x2="220" y2="85" stroke="currentColor" stroke-width="1.2" marker-end="url(#fpiar)"/>
  <circle cx="235" cy="85" r="6" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="235" y="112" text-anchor="middle" font-size="8.5" fill="currentColor">feedpoint</text>
  <line x1="241" y1="85" x2="300" y2="45" stroke="currentColor" stroke-width="1.8"/>
  <line x1="241" y1="85" x2="300" y2="125" stroke="currentColor" stroke-width="1.8"/>
  <rect x="330" y="45" width="120" height="70" fill="none" stroke="currentColor" stroke-width="1" stroke-opacity="0.6"/>
  <text x="390" y="68" text-anchor="middle" font-size="10" fill="currentColor">Z = R + jX</text>
  <text x="390" y="86" text-anchor="middle" font-size="8" fill="currentColor">R = R_rad + R_loss</text>
  <text x="390" y="104" text-anchor="middle" font-size="8" fill="currentColor">X = 0 at resonance</text>
</svg>
<figcaption>The feedpoint impedance is the complex load the feedline drives; matching its real part to the line and cancelling its reactance transfers maximum power.</figcaption>
</figure>

## How it works

Current flowing into the antenna does two things: it radiates energy into space, and it dissipates
a little as heat in the conductors and nearby ground. Modelled as a resistance, the radiated part
is the **radiation resistance** *R*_rad and the wasted part is the **loss resistance** *R*_loss;
the ratio *R*_rad / (*R*_rad + *R*_loss) is the antenna's radiation efficiency. On top of this
resistive part, the antenna generally stores energy in near-field electric or magnetic fields,
which appears as **reactance** *X* — capacitive (negative) when the antenna is electrically short,
inductive (positive) when it is long.

At **resonance** the reactance passes through zero and the feedpoint looks purely resistive. The
value of that resistance depends on the antenna type and the feed location:

- A half-wave [dipole](/reference/dipole-antenna/) in free space is resonant near 73 Ω resistive
  — close enough to 50 Ω that it is often fed directly.
- A quarter-wave [monopole](/reference/monopole-antenna/) over a good ground is about half that,
  near 36 Ω.
- A folded dipole steps up to roughly 300 Ω. Feeding a dipole off-centre or at the end raises the
  impedance dramatically, into the thousands of ohms at a voltage maximum.

## In practice

Getting maximum power transfer requires two conditions: the resistive parts must be equal and the
reactance must be cancelled (conjugate match). Techniques include:

- **Trimming to resonance** so *X* → 0, then relying on the natural *R* being near 50 Ω.
- **Matching networks** — a gamma match, hairpin, or transformer at the feed, or an external
  [antenna tuner](/reference/antenna-tuner/) that synthesizes the conjugate.
- A [balun](/reference/balun/), which does not itself transform 50 Ω but ensures a balanced
  antenna is fed by an unbalanced coax without common-mode current corrupting the pattern and
  shifting the apparent impedance.

Feedpoint impedance is also affected by everything nearby — height above ground, adjacent metal,
the mast — so the free-space textbook value is only a starting point; the installed impedance is
what a [vector network analyzer](/reference/vector-network-analyzer/) or antenna analyzer measures.

## Relevance to SDR

For a receive-only SDR the consequences of a mismatch are milder than for a transmitter — no
power is being reflected back into a PA — but they are real: a badly mismatched antenna reflects
signal away from the receiver, lowering the delivered signal level and worsening the noise figure
of the front end. A wideband scanning antenna is deliberately a compromise, presenting something
near 50 Ω over a broad band rather than a perfect match at any one frequency. **GopherTrunk**
never sees impedance directly — it processes the IQ samples the SDR produces — but a well-matched
feedpoint maximizes the signal-to-noise ratio reaching the decoder, which is what determines
whether a marginal [P25](/reference/p25-phase-1/) or [DMR](/reference/dmr/) signal locks.

## Sources

[^wiki]: [Antenna impedance](https://en.wikipedia.org/wiki/Antenna_(radio)#Impedance) — Wikipedia, for the decomposition into radiation resistance, loss resistance, and reactance.
