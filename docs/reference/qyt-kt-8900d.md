---
slug: qyt-kt-8900d
title: QYT KT-8900D
entry_type: hardware
category: ham-radios
description: "The QYT KT-8900D is the cheapest way to put 20+ watts of 2m/70cm FM in a vehicle — a ~$100 quad-standby mini mobile that CHIRP programs, with the QC lottery and marginal spectral purity of its class."
keywords: QYT KT-8900D, KT-8900D review, cheapest mobile ham radio, budget dual band mobile, quad standby mobile radio, mini mobile transceiver, Chinese mobile ham radio, CHIRP KT8900D, best cheap ham radio 2026
aka: [KT-8900D, KT8900D]
autolink: true
affiliate: true
product:
  name: "QYT KT-8900D"
  brand: QYT
  category: Mobile ham transceiver
  lowPrice: "90"
  highPrice: "110"
  url: https://www.amazon.com/dp/B01MYY5JWF?tag=gophertrunk-20
infobox:
  - { label: Type, value: Mini dual-band mobile transceiver }
  - { label: Bands, value: "2m / 70cm (quad standby)" }
  - { label: Modes, value: "FM only (no digital)" }
  - { label: Power, value: "~25 W VHF / ~20 W UHF (claimed)" }
  - { label: Programming, value: "CHIRP (community standard) / QYT software" }
  - { label: Price, value: around $100 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B01MYY5JWF?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [btech-uv-25x2, icom-ic-2730a, kenwood-tm-v71a, rtl-sdr, police-scanner]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Best SDR for GopherTrunk", url: /best-sdr-for-gophertrunk/ }
cite_urls:
  - https://www.twowayradio.us/product/qyt-kt-8900d-25w-vehicle-mounted-two-way-radio-mini-mobile-radio/
faq:
  - q: "Is the QYT KT-8900D any good?"
    a: "As a ~$100 beater or backup, yes — it's the cheapest way to put 20+ watts in a vehicle, and CHIRP programs it easily. As a primary rig for demanding use, no: measured power often lands below spec, spectral purity is marginal on some units, and QC is a lottery. Buy it knowing exactly what it is."
  - q: "What is quad standby on the KT-8900D?"
    a: "The display shows four frequencies at once and the radio watches all four, receiving whichever goes active (one at a time). It's genuinely handy for keeping a couple of repeaters plus two simplex channels in view."
  - q: "Does the KT-8900D work with CHIRP?"
    a: "Yes — CHIRP has a KT8900D driver and is the community-standard way to program it (the radio is also sold rebadged as the CRT 279 UV). QYT's factory software exists but CHIRP is the saner route."
  - q: "Is there a GMRS version of the KT-8900D?"
    a: "Yes, and that's a buying trap: a GMRS-labeled variant exists under a separate listing. For ham use, make sure you're buying the amateur 2m/70cm model, not the GMRS one."
  - q: "Do I need a license for the KT-8900D?"
    a: "Transmitting on ham bands requires an FCC amateur license (Part 97, Technician minimum). Listening requires no license — and a $30 RTL-SDR with free GopherTrunk is a better listening tool than this radio anyway."
---
**The QYT KT-8900D** is the cheapest way to put **20-plus watts** of 2m/70cm FM
in a vehicle: a palm-sized mobile with a **quad-standby** display, 200
memories, and CHIRP support for around **$100**.[^qyt] It earns a slot on our
mobile list as the **most affordable** pick — and it keeps that slot only
because we're upfront that this is a beater-grade radio, not a primary rig.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B01MYY5JWF?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The most affordable real mobile.** ~25 W VHF / ~20 W UHF claimed, quad-watch
display, 200 memories, palm-sized, under 10 A draw, **CHIRP-programmable**.
**~$90–110** (Amazon pricing fluctuates). The honest ledger: measured power
often below spec, **marginal spurious/harmonic performance on some units**,
receiver desense near strong signals, tinny speaker, QC lottery, display
washes out in sunlight. **Don't buy the GMRS-labeled variant by mistake.**
Fine as a beater/backup; not for demanding use. License to transmit; none to
listen.
</div>

## Overview

QYT's long-running budget mobile does one thing genuinely well: it puts real
mobile power in a vehicle for handheld money. The **quad standby** display —
four frequencies visible, one active at a time — is its signature feature and
is legitimately handy for watching two repeaters and two simplex channels at
once. The chassis is tiny, draw is under 10 A, receive covers 136–174 and
400–480 MHz, and there is no detachable face, no GPS/APRS, no digital modes,
and no cross-band repeat.

Now the part most listings skip. Community and lab-test experience with
QYT-class radios is consistent: **measured output often lands below the claimed
spec**, and **spurious-emission and harmonic performance is marginal on some
units** — a recurring lab complaint for QYT and its rebadged clones, and your
responsibility as the licensee. Add receiver desense near strong signals, a
tinny speaker, a "D" display that washes out in sunlight, and build-quality
roulette between units. Prices float around $90–110 on Amazon. One more trap:
a **GMRS-labeled variant** of this radio exists under a separate listing —
make sure you buy the amateur 2m/70cm model.

## Modes &amp; coverage

- **Analog FM only** — no [DMR](/reference/dmr/), [D-STAR](/reference/d-star/),
  [C4FM](/reference/c4fm/), or [APRS](/reference/aprs/).
- **Quad standby** across RX 136–174 / 400–480 MHz; 200 memories.
- Claimed ~25 W VHF / ~20 W UHF — treat as optimistic.

## Programming

**CHIRP** is the community standard (KT8900D driver; the radio is also sold
rebadged as the CRT 279 UV). QYT's own Windows software exists, but CHIRP is
the saner, better-documented path and rescues you from the front-panel menus.

## GopherTrunk alternative

GopherTrunk **receives; it cannot transmit** — the KT-8900D's one clear win.
But if what draws you to a $100 radio is mostly *listening*, stop: a ~$30
[RTL-SDR](/reference/rtl-sdr/) running free GopherTrunk out-receives this
radio's desense-prone front end, covers the digital modes it can't touch, and
**records and logs every call**. Monitor your local repeaters free, figure out
what's active, and then decide whether the transmitter you buy should be this
one or something better. Listening requires no license; transmitting on ham
bands requires an FCC amateur license (Part 97, Technician class minimum).
See [best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).

## Who it's for

- **Buy the KT-8900D** as a glovebox backup, a farm-truck beater, a first
  experiment, or a loaner — uses where $100 and "mostly works" is the right
  call.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B01MYY5JWF?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [BTECH UV-25X2](/reference/btech-uv-25x2/)** (~$130) for a
  noticeably better-sorted radio in the same budget class.
- **Buy the [Icom IC-2730A](/reference/icom-ic-2730a/)** (~$330) or a used
  [Kenwood TM-V71A](/reference/kenwood-tm-v71a/) when the radio is your
  primary — the receiver difference is not subtle.
- Everything ranked: [the 10 best mobile ham radios](/best-mobile-ham-radios/).

## Sources

[^qyt]: [QYT KT-8900D dealer reference page](https://www.twowayradio.us/product/qyt-kt-8900d-25w-vehicle-mounted-two-way-radio-mini-mobile-radio/) — on the quad-standby display, claimed power output, frequency coverage, and mini-mobile form factor (QYT's own site is intermittently available).
