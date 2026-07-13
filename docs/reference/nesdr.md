---
slug: nesdr
title: Nooelec NESDR
entry_type: hardware
category: sdr-devices
description: "Nooelec NESDR is a family of purpose-built RTL-SDR receivers with a TCXO, metal case and proper connectors, upgrading the generic RTL2832U dongle."
keywords: NESDR, Nooelec NESDR, NESDR SMArt, NESDR Nano, NESDR SMArTee, NESDR XTR, RTL-SDR variant, RTL2832U, R820T2, TCXO, bias tee
aka: [NESDR, Nooelec NESDR, NooElec NESDR]
autolink: true
infobox:
  - { label: Type, value: USB SDR receiver (RTL-SDR) }
  - { label: Vendor, value: Nooelec }
  - { label: Bridge chip, value: RTL2832U }
  - { label: Tuner, value: R820T2 / R860 (E4000 on XTR) }
  - { label: ADC, value: 8-bit }
  - { label: Range, value: "~100 kHz – 1.75 GHz" }
  - { label: TX, value: No (receive only) }
  - { label: Typical price, value: "$25 – $45" }
see_also: [rtl-sdr, rtl2832u, r820t-tuner, e4000-tuner, bias-tee, software-defined-radio]
cite_urls:
  - https://en.wikipedia.org/wiki/RTL-SDR
  - https://www.nooelec.com/store/sdr/sdr-receivers.html
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

## Sources

[^wiki]: [RTL-SDR](https://en.wikipedia.org/wiki/RTL-SDR) — Wikipedia, on RTL-SDR variants including purpose-built dongles with TCXOs.
[^nooelec]: [Nooelec SDR receivers](https://www.nooelec.com/store/sdr/sdr-receivers.html) — the vendor, for NESDR model specifications and options.
