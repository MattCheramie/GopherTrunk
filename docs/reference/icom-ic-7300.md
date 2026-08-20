---
slug: icom-ic-7300
title: Icom IC-7300
entry_type: hardware
category: ham-radios
description: "The Icom IC-7300 is the 100W direct-sampling HF/6m SDR transceiver that redefined the entry-level ham base station — touchscreen waterfall, built-in tuner, one-cable digital modes — now in run-out as the IC-7300MK2 arrives."
keywords: Icom IC-7300, IC-7300 review, IC-7300 vs IC-7300MK2, best HF transceiver, direct sampling transceiver, HF SDR transceiver, Icom IC-7300 discontinued, IC-7300 FT8, ham radio base station, 100 watt HF radio, IC-7300 price 2026
aka: [IC-7300, "7300"]
autolink: true
affiliate: true
product:
  name: "Icom IC-7300"
  brand: Icom
  category: Ham radio base station (HF/6m transceiver)
  lowPrice: "940"
  highPrice: "1040"
  url: https://www.amazon.com/dp/B01M10HJXW?tag=gophertrunk-20
infobox:
  - { label: Type, value: "HF/6m base transceiver (direct-sampling SDR)" }
  - { label: Bands, value: "160–6 m TX; 0.030–74.8 MHz RX" }
  - { label: Modes, value: "SSB, CW, RTTY, AM, FM" }
  - { label: Power, value: "100 W (25 W AM)" }
  - { label: Programming, value: "USB CAT + audio codec, SD card, CI-V" }
  - { label: Price, value: "around $1,040 (~$940 after rebate)" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B01M10HJXW?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [yaesu-ftdx10, icom-ic-7610, yaesu-ft-991a, xiegu-g90, rtl-sdr, direct-sampling]
related_lessons:
  - { title: "Learn RF & SDR", url: /learn/rf-sdr/ }
related_reading:
  - { title: "Best ham radio base stations", url: /best-ham-radio-base-stations/ }
cite_urls:
  - https://www.icomamerica.com/lineup/products/IC-7300/
faq:
  - q: "Is the Icom IC-7300 still worth buying in 2026?"
    a: "Yes — while new stock lasts. Icom has announced the IC-7300MK2 (HDMI, USB-C, native LAN remote) and the original is out of production, but US dealers still sell new units around $1,040 with an active rebate. Nothing near the price matches its receiver, waterfall, and ten years of community knowledge."
  - q: "Is the IC-7300 discontinued?"
    a: "Effectively, yes. UK dealers already list it as no longer in production and the IC-7300MK2 is its announced successor, but as of August 2026 major US dealers still have new run-out stock with an Icom rebate. Treat it as a last-call buy at a good price, not a dead product."
  - q: "Does the IC-7300 do FT8 and digital modes?"
    a: "Very well. A single USB cable carries both CAT control and the built-in audio codec, so WSJT-X, fldigi, and friends work with no external interface — one reason it became the default first HF rig. There's also an SD slot for recording and screen captures."
  - q: "Does the IC-7300 cover VHF and UHF?"
    a: "No. It transmits 160 through 6 meters and receives 0.030–74.8 MHz general coverage. For 2 m/70 cm repeaters in the same box, look at the Yaesu FT-991A; for local monitoring only, a ~$30 RTL-SDR running free GopherTrunk covers VHF/UHF."
  - q: "Do I need a license to use the IC-7300?"
    a: "To transmit, yes — an FCC amateur license (Part 97, Technician class minimum, with General unlocking most HF). Listening requires no license at all, on this radio or on a cheap SDR running GopherTrunk."
---
**The Icom IC-7300** is the radio that mainstreamed the
[direct-sampling](/reference/direct-sampling/) SDR receiver, and a decade on it is
still the default answer to "what's my first real HF base station?" — 100 W on
160–6 m, a 4.3-inch touchscreen with a real-time spectrum scope and
[waterfall](/reference/waterfall-display/), a built-in
[antenna tuner](/reference/antenna-tuner/), and one-cable USB digital modes for
around $1,040.[^icom] Icom has announced its successor, the **IC-7300MK2**, so the
original is now a run-out buy — still widely available new from US dealers, often
with a rebate.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B01M10HJXW?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Our best-overall base station** in
[best ham radio base stations](/best-ham-radio-base-stations/). **Direct-sampling
SDR receiver** with the touchscreen scope that changed the market, **built-in
tuner**, **one-cable FT8**. **160–6 m, 100 W** — no VHF/UHF (that's the
[FT-991A](/reference/yaesu-ft-991a/)'s job). **End of production:** the IC-7300MK2
is coming, but new run-out stock is real and rebated at **~$940–1,040**.
**Transmitting needs an FCC license; listening doesn't** — try a
[$30 RTL-SDR + GopherTrunk](/best-sdr-for-gophertrunk/) first.
</div>

## Overview

When the IC-7300 shipped, a real-time spectrum scope was flagship money. Icom put
an RF [direct-sampling](/reference/direct-sampling/) receiver — the ADC digitizes
the band directly, no conventional mixer chain — behind 15 discrete band-pass
filters and a color touchscreen, and sold it at an entry-level price. The result
has been *the* recommended first HF rig for ten years, with an enormous install
base: presets, mods, YouTube walkthroughs, and working WSJT-X configs for
everything you'd ever plug into it.

The 2026 wrinkle is the **IC-7300MK2** transition. Icom has announced the
successor (it adds HDMI out, USB-C, and native LAN remote), and UK dealers already
list the original as no longer in production — yet US dealers still carry new
stock with an active Icom rebate. That makes this an honest last-call buy, not a
trap: if the MK2's connectivity matters to you, wait; if you want the proven radio
at its best-ever effective price, the run-out IC-7300 is still the value king.
Used units commonly run $700–900.

One licensing note before the specs: **transmitting on the ham bands requires an
FCC amateur license** (Part 97 — Technician class at minimum, General for most HF
privileges). **Listening requires none**, on this radio or anything else.

## Bands, modes &amp; power

- **TX:** 160 through 6 meters, **100 W** SSB/CW/RTTY/FM (25 W AM). UK/EU
  versions add 4 m.
- **RX:** 0.030–74.8 MHz general coverage — [shortwave
  broadcast](/reference/shortwave-broadcast/), utilities, and the ham bands.
- **Modes:** [SSB](/reference/single-sideband/), [CW](/reference/morse-code/),
  RTTY, AM, FM. No digital voice — this is a classic HF operating radio, not a
  DMR/Fusion box.
- **Tuner:** built-in automatic antenna tuner. Single SO-239 antenna jack — no
  dedicated RX antenna port, one of the few spec-sheet gaps.
- **Power draw:** 13.8 V DC at ~21 A on transmit; budget a 25 A supply.

## Receiver &amp; interface

The receiver is the story. The direct-sampling front end plus the touchscreen
scope/waterfall means you *see* the band and pounce, instead of tuning blind. In
genuinely brutal contest-superstation environments the front end can overload
where the [IC-7610](/reference/icom-ic-7610/) and flagships hold up — the built-in
digi-sel and attenuator mitigate it — and the fan is audible on long digital
transmissions, but neither is a real-world complaint for most stations. The
reliability record over a decade is very good.

Digital modes are where it spoiled everyone: **one USB cable** carries CAT control
and a built-in audio codec, so [FT8](/reference/ft8/) via WSJT-X, fldigi, and
logger integration need no external interface. An SD card slot records QSOs and
screen captures.

## Programming &amp; software

CI-V CAT control works with effectively everything — WSJT-X, fldigi, N1MM,
Win4Icom, RT Systems. CHIRP-next currently lists IC-7300 memory support, though
that's of limited value on an HF rig (most owners use the SD card or RT Systems)
and worth verifying against the current CHIRP model list if it matters to you.
Firmware updates load from the SD card.

A note on the Amazon listing: like most current ham gear on Amazon, the IC-7300
is carried as reseller **bundles** (the confirmed listing pairs the radio with a
pocket reference card; others add a microphone). The radio is the same unit
dealers sell — just read what's in the box.

## GopherTrunk alternative

GopherTrunk **receives; it cannot transmit** — so it is not a substitute for the
IC-7300. It is the smart *precursor*. A ~$30 [RTL-SDR](/reference/rtl-sdr/)
running free GopherTrunk lets you monitor local repeaters and digital ham traffic
before you spend a dollar on a transceiver or a license: you learn what's actually
active in your area, and it **records and logs** activity in a way no transceiver
does. Listen first, then buy the rig that matches what you heard. See
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/) for the hardware
short-list, and [best HF SDR](/best-hf-sdr/) if HF listening is the goal.

## Who it's for

- **Buy the IC-7300** if you want the best all-around HF/6m base station under
  $1,500 — first rig or forever rig — and buy it while run-out stock and the
  rebate last.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B01M10HJXW?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Wait for the IC-7300MK2** if HDMI/USB-C/LAN remote are must-haves, or step up
  to the [IC-7610](/reference/icom-ic-7610/) for dual receivers. Want VHF/UHF and
  [C4FM](/reference/c4fm/) in the same box? [FT-991A](/reference/yaesu-ft-991a/).
- **Spend less** with the [Xiegu G90](/reference/xiegu-g90/) (~$450, 20 W) or the
  [FT-891](/reference/yaesu-ft-891/) (~$650, 100 W, no tuner). Full rankings:
  [best ham radio base stations](/best-ham-radio-base-stations/).

## Sources

[^icom]: [Icom IC-7300 product page](https://www.icomamerica.com/lineup/products/IC-7300/) — Icom America, on the RF direct-sampling receiver, 15 band-pass filters, real-time touchscreen scope/waterfall, built-in ATU, band coverage, and power specs.
