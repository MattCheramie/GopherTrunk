---
layout: page
title: "Best NXDN Scanner (IDAS, NEXEDGE, RAN)"
description: "The best NXDN scanners for IDAS and NEXEDGE systems — Whistler TRX-1, TRX-2 and the Uniden SDS series — plus how GopherTrunk decodes NXDN free from a $30 SDR dongle with no upgrade fee."
keywords: best NXDN scanner, NXDN scanner, IDAS scanner, NEXEDGE scanner, Whistler TRX-1 NXDN, Uniden SDS100 NXDN, NXDN RAN, NXDN 4800 6250, digital scanner NXDN
permalink: /nxdn-scanner/
nav_group: Hardware
affiliate: true
faq:
  - q: "What is the best NXDN scanner in 2026?"
    a: "The Whistler TRX-1 (handheld) and TRX-2 (base/mobile) are the best NXDN scanners because NXDN is included with no upgrade fee. The Uniden SDS100/SDS200 also decode NXDN with a better front end, though Uniden sometimes charges a paid upgrade. For zero hardware cost, a $30 RTL-SDR plus free GopherTrunk decodes NXDN too."
  - q: "What is NXDN, and what are IDAS and NEXEDGE?"
    a: "NXDN is a narrowband digital voice and trunking protocol jointly developed by Icom and Kenwood. IDAS is Icom's brand of NXDN; NEXEDGE is Kenwood's. They are the same air interface, so a scanner that decodes NXDN handles both IDAS and NEXEDGE systems."
  - q: "What is a RAN in NXDN?"
    a: "RAN (Radio Access Number) is NXDN's network identifier, 0–63, that separates overlapping systems sharing a frequency — the NXDN equivalent of a DMR color code or a CTCSS tone. Trunk-tracking scanners and GopherTrunk read the RAN automatically; you only set it by hand on conventional single-site listening."
  - q: "Do I have to pay extra for NXDN on a Uniden scanner?"
    a: "Sometimes. Several Uniden models treat NXDN (and DMR) as a paid firmware upgrade unlocked per serial number. Whistler's TRX-1 and TRX-2 include NXDN in the box, and GopherTrunk decodes it free. Check the exact Uniden model's firmware status before buying if NXDN is your priority."
  - q: "What is the difference between NXDN 4800 and 6250?"
    a: "NXDN comes in a 6.25 kHz very-narrow mode (2400/4800 bps class) and a 12.5 kHz mode (roughly 9600 bps). Some systems mix them. A capable NXDN scanner or GopherTrunk auto-detects the variant; verify wide/narrow support if your system uses the 6.25 kHz channels."
  - q: "Can I decode NXDN for free with GopherTrunk?"
    a: "Yes. A ~$30 RTL-SDR dongle plus free GopherTrunk on a PC decodes NXDN (IDAS/NEXEDGE), reads the RAN and talkgroups, follows the control channel, and records every call with timestamps — with no per-mode upgrade fee."
---

# Best NXDN Scanner (IDAS, NEXEDGE, RAN)

**The best NXDN scanner is the one that decodes NXDN without a paid upgrade and reads the RAN so it stays locked to your system.** [NXDN](/reference/nxdn/) is the narrowband digital protocol jointly developed by Icom and Kenwood — sold as **IDAS** by Icom and **NEXEDGE** by Kenwood — and used widely by railroads, utilities, business fleets, and some public-safety agencies. All three names describe the same air interface.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Best value (NXDN included):** [Whistler TRX-1](/reference/whistler-trx-1/) (handheld) / [TRX-2](/reference/whistler-trx-2/) (base). **Best front end:** [Uniden SDS100](/reference/uniden-sds100/) / [SDS200](/reference/uniden-sds200/) — but Uniden sometimes charges a paid NXDN upgrade. **Free route:** a $30 [RTL-SDR](/reference/rtl-sdr/) + [GopherTrunk](/downloads.html) decodes [NXDN](/reference/nxdn/) with **no fee**. **Nothing here decodes [encryption](/police-scanner-encryption/).**
</div>

## IDAS, NEXEDGE, and the upgrade-fee question

- **IDAS = Icom's NXDN. NEXEDGE = Kenwood's NXDN.** Same protocol. A scanner that lists "NXDN" decodes both — you do not need a separate "IDAS scanner."
- **Whistler [TRX-1](/reference/whistler-trx-1/) / [TRX-2](/reference/whistler-trx-2/):** NXDN **and** DMR are **included** with no upgrade fee.
- **Uniden [SDS100](/reference/uniden-sds100/) / [SDS200](/reference/uniden-sds200/):** best-in-class True I/Q front end, but Uniden has historically treated NXDN/DMR as a **paid firmware unlock** per serial number. Confirm before buying if NXDN is why you're buying.
- **[GopherTrunk](/downloads.html) on a $30 [RTL-SDR](/reference/rtl-sdr/):** decodes NXDN free, forever, no per-mode charge.

> **NXDN is narrowband — front end matters.** NXDN's 6.25 kHz channels are easy to miss with a sloppy receiver. The Uniden True I/Q design and GopherTrunk's software DSP both handle narrow NXDN cleanly; check wide/narrow support on any bargain radio.

## Quick picks

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">Best value — NXDN included</span>
<h3>Whistler TRX-1 / TRX-2</h3>
<p class="pick-card__price">around $550–$600</p>
<p>NXDN and DMR decoding included with no upgrade fee. TRX-1 is the handheld; TRX-2 is base/mobile. Reads RAN and follows trunked NXDN.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B01H3XYNUO?tag=gophertrunk-20" rel="nofollow sponsored noopener">TRX-1 on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/whistler-trx-1/">TRX-1 details</a> · <a href="/reference/whistler-trx-2/">TRX-2 details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Best front end</span>
<h3>Uniden SDS100 / SDS200</h3>
<p class="pick-card__price">around $650</p>
<p>True I/Q decoding is the best in class for weak, narrowband NXDN — but check whether the NXDN upgrade is already unlocked on the unit.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B07DK26FDN?tag=gophertrunk-20" rel="nofollow sponsored noopener">SDS100 on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/uniden-sds100/">SDS100 details</a> · <a href="/reference/uniden-sds200/">SDS200 details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Free / lowest cost</span>
<h3>RTL-SDR + GopherTrunk</h3>
<p class="pick-card__price">around $30</p>
<p>Decodes NXDN (IDAS/NEXEDGE), reads the RAN and talkgroups, records every call — no per-mode upgrade fee. Needs a PC.</p>
<a class="btn btn--buy" href="/downloads.html">Get GopherTrunk &rarr;</a>
<p class="pick-card__note"><a href="/reference/rtl-sdr/">RTL-SDR details</a> · <a href="/police-scanner-vs-sdr/">Scanner vs SDR</a></p>
</div>
</div>

## Full comparison

| Scanner | Form | NXDN included? | Trunked NXDN | Programming | Approx. price |
|---|---|---|---|---|---|
| [Whistler TRX-1](/reference/whistler-trx-1/) | Handheld | **Yes, no fee** | Yes | DB / SW | ~$550 |
| [Whistler TRX-2](/reference/whistler-trx-2/) | Base/mobile | **Yes, no fee** | Yes | DB / SW | ~$600 |
| [Uniden SDS100](/reference/uniden-sds100/) | Handheld | Sometimes paid | Yes | ZIP / DB | ~$650 |
| [Uniden SDS200](/reference/uniden-sds200/) | Base/mobile | Sometimes paid | Yes | ZIP / DB | ~$650 |
| **SDR + [GopherTrunk](/police-scanner-vs-sdr/)** | PC + dongle | **Yes, free** | Yes | Config file | **~$30 (free SW)** |

## RAN, talkgroups, and trunk-tracking

- **[RAN (Radio Access Number)](/reference/ran-nxdn/), 0–63.** NXDN's network ID — it separates overlapping systems on a shared frequency, the NXDN counterpart to a [DMR color code](/reference/color-code/). Trunk-tracking scanners and GopherTrunk read the RAN from the control channel automatically.
- **Talkgroups.** NXDN trunked systems group users into [talkgroups](/reference/talkgroup/); a good scanner lets you follow, hold, or lock each one.
- **Trunk-tracking.** NXDN Type-C/Type-D trunking steers calls via a [control channel](/reference/control-channel/). The Whistler TRX and GopherTrunk follow it; analog-only radios cannot.

## The free NXDN route: GopherTrunk on an SDR

A **~$30 [RTL-SDR](/reference/rtl-sdr/)** plus free **[GopherTrunk](/downloads.html)** decodes **NXDN (IDAS/NEXEDGE)**, reads the RAN and talkgroups, follows the control channel, and **records and timestamps every call** — with **no per-mode upgrade fee**. The trade-off is a PC and setup versus a scanner's portability; see [police scanner vs SDR](/police-scanner-vs-sdr/).

## Bottom line

For NXDN, the **[Whistler TRX-1/TRX-2](/reference/whistler-trx-1/)** is the cleanest buy — NXDN (and DMR) **included, no upgrade fee**. The **[Uniden SDS100/SDS200](/reference/uniden-sds100/)** has the better front end and is the pick if you also want top-tier P25 simulcast, provided the NXDN unlock is confirmed. And with a PC, a **[$30 RTL-SDR + GopherTrunk](/police-scanner-vs-sdr/)** decodes NXDN free and records everything. **No scanner or SDR decodes encrypted NXDN** — buy for the [talkgroups](/reference/talkgroup/) still in the clear. See the full [best police scanners](/best-police-scanners/) guide for cross-mode picks.
