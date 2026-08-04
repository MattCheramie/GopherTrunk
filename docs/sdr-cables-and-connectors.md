---
layout: page
title: "SDR Cables & Connectors Guide"
description: "A plain-English guide to SDR cables and connectors for GopherTrunk — SMA, BNC, N, F, and UHF/PL-259 explained, plus adapter kits, RG316 pigtails, and which coax to use to avoid signal loss."
keywords: SDR cables and connectors, SMA connector, BNC connector, PL-259, RG316, RG58, SDR adapter kit, SDR coax, SMA to BNC, antenna connector guide
permalink: /sdr-cables-and-connectors/
nav_group: Hardware
affiliate: true
faq:
  - q: "What connector does an SDR use?"
    a: "Most modern SDRs — RTL-SDR Blog, Airspy, HackRF — use an SMA jack. Older or cheaper dongles sometimes use the smaller MCX or the TV-style PAL/F. Many antennas, meanwhile, use BNC, UHF (PL-259/SO-239), or N. An SMA adapter kit bridges almost any mismatch for a few dollars."
  - q: "What is the difference between SMA and RP-SMA?"
    a: "They look identical but the pin and socket are reversed. Standard SMA (used by SDRs and most antennas) has a center pin on the male side; RP-SMA (common on Wi-Fi gear) has a center socket. They do not mate properly. Buy plain SMA for SDR work and check listings carefully."
  - q: "Which coax should I use for an SDR antenna?"
    a: "For short bench runs, thin RG316 pigtails are fine and flexible. For any real distance, use lower-loss cable — RG58 or RG8X — because loss climbs steeply at the UHF frequencies trunked systems use. Keep every run as short as practical."
  - q: "Do adapters and cables lose signal?"
    a: "A little. Each adapter and connector adds a small insertion loss, and cable adds loss per foot that worsens with frequency. One or two quality adapters are negligible; a stack of cheap ones on a long thin cable is not. Minimize the number of junctions and the cable length."
  - q: "What is a PL-259 / UHF connector?"
    a: "UHF connectors — the plug is a PL-259, the jack an SO-239 — are the large screw-on connectors common on scanners, CB, and ham gear. Despite the name they are used mostly at HF/VHF. An SMA-to-UHF adapter lets an SDR use an antenna or coax terminated this way."
  - q: "Do I need a whole connector kit or just one adapter?"
    a: "If you know both ends, a single adapter is enough. If you are not sure what your antenna terminates in, or you tinker with different gear, a 16-piece SMA adapter kit costs about the price of two single adapters and covers BNC, UHF, N, and F in both directions."
---

# SDR Cables & Connectors Guide

**Your SDR is almost certainly SMA; your antenna might not be — and the cable
between them can quietly throw away signal.** This is the unglamorous plumbing of
an [SDR scanner](/best-sdr-for-gophertrunk/), but getting it right is the
difference between a clean [control-channel](/reference/trunked-radio/) lock and a
mysteriously deaf setup. Here is what plugs into what, and which
[coax](/reference/coaxial-cable/) to run.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Most SDRs use [SMA](/reference/sma-connector/)**; antennas often use BNC, UHF
(PL-259), N, or F. **An adapter kit bridges any mismatch** for a few dollars.
**SMA ≠ RP-SMA** — the pins are reversed. **[Coax](/reference/coaxial-cable/) loss
climbs with frequency** — keep runs short, use RG58/RG8X for distance. **Every
junction costs a little signal** — minimize them.
</div>

## The connectors you will meet

| Connector | Where you see it | Notes |
|---|---|---|
| **[SMA](/reference/sma-connector/)** | Most SDRs, small antennas | The SDR default. Watch for RP-SMA (reversed pin). |
| **MCX** | Some older/cheaper dongles | Tiny snap-on; adapt to SMA. |
| **BNC** | Scanners, test gear, some antennas | Quarter-turn bayonet; common on whips. |
| **UHF (PL-259 / SO-239)** | Scanners, CB, ham, discones | Large screw-on; despite the name, mostly HF/VHF. |
| **N** | Outdoor / higher-end antennas | Rugged, weatherproof, good to UHF. |
| **F** | TV-style dongles, cable TV | Push/screw; low quality for RF work. |

> **SMA and RP-SMA are not interchangeable.** They look the same but the center
> conductor is reversed — RP-SMA (Wi-Fi gear) will not properly mate with the
> plain SMA on your SDR. When in doubt, buy standard SMA and read the listing.

## Adapters and pigtails to keep on hand

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">Most useful</span>
<h3>16-piece SMA adapter kit</h3>
<p class="pick-card__price">around $13</p>
<p>SMA to/from BNC, UHF, N, and F in both genders. Covers almost any SDR-to-antenna mismatch you will hit — the one thing to buy if unsure.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B07PXCC5G2?tag=gophertrunk-20" rel="nofollow sponsored noopener">Adapter kit on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/sma-connector/">SMA details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Flexible pigtails</span>
<h3>RTL-SDR Blog RG316 pigtail kit</h3>
<p class="pick-card__price">around $15</p>
<p>Short flexible RG316 pigtails with SMA and common terminations — perfect for connecting a dongle to a fixed antenna or feedline without stress on the jack.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B0132N1DM0?tag=gophertrunk-20" rel="nofollow sponsored noopener">Pigtail kit on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/coaxial-cable/">coax details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Cable assortment</span>
<h3>NooElec SMA connectivity kit</h3>
<p class="pick-card__price">around $18</p>
<p>Eight assorted SMA cables and adapters — extensions, gender changers, and jumpers to reach a window or bench-mounted antenna cleanly.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B077H87LTS?tag=gophertrunk-20" rel="nofollow sponsored noopener">Cable kit on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/sdr-cables-and-connectors/#coax-and-loss">coax loss below</a></p>
</div>
</div>

## Coax and loss {#coax-and-loss}

Cable is not free signal — it is a lossy component, and the loss gets worse as
frequency rises. That matters because trunked public-safety systems sit at UHF
(700/800 MHz), right where thin cable bleeds the most.

| Coax | Loss | Flexibility | Best for |
|---|---|---|---|
| **RG316** | High (per foot) | Very flexible, thin | Short bench pigtails, tight bends |
| **RG58** | Medium | Moderate | General jumpers, medium runs |
| **RG8X** | Lower | Stiffer, thicker | Longer outdoor feedlines |

Rules of thumb that keep signal on the wire:

- **Short is better than good.** The cheapest way to cut loss is a shorter run.
  Put the SDR near the antenna and send USB or network the long way, not RF.
- **Match the cable to the distance.** RG316 for a foot or two; RG58/RG8X for
  anything across a room or up a mast.
- **Fewer junctions.** Each adapter adds a little loss and a possible bad contact.
  One clean adapter is fine; a tower of six is not.
- **Long run? Amplify at the antenna.** If distance is unavoidable, a mast-mounted
  [LNA](/best-sdr-lna/) recovers feedline loss *before* it happens — but only if
  the signal environment is not already strong enough to overload.

> **Weatherproof outdoor connections.** Any junction exposed to weather needs
> self-amalgamating tape or a proper boot. Water in coax raises loss permanently
> and corrodes connectors — the slow death of many rooftop antennas.

## A typical clean setup

For most people: an [SDR](/best-rtl-sdr/) with an SMA jack, a short
[RG316](/reference/coaxial-cable/) pigtail or SMA jumper out to a
[dipole](/best-sdr-antenna/) or discone, and — if the antenna terminates in BNC or
UHF — one adapter from the [kit](/reference/sma-connector/). That is the whole RF
chain. Keep it short, keep it clean, and the [dongle](/best-rtl-sdr/) will hear
everything the antenna does.

## Bottom line

Assume your SDR is **[SMA](/reference/sma-connector/)** and your antenna might be
anything — so keep a **cheap adapter kit** on hand and you will never be stuck.
Use **[RG316](/reference/coaxial-cable/)** for short jumpers and thicker RG58/RG8X
for distance, keep every run as short as you can, and minimize junctions. Get the
plumbing right and a [$35 dongle](/best-rtl-sdr/) on a good
[antenna](/best-sdr-antenna/) runs [GopherTrunk](/downloads.html) flawlessly.
Next stop: the [what-you-need checklist](/what-do-i-need-for-gophertrunk/) and the
[hardware setup guide](/hardware.html).
