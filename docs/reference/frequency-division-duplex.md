---
slug: frequency-division-duplex
title: Frequency-division duplex (FDD)
entry_type: concept
category: cellular
description: Frequency-division duplex carries uplink and downlink at the same time on two separate paired frequency bands kept apart by a duplex gap, the classic scheme for GSM, UMTS, and macro LTE.
keywords: FDD, frequency division duplex, paired spectrum, duplex gap, uplink downlink, duplexer, GSM, UMTS, LTE
aka: [FDD, Frequency-division duplex, paired spectrum]
autolink: true
infobox:
  - { label: Type, value: Duplexing scheme }
  - { label: Separation, value: By frequency (paired bands) }
  - { label: Traffic, value: Simultaneous both directions }
see_also: [time-division-duplex, duplexer, fdma, lte, frequency-bands, guard-band]
cite_urls:
  - https://en.wikipedia.org/wiki/Duplex_(telecommunications)
  - https://en.wikipedia.org/wiki/Frequency-division_duplexing
---

**Frequency-division duplex** (**FDD**) is a two-way radio scheme in which the uplink and
downlink run **simultaneously** on two **separate, paired frequency bands**.[^wiki] The two
bands are held apart by a *duplex gap* — a slice of unused spectrum — so a
[duplexer](/reference/duplexer/) can let a device transmit and receive at the same instant
without its own transmitter deafening its receiver. FDD is the classic arrangement for
GSM, UMTS, and most macro-cell [LTE](/reference/lte/) bands, and it stands in contrast to
[time-division duplex](/reference/time-division-duplex/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A frequency axis showing an uplink band on the left, a duplex gap in the middle, and a downlink band on the right, both bands active at the same time." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="110" x2="440" y2="110" stroke="currentColor" stroke-opacity="0.5"/>
  <text x="410" y="128" font-size="9" fill="currentColor">frequency →</text>
  <rect x="40" y="50" width="120" height="60" rx="3" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.3"/>
  <text x="100" y="44" text-anchor="middle" font-size="10" fill="currentColor">uplink band</text>
  <text x="100" y="84" text-anchor="middle" font-size="8" fill="currentColor">device → tower</text>
  <rect x="300" y="50" width="120" height="60" rx="3" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.3"/>
  <text x="360" y="44" text-anchor="middle" font-size="10" fill="currentColor">downlink band</text>
  <text x="360" y="84" text-anchor="middle" font-size="8" fill="currentColor">tower → device</text>
  <line x1="160" y1="70" x2="300" y2="70" stroke="currentColor" stroke-dasharray="4 3" stroke-opacity="0.6"/>
  <text x="230" y="64" text-anchor="middle" font-size="9" fill="currentColor">duplex gap</text>
  <text x="230" y="136" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.85">both bands carry traffic at the same time</text>
</svg>
<figcaption>FDD splits uplink and downlink onto two paired bands separated by a duplex gap; both are active simultaneously, so the link is continuous in each direction.</figcaption>
</figure>

## How it works

An FDD deployment allocates two blocks of spectrum a fixed distance apart. The lower block
usually carries the uplink (device to tower) and the upper the downlink (tower to device),
with the *duplex spacing* between their centre frequencies fixed by the band plan. Because
transmit and receive sit on different frequencies, the device's [duplexer](/reference/duplexer/)
— a pair of sharp filters — passes the receive band to the receiver while blocking the
device's own transmit energy from swamping it. The duplex gap gives those filters room to
roll off; a [guard band](/reference/guard-band/) at the edge of each block keeps adjacent
operators from interfering.

The defining property is **continuity**: since each direction owns its own frequency full
time, data flows both ways without interruption and with low, constant latency. There is no
switching overhead and no need to reserve time for turning the link around. The cost is
**paired spectrum** — a regulator must hand out two matched blocks, and the duplex gap
between them cannot be used for traffic, so some spectrum is spent purely on separation.
FDD is naturally symmetric: uplink and downlink get equal bandwidth whether or not the
traffic is balanced.

## Band plans and paired spectrum

Cellular [frequency bands](/reference/frequency-bands/) are numbered, and each FDD band
specifies both halves of the pair plus the spacing between them — for example, LTE Band 1
pairs a 1920–1980 MHz uplink with a 2110–2170 MHz downlink, 190 MHz apart. Multiple users
share each band through [FDMA](/reference/fdma/) and other multiplexing on top of the
duplex split. Because download traffic now dominates, the rigid symmetry of FDD is
sometimes a poor fit for data-heavy use, which is one reason newer mid-band allocations
lean on time-division duplex instead. Established FDD bands remain the backbone of wide-area
macro coverage, where their simplicity, range, and predictable latency are assets.

## Relevance to SDR

FDD is why a cellular signal you find on an SDR spectrum display appears as two mirror
blocks of activity separated by a quiet gap: the busy tower downlink is the one an
uncoordinated receiver hears strongly, while the uplink sits in its own band far below the
noise unless a handset is nearby. Recognising the paired structure — and the fixed duplex
spacing of each band — helps identify which cellular technology and band a signal belongs
to. GopherTrunk targets land-mobile trunking rather than cellular, but the same duplexing
concepts describe how trunked repeaters separate their inbound and outbound paths, so FDD
is useful background for reading any two-way RF spectrum.

## Sources

[^wiki]: [Frequency-division duplexing](https://en.wikipedia.org/wiki/Frequency-division_duplexing) — Wikipedia, for the definition of FDD, paired bands, and the duplex gap.
[^duplex]: [Duplex (telecommunications)](https://en.wikipedia.org/wiki/Duplex_(telecommunications)) — Wikipedia, for full-duplex operation and the contrast between frequency- and time-division duplexing.
