---
slug: baofeng-bf-f8hp
title: Baofeng BF-F8HP
entry_type: hardware
category: ham-radios
description: "The Baofeng BF-F8HP is the 8-watt 'UV-5R 3rd gen' — the same dirt-cheap dual-band platform with more power, a bigger battery, a better antenna and US-based BTECH support, for around $60."
keywords: Baofeng BF-F8HP, BF-F8HP review, BF-F8HP vs UV-5R, 8 watt Baofeng, cheap ham handheld, budget dual band radio, Baofeng 8W, CHIRP Baofeng, beater ham radio
aka: [BF-F8HP]
autolink: true
affiliate: true
product:
  name: "Baofeng BF-F8HP"
  brand: Baofeng
  category: Ham handheld transceiver
  lowPrice: "40"
  highPrice: "70"
  url: https://www.amazon.com/dp/B00MAULSOK?tag=gophertrunk-20
infobox:
  - { label: Type, value: Dual-band handheld transceiver }
  - { label: Bands, value: "TX 136–174 / 400–520 MHz" }
  - { label: Modes, value: Analog FM only }
  - { label: Power, value: "8/4/1 W tri-power" }
  - { label: Programming, value: "CHIRP (mature driver)" }
  - { label: Price, value: around $60 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B00MAULSOK?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [baofeng-uv-5r, btech-uv-pro, yaesu-ft-60r, anytone-at-d878uvii-plus, rtl-sdr, ctcss]
related_lessons:
  - { title: "Analog vs. digital voice", url: /learn/rf-sdr/digital-voice/ }
  - { title: "Legal & ethical monitoring", url: /learn/rf-sdr/legal-ethical/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://baofengtech.com/product/bf-f8hp/
faq:
  - q: "Is the BF-F8HP better than the UV-5R?"
    a: "Yes, meaningfully, for about $35 more: 8 W tri-power instead of 4–5, a 2,100 mAh battery, a better stock antenna, and US-based BTECH support with a one-year warranty. It's still the same UV-5R platform underneath — same weak front end, same hostile menus."
  - q: "Does 8 watts really make a difference?"
    a: "Less than you'd hope. 8 W vs 5 W is about 2 dB — barely noticeable — and the radio runs hot at full power. Antenna height and quality matter far more than the extra watts."
  - q: "Does the BF-F8HP work with CHIRP?"
    a: "Yes — a mature CHIRP driver (select BF-F8HP). CHIRP is effectively mandatory: the front-panel menu experience is hostile without it. Buy a genuine-FTDI programming cable; counterfeits are rampant."
  - q: "Is the BF-F8HP waterproof or digital?"
    a: "Neither. No IP rating, analog FM only, no GPS or Bluetooth. For those features near this price bracket, look at the BTECH UV-PRO (~$165)."
  - q: "Do I need a license to use a BF-F8HP?"
    a: "Transmitting on the ham bands requires an FCC amateur license (Technician class minimum) — and note this radio will happily transmit outside the ham bands, which is not legal without the relevant authorization. Listening requires no license."
---
**The Baofeng BF-F8HP** is what BTECH markets as the "UV-5R 3rd generation": the
same ubiquitous dual-band budget platform as the
[UV-5R](/reference/baofeng-uv-5r/), upgraded to **8 W** tri-power, a 2,100 mAh
battery, a better stock antenna and US-based BTECH support with a one-year
warranty — for around $60.[^btech] It is the sensible way to buy a Baofeng, and
it is still a Baofeng.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B00MAULSOK?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The "better UV-5R."** More power, better battery and antenna, real US
support, same dirt-cheap accessory ecosystem, mature CHIRP driver. It also
inherits the platform's faults wholesale: a **wide-open front end** that
intermods near strong transmitters, mediocre selectivity, mushy squelch, and
menus that are hostile without CHIRP. The 8 W advantage over 5 W is marginal
in practice and runs hot. **~$60.** Transmitting requires an FCC Technician
license; listening requires none.
</div>

## Overview

The BF-F8HP exists because the UV-5R's weakest points — anemic stock battery,
poor stock antenna, no accountable support — were cheap to fix. BTECH fixed
them, stamped a tri-power 8 W final on it, and sells the result as a "full kit"
with antenna, earpiece and charger under a US warranty umbrella. For a
glovebox, go-bag or loaner radio, that package at $60 is genuinely hard to
argue with.

What it doesn't fix is the platform. The receiver is the same wide-open design:
park near a paging transmitter or broadcast site and it collapses into
intermod a [Yaesu FT-60R](/reference/yaesu-ft-60r/) shrugs off. Selectivity is
mediocre, the squelch is mushy, and the menu system remains an exercise in
memorizing two-digit codes. BTECH cites Part 90 certification carried over
from the UV-5R family — if the certification status matters to you, verify the
grant for the specific unit, a caution the whole Baofeng family has earned
(see the [UV-5R page](/reference/baofeng-uv-5r/) for that story). Note also the
newer **BF-F8HP PRO** is a separate model (10 W tri-band, GPS, USB-C) that has
begun superseding this one technologically.

## Modes &amp; features

- **Analog FM only** on 136–174 / 400–520 MHz — no digital voice, GPS or
  Bluetooth.
- **8/4/1 W tri-power**; expect heat at 8 W and a barely-audible real-world
  gain over 5 W.
- **2,100 mAh Li-ion** — noticeably better endurance than a stock UV-5R.
- **[CTCSS](/reference/ctcss/)/[DCS](/reference/dcs/)**, FM broadcast receive,
  128 channels, LED flashlight.
- **No IP rating** — keep it out of the rain.

## Programming

**CHIRP** has a mature BF-F8HP driver and is the community-standard way to
program it — realistically the only pleasant way. Buy a programming cable with
a genuine FTDI chip; counterfeit cables are the single most common Baofeng
setup failure.

## GopherTrunk alternative

GopherTrunk can't transmit, so it doesn't replace even a $60 HT. But if the
BF-F8HP appeals as a cheap way *into* radio, consider that a ~$30
[RTL-SDR](/reference/rtl-sdr/) running free GopherTrunk is the cheap way into
*listening*: it monitors local repeaters and digital traffic the Baofeng can't
decode ([DMR](/reference/dmr/), [C4FM](/reference/c4fm/),
[D-STAR](/reference/d-star/)), and records and logs everything. Listen first,
find out what's active, then buy the transmitter that matches — see
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).

## Who it's for

- **Buy the BF-F8HP** as a beater, loaner, go-bag or first-experiment radio —
  it's the best-sorted version of the cheapest viable HT platform.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B00MAULSOK?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [Yaesu FT-60R](/reference/yaesu-ft-60r/)** (~$180) instead the
  moment the radio matters — the front end and build quality are a different
  class.
- **Spend $165** on a [BTECH UV-PRO](/reference/btech-uv-pro/) for APRS, GPS
  and IP67, or **$25** on a bare [UV-5R](/reference/baofeng-uv-5r/) if even $60
  is too much. Full rankings:
  [best handheld ham radios](/best-handheld-ham-radios/).

## Sources

[^btech]: [Baofeng BF-F8HP product page](https://baofengtech.com/product/bf-f8hp/) — BTECH, on tri-power output, battery and kit contents, frequency coverage, and warranty/support terms.
