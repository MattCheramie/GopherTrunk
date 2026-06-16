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
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/vocoders/ }
external:
  - { title: "Multi-Band Excitation (Wikipedia)", url: https://en.wikipedia.org/wiki/Multi-Band_Excitation }
---

**Multi-band excitation** (**MBE**) is a speech-modelling method that represents a voice
by its pitch and a decision, per frequency band, of whether that band is *voiced*
(pitched) or *unvoiced* (noisy), plus the spectral envelope.

## How it works

Transmitting these compact parameters lets the receiver re-synthesise intelligible speech
at a few kbps. MBE underlies the [IMBE](/reference/imbe/) and [AMBE](/reference/ambe/) /
[AMBE+2](/reference/ambe-plus-2/) [vocoders](/reference/vocoder/) from
[DVSI](/reference/dvsi/).

## Relevance to SDR

Understanding MBE explains why decoded digital voice can sound robotic and why bit errors
produce characteristic warbles.
