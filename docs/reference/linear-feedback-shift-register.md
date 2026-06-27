---
slug: linear-feedback-shift-register
title: Linear-feedback shift register (LFSR)
entry_type: algorithm
category: cryptography
description: A linear-feedback shift register is a shift register whose next input bit is a linear (XOR) function of selected tap positions, generating long pseudo-random bit sequences used for scrambling and test patterns.
keywords: LFSR, linear feedback shift register, taps, feedback polynomial, PRBS, pseudo-random, scrambling, whitening, stream cipher, Galois, Fibonacci
aka: [LFSR, PRBS generator]
autolink: true
infobox:
  - { label: Type, value: Sequence generator }
  - { label: Feedback, value: XOR of tap bits }
  - { label: Output, value: Pseudo-random bit stream }
see_also: [keystream, stream-cipher, scrambling, one-time-pad, cryptanalysis]
cite_urls:
  - https://en.wikipedia.org/wiki/Linear-feedback_shift_register
---

**A linear-feedback shift register (LFSR)** is a shift register whose incoming bit is a
linear (XOR) function of selected bits, called *taps*, producing a long, repeatable
pseudo-random sequence from a small amount of state.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="Four shift-register cells feeding right, with two taps XORed and fed back to the input." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="120" y="34" width="40" height="26" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="140" y="51">b0</text>
    <rect x="170" y="34" width="40" height="26" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="190" y="51">b1</text>
    <rect x="220" y="34" width="40" height="26" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="240" y="51">b2</text>
    <rect x="270" y="34" width="40" height="26" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="290" y="51">b3</text>
    <line x1="160" y1="47" x2="170" y2="47" stroke="currentColor"/>
    <line x1="210" y1="47" x2="220" y2="47" stroke="currentColor"/>
    <line x1="260" y1="47" x2="270" y2="47" stroke="currentColor"/>
    <circle cx="80" cy="47" r="11" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="80" y="51" font-size="11">⊕</text>
    <line x1="310" y1="47" x2="360" y2="47" stroke="currentColor" marker-end="url(#lfsrar)"/><text x="385" y="51">output</text>
    <path d="M240 60 L240 80 L80 80 L80 58" fill="none" stroke="currentColor" stroke-dasharray="3 2"/>
    <path d="M290 60 L290 88 L80 88 L80 58" fill="none" stroke="currentColor" stroke-dasharray="3 2"/>
    <line x1="80" y1="36" x2="80" y2="20" stroke="currentColor"/><line x1="80" y1="20" x2="120" y2="20" stroke="currentColor"/><line x1="120" y1="20" x2="120" y2="34" stroke="currentColor" marker-end="url(#lfsrar)"/>
  </g>
  <defs><marker id="lfsrar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>An LFSR shifts each clock and XORs its tap bits back into the input, cycling through a long pseudo-random sequence.</figcaption>
</figure>

## How it works

An LFSR holds a few bits of state. On each clock the bits shift one position, the bit
shifted out becomes the output, and a new input bit is computed by XORing the selected tap
positions together. The choice of taps is described by a *feedback polynomial*; a
"maximal-length" polynomial of *n* bits cycles through all 2ⁿ−1 non-zero states before
repeating, giving a long pseudo-random binary sequence (PRBS).

- **Cheap** — a handful of flip-flops and XOR gates; common in hardware.
- **Deterministic** — the same seed and taps always reproduce the same
  [keystream](/reference/keystream/), so a receiver can regenerate it exactly.
- **Linear** — because the feedback is pure XOR, the sequence is predictable: observing a
  modest run of output lets an attacker solve for the taps and state, so an LFSR *alone* is
  not a secure [stream cipher](/reference/stream-cipher/).

## Relevance to SDR

LFSRs are everywhere in digital radio, but as *scrambling* rather than secrecy. Many
trunked and digital-voice protocols [scramble](/reference/scrambling/) (whiten) their bit
stream with a fixed LFSR sequence to remove long runs and DC bias; because the polynomial
and seed are public, GopherTrunk simply regenerates the same sequence and XORs it back out
— no key is involved. That is the opposite of a [one-time pad](/reference/one-time-pad/),
whose key is secret and never reused.

The linearity of an LFSR also makes it a natural first hypothesis when reverse-engineering
an unknown bit transform. In the clean-room talker-alias analysis (issue #773), an
LFSR-style update was tested against captured data and *ruled out* — the observed mapping
was nonlinear, pointing instead at a fixed substitution table rather than a linear feedback
sequence.

## Sources

[^wiki]: [Linear-feedback shift register](https://en.wikipedia.org/wiki/Linear-feedback_shift_register) — Wikipedia, for taps, feedback polynomials, maximal-length sequences, and the linearity weakness.
