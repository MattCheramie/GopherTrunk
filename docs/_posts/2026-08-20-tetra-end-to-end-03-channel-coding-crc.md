---
title: "TETRA End to End, Part 3: Channel Coding — RCPC, Viterbi & the CRC That Isn't an LFSR"
description: Walking TETRA's type-5 bits back to information — two K=5 RCPC mother codes, depuncturing into a shared 16-state Viterbi, the block interleavers, and the class-2 speech CRC that is a fixed parity-check matrix, whose LFSR misimplementation silently dropped every on-air voice burst while synthetic round-trips stayed green.
category: deep-dives
keywords: tetra channel coding, rcpc convolutional code, tetra viterbi decoder, tetra puncturing, tetra block interleaver, tetra crc tab_crc, parity check matrix crc, self-consistent test trap, type-1 type-5 bits, gophertrunk tetra
tags: [tetra-end-to-end, tetra, fec, viterbi, crc, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "TETRA End to End"
series_part: 3
---

*Part 3 of **TETRA End to End**, a 14-part deep dive into how GopherTrunk turns
one real 25 kHz TETRA carrier into clear recorded voice.
[Part 2]({{ '/blog/deep-dives/tetra-end-to-end-02-bursts-slot-grid/' | relative_url }})
sliced the downlink into bursts and gave each one its 216 or 432 type-5 bits —
still scrambled, interleaved, punctured and convolved. This part walks those
bits back to information. It is also where the series' villain claims its most
expensive victim: a CRC implemented as the "obvious" shift register when the
spec meant a fixed parity-check matrix, a bug that dropped **every** on-air
speech burst while every synthetic round-trip passed — because both sides of
the round trip shared it.*

> **TL;DR:** TETRA channel coding is ETSI's type-1 → type-5 pipeline: CRC +
> tail bits + RCPC convolutional code + block interleave + scramble. There are
> **two K=5 mother codes** — an R=1/4 for signalling channels
> (`framing.EncodeRCPCTetraSigMother`, EN 300 392-2 §8.2.3.1) and an R=1/3 for
> speech (`framing.EncodeRCPCTetraMother`, EN 300 395-2 §5.4.3) — both decoded
> by 16-state Viterbi with erasure-aware depuncturing (`DepunctureMark`) and a
> forced state-0 terminal. Signalling uses a genuine CRC-CCITT LFSR
> (`0x1021`/`FFFF`/`FFFF`). The speech class-2 CRC does **not**: it is the
> reference codec's `TAB_CRC` fixed parity-check matrix (`tchCRCTaps`,
> `internal/radio/tetra/tch_tables.go`), and an earlier `G(X)=1+X³+X⁷`
> approximation failed every real burst while passing every self-consistent
> synthetic test.

**Key takeaways**

- **One pipeline, many channels.** BSCH, SCH/HD, SCH/HU and SCH/F are the same
  `signalingEncode`/`signalingDecode` chain with different interleaver
  constants; only AACH deviates (a (30,14) Reed-Muller block code, no
  convolution).
- **Two mother codes, one Viterbi shape.** Signalling is K=5 R=1/4 with four
  generators; speech is K=5 R=1/3 with three. Same 16-state trellis
  discipline, same tail-bit flush to state 0, separate primitives so the
  polynomials can never cross-contaminate.
- **Depuncturing is erasure marking, not guessing.** Punctured positions are
  filled with `DepunctureMark` and the Viterbi cost accumulator skips them —
  the decoder is *told* what was never transmitted.
- **"CRC" is a claim about a spec, not an algorithm.** The class-2 speech
  check is a parity-check matrix. Implementing it as a polynomial LFSR
  produced a decoder that agreed perfectly with its own encoder and disagreed
  with every base station on Earth.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Signalling mother code | K=5 R=1/4, G₁..G₄ per §8.2.3.1 | `internal/radio/framing/rcpc_tetra_sig.go` (`EncodeRCPCTetraSigMother`) |
| Speech mother code | K=5 R=1/3, G₁..G₃ per EN 300 395-2 §5.4.3 | `internal/radio/framing/rcpc_tetra.go` (`EncodeRCPCTetraMother`) |
| Viterbi (speech) | 16 states, erasure-aware, state-0 terminal | `rcpc_tetra.go` (`DecodeRCPCTetraMother`) |
| Puncture / depuncture | §5.4.3.2 formula, spec-verbatim tables | `rcpc_tetra.go` (`PunctureRCPCTetra`, `RCPCTetraPuncture23`) |
| Signalling chain | CRC-16 + tail + RCPC 2/3 + interleave + scramble | `internal/radio/tetra/channel_coding.go` (`signalingDecode`) |
| Speech class-2 CRC | fixed parity-check matrix (`TAB_CRC`) | `internal/radio/tetra/tch_tables.go` (`tchCRCTaps`), `tch.go` (`crcTCHClass2`) |

## In this post

- **The type-1 to type-5 pipeline** — ETSI's names for the stages, and the one chain five channels share.
- **Two mother codes** — why signalling and speech convolve differently.
- **Depuncture and Viterbi** — erasures, the trellis, and the state-0 constraint.
- **The CRC that isn't an LFSR** — the class-2 parity-check matrix and the bug it hid.
- **Naming the trap** — what this failure class looks like so you recognise it next time.

## The type-1 to type-5 pipeline

ETSI names the stages of every TETRA logical channel's coding chain: type-1
bits are raw information, and each transform — block code, convolutional code,
interleave, scramble — increments the number until type-5 is what's on air.
GopherTrunk composes the whole signalling chain from framing primitives in one
function pair:

```go
// internal/radio/tetra/channel_coding.go (shape)
func signalingEncode(type1 []byte, interleaverK, interleaverA int, colourCode uint32) []byte {
    withCRC := appendCRC16(type1)       // K1 + 16 (CRC-CCITT 0x1021, FFFF/FFFF)
    type2 := appendTailBits(withCRC, 4) // K1 + 20 (K−1 zero tail bits)
    type3 := encodeRCPCRate23(type2)    // × 3/2 (R=1/4 mother + 2/3 puncture)
    type4 := framing.BlockInterleaveTetra(type3, interleaverK, interleaverA)
    type5 := framing.ScrambleTetra(type4, colourCode)
    return type5
}
```

Decoding is the strict mirror: descramble, deinterleave, depuncture +
Viterbi, strip tail, verify CRC — `signalingDecode` returns the info bits plus
a pass/fail flag, and a failing CRC means "best Viterbi guess, do not trust."
Every signalling channel is this chain with different interleaver constants:
BSCH is 60 → 120, SCH/HD (also serving BNCH and STCH) 124 → 216, SCH/HU
92 → 168, SCH/F 268 → 432. Only the AACH deviates — 14 bits in a (30,14)
Reed-Muller code plus scramble, no convolution, because it must decode in
every single slot cheaply (see the
[RM(30,14) reference]({{ '/reference/tetra-rm-30-14/' | relative_url }})). The
composition-over-monolith argument for this design is made in
[Protocol Decoders Part 7]({{ '/blog/deep-dives/protocol-decoders-07-tetra/' | relative_url }});
here we go a level deeper, into the primitives.

## Two mother codes

A detail the one-post survey glossed over: TETRA has **two** RCPC mother
codes, from two different standards documents, and they are different codes.

Every signalling channel uses the K=5, rate-1/4 code of EN 300 392-2
§8.2.3.1 — four generator polynomials (G₁ = 0x13, G₂ = 0x1D, G₃ = 0x17,
G₄ = 0x1B), punctured down to rate 2/3. The speech traffic channel uses the
K=5, rate-1/3 code of EN 300 395-2 §5.4.3 — three generators:

```go
// internal/radio/framing/rcpc_tetra.go (shape) — the SPEECH mother code
// G_1(D) = 1 + D + D^2 + D^3 + D^4  (0x1F)
// G_2(D) = 1 + D + D^3 + D^4        (0x1B)
// G_3(D) = 1 + D^2 + D^4            (0x15)
func EncodeRCPCTetraMother(input []byte) []byte {
    out := make([]byte, 3*len(input))
    var d1, d2, d3, d4 byte
    for i, in := range input {
        bit := in & 1
        out[3*i] = bit ^ d1 ^ d2 ^ d3 ^ d4
        out[3*i+1] = bit ^ d1 ^ d3 ^ d4
        out[3*i+2] = bit ^ d2 ^ d4
        d4, d3, d2, d1 = d3, d2, d1, bit
    }
    return out
}
```

Same 16-state structure, different outputs per input, different puncturing
table families — so GopherTrunk keeps them as separate primitives
(`rcpc_tetra_sig.go` vs `rcpc_tetra.go`) rather than one parameterised
encoder. The speech code is punctured two ways within a single slot: rate 8/12
over the class-1 bits and rate 8/18 over class 2 + CRC + tail
(`RCPCTetraPuncture23` = `{1,2,4}` on period 6; `RCPCTetraPuncture818` on
period 12, both verbatim from the spec) — unequal error protection, with the
perceptually critical class-2 bits armoured hardest. Part 5 shows where those
classes come from.

## Depuncture and Viterbi

Puncturing transmits only a subset of mother-code bits; the decoder's first
job is to be honest about the holes. `DepunctureRCPCTetra` allocates the full
mother-length buffer, fills it with `DepunctureMark`, and copies received bits
into the puncture-map positions. The Viterbi then *skips* marked positions in
its cost accumulator — an untransmitted bit contributes no evidence either
way:

```go
// internal/radio/framing/rcpc_tetra.go (shape) — DecodeRCPCTetraMother
for input := 0; input < 2; input++ {
    g1 := byte(input^d1^d2^d3^d4) & 1
    /* … g2, g3 … */
    cost := pm[cur]
    if rxG1 != DepunctureMark && g1 != rxG1 { cost++ }
    if rxG2 != DepunctureMark && g2 != rxG2 { cost++ }
    if rxG3 != DepunctureMark && g3 != rxG3 { cost++ }
    /* … relax npm[next], record traceback … */
}
// Encoder is flushed to state 0 by the tail bits — pick state 0 unconditionally.
final := 0
```

Two constraints do quiet work here. The four zero tail bits appended at encode
time drive the encoder back to state 0, so the traceback starts from a *known*
terminal state instead of the cheapest one — one more equation the received
bits must satisfy. And the path metric comes back to the caller: 0 means a
clean channel, small positive values mean corrected errors, and the traffic
diagnostics histogram those metrics to grade a capture. The general theory —
trellises, survivors, why convolutional codes like soft inputs — is in the
[framing & FEC deep dive]({{ '/blog/deep-dives/sdr-internals-09-framing-fec/' | relative_url }});
the soft-input version of this exact decoder
(`DecodeRCPCTetraMotherSoft`) is Part 8's payoff.

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="The TETRA decode chain from type-5 to type-1: descramble, deinterleave, then depuncture where missing positions are filled with erasure marks, then a 16-state Viterbi decoder constrained to end in state zero, then the CRC check. A fork above the CRC stage contrasts the signalling CRC-16, a genuine LFSR, with the speech class-2 CRC, a fixed parity-check matrix.">
  <rect x="8" y="80" width="96" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="56" y="98" text-anchor="middle" fill="currentColor" font-size="10">descramble</text>
  <text x="56" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">type-5 → 4</text>
  <line x1="104" y1="101" x2="126" y2="101" stroke="currentColor"/><polygon points="126,97 135,101 126,105" fill="currentColor"/>
  <rect x="135" y="80" width="100" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="185" y="98" text-anchor="middle" fill="currentColor" font-size="10">deinterleave</text>
  <text x="185" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">type-4 → 3</text>
  <line x1="235" y1="101" x2="257" y2="101" stroke="currentColor"/><polygon points="257,97 266,101 257,105" fill="currentColor"/>
  <rect x="266" y="80" width="106" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="319" y="98" text-anchor="middle" fill="currentColor" font-size="10">depuncture</text>
  <text x="319" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">holes = erasures</text>
  <line x1="372" y1="101" x2="394" y2="101" stroke="currentColor"/><polygon points="394,97 403,101 394,105" fill="currentColor"/>
  <rect x="403" y="80" width="100" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="453" y="98" text-anchor="middle" fill="var(--accent)" font-size="10">Viterbi K=5</text>
  <text x="453" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">ends in state 0</text>
  <line x1="503" y1="101" x2="525" y2="101" stroke="currentColor"/><polygon points="525,97 534,101 525,105" fill="currentColor"/>
  <rect x="534" y="80" width="138" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="603" y="98" text-anchor="middle" fill="var(--accent)" font-size="10">CRC gate</text>
  <text x="603" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">pass ⇒ type-1 trusted</text>
  <text x="603" y="34" text-anchor="middle" fill="var(--fg-muted)" font-size="9">signalling: CRC-CCITT LFSR (0x1021)</text>
  <text x="603" y="48" text-anchor="middle" fill="var(--accent)" font-size="9">speech class 2: TAB_CRC parity matrix</text>
  <line x1="603" y1="54" x2="603" y2="76" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="340" y="170" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the same pipeline serves BSCH / SCH/HD / SCH/HU / SCH/F — only the interleaver constants and the CRC family change</text>
</svg>
<figcaption>Type-5 back to type-1: the shared decode pipeline, with the CRC stage forking between a genuine LFSR (signalling) and a fixed parity-check matrix (speech class 2).</figcaption>
</figure>

## The CRC that isn't an LFSR

Now the star exhibit. The signalling CRC-16 is exactly what the name suggests —
the textbook CRC-CCITT-FALSE shift register, polynomial `0x1021`, init
`0xFFFF`, final XOR `0xFFFF` (`crcTetraK1Plus16`). So when the TCH/S speech
channel called for "an 8-bit CRC over the 60 class-2 bits," the natural move
was another polynomial register — and an early implementation used
`G(X) = 1 + X³ + X⁷`.

But EN 300 395-2 §5.5.1 does not define a cyclic code. It defines eight parity
equations — the reference codec ships them as tables named `TAB_CRC1..8` — and
GopherTrunk now implements them as exactly that:

```go
// internal/radio/tetra/tch.go (shape) — crcTCHClass2
// Each CRC bit is the even parity (XOR) of the class-2 bits at the ranks in
// tchCRCTaps — the ETSI reference's fixed parity-check matrix, NOT a G(X) LFSR.
func crcTCHClass2(class2 []byte) []byte {
    out := make([]byte, tchCRCBits)
    for k := range tchCRCTaps {
        var parity byte
        for _, rank := range tchCRCTaps[k] {
            parity ^= class2[rank-1] & 1 // ranks are 1-based
        }
        out[k] = parity
    }
    return out
}
```

The failure mode of the LFSR version is what earns this bug its place in the
series. Encode and decode shared the wrong CRC, so every synthetic round-trip
— encode two speech frames, push them through the full chain, decode, compare
— passed. Every self-test was green. Meanwhile on real air, where the base
station computes the *spec's* CRC, every received TCH/S burst failed the check
and was silently gated out: the multi-slot replay harness of that era recorded
the symptom as CRC passes at "the ~1/256 chance floor" — indistinguishable
from noise, on a capture whose signalling decoded perfectly. No error, no
panic, just no voice.

## Naming the trap

This is the **self-consistent-synthetic trap**, and it is worth a name because
it recurs: the training-sequence placeholder constants of Part 2, this CRC,
the scrambler seed of Part 4, and a DMO descramble skip in Part 12 are all the
same species. A round-trip test validates that your encoder and decoder agree
with *each other* — it cannot validate that either agrees with the *world*.
The full taxonomy, with the repo's countermeasures, is in
[From the Issue Tracker Part 20]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}).
For channel coding specifically the countermeasures are concrete: pin decoders
against externally produced vectors (a real capture, a reference codec's
output — Part 7's whole subject), and treat "synthetic passes, air fails" as a
*diagnosis*, not a mystery. When voice doesn't decode but the vocoder's unit
tests pass, suspect the channel coding — the chain between demod and vocoder —
before either neighbour. That sentence, hard-earned here, is now standing
guidance in the repo.

## Where this goes next

One stage of the pipeline got only a sentence: the scramble. It looks like the
most trivial stage — XOR with a pseudo-random sequence — and it has produced
more subtle failures than any other.
[Part 4]({{ '/blog/deep-dives/tetra-end-to-end-04-scrambling-colour-codes/' | relative_url }})
covers the LFSR, the 30-bit extended colour code that seeds it, and the two
real bugs it hid — including why `NewScramblerTetra(0)` is emphatically not a
no-op, the fact that sets up the DMO disaster of Part 12.

## FAQ

**Why does TETRA use two different convolutional codes?**
They come from two documents with two jobs: EN 300 392-2 defines the air
interface (signalling channels, R=1/4 mother), EN 300 395-2 defines the speech
codec and its channel protection (R=1/3 mother with unequal error protection
across bit classes). GopherTrunk mirrors the document boundary in its package
layout — a decoder per spec, no shared polynomial tables to get subtly wrong.

**What does the Viterbi path metric actually tell you?**
For the hard decoder it's the count of received bits the surviving path had to
disagree with — 0 is a clean burst, a handful is FEC doing its job, dozens
means the burst is likely garbage even if the CRC accidentally passes. The
replay harnesses histogram it per burst as a channel-quality profile.

**Why force the Viterbi to end in state 0 instead of picking the best final state?**
Because the encoder provably ends there — the four zero tail bits flush it.
Honouring the constraint uses information the cheapest-final-state heuristic
throws away, and on a marginal burst that difference decides whether the CRC
sees the right bits.

**Couldn't the class-2 CRC bug have been caught without a capture?**
Only by an external reference: decoding a frame encoded by someone else's
implementation, or checking the CRC of a known-good on-air burst. That's the
general lesson — a codec bug that is symmetric under round-trip is invisible
to any test you generate yourself. The conformance harnesses in Part 7 exist
precisely to make external vectors routine.

**Is the interleaver ever the thing that's wrong?**
It's the same failure class — a wrong permutation applied consistently on both
sides round-trips perfectly. The speech interleaver (`tchInterleave`, a 24×18
matrix read column-wise) and the signalling `BlockInterleaveTetra` are pinned
by tests against spec-derived positions, and the
[block-interleaver reference]({{ '/reference/tetra-block-interleaver/' | relative_url }})
documents the layouts.

## Series navigation

**Part 3 of 14** · ←
[Part 2: The Burst Zoo & the Slot Grid]({{ '/blog/deep-dives/tetra-end-to-end-02-bursts-slot-grid/' | relative_url }})
· Next →
[Part 4: Scrambling & Colour Codes — Why Colour 0 Is Not a No-Op]({{ '/blog/deep-dives/tetra-end-to-end-04-scrambling-colour-codes/' | relative_url }})
