---
slug: uniden-bc125at
title: Uniden BC125AT
entry_type: hardware
category: consumer-scanners
description: "The Uniden BC125AT is a 500-channel analog handheld scanner with alpha tagging and Close Call — a classic for aviation, marine, rail, and racing — but it decodes no digital P25/DMR/NXDN voice."
keywords: Uniden BC125AT, BC125AT scanner, analog handheld scanner, aviation scanner, air band scanner, Close Call, alpha tag scanner, NASCAR scanner, marine scanner, Uniden BC125AT review
aka: [BC125AT]
autolink: true
affiliate: true
product:
  name: "Uniden BC125AT"
  brand: Uniden
  category: Police scanner (analog handheld)
  lowPrice: "101"
  highPrice: "119"
  url: https://www.amazon.com/dp/B00772MR0K?tag=gophertrunk-20
infobox:
  - { label: Type, value: Analog handheld scanner }
  - { label: Modes, value: "Analog FM/AM only — no digital voice" }
  - { label: Channels, value: "500 alpha-tagged, Close Call" }
  - { label: Coverage, value: "Air (civ+mil), marine, rail, racing" }
  - { label: Programming, value: "PC (free software) or keypad" }
  - { label: Price, value: around $110 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B00772MR0K?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [uniden-sr30c, uniden-bc75xlt, police-scanner, trunking-scanner, uniden-sds200, whistler-trx-1]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://www.uniden.com/products/bc125at
faq:
  - q: "Does the Uniden BC125AT decode digital police?"
    a: "No. The BC125AT is analog-only. It cannot decode P25, DMR, or NXDN digital voice. In areas where police, fire, or EMS have gone digital, those channels will sound like buzzing static. Check your county on RadioReference first."
  - q: "Is the BC125AT good for aviation?"
    a: "Yes — it is one of the most popular budget air-band scanners. It covers the civilian VHF air band and military UHF air band (both AM), so it is a favorite for airshows, airports, and plane spotting."
  - q: "What is alpha tagging?"
    a: "Alpha tagging lets you give each channel a text name (like 'TOWER' or 'CH1 PIT') that shows on screen instead of a raw frequency, so you know at a glance who you are hearing. The BC125AT supports it on all 500 channels."
  - q: "Is the BC125AT good for NASCAR and racing?"
    a: "Very. Race teams and officials use analog frequencies, and the BC125AT's Close Call plus alpha tags make it a longtime trackside standard. The same applies to marine VHF and railroad monitoring."
  - q: "BC125AT vs SR30C?"
    a: "They are close. The BC125AT is the older classic with a strong aviation reputation and alpha tagging; the SR30C is a newer, similar analog handheld. Both are analog-only — pick on price and feel. Neither hears digital."
  - q: "Can I get digital decoding cheaply instead?"
    a: "Yes. A ~$30 RTL-SDR plus free GopherTrunk decodes P25/DMR/NXDN on a PC and records every call — capability the analog BC125AT lacks. See our comparison."
---

**The Uniden BC125AT** is a 500-channel **analog handheld** scanner with alpha
tagging and [Close Call](/reference/police-scanner/) RF capture — a longtime budget
classic for **aviation, marine, rail, and racing**.[^uniden] Like every analog
scanner, it decodes **no digital voice**: it cannot hear
[P25](/reference/project-25/), [DMR](/reference/dmr/), or [NXDN](/reference/nxdn/),
so it only earns its keep where traffic is still analog.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B00772MR0K?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The budget analog classic.** 500 **alpha-tagged** channels,
[Close Call](/reference/police-scanner/), rock-solid for
**air (civilian + military), marine VHF, railroad, and NASCAR/racing**.
**Analog only** — no [P25](/reference/project-25/)/[DMR](/reference/dmr/)/[NXDN](/reference/nxdn/)
voice. **~$110.** If your police/fire went digital, it hears static there — get a
[digital scanner](/best-police-scanners/) or a free
[SDR + GopherTrunk](/police-scanner-vs-sdr/) instead.
</div>

## Overview

The BC125AT is one of the best-selling scanners ever made, and for good reason:
it is cheap, durable, PC-programmable, and covers the analog bands enthusiasts care
about most. It stores **500 channels in 10 banks**, each channel with an
**alpha tag** so the display reads "GROUND" or "PIT 24" instead of a frequency, and
its **Close Call** feature locks onto a strong nearby transmission you never
programmed.

It is, however, an **analog receiver**. There is no digital decoder inside — so
whether it is the right radio depends entirely on whether your targets are analog.

## What it hears — and what it can't

- **Aviation:** civilian VHF air band and **military UHF air band** (AM). This is
  the BC125AT's signature strength — airshows, airports, and plane spotting.
- **Marine VHF, railroad, and auto racing (NASCAR)** — all still analog, all well
  covered, and a big part of why this radio stays popular.
- **Analog police, fire, EMS, business, ham, and NOAA weather.**
- **Cannot hear digital voice.** [P25 Phase I/II](/reference/p25-phase-2/),
  [DMR](/reference/dmr/), and [NXDN](/reference/nxdn/) all pass straight through as
  unintelligible noise — the BC125AT has no [vocoder](/reference/vocoder/).
- **Cannot follow trunking.** It is not a
  [trunking scanner](/reference/trunking-scanner/) and cannot track talkgroups
  across a [trunked system](/reference/trunked-radio/).

> **Digital agencies = static.** If [RadioReference](/reference/radioreference/)
> lists your police or fire as P25/DMR/NXDN, the BC125AT will not decode them.

## Programming

Enter channels from the keypad, or — far faster — use Uniden's **free PC software**
over the included USB cable to load big frequency lists (for example, RadioReference
exports) and set alpha tags in bulk. There is no ZIP-code database; you supply the
frequencies.

## GopherTrunk alternative

The BC125AT is a superb **analog** pocket scanner and nothing here changes that. But
it will never hear a digital system, and it can't record or log.

**GopherTrunk** covers the gap. A ~$30 [RTL-SDR](/reference/rtl-sdr/) plus the free,
open-source software decodes [P25](/reference/project-25/), [DMR](/reference/dmr/),
and [NXDN](/reference/nxdn/), follows [trunked](/reference/trunked-radio/) talkgroups
the BC125AT can't, and **records, logs, and timestamps every call** across unlimited
channels. It needs a PC and isn't pocketable, but it does what an analog handheld
physically cannot. Read the honest
[police scanner vs GopherTrunk](/police-scanner-vs-sdr/) write-up, or just
[download it](/downloads.html) and try a dongle.

As always: **no scanner and no SDR can decode
[AES encryption](/police-scanner-encryption/)** — that limit is universal.

## Who it's for

- **Buy the BC125AT** for analog aviation, marine, rail, racing, or an area whose
  public safety is still analog.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B00772MR0K?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Consider the [SR30C](/reference/uniden-sr30c/)** or
  [BC75XLT](/reference/uniden-bc75xlt/) as similar analog handhelds.
- **Skip it** for a [digital scanner](/best-police-scanners/) or free
  [SDR + GopherTrunk](/police-scanner-vs-sdr/) if your agencies run P25/DMR/NXDN.

## Bottom line

For [cheap analog scanning](/cheap-police-scanner/) — especially aviation and
racing — the BC125AT remains a gold standard at around $110. Just verify your
public-safety targets are still analog; if they've gone digital, this radio can't
follow, and a [digital scanner](/best-police-scanners/) or
[GopherTrunk](/police-scanner-vs-sdr/) is what you need.

## Sources

[^uniden]: [Uniden BC125AT product page](https://www.uniden.com/products/bc125at) — Uniden America, on 500 alpha-tagged channels, Close Call, band coverage (civil/military air, marine), and PC programming (analog-only receiver).
