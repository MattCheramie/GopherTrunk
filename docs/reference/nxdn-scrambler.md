---
slug: nxdn-scrambler
title: NXDN Scrambler
entry_type: algorithm
category: cryptography
description: "The NXDN scrambler is the optional privacy mode that XORs a keystream from a 15-bit LFSR, seeded by a 15-bit key, onto the voice/data field. Its 2^15 key space is trivially brute-forceable — and GopherTrunk's LFSR model is synthetic, not yet bit-confirmed against a known-key capture."
keywords: NXDN scrambler, NXDN encryption, 15-bit LFSR, Fibonacci LFSR, keystream, XOR privacy, brute force, 32768 keys, Kenwood scrambler
aka: ["NXDN scrambler", "NXDN encryption", "NXDN privacy"]
autolink: true
infobox:
  - { label: Type, value: additive (XOR) keystream }
  - { label: Generator, value: "15-bit LFSR, x^15 + x^14 + 1" }
  - { label: Key space, value: "2^15 = 32768 (brute-forceable)" }
  - { label: Status, value: "synthetic model, not bit-confirmed" }
see_also: [linear-feedback-shift-register, scrambling, nxdn, nxdn-frame-structure, cyclic-redundancy-check]
cite_urls:
  - https://en.wikipedia.org/wiki/Linear-feedback_shift_register
  - https://en.wikipedia.org/wiki/Scrambler
---

The **NXDN scrambler** is the optional privacy mode NXDN radios expose — the setting sometimes
labelled "encryption" — that obscures the voice or data field by XORing a pseudo-random
keystream over it.[^scr] The keystream comes from a 15-bit
[linear-feedback shift register](/reference/linear-feedback-shift-register/) (LFSR) seeded by a
15-bit **scrambling key** in the range 0 to 32767. It is not cryptography in any meaningful
sense: the key space is only 2¹⁵ = 32768 values, small enough to try exhaustively in a
fraction of a second, so the "privacy" it offers is defeated by brute force alone.[^lfsr]

> **Status — synthetic model, not hardware-confirmed.** GopherTrunk's tap polynomial and
> seed mapping are an internally-consistent *working model* — a maximal-length 15-bit Fibonacci
> LFSR, x¹⁵ + x¹⁴ + 1 — chosen so the scrambler is its own inverse and emits a balanced
> m-sequence of period 32767. They have **not** been confirmed bit-exact against a known-key
> capture. The spec-level facts the tooling relies on are the XOR-symmetric (self-inverse)
> structure and the 15-bit key space; the exact feedback is provisional and should be adjusted
> when a known-key capture is available.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="A 15-stage shift register whose top two stages feed an exclusive-or gate back into the input, generating a keystream bit each step from the register's most significant stage; that keystream bit is exclusive-ored with a data bit to produce the scrambled bit." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1" fill="none">
    <rect x="20" y="34" width="260" height="24"/>
    <line x1="45" y1="34" x2="45" y2="58"/><line x1="70" y1="34" x2="70" y2="58"/>
    <line x1="230" y1="34" x2="230" y2="58"/><line x1="255" y1="34" x2="255" y2="58"/>
  </g>
  <text x="150" y="50" text-anchor="middle" font-size="8" fill="currentColor">15-bit shift register</text>
  <text x="32" y="70" font-size="7" fill="currentColor">bit14</text>
  <text x="243" y="70" font-size="7" fill="currentColor">bit0</text>
  <circle cx="330" cy="46" r="12" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="330" y="50" text-anchor="middle" font-size="10" fill="currentColor">⊕</text>
  <path d="M32 34 L32 20 L330 20 L330 34" fill="none" stroke="currentColor" stroke-width="1"/>
  <path d="M57 34 L57 24 L318 24" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="352" y="42" font-size="7" fill="currentColor">feedback</text>
  <path d="M342 46 L280 46" fill="none" stroke="currentColor" stroke-width="1" marker-end="url(#k)"/>
  <path d="M20 46 L8 46 L8 100" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="24" y="118" font-size="7.5" fill="currentColor">keystream bit = MSB</text>
  <circle cx="120" cy="118" r="11" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="120" y="122" text-anchor="middle" font-size="10" fill="currentColor">⊕</text>
  <text x="120" y="142" text-anchor="middle" font-size="7.5" fill="currentColor">data bit</text>
  <path d="M131 118 L180 118" fill="none" stroke="currentColor" stroke-width="1" marker-end="url(#k)"/>
  <text x="210" y="121" font-size="7.5" fill="currentColor">scrambled bit</text>
  <defs><marker id="k" markerWidth="6" markerHeight="6" refX="5" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The scrambler advances a 15-stage LFSR — the top two stages exclusive-ored back into the input — and emits its most significant stage as the keystream bit, which is XORed onto each data bit. Because XOR is symmetric, the identical operation both scrambles and descrambles.</figcaption>
</figure>

## How the keystream is generated

Each step the register outputs its most significant stage, then shifts left and inserts the XOR
of the top two stages as the new least significant bit. Seeding the register with the key and
running it over the information field produces the additive keystream:

```go
func (s *Scrambler) Next() byte {
    out := byte((s.state >> 14) & 1)
    fb := ((s.state >> 14) ^ (s.state >> 13)) & 1
    s.state = ((s.state << 1) | fb) & ScramblerKeyMax // mask to 15 bits
    return out
}
```

Key 0 seeds an all-zero register, which produces an all-zero keystream — that is the "clear"
(unscrambled) case, so key 0 means no privacy at all. Every other key produces a distinct
maximal-length sequence. Because scrambling is a bare XOR, `Descramble` is literally the same
routine as `Scramble`; the receive side reuses it with the same key.

## Why it does not provide security

Two facts collapse the scrambler's protection. First, the key is only 15 bits, so there are
just 32768 possible keys — an attacker tries them all. Second, NXDN frames carry
[CRCs](/reference/cyclic-redundancy-check/) over their decoded contents, which hand the
attacker a free correctness oracle: descramble a captured field under a candidate key, check
whether the frame's CRC now validates, and the one key in 32768 that produces consistently
valid CRCs is the answer. GopherTrunk's offline cryptolab `nxdn` tool does exactly this — it
brute-forces the full key space using this LFSR primitive plus the frame CRCs as the oracle.
An additive stream obscured by a 15-bit register is a scrambler in the historical sense, not an
encryption scheme; it stops a casual listener with a stock radio, nothing more.

## Relevance to SDR

`internal/radio/nxdn/scramble.go` implements the `Scrambler` (a 15-bit LFSR state), the
`Scramble` / `Descramble` XOR helpers, and the `ScramblerKeyMax` bound the brute-forcer sweeps.
The design deliberately separates the *spec-level* facts it is confident in — XOR symmetry and
the 2¹⁵ key space, on which the brute-force loop and CRC oracles depend — from the *provisional*
feedback polynomial, which is a plausible working model pending a bit-exact capture. That
separation is what lets the tooling be useful today (the key-recovery approach is correct
regardless of the exact taps) while being honest that the keystream itself is not yet verified
against real hardware.

## Sources

[^scr]: [Scrambler](https://en.wikipedia.org/wiki/Scrambler) — Wikipedia, on additive/multiplicative scramblers and their distinction from encryption.
[^lfsr]: [Linear-feedback shift register](https://en.wikipedia.org/wiki/Linear-feedback_shift_register) — Wikipedia, on the LFSR construction and maximal-length sequences the keystream is built from.
