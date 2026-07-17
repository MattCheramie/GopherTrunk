---
slug: smartphone
title: Smartphone
entry_type: hardware
category: hw-mobile
description: A smartphone is a pocket-sized touchscreen computer with a cellular connection, sensors, and a battery that runs apps, making it the most widespread computer on earth.
keywords: smartphone, mobile phone, iOS, Android, app, touchscreen computer, cellular, SoC, sensors
aka: [Smartphone, Smart phone, Mobile phone]
infobox:
  - { label: Type, value: Pocket touchscreen computer }
  - { label: Connectivity, value: Cellular, Wi-Fi }
  - { label: Platforms, value: iOS, Android }
  - { label: Runs, value: Apps }
  - { label: Role, value: Deployment target }
see_also: [tablet, mobile-app-development, cellular-modem, smartwatch, system-on-a-chip, operating-system]
related_lessons:
  - { title: "Smartphones", url: /learn/intro-hardware/smartphones/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Smartphone
---

**A smartphone** is a pocket-sized touchscreen computer with a cellular
connection, sensors, and a battery, running apps — the most widespread computer
on earth.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="A block diagram of a smartphone's internals. A central system-on-a-chip connects to five surrounding blocks: the cellular and Wi-Fi modem, the battery and power management, a cluster of sensors, the touchscreen display, and storage with memory. The SoC sits at the hub with lines to each." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2" font-family="ui-sans-serif, sans-serif">
    <rect x="185" y="66" width="90" height="42" rx="4" fill="currentColor" fill-opacity="0.06"/>
    <rect x="30" y="20" width="96" height="30" rx="3"/>
    <rect x="334" y="20" width="96" height="30" rx="3"/>
    <rect x="20" y="124" width="96" height="30" rx="3"/>
    <rect x="182" y="130" width="96" height="30" rx="3"/>
    <rect x="344" y="124" width="96" height="30" rx="3"/>
    <g stroke-width="0.9">
      <line x1="185" y1="74" x2="126" y2="42"/>
      <line x1="275" y1="74" x2="334" y2="42"/>
      <line x1="192" y1="108" x2="110" y2="124"/>
      <line x1="230" y1="108" x2="230" y2="130"/>
      <line x1="268" y1="108" x2="360" y2="124"/>
    </g>
  </g>
  <g fill="currentColor" stroke="none" font-family="ui-sans-serif, sans-serif" font-size="8.5" text-anchor="middle">
    <text x="230" y="90">SoC (CPU/GPU)</text>
    <text x="78" y="39">modem &#183; Wi-Fi</text>
    <text x="382" y="39">battery / power</text>
    <text x="68" y="143">sensors</text>
    <text x="230" y="149">touchscreen</text>
    <text x="392" y="143">memory &amp; storage</text>
  </g>
</svg>
<figcaption>Inside a phone, a system-on-a-chip is the hub: it drives the touchscreen, talks to the cellular and Wi-Fi modem, reads a cluster of sensors, and runs off a tightly managed battery, with memory and storage close by.</figcaption>
</figure>

## Overview

Inside, a smartphone has the same building blocks as any computer — a
[CPU](/reference/central-processing-unit/), [RAM](/reference/random-access-memory/),
and [storage](/reference/data-storage/) — most of them integrated into a single
[system-on-a-chip](/reference/system-on-a-chip/). Around that hub sit a
touchscreen, a [cellular modem](/reference/cellular-modem/) and Wi-Fi radio, and a
dense cluster of sensors: accelerometer, gyroscope, magnetometer, GPS, and more.

It runs a mobile [operating system](/reference/operating-system/), almost always
Apple's iOS or Google's Android, and the software it runs comes packaged as apps.
The whole design is a negotiation with the battery: performance, screen
brightness, and radio use are all traded against how long a single charge lasts.

## Phone vs tablet vs wearable

The mobile tiers differ mostly in size, screen, and power budget:

| Trait | Smartphone | Tablet | Wearable |
|-------|-----------|--------|----------|
| Screen | ~6 in | ~10 in | ~1–2 in / none |
| Cellular | Standard | Optional | Optional (eSIM) |
| Battery | A day+ | Days | Hours–a day |
| Primary role | Do-everything | Media, light work | Sensing, notify |
| Standalone | Yes | Yes | Often tethered |

The phone is the anchor of the group; tablets scale it up, wearables shrink it down.

## Where it fits

For developers a smartphone is a deployment target, not a development machine:
you write and build the software on a [laptop](/reference/laptop/) or
[desktop](/reference/desktop-computer/), then run it on the phone. Apps are built
with platform languages — [Swift](/reference/swift-language/) for iOS,
[Kotlin](/reference/kotlin-language/) or [Java](/reference/java-language/) for
Android — covered under [mobile app development](/reference/mobile-app-development/).
In an SDR setup a phone is somewhere to view decoded data over the capture node's
web console, not the machine doing the decoding.

## Sources

[^wiki]: [Smartphone](https://en.wikipedia.org/wiki/Smartphone) — Wikipedia, on the pocket touchscreen computer and its platforms.
