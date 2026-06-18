---
slug: scrambling
title: Scrambling (whitening)
entry_type: algorithm
category: algorithms
description: Scrambling, or whitening, XORs data with a pseudo-random sequence to remove long runs of identical bits, aiding clock recovery and spectral balance; it is not encryption.
keywords: scrambling, whitening, pseudo-random, PRBS, clock recovery, DC balance, not encryption
aka: [scrambling, whitening]
autolink: true
infobox:
  - { label: Type, value: Data conditioning }
  - { label: Method, value: XOR with pseudo-random sequence }
  - { label: Not, value: Encryption (sequence is public) }
see_also: [clock-recovery, rc4-cipher, forward-error-correction, demodulation]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
external:
  - { title: "Scrambler (Wikipedia)", url: https://en.wikipedia.org/wiki/Scrambler }
---

**Scrambling** (**whitening**) XORs data with a *public* pseudo-random sequence to break
up long runs of identical bits, which helps [clock recovery](/reference/clock-recovery/)
and keeps the spectrum balanced. It is **not** encryption.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A data stream XORed with a pseudo-random sequence to produce a whitened stream with no long runs." xmlns="http://www.w3.org/2000/svg">
  <g font-family="monospace" font-size="11" fill="currentColor"><text x="30" y="50">data</text><text x="30" y="72">PRBS</text></g>
  <circle cx="200" cy="58" r="12" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="200" y="62" text-anchor="middle" font-size="11" fill="currentColor">⊕</text>
  <line x1="110" y1="48" x2="188" y2="55" stroke="currentColor" stroke-width="1"/><line x1="110" y1="70" x2="188" y2="62" stroke="currentColor" stroke-width="1"/>
  <line x1="212" y1="58" x2="290" y2="58" stroke="currentColor" marker-end="url(#scar)"/>
  <text x="350" y="62" font-size="10" fill="currentColor">whitened stream</text>
  <text x="230" y="98" text-anchor="middle" font-size="9" fill="currentColor">balances 1s and 0s for reliable clock recovery</text>
  <defs><marker id="scar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Scrambling (whitening) XORs data with a pseudo-random sequence to avoid long runs of identical bits.</figcaption>
</figure>

## How it works

Both ends know the same sequence, so the receiver de-scrambles by XORing again. This
guarantees frequent transitions for timing and avoids a DC bias, regardless of the data.

## Relevance to SDR

Recognising that whitening is reversible (unlike [RC4](/reference/rc4-cipher/) encryption)
explains why a scrambled-but-unencrypted signal still decodes.
