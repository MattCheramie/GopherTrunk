---
layout: page
title: "Best DMR Scanner (Tier II & III, Color Codes)"
description: "The best DMR scanners for Tier II and Tier III trunking — Whistler TRX-1, TRX-2, Uniden SDS100 and SDS200 — plus how GopherTrunk decodes DMR free from a $30 SDR dongle, no upgrade fee."
keywords: best DMR scanner, DMR trunking scanner, DMR Tier III scanner, Whistler TRX-1, Whistler TRX-2, Uniden SDS100 DMR, DMR color code, Motorola Capacity Plus scanner
permalink: /dmr-scanner/
nav_group: Hardware
affiliate: true
faq:
  - q: "What is the best DMR scanner in 2026?"
    a: "The Whistler TRX-1 (handheld) and TRX-2 (base/mobile) are the best value because DMR decoding is included with no upgrade fee. The Uniden SDS100/SDS200 also decode DMR with the best front end, but Uniden radios sometimes require a paid DMR upgrade. For zero hardware cost, a $30 RTL-SDR plus free GopherTrunk decodes DMR too."
  - q: "Do I have to pay extra for DMR on a Uniden scanner?"
    a: "Sometimes. Several Uniden models ship P25-ready but treat DMR (and NXDN) as a paid firmware upgrade unlocked per-serial-number. Whistler's TRX-1 and TRX-2 include DMR and NXDN in the box. Check the exact model's current firmware status before buying if DMR is your priority."
  - q: "What is a DMR color code and do I need to set it?"
    a: "A DMR color code (0–15) is the network identifier that distinguishes overlapping repeaters on the same frequency, like a CTCSS tone for digital. Modern trunk-tracking scanners and GopherTrunk read it automatically from the control channel; you rarely enter it by hand except on conventional single-repeater setups."
  - q: "What is the difference between DMR Tier II and Tier III?"
    a: "Tier II is conventional two-slot TDMA on a fixed frequency pair — common for business, utilities, and some public safety. Tier III is fully trunked DMR that steers calls across channels via a control channel. A good DMR scanner should trunk-track Tier III; GopherTrunk and the Whistler TRX follow both."
  - q: "Can a DMR scanner decode encrypted DMR?"
    a: "No. Neither a Whistler TRX, a Uniden scanner, nor an SDR can decode encrypted DMR (basic privacy, enhanced/AES, or proprietary). Encryption is a cryptographic and legal wall. Buy for the DMR talkgroups still transmitted in the clear."
  - q: "Can I decode DMR for free with GopherTrunk?"
    a: "Yes. A ~$30 RTL-SDR dongle plus free GopherTrunk on a PC decodes DMR Tier II and Tier III, reads color codes and talkgroups, follows the control channel, and records every call with timestamps — no per-mode upgrade fee ever."
---

# Best DMR Scanner (Tier II & III, Color Codes)

**The best DMR scanner is the one that decodes DMR without a surprise upgrade fee and trunk-tracks your system's tier.** [DMR (Digital Mobile Radio)](/reference/dmr/) is the ETSI two-slot TDMA standard behind Motorola Capacity Plus/Connect Plus, Hytera systems, and a large share of business, utility, and some public-safety fleets — so "supports DMR" is a spec worth reading closely.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Best value (DMR included):** [Whistler TRX-1](/reference/whistler-trx-1/) (handheld) / [TRX-2](/reference/whistler-trx-2/) (base). **Best front end:** [Uniden SDS100](/reference/uniden-sds100/) / [SDS200](/reference/uniden-sds200/) — but Uniden sometimes charges a paid DMR upgrade. **Free route:** a $30 [RTL-SDR](/reference/rtl-sdr/) + [GopherTrunk](/downloads.html) decodes [DMR](/reference/dmr/) Tier II/III with **no upgrade fee**. **Nothing here decodes [encryption](/police-scanner-encryption/).**
</div>

## The DMR upgrade-fee trap

This is the single most important buying fact for DMR:

- **Whistler [TRX-1](/reference/whistler-trx-1/) / [TRX-2](/reference/whistler-trx-2/):** DMR **and** NXDN are **included** in the box — no extra unlock, no per-serial fee.
- **Uniden [SDS100](/reference/uniden-sds100/) / [SDS200](/reference/uniden-sds200/):** superb True I/Q front end and excellent DMR decoding, but several Uniden models have historically treated DMR/NXDN as a **paid firmware upgrade** tied to the serial number. Confirm the exact unit's current firmware status before you buy if DMR is the reason you're buying.
- **[GopherTrunk](/downloads.html) on a $30 [RTL-SDR](/reference/rtl-sdr/):** decodes DMR free, forever, with **no per-mode charge**.

> **Read the fine print.** A Uniden scanner listed as "digital" is not automatically DMR-ready. If DMR is your target mode, a Whistler TRX or GopherTrunk avoids the upgrade question entirely.

## Quick picks

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">Best value — DMR included</span>
<h3>Whistler TRX-1 / TRX-2</h3>
<p class="pick-card__price">around $550–$600</p>
<p>DMR and NXDN decoding included with no upgrade fee. TRX-1 is the handheld; TRX-2 is base/mobile. Object-oriented database scanning.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B01H3XYNUO?tag=gophertrunk-20" rel="nofollow sponsored noopener">TRX-1 on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/whistler-trx-1/">TRX-1 details</a> · <a href="/reference/whistler-trx-2/">TRX-2 details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Best front end</span>
<h3>Uniden SDS100 / SDS200</h3>
<p class="pick-card__price">around $650</p>
<p>True I/Q decoding is the best in class for weak or simulcast-adjacent DMR — but check whether the DMR upgrade is already unlocked.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B07DK26FDN?tag=gophertrunk-20" rel="nofollow sponsored noopener">SDS100 on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/uniden-sds100/">SDS100 details</a> · <a href="/reference/uniden-sds200/">SDS200 details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Free / lowest cost</span>
<h3>RTL-SDR + GopherTrunk</h3>
<p class="pick-card__price">around $30</p>
<p>Decodes DMR Tier II/III, reads color codes and talkgroups, records every call — no per-mode upgrade fee. Needs a PC.</p>
<a class="btn btn--buy" href="/downloads.html">Get GopherTrunk &rarr;</a>
<p class="pick-card__note"><a href="/reference/rtl-sdr/">RTL-SDR details</a> · <a href="/police-scanner-vs-sdr/">Scanner vs SDR</a></p>
</div>
</div>

## Full comparison

| Scanner | Form | DMR included? | Tier II/III | Programming | Approx. price |
|---|---|---|---|---|---|
| [Whistler TRX-1](/reference/whistler-trx-1/) | Handheld | **Yes, no fee** | II + III | DB / SW | ~$550 |
| [Whistler TRX-2](/reference/whistler-trx-2/) | Base/mobile | **Yes, no fee** | II + III | DB / SW | ~$600 |
| [Uniden SDS100](/reference/uniden-sds100/) | Handheld | Sometimes paid | II + III | ZIP / DB | ~$650 |
| [Uniden SDS200](/reference/uniden-sds200/) | Base/mobile | Sometimes paid | II + III | ZIP / DB | ~$650 |
| **SDR + [GopherTrunk](/police-scanner-vs-sdr/)** | PC + dongle | **Yes, free** | II + III | Config file | **~$30 (free SW)** |

## Color codes, tiers, and what "trunk-tracking" means

- **[Color code](/reference/color-code/) (0–15).** DMR's network identifier — it separates overlapping repeaters sharing a frequency, like a digital CTCSS tone. Trunk-tracking scanners and GopherTrunk read it from the control channel automatically; you only set it by hand on conventional single-repeater listening.
- **Tier II (conventional).** Two-slot TDMA on a fixed frequency pair. Common for business, utilities, and some agencies. Any DMR-capable radio here handles it.
- **Tier III (trunked).** Full DMR trunking that steers calls across channels via a [control channel](/reference/control-channel/). This is where trunk-tracking matters — the Whistler TRX and GopherTrunk follow it; a plain analog radio cannot.
- **Talkgroups.** DMR organizes users into [talkgroups](/reference/talkgroup/); a good scanner lets you follow, hold, or lock out each one.

## The free DMR route: GopherTrunk on an SDR

A **~$30 [RTL-SDR](/reference/rtl-sdr/)** plus free **[GopherTrunk](/downloads.html)** on a PC decodes **DMR Tier II and Tier III**, reads color codes and talkgroups, follows the control channel, and **records and timestamps every call**. There is **no per-mode upgrade fee** — the exact charge that can add cost to a Uniden. The trade-off is a computer and setup versus a scanner's turnkey portability; the full [police scanner vs SDR](/police-scanner-vs-sdr/) page lays it out.

## Bottom line

For DMR specifically, the **[Whistler TRX-1/TRX-2](/reference/whistler-trx-1/)** is the safest buy because DMR is **included with no upgrade fee**. The **[Uniden SDS100/SDS200](/reference/uniden-sds100/)** has the better front end and is the pick if you also chase weak P25 simulcast — just confirm the DMR unlock first. And if you own a PC, a **[$30 RTL-SDR + GopherTrunk](/police-scanner-vs-sdr/)** decodes DMR free and records everything. As always, **no scanner or SDR decodes encrypted DMR** — buy for the [talkgroups](/reference/talkgroup/) still in the clear, and see the full [best police scanners](/best-police-scanners/) guide.
