---
slug: yaesu-ftdx10
title: Yaesu FTDX10
entry_type: hardware
category: ham-radios
description: "The Yaesu FTDX10 is a 100W HF/6m hybrid-SDR base transceiver with near-flagship close-in receiver dynamic range — essentially FTDX101 receive performance for half the money, around $1,700 (less after rebate)."
keywords: Yaesu FTDX10, FTDX10 review, FTDX-10, best HF receiver, hybrid SDR transceiver, Sherwood receiver rankings, FTDX10 vs IC-7300, HF base station, 100 watt HF transceiver, Yaesu HF radio 2026
aka: [FTDX10, FTDX-10]
autolink: true
affiliate: true
product:
  name: "Yaesu FTDX10"
  brand: Yaesu
  category: Ham radio base station (HF/6m transceiver)
  lowPrice: "1400"
  highPrice: "1700"
  url: https://www.amazon.com/dp/B09WG52PP6?tag=gophertrunk-20
infobox:
  - { label: Type, value: "HF/6m base transceiver (hybrid SDR)" }
  - { label: Bands, value: "160–6 m" }
  - { label: Modes, value: "SSB, CW, AM, FM" }
  - { label: Power, value: "100 W" }
  - { label: Programming, value: "USB CAT + audio; touchscreen menus" }
  - { label: Price, value: "around $1,700 (~$1,400 after rebate)" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B09WG52PP6?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [icom-ic-7300, icom-ic-7610, kenwood-ts-590sg, yaesu-ft-891, rtl-sdr, superheterodyne-receiver]
related_lessons:
  - { title: "Learn RF & SDR", url: /learn/rf-sdr/ }
related_reading:
  - { title: "Best ham radio base stations", url: /best-ham-radio-base-stations/ }
cite_urls:
  - https://www.dxengineering.com/parts/ysu-ftdx-10
faq:
  - q: "Is the Yaesu FTDX10 receiver really that good?"
    a: "Yes — its close-in dynamic range figures rank among the best ever measured in its price class (top-5 on the Sherwood table), using the same hybrid narrow-band-SDR plus direct-sampling architecture as the flagship FTDX101 for roughly half the money. If receiver performance is your first criterion, this is the sub-$2,000 radio to beat."
  - q: "FTDX10 or IC-7300 — which should I buy?"
    a: "The FTDX10 has the measurably better receiver and superior CW facilities; the IC-7300 has the slicker touchscreen, easier menus, a bigger community, and costs several hundred dollars less. Contesters and CW operators lean FTDX10; almost everyone else is happier with the 7300."
  - q: "Is the FTDX10 discontinued?"
    a: "No. A groups.io thread speculated about a fire sale when Yaesu ran a $300 rebate through August 2026, but there is no discontinuation announcement — it remains a current-production radio. The rebate just makes a very good receiver cheaper."
  - q: "Does the FTDX10 have a built-in antenna tuner?"
    a: "Yes — a high-speed internal ATU with 100 tuning memories, plus a 5-inch color touchscreen with Yaesu's 3DSS spectrum display and selectable roofing filters (500 Hz and 3 kHz standard, 300 Hz optional)."
  - q: "Do I need a ham license for the FTDX10?"
    a: "To transmit, yes — FCC Part 97, Technician minimum (General for most HF). Just listening needs no license; a ~$30 RTL-SDR with free GopherTrunk is the cheap way to scout the bands first."
---
**The Yaesu FTDX10** buys you a genuinely flagship-class receiver for mid-range
money: a **hybrid SDR** front end (narrow-band SDR plus direct sampling, the same
architecture as the FTDX101) whose close-in dynamic range ranks among the best
ever measured near its price — essentially FTDX101 receive for half the
cost.[^dxe] It runs 100 W on 160–6 m with a 5-inch touchscreen, Yaesu's 3DSS
spectrum display, and a fast built-in tuner, for around $1,700 (about $1,400
after the current rebate).

<a class="btn btn--buy" href="https://www.amazon.com/dp/B09WG52PP6?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The receiver-performance pick** among sub-$2,000 base stations — top-tier
close-in dynamic range, excellent CW facilities, roofing filters at a real 9 MHz
IF. **100 W, 160–6 m, built-in ATU, touchscreen.** The trade: Yaesu's menus and
ergonomics are less loved than the [IC-7300](/reference/icom-ic-7300/)'s, and it
costs more. **~$1,700 (~$1,400 after rebate), in production** — the "fire sale"
rumor is just a rebate. **License to transmit (Part 97, Technician+); none to
listen** — see where it ranks in
[best ham radio base stations](/best-ham-radio-base-stations/).
</div>

## Overview

Introduced in late 2020, the FTDX10 is Yaesu's answer to "I want the FTDX101's
receiver without the FTDX101's price." Instead of pure direct sampling like the
[IC-7300](/reference/icom-ic-7300/), it pairs a direct-sampling path with a
narrow-band SDR: down-conversion to a 9 MHz IF through real roofing filters
(500 Hz and 3 kHz standard, 300 Hz optional) behind 15 band-pass filters. On a
crowded contest weekend with a kilowatt neighbor 2 kHz up, that architecture is
exactly what keeps the band listenable — and it's why the FTDX10 sits in the top
five of the Sherwood close-in dynamic-range table, ahead of radios costing far
more.

The honest counterweight: Yaesu's user interface. Settings live deeper in menus
than Icom buries them, the touchscreen is less slick than the 7300's, early
production runs drew main-dial torque and encoder complaints, and a series of
firmware rounds were needed to settle AGC/popping quirks. None of it affects what
comes out of the speaker — but if you value fluency over figures, the Icom feels
nicer to drive.

Worth stating plainly: **you need an FCC amateur license (Part 97, Technician
class minimum) to transmit** with it. **Listening is license-free** — and cheaper
ways to listen exist (below).

## Bands, modes &amp; power

- **TX:** 160–6 m, **100 W**, [SSB](/reference/single-sideband/),
  [CW](/reference/morse-code/), AM, FM.
- **Receiver:** hybrid SDR — narrow-band SDR with 9 MHz IF roofing filters plus a
  direct-sampling path; 15 band-pass filters. Single receiver, no dedicated RX
  antenna port.
- **Display:** 5-inch color touchscreen with the 3DSS three-dimensional spectrum
  display and [waterfall](/reference/waterfall-display/).
- **Tuner:** built-in high-speed ATU with 100 channel memories.
- **Digital modes:** USB CAT + audio make it FT8-friendly out of the box with
  WSJT-X and N1MM.

## Receiver &amp; operating

CW operators get first-class treatment — full break-in, memory keyer, zero-in
tuning, and the narrow roofing filters doing real selection before the DSP ever
sees the signal. That combination of measured dynamic range and CW facilities is
why this radio shows up at serious contest stations as the "second radio" that
outperforms the main one.

Weaknesses beyond the UI: front-panel jack placement annoys some operators, and
like the IC-7300 there's no separate receive-antenna input for a Beverage or
loop — a real omission at this level, and one the
[TS-590SG](/reference/kenwood-ts-590sg/) actually covers.

A word on the "fire sale" chatter: Yaesu's $300 rebate (running through August
31, 2026) spawned a groups.io thread speculating the FTDX10 is being cleared
out. We found **no discontinuation announcement** — Yaesu runs rolling rebates
the way Icom does, and this one simply makes an exceptional receiver cheaper.
Treat the rumor as a rumor; treat the ~$1,400 net price as the real story. If
Yaesu ever does replace it, the resale floor on a radio with these measured
numbers will stay firm — receivers this good don't depreciate like features do.

## Programming &amp; software

Yaesu CAT over USB drives WSJT-X, N1MM, Win4Yaesu and the rest of the standard
stack. CHIRP is not a use case here — there's no confirmed CHIRP support and no
real need for it on a pure HF radio; memory management happens on the radio or
through CAT software. Firmware updates (which have meaningfully improved this
radio since launch) load over USB.

## GopherTrunk alternative

GopherTrunk is receive-only — it will never key up, so it cannot replace the
FTDX10. What it can do is make sure you buy the *right* radio. A ~$30
[RTL-SDR](/reference/rtl-sdr/) running free GopherTrunk monitors your local
repeaters and digital ham traffic and **records and logs everything**, so before
spending $1,700 you know whether your area's activity justifies it — and
unlicensed listeners get the full experience minus the PTT. Start with
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/); if weak-signal HF
listening is the draw, see [best HF SDR](/best-hf-sdr/).

## Who it's for

- **Buy the FTDX10** if receiver performance and CW are your priorities and
  you'll trade some menu friction for measurably better close-in dynamic range
  than anything near the price.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B09WG52PP6?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [IC-7300](/reference/icom-ic-7300/)** instead for a friendlier radio
  at a lower price, or the [IC-7610](/reference/icom-ic-7610/) if you need true
  dual receivers for split pileups.
- **Buy the [TS-590SG](/reference/kenwood-ts-590sg/)** if you want elite CW/RX
  performance and don't care about a built-in scope. Rankings and rubric:
  [best ham radio base stations](/best-ham-radio-base-stations/).

## Sources

[^dxe]: [Yaesu FTDX10 — DX Engineering product/spec page](https://www.dxengineering.com/parts/ysu-ftdx-10) — on the hybrid SDR architecture, 9 MHz IF roofing filters, 3DSS touchscreen display, built-in high-speed ATU, output power, and current price/rebate. (Yaesu's own site uses session-style URLs that don't cite cleanly.)
