---
slug: field-strength
title: Field strength
entry_type: term
category: rf-fundamentals
description: Field strength is the amplitude of a radio wave's electric field at a point, given in volts per metre, and is directly related to the local power density of the wave.
keywords: field strength, electric field strength, volts per metre, V/m, power density, watts per square metre, near field, far field, microvolts per metre, dBµV/m
aka: [field strength, electric field strength, E-field strength]
autolink: true
infobox:
  - { label: Symbol / Unit, value: "E — volts per metre (V/m)" }
  - { label: Relation, value: "S = E² / 377 Ω (far field)" }
  - { label: Type, value: Point measure of wave amplitude }
see_also: [near-field-far-field, power-spectral-density, antenna, wavelength, free-space-path-loss, dbm]
cite_urls:
  - https://en.wikipedia.org/wiki/Field_strength
---

**Field strength** is the amplitude of the electric field a radio wave produces at a given point in
space, expressed in **volts per metre (V/m)** — or, for the weak fields of received signals, in
microvolts per metre (µV/m) or **dBµV/m**.[^wiki] It is a point property of the wave itself,
independent of any particular receiving antenna, which makes it the natural language for describing
coverage, exposure limits, and how strongly a signal arrives before an [antenna](/reference/antenna/)
converts it into a voltage.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A sinusoidal electric field wave whose peak amplitude in volts per metre is the field strength; a caption relates it to power density S equals E squared over 377 ohms in the far field." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor" stroke="none">
    <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
    <path d="M30 70 Q 55 20 80 70 T 130 70 T 180 70 T 230 70 T 280 70 T 330 70 T 380 70 T 430 70" fill="none" stroke="currentColor"/>
    <line x1="55" y1="70" x2="55" y2="30" stroke="currentColor" stroke-dasharray="3 2"/>
    <text x="60" y="34">E (V/m)</text>
    <text x="30" y="100" font-style="italic">Field strength = peak electric-field amplitude at a point</text>
    <text x="30" y="125">Far-field power density:  S = E² / 377 Ω  (W/m²)</text>
    <text x="30" y="145" fill-opacity="0.7">377 Ω = impedance of free space</text>
  </g>
</svg>
<figcaption>Field strength is the electric-field amplitude in volts per metre; in the far field it maps directly to power density through the 377 Ω impedance of free space.</figcaption>
</figure>

## How it works

A propagating radio wave carries coupled electric (**E**) and magnetic (**H**) fields. In the
**far field** — many wavelengths from the antenna — these are locked in a fixed ratio, the
**impedance of free space**, *η₀ ≈ 377 Ω*, and the wave behaves as a simple plane wave. There, field
strength and **power density** *S* (watts per square metre) are two views of the same thing:

**S = E² / η₀** and **E = √(S · η₀)**

So a measured or predicted power density converts straight to a V/m figure, and vice versa. Power
density itself falls as the inverse square of distance from a point source, so field strength falls
as *1/d* — every doubling of distance halves the volts per metre (−6 dB).

The **[near field](/reference/near-field-far-field/)**, within roughly a wavelength (or a couple of
antenna diameters) of the radiator, behaves differently. There the E and H fields are not in the
377 Ω ratio, energy sloshes reactively in and out of the antenna's surroundings, and E and H must be
specified separately — a single "field strength" number does not capture the situation. This is why
exposure and measurement standards treat near-field and far-field regions with different rules.

## In practice

Field strength is what regulators actually limit and measure. Broadcast coverage contours are drawn
at specific V/m or dBµV/m levels (for example, a protected service contour), and RF-exposure limits
are stated as maximum permitted E-fields. Calibrated field-strength meters and antennas with a known
**antenna factor** convert the voltage at a receiver back to the incident field, letting a measured
receiver reading be reported as an absolute, antenna-independent field level.

## Relevance to SDR

An [SDR](/reference/software-defined-radio/) never reads field strength directly — its front end
sees the *voltage* an antenna develops, which depends on that antenna's gain, aperture, and matching.
Field strength is the upstream, antenna-independent quantity: knowing the E-field at your location
plus your antenna's characteristics tells you the power delivered to the receiver, hence whether the
signal clears the [noise floor](/reference/noise-floor/). It is the bridge between propagation
predictions (which produce V/m or power density) and the [dBm](/reference/dbm/) at the SDR input.

**GopherTrunk** works in receiver-side power terms and does not compute field strength, but the
concept explains coverage maps and why a published field-strength contour translates into a
predictable — or marginal — decode at a given site.

## Sources

[^wiki]: [Field strength](https://en.wikipedia.org/wiki/Field_strength) — Wikipedia, definition in volts per metre and the relationship to power density via free-space impedance.
