---
layout: page
title: "RTL-SDR Blog V3 vs V4: Which to Buy"
description: "RTL-SDR Blog V3 vs V4 compared for scanning with GopherTrunk — R860 direct-sampling HF vs R828D built-in upconverter, HF fold, filtering, insertion loss, and the V4 end-of-line question."
keywords: RTL-SDR Blog V3 vs V4, RTL-SDR V3 vs V4, RTL-SDR Blog V4, RTL-SDR Blog V3, R828D, R860, RTL-SDR HF, RTL-SDR discontinued, which RTL-SDR to buy
permalink: /rtl-sdr-blog-v3-vs-v4/
nav_group: Hardware
affiliate: true
faq:
  - q: "Should I buy the RTL-SDR Blog V3 or V4?"
    a: "For VHF/UHF scanning with GopherTrunk, either is excellent and they decode identically. Buy the V4 if you want built-in HF without an upconverter and slightly better front-end filtering, and you can find one in stock. Buy the V3 if you want a unit you can re-order for years, since the V4 is end-of-line."
  - q: "Is the RTL-SDR Blog V4 discontinued?"
    a: "Effectively yes. The V4's Rafael Micro R828D tuner is discontinued, so RTL-SDR Blog has stated the V4 is the last of its line. It still works perfectly, but once current stock is gone it will not be restocked, whereas the R860-based V3 continues."
  - q: "What is the real difference between V3 and V4 on HF?"
    a: "The V3 receives HF by direct sampling on the ADC's second Nyquist zone, which folds signals above roughly 14.4 MHz and needs a Q-branch mode. The V4 adds a built-in upconverter so all of HF appears cleanly in normal quadrature-sampling mode with no fold and no aliasing images to fight."
  - q: "Does the V4 have better filtering than the V3?"
    a: "Yes. The V4 adds notch filtering and improved input band filtering that reject strong out-of-band signals — broadcast FM and cellular — better than the V3. The trade-off is a couple of dB of insertion loss from those filters, which is rarely a problem in practice."
  - q: "Can either one decode encrypted police?"
    a: "No. Neither the V3, the V4, nor any radio can decode AES-encrypted talkgroups. GopherTrunk decodes clear P25/DMR/NXDN/TETRA only. Choose based on the systems still in the clear in your area."
  - q: "Do I need the V4's HF coverage for scanning?"
    a: "Only if you also want shortwave, ham HF, or utility monitoring. Trunked police, fire, and EMS systems live in VHF/UHF (roughly 136–870 MHz), which both dongles cover natively. If you never touch HF, the HF difference between V3 and V4 does not matter."
---

# RTL-SDR Blog V3 vs V4: Which to Buy

**For scanning with [GopherTrunk](/downloads.html), the V3 and V4 decode
identically — the choice comes down to HF handling and stock.** Buy the
[V4](/reference/rtl-sdr/) for built-in HF and better filtering while it is still
available; buy the [V3](/reference/rtl-sdr/) if you want a dongle you can re-order
for years. Both are proper [RTL-SDR](/reference/rtl-sdr/) units with a real TCXO,
SMA connector, and switchable [bias tee](/reference/bias-tee/).

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**V4 wins on HF** (built-in upconverter, no fold) **and filtering** (notch +
better band filters). **V3 wins on availability** — the V4 is
**end-of-line** because its R828D tuner is discontinued. **For VHF/UHF
trunk-tracking they are equal.** The V4's extra filters cost ~2–3 dB of
insertion loss. **Neither decodes [AES encryption](/police-scanner-encryption/).**
</div>

## Comparison at a glance

| | RTL-SDR Blog V3 | RTL-SDR Blog V4 |
|---|---|---|
| Tuner | R860 (R820T2 family) | Rafael Micro **R828D** |
| HF reception | Direct sampling (Q-branch) | **Built-in upconverter** |
| HF fold / aliasing | Folds above ~14.4 MHz | **None** — clean across HF |
| Input filtering | Basic | **Notch + improved band filters** |
| Insertion loss | Lower | ~2–3 dB higher (the filters) |
| TCXO | 1 ppm | 1 ppm |
| Connector / bias tee | SMA / switchable | SMA / switchable |
| Availability | **Ongoing** | **End-of-line** |
| Approx price (with kit) | ~$45 | ~$40–50 |

## HF: direct sampling vs a built-in upconverter

This is the biggest technical difference. The **V3** reaches HF by *direct
sampling* — feeding the antenna straight into the [ADC](/reference/analog-to-digital-converter/)
and listening on its second Nyquist zone. It works, but signals above roughly
14.4 MHz **fold** back down, you have to switch to a special Q-branch mode, and
strong broadcast stations can alias into your passband.

The **V4** puts a small *upconverter* on the board. It shifts all of HF up into a
range the tuner handles natively, so shortwave, ham HF, and utility signals appear
cleanly in normal quadrature mode with **no fold and no images**. If HF matters to
you, the V4 is the easier, better receiver — no external
[upconverter](/reference/upconverter/) box required.

> **For police/fire/EMS this is a non-issue.** Trunked public-safety systems live
> in VHF/UHF (roughly 136–870 MHz), which both dongles cover natively and
> identically. The HF advantage of the V4 only matters if you also want to listen
> below 30 MHz.

## Filtering vs insertion loss

The V4 adds **notch filtering** (to reject broadcast FM and other strong local
signals) and improved input band filtering. In a strong-signal environment — near
FM transmitters or cellular sites — that cleaner front end can be the difference
between a clean control-channel lock and intermod hash.

The trade-off is honest: those filters add roughly **2–3 dB of insertion loss**.
On a weak, rural signal a couple of dB matters, and a mast-mounted
[LNA](/best-sdr-lna/) more than makes it back. In most suburban and urban setups
the filtering is worth far more than the loss. If your problem is a genuinely weak
signal rather than overload, that is a front-end dynamic-range issue no filter
fixes — see [when you actually need an Airspy](/airspy-vs-rtl-sdr-vs-hackrf/).

## The end-of-line reality

The V4's R828D tuner has been discontinued, and RTL-SDR Blog has said the V4 is
the last production run of that design. **Nothing about your V4 stops working** —
it is a great dongle for years to come — but it will not be restocked once current
inventory sells through. The R860-based **V3** (and the wider
[NESDR SMArt](/reference/nesdr/) class of R820T2/R860 dongles) continue in
production, so they are the safer bet if you are buying several for a
[multi-dongle pool](/multi-dongle-sdr-setup/) or want to standardise on something
you can re-order.

## Software has to know the V4

One practical caveat: the V4 is not "just an RTL-SDR with a different tuner." Its
**R828D** needs V4-aware driver support, and software carrying only the classic
R820T/R820T2 init will tune it wrong and hear **only noise** — exactly what
GopherTrunk's own bring-up hit in
[#264](https://github.com/MattCheramie/GopherTrunk/issues/264). Three things make
the V4 special:

- The V4 keeps the family's **28.8 MHz crystal**, while other R828D designs run
  16 MHz — assume the wrong one and every tuned frequency is off by 1.8×.
- The R828D wants a different **VCO power reference** (1, not the osmocom
  default 2) or the LO mistunes; see the
  [R820T tuner](/reference/r820t-tuner/) entry for the register details.
- The V4's **switched HF/VHF/UHF input bank** must be routed per band. Stock
  R828D init leaves every input off — the board routes no RF at all — so the
  driver has to switch inputs (plus the notch filters and the HF upconverter
  relay) as it tunes.

Current GopherTrunk handles all of this, as do up-to-date builds of the usual SDR
suites. If a V4 "hears nothing" while a V3 works, suspect stale software before
suspecting the dongle.

## Which should you buy?

<div class="pick-cards" markdown="0">
<div class="pick-card pick-card--top">
<span class="pick-card__badge">Best while in stock</span>
<h3>RTL-SDR Blog V4</h3>
<p class="pick-card__price">around $40</p>
<p>Built-in HF, best filtering, the enthusiast reference. Buy now if you want the best single dongle and can find one — it is end-of-line.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B0CD745394?tag=gophertrunk-20" rel="nofollow sponsored noopener">V4 on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/reference/rtl-sdr/">RTL-SDR details</a></p>
</div>
<div class="pick-card">
<span class="pick-card__badge">Best for the long haul</span>
<h3>RTL-SDR Blog V3 + dipole kit</h3>
<p class="pick-card__price">around $45</p>
<p>Proven R860 dongle plus a real antenna, and a design that stays in production. The safe pick if you want to re-order or buy in quantity.</p>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B0BMKB3L47?tag=gophertrunk-20" rel="nofollow sponsored noopener">V3 kit on Amazon &rarr;</a>
<p class="pick-card__note"><a href="/best-rtl-sdr/">best RTL-SDR guide</a></p>
</div>
</div>

## Bottom line

If HF interests you and you can find one, the **V4** is the better all-round
receiver — built-in upconverter, cleaner filtering, the reference dongle. If you
want something you can buy again next year, or you only care about VHF/UHF
scanning, the **V3** is every bit as good on the bands that matter and stays in
production. Either way you are getting a proper TCXO-equipped
[RTL-SDR](/reference/rtl-sdr/) that runs [GopherTrunk](/downloads.html) for about
$40–45, and **neither decodes [encryption](/police-scanner-encryption/)**. Still
deciding across brands? See the [best RTL-SDR](/best-rtl-sdr/) roundup and the
[best SDR overall](/best-sdr-for-gophertrunk/).
