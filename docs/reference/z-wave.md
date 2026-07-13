---
slug: z-wave
title: Z-Wave
entry_type: protocol
category: wireless-data-iot
description: "Z-Wave is a sub-GHz low-power mesh networking protocol for home automation, using GFSK modulation on regional ISM frequencies near 900 MHz for reliable short-range control."
keywords: Z-Wave, home automation, sub-GHz, 908 MHz, 868 MHz, GFSK, mesh network, Z-Wave Alliance, Silicon Labs, smart home, ITU-T G.9959
aka: [Z-Wave, ZWave]
autolink: true
infobox:
  - { label: Type, value: Sub-GHz mesh home automation }
  - { label: Standards body, value: "Z-Wave Alliance; ITU-T G.9959" }
  - { label: Introduced, value: "2001" }
  - { label: Access, value: "CSMA/CA, source-routed mesh" }
  - { label: Channel spacing, value: Regional (e.g. 908.4 MHz US) }
  - { label: Modulation, value: GFSK (9.6/40/100 kbit/s) }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [gfsk, home-automation, internet-of-things, zigbee-802154, frequency-bands, frequency-shift-keying]
cite_urls:
  - https://en.wikipedia.org/wiki/Z-Wave
  - https://en.wikipedia.org/wiki/Gaussian_frequency-shift_keying
---

**Z-Wave** is a low-power wireless mesh protocol for
[home automation](/reference/home-automation/) that operates in sub-GHz ISM bands and
uses [GFSK](/reference/gfsk/) modulation.[^wiki] By running near 900 MHz rather than in
the crowded 2.4 GHz band, it sidesteps Wi-Fi and Bluetooth congestion and gets better
wall penetration and range, at the cost of lower data rates than
[Zigbee](/reference/zigbee-802154/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A Gaussian frequency-shift-keying waveform where a smoothed data stream shifts a sub-GHz carrier between two frequencies to send ones and zeros, with regional carrier frequencies listed." xmlns="http://www.w3.org/2000/svg">
  <path d="M30 70 Q40 70 45 55 T60 55 Q68 55 72 85 T88 85 Q96 85 100 55 T118 55 Q126 55 130 85 T148 85 Q156 85 160 55 T178 55 Q186 55 190 85 T208 85" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <g font-size="9" fill="currentColor">
    <text x="235" y="45">high tone = 1</text>
    <text x="235" y="60">low tone = 0</text>
    <text x="235" y="82">Gaussian-smoothed FSK</text>
    <text x="235" y="104">≈ 868 MHz (EU) · 908 MHz (US) · 921 MHz (AU)</text>
  </g>
  <text x="120" y="122" text-anchor="middle" font-size="9" fill="currentColor">carrier shifts between two frequencies</text>
</svg>
<figcaption>Z-Wave sends bits by shifting a sub-GHz carrier between two Gaussian-shaped tones, using region-specific frequencies near 900 MHz.</figcaption>
</figure>

## Overview

Z-Wave keeps things deliberately simple and robust. Nodes form a source-routed mesh: a
controller learns the topology and can route a command through up to four hops of
mains-powered repeater nodes to reach a distant device. Battery devices sleep most of
the time and wake to poll. Crucially, every region uses a mandated frequency so all
devices in a market interoperate — 908.4 MHz in North America, 868.4 MHz in Europe,
and other assignments elsewhere. The radio is GFSK at 9.6, 40, or 100 kbit/s, with the
long-range variant reaching farther on a star topology.

## Technical characteristics

| Property | Value |
|----------|-------|
| Bands | Sub-GHz ISM, regional (~865–926 MHz) |
| Modulation | GFSK |
| Bit rate | 9.6 / 40 / 100 kbit/s |
| Topology | Source-routed mesh (≤4 hops) |
| PHY/MAC | ITU-T G.9959 |
| Max nodes | 232 per network (classic) |

Running below 1 GHz is Z-Wave's main technical differentiator: less contention and
longer indoor range than 2.4 GHz systems, but narrower channels and lower throughput.
The regional frequency mandate is why a device bought in one region will not talk to a
controller from another.

## History

Z-Wave was created by the Danish firm Zensys in 2001, later acquired by Sigma Designs
and then Silicon Labs. The lower layers were published as ITU-T recommendation G.9959,
and much of the specification was opened to the public in the late 2010s under the
Z-Wave Alliance.

## Deployment

Z-Wave is common in smart locks, thermostats, lighting, and security sensors, and is a
staple of professionally installed home-automation and alarm systems. It competes head
to head with Zigbee; hubs frequently support both.

## Decoding it with GopherTrunk

GopherTrunk does **not** decode Z-Wave. Although its sub-GHz GFSK signals fall within
the tuning range of common SDRs, the protocol's framing, mesh routing, and security are
outside GopherTrunk's scope — it targets land-mobile voice trunking and aeronautical
data, not home-automation control. Hobbyists use dedicated Z-Wave sniffers or SDR flow
graphs for this traffic instead.

## Sources

[^wiki]: [Z-Wave](https://en.wikipedia.org/wiki/Z-Wave) — Wikipedia, on the sub-GHz home-automation mesh protocol, its GFSK radio, regional frequencies, and G.9959 lower layers.
