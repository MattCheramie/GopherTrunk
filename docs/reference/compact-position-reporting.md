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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="An even frame and an odd frame combined to resolve a globally unambiguous latitude and longitude on a grid." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="40" width="70" height="26" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="65" y="57" text-anchor="middle" font-size="9" fill="currentColor">even</text>
  <rect x="30" y="74" width="70" height="26" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.2"/><text x="65" y="91" text-anchor="middle" font-size="9" fill="currentColor">odd</text>
  <line x1="104" y1="70" x2="150" y2="70" stroke="currentColor" marker-end="url(#cprar)"/>
  <g stroke="currentColor" stroke-opacity="0.4"><rect x="170" y="30" width="120" height="80" fill="none"/><line x1="210" y1="30" x2="210" y2="110"/><line x1="250" y1="30" x2="250" y2="110"/><line x1="170" y1="57" x2="290" y2="57"/><line x1="170" y1="83" x2="290" y2="83"/></g>
  <circle cx="230" cy="70" r="4" fill="currentColor"/>
  <text x="360" y="74" text-anchor="middle" font-size="9" fill="currentColor">unambiguous lat/lon</text>
  <defs><marker id="cprar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Compact position reporting (ADS-B) combines an even and odd message to pin down a position with few bits.</figcaption>
</figure>

## How it works

CPR encodes position relative to a grid of zones. A globally unambiguous fix is recovered
from a matched pair of **even** and **odd** format messages, or locally from a known
reference position.

## Relevance to SDR

An ADS-B decoder must implement CPR to turn raw messages into mappable aircraft positions.
