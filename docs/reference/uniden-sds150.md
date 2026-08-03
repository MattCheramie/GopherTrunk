---
slug: uniden-sds150
title: Uniden SDS150
entry_type: hardware
category: consumer-scanners
description: "The Uniden SDS150 is Uniden's newest and most advanced handheld SDR scanner — True I/Q simulcast decode, a large color touchscreen and built-in GPS, decoding P25 Phase I/II, DMR and NXDN."
keywords: Uniden SDS150, SDS150 scanner, True I/Q handheld, touchscreen police scanner, P25 Phase 2 handheld, simulcast handheld scanner, newest Uniden scanner, Uniden SDS150 review
aka: [SDS150]
autolink: true
affiliate: true
product:
  name: "Uniden SDS150"
  brand: Uniden
  category: Police scanner (handheld)
  lowPrice: "739"
  highPrice: "859"
  url: https://www.amazon.com/dp/B0FXNFPB4C?tag=gophertrunk-20
infobox:
  - { label: Type, value: Digital handheld scanner (flagship) }
  - { label: Modes, value: "P25 P1/P2, DMR, NXDN" }
  - { label: Simulcast, value: "True I/Q — best in class" }
  - { label: Programming, value: "ZIP code / HomePatrol DB / SD card" }
  - { label: Extras, value: "Large color touchscreen, built-in GPS" }
  - { label: Price, value: around $800 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B0FXNFPB4C?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [uniden-sds100, uniden-sds200, uniden-bcd436hp, police-scanner, p25-phase-2, simulcast]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://uniden.com/products/sds150
faq:
  - q: "What is the Uniden SDS150?"
    a: "The SDS150 is Uniden's newest and most advanced handheld scanner. It keeps the True I/Q software-defined front end that makes the SDS line the best at simulcast, and adds a large color touchscreen and built-in GPS. It decodes P25 Phase I/II, DMR and NXDN."
  - q: "SDS150 vs SDS100 — which handheld should I buy?"
    a: "Both use the same True I/Q simulcast front end. The SDS150 is the newer, top-of-line model with a bigger touchscreen and built-in GPS at a higher price. The SDS100 is the proven value pick for simulcast — buy it to save money, buy the SDS150 for the newest hardware and touchscreen."
  - q: "Does the SDS150 decode DMR and NXDN?"
    a: "Yes — P25 Phase I/II, DMR and NXDN are supported. As with other Uniden models, some digital modes may require a paid upgrade key, so check the current listing. It cannot decode AES-encrypted traffic — no consumer scanner or SDR can."
  - q: "Does the SDS150 have GPS?"
    a: "Yes, GPS is built in, so location-based scanning mutes systems outside your range automatically as you travel — no external puck required, unlike the GPS-ready SDS100."
  - q: "Is the SDS150 worth the extra cost over the SDS100?"
    a: "If you want the newest hardware, a large touchscreen and built-in GPS, yes. If your goal is simply the best simulcast decode you can carry for the least money, the SDS100 delivers the same True I/Q engine for less."
---

**The Uniden SDS150** is Uniden's newest and most advanced handheld scanner: a
software-defined [True I/Q](/reference/software-defined-radio/) receiver with the
best [simulcast](/reference/simulcast/) decode in the consumer market, now wrapped
around a large color **touchscreen** and **built-in GPS**.[^uniden] It follows and
decodes [P25 Phase I and II](/reference/p25-phase-2/), [DMR](/reference/dmr/) and
[NXDN](/reference/nxdn/), and programs from your ZIP code against Uniden's built-in
USA/Canada database.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0FXNFPB4C?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Top-of-line handheld.** The SDS150 is Uniden's newest flagship — same
[True I/Q](/reference/software-defined-radio/) simulcast front end as the
[SDS100](/reference/uniden-sds100/), plus a large **touchscreen** and **built-in
GPS**. **Priciest handheld at ~$800**; the SDS100 is the value simulcast pick.
**Decodes [P25 P1/P2](/reference/p25-phase-2/), DMR, NXDN** and programs from your
ZIP. **No encryption** — like every scanner (and every SDR), it can't decode
[AES](/police-scanner-encryption/). For a free path, compare it to
[GopherTrunk](/police-scanner-vs-sdr/).
</div>

## Overview

The SDS150 sits at the very top of Uniden's handheld line. It keeps the feature
that defines the SDS series — a **software-defined
[True I/Q](/reference/software-defined-radio/) receiver** rather than a conventional
discriminator — which is what lets it recover voice on
[simulcast](/reference/simulcast/) systems that turn a
[BCD436HP](/reference/uniden-bcd436hp/) or [BCD325P2](/reference/uniden-bcd325p2/)
to garble. What the SDS150 adds over the [SDS100](/reference/uniden-sds100/) is a
larger color **touchscreen** interface and **built-in GPS** for hands-free,
location-aware scanning.

That polish comes at a premium: at roughly $800 it is the most expensive scanner in
Uniden's catalog. If your area is **not** simulcast, the True I/Q front end is moot
and a cheaper radio — or a free SDR — does the same job. And if simulcast is your
only concern, the older [SDS100](/reference/uniden-sds100/) runs the identical
decode engine for less.

## Modes &amp; systems it decodes

- **[P25 Phase I &amp; II](/reference/p25-phase-2/)** — the dominant US public-safety
  standard, conventional and [trunked](/reference/trunked-radio/).
- **[DMR](/reference/dmr/) and [NXDN](/reference/nxdn/)** — business and some public
  systems (some modes have historically needed a paid upgrade key).
- **Motorola, EDACS, LTR** analog trunking; conventional analog FM.
- **Weather (S.A.M.E. alerts), air, marine, rail, and CB/business** conventional
  channels.

It **cannot** decode [encrypted](/police-scanner-encryption/)
[talkgroups](/reference/talkgroup/) — a hard limit for all consumer hardware and all
SDRs, GopherTrunk included.

## Programming

Three ways in, easiest first:

1. **ZIP code / HomePatrol database.** Enter your location and the SDS150 loads
   every known nearby system — the fastest start for a beginner, driven right from
   the touchscreen.
2. **Sentinel software + microSD card** for favorites lists, firmware, and database
   updates from a PC.
3. **Manual** frequency/[talkgroup](/reference/talkgroup/) entry for anything not in
   the database.

## GopherTrunk alternative

The SDS150 is the *newest turnkey* answer. **GopherTrunk** is the *free* one. An
[RTL-SDR](/reference/rtl-sdr/) (~$30) running GopherTrunk decodes the same
[P25](/reference/project-25/), DMR and NXDN, and adds what no scanner does: it
**records, logs and timestamps every call**, follows **unlimited**
[talkgroups](/reference/talkgroup/) and systems at once, and streams to a **web
console** you can reach remotely. GopherTrunk's software
[CMA equalizer](/reference/simulcast/) also cleans up many simulcast systems.

Where the SDS150 still wins: **no PC required, pocket portability, a touchscreen and
built-in GPS, and a purpose-built front end** that edges out a cheap dongle on the
hardest simulcast. The honest, detailed head-to-head is in
[police scanner vs GopherTrunk](/police-scanner-vs-sdr/); if you want to try the
free path first, [download GopherTrunk](/downloads.html) and pair it with a $30
dongle before spending $800.

## Who it's for

- **Buy the SDS150** if you want Uniden's newest handheld, a large touchscreen and
  built-in GPS, and price is not the deciding factor.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B0FXNFPB4C?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [SDS100](/reference/uniden-sds100/)** instead for the same simulcast
  decode at a lower price, or the [SDS200](/reference/uniden-sds200/) for a base/mobile
  install.
- **Skip it** for a [BCD436HP](/reference/uniden-bcd436hp/) (non-simulcast areas) or
  a free [SDR + GopherTrunk](/police-scanner-vs-sdr/). Comparing options? See
  [best police scanners](/best-police-scanners/).

## Sources

[^uniden]: [Uniden SDS150 product page](https://uniden.com/products/sds150) — Uniden America, on True I/Q architecture, touchscreen and GPS, supported modes, and HomePatrol database programming.
