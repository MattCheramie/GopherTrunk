---
slug: compact-position-reporting
title: Compact position reporting (CPR)
entry_type: algorithm
category: estimation-array
description: Compact position reporting is the ADS-B encoding that conveys aircraft latitude and longitude in 17 bits per coordinate, resolved to an unambiguous position from a paired even/odd message.
keywords: CPR, compact position reporting, ADS-B, Mode S, latitude longitude, even odd frame, latitude zones, global decode, local decode, NL function
aka: [compact position reporting, CPR]
autolink: true
infobox:
  - { label: Type, value: Position-encoding scheme }
  - { label: Used by, value: ADS-B (Mode S extended squitter) }
  - { label: Resolves from, value: Even/odd message pair (global) or reference (local) }
see_also: [ads-b, mode-s, multilateration, icao, cyclic-redundancy-check]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Automatic_Dependent_Surveillance%E2%80%93Broadcast
  - https://mode-s.org/decode/content/ads-b/3-airborne-position.html
---

**Compact position reporting** (**CPR**) is the encoding [ADS-B](/reference/ads-b/) uses to
convey an aircraft's latitude and longitude in only **17 bits per coordinate**, by sending a
*relative* position within a grid of zones instead of an absolute one.[^wiki] The saved bits
come at the price of ambiguity — 17 bits cannot label a spot on the whole globe uniquely —
which CPR resolves by alternating between two slightly different grids, the **even** and
**odd** frames, and combining a matched pair.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="An even frame and an odd frame, each on a grid with a different number of latitude zones, combined to resolve one globally unambiguous latitude and longitude." xmlns="http://www.w3.org/2000/svg">
  <rect x="26" y="42" width="66" height="24" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="59" y="58" text-anchor="middle" font-size="9" fill="currentColor">even (60 z)</text>
  <rect x="26" y="76" width="66" height="24" fill="currentColor" fill-opacity="0.26" stroke="currentColor" stroke-width="1.2"/><text x="59" y="92" text-anchor="middle" font-size="9" fill="currentColor">odd (59 z)</text>
  <line x1="96" y1="71" x2="150" y2="71" stroke="currentColor" marker-end="url(#cprar)"/>
  <g stroke="currentColor" stroke-opacity="0.4"><rect x="170" y="26" width="120" height="88" fill="none"/><line x1="210" y1="26" x2="210" y2="114"/><line x1="250" y1="26" x2="250" y2="114"/><line x1="170" y1="55" x2="290" y2="55"/><line x1="170" y1="85" x2="290" y2="85"/></g>
  <circle cx="230" cy="70" r="4" fill="currentColor"/>
  <text x="230" y="128" text-anchor="middle" font-size="8" fill="currentColor">zone index + fraction</text>
  <line x1="298" y1="70" x2="330" y2="70" stroke="currentColor" marker-end="url(#cprar)"/>
  <text x="392" y="66" text-anchor="middle" font-size="9" fill="currentColor">unambiguous</text>
  <text x="392" y="80" text-anchor="middle" font-size="9" fill="currentColor">lat / lon</text>
  <defs><marker id="cprar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>CPR encodes position as a fractional index within latitude zones; the even grid has 60 zones and the odd grid 59, and comparing a matched even/odd pair pins down which zone the aircraft is in.</figcaption>
</figure>

## How it works

CPR tiles the globe into horizontal **latitude zones** and, within each latitude band,
longitude zones. The clever part is that the **even** and **odd** frames use a *different
number of latitude zones* — 60 for even, 59 for odd. Each transmitted coordinate is the
17-bit fractional position of the aircraft *within its current zone*; the zone index itself
is not sent. Because the two grids have zone boundaries that drift apart as you move north or
south, the difference between an even and an odd fractional reading is enough to deduce which
zone the aircraft occupies — like a coarse vernier.

There are two decode modes:

- **Global decoding.** Given a fresh even *and* odd message received close together in time,
  the decoder computes a zone index from their fractional difference (using the number-of-
  longitude-zones function **NL(lat)**), then reconstructs the absolute latitude and
  longitude. This needs no prior knowledge of where the aircraft is, but requires both frame
  parities and assumes the aircraft did not cross a zone boundary between them.
- **Local decoding.** Given a single message *and* a known reference position (a previous fix
  or the receiver's own location), the decoder picks the zone nearest that reference and
  resolves position from one frame alone. This is faster and works per-message once tracking
  is established, but a bad reference can place the aircraft in the wrong zone.

A subtlety is the **NL (number of longitude zones)** transition latitudes: near the boundary
where NL changes, an even and odd pair can straddle two different longitude-zone counts,
which decoders must guard against to avoid a gross error.[^modes]

## In practice

CPR is carried in the ADS-B *airborne position* and *surface position* messages of the
[Mode S](/reference/mode-s/) extended squitter (1090 MHz). Surface messages use a
finer-grained CPR (positions change slowly, so more bits go to resolution). A practical
receiver keeps the last even and odd frame per aircraft (keyed by ICAO address), does a
global decode when it first hears both parities, then switches to per-message local decoding
for the smooth position updates seen on a map. Reasonableness checks — speed since last fix,
NL consistency — catch the occasional mis-resolved zone.

## Relevance to SDR

Any 1090 MHz ADS-B decoder must implement CPR to turn raw
[Mode S](/reference/mode-s/) squitters into mappable latitude/longitude — it is the step
between demodulated bits and a dot on a map. CPR is a self-reported position scheme; it is
distinct from [multilateration](/reference/multilateration/), where ground stations compute
an aircraft's position from time-of-arrival differences without trusting the aircraft's own
coordinates. GopherTrunk includes an ADS-B decode path, and CPR even/odd resolution is
exactly the kind of message-layer processing that path performs to report aircraft
positions.

## Sources

[^wiki]: [Automatic Dependent Surveillance–Broadcast](https://en.wikipedia.org/wiki/Automatic_Dependent_Surveillance%E2%80%93Broadcast) — Wikipedia, for ADS-B and its compact position-reporting encoding.
[^modes]: [The 1090 Megahertz Riddle — Airborne position (CPR)](https://mode-s.org/decode/content/ads-b/3-airborne-position.html) — Junzi Sun, a worked reference for CPR even/odd global and local decoding and the NL function.
