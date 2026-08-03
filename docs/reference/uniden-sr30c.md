---
slug: uniden-sr30c
title: Uniden SR30C
entry_type: hardware
category: consumer-scanners
description: "The Uniden SR30C is a 500-channel analog handheld police scanner with Close Call RF capture and Turbo Search — great for analog agencies, air, marine, rail, and racing, but it cannot decode any."
keywords: Uniden SR30C, SR30C scanner, analog handheld scanner, Close Call scanner, 500 channel scanner, budget police scanner, Uniden SR30C review, analog only scanner
aka: [SR30C]
autolink: true
affiliate: true
product:
  name: "Uniden SR30C"
  brand: Uniden
  category: Police scanner (analog handheld)
  lowPrice: "110"
  highPrice: "129"
  url: https://www.amazon.com/dp/B07S9H8YH3?tag=gophertrunk-20
infobox:
  - { label: Type, value: Analog handheld scanner }
  - { label: Modes, value: "Analog FM only — no digital voice" }
  - { label: Channels, value: "500, with Close Call RF capture" }
  - { label: Programming, value: "PC (Uniden software) or keypad" }
  - { label: Extras, value: "Turbo Search, backlit display" }
  - { label: Price, value: around $120 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B07S9H8YH3?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [uniden-bc125at, uniden-bc75xlt, police-scanner, trunking-scanner, uniden-sds200, whistler-trx-1]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://www.uniden.com/products/sr30c
faq:
  - q: "Does the Uniden SR30C decode digital police (P25, DMR, NXDN)?"
    a: "No. The SR30C is an analog-only scanner. If your local police, fire, or EMS have moved to a digital P25, DMR, or NXDN system — most metro areas have — you will hear nothing but noise on those channels. Check your county on RadioReference before buying."
  - q: "What is Close Call on the SR30C?"
    a: "Close Call is Uniden's RF-capture feature: it instantly detects and tunes a strong nearby transmission even if you never programmed that frequency. It is handy for finding a channel in use at a track, event, or incident right in front of you."
  - q: "What can the SR30C actually hear well?"
    a: "Anything still analog: many rural and suburban police/fire departments, plus civilian and military aviation, marine VHF, railroad, auto racing (NASCAR), business two-way, and NOAA weather. These bands are largely analog, so an analog scanner is all you need."
  - q: "How do I program the SR30C?"
    a: "You can enter frequencies from the keypad, but most owners use Uniden's free PC software over USB to load frequency lists exported from RadioReference. There is no ZIP-code database like the digital HomePatrol models."
  - q: "SR30C or a free SDR + GopherTrunk?"
    a: "If you need portability and only listen to analog, the SR30C is simple and pocketable. If your agencies are digital or you want recording and logging, a ~$30 RTL-SDR running free GopherTrunk decodes P25/DMR/NXDN that the SR30C physically cannot."
---
**The Uniden SR30C** is a compact 500-channel **analog handheld** police scanner
with [Close Call](/reference/police-scanner/) RF capture and Turbo Search.[^uniden]
It is an easy, inexpensive way to monitor **analog** public-safety, aviation,
marine, rail, and racing traffic — but it **does not decode any digital voice**, so
it is the wrong radio wherever agencies have moved to
[P25](/reference/project-25/), [DMR](/reference/dmr/), or [NXDN](/reference/nxdn/).

<a class="btn btn--buy" href="https://www.amazon.com/dp/B07S9H8YH3?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Analog only.** The SR30C hears analog FM and nothing digital — no
[P25](/reference/project-25/)/[DMR](/reference/dmr/)/[NXDN](/reference/nxdn/) voice.
**Great where analog lives:** air, marine, rail, racing, weather, and analog
police/fire. **500 channels + [Close Call](/reference/police-scanner/)** to grab a
nearby transmission you never programmed. **~$120.** Check your county on
[RadioReference](/reference/radioreference/) first — if it went digital, look at a
[digital scanner](/best-police-scanners/) or [SDR + GopherTrunk](/police-scanner-vs-sdr/).
</div>

## Overview

The SR30C is Uniden's simplest modern handheld: a pocket radio with a keypad, a
backlit screen, 500 channels, and no computer required to start listening. Its
headline features are **Close Call**, which detects and jumps to a strong signal
from a transmitter near you, and **Turbo Search**, which sweeps a frequency range
quickly to find active channels.

What it is **not** is a digital scanner. There is no P25 decoder inside — it demod­
ulates analog FM and that is all. That single fact decides whether the SR30C is
right for you.

## What it hears — and what it can't

- **Hears (analog):** conventional analog police, fire, and EMS; civilian and
  military **aviation**; **marine VHF**; **railroad**; **auto racing / NASCAR**;
  business two-way; ham; and **NOAA weather**. These services are still largely
  analog, so the SR30C covers them fully.
- **Cannot hear (digital):** any [P25 Phase I/II](/reference/p25-phase-2/),
  [DMR](/reference/dmr/), or [NXDN](/reference/nxdn/) voice. On those channels a
  digital transmission sounds like a buzzing "digital garble" — the radio has no
  vocoder to turn the bitstream back into speech.
- **Cannot follow trunking.** The SR30C is not a
  [trunking scanner](/reference/trunking-scanner/); it cannot read a
  [control channel](/reference/control-channel/) and follow talkgroups even on an
  analog trunked system.

> **Check before you buy.** Look up your city or county on
> [RadioReference](/reference/radioreference/). If the police/fire columns say P25,
> DMR, or NXDN, the SR30C will not hear them — buy a
> [digital scanner](/best-police-scanners/) instead.

## Programming

Two ways in:

1. **Keypad entry** for a handful of known frequencies — fine for a race weekend
   or an airport visit.
2. **PC software over USB.** Uniden's free application lets you load large
   frequency lists (for example, exports from RadioReference) far faster than
   typing. There is no ZIP-code database like the digital
   [HomePatrol](/reference/police-scanner/)-based models.

## GopherTrunk alternative

If everything you monitor is analog and you want a pocketable radio, the SR30C is a
fine, cheap choice. But if any of your agencies have gone **digital**, no amount of
programming will help — you need a decoder the SR30C doesn't have.

That is where **GopherTrunk** comes in. A ~$30 [RTL-SDR](/reference/rtl-sdr/) dongle
plus free, open-source GopherTrunk decodes [P25](/reference/project-25/),
[DMR](/reference/dmr/), [NXDN](/reference/nxdn/), and follows
[trunked systems](/reference/trunked-radio/) the SR30C can't — and it
**records, logs, and timestamps every call**. It trades pocket portability for
capability and data. See the honest
[police scanner vs GopherTrunk](/police-scanner-vs-sdr/) comparison, or
[download GopherTrunk](/downloads.html) and pair it with a cheap dongle before you
decide.

Neither the SR30C nor GopherTrunk — nor any scanner or SDR — can decode
[AES-encrypted](/police-scanner-encryption/) traffic. That is a hard legal and
cryptographic limit for everyone.

## Who it's for

- **Buy the SR30C** if your area is still analog and you want a simple, portable,
  no-PC scanner for police/fire, air, marine, rail, or racing.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B07S9H8YH3?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Consider the [BC125AT](/reference/uniden-bc125at/)** for the same analog job
  with alpha-tagging and a stronger aviation reputation.
- **Skip it** for a [digital scanner](/best-police-scanners/) or a free
  [SDR + GopherTrunk](/police-scanner-vs-sdr/) if your agencies run P25/DMR/NXDN.

## Bottom line

The SR30C is a good analog handheld at a low price — perfect for
[cheap analog listening](/cheap-police-scanner/) at the airport, marina, trackside,
or in an area that never went digital. Just confirm your agencies are still analog
first; if they aren't, it will hear only static, and a
[digital scanner](/best-police-scanners/) or
[GopherTrunk](/police-scanner-vs-sdr/) is the answer.

## Sources

[^uniden]: [Uniden SR30C product page](https://www.uniden.com/products/sr30c) — Uniden America, on channel capacity, Close Call, Turbo Search, and PC programming (analog-only receiver).
