---
slug: yaesu-ft-60r
title: Yaesu FT-60R
entry_type: hardware
category: ham-radios
description: "The Yaesu FT-60R is the legendary analog dual-band ham handheld — in production for over two decades, nearly indestructible, CHIRP-programmable, and still the default 'first real HT' recommendation at around $180."
keywords: Yaesu FT-60R, FT-60R review, best first ham radio, dual band FM handheld, most durable ham HT, analog ham handheld, Field Day radio, CHIRP compatible radio, Yaesu FT-60R vs Baofeng
aka: [FT-60R, FT-60]
autolink: true
affiliate: true
product:
  name: "Yaesu FT-60R"
  brand: Yaesu
  category: Ham handheld transceiver
  lowPrice: "170"
  highPrice: "190"
  url: https://www.amazon.com/dp/B004P4PDAO?tag=gophertrunk-20
infobox:
  - { label: Type, value: Dual-band handheld transceiver }
  - { label: Bands, value: "TX 144/430 MHz; RX 108–520, 700–999.99 MHz" }
  - { label: Modes, value: "Analog FM only; AM air band receive" }
  - { label: Power, value: 5 W }
  - { label: Programming, value: "CHIRP (mature driver), ADMS-1J, RT Systems" }
  - { label: Price, value: around $180 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B004P4PDAO?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [yaesu-ft-5dr, baofeng-bf-f8hp, baofeng-uv-5r, btech-uv-pro, rtl-sdr, ctcss]
related_lessons:
  - { title: "Analog vs. digital voice", url: /learn/rf-sdr/digital-voice/ }
  - { title: "Legal & ethical monitoring", url: /learn/rf-sdr/legal-ethical/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://www.yaesu.com/product-detail.aspx?Model=FT-60R
faq:
  - q: "Is the Yaesu FT-60R still worth buying in 2026?"
    a: "Yes, if your local scene is analog FM. It has been in production for over twenty years because nothing at the price matches its durability, front-end selectivity and dead-simple operation. It has no digital modes, GPS or Bluetooth — that's the trade, and for many hams it's the right one."
  - q: "What's the difference between an FT-60R and a Baofeng UV-5R?"
    a: "About $155, and you can hear it. The FT-60R's receiver holds up next to strong transmitters where the Baofeng's wide-open front end collapses into intermod, its transmit is clean, and the hardware survives years of abuse. The UV-5R is the beater you don't mind losing; the FT-60R is the radio you keep."
  - q: "Does the FT-60R work with CHIRP?"
    a: "Yes — one of CHIRP's most mature drivers. You'll need a USB programming cable with the 4-conductor plug. Yaesu's ADMS-1J and RT Systems software also cover it."
  - q: "Is the FT-60R waterproof?"
    a: "No formal IP rating — it's weather-resistant and famously survives hard field use, but it isn't submersible. For a rated body, look at the IPX7 FT-5DR or IP67 BTECH UV-PRO."
  - q: "Do I need a license to use an FT-60R?"
    a: "Transmitting requires an FCC amateur license (Technician class minimum). Listening requires none — and its 108–520 MHz receive with AM air band covers a lot of listening."
---
**The Yaesu FT-60R** is the cockroach of ham radio, and that's a compliment: an
analog dual-band 144/430 MHz handheld introduced in 2004 and **still sold new
22 years later**, because it survives the drops, rain and glovebox summers that
kill fancier rigs.[^yaesu] 5 W, 1,000 memories, wide receive with AM air band,
around $180 — and not one digital feature anywhere on it.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B004P4PDAO?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The perennial "first real HT."** Legendary durability, a front end with real
selectivity (night-and-day next to a [UV-5R](/reference/baofeng-uv-5r/) near
strong transmitters), dead-simple controls, mature
CHIRP support, and a cheap, plentiful battery ecosystem. **Analog FM only** —
no [DMR](/reference/dmr/), [C4FM](/reference/c4fm/), [D-STAR](/reference/d-star/),
GPS or Bluetooth — and no formal waterproof rating. **~$180.** Transmitting
takes an FCC Technician license; listening takes none.
</div>

## Overview

Every club has the same story: the FT-60R someone dropped off a roof, left in
the rain at Field Day, and is still using. The polycarbonate case, mechanical
simplicity and conservative RF design have kept this radio the default
recommendation for a new ham's first serious handheld for two decades. It does
one job — analog FM voice on 2 m and 70 cm — and does it with a receiver that
stays quiet next to paging and broadcast transmitters where Baofeng-class
radios fold up, plus clean, well-regarded transmit audio.

The spec sheet is honest about its age: the stock FNB-83 battery is 1,400 mAh
NiMH (spares and clones are cheap and everywhere, and an AA tray exists), the
display is monochrome, and dual watch is a fast scanner rather than two true
receivers. There is no IP rating — its toughness is reputation, not
certification. And in a digital era it will never decode a digital repeater.

## Modes &amp; features

- **Analog FM** on 144/430 MHz at 5 W; [CTCSS](/reference/ctcss/) and
  [DCS](/reference/dcs/) tone signalling, plus Enhanced Paging (EPCS).
- **Receive 108–520 and 700–999.99 MHz** (cellular blocked) including AM air
  band and NOAA weather scan.
- **1,000 memories** with alphanumeric tags.
- **Battery ecosystem**: NiMH FNB-83 stock, cheap spares, FBA-25 AA tray for
  emergencies.
- **No** digital voice, GPS, Bluetooth, APRS or USB — deliberately.

## Programming

The FT-60R is fully supported by **CHIRP** — one of its longest-standing,
most mature drivers — over a 4-conductor USB programming cable. Yaesu's
ADMS-1J and RT Systems software work too, and the front-panel keypad plus
sensible menus make manual entry genuinely practical, which is more than most
HTs can claim.

## GopherTrunk alternative

GopherTrunk can't transmit, so it doesn't replace an FT-60R — but it answers
the question that should precede any radio purchase: *what's actually on the
air here?* A ~$30 [RTL-SDR](/reference/rtl-sdr/) running free GopherTrunk
monitors your local repeaters and digital traffic, and records and logs every
transmission — so you'll know whether your area is analog-FM country (where the
FT-60R is all you need) or has gone digital (where it isn't) before you spend a
dollar on a transceiver. Hardware picks:
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).

## Who it's for

- **Buy the FT-60R** as a first licensed radio, a Field Day/go-bag workhorse,
  or the HT you hand to a new operator — anywhere analog FM is the traffic and
  durability is the priority.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B004P4PDAO?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [Yaesu FT-5DR](/reference/yaesu-ft-5dr/)** instead if your
  repeaters run C4FM or you want APRS/GPS — it's the same brand's digital
  flagship at ~$380.
- **Spend less** on a [Baofeng BF-F8HP](/reference/baofeng-bf-f8hp/) (~$60) or
  [UV-5R](/reference/baofeng-uv-5r/) (~$25) if you just want a beater — with
  the front-end and QC caveats on those pages. Full rankings:
  [best handheld ham radios](/best-handheld-ham-radios/).

## Sources

[^yaesu]: [Yaesu FT-60R product page](https://www.yaesu.com/product-detail.aspx?Model=FT-60R) — Yaesu, on dual-band FM coverage, 5 W output, wide receiver range, 1,000 memories, and CTCSS/DCS signalling.
