---
slug: directional-coupler
title: Directional coupler
entry_type: hardware
category: rf-front-end
description: "A directional coupler taps a small, direction-selective sample of a signal on a transmission line, letting you measure forward and reflected power separately."
keywords: directional coupler, coupled port, coupling factor, directivity, forward power, reflected power, dual directional coupler, bidirectional coupler, VSWR bridge
aka: [directional coupler, coupler]
autolink: true
infobox:
  - { label: Type, value: "Passive four-port network" }
  - { label: Function, value: "Sample forward or reflected wave" }
  - { label: Key specs, value: "Coupling, directivity, insertion loss" }
  - { label: Coupling, value: "Typically 10, 20, 30 dB" }
see_also: [return-loss, standing-wave-ratio, splitter-combiner, rf-power-meter, reflection-coefficient]
cite_urls:
  - https://en.wikipedia.org/wiki/Power_dividers_and_directional_couplers
  - https://en.wikipedia.org/wiki/Standing_wave_ratio
---

A **directional coupler** is a passive four-port network that taps a small, **direction-selective**
sample of a signal travelling on a transmission line.[^wiki] Because it responds to waves going
one way but not the other, it lets you measure **forward** and **reflected** power separately —
the basis of power meters, VSWR bridges, and monitoring taps.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A main transmission line from input to output with a coupled section that samples a small fraction of the forward wave to a coupled port and directs the reverse wave to an isolated port." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="dcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" fill="none" stroke-width="1.4">
    <line x1="40" y1="50" x2="420" y2="50"/>
    <line x1="140" y1="90" x2="320" y2="90"/>
  </g>
  <g stroke="currentColor" stroke-width="1.2">
    <line x1="60" y1="50" x2="150" y2="50" marker-end="url(#dcar)"/>
    <line x1="240" y1="90" x2="150" y2="90" marker-end="url(#dcar)"/>
  </g>
  <g font-size="9" fill="currentColor">
    <text x="45" y="42">in →</text>
    <text x="380" y="42">→ out</text>
    <text x="70" y="105">coupled (fwd sample)</text>
    <text x="300" y="105" text-anchor="end">isolated</text>
    <text x="150" y="72" font-size="8">−20 dB tap</text>
  </g>
</svg>
<figcaption>A directional coupler taps a fixed fraction of the forward wave to a coupled port while sending reverse-travelling energy to the isolated port.</figcaption>
</figure>

## Overview

Three numbers describe a coupler. **Coupling factor** (e.g. 20 dB) is how much weaker the
sampled signal is than the main line — a 20 dB coupler taps one-hundredth of the power.
**Directivity** is how well it distinguishes forward from reverse; high directivity means the
coupled port reports the forward wave with little contamination from the reflected wave.
**Insertion loss** is the small amount the main-line signal drops by giving up that sample. The
main path is nearly transparent, so a coupler can be left permanently in line.

## How it works

A directional coupler places a second transmission line close enough to the main line that
electromagnetic coupling transfers a small, fixed fraction of the energy into it. The geometry
is arranged so the coupling is **phase-sensitive**: a forward-travelling wave adds up toward the
*coupled* port, while a reverse-travelling wave adds up toward the *isolated* port. Terminate
the isolated port and the coupled port then carries a clean sample of just the forward wave.
Swap which wave you sample — or use a **dual (bidirectional)** coupler with taps for both
directions — and you can read forward and reflected power at the same time.

From those two readings you get the [reflection coefficient](/reference/reflection-coefficient/),
and hence the load's [return loss](/reference/return-loss/) and
[standing-wave ratio](/reference/standing-wave-ratio/) — which is exactly how an in-line
SWR/power meter works.

## Variants

- **Single vs dual (bidirectional)** — one coupled port, or separate forward and reverse ports.
- **Coupled-line, branch-line, and Lange couplers** — different geometries and bandwidths.
- **Hybrid couplers** — the special 3 dB, 90°/180° cases used as
  [splitters/combiners](/reference/splitter-combiner/) and in balanced mixers.

## Relevance to SDR

Directional couplers appear wherever RF power must be monitored or a controlled sample tapped
off: transmitter SWR/power metering, antenna-system diagnostics, feedback for
[automatic gain control](/reference/automatic-gain-control/) and pre-distortion, and bench
measurements. On the receive side a low-coupling tap can inject a calibration tone or split off
a monitoring feed without appreciably loading the main path.

**GopherTrunk** is receive-only software and does not include or need a directional coupler.
The device is relevant to GT users chiefly as an antenna-system diagnostic tool — a coupler and
power meter reveal a bad match or feedline fault that would otherwise show up only as weak,
error-prone signals in the decoder.

## Sources

[^wiki]: [Power dividers and directional couplers](https://en.wikipedia.org/wiki/Power_dividers_and_directional_couplers) — Wikipedia, on coupling factor, directivity, and forward/reflected sampling.
