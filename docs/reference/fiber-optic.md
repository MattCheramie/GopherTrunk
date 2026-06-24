---
slug: fiber-optic
title: Fiber-optic
entry_type: concept
category: hw-networking
description: Fiber-optic communication sends data as pulses of light through thin glass strands, offering very high bandwidth over long distances with low loss and immunity to electrical interference.
keywords: fiber-optic, optical fiber, fibre, single-mode, multimode, light, photonics, SFP
aka: [optical fiber, fibre]
infobox:
  - { label: Type, value: Optical transmission medium }
  - { label: Carries, value: Data as light pulses }
  - { label: Strengths, value: High bandwidth, low loss, long reach }
  - { label: Kinds, value: Single-mode, multimode }
see_also: [ethernet, modem, network-switch, coaxial-cable, lan-and-wan, electromagnetic-spectrum]
cite_urls:
  - https://en.wikipedia.org/wiki/Optical_fiber
---

**Fiber-optic** communication sends data as pulses of light through thin strands of glass, delivering very high bandwidth over long distances with low loss.[^wiki]

## Overview

A fiber carries a modulated beam of light by total internal reflection, so the signal travels for kilometres with little attenuation and — being optical — is immune to the electromagnetic interference that affects copper. *Single-mode* fiber uses a tiny core for long-haul, high-rate links; *multimode* fiber has a wider core for shorter runs. Fiber underpins long-distance internet backbones, data-center interconnects, and high-speed [Ethernet](/reference/ethernet/), where pluggable transceivers (SFP modules) terminate the light into a [switch](/reference/network-switch/). Compared with [coaxial cable](/reference/coaxial-cable/), it offers far more capacity and reach.

## Trade-offs

Fiber's bandwidth and noise immunity come at the cost of more delicate cabling and transceivers that cost more than copper ports. For most short LAN runs, copper [Ethernet](/reference/ethernet/) is simpler; fiber earns its place over distance or in electrically noisy environments. The same physics — modulating light instead of an RF carrier — is a cousin of the modulation an SDR like GopherTrunk performs on radio waves.

## Sources

[^wiki]: [Optical fiber](https://en.wikipedia.org/wiki/Optical_fiber) — Wikipedia, on fiber-optic transmission.
