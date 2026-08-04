---
slug: whistler-ws1065
title: Whistler WS1065
entry_type: hardware
category: consumer-scanners
description: "The Whistler WS1065 is a discontinued legacy desktop digital scanner with P25 decoding, object-oriented programming, and a DMR upgrade path — a used-market option only. New buyers should choose a."
keywords: Whistler WS1065, WS1065 scanner, legacy digital scanner, discontinued scanner, P25 desktop scanner, object oriented scanner, used scanner, DMR upgrade scanner, Whistler WS1065 review
aka: [WS1065]
autolink: true
affiliate: true
product:
  name: "Whistler WS1065"
  brand: Whistler
  category: Police scanner (legacy desktop digital)
  lowPrice: "150"
  highPrice: "300"
  url: https://www.amazon.com/s?k=Whistler+WS1065&tag=gophertrunk-20
infobox:
  - { label: Type, value: "Legacy desktop digital scanner (discontinued)" }
  - { label: Modes, value: "P25 (DMR-upgradable) — no NXDN" }
  - { label: Programming, value: "Object-oriented, PC software" }
  - { label: Availability, value: "Used / new-old-stock only" }
  - { label: Price, value: "around $250 used" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/s?k=Whistler+WS1065&tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">Search on Amazon &rarr;</a>" }
see_also: [whistler-trx-2, whistler-trx-1, uniden-sds200, police-scanner, trunking-scanner, p25-phase-1]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://wiki.radioreference.com/index.php/Whistler_WS1065
faq:
  - q: "Is the Whistler WS1065 still made?"
    a: "No. The WS1065 is discontinued. New stock is scarce and you will mostly find it used or as new-old-stock. For a new purchase, choose a current model like the Whistler TRX-2 or Uniden SDS200 instead."
  - q: "Does the WS1065 decode DMR and NXDN?"
    a: "It decodes P25 out of the box and is DMR-upgradable via a paid key, but it does not do NXDN. If you need DMR and NXDN together, the newer TRX-1/TRX-2 include both at no extra cost."
  - q: "Should I buy a used WS1065 today?"
    a: "Only if you find one cheap and understand its limits — no NXDN, older front end, weak on simulcast, and no manufacturer support. Most buyers are better served by a current TRX-2, SDS200, or a free SDR + GopherTrunk."
  - q: "What is object-oriented programming on the WS1065?"
    a: "Like the later TRX scanners, the WS1065 organizes systems, sites, and talkgroups as reusable objects rather than fixed banks, programmed from a PC. It was an early example of the approach Whistler carried into the TRX line."
  - q: "How does the WS1065 handle simulcast?"
    a: "Poorly by modern standards. It predates True I/Q processing, so on distorted P25 Phase II simulcast it struggles. A Uniden SDS200 or GopherTrunk's software equalizer is far better on those systems."
---
**The Whistler WS1065** is a **discontinued** legacy desktop digital scanner with
[P25](/reference/p25-phase-1/) decoding, object-oriented programming, and an optional
DMR upgrade — but **no NXDN**.[^rr] It is a **used-market / new-old-stock** option
only; anyone buying new today should choose a current model such as the
[Whistler TRX-2](/reference/whistler-trx-2/) or
[Uniden SDS200](/reference/uniden-sds200/).

<a class="btn btn--buy" href="https://www.amazon.com/s?k=Whistler+WS1065&tag=gophertrunk-20" rel="nofollow sponsored noopener">Search on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Discontinued — used market only.** New stock is scarce.
**[P25](/reference/p25-phase-1/) desktop, DMR-upgradable, no
[NXDN](/reference/nxdn/).** Object-oriented programming, an early ancestor of the
[TRX](/reference/whistler-trx-2/) line. **Weak on
[simulcast](/reference/simulcast/)** — predates
[True I/Q](/reference/software-defined-radio/). **~$250 used.** Buy a current
[TRX-2](/reference/whistler-trx-2/) / [SDS200](/reference/uniden-sds200/) or a free
[SDR + GopherTrunk](/police-scanner-vs-sdr/) instead unless you find one cheap.
</div>

## Overview

The WS1065 was Whistler's desktop digital scanner from the era before the
[TRX](/reference/whistler-trx-2/) series. It introduced Whistler's
**object-oriented** data model — systems, sites, and talkgroups as reusable objects
programmed from a PC — that the TRX line later refined. It decodes
[P25](/reference/p25-phase-1/) and can be upgraded to
[DMR](/reference/dmr/) with a paid key, but it never gained
[NXDN](/reference/nxdn/), and its front end predates modern
[True I/Q](/reference/software-defined-radio/) simulcast processing.

Today it is **discontinued**. Treat it as a legacy or used-market curiosity, not a
first choice.

## Modes &amp; systems it decodes

- **[P25 Phase I](/reference/p25-phase-1/)** conventional and
  [trunked](/reference/trunked-radio/); Phase II support is limited by its age.
- **[DMR](/reference/dmr/)** via an optional paid upgrade key.
- **Motorola, EDACS, LTR** analog trunking and conventional analog FM.
- **No [NXDN](/reference/nxdn/).** If you need NXDN, this radio can't do it.
- **Weak on [simulcast](/reference/simulcast/)** and, like all receivers, **no
  [encryption](/police-scanner-encryption/) decode.**

> **Buy current instead.** For a new purchase, the
> [TRX-2](/reference/whistler-trx-2/) adds NXDN and better software, and the
> [SDS200](/reference/uniden-sds200/) adds True I/Q simulcast — both are supported
> and available. The WS1065 makes sense only as a cheap used find.

## Programming

The WS1065 programs from a PC using Whistler's object-oriented software, similar in
spirit to the later [EZ-Scan](/reference/police-scanner/) workflow. Because the model
is discontinued, database and firmware support are limited; expect to lean on
community resources like [RadioReference](/reference/radioreference/).

## GopherTrunk alternative

Given the WS1065's age, limits, and lack of support, the free path is especially
compelling here. A ~$30 [RTL-SDR](/reference/rtl-sdr/) running **GopherTrunk**
decodes [P25](/reference/project-25/), [DMR](/reference/dmr/), **and**
[NXDN](/reference/nxdn/) — everything the WS1065 does plus the NXDN it never had —
follows unlimited [trunked](/reference/trunked-radio/) systems, and **records, logs,
and timestamps every call**. Its software [CMA equalizer](/reference/simulcast/) also
handles simulcast the WS1065 can't. Unless you already own a WS1065 or find one very
cheap, [GopherTrunk](/downloads.html) plus a dongle is a better use of the money. The
comparison is in [police scanner vs GopherTrunk](/police-scanner-vs-sdr/).

No receiver, the WS1065 or GopherTrunk included, decodes
[AES encryption](/police-scanner-encryption/).

## Who it's for

- **Consider a used WS1065** only if you find one cheap, don't need
  [NXDN](/reference/nxdn/), and aren't on a simulcast system.
  <a class="btn btn--buy" href="https://www.amazon.com/s?k=Whistler+WS1065&tag=gophertrunk-20" rel="nofollow sponsored noopener">Search on Amazon &rarr;</a>
- **Buy the [TRX-2](/reference/whistler-trx-2/)** for a current desktop with NXDN
  included.
- **Buy the [SDS200](/reference/uniden-sds200/)** for the best simulcast decode.
- **Skip it** for a free [SDR + GopherTrunk](/police-scanner-vs-sdr/) — you gain
  NXDN, recording, logging, and simulcast handling the WS1065 lacks.

## Bottom line

The Whistler WS1065 is a **legacy** scanner worth considering only as an inexpensive
used find. It decodes [P25](/reference/p25-phase-1/), upgrades to
[DMR](/reference/dmr/), but has no NXDN, struggles on
[simulcast](/reference/simulcast/), and is out of production. New buyers should pick
a current [TRX-2](/reference/whistler-trx-2/) or [SDS200](/reference/uniden-sds200/),
or go free with [GopherTrunk](/police-scanner-vs-sdr/).

## Sources

[^rr]: [Whistler WS1065 — RadioReference Wiki](https://wiki.radioreference.com/index.php/Whistler_WS1065) — community reference on the WS1065's P25 decoding, DMR upgrade path, object-oriented programming, and discontinued status.
