---
slug: preamplifier
title: Preamplifier (preamp)
entry_type: hardware
category: rf-front-end
description: "A preamplifier is a low-noise gain stage placed early in a receive chain to lift weak signals above the noise of later stages, improving overall sensitivity."
keywords: preamplifier, preamp, RF preamp, receive amplifier, gain stage, noise figure, sensitivity, mast-mounted amplifier, front end
aka: [preamplifier, preamp, RF preamp]
autolink: true
infobox:
  - { label: Type, value: "Receive-side gain stage" }
  - { label: Function, value: "Amplify weak signals early in chain" }
  - { label: Key spec, value: "Noise figure and gain" }
  - { label: Placed, value: "Near the antenna" }
see_also: [low-noise-amplifier, noise-figure, receiver-sensitivity, bias-tee, superheterodyne-receiver]
cite_urls:
  - https://en.wikipedia.org/wiki/Preamplifier
  - https://en.wikipedia.org/wiki/Low-noise_amplifier
---

A **preamplifier** (**preamp**) is a gain stage placed **early** in a receive chain — ideally
right at the [antenna](/reference/antenna/) — to lift weak signals above the noise added by
everything downstream, improving the receiver's overall sensitivity.[^wiki] In RF practice the
term is used almost interchangeably with **[low-noise amplifier](/reference/low-noise-amplifier/)**:
a preamp *is* an LNA, described by the job it does in the signal chain rather than by a different
circuit.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="An antenna feeds a preamplifier placed early in the chain, whose low noise figure and gain dominate the receiver's noise budget before lossy cable and later stages." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="pream" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="40" y="58">antenna</text>
    <path d="M110 40 L110 72 L152 56 Z" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="128" y="88" font-size="8">preamp</text>
    <rect x="330" y="41" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="370" y="60">receiver</text>
    <line x1="66" y1="56" x2="108" y2="56" stroke="currentColor" stroke-width="1.1"/>
    <line x1="152" y1="56" x2="328" y2="56" stroke="currentColor" stroke-width="1.1" marker-end="url(#pream)"/>
    <text x="240" y="48" font-size="8">lossy coax run</text>
  </g>
</svg>
<figcaption>A preamp early in the chain sets the noise budget, so its gain overcomes the loss of cable and later stages.</figcaption>
</figure>

## Overview

The value of a preamp comes straight from the cascade noise formula (Friis): the **first**
amplifier's [noise figure](/reference/noise-figure/) dominates the whole receiver's noise,
and its gain divides down the noise contribution of every stage after it. Put a low-noise
gain block first and the rest of the chain — mixers, filters, lossy coax, the SDR itself —
barely moves the noise budget. Put it last and it does almost nothing.

## How it works

Two numbers define a preamp: its **gain** (how much it boosts the signal) and its **noise
figure** (how much noise it adds relative to a perfect amplifier). Because it sits first, you
want the lowest noise figure available and just enough gain to make later stages negligible —
typically 10–20 dB. Excess gain is counter-productive: it eats into the receiver's headroom
and can push strong signals into overload, generating
[intermodulation](/reference/intermodulation/). A good preamp is therefore a balance of low
noise, adequate gain, and enough linearity to survive the strongest signals present.

Mounting matters as much as the device. A **mast-mounted** preamp placed before the coax
recovers cable loss that would otherwise degrade sensitivity, and it is commonly powered up
the same coax through a [bias tee](/reference/bias-tee/) so no separate power cable is needed.

## Relevance to SDR

Preamps are a staple of SDR reception. Many popular receivers — RTL-SDR dongles, Airspy,
SDRplay — have modest front-end noise figures, so an external low-noise preamp near the
antenna can noticeably improve weak-signal reception, especially at VHF/UHF with long feed
lines. The caveat is overload: an SDR in a strong-signal environment (near broadcast FM or
paging transmitters) can be made *worse* by a preamp that amplifies everything into the
tuner's compression region. Filtering before the preamp, or using it only where the band is
genuinely quiet, is the usual remedy.

For **GopherTrunk**, a preamp is upstream hardware, not part of the software. It affects the
quality of the I/Q stream GT receives — a better front-end noise figure means cleaner symbols
and fewer errors for the decoders — but GopherTrunk neither controls nor requires one; it
works with whatever signal the SDR delivers.

## Sources

[^wiki]: [Preamplifier](https://en.wikipedia.org/wiki/Preamplifier) — Wikipedia, on preamplifiers as early gain stages and their role in setting system noise performance.
