---
slug: fiber-optic
title: Fiber-optic
entry_type: concept
category: hw-networking
description: Fiber-optic communication sends data as pulses of light guided through thin glass strands by total internal reflection, offering very high bandwidth over long distances with low loss and immunity to electrical interference.
keywords: fiber-optic, optical fiber, fibre, single-mode, multimode, total internal reflection, light, photonics, SFP, core, cladding
aka: [optical fiber, fibre]
infobox:
  - { label: Type, value: Optical transmission medium }
  - { label: Carries, value: Data as light pulses }
  - { label: Guided by, value: Total internal reflection }
  - { label: Strengths, value: High bandwidth, low loss, long reach }
  - { label: Kinds, value: Single-mode, multimode }
see_also: [ethernet, modem, network-switch, coaxial-cable, lan-and-wan, electromagnetic-spectrum]
cite_urls:
  - https://en.wikipedia.org/wiki/Optical_fiber
---

**Fiber-optic** communication sends data as pulses of light through thin strands of glass, delivering very high bandwidth over long distances with low loss.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A cross-section of an optical fiber: a light ray enters the central glass core from a laser and zig-zags down its length, bouncing off the boundary with the surrounding cladding by total internal reflection, reaching a detector at the far end." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor">
    <rect x="60" y="40" width="340" height="56" fill-opacity="0.06" stroke-width="1"/>
    <rect x="60" y="56" width="340" height="24" fill-opacity="0.14" stroke-width="1.2"/>
    <g stroke-width="1.4" fill="none">
      <path d="M60 68 L110 58 L160 78 L210 58 L260 78 L310 58 L360 78 L400 68"/>
    </g>
    <circle cx="52" cy="68" r="5" fill-opacity="0.2" stroke-width="1.2"/>
    <rect x="402" y="60" width="14" height="16" rx="2" fill-opacity="0.2" stroke-width="1.2"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8">
    <text x="52" y="90" text-anchor="middle">laser</text>
    <text x="409" y="90" text-anchor="middle">det.</text>
    <text x="230" y="52" text-anchor="middle" font-weight="600" font-size="8.5">core (high index)</text>
    <text x="230" y="108" text-anchor="middle">cladding (lower index) — reflects the ray back in</text>
    <text x="230" y="124" text-anchor="middle" fill-opacity="0.9" font-size="8">total internal reflection guides the light for kilometres with little loss</text>
  </g>
</svg>
<figcaption>Light launched into the fiber's dense glass core strikes the boundary with the lower-index cladding at a shallow angle and reflects entirely back inward — total internal reflection — so the pulse zig-zags down the strand for kilometres before a detector recovers it.</figcaption>
</figure>

## Overview

A fiber carries a modulated beam of light by *total internal reflection*: a dense glass *core* is wrapped in a lower-index *cladding*, and light striking that boundary at a shallow angle reflects entirely back inward rather than escaping. So the signal travels for kilometres with little attenuation and — being optical — is immune to the electromagnetic interference that plagues copper.

*Single-mode* fiber uses a tiny core that supports just one light path, giving the longest reach and highest rates for long-haul links; *multimode* fiber has a wider core that admits several paths at once, which is cheaper to couple into but limits distance because the paths spread the pulse. Fiber underpins long-distance internet backbones, data-center interconnects, and high-speed [Ethernet](/reference/ethernet/), where pluggable transceivers (SFP modules) convert light to and from electrical signals at a [switch](/reference/network-switch/).

Compared with [coaxial cable](/reference/coaxial-cable/), fiber offers vastly more capacity and reach and does not radiate or pick up noise, which is why carriers have pushed it ever closer to the home.

## Single-mode vs multimode

The two families trade cost against reach:

| Trait | Single-mode | Multimode |
|-------|-------------|-----------|
| Core diameter | ~9 µm (one path) | ~50–62.5 µm (many paths) |
| Reach | Kilometres to hundreds of km | Up to a few hundred metres |
| Source | Laser | LED or VCSEL |
| Cost | Higher (precise coupling) | Lower |
| Typical use | Backbones, long-haul | Within a building or rack |

## Where it fits

Fiber's bandwidth and noise immunity come at the cost of more delicate cabling and pricier transceivers, so for most short LAN runs copper [Ethernet](/reference/ethernet/) is simpler; fiber earns its place over distance or in electrically noisy sites. For a GopherTrunk rooftop or tower installation, a fiber backhaul can carry the decoded stream down a long run without picking up the electrical noise a copper cable would — and its immunity to interference keeps it well clear of the RF the antenna is trying to hear. The underlying idea — modulating light instead of an RF carrier — is a close cousin of the modulation an SDR performs on radio waves.

## Sources

[^wiki]: [Optical fiber](https://en.wikipedia.org/wiki/Optical_fiber) — Wikipedia, on fiber-optic transmission, total internal reflection, and single-mode versus multimode fiber.
