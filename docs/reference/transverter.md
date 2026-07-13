---
slug: transverter
title: Transverter
entry_type: hardware
category: rf-front-end
description: "A transverter shifts a whole band up or down by mixing against a fixed local oscillator, letting a radio work a frequency range its front end cannot reach directly."
keywords: transverter, band converter, up-conversion, down-conversion, IF, intermediate frequency, local oscillator, microwave transverter, HF converter, SDR band extension
aka: [transverter, "band converter"]
autolink: true
infobox:
  - { label: Type, value: "Bidirectional band converter" }
  - { label: Core, value: "Mixer + fixed LO + filters" }
  - { label: Direction, value: "Up and down conversion" }
  - { label: IF, value: "Radio's native tuning range" }
  - { label: TX, value: "Yes (transmit + receive)" }
see_also: [mixer-rf, upconverter, local-oscillator, intermediate-frequency, superheterodyne-receiver]
cite_urls:
  - https://en.wikipedia.org/wiki/Transverter
---

A **transverter** shifts an entire band of frequencies **up or down** by mixing it against a
fixed [local oscillator](/reference/local-oscillator/), so a radio can work a range its own
front end cannot reach directly.[^wiki] The name combines *transmit* and *converter*: unlike a
one-way [upconverter](/reference/upconverter/), a transverter handles both directions,
translating the radio's native tuning range (its **intermediate frequency**, or IF) up to the
target band on transmit and back down on receive. It is the classic way amateur stations reach
VHF, UHF, and microwave using a single capable HF/VHF transceiver as the tuning engine.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A transverter mixes a radio's IF band against a fixed local oscillator to translate it up to a target band, with a mirrored receive path translating the band back down to the IF." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="tvar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="15" y="60" width="70" height="40" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="50" y="78" font-size="8" fill="currentColor" text-anchor="middle">radio</text>
  <text x="50" y="90" font-size="8" fill="currentColor" text-anchor="middle">(IF band)</text>
  <line x1="85" y1="80" x2="120" y2="80" stroke="currentColor" stroke-width="1.6" marker-end="url(#tvar)"/>
  <circle cx="140" cy="80" r="20" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <path d="M126 66 L154 94 M126 94 L154 66" stroke="currentColor" stroke-width="1.2"/>
  <text x="140" y="120" font-size="8" fill="currentColor" text-anchor="middle">mixer</text>
  <line x1="140" y1="135" x2="140" y2="102" stroke="currentColor" stroke-width="1.6" marker-end="url(#tvar)"/>
  <rect x="105" y="135" width="70" height="22" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="140" y="150" font-size="8" fill="currentColor" text-anchor="middle">fixed LO</text>
  <line x1="160" y1="80" x2="200" y2="80" stroke="currentColor" stroke-width="1.6" marker-end="url(#tvar)"/>
  <rect x="200" y="60" width="60" height="40" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="230" y="83" font-size="8" fill="currentColor" text-anchor="middle">filter</text>
  <line x1="260" y1="80" x2="300" y2="80" stroke="currentColor" stroke-width="1.6" marker-end="url(#tvar)"/>
  <path d="M320 55 l14 22 l-28 0 z" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <line x1="320" y1="77" x2="320" y2="100" stroke="currentColor" stroke-width="1.6"/>
  <text x="320" y="118" font-size="8" fill="currentColor" text-anchor="middle">target band</text>
  <text x="390" y="84" font-size="8" fill="currentColor" text-anchor="middle">&#8593; TX up</text>
  <text x="390" y="98" font-size="8" fill="currentColor" text-anchor="middle">&#8595; RX down</text>
</svg>
<figcaption>A transverter mixes the radio's IF band against a fixed LO to reach a target band on transmit, and mirrors the path to bring the band back down on receive.</figcaption>
</figure>

## Overview

The idea is a translation, not a re-tune: the radio still does all the fine tuning,
modulation, and demodulation within its own comfortable IF range, while the transverter simply
slides that whole window to a new part of the spectrum. A 144 MHz transceiver behind a
"144-to-1296 MHz" transverter, for instance, tunes 1296 MHz by tuning 144 MHz — the fixed LO
supplies the constant 1152 MHz offset. Because the offset is fixed, the dial reads directly
once you add it in your head or in software.

## What it is

At its core a transverter is a [mixer](/reference/mixer-rf/), a stable fixed
[local oscillator](/reference/local-oscillator/), and band-defining filters, arranged so the
same block works both ways. On transmit it up-converts the IF to the target band and amplifies
it; on receive it down-converts the target band back to the IF, usually with a
low-noise-amplifier stage first because the transverter's own noise figure now sets the
system's sensitivity at the new band. Filtering matters: the mixer produces both sum and
difference products plus the LO leakage and image, and only the wanted product should reach
the antenna or the radio.

## Variants

- **Microwave transverters** (903 MHz, 1296 MHz, 2.3/3.4/5.7/10 GHz and up) extend a VHF/UHF
  rig into bands where building a full transceiver would be impractical.
- **Receive-only converters** are the one-way, receive-side special case — an
  [upconverter](/reference/upconverter/) that lifts HF into an SDR's tunable range is exactly
  this idea used for reception only.
- **No-transmit "downconverters"** bring a high band down for a receiver that cannot tune it.

The distinction from a plain converter is bidirectionality and a transmit path.

## Relevance to SDR

For an SDR listener, the receive half of the transverter concept is what matters: it extends a
receiver beyond its native tuning limits by translating a band into the range the SDR can
digitise. An HF [upconverter](/reference/upconverter/) ahead of a VHF-only dongle, or a
microwave down-converter feeding an SDR's IF, both apply the same fixed-LO mixing a full
transverter uses. GopherTrunk is decode software and a receiver-side tool — it never
transmits, so the transmit path of a transverter is out of scope — but when a band of interest
sits outside a dongle's reach, a converter/transverter in front of the radio is the hardware
that brings it in-range, and GT then decodes the translated signal exactly as if it had been
received natively.

## Sources

[^wiki]: [Transverter](https://en.wikipedia.org/wiki/Transverter) — Wikipedia, on bidirectional up/down conversion, the fixed-LO mixer architecture, IF concept, and amateur microwave use.
