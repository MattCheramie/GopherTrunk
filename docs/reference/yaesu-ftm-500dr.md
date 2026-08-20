---
slug: yaesu-ftm-500dr
title: Yaesu FTM-500DR
entry_type: hardware
category: ham-radios
description: "The Yaesu FTM-500DR is a 50 W 2m/70cm C4FM System Fusion mobile ham radio with dual receive, built-in GPS and APRS, Bluetooth, and Yaesu's AESS dual-speaker audio — the best all-around FM/digital mobile while new stock lasts."
keywords: Yaesu FTM-500DR, FTM-500DR review, best mobile ham radio, C4FM mobile radio, System Fusion mobile, dual band mobile transceiver, APRS mobile radio, Yaesu mobile radio 2026, FTM-500DR vs FTM-300DR, ham radio for car
aka: [FTM-500DR, FTM-500]
autolink: true
affiliate: true
product:
  name: "Yaesu FTM-500DR"
  brand: Yaesu
  category: Mobile ham transceiver
  lowPrice: "480"
  highPrice: "550"
  url: https://www.amazon.com/dp/B0CLSGMLPC?tag=gophertrunk-20
infobox:
  - { label: Type, value: Dual-band mobile transceiver }
  - { label: Bands, value: "2m / 70cm (dual receive)" }
  - { label: Modes, value: "FM, C4FM System Fusion (AMS)" }
  - { label: Power, value: "50 / 25 / 5 W" }
  - { label: Programming, value: "microSD / ADMS-14 / RT Systems (no CHIRP)" }
  - { label: Price, value: around $500 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B0CLSGMLPC?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [yaesu-ftm-300dr, anytone-at-d578uviii, icom-id-5100a, icom-ic-2730a, rtl-sdr, aprs]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/digital-trunking/ }
related_reading:
  - { title: "Best SDR for GopherTrunk", url: /best-sdr-for-gophertrunk/ }
cite_urls:
  - https://www.yaesu.com/product-detail.aspx?Model=FTM-510DRASP&CatName=VHF%2FUHF+Mobile+Transceivers
  - https://www.dxengineering.com/parts/ysu-ftm-500dr
faq:
  - q: "Is the Yaesu FTM-500DR still worth buying in 2026?"
    a: "Yes, while genuine new stock lasts. Yaesu released the FTM-510DR in 2025 as its direct successor, and dealers now treat the 500DR as late-life — but the radio itself is unchanged and remains the best all-around 2m/70cm mobile at its price. If 500DR stock dries up or prices converge, buy the 510DR instead."
  - q: "Does the FTM-500DR do APRS?"
    a: "Yes — full APRS with a built-in GPS and a 1200/9600 bps modem, one of the best APRS implementations in any mobile. It beacons, receives, and displays stations without any external hardware."
  - q: "Can I program the FTM-500DR with CHIRP?"
    a: "No. CHIRP does not support it. You program it via the microSD card, Yaesu's free ADMS-14 software, or the paid RT Systems ADMS-14 package."
  - q: "Does the FTM-500DR decode both bands in C4FM at once?"
    a: "No. It receives two bands at once (V+U, V+V, or U+U), but only one band decodes C4FM digital at a time — a common gripe on an otherwise excellent dual-receive radio."
  - q: "Do I need a license to use the FTM-500DR?"
    a: "To transmit, yes — an FCC amateur license, Technician class or higher (Part 97). Listening requires no license at all; you can legally monitor the same repeaters with the radio, or for about $30 with an RTL-SDR running free GopherTrunk."
---
**The Yaesu FTM-500DR** is a 50-watt 2m/70cm mobile transceiver that does analog FM
and **C4FM [System Fusion](/reference/system-fusion-ysf/)** digital, receives two
bands at once, and packs in built-in GPS, full [APRS](/reference/aprs/), Bluetooth,
a band scope, and Yaesu's signature **AESS dual-speaker audio**.[^yaesu] It is the
best all-around VHF/UHF mobile you can bolt into a vehicle — with one honest
caveat: Yaesu shipped its successor, the **FTM-510DR**, in 2025, so the 500DR is a
late-life buy.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0CLSGMLPC?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Our best-overall mobile pick** in the [top 10 mobile ham radios](/best-mobile-ham-radios/).
**50 W on 2m and 70cm**, FM + [C4FM](/reference/c4fm/) with auto mode select, **dual
receive**, GPS + [APRS](/reference/aprs/) built in, Bluetooth, detachable color-display
head. **The AESS dual-speaker audio is its signature** — widely praised TX and RX
audio. **~$500 new while stock lasts**; the FTM-510DR (around $650) is the 2025
successor. **No CHIRP** — program via microSD or ADMS-14. Transmitting needs a ham
license; listening doesn't.
</div>

## Overview

The FTM-500DR sits at the top of Yaesu's classic Fusion mobile line, above the
[FTM-300DR](/reference/yaesu-ftm-300dr/). The headline is audio: a front-panel
speaker in the detachable control head plus a body speaker form the **AESS**
acoustic system, and the community consensus is that both its transmit and receive
audio are excellent. The high-visibility 2.4-inch color TFT uses a "funnel"
heads-up layout that is genuinely readable at a glance while driving, and the
menus are far cleaner than the old FTM-400's — though there is **no touchscreen**,
which some buyers expected at this price.

RF is what you'd expect from Yaesu's flagship mobile: a solid receiver, 50/25/5 W
on both bands, and dual receive in any combination (V+U, V+V, U+U). Memory
management runs to roughly 1,100 channels with microSD backup and cloning, and the
radio can act as a portable WiRES-X digital node. There is no cross-band repeat.
Budget ~13–15 A of 13.8 V supply at full power.

**The late-life note, plainly:** Yaesu's 2025 **FTM-510DR** adds a "Super-DX"
receive preamp mode and ASP digital audio processing, and it now occupies Yaesu's
current mobile page while dealers move the 500DR to special-order status. Yaesu
has not published a formal discontinuation date, and new 500DR stock is still
real at Amazon and the big ham dealers. While that stock is genuinely new and
~$150 cheaper than the 510DR, the 500DR remains the smarter buy; once it's gone,
move on to the successor rather than paying scalper prices.

## Modes &amp; features

- **Analog FM and [C4FM System Fusion](/reference/system-fusion-ysf/)** with AMS
  (auto mode select) — the radio follows whatever the repeater is doing. Note only
  **one band decodes C4FM at a time** on dual receive.
- **Built-in GPS and full [APRS](/reference/aprs/)** with a 1200/9600 bps modem —
  beaconing, messaging and station lists with zero outboard boxes. One of the best
  APRS implementations in any mobile.
- **Bluetooth** for headsets (pairing is picky about which headsets — a known
  gripe), **band scope**, microSD logging/cloning, and WiRES-X portable digital
  node capability.
- **Honest mode caveat:** Fusion is a distant third digital network behind
  [DMR](/reference/dmr/) and [D-STAR](/reference/d-star/) in many regions. Check
  what your local repeaters actually run before you pick your digital camp — the
  [GopherTrunk alternative](#gophertrunk-alternative) below is the free way to do
  exactly that.

## Programming

CHIRP does **not** support the FTM-500DR. Three paths, easiest first:

1. **microSD card** — export, edit, and clone the whole configuration.
2. **Yaesu ADMS-14** — Yaesu's free PC programmer.
3. **RT Systems ADMS-14** — the paid, more polished third-party option.

## GopherTrunk alternative

GopherTrunk **receives; it cannot transmit**, so it is not a substitute for the
FTM-500DR — but it is the smartest thing to run *before* you buy it. A ~$30
[RTL-SDR](/reference/rtl-sdr/) running free GopherTrunk lets you monitor your
local repeaters and digital ham traffic and see whether the Fusion, DMR, or
D-STAR side of town is actually active — before you commit $500 to a digital
camp. It also **records and logs** everything it hears, which no mobile
transceiver does. Listening requires no license; transmitting on ham bands
requires an FCC Technician (or higher) license under Part 97. See
[the best SDRs for GopherTrunk](/best-sdr-for-gophertrunk/) for hardware that
pairs well.

## Who it's for

- **Buy the FTM-500DR** if you want the best overall FM/C4FM mobile — top-tier
  audio, dual receive, GPS/APRS, Bluetooth — and can still get it at ~$500 new.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B0CLSGMLPC?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [FTM-300DR](/reference/yaesu-ftm-300dr/)** for most of the same Fusion
  feature set around $400 in new-old-stock, or the FTM-510DR if 500DR stock is gone.
- **Buy the [AnyTone AT-D578UVIII Plus](/reference/anytone-at-d578uviii/)** instead
  if your area runs [DMR](/reference/dmr/), or the
  [Icom ID-5100A](/reference/icom-id-5100a/) for [D-STAR](/reference/d-star/).
- Still comparing? See the full
  [best mobile ham radios guide](/best-mobile-ham-radios/).

## Sources

[^yaesu]: [Yaesu VHF/UHF mobile lineup — FTM-510DR/ASP product page](https://www.yaesu.com/product-detail.aspx?Model=FTM-510DRASP&CatName=VHF%2FUHF+Mobile+Transceivers) — Yaesu, showing the FTM-510DR as the current-line successor; and [DX Engineering FTM-500DR listing](https://www.dxengineering.com/parts/ysu-ftm-500dr) — dealer status, specifications, and pricing for the FTM-500DR.
