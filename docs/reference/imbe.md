---
slug: imbe
title: IMBE
entry_type: technology
category: voice-coding
description: IMBE (Improved Multi-Band Excitation) is the vocoder used by P25 Phase 1, encoding voice at about 7.2 kbps including error correction.
keywords: IMBE, Improved Multi-Band Excitation, P25 Phase 1, vocoder, DVSI, 7200 bps, 4400 bps, multi-band excitation
aka: [IMBE]
autolink: true
infobox:
  - { label: Type, value: Speech vocoder (MBE family) }
  - { label: Bit rate, value: ~7.2 kbps (incl. FEC) }
  - { label: Used by, value: P25 Phase 1 }
  - { label: Licensor, value: DVSI }
see_also: [vocoder, ambe, ambe-plus-2, multi-band-excitation, p25-phase-1, dvsi, forward-error-correction, twelp]
related_lessons:
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/rf-sdr/vocoders/ }
related_reading:
  - { title: "SDR Internals, Part 12: Voice coding & vocoders", url: /blog/deep-dives/sdr-internals-12-voice-coding-vocoders/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Multi-Band_Excitation
  - https://en.wikipedia.org/wiki/Project_25
---

**IMBE** (**Improved Multi-Band Excitation**) is the [vocoder](/reference/vocoder/) of
[P25 Phase 1](/reference/p25-phase-1/), part of the
[MBE](/reference/multi-band-excitation/) codec family from [DVSI](/reference/dvsi/).[^wiki]
It was developed in the early 1990s (a refinement of the MBE coder first flown on the
Inmarsat-M satphone system) and was adopted by the APCO Project 25 standard as its sole
voice codec, so every clear P25 conventional and Phase 1 trunked call you hear is IMBE.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A speech spectrum split into frequency bands, each marked voiced or unvoiced, as in multi-band excitation." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="90" x2="430" y2="90" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-width="1.2"><line x1="90" y1="90" x2="90" y2="35"/><line x1="150" y1="90" x2="150" y2="50"/><line x1="210" y1="90" x2="210" y2="42"/><line x1="270" y1="90" x2="270" y2="60"/><line x1="330" y1="90" x2="330" y2="48"/></g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="90" y="105">V</text><text x="150" y="105">V</text><text x="210" y="105">U</text><text x="270" y="105">V</text><text x="330" y="105">U</text></g>
  <text x="230" y="24" text-anchor="middle" font-size="9" fill="currentColor">each band marked voiced (V) or unvoiced (U)</text>
</svg>
<figcaption>IMBE (used by P25 Phase 1) is a multi-band excitation codec that labels each spectral band voiced or unvoiced.</figcaption>
</figure>

## How it works

IMBE analyzes each 20 ms speech frame and estimates a **fundamental (pitch) frequency**,
the number of harmonics that fit in the speech band, a **voiced/unvoiced decision for each
group of harmonics**, and the **spectral magnitude** at every harmonic. The "multi-band"
insight is that a single frame is rarely all-voiced or all-noise: a voiced vowel can have
noisy high-frequency energy, so labeling each sub-band independently reproduces natural
speech far better than a single voicing flag. The decoder synthesizes voiced bands as a
sum of sinusoids at harmonics of the pitch and unvoiced bands as filtered noise, then adds
them.

The P25 IMBE frame is **88 bits of vocoder parameters every 20 ms (4400 bps)**, wrapped in
[forward error correction](/reference/forward-error-correction/) to a total of **7.2 kbps**
over the air. The FEC is unequal: the perceptually critical bits (pitch, gain, the most
significant magnitude bits) get strong Golay coding while less critical bits get lighter
protection, because a flipped pitch bit ruins a frame while a flipped low-order magnitude
bit is barely audible. This is why IMBE degrades gracefully into a warble rather than
silence as the signal weakens.

## Variants

IMBE is a member of the [Multi-Band Excitation](/reference/multi-band-excitation/) lineage
that also includes [AMBE](/reference/ambe/) and [AMBE+2](/reference/ambe-plus-2/). The
"7200x4400" designation (7.2 kbps gross, 4.4 kbps net) identifies the exact P25 variant;
an earlier 6.4 kbps IMBE was used by Inmarsat. IMBE is *not* interoperable with the later
AMBE codecs — a P25 Phase 1 radio and a Phase 2 radio use different vocoders and cannot
directly exchange audio without transcoding.

## In practice

IMBE is proprietary; [DVSI](/reference/dvsi/) held the patents and licensed hardware DSP
chips (the AMBE-x000 series) that early scanners and radios used. The core patents have
since expired, which is what allows independent software implementations. Compared with its
successor AMBE+2 at the same bit rate, IMBE is somewhat less efficient and can sound
slightly rougher, which is one reason P25 Phase 2 moved to AMBE+2.

## Relevance to SDR

GopherTrunk decodes IMBE frames from [P25 Phase 1](/reference/p25-phase-1/) in pure Go and
synthesizes audio directly, with no DVSI dongle. It handles the FEC layers, extracts the
88-bit parameter set, and runs the harmonic synthesizer to produce clear voice; encrypted
P25 (DES or AES) still yields IMBE frames, but scrambled ones that cannot be resynthesized
without the key. Its successor, [AMBE+2](/reference/ambe-plus-2/), covers the newer
systems.

## Sources

[^wiki]: [Multi-Band Excitation](https://en.wikipedia.org/wiki/Multi-Band_Excitation) — Wikipedia, on the MBE vocoder family that includes IMBE.
[^p25]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, for IMBE's role as the P25 Phase 1 voice codec at 7.2 kbps.
