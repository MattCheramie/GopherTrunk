---
slug: hydrasdr
title: HydraSDR RFOne
entry_type: hardware
category: sdr-devices
description: "The HydraSDR RFOne is a high-performance VHF/UHF software-defined radio receiver — a successor to the Airspy R2 with a 12-bit ADC, up to 10 MS/s, and a clean front end, designed in France and made in the USA."
keywords: HydraSDR, HydraSDR RFOne, RFOne, Airspy R2 successor, high performance SDR, VHF UHF receiver, 12-bit ADC, R820T2, 10 MSPS, wideband capture, best SDR USA
aka: [HydraSDR, RFOne]
autolink: true
infobox:
  - { label: Type, value: VHF/UHF SDR receiver }
  - { label: Vendor/Chip, value: "HydraSDR (Benjamin Vernoux); R820T2 tuner + LPC4370" }
  - { label: Models, value: HydraSDR RFOne }
  - { label: ADC, value: 12-bit }
  - { label: Range, value: ~24 MHz – 1.8 GHz }
  - { label: Bandwidth, value: up to ~10 MHz }
  - { label: TX, value: No (receive only) }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.digikey.com/en/products/detail/benjamin-vernoux/HYDRASDR-RFONE/26256067\" rel=\"nofollow noopener\">Buy at DigiKey &rarr;</a>" }
see_also: [airspy, airspy-hf-plus, rtl-sdr, r820t-tuner, sdrplay-rspdx, dynamic-range, software-defined-radio]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/rf-sdr/sdr-hardware/ }
related_reading:
  - { title: "RF Front End, Part 10: Airspy — real to complex", url: /blog/deep-dives/rf-front-end-10-airspy-real-to-complex/ }
cite_urls:
  - https://hydrasdr.com/
  - https://en.wikipedia.org/wiki/Software-defined_radio
---

**HydraSDR RFOne** is a high-performance VHF/UHF
[software-defined radio](/reference/software-defined-radio/) receiver in the lineage of
the [Airspy R2](/reference/airspy/) — same 12-bit oversampling architecture and
[R820T2](/reference/r820t-tuner/) front end, carried forward as an independent open
project. It samples up to ~10 MHz of spectrum anywhere from ~24 MHz to 1.8 GHz.[^hydra]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A frequency coverage bar for HydraSDR RFOne (~24 MHz–1.8 GHz) on an axis from about 0 to 6 gigahertz." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="86">0</text><text x="163" y="86">2 GHz</text><text x="296" y="86">4 GHz</text><text x="430" y="86">6 GHz</text></g>
  <rect x="32" y="40" width="120" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="28" text-anchor="middle" font-size="10" fill="currentColor">HydraSDR RFOne (~24 MHz–1.8 GHz) coverage</text>
</svg>
<figcaption>HydraSDR RFOne covers the VHF/UHF bands scanners use, at Airspy-class resolution.</figcaption>
</figure>

## Overview

The RFOne is aimed squarely at the niche the [Airspy R2](/reference/airspy/) created: a
receiver with the sensitivity, [dynamic range](/reference/dynamic-range/), and capture
[bandwidth](/reference/bandwidth/) an [RTL-SDR](/reference/rtl-sdr/) can't reach, in the
bands public-safety and business radio actually use. It is designed in France by
Benjamin Vernoux and manufactured in the USA, which — with the recent rise in RTL-SDR
dongle prices — makes it one of the better value high-performance receivers on the US
market.[^hydra]

## How it works

Like the Airspy R2, the RFOne pairs a Rafael Micro [R820T2](/reference/r820t-tuner/)
tuner with a **12-bit** [ADC](/reference/analog-to-digital-converter/) driven by an NXP
LPC4370, replacing the [RTL2832U](/reference/rtl2832u/)'s 8-bit ADC and USB bridge. Two
design choices give it its edge over a dongle:

- **More bits.** A 12-bit ADC carries roughly **72 dB** of theoretical
  [dynamic range](/reference/dynamic-range/) against the RTL2832U's ~48 dB — the
  headroom that lets a weak signal survive next to a strong one without the front end
  clipping.
- **Oversampling and real-to-complex conversion.** The receiver samples the IF at a high
  real rate and digitally converts it to complex baseband on the way out, decimating to
  the requested rate. Averaging many high-rate samples into each output adds effective
  bits (process gain), so the delivered stream is quieter than the raw ADC alone.

It delivers up to about **10 MS/s** of complex [bandwidth](/reference/bandwidth/) and is
**receive-only** — there is no transmit path.

## Variants

- **HydraSDR RFOne** — the single current model: ~24 MHz–1.8 GHz, up to ~10 MS/s, 12-bit,
  with a clock reference and a [bias tee](/reference/bias-tee/) for powering an inline
  [LNA](/reference/low-noise-amplifier/).

Compared with the [Airspy R2](/reference/airspy/) it is architecturally very close — a
practical alternative in the same slot. Against an [SDRplay RSPdx](/reference/sdrplay-rspdx/)
it trades the RSPdx's 14-bit ADC and HF coverage for the R820T2 path's simplicity and a
pure-USB streaming model. As with any 12-bit direct-sampling receiver, put a
[broadcast-FM notch filter](/reference/fm-broadcast-filter/) in front of it in a strong
signal environment to keep the front end out of overload.

## Relevance to SDR

GopherTrunk drives the HydraSDR over USB for demanding reception where an RTL-SDR's
bandwidth or sensitivity falls short — a congested band, a distant control channel, or a
multi-site system whose control channels are spread too far apart to fit one RTL-SDR
capture. The extra ADC bits and wider capture are exactly what a wideband, multi-tap
channelizer wants, so like the Airspy it is a strong choice for the `role: wideband` use.
It remains a receiver only, so it decodes clear and scrambled traffic, never keyed
encryption.

## Where to buy

The RFOne is sold through **[DigiKey](https://www.digikey.com/en/products/detail/benjamin-vernoux/HYDRASDR-RFONE/26256067)**
(~US$189) and listed on the manufacturer's site at
**[hydrasdr.com](https://hydrasdr.com/)** — it is not currently carried on Amazon.

## Sources

[^hydra]: [HydraSDR](https://hydrasdr.com/) — manufacturer site, on the RFOne's R820T2 + 12-bit oversampling architecture, ~24 MHz–1.8 GHz range, up-to-10 MS/s capture, and France-designed / USA-made origin.
