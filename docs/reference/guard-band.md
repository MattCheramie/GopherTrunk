---
slug: guard-band
title: Guard band
entry_type: term
category: rf-fundamentals
description: A guard band is an unused slice of spectrum left between adjacent channels or services to keep their emissions from spilling over and causing adjacent-channel interference.
keywords: guard band, guardband, adjacent channel interference, channel spacing, spectral separation, unused spectrum, roll-off, occupied bandwidth, ACI
aka: [guard band, guardband]
autolink: true
infobox:
  - { label: Type, value: Spectral separation }
  - { label: Purpose, value: Prevent adjacent-channel interference }
  - { label: Cost, value: Spectrum left unused }
see_also: [occupied-bandwidth, bandwidth, spectral-efficiency, spurious-emissions, roll-off-factor]
cite_urls:
  - https://en.wikipedia.org/wiki/Guard_band
---

A **guard band** is a deliberately unused strip of spectrum placed between two adjacent channels,
blocks, or services so that energy from one does not leak into the next.[^wiki] It is insurance
against imperfect filters, finite [roll-off](/reference/roll-off-factor/), oscillator drift, and
[spurious emissions](/reference/spurious-emissions/): by separating the
[occupied bandwidth](/reference/occupied-bandwidth/) of neighbours with a small vacant margin, a
guard band trades a little spectral efficiency for a large reduction in adjacent-channel
interference (ACI).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Two channel spectral masks sit side by side; the empty gap between the edge of one channel and the start of the next is labelled the guard band, and the channel spacing spans from centre to centre." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor" stroke="none">
    <line x1="30" y1="115" x2="440" y2="115" stroke="currentColor" stroke-opacity="0.4"/>
    <path d="M60 115 L90 40 L150 40 L180 115" fill="none" stroke="currentColor"/>
    <path d="M280 115 L310 40 L370 40 L400 115" fill="none" stroke="currentColor"/>
    <text x="95" y="55">Ch A</text>
    <text x="315" y="55">Ch B</text>
    <rect x="180" y="95" width="100" height="20" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-dasharray="3 2"/>
    <text x="188" y="109">guard band</text>
    <line x1="120" y1="128" x2="340" y2="128" stroke="currentColor" stroke-opacity="0.6"/>
    <text x="185" y="142">channel spacing</text>
  </g>
</svg>
<figcaption>A guard band is the vacant gap between the occupied bandwidths of adjacent channels; channel spacing equals occupied bandwidth plus guard band.</figcaption>
</figure>

## How it works

Every real transmitter's spectrum has skirts — its power does not stop abruptly at the channel edge
but rolls off over a finite transition, and small amounts of energy appear beyond the nominal band.
A neighbouring receiver, meanwhile, cannot filter with infinitely steep sides, so it inevitably
admits some energy from just outside its channel. The guard band exists so that where one
transmitter's residual skirt overlaps the next receiver's filter skirt, both are already far down —
turning what would be interference into negligible noise.

Concretely, **channel spacing = occupied bandwidth + guard band**. A regulator or standard picks a
spacing that comfortably exceeds the emission's occupied bandwidth; the surplus is the guard band. A
signal with sharp [pulse shaping](/reference/pulse-shaping/) and a small
[roll-off factor](/reference/roll-off-factor/) needs less guard band because its skirts are steeper;
a sloppy or drifting emitter needs more. The guard band therefore couples directly to
[spectral efficiency](/reference/spectral-efficiency/): wider guards waste spectrum, tighter guards
pack more channels but demand cleaner transmitters and better receiver selectivity.

## In practice

Guard bands appear at every scale of the spectrum. Land-mobile channel plans leave a guard between
each 12.5 kHz or 25 kHz channel; broadcast FM and TV allocations space stations so their masks do not
collide; cellular and Wi-Fi blocks reserve guard bands at the edges of each operator's allocation and
between the block and its neighbours. In multicarrier systems like [OFDM](/reference/ofdm/), unused
edge subcarriers act as an internal guard band, and a **guard interval** (cyclic prefix) plays the
analogous role in the *time* domain against inter-symbol interference. Whenever two services must
coexist without coordination — such as a licensed band beside an unlicensed one — the guard band is
what keeps them from stepping on each other.

## Relevance to SDR

For a receive-only [SDR](/reference/software-defined-radio/), guard bands are why adjacent channels
are separable at all: the vacant margin lets a channel filter isolate the wanted signal without
dragging in the neighbour. When guard bands are thin and a nearby channel is much stronger, ACI can
still bleed through, especially if the front end is not linear enough — a case where a narrower
receive filter, more selectivity, or an [attenuator](/reference/attenuator/) to reduce the strong
neighbour helps. Knowing the channel plan's spacing and guard band tells you how tightly you can tune
and filter before a neighbour intrudes.

**GopherTrunk** channelizes wideband captures into per-channel streams, and adequate guard bands in
the source system's plan are part of why those channels can be cleanly split. GopherTrunk does not
create guard bands — they are a property of the transmitted spectrum — but its channelizer relies on
them existing.

## Sources

[^wiki]: [Guard band](https://en.wikipedia.org/wiki/Guard_band) — Wikipedia, definition of the unused spectrum between channels and its role in preventing interference.
