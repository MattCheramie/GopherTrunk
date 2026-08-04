---
slug: tetra-scrambler
title: TETRA scrambler
entry_type: algorithm
category: error-correction
description: The TETRA scrambler is a 32-tap linear-feedback shift register, tap mask 0x82608EDB, seeded by the 30-bit extended colour code, whose output XORs the coded bits of every logical channel except the BSCH so a receiver must know the cell identity to descramble.
keywords: TETRA scrambler, TETRA scrambling, 0x82608EDB, extended colour code seed, TETRA LFSR, connection polynomial, type-4 type-5 bits, EN 300 392-2 8.2.5
aka: [TETRA scrambling sequence, "TETRA LFSR scrambler"]
autolink: true
infobox:
  - { label: Type, value: "32-tap LFSR (additive)" }
  - { label: Tap mask, value: "0x82608EDB" }
  - { label: Seed, value: 30-bit extended colour code }
  - { label: Spec, value: EN 300 392-2 §8.2.5 }
see_also: [scrambling, linear-feedback-shift-register, tetra-extended-colour-code, cyclic-redundancy-check, tetra-rcpc-code, tetra-logical-channels, tetra, color-code, tetra-burst-formats]
cite_urls:
  - https://en.wikipedia.org/wiki/Scrambler
  - https://en.wikipedia.org/wiki/Linear-feedback_shift_register
---

The **TETRA scrambler** is the [linear-feedback shift register](/reference/linear-feedback-shift-register/)
whose pseudo-random output is XORed onto the coded bits of every [TETRA](/reference/tetra/) logical channel
except the initial BSCH.[^scr] It is an *additive* [scrambler](/reference/scrambling/): the transmitter
turns type-4 channel bits into type-5 on-air bits by XOR with a bit sequence `p(k)`, and the receiver
recovers the type-4 bits by XOR with the identical sequence. Crucially the LFSR is seeded by the cell's
30-bit [extended colour code](/reference/tetra-extended-colour-code/), so a receiver must first learn that
identity — from the unscrambled synchronisation channel — before any other channel becomes readable.[^lfsr]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 132" role="img" aria-label="A 32-stage shift register with feedback taps combined by XOR feeding a new bit into the low end, its output bit XORed with an incoming coded bit to produce the on-air bit; the register is seeded from the extended colour code." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="30" width="300" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <g stroke="currentColor" stroke-width="0.8">
    <line x1="90" y1="30" x2="90" y2="56"/><line x1="140" y1="30" x2="140" y2="56"/>
    <line x1="190" y1="30" x2="190" y2="56"/><line x1="240" y1="30" x2="240" y2="56"/>
    <line x1="290" y1="30" x2="290" y2="56"/>
  </g>
  <text x="190" y="47" text-anchor="middle" font-size="8" fill="currentColor">32-tap LFSR · mask 0x82608EDB</text>
  <text x="190" y="74" text-anchor="middle" font-size="7.5" fill="currentColor">taps → XOR → new bit p(k) shifted in at bit 0</text>
  <path d="M340 43 L370 43" stroke="currentColor" stroke-width="1.1" fill="none"/>
  <circle cx="385" cy="43" r="10" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="385" y="47" text-anchor="middle" font-size="10" fill="currentColor">⊕</text>
  <path d="M385 20 L385 33" stroke="currentColor" stroke-width="1.1" fill="none"/>
  <text x="385" y="16" text-anchor="middle" font-size="7.5" fill="currentColor">coded bit</text>
  <path d="M395 43 L430 43" stroke="currentColor" stroke-width="1.1" fill="none"/>
  <text x="432" y="46" font-size="7.5" fill="currentColor">on-air</text>
  <text x="40" y="104" font-size="7.5" fill="currentColor">seed: extended colour code e(1)..e(30) in state bits, fill 0xC0000000</text>
</svg>
<figcaption>The LFSR feeds tapped state bits through XOR to generate p(k); each output bit is XORed with a coded bit to scramble it. The register is seeded from the extended colour code, so descrambling requires the cell identity.</figcaption>
</figure>

## The generator

The connection polynomial (§8.2.5.2 eq. 8.40) has 14 feedback taps, which GopherTrunk packs into a 32-bit
mask so each step is a masked-AND plus a parity:

```go
// internal/radio/framing/scramble_tetra.go — ETSI EN 300 392-2 §8.2.5.
// Taps at i = 1,2,4,5,7,8,10,11,12,16,22,23,26,32.
const scrambleTetraTapMask uint32 = 0x82608EDB

func (s *ScramblerTetra) Next() byte {
    v := s.state & scrambleTetraTapMask
    bit := byte(popcount32(v) & 1)     // new p(k) = XOR of tapped bits
    s.state = (s.state << 1) | uint32(bit) // shift left, insert at bit 0
    return bit
}
```

The register is initialised (§8.2.5.2 eq. 8.42) with the extended colour code in its low 30 state bits and
a constant `0xC0000000` fill in the top two (`p(−30) = p(−31) = 1`); output begins at `p(1)`. For the BSCH
and BSCH-Q the colour-code bits are all zero, so a cold receiver can always descramble the synchronisation
channel with a zero seed.

## Two invisible bugs

The scrambler is a cautionary tale in why round-trip tests are not enough (issue #925). Two independent bugs
each broke every real, externally-scrambled burst while passing every synthetic test. First, the LFSR
originally **shifted the wrong direction** — right, inserting at bit 31 — which reversed the register relative
to the tap convention and diverged from the spec sequence starting at `p(2)`. Second, the 30-bit seed is
carried MSB-first (e(1) in the high bit) but the LFSR wants e(i+1) in state bit i, so the low 30 bits must be
**bit-reversed** on the way in. Both faults are self-consistent: because scrambling and descrambling share the
same generator, a wrong-but-matching sequence cancels perfectly in any encode-then-decode round-trip, and the
zero-seeded BSCH is a fixed point of bit-reversal. The only symptom was on real air — the receiver would
*lock* off the unscrambled synchronisation training sequence, yet no BNCH or SCH message would ever decode.
This is the general TETRA trap: validate the scrambled path against a real capture, not just a self-round-trip.

## Relevance to SDR

`ScrambleTetra` / `DescrambleTetra` are the XOR-symmetric entry points the whole channel stack calls after
[colour-code](/reference/tetra-extended-colour-code/) learning. The traffic extractor descrambles each raw
type-5 block before the TCH/S decoder sees it, and the AACH, BNCH, and SCH decoders all descramble first. A
soft-decision variant, `DescrambleTetraSoft`, applies the same per-bit sign flips to LLRs so soft-decision
decoding works end to end. Getting the tap mask, shift direction, and seed orientation all exactly right is
the difference between a control channel that *locks and decodes* and one that only locks.

## Sources

[^scr]: [Scrambler](https://en.wikipedia.org/wiki/Scrambler) — Wikipedia, on additive (synchronous) scrambling by XOR with a pseudo-random sequence.
[^lfsr]: [Linear-feedback shift register](https://en.wikipedia.org/wiki/Linear-feedback_shift_register) — Wikipedia, on the register and feedback-polynomial construction the scrambler uses.
