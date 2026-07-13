---
slug: ant-plus
title: ANT+
entry_type: protocol
category: wireless-data-iot
description: "ANT+ is a 2.4 GHz ultra-low-power wireless protocol for fitness and health sensors, using GFSK on adaptive isochronous channels to link heart-rate straps, power meters, and cadence sensors."
keywords: ANT, ANT+, ANT plus, fitness sensor, heart rate monitor, cycling power meter, cadence, 2.4 GHz, GFSK, ultra low power, Garmin, Dynastream, adaptive isochronous
aka: [ANT, ANT+, ANT plus]
autolink: true
infobox:
  - { label: Type, value: Ultra-low-power sensor network }
  - { label: Standards body, value: "ANT+ Alliance (Garmin/Dynastream)" }
  - { label: Introduced, value: "2004; ANT+ 2008" }
  - { label: Access, value: Adaptive isochronous TDMA }
  - { label: Channel spacing, value: 1 MHz (2.4 GHz ISM) }
  - { label: Modulation, value: "GFSK (1 Mbit/s, ~4 byte msgs)" }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [bluetooth-le, gfsk, internet-of-things, frequency-shift-keying, bluetooth-rf, sensor]
cite_urls:
  - https://en.wikipedia.org/wiki/ANT_(network)
  - https://en.wikipedia.org/wiki/Gaussian_frequency-shift_keying
---

**ANT+** is an ultra-low-power 2.4 GHz wireless protocol built for fitness and health
sensors, layering standardised "device profiles" on the underlying **ANT** radio, which
uses [GFSK](/reference/gfsk/) modulation.[^wiki] It is the technology that links a
heart-rate strap, cycling power meter, or cadence sensor to a bike computer or watch, and
it long predated (and now coexists with) [Bluetooth LE](/reference/bluetooth-le/) in the
same role.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="One low-power ANT+ sensor such as a heart-rate strap broadcasting to several displays at once — a watch, a bike computer, and a phone — over a shared 2.4 GHz channel." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="antar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle" stroke="currentColor">
    <circle cx="80" cy="75" r="26" fill="currentColor" fill-opacity="0.25"/><text x="80" y="72" stroke="none">HR</text><text x="80" y="82" stroke="none">sensor</text>
    <line x1="108" y1="60" x2="300" y2="35" stroke-width="1" marker-end="url(#antar)"/>
    <line x1="110" y1="75" x2="300" y2="75" stroke-width="1" marker-end="url(#antar)"/>
    <line x1="108" y1="90" x2="300" y2="115" stroke-width="1" marker-end="url(#antar)"/>
    <rect x="305" y="22" width="90" height="26" rx="3" fill="none"/><text x="350" y="39" stroke="none">watch</text>
    <rect x="305" y="62" width="90" height="26" rx="3" fill="none"/><text x="350" y="79" stroke="none">bike computer</text>
    <rect x="305" y="102" width="90" height="26" rx="3" fill="none"/><text x="350" y="119" stroke="none">phone app</text>
  </g>
  <text x="230" y="145" text-anchor="middle" font-size="9" fill="currentColor">one broadcast sensor → many receivers</text>
</svg>
<figcaption>An ANT+ sensor broadcasts once and can be received by several displays at the same time, unlike a one-to-one paired link.</figcaption>
</figure>

## Overview

ANT is a proprietary but openly documented protocol from Dynastream (a Garmin company).
Its radio sends short (~8-byte) messages using GFSK at 1 Mbit/s on the 2.4 GHz band,
coordinated by an "adaptive isochronous" scheme: channels transmit in scheduled slots and
shift timing to avoid colliding, letting many independent sensor networks share the air.
A hallmark is *broadcast* operation — a heart-rate sensor can transmit to a watch, a bike
computer, and a phone simultaneously, without pairing to each. **ANT+** is the
interoperability layer: agreed device profiles (heart rate, bike power, speed/cadence,
running dynamics) that guarantee any ANT+ display understands any ANT+ sensor of that
type.

## Technical characteristics

| Property | Value |
|----------|-------|
| Band | 2.4 GHz ISM |
| Modulation | GFSK |
| Bit rate | 1 Mbit/s (short messages) |
| Access | Adaptive isochronous TDMA channels |
| Message size | ~8 bytes payload |
| Topology | Broadcast, star, mesh, shared channels |

The extreme frugality — a sensor can run for years on a coin cell — comes from tiny,
infrequent messages and a very simple radio, trading throughput for battery life.

## History

Dynastream introduced ANT in 2004 and the ANT+ interoperability profiles around 2008.
Garmin's acquisition entrenched it across sports electronics. Many modern sensors now
transmit ANT+ and Bluetooth LE at once so they work with both ecosystems.

## Deployment

ANT+ is ubiquitous in cycling, running, and fitness equipment — power meters, heart-rate
straps, indoor trainers, and gym machines — where its broadcast-to-many model and low
power suit multiple simultaneous displays. BLE has eroded its exclusivity but not
displaced it in serious sports gear.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode ANT+. Its 2.4 GHz GFSK, adaptive channel timing, and
proprietary framing sit outside GopherTrunk's land-mobile decode chain; enthusiasts use
dedicated ANT USB sticks or SDR flow graphs for it. It is unrelated to GopherTrunk's
trunking and aeronautical targets.

## Sources

[^wiki]: [ANT (network)](https://en.wikipedia.org/wiki/ANT_(network)) — Wikipedia, on the ANT/ANT+ ultra-low-power sensor protocol, its 2.4 GHz GFSK radio, adaptive isochronous channels, and device profiles.
