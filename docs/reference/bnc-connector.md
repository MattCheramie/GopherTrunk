---
slug: bnc-connector
title: BNC connector
entry_type: hardware
category: rf-front-end
description: "BNC is a quick-connect bayonet coaxial connector for RF and video up to about 4 GHz, common on scanners, test gear, and older SDR front ends."
keywords: BNC connector, bayonet Neill-Concelman, 50 ohm, 75 ohm, bayonet coupling, coaxial connector, scanner antenna port, oscilloscope
aka: [BNC, "Bayonet Neill-Concelman"]
autolink: true
affiliate: true
product:
  name: "SMA to BNC adapter kit (8-piece, both genders)"
  brand: Generic
  category: BNC-to-SMA coaxial adapter kit
  lowPrice: "8"
  highPrice: "13"
  url: https://www.amazon.com/dp/B078K5563Y?tag=gophertrunk-20
infobox:
  - { label: Type, value: "Bayonet coaxial connector" }
  - { label: Impedance, value: "50 Ω or 75 Ω" }
  - { label: Range, value: "DC to ~4 GHz" }
  - { label: Coupling, value: "Quarter-turn bayonet" }
  - { label: TX, value: "Yes (low power)" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B078K5563Y?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [sma-connector, n-type-connector, uhf-connector-pl259, coaxial-cable, sma-adapter-kit, coax-pigtail]
cite_urls:
  - https://en.wikipedia.org/wiki/BNC_connector
faq:
  - q: "Which BNC-to-SMA adapter should I buy for an SDR?"
    a: "A small SMA-to-BNC adapter kit (8 pieces, both genders, around $10) is the safe pick because it covers every direction — BNC male or female to SMA male or female — so a BNC scanner antenna, jumper, or piece of test gear can reach the SMA jack on your dongle no matter which way the pins run. If you already know the exact genders, a single BNC-female-to-SMA-male adapter is a couple of dollars."
  - q: "Does a BNC-to-SMA adapter lose signal?"
    a: "Barely, at the frequencies scanning uses. Each adapter adds a small insertion loss and a tiny reflection, but at the VHF/UHF land-mobile bands where P25, DMR, and NXDN live — well under 1 GHz — a good BNC joint is essentially transparent. Stacking several cheap adapters on a long thin cable is what actually costs signal; one clean adapter does not."
  - q: "Is BNC 50 Ω or 75 Ω, and does it matter for scanning?"
    a: "Both exist and look identical. Buy the 50 Ω version for radio work — it matches your SDR and antennas. A 75 Ω BNC (from video/CCTV gear) mates mechanically and is usually harmless at VHF, but raises reflections as frequency climbs, so keep 75 Ω parts out of a UHF trunking feedline."
  - q: "BNC or SMA for a scanner antenna?"
    a: "BNC is the traditional quick-connect port on handheld scanners and wideband whips; SMA is the near-universal SDR port. Neither is better electrically below 1 GHz — you simply adapt whichever your antenna uses to the SMA jack on the dongle with one cheap adapter."
---

**BNC** (Bayonet Neill-Concelman) is a [coaxial](/reference/coaxial-cable/) connector with a
**bayonet** coupling: you push it on and give a quarter turn to lock, rather than screwing a
thread.[^wiki] That fast, tactile mate makes it the classic port on scanners, oscilloscopes,
signal generators, and older radio gear, usable to roughly **4 GHz**. It comes in both
**50 Ω** and **75 Ω** versions that look identical but should not be freely mixed.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A BNC plug with two bayonet pins engaging the slots of a jack's coupling ring, locked with a quarter turn." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="75" x2="120" y2="75" stroke="currentColor" stroke-width="2"/>
  <g stroke="currentColor" stroke-width="1.4" fill="none">
    <rect x="120" y="50" width="70" height="50"/>
    <circle cx="185" cy="60" r="4" fill="currentColor"/>
    <circle cx="185" cy="90" r="4" fill="currentColor"/>
  </g>
  <circle cx="205" cy="75" r="4" fill="currentColor"/>
  <line x1="209" y1="75" x2="235" y2="75" stroke="currentColor" stroke-width="2"/>
  <g stroke="currentColor" stroke-width="1.4" fill="none">
    <rect x="235" y="45" width="70" height="60"/>
    <path d="M305 55 L318 55 L318 68" />
    <path d="M305 95 L318 95 L318 82" />
  </g>
  <line x1="305" y1="75" x2="430" y2="75" stroke="currentColor" stroke-width="2"/>
  <g font-size="9" fill="currentColor"><text x="155" y="125" text-anchor="middle">plug (bayonet pins)</text><text x="270" y="128" text-anchor="middle">jack (slots)</text></g>
</svg>
<figcaption>A BNC plug's two pins slide into the jack's slotted ring; a quarter turn cams them tight for a quick, secure lock.</figcaption>
</figure>

## Overview

BNC descended from the earlier N and C connectors — the "N-C" in the name honours engineers
Paul Neill and Carl Concelman — and shrank their idea into a small bayonet fitting in the
1950s. The coupling uses two pins on the plug that ride into slotted cams on the jack, so a
quarter turn draws the contacts together and holds them under spring pressure. This makes
BNC ideal wherever cables are connected and disconnected often: a lab bench, an antenna
switch, or a scanner front panel.

## What it is

Compared with a threaded [SMA](/reference/sma-connector/), BNC is bulkier and gives up
bandwidth — its usable range tops out near 4 GHz and its match degrades above about 1 GHz —
but it wins decisively on handling speed and mating-cycle life (typically 500+ cycles). The
bayonet also tolerates rough field use better than a fine thread that can cross-thread or
seize. Those trade-offs are why test equipment, which is plugged and unplugged constantly,
standardised on BNC while microwave and permanent installations moved to SMA and
[N-type](/reference/n-type-connector/).

## Variants

- **50 Ω BNC** is the RF standard for scanners, radios, and lab RF gear.
- **75 Ω BNC** is used for video and digital broadcast (SDI) signals. It has a slightly
  different centre-contact geometry; mating 50 Ω and 75 Ω parts works mechanically and is
  usually harmless at low frequency, but raises reflections
  ([SWR](/reference/standing-wave-ratio/)) as frequency climbs, so match the impedance for
  RF work.
- **Twin BNC** and **triaxial BNC** carry balanced or shielded-plus-guard signals in
  instrumentation and are not interchangeable with the standard single type.
- **Reverse-polarity BNC** exists but is uncommon.

## Relevance to SDR

Many discone and wideband scanner antennas terminate in BNC, and classic communications
receivers and spectrum analysers use it as the primary port, so a listener frequently needs
a **BNC-to-[SMA](/reference/sma-connector/)** adapter to reach a modern dongle. Each adapter
adds a small insertion loss and reflection, but at the VHF/UHF land-mobile bands where P25,
DMR, and NXDN live — well under 1 GHz — a good BNC joint is essentially transparent and its
convenience often outweighs the loss. GopherTrunk itself is decode software and never sees a
connector, yet the feedline chain that ends in a BNC sets the signal quality reaching the
receiver: a worn or cross-impedance BNC quietly costs signal-to-noise before any sample is
ever captured. And because no receiver can recover
[AES-encrypted](/police-scanner-encryption/) traffic, the connector chain only ever
determines *how cleanly* the clear traffic arrives — never whether encrypted traffic can
be decoded.

## Where to buy

Most people do not need a BNC-specific part — they need to get a **BNC** antenna or
jumper onto the **[SMA](/reference/sma-connector/)** jack the SDR actually uses. A small
**SMA-to-BNC adapter kit** (8 pieces, both genders, around $10) does that in every
direction for a few dollars, so you are never stuck on the wrong pin. If you already own
several mismatched antennas, the broader **[SMA adapter kit](/reference/sma-adapter-kit/)**
adds UHF, N, and F in the same box; for a flexible lead rather than a rigid barrel, use an
RG316 **[coax pigtail](/reference/coax-pigtail/)**.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B078K5563Y?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

For the full connector map and which coax to run, see the
[SDR cables and connectors guide](/sdr-cables-and-connectors/).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra
cost to you. It never changes what we recommend.*

## Sources

[^wiki]: [BNC connector](https://en.wikipedia.org/wiki/BNC_connector) — Wikipedia, on bayonet coupling, the Neill-Concelman naming, 50/75 Ω variants, and the ~4 GHz frequency range.
