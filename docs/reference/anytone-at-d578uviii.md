---
slug: anytone-at-d578uviii
title: AnyTone AT-D578UVIII Plus
entry_type: hardware
category: ham-radios
description: "The AnyTone AT-D578UVIII Plus is the default DMR mobile ham radio — tri-band 2m/1.25m/70cm, 50 W, analog FM + DMR Tier I/II, GPS with true FM and DMR APRS, Bluetooth, and a 500,000-contact database, around $460 in 2026."
keywords: AnyTone AT-D578UVIII Plus, AT-D578UVIII review, best DMR mobile radio, DMR mobile ham radio, tri-band mobile radio, 220 MHz mobile, DMR APRS radio, AnyTone mobile, hotspot DMR radio, best mobile ham radio 2026
aka: [AT-D578UVIII, D578UVIII, D578]
autolink: true
affiliate: true
product:
  name: "AnyTone AT-D578UVIII Plus"
  brand: AnyTone
  category: Mobile ham transceiver
  lowPrice: "430"
  highPrice: "500"
  url: https://www.amazon.com/dp/B09CG83Q3Z?tag=gophertrunk-20
infobox:
  - { label: Type, value: Tri-band mobile transceiver }
  - { label: Bands, value: "2m / 1.25m / 70cm" }
  - { label: Modes, value: "FM, DMR Tier I/II" }
  - { label: Power, value: "50 W VHF / ~25 W 220 / 45 W UHF" }
  - { label: Programming, value: "AnyTone CPS (free, Windows; no CHIRP)" }
  - { label: Price, value: around $460 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B09CG83Q3Z?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [yaesu-ftm-500dr, icom-id-5100a, btech-uv-25x2, yaesu-ftm-300dr, rtl-sdr, dmr]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Best SDR for GopherTrunk", url: /best-sdr-for-gophertrunk/ }
cite_urls:
  - https://www.anytone.net/video/products-detail-935418
  - https://www.bridgecomsystems.com/products/anytone-at-d578uviii-plus-tri-band-amateur-dmr-mobile-radio
faq:
  - q: "Is the AnyTone AT-D578UVIII Plus the best DMR mobile?"
    a: "For hams, it is the default: a huge digital-ID contact database with caller info on screen, APRS on both analog FM and DMR, tri-band transmit including 220 MHz, Bluetooth, and constant firmware development. Its receiver front end trails Icom and Yaesu on crowded RF sites — that is the honest trade."
  - q: "Does the AT-D578UVIII Plus do APRS?"
    a: "Yes, both kinds: true FM APRS transmit and receive via its built-in GPS, plus DMR APRS. Among digital mobiles this dual-APRS support is a genuine differentiator."
  - q: "Which AT-D578 variant should I buy?"
    a: "The family is confusing — there are III Plus, III Pro, and dual-band Plus variants. The III Plus is the tri-band amateur DMR flagship covered here; check the listing carefully so you get the tri-band amateur model."
  - q: "Can I program the AT-D578UVIII with CHIRP?"
    a: "No — DMR codeplug radios are outside CHIRP's scope. You use AnyTone's free D578UV CPS, which is Windows-only and quirky but capable. Budget an evening to learn codeplug basics."
  - q: "Do I need a license for the AT-D578UVIII Plus?"
    a: "Transmitting requires an FCC amateur license (Part 97, Technician minimum) — and on DMR you'll also register for a DMR ID. Listening requires no license; a $30 RTL-SDR with free GopherTrunk decodes and records local DMR traffic so you can scout the network first."
---
**The AnyTone AT-D578UVIII Plus** is the default [DMR](/reference/dmr/) mobile
for hams: **tri-band** transmit on 2m, 1.25m, and 70cm, analog FM plus **DMR
Tier I/II**, a built-in GPS doing **both true FM [APRS](/reference/aprs/) and
DMR APRS**, Bluetooth audio and PTT, air-band receive, and a contact database
around **500,000 digital IDs** that puts a name on every caller.[^anytone] At
roughly $430–500 it delivers more features per dollar than anything else on the
mobile list — which is exactly why it's our **best-features-for-the-price**
pick.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B09CG83Q3Z?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Best features for the price** in our
[mobile top 10](/best-mobile-ham-radios/). FM + **DMR Tier I/II**
(MOTOTRBO-compatible), **tri-band** with 50 W VHF / 45 W UHF (about 25 W on
220 MHz per spec sheets — verify against the current manual), ~4,000 channels,
10,000 talkgroups, ~500,000-contact database, **FM + DMR APRS**, Bluetooth,
air-band RX, roaming, hotspot-friendly. Trade-offs: Windows-only quirky CPS,
occasional firmware regressions, loud fan, and a front end below Icom/Yaesu on
crowded sites. **~$460.** Ham license required to transmit; none to listen.
</div>

## Overview

AnyTone's D578 is what happens when a Chinese OEM iterates fast on what hams
actually ask for. The **III Plus** — mind the variant soup of III Pro and
dual-band models — is the tri-band amateur DMR flagship: color TFT, detachable
faceplate, digital voice recording, roaming between repeaters, and friendly
behavior with Pi-Star-style hotspots. TX draw is a modest ~11–12 A.

The community's verdict is consistent. In its favor: the huge digital-ID
database showing callsign and name for every DMR caller, a genuinely loud
speaker, APRS on both analog and DMR, and constant firmware development. Against
it: the CPS is Windows-only and quirky, firmware updates occasionally introduce
regressions, the fan is loud, menus run deep, and the receiver front end —
decent as it is — sits below Icom and Yaesu when a crowded commercial site is
hammering the input. One more honest note: analog **cross-band repeat is not a
supported feature** (a single-band repeater mode has appeared in some firmware,
but it is firmware-dependent — don't buy on it).

## Modes &amp; features

- **Analog FM and [DMR](/reference/dmr/) Tier I/II**, MOTOTRBO-compatible —
  the dominant amateur digital mode in many regions, and the one with the
  cheapest ecosystem.
- **Tri-band transmit:** 50 W on 2m, ~25 W on 220 MHz (spec-sheet figure —
  check the current manual), 45 W on 70cm; **air-band receive** on top.
- **GPS with true FM [APRS](/reference/aprs/) TX/RX plus DMR APRS** — rare
  even among flagships.
- ~4,000 channels, 10,000 talkgroups, ~500,000 digital contacts, Bluetooth
  audio + PTT, voice recording, roaming, hotspot-friendly.

## Programming

No CHIRP — DMR codeplug radios are out of CHIRP's scope. The free **AnyTone
D578UV CPS** (Windows-only) is the tool; it is quirky but complete, and the
DMR community publishes shareable codeplugs for most metro areas. Budget an
evening for codeplug concepts (talkgroups, zones, receive groups) if DMR is
new to you.

## GopherTrunk alternative

GopherTrunk **receives; it cannot transmit** — it will never key up a DMR
talkgroup for you. What it does brilliantly is scout: a ~$30
[RTL-SDR](/reference/rtl-sdr/) running free GopherTrunk **decodes and records
DMR** (and P25, NXDN, and more), so you can watch your local repeaters and
confirm which talkgroups actually carry traffic before building a $460
codeplug around them — with logs and timestamps no transceiver keeps.
Listening needs no license; transmitting on ham bands requires an FCC amateur
license (Part 97, Technician minimum). Pick hardware from
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).

## Who it's for

- **Buy the AT-D578UVIII Plus** if DMR is your area's digital mode and you
  want the most radio per dollar — database, dual APRS, tri-band, Bluetooth —
  and can live with a quirky Windows CPS.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B09CG83Q3Z?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [Yaesu FTM-500DR](/reference/yaesu-ftm-500dr/)** for
  [C4FM](/reference/c4fm/) country and better RF refinement, or the
  [Icom ID-5100A](/reference/icom-id-5100a/) for
  [D-STAR](/reference/d-star/).
- **Spend less** with the [BTECH UV-25X2](/reference/btech-uv-25x2/) if you
  only need analog FM.
- Full rankings: [the 10 best mobile ham radios](/best-mobile-ham-radios/).

## Sources

[^anytone]: [AnyTone AT-D578UVIII factory product page](https://www.anytone.net/video/products-detail-935418) and [BridgeCom Systems AT-D578UVIII Plus listing](https://www.bridgecomsystems.com/products/anytone-at-d578uviii-plus-tri-band-amateur-dmr-mobile-radio) — manufacturer and primary US distributor, on tri-band coverage, DMR Tier I/II, APRS, database capacities, and pricing.
