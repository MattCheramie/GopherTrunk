---
slug: andrew-viterbi
title: Andrew Viterbi
entry_type: person
category: people
description: Andrew Viterbi (b. 1935) is an American engineer who devised the Viterbi algorithm for decoding convolutional codes and co-founded Qualcomm.
keywords: Andrew Viterbi, Viterbi algorithm, convolutional codes, Qualcomm, CDMA
aka: [Andrew Viterbi]
autolink: true
infobox:
  - { label: Born, value: "1935" }
  - { label: Field, value: Electrical engineering }
  - { label: Known for, value: Viterbi algorithm; co-founding Qualcomm }
see_also: [viterbi-algorithm, convolutional-code, forward-error-correction]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Andrew_Viterbi
---

**Andrew Viterbi** (born 1935) is an American electrical engineer who devised the
**[Viterbi algorithm](/reference/viterbi-algorithm/)** for maximum-likelihood decoding of
[convolutional codes](/reference/convolutional-code/), and co-founded Qualcomm.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A trellis with one highlighted most-likely path, the Viterbi algorithm." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor"><circle cx="40" cy="40" r="3"/><circle cx="40" cy="95" r="3"/><circle cx="160" cy="40" r="3"/><circle cx="160" cy="95" r="3"/><circle cx="280" cy="40" r="3"/><circle cx="280" cy="95" r="3"/><circle cx="400" cy="40" r="3"/><circle cx="400" cy="95" r="3"/></g>
  <g stroke="currentColor" stroke-opacity="0.3"><line x1="40" y1="40" x2="160" y2="40"/><line x1="40" y1="40" x2="160" y2="95"/><line x1="40" y1="95" x2="160" y2="40"/><line x1="40" y1="95" x2="160" y2="95"/><line x1="160" y1="40" x2="280" y2="40"/><line x1="160" y1="95" x2="280" y2="40"/><line x1="160" y1="95" x2="280" y2="95"/><line x1="280" y1="40" x2="400" y2="95"/><line x1="280" y1="95" x2="400" y2="95"/></g>
  <polyline points="40,95 160,40 280,40 400,95" fill="none" stroke="currentColor" stroke-width="2.4"/>
  <text x="220" y="122" text-anchor="middle" font-size="9" fill="currentColor">the Viterbi algorithm</text>
</svg>
<figcaption>Viterbi devised the maximum-likelihood decoding algorithm now used to decode convolutional codes everywhere.</figcaption>
</figure>

## Life and work

Viterbi introduced his algorithm in 1967 as a practical way to decode convolutional
codes; it became foundational to digital communications and storage.[^wiki]

## Contribution

His algorithm makes strong [forward error correction](/reference/forward-error-correction/)
practical, improving reception at low SNR.

## Legacy

Viterbi decoding appears throughout modern radio, from amateur [M17](/reference/m17/) to
cellular systems.

## Sources

[^wiki]: [Andrew Viterbi](https://en.wikipedia.org/wiki/Andrew_Viterbi) — Wikipedia, for biography and his algorithm for decoding convolutional codes.
