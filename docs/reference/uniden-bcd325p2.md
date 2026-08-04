---
slug: uniden-bcd325p2
title: Uniden BCD325P2
entry_type: hardware
category: consumer-scanners
description: "The Uniden BCD325P2 is Uniden's cheapest P25 Phase I/II digital handheld — 25,000 channels, TrunkTracker V and Close Call, programmed manually or with Sentinel software. The handheld version of the."
keywords: Uniden BCD325P2, BCD325P2 scanner, cheapest P25 handheld, P25 Phase 2 handheld, TrunkTracker V, Close Call handheld, manual programming scanner, Uniden BCD325P2 review
aka: [BCD325P2]
autolink: true
affiliate: true
product:
  name: "Uniden BCD325P2"
  brand: Uniden
  category: Police scanner (handheld)
  lowPrice: "349"
  highPrice: "409"
  url: https://www.amazon.com/dp/B00V91IN62?tag=gophertrunk-20
infobox:
  - { label: Type, value: Digital handheld scanner }
  - { label: Modes, value: "P25 P1/P2, TrunkTracker V" }
  - { label: Simulcast, value: "Fair (conventional front end)" }
  - { label: Programming, value: "Manual / Sentinel software (no ZIP DB)" }
  - { label: Extras, value: "25,000 channels, Close Call, S.A.M.E." }
  - { label: Price, value: around $380 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B00V91IN62?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [uniden-bcd996p2, uniden-bcd436hp, uniden-sds100, police-scanner, p25-phase-2, simulcast]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://uniden.com/products/bcd325p2
faq:
  - q: "Is the BCD325P2 the cheapest Uniden digital handheld?"
    a: "Yes — at around $380 it's the lowest-priced Uniden that decodes P25 Phase I/II. It skips the HomePatrol ZIP database, so you program it manually or with free Sentinel software, which is how it hits that price."
  - q: "What's the difference between the BCD325P2 and the BCD436HP?"
    a: "The BCD436HP adds Uniden's HomePatrol database, so you can program it from your ZIP code. The BCD325P2 has no database — you program it manually or with Sentinel software — which makes it cheaper. Same P25 Phase I/II decode class."
  - q: "Is the BCD325P2 just the handheld BCD996P2?"
    a: "Essentially yes. The BCD325P2 is the battery-powered handheld with the same P25 Phase I/II decoding, TrunkTracker V, 25,000 channels, Close Call, and manual/Sentinel programming as the BCD996P2 base/mobile. Choose by form factor."
  - q: "Does the BCD325P2 decode DMR and NXDN?"
    a: "No — it covers P25 Phase I/II plus Motorola/EDACS/LTR and analog via TrunkTracker V. For DMR and NXDN you want an SDS-series Uniden, a Whistler TRX, or a free SDR running GopherTrunk. It cannot decode AES encryption — nothing consumer can."
  - q: "Is there a cheaper way to hear P25?"
    a: "A ~$30 RTL-SDR plus free GopherTrunk decodes P25 (and DMR/NXDN) and records every call — far cheaper than any scanner if you already have a PC. The BCD325P2 wins on pocket portability and needing no computer."
---
**The Uniden BCD325P2** is Uniden's most affordable digital handheld: it follows and
decodes [P25 Phase I and II](/reference/p25-phase-2/) with **TrunkTracker V**, holds
**25,000 channels**, and includes **Close Call** near-signal capture.[^uniden] It has
**no HomePatrol ZIP database**, so you program it manually or with Uniden's free
Sentinel software — the same trade the [BCD996P2](/reference/uniden-bcd996p2/) makes,
in a pocket-sized body.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B00V91IN62?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Cheapest Uniden digital handheld.** The BCD325P2 decodes
[P25 P1/P2](/reference/p25-phase-2/) with **TrunkTracker V** and **25,000 channels**
for the least money Uniden charges — because you **program it manually or with
Sentinel software**, not from a ZIP code. **Fair
[simulcast](/reference/simulcast/)** — not True I/Q, so a hard simulcast metro wants
the [SDS100](/reference/uniden-sds100/). **~$380.** **No encryption** — like every
scanner (and every SDR), it can't decode [AES](/police-scanner-encryption/). For a
free path, compare it to [GopherTrunk](/police-scanner-vs-sdr/).
</div>

## Overview

The BCD325P2 is the handheld value pick in Uniden's digital line; think of it as the
pocket version of the [BCD996P2](/reference/uniden-bcd996p2/). It decodes the same
[P25 Phase I and II](/reference/p25-phase-2/) as pricier Unidens, but it drops the
HomePatrol database (no enter-your-ZIP shortcut) to hit the lowest price of any
Uniden digital handheld. You get a capable, battery-powered
[trunk-tracking](/reference/trunked-radio/) scanner with 25,000 channels and Close
Call — you just have to tell it what to listen to.

Like the other non-SDS Unidens, its conventional front end gives *fair*
[simulcast](/reference/simulcast/) — acceptable in many areas, but it distorts on the
toughest P25 Phase II simulcast systems, where only the True I/Q
[SDS100](/reference/uniden-sds100/) holds a clean handheld lock.

## Modes &amp; systems it decodes

- **[P25 Phase I &amp; II](/reference/p25-phase-2/)** — the dominant US public-safety
  standard, conventional and [trunked](/reference/trunked-radio/).
- **Motorola, EDACS, LTR** analog trunking and conventional analog FM, all followed
  by **TrunkTracker V** across **25,000 channels**.
- **Weather (S.A.M.E. alerts), air, marine, rail, and CB/business** conventional
  channels, plus **Close Call** to grab a nearby transmitter you don't have listed.
- **No [DMR](/reference/dmr/)/[NXDN](/reference/nxdn/)** — for those, step up to an
  SDS-series Uniden, a Whistler TRX, or a free SDR.

It **cannot** decode [encrypted](/police-scanner-encryption/)
[talkgroups](/reference/talkgroup/) — a hard limit for all consumer hardware and all
SDRs.

## Programming

No ZIP-code database, so two paths:

1. **Sentinel software (free).** Build favorites lists on a PC — frequencies,
   [talkgroups](/reference/talkgroup/), and scan settings — and load them over USB.
   The practical way to set it up.
2. **Manual entry.** Key in frequencies and talkgroups by hand from a source like
   RadioReference. Slower, but works with no PC.

The manual setup is the price you pay for the low cost — or a nudge toward a free SDR
that auto-follows systems for you.

## GopherTrunk alternative

The BCD325P2 is the *cheap pocketable* answer. **GopherTrunk** is the *free* one. An
[RTL-SDR](/reference/rtl-sdr/) (~$30) running GopherTrunk decodes the same
[P25](/reference/project-25/) — and [DMR](/reference/dmr/) and
[NXDN](/reference/nxdn/) the BCD325P2 can't — and **auto-discovers and follows**
[trunked systems](/reference/trunked-radio/) rather than making you hand-enter every
[talkgroup](/reference/talkgroup/). It also **records, logs and timestamps every
call**, follows **unlimited** systems at once, and streams to a **web console**.

Where the BCD325P2 still wins: **no PC required and true pocket portability** — it's
a self-contained radio you can clip to a belt. If you already have a laptop, though,
free GopherTrunk plus a $30 [dongle](/reference/rtl-sdr/) is cheaper still. The honest
head-to-head is in [police scanner vs GopherTrunk](/police-scanner-vs-sdr/); to try
it, [download GopherTrunk](/downloads.html) first.

## Who it's for

- **Buy the BCD325P2** if you want the cheapest P25 Phase II handheld and don't mind
  manual/Sentinel programming or need simulcast.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B00V91IN62?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [BCD436HP](/reference/uniden-bcd436hp/)** instead for ZIP-code database
  programming, or the [SDS100](/reference/uniden-sds100/) for True I/Q simulcast.
- **Go base/mobile** with the [BCD996P2](/reference/uniden-bcd996p2/), or go free with
  a [SDR + GopherTrunk](/police-scanner-vs-sdr/). New to this? See
  [best police scanners](/best-police-scanners/).

## Sources

[^uniden]: [Uniden BCD325P2 product page](https://uniden.com/products/bcd325p2) — Uniden America, on P25 Phase I/II, TrunkTracker V, 25,000 channels, Close Call, and Sentinel/manual programming.
