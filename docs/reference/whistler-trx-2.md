---
slug: whistler-trx-2
title: Whistler TRX-2
entry_type: hardware
category: consumer-scanners
description: "The Whistler TRX-2 is a digital base/mobile police scanner that includes P25 Phase I/II, DMR, and NXDN at no extra fee — the base version of the TRX-1 — with EZ-Scan object-oriented programming and."
keywords: Whistler TRX-2, TRX-2 scanner, digital base scanner, mobile scanner, P25 DMR NXDN scanner, DMR included scanner, EZ-Scan, Whistler TRX-2 review, base mobile scanner
aka: [TRX-2, TRX2]
autolink: true
affiliate: true
product:
  name: "Whistler TRX-2"
  brand: Whistler
  category: Police scanner (digital base/mobile)
  lowPrice: "552"
  highPrice: "648"
  url: https://www.amazon.com/dp/B01H3XYJS0?tag=gophertrunk-20
infobox:
  - { label: Type, value: Digital base/mobile scanner }
  - { label: Modes, value: "P25 P1/P2, DMR, NXDN (all included)" }
  - { label: Simulcast, value: "Fair — not True I/Q" }
  - { label: Programming, value: "EZ-Scan software, USA/Canada DB" }
  - { label: Extras, value: "Object-oriented, microSD, GPS-ready" }
  - { label: Price, value: around $600 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B01H3XYJS0?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [whistler-trx-1, whistler-ws1065, uniden-sds200, police-scanner, trunking-scanner, p25-phase-2, dmr, nxdn]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://www.whistlergroup.com/products/trx-2
faq:
  - q: "Does the Whistler TRX-2 include DMR and NXDN?"
    a: "Yes. P25 Phase I/II, DMR, and NXDN are all built in at no extra fee — the same mode set as the handheld TRX-1. On many Uniden scanners DMR and NXDN are paid upgrades, so the TRX-2 can be the better value for those modes."
  - q: "What is the difference between the TRX-2 and TRX-1?"
    a: "The decode engine, modes, and EZ-Scan software are identical. The TRX-2 is the base/mobile unit — mains or 12V power, made to sit on a desk or mount in a vehicle — while the TRX-1 is the battery-powered handheld."
  - q: "How does the TRX-2 handle simulcast?"
    a: "Fair, not great — it is a conventional digital receiver, not True I/Q. On harsh P25 Phase II simulcast the Uniden SDS200 decodes better, and GopherTrunk's software equalizer also handles many simulcast systems the TRX-2 struggles with."
  - q: "TRX-2 or Uniden SDS200?"
    a: "Buy the TRX-2 for included DMR/NXDN and lower cost when your systems aren't harsh simulcast. Buy the SDS200 if you're in a simulcast metro and want the best possible decode from its True I/Q front end."
  - q: "Can the TRX-2 record audio?"
    a: "Yes, it can record to the microSD card. For comprehensive, timestamped logging of every call across many systems, though, a PC-based GopherTrunk setup is far more capable."
  - q: "Does the TRX-2 decode encryption?"
    a: "No. No scanner and no SDR — including GopherTrunk — can decode AES-encrypted traffic. The TRX-2 identifies the talkgroup but plays silence on encrypted calls."
---
**The Whistler TRX-2** is the **base/mobile** version of the
[TRX-1](/reference/whistler-trx-1/): a digital scanner that decodes
[P25 Phase I and II](/reference/p25-phase-2/), [DMR](/reference/dmr/), and
[NXDN](/reference/nxdn/) — **all included at no extra fee**, an advantage over Uniden
where DMR/NXDN are often paid keys.[^whistler] It shares the TRX-1's
object-oriented **EZ-Scan** software and USA/Canada database, and the same **fair**
[simulcast](/reference/simulcast/) performance — a conventional front end, not
[True I/Q](/reference/software-defined-radio/).

<a class="btn btn--buy" href="https://www.amazon.com/dp/B01H3XYJS0?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Base/mobile TRX.** Same modes as the [TRX-1](/reference/whistler-trx-1/) in a
desk/vehicle unit — mains or 12 V. **DMR + NXDN included** with
[P25](/reference/project-25/), no paid keys. **Fair simulcast** — not
[True I/Q](/reference/software-defined-radio/); the
[SDS200](/reference/uniden-sds200/) wins on harsh
[simulcast](/reference/simulcast/). **~$600.** No
[encryption](/police-scanner-encryption/) decode. Free path:
[SDR + GopherTrunk](/police-scanner-vs-sdr/).
</div>

## Overview

The TRX-2 is Whistler's flagship **base/mobile** scanner and the rival to Uniden's
[SDS200](/reference/uniden-sds200/) and [BCD536HP](/reference/police-scanner/). It is
electrically the same radio as the handheld [TRX-1](/reference/whistler-trx-1/) —
identical modes, identical [EZ-Scan](/reference/police-scanner/) object-oriented
programming — but packaged for a desk or dashboard with mains/12 V power, a larger
display, and GPS support for location-based scanning.

Its strength is **mode inclusion**: [DMR](/reference/dmr/) and
[NXDN](/reference/nxdn/) are built in. Its weakness is the **front end** — conventional,
not [True I/Q](/reference/software-defined-radio/) — so on distorted
[Phase II simulcast](/reference/simulcast/) it is only **fair**.

## Modes &amp; systems it decodes

- **[P25 Phase I &amp; II](/reference/p25-phase-2/)** — conventional and
  [trunked](/reference/trunked-radio/).
- **[DMR](/reference/dmr/) and [NXDN](/reference/nxdn/)** — **included at no extra
  cost.**
- **Motorola, EDACS, LTR** analog trunking and conventional analog FM.
- **Cannot decode [encryption](/police-scanner-encryption/)** — AES talkgroups show
  but stay silent, as on every receiver.

> **Simulcast caveat.** In a heavy simulcast metro the TRX-2 can garble where an
> [SDS200](/reference/uniden-sds200/) locks. Confirm whether your P25 system is
> simulcast.

## Programming

As with the TRX-1, everything is built in **EZ-Scan** on a PC against the USA/Canada
database, then moved to the radio on a microSD card. The object-oriented model makes
large multi-system setups manageable, and an optional GPS receiver enables
location-based system selection for mobile use.

## GopherTrunk alternative

The TRX-2 is a capable base scanner with a strong mode set for the money. But it is
one receiver with a fair front end, and its logging is basic.

**GopherTrunk** is the free, open-source alternative — and since the TRX-2 already
lives on a desk near a computer, the trade-off is smaller than for a handheld. A
~$30 [RTL-SDR](/reference/rtl-sdr/) (or a better front end for simulcast) decodes the
same [P25](/reference/project-25/), [DMR](/reference/dmr/), and
[NXDN](/reference/nxdn/), follows **unlimited** systems at once, and **records,
logs, and timestamps every call** to a web console you can reach remotely. Its
software [CMA equalizer](/reference/simulcast/) also handles many simulcast systems
the TRX-2 struggles with. The TRX-2 keeps a **turnkey, no-PC-required** experience
and FCC certification. See [police scanner vs GopherTrunk](/police-scanner-vs-sdr/),
or [download GopherTrunk](/downloads.html) to try it.

## Who it's for

- **Buy the TRX-2** for a base/mobile with [P25](/reference/project-25/) plus
  **included DMR/NXDN**, where your systems aren't harsh simulcast.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B01H3XYJS0?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [SDS200](/reference/uniden-sds200/)** instead in a
  [simulcast](/reference/simulcast/) metro where decode quality is critical.
- **Buy the [TRX-1](/reference/whistler-trx-1/)** for the handheld version.
- **Skip it** for a free [SDR + GopherTrunk](/police-scanner-vs-sdr/) if you want
  unlimited channels, recording, and logging on the PC that's already on your desk.

## Bottom line

The Whistler TRX-2 is the base/mobile value pick for monitoring
[DMR](/reference/dmr/)/[NXDN](/reference/nxdn/) alongside
[P25](/reference/project-25/) without paid upgrade keys. Choose it when simulcast
isn't a problem; step up to the [SDS200](/reference/uniden-sds200/) or lean on
[GopherTrunk](/police-scanner-vs-sdr/) when it is — and remember no receiver decodes
[encryption](/police-scanner-encryption/).

## Sources

[^whistler]: [Whistler TRX-2 product page](https://www.whistlergroup.com/products/trx-2) — Whistler Group, on included P25 Phase I/II, DMR, and NXDN modes, object-oriented programming, EZ-Scan software, and base/mobile form factor.
