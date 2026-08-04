---
slug: rf-filter
title: RF filter
entry_type: hardware
category: rf-front-end
description: "An RF filter is a frequency-selective network (low-pass, high-pass, band-pass, or notch) that passes wanted signals and rejects others in a radio front end."
keywords: RF filter, low-pass filter, high-pass filter, band-pass filter, notch filter, preselector, front-end selectivity, LPF, HPF, BPF, insertion loss, stopband
aka: [RF filter, "radio-frequency filter", preselector]
autolink: true
affiliate: true
product:
  name: "RTL-SDR Blog Flamingo+ FM broadcast notch filter"
  brand: RTL-SDR Blog
  category: FM broadcast notch filter
  lowPrice: "21"
  highPrice: "29"
  url: https://www.amazon.com/dp/B07XKY8YKB?tag=gophertrunk-20
infobox:
  - { label: Type, value: "Passive/active frequency-selective network" }
  - { label: Responses, value: "Low-pass, high-pass, band-pass, notch" }
  - { label: Key spec, value: "Insertion loss, stopband rejection, shape factor" }
  - { label: TX, value: "Yes (power-rated types)" }
  - { label: Typical price, value: "$2–$50 (module)" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B07XKY8YKB?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [saw-filter, cavity-filter, crystal-filter, helical-filter, digital-filter, low-noise-amplifier]
cite_urls:
  - https://en.wikipedia.org/wiki/Electronic_filter
  - https://en.wikipedia.org/wiki/Band-pass_filter
faq:
  - q: "Which RF filter fixes SDR overload from strong FM stations?"
    a: "A broadcast FM notch (band-stop) filter. The RTL-SDR Blog Flamingo+ FM (around $25) rejects the 88–108 MHz FM band while passing everything else, which stops strong local FM transmitters from overloading a wideband dongle and spraying spurious products across your scanning range. For strong AM broadcast interference the Flamingo AM notch does the same below the medium-wave band."
  - q: "Notch filter or band-pass filter — which do I need?"
    a: "Use a notch (band-stop) filter to kill one strong offender, such as the FM or AM broadcast band, while keeping the rest of the spectrum. Use a band-pass filter when you only care about one service and want to reject everything else — a preselector cut for your target VHF/UHF band. Notches are the common first fix for broadcast-band overload."
  - q: "Will a filter hurt my weak signals?"
    a: "A little. Every filter has some passband insertion loss — a fraction of a dB for a good part. That trade is usually worth it: by restoring dynamic range and lowering intermodulation, a well-chosen preselector recovers far more than the small loss it costs when strong out-of-band signals are present."
  - q: "Does GopherTrunk need an RF filter?"
    a: "GopherTrunk is pure software and contains no physical filter — that job belongs to the analog hardware ahead of the SDR. Its digital filters cannot undo an overloaded ADC, so on a site crowded with strong FM, pager, or cellular signals an analog notch or band-pass filter is often what turns an unusable spectrum into a clean control-channel lock."
---

An **RF filter** is a frequency-selective network that passes signals inside a
desired frequency range and attenuates those outside it, shaping the spectrum
that reaches a receiver or leaves a transmitter.[^wiki] Filters are the front
end's first line of defence: they set *selectivity*, keeping strong out-of-band
energy from overloading the [low-noise amplifier](/reference/low-noise-amplifier/)
and mixer of a [superheterodyne receiver](/reference/superheterodyne-receiver/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Four filter responses on a frequency axis: low-pass passing low frequencies, high-pass passing high frequencies, band-pass passing a middle band, and notch rejecting a narrow band." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="60" x2="130" y2="60" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M30 40 L70 40 C80 40 85 90 100 90 L130 90" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="80" y="115" text-anchor="middle" font-size="9" fill="currentColor">low-pass</text>
  <path d="M150 90 L185 90 C200 90 205 40 215 40 L250 40" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="200" y="115" text-anchor="middle" font-size="9" fill="currentColor">high-pass</text>
  <path d="M270 90 L300 90 C312 90 312 40 322 40 L332 40 C342 40 342 90 354 90 L390 90" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="330" y="115" text-anchor="middle" font-size="9" fill="currentColor">band-pass</text>
  <path d="M400 40 L420 40 C428 40 428 90 434 90 L436 90 C442 90 442 40 450 40" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="428" y="115" text-anchor="middle" font-size="9" fill="currentColor">notch</text>
</svg>
<figcaption>The four canonical RF filter responses, plotted as gain versus frequency.</figcaption>
</figure>

## Overview

Every RF filter is characterised by a few numbers. **Insertion loss** is the
signal it eats inside the passband (fractions of a dB for a good cavity, a couple
of dB for a small ceramic part). **Stopband rejection** is how deeply it kills
unwanted frequencies, in dB. The **shape factor** — the ratio of stopband width
to passband width — measures how steep the skirts are; a low shape factor means a
sharp, nearly rectangular response. **Ripple**, **group delay**, and **power
handling** round out the picture. Sharp skirts, low loss, and low ripple pull in
different directions, so filter design is always a trade among them.

## Variants

- **Low-pass (LPF)** — passes DC up to a cutoff, rejects above it. Used as an
  anti-alias filter ahead of an [analog-to-digital converter](/reference/analog-to-digital-converter/)
  and to suppress transmitter [harmonics](/reference/harmonics/).
- **High-pass (HPF)** — the mirror image, blocking low frequencies. Often paired
  with an LPF (or built as a [diplexer](/reference/diplexer/)) to split bands.
- **Band-pass (BPF)** — passes one band and rejects both above and below. The
  classic *preselector*, implemented with LC sections, ceramic resonators,
  [SAW](/reference/saw-filter/) devices, [helical](/reference/helical-filter/)
  cans, or [cavity](/reference/cavity-filter/) resonators depending on frequency
  and Q.
- **Band-stop / notch** — rejects a narrow slice while passing everything else,
  used to kill a single strong interferer (a nearby pager or FM broadcast
  carrier) or, in high-Q [crystal](/reference/crystal-filter/) form, to shape an
  IF.

Filter *families* — Butterworth (maximally flat), Chebyshev (steeper skirts at the
cost of passband ripple), elliptic/Cauer (steepest, with ripple in both bands),
and Bessel (linear phase, gentle skirts) — describe the mathematical response
shape independent of the physical medium.

## Relevance to SDR

Wideband SDRs are especially vulnerable to out-of-band overload because a
[direct-sampling](/reference/direct-sampling/) or
[zero-IF](/reference/zero-if/) front end presents its whole tuning range to the
first amplifier at once. A strong FM broadcast, TV, or cellular signal well
outside the band of interest can drive the LNA or ADC into
[intermodulation](/reference/intermodulation/) and spray spurious products across
the passband. A well-chosen preselector — an FM broadcast-reject HPF for scanning
above 108 MHz, or a band-pass module for a specific service — restores usable
[dynamic range](/reference/dynamic-range/) and lowers the
[noise floor](/reference/noise-floor/) that matters at the antenna.

GopherTrunk is pure software and does not contain any physical RF filter; that
role belongs to the analog hardware ahead of the SDR. What GopherTrunk *does*
provide is the digital equivalent — [FIR](/reference/fir-filter/) and
[digital](/reference/digital-filter/) filters in its
[digital down-converter](/reference/digital-down-converter/) that channelise and
band-limit the sampled stream. Those digital filters cannot undo damage already
done in the analog front end (an overloaded ADC has already clipped), which is
why a good analog RF filter and clean gain staging remain essential for reliable
trunking decodes. In practice, users running RTL-SDR or Airspy dongles for P25 or
DMR often add an inexpensive band-pass module for the target VHF/UHF band to keep
strong nearby transmitters from desensitising the receiver.

## Where to buy

For the most common SDR problem — a wideband dongle overloaded by strong local FM
broadcast stations — a notch filter is the fix. The **RTL-SDR Blog Flamingo+ FM**
(around $25) rejects the 88–108 MHz FM band while passing everything else. For strong
AM broadcast interference the **Flamingo AM** notch does the same below the
medium-wave band.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B07XKY8YKB?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>
<a class="btn btn--buy" href="https://www.amazon.com/dp/B079CMB44V?tag=gophertrunk-20" rel="nofollow sponsored noopener">AM notch on Amazon &rarr;</a>

For band-pass preselectors and a fuller rundown of front-end filtering, see the
[SDR filters guide](/sdr-filters/).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra
cost to you. It never changes what we recommend.*

## Sources

[^wiki]: [Electronic filter](https://en.wikipedia.org/wiki/Electronic_filter) — Wikipedia, overview of passive and active frequency-selective networks and their responses.
