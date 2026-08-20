---
slug: btech-uv-pro
title: BTECH UV-PRO
entry_type: hardware
category: ham-radios
description: "The BTECH UV-PRO is an IP67 dual-band ham handheld with GPS, APRS, Bluetooth, USB-C charging and phone-app programming for around $165 — the feature-per-dollar standout of the budget HT class."
keywords: BTECH UV-PRO, UV-PRO review, best budget ham handheld, IP67 ham radio, APRS handheld cheap, Bluetooth ham radio, USB-C ham handheld, BTECH review, Vero VR-N76, off-grid radio
aka: [UV-PRO]
autolink: true
affiliate: true
product:
  name: "BTECH UV-PRO"
  brand: BTECH
  category: Ham handheld transceiver
  lowPrice: "160"
  highPrice: "170"
  url: https://www.amazon.com/dp/B0DBW24N8M?tag=gophertrunk-20
infobox:
  - { label: Type, value: Dual-band handheld transceiver }
  - { label: Bands, value: "TX 144/430 MHz; RX incl. AM air band, NOAA, FM broadcast" }
  - { label: Modes, value: "Analog FM; APRS TX/RX; radio-to-radio text" }
  - { label: Power, value: "~5–6 W (advertised figures vary)" }
  - { label: Programming, value: "BTECH phone app over Bluetooth (no CHIRP as of 2026)" }
  - { label: Price, value: around $165 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B0DBW24N8M?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [baofeng-bf-f8hp, baofeng-uv-5r, yaesu-ft-60r, yaesu-ft-5dr, rtl-sdr, aprs]
related_lessons:
  - { title: "Analog vs. digital voice", url: /learn/rf-sdr/digital-voice/ }
  - { title: "Legal & ethical monitoring", url: /learn/rf-sdr/legal-ethical/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://baofengtech.com/product/uv-pro/
faq:
  - q: "Does the BTECH UV-PRO do APRS?"
    a: "Yes — APRS transmit and receive with built-in GPS, plus a Bluetooth KISS-style TNC that popular phone APRS and off-grid messaging apps can use. That combination under $170 is the radio's headline feature."
  - q: "How many watts is the UV-PRO?"
    a: "Treat it as roughly 5–6 W. Some marketing copy says 'up to 10 W' for the platform, but that figure likely belongs to the UV-50PRO sibling — don't buy on the 10 W claim."
  - q: "Can I program the UV-PRO with CHIRP?"
    a: "Not as of our 2026 research — there's an open CHIRP new-model request, and BTECH itself routes owners to its phone app over Bluetooth for programming. Re-verify at chirpmyradio.com if CHIRP support is a dealbreaker."
  - q: "Is the UV-PRO waterproof?"
    a: "Yes — IP67 rated, which is remarkable at the price. It also charges over USB-C from its 2,600 mAh battery."
  - q: "Do I need a license to use the UV-PRO?"
    a: "Transmitting on 144/430 MHz requires an FCC amateur license (Technician class minimum). Listening — including its AM air band, NOAA and FM broadcast receive — requires none."
---
**The BTECH UV-PRO** packs IP67 waterproofing, built-in **GPS and
[APRS](/reference/aprs/)**, Bluetooth, USB-C charging and an AM air band
scanner into a dual-band 144/430 MHz handheld for around **$165** — a feature
list that undercuts radios costing twice as much.[^btech] The catch that
defines it: it's an **app-first** radio, configured over Bluetooth from BTECH's
phone app rather than from the front panel or CHIRP.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0DBW24N8M?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The feature-per-dollar standout of the budget class.** IP67 + GPS + APRS +
Bluetooth TNC + USB-C under $170; phone-app setup that newcomers genuinely find
easy. The trades: **app dependence** for deep configuration, a SoC-class
receiver that overloads near strong transmitters, no digital voice
([DMR](/reference/dmr/)/[C4FM](/reference/c4fm/)/[D-STAR](/reference/d-star/)),
mixed audio reports, and fast-moving firmware that changes behavior between
versions. Output is realistically ~5–6 W despite some "10 W" marketing.
Transmitting needs an FCC Technician license; listening needs none.
</div>

## Overview

The UV-PRO is built on the Vero VR-N76 platform and has become the darling of
the preparedness and APRS-messaging crowd for one reason: its Bluetooth link
exposes a KISS-style TNC that phone apps can drive, turning the radio plus a
phone into an off-grid position-reporting and text-messaging system. Add
radio-to-radio text without any infrastructure, NOAA weather alerts, an AM air
band scanner, and a 2,600 mAh USB-C battery in an IP67 body, and it's easy to
see why it sells.

The honest counterweight: this is a software-defined-on-a-chip budget radio.
The front end overloads near strong transmitters like the rest of its class,
speaker and transmit audio reports are mixed, and firmware evolves fast enough
that behavior genuinely changes between versions. Traditionalists who want
knobs, menus and CHIRP will chafe at doing everything through a phone. And
despite the connected feel, there's no digital voice mode of any kind.

## Modes &amp; features

- **Analog FM** on 144/430 MHz, ~5–6 W realistic output (hedge the platform's
  "10 W" marketing — that figure likely belongs to the UV-50PRO sibling).
- **[APRS](/reference/aprs/) TX/RX** with built-in **GPS**, plus a
  BLE KISS-style TNC used by phone APRS and off-grid mesh/text apps.
- **Radio-to-radio text messaging** with no infrastructure.
- **Receive extras**: AM air band scanning, NOAA weather alerts, FM broadcast.
- **IP67** waterproof; 2,600 mAh Li-ion with **USB-C** charging.

## Programming

The primary method — per BTECH itself — is the **BTECH programming app**
(iOS/Android) over Bluetooth, and it's genuinely one of the easiest setups in
ham radio for a newcomer. **CHIRP does not support it** as of our research
date (an open new-model request exists); re-check chirpmyradio.com if that
matters to you, and expect the app to remain the deep-configuration path
either way.

## GopherTrunk alternative

GopherTrunk receives only — no transmitter, so no substitute for a UV-PRO. Its
role is upstream of the purchase: a ~$30 [RTL-SDR](/reference/rtl-sdr/) running
free GopherTrunk shows you what's live on your local repeaters — analog FM,
[APRS](/reference/aprs/) activity, or a digital mode this radio can't do — and
records and logs all of it, which no HT will. If your area turns out to run
DMR, the money is better spent on an
[AT-D878UVII Plus](/reference/anytone-at-d878uvii-plus/). Hardware picks:
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).

## Who it's for

- **Buy the UV-PRO** if you want APRS + GPS + waterproofing on a budget, you're
  comfortable configuring from a phone, or off-grid text/position reporting is
  the goal.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B0DBW24N8M?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [Yaesu FT-60R](/reference/yaesu-ft-60r/)** instead for a
  traditional, tougher, CHIRP-programmable analog HT with a far better front
  end (~$180).
- **Spend up** to a [Yaesu FT-5DR](/reference/yaesu-ft-5dr/) for digital voice
  plus deeper APRS, or **down** to a
  [Baofeng BF-F8HP](/reference/baofeng-bf-f8hp/) if you only need a beater.
  Full rankings: [best handheld ham radios](/best-handheld-ham-radios/).

## Sources

[^btech]: [BTECH UV-PRO product page](https://baofengtech.com/product/uv-pro/) — BTECH, on APRS/GPS, Bluetooth programming and TNC features, IP67 rating, USB-C charging, and receive coverage.
