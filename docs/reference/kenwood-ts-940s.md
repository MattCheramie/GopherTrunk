---
slug: kenwood-ts-940s
title: Kenwood TS-940S
entry_type: hardware
category: ham-radios
description: "The Kenwood TS-940S is the 1985 flagship HF transceiver that still anchors classic stations — silky analog receiver, battleship build, built-in AC supply — used-market only at roughly $400–800, with well-documented aging faults to check first."
keywords: Kenwood TS-940S, TS-940S review, TS-940S for sale, vintage HF transceiver, classic ham radio, TS-940S dots display, TS-940S power supply repair, used HF radio, Kenwood flagship 1985, TS-940S buying guide
aka: [TS-940S, TS-940, TS-940S AT]
autolink: true
affiliate: true
product:
  name: "Kenwood TS-940S"
  brand: Kenwood
  category: Ham radio base station (vintage HF transceiver)
  seller: eBay
  itemCondition: used
  lowPrice: "400"
  highPrice: "800"
  url: "https://www.ebay.com/sch/i.html?_nkw=kenwood+ts-940s"
infobox:
  - { label: Type, value: "Vintage HF base transceiver (analog superhet)" }
  - { label: Bands, value: "160–10 m TX; 0.15–30 MHz RX" }
  - { label: Modes, value: "SSB, CW, AM, FM, FSK" }
  - { label: Power, value: "100 W+ (40 W AM); built-in AC supply" }
  - { label: Programming, value: "None as shipped (optional IF-10 serial kit)" }
  - { label: Price, value: "around $400–800 used" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.ebay.com/sch/i.html?_nkw=kenwood+ts-940s\" rel=\"nofollow noopener\">Find on eBay &rarr;</a>" }
see_also: [yaesu-ft-1000mp, kenwood-ts-590sg, icom-ic-718, icom-ic-7300, rtl-sdr, superheterodyne-receiver]
related_lessons:
  - { title: "Learn RF & SDR", url: /learn/rf-sdr/ }
related_reading:
  - { title: "Best ham radio base stations", url: /best-ham-radio-base-stations/ }
cite_urls:
  - https://www.rigpix.com/kenwood/ts940s.htm
  - https://www.w3afc.com/uploads/1/3/2/2/132249121/ts-940_troubleshooting_hints_and_updated_adjustment_procedures-_version_8.00_wip.pdf
faq:
  - q: "Is the Kenwood TS-940S still a good radio?"
    a: "As a classic-station centerpiece, yes — the analog receiver is still praised as silky, the audio is superb, and the build is battleship-grade. As your only radio it's a project: expect to deal with (or pay for) recapping and connector reflow on a 40-year-old flagship."
  - q: "What is the TS-940S 'dots display' problem?"
    a: "The signature TS-940 fault: the frequency display shows a row of dots instead of digits, meaning the PLL has unlocked. It traces to the PLL/VCO boards, failing board connectors, and cold solder joints; the standard fix is a resolder and VCO re-peak, and it's well documented in the W3AFC troubleshooting guide."
  - q: "How much should I pay for a TS-940S?"
    a: "Roughly $400–800 depending on condition and whether it's the AT (internal tuner) version — a TS-940S AT sold on eBay for $574 in July 2026, and serviced units with accessories ask around $800. Pay the top of the range only for a recapped, serviced example from a known tech."
  - q: "What should I check before buying a TS-940S?"
    a: "Power supply health (aging electrolytics — recap kits are a cottage industry), the dots display/PLL fault, dial drift or erratic frequency (same PLL/VCO and connector issues), and the internal tuner on AT units. Assume an unserviced unit needs a recap and connector reflow, and price accordingly."
  - q: "Do I need a license to use a TS-940S?"
    a: "To transmit, yes — an FCC amateur license (Part 97, Technician minimum; General for most HF phone). Listening on its 150 kHz–30 MHz general-coverage receiver requires no license."
---
**The Kenwood TS-940S** was Kenwood's flagship from 1985 to about 1991 and it
still earns its bench space: a triple-conversion analog
[superheterodyne](/reference/superheterodyne-receiver/) whose receiver owners
describe as *silky*, superb transmit and receive audio, a heavy cast chassis, and
a built-in AC supply — no external 13.8 V brick, just a power cord.[^rigpix]
It has been **out of production for over three decades**: the used market —
eBay, hamfests, QRZ swapmeet — is where it lives, at roughly **$400–800**.

<a class="btn btn--buy" href="https://www.ebay.com/sch/i.html?_nkw=kenwood+ts-940s" rel="nofollow noopener">Find on eBay &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The classic-Kenwood station centerpiece — buy it serviced or budget to
service it.** Silky analog receiver with VBT/Slope Tune IF shaping, 100 W+,
built-in AC supply, ~18 kg of 1985 flagship build. **Used only, ~$400–800**
(a TS-940S AT sold for $574 in July 2026). **Known faults are the whole buying
game:** aging power-supply electrolytics, the "dots display" PLL fault, dial
drift, and AT-tuner issues — all well documented, all fixable. **Part 97
license to transmit; none to listen.** Where it ranks:
[best ham radio base stations](/best-ham-radio-base-stations/).
</div>

## Overview

In 1985 the TS-940S was the radio the others were measured against, and the
things that made it flagship-grade are the things that survive: conservative
ratings (100 W+ output), Kenwood's VBT and Slope Tune analog IF shaping that
lets you carve a passband by feel, optional crystal filters, a fluorescent
display flanked by real meters, and a chassis built like a transformer vault.
The AT version adds a built-in antenna tuner. There is no computer interface as
shipped (an optional IF-10 kit added vintage serial control), no scope, and no
DSP — this is deliberate, tactile, analog operating.

Superseded by the TS-950 around 1991, it's been used-market-only ever since.
No dealer sells it new; parts are hobbyist and aftermarket (eBay capacitor kits
are literally a cottage industry). That's not a warning against buying one —
it's the frame for *how* to buy one, below.

**License note:** transmitting requires an FCC amateur license (Part 97,
Technician minimum, General for most HF privileges). Listening requires none.

## Bands, modes &amp; power

- **TX:** 160–10 m, **100 W+** (rated conservatively; 40 W AM).
- **RX:** 150 kHz–30 MHz general coverage.
- **Modes:** [SSB](/reference/single-sideband/), [CW](/reference/morse-code/),
  AM, FM, FSK.
- **Receiver:** triple-conversion analog superhet, VBT/Slope Tune IF shaping,
  optional filters.
- **Power:** built-in **AC supply** — no external 13.8 V needed. Weight ~18 kg.

## Buying used: what to check

This is the section that matters. The TS-940S's aging faults are unusually
well documented — the W3AFC troubleshooting guide is excellent — and they
cluster in four places:[^w3afc]

1. **Internal power supply / electrolytics.** Forty-year-old capacitors are the
   number-one issue; recap kits are widely sold, and some owners swap in modern
   switching supplies. Ask whether a recap has been done, by whom, and when.
2. **The "dots display" fault.** A row of dots instead of a frequency readout
   means the PLL has unlocked — the signature TS-940 ailment, traced to the
   PLL/VCO boards, failing connectors, and cold solder joints. The standard fix
   is a connector reflow/resolder and VCO re-peak.
3. **Dial drift or erratic frequency** — the same PLL/VCO and board-connector
   roots as the dots fault, same fixes.
4. **The AT tuner** on AT versions — exercise it across bands before money
   changes hands.

Pricing follows service history: a TS-940S AT sold on eBay for **$574** in July
2026; asking prices run to **~$800** for serviced units with accessories
(EU sales €650–1,000). A "recapped, reflowed, serviced by a known tech" example
is worth the top of the range; an unserviced attic unit should be priced with a
recap and reflow already subtracted. Working against you: no factory support.
Working for you: through-hole construction, superb documentation, and a
community that has fixed all of this a thousand times.

## GopherTrunk alternative

GopherTrunk receives; it can't transmit — so it's no substitute for a
TS-940S, but it's the ideal sidekick to one. A ~$30
[RTL-SDR](/reference/rtl-sdr/) running free GopherTrunk shows you what's active
locally — and **records and logs** it — before you hunt the used market, and
afterward it gives your analog classic the one thing it never had: a modern
[waterfall](/reference/waterfall-display/) view of the band beside it. Start
with [best SDR for GopherTrunk](/best-sdr-for-gophertrunk/) or, for HF
listening, [best HF SDR](/best-hf-sdr/).

## Who it's for

- **Buy the TS-940S** if you want a classic-station centerpiece with a silky
  analog receiver and you're comfortable owning (or commissioning) vintage
  service work.
  <a class="btn btn--buy" href="https://www.ebay.com/sch/i.html?_nkw=kenwood+ts-940s" rel="nofollow noopener">Find on eBay &rarr;</a>
- **Buy the [FT-1000MP](/reference/yaesu-ft-1000mp/)** instead if your classic
  leanings run to contest iron with dual receive, or the modern
  [TS-590SG](/reference/kenwood-ts-590sg/) for today's Kenwood receiver in a
  supported radio.
- **Not a project person?** A used [IC-7300](/reference/icom-ic-7300/) costs
  similar money serviced-equivalent and just works. Full rankings:
  [best ham radio base stations](/best-ham-radio-base-stations/).

## Sources

[^rigpix]: [Kenwood TS-940S — RigPix](https://www.rigpix.com/kenwood/ts940s.htm) — specifications: bands, modes, power, receiver architecture, built-in AC supply, and the AT tuner variant.
[^w3afc]: [TS-940 Troubleshooting Hints and Updated Adjustment Procedures (W3AFC, v8.00)](https://www.w3afc.com/uploads/1/3/2/2/132249121/ts-940_troubleshooting_hints_and_updated_adjustment_procedures-_version_8.00_wip.pdf) — the standard service reference for the power supply/recap work, the "dots display" PLL fault, and drift/connector repairs.
