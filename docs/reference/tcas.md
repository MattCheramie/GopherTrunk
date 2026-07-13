---
slug: tcas
title: TCAS
entry_type: technology
category: aviation-marine
description: "TCAS (Traffic Collision Avoidance System) is an airborne system that interrogates nearby transponders on 1030/1090 MHz and issues resolution advisories to prevent mid-air collisions."
keywords: TCAS, ACAS, Traffic Collision Avoidance System, resolution advisory, traffic advisory, Mode S interrogation, 1030 MHz, 1090 MHz, mid-air collision, climb descend
aka: [TCAS, ACAS, TCAS II]
autolink: true
infobox:
  - { label: Type, value: Airborne collision avoidance }
  - { label: Idea, value: Interrogate transponders, coordinate an escape maneuver }
  - { label: Interrogates, value: "Mode S and Mode A/C on 1030/1090 MHz" }
see_also: [mode-s, mode-ac, ads-b, multilateration, icao]
cite_urls:
  - https://en.wikipedia.org/wiki/Traffic_collision_avoidance_system
  - https://en.wikipedia.org/wiki/Airborne_collision_avoidance_system
---

**TCAS** (**Traffic Collision Avoidance System**), known internationally as ACAS, is an
airborne system that independently interrogates the transponders of nearby aircraft on
**1030/1090 MHz** and, when a collision threatens, issues an aural and visual
**resolution advisory** ("CLIMB, CLIMB" / "DESCEND, DESCEND") to the pilots.[^wiki] It
works without any ground infrastructure, reusing the same [Mode S](/reference/mode-s/)
and [Mode A/C](/reference/mode-ac/) transponder signalling that secondary radar uses.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Two aircraft on converging paths interrogate each other's transponders, exchange coordinated Mode S messages, and one climbs while the other descends." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="tcasar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <path d="M50 100 l30 8 l-8 -14 m8 14 l14 -10" stroke="currentColor" stroke-width="1.4" fill="none"/>
  <text x="55" y="122" font-size="7.5" fill="currentColor">own</text>
  <path d="M380 60 l-30 8 l8 -14 m-8 14 l-14 -10" stroke="currentColor" stroke-width="1.4" fill="none"/>
  <text x="378" y="52" font-size="7.5" fill="currentColor">intruder</text>
  <path d="M100 95 L340 65" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3" marker-end="url(#tcasar)"/>
  <text x="220" y="72" text-anchor="middle" font-size="7.5" fill="currentColor">1030 MHz interrogation</text>
  <path d="M340 75 L100 105" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3" marker-end="url(#tcasar)"/>
  <text x="220" y="112" text-anchor="middle" font-size="7.5" fill="currentColor">1090 MHz reply + RA coordination</text>
  <path d="M80 108 v22" stroke="currentColor" stroke-width="1.4" marker-end="url(#tcasar)"/><text x="95" y="128" font-size="7.5" fill="currentColor">DESCEND</text>
  <path d="M360 62 v-22" stroke="currentColor" stroke-width="1.4" marker-end="url(#tcasar)"/><text x="375" y="34" font-size="7.5" fill="currentColor">CLIMB</text>
</svg>
<figcaption>TCAS interrogates an intruder's transponder, and the two units coordinate over Mode S so one climbs and the other descends.</figcaption>
</figure>

## How it works

TCAS carries its own interrogator. Roughly once per second it transmits
[Mode S](/reference/mode-s/) and [Mode A/C](/reference/mode-ac/) interrogations on
**1030 MHz** and listens for replies on **1090 MHz**. From each reply's round-trip
delay it measures **range**, from the change in range it estimates **closure rate**,
and from the reported (Mode C / Mode S) altitude it tracks vertical separation. It does
*not* rely on bearing for the core logic — it reasons primarily in the range/altitude
"tau" domain, projecting time-to-closest-approach.

The alerting has two tiers:

- **Traffic Advisory (TA)** — "TRAFFIC, TRAFFIC" draws the crew's attention to a
  converging target but commands no maneuver.
- **Resolution Advisory (RA)** — a vertical command (climb, descend, or maintain rate)
  that provides guaranteed vertical miss distance.

Crucially, when two TCAS II aircraft threaten each other, they **coordinate** their RAs
over the Mode S data link so the maneuvers are complementary — if one is told to climb,
the other is told to descend. This handshake is what makes TCAS safe against the failure
mode where both aircraft dodge the same way. TCAS II is mandated for large transport
aircraft worldwide; the simpler TCAS I gives only traffic advisories.

## Relevance to SDR

TCAS is not a signal an SDR user "decodes" as a product in itself, but its interrogations
and coordination messages are a prominent, high-power feature of the 1030/1090 MHz
environment that anyone monitoring aviation bands will encounter. TCAS interrogations
add to the interrogation load that a passive 1090 MHz receiver sees, and TCAS-equipped
aircraft's own [Mode S](/reference/mode-s/) replies (including any
[ADS-B](/reference/ads-b/) extended squitters) are decoded like any other. TCAS also
illustrates the same range-from-timing principle that ground-based
[multilateration](/reference/multilateration/) networks exploit to locate non-ADS-B
aircraft.

**GopherTrunk** does not implement or decode TCAS: it is a receiver focused on
land-mobile trunking plus 1090 MHz ADS-B, and it neither interrogates transponders nor
reconstructs collision-avoidance logic. TCAS is included here as context for the
1030/1090 MHz aviation ecosystem GopherTrunk's ADS-B decoder lives alongside.

## Sources

[^wiki]: [Traffic collision avoidance system](https://en.wikipedia.org/wiki/Traffic_collision_avoidance_system) — Wikipedia, for TCAS/ACAS operation, 1030/1090 MHz Mode S/Mode A/C interrogation, range-and-altitude threat logic, and coordinated resolution advisories.
