---
slug: baofeng-uv-5r
title: Baofeng UV-5R
entry_type: hardware
category: ham-radios
description: "The Baofeng UV-5R is the $25 dual-band handheld that made ham radio accessible to millions — an honest review covering the spurious-emissions lab findings, the QC lottery, the FCC certification mess, and the GT-5R compliant variant."
keywords: Baofeng UV-5R, UV-5R review, cheapest ham radio, is the UV-5R legal, Baofeng FCC certification, UV-5R spurious emissions, GT-5R, budget ham handheld, CHIRP UV-5R, Baofeng problems
aka: [UV-5R, UV5R]
autolink: true
affiliate: true
product:
  name: "Baofeng UV-5R"
  brand: Baofeng
  category: Ham handheld transceiver
  lowPrice: "20"
  highPrice: "30"
  url: https://www.amazon.com/dp/B007UYKG4E?tag=gophertrunk-20
infobox:
  - { label: Type, value: Dual-band handheld transceiver }
  - { label: Bands, value: "TX 136–174 / 400–480(520) MHz" }
  - { label: Modes, value: Analog FM only }
  - { label: Power, value: "4–5 W nominal (variants claim 8 W)" }
  - { label: Programming, value: "CHIRP (its flagship radio)" }
  - { label: Price, value: around $25 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B007UYKG4E?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [baofeng-bf-f8hp, btech-uv-pro, yaesu-ft-60r, kenwood-th-f6a, rtl-sdr, fcc]
related_lessons:
  - { title: "Analog vs. digital voice", url: /learn/rf-sdr/digital-voice/ }
  - { title: "Legal & ethical monitoring", url: /learn/rf-sdr/legal-ethical/ }
related_reading:
  - { title: "Police scanner vs GopherTrunk", url: /police-scanner-vs-sdr/ }
cite_urls:
  - https://baofengtech.com/product/uv-5r/
faq:
  - q: "Is the Baofeng UV-5R legal to use?"
    a: "Owning and listening: yes, no license needed. Transmitting on ham bands: legal with an FCC amateur license (Technician minimum). The murky part is the hardware itself — the original Part 90 certification didn't cover typical amateur/consumer use, many clones ship with no valid grant, and the FCC's 2018 enforcement advisory targeted exactly this class of import radio. Baofeng's GT-5R is the explicitly FCC-compliant variant."
  - q: "Why do people say the UV-5R is 'dirty'?"
    a: "Independent lab tests from the mid-2010s onward (ARRL among others) found many UV-5R-family units exceeding FCC spurious-emission limits — transmitting measurable harmonics outside their intended frequency, especially on VHF at high power. Quality varies unit to unit; some are clean, some aren't, and you can't tell from the outside."
  - q: "What is the difference between the UV-5R and the GT-5R?"
    a: "The GT-5R is Baofeng's own answer to the criticism: an explicitly FCC-compliant version of the UV-5R with cleaned-up emissions and TX locked to the amateur bands. Its existence is Baofeng conceding the point. If you want a UV-5R in 2026, the GT-5R is the defensible one to buy."
  - q: "Is the UV-5R a good first ham radio?"
    a: "It's a good first $25. As a beater, go-bag spare or classroom radio it has real value, and CHIRP makes it usable. As your only radio it will frustrate you — deaf receiver near strong signals, QC lottery, hostile menus. A Yaesu FT-60R is the better first real radio if the budget reaches $180."
  - q: "Which UV-5R variant should I buy?"
    a: "They're legion — UV-5R v2+, UV-5RA, UV-5RE, UV-5R+ are cosmetic shells of the same radio, while '8W', 'Plus' and UV-5RH-style models share the name but not the internals. For a known quantity, buy the classic kit or the GT-5R, and assume any '8 W' claim is optimistic."
---
**The Baofeng UV-5R** is the most-cloned handheld on Earth and, at around
**$25**, the radio that made ham radio accessible to millions.[^baofeng] It is
also a radio whose family has repeatedly **failed independent spurious-emission
lab tests**, whose quality control is a lottery, and whose FCC certification
story is messy enough that Baofeng itself sells a cleaned-up variant. Both
halves of that sentence are true, and an honest review has to hold them
together.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B007UYKG4E?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**A $25 beater with caveats — buy it as exactly that.** Dual-band analog FM,
CHIRP's flagship radio, an enormous accessory ecosystem. The caveats are real:
**many units exceed FCC spurious-emission limits** (ARRL and other lab
findings), the **QC lottery** ("buy two" is a running joke), and a
**certification mess** the FCC's 2018 enforcement advisory targeted. The
**GT-5R** is Baofeng's FCC-compliant fix — buy that variant. Transmitting
requires an FCC amateur license (Technician minimum); listening requires none.
</div>

## Overview

Since roughly 2012 the UV-5R has been the answer to "what's the cheapest way to
transmit?" — 136–174 and 400–480 (or 520) MHz, 4–5 W nominal, 128 channels, an
1,800 mAh battery, a flashlight, and a street price that dips near $20 on
sale. It spawned a giant ecosystem of batteries, antennas and cases, and made
CHIRP a household name. A radio you can afford to lose, lend or leave in a kit
has genuine value, and pretending otherwise is snobbery.

The problems are equally documented. **Emissions**: independent lab tests
(ARRL and others, mid-2010s onward) found many UV-5R-family units exceeding
FCC spurious-emission limits, especially on VHF at high power — and it varies
unit to unit, so a clean review sample proves nothing about yours.
**QC**: frequency drift, deaf receivers and failed finals are common enough
that "buy two" is the standing advice. **Certification**: the original Part 90
grant never covered how most consumers actually use the radio, countless clones
ship with no valid grant at all, and the [FCC](/reference/fcc/)'s 2018
Enforcement Advisory took aim at precisely this class of import. Baofeng's
answer was the **GT-5R** — an explicitly FCC-compliant UV-5R with cleaned-up
emissions and TX locked to the amateur bands
(<a href="https://www.amazon.com/dp/B08VNM1CX4?tag=gophertrunk-20" rel="nofollow sponsored noopener">GT-5R on Amazon</a>) —
and its existence is Baofeng conceding every point above. **Receiver**: the
wide-open front end overloads next to any strong transmitter.

## Modes &amp; features

- **Analog FM only**; no digital, GPS, Bluetooth or IP rating.
- **4–5 W nominal** (variant "8 W" claims are marketing; the internals vary).
- **128 channels**, [CTCSS](/reference/ctcss/)/[DCS](/reference/dcs/), FM
  broadcast receive, flashlight.
- **Variant chaos**: UV-5RA/RE/R+ are cosmetic shells; "Plus"/UV-5RH-style
  successors share the name, not the internals.

## Programming

The UV-5R is arguably **CHIRP's flagship radio** — programming it any other way
is masochism. The one gotcha is the USB cable: counterfeit chips are rampant,
so buy a genuine-FTDI cable.

## GopherTrunk alternative

Here's the budget math worth knowing: GopherTrunk receives only and can't
replace even a $25 transmitter — but for the *listening* half of the hobby, a
~$30 [RTL-SDR](/reference/rtl-sdr/) running free GopherTrunk beats the UV-5R
hollow. It hears the digital modes ([DMR](/reference/dmr/),
[C4FM](/reference/c4fm/), [D-STAR](/reference/d-star/), even
[trunked systems](/reference/trunked-radio/)) the Baofeng can't, and it records
and logs every transmission. Run both for under $60: the SDR to learn what's
active, the UV-5R to talk on it once you're licensed. See
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).

## Who it's for

- **Buy a UV-5R** — ideally the
  <a href="https://www.amazon.com/dp/B08VNM1CX4?tag=gophertrunk-20" rel="nofollow sponsored noopener">FCC-compliant GT-5R variant</a> —
  as a beater, go-bag spare, loaner or classroom radio, with the caveats above
  understood.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B007UYKG4E?tag=gophertrunk-20" rel="nofollow sponsored noopener">UV-5R on Amazon &rarr;</a>
- **Spend $60** on the [BF-F8HP](/reference/baofeng-bf-f8hp/) for the sorted
  version of this platform, or **$180** on a
  [Yaesu FT-60R](/reference/yaesu-ft-60r/) for a radio that will outlive them
  both.
- **Full rankings**: [best handheld ham radios](/best-handheld-ham-radios/).

## Sources

[^baofeng]: [Baofeng UV-5R product page](https://baofengtech.com/product/uv-5r/) — BTECH/Baofeng, on frequency coverage, output power, battery, and kit contents. Emissions, QC and certification history summarized from independent lab findings (ARRL, mid-2010s onward) and the FCC's 2018 Enforcement Advisory on non-compliant import radios.
