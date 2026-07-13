---
slug: zero-if
title: Zero-IF (zero intermediate frequency)
entry_type: term
category: sdr-dsp
description: "Zero-IF places a receiver's intermediate frequency at 0 Hz, so the wanted signal sits on baseband as a complex I/Q pair — the standard SDR front end."
keywords: zero-IF, zero intermediate frequency, ZIF, direct conversion, homodyne, baseband receiver, DC offset, IQ front end
aka: [zero intermediate frequency, ZIF]
autolink: true
infobox:
  - { label: Type, value: Receiver front-end scheme }
  - { label: IF, value: 0 Hz }
  - { label: Weakness, value: DC offset, IQ imbalance }
see_also: [direct-conversion-receiver, low-if, dc-offset, iq-imbalance, baseband, superheterodyne-receiver]
cite_urls:
  - https://en.wikipedia.org/wiki/Direct-conversion_receiver
  - https://en.wikipedia.org/wiki/Intermediate_frequency
---

**Zero-IF** (**zero intermediate frequency**, or **ZIF**) is the receiver arrangement in
which the [intermediate frequency](/reference/intermediate-frequency/) is placed at
**0 Hz**: the wanted signal is mixed down until its centre sits exactly on
[baseband](/reference/baseband/).[^wiki] It is the defining property of the
[direct-conversion receiver](/reference/direct-conversion-receiver/) — "zero-IF" names
the frequency plan, "direct conversion" names the one-step architecture that achieves it
— and it is the format nearly all software-defined radios present to the host.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A spectrum showing a signal channel centred at zero hertz after down-conversion, with the local oscillator arrow landing on the carrier and the signal occupying frequencies both below and above zero." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="110" x2="440" y2="110" stroke="currentColor"/>
  <line x1="235" y1="40" x2="235" y2="118" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.6"/>
  <text x="222" y="132" font-size="9" fill="currentColor">0 Hz</text>
  <text x="60" y="132" font-size="8" fill="currentColor">−B/2</text>
  <text x="392" y="132" font-size="8" fill="currentColor">+B/2</text>
  <path d="M150 110 Q235 40 320 110 Z" fill="currentColor" fill-opacity="0.18" stroke="currentColor"/>
  <text x="150" y="30" font-size="9" fill="currentColor">wanted channel centred on baseband</text>
  <circle cx="235" cy="110" r="4" fill="currentColor"/>
  <text x="248" y="60" font-size="8" fill="currentColor">LO = carrier</text>
</svg>
<figcaption>With the local oscillator on the carrier, the channel straddles 0 Hz; the I/Q pair keeps the halves below and above zero distinct.</figcaption>
</figure>

## How it works

To land a signal on 0 Hz, the receiver tunes its [local oscillator](/reference/local-oscillator/)
to the signal's carrier frequency and mixes the two together. The difference product
comes out centred on zero. Because a signal has content both just below and just above
its carrier, and a single real mixer would fold those two halves on top of each other,
zero-IF receivers always work in quadrature: an in-phase (I) and a quadrature (Q) channel
together form a complex signal that keeps negative and positive baseband frequencies
separate. This complex pair is what an [ADC](/reference/analog-to-digital-converter/)
digitises, which is why zero-IF and [IQ data](/reference/iq-data/) go hand in hand.

## In practice

Putting the signal on DC is efficient but exposes it to two impairments that live at or
near 0 Hz:

- **[DC offset](/reference/dc-offset/).** Local-oscillator leakage self-mixes to a
  constant term at 0 Hz, sitting squarely on the middle of the channel. This is the fixed
  centre spike seen in almost every SDR [waterfall](/reference/waterfall-display/).
- **[IQ imbalance](/reference/iq-imbalance/).** Gain or phase mismatch between the I and Q
  paths lets a mirror image of each signal leak in at the negative of its frequency.

The [low-IF](/reference/low-if/) plan sidesteps both by nudging the signal a little away
from 0 Hz, at the cost of needing a slightly wider ADC bandwidth and a digital
down-conversion step to recentre it.

## Relevance to SDR

Zero-IF is the native output format of the tuners in
[RTL-SDR](/reference/rtl-sdr/) dongles, [Airspy](/reference/airspy/),
[HackRF](/reference/hackrf/), and most other consumer SDRs: they hand the host a complex
baseband stream already centred on the tuned frequency. Recognising the zero-IF signature
— a persistent DC spike and faint mirror images — is part of reading any SDR display
correctly.

GopherTrunk operates on this baseband IQ. Its digital
[down-converter](/reference/digital-down-converter/) re-centres each channel of interest
and its filters reject the rest, so the receiver's zero-IF format is simply the starting
point of GopherTrunk's software chain. Where a DC offset would otherwise sit on a wanted
carrier, the down-conversion and filtering move the signal off 0 Hz before demodulation.

## Sources

[^wiki]: [Direct-conversion receiver](https://en.wikipedia.org/wiki/Direct-conversion_receiver) — Wikipedia, on the zero intermediate frequency plan and its DC/imbalance trade-offs.
