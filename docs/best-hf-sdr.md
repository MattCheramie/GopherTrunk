---
layout: page
title: "Best SDR for HF (Shortwave, Ham, Utility)"
description: "The best SDR for HF listening — Airspy HF+ Discovery vs the RTL-SDR Blog V4's built-in upconverter vs an external Ham It Up upconverter on a VHF dongle, compared for shortwave, ham, and utility."
keywords: best HF SDR, Airspy HF+ Discovery, RTL-SDR HF, upconverter, Ham It Up, shortwave SDR, ham radio SDR, utility monitoring, direct sampling, GopherTrunk HF
permalink: /best-hf-sdr/
nav_group: Hardware
affiliate: true
faq:
  - q: "What is the best SDR for HF?"
    a: "For dedicated HF listening the Airspy HF+ Discovery is the clear best pick — its high-dynamic-range front end pulls weak shortwave, ham, and utility signals out of a band full of strong broadcasters. If you also scan VHF/UHF and only dabble in HF, an RTL-SDR Blog V4 with its built-in upconverter is a fine one-radio compromise."
  - q: "Can an RTL-SDR receive HF?"
    a: "Yes. The RTL-SDR Blog V4 has a built-in HF upconverter, and the older V3 uses direct sampling, so both cover roughly 500 kHz to 30 MHz out of the box. A plain VHF-only dongle needs an external upconverter such as a Ham It Up to reach HF."
  - q: "What is an upconverter and do I need one?"
    a: "An upconverter shifts the HF band up into the VHF range a standard RTL-SDR can tune, so it can hear shortwave. You need one only if your dongle has no HF capability of its own. The RTL-SDR Blog V4 already includes an upconverter internally, so you do not need an external one with it."
  - q: "Is the Airspy HF+ Discovery worth the extra money over an RTL-SDR?"
    a: "For HF, yes. HF is a high-dynamic-range environment — a distant weak signal sits right next to a 60 dB stronger broadcaster. The HF+ Discovery's front end handles that far better than an 8-bit RTL-SDR plus upconverter, so weak-signal work is dramatically cleaner."
  - q: "Does GopherTrunk decode HF signals?"
    a: "GopherTrunk targets VHF/UHF digital trunking (P25, DMR, NXDN, TETRA), which does not live on HF. Use an HF SDR for shortwave, ham, and utility listening in general SDR software; pick a VHF/UHF dongle for GopherTrunk trunk-tracking. Some people own both."
  - q: "Will the RTL-SDR Blog V4 do both HF and trunk-tracking?"
    a: "Yes — it covers HF via its internal upconverter and VHF/UHF for GopherTrunk, making it the most versatile single dongle. It will not match a dedicated Airspy HF+ Discovery on weak HF, but it is the best value if you want one radio for everything."
---

# Best SDR for HF (Shortwave, Ham, Utility)

**The best SDR for serious HF listening is the [Airspy HF+ Discovery](/reference/airspy-hf-plus/);
the best value if you also scan VHF/UHF is an [RTL-SDR Blog V4](/reference/rtl-sdr/)
with its built-in [upconverter](/reference/upconverter/).** HF — shortwave broadcast,
ham bands, and utility stations — is a different game from VHF/UHF: the band is packed
with signals of wildly different strengths, so front-end dynamic range matters more
than anything else.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Best for HF:** [Airspy HF+ Discovery](/reference/airspy-hf-plus/) (~$170) — outstanding
dynamic range for weak shortwave/ham/utility. **Best value / one radio:**
[RTL-SDR Blog V4](/reference/rtl-sdr/) (~$40), HF included via its internal upconverter.
**Own a VHF-only dongle already?** Add a
[Ham It Up upconverter](https://www.amazon.com/dp/B009LQT3G6?tag=gophertrunk-20) (~$50).
**Note:** GopherTrunk decodes VHF/UHF trunking, not HF — this page is for general HF
listening.
</div>

## Three ways to get on HF

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">Best for HF</span>
<h3>Airspy HF+ Discovery</h3>
<p class="pick-card__price">around $170</p>
<p>Polyphase-harmonic-rejection front end with exceptional dynamic range. Hears weak DX right next to broadcast blowtorches. HF plus 60–260 MHz.</p>
<a class="btn btn--buy" href="https://www.amazon.com/s?k=Airspy+HF+Discovery&tag=gophertrunk-20" rel="nofollow sponsored noopener">HF+ on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/airspy-hf-plus/">HF+ details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Best value</span>
<h3>RTL-SDR Blog V4</h3>
<p class="pick-card__price">around $40</p>
<p>HF built in via an internal upconverter, plus full VHF/UHF for GopherTrunk. One dongle covers shortwave and trunk-tracking.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B0CD745394?tag=gophertrunk-20" rel="nofollow sponsored noopener">V4 on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/rtl-sdr/">RTL-SDR details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Add-on</span>
<h3>Ham It Up upconverter</h3>
<p class="pick-card__price">around $50</p>
<p>Bolts HF onto any VHF-only RTL-SDR by shifting 0–30 MHz up to where the dongle can tune. Use it to give an existing dongle shortwave.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B009LQT3G6?tag=gophertrunk-20" rel="nofollow sponsored noopener">Ham It Up on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/upconverter/">Upconverter details</a></p>
</div>
</div>

## Comparison

| Path | How it reaches HF | HF dynamic range | Also does VHF/UHF? | Approx price |
|---|---|---|---|---|
| [Airspy HF+ Discovery](/reference/airspy-hf-plus/) | Native HF front end | Excellent | Yes (60–260 MHz) | ~$170 |
| [RTL-SDR Blog V4](/reference/rtl-sdr/) | Built-in upconverter | Modest (8-bit) | Yes (full) | ~$40 |
| [RTL-SDR Blog V3](/reference/rtl-sdr/) | Direct sampling (Q-branch) | Modest, needs care | Yes (full) | ~$45 |
| VHF dongle + [Ham It Up](/reference/upconverter/) | External upconverter | Modest (8-bit) | Yes (full) | dongle + ~$50 |

## Why HF punishes cheap front ends

On VHF/UHF you usually chase one signal in a fairly quiet neighborhood. **HF is the
opposite:** the shortwave broadcast bands are jammed with signals, and after dark a
50 kW international broadcaster can sit a few kilohertz from the faint ham or utility
station you want. That is a high-dynamic-range problem, and it is exactly where an
8-bit [RTL-SDR](/reference/rtl-sdr/) struggles — the strong signal drives its front end
toward overload and buries the weak one.

The [Airspy HF+ Discovery](/reference/airspy-hf-plus/) is engineered specifically for
this: its high-dynamic-range design and sharp filtering let the weak signal survive next
to the strong one. That is what you are paying the extra money for, and on HF it is
worth it.

> **On HF, a [filter](/sdr-filters/) still helps.** Even a good SDR benefits from an
> [AM broadcast notch](/sdr-filters/) if a local mediumwave station is overwhelming the
> input. Filtering the interferer out in hardware protects the front end before the ADC.

## Upconverter vs built-in HF

If you are buying fresh and want HF, **do not buy a bare VHF dongle plus an external
upconverter** — the [RTL-SDR Blog V4](/reference/rtl-sdr/) already has an upconverter
inside it for around the same total money and far less cabling. An external
[Ham It Up](https://www.amazon.com/dp/B009LQT3G6?tag=gophertrunk-20) makes sense in one
situation: **you already own a VHF-only dongle** (an older NESDR, a FlightAware stick)
and want to add HF without replacing it. The direct-sampling V3 also reaches HF but
requires more care with imaging and gain than the V4's cleaner upconverter path.

## Where GopherTrunk fits

Be clear about the split: **GopherTrunk decodes VHF/UHF digital trunking** — 
[P25](/reference/project-25/), DMR, NXDN, TETRA — which does not live on HF. So HF
capability is a *bonus*, not a GopherTrunk requirement. If you want one radio that does
GopherTrunk trunk-tracking *and* lets you tune shortwave in other SDR software, the
[V4](/reference/rtl-sdr/) is the natural choice. If HF weak-signal work is your main
hobby and trunking is secondary, buy the [HF+ Discovery](/reference/airspy-hf-plus/) for
HF and a cheap dongle for GopherTrunk.

## Bottom line

For dedicated HF — shortwave DX, ham weak-signal, utility monitoring — the
**[Airspy HF+ Discovery](/reference/airspy-hf-plus/)** is the best SDR, full stop, thanks
to its dynamic range. For the best *value*, and for anyone who also wants to run
GopherTrunk on VHF/UHF, the **[RTL-SDR Blog V4](/reference/rtl-sdr/)** covers HF with its
built-in [upconverter](/reference/upconverter/) at a fraction of the price. Only reach
for an external [Ham It Up](https://www.amazon.com/dp/B009LQT3G6?tag=gophertrunk-20) when
you are adding HF to a dongle you already own. See the full lineup in
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/) and the starter list in
[what do I need](/what-do-i-need-for-gophertrunk/).
