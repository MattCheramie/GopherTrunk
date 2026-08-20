---
slug: midland-mxt400
title: Midland MXT400
entry_type: hardware
category: two-way-radios
description: "The Midland MXT400 was the 40-watt MicroMobile GMRS flagship before the MXT500 replaced it — discontinued, narrowband-only, no NOAA, and now a used-market radio at roughly $100–180 with a known channel 15–22 transmit bug to test for."
keywords: Midland MXT400, MXT400 review, MXT400 discontinued, used GMRS mobile radio, 40 watt GMRS radio, MXT400 vs MXT500, gmrs license, MicroMobile MXT400, MXT400 programming, MXT400 split tones, cheap used GMRS radio
aka: [MXT400, MicroMobile MXT400]
autolink: true
affiliate: true
product:
  name: "Midland MXT400"
  brand: Midland
  category: GMRS mobile radio (discontinued)
  itemCondition: used
  lowPrice: "100"
  highPrice: "180"
  url: https://www.amazon.com/dp/B01N0X0LHF?tag=gophertrunk-20
infobox:
  - { label: Type, value: "GMRS mobile radio (discontinued 2022)" }
  - { label: Service, value: "GMRS — 15 channels + 8 repeater channels" }
  - { label: Power, value: "40 W (narrowband-only TX)" }
  - { label: Repeater, value: "Yes — split tones only via DBR1 cable + software" }
  - { label: License, value: "GMRS — $35 FCC, no test, covers family" }
  - { label: Price, value: "around $100–180 used (thin data)" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B01N0X0LHF?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">Check listing on Amazon &rarr;</a>" }
see_also: [midland-mxt500, btech-gmrs-50x1, midland-mxt575, midland-mxt115, radioddity-db20-g, rtl-sdr]
related_lessons:
  - { title: "Learn RF & SDR", url: /learn/rf-sdr/ }
related_reading:
  - { title: "Best GMRS mobile & base radios", url: /best-gmrs-mobile-radios/ }
cite_urls:
  - https://support.midlandusa.com/hc/en-us/articles/26844933665559-MXT400-Programming-Software-and-Instructions-Discontinued
  - https://www.buytwowayradios.com/midland-mxt400.html
faq:
  - q: "Is the Midland MXT400 discontinued?"
    a: "Yes — confirmed by Midland's own support site, which files it under Discontinued Mobile Radios. It was superseded in 2022 by the MXT500, which fixed its three headline flaws: 40 W became 50 W, narrowband-only became selectable wideband, and NOAA weather was added."
  - q: "How much is a used Midland MXT400 worth?"
    a: "Roughly $100–180 depending on condition and accessories, based on observed eBay listings — but the sold-comps data is thin, so treat that range as a guide, not gospel. Original MSRP was $249.99. A leftover third-party Amazon listing exists; check it against eBay pricing for gouging."
  - q: "Do I need a license for the Midland MXT400?"
    a: "To transmit, yes — a GMRS license: $35 to the FCC, valid 10 years, no test, covering your immediate family. Listening requires no license, used radio or not."
  - q: "What should I check before buying a used MXT400?"
    a: "Test transmit on channels 15–22 first — there's a documented firmware glitch where the radio refuses to TX on those channels without a reboot, and Midland support couldn't fix it. Also bench-test on a ~30 A supply, ask about programming history, confirm the fan runs, and ask whether the DBR1 cable is included."
  - q: "Can the MXT400 do repeater split tones?"
    a: "Not out of the box. Split CTCSS/DCS tones are only possible via the optional DBR1 programming cable and Midland's software — and that cable is no longer easy to buy, which is why used listings that include it are worth more."
---
**The Midland MXT400** was the MicroMobile flagship until 2022: 40 watts of
name-brand GMRS in a simple, tough package. It is now **discontinued —
confirmed by Midland's own support site** — and superseded by the
[MXT500](/reference/midland-mxt500/), which fixed its three defining flaws
(40 W → 50 W, narrowband-only → selectable wideband, no NOAA →
NOAA).[^midland] Today it's a used-market radio at roughly **$100–180**,
though the sold-price data is thin enough that you should treat that range as
a sketch.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B01N0X0LHF?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check listing on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**A cheap used 40-watter with eyes-open flaws.** 15 GMRS + 8 repeater
channels, Part 95E certified, Midland build. **Narrowband-only TX** (quiet
into wideband repeaters), **no NOAA**, and **split tones only via the
optional DBR1 cable + software**. A residual third-party Amazon listing
exists, but **eBay and the used market are the practical way to buy one** —
and used pricing (~$100–180) is thin-data territory. **Test TX on channels
15–22 before paying** — a documented firmware bug. **GMRS license to
transmit: $35, 10 years, no test, covers family.** Rankings:
[best GMRS mobile radios](/best-gmrs-mobile-radios/).
</div>

## Overview

The MXT400 earned its following honestly: it was the first serious-wattage
Midland GMRS mobile, built simply — a basic segment display, a
non-detachable face, an SO-239 jack for standard
[PL-259](/reference/uhf-connector-pl259/) feedline — and it kept working.
But its design choices aged into its chief complaints. Transmit is
**narrowband-only**, so it always sounded quiet into wideband repeaters
(there's no wide/narrow toggle to fix it, unlike everything Midland has made
since). There's **no weather receive at all**. And repeater flexibility is
locked behind the optional DBR1 programming cable and Midland's software —
without them, no split [CTCSS](/reference/ctcss/)/[DCS](/reference/dcs/)
tones, period. Midland retired it in 2022 precisely because those three
things had become the line's loudest complaints.

Where to buy is part of the honest story: an Amazon listing for the MXT400
still exists, but it's residual third-party stock — check it for price
gouging. **The practical market for this radio is
<a href="https://www.ebay.com/sch/i.html?_nkw=Midland+MXT400" rel="nofollow noopener">eBay</a>
and the used channels**, which is also where the $100–180 figure comes from
(against an original $249.99 MSRP).

**License note:** transmitting on GMRS requires a license — $35 to the
[FCC](/reference/fcc/), valid 10 years, no test, covering your immediate
family. Listening needs nothing, on this or any radio.

## Buying used: what to check

This is the section that matters, because the MXT400's failure modes are
specific and testable:

1. **Test transmit on channels 15–22 before money changes hands.** There's a
   documented case of an MXT400 refusing to TX on channels 15–22 without a
   reboot — a firmware glitch Midland support could not fix. Since 15–22 are
   the main repeater-adjacent channels, a unit with this bug is badly
   compromised. Key up on them, every one.
2. **Ask about programming history.** Programming-software/firmware version
   mismatches cause errors; ask the seller whether it's been programmed, and
   with what.
3. **Bench-test on a real supply.** At 40 W it draws heavy current — use a
   supply good for ~30 A continuous. Weak supplies cause resets that look
   like radio faults.
4. **Ask whether the DBR1 cable is included.** It's required for split tones
   and no longer easy to buy — a package with the cable is worth real extra
   money.
5. **Confirm the fan runs.**

Priced against the alternatives: a used
[BTECH GMRS-50X1](/reference/btech-gmrs-50x1/) in the same $80–160 band
brings 50 W, split tones, wideband receive, and CHIRP — a stronger used buy
unless Midland simplicity is specifically what you want. And a **new**
[Radioddity DB20-G](/reference/radioddity-db20-g/) at ~$120 with a warranty
undercuts most MXT400 listings while doing split tones out of the box.

## GopherTrunk alternative

FRS and GMRS are analog narrowband [FM](/reference/frequency-modulation/) at
462/467 MHz — easy pickings for a ~$30 [RTL-SDR](/reference/rtl-sdr/)
running free GopherTrunk, which monitors and records local FRS/GMRS and
repeater activity. That's doubly useful when you're eyeing a used MXT400: a
week of recordings tells you whether your local repeaters need split tones
(which this radio only does with the elusive DBR1 cable) before you bid.
GopherTrunk is receive-only — it complements a transmitting radio, never
replaces one. [Download GopherTrunk](/downloads.html) and scout the band
first.

## Who it's for

- **Buy a used MXT400** if you want cheap name-brand 40-watt simplex power,
  the seller lets you test channels 15–22, and NOAA/split tones aren't on
  your list.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B01N0X0LHF?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check listing on Amazon &rarr;</a>
- **Buy the [MXT500](/reference/midland-mxt500/)** instead if you can spend
  new-radio money — it exists specifically to fix everything wrong with this
  one.
- **Skip it** for a used [BTECH GMRS-50X1](/reference/btech-gmrs-50x1/) (more
  radio for the same used money) or a new
  [Radioddity DB20-G](/reference/radioddity-db20-g/) (warranty, split tones,
  ~$120). Full rankings:
  [best GMRS mobile radios](/best-gmrs-mobile-radios/).

## Sources

[^midland]: [MXT400 Programming Software and Instructions (Discontinued) — Midland support](https://support.midlandusa.com/hc/en-us/articles/26844933665559-MXT400-Programming-Software-and-Instructions-Discontinued) — Midland's own support site confirming discontinued status and the DBR1 cable/software requirement for split tones; dealer archive: [Midland MXT400 — Buy Two Way Radios](https://www.buytwowayradios.com/midland-mxt400.html) with the 40 W/narrowband/no-NOAA specifications.
