---
slug: richard-hamming
title: Richard Hamming
entry_type: person
category: people
description: Richard Hamming (1915–1998) was an American mathematician who created the first practical error-correcting codes, the Hamming codes, founding the field of coding theory.
keywords: Richard Hamming, Hamming code, error correction, coding theory, Bell Labs, Hamming distance
aka: [Richard Hamming]
autolink: true
infobox:
  - { label: Lived, value: "1915–1998" }
  - { label: Field, value: Mathematics }
  - { label: Known for, value: Hamming codes, Hamming distance }
see_also: [hamming-code, forward-error-correction, golay-code]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
external:
  - { title: "Richard Hamming (Wikipedia)", url: https://en.wikipedia.org/wiki/Richard_Hamming }
---

**Richard Hamming** (1915–1998) was an American mathematician who created the first
practical **error-correcting codes** — the [Hamming codes](/reference/hamming-code/) —
launching the field of coding theory.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="Data bits with interspersed parity bits that detect and correct a single error." xmlns="http://www.w3.org/2000/svg">
  <g font-family="monospace" font-size="11" fill="currentColor" text-anchor="middle"><text x="60" y="50">P</text><text x="100" y="50">P</text><text x="140" y="50">D</text><text x="180" y="50">P</text><text x="220" y="50">D</text><text x="260" y="50">D</text><text x="300" y="50">D</text></g>
  <path d="M60 60 q40 16 80 0" fill="none" stroke="currentColor" stroke-opacity="0.6"/><path d="M100 64 q60 20 120 0" fill="none" stroke="currentColor" stroke-opacity="0.6"/>
  <text x="230" y="96" text-anchor="middle" font-size="9" fill="currentColor">parity bits locate and fix a single-bit error</text>
</svg>
<figcaption>Hamming created the first practical error-correcting codes, the ancestors of the FEC used in digital radio.</figcaption>
</figure>

## Life and work

Frustrated by computers halting on detected errors, Hamming devised codes that could
*correct* single-bit errors automatically, and defined the "Hamming distance" measuring
how many bits differ between codewords.

## Contribution

His work began the discipline of [forward error correction](/reference/forward-error-correction/)
on which all reliable digital radio depends.

## Legacy

Hamming codes and their descendants protect data across radio, storage, and networking.
