---
slug: codec2
title: Codec 2
entry_type: technology
category: voice-coding
description: Codec 2 is an open-source, royalty-free low-bitrate speech vocoder by David Rowe, used by the M17 protocol and FreeDV as a patent-free alternative to AMBE.
keywords: Codec 2, open source vocoder, David Rowe, M17, FreeDV, royalty-free, low bitrate speech
aka: [Codec 2, Codec2]
autolink: true
infobox:
  - { label: Type, value: Open-source speech vocoder }
  - { label: Author, value: David Rowe }
  - { label: Licensing, value: Royalty-free (LGPL) }
  - { label: Used by, value: M17, FreeDV }
see_also: [vocoder, m17, ambe-plus-2, m17-project]
related_lessons:
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/vocoders/ }
external:
  - { title: "Codec 2 (Wikipedia)", url: https://en.wikipedia.org/wiki/Codec_2 }
---

**Codec 2** is an open-source, **royalty-free** low-bitrate speech
[vocoder](/reference/vocoder/) created by David Rowe. It provides intelligible voice
from roughly 700 bps to 3200 bps and is the patent-free alternative to
[AMBE](/reference/ambe/)-family codecs.

## How it works

Like other vocoders it models speech with pitch and spectral parameters, but its
open licence lets anyone implement it freely — the reason [M17](/reference/m17/) and
the FreeDV digital-voice mode adopted it.

## Relevance to SDR

Codec 2 lets fully open decoders (including M17 support) render digital voice without
proprietary vocoder licensing.
