---
layout: page
title: "Best RTL-SDR for Scanning (2026)"
description: "The best RTL-SDR dongles for scanning P25, DMR, NXDN, and TETRA with GopherTrunk in 2026 — RTL-SDR Blog V4 vs V3 vs NooElec NESDR SMArt v5 vs generic, ranked by need and stock."
keywords: best RTL-SDR, best RTL-SDR dongle, RTL-SDR Blog V4, RTL-SDR Blog V3, NESDR SMArt v5, RTL-SDR scanning, RTL2832U, P25 RTL-SDR, RTL-SDR for GopherTrunk
permalink: /best-rtl-sdr/
nav_group: Hardware
affiliate: true
faq:
  - q: "What is the best RTL-SDR for scanning?"
    a: "For most people the NooElec NESDR SMArt v5 (around $35) is the best RTL-SDR to buy today — a 0.5 ppm TCXO, aluminium heatsink case, and reliable stock. The RTL-SDR Blog V4 is the enthusiast reference and slightly better on HF, but it is end-of-line, so buy it while stock lasts and treat the NESDR as the always-available pick."
  - q: "Is the RTL-SDR Blog V4 discontinued?"
    a: "Effectively yes. The Rafael Micro R828D tuner the V4 depends on is discontinued, so RTL-SDR Blog has said the V4 is the last of that line. It still works perfectly for GopherTrunk, but once current stock sells through, the V3 and NESDR-class R820T2/R860 dongles are what remain."
  - q: "Do I need the V4, or is a cheaper dongle fine?"
    a: "A cheaper R820T2/R860 dongle like the NESDR SMArt v5 decodes the exact same P25, DMR, and NXDN voice. The V4's advantages are built-in HF coverage without an upconverter and a bit better front-end filtering — nice to have, not required for VHF/UHF trunk-tracking."
  - q: "Should I avoid generic $15 RTL-SDR dongles?"
    a: "For scanning, yes. Generic dongles usually lack a TCXO, so they drift with temperature and lose the control channel, and their cheap regulators add noise. Spend the extra $15–20 on a NESDR SMArt or RTL-SDR Blog unit — it is the difference between a lock and constant retunes."
  - q: "Can an RTL-SDR decode encrypted police?"
    a: "No. No RTL-SDR — and no scanner — can decode AES-encrypted talkgroups. GopherTrunk decodes clear P25/DMR/NXDN/TETRA, never keyed encryption. Buy based on the systems in your area that are still in the clear."
  - q: "How many trunked channels can one RTL-SDR follow?"
    a: "One dongle cleanly follows one control channel and the voice grants on that site's band. For multi-site systems or control channels spread far apart, GopherTrunk can drive a pool of dongles or a wideband Airspy."
---

# Best RTL-SDR for Scanning (2026)

**The best RTL-SDR for scanning is the cheapest one with a real TCXO that cleanly
hears your local control channel** — and in 2026 that is a
[NooElec NESDR SMArt v5](/reference/nesdr/) or an
[RTL-SDR Blog V4](/reference/rtl-sdr/) for about $35–40. Both drive
[GopherTrunk](/downloads.html) — free, open-source
[software-defined radio](/reference/software-defined-radio/) — as a
P25/DMR/NXDN/TETRA trunking scanner. The dongle is the only thing you buy.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Best overall / always in stock:** [NESDR SMArt v5](/reference/nesdr/) (~$35).
**Enthusiast reference (but end-of-line):** [RTL-SDR Blog V4](/reference/rtl-sdr/) (~$40)
— buy while stock lasts. **Still great:** [RTL-SDR Blog V3](/reference/rtl-sdr/)
(~$45 with kit). **Skip:** unbranded $15 dongles with no TCXO. **You only buy the
radio** — the software is free. **No RTL-SDR decodes
[AES encryption](/police-scanner-encryption/).**
</div>

## Quick picks

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">Best overall</span>
<h3>NooElec NESDR SMArt v5</h3>
<p class="pick-card__price">around $35</p>
<p>0.5 ppm TCXO, aluminium heatsink case, R820T2/R860 tuner. The mainstream "just buy this" RTL-SDR — reliable stock and rock-steady control-channel lock.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B01HA642SW?tag=gophertrunk-20" rel="nofollow sponsored noopener">NESDR on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/nesdr/">NESDR details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Enthusiast reference</span>
<h3>RTL-SDR Blog V4</h3>
<p class="pick-card__price">around $40</p>
<p>1 ppm TCXO, built-in HF upconverter, front-end filtering, SMA, switchable bias tee. The reference dongle — but end-of-line, so buy while it is in stock.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B0CD745394?tag=gophertrunk-20" rel="nofollow sponsored noopener">V4 on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/rtl-sdr/">RTL-SDR details</a> · <a href="/rtl-sdr-blog-v3-vs-v4/">V3 vs V4</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Best starter bundle</span>
<h3>RTL-SDR Blog V3 + dipole kit</h3>
<p class="pick-card__price">around $45</p>
<p>The proven R860 dongle plus the multipurpose dipole antenna kit. Everything you need to hear a trunked system out of one box.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B0BMKB3L47?tag=gophertrunk-20" rel="nofollow sponsored noopener">V3 kit on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/rtl-sdr/">RTL-SDR details</a> · <a href="/best-sdr-antenna/">antenna guide</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Bias-tee variant</span>
<h3>NESDR SMArTee v2</h3>
<p class="pick-card__price">around $40</p>
<p>An always-on 4.5 V bias tee to power a mast-mounted <a href="/best-sdr-lna/">LNA</a> up the coax. The same solid NESDR front end otherwise.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B079C3FHPG?tag=gophertrunk-20" rel="nofollow sponsored noopener">SMArTee on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/bias-tee/">bias-tee details</a></p>
</div>
</div>

## Full comparison

| Dongle | Tuner | TCXO | HF | Notes | Approx price |
|---|---|---|---|---|---|
| [NESDR SMArt v5](/reference/nesdr/) | R820T2 / R860 | 0.5 ppm | via upconverter | Always in stock, aluminium case | ~$35 |
| [RTL-SDR Blog V4](/reference/rtl-sdr/) | R828D | 1 ppm | **built-in** | Best filtering; **end-of-line** | ~$40 |
| [RTL-SDR Blog V3](/reference/rtl-sdr/) | R860 | 1 ppm | direct-sampling Q | Proven, kit includes dipole | ~$45 |
| [NESDR SMArTee v2](/reference/nesdr/) | R820T2 | 0.5 ppm | via upconverter | Always-on bias tee for an LNA | ~$40 |
| Generic no-name | R820T2 | usually none | — | Drifts with heat — **skip for scanning** | ~$15 |

> **The TCXO is the whole point.** A trunking scanner has to sit on one control
> channel for hours. A generic dongle with no temperature-compensated oscillator
> drifts as it warms up and slides off the control channel; a $35 NESDR or
> RTL-SDR Blog unit stays locked. This one part is why "just buy the cheap one"
> is bad advice for [trunk-tracking](/reference/trunked-radio/).

## The V4 end-of-line situation

The [RTL-SDR Blog V4](/reference/rtl-sdr/) is the dongle enthusiasts reach for,
thanks to its Rafael Micro R828D tuner, built-in HF upconverter (no separate
[upconverter](/reference/upconverter/) box), and improved input filtering. But the
R828D has been discontinued, so RTL-SDR Blog has confirmed the V4 is the last of
that line. It still works flawlessly with GopherTrunk today — nothing about it
"expires" — but stock is finite.

Practically, that means: if you want the best single dongle and can find one, buy
the V4 now. If you want something you can re-order in a year, or you are buying
several for a [multi-dongle pool](/multi-dongle-sdr-setup/), standardise on the
[NESDR SMArt v5](/reference/nesdr/) — its R820T2/R860 tuner is not going anywhere.
See the full [V3 vs V4 breakdown](/rtl-sdr-blog-v3-vs-v4/) if you are torn between
the two RTL-SDR Blog models.

## How to choose

- **Just want it to work, forever in stock?** [NESDR SMArt v5](/reference/nesdr/).
  It is the boring, correct answer for most people.
- **Want the best single dongle and HF without an upconverter?**
  [RTL-SDR Blog V4](/reference/rtl-sdr/), bought while stock lasts.
- **Starting from nothing?** The [V3 + dipole kit](/best-sdr-antenna/) puts a
  proven dongle and a real antenna in one order.
- **Powering a mast-mounted [LNA](/best-sdr-lna/)?** [NESDR SMArTee v2](/reference/nesdr/)
  with its always-on [bias tee](/reference/bias-tee/).
- **Tough or congested RF?** No RTL-SDR fixes an overloaded 8-bit front end — that
  is when you step up to a 12-bit [Airspy](/reference/airspy/). See
  [Airspy vs RTL-SDR vs HackRF](/airspy-vs-rtl-sdr-vs-hackrf/).

## Don't forget the rest of the kit

The dongle is one piece. You also need a decent
**[antenna](/best-sdr-antenna/)** (it matters more than the dongle), the right
**[cables and adapters](/sdr-cables-and-connectors/)**, and a computer — any PC or
a [Raspberry Pi for 24/7](/raspberry-pi-sdr-scanner/). Our
[what-you-need checklist](/what-do-i-need-for-gophertrunk/) covers every part, and
the [hardware setup guide](/hardware.html) gets you decoding.

## Bottom line

Buy the **[NESDR SMArt v5](/reference/nesdr/)** if you want one dongle that is
always in stock, or an **[RTL-SDR Blog V4](/reference/rtl-sdr/)** while it lasts
if you want the enthusiast reference with built-in HF. Both run
[GopherTrunk](/downloads.html) beautifully for about $35–40, both decode the same
P25/DMR/NXDN voice, and **neither — nor any radio — decodes
[encryption](/police-scanner-encryption/)**. Skip the no-name $15 dongles; the
TCXO is worth every cent. Want more front end for tough RF? Compare against the
[best SDRs overall](/best-sdr-for-gophertrunk/).
