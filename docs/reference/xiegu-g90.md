---
slug: xiegu-g90
title: Xiegu G90
entry_type: hardware
category: ham-radios
description: "The Xiegu G90 is a 20W HF SDR transceiver with a famously wide-range built-in antenna tuner and a working waterfall for around $450 — the cheapest real path onto HF and a POTA favorite."
keywords: Xiegu G90, Xiegu G90 review, budget HF transceiver, cheapest HF radio, 20 watt HF radio, POTA radio, portable HF transceiver, G90 antenna tuner, first HF radio, best budget ham radio 2026
aka: [G90]
autolink: true
affiliate: true
product:
  name: "Xiegu G90"
  brand: Xiegu
  category: Ham radio base station (HF transceiver)
  lowPrice: "445"
  highPrice: "450"
  url: https://www.amazon.com/dp/B08X6Z6KN2?tag=gophertrunk-20
infobox:
  - { label: Type, value: "HF transceiver (SDR, detachable face)" }
  - { label: Bands, value: "160–10 m TX; 0.5–30 MHz RX" }
  - { label: Modes, value: "SSB, CW, AM (10 m FM varies by firmware)" }
  - { label: Power, value: "20 W" }
  - { label: Programming, value: "CHIRP, Icom-style CAT; no USB soundcard" }
  - { label: Price, value: "around $450" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B08X6Z6KN2?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [icom-ic-7300, yaesu-ft-891, icom-ic-718, rtl-sdr, antenna-tuner, waterfall-display]
related_lessons:
  - { title: "Learn RF & SDR", url: /learn/rf-sdr/ }
related_reading:
  - { title: "Best ham radio base stations", url: /best-ham-radio-base-stations/ }
cite_urls:
  - https://www.radioddity.com/products/xiegu-g90-hf-transceiver
faq:
  - q: "Is the Xiegu G90 a good first HF radio?"
    a: "Yes — it's the cheapest radio we'd actually recommend for getting on HF. Around $450 buys 20 W SSB/CW, a genuinely usable SDR receiver with a small waterfall, and a built-in tuner that matches almost anything. Its limits are real (20 W, tiny screen, no USB audio), but nothing else does this much for this little."
  - q: "Is 20 watts enough for SSB?"
    a: "Usually, with honest caveats. CW and FT8 at 20 W work the world; SSB in poor conditions is work, and you'll lose some pileups to 100 W stations. POTA activators run G90s daily because the wide-range tuner plus 20 W into a wire gets contacts made."
  - q: "How good is the G90's built-in antenna tuner?"
    a: "It's the radio's killer feature — famously wide-range, happily matching end-feds, whips, and compromised portable wires that would stop the narrow tuners in radios costing three times more. Expect some relay clatter while it tunes."
  - q: "Does the Xiegu G90 do FT8?"
    a: "Yes, but it needs help: there's no built-in USB soundcard, so digital modes require an external audio interface (Xiegu's CE-19 expansion or a third-party dongle) plus CAT. Budget another $30–60 and some cabling patience."
  - q: "Do I need a license to use the G90?"
    a: "To transmit, yes — an FCC amateur license (Part 97, Technician minimum; General for most HF phone). Listening requires no license, and a $30 RTL-SDR running free GopherTrunk is an even cheaper way to start listening."
---
**The Xiegu G90** is the cheapest real ticket onto HF: around $450 buys a 20 W
SSB/CW transceiver with a 24-bit SDR architecture, a small but working spectrum
[waterfall](/reference/waterfall-display/), and — the killer feature — a built-in
[antenna tuner](/reference/antenna-tuner/) with a matching range the community
summarizes as "it'll tune a wet noodle."[^rd] It gives up power, screen size,
and refinement to the Japanese rigs, and it doesn't care: nothing near its price
does half as much.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B08X6Z6KN2?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Most affordable pick** in
[best ham radio base stations](/best-ham-radio-base-stations/). **~$450** buys
160–10 m at **20 W**, a usable SDR receiver with waterfall, a detachable
faceplate, and a **wide-range internal tuner** that matches the compromise
antennas beginners and POTA operators actually have. The trades: 20 W ceiling,
1.8-inch screen, no USB soundcard (FT8 needs a dongle), strong-signal overload
vs the big rigs, importer-level support. **Part 97 license to transmit; none to
listen** — a [$30 RTL-SDR + GopherTrunk](/best-sdr-for-gophertrunk/) is the
zero-commitment way to scout first.
</div>

## Overview

Xiegu introduced the G90 in 2019 and it has been the eHam budget favorite since:
a Chinese-built HF rig, sold in the US through Radioddity, that undercuts
everything from Japan by half while delivering the two things a new HF operator
actually needs — a receiver good enough to hear real DX, and a tuner forgiving
enough to work with whatever wire they managed to hang. The detachable faceplate
and ~8 A draw make it equally at home as a desk base, a trail radio, or a truck
rig.

Honesty about the ceiling: **20 W** is a real constraint on SSB when conditions
sag — [CW](/reference/morse-code/) and [FT8](/reference/ft8/) shrug it off, but
phone operators will lose pileups to 100 W stations. The 1.8-inch screen shows a
functional ~48 kHz waterfall, not an [IC-7300](/reference/icom-ic-7300/)
panorama. The receiver is genuinely usable but more prone to strong-signal
overload and birdies than the Japanese rigs. Firmware and QC vary batch to batch
in typical Xiegu fashion, and support means the importer, not a national service
network. Every one of those trades is priced in — twice over.

**License note:** transmitting takes an FCC amateur license (Part 97 —
Technician class minimum, General for most HF voice). Listening takes none.

## Bands, modes &amp; power

- **TX:** 160–10 m ham bands, **20 W**. **RX:** 0.5–30 MHz.
- **Modes:** [SSB](/reference/single-sideband/), CW, AM. Some sellers list 10 m
  FM — treat that as firmware-dependent and verify on your unit. No digital
  voice.
- **Architecture:** 24-bit SDR (digital IF) with spectrum/waterfall display.
- **Tuner:** built-in, famously wide-range — end-feds, whips, and compromise
  portable antennas are its home turf.
- **Power:** 13.8 V DC at ~8 A max — small-battery friendly.

## What owners praise and gripe about

Praise: astonishing capability per dollar, the tuner, a receiver that hears what
it should, and a form factor that goes anywhere — it's arguably *the* POTA
radio. Gripes: the 20 W ceiling on SSB, the tiny screen, fan and tuner-relay
noise, no native USB audio, and the batch-to-batch firmware/QC variability that
comes with the price. None of the gripes are surprises; all of them are
documented in a very large, very active owner community.

The detachable faceplate deserves its own mention: with the head remoted, the
body tucks under a seat or into a pack frame, which is why G90s turn up as
truck rigs and bicycle-mobile stations as often as desk radios. Between that,
the ~8 A draw, and the tuner, it's the rare radio where the "base station"
label undersells it — it's really a go-anywhere HF station that happens to be
cheap. Watch Radioddity's coupon churn before paying sticker; the street price
moves.

## Programming &amp; software

CAT control uses an Icom-like protocol subset, so most logger/digimode software
talks to it. **CHIRP lists G90 support** for memory programming. The missing
piece is audio: there's **no built-in USB soundcard**, so FT8 and friends need
the CE-19 expansion port with an external interface or a third-party dongle.
Firmware updates come through Xiegu/Radioddity tools and are worth applying —
they've fixed real issues over the years.

## GopherTrunk alternative

GopherTrunk receives; it can't transmit — so it isn't a G90 substitute even at
a tenth the price. It *is* the cheaper first step. A ~$30
[RTL-SDR](/reference/rtl-sdr/) running free GopherTrunk lets you monitor local
repeaters and digital ham traffic, **recording and logging** everything, so you
learn what's active before licensing or buying. If your G90 budget is genuinely
the limit, spend $30 first and put the rest toward antenna wire — the antenna
matters more than the radio. Picks:
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/) and
[best HF SDR](/best-hf-sdr/).

## Who it's for

- **Buy the G90** as your first HF radio, your POTA/portable rig, or the
  cheapest way to find out whether HF grabs you — knowing you may upgrade the
  radio later and keep it as the field unit.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B08X6Z6KN2?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Spend up** to the [FT-891](/reference/yaesu-ft-891/) (~$650) for 100 W in a
  similar footprint (minus the tuner), or the
  [IC-7300](/reference/icom-ic-7300/) (~$1,040) for the full base-station
  experience.
- **Skip the [IC-718](/reference/icom-ic-718/)** at new-old-stock prices — the
  G90 outclasses it for half the money. Rankings:
  [best ham radio base stations](/best-ham-radio-base-stations/).

## Sources

[^rd]: [Xiegu G90 — Radioddity product page](https://www.radioddity.com/products/xiegu-g90-hf-transceiver) — the US importer, on the 24-bit SDR architecture, 20 W output, 0.5–30 MHz receive, built-in wide-range ATU, detachable faceplate, and CE-19 digital-mode expansion.
