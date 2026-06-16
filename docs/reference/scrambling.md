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
  - { title: "The demodulation pipeline", url: /learn/demodulation-pipeline/ }
external:
  - { title: "Scrambler (Wikipedia)", url: https://en.wikipedia.org/wiki/Scrambler }
---

**Scrambling** (**whitening**) XORs data with a *public* pseudo-random sequence to break
up long runs of identical bits, which helps [clock recovery](/reference/clock-recovery/)
and keeps the spectrum balanced. It is **not** encryption.

## How it works

Both ends know the same sequence, so the receiver de-scrambles by XORing again. This
guarantees frequent transitions for timing and avoids a DC bias, regardless of the data.

## Relevance to SDR

Recognising that whitening is reversible (unlike [RC4](/reference/rc4-cipher/) encryption)
explains why a scrambled-but-unencrypted signal still decodes.
