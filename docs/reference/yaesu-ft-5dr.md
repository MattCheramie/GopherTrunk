---
slug: yaesu-ft-5dr
title: Yaesu FT-5DR
entry_type: hardware
category: ham-radios
description: "The Yaesu FT-5DR is a C4FM System Fusion handheld with built-in APRS, GPS, Bluetooth, true dual-band receive and IPX7 submersion rating — the most feature-dense ham HT under $400."
keywords: Yaesu FT-5DR, FT-5DR review, C4FM handheld, System Fusion HT, best C4FM radio 2026, APRS handheld, IPX7 ham radio, WiRES-X node radio, Yaesu digital handheld, FT3DR successor
aka: [FT-5DR, FT5DR, FT-5D]
autolink: true
affiliate: true
product:
  name: "Yaesu FT-5DR"
  brand: Yaesu
  category: Ham handheld transceiver
  lowPrice: "370"
  highPrice: "400"
  url: https://www.amazon.com/dp/B09GS924GT?tag=gophertrunk-20
infobox:
  - { label: Type, value: Dual-band handheld transceiver }
  - { label: Bands, value: "TX 144/430 MHz; RX 0.5–999.9 MHz" }
  - { label: Modes, value: "C4FM (System Fusion II), FM; APRS" }
  - { label: Power, value: 5 W }
  - { label: Programming, value: "Yaesu ADMS-14, RT Systems (no CHIRP)" }
  - { label: Price, value: around $380 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B09GS924GT?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [kenwood-th-d75a, icom-id-52a, yaesu-ft-60r, yaesu-vx-8dr, rtl-sdr, c4fm]
related_lessons:
  - { title: "Analog vs. digital voice", url: /learn/rf-sdr/digital-voice/ }
  - { title: "Legal & ethical monitoring", url: /learn/rf-sdr/legal-ethical/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://yaesu.com/product-detail.aspx?Model=FT5DR&CatName=VHF%2FUHF+Handhelds
faq:
  - q: "Is the Yaesu FT-5DR still in production?"
    a: "Yes — as of 2026 it is Yaesu's current flagship digital handheld, with active firmware updates (most recently November 2025). It replaced the FT3DR, which is now discontinued."
  - q: "Does the FT-5DR do APRS?"
    a: "Yes — a built-in 1200/9600 bps APRS modem with GPS. It's a solid implementation, though less deep than the Kenwood TH-D75A's (no KISS TNC for packet or Winlink work)."
  - q: "Is the FT-5DR waterproof?"
    a: "Yes — IPX7, rated submersible to 1 m for 30 minutes. That plus APRS, GPS, Bluetooth and C4FM at around $380 is the radio's whole argument."
  - q: "Can I program the FT-5DR with CHIRP?"
    a: "No — like Yaesu's other C4FM radios it isn't CHIRP-supported. Use Yaesu's free ADMS-14 software via microSD or USB, or RT Systems."
  - q: "Do I need a ham license for the FT-5DR?"
    a: "To transmit, yes — FCC amateur license, Technician class minimum. Listening needs no license, and its 0.5–999.9 MHz receiver with AM air band makes it a capable scanner-substitute on analog traffic."
---
**The Yaesu FT-5DR** is the most feature-dense ham handheld under $400:
[C4FM](/reference/c4fm/) ([System Fusion II](/reference/system-fusion-ysf/))
digital voice plus FM, built-in [APRS](/reference/aprs/) with GPS, Bluetooth,
true dual-band simultaneous receive, a color touch screen, and an **IPX7**
submersible body — roughly half the price of a
[TH-D75A](/reference/kenwood-th-d75a/).[^yaesu] It replaced the FT3DR and is
Yaesu's current digital flagship HT.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B09GS924GT?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The value flagship.** APRS + GPS + Bluetooth + C4FM + IPX7 + 999.9 MHz
wideband receive for **around $380** — feature-for-dollar the strongest spec
sheet in [our handheld top 10](/best-handheld-ham-radios/). The catch is
Yaesu's **menu system**: a steep learning curve and a small, fiddly resistive
touch screen. C4FM only — no [DMR](/reference/dmr/) or
[D-STAR](/reference/d-star/). Transmitting requires an FCC amateur license
(Technician minimum); listening requires none.
</div>

## Overview

Yaesu's digital HT line has iterated fast — FT1DR, FT2DR, FT3DR, now FT-5DR —
and each generation piled on features while holding the price near $400. The
FT-5DR lands with true dual-band simultaneous receive (V+V, U+U or V+U),
wideband coverage from 0.5 to 999.9 MHz including AM air band, a 2,200 mAh
battery with solid endurance, 900+ memories, and a microSD slot for memories,
GPS logs and voice recording. It can also act as a portable **WiRES-X** digital
node, linking a local simplex frequency into Yaesu's internet-connected rooms.

The trade-offs are consistent Yaesu: the menu tree is deep and unintuitive
until it clicks, the resistive touch screen is small and fiddly, Bluetooth is
finicky with non-Yaesu headsets, and the APRS implementation — while genuinely
useful — lacks the KISS TNC that makes the Kenwood the packet operator's
choice. And C4FM's ecosystem, strong in many US regions, is smaller than DMR's
in others: check what your local repeaters actually run before picking a
digital mode.

## Modes &amp; features

- **[C4FM](/reference/c4fm/) System Fusion II** + analog FM, with AMS
  auto-mode-select switching per received signal.
- **[APRS](/reference/aprs/)** 1200/9600 bps modem with built-in **GPS**.
- **True dual receive** — two bands (or two channels on one band) at once.
- **IPX7** — submersible 1 m / 30 min.
- **WiRES-X** portable digital node capability; **Bluetooth**; microSD.
- **5 W** TX; charges via supplied cradle or 12 V (USB is data only).

## Programming

Yaesu's free ADMS-14 programmer moves memories over microSD or USB, and
RT Systems sells the usual alternative. **CHIRP does not support it** — Yaesu's
C4FM radios generally aren't covered.

## GopherTrunk alternative

GopherTrunk receives only — it can't transmit, so it doesn't replace an
FT-5DR. Where it earns its keep is *before* the purchase: a ~$30
[RTL-SDR](/reference/rtl-sdr/) running free GopherTrunk shows you what your
local repeaters actually carry — [C4FM](/reference/c4fm/), DMR, D-STAR or plain
FM — and records and logs the traffic so you can gauge real activity, not
directory listings. That's exactly the data that decides between this radio,
an [AT-D878UVII Plus](/reference/anytone-at-d878uvii-plus/) and an
[ID-52A](/reference/icom-id-52a/). See
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/) for the hardware.

## Who it's for

- **Buy the FT-5DR** if your area runs System Fusion, or if you want the most
  radio per dollar — APRS, GPS and waterproofing included — and can live with
  Yaesu menus.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B09GS924GT?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [Kenwood TH-D75A](/reference/kenwood-th-d75a/)** instead for
  serious APRS/packet work or D-STAR, or the
  [Icom ID-52A](/reference/icom-id-52a/) for D-STAR with IP67 ruggedness.
- **Skip the digital premium** entirely with a
  [Yaesu FT-60R](/reference/yaesu-ft-60r/) if your local scene is analog FM.
  Full rankings: [best handheld ham radios](/best-handheld-ham-radios/).

## Sources

[^yaesu]: [Yaesu FT-5DR product page](https://yaesu.com/product-detail.aspx?Model=FT5DR&CatName=VHF%2FUHF+Handhelds) — Yaesu, on C4FM/FM, APRS and GPS, dual-band simultaneous receive, IPX7 rating, WiRES-X node capability, and wideband receive coverage.
