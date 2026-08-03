---
layout: page
title: "Cheap Police Scanners (Best Under $150 + Free Options)"
description: "The best cheap police scanners under $150 — Uniden BC75XLT, BC125AT and SR30C — why cheap means analog, and the truly cheapest digital route: a $30 RTL-SDR plus free GopherTrunk software."
keywords: cheap police scanner, best cheap scanner, police scanner under 150, budget police scanner, Uniden BC125AT, Uniden BC75XLT, Uniden SR30C, cheapest digital scanner, RTL-SDR scanner
permalink: /cheap-police-scanner/
nav_group: Hardware
affiliate: true
faq:
  - q: "What is the best cheap police scanner?"
    a: "For analog agencies, the ~$110 Uniden BC125AT is the best cheap police scanner — a proven handheld that also covers air, marine, rail, and racing. The ~$100 BC75XLT and ~$120 SR30C are close alternatives. All three are analog only; if your area is digital, no sub-$150 scanner will hear it."
  - q: "Why are cheap police scanners analog only?"
    a: "Digital decoding (P25, DMR, NXDN) needs more processing and licensing, which pushes the price up. Every scanner under about $150 is analog only. If your police, fire, or EMS moved to a digital trunked system, a cheap analog scanner will scan the frequency but hear only noise."
  - q: "What is the cheapest way to hear digital police?"
    a: "A ~$30 RTL-SDR dongle plus free GopherTrunk software on a PC. It decodes P25 Phase I/II, DMR, NXDN, and TETRA and records every call — a fraction of the ~$380 a dedicated digital scanner (BCD325P2) costs. The trade-off is that it needs a computer and setup rather than being turnkey."
  - q: "What is the cheapest dedicated digital scanner?"
    a: "The Uniden BCD325P2 at around $380 is the cheapest handheld that decodes P25 Phase I and II on its own with no PC. It is the budget entry point for digital-only areas if you want a self-contained radio rather than an SDR."
  - q: "Will a cheap scanner pick up my local police?"
    a: "Only if they are still analog. Check your county on RadioReference first. If it lists analog conventional or analog trunked systems, a ~$100 BC125AT works. If it lists P25, DMR, or NXDN, you need a digital scanner (~$380+) or a $30 SDR with GopherTrunk."
  - q: "Can any cheap scanner or SDR decode encrypted police?"
    a: "No. Neither a $100 scanner, a $650 scanner, nor a $30 SDR can decode AES-encrypted talkgroups. Encryption is a cryptographic and legal wall. Spend based on the systems in your area that are still in the clear."
---

# Cheap Police Scanners (Best Under $150 + Free Options)

**The cheapest scanner that actually works depends on one fact: are your agencies analog or digital?** Get that wrong and a bargain radio is a paperweight. Under about $150 you can only buy **analog** scanners — the truly cheap way to hear **digital** police is a $30 SDR dongle plus free [GopherTrunk](/downloads.html) software.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Best cheap analog:** [Uniden BC125AT](/reference/uniden-bc125at/) (~$110). **Alternatives:** [BC75XLT](/reference/uniden-bc75xlt/) (~$100), [SR30C](/reference/uniden-sr30c/) (~$120). **Cheapest DIGITAL route:** a $30 [RTL-SDR](/reference/rtl-sdr/) + free [GopherTrunk](/downloads.html). **Cheapest dedicated digital scanner:** BCD325P2 (~$380). **Check [RadioReference](https://www.radioreference.com/) first** — cheap analog radios hear only analog systems. **Nothing decodes [encryption](/police-scanner-encryption/).**
</div>

## Why "cheap" means "analog"

Digital decoding costs money — the DSP horsepower and mode licensing for [P25](/reference/project-25/), [DMR](/reference/dmr/), and [NXDN](/reference/nxdn/) push a radio's price well past $150. So the rule is simple:

- **Everything under ~$150 is analog only.** It will scan a digital frequency and give you nothing but digital hash.
- **Cheap radios still shine on analog.** Plenty of fire departments, public works, rail, aviation, marine, and racing are still analog — a $100 handheld is perfect there.
- **Check before you buy.** Look up your county on [RadioReference](https://www.radioreference.com/). If it says analog conventional or analog trunked, buy cheap. If it says P25/DMR/NXDN, jump to the digital section below.

> **The trap.** The #1 reason a cheap scanner "doesn't work" is that the buyer's agencies went digital. The radio is fine — it just can't decode the mode. Read the system first.

## Best cheap analog scanners

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">Best cheap overall</span>
<h3>Uniden BC125AT</h3>
<p class="pick-card__price">around $110</p>
<p>The budget classic — 500 channels, close-call, and air/marine/rail/racing coverage. Analog only, but the one to get if your area hasn't gone digital.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B00772MR0K?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/uniden-bc125at/">BC125AT details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Cheapest</span>
<h3>Uniden BC75XLT</h3>
<p class="pick-card__price">around $100</p>
<p>The lowest-cost name-brand handheld, with weather alerts. Fewer bells than the BC125AT but the same analog reality.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B00A1VSO9M?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/uniden-bc75xlt/">BC75XLT details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Simple menus</span>
<h3>Uniden SR30C</h3>
<p class="pick-card__price">around $120</p>
<p>A newer, friendlier analog handheld with easy setup. Same analog-only limit — good for a first radio in an analog area.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B07S9H8YH3?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/uniden-sr30c/">SR30C details</a></p>
</div>
</div>

## The truly cheapest digital route: $30 SDR + GopherTrunk

If your agencies are **digital**, the cheapest way to hear them by far is **not** a scanner:

- **Hardware: ~$30.** An [RTL-SDR](/reference/rtl-sdr/) dongle plugs into any PC. That's roughly a *tenth* the cost of the cheapest digital scanner.
- **Software: free.** [GopherTrunk](/downloads.html) is free and open-source. It decodes **P25 Phase I/II, DMR, NXDN, and even TETRA**, follows the [control channel](/reference/control-channel/), and **records and timestamps every call** to disk.
- **The trade-off is honest.** An SDR needs a computer and a bit of setup; it isn't turnkey or pocket-portable like a scanner. But for pure cost-to-hear-digital, nothing beats it. See [police scanner vs SDR](/police-scanner-vs-sdr/) for the full comparison.

> **Own a computer already?** Then the marginal cost of hearing digital police is about $30. Start there before spending $380+ on a dedicated digital radio.

## Budget tiers at a glance

| Tier | Option | Approx. cost | Decodes | Best for |
|---|---|---|---|---|
| Cheapest analog | [BC75XLT](/reference/uniden-bc75xlt/) | ~$100 | Analog only | Analog areas, first radio |
| Best cheap analog | [BC125AT](/reference/uniden-bc125at/) | ~$110 | Analog only | Analog + air/marine/rail/race |
| Easy analog | [SR30C](/reference/uniden-sr30c/) | ~$120 | Analog only | Simple setup, analog areas |
| **Cheapest digital (any)** | **[RTL-SDR](/reference/rtl-sdr/) + [GopherTrunk](/downloads.html)** | **~$30 + PC** | **P25/DMR/NXDN/TETRA** | Digital areas, want it free + recorded |
| Cheapest dedicated digital | BCD325P2 | ~$380 | P25 P1/P2 | Digital areas, want turnkey, no PC |

## How to spend the least and still hear your area

- **Analog area, want turnkey?** [BC125AT](/reference/uniden-bc125at/) (~$110) — done.
- **Digital area, own a PC?** [$30 RTL-SDR](/reference/rtl-sdr/) + free [GopherTrunk](/downloads.html). Cheapest possible digital.
- **Digital area, no PC / want it in your pocket?** The BCD325P2 (~$380) is the cheapest self-contained digital scanner — see the [best police scanners](/best-police-scanners/) guide.
- **Not sure which you have?** Check [RadioReference](https://www.radioreference.com/) before spending anything.

## Bottom line

The best **cheap police scanner** is the **[Uniden BC125AT](/reference/uniden-bc125at/)** (~$110) — but only if your agencies are still **analog**; every sub-$150 radio is analog only. If they've gone **digital**, skip the cheap-scanner aisle entirely: a **$30 [RTL-SDR](/reference/rtl-sdr/) + free [GopherTrunk](/downloads.html)** is the cheapest way to hear digital police and records every call, while the cheapest *dedicated* digital scanner (BCD325P2) runs ~$380. Check [RadioReference](https://www.radioreference.com/) first, and remember: **no scanner or SDR at any price decodes [encryption](/police-scanner-encryption/)**.
