---
slug: log-periodic-antenna
title: Log-periodic antenna (LPDA)
entry_type: term
category: antennas
description: A log-periodic dipole array is a wideband directional antenna whose scaled, alternately-fed elements keep constant gain across a decade or more of frequency.
keywords: log-periodic antenna, LPDA, log periodic dipole array, wideband directional antenna, frequency independent antenna, scaled elements, active region
aka: [log-periodic dipole array, LPDA, log periodic]
autolink: true
affiliate: true
product:
  name: "Antenna World COM-072708LP wideband log-periodic antenna (700–2700 MHz, 9 dBi)"
  brand: Antenna World
  category: Wideband log-periodic (LPDA) directional antenna
  lowPrice: "39"
  highPrice: "55"
  url: https://www.amazon.com/dp/B01N2TJB87?tag=gophertrunk-20
infobox:
  - { label: Type, value: Wideband directional array }
  - { label: Elements, value: Scaled dipoles, alternating feed }
  - { label: Pattern, value: Unidirectional, near-constant gain }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B01N2TJB87?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [yagi-uda-antenna, antenna-gain, radiation-pattern, dipole-antenna, collinear-antenna, bandwidth]
cite_urls:
  - https://en.wikipedia.org/wiki/Log-periodic_antenna
  - https://en.wikipedia.org/wiki/Frequency-independent_antenna
faq:
  - q: "Which log-periodic antenna should I buy for wideband scanning?"
    a: "A wideband LPDA such as the Antenna World COM-072708LP (700–2700 MHz, ~9 dBi, around $45) gives directional gain and rear rejection across many bands from one feedpoint — useful for aiming at a distant 700/800 MHz P25 or trunking site without retuning or swapping antennas. Sold for cellular use, these panels double as scanner beams; the higher UHF bands they cover are exactly where P25 Phase 1/2 and much trunking lives."
  - q: "Log-periodic or Yagi for a distant site?"
    a: "A Yagi gives more gain on the one frequency it is cut for; an LPDA gives a bit less gain but holds that gain and pattern across a huge band. If you chase a single known channel, a Yagi wins; if you want one directional antenna that stays useful from UHF up through 800 MHz as you move between systems, the log-periodic is the practical choice."
  - q: "Will a 700–2700 MHz cellular LPDA work for VHF scanning?"
    a: "No — those panels start around 700 MHz, so they cover UHF and 800/900 MHz trunking but not the 136–174 MHz VHF band. If you need VHF directivity, use a VHF/UHF Yagi instead. Match the antenna's band to the systems you actually monitor."
  - q: "How do I mount and aim a log-periodic?"
    a: "Fix it to a mast with the elements vertical (to match land-mobile polarization) and point the short front end of the boom at the site — an LPDA fires off its shortest elements. Keep the coax short and low-loss, especially at 800 MHz where cable loss is high."
---

A **log-periodic antenna**, most often a **log-periodic dipole array (LPDA)**, is a
directional [antenna](/reference/antenna/) whose electrical properties repeat
periodically with the *logarithm* of frequency, giving it nearly constant
[gain](/reference/antenna-gain/) and [pattern](/reference/radiation-pattern/) across a
very wide band — often a decade (10:1) or more.[^wiki] It looks like a Yagi with many
elements, but every [dipole](/reference/dipole-antenna/) is connected to the feedline in
alternating phase, and the elements grow in a fixed geometric ratio along the boom.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A log-periodic array of dipoles increasing in length along a crossed twin-boom feedline, with the beam directed off the short front end and an active region of resonant elements highlighted in the middle." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="lpar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="60" y1="76" x2="410" y2="76" stroke="currentColor" stroke-width="1.5"/>
  <line x1="60" y1="84" x2="410" y2="84" stroke="currentColor" stroke-width="1.5"/>
  <line x1="80" y1="66" x2="80" y2="94" stroke="currentColor" stroke-width="2.5"/>
  <line x1="120" y1="60" x2="120" y2="100" stroke="currentColor" stroke-width="2.5"/>
  <line x1="165" y1="52" x2="165" y2="108" stroke="currentColor" stroke-width="2.5"/>
  <line x1="215" y1="44" x2="215" y2="116" stroke="currentColor" stroke-width="2.5"/>
  <line x1="270" y1="36" x2="270" y2="124" stroke="currentColor" stroke-width="2.5"/>
  <line x1="330" y1="28" x2="330" y2="132" stroke="currentColor" stroke-width="2.5"/>
  <ellipse cx="165" cy="80" rx="35" ry="34" fill="none" stroke="currentColor" stroke-opacity="0.4" stroke-dasharray="3 3"/>
  <text x="132" y="150" font-size="9" fill="currentColor">active region</text>
  <line x1="70" y1="80" x2="30" y2="80" stroke="currentColor" stroke-width="2" marker-end="url(#lpar)"/>
  <text x="18" y="70" font-size="9" fill="currentColor">beam</text>
  <text x="350" y="150" font-size="9" fill="currentColor">longest = lowest f</text>
</svg>
<figcaption>In an LPDA only the dipoles near resonance (the active region) radiate; that region shifts along the scaled array as frequency changes, keeping gain constant.</figcaption>
</figure>

## How it works

The design is defined by two constants: a scale factor **τ** (each element's length and
spacing is τ times the next) and a spacing factor **σ**. Because the geometry scales
logarithmically, the array looks electrically the same at frequencies separated by
factors of τ — hence "log-periodic." At any given frequency only a handful of elements —
those close to a half wavelength — are actually resonant. This **active region** does the
radiating; longer elements behind it act like reflectors and shorter ones ahead act like
directors, so locally the array behaves like a small [Yagi](/reference/yagi-uda-antenna/).

The key trick is the **alternating (transposed) feed**: adjacent dipoles connect to
opposite sides of the twin-boom feedline. This 180° flip, combined with the physical
spacing, gives the active-region elements the phase relationship needed to fire the beam
toward the *short* end of the array. As frequency rises or falls, the active region simply
slides to a different set of elements, but its size and behaviour stay the same — so gain,
beamwidth, and [SWR](/reference/standing-wave-ratio/) hold steady across the whole design
band.

The trade-off against a Yagi is gain for [bandwidth](/reference/bandwidth/). Because only
a few elements are ever active, an LPDA of a given boom length has modest gain (typically
6–9 dBi) compared with a Yagi that dedicates all its elements to one frequency. What it
buys is enormous frequency coverage from a single feedpoint with a stable pattern and
match.

## Relevance to SDR

Log-periodics are the natural directional antenna for **wideband scanning and
surveillance**. A single LPDA can cover, say, 400–1000 MHz — spanning multiple public-
safety and trunking bands — while giving the forward gain and rear rejection of a beam.
For an SDR user who wants directivity toward a distant [trunking site](/reference/trunking-site/)
but does not want to retune or swap antennas for every band, an LPDA is the practical
choice.

GopherTrunk is a receive-only decoder and does not steer or select elements; the antenna's
directivity is fixed by where the boom points, exactly as with a Yagi. Where an LPDA wins
is that its broadband nature keeps the same beam usable across the many bands GT can
decode, from low-VHF LTR up through 800 MHz P25. For maximum gain on one known frequency a
[Yagi](/reference/yagi-uda-antenna/) still beats it; for coverage, the log-periodic wins.

## Where to buy

For directional gain that stays usable across many bands, a wideband **LPDA like the
Antenna World COM-072708LP** (700–2700 MHz, ~9 dBi, around $45) points at a distant
UHF/700/800 MHz [trunking site](/reference/trunking-site/) and keeps its beam as you move
between systems — no retuning, no antenna swap. Sold as a cellular panel, it doubles as a
scanner beam across exactly the bands where P25 and 800 MHz trunking live. Mount the
elements vertical and aim the short end of the boom at the tower.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B01N2TJB87?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

Note the band: these panels do not cover 136–174 MHz VHF — use a
[Yagi](/reference/yagi-uda-antenna/) there. And directivity fixes weak signal, not
[encryption](/police-scanner-encryption/). See the
[best scanner antenna guide](/best-scanner-antenna/) for the full antenna line-up.

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra
cost to you. It never changes what we recommend.*

## Sources

[^wiki]: [Log-periodic antenna](https://en.wikipedia.org/wiki/Log-periodic_antenna) — Wikipedia, for the scaled-element geometry, alternating feed, active region, and the wideband constant-gain behaviour.
