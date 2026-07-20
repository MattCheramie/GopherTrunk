---
slug: time-division-duplex
title: Time-division duplex (TDD)
entry_type: concept
category: cellular
description: Time-division duplex shares one frequency for uplink and downlink by alternating them in time slots separated by a guard period, needing no paired spectrum and allowing an asymmetric split, used by TD-LTE and much of 5G NR.
keywords: TDD, time division duplex, unpaired spectrum, guard period, uplink downlink, TD-LTE, 5G NR, asymmetric traffic
aka: [TDD, Time-division duplex]
autolink: true
infobox:
  - { label: Type, value: Duplexing scheme }
  - { label: Separation, value: By time (one shared band) }
  - { label: Traffic, value: Alternating, splittable ratio }
see_also: [frequency-division-duplex, tdma, lte, 5g-nr, guard-band]
cite_urls:
  - https://en.wikipedia.org/wiki/Time-division_duplex
  - https://en.wikipedia.org/wiki/Duplex_(telecommunications)
---

**Time-division duplex** (**TDD**) is a two-way radio scheme in which the uplink and
downlink **share one frequency**, taking turns in **time slots** separated by a short
*guard period*.[^wiki] Because both directions use the same channel, TDD needs no paired
spectrum, and the split between uplink and downlink time can be tuned to match asymmetric,
download-heavy traffic. It powers TD-LTE and much of [5G NR](/reference/5g-nr/) in the
mid-band and mmWave ranges, and it stands in contrast to
[frequency-division duplex](/reference/frequency-division-duplex/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A single frequency shown over a time axis, alternating downlink and uplink slots with a small guard gap between each transmission." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="120" x2="440" y2="120" stroke="currentColor" stroke-opacity="0.5"/>
  <text x="405" y="138" font-size="9" fill="currentColor">time →</text>
  <text x="20" y="40" font-size="9" fill="currentColor">one frequency</text>
  <g stroke="currentColor" stroke-width="1.3">
    <rect x="40" y="50" width="90" height="60" rx="3" fill="currentColor" fill-opacity="0.22"/>
    <rect x="150" y="50" width="40" height="60" rx="3" fill="currentColor" fill-opacity="0.08"/>
    <rect x="210" y="50" width="90" height="60" rx="3" fill="currentColor" fill-opacity="0.22"/>
    <rect x="320" y="50" width="40" height="60" rx="3" fill="currentColor" fill-opacity="0.08"/>
    <rect x="380" y="50" width="55" height="60" rx="3" fill="currentColor" fill-opacity="0.22"/>
  </g>
  <text x="85" y="84" text-anchor="middle" font-size="9" fill="currentColor">DL</text>
  <text x="170" y="84" text-anchor="middle" font-size="7" fill="currentColor">UL</text>
  <text x="255" y="84" text-anchor="middle" font-size="9" fill="currentColor">DL</text>
  <text x="340" y="84" text-anchor="middle" font-size="7" fill="currentColor">UL</text>
  <text x="407" y="84" text-anchor="middle" font-size="9" fill="currentColor">DL</text>
  <text x="170" y="44" text-anchor="middle" font-size="7" fill="currentColor">guard</text>
</svg>
<figcaption>TDD keeps a single frequency and alternates downlink and uplink in time; a small guard period between them lets one side stop before the other starts. The DL/UL ratio can be skewed toward download.</figcaption>
</figure>

## How it works

A TDD frame is divided into slots that are assigned to the downlink or the uplink on a
repeating schedule. At any instant only one direction is transmitting on the channel, so
transmit and receive never overlap and no [duplexer](/reference/duplexer/) filter pair is
required — the radio simply switches between transmitting and receiving. Between a downlink
burst and the following uplink burst the network inserts a **guard period**: a brief silence
that lets the previous transmission's signal clear the air (accounting for propagation delay
across the cell) before the other end begins, preventing the two directions from colliding.
This is the same time-sharing idea as [TDMA](/reference/tdma/) applied to the duplex
direction rather than to separating users.

The two big advantages are **unpaired spectrum** and an **adjustable split**. A regulator
can allocate a single contiguous block, and the operator decides how many slots go to
download versus upload — a natural fit for modern traffic, which is heavily download-biased.
The costs are **tight synchronization** (every cell must agree on the frame timing, or one
cell's uplink will clash with a neighbour's downlink) and the **guard time** itself, which is
spectrum-efficiency lost to the turnaround and grows with cell size.

## TDD in cellular

TD-LTE (the TDD variant of [LTE](/reference/lte/)) and the majority of
[5G NR](/reference/5g-nr/) mid-band and millimetre-wave deployments use TDD, because the
prime new spectrum for those systems came in large unpaired blocks and because massive-MIMO
beamforming benefits from channel *reciprocity* — the uplink and downlink share the same
frequency, so the base station can infer the downlink channel from what it measured on the
uplink. The [guard band](/reference/guard-band/) at the block edges still separates operators
in frequency, while the guard period separates the directions in time. The trade-off against
FDD is fundamental: TDD wins flexibility and unpaired-spectrum use, FDD wins continuous
low-latency symmetric links.

## Relevance to SDR

On a spectrum display a TDD signal looks different from an FDD pair: instead of two mirrored
bands, a single block pulses on and off as the frame alternates direction, and a waterfall
shows the busy downlink slots interleaved with quieter uplink ones on the *same* centre
frequency. Recognising that on/off, single-band pattern is the quickest way to tell TDD from
FDD when surveying spectrum. GopherTrunk decodes land-mobile trunking rather than cellular,
but the time-sharing principle behind TDD is the same one that lets a single trunked channel
carry alternating inbound and outbound traffic, so it is useful context for reading two-way RF.

## Sources

[^wiki]: [Time-division duplex](https://en.wikipedia.org/wiki/Time-division_duplex) — Wikipedia, for the definition of TDD, the guard period, and the adjustable uplink/downlink split.
[^duplex]: [Duplex (telecommunications)](https://en.wikipedia.org/wiki/Duplex_(telecommunications)) — Wikipedia, for full-duplex operation and the contrast between time- and frequency-division duplexing.
