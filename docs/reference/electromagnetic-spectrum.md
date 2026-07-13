---
slug: electromagnetic-spectrum
title: Electromagnetic spectrum
entry_type: term
category: rf-fundamentals
description: The electromagnetic spectrum is the full range of radiation by frequency — from radio waves through microwaves and visible light to X-rays and gamma rays.
keywords: electromagnetic spectrum, EM spectrum, radio waves, frequency range, light, microwaves, radio spectrum, ITU bands
aka: [electromagnetic spectrum, EM spectrum]
autolink: true
infobox:
  - { label: Type, value: Physical concept }
  - { label: Radio portion, value: ~3 kHz – 300 GHz }
  - { label: Travels at, value: Speed of light }
see_also: [radio-wave, frequency, wavelength, frequency-bands, radio-propagation, antenna]
related_lessons:
  - { title: "What is a radio wave?", url: /learn/rf-sdr/radio-waves/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Electromagnetic_spectrum
  - https://science.nasa.gov/ems/
---

The **electromagnetic spectrum** is the full range of electromagnetic radiation
ordered by [frequency](/reference/frequency/) (or, equivalently,
[wavelength](/reference/wavelength/)). It spans from low-frequency
[radio waves](/reference/radio-wave/) through microwaves, infrared, visible light,
ultraviolet, X-rays, and gamma rays — all the same physical phenomenon, an oscillating
electric and magnetic field propagating through space, differing only in how fast it
vibrates.[^wiki] Every region is one continuous whole; the named bands are human
labels of convenience, not physical boundaries.

<figure class="figure" markdown="0">
<svg viewBox="0 0 480 110" role="img" aria-label="The electromagnetic spectrum from radio waves through microwaves, infrared, visible light, ultraviolet, X-rays and gamma rays, with the radio portion highlighted and frequency increasing to the right." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="30" width="60" height="26" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.2"/>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <rect x="80" y="30" width="60" height="26"/><rect x="140" y="30" width="60" height="26"/>
    <rect x="200" y="30" width="60" height="26"/><rect x="260" y="30" width="60" height="26"/>
    <rect x="320" y="30" width="60" height="26"/><rect x="380" y="30" width="80" height="26"/>
  </g>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <text x="50" y="47">Radio</text><text x="110" y="47">Micro</text><text x="170" y="47">IR</text>
    <text x="230" y="47">Visible</text><text x="290" y="47">UV</text><text x="350" y="47">X-ray</text><text x="420" y="47">Gamma</text>
  </g>
  <text x="20" y="78" font-size="9" fill="currentColor">lower frequency · longer wavelength</text>
  <text x="460" y="78" font-size="9" fill="currentColor" text-anchor="end">higher frequency</text>
  <line x1="20" y1="66" x2="460" y2="66" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#emar)"/>
  <defs><marker id="emar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Radio occupies the low-frequency, long-wavelength end of the same electromagnetic spectrum as light and X-rays.</figcaption>
</figure>

## How it works

Every part of the spectrum is electromagnetic energy travelling at the speed of light
*c* (≈299,792,458 m/s in vacuum); only the frequency — and therefore the wavelength,
since *λ = c / f* — changes. A single physical law, Maxwell's equations, governs the
whole range: a changing electric field creates a magnetic field and vice versa, and the
pair sustains itself as a wave. What differs from one region to the next is the energy
each cycle carries. Photon energy rises with frequency (*E = hf*), so gamma rays are
ionising and dangerous while radio photons are far too weak to break chemical bonds.

Radio occupies the low-frequency end, conventionally about **3 kHz to 300 GHz**. This
range is special for engineering reasons: the oscillations are slow enough that ordinary
electronic circuits — oscillators, amplifiers, and [antennas](/reference/antenna/) — can
generate, radiate, and detect them directly. Above the radio range, into infrared and
visible light, we mostly resort to optics and photonics because no circuit switches fast
enough. The radio portion is itself subdivided by the ITU into named
[frequency bands](/reference/frequency-bands/) — VLF, LF, MF, HF, VHF, UHF, SHF, EHF —
each a decade of frequency (a factor of ten), with characteristic propagation and
antenna sizes.

## In practice

Where a signal sits in the spectrum decides almost everything practical about it:

- **Propagation.** HF (3–30 MHz) waves refract off the ionosphere and can travel
  worldwide; VHF/UHF are largely line-of-sight; microwaves are blocked by terrain and
  attenuated by rain. See [radio propagation](/reference/radio-propagation/).
- **Antenna size.** A resonant antenna is a fraction of a wavelength, so lower
  frequencies demand physically larger antennas — a practical limit at the low end.
- **Available bandwidth.** Higher bands have more absolute spectrum to spare, which is
  why 5G and Wi-Fi keep climbing toward millimetre waves for capacity.
- **Regulation.** The ITU and national regulators (FCC, Ofcom, and others) allocate
  slices of the radio spectrum to services, so a given frequency legally belongs to
  broadcasting, aviation, land-mobile radio, and so on.

Radio itself is a small, crowded strip of an enormous continuum: the visible-light
octave alone spans more frequency than the entire radio range below it.

## Relevance to SDR

Software-defined radios operate strictly within the radio portion of the spectrum,
limited by their tuner and analog-to-digital converter — an RTL-SDR reaches roughly
24 MHz–1.7 GHz, wideband devices like the HackRF up to ~6 GHz. The trunking systems
GopherTrunk decodes (P25, DMR, NXDN, TETRA) live in the VHF and UHF land-mobile bands,
typically 136–174 MHz, 380–520 MHz, and 700–900 MHz. Where a target sits in the
spectrum dictates the antenna, how far it will be heard, and which SDR hardware can
receive it at all.

## Sources

[^wiki]: [Electromagnetic spectrum](https://en.wikipedia.org/wiki/Electromagnetic_spectrum) — Wikipedia, overview of the full range of electromagnetic radiation and its named regions.
[^nasa]: [Introduction to the Electromagnetic Spectrum](https://science.nasa.gov/ems/) — NASA Science, tutorial on the spectrum, photon energy, and how each region interacts with matter.
