---
slug: gps-receiver
title: GPS receiver
entry_type: hardware
category: hw-mobile
description: A GPS receiver determines a device's position by timing signals from a constellation of navigation satellites, providing location, speed, and precise time to phones, wearables, and embedded systems.
keywords: GPS receiver, GNSS, satellite navigation, GLONASS, Galileo, positioning, location, GPS chip, PPS, timing
aka: [GNSS receiver]
infobox:
  - { label: Type, value: Satellite navigation receiver }
  - { label: Provides, value: Position, speed, time }
  - { label: Systems, value: GPS, GLONASS, Galileo, BeiDou }
  - { label: Needs, value: Sky view, ≥4 satellites }
see_also: [cellular-modem, system-on-a-chip, smartphone, smartwatch, wearable-computer, antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/Satellite_navigation
  - https://en.wikipedia.org/wiki/Global_Positioning_System
---

A **GPS receiver** determines a device's position by timing radio signals from a constellation of navigation satellites, yielding location, speed, and a very precise time reference.[^gnss]

## Overview

The receiver listens for the faint signals of satellites overhead and, from the time each took to arrive, solves for its own position — needing a clear sky view and signals from at least four satellites for a 3D fix. Modern chips are *GNSS* receivers, combining the US GPS system with GLONASS, Galileo, and BeiDou for faster, more reliable fixes.[^gps] In a phone the GPS block sits in or beside the [SoC](/reference/system-on-a-chip/), with its own [antenna](/reference/antenna/), and often takes hints from the [cellular modem](/reference/cellular-modem/) (assisted GPS) to lock faster.

## Where it fits

Location is the killer feature GPS adds to phones, [smartwatches](/reference/smartwatch/), and [wearables](/reference/wearable-computer/) — maps, fitness routes, geotagging. Its precise one-pulse-per-second timing is also valuable beyond navigation: a GPS-disciplined clock can give an SDR setup like GopherTrunk an accurate frequency and time reference, useful for stable tuning and for timestamping decoded calls across distributed capture nodes.

## Sources

[^gnss]: [Satellite navigation](https://en.wikipedia.org/wiki/Satellite_navigation) — Wikipedia, on GNSS positioning.
[^gps]: [Global Positioning System](https://en.wikipedia.org/wiki/Global_Positioning_System) — Wikipedia, on the GPS system specifically.
