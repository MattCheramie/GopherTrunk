---
slug: gnss
title: Global Navigation Satellite System (GNSS)
entry_type: technology
category: satellite-gnss
description: "GNSS is the umbrella term for satellite constellations that broadcast timing signals a receiver uses to compute position, velocity, and time anywhere on Earth."
keywords: GNSS, Global Navigation Satellite System, satellite navigation, satnav, GPS, GLONASS, Galileo, BeiDou, positioning, PNT, ranging
aka: [GNSS, satnav, satellite navigation, PNT]
autolink: true
infobox:
  - { label: Type, value: Satellite positioning umbrella }
  - { label: Idea, value: One-way ranging from timing broadcasts }
  - { label: Examples, value: "GPS, GLONASS, Galileo, BeiDou" }
see_also: [gps-gnss, glonass, galileo, beidou, doppler-shift, multilateration]
cite_urls:
  - https://en.wikipedia.org/wiki/Satellite_navigation
  - https://www.gps.gov/systems/gnss/
---

**GNSS** (**Global Navigation Satellite System**) is the collective name for the
space-based systems that let a receiver fix its position, velocity, and time (PNT)
anywhere on Earth by listening to timing signals from orbiting satellites.[^wiki] It
is an umbrella term, not a single system: the four global constellations are the
United States' [GPS](/reference/gps-gnss/), Russia's
[GLONASS](/reference/glonass/), the European Union's [Galileo](/reference/galileo/),
and China's [BeiDou](/reference/beidou/), joined by regional systems such as India's
NavIC and Japan's QZSS.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="A receiver on Earth computes its position from the timing signals of four satellites overhead." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="gnar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <circle cx="60" cy="30" r="7" fill="currentColor"/>
  <circle cx="180" cy="18" r="7" fill="currentColor"/>
  <circle cx="300" cy="26" r="7" fill="currentColor"/>
  <circle cx="410" cy="40" r="7" fill="currentColor"/>
  <path d="M230 165 L66 40" stroke="currentColor" stroke-opacity="0.6" marker-end="url(#gnar)"/>
  <path d="M230 165 L184 28" stroke="currentColor" stroke-opacity="0.6" marker-end="url(#gnar)"/>
  <path d="M230 165 L296 36" stroke="currentColor" stroke-opacity="0.6" marker-end="url(#gnar)"/>
  <path d="M230 165 L404 50" stroke="currentColor" stroke-opacity="0.6" marker-end="url(#gnar)"/>
  <rect x="218" y="165" width="24" height="16" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
  <text x="230" y="196" text-anchor="middle" font-size="10" fill="currentColor">receiver solves x, y, z, and clock bias</text>
  <text x="60" y="16" text-anchor="middle" font-size="9" fill="currentColor">sat 1</text>
  <text x="180" y="8" text-anchor="middle" font-size="9" fill="currentColor">sat 2</text>
  <text x="410" y="28" text-anchor="middle" font-size="9" fill="currentColor">sat 4</text>
</svg>
<figcaption>Ranges to four or more satellites resolve three position unknowns plus the receiver's clock error.</figcaption>
</figure>

## How it works

Every GNSS shares the same principle: one-way ranging. Each satellite carries an
atomic clock and continuously broadcasts a signal stamped with the exact time of
transmission and the satellite's own orbit (the ephemeris). A receiver measures how
long each signal took to arrive by comparing the incoming timestamp with its own
clock, and multiplies that delay by the speed of light to get a *pseudorange* — the
distance to that satellite, biased by the receiver's own clock error.

Because the receiver clock is cheap and imperfect, that clock bias is treated as a
fourth unknown alongside the three spatial coordinates. Solving for four unknowns
needs four equations, so a receiver must hear **at least four satellites**
simultaneously; more satellites over-determine the solution and improve accuracy.
Geometrically this is [multilateration](/reference/multilateration/): each
pseudorange defines a sphere around a satellite, and the position is where the
spheres intersect once the common clock error is absorbed. The satellite orbits at
roughly 20,000 km altitude in medium Earth orbit, so signals arrive extremely weak —
below the receiver's thermal noise floor — and are recovered by correlating against a
known spreading code (see [GPS](/reference/gps-gnss/)).

The constellations differ in the details that ride on top of this shared idea. GPS,
Galileo, and BeiDou separate satellites by unique spreading codes on a shared
frequency ([CDMA](/reference/cdma/)); classic GLONASS instead gives each satellite
its own frequency ([FDMA](/reference/fdma/)). All of them place signals in the L-band
(roughly 1.1–1.6 GHz), and modern receivers combine several constellations and
frequencies at once for faster fixes and better resilience.

Two effects complicate the measurement. Satellites move fast relative to the ground,
so their carriers arrive with a substantial [Doppler shift](/reference/doppler-shift/)
of several kilohertz that the receiver must search over and track. And the signal
passes through the ionosphere, which delays it by a frequency-dependent amount;
dual-frequency receivers cancel most of this error by comparing two bands.

## Relevance to SDR

GNSS is everywhere in radio, but mostly as infrastructure rather than as a decode
target. Any system needing precise time or frequency — cellular base stations, the
timestamps in trunked-radio simulcast, a [GPSDO](/reference/gpsdo/) disciplining an
SDR's reference oscillator — leans on a GNSS receiver in the background. A dedicated
[GPS receiver](/reference/gps-receiver/) chip does the heavy correlation and simply
outputs position and a one-pulse-per-second tick.

Decoding GNSS from raw IQ with a software-defined radio is a well-known but demanding
exercise: the signals are below the noise floor, so recovery requires long coherent
integration and a code/Doppler search, and a typical SDR needs an active L-band
antenna with a low-noise amplifier to see them at all. Open projects (GNSS-SDR and
others) do exactly this. **GopherTrunk** is a land-mobile trunking scanner and does
**not** decode GNSS — satellite navigation is out of scope for its VHF/UHF trunking
focus. GNSS matters to GopherTrunk only indirectly, as the timing source that can
discipline the receiver hardware it runs on.

## Sources

[^wiki]: [Satellite navigation](https://en.wikipedia.org/wiki/Satellite_navigation) — Wikipedia, for the definition of GNSS, the four global constellations, the one-way ranging principle, and the need for four satellites.
