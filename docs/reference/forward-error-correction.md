---
slug: forward-error-correction
title: Forward error correction (FEC)
entry_type: term
category: algorithms
description: Forward error correction adds structured redundancy to transmitted data so the receiver can detect and correct errors without retransmission — essential for one-way radio links.
keywords: forward error correction, FEC, redundancy, error correcting code, coding gain
aka: [forward error correction, FEC]
autolink: true
infobox:
  - { label: Type, value: Error-control strategy }
  - { label: Adds, value: Redundancy for correction }
  - { label: Examples, value: Reed–Solomon, BCH, Golay, convolutional }
see_also: [reed-solomon-code, bch-code, golay-code, hamming-code, convolutional-code, interleaving]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/demodulation-pipeline/ }
external:
  - { title: "Forward error correction (Wikipedia)", url: https://en.wikipedia.org/wiki/Error_correction_code }
---

**Forward error correction** (**FEC**) adds structured redundancy to transmitted data so
the receiver can **correct** errors on its own, without asking for retransmission —
essential for broadcast and one-way radio links.

## How it works

Encoders such as [Reed–Solomon](/reference/reed-solomon-code/),
[BCH](/reference/bch-code/), [Golay](/reference/golay-code/), and
[convolutional](/reference/convolutional-code/) codes add parity that lets the decoder fix
a bounded number of errors, often aided by [interleaving](/reference/interleaving/).

## Relevance to SDR

FEC is why a digital signal stays perfect until it abruptly fails (the "cliff effect"):
the decoder fixes errors until it can't, after which audio drops out.
