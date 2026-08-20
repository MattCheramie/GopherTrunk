---
slug: yaesu-ft-991a
title: Yaesu FT-991A
entry_type: hardware
category: ham-radios
description: "The Yaesu FT-991A is the all-band shack-in-a-box: HF, 6m, 2m and 70cm with SSB/CW/AM/FM plus C4FM System Fusion in one 100W radio. Discontinued May 2026 — dealer stock remains, making it a last-call buy."
keywords: Yaesu FT-991A, FT-991A review, FT-991A discontinued, shack in a box ham radio, all band all mode transceiver, HF VHF UHF transceiver, C4FM System Fusion radio, FT-991A vs IC-7300, best all-in-one ham radio, FT-991A price 2026
aka: [FT-991A, FT991A]
autolink: true
affiliate: true
product:
  name: "Yaesu FT-991A"
  brand: Yaesu
  category: Ham radio base station (HF/VHF/UHF all-mode transceiver)
  lowPrice: "1370"
  highPrice: "1400"
  url: https://www.amazon.com/dp/B01MDU5VYH?tag=gophertrunk-20
infobox:
  - { label: Type, value: "All-band all-mode base transceiver" }
  - { label: Bands, value: "HF + 6 m + 2 m + 70 cm" }
  - { label: Modes, value: "SSB, CW, AM, FM + C4FM System Fusion" }
  - { label: Power, value: "100 W HF/6 m; 50 W VHF/UHF" }
  - { label: Programming, value: "USB CAT + audio; CHIRP / RT Systems / ADMS" }
  - { label: Price, value: "around $1,370–1,400 while stock lasts" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B01MDU5VYH?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [icom-ic-7300, yaesu-ft-891, yaesu-ftdx10, xiegu-g90, c4fm, system-fusion-ysf, rtl-sdr]
related_lessons:
  - { title: "Learn RF & SDR", url: /learn/rf-sdr/ }
related_reading:
  - { title: "Best ham radio base stations", url: /best-ham-radio-base-stations/ }
cite_urls:
  - https://www.dxengineering.com/parts/ysu-ft-991a
faq:
  - q: "Is the Yaesu FT-991A discontinued?"
    a: "Yes — its discontinuation was reported in May 2026 after a roughly nine-year run. Major dealers still sell new sell-through stock around $1,370–1,400 as of August 2026, so it's a last-call buy: nothing else currently sold new covers HF through 70 cm, all modes plus C4FM Fusion, in one box."
  - q: "Should I still buy the FT-991A in 2026?"
    a: "If you want one radio for HF DX, local FM repeaters, and C4FM digital, yes — buy while dealer stock lasts, because no in-production radio replaces it. If you only operate HF, an IC-7300 gives you a better receiver and scope for less money."
  - q: "Does the FT-991A do C4FM System Fusion and DMR?"
    a: "It does C4FM System Fusion natively, including WIRES-X. It does not do DMR or D-STAR — no radio in this class decodes every digital voice mode. For monitoring DMR alongside everything else, a ~$30 RTL-SDR running free GopherTrunk covers it."
  - q: "Can the FT-991A work satellites?"
    a: "With workarounds. It covers 2 m and 70 cm all-mode, which is the right hardware, but it has a single receiver and no full duplex — so you can't hear your own downlink while transmitting. Casual FM birds are workable; serious satellite operators want a full-duplex setup."
  - q: "Do I need a license to use the FT-991A?"
    a: "Transmitting on any ham band requires an FCC amateur license (Part 97 — Technician class minimum; Technician alone already unlocks its 2 m/70 cm sides). Listening requires no license at all."
---
**The Yaesu FT-991A** is the last true **shack-in-a-box**: HF through 70 cm — 160
through 6 meters at 100 W, 2 m and 70 cm at 50 W — with every analog mode plus
native [C4FM](/reference/c4fm/) [System Fusion](/reference/system-fusion-ysf/)
digital voice and WIRES-X, in a single compact radio with a touchscreen and a
built-in tuner.[^dxe] Yaesu **discontinued it in May 2026** after a nine-year
run; dealers still have new stock around $1,370–1,400, and no current-production
radio replaces what it does.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B01MDU5VYH?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Best features for the price** in
[best ham radio base stations](/best-ham-radio-base-stations/) — unmatched
band/mode coverage per dollar: **HF + 6 m + 2 m + 70 cm, all modes, plus C4FM
Fusion.** **Discontinued May 2026, sell-through stock remains** — a genuine
last-call buy. The trades: an average (non-SDR) receiver, a slow sweeping scope,
menu-heavy UI, no full duplex for satellites. **~$1,370–1,400 new while it
lasts.** **Part 97 license to transmit; none to listen** — scout your local
repeaters first with a [$30 RTL-SDR + GopherTrunk](/best-sdr-for-gophertrunk/).
</div>

## Overview

For nine years the FT-991A answered a question no other single radio did: "what
if I want HF DX *and* my local repeaters *and* digital voice, with one antenna
jack per side and one power cable?" It transmits from 160 m to 70 cm, does
[SSB](/reference/single-sideband/)/[CW](/reference/morse-code/)/AM/FM everywhere
that makes sense, and speaks C4FM Fusion natively — which made it the default
"only radio" for small shacks, apartments, and field setups.

Now the honest part. **Yaesu discontinued the FT-991A in May 2026** (reported by
simonthewizard on May 20). This is sell-through, not vaporware: R&L had it at
$1,369.95 and DX Engineering at $1,399.95 in August 2026. We keep it as our
"best features for the price" pick *while that stock lasts*, because its
coverage-per-dollar is still unmatched — but know you're buying a discontinued
radio, and used prices (~$900–1,100) will be the story from here.

Technically it's a triple-conversion
[superheterodyne](/reference/superheterodyne-receiver/) with a 3 kHz roofing
filter and 32-bit IF DSP — a competent conventional design, not an SDR. The
3.5-inch touchscreen's sweeping spectrum scope is genuinely useful but slow next
to the fluid waterfalls on the [IC-7300](/reference/icom-ic-7300/) and
[FTDX10](/reference/yaesu-ftdx10/), and by Sherwood-table standards the receiver
is average for the price. You're paying for breadth, not receiver figures.

## Bands, modes &amp; systems

- **HF + 6 m:** 100 W, all modes, with the built-in ATU (note its narrow ~3:1
  matching range — resonant-ish antennas only).
- **2 m + 70 cm:** 50 W, FM and all-mode — repeaters, simplex, and
  SSB/CW weak-signal VHF work.
- **[C4FM](/reference/c4fm/) System Fusion** digital voice with **WIRES-X**
  linking. No [DMR](/reference/dmr/), no [D-STAR](/reference/d-star/).
- **Satellites:** possible with workarounds, but a single receiver and no full
  duplex means you can't hear your downlink while transmitting.
- **I/O:** USB CAT + audio for one-cable digital modes; 13.8 V DC at ~23 A.

## What owners praise and gripe about

Praise: the coverage, obviously; solid transmit audio; a body compact enough to
take to the field. Gripes are consistent and worth knowing: the **menu-heavy
UI**, the **narrow internal tuner range**, **fan noise** on digital modes, and
the receiver being merely good in an era when similar money buys direct-sampling.
None of these killed its popularity, because nothing else does its job.

The discontinuation sharpens the used-market question too. Used FT-991As run
**$900–1,100**, and with production ended that supply is what the "one radio
for everything" niche will live on — so a new unit at $1,370 with a three-year
warranty is arguably the better deal than a used one at $1,000, a calculus that
flips for most radios. If Fusion doesn't matter to you and repeaters are an
afterthought, though, don't pay the coverage premium at all: the money goes
further on a dedicated HF rig.

## Programming &amp; software

The VHF/UHF side gives it real memory-programming needs, and it's covered:
**CHIRP** supports the FT-991/991A memory map, along with RT Systems and Yaesu's
ADMS software. CAT over USB drives WSJT-X and loggers, and the built-in USB audio
codec means FT8 needs no external interface.

## GopherTrunk alternative

GopherTrunk receives — it can't transmit — so it doesn't replace an FT-991A. Use
it to decide whether you need one. A ~$30 [RTL-SDR](/reference/rtl-sdr/) running
free GopherTrunk monitors your local 2 m/70 cm repeaters and digital traffic
(including modes the FT-991A *doesn't* decode, like [DMR](/reference/dmr/)) and
**records and logs every transmission** — so before spending $1,400 on
last-call stock you'll know exactly what's active in your area. See
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/) to pick the dongle.

## Who it's for

- **Buy the FT-991A** if you want one radio for everything — HF, repeaters, and
  Fusion — and buy it now, while new sell-through stock exists.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B01MDU5VYH?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [IC-7300](/reference/icom-ic-7300/)** if you're really an HF operator
  — better receiver, better scope, less money — and add a cheap FM mobile for
  repeaters later.
- **Buy the [FT-891](/reference/yaesu-ft-891/)** (~$650) if "compact 100 W HF"
  was the part you wanted and VHF/UHF wasn't. Compare them all:
  [best ham radio base stations](/best-ham-radio-base-stations/).

## Sources

[^dxe]: [Yaesu FT-991A — DX Engineering product/spec page](https://www.dxengineering.com/parts/ysu-ft-991a) — on the HF/6 m/2 m/70 cm coverage, output power, C4FM System Fusion with WIRES-X, triple-conversion receiver with 32-bit IF DSP, touchscreen scope, and internal ATU. (Yaesu's own site uses session-style URLs that don't cite cleanly; the May 2026 discontinuation was reported by simonthewizard.com.)
