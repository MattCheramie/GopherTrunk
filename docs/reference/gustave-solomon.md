---
slug: gustave-solomon
title: Gustave Solomon
entry_type: person
category: people
description: "Gustave Solomon (1930–1996) was an American mathematician who co-invented the Reed-Solomon error-correcting code widely used in digital radio and storage."
keywords: Gustave Solomon, Reed-Solomon code, error correction, coding theory, algebraic codes, mathematician
aka: [Gustave Solomon, Gus Solomon]
autolink: true
infobox:
  - { label: Lived, value: "1930–1996" }
  - { label: Field, value: "Mathematics" }
  - { label: Known for, value: "Reed-Solomon codes" }
see_also: [reed-solomon-code, irving-reed, reed-muller-code, forward-error-correction, richard-hamming]
cite_urls:
  - https://en.wikipedia.org/wiki/Gustave_Solomon
  - https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction
---

**Gustave Solomon** (1930–1996) was an American mathematician who, with
[Irving S. Reed](/reference/irving-reed/), co-invented the
**[Reed-Solomon code](/reference/reed-solomon-code/)** in 1960 — an algebraic
error-correcting code that went on to protect data on compact discs, deep-space links,
QR codes, and digital broadcasting.[^wiki][^rs] Reed-Solomon is one of the most widely
deployed forms of [forward error correction](/reference/forward-error-correction/) ever
devised.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A polynomial curve sampled at several points, illustrating that Reed-Solomon encodes data as a polynomial evaluated at many points so a few lost samples can be recovered." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="120" x2="430" y2="120" stroke="currentColor"/>
  <path d="M40 110 Q160 20 260 70 T430 40" fill="none" stroke="currentColor" stroke-width="2"/>
  <g fill="currentColor">
    <circle cx="70" cy="92" r="4"/><circle cx="130" cy="55" r="4"/><circle cx="200" cy="52" r="4"/><circle cx="270" cy="67" r="4"/><circle cx="340" cy="58" r="4"/><circle cx="400" cy="44" r="4"/>
  </g>
  <text x="230" y="145" text-anchor="middle" font-size="10" fill="currentColor">data = one polynomial, sampled at many points</text>
</svg>
<figcaption>Reed-Solomon treats a message as the coefficients of a polynomial and transmits its values at many points; extra samples let a receiver recover the polynomial despite errors.</figcaption>
</figure>

## Life and work

Solomon earned his PhD in mathematics from MIT in 1956 and worked at MIT's Lincoln
Laboratory, where he and Reed collaborated, and later at the Jet Propulsion Laboratory
and in the aerospace industry.[^wiki] A gifted algebraist with wide interests — he was
also an accomplished singer and music theorist — he brought a pure-mathematics
sensibility to the practical problem of protecting digital data. His 1960 paper with
Reed, only a few pages long, defined the code that carries both their names.

## Contribution

The insight behind the **Reed-Solomon code** is to treat a block of data symbols as the
coefficients of a **polynomial over a finite field**, and to transmit that polynomial's
*values* at many points rather than the coefficients themselves.[^rs] A polynomial of
degree *k*−1 is fixed by any *k* of its values, so sending *n* > *k* values builds in
redundancy: even if several transmitted values are corrupted, enough correct ones remain
to reconstruct the original polynomial, and hence the data.

Because the symbols are drawn from a large alphabet (typically 8-bit bytes), the code
excels at **burst errors** — a long run of damaged bits corrupts only a handful of
symbols, which the code can locate and correct. A Reed-Solomon code with 2*t* parity
symbols corrects up to *t* symbol errors, and it is a *maximum-distance-separable* code,
meaning it achieves the best possible error protection for its amount of redundancy.

## Legacy

Solomon and Reed's short paper defined a code whose practical importance grew for
decades. Efficient decoding methods — notably the Berlekamp–Massey algorithm and the
Forney and Chien search procedures — turned the elegant construction into fielded
hardware. Reed-Solomon now stands beside the earlier block codes of
[Richard Hamming](/reference/richard-hamming/) and Reed's own
[Reed-Muller code](/reference/reed-muller-code/) as a cornerstone of coding theory.

## Relevance to SDR

Reed-Solomon coding is ubiquitous wherever digital data crosses an unreliable channel:
optical discs, QR codes, DSL, DVB and ATSC digital television, and deep-space telemetry
all use it. In land-mobile radio it appears inside P25, where Reed-Solomon codes guard
trunking control messages and header words against burst corruption. GopherTrunk decodes
P25 and related systems, so Reed-Solomon decoding is part of its control-channel and
header processing — a direct legacy of Solomon's 1960 work.

## Sources

[^wiki]: [Gustave Solomon](https://en.wikipedia.org/wiki/Gustave_Solomon) — Wikipedia, for biography and his role in the code.
[^rs]: [Reed–Solomon error correction](https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction) — Wikipedia, for the polynomial-evaluation construction and burst-error correction.
