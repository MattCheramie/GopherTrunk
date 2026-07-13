---
slug: multi-band-excitation
title: Multi-band excitation (MBE)
entry_type: algorithm
category: voice-coding
description: Multi-band excitation is a speech-coding model that declares each harmonic band voiced or unvoiced and transmits spectral amplitudes and pitch; it underlies the IMBE and AMBE vocoder families.
keywords: multi-band excitation, MBE, speech model, voiced unvoiced, spectral amplitudes, pitch, IMBE, AMBE, AMBE+2, vocoder, harmonic model
aka: [multi-band excitation, MBE]
autolink: true
infobox:
  - { label: Type, value: Parametric speech model }
  - { label: Models, value: Pitch + per-band voicing + spectral amplitudes }
  - { label: Underlies, value: IMBE, AMBE, AMBE+2 }
see_also: [imbe, ambe, ambe-plus-2, vocoder, linear-predictive-coding, code-excited-linear-prediction]
related_lessons:
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/rf-sdr/vocoders/ }
related_reading:
  - { title: "SDR Internals, Part 12: Voice coding & vocoders", url: /blog/deep-dives/sdr-internals-12-voice-coding-vocoders/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Multi-Band_Excitation
  - https://en.wikipedia.org/wiki/Vocoder
---

**Multi-band excitation** (**MBE**) is a parametric speech-coding model that represents a
voice frame by its **pitch**, a set of **spectral amplitudes** (the harmonic envelope), and
— its defining idea — a separate **voiced/unvoiced decision for each frequency band**.[^wiki]
Rather than forcing a whole frame to be either pitched or noisy, MBE lets the low harmonics
be voiced while, say, a fricative's high-frequency energy is coded as noise in the same
frame, which is what gives MBE-based [vocoders](/reference/vocoder/) their natural sound at
low bit rates.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A speech spectrum drawn as harmonic amplitude bars divided into frequency bands, each band independently labelled voiced or unvoiced." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="95" x2="430" y2="95" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-width="1.4"><line x1="70" y1="95" x2="70" y2="40"/><line x1="110" y1="95" x2="110" y2="52"/><line x1="150" y1="95" x2="150" y2="46"/><line x1="200" y1="95" x2="200" y2="58"/><line x1="240" y1="95" x2="240" y2="66"/><line x1="300" y1="95" x2="300" y2="54"/><line x1="340" y1="95" x2="340" y2="72"/><line x1="380" y1="95" x2="380" y2="80"/></g>
  <g stroke="currentColor" stroke-opacity="0.35" stroke-dasharray="3 3"><line x1="175" y1="30" x2="175" y2="100"/><line x1="270" y1="30" x2="270" y2="100"/><line x1="360" y1="30" x2="360" y2="100"/></g>
  <g font-size="8.5" fill="currentColor" text-anchor="middle"><text x="105" y="112">V</text><text x="222" y="112">U</text><text x="315" y="112">V</text><text x="395" y="112">U</text></g>
  <text x="230" y="24" text-anchor="middle" font-size="9" fill="currentColor">voiced (V) / unvoiced (U) decision per band, over the harmonic spectrum</text>
</svg>
<figcaption>Multi-band excitation models a frame as a pitched harmonic spectrum, then flags each band voiced or unvoiced — mixing periodic and noise excitation within one frame.</figcaption>
</figure>

## How it works

For each ~20 ms frame the analyser estimates:

1. **Pitch (fundamental frequency).** This sets the harmonic grid — the frequencies at which
   the spectrum is sampled — and the number of harmonics in the band.
2. **Spectral amplitudes.** The magnitude of the speech spectrum at each harmonic, forming
   the envelope that shapes the synthesised voice.
3. **Per-band voicing.** The harmonics are grouped into bands, and each band is declared
   **voiced** (regenerate it from a pitched sinusoid) or **unvoiced** (fill it with shaped
   noise). This binary map is the "multi-band excitation."

The synthesiser rebuilds the frame by summing sinusoids at the voiced harmonics and adding
band-limited noise where bands were flagged unvoiced, weighted by the transmitted
amplitudes. Because the parameters are compact — pitch, an amplitude vector, and a bit map —
the whole frame fits in a few dozen bits, letting intelligible speech ride in a few kbps.

This is a fundamentally different strategy from the two other major vocoder families:

- **Versus [linear predictive coding](/reference/linear-predictive-coding/) (LPC).** Classic
  LPC vocoders model the vocal tract as an all-pole filter and drive it with a *single*
  voiced-or-unvoiced excitation per frame; the harsh, buzzy artefacts of low-rate LPC come
  largely from that one-bit global voicing decision. MBE's per-band decision is exactly what
  removes that limitation.
- **Versus [CELP](/reference/code-excited-linear-prediction/).** CELP keeps the LPC vocal-
  tract filter but searches a codebook for the best excitation waveform; MBE instead works
  in the frequency domain with an explicit harmonic-plus-noise description.

## Variants and lineage

MBE was introduced by Griffin and Lim (1988). The lineage that reached radio runs through
Digital Voice Systems Inc. ([DVSI](/reference/dvsi/)):

- **[IMBE](/reference/imbe/)** — Improved MBE, the 7.2 kbit/s (with FEC) vocoder adopted for
  APCO [Project 25](/reference/project-25/) Phase 1 voice.
- **[AMBE](/reference/ambe/)** — Advanced MBE, a more efficient successor used across DMR,
  D-STAR, and many satellite phones.
- **[AMBE+2](/reference/ambe-plus-2/)** — the current generation, used in P25 Phase 2, NXDN,
  and DMR at rates down to ~2 kbit/s of voice plus FEC.

All share the MBE core: pitch, spectral amplitudes, and per-band voicing, refined with
better quantisation, prediction between frames, and forward error correction.

## Relevance to SDR

MBE-family vocoders carry the actual voice payload of nearly every digital land-mobile
system a scanner meets — P25 (IMBE / AMBE+2), DMR and NXDN (AMBE+2), D-STAR (AMBE). Knowing
the model explains the characteristic sound of decoded digital voice: robotic timbre from
the harmonic re-synthesis, and warbling artefacts when bit errors corrupt the voicing map or
amplitude bits. GopherTrunk demodulates and frames these signals and can extract the coded
voice bits; producing audible speech requires an MBE-family decoder, and the AMBE/IMBE codecs
themselves are patented and proprietary to DVSI, so GopherTrunk does not ship a built-in
vocoder for them.

## Sources

[^wiki]: [Multi-Band Excitation](https://en.wikipedia.org/wiki/Multi-Band_Excitation) — Wikipedia, for the harmonic speech model with per-band voiced/unvoiced decisions that underlies IMBE and AMBE.
