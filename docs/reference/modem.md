---
slug: modem
title: Modem
entry_type: hardware
category: hw-networking
description: A modem modulates and demodulates signals so digital data can travel over a medium not designed for it — a phone line, coax, fiber, or radio link — terminating an internet provider's line and handing clean Ethernet to a router.
keywords: modem, modulator demodulator, cable modem, DSL, fiber modem, ONT, cellular modem, broadband, carrier
aka: [modulator-demodulator]
infobox:
  - { label: Type, value: Networking device }
  - { label: Job, value: Modulate / demodulate data }
  - { label: Bridges, value: Carrier medium to digital network }
  - { label: Kinds, value: Cable, DSL, fiber, cellular }
  - { label: Hands off, value: Ethernet to a router }
see_also: [router, cellular-modem, gateway, fiber-optic, ethernet, network-interface-card]
cite_urls:
  - https://en.wikipedia.org/wiki/Modem
---

A **modem** (modulator–demodulator) is a device that encodes digital data onto a carrier signal for transmission over a medium not built for it, and decodes it again at the far end.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A modem's two-way signal flow: on the outbound path digital bits enter a modulator that impresses them onto a wavy carrier sent down the line; on the return path an incoming carrier enters a demodulator that recovers the original bits." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor">
    <g stroke-width="1.3">
      <rect x="150" y="24" width="90" height="34" rx="4" fill-opacity="0.14"/>
      <rect x="150" y="92" width="90" height="34" rx="4" fill-opacity="0.14"/>
    </g>
    <g stroke-width="1.4" fill="none">
      <line x1="20" y1="41" x2="150" y2="41"/>
      <path d="M240 41 q10 -8 20 0 t20 0 t20 0 t20 0 t20 0 t20 0" />
      <path d="M20 109 q10 -8 20 0 t20 0 t20 0 t20 0 t20 0 t20 0 t20 0" />
      <line x1="240" y1="109" x2="440" y2="109"/>
    </g>
    <g stroke-width="1.2">
      <path d="M143 41 l-8 -4 v8 Z" fill="currentColor"/>
      <path d="M433 41 l8 -4 v8 Z" fill="currentColor" transform="translate(0,0)"/>
      <path d="M247 109 l-8 -4 v8 Z" fill="currentColor"/>
    </g>
  </g>
  <g fill="currentColor" stroke="none" font-size="8.5">
    <text x="20" y="34">digital bits</text>
    <text x="195" y="45" text-anchor="middle" font-weight="600">modulate</text>
    <text x="360" y="34" text-anchor="middle">carrier out (to line)</text>
    <text x="100" y="102" text-anchor="middle">carrier in (from line)</text>
    <text x="195" y="113" text-anchor="middle" font-weight="600">demodulate</text>
    <text x="360" y="102" text-anchor="middle">digital bits</text>
    <text x="230" y="142" text-anchor="middle" fill-opacity="0.9">same box, two directions: bits ⇄ a signal the medium can carry</text>
  </g>
</svg>
<figcaption>A modem works both ways at once: a modulator impresses outgoing bits onto a carrier the line can carry, while a demodulator recovers incoming bits from the carrier arriving from the far end — the two halves that give the device its name.</figcaption>
</figure>

## Overview

The name describes the job: *modulate* outgoing data onto a carrier — a tone on a phone line, an RF channel on a coax cable, light on a [fiber](/reference/fiber-optic/) strand — and *demodulate* the incoming signal back into bits. Because each medium has its own physics, the modulation scheme differs, but the role is always the same: turn a stream of bits into something the line can carry, and back again.

A modem is what terminates the link from an internet provider and hands clean [Ethernet](/reference/ethernet/) to a [router](/reference/router/). It speaks the provider's line protocol on one side and standard local networking on the other, so everything downstream can ignore how the last mile actually works.

Variants include cable, DSL, fiber (an ONT, or optical network terminal), and [cellular modems](/reference/cellular-modem/) that ride a mobile network. In consumer gear the modem and router are frequently combined into one box, which blurs the boundary but not the two distinct jobs inside.

## Kinds of modem

Each type is matched to the medium the provider runs to the premises:

| Kind | Carrier medium | Modulates onto |
|------|----------------|----------------|
| DSL | Telephone twisted pair | High-frequency tones |
| Cable | Coaxial cable (DOCSIS) | RF channels |
| Fiber (ONT) | Optical fiber | Pulses of light |
| Cellular | Radio (LTE / 5G) | RF carrier over the air |

## Where it fits

The modem is the boundary between a private network and the provider's carrier medium; downstream of it the [router](/reference/router/) acts as the [gateway](/reference/gateway/) to the local [LAN](/reference/lan-and-wan/). The modulation/demodulation idea is the very same one at the heart of an SDR — a GopherTrunk decoder demodulates an RF carrier into symbols exactly as a modem recovers bits from a line signal, which is why the two fields share so much vocabulary.

## Sources

[^wiki]: [Modem](https://en.wikipedia.org/wiki/Modem) — Wikipedia, on the modulator-demodulator and its cable, DSL, fiber, and cellular variants.
