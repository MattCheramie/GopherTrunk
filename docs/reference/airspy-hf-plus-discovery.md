---
slug: airspy-hf-plus-discovery
title: Airspy HF+ Discovery
entry_type: hardware
category: sdr-devices
description: "The Airspy HF+ Discovery is a high-dynamic-range HF/VHF software-defined radio (0.5 kHz–31 MHz and 60–260 MHz) built for shortwave, ham and weak-signal work — not the UHF band most trunking lives on."
keywords: Airspy HF+ Discovery, HF Discovery SDR, high dynamic range SDR, HF VHF receiver, shortwave SDR, polyphase harmonic rejection, weak signal SDR, Airspy HF plus
aka: [Airspy HF+ Discovery, HF+ Discovery]
autolink: true
affiliate: true
product:
  name: "Airspy HF+ Discovery"
  brand: Airspy
  category: Software-defined radio
  lowPrice: "169"
  highPrice: "199"
  url: https://www.amazon.com/s?k=Airspy+HF+Discovery+SDR&tag=gophertrunk-20
infobox:
  - { label: Type, value: HF/VHF SDR receiver }
  - { label: Vendor, value: Airspy }
  - { label: ADC, value: 18-bit (HDR, effective) }
  - { label: Range, value: "0.5 kHz–31 MHz, 60–260 MHz" }
  - { label: Bandwidth, value: up to ~768 kHz }
  - { label: TX, value: No (receive only) }
  - { label: Price, value: around $169–199 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/s?k=Airspy+HF+Discovery+SDR&tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [airspy-hf-plus, airspy, airspy-r2, airspy-mini, sdrplay-rsp1b, software-defined-radio, dynamic-range]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/rf-sdr/sdr-hardware/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
  - https://airspy.com/airspy-hf-discovery/
faq:
  - q: "Is the Airspy HF+ Discovery good for scanning trunked systems?"
    a: "Not really — and that's by design. Its coverage is 0.5 kHz–31 MHz (HF) and 60–260 MHz (low VHF), which stops well below the 380–870 MHz UHF bands where most P25/DMR/NXDN/TETRA trunking lives. It's a specialist HF and weak-signal receiver. For trunking, use an Airspy R2/Mini, an RTL-SDR, or an SDRplay RSP1B."
  - q: "What is the HF+ Discovery actually for?"
    a: "Shortwave broadcast and utility listening, ham HF, low-VHF weak-signal work, and any situation where strong nearby transmitters would overload a lesser receiver. Its headline is world-class dynamic range and harmonic rejection, not wide capture — the visible span is narrow (up to ~768 kHz)."
  - q: "HF+ Discovery or the original HF+?"
    a: "The Discovery is the smaller, refreshed HF+ with wider VHF coverage and a lower price. Both target the same HF/low-VHF high-dynamic-range niche. If you're choosing for GopherTrunk trunking specifically, neither is the right band — pick a VHF/UHF Airspy instead."
  - q: "Does the HF+ Discovery decode encrypted or trunked voice?"
    a: "It's a capable receiver, but it doesn't reach the UHF trunking bands, and like every SDR it cannot decode AES encryption. GopherTrunk can drive Airspy hardware, but this model's band coverage makes it the wrong tool for trunk-tracking."
---

**The Airspy HF+ Discovery** is a high-dynamic-range HF/VHF
[software-defined radio](/reference/software-defined-radio/) receiver covering
**0.5 kHz–31 MHz** and **60–260 MHz** — a specialist shortwave, ham and weak-signal radio
built around world-class strong-signal handling rather than wide capture.[^wiki] It is the
compact, refreshed member of the [Airspy HF+](/reference/airspy-hf-plus/) family, and, unlike
the VHF/UHF [Airspy R2](/reference/airspy-r2/) and [Mini](/reference/airspy-mini/), it stops
below the UHF bands where most trunking lives.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A frequency coverage bar for the Airspy HF+ Discovery showing HF from near zero to 31 megahertz and a low-VHF segment from 60 to 260 megahertz, both well below the UHF trunking bands, on an axis from 0 to 1 gigahertz." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="86">0</text><text x="150" y="86">300 MHz</text><text x="270" y="86">600 MHz</text><text x="430" y="86">1 GHz</text></g>
  <rect x="30" y="40" width="14" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <rect x="54" y="40" width="80" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="28" text-anchor="middle" font-size="10" fill="currentColor">HF (≤31 MHz) + low VHF (60–260 MHz) — below UHF trunking</text>
</svg>
<figcaption>The HF+ Discovery's coverage is HF and low VHF only — it does not reach the UHF public-safety trunking bands.</figcaption>
</figure>

<a class="btn btn--buy" href="https://www.amazon.com/s?k=Airspy+HF+Discovery+SDR&tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**A specialist HF/low-VHF receiver — not a trunking radio.** The HF+ Discovery's headline is
world-class [dynamic range](/reference/dynamic-range/) and harmonic rejection for shortwave,
ham and weak-signal work, but its coverage (**0.5 kHz–31 MHz** and **60–260 MHz**) stops well
below the **380–870 MHz** UHF bands where most
[P25](/reference/project-25/)/[DMR](/reference/dmr/)/[NXDN](/reference/nxdn/)/[TETRA](/reference/tetra/)
trunking lives. **For GopherTrunk trunk-tracking, buy an
[Airspy R2](/reference/airspy-r2/)/[Mini](/reference/airspy-mini/), an
[RTL-SDR](/reference/rtl-sdr/), or an [SDRplay RSP1B](/reference/sdrplay-rsp1b/) instead.**
**Receive-only, ~$169–199.** Airspy is distributor-sold, so Amazon stock is intermittent — the
button tracks live listings. Like every receiver it can't decode
[AES encryption](/police-scanner-encryption/).
</div>

## Overview

The HF+ Discovery is a different design from the VHF/UHF [Airspy](/reference/airspy/) receivers.
Where the [R2](/reference/airspy-r2/) and [Mini](/reference/airspy-mini/) chase wide capture
across VHF/UHF, the Discovery optimises for **dynamic range and selectivity** in the crowded HF
spectrum, where a distant weak signal often sits beside a booming local broadcaster. Its
polyphase harmonic-rejection architecture and high effective resolution let it pull weak
shortwave and ham signals out from under strong neighbours that would desensitise or overload a
general-purpose dongle — at the cost of a narrow visible [bandwidth](/reference/bandwidth/)
(up to roughly 768 kHz).[^airspy]

## What it is for

This is a receiver for **shortwave broadcast, utility and ham HF**, plus **low-VHF weak-signal**
work (6 m, 2 m at the top of its range, and the segments in between). It is a favourite of DXers
and HF listeners precisely because of its front-end quality. What it is **not** is a scanner-band
radio: its coverage tops out at 260 MHz, so the UHF land-mobile and public-safety trunking
allocations are simply out of range. If you already own one for HF, it will not double as your
trunk-tracking receiver.

## Relevance to GopherTrunk

GopherTrunk can drive Airspy hardware natively over USB, but band coverage decides fit, and the
HF+ Discovery's does not reach the frequencies GopherTrunk decodes on most systems. Treat it as
the **HF companion** to a GopherTrunk setup, not the capture device: use it for shortwave and
ham listening, and pair a VHF/UHF [Airspy R2](/reference/airspy-r2/) or
[Mini](/reference/airspy-mini/), an [RTL-SDR](/reference/rtl-sdr/), or an
[SDRplay RSP1B](/reference/sdrplay-rsp1b/) for the actual
[P25](/reference/project-25/)/[DMR](/reference/dmr/)/[NXDN](/reference/nxdn/) trunking. As with
every SDR, it decodes clear signals only — never [AES](/police-scanner-encryption/).

## Where to buy

Airspy is sold through its own distributor network, so Amazon stock comes and goes — the button
is a tagged search that always resolves to current listings. Buy the **HF+ Discovery** if HF and
low-VHF weak-signal reception is your goal; for **GopherTrunk trunk-tracking** buy a VHF/UHF
radio instead — the [Airspy R2](/reference/airspy-r2/) or [Mini](/reference/airspy-mini/), a
cheap [RTL-SDR](/reference/rtl-sdr/), or an [SDRplay RSP1B](/reference/sdrplay-rsp1b/).

<a class="btn btn--buy" href="https://www.amazon.com/s?k=Airspy+HF+Discovery+SDR&tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

Deciding between radios? See [best SDR for GopherTrunk](/best-sdr-for-gophertrunk/) and the
wider [Airspy HF+](/reference/airspy-hf-plus/) family, then grab GopherTrunk from the
[downloads page](/downloads.html).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost
to you. It never changes what we recommend.*

## Sources

[^wiki]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, background on high-dynamic-range HF SDR receivers.
[^airspy]: [Airspy HF+ Discovery](https://airspy.com/airspy-hf-discovery/) — Airspy, official page on the HF+ Discovery's 0.5 kHz–31 MHz and 60–260 MHz coverage, dynamic range, and harmonic rejection.
