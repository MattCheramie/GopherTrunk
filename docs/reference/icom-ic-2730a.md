---
slug: icom-ic-2730a
title: Icom IC-2730A
entry_type: hardware
category: ham-radios
description: "The Icom IC-2730A is a 50 W analog 2m/70cm mobile ham radio with true simultaneous dual receive, wideband RX including air band, and Icom build quality — the consensus no-nonsense FM mobile, around $330 in 2026."
keywords: Icom IC-2730A, IC-2730A review, best analog mobile ham radio, dual band mobile transceiver, dual receive mobile radio, 2m 70cm mobile radio, Icom FM mobile, CHIRP mobile radio, best mobile ham radio 2026
aka: [IC-2730A, IC-2730]
autolink: true
affiliate: true
product:
  name: "Icom IC-2730A"
  brand: Icom
  category: Mobile ham transceiver
  lowPrice: "330"
  highPrice: "360"
  url: https://www.amazon.com/dp/B00QOR05EY?tag=gophertrunk-20
infobox:
  - { label: Type, value: Dual-band mobile transceiver }
  - { label: Bands, value: "2m / 70cm (true dual receive)" }
  - { label: Modes, value: "FM only (no digital)" }
  - { label: Power, value: "50 / 15 / 5 W" }
  - { label: Programming, value: "CHIRP / CS-2730 / RT Systems" }
  - { label: Price, value: around $330 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B00QOR05EY?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [icom-id-5100a, kenwood-tm-v71a, btech-uv-25x2, yaesu-ftm-300dr, rtl-sdr, police-scanner]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Best SDR for GopherTrunk", url: /best-sdr-for-gophertrunk/ }
cite_urls:
  - https://www.icomamerica.com/lineup/products/IC-2730A/
faq:
  - q: "Is the Icom IC-2730A still in production?"
    a: "Yes — as of August 2026 it is current on both Icom America's and Icom Japan's lineup pages and stocked new at the major dealers, typically around $330."
  - q: "Does the IC-2730A have digital modes or APRS?"
    a: "No. It is analog FM only — no D-STAR, no DMR, no C4FM, and no GPS or APRS modem. You are paying for RF quality and true dual receive, not features. For digital, look at the ID-5100A (D-STAR) or AnyTone AT-D578UVIII Plus (DMR)."
  - q: "Can the IC-2730A receive the air band?"
    a: "Yes — its wideband receive covers 118–174 MHz including AM air band, plus 375–550 MHz. Transmit is amateur 2m/70cm only."
  - q: "Does CHIRP support the IC-2730A?"
    a: "Yes, CHIRP has an IC-2730 driver, and Icom's CS-2730 software and RT Systems are also available. Programming this radio is easy by any route."
  - q: "Do I need a license for the IC-2730A?"
    a: "To transmit, yes — an FCC amateur license, Technician class or above (Part 97). Listening needs no license, on this radio or on a $30 RTL-SDR running free GopherTrunk."
---
**The Icom IC-2730A** is the consensus pick for a no-nonsense analog dual-band
mobile: **50 watts on 2m and 70cm**, **true simultaneous dual receive**
(including V/V and U/U), wideband receive down to the AM air band, and Icom
build quality — with no digital modes, no GPS, and no gimmicks.[^icom] Still in
production in 2026 at around $330, it is the radio you buy when you want the RF
done right and nothing else.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B00QOR05EY?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The best pure-FM mobile.** Clean, sensitive receiver; loud audio; **true dual
receive** in any band combination; wideband RX 118–174 / 375–550 MHz including
air band AM; dead-simple operation; **CHIRP-supported**. **No digital voice, no
[APRS](/reference/aprs/), no GPS** — that's the deal. Remote-head-only design
(separation cable included, but mounting brackets cost extra). **~$330**, in
production. Transmitting needs an FCC ham license (Technician+); listening
doesn't.
</div>

## Overview

The IC-2730A is a remote-head-only design: the controller separates from the RF
deck, and the separation cable comes in the box. The catch every buyer should
know up front is that the **mounting hardware for the head costs extra** — the
MBF-1 and MBA-2 brackets are add-ons — and some controls split between the head
and the DTMF microphone, which takes a little adjustment.

What you get for the money is Icom's RF: a clean, sensitive receiver that holds
up where budget imports desense, loud audio, and a large backlit LCD (not color,
not touch) that is legible at a glance. **True simultaneous dual receive**
covers V/V and U/U as well as V/U — genuinely two receivers, not dual watch.
It holds 1,000 memories, transmits 50/15/5 W on both bands at ~13 A full-power
draw, and offers an optional VS-3 Bluetooth headset via the optional UT-133A
module. No GPS, no APRS modem, no cross-band repeat, no digital voice.

That last line is the whole trade: rigs like the
[AnyTone AT-D578UVIII Plus](/reference/anytone-at-d578uviii/) pile on features
for similar money, but none of the budget feature-radios match this receiver.
You pay Icom for RF quality, not a spec sheet.

## Modes &amp; coverage

- **Analog FM only** on amateur 2m and 70cm — no
  [D-STAR](/reference/d-star/), [DMR](/reference/dmr/), or
  [C4FM](/reference/c4fm/).
- **Wideband receive:** 118–174 MHz (including **AM air band**) and 375–550
  MHz — handy for monitoring public safety and aviation between QSOs, within
  the limits of an FM mobile's scanning. For serious monitoring, a
  [police scanner](/reference/police-scanner/) or an SDR does it better.
- **True dual receive** with independent volume/squelch per band.

## Programming

Every route works, and all are painless:

1. **CHIRP** (free) — the IC-2730 driver is mature; this is the easy path.
2. **Icom CS-2730** — Icom's own software.
3. **RT Systems** — the paid, polished option.

## GopherTrunk alternative

GopherTrunk **receives only — it cannot transmit** — so it is no substitute for
the IC-2730A's half of a QSO. It is the smart first step, though: before
spending $330, a ~$30 [RTL-SDR](/reference/rtl-sdr/) running free GopherTrunk
lets you hear every local repeater and see how active your area's 2m/70cm
scene really is — and unlike any mobile, it **records, logs, and timestamps**
what it hears, covering the digital modes this radio skips. Listening requires
no license; transmitting on the ham bands requires an FCC amateur license
(Part 97, Technician class minimum). Dongle recommendations are in
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).

## Who it's for

- **Buy the IC-2730A** if you want the best-behaved analog FM mobile — a
  commuter/repeater rig with a receiver that holds up in RF-dense areas — and
  you don't need digital voice or APRS.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B00QOR05EY?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [ID-5100A](/reference/icom-id-5100a/)** for the same Icom RF plus
  [D-STAR](/reference/d-star/) and a touchscreen, ~$130 more.
- **Buy a used [Kenwood TM-V71A](/reference/kenwood-tm-v71a/)** for similar
  analog quality plus cross-band repeat at a lower used price, or a
  [BTECH UV-25X2](/reference/btech-uv-25x2/) if $330 is out of reach.
- Full field: [the 10 best mobile ham radios](/best-mobile-ham-radios/).

## Sources

[^icom]: [Icom IC-2730A product page](https://www.icomamerica.com/lineup/products/IC-2730A/) — Icom America, on the 50/15/5 W output, simultaneous dual receive, 118–174 / 375–550 MHz wideband receive, remote-head design, and options.
