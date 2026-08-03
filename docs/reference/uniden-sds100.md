---
slug: uniden-sds100
title: Uniden SDS100
entry_type: hardware
category: consumer-scanners
description: "The Uniden SDS100 is a True I/Q handheld police scanner with the best simulcast decode performance you can carry — P25 Phase I/II, DMR, NXDN and ProVoice, weatherproof and programmed from your ZIP."
keywords: Uniden SDS100, SDS100 scanner, True I/Q handheld, P25 Phase 2 handheld scanner, simulcast handheld scanner, best handheld police scanner, Uniden SDS100 review
aka: [SDS100]
autolink: true
affiliate: true
product:
  name: "Uniden SDS100"
  brand: Uniden
  category: Police scanner (handheld)
  lowPrice: "599"
  highPrice: "699"
  url: https://www.amazon.com/dp/B07DK26FDN?tag=gophertrunk-20
infobox:
  - { label: Type, value: Digital handheld scanner }
  - { label: Modes, value: "P25 P1/P2, DMR, NXDN, ProVoice, X2-TDMA" }
  - { label: Simulcast, value: "True I/Q — best in class" }
  - { label: Programming, value: "ZIP code / HomePatrol DB / SD card" }
  - { label: Extras, value: "Weatherproof JIS4/IPX4, GPS-ready, color LCD" }
  - { label: Price, value: around $650 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B07DK26FDN?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [uniden-sds200, uniden-sds150, uniden-bcd436hp, police-scanner, p25-phase-2, simulcast]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://uniden.com/products/sds100
faq:
  - q: "Is the Uniden SDS100 the best handheld police scanner?"
    a: "For simulcast areas, yes. The SDS100 is the only handheld with a True I/Q software-defined front end, so it decodes distorted P25 Phase II simulcast that lesser handhelds turn to garble. If your area is not simulcast, a cheaper BCD436HP does the same job."
  - q: "What is the difference between the SDS100 and the SDS200?"
    a: "Same decode engine and same True I/Q front end. The SDS100 is the battery-powered, weatherproof handheld; the SDS200 is the mains/12V base and mobile version with a bigger screen and Ethernet. Buy the SDS100 if you need to carry it."
  - q: "Does the SDS100 decode DMR and NXDN?"
    a: "Yes — P25 Phase I/II, DMR, NXDN, ProVoice and X2-TDMA are all supported. Some digital modes have historically required a paid Uniden upgrade key, so check the current listing. It cannot decode AES-encrypted traffic — no consumer scanner or SDR can."
  - q: "Is the SDS100 weatherproof?"
    a: "It carries a JIS4/IPX4 splash-resistant rating, so it shrugs off rain and spray at a fireground or trackside. It is not a submersible dive radio, but it is built for real outdoor field use."
  - q: "Can I get the same decoding for free?"
    a: "A ~$30 RTL-SDR plus free GopherTrunk decodes the same P25/DMR/NXDN and records every call. The SDS100's front end still edges out a cheap dongle on the hardest simulcast, and it needs no PC. See our honest comparison."
---
**The Uniden SDS100** is a digital handheld police scanner whose **True I/Q**
software-defined front end gives it the best [simulcast](/reference/simulcast/)
decode performance of any scanner you can carry.[^uniden] It follows and decodes
[P25 Phase I and II](/reference/p25-phase-2/), [DMR](/reference/dmr/),
[NXDN](/reference/nxdn/) and ProVoice, is weatherproof, and programs from your ZIP
code against Uniden's built-in USA/Canada database.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B07DK26FDN?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Best handheld for simulcast.** The SDS100's [True I/Q](/reference/software-defined-radio/)
processing decodes the distorted [Phase II simulcast](/reference/simulcast/)
signals that defeat ordinary handhelds. **Battery-portable and weatherproof**
(JIS4/IPX4), GPS-ready; the [SDS200](/reference/uniden-sds200/) is the base twin.
**Programs from your ZIP.** **~$650.** **No encryption** — like every scanner (and
every SDR), it can't decode [AES](/police-scanner-encryption/). For a free path,
compare it to [GopherTrunk](/police-scanner-vs-sdr/).
</div>

## Overview

The SDS100 is the handheld half of Uniden's flagship pair; the
[SDS200](/reference/uniden-sds200/) is the base/mobile sibling. Both share the one
feature that matters most in a tough metro RF environment: a **software-defined
[True I/Q](/reference/software-defined-radio/) receiver** instead of a conventional
discriminator. On difficult [simulcast](/reference/simulcast/) systems — where
several transmitters send the same signal and the overlapping waves smear the
constellation — the SDS-series front end recovers voice that a
[BCD436HP](/reference/uniden-bcd436hp/) or a
[BCD325P2](/reference/uniden-bcd325p2/) will drop into static.

Because it is a handheld, the SDS100 adds a rechargeable battery, a
splash-resistant JIS4/IPX4 body, and GPS-ready location scanning that mutes systems
outside your range as you drive. If your county is **not** simulcast, that front-end
advantage is moot and a cheaper handheld — or a free SDR — does the same job.

## Modes &amp; systems it decodes

- **[P25 Phase I &amp; II](/reference/p25-phase-2/)** — the dominant US public-safety
  standard, conventional and [trunked](/reference/trunked-radio/).
- **[DMR](/reference/dmr/) and [NXDN](/reference/nxdn/)** — business and some public
  systems (some modes have historically needed a paid upgrade key).
- **Motorola, EDACS, LTR** analog trunking; conventional analog FM; **ProVoice**
  and **X2-TDMA**.
- **Weather (S.A.M.E. alerts), air, marine, rail, and CB/business** conventional
  channels.

It **cannot** decode [encrypted](/police-scanner-encryption/)
[talkgroups](/reference/talkgroup/) — that is a hard limit for all consumer hardware
and all SDRs, GopherTrunk included.

## Programming

Three ways in, easiest first:

1. **ZIP code / HomePatrol database.** Enter your location and the SDS100 loads
   every known nearby system — the fastest start for a beginner.
2. **Sentinel software + microSD card** for favorites lists, firmware, and database
   updates from a PC.
3. **Manual** frequency/talkgroup entry for anything not in the database.

## GopherTrunk alternative

The SDS100 is the *turnkey, pocketable* answer. **GopherTrunk** is the *free* one.
An [RTL-SDR](/reference/rtl-sdr/) (~$30) or an Airspy running GopherTrunk decodes
the same [P25](/reference/project-25/), DMR and NXDN, and adds what no scanner does:
it **records, logs and timestamps every call**, follows **unlimited**
[talkgroups](/reference/talkgroup/) and systems at once, and streams to a **web
console** you can reach remotely. GopherTrunk's software
[CMA equalizer](/reference/simulcast/) also cleans up many simulcast systems.

Where the SDS100 still wins: **no PC required, true pocket portability, a
weatherproof body and a purpose-built front end** that edges out a cheap dongle on
the hardest simulcast. The honest, detailed head-to-head is in
[police scanner vs GopherTrunk](/police-scanner-vs-sdr/); if you want to try the
free path first, [download GopherTrunk](/downloads.html) and pair it with a $30
dongle before spending $650.

## Who it's for

- **Buy the SDS100** if you need the best decode you can carry into a
  [simulcast](/reference/simulcast/) metro — trackside, on foot, or in the car.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B07DK26FDN?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [SDS200](/reference/uniden-sds200/)** instead for a fixed base or vehicle
  install, or the [SDS150](/reference/uniden-sds150/) for Uniden's newest touchscreen
  flagship.
- **Skip it** for a [BCD436HP](/reference/uniden-bcd436hp/) (non-simulcast areas) or
  a free [SDR + GopherTrunk](/police-scanner-vs-sdr/) (you have a PC and want
  recording/logging). Still deciding? See [best police scanners](/best-police-scanners/).

## Sources

[^uniden]: [Uniden SDS100 product page](https://uniden.com/products/sds100) — Uniden America, on True I/Q architecture, supported modes, weatherproof rating, and HomePatrol database programming.
