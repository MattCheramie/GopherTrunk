---
slug: nesdr
title: Nooelec NESDR
entry_type: hardware
category: sdr-devices
description: "Nooelec NESDR is a family of purpose-built RTL-SDR receivers with a TCXO, metal case and proper connectors, upgrading the generic RTL2832U dongle."
keywords: NESDR, Nooelec NESDR, NESDR SMArt, NESDR Nano, NESDR SMArTee, NESDR XTR, RTL-SDR variant, RTL2832U, R820T2, TCXO, bias tee
aka: [NESDR, Nooelec NESDR, NooElec NESDR]
autolink: true
affiliate: true
product:
  name: "NooElec NESDR SMArt v5"
  brand: NooElec
  category: Software-defined radio
  lowPrice: "30"
  highPrice: "40"
  url: https://www.amazon.com/dp/B01HA642SW?tag=gophertrunk-20
infobox:
  - { label: Type, value: USB SDR receiver (RTL-SDR) }
  - { label: Vendor, value: Nooelec }
  - { label: Bridge chip, value: RTL2832U }
  - { label: Tuner, value: R820T2 / R860 (E4000 on XTR) }
  - { label: ADC, value: 8-bit }
  - { label: Range, value: "~100 kHz – 1.75 GHz" }
  - { label: TX, value: No (receive only) }
  - { label: Typical price, value: "$25 – $45" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B01HA642SW?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [rtl-sdr, rtl2832u, r820t-tuner, e4000-tuner, bias-tee, software-defined-radio]
cite_urls:
  - https://en.wikipedia.org/wiki/RTL-SDR
  - https://www.nooelec.com/store/sdr/sdr-receivers.html
faq:
  - q: "Which NESDR should I buy for GopherTrunk?"
    a: "The NESDR SMArt v5 for most people — it is the mainstream 'just buy this' RTL-SDR, an R820T2/R860 dongle with a 0.5 ppm TCXO in an aluminium case, and it's essentially always in stock. Get the SMArTee v2 instead if you need an always-on bias tee to power an inline LNA or active antenna, the Nano 3 for a tiny embedded build, or the SMArt XTR (E4000) if you need tuning up toward ~2.2 GHz."
  - q: "Is a NESDR better than a generic RTL-SDR dongle?"
    a: "Yes, for a small premium. A NESDR is functionally an RTL-SDR — same 8-bit RTL2832U and ~2.4 MHz of bandwidth — but the board around the chips fixes a generic stick's worst faults: a 0.5 ppm TCXO holds frequency as it warms up instead of drifting tens of kHz, plus better shielding, ESD protection and a proper SMA connector. That stability means steadier control-channel lock for long trunk-tracking sessions."
  - q: "Does GopherTrunk need Zadig or extra drivers for a NESDR?"
    a: "On Windows you bind the driver with Zadig once, exactly as for any RTL-SDR. On Linux it works with the standard librtlsdr-style raw-IQ interface. GopherTrunk then drives it like any other RTL-SDR, including in a pool of several dongles to cover channels spread across a band."
  - q: "Can a NESDR decode encrypted or trunked police traffic?"
    a: "It decodes clear P25/DMR/NXDN/TETRA trunked systems in software with GopherTrunk, which is receive-only. No RTL-SDR — and no scanner — can decode AES-encrypted transmissions."
---

**Nooelec NESDR** is a family of purpose-built [RTL-SDR](/reference/rtl-sdr/) receivers
from Nooelec — the same [RTL2832U](/reference/rtl2832u/) bridge and
[R820T2](/reference/r820t-tuner/) tuner at the core of every RTL-SDR, but wrapped in a
board and enclosure engineered for radio use rather than a repurposed TV dongle.[^wiki]
The NESDR line's calling card is a low-drift **TCXO**, a metal case, and a proper
antenna connector, which together deliver far steadier tuning and cleaner reception than
a generic no-name stick at only a small price premium.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="Block diagram of a NESDR: antenna into an R820T2 tuner into the RTL2832U with its 8-bit ADC, out over USB, with a low-drift TCXO clocking the RTL2832U." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="30" y="52">antenna</text>
    <rect x="76" y="34" width="92" height="34" rx="4" fill="none" stroke="currentColor" stroke-opacity="0.7"/>
    <text x="122" y="48">R820T2</text><text x="122" y="60" font-size="7">tuner</text>
    <rect x="200" y="34" width="118" height="34" rx="4" fill="none" stroke="currentColor" stroke-opacity="0.7"/>
    <text x="259" y="48">RTL2832U</text><text x="259" y="60" font-size="7">8-bit ADC + USB</text>
    <text x="392" y="52">computer (IQ)</text>
    <rect x="200" y="80" width="118" height="16" rx="3" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
    <text x="259" y="92" font-size="7">0.5 ppm TCXO</text>
  </g>
  <g stroke="currentColor" stroke-opacity="0.6">
    <line x1="55" y1="47" x2="74" y2="47"/><line x1="168" y1="47" x2="198" y2="47"/><line x1="318" y1="47" x2="352" y2="47"/>
    <line x1="259" y1="68" x2="259" y2="80"/>
  </g>
</svg>
<figcaption>A NESDR is a well-built RTL-SDR: the same two chips, plus a 0.5 ppm TCXO for stable tuning, in a metal case.</figcaption>
</figure>

<a class="btn btn--buy" href="https://www.amazon.com/dp/B01HA642SW?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The mainstream "just buy this" [RTL-SDR](/reference/rtl-sdr/).** A NooElec NESDR is a
purpose-built RTL-SDR — same 8-bit [RTL2832U](/reference/rtl2832u/) and
[R820T2](/reference/r820t-tuner/)/R860 tuner as any dongle, but with a **0.5 ppm TCXO**,
a metal case and a proper SMA connector that fix a generic stick's drift and shielding
faults. The **SMArt v5** is the model to get for a first GopherTrunk radio — cheap
(~$35) and reliably in stock. Variants add a bias tee (SMArTee v2), a tiny body
(Nano 3) or extended range (SMArt XTR). **Receive-only;** it decodes clear
[P25](/reference/project-25/)/[DMR](/reference/dmr/)/[NXDN](/reference/nxdn/) but no
dongle can decode [AES encryption](/police-scanner-encryption/). See
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).
</div>

## Overview

Nooelec was one of the first vendors to ship RTL2832U dongles refined specifically for
SDR rather than digital-TV reception. Every NESDR is functionally an RTL-SDR — **8-bit**
converters, roughly **2.4 MHz** of usable [bandwidth](/reference/bandwidth/), and
**receive-only** — so it shares the platform's hard ceilings. What you pay a little extra
for is the *board around the chips*: a **0.5 ppm temperature-compensated oscillator**
(TCXO) that holds frequency as the dongle warms up instead of drifting tens of kHz,
better shielding and ESD protection, and an SMA connector.

## Variants

- **NESDR SMArt v5** — the mainstream "buy this" model: [R820T2](/reference/r820t-tuner/)
  / R860, 100 kHz – 1.75 GHz with direct-sampling HF, in an aluminium case. Markedly
  better HF/VHF SNR and tuning stability than a generic stick.
- **NESDR Nano 3** — the same TCXO and tuner in a tiny body for embedded or portable use.
- **NESDR SMArTee v2** — adds an **always-on 4.5 V [bias tee](/reference/bias-tee/)** to
  power an inline LNA or active antenna without hardware modification.
- **NESDR SMArt XTR / SMArTee XTR** — built on the Elonics
  [E4000](/reference/e4000-tuner/) tuner, trading some sensitivity for extended tuning up
  toward ~2.2 GHz where the R820T2 cannot reach.
- **NESDR Mini / Mini 2** — earlier compact entry models, largely superseded by the SMArt
  line.

## In practice

The NESDR family occupies the sweet spot between a $12 lottery-quality generic dongle and
a higher-end receiver like an [Airspy](/reference/airspy/). At **$25–$45** it removes the
generic dongle's worst faults — thermal drift, poor shielding, flimsy connectors — while
keeping the low cost and the huge software ecosystem of the RTL-SDR platform. The limits
that remain are inherent to the [RTL2832U](/reference/rtl2832u/): the 8-bit ADC's modest
dynamic range (set [gain](/reference/automatic-gain-control/) carefully to avoid
overload) and the ~2.4 MHz capture width. For HF below the tuner's reach it relies on
direct sampling, with the usual Nyquist fold, or an external upconverter.

## Relevance to GopherTrunk

A NESDR is an excellent, well-behaved entry point for GopherTrunk — its stable TCXO
means less frequency correction and steadier lock on a control channel than a drifting
generic dongle, which matters when tracking a trunked system for long sessions.
GopherTrunk drives it exactly like any other [RTL-SDR](/reference/rtl-sdr/), including in
a **pool** of several dongles to cover channels spread across a band, and on Windows you
bind the driver with Zadig first. For a first radio, an R820T2/R860-class NESDR SMArt v5
is a strong choice. See the [hardware guide](/hardware.html) for GopherTrunk's tested
devices.

## Where to buy

The **NESDR SMArt v5** is the one to buy for a first GopherTrunk radio; the other
NESDRs cover specific needs (an always-on bias tee, a tiny body, or extended range).
All are widely stocked on Amazon.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B01HA642SW?tag=gophertrunk-20" rel="nofollow sponsored noopener">NESDR SMArt v5 on Amazon &rarr;</a>

| Model | Tuner | Best for | Amazon |
|-------|-------|----------|--------|
| **NESDR SMArt v5** | R820T2/R860 | The mainstream pick — buy this first (~$35) | <a href="https://www.amazon.com/dp/B01HA642SW?tag=gophertrunk-20" rel="nofollow sponsored noopener">product</a> |
| **NESDR SMArt v5** (3-antenna bundle) | R820T2/R860 | Same dongle with a set of antennas | <a href="https://www.amazon.com/dp/B01GDN1T4S?tag=gophertrunk-20" rel="nofollow sponsored noopener">product</a> |
| **NESDR SMArTee v2** (bias tee) | R820T2 | Powering an inline LNA / active antenna | <a href="https://www.amazon.com/dp/B079C3FHPG?tag=gophertrunk-20" rel="nofollow sponsored noopener">product</a> |
| **NESDR Nano 3** (tiny) | R820T2 | Embedded / portable builds | <a href="https://www.amazon.com/dp/B073JZ8CC2?tag=gophertrunk-20" rel="nofollow sponsored noopener">product</a> |
| **NESDR SMArt XTR** (extended range) | E4000 | Tuning up toward ~2.2 GHz | <a href="https://www.amazon.com/dp/B06Y1HKLHY?tag=gophertrunk-20" rel="nofollow sponsored noopener">product</a> |

Comparing radios? See [best SDR for GopherTrunk](/best-sdr-for-gophertrunk/), the
[RTL-SDR](/reference/rtl-sdr/) family overview, or step up to an
[Airspy](/reference/airspy/) for tougher RF. Weighing a handheld instead? Read
[police scanner vs SDR](/police-scanner-vs-sdr/). Then grab the software from the
[downloads page](/downloads.html).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost
to you. It never changes what we recommend.*

## Sources

[^wiki]: [RTL-SDR](https://en.wikipedia.org/wiki/RTL-SDR) — Wikipedia, on RTL-SDR variants including purpose-built dongles with TCXOs.
[^nooelec]: [Nooelec SDR receivers](https://www.nooelec.com/store/sdr/sdr-receivers.html) — the vendor, for NESDR model specifications and options.
