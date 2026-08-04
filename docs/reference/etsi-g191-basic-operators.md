---
slug: etsi-g191-basic-operators
title: G.191 basic operators
entry_type: term
category: voice-coding
description: "The ITU-T G.191 basic operators are the bit-exact 16/32-bit saturating primitives (add, sub, mult, L_mac, shr, L_extract, log2/pow2) the ETSI reference speech codecs are built from; Word32 must be a 32-bit int or every op returns garbage on LP64."
keywords: G.191 basic operators, STL, fixed-point primitives, saturating arithmetic, Word16 Word32, L_mac, L_extract, ETSI reference codec, LP64 long, TETRA vocoder
aka: [STL basic operators, "G.191 STL", "basic operators"]
autolink: true
infobox:
  - { label: Source, value: ITU-T G.191 STL }
  - { label: Types, value: "Word16 (16-bit), Word32 (32-bit)" }
  - { label: Behaviour, value: Bit-exact saturating arithmetic }
  - { label: LP64 trap, value: "long is 64-bit, breaks Word32" }
see_also: [fixed-point-vs-floating-point, acelp, acelp-gain-quantization, acelp-post-process, quantization]
cite_urls:
  - https://en.wikipedia.org/wiki/Saturation_arithmetic
  - https://en.wikipedia.org/wiki/Fixed-point_arithmetic
  - https://en.wikipedia.org/wiki/Q_(number_format)
---

The **G.191 basic operators** are the small set of bit-exact fixed-point primitives — saturating
`add`, `sub`, `mult`, the multiply-accumulate `L_mac`, the shifts `shr`/`L_shl`, `L_extract`, and
the `log2`/`pow2` table kernels — that the ETSI and ITU-T reference speech codecs are *defined*
in.[^stl] A standards-body speech codec is not published as a formula but as C source built
entirely from these operators, so their exact overflow, rounding, and saturation behaviour **is**
the specification. Re-implement them faithfully and a decoder is bit-exact with the reference;
get one detail wrong and the whole codec drifts. GopherTrunk's TETRA
[ACELP](/reference/acelp/) decoder ports the ETSI reference operator library into
`internal/voice/acelp/ops.go` and `mathfp.go`.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 130" role="img" aria-label="Two 16-bit Word16 values feed a saturating operator whose 32-bit result is clamped into the Word16 range; a correct 32-bit Word32 accumulator saturates at plus or minus two-to-the-31, while a mistaken 64-bit long never reaches the clamp so the saturating operator silently returns wrong values." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.1" fill="none"><rect x="16" y="30" width="70" height="26"/><rect x="16" y="66" width="70" height="26"/><rect x="150" y="48" width="80" height="26"/><rect x="300" y="48" width="90" height="26"/></g>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <text x="51" y="46">Word16 a</text><text x="51" y="82">Word16 b</text>
    <text x="190" y="64">op (Word32)</text>
    <text x="345" y="60">saturate</text><text x="345" y="70">→ Word16</text>
  </g>
  <g stroke="currentColor" stroke-width="1" fill="none">
    <line x1="86" y1="43" x2="150" y2="56" marker-end="url(#gopar)"/>
    <line x1="86" y1="79" x2="150" y2="66" marker-end="url(#gopar)"/>
    <line x1="230" y1="61" x2="300" y2="61" marker-end="url(#gopar)"/>
    <defs><marker id="gopar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  </g>
  <text x="190" y="98" text-anchor="middle" font-size="7.5" fill="currentColor">Word32 MUST be 32-bit: a 64-bit long never hits the ±2³¹ clamp</text>
</svg>
<figcaption>Every operator computes in a 32-bit Word32 accumulator and clamps back to Word16; if Word32 is silently widened to 64 bits the clamp never triggers and results diverge.</figcaption>
</figure>

## What the operators guarantee

Each operator models a specific piece of DSP-chip arithmetic. `sature` clamps a 32-bit value into
the [Q15](/reference/fixed-point-vs-floating-point/) 16-bit range; `add`/`sub` do 16-bit
saturating arithmetic; `mult` and `L_mult` are the fractional multiplies; `L_mac`/`L_msu` are the
multiply-accumulate and multiply-subtract that dominate filter loops; the shifts saturate on
overflow; and `L_extract` splits a 32-bit value into a high/low pair for double-precision work.
On top of these, `mathfp.go` provides the `log2fp`/`pow2fp` table kernels the
[gain dequantizer](/reference/acelp-gain-quantization/) needs. Because every operation clamps
rather than wraps, intermediate overflow degrades gracefully instead of flipping sign — the same
[saturation](/reference/quantization/) discipline the reference chips enforced in hardware.

A representative clamp — GopherTrunk's `sature`, the primitive every 16-bit result funnels
through — shows the shape of the whole library:

```go
// sature clamps a 32-bit value into the 16-bit Word16 range,
// setting the reference codec's global overflow flag on clamp.
func sature(l int32) int16 {
    switch {
    case l > int32(maxWord16): // +32767
        overflow = true
        return maxWord16
    case l < int32(minWord16): // -32768
        overflow = true
        return minWord16
    default:
        overflow = false
        return int16(l)
    }
}
```

## The Word32 trap

The single most expensive detail is the width of `Word32`. The ETSI/ITU reference sources declare
it as `typedef long Word32`. On the ILP32 platforms the codecs were written for, `long` is 32
bits, and every saturating operator relies on that width — the accumulator is *meant* to reach
±2³¹ and clamp there. Build the same C on a modern **LP64** system, where `long` is **64 bits**,
and `Word32` silently becomes 64-bit: the intermediate values that should overflow and saturate no
longer reach the 32-bit clamp, so `L_mac`, the shifts, and everything built on them return wrong
results, and the codec produces garbage. The fix is to define `Word32` as a **32-bit int**
explicitly. GopherTrunk sidesteps the problem entirely by using Go's fixed-width `int32`, so the
operators saturate at exactly the reference boundaries regardless of host platform. This lesson —
build `Word32` as 32-bit or every saturating op is garbage on LP64 — is recorded in the project's
TETRA voice notes precisely because it silently defeats an otherwise-correct port.

## Why a faithful port matters

Speech-codec bugs are notoriously self-consistent: a synthetic encode→decode round-trip inside one
codebase can pass while the decoder still fails on real off-air frames, because both ends share the
same wrong arithmetic. Anchoring the basic operators to the reference is what lets GopherTrunk test
the *decoder alone* against the ETSI reference C codec — feed both the same bitstream and demand
bit-identical PCM — rather than trusting a closed self-check.

## Relevance to SDR

For a software scanner with no vendor DSP dongle, these operators are the foundation that makes a
pure-Go [ACELP](/reference/acelp/) (and, in the same spirit, MBE) decoder trustworthy. They are
not glamorous, but they are load-bearing: the correctness of every higher-level codec block —
LSP reconstruction, excitation, gain, and the [output scaling](/reference/acelp-post-process/) —
rests on this arithmetic behaving exactly as the standard prescribes.

## Sources

[^stl]: [Saturation arithmetic](https://en.wikipedia.org/wiki/Saturation_arithmetic) — Wikipedia, on the clamping arithmetic the G.191 operators implement.
[^fp]: [Fixed-point arithmetic](https://en.wikipedia.org/wiki/Fixed-point_arithmetic) — Wikipedia, on the Word16/Word32 fixed-point number model.
