---
slug: whistler-trx-1
title: Whistler TRX-1
entry_type: hardware
category: consumer-scanners
description: "The Whistler TRX-1 is a digital handheld police scanner that includes P25 Phase I/II, DMR, and NXDN at no extra fee — a real advantage over Uniden, where DMR/NXDN are paid upgrades — with EZ-Scan."
keywords: Whistler TRX-1, TRX-1 scanner, digital handheld scanner, P25 DMR NXDN scanner, DMR included scanner, EZ-Scan, object oriented scanner, Whistler TRX-1 review, simulcast scanner
aka: [TRX-1, TRX1]
autolink: true
affiliate: true
product:
  name: "Whistler TRX-1"
  brand: Whistler
  category: Police scanner (digital handheld)
  lowPrice: "506"
  highPrice: "594"
  url: https://www.amazon.com/dp/B01H3XYNUO?tag=gophertrunk-20
infobox:
  - { label: Type, value: Digital handheld scanner }
  - { label: Modes, value: "P25 P1/P2, DMR, NXDN (all included)" }
  - { label: Simulcast, value: "Fair — not True I/Q" }
  - { label: Programming, value: "EZ-Scan software, USA/Canada DB" }
  - { label: Extras, value: "Object-oriented, microSD, upgradable" }
  - { label: Price, value: around $550 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B01H3XYNUO?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [whistler-trx-2, whistler-ws1065, uniden-sds200, police-scanner, trunking-scanner, p25-phase-2, dmr, nxdn]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://www.whistlergroup.com/products/trx-1
faq:
  - q: "Does the Whistler TRX-1 include DMR and NXDN?"
    a: "Yes — and that is its main selling point. P25 Phase I/II, DMR, and NXDN are all built in at no extra fee. On many Uniden scanners, DMR and NXDN are paid upgrade keys, so the TRX-1 can be the better value if you monitor those modes."
  - q: "How is the TRX-1 on simulcast systems?"
    a: "Fair, not great. The TRX-1 is a conventional digital front end, not True I/Q. On tough P25 Phase II simulcast systems it will struggle where a Uniden SDS100/SDS200 — or GopherTrunk's software equalizer — pulls voice through the distortion."
  - q: "What is object-oriented programming on the TRX-1?"
    a: "Whistler's TRX scanners organize everything as reusable objects (systems, sites, talkgroups) rather than fixed banks, which makes complex setups flexible. You manage it all in the free EZ-Scan PC software with a USA/Canada database."
  - q: "TRX-1 or Uniden SDS100?"
    a: "Buy the TRX-1 if DMR/NXDN inclusion and price matter and your systems aren't harsh simulcast. Buy the SDS100 if you're in a simulcast metro and need the best decode — its True I/Q front end is a real step up on distorted signals."
  - q: "Can the TRX-1 decode encrypted police?"
    a: "No. Like every scanner and every SDR (including GopherTrunk), the TRX-1 cannot decode AES-encrypted traffic. It will identify the talkgroup but play silence."
  - q: "Is there a base/mobile version?"
    a: "Yes, the Whistler TRX-2 is the base/mobile sibling with the same modes and software. The TRX-1 is the handheld, battery-powered version."
---
**The Whistler TRX-1** is a digital **handheld** police scanner that decodes
[P25 Phase I and II](/reference/p25-phase-2/), [DMR](/reference/dmr/), and
[NXDN](/reference/nxdn/) — **all included at no extra fee**, a genuine advantage over
Uniden, where DMR and NXDN are often paid upgrade keys.[^whistler] It uses Whistler's
object-oriented **EZ-Scan** software and a USA/Canada database, and its simulcast
performance is **fair** — a conventional front end, not the
[True I/Q](/reference/software-defined-radio/) of Uniden's SDS series.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B01H3XYNUO?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**DMR + NXDN included.** No paid upgrade keys — [P25](/reference/project-25/),
[DMR](/reference/dmr/), and [NXDN](/reference/nxdn/) all work out of the box, a real
edge over Uniden if you monitor those modes. **Handheld,** object-oriented
programming via **EZ-Scan**. **Fair simulcast** — not
[True I/Q](/reference/software-defined-radio/); the
[SDS100](/reference/uniden-sds100/) is better on harsh
[simulcast](/reference/simulcast/). **~$550.** No
[encryption](/police-scanner-encryption/) decode — nothing consumer can. Free path:
[SDR + GopherTrunk](/police-scanner-vs-sdr/).
</div>

## Overview

The TRX-1 is Whistler's flagship handheld and the natural rival to Uniden's
[SDS100](/reference/uniden-sds100/) and [BCD436HP](/reference/police-scanner/). Its
headline advantage is **mode inclusion**: [DMR](/reference/dmr/) and
[NXDN](/reference/nxdn/) are built in, where several Uniden models charge for them.
It uses an **object-oriented** data model — systems, sites, and talkgroups are
reusable objects rather than rigid banks — managed in the free **EZ-Scan** PC
software against a USA/Canada database, with a microSD slot for storage and audio
recording.

Where it trails Uniden is the **front end**: the TRX-1 is a conventional digital
receiver, not [True I/Q](/reference/software-defined-radio/), so on distorted
[P25 Phase II simulcast](/reference/simulcast/) it is merely **fair**.

## Modes &amp; systems it decodes

- **[P25 Phase I &amp; II](/reference/p25-phase-2/)** — conventional and
  [trunked](/reference/trunked-radio/), the dominant US public-safety standard.
- **[DMR](/reference/dmr/) and [NXDN](/reference/nxdn/)** — **included at no extra
  cost** (the key selling point).
- **Motorola, EDACS, LTR** analog trunking, plus conventional analog FM.
- **Cannot decode [encryption](/police-scanner-encryption/).** AES-keyed talkgroups
  are identified but silent — a universal limit.

> **Simulcast caveat.** In a tough simulcast metro the TRX-1 can garble where an
> [SDS100](/reference/uniden-sds100/) locks. Check whether your P25 system is
> simulcast before choosing.

## Programming

Everything runs through **EZ-Scan** on a PC: pick your location or systems from the
USA/Canada database, and the software builds the object set for you. Advanced users
can hand-craft systems and talkgroups. Firmware and database updates and audio
recordings move over the microSD card and USB.

## GopherTrunk alternative

The TRX-1 is a strong value for DMR/NXDN monitoring in a handheld. But it is still a
single portable radio with a fair front end and no true logging.

**GopherTrunk** is the free, open-source alternative. A ~$30
[RTL-SDR](/reference/rtl-sdr/) — or a better front end for simulcast — decodes the
same [P25](/reference/project-25/), [DMR](/reference/dmr/), and
[NXDN](/reference/nxdn/), follows **unlimited** [trunked](/reference/trunked-radio/)
systems at once, and **records, logs, and timestamps every call** to a web console.
Its software [CMA equalizer](/reference/simulcast/) also cleans up many simulcast
systems the TRX-1 struggles with. What the TRX-1 keeps is **battery portability and
a no-PC, turnkey** experience. See the honest
[police scanner vs GopherTrunk](/police-scanner-vs-sdr/), or
[download GopherTrunk](/downloads.html) and try a dongle first.

## Who it's for

- **Buy the TRX-1** if you want [P25](/reference/project-25/) plus **included
  DMR/NXDN** in a handheld and your systems aren't harsh simulcast.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B01H3XYNUO?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [SDS100](/reference/uniden-sds100/)** instead in a
  [simulcast](/reference/simulcast/) metro where decode quality is critical.
- **Buy the [TRX-2](/reference/whistler-trx-2/)** for the base/mobile version.
- **Skip it** for a free [SDR + GopherTrunk](/police-scanner-vs-sdr/) if you want
  recording/logging and have a PC.

## Bottom line

The Whistler TRX-1 is the value pick for monitoring
[DMR](/reference/dmr/)/[NXDN](/reference/nxdn/) alongside
[P25](/reference/project-25/) in a handheld, because those modes are included rather
than paid extras. Accept the fair (not True I/Q) simulcast performance — if your
metro is heavy simulcast, step up to an [SDS100](/reference/uniden-sds100/) or use
[GopherTrunk](/police-scanner-vs-sdr/) — and remember that
[encryption](/police-scanner-encryption/) is off-limits to every receiver.

## Sources

[^whistler]: [Whistler TRX-1 product page](https://www.whistlergroup.com/products/trx-1) — Whistler Group, on included P25 Phase I/II, DMR, and NXDN modes, object-oriented programming, and EZ-Scan software with USA/Canada database.
