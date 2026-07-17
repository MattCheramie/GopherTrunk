---
slug: gps-receiver
title: GPS receiver
entry_type: hardware
category: hw-mobile
description: A GPS receiver determines a device's position by timing signals from a constellation of navigation satellites, providing location, speed, and precise time to phones, wearables, and embedded systems.
keywords: GPS receiver, GNSS, satellite navigation, GLONASS, Galileo, BeiDou, positioning, location, GPS chip, PPS, timing, trilateration
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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="A trilateration sketch. Three satellites near the top each broadcast a timing signal. From each satellite, the measured travel time defines a range shown as an arc. The three range arcs intersect at one point on the ground, which is the receiver's solved position. A fourth satellite is needed to also solve for the receiver's clock offset." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2" font-family="ui-sans-serif, sans-serif">
    <g fill-opacity="0" >
      <rect x="70" y="20" width="26" height="16" rx="2"/>
      <rect x="217" y="14" width="26" height="16" rx="2"/>
      <rect x="360" y="24" width="26" height="16" rx="2"/>
    </g>
    <g stroke-width="0.9">
      <line x1="64" y1="24" x2="58" y2="18"/><line x1="102" y1="24" x2="108" y2="18"/>
      <line x1="211" y1="18" x2="205" y2="12"/><line x1="249" y1="18" x2="255" y2="12"/>
      <line x1="354" y1="28" x2="348" y2="22"/><line x1="392" y1="28" x2="398" y2="22"/>
    </g>
    <path d="M120 150 A83 83 0 0 1 30 128" stroke-dasharray="4 3"/>
    <path d="M230 150 A120 120 0 0 1 230 30" stroke-dasharray="4 3"/>
    <path d="M140 150 A80 80 0 0 0 300 132" stroke-dasharray="4 3"/>
    <g stroke-width="0.8" stroke-dasharray="2 3">
      <line x1="83" y1="36" x2="205" y2="150"/>
      <line x1="230" y1="30" x2="205" y2="150"/>
      <line x1="373" y1="40" x2="205" y2="150"/>
    </g>
  </g>
  <circle cx="205" cy="150" r="4" fill="currentColor" stroke="none"/>
  <g fill="currentColor" stroke="none" font-family="ui-sans-serif, sans-serif" font-size="8.5" text-anchor="middle">
    <text x="83" y="50">sat 1</text>
    <text x="230" y="44">sat 2</text>
    <text x="373" y="54">sat 3</text>
    <text x="205" y="168">receiver fix</text>
    <text x="230" y="184" font-size="8">a 4th satellite solves the clock offset</text>
  </g>
</svg>
<figcaption>Each satellite's signal travel time gives a distance — a range sphere seen edge-on as an arc. Three ranges cross at the receiver's position; a fourth satellite pins down the receiver clock so the ranges are consistent.</figcaption>
</figure>

## Overview

The receiver listens for the faint signals of satellites overhead and, from the time each took to arrive, solves for its own position — needing a clear sky view and signals from at least four satellites: three for position and one to resolve the receiver's own clock error. Modern chips are *GNSS* receivers, combining the US GPS system with GLONASS, Galileo, and BeiDou for faster, more reliable fixes.[^gps]

In a phone the GPS block sits in or beside the [SoC](/reference/system-on-a-chip/), with its own [antenna](/reference/antenna/), and often takes hints from the [cellular modem](/reference/cellular-modem/) (assisted GPS, which downloads the satellite almanac over the network) to lock far faster than from a cold start.

## The GNSS constellations

Four independent systems now share the sky, and a modern receiver uses several at once:

| System | Operator | Note |
|--------|----------|------|
| GPS | United States | The original, global |
| GLONASS | Russia | Global, aids high latitudes |
| Galileo | European Union | Global, civil-run |
| BeiDou | China | Global since ~2020 |

More visible satellites means faster locks and better accuracy in urban canyons where sky view is limited.

## Where it fits

Location is the killer feature GPS adds to phones, [smartwatches](/reference/smartwatch/), and [wearables](/reference/wearable-computer/) — maps, fitness routes, geotagging. Its precise one-pulse-per-second timing is also valuable beyond navigation: a GPS-disciplined clock can give an SDR setup like GopherTrunk an accurate frequency and time reference, useful for stable tuning and for timestamping decoded calls consistently across distributed capture nodes.

## Sources

[^gnss]: [Satellite navigation](https://en.wikipedia.org/wiki/Satellite_navigation) — Wikipedia, on GNSS positioning.
[^gps]: [Global Positioning System](https://en.wikipedia.org/wiki/Global_Positioning_System) — Wikipedia, on the GPS system specifically.
