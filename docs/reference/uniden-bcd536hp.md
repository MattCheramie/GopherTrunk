---
slug: uniden-bcd536hp
title: Uniden BCD536HP
entry_type: hardware
category: consumer-scanners
description: "The Uniden BCD536HP is a HomePatrol-database base/mobile police scanner with built-in Wi-Fi and ZIP-code programming — the easiest base scanner to set up, decoding P25 Phase I/II and controllable."
keywords: Uniden BCD536HP, BCD536HP scanner, HomePatrol base scanner, Wi-Fi police scanner, ZIP code scanner, P25 Phase 2 base scanner, TrunkTracker V, Uniden BCD536HP review
aka: [BCD536HP]
autolink: true
affiliate: true
product:
  name: "Uniden BCD536HP"
  brand: Uniden
  category: Police scanner (base/mobile)
  lowPrice: "509"
  highPrice: "589"
  url: https://www.amazon.com/dp/B00HZOW5K2?tag=gophertrunk-20
infobox:
  - { label: Type, value: Digital base/mobile scanner }
  - { label: Modes, value: "P25 P1/P2, TrunkTracker V" }
  - { label: Simulcast, value: "Fair (conventional front end)" }
  - { label: Programming, value: "ZIP code / HomePatrol DB / Wi-Fi" }
  - { label: Extras, value: "Built-in Wi-Fi, phone app, S.A.M.E. weather" }
  - { label: Price, value: around $550 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B00HZOW5K2?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [uniden-bcd436hp, uniden-sds200, uniden-bcd996p2, police-scanner, p25-phase-2, simulcast]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://uniden.com/products/bcd536hp
faq:
  - q: "What makes the Uniden BCD536HP easy to use?"
    a: "Two things: you program it by entering your ZIP code against Uniden's built-in HomePatrol database, and it has built-in Wi-Fi so you can monitor and control it from a phone or tablet app. That combination makes it the easiest base scanner to set up and live with."
  - q: "Does the BCD536HP handle simulcast?"
    a: "It uses a conventional front end, so its simulcast performance is fair — fine in many areas but not immune to the distortion of P25 Phase II simulcast systems. If your metro runs simulcast, the True I/Q SDS200 is the scanner that decodes it cleanly."
  - q: "Does the BCD536HP decode DMR and NXDN?"
    a: "The BCD536HP focuses on P25 Phase I/II and the analog and Motorola/EDACS/LTR trunking formats via TrunkTracker V. For DMR and NXDN you want an SDS-series Uniden, a Whistler TRX, or a free SDR running GopherTrunk. It cannot decode AES encryption — nothing consumer can."
  - q: "What is the difference between the BCD536HP and the BCD436HP?"
    a: "Same HomePatrol database and ZIP-code programming and the same decode class. The BCD536HP is the base/mobile unit with built-in Wi-Fi; the BCD436HP is the battery-powered handheld version. Choose by form factor."
  - q: "Can I get the same thing for free?"
    a: "A ~$30 RTL-SDR plus free GopherTrunk decodes P25 and records every call, with a web console you reach from any device — a superset of the BCD536HP's Wi-Fi app. The BCD536HP wins on turnkey ZIP-code setup and needing no PC. See the comparison."
---
**The Uniden BCD536HP** is a digital base/mobile police scanner built around
Uniden's **HomePatrol database** and **built-in Wi-Fi**: you program it from your
ZIP code and control it from a phone app.[^uniden] It follows and decodes
[P25 Phase I and II](/reference/p25-phase-2/) and the analog trunking formats via
**TrunkTracker V**, with S.A.M.E. weather alerts built in.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B00HZOW5K2?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Easiest base scanner.** Program the BCD536HP by [ZIP code](/reference/police-scanner/)
against Uniden's [HomePatrol](/reference/uniden-home-patrol-2/) database, then monitor
and control it over **built-in Wi-Fi** from your phone. **Decodes
[P25 P1/P2](/reference/p25-phase-2/)** with **TrunkTracker V**. **Fair
[simulcast](/reference/simulcast/)** — not True I/Q, so a hard simulcast metro wants
the [SDS200](/reference/uniden-sds200/). **~$550.** **No encryption** — like every
scanner (and every SDR), it can't decode [AES](/police-scanner-encryption/). For a
free path, compare it to [GopherTrunk](/police-scanner-vs-sdr/).
</div>

## Overview

The BCD536HP is the base/mobile member of Uniden's HomePatrol-database family; its
handheld twin is the [BCD436HP](/reference/uniden-bcd436hp/). Its whole pitch is
**ease of setup**. Enter your ZIP code and Uniden's built-in database loads every
known nearby system automatically — no frequency lists, no PC required. Add
**built-in Wi-Fi** and a companion phone app, and you can put the scanner on a shelf
by the antenna and listen from anywhere in the house on a tablet.

What it is **not** is a simulcast specialist. Its conventional front end gives *fair*
[simulcast](/reference/simulcast/) performance — acceptable in many areas, but it can
distort on the toughest P25 Phase II simulcast systems, where only the True I/Q
[SDS200](/reference/uniden-sds200/) and [SDS100](/reference/uniden-sds100/) hold a
clean lock.

## Modes &amp; systems it decodes

- **[P25 Phase I &amp; II](/reference/p25-phase-2/)** — the dominant US public-safety
  standard, conventional and [trunked](/reference/trunked-radio/).
- **Motorola, EDACS, LTR** analog trunking and conventional analog FM, all followed
  by **TrunkTracker V**.
- **Weather (S.A.M.E. alerts), air, marine, rail, and CB/business** conventional
  channels.
- **No [DMR](/reference/dmr/)/[NXDN](/reference/nxdn/)** — for those digital modes,
  step up to an SDS-series Uniden, a Whistler TRX, or a free SDR.

It **cannot** decode [encrypted](/police-scanner-encryption/)
[talkgroups](/reference/talkgroup/) — a hard limit for all consumer hardware and all
SDRs.

## Programming

Three ways in, easiest first:

1. **ZIP code / HomePatrol database.** Enter your location and the BCD536HP loads
   every known nearby system — the fastest start for a beginner.
2. **Wi-Fi + Sentinel software** for favorites lists, firmware, and database updates,
   plus remote monitoring from the phone app.
3. **Manual** frequency/[talkgroup](/reference/talkgroup/) entry for anything not in
   the database.

## GopherTrunk alternative

The BCD536HP is the *turnkey* answer. **GopherTrunk** is the *free* one. An
[RTL-SDR](/reference/rtl-sdr/) (~$30) running GopherTrunk decodes the same
[P25](/reference/project-25/) — and [DMR](/reference/dmr/) and
[NXDN](/reference/nxdn/) the BCD536HP can't — while adding what no scanner does: it
**records, logs and timestamps every call**, follows **unlimited**
[talkgroups](/reference/talkgroup/) and systems at once, and streams to a **web
console** you reach from any device. That web console is effectively a bigger version
of the BCD536HP's Wi-Fi app, with recording built in.

Where the BCD536HP still wins: **no PC required and instant ZIP-code setup** — plug
it in, type your ZIP, and listen. The honest head-to-head is in
[police scanner vs GopherTrunk](/police-scanner-vs-sdr/); to try the free path,
[download GopherTrunk](/downloads.html) and pair it with a $30
[dongle](/reference/rtl-sdr/) first.

## Who it's for

- **Buy the BCD536HP** if you want the simplest base setup — ZIP-code programming
  plus phone control over Wi-Fi — and your area is not a hard simulcast metro.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B00HZOW5K2?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [BCD436HP](/reference/uniden-bcd436hp/)** instead if you need it handheld,
  or the [SDS200](/reference/uniden-sds200/) if you need True I/Q simulcast.
- **Skip it** for a [BCD996P2](/reference/uniden-bcd996p2/) (cheaper, if you don't
  need the database/Wi-Fi) or a free [SDR + GopherTrunk](/police-scanner-vs-sdr/).
  Weighing options? See [best police scanners](/best-police-scanners/).

## Sources

[^uniden]: [Uniden BCD536HP product page](https://uniden.com/products/bcd536hp) — Uniden America, on HomePatrol database, built-in Wi-Fi, TrunkTracker V, supported modes, and S.A.M.E. weather alerts.
