---
slug: vocoder
title: Vocoder
entry_type: technology
category: voice-coding
description: A vocoder is a speech codec that compresses voice to a few kilobits per second by modelling how speech is produced rather than recording the waveform; it underpins all digital voice radio.
keywords: vocoder, voice codec, speech coding, IMBE, AMBE, source filter model, digital voice
aka: [vocoder]
autolink: true
infobox:
  - { label: Type, value: Speech codec }
  - { label: Approach, value: Model speech (pitch + spectrum) }
  - { label: Bit rate, value: A few kbps }
  - { label: Examples, value: IMBE, AMBE+2, Codec 2 }
see_also: [imbe, ambe, ambe-plus-2, codec2, multi-band-excitation]
related_lessons:
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/vocoders/ }
  - { title: "Analog vs. digital voice", url: /learn/digital-voice/ }
external:
  - { title: "Vocoder (Wikipedia)", url: https://en.wikipedia.org/wiki/Vocoder }
  - { title: "GopherTrunk vocoders", url: /vocoders.html }
---

A **vocoder** (voice coder) is a speech codec that compresses voice into a few kilobits
per second by **modelling how speech is produced** — pitch, voicing, and spectral shape
— rather than recording the waveform. It is what makes
[digital voice](/learn/digital-voice/) radio possible.

## How it works

Many times a second the vocoder extracts compact parameters of a short speech segment
and transmits only those; the receiver re-synthesises an audible voice from them. This
is why digital voice can sound slightly robotic, especially on a weak signal.

## Relevance to SDR

Decoding digital voice requires running the **matching** vocoder — [IMBE](/reference/imbe/)
for [P25 Phase 1](/reference/p25-phase-1/), [AMBE+2](/reference/ambe-plus-2/) for DMR and
P25 Phase 2, or [Codec 2](/reference/codec2/) for [M17](/reference/m17/).
