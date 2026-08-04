---
slug: pn44-scrambler
title: PN44 scrambler
entry_type: algorithm
category: error-correction
description: PN44 is the P25 Phase 2 bit-scrambler — a 44-bit LFSR with generator G(x)=x^44+x^40+x^35+x^29+x^24+x^10+1 whose seed is derived from the system's WACN, System ID and Color Code, XORed over every coded burst between demodulation and FEC.
keywords: PN44 scrambler, P25 Phase 2 scrambler, 44-bit LFSR, WACN System ID Color Code seed, SH(243) inbound seed, slot offset, TIA-102.BBAC-1 7.2.5, descrambler
aka: [PN44, "P25 Phase 2 scrambler", "PN sequence scrambler"]
autolink: true
infobox:
  - { label: Register, value: 44-bit LFSR }
  - { label: Polynomial, value: "x⁴⁴+x⁴⁰+x³⁵+x²⁹+x²⁴+x¹⁰+1" }
  - { label: Seed from, value: WACN · System ID · Color Code }
  - { label: Spec, value: TIA-102.BBAC-1 §7.2.5 }
see_also: [scrambling, linear-feedback-shift-register, color-code, wacn, system-id, p25-phase-2, p25-phase-2-superframe, reed-solomon-code]
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Scrambler
  - https://en.wikipedia.org/wiki/Linear-feedback_shift_register
---

**PN44** is the bit-level [scrambler](/reference/scrambling/) that P25 Phase 2 XORs over every voice
and signalling burst between demodulation and forward error correction.[^scr] It is a 44-bit
[linear-feedback shift register](/reference/linear-feedback-shift-register/) whose pseudo-random output
is added modulo-2 to the coded channel bits; because XOR is self-inverse, running the identical
sequence again at the receiver descrambles them.[^lfsr] What makes PN44 system-specific — not a fixed
pattern like a sync word — is that its **seed is derived from the system identity**: the
[WACN](/reference/wacn/), the [System ID](/reference/system-id/), and the
[Color Code](/reference/color-code/), the same three values the Network Status Broadcast MAC message
publishes. The generator polynomial is `G(x) = x⁴⁴ + x⁴⁰ + x³⁵ + x²⁹ + x²⁴ + x¹⁰ + 1`.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 170" role="img" aria-label="A 44-bit shift register outputs its most-significant bit as the PN44 sequence; the feedback bit is the XOR of the register taps at positions 40, 35, 29, 24, 10 and 0, clocked back into the low end. The seed is packed from the 20-bit WACN, 12-bit System ID and 12-bit Color Code." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="46" width="330" height="26" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="185" y="63" text-anchor="middle" font-size="8.5" fill="currentColor">44-bit LFSR state</text>
  <text x="356" y="63" font-size="8" fill="currentColor">→ out (bit 43)</text>
  <g stroke="currentColor" stroke-width="1"><line x1="60" y1="46" x2="60" y2="30"/><line x1="120" y1="46" x2="120" y2="30"/><line x1="180" y1="46" x2="180" y2="30"/><line x1="230" y1="46" x2="230" y2="30"/><line x1="285" y1="46" x2="285" y2="30"/><line x1="335" y1="46" x2="335" y2="30"/></g>
  <line x1="60" y1="30" x2="335" y2="30" stroke="currentColor" stroke-width="1"/>
  <text x="197" y="24" text-anchor="middle" font-size="7.5" fill="currentColor">taps 0,10,24,29,35,40 → XOR feedback → bit 0</text>
  <path d="M60 30 L40 30 L40 59 L18 59" fill="none" stroke="currentColor" stroke-width="1"/>
  <path d="M24 55 L18 59 L24 63" fill="none" stroke="currentColor" stroke-width="1"/>
  <text x="20" y="104" font-size="8" fill="currentColor">seed = WACN·2²⁴ + SystemID·2¹² + ColorCode   (0 → 2⁴⁴−1)</text>
  <text x="20" y="124" font-size="8" fill="currentColor">inbound seed = outbound advanced 243 clocks  ·  SH(243)</text>
  <text x="20" y="150" font-size="8" fill="currentColor">sequence restarts every 360 ms superframe (4320 bits); each slot offset = k·360</text>
</svg>
<figcaption>PN44 shifts one bit per clock, emitting bit 43 while XOR feedback from six taps clocks into bit 0; its seed is packed from the WACN, System ID and Color Code, and the sequence restarts each superframe.</figcaption>
</figure>

## Seed and register

The seed packs the three identity fields into the 44-bit field exactly as the spec's equation (5)
defines, and the degenerate all-zero case maps to the all-ones state so the register never stalls:

```go
// seed_external = WACN·2^24 + System_ID·2^12 + Color_Code
//   WACN 20 bits, System_ID 12 bits, Color_Code 12 bits.
seed := (uint64(wacnID&0xFFFFF) << 24) |
        (uint64(systemID&0xFFF) << 12) |
        uint64(colorCode & 0xFFF)
if seed == 0 { seed = (1 << 44) - 1 }

// One clock: emit bit 43, feed back the tap XOR into bit 0.
out := byte((state >> 43) & 1)
fb := byte((state>>40 ^ state>>35 ^ state>>29 ^
            state>>24 ^ state>>10 ^ state) & 1)
state = ((state << 1) | uint64(fb)) & ((1 << 44) - 1)
```

The output is read from position 43 (the register's MSB) *before* the shift; the feedback bit is the
XOR of the state bits at the polynomial's non-leading tap positions 0, 10, 24, 29, 35 and 40, and it
clocks into position 0 as the register shifts left. The [Color Code](/reference/color-code/) that seeds
it equals the Phase 1 NAC per the spec's derivation rule.

## Directions, offsets, and blind probing

The uplink and downlink use *different* PN44 phases. Per equation (8) the **inbound** seed is the
outbound seed advanced by **243 LFSR cycles** — the SH(243) state-transition matrix in the spec's
Figure 7-4, which GopherTrunk computes by simply clocking the register 243 times rather than
transcribing a 44×44 matrix (the two are equivalent and the iterative form is trivially testable).

The sequence restarts at the start of every **360 ms superframe (4320 bits)**, and each of the 12
slots begins at a fixed **offset of k·360 bits** into that sequence (30 ms of channel bits per slot).
A receiver that already has superframe sync descrambles at the known slot offset; one that does not can
**blind-probe**, descrambling at each of the 12 candidate offsets and accepting the phase whose outer
[Reed-Solomon RS(24,16,9)](/reference/reed-solomon-code/) syndromes come out zero. A wrong offset packs
into random bytes that satisfy the 8-parity-symbol check with probability ≈2⁻⁴⁸, so the sweep never
accepts garbage.

One ordering detail cost real time: the scrambler wraps the *coded* burst, so the descramble must run
**first**, on the raw channel dibits, before de-interleave and trellis decoding. Descrambling the
recovered information bits *after* the trellis can never satisfy the outer RS check, because the trellis
code does not commute with the XOR — the symptom was Phase 2 traffic decoding with `mac_rs_valid = 0`
regardless of seed.

## Relevance to SDR

`internal/radio/framing/pn44_p25_phase2.go` implements the whole primitive — `PN44Scrambler` with
`Next`/`Advance`/`Apply`, `PN44SeedFromIdentity`, `PN44SeedInbound`, and the per-slot offset tables —
and it is wired into the Phase 2 control-channel process adapter through the scrambler modes. Getting
the seed derivation and the descramble-before-FEC ordering right is what lets GopherTrunk read a real
Phase 2 MAC PDU or voice burst instead of noise. The spec is TIA-102.BBAC-1 §7.2.5.

## Sources

[^scr]: [Scrambler](https://en.wikipedia.org/wiki/Scrambler) — Wikipedia, on additive (synchronous) scramblers and their self-inverse XOR property.
[^lfsr]: [Linear-feedback shift register](https://en.wikipedia.org/wiki/Linear-feedback_shift_register) — Wikipedia, on LFSR generator polynomials and pseudo-random sequences.
