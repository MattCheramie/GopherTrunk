---
slug: uniden-home-patrol-2
title: Uniden HomePatrol-2
entry_type: hardware
category: consumer-scanners
description: "The Uniden HomePatrol-2 is the simplest digital police scanner to use — a color touchscreen you program by ZIP code, decoding P25 Phase I/II with TrunkTracker V and S.A.M.E. weather alerts. An older but easy design."
keywords: Uniden HomePatrol-2, HomePatrol 2 scanner, easiest police scanner, touchscreen scanner, ZIP code scanner, P25 Phase 2 scanner, beginner digital scanner, Uniden HomePatrol-2 review
aka: [HomePatrol-2]
autolink: true
affiliate: true
product:
  name: "Uniden HomePatrol-2"
  brand: Uniden
  category: Police scanner (base/portable)
  lowPrice: "419"
  highPrice: "485"
  url: https://www.amazon.com/dp/B00JJY6S72?tag=gophertrunk-20
infobox:
  - { label: Type, value: Digital touchscreen scanner }
  - { label: Modes, value: "P25 P1/P2, TrunkTracker V" }
  - { label: Simulcast, value: "Fair (conventional front end)" }
  - { label: Programming, value: "ZIP code / HomePatrol DB (touchscreen)" }
  - { label: Extras, value: "Color touchscreen, S.A.M.E. weather, microSD" }
  - { label: Price, value: around $450 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B00JJY6S72?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [uniden-bcd436hp, uniden-bcd536hp, uniden-sds100, police-scanner, p25-phase-2, simulcast]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://uniden.com/products/homepatrol-2
faq:
  - q: "Is the HomePatrol-2 the easiest scanner to use?"
    a: "It's designed to be. You enter your ZIP code on the color touchscreen and it loads nearby systems from Uniden's built-in database — no menus, no frequency lists, no PC. For pure simplicity it's the gentlest digital scanner Uniden makes."
  - q: "Is the HomePatrol-2 outdated?"
    a: "It's an older design. The decode class is still current P25 Phase I/II with TrunkTracker V, but newer models like the BCD436HP handheld or the True I/Q SDS series have better front ends, more features, and — in the SDS case — far better simulcast. Buy the HomePatrol-2 for simplicity, not cutting-edge performance."
  - q: "Does the HomePatrol-2 handle simulcast?"
    a: "It uses a conventional front end, so simulcast performance is fair — okay in many areas but not immune to P25 Phase II simulcast distortion. If your metro is simulcast, the True I/Q SDS100 or SDS200 is the scanner that decodes it cleanly."
  - q: "Does the HomePatrol-2 decode DMR and NXDN?"
    a: "No — it covers P25 Phase I/II plus Motorola/EDACS/LTR and analog via TrunkTracker V. For DMR and NXDN you want an SDS-series Uniden, a Whistler TRX, or a free SDR running GopherTrunk. It cannot decode AES encryption — nothing consumer can."
  - q: "HomePatrol-2 or BCD436HP?"
    a: "Both program by ZIP code from the same database. The HomePatrol-2 has a touchscreen and is the simplest to operate; the BCD436HP is a more modern handheld with more features and Close Call. If touchscreen simplicity matters most, pick the HomePatrol-2; otherwise the BCD436HP is the newer choice."
---

**The Uniden HomePatrol-2** is the simplest digital police scanner to operate: a
**color touchscreen** you program by entering your **ZIP code** against Uniden's
built-in database.[^uniden] It follows and decodes
[P25 Phase I and II](/reference/p25-phase-2/) with **TrunkTracker V** and includes
S.A.M.E. weather alerts. It's an older design, but nothing is easier to set up.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B00JJY6S72?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Simplest digital scanner.** The HomePatrol-2 is a
[ZIP-code](/reference/police-scanner/) **touchscreen** — type your location and it
loads nearby systems from the [HomePatrol](/reference/uniden-bcd436hp/) database, no
menus or PC. **Decodes [P25 P1/P2](/reference/p25-phase-2/)** with **TrunkTracker V**.
**Older design** — newer Unidens have better front ends. **Fair
[simulcast](/reference/simulcast/)**, not True I/Q, so a hard simulcast metro wants
the [SDS100](/reference/uniden-sds100/). **~$450.** **No encryption** — like every
scanner (and every SDR), it can't decode [AES](/police-scanner-encryption/). For a
free path, compare it to [GopherTrunk](/police-scanner-vs-sdr/).
</div>

## Overview

The HomePatrol-2 is Uniden's ease-of-use flagship, built around a
touchscreen and the same **HomePatrol database** as the
[BCD436HP](/reference/uniden-bcd436hp/) and [BCD536HP](/reference/uniden-bcd536hp/).
Its entire design goal is *don't make the user learn anything*: power it on, tap in
your ZIP code, and it starts scanning every known nearby
[P25](/reference/project-25/) and analog [trunked](/reference/trunked-radio/) system.
For a non-technical buyer — a spouse, a parent, a newsroom front desk — that
touchscreen simplicity is the whole point.

The trade is that it's an **older design**. The decode class is still current
[P25 Phase I/II](/reference/p25-phase-2/), but its conventional front end gives only
*fair* [simulcast](/reference/simulcast/), and newer radios have more features and,
in the True I/Q [SDS100](/reference/uniden-sds100/)/[SDS200](/reference/uniden-sds200/),
dramatically better simulcast decode. Buy it for simplicity, not performance.

## Modes &amp; systems it decodes

- **[P25 Phase I &amp; II](/reference/p25-phase-2/)** — the dominant US public-safety
  standard, conventional and [trunked](/reference/trunked-radio/).
- **Motorola, EDACS, LTR** analog trunking and conventional analog FM, all followed
  by **TrunkTracker V**.
- **Weather (S.A.M.E. alerts), air, marine, rail, and CB/business** conventional
  channels.
- **No [DMR](/reference/dmr/)/[NXDN](/reference/nxdn/)** — for those, step up to an
  SDS-series Uniden, a Whistler TRX, or a free SDR.

It **cannot** decode [encrypted](/police-scanner-encryption/)
[talkgroups](/reference/talkgroup/) — a hard limit for all consumer hardware and all
SDRs.

## Programming

Built for one easy path:

1. **ZIP code / HomePatrol database (touchscreen).** Tap in your location and the
   HomePatrol-2 loads every known nearby system — the simplest setup of any digital
   scanner, no computer needed.
2. **Sentinel software + microSD card** for favorites lists, firmware, and database
   updates from a PC, when you want more control.
3. **Manual** frequency/[talkgroup](/reference/talkgroup/) entry for anything not in
   the database.

## GopherTrunk alternative

The HomePatrol-2 is the *simplest turnkey* answer. **GopherTrunk** is the *free* one.
An [RTL-SDR](/reference/rtl-sdr/) (~$30) running GopherTrunk decodes the same
[P25](/reference/project-25/) — and [DMR](/reference/dmr/) and
[NXDN](/reference/nxdn/) the HomePatrol-2 can't — and adds what no scanner does: it
**records, logs and timestamps every call**, follows **unlimited**
[talkgroups](/reference/talkgroup/) and systems at once, and streams to a **web
console** you can reach remotely.

Where the HomePatrol-2 still wins: **no PC required and the gentlest possible
touchscreen setup** — for a buyer who will never touch a computer, that matters more
than any feature list. GopherTrunk asks you to run software; the HomePatrol-2 asks you
to type a ZIP. The honest head-to-head is in
[police scanner vs GopherTrunk](/police-scanner-vs-sdr/); to try the free path,
[download GopherTrunk](/downloads.html) and pair it with a $30
[dongle](/reference/rtl-sdr/) first.

## Who it's for

- **Buy the HomePatrol-2** if absolute simplicity is the goal — a touchscreen, a ZIP
  code, done — and your area isn't hard simulcast.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B00JJY6S72?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [BCD436HP](/reference/uniden-bcd436hp/)** instead for a newer handheld
  with the same ZIP-code database, or the [SDS100](/reference/uniden-sds100/) for True
  I/Q simulcast.
- **Go free** with a [SDR + GopherTrunk](/police-scanner-vs-sdr/) if you have a PC and
  want recording/logging. Comparing options? See
  [best police scanners](/best-police-scanners/).

## Sources

[^uniden]: [Uniden HomePatrol-2 product page](https://uniden.com/products/homepatrol-2) — Uniden America, on the touchscreen HomePatrol database, ZIP-code programming, P25 Phase I/II, TrunkTracker V, and S.A.M.E. weather alerts.
