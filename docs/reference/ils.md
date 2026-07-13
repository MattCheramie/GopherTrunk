---
slug: ils
title: Instrument Landing System (ILS)
entry_type: technology
category: aviation-marine
description: ILS (Instrument Landing System) is an aviation precision-approach navaid whose localizer and glideslope beams guide an aircraft to a runway using the depth of two overlapping amplitude-modulated tones.
keywords: ILS, Instrument Landing System, precision approach, localizer, glideslope, glide path, 90 Hz 150 Hz, difference in depth of modulation, DDM, marker beacon, CAT III
aka: [ILS]
autolink: true
infobox:
  - { label: Type, value: Precision-approach navaid }
  - { label: Idea, value: Balance of 90 Hz and 150 Hz tones = on-course }
  - { label: Beams, value: Localizer (VHF) + glideslope (UHF) }
see_also: [amplitude-modulation, vor, dme, frequency-modulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Instrument_landing_system
  - https://www.icao.int/
---

**ILS** (**Instrument Landing System**) is a precision-approach navaid that guides an
aircraft down to a runway along a fixed lateral and vertical path, even in poor
visibility. It uses two radiated beams — a **localizer** for left/right alignment and a
**glideslope** for the descent angle — each formed from two overlapping tones. Where the
tones are equally strong the aircraft is on course; an imbalance drives the cockpit
needle toward the stronger side.[^wiki] Because the guidance is encoded in the **depth of
two [amplitude-modulated](/reference/amplitude-modulation/) tones**, an ILS receiver only
has to compare tone strengths, not measure phase.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A localizer antenna radiating overlapping 90 hertz and 150 hertz lobes down a runway centerline, with an aircraft on course where the two tones are equal." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="120" width="360" height="16" fill="currentColor" fill-opacity="0.15" stroke="currentColor"/>
  <text x="220" y="132" text-anchor="middle" font-size="8" fill="currentColor">runway</text>
  <path d="M400 128 L60 90 L400 116 Z" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-opacity="0.5"/>
  <path d="M400 128 L60 138 L400 140 Z" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-opacity="0.5"/>
  <line x1="400" y1="122" x2="60" y2="114" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="120" y="80" font-size="8" fill="currentColor">150 Hz lobe</text>
  <text x="120" y="152" font-size="8" fill="currentColor">90 Hz lobe</text>
  <path d="M70 110 l-8 4 l8 4 z" fill="currentColor"/>
  <text x="80" y="106" font-size="8" fill="currentColor">on course: 90 Hz = 150 Hz</text>
</svg>
<figcaption>The ILS localizer overlaps a 90 Hz and a 150 Hz lobe along the centerline; equal tone depth means on-course, imbalance shows the deviation.</figcaption>
</figure>

## How it works

Both ILS beams work by **difference in depth of modulation (DDM)**. The localizer, near
110 MHz in the VHF band, radiates a carrier modulated on one side of the runway
centerline mostly by a 150 Hz tone and on the other side mostly by a 90 Hz tone; the two
patterns overlap along the centerline where their modulation depths are equal. The
receiver demodulates the AM envelope, measures how much 90 Hz versus 150 Hz is present,
and deflects the course-deviation indicator toward the tone that dominates. The
**glideslope**, near 330 MHz in the UHF band, does the same thing in the vertical plane
so the aircraft can hold a typical 3° descent.

Along the approach, **marker beacons** (or, increasingly, [DME](/reference/dme/) distance)
mark fixed points to the threshold. ILS installations are certified in categories (CAT I,
II, III) according to how low a decision height and visibility they support, with CAT III
enabling near-zero-visibility autoland. The localizer also carries a Morse identifier,
much like [VOR](/reference/vor/).

## Relevance to SDR

ILS is a compact demonstration of AM tone-depth signalling: a
[software-defined radio](/reference/software-defined-radio/) parked on a localizer
frequency can recover the composite audio, filter out the 90 Hz and 150 Hz tones, and
compute their relative depth to reproduce the cockpit needle. It shares the aeronautical
navigation bands with VOR and DME and rounds out the classic navaid set. **GopherTrunk**
does not decode ILS; it is a land-mobile trunking scanner, and ILS appears here to give
the AM-modulation family concrete real-world context.

## Sources

[^wiki]: [Instrument landing system](https://en.wikipedia.org/wiki/Instrument_landing_system) — Wikipedia, for the localizer/glideslope structure, the 90 Hz and 150 Hz DDM principle, marker beacons, and CAT I/II/III performance categories.
