---
slug: multi-band-excitation
title: Multi-band excitation (MBE)
entry_type: algorithm
category: algorithms
description: Multi-band excitation is a speech-modelling method that splits the spectrum into bands declared voiced or unvoiced; it underlies the IMBE and AMBE vocoder families.
keywords: multi-band excitation, MBE, speech model, voiced unvoiced, IMBE, AMBE, vocoder
aka: [multi-band excitation, MBE]
autolink: true
infobox:
  - { label: Type, value: Speech-coding model }
  - { label: Models, value: Pitch + per-band voicing + spectrum }
  - { label: Underlies, value: IMBE, AMBE, AMBE+2 }
see_also: [vocoder, imbe, ambe, ambe-plus-2, dvsi]
related_lessons:
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/rf-sdr/vocoders/ }
external:
  - { title: "Multi-Band Excitation (Wikipedia)", url: https://en.wikipedia.org/wiki/Multi-Band_Excitation }
---

**Multi-band excitation** (**MBE**) is a speech-modelling method that represents a voice
by its pitch and a decision, per frequency band, of whether that band is *voiced*
(pitched) or *unvoiced* (noisy), plus the spectral envelope.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A speech spectrum divided into frequency bands, each independently declared voiced or unvoiced." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="90" x2="430" y2="90" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-width="1.4"><line x1="80" y1="90" x2="80" y2="38"/><line x1="140" y1="90" x2="140" y2="52"/><line x1="200" y1="90" x2="200" y2="44"/><line x1="260" y1="90" x2="260" y2="62"/><line x1="320" y1="90" x2="320" y2="50"/><line x1="380" y1="90" x2="380" y2="70"/></g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="80" y="104">V</text><text x="140" y="104">V</text><text x="200" y="104">U</text><text x="260" y="104">V</text><text x="320" y="104">U</text><text x="380" y="104">V</text></g>
  <text x="230" y="26" text-anchor="middle" font-size="9" fill="currentColor">voiced (V) / unvoiced (U) decision per band</text>
</svg>
<figcaption>Multi-band excitation models speech as a pitched spectrum with a voiced/unvoiced flag per band — the basis of IMBE/AMBE.</figcaption>
</figure>

## How it works

Transmitting these compact parameters lets the receiver re-synthesise intelligible speech
at a few kbps. MBE underlies the [IMBE](/reference/imbe/) and [AMBE](/reference/ambe/) /
[AMBE+2](/reference/ambe-plus-2/) [vocoders](/reference/vocoder/) from
[DVSI](/reference/dvsi/).

## Relevance to SDR

Understanding MBE explains why decoded digital voice can sound robotic and why bit errors
produce characteristic warbles.
