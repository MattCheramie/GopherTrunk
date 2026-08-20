---
slug: yaesu-ft-891
title: Yaesu FT-891
entry_type: hardware
category: ham-radios
description: "The Yaesu FT-891 packs a full 100W of HF/6m into a 4.5-pound body for around $650 — the POTA and field-ops favorite with a surprisingly strong receiver, no internal tuner, and legendary menu depth."
keywords: Yaesu FT-891, FT-891 review, best value 100 watt HF radio, compact HF transceiver, POTA radio, field HF radio, FT-891 vs G90, FT-891 vs FT-991A, cheap 100W ham radio, Yaesu HF transceiver 2026
aka: [FT-891, FT891]
autolink: true
affiliate: true
product:
  name: "Yaesu FT-891"
  brand: Yaesu
  category: Ham radio base station (HF/6m transceiver)
  lowPrice: "640"
  highPrice: "700"
  url: https://www.amazon.com/dp/B01LZHA0I4?tag=gophertrunk-20
infobox:
  - { label: Type, value: "Compact HF/6m transceiver (mobile form, base use)" }
  - { label: Bands, value: "160–6 m" }
  - { label: Modes, value: "SSB, CW, AM, FM" }
  - { label: Power, value: "100 W" }
  - { label: Programming, value: "CHIRP / RT Systems ADMS-13; USB CAT (no USB audio)" }
  - { label: Price, value: "around $640–700" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B01LZHA0I4?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [xiegu-g90, icom-ic-7300, yaesu-ft-991a, icom-ic-718, rtl-sdr, antenna-tuner]
related_lessons:
  - { title: "Learn RF & SDR", url: /learn/rf-sdr/ }
related_reading:
  - { title: "Best ham radio base stations", url: /best-ham-radio-base-stations/ }
cite_urls:
  - https://www.dxengineering.com/parts/ysu-ft-891
faq:
  - q: "Is the Yaesu FT-891 good as a base station?"
    a: "Yes — it's marketed as a mobile but hugely used as a compact base and field radio, and it earns a spot on our base-station list. You give up an internal tuner, USB audio, and screen real estate; you keep a full 100 W and a receiver that measures noticeably well for the class."
  - q: "Does the FT-891 have a built-in antenna tuner?"
    a: "No. Pair it with Yaesu's FC-50 or an LDG autotuner, or use resonant antennas. That's the biggest practical difference versus the Xiegu G90, whose wide-range internal tuner is its party trick — the FT-891 answers with five times the power."
  - q: "Can the FT-891 do FT8?"
    a: "Yes, with an external audio interface. USB provides CAT control only — there's no built-in soundcard — so digital modes need Yaesu's SCU-17 or a third-party interface. Once cabled, 100 W of FT8 works fine (most operators run less)."
  - q: "FT-891 or Xiegu G90 — which should I buy?"
    a: "FT-891 if you want 100 W and the better receiver for about $200 more, and you have (or will buy) a tuner. G90 if your antennas are compromised wires, your battery is small, or $450 is the budget. Both are POTA legends; the 891 wins on radio, the G90 on convenience per dollar."
  - q: "Do I need a license to use the FT-891?"
    a: "Transmitting requires an FCC amateur license (Part 97 — Technician minimum, General for most HF phone). Listening requires none — and a ~$30 RTL-SDR with free GopherTrunk will let you hear your local bands before you buy anything."
---
**The Yaesu FT-891** is the community's standing answer to "best value 100 W
radio, period": a full **100 W on 160–6 m** from a 4.5-pound,
mobile-form-factor body that mostly lives on desks and picnic tables, for around
$640–700.[^dxe] The receiver measures noticeably well for the class, the build
is bulletproof — and the price is paid in menus, a small screen, and the
absence of an internal tuner and USB audio.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B01LZHA0I4?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The value-per-watt king.** ~$650 buys **100 W**, a strong receiver with
effective DSP and [noise blanker](/reference/noise-blanker/), and a detachable
head — the POTA/field-ops darling that doubles as a budget base. The trades:
**no internal [tuner](/reference/antenna-tuner/)**, **no USB soundcard** (FT8
needs an interface), 100+ menus behind a small mono display, audible fan.
Still in production while the [FT-991A](/reference/yaesu-ft-991a/) bows out.
**Part 97 license to transmit; none to listen.** Rankings:
[best ham radio base stations](/best-ham-radio-base-stations/).
</div>

## Overview

Yaesu sells the FT-891 as a mobile; hams overwhelmingly use it as a compact
base and field radio, which is why it belongs on a base-station list. Since
2016 it has been the default recommendation for "most radio for the least
money": a triple-conversion
[superheterodyne](/reference/superheterodyne-receiver/) (69.45 MHz first IF,
3 kHz roofing filter) with 32-bit IF DSP whose real-world receive performance —
strong dynamic range, an effective noise blanker and DSP noise reduction —
embarrasses radios costing more. Notably, it's the survivor of Yaesu's 2026
lineup shake-up: still in production with no discontinuation indications, while
the FT-991A ends its run.

What you give up is comfort. The monochrome dot-matrix display is cramped for
base use and its spectrum sweep is basic. Settings live in **more than a hundred
menus** — "menu hell" is the accepted term — which is fine once configured and
maddening while configuring. There's no internal tuner (budget for an FC-50 or
LDG), no built-in USB soundcard, the fan is audible, and some owners report TX
pop/relay clicks. None of that stops it being the best pure radio under $1,000.

**License note:** transmitting needs an FCC amateur license (Part 97 —
Technician minimum; most HF phone comes with General). Listening needs none.

## Bands, modes &amp; power

- **TX:** 160–6 m, **100 W**, [SSB](/reference/single-sideband/),
  [CW](/reference/morse-code/), AM, FM.
- **Receiver:** triple-conversion superhet, 3 kHz roofing filter, 32-bit DSP —
  noticeably good for the class.
- **Display:** small monochrome dot-matrix with a basic spectrum sweep;
  detachable head for mobile/field mounting.
- **Tuner:** none — external FC-50/LDG or resonant antennas.
- **I/O:** USB CAT (control only — **no USB audio**; digital modes need an
  SCU-17 or third-party interface). 13.8 V DC at ~23 A: a real 25 A supply or a
  serious battery.

## What owners praise and gripe about

Praise: full power in a tiny box, the receiver, the toughness — this is the rig
that lives in backpacks and truck cabs and keeps working. It's a
[POTA](/reference/aprs/) -era phenomenon: cheap enough to risk outdoors, good
enough to win pileups from a picnic table. Gripes: the menus, the screen, the
missing tuner and USB audio, the fan. The pattern is consistent — everything
wrong with it is ergonomic, everything right with it is RF.

## Programming &amp; software

**CHIRP lists FT-891 support** for memory programming, and RT Systems' ADMS-13
covers it commercially — more useful here than on most HF rigs since field
operators lean on memories. CAT over USB drives WSJT-X, N1MM, and loggers;
remember audio needs the external interface. Firmware updates arrive via USB.

## GopherTrunk alternative

GopherTrunk is receive-only — it can't transmit, so it won't replace an FT-891.
It's the $30 reconnaissance step: an [RTL-SDR](/reference/rtl-sdr/) running free
GopherTrunk monitors your local repeaters and digital traffic and **records and
logs everything**, telling you what's worth chasing before you buy a
transceiver, a tuner, and a power supply. It also keeps listening at home while
your FT-891 is in the field. Hardware picks:
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/); HF-specific listening:
[best HF SDR](/best-hf-sdr/).

## Who it's for

- **Buy the FT-891** if you want the most transmit-and-receive capability per
  dollar in the hobby and you'll tolerate menus and buy a tuner — first HF
  radio, field rig, or budget base.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B01LZHA0I4?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [Xiegu G90](/reference/xiegu-g90/)** (~$450) instead if a
  wide-range internal tuner and lower battery draw matter more than 100 W.
- **Spend up** to the [IC-7300](/reference/icom-ic-7300/) for the touchscreen
  waterfall, internal tuner, and one-cable digital modes in a true base
  package. Rankings:
  [best ham radio base stations](/best-ham-radio-base-stations/).

## Sources

[^dxe]: [Yaesu FT-891 — DX Engineering product/spec page](https://www.dxengineering.com/parts/ysu-ft-891) — on the 160–6 m coverage, 100 W output, triple-conversion receiver with 3 kHz roofing filter and 32-bit IF DSP, detachable front panel, and current street price. (Yaesu's own site uses session-style URLs that don't cite cleanly.)
