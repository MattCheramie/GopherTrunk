---
slug: power-amplifier
title: Power amplifier (PA)
entry_type: hardware
category: rf-front-end
description: "A power amplifier raises a signal to the wattage needed to drive an antenna; its efficiency and linearity trade off through amplifier classes A through F and backoff."
keywords: power amplifier, PA, RF power amplifier, class A, class AB, class C, class D, class E, class F, backoff, PAPR, linearity, efficiency, transmitter
aka: [power amplifier, PA, RF PA]
autolink: true
infobox:
  - { label: Type, value: "Transmit RF amplifier" }
  - { label: Function, value: "Boost signal to antenna power level" }
  - { label: Key trade-off, value: "Efficiency vs linearity" }
  - { label: Classes, value: "A, AB, B, C, D, E, F" }
see_also: [crest-factor-papr, 1-db-compression-point, intermodulation, spectral-efficiency, low-noise-amplifier]
cite_urls:
  - https://en.wikipedia.org/wiki/Power_amplifier_classes
  - https://en.wikipedia.org/wiki/RF_power_amplifier
---

A **power amplifier** (**PA**) is the final gain stage of a transmitter, raising a modulated
signal to the wattage needed to drive an [antenna](/reference/antenna/) across the intended
range.[^classes] Unlike a receive-side amplifier, a PA is judged mainly by its **output power,
efficiency, and linearity** rather than its noise — and those last two pull in opposite
directions, which is why PAs come in a family of *classes*.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A transfer curve showing linear amplification at low drive and gain compression near saturation, with a backoff arrow marking the linear operating point." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="paar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <line x1="55" y1="120" x2="420" y2="120"/>
    <line x1="55" y1="120" x2="55" y2="20"/>
    <path d="M55 120 L250 45 Q320 25 400 22" stroke-width="1.6"/>
    <line x1="55" y1="120" x2="300" y2="26" stroke-dasharray="4 3"/>
  </g>
  <g font-size="8" fill="currentColor">
    <text x="200" y="138">input drive</text>
    <text x="20" y="70" transform="rotate(-90 20 70)">output power</text>
    <text x="305" y="40">compression</text>
    <text x="95" y="88">linear region</text>
    <line x1="250" y1="110" x2="250" y2="52" stroke="currentColor" stroke-width="1" marker-end="url(#paar)"/>
    <text x="255" y="105" font-size="8">backoff</text>
  </g>
</svg>
<figcaption>A PA is linear only well below saturation; a peaky signal must be backed off from the compression point to stay clean.</figcaption>
</figure>

## Overview

The core tension in PA design is that the most **efficient** operating point — driving the
transistor hard toward saturation — is also the most **nonlinear**. A signal pushed into
compression develops harmonics and [intermodulation](/reference/intermodulation/) products
that splatter energy into adjacent channels. How much distortion is tolerable depends on the
modulation: constant-envelope schemes can run a PA flat-out, while amplitude-varying schemes
must keep the PA in its linear region.

## How it works

Amplifier **classes** describe how much of each RF cycle the active device conducts, which
sets the efficiency/linearity balance:

- **Class A** — device conducts for the full cycle. Most linear, worst efficiency (theoretical
  max 50%, often ~20–30% in practice).
- **Class AB / B** — conducts for roughly half to somewhat more than half the cycle. A
  practical compromise widely used for linear RF; class B tops out near 78.5%.
- **Class C** — conducts for less than half the cycle. Efficient but nonlinear, suited only to
  constant-envelope signals (FM, CW).
- **Switching classes D, E, F** — the device acts as a switch rather than a linear element,
  reaching 80–90%+ efficiency, but they amplify constant-envelope or specially shaped
  waveforms rather than arbitrary linear signals.

## In practice — backoff and PAPR

Modern digital waveforms (OFDM, and the shaped single-carrier schemes in land-mobile radio)
have a high **peak-to-average power ratio**. The peaks, not the average, are what drive a PA
into compression, so the amplifier must be operated with its average power set well below
saturation — **backoff** equal to roughly the signal's
[crest factor / PAPR](/reference/crest-factor-papr/). More backoff means cleaner output but
lower efficiency and wasted DC power. The usable ceiling before distortion grows is anchored
by the PA's [1 dB compression point](/reference/1-db-compression-point/); designers also watch
third-order intermodulation, since that is what lands in neighbouring channels. Techniques
such as digital pre-distortion, envelope tracking, and Doherty amplification recover
efficiency while keeping linearity.

## Relevance to SDR

Power amplifiers live on the **transmit** side, so they matter for SDR *transceivers* — HackRF,
LimeSDR, PlutoSDR, USRP — and for the base stations and mobiles that GopherTrunk listens to.
The choice of PA class explains a signal's spectral shape at the receiver: a splattering,
over-driven transmitter widens occupied bandwidth and raises the noise a nearby receiver sees.

GopherTrunk is a **receive-only** decoder and contains no power amplifier or transmit chain.
PA behaviour is relevant to it only indirectly, as one physical cause of the adjacent-channel
interference and signal impairments its DSP must cope with.

## Sources

[^classes]: [Power amplifier classes](https://en.wikipedia.org/wiki/Power_amplifier_classes) — Wikipedia, on conduction-angle classes A–F and their efficiency/linearity trade-offs.
