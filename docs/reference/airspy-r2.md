---
slug: airspy-r2
title: Airspy R2
entry_type: hardware
category: sdr-devices
description: "The Airspy R2 is a 12-bit VHF/UHF software-defined radio that captures up to ~10 MS/s with a clock output and bias tee — GopherTrunk's recommended wideband, multi-site channelizer."
keywords: Airspy R2, Airspy R2 SDR, 12-bit SDR, VHF UHF receiver, wideband capture, 10 MSPS, R820T2, multi-site channelizer, high performance SDR
aka: [Airspy R2]
autolink: true
affiliate: true
product:
  name: "Airspy R2"
  brand: Airspy
  category: Software-defined radio
  lowPrice: "150"
  highPrice: "185"
  url: https://www.amazon.com/s?k=Airspy+R2+SDR&tag=gophertrunk-20
infobox:
  - { label: Type, value: VHF/UHF SDR receiver }
  - { label: Vendor/Chip, value: "Airspy; R820T2 tuner + LPC4370" }
  - { label: ADC, value: 12-bit }
  - { label: Range, value: ~24 MHz – 1.8 GHz }
  - { label: Bandwidth, value: up to ~10 MS/s }
  - { label: Extras, value: "Clock output, 4.5 V bias tee" }
  - { label: TX, value: No (receive only) }
  - { label: Price, value: around $150–185 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/s?k=Airspy+R2+SDR&tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [airspy, airspy-mini, airspy-hf-plus-discovery, airspy-rate-selection, rtl-sdr, sdrplay-rsp1b, hydrasdr, software-defined-radio, dynamic-range]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/rf-sdr/sdr-hardware/ }
related_reading:
  - { title: "RF Front End, Part 10: Airspy — real to complex", url: /blog/deep-dives/rf-front-end-10-airspy-real-to-complex/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
  - https://airspy.com/airspy-r2/
faq:
  - q: "Is the Airspy R2 worth it over an RTL-SDR for GopherTrunk?"
    a: "For tough or busy RF, yes. The R2's 12-bit ADC carries far more dynamic range than an RTL-SDR's 8-bit converter, and its ~10 MS/s capture lets one dongle channelize several control channels — including multiple sites of one P25 system — at once. For a first radio on a clean single-site system, a $30 RTL-SDR is still the cheapest thing that works."
  - q: "Airspy R2 or Airspy Mini?"
    a: "Same 12-bit architecture. The R2 captures up to ~10 MS/s and adds a clock output and a 4.5 V bias tee; the Mini tops out around 6 MS/s in a smaller, cheaper dongle. Choose the R2 for wideband multi-site channelizing and inline LNA power; the Mini for most of the quality at a lower price."
  - q: "Does GopherTrunk need libairspy or SoapySDR for the R2?"
    a: "No. GopherTrunk drives the Airspy R2 directly over USB with a pure-Go backend — no libairspy, no SoapySDR. It's the recommended device for GopherTrunk's wideband, multi-tap channelizer role."
  - q: "Can the Airspy R2 decode encrypted police channels?"
    a: "No. The R2 is a receiver and GopherTrunk is receive-only. It decodes clear P25/DMR/NXDN/TETRA, but no radio or scanner can decode AES-encrypted transmissions."
---

**The Airspy R2** is a high-performance VHF/UHF
[software-defined radio](/reference/software-defined-radio/) receiver with a **12-bit**
[ADC](/reference/analog-to-digital-converter/), capture up to about **10 MS/s**, a clock
output for chaining, and a 4.5 V [bias tee](/reference/bias-tee/) — the flagship of the
[Airspy](/reference/airspy/) line and **GopherTrunk's recommended wideband, multi-site
channelizer**.[^wiki] It carries far more [dynamic range](/reference/dynamic-range/) and
wider [bandwidth](/reference/bandwidth/) than an [RTL-SDR](/reference/rtl-sdr/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A frequency coverage bar for the Airspy R2 (~24 MHz–1.8 GHz) on an axis from about 0 to 6 gigahertz." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="30" y="86">0</text><text x="163" y="86">2 GHz</text><text x="296" y="86">4 GHz</text><text x="430" y="86">6 GHz</text></g>
  <rect x="32" y="40" width="120" height="20" rx="3" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="28" text-anchor="middle" font-size="10" fill="currentColor">Airspy R2 (~24 MHz–1.8 GHz) coverage</text>
</svg>
<figcaption>The R2 covers VHF/UHF with a 12-bit converter and up to ~10 MS/s of capture.</figcaption>
</figure>

<a class="btn btn--buy" href="https://www.amazon.com/s?k=Airspy+R2+SDR&tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**GopherTrunk's recommended wideband radio.** The Airspy R2's **12-bit** ADC (~72 dB
[dynamic range](/reference/dynamic-range/) vs an RTL-SDR's ~48 dB) and **~10 MS/s** capture
let one dongle channelize several [control channels](/reference/control-channel/) — including
multiple **sites** of one [P25](/reference/project-25/) system — out of a single IQ capture.
Adds a **clock output** and a **4.5 V [bias tee](/reference/bias-tee/)** for an inline
[LNA](/reference/low-noise-amplifier/). **Receive-only, ~$150–185.** Airspy is
distributor-sold, so Amazon stock is intermittent — the button tracks live listings.
For most of the quality cheaper, see the [Airspy Mini](/reference/airspy-mini/); like every
receiver it can't decode [AES encryption](/police-scanner-encryption/). New here? See
[best SDR for GopherTrunk](/best-sdr-for-gophertrunk/).
</div>

## Overview

The R2 is the radio you reach for when an [RTL-SDR](/reference/rtl-sdr/)'s 8-bit ADC or
~2.4 MHz of usable bandwidth is the thing standing between you and a decode. Its ~10 MS/s
capture is useful when a system's channels are spread across a band, or in tough RF where a
strong neighbour would push an 8-bit front end into clipping. For the lower bands, the
[Airspy HF+ Discovery](/reference/airspy-hf-plus-discovery/) is the specialised choice; the
smaller [Airspy Mini](/reference/airspy-mini/) is the same design in a cheaper dongle.

## How it works

An R2 shares the front-end tuner with an RTL-SDR — a Rafael Micro
[R820T2](/reference/r820t-tuner/) — but replaces everything behind it. Instead of the
[RTL2832U](/reference/rtl2832u/)'s 8-bit ADC and USB bridge, the Airspy digitises the tuner's
IF with a **12-bit** [ADC](/reference/analog-to-digital-converter/) driven by an NXP LPC4370
microcontroller, then streams the samples over USB 2.0. Two design choices give it its edge:

- **More bits.** A 12-bit ADC carries roughly **72 dB** of theoretical
  [dynamic range](/reference/dynamic-range/) against the RTL2832U's ~48 dB — the headroom
  that lets a weak signal survive next to a strong one without the front end clipping.
- **Oversampling and real-to-complex conversion.** The Airspy samples the IF at a high real
  rate and digitally converts it to complex baseband on the way out, decimating to the
  requested rate. Averaging many high-rate samples into each output sample adds effective
  bits (process gain), so the delivered stream is quieter than the raw ADC alone.[^airspy]

The R2 is **receive-only** — there is no transmit path. The
[HydraSDR RFOne](/reference/hydrasdr/) is an independent successor built on the same 12-bit
R820T2 architecture, and an [SDRplay RSP1B](/reference/sdrplay-rsp1b/) is the closest 14-bit
alternative in a similar niche.

## Relevance to GopherTrunk

GopherTrunk supports the R2 for demanding reception where an RTL-SDR's bandwidth or
sensitivity falls short — a congested band, a distant control channel, or a multi-site system
whose control channels are spread too far apart to fit one RTL-SDR capture. The extra ADC bits
and wider capture are exactly what a wideband, multi-tap channelizer wants, which is why the R2
is GopherTrunk's recommended device for the `role: wideband` use. GopherTrunk drives it over
USB with a pure-Go backend (no `libairspy` needed); it remains a receiver only, so it decodes
clear and scrambled traffic, never keyed encryption.

## Wideband multi-site monitoring

An R2 pinned to `role: wideband` can channelize several control channels — including multiple
**sites** of one P25 system — out of a single IQ capture, all decoded in parallel. Every tap
shares one antenna, one centre frequency and one gain, and the channelizer is **gain-flat
across taps**. So if one site decodes cleanly while others sit at the noise floor, the cause is
RF, not the DDC:

- **Front-end overload.** A strong (often hilltop) site can drive the shared ADC into
  clipping, raising the noise floor and burying weaker sites. Gain is in **tenths of a dB**
  (`gain: 600` means 60 dB). If the input clip ratio is non-zero, **lower the gain or add
  attenuation** — do not raise it. In a metro the usual culprit is broadcast FM; an
  [FM broadcast notch filter](/reference/fm-broadcast-filter/) inline is often the cheapest fix.
- **A genuinely weak/distant site** may not survive a capture optimised for a stronger one.
  Give it a dedicated dongle if it matters.

## Choosing a sample rate

The R2 exposes exactly **two native rates: 10 and 2.5 MS/s** (both IQ output rates — the
firmware's rate table is in IQ rates, not raw ADC rates, a distinction that once caused a
half-rate driver regression, [#851](https://github.com/MattCheramie/GopherTrunk/issues/851)).
Counter-intuitively, **2.5 MS/s is the cleaner path**: it is the FPGA's decimate-by-4 of
the 10 MS/s ADC stream, and captures made at the native 10 MS/s rate measured about 10 dB
worse in-channel than the same signal at 2.5 MS/s
([#764](https://github.com/MattCheramie/GopherTrunk/issues/764),
[#771](https://github.com/MattCheramie/GopherTrunk/issues/771)). The R2 is also a
**real-sampling** device — the host converts the bare ADC stream to complex baseband
([#454](https://github.com/MattCheramie/GopherTrunk/issues/454)). The full story, with the
diagnostic recipe, is in [Airspy sample-rate selection](/reference/airspy-rate-selection/).

## Where to buy

Airspy is sold through its own distributor network, so Amazon stock comes and goes — the
button is a tagged search that always resolves to current listings rather than a single page
that may be out of stock. Get the **R2** for the full ~10 MS/s wideband capture, clock output
and bias tee; for most of the quality at a lower price, the
[Airspy Mini](/reference/airspy-mini/) tops out around 6 MS/s.

<a class="btn btn--buy" href="https://www.amazon.com/s?k=Airspy+R2+SDR&tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

Deciding between radios? See [best SDR for GopherTrunk](/best-sdr-for-gophertrunk/), compare
against the [RTL-SDR](/reference/rtl-sdr/) and [HackRF One](/reference/hackrf/), or, for
shortwave/low-VHF, the [Airspy HF+ Discovery](/reference/airspy-hf-plus-discovery/). Then grab
GopherTrunk from the [downloads page](/downloads.html).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost
to you. It never changes what we recommend.*

## Sources

[^wiki]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, for background on Airspy-class high-performance VHF/UHF SDR receivers.
[^airspy]: [Airspy R2](https://airspy.com/airspy-r2/) — Airspy, on the R2's R820T2 front end, 12-bit oversampling architecture, up-to-10 MS/s capture, clock output and bias tee.
