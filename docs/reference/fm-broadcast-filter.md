---
slug: fm-broadcast-filter
title: FM broadcast notch filter
entry_type: hardware
category: rf-front-end
description: "An FM broadcast notch (band-stop) filter rejects the strong 88–108 MHz broadcast band before it reaches an SDR's front end, stopping the overload and raised noise floor that broadcast FM causes on direct-sampling and DDC receivers."
keywords: FM notch filter, FM band stop filter, broadcast FM filter, 88-108 MHz filter, FM trap, SDR front end overload, RTL-SDR FM filter, noise floor, intermodulation, DDC SDR filter
aka: [FM notch filter, FM band-stop filter, FM trap, broadcast FM filter]
autolink: true
affiliate: true
product:
  name: "RTL-SDR Blog Broadcast FM Block Filter (88–108 MHz)"
  brand: RTL-SDR Blog
  category: RF band-stop filter
  price: "24.95"
  url: https://www.amazon.com/dp/B01LE9LRPM?tag=gophertrunk-20
infobox:
  - { label: Type, value: Band-stop (notch) filter }
  - { label: Stop band, value: "~88–108 MHz (broadcast FM)" }
  - { label: Attenuation, value: ">50 dB in band" }
  - { label: Insertion loss, value: "<0.5 dB out of band" }
  - { label: Connectors, value: "SMA (typical)" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B01LE9LRPM?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [rf-filter, low-noise-amplifier, attenuator, bias-tee, dynamic-range, rtl-sdr, airspy, software-defined-radio]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/rf-sdr/sdr-hardware/ }
cite_urls:
  - https://www.rtl-sdr.com/
  - https://en.wikipedia.org/wiki/Band-stop_filter
---

An **FM broadcast notch filter** is a [band-stop filter](/reference/rf-filter/) that
rejects the **88–108 MHz** broadcast-FM band before it reaches a receiver's front end.
On [software-defined radios](/reference/software-defined-radio/) — especially cheap
direct-sampling dongles and wideband DDC receivers — the local broadcast-FM transmitters
are often the strongest signals at the antenna by tens of dB, and left unfiltered they
push the front end into overload.[^rtlsdr]

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Every DDC / direct-sampling SDR wants one.** Broadcast FM (88–108 MHz) is usually the
loudest thing at your antenna; unfiltered it **overloads the front end**, raising the
[noise floor](/reference/dynamic-range/) and generating
[intermodulation](/reference/intermodulation/) products across the band you actually
want. A ~$25 band-stop filter drops broadcast FM by >50 dB with <0.5 dB loss elsewhere —
often the single cheapest way to lower your noise floor.
</div>

## Overview

A wideband receiver shares one [ADC](/reference/analog-to-digital-converter/) and one
front-end gain stage across everything in its capture. Its
[dynamic range](/reference/dynamic-range/) is finite, so the **strongest** signal present
sets the gain the whole capture must live within. In most populated areas that strongest
signal is broadcast FM. When it drives the front end near clipping, two things happen:
the [noise floor](/reference/dynamic-range/) rises (burying weak signals like a distant
[control channel](/reference/control-channel/)), and non-linear mixing produces
[intermodulation](/reference/intermodulation/) products — phantom carriers — elsewhere in
the spectrum.

Because broadcast FM sits in a fixed, well-defined band, a **band-stop filter** tuned to
88–108 MHz removes the offender while leaving the VHF/UHF public-safety and business bands
essentially untouched. This is distinct from a [broadcast-FM band-pass](/reference/rf-filter/)
(which keeps only FM) — a notch keeps everything *except* FM.

## How it works

The filter is a passive LC (or SAW) network with a deep rejection null across ~88–108 MHz
and low [insertion loss](/reference/rf-filter/) on either side. Typical numbers for the
common SDR-oriented parts: **>50 dB** of stop-band attenuation, **<0.5 dB** insertion loss
outside the notch, on SMA connectors. Good designs keep the adjacent VHF airband
(108–137 MHz) minimally affected. Install it inline between the antenna (or the
[LNA](/reference/low-noise-amplifier/)) and the SDR; being passive and bidirectional, it
needs no power.

## Relevance to SDR

GopherTrunk decodes weak [P25](/reference/project-25/), [DMR](/reference/dmr/) and
[TETRA](/reference/tetra/) [control channels](/reference/control-channel/) that a raised
noise floor can bury, and the wideband `role: wideband` path is *especially* sensitive
because one overloaded capture starves every tap on it — the same front-end-overload
failure the [Airspy](/reference/airspy/) wideband notes describe (`clip_ratio` non-zero,
weak sites at the noise floor). If a wideband dongle shows clipping or a stubbornly high
noise floor in a metro area, an FM notch filter is usually the first and cheapest fix,
ahead of adding an [attenuator](/reference/attenuator/) or lowering gain.

## Where to buy

Common, inexpensive options on Amazon: the
**[RTL-SDR Blog Broadcast FM Block Filter](https://www.amazon.com/dp/B01LE9LRPM?tag=gophertrunk-20)**
(~$25, >50 dB rejection) and the
**[Nooelec Flamingo+ FM](https://www.amazon.com/dp/B07XKY8YKB?tag=gophertrunk-20)** (higher
rejection, low airband impact). Equivalent notch filters are also widely sold on eBay and
AliExpress.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B01LE9LRPM?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

## Sources

[^rtlsdr]: [RTL-SDR.com](https://www.rtl-sdr.com/) — background on broadcast-FM overload of SDR front ends and the use of 88–108 MHz band-stop filters to lower the noise floor and suppress intermodulation on direct-sampling and DDC receivers.
