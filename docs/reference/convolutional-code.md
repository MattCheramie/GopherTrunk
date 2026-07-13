---
slug: convolutional-code
title: Convolutional code
entry_type: algorithm
category: error-correction
description: A convolutional code is a forward-error-correction code that encodes data as a sliding function of recent input bits through a shift register, typically decoded with the Viterbi algorithm.
keywords: convolutional code, FEC, constraint length, code rate, generator polynomial, shift register, puncturing, Viterbi, trellis, M17
aka: [convolutional code, convolutional coding]
autolink: true
infobox:
  - { label: Type, value: Error-correction code }
  - { label: Encodes, value: Sliding window of input bits }
  - { label: Decoded by, value: Viterbi algorithm }
see_also: [viterbi-algorithm, puncturing, turbo-code, trellis-coded-modulation, forward-error-correction, m17]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Convolutional_code
  - https://en.wikipedia.org/wiki/Error_correction_code
---

A **convolutional code** is a [forward-error-correction](/reference/forward-error-correction/)
code in which each output bit depends not on one input bit but on a sliding window of the
most recent input bits.[^wiki] Unlike a block code, which chops data into independent
codewords, a convolutional encoder produces a continuous coded stream whose redundancy is
spread across time — which is exactly what lets a decoder like the
[Viterbi algorithm](/reference/viterbi-algorithm/) exploit context to correct errors.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A shift register of memory cells whose taps are XOR-combined to produce two output bits per input bit." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.3" fill="none"><rect x="60" y="40" width="40" height="30"/><rect x="110" y="40" width="40" height="30"/><rect x="160" y="40" width="40" height="30"/></g>
  <text x="35" y="59" font-size="9" fill="currentColor">in</text><line x1="44" y1="55" x2="59" y2="55" stroke="currentColor"/>
  <circle cx="260" cy="35" r="10" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="260" y="39" text-anchor="middle" font-size="10" fill="currentColor">⊕</text>
  <circle cx="260" cy="95" r="10" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="260" y="99" text-anchor="middle" font-size="10" fill="currentColor">⊕</text>
  <g stroke="currentColor" stroke-width="1" stroke-opacity="0.7"><line x1="80" y1="40" x2="80" y2="35" /><line x1="80" y1="35" x2="250" y2="35"/><line x1="180" y1="40" x2="180" y2="35"/><line x1="130" y1="70" x2="130" y2="95"/><line x1="130" y1="95" x2="250" y2="95"/><line x1="180" y1="70" x2="180" y2="95"/></g>
  <line x1="270" y1="35" x2="320" y2="35" stroke="currentColor"/><text x="330" y="39" font-size="9" fill="currentColor">out1</text>
  <line x1="270" y1="95" x2="320" y2="95" stroke="currentColor"/><text x="330" y="99" font-size="9" fill="currentColor">out2</text>
</svg>
<figcaption>A convolutional code outputs parity bits computed from the current and recent input bits via a shift register.</figcaption>
</figure>

## How it works

The encoder is a **shift register**: input bits march through a chain of memory cells, and
after each shift the encoder XORs together a fixed pattern of taps to form its output bits.
A few parameters fully describe it:

- **Constraint length (K)** — the number of input bits that influence each output,
  i.e. the register length plus one. Larger K means more error-correcting power but an
  exponentially larger decoding trellis (2^(K−1) states).
- **Code rate (k/n)** — input bits per output bits. A rate-1/2 encoder emits two bits per
  input bit; rate 1/3 emits three, adding more protection at the cost of bandwidth.
- **Generator polynomials** — the tap patterns, one per output, conventionally written in
  octal (the classic K=7, rate-1/2 code uses generators 171 and 133).

Because the output is a running (convolution-like) function of the input, the code has a
*trellis* structure, and the receiver finds the most likely transmitted path through that
trellis with the [Viterbi algorithm](/reference/viterbi-algorithm/), recovering the bits
even when several are wrong.

## Variants

- **Puncturing** — a base rate-1/2 code can be turned into rate 2/3, 3/4, and so on by
  deliberately *not transmitting* some coded bits according to a fixed
  [puncturing](/reference/puncturing/) pattern; the decoder inserts erasures where the bits
  were dropped. This lets one encoder/decoder serve many rates.
- **Recursive systematic convolutional (RSC) codes** — feed part of the output back into
  the register and transmit the raw data bits alongside the parity. Two RSC codes joined by
  an interleaver form a [turbo code](/reference/turbo-code/), whose iterative decoding
  approaches the Shannon limit.
- **Trellis-coded modulation** — combining the convolutional trellis with the modulation
  mapping, [TCM](/reference/trellis-coded-modulation/) wins coding gain without spending
  extra bandwidth.

## Relevance to SDR

Convolutional coding is one of the workhorses of digital radio: it protects GSM, satellite
and deep-space links, 802.11a/g, and, close to the scanner world, [M17](/reference/m17/)
(a K=5 code). GopherTrunk decodes these convolutionally coded fields with a Viterbi decoder
to lower the bit-error rate before framing and payload extraction.

## Sources

[^wiki]: [Convolutional code](https://en.wikipedia.org/wiki/Convolutional_code) — Wikipedia, for the sliding-window encoder, constraint length, code rate, generator polynomials, and puncturing.
