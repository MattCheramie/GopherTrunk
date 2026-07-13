---
slug: gray-code
title: Gray code
entry_type: algorithm
category: error-correction
description: A Gray code orders values so consecutive ones differ in a single bit; used to map bits to QAM/PSK constellation points so a neighbouring-symbol slip costs only one bit error.
keywords: Gray code, reflected binary code, Gray mapping, constellation mapping, QAM, PSK, single-bit change, rotary encoder, bit error rate, symbol mapping
aka: [Gray code, reflected binary code, Gray mapping]
autolink: true
infobox:
  - { label: Type, value: Binary ordering / bit mapping }
  - { label: Property, value: Adjacent values differ by 1 bit }
  - { label: Used by, value: QAM/PSK mapping, rotary encoders }
see_also: [quadrature-amplitude-modulation, phase-shift-keying, constellation-diagram, forward-error-correction, pi-4-dqpsk, symbol-rate]
cite_urls:
  - https://en.wikipedia.org/wiki/Gray_code
  - https://en.wikipedia.org/wiki/Quadrature_amplitude_modulation
---

A **Gray code** (or *reflected binary code*) is an ordering of binary values in which
**each value differs from the next in exactly one bit**.[^wiki] In digital radio it is
the standard rule for mapping bits onto
[constellation](/reference/constellation-diagram/) points: adjacent points in a
[QAM](/reference/quadrature-amplitude-modulation/) or
[PSK](/reference/phase-shift-keying/) diagram are labelled so that neighbours differ by
a single bit. The result is that the *most common* demodulation mistake — sliding to an
adjacent symbol — corrupts only **one** bit instead of several, directly lowering the
bit error rate.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="An eight-point PSK constellation on a circle with three-bit Gray labels, where every neighbouring point differs in exactly one bit, so slipping to an adjacent symbol causes a single bit error." xmlns="http://www.w3.org/2000/svg">
  <circle cx="150" cy="90" r="70" fill="none" stroke="currentColor" stroke-width="0.7" stroke-opacity="0.5"/>
  <g fill="currentColor"><circle cx="220" cy="90" r="3.5"/><circle cx="199" cy="41" r="3.5"/><circle cx="150" cy="20" r="3.5"/><circle cx="101" cy="41" r="3.5"/><circle cx="80" cy="90" r="3.5"/><circle cx="101" cy="139" r="3.5"/><circle cx="150" cy="160" r="3.5"/><circle cx="199" cy="139" r="3.5"/></g>
  <g font-size="10" fill="currentColor" font-family="monospace">
    <text x="228" y="93">000</text><text x="205" y="34">001</text><text x="136" y="14">011</text><text x="66" y="34">010</text>
    <text x="44" y="93">110</text><text x="70" y="152">111</text><text x="136" y="176">101</text><text x="205" y="152">100</text>
  </g>
  <path d="M212 78 A70 70 0 0 0 207 50" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="300" y="70" font-size="9" fill="currentColor">neighbours 000 ↔ 001</text>
  <text x="300" y="84" font-size="9" fill="currentColor">differ in 1 bit</text>
  <text x="300" y="106" font-size="9" fill="currentColor">→ an adjacent-symbol</text>
  <text x="300" y="120" font-size="9" fill="currentColor">slip costs 1 bit error</text>
</svg>
<figcaption>Gray-labelled 8-PSK: every adjacent constellation point differs by one bit, so the dominant error — a slip to a neighbouring symbol — flips only one bit.</figcaption>
</figure>

## How it works

The construction is the *reflected* rule that gives the code its second name. Build the
`(n+1)`-bit Gray sequence from the `n`-bit one by writing the list, mirroring it below,
prefixing `0` to the top half and `1` to the bottom half. Equivalently, the Gray code
`g` of an ordinary binary number `b` is `g = b XOR (b >> 1)`, and the inverse recovers
`b` by cumulatively XOR-ing the Gray bits from the most significant down. Either way,
stepping from one value to the next toggles a single bit.

Applied to modulation, the constellation points are placed on their grid (for QAM) or
circle (for PSK) and then labelled by walking a Gray sequence so that **horizontal and
vertical neighbours differ in one bit**. Why this matters: at moderate signal-to-noise
ratio, symbol errors are overwhelmingly *nearest-neighbour* errors — noise nudges the
received point just across the decision boundary into the adjacent cell. With a Gray
map that dominant error contributes exactly one bit error; with a naive binary map it
might flip two or three. For a large constellation this pushes the bit-error-rate curve
measurably to the left "for free," since Gray mapping adds no redundancy and no
bandwidth — it only chooses *which* label goes on *which* point.

## In practice

- **QAM and PSK links** — Wi-Fi, LTE, 5G NR, DVB, cable and DSL modems all Gray-map
  their 16-QAM, 64-QAM, 256-QAM (and higher) constellations. Two-dimensional QAM maps
  are Gray-coded independently along the in-phase and quadrature axes.
- **Differential schemes** — systems like [π/4-DQPSK](/reference/pi-4-dqpsk/) carry
  information in *phase transitions* rather than absolute positions, and the transitions
  are Gray-mapped so a misjudged step costs one bit.
- **Rotary encoders and mechanics** — the original engineering use: a Gray-coded
  position disc guarantees that as it rotates past a boundary, only one track changes
  at a time, eliminating the transient glitches a straight binary disc would produce
  when several bits flip together.

Gray coding is a *labelling* choice, not [forward error correction](/reference/forward-error-correction/):
it reduces how many bit errors each symbol slip produces, but it adds no parity and
cannot correct anything on its own. It composes with FEC — the demapper hands
Gray-derived soft bits to a downstream decoder.

## Relevance to SDR

Every multi-level digital-radio waveform a software receiver demodulates relies on
Gray mapping to get from noisy IQ samples back to bits with the fewest errors, so the
**soft-decision bit demapper** in an SDR chain assumes the Gray labelling defined by
each standard. **GopherTrunk** decodes primarily constant-envelope C4FM/CQPSK and
π/4-DQPSK trunking waveforms, and where those carry multi-bit symbols (dibits in P25,
the QPSK-family constellations elsewhere) it must apply the correct Gray-to-bit mapping
when converting demodulated symbols to bits — otherwise a single symbol slip would
scramble several bits and defeat the frame's error-correction coding downstream.

## Sources

[^wiki]: [Gray code](https://en.wikipedia.org/wiki/Gray_code) — Wikipedia, for the reflected-binary construction, the `b XOR (b >> 1)` conversion, single-bit adjacency, and the constellation-mapping and rotary-encoder applications.
