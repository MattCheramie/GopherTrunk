---
slug: uniden-bc75xlt
title: Uniden BC75XLT
entry_type: hardware
category: consumer-scanners
description: "The Uniden BC75XLT is a 300-channel analog handheld scanner with Close Call and Weather Alert — a low-cost entry radio for analog police, air, marine, and weather, with no digital P25/DMR/NXDN."
keywords: Uniden BC75XLT, BC75XLT scanner, analog handheld scanner, weather alert scanner, Close Call, 300 channel scanner, cheap police scanner, entry level scanner, Uniden BC75XLT review
aka: [BC75XLT]
autolink: true
affiliate: true
product:
  name: "Uniden BC75XLT"
  brand: Uniden
  category: Police scanner (analog handheld)
  lowPrice: "92"
  highPrice: "108"
  url: https://www.amazon.com/dp/B00A1VSO9M?tag=gophertrunk-20
infobox:
  - { label: Type, value: Analog handheld scanner }
  - { label: Modes, value: "Analog FM only — no digital voice" }
  - { label: Channels, value: "300, with Close Call RF capture" }
  - { label: Extras, value: "Weather Alert (S.A.M.E.)" }
  - { label: Programming, value: "Keypad or PC software" }
  - { label: Price, value: around $100 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B00A1VSO9M?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [uniden-bc125at, uniden-sr30c, police-scanner, trunking-scanner, uniden-sds200, whistler-trx-1]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://www.uniden.com/products/bc75xlt
faq:
  - q: "Does the Uniden BC75XLT decode digital police?"
    a: "No. The BC75XLT is analog-only and cannot decode P25, DMR, or NXDN. Where agencies have gone digital, those channels are just noise. Confirm your county on RadioReference before buying."
  - q: "What is the Weather Alert feature?"
    a: "The BC75XLT can monitor NOAA weather channels and sound an alert on a S.A.M.E. warning tone for your area, so it doubles as an emergency weather radio for storms and alerts."
  - q: "How many channels does the BC75XLT hold?"
    a: "300 channels in 10 banks — fewer than the 500-channel SR30C and BC125AT, which is why it costs a little less. It still includes Close Call RF capture."
  - q: "Is the BC75XLT a good first scanner?"
    a: "It is one of the cheapest ways to try the hobby, and it is simple to use. Just make sure your local agencies are still analog — if they run digital, a beginner is better served by a digital scanner or a free SDR + GopherTrunk."
  - q: "BC75XLT vs BC125AT?"
    a: "The BC125AT adds alpha tagging and more channels (500 vs 300) and has a stronger aviation reputation for a few dollars more. Both are analog-only. If air-band matters to you, step up to the BC125AT."
---
**The Uniden BC75XLT** is a 300-channel **analog handheld** scanner with
[Close Call](/reference/police-scanner/) RF capture and NOAA **Weather Alert** — one
of the cheapest ways to start scanning.[^uniden] Because it is analog-only, it
decodes **no digital voice** ([P25](/reference/project-25/),
[DMR](/reference/dmr/), [NXDN](/reference/nxdn/)) and is useful only where traffic
is still analog.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B00A1VSO9M?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Cheapest way in.** 300 channels, [Close Call](/reference/police-scanner/), and
**NOAA Weather Alert** for around $100. **Analog only** — no
[P25](/reference/project-25/)/[DMR](/reference/dmr/)/[NXDN](/reference/nxdn/) voice.
Good for **analog police/fire, air, marine, and weather**; useless where agencies
went digital. Check [RadioReference](/reference/radioreference/) first, and consider
a [digital scanner](/best-police-scanners/) or free
[SDR + GopherTrunk](/police-scanner-vs-sdr/) if you need digital.
</div>

## Overview

The BC75XLT is Uniden's entry-level handheld: 300 channels in 10 banks, keypad or
PC programming, **Close Call** to grab a strong nearby signal, and a **Weather
Alert** mode that watches NOAA channels and sounds an alarm on a S.A.M.E. warning.
It is small, simple, and about as inexpensive as a name-brand scanner gets.

It is an **analog receiver** with no digital decoder — so, as with its siblings, the
only question that matters is whether your targets are still analog.

## What it hears — and what it can't

- **Analog police, fire, and EMS** where they haven't migrated to digital.
- **Aviation, marine VHF, railroad, and racing** analog channels.
- **NOAA weather** with S.A.M.E. alerting — a genuine bonus that turns it into an
  emergency weather radio.
- **Cannot hear digital voice** — [P25 Phase I/II](/reference/p25-phase-2/),
  [DMR](/reference/dmr/), [NXDN](/reference/nxdn/) all pass as noise. No
  [vocoder](/reference/vocoder/) inside.
- **Cannot follow trunking** — it is not a
  [trunking scanner](/reference/trunking-scanner/) and can't track
  [talkgroups](/reference/talkgroup/) on a [trunked system](/reference/trunked-radio/).

> **Verify first.** If [RadioReference](/reference/radioreference/) shows your
> agencies as P25/DMR/NXDN, the BC75XLT can't decode them.

## Programming

Enter frequencies from the keypad, or connect USB and use Uniden's **free PC
software** to load lists in bulk (RadioReference exports work well). There is no
ZIP-code database; you provide the frequencies. Weather Alert is enabled from the
menu.

## GopherTrunk alternative

At this price the BC75XLT is a fine analog starter — but it can't touch a digital
system, and it can't record. If either matters, the better first purchase may be a
dongle instead of a handheld.

**GopherTrunk** is free and open source. Paired with a ~$30
[RTL-SDR](/reference/rtl-sdr/), it decodes [P25](/reference/project-25/),
[DMR](/reference/dmr/), and [NXDN](/reference/nxdn/), follows
[trunked](/reference/trunked-radio/) talkgroups, and **records, logs, and
timestamps every call**. It needs a PC and gives up pocket portability, but it does
what an analog handheld can't. See
[police scanner vs GopherTrunk](/police-scanner-vs-sdr/), or
[download GopherTrunk](/downloads.html) to try it.

And to be clear: **no scanner, and no SDR, decodes
[AES encryption](/police-scanner-encryption/)** — that wall is the same for
everyone.

## Who it's for

- **Buy the BC75XLT** as a cheap analog starter for police/fire, air, marine, or as
  a scanner-plus-weather-radio, in an area that's still analog.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B00A1VSO9M?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Step up to the [BC125AT](/reference/uniden-bc125at/)** for alpha tags, 500
  channels, and better air-band coverage.
- **Skip it** for a [digital scanner](/best-police-scanners/) or free
  [SDR + GopherTrunk](/police-scanner-vs-sdr/) if your agencies run digital.

## Bottom line

The BC75XLT is the budget on-ramp to scanning and a handy
[cheap analog radio](/cheap-police-scanner/) with a weather-alert bonus — as long as
your public-safety targets remain analog. If they've gone digital, it won't hear
them, and a [digital scanner](/best-police-scanners/) or
[GopherTrunk](/police-scanner-vs-sdr/) is the right call.

## Sources

[^uniden]: [Uniden BC75XLT product page](https://www.uniden.com/products/bc75xlt) — Uniden America, on 300 channels, Close Call, and Weather Alert (analog-only receiver).
