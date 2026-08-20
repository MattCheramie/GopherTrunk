---
slug: yaesu-ftm-300dr
title: Yaesu FTM-300DR
entry_type: hardware
category: ham-radios
description: "The Yaesu FTM-300DR is a discontinued but still-available 50 W 2m/70cm C4FM System Fusion mobile with true dual receive, built-in GPS and APRS, and Bluetooth — the best value in the Fusion mobile lineup while new-old-stock lasts."
keywords: Yaesu FTM-300DR, FTM-300DR review, C4FM mobile radio, System Fusion mobile radio, dual receive mobile ham radio, APRS mobile radio, FTM-300DR discontinued, FTM-300DR vs FTM-500DR, best mobile ham radio 2026
aka: [FTM-300DR, FTM-300]
autolink: true
affiliate: true
product:
  name: "Yaesu FTM-300DR"
  brand: Yaesu
  category: Mobile ham transceiver
  lowPrice: "300"
  highPrice: "430"
  url: https://www.amazon.com/dp/B08884HN79?tag=gophertrunk-20
infobox:
  - { label: Type, value: Dual-band mobile transceiver (discontinued — NOS remains) }
  - { label: Bands, value: "2m / 70cm (true dual receive)" }
  - { label: Modes, value: "FM, C4FM System Fusion (AMS)" }
  - { label: Power, value: "50 / 25 / 5 W" }
  - { label: Programming, value: "microSD / ADMS-11 / RT Systems (no CHIRP)" }
  - { label: Price, value: "around $400 NOS, $300–380 used" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B08884HN79?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [yaesu-ftm-500dr, icom-id-5100a, anytone-at-d578uviii, kenwood-tm-v71a, rtl-sdr, aprs]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Best SDR for GopherTrunk", url: /best-sdr-for-gophertrunk/ }
cite_urls:
  - https://www.yaesu.com/product-detail.aspx?Model=FTM-300DR&CatName=Legacy
faq:
  - q: "Is the Yaesu FTM-300DR discontinued?"
    a: "Yes. Yaesu's own site now files the FTM-300DR under its Legacy category, with the end of production landing somewhere around 2024–2025 (Yaesu never announced a formal date). New-old-stock is still on dealer shelves and Amazon, and it remains a fully supported, current-feeling radio."
  - q: "Is the FTM-300DR still worth buying in 2026?"
    a: "Yes, at new-old-stock prices around $400 it is arguably the best value in the entire System Fusion lineup: true dual-band receive with independent C4FM decode, GPS and APRS standard, and rock-solid RF. Just don't pay more than FTM-500DR money for it."
  - q: "What's the difference between the FTM-300DR and FTM-500DR?"
    a: "The 500DR adds the AESS dual-speaker audio system, a bigger 2.4-inch display with the funnel layout, and a head-mounted speaker. The 300DR counters with true independent dual receive (both bands can decode C4FM) at roughly $100 less. Audio and screen aside, they are close siblings."
  - q: "Does the FTM-300DR work with CHIRP?"
    a: "No — CHIRP does not support it. Program it via the microSD card, Yaesu's free ADMS-11 software, or RT Systems."
  - q: "Do I need a ham license for the FTM-300DR?"
    a: "To transmit, yes — an FCC amateur license (Technician class minimum, Part 97). Listening is license-free, and a $30 RTL-SDR with free GopherTrunk will let you hear the same repeaters before you buy anything."
---
**The Yaesu FTM-300DR** is a 50-watt 2m/70cm mobile with analog FM and **C4FM
[System Fusion](/reference/system-fusion-ysf/)**, **true dual receive** with
independent decode on both bands, built-in GPS with full
[APRS](/reference/aprs/), and Bluetooth.[^yaesu] Yaesu has quietly
**discontinued** it — the model now lives on Yaesu's "Legacy" page — but
new-old-stock is still on dealer and Amazon shelves around $400, and at that
price the community regards it as **the best value in the Fusion mobile lineup**.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B08884HN79?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The value Fusion mobile.** 50 W on 2m/70cm, FM + [C4FM](/reference/c4fm/) with
auto mode select, and — unlike the pricier
[FTM-500DR](/reference/yaesu-ftm-500dr/) — **true dual receive that decodes
digital on both bands**. GPS + [APRS](/reference/aprs/) 1200/9600 standard,
Bluetooth, detachable head, microSD cloning. **Discontinued (Yaesu "Legacy"),
but NOS remains** — around $400 new, $300–380 used. **No CHIRP.** Small 1.8"
non-touch display is the main compromise. Transmitting needs a ham license;
listening doesn't.
</div>

## Overview

The FTM-300DR was the middle child of Yaesu's Fusion mobile line and outlived
that role: it packs nearly all of the flagship's capability into a smaller,
cheaper box. You get 50/25/5 W on both bands, a detachable front panel
(separation kit approach), roughly 999 memories per band, a band scope, microSD
backup and cloning, WiRES-X portable node capability, and ~13 A maximum draw.
There is no cross-band repeat.

Its party trick is **true dual receive** — V+V, U+U, or V+U — with *independent*
decode, so it can sit on two C4FM channels at once, something the FTM-500DR
cannot do. RF is, in the community's words, rock solid. The compromises are
physical: a small 1.8-inch color TFT that is not a touchscreen, a speaker in the
body rather than the head (remote-head installs want an external speaker), Yaesu
menu conventions that take learning, and the usual Yaesu Bluetooth pairing
quirks.

**Production status, plainly:** Yaesu's own site files the FTM-300DR under
**Legacy**, with production ending somewhere around 2024–2025 — no formal date
was ever announced, so treat the year as approximate. New-old-stock is still
listed at the major ham dealers and Amazon. NOS at ~$400 is a fair deal; used
units run about $300–380 (a range estimated from listing spread, so check
recent sold prices).

## Modes &amp; features

- **Analog FM and [C4FM System Fusion](/reference/system-fusion-ysf/)** with AMS
  auto mode select — and independent digital decode on both receivers.
- **Built-in GPS + [APRS](/reference/aprs/)** at 1200/9600 bps, standard.
- **Bluetooth**, band scope, microSD clone/backup, detachable front panel,
  WiRES-X portable digital node.
- The same honest network caveat as every Fusion radio: C4FM trails
  [DMR](/reference/dmr/) and [D-STAR](/reference/d-star/) in many regions —
  confirm your local repeaters actually run Fusion first.

## Programming

No CHIRP support. Use the **microSD card**, Yaesu's free **ADMS-11** PC
software, or the paid **RT Systems** package.

## GopherTrunk alternative

GopherTrunk **receives only — it cannot transmit** — so it doesn't replace an
FTM-300DR. What it does is tell you whether to buy one. A ~$30
[RTL-SDR](/reference/rtl-sdr/) running free GopherTrunk monitors your local
repeaters and digital traffic, so you can see whether Fusion, DMR, or D-STAR is
what's actually on the air in your area before spending $400 — and it records
and logs every transmission, which no transceiver does. No license is needed to
listen; transmitting requires an FCC amateur license (Part 97, Technician
minimum). Hardware picks are in
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).

## Who it's for

- **Buy the FTM-300DR** if you want Fusion, GPS/APRS, and genuine dual-band
  digital receive at the best price in the lineup — while NOS lasts.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B08884HN79?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [FTM-500DR](/reference/yaesu-ftm-500dr/)** for the bigger display and
  the AESS speaker system if the ~$100 premium doesn't sting.
- **Skip both** for the [AnyTone AT-D578UVIII Plus](/reference/anytone-at-d578uviii/)
  in DMR country or the [Icom ID-5100A](/reference/icom-id-5100a/) for D-STAR.
- Full rankings: [the 10 best mobile ham radios](/best-mobile-ham-radios/).

> **Amazon note.** The FTM-300DR is discontinued, so the Amazon listing is
> remaining new-old-stock and third-party sellers — check the price against the
> big ham dealers before clicking buy.

## Sources

[^yaesu]: [Yaesu FTM-300DR product page (Legacy category)](https://www.yaesu.com/product-detail.aspx?Model=FTM-300DR&CatName=Legacy) — Yaesu, confirming the model's move to Legacy status and documenting dual-receive C4FM, GPS/APRS, Bluetooth, and power specifications.
