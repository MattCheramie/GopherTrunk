---
slug: sma-adapter-kit
title: SMA adapter kit
entry_type: hardware
category: rf-front-end
description: "An SMA adapter kit bundles SMA-to-BNC, UHF, N, and F adapters in both genders — the cheap buy-if-unsure part that mates any antenna to an SDR's SMA jack for GopherTrunk."
keywords: SMA adapter kit, SMA to BNC, SMA to UHF, SMA to N, SMA to F, RF adapter kit, SDR adapter, connector kit, antenna adapter, gender changer
aka: [SMA adapter set, RF adapter kit, connector kit, SMA to everything kit]
autolink: true
affiliate: true
product:
  name: "SMA adapter kit (SMA to BNC/UHF/F/N, 16pc)"
  brand: Generic
  category: SMA connector adapter kit
  lowPrice: "11"
  highPrice: "15"
  url: https://www.amazon.com/dp/B07PXCC5G2?tag=gophertrunk-20
infobox:
  - { label: Type, value: "RF adapter assortment" }
  - { label: Covers, value: "SMA ↔ BNC, UHF, N, F" }
  - { label: Genders, value: "Male and female both ways" }
  - { label: Impedance, value: "50 Ω" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B07PXCC5G2?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [sma-connector, bnc-connector, n-type-connector, uhf-connector-pl259, coax-pigtail, coaxial-cable]
cite_urls:
  - https://en.wikipedia.org/wiki/SMA_connector
faq:
  - q: "Which SMA adapter kit should I buy for GopherTrunk?"
    a: "A 16-piece SMA adapter kit (around $13) is the single best 'buy if unsure' RF part. It carries SMA to and from BNC, UHF (PL-259/SO-239), N, and F in both genders, so almost any antenna, jumper, or piece of test gear you own can mate with the SMA jack on an SDR dongle. It costs about the same as two single adapters but never leaves you stuck on the wrong connector."
  - q: "Do I need a whole kit or just one adapter?"
    a: "If you know exactly what both ends are, a single adapter is enough and cheaper. Buy the kit when you are not sure what your antenna terminates in, or when you tinker with different gear — it covers BNC, UHF, N, and F in both directions for about the price of two singles, so you are never blocked mid-setup by a missing gender."
  - q: "Will an adapter kit hurt my signal?"
    a: "Each adapter adds a small insertion loss and a tiny reflection, so a clean run uses the fewest transitions possible. For an occasional connection a good adapter is transparent at the sub-GHz bands scanning uses; for a permanent install, put the correct connector on the cable rather than stacking three or four adapters."
  - q: "Does the kit include RP-SMA?"
    a: "Usually not — and that is a good thing for SDR work. SDRs and most antennas use standard SMA (center pin on the male side); RP-SMA is a reversed-pin Wi-Fi variant that will not mate properly. Buy plain SMA and check any antenna listing for the letters 'RP.'"
  - q: "What about connecting two SMA cables together?"
    a: "Most kits include an SMA-female-to-SMA-female barrel (a 'gender changer' or 'coupler') for exactly that, plus male-to-male versions. If you need a flexible extension rather than a rigid barrel, use an RG316 coax pigtail instead so the joint is not stressing the dongle's jack."
---

An **SMA adapter kit** is an inexpensive assortment of [coaxial](/reference/coaxial-cable/)
adapters that convert between [SMA](/reference/sma-connector/) — the near-universal
software-defined-radio antenna port — and the other connectors you meet in the wild:
[BNC](/reference/bnc-connector/), [UHF (PL-259/SO-239)](/reference/uhf-connector-pl259/),
[N-type](/reference/n-type-connector/), and the TV-style F. Because it includes both genders
of each transition, **it is the one RF part to buy if you are not sure what your antenna
terminates in** — a few dollars that guarantee you can always reach the SDR's SMA jack.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A central SMA jack fanning out to four adapters labelled BNC, UHF, N, and F, showing one kit bridging any antenna connector to an SDR's SMA port." xmlns="http://www.w3.org/2000/svg">
  <rect x="200" y="72" width="60" height="26" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.4"/>
  <text x="230" y="89" font-size="10" fill="currentColor" text-anchor="middle">SMA</text>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="200" y1="80" x2="90" y2="35"/>
    <line x1="200" y1="88" x2="90" y2="90"/>
    <line x1="260" y1="80" x2="370" y2="35"/>
    <line x1="260" y1="88" x2="370" y2="90"/>
    <line x1="230" y1="98" x2="230" y2="140"/>
  </g>
  <g font-size="10" fill="currentColor">
    <rect x="40" y="24" width="46" height="22" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="63" y="39" text-anchor="middle">BNC</text>
    <rect x="40" y="79" width="46" height="22" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="63" y="94" text-anchor="middle">UHF</text>
    <rect x="372" y="24" width="46" height="22" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="395" y="39" text-anchor="middle">N</text>
    <rect x="372" y="79" width="46" height="22" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="395" y="94" text-anchor="middle">F</text>
    <rect x="207" y="140" width="46" height="22" rx="3" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="230" y="155" text-anchor="middle">SDR</text>
  </g>
</svg>
<figcaption>One kit bridges an SMA dongle port to whatever your antenna, jumper, or coax uses — in either gender.</figcaption>
</figure>

## What it is

The kit is not a single clever part; it is a small box of brass barrels, each a fixed
transition between two connector families. A typical 16-piece set includes SMA-to-BNC,
SMA-to-UHF, SMA-to-N, and SMA-to-F adapters, and for each it supplies the pieces needed to
handle either gender on the far side — an SMA male on one side and an SMA female on another,
a BNC plug on one and a BNC jack on another. Because connector gender is the thing you most
often get wrong when buying blind, having every combination on hand is what turns "I need to
order the right adapter and wait" into "it is already in the drawer."

All the adapters are nominally **50 Ω**, matching the SDR, its antennas, and its coax. That
matters most at the top of the scanning range: below 1 GHz — which covers essentially all
VHF/UHF land-mobile [trunking](/reference/trunked-radio/) — a good adapter is electrically
almost invisible, so the kit costs you convenience-level loss, not real signal.

## Relevance to SDR

Almost every SDR that matters to a trunking listener — [RTL-SDR](/reference/rtl-sdr/) V3/V4,
Airspy, HackRF — presents an SMA jack, as do the bias-tee [LNAs](/reference/low-noise-amplifier/)
and [filters](/sdr-filters/) that sit in front of the radio. Antennas, meanwhile, are a
zoo: a wideband [discone](/reference/discone-antenna/) may use SO-239, a scanner whip a BNC,
a rooftop antenna an N. The adapter kit is the universal translator between the two worlds,
which is why it appears in nearly every [what-you-need](/what-do-i-need-for-gophertrunk/)
list for the price of a coffee.

GopherTrunk itself is decode software and never touches a connector — but a decode is only
as clean as the signal reaching the receiver, and a missing adapter is the most common
reason a perfectly good antenna is sitting unused in a box. The caveat that never changes:
no adapter, antenna, or SDR can decode [AES-encrypted](/police-scanner-encryption/) traffic;
good plumbing only ensures the *clear* traffic arrives with full signal-to-noise.

## Where to buy

Buy a **16-piece SMA adapter kit** (around $13) and keep it in the same drawer as your
dongle. It covers SMA to and from **BNC**, **UHF/PL-259**, **N**, and **F** in both genders
— so whatever antenna or feedline you bring home, you can already mate it to the SDR. It is
the cheapest insurance in the whole RF chain and pays for itself the first time you would
otherwise have waited two days for a single adapter.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B07PXCC5G2?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

When you want a short *flexible* lead rather than a rigid barrel, add an RG316
[coax pigtail](/reference/coax-pigtail/); for a real distance run, see
[coaxial cable](/reference/coaxial-cable/) and the
[SDR cables and connectors guide](/sdr-cables-and-connectors/).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra
cost to you. It never changes what we recommend.*

## Sources

[^wiki]: [SMA connector](https://en.wikipedia.org/wiki/SMA_connector) — Wikipedia, on the SMA interface these adapters convert to and from, its 50 Ω impedance, and the RP-SMA variant to avoid.
