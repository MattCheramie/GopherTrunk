---
slug: compact-position-reporting
title: Compact position reporting (CPR)
entry_type: algorithm
category: algorithms
description: Compact position reporting is the ADS-B encoding that conveys aircraft latitude and longitude in few bits, resolved to a precise position from a pair of even/odd messages.
keywords: CPR, compact position reporting, ADS-B, latitude longitude, even odd, Mode S
aka: [compact position reporting, CPR]
autolink: true
infobox:
  - { label: Type, value: Position-encoding scheme }
  - { label: Used by, value: ADS-B }
  - { label: Resolves from, value: Even/odd message pair }
see_also: [ads-b, cyclic-redundancy-check, icao]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/other-signals/ }
external:
  - { title: "Compact Position Reporting", url: https://en.wikipedia.org/wiki/Automatic_Dependent_Surveillance%E2%80%93Broadcast }
---

**Compact position reporting** (**CPR**) is the encoding [ADS-B](/reference/ads-b/) uses to
convey an aircraft's latitude and longitude in few bits, trading a small amount of
ambiguity for compactness.

## How it works

CPR encodes position relative to a grid of zones. A globally unambiguous fix is recovered
from a matched pair of **even** and **odd** format messages, or locally from a known
reference position.

## Relevance to SDR

An ADS-B decoder must implement CPR to turn raw messages into mappable aircraft positions.
