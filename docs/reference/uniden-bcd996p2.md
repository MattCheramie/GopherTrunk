---
slug: uniden-bcd996p2
title: Uniden BCD996P2
entry_type: hardware
category: consumer-scanners
description: "The Uniden BCD996P2 is a value-priced P25 Phase I/II base/mobile police scanner with 25,000 channels, TrunkTracker V and Close Call — no HomePatrol database, so you program it manually or with Sentinel software."
keywords: Uniden BCD996P2, BCD996P2 scanner, P25 Phase 2 base scanner, value digital scanner, TrunkTracker V, Close Call, manual programming scanner, Uniden BCD996P2 review
aka: [BCD996P2]
autolink: true
affiliate: true
product:
  name: "Uniden BCD996P2"
  brand: Uniden
  category: Police scanner (base/mobile)
  lowPrice: "439"
  highPrice: "505"
  url: https://www.amazon.com/dp/B00UJU5MUE?tag=gophertrunk-20
infobox:
  - { label: Type, value: Digital base/mobile scanner }
  - { label: Modes, value: "P25 P1/P2, TrunkTracker V" }
  - { label: Simulcast, value: "Fair (conventional front end)" }
  - { label: Programming, value: "Manual / Sentinel software (no ZIP DB)" }
  - { label: Extras, value: "25,000 channels, Close Call, S.A.M.E." }
  - { label: Price, value: around $470 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B00UJU5MUE?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [uniden-bcd325p2, uniden-bcd536hp, uniden-sds200, police-scanner, p25-phase-2, simulcast]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://uniden.com/products/bcd996p2
faq:
  - q: "Why is the Uniden BCD996P2 cheaper than the BCD536HP?"
    a: "The BCD996P2 skips the HomePatrol ZIP-code database and built-in Wi-Fi. You program it manually or with free Sentinel software instead. Same P25 Phase I/II decode class, less setup convenience — which is why it's the value pick for a digital base scanner."
  - q: "Does the BCD996P2 handle simulcast?"
    a: "It uses a conventional front end, so simulcast performance is fair — fine in many areas but not immune to P25 Phase II simulcast distortion. If your metro is simulcast, only the True I/Q SDS200 decodes it cleanly."
  - q: "How do I program the BCD996P2 without a ZIP database?"
    a: "Two ways: enter frequencies and talkgroups by hand from a source like RadioReference, or use Uniden's free Sentinel software to build favorites lists on a PC and load them over USB. There is no enter-your-ZIP shortcut like the HomePatrol models."
  - q: "Does the BCD996P2 decode DMR and NXDN?"
    a: "No — it covers P25 Phase I/II plus Motorola/EDACS/LTR and analog via TrunkTracker V. For DMR and NXDN you want an SDS-series Uniden, a Whistler TRX, or a free SDR running GopherTrunk. It cannot decode AES encryption — nothing consumer can."
  - q: "Is the BCD996P2 a good value?"
    a: "Yes — it's the best-value digital base scanner if you don't need the simulcast premium or the ZIP-code database. If you'd rather not program by hand at all, a free SDR + GopherTrunk auto-follows systems and records every call for the price of a $30 dongle."
---

**The Uniden BCD996P2** is a value-priced digital base/mobile police scanner that
follows and decodes [P25 Phase I and II](/reference/p25-phase-2/) with **TrunkTracker
V**, holds **25,000 channels**, and includes **Close Call** near-signal capture.[^uniden]
It has **no HomePatrol ZIP database**, so you program it manually or with Uniden's
free Sentinel software — which is exactly why it costs less than the database models.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B00UJU5MUE?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Best-value digital base.** The BCD996P2 decodes
[P25 P1/P2](/reference/p25-phase-2/) with **TrunkTracker V** and **25,000 channels**
for less than the HomePatrol models — because you **program it manually or with
Sentinel software**, not from a ZIP code. **Fair
[simulcast](/reference/simulcast/)** — not True I/Q, so a hard simulcast metro wants
the [SDS200](/reference/uniden-sds200/). **~$470.** **No encryption** — like every
scanner (and every SDR), it can't decode [AES](/police-scanner-encryption/). For a
free path, compare it to [GopherTrunk](/police-scanner-vs-sdr/).
</div>

## Overview

The BCD996P2 is the base/mobile value pick in Uniden's digital line; its handheld
twin is the [BCD325P2](/reference/uniden-bcd325p2/). It decodes the same
[P25 Phase I and II](/reference/p25-phase-2/) as pricier Unidens, but it drops two
convenience features to hit its price: there is **no HomePatrol database** (so no
enter-your-ZIP shortcut) and **no built-in Wi-Fi**. What you get instead is a big,
capable [trunk-tracking](/reference/trunked-radio/) base radio with 25,000 channels
and Close Call at the lowest price Uniden charges for full P25 Phase II.

Like the other non-SDS Unidens, its conventional front end gives *fair*
[simulcast](/reference/simulcast/) — acceptable in many areas, but it distorts on the
toughest P25 Phase II simulcast systems, where only the True I/Q
[SDS200](/reference/uniden-sds200/) holds a clean lock.

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

If typing in systems by hand sounds tedious, that's the trade you make for the lower
price — or a reason to look at a free SDR that auto-follows systems for you.

## GopherTrunk alternative

The BCD996P2 is the *cheap turnkey* answer. **GopherTrunk** is the *free* one. An
[RTL-SDR](/reference/rtl-sdr/) (~$30) running GopherTrunk decodes the same
[P25](/reference/project-25/) — and [DMR](/reference/dmr/) and
[NXDN](/reference/nxdn/) the BCD996P2 can't — and **auto-discovers and follows**
[trunked systems](/reference/trunked-radio/) rather than making you hand-enter
[talkgroups](/reference/talkgroup/). It also **records, logs and timestamps every
call**, follows **unlimited** systems at once, and streams to a **web console**.

Where the BCD996P2 still wins: **no PC required**, standalone base operation, and
FCC-certified turnkey hardware. But if your objection to pricier scanners was cost,
note that free GopherTrunk plus a $30 [dongle](/reference/rtl-sdr/) undercuts even
this. The honest head-to-head is in
[police scanner vs GopherTrunk](/police-scanner-vs-sdr/); to try it,
[download GopherTrunk](/downloads.html) first.

## Who it's for

- **Buy the BCD996P2** if you want the cheapest full P25 Phase II base scanner and
  you don't mind manual/Sentinel programming or need simulcast.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B00UJU5MUE?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [BCD536HP](/reference/uniden-bcd536hp/)** instead for ZIP-code database
  and Wi-Fi, or the [SDS200](/reference/uniden-sds200/) for True I/Q simulcast.
- **Go handheld** with the [BCD325P2](/reference/uniden-bcd325p2/), or go free with a
  [SDR + GopherTrunk](/police-scanner-vs-sdr/). Comparing? See
  [best police scanners](/best-police-scanners/).

## Sources

[^uniden]: [Uniden BCD996P2 product page](https://uniden.com/products/bcd996p2) — Uniden America, on P25 Phase I/II, TrunkTracker V, 25,000 channels, Close Call, and Sentinel/manual programming.
