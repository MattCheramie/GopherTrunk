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
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
external:
  - { title: "Forward error correction (Wikipedia)", url: https://en.wikipedia.org/wiki/Error_correction_code }
---

**Forward error correction** (**FEC**) adds structured redundancy to transmitted data so
the receiver can **correct** errors on its own, without asking for retransmission —
essential for broadcast and one-way radio links.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="Data plus added redundancy is transmitted; a bit is flipped in transit, and the receiver corrects it without retransmission." xmlns="http://www.w3.org/2000/svg">
  <g font-family="monospace" font-size="11" fill="currentColor"><text x="40" y="45">1011</text><text x="100" y="45" fill-opacity="0.6">+ parity</text></g>
  <line x1="180" y1="40" x2="240" y2="40" stroke="currentColor" marker-end="url(#fecar)"/><text x="210" y="32" text-anchor="middle" font-size="8" fill="currentColor">error</text>
  <g font-family="monospace" font-size="11" fill="currentColor"><text x="255" y="45">1<tspan fill="currentColor" font-weight="bold">1</tspan>11</text></g>
  <line x1="320" y1="40" x2="380" y2="40" stroke="currentColor" marker-end="url(#fecar)"/>
  <g font-family="monospace" font-size="11" fill="currentColor"><text x="395" y="45">1011</text></g>
  <text x="230" y="90" text-anchor="middle" font-size="9" fill="currentColor">redundancy lets the receiver fix errors with no retransmission</text>
  <defs><marker id="fecar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Forward error correction adds redundancy so the receiver can repair bit errors on its own.</figcaption>
</figure>

## How it works

Encoders such as [Reed–Solomon](/reference/reed-solomon-code/),
[BCH](/reference/bch-code/), [Golay](/reference/golay-code/), and
[convolutional](/reference/convolutional-code/) codes add parity that lets the decoder fix
a bounded number of errors, often aided by [interleaving](/reference/interleaving/).

## Relevance to SDR

FEC is why a digital signal stays perfect until it abruptly fails (the "cliff effect"):
the decoder fixes errors until it can't, after which audio drops out.
