---
slug: isolator
title: Isolator
entry_type: hardware
category: rf-front-end
description: "An isolator is a two-port ferrite device that passes signals one way and absorbs reflections in the other, protecting a source such as a power amplifier from reflected power."
keywords: isolator, ferrite isolator, two-port, nonreciprocal, reverse isolation, reflected power protection, circulator, matched load, VSWR protection
aka: [isolator, ferrite isolator]
autolink: true
infobox:
  - { label: Type, value: "Ferrite nonreciprocal device" }
  - { label: Ports, value: "Two (a terminated three-port)" }
  - { label: Function, value: "Pass forward, absorb reflected" }
  - { label: Key spec, value: "Isolation and insertion loss" }
see_also: [circulator, duplexer, return-loss, standing-wave-ratio, power-amplifier]
cite_urls:
  - https://en.wikipedia.org/wiki/Isolator_(microwave)
  - https://en.wikipedia.org/wiki/Circulator
---

An **isolator** is a two-port passive device that lets a signal pass in **one direction** while
strongly **absorbing** anything travelling the other way.[^wiki] It is, in effect, a
[circulator](/reference/circulator/) with one of its three ports permanently terminated in a
matched load — so power reflected back toward the source is dumped into that load instead of
returning. Its usual job is to protect a source, most often a
[power amplifier](/reference/power-amplifier/), from reflected power.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="An isolator drawn as a three-port circulator with the third port terminated in a matched load, so forward power passes to the output while reflected power is absorbed in the load." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="isoar" markerWidth="8" markerHeight="8" refX="5" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <circle cx="230" cy="80" r="42" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <path d="M207 58 A34 34 0 0 1 260 70" fill="none" stroke="currentColor" stroke-width="1.3" marker-end="url(#isoar)"/>
  <g stroke="currentColor" stroke-width="1.3">
    <line x1="120" y1="80" x2="188" y2="80"/>
    <line x1="272" y1="80" x2="345" y2="80"/>
    <line x1="230" y1="122" x2="230" y2="140"/>
  </g>
  <path d="M218 140 L242 140 L230 155 Z" fill="currentColor"/>
  <g font-size="9" fill="currentColor">
    <text x="88" y="83">in</text>
    <text x="352" y="83">out</text>
    <text x="200" y="152" font-size="8">matched load</text>
    <text x="255" y="50" font-size="8">forward</text>
  </g>
</svg>
<figcaption>An isolator is a circulator with the third port terminated: forward power reaches the output while any reflected power is absorbed in the internal load.</figcaption>
</figure>

## Overview

An isolator solves a specific problem: what happens when a source drives a load that is not
perfectly matched. A poorly matched [antenna](/reference/antenna/) or a switching filter
reflects part of the incident power back toward the transmitter. That reflected wave can pull
an oscillator off frequency, distort an amplifier, or in the worst case damage the output
device. An isolator interposes a one-way path so the source always sees a good match and never
sees the reflection.

## How it works

Like a circulator, an isolator relies on a **ferrite** element in a static magnetic field,
giving a direction-dependent phase shift. Forward-travelling energy passes port 1 to port 2
with low **insertion loss**; reverse-travelling energy is routed toward the third (internal)
port, where a **matched load** absorbs it as heat. The device is rated by its **isolation**
(how much reverse power it blocks, often 20–30 dB) and its forward insertion loss (ideally a
few tenths of a dB). Because the reflected energy is dissipated rather than returned, the
source sees a clean, near-constant load regardless of what the antenna is doing.

The amount of reflected power an isolator has to absorb is set by the load match — described
equivalently by [return loss](/reference/return-loss/) or
[standing-wave ratio](/reference/standing-wave-ratio/). A worse match means more energy heading
back into the isolator's load, which is why power-rated isolators have substantial heat-sinking.

## Relevance to SDR

Isolators are standard on transmit chains: they guard base-station and repeater PAs against
antenna mismatch and are common in test setups to keep reflections from corrupting
measurements. They are the terminated cousin of the [circulator](/reference/circulator/) and
serve a different aim than a [duplexer](/reference/duplexer/), which separates TX and RX by
frequency rather than protecting a source by direction.

For **GopherTrunk** an isolator is transmit-side hardware and out of scope — GT is a
receive-only decoder that does not transmit. It is worth knowing about only as part of the
transmitter and repeater equipment whose signals GT ultimately decodes.

## Sources

[^wiki]: [Isolator (microwave)](https://en.wikipedia.org/wiki/Isolator_(microwave)) — Wikipedia, on ferrite isolators as terminated circulators that protect sources from reflected power.
