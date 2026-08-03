---
layout: page
title: "Best SDR Antenna for Scanning"
description: "The best antennas for SDR scanning with GopherTrunk — RTL-SDR dipole kit for starters, a discone for outdoor wideband coverage, and the Nagoya NA-771 whip for portable use, plus mounting and coax."
keywords: best SDR antenna, best scanner antenna SDR, RTL-SDR antenna, discone antenna, dipole antenna kit, Nagoya NA-771, SDR antenna for scanning, wideband scanner antenna
permalink: /best-sdr-antenna/
nav_group: Hardware
affiliate: true
faq:
  - q: "What is the best antenna for SDR scanning?"
    a: "For starting out, the RTL-SDR Blog multipurpose dipole kit (around $25) is the best value — it tunes to your target band and works indoors or on a window. For the best coverage, an outdoor discone like the Tram/D3000 (around $40) hears everything from 25 MHz to 3 GHz. For portable use, the Nagoya NA-771 whip is a solid compromise."
  - q: "Does the antenna matter more than the SDR?"
    a: "Yes. A better antenna, mounted higher and outdoors, improves reception far more than upgrading the dongle. A $35 RTL-SDR on a rooftop discone will out-hear a $170 Airspy on the stock indoor whip. Spend on the antenna and its placement first."
  - q: "What is a discone antenna and why use one?"
    a: "A discone is a wideband omnidirectional antenna that receives across a huge range — roughly 25 MHz to 3 GHz — without tuning. That makes it ideal for scanning, where you want to hear many bands at once, and for feeding a wideband SDR that channelizes multiple signals."
  - q: "Can I use my SDR antenna indoors?"
    a: "You can, and the dipole kit is designed for it, but indoors is the worst place for reception — walls, wiring, and electronics all attenuate and add noise. Even moving the antenna to a window helps; getting it outdoors and up high helps most."
  - q: "Do I need a special connector for an SDR antenna?"
    a: "Most SDRs use an SMA jack, while many antennas terminate in BNC, UHF (PL-259), or N. A cheap SMA adapter kit bridges the gap. Match the connectors before you order, or grab an adapter kit so you are covered — see our cables and connectors guide."
  - q: "Does coax length hurt reception?"
    a: "Yes — every foot of coax adds loss, and thin cable like RG316 loses a lot at UHF. Use as short a run as practical, use better cable (RG58 or RG8X) for long outdoor runs, and if the run is long, a mast-mounted LNA recovers the loss."
---
# Best SDR Antenna for Scanning

**The antenna matters more than the dongle.** A cheap
[RTL-SDR](/reference/rtl-sdr/) on a well-placed outdoor antenna will out-hear an
expensive [Airspy](/reference/airspy/) on the stock whip every time. For
[GopherTrunk](/downloads.html) to lock a [trunked](/reference/trunked-radio/)
control channel and hold it, get the antenna right first — the right type, tuned
to your band, mounted as high and as far outdoors as you can manage.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Best starter / indoor:** [RTL-SDR dipole kit](/reference/dipole-antenna/) (~$25)
— tunable, works on a window. **Best coverage / outdoor:**
[discone D3000](/reference/discone-antenna/) (~$40) — 25 MHz–3 GHz, no tuning.
**Best portable:** Nagoya NA-771 whip (~$18). **Antenna placement beats dongle
price.** **Mind your [connectors](/sdr-cables-and-connectors/) and coax loss.**
</div>

## Quick picks

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">Best starter</span>
<h3>RTL-SDR Multipurpose Dipole Kit</h3>
<p class="pick-card__price">around $25</p>
<p>Adjustable dipole with a tripod, window mount, and telescoping elements you tune to your target band. The right first antenna for almost everyone.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B075445JDF?tag=gophertrunk-20" rel="nofollow sponsored noopener">Dipole kit on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/dipole-antenna/">dipole details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Best coverage</span>
<h3>Discone D3000 (25–3000 MHz)</h3>
<p class="pick-card__price">around $40</p>
<p>Wideband omnidirectional — hears everything from low VHF to L-band with no tuning. Mount it outdoors and up high for the best all-band scanning.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B0CL5ZBN94?tag=gophertrunk-20" rel="nofollow sponsored noopener">Discone on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/discone-antenna/">discone details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Best portable</span>
<h3>Nagoya NA-771 Whip (SMA-F)</h3>
<p class="pick-card__price">around $18</p>
<p>Flexible dual-band whip that screws straight onto an SMA SDR. A big step up from the stub antenna for portable or handheld setups.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B00KC4PWQQ?tag=gophertrunk-20" rel="nofollow sponsored noopener">NA-771 on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/best-scanner-antenna/">scanner antenna guide</a></p>
</div>
</div>

## Which type for which job

| Antenna | Coverage | Where | Best for | Approx price |
|---|---|---|---|---|
| [Dipole kit](/reference/dipole-antenna/) | Tunable band | Indoor / window | Starting out, single band | ~$25 |
| [Discone D3000](/reference/discone-antenna/) | 25 MHz–3 GHz | Outdoor / mast | All-band scanning, wideband SDR | ~$40 |
| Nagoya NA-771 whip | Dual-band VHF/UHF | Portable | Handheld, travel, quick setup | ~$18 |

> **Tune the dipole to your control channel.** The [dipole kit](/reference/dipole-antenna/)
> is only broadband if you leave it stubby and lossy. Set the element length to a
> quarter-wave of your [target frequency](/reference/trunked-radio/) (RadioReference
> lists it) and it becomes a genuinely resonant, low-noise antenna on that band.

## Why the antenna beats the dongle

Reception is set by three things, in order: **where the antenna is, what type it
is, and only then the receiver.** A resonant antenna outdoors and up high collects
far more signal and far less local noise than any radio can recover from a whip
sitting next to a PC. This is why upgrading from the stock stub to even a $25
dipole is the single biggest improvement most people make — and why we tell people
to fix the antenna before considering a pricier [Airspy](/airspy-vs-rtl-sdr-vs-hackrf/).

## Mounting, height, and placement

- **Get it outside and up.** Every wall between the antenna and the signal costs
  you. A discone on the roof or a mast beats the same antenna in the attic.
- **Keep it clear of metal and noise.** Mount away from gutters, HVAC units, and
  the mess of RF noise a house radiates. A few feet of separation matters.
- **Vertical for scanning.** Public-safety VHF/UHF is vertically polarised — mount
  whips and discones vertically to match.
- **Ground and protect an outdoor run.** An outdoor antenna wants a proper ground
  and, ideally, a lightning arrestor on the feedline.

## Don't let the coax undo it

A great antenna on bad cable is a compromised antenna. Every foot of coax adds
loss, and it gets worse at UHF where trunked systems live. Thin
[RG316](/reference/coaxial-cable/) is fine for a short bench pigtail but bleeds
signal over distance; for a long outdoor run use thicker RG58 or RG8X, keep the
run as short as practical, and if it must be long, put a mast-mounted
[LNA](/best-sdr-lna/) at the antenna to recover the loss before it happens. Match
your [connectors](/sdr-cables-and-connectors/) too — most SDRs are SMA while many
antennas are BNC, UHF, or N, so an adapter kit is worth having on hand.

> **Different from a handheld scanner antenna.** These pick the same physics but
> terminate in SDR-friendly connectors. If you are outfitting a portable
> [police scanner](/best-police-scanners/) instead, see the
> [best scanner antenna](/best-scanner-antenna/) sibling guide.

## Bottom line

Start with the **[RTL-SDR dipole kit](/reference/dipole-antenna/)** and actually
tune it to your control channel — it is the best $25 you can spend on reception.
When you are ready for real coverage, put a **[discone](/reference/discone-antenna/)**
outdoors and up high, and keep the **coax short and the
[connectors](/sdr-cables-and-connectors/) right**. Do that and a plain
[$35 dongle](/best-rtl-sdr/) will hear more than most people ever get from an
expensive receiver on the stock whip. Then [download GopherTrunk](/downloads.html)
and follow the [hardware setup](/hardware.html).
