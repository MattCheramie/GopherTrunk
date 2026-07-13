---
slug: rf-switch
title: RF switch
entry_type: hardware
category: rf-front-end
description: "An RF switch routes radio-frequency signals between ports using PIN diodes, FETs, or MEMS, enabling antenna selection, band switching, and TX/RX changeover."
keywords: RF switch, PIN diode switch, MEMS switch, FET switch, antenna switch, TR switch, SPDT, SP4T, band switching, transmit receive changeover, isolation, insertion loss
aka: [RF switch, "microwave switch", "antenna switch", "TR switch"]
autolink: true
infobox:
  - { label: Type, value: "RF signal-routing element" }
  - { label: Technology, value: "PIN diode, FET/SOI, MEMS" }
  - { label: Key spec, value: "Insertion loss, isolation, switching speed" }
  - { label: TX, value: "Yes (power-rated types)" }
  - { label: Typical price, value: "$1–$40 (part or module)" }
see_also: [antenna-diversity, attenuator, low-noise-amplifier, mixer-rf, rf-filter]
cite_urls:
  - https://en.wikipedia.org/wiki/RF_switch
  - https://en.wikipedia.org/wiki/PIN_diode
---

An **RF switch** is a component that routes a radio-frequency signal from one port
to another under electronic (or, in coax relays, mechanical) control.[^wiki] Unlike
a DC switch, it must preserve impedance match and pass signals cleanly to gigahertz
frequencies, so it is judged by **insertion loss** in the on state, **isolation**
in the off state, **switching speed**, **power handling**, and **linearity**. RF
switches make antenna selection, band switching, and transmit/receive changeover
possible without physically re-cabling.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A single-pole double-throw RF switch connecting one common port to one of two selectable antenna ports under a control line." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="rfswar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <circle cx="60" cy="70" r="4" fill="currentColor"/>
  <text x="60" y="95" text-anchor="middle" font-size="9" fill="currentColor">common</text>
  <line x1="64" y1="70" x2="150" y2="70" stroke="currentColor" stroke-width="1.6"/>
  <line x1="150" y1="70" x2="230" y2="40" stroke="currentColor" stroke-width="1.8"/>
  <line x1="150" y1="70" x2="230" y2="100" stroke="currentColor" stroke-dasharray="3 3"/>
  <circle cx="240" cy="38" r="4" fill="currentColor"/>
  <circle cx="240" cy="102" r="4" fill="none" stroke="currentColor"/>
  <text x="300" y="42" font-size="9" fill="currentColor">port 1 (on)</text>
  <text x="300" y="106" font-size="9" fill="currentColor">port 2 (off)</text>
  <line x1="150" y1="120" x2="150" y2="132" stroke="currentColor" stroke-width="1.4" marker-end="url(#rfswar)"/>
  <text x="150" y="132" text-anchor="start" font-size="8" fill="currentColor"> control</text>
</svg>
<figcaption>A single-pole double-throw (SPDT) RF switch selects one of two ports; isolation keeps the off port quiet.</figcaption>
</figure>

## Overview

Switch families are named by pole and throw count: SPST (a simple on/off), SPDT
(one common, two selectable), and SP*n*T (multi-throw, e.g. SP4T for a four-antenna
selector). The two headline specs pull against each other. **Insertion loss** is
the signal lost in the connected path — a fraction of a dB is good, and it adds
directly to the system [noise figure](/reference/noise-figure/) when the switch
sits ahead of the [low-noise amplifier](/reference/low-noise-amplifier/).
**Isolation** is how well the off ports are silenced; poor isolation lets an
unselected antenna or a transmitter leak into the receive path.

## Variants

- **PIN-diode switches** — a forward-biased PIN diode looks like a low resistance
  to RF and a reverse-biased one like a small capacitance. They handle high power,
  are cheap, and switch in microseconds; the trade is a bias current and finite
  linearity that can generate [intermodulation](/reference/intermodulation/).
- **FET / SOI switches** — silicon-on-insulator MOSFET switches integrate many
  throws on one die with fast, low-current CMOS control; ubiquitous in phone and
  SDR front ends.
- **MEMS switches** — micro-machined mechanical contacts give near-zero loss and
  excellent isolation and linearity, at the cost of slower switching and higher
  price.
- **Electromechanical coaxial relays** — physical relays for the lowest loss and
  highest isolation at HF/VHF, used in test benches and high-power TR switching.

## Relevance to SDR

RF switches are everywhere in radio hardware. A transmit/receive (TR) switch lets a
transceiver share one antenna between the power amplifier and the receiver, and its
isolation and switching speed matter for time-division systems like
[TDMA](/reference/tdma/) [DMR](/reference/dmr/) and
[P25 Phase 2](/reference/p25-phase-2/). Multi-throw switches implement band
selection in wideband tuners and let a single receiver scan several antennas.

The most directly relevant use for a listener is
[antenna diversity](/reference/antenna-diversity/): switching (or combining)
between two antennas to combat [multipath](/reference/multipath-propagation/)
fading, which is especially valuable against [simulcast](/reference/simulcast/)
distortion on trunked systems. Some multi-channel SDRs and diversity front ends use
fast RF switches to sample several antennas.

GopherTrunk is software and controls no RF switch itself — antenna selection and
TR changeover live in the analog hardware and the SDR device. What matters to
GopherTrunk is the consequence: a low-loss, well-isolated switch preserves the
[dynamic range](/reference/dynamic-range/) and sensitivity the decoder depends on,
while a lossy or leaky switch raises the noise floor and can inject spurs that
degrade a control-channel lock.

## Sources

[^wiki]: [RF switch](https://en.wikipedia.org/wiki/RF_switch) — Wikipedia, on RF signal-routing components and the PIN-diode, FET, and MEMS technologies used to build them.
