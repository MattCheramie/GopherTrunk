---
layout: page
title: "Best SDR for P25 Trunking (GopherTrunk)"
description: "The best SDR for P25 trunking with GopherTrunk — RTL-SDR V4 or NESDR for a single site, a wideband Airspy to channelize multiple control channels, plus P25 Phase I/II and simulcast notes."
keywords: best SDR for P25, P25 trunking SDR, P25 Phase 2 SDR, RTL-SDR P25, Airspy P25, simulcast P25, control channel, wideband SDR trunking, GopherTrunk P25
permalink: /best-sdr-for-p25-trunking/
nav_group: Hardware
affiliate: true
faq:
  - q: "What is the best SDR for P25 trunking?"
    a: "For a single P25 site, a ~$35–40 RTL-SDR Blog V4 or NooElec NESDR SMArt v5 is the best pick — one dongle tracks the control channel and follows voice grants across the site's channels. For multi-site systems or control channels spread across a band, a wideband Airspy R2 that channelizes several at once is the better tool."
  - q: "Can one RTL-SDR follow a whole P25 system?"
    a: "One RTL-SDR follows one P25 site: it locks the control channel and retunes to each voice grant. That covers most listeners. A system with multiple sites on widely separated frequencies needs either several dongles or a wideband Airspy capturing the whole span at once."
  - q: "Does GopherTrunk decode P25 Phase II?"
    a: "Yes. GopherTrunk decodes both P25 Phase I (C4FM/FDMA) and Phase II (H-DQPSK/TDMA) trunked voice. The SDR hardware requirement is the same for both — any good RTL-SDR receives Phase II; the decoding is done in software."
  - q: "Can an SDR decode P25 simulcast?"
    a: "Simulcast is difficult for every receiver because multiple synchronized transmitters create multipath distortion. A higher-dynamic-range SDR such as an Airspy gives GopherTrunk cleaner samples to work with, which helps, but no receiver fully escapes simulcast distortion — dedicated Uniden SDS scanners have the strongest simulcast front ends."
  - q: "Do I need an Airspy for P25?"
    a: "No. An 8-bit RTL-SDR decodes P25 voice perfectly well on a clean single site. Step up to an Airspy only when the band is congested, signals are weak, you fight simulcast, or you want to channelize multiple control channels from one wideband capture."
  - q: "Can an SDR decode encrypted P25?"
    a: "No. No SDR and no scanner can decode AES-encrypted P25 talkgroups. GopherTrunk decodes clear P25 (and identifies encrypted calls) but cannot break the encryption. Buy based on the talkgroups still transmitted in the clear."
---

# Best SDR for P25 Trunking (GopherTrunk)

**The best SDR for P25 trunking is the cheapest one that cleanly hears your site's
[control channel](/reference/project-25/): a ~$35–40 [RTL-SDR](/reference/rtl-sdr/) for a
single site, or a wideband [Airspy](/reference/airspy/) when you need to channelize
several control channels at once.** GopherTrunk decodes both
[P25 Phase I and Phase II](/reference/p25-phase-2/) in software, so the radio is the only
variable.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Single P25 site:** [RTL-SDR Blog V4](/reference/rtl-sdr/) or
[NESDR SMArt v5](/reference/nesdr/) (~$35–40) — one dongle tracks the control channel and
follows grants. **Multi-site / wideband:** [Airspy R2](/reference/airspy/) channelizes
several [control channels](/reference/project-25/) from one capture. **Phase I & II**
both decode on any good dongle — it is software. **Simulcast** is hard for every
receiver; more dynamic range helps. **No SDR decodes
[AES-encrypted P25](/police-scanner-encryption/).**
</div>

## Quick picks

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">Best single site</span>
<h3>RTL-SDR Blog V4</h3>
<p class="pick-card__price">around $40</p>
<p>One dongle tracks a P25 site's control channel and retunes to each voice grant. Decodes Phase I and Phase II. The default P25 receiver.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B0CD745394?tag=gophertrunk-20" rel="nofollow sponsored noopener">V4 on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/rtl-sdr/">RTL-SDR details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Best value</span>
<h3>NESDR SMArt v5</h3>
<p class="pick-card__price">around $35</p>
<p>0.5 ppm TCXO and a shielded case — rock-stable tuning for tracking a control channel all day. Always in stock.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B01HA642SW?tag=gophertrunk-20" rel="nofollow sponsored noopener">NESDR on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/nesdr/">NESDR details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Best wideband / multi-site</span>
<h3>Airspy R2</h3>
<p class="pick-card__price">around $170</p>
<p>12-bit ADC and up to ~10 MHz capture — channelize several control channels at once and pull P25 out of congested or weak conditions.</p>
<a class="btn btn--buy" href="https://www.amazon.com/s?k=Airspy+R2+SDR&tag=gophertrunk-20" rel="nofollow sponsored noopener">Airspy on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/airspy/">Airspy details</a></p>
</div>
</div>

## How P25 trunking uses your SDR

A P25 trunked system does not park voice on a fixed frequency. One or more
**[control channels](/reference/project-25/)** continuously broadcast which talkgroup is
active and on which voice frequency. GopherTrunk locks the control channel, reads those
grant messages, and follows the conversation. That is why **a single
[RTL-SDR](/reference/rtl-sdr/) can follow an entire site**: it only needs to hear one
control channel and retune to grants.

The question that decides your hardware is simply: **how many control channels do you
need to watch at once, and how far apart are they?**

## One dongle per site vs a wideband Airspy

| Scenario | Best approach | Why |
|---|---|---|
| One local P25 site | Single [RTL-SDR](/reference/rtl-sdr/) | Tracks the control channel, follows grants — done |
| Two sites, control channels close together | One wideband [Airspy R2](/reference/airspy/) | [Channelize](/multi-dongle-sdr-setup/) both from one capture |
| Several sites spread across a band | [Multi-dongle pool](/multi-dongle-sdr-setup/) | Assign a dongle per site/role |
| Congested band, weak signal, simulcast | [Airspy](/reference/airspy/) | 12-bit front end resists overload, cleaner samples |

> **Read your system on [RadioReference](https://www.radioreference.com/) first.** Note
> how many sites you can hear and where their control channels sit. If they fall inside a
> few MHz of each other, one wideband [Airspy](/reference/airspy/) can
> [channelize them all at once](/multi-dongle-sdr-setup/). If they are scattered, a
> [pool of cheap dongles](/multi-dongle-sdr-setup/) is more flexible and often cheaper.

## Phase I, Phase II, and simulcast

**[P25 Phase I](/reference/project-25/)** uses C4FM/FDMA; **[Phase II](/reference/p25-phase-2/)**
uses H-DQPSK TDMA to fit two voice paths in one channel. GopherTrunk decodes **both** —
and importantly, the SDR requirement is identical. Any good [RTL-SDR](/reference/rtl-sdr/)
receives Phase II; the difference is entirely in software decoding, not hardware.

**[Simulcast](/reference/simulcast/)** is the one place hardware really strains. When
several transmitters send the same signal in sync, their overlapping arrivals create
multipath distortion that smears the constellation. A higher-dynamic-range
[Airspy](/reference/airspy/) gives GopherTrunk cleaner input to work with, which helps,
but be honest: **no receiver fully escapes simulcast distortion**, and dedicated Uniden
SDS scanners still hold the edge on the worst simulcast fringes. See the honest
comparison in [police scanner vs SDR](/police-scanner-vs-sdr/).

## What an SDR can't do

No SDR — and no scanner — decodes **[AES-encrypted P25](/police-scanner-encryption/)**.
GopherTrunk will identify encrypted calls but cannot break them. Check
[RadioReference](https://www.radioreference.com/) to see which of your local talkgroups
are still in the clear before buying; in most areas that still includes plenty of
dispatch and nearly all fire/EMS.

## Bottom line

For one P25 site, buy the cheapest capable dongle — a
**[RTL-SDR Blog V4](/reference/rtl-sdr/)** or
**[NESDR SMArt v5](/reference/nesdr/)** — and GopherTrunk will track the control channel
and follow every grant, Phase I or [Phase II](/reference/p25-phase-2/). Step up to a
wideband **[Airspy R2](/reference/airspy/)** only when you need to
[channelize multiple control channels](/multi-dongle-sdr-setup/), fight congestion or
weak signal, or claw back some [simulcast](/reference/simulcast/) margin. See the full
lineup in [best SDR for GopherTrunk](/best-sdr-for-gophertrunk/) and the software side in
the [P25 scanner guide](/p25-scanner/).
