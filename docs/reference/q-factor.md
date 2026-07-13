---
slug: q-factor
title: Q Factor (Quality Factor)
entry_type: term
category: rf-fundamentals
description: The Q factor measures a resonator's selectivity as centre frequency divided by −3 dB bandwidth; high Q means low loss, narrow bandwidth, and sharp filtering.
keywords: Q factor, quality factor, loaded Q, unloaded Q, resonator, bandwidth, selectivity, resonance, damping
aka: [Q, quality factor]
autolink: true
infobox:
  - { label: Symbol, value: "Q" }
  - { label: Unit, value: "dimensionless" }
  - { label: Formula, value: "Q = f0 / Δf(−3 dB)" }
see_also: [resonance, rf-filter, crystal-filter, cavity-filter, phase-noise, bandwidth]
cite_urls:
  - https://en.wikipedia.org/wiki/Q_factor
  - https://en.wikipedia.org/wiki/Resonance
---

**The Q factor** (quality factor) is a dimensionless measure of how selective and
low-loss a resonant system is, defined as its centre frequency divided by its −3 dB
bandwidth: **Q = f₀ / Δf**.[^wiki] Equivalently it is 2π times the energy stored per
cycle divided by the energy dissipated per cycle. A high Q means a sharp, narrow
[resonance](/reference/resonance/) that rings for many cycles; a low Q means a broad,
heavily damped response that dies quickly. Q sets the selectivity of tuned circuits,
filters, and oscillators.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="Two resonance curves centred on the same frequency f0: a tall narrow high-Q peak and a short broad low-Q peak, with the minus 3 dB bandwidth marked on each." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="qfar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="50" y1="20" x2="50" y2="140" stroke="currentColor" stroke-width="1" marker-end="url(#qfar)"/>
  <line x1="50" y1="140" x2="440" y2="140" stroke="currentColor" stroke-width="1" marker-end="url(#qfar)"/>
  <text x="235" y="155" font-size="9" text-anchor="middle" fill="currentColor">frequency</text>
  <text x="235" y="152" font-size="8" text-anchor="middle" fill="currentColor"> </text>
  <line x1="245" y1="30" x2="245" y2="140" stroke="currentColor" stroke-width="0.7" stroke-dasharray="3 3"/>
  <text x="245" y="152" font-size="9" text-anchor="middle" fill="currentColor">f0</text>
  <path d="M120 138 Q240 -30 360 138" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="300" y="45" font-size="9" fill="currentColor">high Q (narrow)</text>
  <path d="M60 138 Q245 90 430 138" fill="none" stroke="currentColor" stroke-width="1.5" stroke-dasharray="5 3"/>
  <text x="60" y="105" font-size="9" fill="currentColor">low Q (broad)</text>
  <line x1="205" y1="70" x2="285" y2="70" stroke="currentColor" stroke-width="1"/>
  <text x="245" y="66" font-size="8" text-anchor="middle" fill="currentColor">Δf (−3 dB)</text>
</svg>
<figcaption>Q is centre frequency over −3 dB bandwidth: the tall narrow curve has high Q and sharp selectivity, the broad curve has low Q. Same f0, very different bandwidths.</figcaption>
</figure>

## How it works

A resonator stores energy and swaps it back and forth between two forms — in an LC
circuit, between the inductor's magnetic field and the capacitor's electric field.
Every cycle a little energy is lost to resistance, radiation, or other damping. Q
captures the ratio of what is stored to what is lost:

*Q = 2π × (energy stored) / (energy dissipated per cycle)*

Because a lightly damped resonator loses little per cycle, it responds strongly only
to frequencies very close to [resonance](/reference/resonance/), producing a tall,
narrow peak — and, viewed in time, it rings for roughly Q cycles after being struck.
For a series RLC circuit *Q = (1/R)·√(L/C)*, so lowering the loss resistance *R*
raises Q. The link to bandwidth follows directly: a narrow response equals a high Q,
which is why *Q = f₀/Δf* is the practical measuring recipe.

## Loaded versus unloaded Q

Two Q values matter and are often confused:

- **Unloaded Q (Qu)** is the resonator's own quality, set only by its internal
  losses, with nothing drawing power from it. It is the ceiling the component can
  reach.
- **Loaded Q (QL)** is what you actually see once the resonator is connected to a
  source and load, which add their own damping. Coupling to external circuitry always
  lowers Q, so QL ≤ Qu.

A filter designer trades these deliberately: tighter coupling gives wider bandwidth
(lower loaded Q) but lower insertion loss, while loose coupling gives a narrower,
sharper response at the cost of more loss. Typical unloaded Q spans a huge range —
tens for a simple wire-wound inductor, hundreds to low thousands for a good LC
circuit, tens of thousands for a [cavity filter](/reference/cavity-filter/), and
hundreds of thousands for a quartz [crystal](/reference/crystal-filter/) resonator.

## Relevance to SDR

Q governs how sharply RF hardware can carve the spectrum. Preselector and
[RF filter](/reference/rf-filter/) stages ahead of an SDR rely on resonators with
enough Q to pass the wanted band while rejecting strong out-of-band signals that
would otherwise overload the front end. In oscillators, a high-Q resonator (crystal,
cavity, or dielectric) narrows the [phase noise](/reference/phase-noise/) of the
[local oscillator](/reference/local-oscillator/), directly improving the purity of
the reference the SDR mixes against. High Q is thus a recurring goal wherever
frequency selectivity or spectral purity matters.

GopherTrunk performs its channel filtering in software on the IQ stream, where the
"selectivity" is set by digital filter design rather than a physical Q. The physical
Q of the analog preselector and of the SDR's reference oscillator still bounds what
reaches the ADC, so a clean, well-filtered front end lets GopherTrunk's digital
stages start from better samples. The concept translates: a narrow digital
channelizer is the software analogue of a high-Q resonator.

## Sources

[^wiki]: [Q factor](https://en.wikipedia.org/wiki/Q_factor) — Wikipedia, the energy-ratio and f0/Δf definitions and the loaded-versus-unloaded distinction.
