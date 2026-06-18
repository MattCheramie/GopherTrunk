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
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/rf-sdr/vocoders/ }
external:
  - { title: "Codec 2 (Wikipedia)", url: https://en.wikipedia.org/wiki/Codec_2 }
---

**Codec 2** is an open-source, **royalty-free** low-bitrate speech
[vocoder](/reference/vocoder/) created by David Rowe. It provides intelligible voice
from roughly 700 bps to 3200 bps and is the patent-free alternative to
[AMBE](/reference/ambe/)-family codecs.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A royalty-free Codec 2 frame shown at a very low bit rate compared with a larger uncompressed audio block." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="40" width="200" height="34" fill="none" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.5"/><text x="140" y="61" text-anchor="middle" font-size="9" fill="currentColor">raw audio (large)</text>
  <line x1="250" y1="57" x2="290" y2="57" stroke="currentColor" marker-end="url(#c2ar)"/>
  <rect x="300" y="44" width="56" height="26" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.2"/><text x="328" y="61" text-anchor="middle" font-size="8" fill="currentColor">~1.2–3.2 kbps</text>
  <text x="400" y="61" font-size="9" fill="currentColor">open</text>
  <defs><marker id="c2ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Codec 2 is an open, royalty-free low-bitrate speech codec — the vocoder used by M17.</figcaption>
</figure>

## How it works

Like other vocoders it models speech with pitch and spectral parameters, but its
open licence lets anyone implement it freely — the reason [M17](/reference/m17/) and
the FreeDV digital-voice mode adopted it.

## Relevance to SDR

Codec 2 lets fully open decoders (including M17 support) render digital voice without
proprietary vocoder licensing.
