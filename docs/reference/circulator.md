---
slug: circulator
title: Circulator
entry_type: hardware
category: rf-front-end
description: "A circulator is a three-port ferrite device that routes signals one way around its ports, letting a transmitter and receiver share one antenna without feeding each other."
keywords: circulator, ferrite circulator, three-port, nonreciprocal, Faraday rotation, isolator, duplexer, TX RX sharing, reverse isolation
aka: [circulator, ferrite circulator]
autolink: true
infobox:
  - { label: Type, value: "Ferrite nonreciprocal device" }
  - { label: Ports, value: "Three (occasionally four)" }
  - { label: Function, value: "One-way routing between ports" }
  - { label: Key spec, value: "Isolation and insertion loss" }
see_also: [isolator, duplexer, diplexer, return-loss, rf-switch]
cite_urls:
  - https://en.wikipedia.org/wiki/Circulator
  - https://en.wikipedia.org/wiki/Ferrite_(magnet)
---

A **circulator** is a passive three-port device that routes a signal **one way** around its
ports: power entering port 1 leaves at port 2, power into port 2 leaves at port 3, and power
into port 3 leaves at port 1 — but not in reverse.[^wiki] This **nonreciprocal** behaviour lets
a transmitter and a receiver share a single [antenna](/reference/antenna/) while keeping the
transmit power out of the receiver.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A three-port circulator drawn as a circle with a curved arrow showing signals passing only from port 1 to port 2 to port 3 and back to port 1." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="cirar" markerWidth="8" markerHeight="8" refX="5" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <circle cx="230" cy="85" r="45" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <path d="M205 60 A38 38 0 0 1 262 74" fill="none" stroke="currentColor" stroke-width="1.4" marker-end="url(#cirar)"/>
  <g stroke="currentColor" stroke-width="1.3">
    <line x1="120" y1="85" x2="185" y2="85"/>
    <line x1="275" y1="60" x2="330" y2="30"/>
    <line x1="275" y1="110" x2="330" y2="140"/>
  </g>
  <g font-size="9" fill="currentColor">
    <text x="88" y="88">1 TX</text>
    <text x="336" y="30">2 antenna</text>
    <text x="336" y="144">3 RX</text>
  </g>
</svg>
<figcaption>Signals circulate one way: the transmitter on port 1 reaches the antenna on port 2, and antenna signals reach the receiver on port 3 — never straight from TX to RX.</figcaption>
</figure>

## Overview

A circulator's usefulness comes entirely from its **one-way** routing. In a shared-antenna
radio the transmitter connects to port 1, the antenna to port 2, and the receiver to port 3.
Transmit power flows 1→2 out the antenna; incoming signals flow 2→3 into the receiver; and the
huge transmit signal, which would swamp the receiver, is *not* routed 1→3. It is the
nonreciprocity — the fact that the path depends on direction — that makes this possible; you
cannot build the same behaviour from ordinary reciprocal components.

## How it works

The one-way action comes from a **ferrite** element held in a static magnetic field from a
permanent magnet. Microwave energy passing through a magnetised ferrite experiences a
direction-dependent phase shift (a gyromagnetic effect related to Faraday rotation). The ports
are arranged so that these phase shifts add constructively toward the "next" port and cancel
toward the "previous" one. The result is low **insertion loss** in the wanted direction
(a few tenths of a dB) and high **isolation** (often 20 dB or more) in the reverse direction.

Real circulators are band-limited and their isolation is finite, so the small leakage that
does reach the receiver still matters in high-power systems. The quality of the antenna match
also matters: energy reflected from a poorly matched antenna (see
[return loss](/reference/return-loss/)) circulates onward to the next port rather than back to
the source.

## Relevance to SDR

Circulators are common in radar, high-power base stations, and any system where TX and RX must
coexist on one antenna, and they are the building block of the closely related
[isolator](/reference/isolator/) — a circulator with one port terminated in a matched load to
soak up reflected power and protect an amplifier. They achieve the same TX/RX-sharing goal as a
[duplexer](/reference/duplexer/) but by direction rather than by frequency, so unlike a duplexer
they let transmit and receive occupy the *same* frequency.

For **GopherTrunk**, a circulator is transmit-side infrastructure well outside its scope: GT is
a receive-only decoder and does not transmit, so it never needs one. A circulator is relevant
only as part of the base-station and repeater hardware that generates the signals GT listens to.

## Sources

[^wiki]: [Circulator](https://en.wikipedia.org/wiki/Circulator) — Wikipedia, on ferrite circulators, nonreciprocal routing, isolation, and shared-antenna use.
