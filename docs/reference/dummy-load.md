---
slug: dummy-load
title: Dummy load
entry_type: hardware
category: rf-front-end
description: "A dummy load is a non-radiating 50-ohm resistive termination that absorbs RF power for testing a transmitter or terminating a port without emitting a signal."
keywords: dummy load, RF load, 50 ohm termination, resistive termination, non-radiating load, transmitter test load, power resistor, matched termination, cantenna
aka: [dummy load, "RF dummy load", "50-ohm termination", "matched load"]
autolink: true
infobox:
  - { label: Type, value: "Non-radiating resistive termination" }
  - { label: Impedance, value: "50 Ω (sometimes 75 Ω)" }
  - { label: Key spec, value: "Power rating, VSWR, duty cycle" }
  - { label: TX, value: "Yes (absorbs transmit power)" }
  - { label: Typical price, value: "$5–$300 by power rating" }
see_also: [impedance, standing-wave-ratio, return-loss, attenuator, rf-power-meter]
cite_urls:
  - https://en.wikipedia.org/wiki/Dummy_load
  - https://en.wikipedia.org/wiki/Characteristic_impedance
---

A **dummy load** is a resistive termination that presents a radio's nominal
[impedance](/reference/impedance/) — almost always 50 Ω — while absorbing the RF
power delivered to it and radiating essentially none of it.[^wiki] It lets a
transmitter be tuned, measured, and tested without putting a signal on the air, and
more generally terminates any RF port cleanly so that energy is absorbed rather
than reflected back down the line.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A transmitter feeds a coax line into a 50-ohm resistive dummy load, where the RF power is dissipated as heat instead of being radiated." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="dlar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="30" y="45" width="70" height="40" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <text x="65" y="70" text-anchor="middle" font-size="9" fill="currentColor">TX</text>
  <line x1="100" y1="65" x2="250" y2="65" stroke="currentColor" stroke-width="1.8" marker-end="url(#dlar)"/>
  <text x="175" y="55" text-anchor="middle" font-size="8" fill="currentColor">RF power</text>
  <rect x="260" y="45" width="55" height="40" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.6"/>
  <text x="287" y="70" text-anchor="middle" font-size="10" fill="currentColor">50Ω</text>
  <path d="M330 50 q6 -10 12 0 q6 10 12 0" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <path d="M330 65 q6 -10 12 0 q6 10 12 0" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="360" y="82" font-size="8" fill="currentColor">heat</text>
</svg>
<figcaption>A dummy load absorbs transmit power in a matched 50-ohm resistance, dissipating it as heat rather than radiating it.</figcaption>
</figure>

## Overview

A good dummy load is judged by three numbers. Its **impedance** must stay close to
50 Ω across the frequency range so its [SWR](/reference/standing-wave-ratio/) is low
(ideally under 1.2:1) and its [return loss](/reference/return-loss/) high — a poor
match reflects power back and defeats the purpose. Its **power rating** must exceed
the transmitter output, and that rating is tied to a **duty cycle** and time limit:
a load rated "1 kW for 10 minutes" relies on its thermal mass and cannot dissipate
that continuously. The resistive element must also be non-inductive so it looks
resistive at RF, not just at DC.

## Variants

- **Air-cooled resistor loads** — one or more non-inductive power resistors on a
  heatsink; the common bench load for QRP through moderate power.
- **Oil-cooled loads** — a resistor immersed in mineral oil (the classic
  "cantenna") whose oil bath extends the power and duty-cycle rating.
- **Attenuating loads** — a load combined with an [attenuator](/reference/attenuator/)
  tap so a small, calibrated sample can be measured on test gear while the bulk of
  the power is absorbed.
- **Termination "caps"** — small connectorized 50-Ω terminations (a watt or less)
  for closing off unused ports on splitters, filters, and switches.

## Relevance to SDR

The dummy load is a transmit-side tool, so it matters most where an SDR platform
also transmits — full-duplex boards such as HackRF, LimeSDR, PlutoSDR, USRP, and
bladeRF are routinely bench-tested into a load before an antenna is ever connected,
both to protect the hardware and to avoid radiating spurious signals during
development. Terminating a transmit port into a load while probing it with a
[power meter](/reference/rf-power-meter/) or spectrum analyzer is standard practice.

On the receive side, small 50-Ω terminations serve a quieter purpose: capping the
unused outputs of a [splitter](/reference/splitter-combiner/), the isolated port of
a [circulator](/reference/circulator/), or a spare antenna input so that reflections
and stray pickup do not degrade the wanted path. A terminated input is also the
canonical way to measure a receiver's own [noise floor](/reference/noise-floor/),
since it presents a matched, signal-free source.

GopherTrunk is a receive-only software decoder; it neither transmits nor contains
any hardware, so it never drives a dummy load itself. The relevance is contextual —
a dummy load belongs to the test bench and TX hardware around an SDR, and it is a
useful reference point for the matched-termination and SWR concepts that govern how
cleanly signal power moves through the analog chain feeding GopherTrunk.

## Sources

[^wiki]: [Dummy load](https://en.wikipedia.org/wiki/Dummy_load) — Wikipedia, on non-radiating resistive terminations used to absorb RF power for testing.
