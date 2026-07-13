---
slug: frequency-bands
title: Frequency bands (HF/VHF/UHF)
entry_type: term
category: rf-fundamentals
description: Frequency bands are conventional divisions of the radio spectrum — VLF, LF, MF, HF, VHF, UHF, SHF — each with characteristic propagation and uses.
keywords: frequency bands, HF, VHF, UHF, SHF, band plan, spectrum allocation, ITU bands
aka: [VHF, UHF, SHF]
autolink: true
infobox:
  - { label: Type, value: Spectrum divisions }
  - { label: HF, value: "3–30 MHz" }
  - { label: VHF, value: "30–300 MHz" }
  - { label: UHF, value: "300 MHz – 3 GHz" }
see_also: [electromagnetic-spectrum, frequency, radio-propagation, ionospheric-propagation, radio-horizon, sky-wave, path-loss]
related_lessons:
  - { title: "Frequency, bands & the spectrum", url: /learn/rf-sdr/frequency-and-spectrum/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Radio_spectrum
  - https://en.wikipedia.org/wiki/International_Telecommunication_Union
---

**Frequency bands** are the conventional decade-wide divisions of the radio
[spectrum](/reference/electromagnetic-spectrum/): VLF, LF, MF, **HF** (3–30 MHz),
**VHF** (30–300 MHz), **UHF** (300 MHz–3 GHz), and SHF (3–30 GHz).[^wiki] Each decade
behaves so differently in propagation, antenna size, and available bandwidth that it is
treated as a distinct engineering regime.

<figure class="figure" markdown="0">
<svg viewBox="0 0 480 120" role="img" aria-label="A ruler of radio bands from VLF through SHF with VHF and UHF highlighted as the main scanner bands, annotated with sky-wave propagation at HF and line-of-sight at VHF and above." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.2">
    <rect x="20" y="34" width="55" height="24" fill="none"/>
    <rect x="75" y="34" width="55" height="24" fill="none"/>
    <rect x="130" y="34" width="55" height="24" fill="none"/>
    <rect x="185" y="34" width="55" height="24" fill="none"/>
    <rect x="240" y="34" width="80" height="24" fill="currentColor" fill-opacity="0.22"/>
    <rect x="320" y="34" width="80" height="24" fill="currentColor" fill-opacity="0.22"/>
    <rect x="400" y="34" width="60" height="24" fill="none"/>
  </g>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="47" y="50">VLF</text><text x="102" y="50">LF</text><text x="157" y="50">MF</text><text x="212" y="50">HF</text>
    <text x="280" y="50">VHF</text><text x="360" y="50">UHF</text><text x="430" y="50">SHF</text>
  </g>
  <text x="20" y="80" font-size="9" fill="currentColor">3 kHz</text>
  <text x="460" y="80" font-size="9" fill="currentColor" text-anchor="end">300 GHz</text>
  <text x="130" y="98" text-anchor="middle" font-size="9" fill="currentColor">sky-wave / worldwide</text>
  <text x="360" y="98" text-anchor="middle" font-size="9" fill="currentColor">line-of-sight scanning</text>
</svg>
<figcaption>The radio spectrum divided into bands; VHF and UHF carry most scanner and trunked-radio traffic, while HF relies on ionospheric skip.</figcaption>
</figure>

## How it works

The bands are numbered by order of magnitude, and each decade is centred on a
[wavelength](/reference/wavelength/) that governs its behaviour. Because wavelength = c /
frequency, HF wavelengths are tens of metres, VHF a few metres, and UHF centimetres to a
metre — which directly sets practical [antenna](/reference/antenna/) size (an efficient
antenna is a meaningful fraction of a wavelength).

Propagation is what most distinguishes the bands:

- **HF (3–30 MHz)** refracts off the [ionosphere](/reference/ionospheric-propagation/),
  giving [sky-wave](/reference/sky-wave/) "skip" that can span continents with modest
  power. Conditions vary with the sun, time of day, and the 11-year solar cycle.
- **VHF/UHF (30 MHz–3 GHz)** normally punch through the ionosphere and travel in near
  straight lines, so range is set by the [radio horizon](/reference/radio-horizon/) and
  antenna height. This predictability is why land-mobile, aviation, and public-safety
  systems live here. Occasional tropospheric ducting and sporadic-E can extend VHF far
  beyond the horizon.
- **SHF and above** are strongly line-of-sight, offer huge bandwidth for radar, satellite,
  and 5G, and begin to suffer atmospheric absorption and rain fade.

Higher frequencies also carry more [path loss](/reference/path-loss/) for a given
distance and antenna, and penetrate buildings less, trading range for the wide bandwidth
that high-rate digital systems need.

## In practice

Regulators — coordinated globally by the [ITU](/reference/itu/) and administered
nationally (the FCC in the US, Ofcom in the UK, and others) — publish band plans that
carve each band into slices for broadcast, aviation, marine, amateur, and public-safety
use.[^itu] For a scanner operator these allocations are the map of where to listen:

- **VHF high band (136–174 MHz):** marine, aviation voice nearby, older public-safety and
  business radio.
- **UHF (400–520 MHz):** business, public-safety, and many trunked systems.
- **700/800 MHz:** modern public-safety trunked radio (P25) in North America.

Matching the target allocation to the band determines the whole install: antenna type and
size, feedline loss, and whether the SDR's tuning range even covers it.

## Relevance to SDR

Most trunked-radio scanning happens in VHF, UHF, and the 700/800 MHz bands, so those
decades drive hardware choices. Matching an SDR's tuning range and a resonant
[antenna](/reference/antenna/) to the target band is the first hardware decision, ahead of
any software configuration. GopherTrunk decodes whatever the front end delivers within its
sampled bandwidth; the band determines the antenna, the expected
[path loss](/reference/path-loss/), and which protocols are likely to appear.

## Sources

[^wiki]: [Radio spectrum](https://en.wikipedia.org/wiki/Radio_spectrum) — Wikipedia, the conventional band divisions and their frequency ranges.
[^itu]: [International Telecommunication Union](https://en.wikipedia.org/wiki/International_Telecommunication_Union) — Wikipedia, the body that coordinates global spectrum allocation and band plans.
