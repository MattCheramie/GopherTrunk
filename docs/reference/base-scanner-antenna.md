---
slug: base-scanner-antenna
title: Base scanner antenna
entry_type: hardware
category: antennas
description: "A base scanner antenna is an outdoor rooftop or mast-mounted antenna for a fixed scanning station; a wideband discone is the usual one-antenna answer, mounted high with a clear horizon."
keywords: base scanner antenna, outdoor scanner antenna, rooftop scanner antenna, discone, wideband base antenna, fixed station antenna, VHF UHF base antenna, scanner mast antenna
aka: [outdoor scanner antenna, rooftop scanner antenna, fixed base antenna]
autolink: true
affiliate: true
product:
  name: "Tram 1411 wideband super discone base scanner antenna (25–1300 MHz)"
  brand: Tram
  category: Outdoor wideband base scanner antenna
  lowPrice: "49"
  highPrice: "69"
  url: https://www.amazon.com/dp/B00QVNI1V0?tag=gophertrunk-20
infobox:
  - { label: Type, value: Wideband vertical (discone) }
  - { label: Coverage, value: ~25–1300 MHz }
  - { label: Pattern, value: Omnidirectional, vertical }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B00QVNI1V0?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [discone-antenna, ground-plane-antenna, collinear-antenna, whip-antenna, mobile-scanner-antenna, polarization]
cite_urls:
  - https://en.wikipedia.org/wiki/Discone_antenna
  - https://en.wikipedia.org/wiki/Ground_plane
faq:
  - q: "What is the best outdoor base antenna for a scanner?"
    a: "For one antenna that covers everything, a wideband discone like the Tram 1411 (around $59) is the usual answer: 25–1300 MHz from a single feedline, omnidirectional, and vertically polarized to match land-mobile P25/DMR/NXDN traffic. Mount it as high as you can with a clear horizon. If you instead focus on one VHF or UHF band and want more range, a gain vertical or collinear beats a discone there; if you chase one weak distant site, add a directional Yagi."
  - q: "Discone, vertical, or collinear for a base station?"
    a: "It is a bandwidth-versus-gain choice. A discone covers a huge frequency range at modest (roughly unity) gain — best when you monitor many bands at once. A band-cut vertical or collinear gives more gain but only across its design band — best when you concentrate on one region's VHF or UHF trunking. Most listeners start with a discone for its do-everything breadth, then add a band-specific antenna if one system needs more signal."
  - q: "How high should a base scanner antenna be mounted?"
    a: "As high and clear as practical — height and an unobstructed horizon matter far more than the antenna model. A vertical radiates toward the horizon, so a rooftop or mast mount above the roofline, clear of metal and obstructions, is worth more than any upgrade in antenna type. Keep the coax run short and low-loss, and use proper low-loss cable for a long masthead run."
  - q: "Do I need to ground an outdoor scanner antenna?"
    a: "Yes, for safety. An outdoor mast and coax should be bonded to your home's grounding system and ideally fitted with a coax surge/lightning arrestor, per your local electrical code. This protects the receiver and the house from static buildup and nearby strikes. Weatherproof the connectors, too — water in coax raises loss dramatically."
---

A **base scanner antenna** is the outdoor, fixed-station antenna for a home scanning
setup — mounted on a rooftop, mast, or eave, and fed indoors by [coax](/reference/coaxial-cable/)
to a [scanner](/reference/trunking-scanner/) or [SDR](/reference/software-defined-radio/).
It is the single biggest upgrade available to a fixed listener, because height and a clear
horizon do more for reception than any change of receiver. This page is the "one outdoor
antenna" buyer's pick; the individual antenna types — [discone](/reference/discone-antenna/),
[ground-plane](/reference/ground-plane-antenna/) vertical, and
[collinear](/reference/collinear-antenna/) — have their own detailed pages.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="A house roofline with a mast rising above it carrying a wideband vertical antenna, a coax cable running down the mast into the house, and an arrow showing the clear horizon the antenna needs." xmlns="http://www.w3.org/2000/svg">
  <path d="M60 170 L160 110 L260 170 Z" fill="none" stroke="currentColor" stroke-width="2"/>
  <line x1="60" y1="170" x2="420" y2="170" stroke="currentColor" stroke-width="2"/>
  <line x1="160" y1="110" x2="160" y2="40" stroke="currentColor" stroke-width="2"/>
  <text x="120" y="150" font-size="9" fill="currentColor">mast</text>
  <line x1="160" y1="40" x2="160" y2="15" stroke="currentColor" stroke-width="3"/>
  <line x1="148" y1="52" x2="172" y2="52" stroke="currentColor" stroke-width="2"/>
  <line x1="150" y1="60" x2="170" y2="60" stroke="currentColor" stroke-width="2"/>
  <text x="176" y="30" font-size="10" fill="currentColor">wideband vertical</text>
  <path d="M160 52 C 150 80, 150 120, 158 168" fill="none" stroke="currentColor" stroke-width="1.4" stroke-opacity="0.6"/>
  <line x1="180" y1="30" x2="290" y2="30" stroke="currentColor" stroke-width="1.5" stroke-dasharray="4 3"/>
  <text x="295" y="34" font-size="9" fill="currentColor">clear horizon</text>
</svg>
<figcaption>A base antenna's job is height and a clear horizon; a wideband vertical on a mast above the roofline is the usual one-antenna answer for a fixed scanning station.</figcaption>
</figure>

## Choosing one outdoor antenna

Nearly every fixed scanning station comes down to one trade-off: **bandwidth versus
gain.**

- **Wideband discone** — a [discone](/reference/discone-antenna/) covers a decade of
  frequency (roughly 25–1300 MHz) from one feedline at modest, near-unity gain. It hears
  aviation, marine, public-safety trunking, and business bands without swapping hardware,
  which makes it the natural "do everything" scanner antenna and the default recommendation
  for a first outdoor install.
- **Band-cut vertical or collinear** — a [ground-plane](/reference/ground-plane-antenna/)
  vertical or a [collinear](/reference/collinear-antenna/) gives more gain, but only across
  its design band. If you concentrate on one region's VHF or UHF trunking, one of these
  hears weak sites a discone would miss.
- **Directional Yagi or LPDA** — a [Yagi](/reference/yagi-uda-antenna/) or
  [log-periodic](/reference/log-periodic-antenna/) adds real forward gain and rejects
  interference, but only in the direction you aim it. It is an add-on for one stubborn
  distant site, not a general base antenna.

For most people the honest first buy is a **wideband discone**: it maximizes the number of
systems you can hear from a single antenna, and you can always add a band-specific or
directional antenna later if one system needs more signal.

## How it works

A base antenna is almost always a vertical, omnidirectional radiator, vertically
[polarized](/reference/polarization/) to match land-mobile signals and radiating toward the
horizon with a null overhead. The discone achieves its enormous
[bandwidth](/reference/bandwidth/) by being a **frequency-independent** shape — a disc over
a flaring cone whose continuously changing dimensions keep the feedpoint near 50 Ω across a
wide range, so [SWR](/reference/standing-wave-ratio/) stays low without retuning. A
ground-plane or collinear, by contrast, is a resonant design tuned to one band, trading that
breadth for gain.

Whatever the type, the install dominates the result. Mount the antenna **above the
roofline** on a [mast](/reference/antenna-mast/) or [eave mount](/reference/eave-mount/),
clear of metal and obstructions, and keep the feedline short and low-loss — a long run of
thin coax can throw away several dB at UHF and undo the whole upgrade. Bond the mast and add
a coax surge arrestor with a proper [grounding kit](/reference/grounding-kit/) for safety,
and weatherproof every outdoor connector.

## Relevance to GopherTrunk

For a fixed GopherTrunk station, the base antenna is usually the single most valuable piece
of hardware after the SDR itself. GT decodes whatever the front end delivers and cares only
about signal-to-noise, and an outdoor vertical mounted high with a clear horizon delivers a
far stronger, cleaner signal than any indoor whip — improving lock on weak
[control channels](/reference/control-channel/) across every band GopherTrunk supports. A
wideband discone in particular suits GT's multi-protocol, multi-band nature: leave it
connected while monitoring P25, DMR, NXDN, and TETRA systems scattered across the spectrum.
No antenna, however, defeats [encryption](/police-scanner-encryption/) — a base antenna
improves reception, never decryption.

## Where to buy

For one outdoor antenna that covers everything, a wideband **Tram 1411 super discone**
(25–1300 MHz, around $59) is the usual answer: a single feedline for VHF, UHF, and beyond,
omnidirectional so it hears every [trunking site](/reference/trunking-site/) at once, and
vertically polarized to match land-mobile traffic. Mount it as high as you can with a clear
horizon and keep the coax short.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B00QVNI1V0?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

If you focus on one band and want more range, compare a
[ground-plane](/reference/ground-plane-antenna/) vertical or a
[collinear](/reference/collinear-antenna/); for a stubborn distant site, add a
[Yagi](/reference/yagi-uda-antenna/). The full line-up is in the
[best scanner antenna guide](/best-scanner-antenna/) and
[best SDR antenna guide](/best-sdr-antenna/).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra
cost to you. It never changes what we recommend.*

## Sources

[^wiki]: [Discone antenna](https://en.wikipedia.org/wiki/Discone_antenna) — Wikipedia, for the wideband disc-over-cone geometry that makes a discone the default outdoor scanner antenna.
