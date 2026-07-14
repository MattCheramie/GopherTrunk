---
slug: transmit-chain
title: Transmit chain
entry_type: concept
category: sdr-app-building
description: A transmit chain is the ordered TX pipeline — encode, modulate, pulse-shape, upconvert — that turns data into a bandlimited RF waveform, the mirror image of the receiver chain.
keywords: transmit chain, TX chain, transmitter pipeline, encode modulate pulse-shape upconvert, DAC, RF transmit, pulse shaping, HackRF transmit, upconversion
aka: [TX chain, transmit pipeline, transmitter chain]
autolink: true
infobox:
  - { label: Type, value: TX software / RF pipeline }
  - { label: Stages, value: "Encode → modulate → pulse-shape → upconvert" }
  - { label: GopherTrunk, value: Not applicable (RX-only) }
see_also: [modulation, pulse-shaping, hackrf, power-amplifier, receiver-chain, digital-to-analog-converter]
cite_urls:
  - https://en.wikipedia.org/wiki/Transmitter
  - https://en.wikipedia.org/wiki/Pulse_shaping
---

A **transmit chain** is the ordered pipeline that turns data into a bandlimited RF waveform:
**encode → modulate → pulse-shape → upconvert**.[^tx] It is the mirror image of the
[receiver chain](/reference/receiver-chain/) — every stage undoes, at the sending end, what a
corresponding receiver stage will later recover. In a software-defined transmitter these steps
run in code up to a [digital-to-analog converter](/reference/digital-to-analog-converter/),
after which analog stages upconvert and amplify.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 95" role="img" aria-label="Data enters four transmit blocks in sequence — encode, modulate, pulse-shape, upconvert — and leaves as an RF signal to an antenna." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="txchar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="10" y="50" font-size="7" fill="currentColor">data</text>
  <g font-size="7" fill="currentColor" text-anchor="middle">
    <rect x="36" y="36" width="58" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="65" y="51">encode</text>
    <rect x="110" y="36" width="62" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="141" y="51">modulate</text>
    <rect x="188" y="36" width="76" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="226" y="51">pulse-shape</text>
    <rect x="280" y="36" width="72" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="316" y="51">upconvert</text>
  </g>
  <g stroke="currentColor" stroke-width="1">
    <line x1="94" y1="48" x2="109" y2="48" marker-end="url(#txchar)"/>
    <line x1="172" y1="48" x2="187" y2="48" marker-end="url(#txchar)"/>
    <line x1="264" y1="48" x2="279" y2="48" marker-end="url(#txchar)"/>
    <line x1="352" y1="48" x2="410" y2="48" marker-end="url(#txchar)"/>
  </g>
  <path d="M420 48 L420 22 M412 22 L428 22" stroke="currentColor" stroke-width="1.4" fill="none"/>
</svg>
<figcaption>The transmit chain builds a bandlimited RF waveform from data — the inverse of the receiver chain.</figcaption>
</figure>

## How it works

- **Encode.** Source and channel coding turn the message into a bitstream: framing adds
  preamble, sync and headers, and forward-error-correction adds redundancy the receiver uses
  to correct errors. Bits are mapped to symbols (a constellation point per group of bits).
- **Modulate.** [Modulation](/reference/modulation/) impresses the symbols onto a carrier by
  varying its amplitude, frequency, or phase — the same axes a demodulator later reads.
- **Pulse-shape.** Each symbol is convolved with a [pulse-shaping](/reference/pulse-shaping/)
  filter — typically a [root-raised-cosine](/reference/root-raised-cosine-filter/) — so the
  transmitted spectrum stays inside its channel and the receiver, using the matched half of the
  filter, sees zero intersymbol interference at the symbol instants.
- **Upconvert.** A mixer translates the shaped baseband signal up to the RF carrier, and a
  [power amplifier](/reference/power-amplifier/) raises it to the level the antenna radiates.

## In practice

Because pulse-shaping and modulation set the occupied bandwidth and spectral cleanliness,
regulators and standards specify them tightly: a transmitter that shapes poorly splatters into
adjacent channels. The [power amplifier](/reference/power-amplifier/) is the other critical
stage — driving it into compression regrows the spectrum the pulse-shaping filter carefully
constrained, so real transmit chains trade linearity against efficiency and back the amplifier
off from saturation.

## Relevance to SDR

Full-duplex SDR platforms such as the [HackRF](/reference/hackrf/),
[LimeSDR](/reference/limesdr/), and [USRP](/reference/usrp-ettus/) expose a DAC and TX mixer,
letting the entire transmit chain — encode, modulate, pulse-shape — run in software before the
signal ever reaches analog hardware. This symmetry with the receive path is why SDR
frameworks describe flowgraphs the same way in both directions.

**GopherTrunk has no transmit chain.** It is a receive-only scanner/decoder: it demodulates
and decodes P25, DMR, NXDN, TETRA and other traffic but never generates RF, and its hardware
support targets receive-capable front ends. The transmit chain is included here for context —
understanding how a signal was *built* explains why the [receiver chain](/reference/receiver-chain/)
is structured the way it is, since each RX stage inverts a TX stage. GopherTrunk does
synthesise reference waveforms internally for testing its decoders, but that is offline signal
generation, not radio transmission.

## Sources

[^tx]: [Transmitter](https://en.wikipedia.org/wiki/Transmitter) — Wikipedia, on the stages that build and radiate an RF signal.
[^ps]: [Pulse shaping](https://en.wikipedia.org/wiki/Pulse_shaping) — Wikipedia, on bandlimiting symbols to control occupied bandwidth and intersymbol interference.
