---
slug: uniden-bcd436hp
title: Uniden BCD436HP
entry_type: hardware
category: consumer-scanners
description: "The Uniden BCD436HP is a HomePatrol-database handheld police scanner you program from your ZIP code — the best beginner digital handheld, decoding P25 Phase I/II with TrunkTracker V and S.A.M.E."
keywords: Uniden BCD436HP, BCD436HP scanner, HomePatrol handheld, ZIP code scanner, beginner police scanner, P25 Phase 2 handheld, TrunkTracker V, Uniden BCD436HP review
aka: [BCD436HP]
autolink: true
affiliate: true
product:
  name: "Uniden BCD436HP"
  brand: Uniden
  category: Police scanner (handheld)
  lowPrice: "479"
  highPrice: "559"
  url: https://www.amazon.com/dp/B00I33XDAK?tag=gophertrunk-20
infobox:
  - { label: Type, value: Digital handheld scanner }
  - { label: Modes, value: "P25 P1/P2, TrunkTracker V" }
  - { label: Simulcast, value: "Fair (conventional front end)" }
  - { label: Programming, value: "ZIP code / HomePatrol DB" }
  - { label: Extras, value: "S.A.M.E. weather alert, Close Call, microSD" }
  - { label: Price, value: around $520 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B00I33XDAK?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [uniden-bcd536hp, uniden-sds100, uniden-bcd325p2, police-scanner, p25-phase-2, simulcast]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://uniden.com/products/bcd436hp
faq:
  - q: "Is the Uniden BCD436HP good for beginners?"
    a: "It is the best beginner digital handheld. You program it by entering your ZIP code against Uniden's built-in HomePatrol database, so it loads nearby systems with no frequency lists and no PC — the gentlest possible on-ramp to P25 scanning."
  - q: "Does the BCD436HP handle simulcast?"
    a: "It uses a conventional front end, so simulcast performance is fair — good in many areas but not immune to P25 Phase II simulcast distortion. If your metro is simulcast, the True I/Q SDS100 handheld is the one that decodes it cleanly."
  - q: "Does the BCD436HP decode DMR and NXDN?"
    a: "No — it focuses on P25 Phase I/II plus Motorola/EDACS/LTR and analog via TrunkTracker V. For DMR and NXDN, look at an SDS-series Uniden, a Whistler TRX, or a free SDR running GopherTrunk. It cannot decode AES encryption — nothing consumer can."
  - q: "BCD436HP vs BCD536HP — what's the difference?"
    a: "Same HomePatrol database, ZIP-code programming and decode class. The BCD436HP is the battery-powered handheld; the BCD536HP is the base/mobile unit with built-in Wi-Fi. Choose by form factor."
  - q: "Can I get the same thing for free?"
    a: "A ~$30 RTL-SDR plus free GopherTrunk decodes P25 (and DMR/NXDN the BCD436HP can't) and records every call. The BCD436HP wins on pocket portability, instant ZIP-code setup, and needing no PC. See the honest comparison."
---
**The Uniden BCD436HP** is a digital handheld police scanner built around Uniden's
**HomePatrol database**: you enter your ZIP code and it loads every known nearby
system automatically.[^uniden] It follows and decodes
[P25 Phase I and II](/reference/p25-phase-2/) plus the analog trunking formats via
**TrunkTracker V**, with **Close Call** near-signal capture and S.A.M.E. weather
alerts built in. It is the easiest digital handheld for a beginner.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B00I33XDAK?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Best beginner handheld.** Program the BCD436HP by
[ZIP code](/reference/police-scanner/) against Uniden's
[HomePatrol](/reference/uniden-home-patrol-2/) database — no frequency lists, no PC.
**Decodes [P25 P1/P2](/reference/p25-phase-2/)** with **TrunkTracker V**. **Fair
[simulcast](/reference/simulcast/)** — not True I/Q, so a hard simulcast metro wants
the [SDS100](/reference/uniden-sds100/). **~$520.** **No encryption** — like every
scanner (and every SDR), it can't decode [AES](/police-scanner-encryption/). For a
free path, compare it to [GopherTrunk](/police-scanner-vs-sdr/).
</div>

## Overview

The BCD436HP is the handheld member of Uniden's HomePatrol-database family; its
base/mobile twin is the [BCD536HP](/reference/uniden-bcd536hp/). Its whole pitch is
**ease of setup for a first digital scanner**. Enter your ZIP code and Uniden's
built-in database loads every known nearby system — no frequency lists to hunt down,
no PC required, no programming cables. That makes it the standard recommendation for
someone buying their first [P25](/reference/project-25/)-capable handheld.

What it is **not** is a simulcast specialist. Its conventional front end gives *fair*
[simulcast](/reference/simulcast/) performance — acceptable in many areas, but it can
distort on the toughest P25 Phase II simulcast systems, where only the True I/Q
[SDS100](/reference/uniden-sds100/) holds a clean handheld lock.

## Modes &amp; systems it decodes

- **[P25 Phase I &amp; II](/reference/p25-phase-2/)** — the dominant US public-safety
  standard, conventional and [trunked](/reference/trunked-radio/).
- **Motorola, EDACS, LTR** analog trunking and conventional analog FM, all followed
  by **TrunkTracker V**.
- **Weather (S.A.M.E. alerts), air, marine, rail, and CB/business** conventional
  channels, plus **Close Call** to grab a nearby transmitter you don't have listed.
- **No [DMR](/reference/dmr/)/[NXDN](/reference/nxdn/)** — for those, step up to an
  SDS-series Uniden, a Whistler TRX, or a free SDR.

It **cannot** decode [encrypted](/police-scanner-encryption/)
[talkgroups](/reference/talkgroup/) — a hard limit for all consumer hardware and all
SDRs.

## Programming

Three ways in, easiest first:

1. **ZIP code / HomePatrol database.** Enter your location and the BCD436HP loads
   every known nearby system — the fastest start for a beginner.
2. **Sentinel software + microSD card** for favorites lists, firmware, and database
   updates from a PC.
3. **Manual** frequency/[talkgroup](/reference/talkgroup/) entry for anything not in
   the database.

## GopherTrunk alternative

The BCD436HP is the *turnkey* answer. **GopherTrunk** is the *free* one. An
[RTL-SDR](/reference/rtl-sdr/) (~$30) running GopherTrunk decodes the same
[P25](/reference/project-25/) — and [DMR](/reference/dmr/) and
[NXDN](/reference/nxdn/) the BCD436HP can't — and adds what no scanner does: it
**records, logs and timestamps every call**, follows **unlimited**
[talkgroups](/reference/talkgroup/) and systems at once, and streams to a **web
console** you can reach remotely.

Where the BCD436HP still wins: **no PC required, pocket portability, and instant
ZIP-code setup**. For a beginner who wants to type a ZIP and listen, that simplicity
is worth a lot. The honest head-to-head is in
[police scanner vs GopherTrunk](/police-scanner-vs-sdr/); to try the free path,
[download GopherTrunk](/downloads.html) and pair it with a $30
[dongle](/reference/rtl-sdr/) first.

## Who it's for

- **Buy the BCD436HP** if it's your first digital handheld and you want the simplest
  ZIP-code setup, in an area that isn't hard simulcast.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B00I33XDAK?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [SDS100](/reference/uniden-sds100/)** instead if you need True I/Q
  simulcast in a handheld, or the [BCD536HP](/reference/uniden-bcd536hp/) for a
  base/mobile with Wi-Fi.
- **Save money** with a [BCD325P2](/reference/uniden-bcd325p2/) if you'll program
  manually, or go free with a [SDR + GopherTrunk](/police-scanner-vs-sdr/). New to
  this? Start at [best police scanners](/best-police-scanners/).

## Sources

[^uniden]: [Uniden BCD436HP product page](https://uniden.com/products/bcd436hp) — Uniden America, on HomePatrol database, ZIP-code programming, TrunkTracker V, Close Call, supported modes, and S.A.M.E. weather alerts.
