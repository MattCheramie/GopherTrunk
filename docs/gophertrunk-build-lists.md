---
layout: page
title: "GopherTrunk Build Lists — Complete Shopping Lists for Every Setup"
description: "Complete, priced hardware shopping lists for every way to run GopherTrunk — cheap PC build, always-on Raspberry Pi, rooftop base station, portable field rig, and multi-dongle wideband. Pick a build and buy the parts."
keywords: GopherTrunk build list, SDR scanner build, RTL-SDR shopping list, SDR bill of materials, police scanner build, GopherTrunk hardware kit, SDR base station build, portable SDR scanner
permalink: /gophertrunk-build-lists/
nav_group: Hardware
affiliate: true
faq:
  - q: "Which GopherTrunk build should I choose?"
    a: "Start with the PC build if you already own a computer — it is the cheapest way to hear a trunked system, about $60–80 in hardware. Move to the Raspberry Pi build for a silent always-on scanner, the outdoor base build for serious range from a rooftop antenna, the portable build for the field, or the multi-dongle build to cover several sites or channels at once."
  - q: "How much does a GopherTrunk setup cost?"
    a: "GopherTrunk itself is free and open source, so you only buy hardware. A PC build runs about $60–80, a dedicated Raspberry Pi build about $180–220, a portable field rig about $120–160, a multi-dongle pool about $180–260, and a full rooftop base station about $300–450 depending on feedline length and grounding."
  - q: "Do I need to buy all of these builds?"
    a: "No — each build is a complete, standalone shopping list for one way to run GopherTrunk. Pick the one that matches how you want to listen. Many people start with the PC build and later add an outdoor antenna or a Raspberry Pi as they get more serious."
  - q: "What is the single most important part in any build?"
    a: "The antenna, followed by where you mount it. A $35 dongle on a good outdoor antenna beats an expensive receiver on a bad indoor whip every time. Spend on the antenna and feedline before you spend on the radio."
  - q: "Will any of these builds decode encrypted police?"
    a: "No. No SDR or scanner — and no build on this page — can decode AES-encrypted talkgroups. That is a cryptographic wall, not a hardware limit. Every build hears whatever is still transmitted in the clear, which in most areas includes plenty of dispatch and nearly all fire and EMS."
---

# GopherTrunk Build Lists

**Pick how you want to listen and buy exactly the parts for it.** Each build below is a
**complete, priced shopping list** — a bill of materials with picks and running totals — for
one way to run [GopherTrunk](/downloads.html), the free, open-source
[P25](/reference/project-25/)/DMR/NXDN/TETRA trunking scanner. The software costs nothing;
these pages cover only the hardware, from a $60 desk setup to a full rooftop base station.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Five complete builds:** ① [PC build](/gophertrunk-pc-build/) (~$60–80, cheapest — use a
computer you own), ② [Raspberry Pi build](/gophertrunk-sbc-build/) (~$180–220, silent
always-on), ③ [outdoor base build](/gophertrunk-outdoor-base-build/) (~$300–450, the
flagship rooftop install), ④ [portable build](/gophertrunk-portable-build/) (~$120–160,
field/storm-chasing), ⑤ [multi-dongle build](/gophertrunk-multi-dongle-build/) (~$180–260,
wideband/multi-site). **Software is free.** **The antenna matters more than the dongle.**
**No build decodes [encryption](/police-scanner-encryption/).**
</div>

## Pick your build

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">Cheapest start</span>
<h3>PC / Laptop build</h3>
<p class="pick-card__price">~$60–80 in hardware</p>
<p>Run GopherTrunk on the Windows, Mac, or Linux computer you already own. Add a dongle, an antenna, and an adapter — the fastest, cheapest way to hear a trunked system today.</p>
<a class="btn btn--buy" href="/gophertrunk-pc-build/">See the PC build &rarr;</a>
<p class="pick-card__note"><a href="/what-do-i-need-for-gophertrunk/">what you need</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Always-on</span>
<h3>Raspberry Pi / SBC build</h3>
<p class="pick-card__price">~$180–220</p>
<p>A silent, headless single-board computer that logs every call 24/7, controlled from any browser. The set-and-forget scanner that lives next to the antenna.</p>
<a class="btn btn--buy" href="/gophertrunk-sbc-build/">See the Pi build &rarr;</a>
<p class="pick-card__note"><a href="/raspberry-pi-sdr-scanner/">Pi how-to</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Flagship</span>
<h3>Outdoor base station</h3>
<p class="pick-card__price">~$300–450</p>
<p>The serious install: a rooftop base antenna on a mast, low-loss feedline, grounding and a lightning arrestor, and a mast-mounted LNA feeding a PC or Pi indoors. Maximum range.</p>
<a class="btn btn--buy" href="/gophertrunk-outdoor-base-build/">See the base build &rarr;</a>
<p class="pick-card__note"><a href="/antenna-mast-and-mounting-guide/">mast &amp; mounting</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Field / travel</span>
<h3>Portable build</h3>
<p class="pick-card__price">~$120–160</p>
<p>A laptop or Pi, a power bank, a portable whip, and a dongle — a go-bag scanner for storm-chasing, events, and travel where there is no wall power.</p>
<a class="btn btn--buy" href="/gophertrunk-portable-build/">See the portable build &rarr;</a>
<p class="pick-card__note"><a href="/best-scanner-antenna/">antenna guide</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Wideband / multi-site</span>
<h3>Multi-dongle build</h3>
<p class="pick-card__price">~$180–260</p>
<p>A powered USB hub, several dongles, and more compute so one GopherTrunk instance covers multiple sites or control channels at once. Room to grow into a wideband pool.</p>
<a class="btn btn--buy" href="/gophertrunk-multi-dongle-build/">See the multi-dongle build &rarr;</a>
<p class="pick-card__note"><a href="/multi-dongle-sdr-setup/">multi-dongle how-to</a></p>
</div>
</div>

## Compare the builds

| Build | Best for | Hardware cost | Effort | Runs on |
|---|---|---|---|---|
| [PC / laptop](/gophertrunk-pc-build/) | Trying it, cheapest entry | **~$60–80** | Low — plug in and go | A computer you own |
| [Raspberry Pi / SBC](/gophertrunk-sbc-build/) | Silent 24/7 logging | **~$180–220** | Medium — flash + configure | Dedicated Pi 5 |
| [Outdoor base](/gophertrunk-outdoor-base-build/) | Maximum range, permanent | **~$300–450** | High — rooftop install | PC or Pi indoors |
| [Portable](/gophertrunk-portable-build/) | Field, events, storms | **~$120–160** | Low — go-bag | Laptop or Pi + battery |
| [Multi-dongle](/gophertrunk-multi-dongle-build/) | Several sites/channels | **~$180–260** | Medium–high | PC or strong SBC |

Prices are hardware only and approximate — [GopherTrunk is free](/downloads.html). Every
build shares the same four essentials (computer, [SDR dongle](/best-sdr-for-gophertrunk/),
[antenna](/best-sdr-antenna/), [adapter](/sdr-cables-and-connectors/)); the builds differ in
the *computer*, the *antenna and mounting*, and the *number of receivers*.

## How to read each build page

Every build page gives you the same three things:

1. **A bill-of-materials table** — numbered parts, why each is there, a linked pick, and an
   approximate price, with a running total. This is the shopping list.
2. **Pick cards** — the specific products we recommend for the key parts, with buy buttons.
3. **Prose and a bottom line** — how the parts fit together, what to skip, and what to add.

Start from the [what-you-need checklist](/what-do-i-need-for-gophertrunk/) if you are brand
new, then come back and pick the build that fits. Not sure which dongle or antenna to buy in
the abstract? The [best SDR for GopherTrunk](/best-sdr-for-gophertrunk/) and
[best SDR antenna](/best-sdr-antenna/) guides rank the parts on their own.

## What every build has in common

- **The software is free.** [Download GopherTrunk](/downloads.html), follow the
  [hardware setup guide](/hardware.html), enter your system's control channel from
  [RadioReference](https://www.radioreference.com/), and decode.
- **The antenna and its height matter most.** A modest [dongle](/best-rtl-sdr/) on a good,
  high [antenna](/best-scanner-antenna/) outperforms a premium receiver on a bad one.
- **Keep RF runs short; send USB or network the long way.** Coax
  [loss climbs with frequency](/sdr-cables-and-connectors/) at the 700/800 MHz where
  trunked systems live.
- **Nothing here decodes [encryption](/police-scanner-encryption/).** No SDR or scanner can
  break AES. Buy for the traffic that is still in the clear — usually plenty of dispatch and
  nearly all fire/EMS.

## Bottom line

If you already own a computer, start with the **[PC build](/gophertrunk-pc-build/)** — about
$60–80 buys the whole radio chain. Want it silent and always on? The
**[Raspberry Pi build](/gophertrunk-sbc-build/)**. Chasing maximum range from a rooftop? The
**[outdoor base build](/gophertrunk-outdoor-base-build/)** is the flagship. Heading into the
field? The **[portable build](/gophertrunk-portable-build/)**. Covering several sites at once?
The **[multi-dongle build](/gophertrunk-multi-dongle-build/)**. Pick one, buy the list, and
[GopherTrunk](/downloads.html) does the rest — free.
