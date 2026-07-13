---
slug: saw-filter
title: SAW filter
entry_type: hardware
category: rf-front-end
description: "A SAW (surface acoustic wave) filter is a compact, sharp band-pass filter used in SDR front ends to pass one band (e.g. 1090 MHz ADS-B) and reject out-of-band signals."
keywords: SAW filter, surface acoustic wave, band-pass, front-end filter, 1090 MHz, ADS-B, preselector, piezoelectric, interdigital transducer, IDT
aka: [SAW filter, "surface acoustic wave filter"]
autolink: true
infobox:
  - { label: Type, value: Band-pass RF filter }
  - { label: Principle, value: Acoustic wave on piezoelectric substrate }
  - { label: Used for, value: Front-end preselection (e.g. 1090 MHz) }
see_also: [rf-filter, crystal-filter, cavity-filter, low-noise-amplifier, ads-b, attenuation]
related_lessons:
  - { title: "Antennas 101", url: /learn/rf-sdr/antennas/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Surface_acoustic_wave
  - https://en.wikipedia.org/wiki/Interdigital_transducer
---

A **SAW** (**surface acoustic wave**) filter is a compact, sharp **band-pass**
[filter](/reference/rf-filter/) built on a piezoelectric substrate.[^wiki] In an SDR front
end it acts as a *preselector* — passing only the wanted band and strongly rejecting
out-of-band signals that would otherwise [desensitise](/reference/desensitization/) or
overload the receiver. A single SAW device can give the kind of steep skirts that would
take many stages of an [LC filter](/reference/rf-filter/), in a package a few millimetres
across.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A sharp band-pass response curve passing a narrow band and rejecting everything outside it, with an inset of interdigital transducers on a piezoelectric substrate." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="95" x2="430" y2="95" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="40" y1="20" x2="40" y2="95" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M40 88 L200 88 C 215 88 215 30 235 30 L 245 30 C 265 30 265 88 280 88 L 430 88" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.8"/>
  <text x="240" y="24" text-anchor="middle" font-size="9" fill="currentColor">passband (e.g. 1090 MHz)</text>
  <text x="110" y="80" font-size="8" fill="currentColor">rejected</text><text x="360" y="80" font-size="8" fill="currentColor">rejected</text>
  <text x="435" y="99" font-size="8" fill="currentColor" text-anchor="end">frequency →</text>
</svg>
<figcaption>A SAW filter passes one narrow band with steep skirts, protecting the receiver from strong out-of-band signals.</figcaption>
</figure>

## How it works

A SAW filter turns the electrical signal into a mechanical wave and back again. Two
**interdigital transducers** (IDTs) — interleaved comb-shaped metal electrodes — are
deposited on a piezoelectric crystal such as lithium niobate or quartz.[^idt] Applying RF
to the input IDT flexes the substrate and launches a **surface acoustic wave** that ripples
across the chip at roughly 3,000–4,000 m/s — about a hundred-thousandth the speed of the
electromagnetic wave. The output IDT converts that mechanical wave back to electricity.

Because the wave is so slow, one acoustic wavelength is tiny, and the geometry of the
electrode fingers sets the response with great precision. The finger spacing fixes the
centre frequency; the number of fingers and their weighting sets the bandwidth and skirt
steepness. The result is a fixed-frequency band-pass with excellent selectivity and no
tuning. The trade-off is **insertion loss** — a passive SAW typically costs a few dB, which
is why a receive SAW is usually followed immediately by a
[low-noise amplifier](/reference/low-noise-amplifier/) to restore the
[noise budget](/reference/noise-figure/).

## Variants

- **Band-pass SAW** — the common receive preselector (ADS-B 1090 MHz, GPS L1 1575 MHz,
  ISM bands).
- **SAW resonator / oscillator** — a one-port device used to stabilise oscillators, close
  cousin of the [crystal filter](/reference/crystal-filter/).
- **SAW delay line** — exploits the slow acoustic velocity to build compact delays for
  radar and correlation receivers.
- For applications needing far higher rejection or power handling than a SAW can offer, a
  [cavity filter](/reference/cavity-filter/) is the bulkier alternative.

## In practice

The canonical hobby use is the **1090 MHz ADS-B** chain: a SAW plus LNA "green" filter
board sits between the antenna and an [RTL-SDR](/reference/rtl-sdr/) so that nearby cellular
(700–900 MHz, 1800 MHz) and broadcast transmitters cannot swamp the tuner. Similar filtered
front ends exist for the 137 MHz weather-satellite band and for GPS. Because a SAW is
fixed-frequency, it only helps the one service it was cut for — it is preselection, not a
tunable filter.

## Relevance to SDR

SAW-based filter-and-amp modules are ubiquitous in SDR reception of ADS-B, ACARS, weather
satellites, and GNSS, precisely because broadband dongles have weak inherent selectivity
and overload easily. GopherTrunk decodes the resulting samples but does not manage the
filter; on a trunking site crowded with strong pager, cellular, and broadcast signals, a
band-appropriate SAW or [cavity filter](/reference/cavity-filter/) ahead of the SDR is often
what turns an unusable, intermod-riddled spectrum into a clean control channel GopherTrunk
can lock.

## Sources

[^wiki]: [Surface acoustic wave](https://en.wikipedia.org/wiki/Surface_acoustic_wave) — Wikipedia, on the surface-acoustic-wave devices used to build compact sharp band-pass filters.
[^idt]: [Interdigital transducer](https://en.wikipedia.org/wiki/Interdigital_transducer) — Wikipedia, on the comb electrodes that launch and receive the acoustic wave and set a SAW filter's response.
