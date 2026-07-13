---
slug: upconverter
title: Upconverter
entry_type: hardware
category: rf-front-end
description: An upconverter shifts HF signals up into the tuning range of a VHF/UHF SDR such as an RTL-SDR, enabling shortwave reception on radios that cannot tune HF directly.
keywords: upconverter, HF converter, Ham It Up, shortwave, RTL-SDR HF, frequency shifting, mixer, local oscillator, image frequency
aka: [upconverter]
autolink: true
infobox:
  - { label: Type, value: External RF converter }
  - { label: Function, value: Shifts HF up into VHF tuning range }
  - { label: Enables, value: HF reception on VHF/UHF SDRs }
see_also: [mixer-rf, local-oscillator, rtl-sdr, airspy-hf-plus, frequency-bands, image-frequency]
related_lessons:
  - { title: "Frequency, bands & the spectrum", url: /learn/rf-sdr/frequency-and-spectrum/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Frequency_mixer
  - https://en.wikipedia.org/wiki/Heterodyne
---

An **upconverter** is an external device that **shifts HF signals up** into the tuning
range of a VHF/UHF SDR such as an [RTL-SDR](/reference/rtl-sdr/), letting radios that cannot
tune [HF](/reference/frequency-bands/) directly receive shortwave.[^wiki] It is essentially
a single [mixer](/reference/mixer-rf/) stage in a box: feed in a low-frequency antenna
signal, and it comes out translated to a higher band the dongle *can* reach.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A low HF input mixed with a fixed local oscillator and shifted upward by that offset into the RTL-SDR's tunable range." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="430" y2="70" stroke="currentColor" stroke-opacity="0.4"/>
  <rect x="40" y="55" width="30" height="15" fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="1.1"/><text x="55" y="90" font-size="8" fill="currentColor" text-anchor="middle">HF 7 MHz</text>
  <line x1="80" y1="48" x2="250" y2="48" stroke="currentColor" stroke-dasharray="4 3" marker-end="url(#upar)"/><text x="165" y="42" font-size="8" fill="currentColor" text-anchor="middle">+ LO 125 MHz</text>
  <rect x="255" y="55" width="34" height="15" fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="1.1"/><text x="272" y="90" font-size="8" fill="currentColor" text-anchor="middle">132 MHz</text>
  <text x="370" y="62" font-size="8" fill="currentColor">RTL-SDR range</text>
  <text x="435" y="80" font-size="8" fill="currentColor" text-anchor="end">frequency →</text>
  <defs><marker id="upar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>An upconverter mixes HF up by a fixed offset into the RTL-SDR's tunable range, since the dongle can't reach HF directly.</figcaption>
</figure>

## How it works

An upconverter is a [heterodyne](/reference/mixer-rf/) frequency translator. It mixes the
incoming HF signal with a fixed [local oscillator](/reference/local-oscillator/) — commonly
100 MHz or 125 MHz — in a [mixer](/reference/mixer-rf/), producing sum and difference
products.[^het] The **sum** product (input + LO) is kept: a 7 MHz signal with a 125 MHz LO
appears at 132 MHz, comfortably inside the dongle's tuning window. The whole HF spectrum,
0–30 MHz, translates as a block to 125–155 MHz. In the SDR software you simply tell the app
the LO offset and it subtracts 125 MHz so the display and logs read the true HF frequency.

Because the LO is quartz-referenced, its stability and [phase noise](/reference/phase-noise/)
add directly to whatever the dongle contributes — a poor upconverter LO smears weak signals.
Mixing also creates an [image](/reference/image-frequency/): a signal at LO + 7 MHz and one
at LO − 7 MHz both land at the same output if the front end is not filtered, so a good
upconverter includes a low-pass filter on its HF input to reject anything above ~30 MHz
before mixing. Cheaper units also let the strong FM broadcast band leak through as spurs.

## Variants

- **Passive vs powered LO** — most hobby upconverters (the classic *Ham It Up*, the
  SpyVerter) use a crystal oscillator and are USB- or bias-tee-powered.
- **Switchable / bypass** — better boards let you route VHF/UHF straight through, so you can
  leave the converter inline and only engage it for HF.
- **Different LO choices** — 100 MHz keeps the arithmetic round (subtract 100); 125 MHz
  pushes the FM broadcast image further from the HF output.

## In practice

An upconverter is the **budget route** to HF on an RTL-SDR and works well for casual
shortwave, ham bands, and utility listening. Its limits are the ones you would expect from
adding a mixer ahead of an already-noisy tuner: modest dynamic range, LO phase noise, and
possible images from strong signals. For serious HF work a
[direct-sampling](/reference/direct-sampling/) receiver or a dedicated
[Airspy HF+](/reference/airspy-hf-plus/) — which digitises HF natively with far better
[dynamic range](/reference/dynamic-range/) — is the higher-performance alternative.

## Relevance to SDR

Upconverters are a staple of the RTL-SDR-plus-HF hobby, opening shortwave broadcast, ham
SSB/CW, and digital HF modes to a dongle that otherwise stops around 24 MHz. GopherTrunk
targets trunked land-mobile systems in VHF/UHF and does not itself do HF work, so an
upconverter is not part of a typical GopherTrunk setup; it is included here as core
front-end vocabulary for SDR users whose interests reach below the dongle's native range.

## Sources

[^wiki]: [Frequency mixer](https://en.wikipedia.org/wiki/Frequency_mixer) — Wikipedia, on the mixing principle an upconverter uses to shift HF up into a VHF/UHF tuning range.
[^het]: [Heterodyne](https://en.wikipedia.org/wiki/Heterodyne) — Wikipedia, on producing sum and difference frequencies by mixing with a local oscillator, the basis of frequency translation.
