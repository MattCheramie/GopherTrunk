---
slug: fire-code
title: Fire code
entry_type: algorithm
category: error-correction
description: A Fire code is a cyclic block code engineered to correct or detect a single burst of consecutive bit errors; it is the classic burst code used on GSM signalling channels.
keywords: Fire code, cyclic code, burst error correction, burst detection, GSM signalling, FACCH, SACCH, generator polynomial, shift register, Philip Fire
aka: [Fire code]
autolink: true
infobox:
  - { label: Type, value: Cyclic burst-correcting code }
  - { label: Targets, value: Single error burst }
  - { label: Used by, value: GSM control channels }
see_also: [cyclic-redundancy-check, bch-code, forward-error-correction, interleaving, reed-solomon-code, linear-feedback-shift-register]
cite_urls:
  - https://en.wikipedia.org/wiki/Burst_error-correcting_code
  - https://en.wikipedia.org/wiki/GSM
---

A **Fire code** is a [cyclic](/reference/cyclic-redundancy-check/) block code designed
specifically to handle **burst errors** — runs of corrupted bits packed close
together — rather than the scattered, independent errors that random-error codes
target.[^wiki] Devised by Philip Fire in 1959, it is built from a carefully chosen
generator polynomial so that a single contiguous burst up to a guaranteed length can
be located and corrected (or, when configured for detection, flagged), using nothing
more than a [shift register](/reference/linear-feedback-shift-register/) and a small
amount of logic.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A block of bits with one contiguous span of shaded error bits, labelled as a single burst, that the Fire code parity is able to locate and correct." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1" fill="none">
    <rect x="24" y="45" width="22" height="30"/><rect x="46" y="45" width="22" height="30"/><rect x="68" y="45" width="22" height="30"/>
    <rect x="90" y="45" width="22" height="30" fill="currentColor" fill-opacity="0.4"/><rect x="112" y="45" width="22" height="30" fill="currentColor" fill-opacity="0.4"/><rect x="134" y="45" width="22" height="30" fill="currentColor" fill-opacity="0.4"/><rect x="156" y="45" width="22" height="30" fill="currentColor" fill-opacity="0.4"/>
    <rect x="178" y="45" width="22" height="30"/><rect x="200" y="45" width="22" height="30"/><rect x="222" y="45" width="22" height="30"/><rect x="244" y="45" width="22" height="30"/>
    <rect x="300" y="45" width="22" height="30" fill="currentColor" fill-opacity="0.15"/><rect x="322" y="45" width="22" height="30" fill="currentColor" fill-opacity="0.15"/><rect x="344" y="45" width="22" height="30" fill="currentColor" fill-opacity="0.15"/><rect x="366" y="45" width="22" height="30" fill="currentColor" fill-opacity="0.15"/>
  </g>
  <line x1="90" y1="38" x2="178" y2="38" stroke="currentColor" stroke-width="0.9"/>
  <text x="134" y="32" text-anchor="middle" font-size="9" fill="currentColor">one burst (≤ b bits)</text>
  <text x="134" y="98" text-anchor="middle" font-size="9" fill="currentColor">data bits (shaded = corrupted)</text>
  <text x="344" y="98" text-anchor="middle" font-size="9" fill="currentColor">Fire parity</text>
</svg>
<figcaption>A Fire code adds parity so a single contiguous burst of corrupted bits — however severe within its guaranteed length — can be pinpointed and repaired.</figcaption>
</figure>

## How it works

The Fire code's power comes from the *shape* of its generator polynomial:

`g(x) = (x^c − 1) · p(x)`

where `p(x)` is an irreducible polynomial of degree `m` whose period does not divide
`c`. This factored form is what buys **burst** performance rather than random-error
performance. The `p(x)` factor acts like a cyclic-hash fingerprint that fixes the
*pattern* of a burst, while the `(x^c − 1)` factor pins down the burst's *position*
within the block. Together they guarantee correction of any single burst up to length
`b = min(m, c − m + 1)` (and typically longer detection). Because the code is cyclic,
both encoding and syndrome computation are just clocking the received bits through an
LFSR built from `g(x)`; decoding is **error trapping** — cycle the syndrome register
until the nonzero part collapses into a window of at most `b` bits, revealing exactly
where the burst sits and what to flip.

The trade-off is deliberate and narrow: a Fire code is superb against **one** clustered
burst and makes no promise about several independent scattered errors, which is why it
is paired with the assumption that fading and impulse noise arrive in bursts on a radio
link.

## In practice

The canonical deployment is **GSM signalling**. GSM's control channels — the FACCH and
SACCH that carry handover commands, measurement reports and paging — protect their
184-bit message blocks with a shortened binary Fire code whose generator is

`g(x) = (x^23 + 1)(x^17 + x^3 + 1) = x^40 + x^26 + x^23 + x^17 + x^3 + 1`,

adding **40 parity bits** to make a 224-bit block. GSM uses this code chiefly for
**burst detection** (a failed check marks the block as erased before the convolutional
decoder runs), exploiting the same cyclic, LFSR-friendly structure. Fire codes also
appeared in early magnetic-disk and tape controllers, where head defects and media
dropouts produce exactly the clustered errors they were built for.

## Relevance to SDR

Fire codes belong to the **cellular and storage** lineage rather than the land-mobile
trunking stack, so **GopherTrunk** does not implement a Fire decoder — the P25, DMR,
NXDN and TETRA control channels it reads use [BCH](/reference/bch-code/),
[Golay](/reference/golay-code/) and [Reed–Solomon](/reference/reed-solomon-code/)
codes, combined with [interleaving](/reference/interleaving/) to convert channel
bursts into the scattered errors those codes prefer. That contrast is the useful
lesson: a Fire code *embraces* bursts and repairs them in place, whereas the trunking
world *spreads* bursts out with an interleaver and then leans on a random-error code —
two opposite answers to the same fading channel. Anyone reverse-engineering a GSM
signalling capture in a software radio, however, will meet the Fire code directly as
the first integrity gate on every control block.

## Sources

[^wiki]: [Burst error-correcting code](https://en.wikipedia.org/wiki/Burst_error-correcting_code) — Wikipedia, for the Fire code generator polynomial `(x^c − 1)·p(x)`, its guaranteed burst-correction length, error-trapping decoding, and the GSM signalling application.
