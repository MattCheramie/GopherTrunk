---
slug: multilateration
title: Multilateration (MLAT)
entry_type: algorithm
category: estimation-array
description: Multilateration locates an emitter from the time differences of arrival of its signal at several synchronised receivers, each pair defining a hyperbola whose intersection is the position.
keywords: multilateration, MLAT, TDOA, time difference of arrival, hyperbolic positioning, ADS-B, Mode S, aircraft tracking, emitter geolocation, receiver synchronisation
aka: [multilateration, MLAT, hyperbolic positioning, TDOA positioning]
autolink: true
infobox:
  - { label: Type, value: TDOA position estimator }
  - { label: Recovers, value: Emitter position }
  - { label: Used by, value: ADS-B/Mode S MLAT, geolocation }
see_also: [mode-s, ads-b, gps-receiver, kalman-filter, compact-position-reporting]
cite_urls:
  - https://en.wikipedia.org/wiki/Multilateration
  - https://en.wikipedia.org/wiki/Pseudorange_multilateration
---

**Multilateration** (MLAT) computes the position of a signal source from the *differences*
in arrival time of one transmission recorded at several receivers at known, surveyed
locations.[^wiki] A single time-difference-of-arrival ([TDOA](/reference/mode-s/)) between
two receivers constrains the emitter to a **hyperbola** (a hyperboloid in 3-D) — the locus
of points whose range difference to that receiver pair is fixed — and the intersection of
several such hyperbolas from multiple pairs pins down the location. Crucially it needs only
*relative* timing, so the emitter itself carries no clock and need not cooperate.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="Three ground receivers each record the same transmission at slightly different times; each pair's time difference defines a hyperbola, and the three hyperbolas intersect at the emitter's position." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <path d="M60 150 Q 150 60 250 40" fill="none" stroke="currentColor" stroke-width="1" opacity="0.7"/>
    <path d="M410 150 Q 300 70 250 40" fill="none" stroke="currentColor" stroke-width="1" opacity="0.7"/>
    <path d="M150 165 Q 220 90 250 40" fill="none" stroke="currentColor" stroke-width="1" opacity="0.7"/>
    <text x="120" y="100" opacity="0.8">hyperbola (RX1,RX2)</text>
    <text x="340" y="105" opacity="0.8">hyperbola (RX2,RX3)</text>
    <circle cx="250" cy="40" r="5" fill="currentColor"/><text x="250" y="30">emitter</text>
    <g stroke="currentColor" stroke-width="1.2" fill="none"><rect x="45" y="150" width="16" height="12"/><rect x="200" y="160" width="16" height="12"/><rect x="400" y="150" width="16" height="12"/></g>
    <text x="53" y="174">RX1 t₁</text><text x="208" y="184">RX2 t₂</text><text x="408" y="174">RX3 t₃</text>
    <text x="230" y="130" text-anchor="middle">Δt₁₂, Δt₂₃, Δt₁₃  →  intersection = position</text>
  </g>
  <defs><marker id="mlar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Each receiver pair's time difference of arrival defines a hyperbola of constant range difference; the emitter sits where the hyperbolas from independent pairs cross.</figcaption>
</figure>

## How it works

Radio waves travel at a known speed, so a difference in arrival time maps directly to a
difference in path length. With receivers at surveyed positions and a common time reference:

- **Measure TDOAs.** Cross-correlate the same waveform captured at each receiver to get the
  time offset between every pair. Each `Δt` fixes a range *difference*, hence one hyperbola.
- **Solve the geometry.** Two hyperbolas give a 2-D fix; a third (or receivers at differing
  heights) resolves altitude and the ambiguity of which branch. With more receivers than
  unknowns the system is overdetermined and solved by least squares, improving accuracy and
  detecting bad measurements.
- **Smooth over time.** Because a target moves, successive fixes are usually fed to a
  [Kalman filter](/reference/kalman-filter/) that fuses them with a motion model to produce a
  clean track and reject outliers.

The hard engineering problem is **time synchronisation**: 1 nanosecond of timing error is
about 30 cm of position error, so the receivers must share a common clock (GNSS-disciplined,
or calibrated against a reference transmitter at a known location). Accuracy also depends on
geometry — the dilution of precision is poor when the receivers and target are nearly
collinear.

## Relevance to SDR

The highest-profile use is aviation: [Mode S](/reference/mode-s/) and
[ADS-B](/reference/ads-b/) MLAT networks locate aircraft from the arrival times of their
1090 MHz replies and squitters at multiple ground stations. This provides independent
surveillance for aircraft that broadcast an identity but no position, and a cross-check on
the GPS-derived positions that ADS-B aircraft self-report via
[compact position reporting](/reference/compact-position-reporting/). The same TDOA
principle geolocates emitters generally — interference hunting, SIGINT, and wildlife
tags — and is the mirror image of how a [GPS receiver](/reference/gps-receiver/) works
(there the *receiver* solves its own position from many synchronised *transmitters*).
**GopherTrunk** decodes the ADS-B/Mode S messages a single receiver can hear, but
multilateration itself requires several time-synchronised receivers sharing captures, which
is a network-level system beyond what a standalone GT node does.

## Sources

[^wiki]: [Multilateration](https://en.wikipedia.org/wiki/Multilateration) — Wikipedia, on hyperbolic position estimation from time-difference-of-arrival measurements.
