---
slug: modem
title: Modem
entry_type: hardware
category: hw-networking
description: A modem is a device that modulates and demodulates signals so digital data can travel over a medium not designed for it, such as a phone line, cable, or radio link.
keywords: modem, modulator demodulator, cable modem, DSL, fiber modem, broadband
aka: [modulator-demodulator]
infobox:
  - { label: Type, value: Networking device }
  - { label: Job, value: Modulate / demodulate data }
  - { label: Bridges, value: Carrier medium to digital network }
  - { label: Kinds, value: Cable, DSL, fiber, cellular }
see_also: [router, cellular-modem, gateway, fiber-optic, ethernet, network-interface-card]
cite_urls:
  - https://en.wikipedia.org/wiki/Modem
---

A **modem** (modulator–demodulator) is a device that encodes digital data onto a carrier signal for transmission over a medium not built for it, and decodes it again at the far end.[^wiki]

## Overview

The name describes the job: *modulate* outgoing data onto a carrier — a tone on a phone line, an RF channel on a coax cable, light on a [fiber](/reference/fiber-optic/) strand — and *demodulate* the incoming signal back into bits. A modem is what terminates the link from an internet provider and hands clean [Ethernet](/reference/ethernet/) to a [router](/reference/router/). Variants include cable, DSL, fiber (ONT), and [cellular modems](/reference/cellular-modem/) that ride a mobile network. In consumer gear the modem and router are often combined in one box.

## Where it fits

The modem is the boundary between a private network and the provider's carrier medium; downstream of it the [router](/reference/router/) acts as the [gateway](/reference/gateway/) to the local [LAN](/reference/lan-and-wan/). The modulation/demodulation idea is the same one at the heart of an SDR — a GopherTrunk decoder demodulates an RF carrier into symbols, just as a modem recovers bits from a line signal.

## Sources

[^wiki]: [Modem](https://en.wikipedia.org/wiki/Modem) — Wikipedia, on the modulator-demodulator and its variants.
