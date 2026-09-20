---
slug: baofeng-uv-5rm
title: Baofeng UV-5RM
entry_type: hardware
category: ham-radios
description: "The Baofeng UV-5RM is the modernized UV-5R — USB-C charging, 999 channels, a color screen, multi-band receive and NOAA weather for around $30 — an honest review covering the '10W' marketing, the inherited front-end and FCC-certification baggage, and where a $30 SDR beats it."
keywords: Baofeng UV-5RM, UV-5RM review, UV-5RM vs UV-5R, 10W Baofeng, 999 channel Baofeng, USB-C ham radio, UV-5RM CHIRP, multi-band Baofeng, NOAA weather handheld, budget ham handheld
aka: [UV-5RM, UV5RM, UV-5RM Plus]
autolink: true
affiliate: true
product:
  name: "Baofeng UV-5RM"
  brand: Baofeng
  category: Ham handheld transceiver
  lowPrice: "30"
  highPrice: "60"
  url: https://www.amazon.com/dp/B0D7Q3X7XL?tag=gophertrunk-20
infobox:
  - { label: Type, value: Dual-band handheld transceiver }
  - { label: Bands, value: "TX 144–148 / 420–450 MHz; multi-band RX" }
  - { label: Modes, value: Analog FM only }
  - { label: Power, value: "10/5/1 W tri-power (claimed)" }
  - { label: Channels, value: "999, USB-C charging" }
  - { label: Programming, value: "CHIRP (CHIRP-next)" }
  - { label: Price, value: around $30 (2-pack ~$58) }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B0D7Q3X7XL?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [baofeng-uv-5r, baofeng-bf-f8hp, btech-uv-pro, yaesu-ft-60r, rtl-sdr, fcc]
related_lessons:
  - { title: "Analog vs. digital voice", url: /learn/rf-sdr/digital-voice/ }
  - { title: "Legal & ethical monitoring", url: /learn/rf-sdr/legal-ethical/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://www.baofengradio.com/products/uv-5rm-plus-8w-multi-band-radio
faq:
  - q: "What is the difference between the UV-5RM and the UV-5R?"
    a: "Same platform, modernized shell. The UV-5RM adds USB-C charging, 999 channels instead of 128, a 1.77-inch color screen, wider multi-band receive (including AM aircraft and NOAA weather), and a higher '10W' power claim. Underneath it is still the analog-FM UV-5R family — same wide-open front end, same variant chaos, same CHIRP dependency."
  - q: "Does the UV-5RM really put out 10 watts?"
    a: "Treat '10W' the way you'd treat the BF-F8HP's '8W': optimistic. Independent testing of these radios routinely measures below the label, the step from 5 W to a real 8–10 W is only about 2–3 dB (barely a signal-report notch), and running high power cooks the finals and drains the battery fast. Antenna height and a decent antenna matter far more than the wattage number."
  - q: "Is the Baofeng UV-5RM legal to use?"
    a: "Owning and listening: yes, no license needed. Transmitting on the amateur bands: legal with an FCC amateur license (Technician minimum). The hardware caveat is inherited from the whole UV-5R family — the front panel will happily key up outside the ham bands, which is not legal without the relevant authorization, and the family's spurious-emission and certification history (the FCC's 2018 Enforcement Advisory targeted exactly this class of import radio) applies here too."
  - q: "Does the UV-5RM work with CHIRP?"
    a: "Yes, via a current CHIRP-next build (select the UV-5RM model) — and CHIRP is effectively mandatory, because hand-entering 999 channels on the keypad is masochism. Buy a genuine-FTDI programming cable; counterfeit chips are rampant and are the usual cause of 'my Baofeng won't connect.'"
  - q: "Can the UV-5RM receive DMR or digital voice?"
    a: "No. It is analog FM only — no DMR, C4FM, D-STAR, P25 or any trunked digital. If you want to hear the digital and trunked systems most agencies now use, that is a receiver job: a ~$30 RTL-SDR running GopherTrunk decodes them and the UV-5RM cannot."
  - q: "Should I buy the UV-5RM or the BF-F8HP?"
    a: "The BF-F8HP is the settled, mature version of this platform with US-based BTECH support. The UV-5RM is the newer, better-equipped one — USB-C, color screen, far more channels and wider receive — at a lower price. If you want the modern feature set and can accept newer-model QC risk, the UV-5RM; if you want the known quantity, the BF-F8HP."
---
**The Baofeng UV-5RM** is what the [UV-5R](/reference/baofeng-uv-5r/) looks like
in 2026 — **USB-C charging**, **999 channels**, a small color screen, wide
**multi-band receive** and NOAA weather, all for around **$30** (about **$58**
for the two-pack).[^baofeng] It is a genuine, useful modernization of the
cheapest handheld in radio. It also inherits the whole UV-5R family's baggage:
the **wide-open front end**, the **QC lottery**, the **spurious-emission and
FCC-certification history**, and a **"10W" claim** worth the same skepticism as
every other Baofeng power number. Both halves are true, and an honest review has
to hold them together.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0D7Q3X7XL?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The UV-5R, brought up to date — buy it as a modern $30 beater.** USB-C so you
can charge it off the same brick as everything else, 999 channels, a color
screen, multi-band RX with AM airband and NOAA weather, and CHIRP support. The
caveats travel with the platform: the **"10W" is marketing** (call it 5–8 W
real), the **front end still overloads** near strong transmitters, **QC still
varies** unit to unit, and the family's **certification mess** applies. Analog
FM only — no digital. Transmitting needs an FCC amateur license (Technician
minimum); listening needs none.
</div>

## Overview

The UV-5RM is Baofeng's answer to a decade of "the UV-5R is great, but…" It
keeps the thing people actually liked — a radio you can afford to lose, lend or
leave in a go-bag — and fixes the most-complained-about ergonomics. **USB-C**
means no more proprietary desk cradle and dead wall-warts; you charge it off a
phone brick or a battery pack. **999 memory channels** replace the classic 128,
which matters the moment you load a real repeater directory. A **1.77-inch color
TFT** and a cleaned-up menu make it legible where the original was cryptic. And
the **receive** coverage is genuinely broader — FM broadcast, AM aircraft band,
136–174 and 400–520 MHz, plus 220-ish and NOAA weather with alert — so as a
*listen-around* radio it does more than a UV-5R ever did.

The problems are the platform's, not the shell's. The **"10W" tri-power** claim
(10/5/1 W) deserves the same treatment as the [BF-F8HP](/reference/baofeng-bf-f8hp/)'s
"8 W": measured output tends to land under the label, the difference from 5 W is
a couple of dB, and high power runs the radio hot. The **receiver front end** is
still wide open — park it next to a strong pager, FM tower or public-safety site
and it overloads and desenses, same as its ancestors. **QC** on a young model is
its own gamble; "buy two" remains Baofeng folk wisdom for a reason. And the
**certification history** — many UV-5R-family units failing independent
spurious-emission tests (ARRL and others), the murky Part 90 grants, the
[FCC](/reference/fcc/)'s 2018 Enforcement Advisory aimed squarely at this class
of import — rides along with the name.

## Modes &amp; features

- **Analog FM only** — no DMR, [C4FM](/reference/c4fm/),
  [D-STAR](/reference/d-star/), P25 or [trunking](/reference/trunked-radio/).
- **Multi-band RX**: FM broadcast, AM aircraft, VHF/UHF ham and commercial, plus
  NOAA weather channels with weather-alert.
- **999 channels**, [CTCSS](/reference/ctcss/)/[DCS](/reference/dcs/), dual
  watch, VOX, DTMF, scan, one-key frequency copy.
- **USB-C charging** on the radio and a ~2,500 mAh battery — the standout
  practical upgrade over the classic UV-5R.

## Programming

Like every Baofeng, the UV-5RM is a **CHIRP** radio first. Use a current
CHIRP-next build, pick the UV-5RM model, and load your channels there rather than
thumbing 999 memories in by hand. The one recurring gotcha is the cable:
counterfeit USB-serial chips are everywhere, so buy a genuine-FTDI programming
cable and half of the "won't connect" complaints disappear.

## GopherTrunk alternative

Here's the budget math worth knowing: GopherTrunk receives only and can't
replace even a $30 transmitter — but for the *listening* half of the hobby, a
~$30 [RTL-SDR](/reference/rtl-sdr/) running free GopherTrunk beats the UV-5RM
hollow. The Baofeng's multi-band RX is analog-FM only; the SDR hears the digital
and trunked modes ([DMR](/reference/dmr/), [C4FM](/reference/c4fm/),
[D-STAR](/reference/d-star/), even [trunked systems](/reference/trunked-radio/))
the Baofeng can't, and it records and logs every transmission instead of playing
it once. Run both for under $70: the SDR to learn what's active in your area, the
UV-5RM to talk on it once you're licensed. See
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).

## Who it's for

- **Buy a UV-5RM** as a modern, do-everything-cheap beater — go-bag spare,
  loaner, classroom radio, or a first handheld with USB-C and a color screen —
  with the front-end and licensing caveats above understood.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B0D7Q3X7XL?tag=gophertrunk-20" rel="nofollow sponsored noopener">UV-5RM on Amazon &rarr;</a>
- **Prefer the settled version?** The [BF-F8HP](/reference/baofeng-bf-f8hp/) is
  the mature, BTECH-supported take on this platform, or spend **$180** on a
  [Yaesu FT-60R](/reference/yaesu-ft-60r/) for a radio that will outlive a stack
  of Baofengs.
- **Full rankings**: [best handheld ham radios](/best-handheld-ham-radios/).

## Sources

[^baofeng]: [Baofeng UV-5RM Plus product page](https://www.baofengradio.com/products/uv-5rm-plus-8w-multi-band-radio) — Baofeng, on frequency coverage, channel count, USB-C charging, battery, screen and NOAA weather. Power output, emissions, QC and certification history summarized from independent lab findings (ARRL, mid-2010s onward) across the UV-5R family and the FCC's 2018 Enforcement Advisory on non-compliant import radios; "10W" treated as a nominal marketing figure.
