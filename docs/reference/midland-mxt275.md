---
slug: midland-mxt275
title: Midland MXT275
entry_type: hardware
category: two-way-radios
description: "The Midland MXT275 is the best-selling all-controls-in-mic GMRS mobile — 15 watts, hideaway body, NOAA weather, around $150–180. Split tones only on the newer USB-C revision, and its narrowband transmit runs quiet into wideband repeaters."
keywords: Midland MXT275, MXT275 review, GMRS mobile radio, MicroMobile MXT275, 15 watt GMRS radio, gmrs license, MXT275 split tones, MXT275 vs MXT575, overlanding GMRS radio, GMRS radio install, best GMRS radio under 200
aka: [MXT275, MicroMobile MXT275]
autolink: true
affiliate: true
product:
  name: "Midland MXT275"
  brand: Midland
  category: GMRS mobile radio
  lowPrice: "150"
  highPrice: "200"
  url: https://www.amazon.com/dp/B07FN2FBML?tag=gophertrunk-20
infobox:
  - { label: Type, value: "GMRS mobile radio (all-controls-in-mic)" }
  - { label: Service, value: "GMRS — 15 channels + 8 repeater channels" }
  - { label: Power, value: "15 W" }
  - { label: Repeater, value: "Yes — split tones on USB-C revision only" }
  - { label: License, value: "GMRS — $35 FCC, no test, covers family" }
  - { label: Price, value: "around $150–180" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B07FN2FBML?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [midland-mxt575, midland-mxt115, radioddity-db20-g, wouxun-kg-xs20g, btech-gmrs-pro, rtl-sdr]
related_lessons:
  - { title: "Learn RF & SDR", url: /learn/rf-sdr/ }
related_reading:
  - { title: "Best GMRS mobile & base radios", url: /best-gmrs-mobile-radios/ }
cite_urls:
  - https://midlandusa.com/products/mxt275-micromobile-two-way-radio
faq:
  - q: "Do I need a license for the Midland MXT275?"
    a: "To transmit, yes — a GMRS license: $35 to the FCC, valid 10 years, no test required, and one license covers your immediate family. Listening requires no license."
  - q: "Does the Midland MXT275 support repeater split tones?"
    a: "Only the newer USB-C revision does. Original MXT275 revisions cannot do split tones, which locks them out of repeaters that use different encode and decode tones — if you're buying used or old stock, verify the revision first."
  - q: "Why do people say the MXT275 sounds quiet on repeaters?"
    a: "Its transmit is narrowband (±2.5 kHz deviation). Into a wideband repeater that reads as weak, quiet audio — the classic 'others could barely hear me' complaint — and tone decode on the repeater end can occasionally be erratic. It's a documented characteristic, not a defect in your unit."
  - q: "What is the range of the Midland MXT275?"
    a: "Simplex, plan on roughly 1–5 miles in real terrain — UHF is line-of-sight-limited, and 15 watts doesn't change that much. Through a GMRS repeater it can cover 20–50+ miles, which is the real reason to buy a repeater-capable mobile."
  - q: "Can I program the MXT275 with CHIRP?"
    a: "No — it isn't CHIRP-programmable, and it has no channel labels or multi-repeater presets per channel. It's a set-the-channel-and-talk radio; if you want deep programmability, look at the Radioddity DB20-G."
---
**The Midland MXT275** is the radio that proved the all-controls-in-mic idea:
a 15-watt hideaway box with every control, the display, and the speaker in the
microphone, sold in volume since about 2018.[^midland] It is the simplest
credible GMRS mobile install there is — around **$150–180** — with two honest
caveats: split tones only arrived with the newer USB-C revision, and its
narrowband transmit runs quiet into wideband repeaters.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B07FN2FBML?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The easiest real GMRS install, with known limits.** 15 W, 15 GMRS + 8
repeater channels, 142 privacy codes, NOAA Scan + Alert, tiny footprint.
**Split tones on the current USB-C revision only** — older units can't, and
that locks them out of many club repeaters. **Narrowband TX** means quiet
audio into wideband repeaters (the classic complaint). No CHIRP, no channel
labels. **GMRS license to transmit: $35, 10 years, no test, covers family.**
Rankings: [best GMRS mobile radios](/best-gmrs-mobile-radios/).
</div>

## Overview

The MXT275 has been a best-seller for the better part of a decade for one
reason: it makes a permanent vehicle radio feel like plugging in a phone
charger. The main body hides anywhere; the RJ45-corded mic is the whole user
interface. Overlanders have run them for years, and the consensus is that they
just keep working, with clear audio at highway speed. Midland later scaled the
same design up to 50 watts as the [MXT575](/reference/midland-mxt575/).

The current revision programs and charges over USB-C, and — this matters —
**only that newer revision supports split tones**. Buying used or old stock
means verifying which revision you're getting, because the original ones
cannot be upgraded into split-tone repeater service.

**License note:** transmitting on GMRS takes a license — $35 to the
[FCC](/reference/fcc/), 10 years, no exam, one license covering your immediate
family. Listening needs nothing.

## Channels, power &amp; repeater use

- **Power:** 15 W — plenty for simplex convoy work, adequate for reasonably
  close repeaters.
- **Channels:** 15 GMRS channels plus the 8 repeater channels, 142 privacy
  codes.
- **Repeaters:** [CTCSS](/reference/ctcss/)/[DCS](/reference/dcs/) tones, with
  **split tones on the USB-C revision only**. There are no channel labels and
  no way to preset multiple repeaters on one channel.
- **Weather:** NOAA Weather Scan + Alert.
- **Antenna:** the 15 W Midlands historically use a small SMA-type radio-side
  connector with included cable/adapters rather than the SO-239 of the
  MXT400/500/575 — verify the connector on the current revision before buying
  mounts and [feedline](/reference/coax-feedline/).

## The narrowband-audio complaint

The MXT275's transmit is narrowband — ±2.5 kHz deviation — and this is the
root of the most common owner complaint: **weak, quiet-sounding audio into
wideband repeaters**, and occasionally erratic tone decode on the repeater
end. Your signal is there; it just modulates half as far as the wideband
radios on the same machine expect. For family simplex use nobody notices. For
serious repeater work it's a real strike, and it's the main reason to spend up
to the [MXT575](/reference/midland-mxt575/) (wide/narrow selectable) or look
at a 20 W import like the [Radioddity DB20-G](/reference/radioddity-db20-g/)
or [Wouxun KG-XS20G Plus](/reference/wouxun-kg-xs20g/), both of which also do
split tones at or below this price. The MXT275 holds a genuine Part 95E
grant, as does everything we recommend.

## GopherTrunk alternative

FRS and GMRS live at 462/467 MHz as analog narrowband
[FM](/reference/frequency-modulation/) — easy work for a ~$30
[RTL-SDR](/reference/rtl-sdr/). Running free GopherTrunk, it monitors and
records every local FRS/GMRS channel and repeater at once, so before you buy
the MXT275 (or pay the $35 license) you can find out what's actually active
around you — including whether the local repeaters your radio would need split
tones for are worth chasing. GopherTrunk is receive-only; it complements a
transmitting radio, never replaces it.
[Download GopherTrunk](/downloads.html) and have a listen first.

## Who it's for

- **Buy the MXT275** if you want the simplest permanent GMRS install for
  family, trail, and convoy use — and buy the current USB-C revision.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B07FN2FBML?tag=gophertrunk-20" rel="nofollow sponsored noopener">View on Amazon &rarr;</a>
- **Buy the [MXT575](/reference/midland-mxt575/)** instead if repeaters are
  the point — 50 W and selectable wideband fix this radio's two limits while
  keeping the mic-centric design.
- **Skip it** for the [Radioddity DB20-G](/reference/radioddity-db20-g/) if
  you'd trade Midland polish for 20 W, wideband receive, and CHIRP at around
  $120. Full rankings:
  [best GMRS mobile radios](/best-gmrs-mobile-radios/).

## Sources

[^midland]: [Midland MXT275 MicroMobile product page](https://midlandusa.com/products/mxt275-micromobile-two-way-radio) — Midland Radio, on the integrated control mic design, 15 W output, channel/privacy-code counts, repeater channels, and NOAA weather.
