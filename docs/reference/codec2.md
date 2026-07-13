---
slug: codec2
title: Codec 2
entry_type: technology
category: voice-coding
description: Codec 2 is an open-source, royalty-free low-bitrate speech vocoder by David Rowe, used by the M17 protocol and FreeDV as a patent-free alternative to AMBE.
keywords: Codec 2, open source vocoder, David Rowe, M17, FreeDV, royalty-free, low bitrate speech, harmonic sinusoidal coding, LPC
aka: [Codec 2, Codec2]
autolink: true
infobox:
  - { label: Type, value: Open-source speech vocoder }
  - { label: Author, value: David Rowe }
  - { label: Licensing, value: Royalty-free (LGPL) }
  - { label: Used by, value: M17, FreeDV }
see_also: [vocoder, m17, ambe-plus-2, m17-project, linear-predictive-coding, multi-band-excitation, ralcwi]
related_lessons:
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/rf-sdr/vocoders/ }
related_reading:
  - { title: "SDR Internals, Part 12: Voice coding & vocoders", url: /blog/deep-dives/sdr-internals-12-voice-coding-vocoders/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Codec_2
  - https://www.rowetel.com/?page_id=452
---

**Codec 2** is an open-source, **royalty-free** low-bitrate speech
[vocoder](/reference/vocoder/) created by David Rowe.[^wiki] It provides intelligible voice
from roughly 700 bps to 3200 bps and is the patent-free alternative to
[AMBE](/reference/ambe/)-family codecs — the reason a fully open digital-voice stack is
possible at all.

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

Codec 2 is a **harmonic sinusoidal** coder built on
[linear-predictive-coding](/reference/linear-predictive-coding/). For each ~20–40 ms frame
it estimates a **pitch (fundamental) frequency**, the **energy**, a **voicing decision**,
and the **spectral envelope** — the latter captured as a small set of LPC coefficients that
are converted to line spectral pairs and vector-quantized. The decoder places sinusoids at
harmonics of the pitch, scales each by the sampled envelope, and applies the voicing to
decide how much is periodic versus noise-like. Conceptually it sits close to the
[MBE](/reference/multi-band-excitation/) family, but its parameter set and quantization were
designed independently and, crucially, are unencumbered by patents.

## Variants

Codec 2 ships as a set of fixed-rate **modes** named by their bit rate — 3200, 2400, 1600,
1400, 1300, 1200, 700C — that trade audio quality against bandwidth. The 3200 and 2400
modes sound the most natural; the 700-bps modes are remarkably intelligible for their size
and are used where the channel is extremely tight. [M17](/reference/m17/) uses the 3200-bps
mode for its 9600-baud RF stream. There is also a companion neural-net vocoder (LPCNet) that
Rowe's FreeDV work pairs with Codec 2 parameters for higher quality at very low rates.

## In practice

The open licence (LGPL) is the whole point: unlike [AMBE+2](/reference/ambe-plus-2/), Codec 2
carries no per-unit royalty and no DVSI hardware dongle, so anyone can embed it in firmware
or software freely. That is why the amateur-radio [M17 project](/reference/m17-project/)
and the FreeDV HF digital-voice mode both adopted it, and why it is the natural choice for
hobbyist and open-hardware transceivers. The cost is that its ecosystem is smaller than the
entrenched AMBE world, so it is not found in commercial public-safety radio.

## Relevance to SDR

Codec 2 lets fully open decoders render digital voice without proprietary vocoder
licensing. GopherTrunk supports [M17](/reference/m17/), whose voice payload is Codec 2, so
the same pure-Go philosophy that avoids AMBE dongles applies here with no licensing
asterisk at all — the codec itself is free software. For FreeDV and other Codec 2 modes it
is the reference for what an unencumbered land-mobile-style vocoder can achieve.

## Sources

[^wiki]: [Codec 2](https://en.wikipedia.org/wiki/Codec_2) — Wikipedia, on the open-source royalty-free low-bitrate speech codec.
[^rowe]: [Codec 2](https://www.rowetel.com/?page_id=452) — David Rowe (author), project page describing the harmonic sinusoidal model and the fixed-rate modes from 700 bps to 3200 bps.
