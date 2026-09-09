---
title: "From Spec to Shipping, Part 4: The Conformance Harness — Bit-Identical or Bust"
description: "Building a conformance harness against the ETSI EN 300 395-2 reference codec: one 137-bit bitstream into both decoders, zero PCM mismatches allowed, the LP64 Word32 gotcha that makes the reference itself lie, and why conformance at two layers brackets everything in between."
category: deep-dives
keywords: etsi reference codec conformance, bit exact vocoder test, acelp reference decoder, tetra en 300 395-2, fixed point codec port, word32 lp64 typedef long, skip guarded test harness, validate whole decode chain, gophertrunk from spec to shipping
tags: [from-spec-to-shipping, acelp, vocoder, conformance, testing, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From Spec to Shipping"
series_part: 4
---

*Part 4 of **From Spec to Shipping**, a 14-part series on how a protocol
decoder actually gets written — from standards documents and independent
references to code you can trust on air.
[Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
pinned parsers with literal byte vectors — discrete facts, one field at a
time. But a vocoder is tens of thousands of saturating fixed-point
operations whose correctness only shows in the aggregate, and for that the
vector becomes a stream and the assertion becomes absolute: feed your
decoder and the reference the same bitstream and demand **bit-identical
output**. This part builds that harness around GopherTrunk's clean-room
TETRA ACELP decoder — and meets the build gotcha that makes the reference
itself produce garbage on a modern machine.*

> **TL;DR:** `internal/voice/acelp/etsi_reference_test.go` is the
> authoritative check on GopherTrunk's TETRA vocoder: the ETSI EN 300 395-2
> reference tools (`scoder`/`sdecoder`) produce a serial bitstream (int16
> LE, **138 words per frame** — 1 BFI flag + 137 coded speech bits) and a
> reference PCM file (**240 samples per frame**); the harness feeds the
> same bits to GopherTrunk's `Decoder.Decode` and requires **zero sample
> mismatches** — fixed-point integer maths means a faithful port matches
> exactly, so "close" is failure. Two traps: build the reference with
> `Word32` as a 32-bit int (on LP64 the default `typedef long` is 64-bit
> and **every saturating op returns garbage**), and remember conformance
> only covers the layer you tested — the class-2 CRC bug that silently
> dropped every on-air TCH/S burst lived *outside* the vocoder, which is
> why a second pass (`TestTETRAMultiSlotReplay`, a real cs16 capture to
> per-slot audio) brackets the chain from the other end.

**Key takeaways**

- **Fixed-point references permit an exact assertion — take it.** The
  reference codec is integer arithmetic with defined saturation; a faithful
  port matches sample-for-sample, so the pass criterion is 0 mismatches,
  not a similarity score that would hide systematic drift.
- **The reference can lie if you build it wrong.** ETSI's basic ops assume
  a 32-bit `long`; compiled untouched on LP64, saturation never fires and
  the "reference" output is garbage. Validate the oracle before validating
  against it.
- **Skip-guarded harnesses keep CI honest and results reproducible.** The
  test skips unless env vars point at reference vectors — CI stays green
  without copyrighted ETSI sources in the tree, and anyone can re-run the
  exact conformance pass locally.
- **Conformance at one layer proves that layer only.** The vocoder passed
  while on-air voice decoded nothing — the bug was in the channel coding
  above it. Bracket the chain: bit-exact at the bottom, capture-replay at
  the top, and a defect must live between two pinned surfaces.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Bit-exact conformance | same 137-bit frames → GT and ETSI ref → 0 PCM mismatches | `internal/voice/acelp/etsi_reference_test.go` (`TestETSIReferenceConformance`) |
| Reference tools | `scoder` / `sdecoder` from EN 300 395-2 | ETSI distribution (copyrighted, not committed) |
| Serial format | int16 LE, 138 words/frame: BFI + 137 bits | test doc comment; matches `Bits2prm_Tetra` |
| Output alignment | reference `Post_Process` = saturating ×2 | `postProcessX2` in the harness |
| The build gotcha | `Word32` must be 32-bit; LP64 `long` breaks saturation | test comment; GT mirror in `acelp/ops.go` (`maxWord32`) |
| Second-layer pass | real cs16 IQ → receiver → TCH/S → per-slot PCM | `cmd/gophertrunk/tetra_multislot_replay_test.go` (`TestTETRAMultiSlotReplay`) |
| The layer in between | TCH/S channel decode, class-2 CRC | `internal/radio/tetra/tch.go` |

## In this post

- **Why bit-identical, not "sounds right"** — what an exact assertion buys.
- **The harness, end to end** — reference tools, serial format, the
  comparison loop.
- **The Word32 gotcha** — when the oracle itself is broken.
- **Skip-guarded but reproducible** — copyrighted vectors and honest CI.
- **Bracketing: conformance at two layers** — and the CRC bug that proved
  why one layer is not enough.

## Why bit-identical, not "sounds right"

A vocoder invites the worst validation standard in engineering: play the
output and listen. Speech is intelligible through astonishing amounts of
systematic error — a wrong gain table, a swapped codebook, crushed
high-band energy — so "sounds okay" bounds almost nothing. GopherTrunk has
a measured case on file: the AMBE+2 3600×2450 decode produced perfectly
intelligible speech with 4–10× less energy above 1 kHz than mbelib's decode
of the *same* frames — content intact, spectrum systematically wrong, and
only octave-band measurement against a reference made the deficit visible
(the [vocoders guide]({{ '/vocoders.html' | relative_url }}) documents it).

EN 300 395-2 removes the excuse for that standard entirely, because the
codec it defines is **fixed-point integer arithmetic** — every multiply,
add and shift saturates to defined 16- and 32-bit bounds. There is no
floating-point wiggle, no platform rounding: a faithful implementation
produces the same int16 as the reference at every sample index, forever.
So the harness demands exactly that:

```go
// internal/voice/acelp/etsi_reference_test.go (shape) — TestETSIReferenceConformance
dec := NewDecoder()
for f := 0; f < nFrames; f++ {
    bfi := serial[base] != 0          // word 0: bad-frame indicator
    /* … words 1..137 → bits … */
    out := dec.Decode(bits, bfi)      // 240 samples
    for i, s := range out {
        got := postProcessX2(s)       // mirror the reference Post_Process
        if got != ref[f*pcmPerFrame+i] {
            mismatches++
        }
    }
}
if mismatches != 0 {
    t.Errorf("ACELP decode diverges from the ETSI reference: %d/%d samples differ …")
}
```

**Zero.** Not a correlation threshold, not an SNR floor. The moment the
criterion is exact, every class of subtle bug — an off-by-one in a table, a
saturation applied one op too late, an interpolation half-done — becomes a
loud, countable failure with a first-bad-frame index to start debugging
from. The harness logs `mismatches`, `maxAbsDiff` and `firstBadFrame` for
exactly that reason: a conformance failure should arrive with its own
triage data.

One alignment detail earns its comment: the reference decoder's
`Post_Process` applies a saturating ×2 to every output sample after
`Decod_Tetra`. GopherTrunk omits that scaling in production, so the harness
applies `postProcessX2` to GT's output before comparing — the *harness*
absorbs representation differences, keeping the shipped decoder clean and
the comparison exact.

## The harness, end to end

The moving parts, and the one contract binding them:

<figure class="lab-figure">
<svg viewBox="0 0 680 240" width="680" height="240" role="img" aria-label="The conformance harness: one serial bitstream of 138 int16 words per frame, produced by the ETSI scoder, feeds two decoders in parallel — the ETSI sdecoder reference and GopherTrunk's clean-room ACELP decoder. The reference emits 240 PCM samples per frame; GopherTrunk's output passes through a saturating times-two post-process, then both meet a comparator whose only passing verdict is zero mismatched samples.">
  <rect x="230" y="10" width="220" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="340" y="24" text-anchor="middle" fill="currentColor" font-size="10">serial.bin — int16 LE, 138 words/frame</text>
  <text x="340" y="38" text-anchor="middle" fill="var(--fg-muted)" font-size="10">word 0 = BFI · words 1..137 = coded bits</text>
  <line x1="290" y1="44" x2="180" y2="78" stroke="currentColor"/><polygon points="184,72 174,80 186,84" fill="currentColor"/>
  <line x1="390" y1="44" x2="500" y2="78" stroke="currentColor"/><polygon points="494,72 506,80 494,84" fill="currentColor"/>
  <rect x="60" y="82" width="240" height="44" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="180" y="100" text-anchor="middle" fill="currentColor" font-size="10">ETSI sdecoder (reference C)</text>
  <text x="180" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="10">Word32 MUST be 32-bit — LP64 breaks it</text>
  <rect x="390" y="82" width="230" height="44" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="505" y="100" text-anchor="middle" fill="currentColor" font-size="10">GT clean-room decoder (Go)</text>
  <text x="505" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="10">internal/voice/acelp — Decoder.Decode</text>
  <line x1="180" y1="126" x2="180" y2="160" stroke="currentColor"/><polygon points="176,158 180,166 184,158" fill="currentColor"/>
  <line x1="505" y1="126" x2="505" y2="140" stroke="currentColor"/><polygon points="501,138 505,146 509,138" fill="currentColor"/>
  <rect x="430" y="146" width="150" height="22" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="505" y="161" text-anchor="middle" fill="var(--fg-muted)" font-size="10">postProcessX2 (saturating ×2)</text>
  <line x1="505" y1="168" x2="505" y2="180" stroke="currentColor"/>
  <line x1="180" y1="166" x2="180" y2="180" stroke="currentColor"/>
  <rect x="150" y="180" width="380" height="34" rx="6" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="340" y="194" text-anchor="middle" fill="var(--accent)" font-size="10">comparator: 240 int16 samples/frame, every frame</text>
  <text x="340" y="208" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">pass = 0 mismatches — bit-identical or bust</text>
  <text x="340" y="232" text-anchor="middle" fill="var(--fg-muted)" font-size="10">logged on failure: mismatches, maxAbsDiff, firstBadFrame — a failure arrives with its own triage data</text>
</svg>
<figcaption>One bitstream, two decoders, one comparator — and the only passing verdict is zero, because fixed-point arithmetic leaves nothing to round.</figcaption>
</figure>

The serial format is the contract: int16 little-endian, 138 words per
frame — word 0 the bad-frame indicator, words 1–137 the coded speech bits,
exactly what the reference's `Bits2prm_Tetra` reads and what GT's
`Decoder.Decode` consumes. Because the format is the reference's own, the
same `serial.bin` drives both sides with no adapter that could smuggle in
an assumption — [Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})'s
rule about generated fixtures, applied at stream scale. Feeding BFI frames
through the same path also exercises the error-concealment machinery under
the same exact criterion — muted and interpolated frames must match too.

## The Word32 gotcha: validate the oracle

The reference C code was written when `long` meant 32 bits. Its `Word32`
is `typedef long`, and every saturating basic op — `L_add`, `L_sub`, the
lot — detects overflow by wrapping behavior at the 32-bit boundary. Compile
it unmodified on a 64-bit Linux host (LP64: `long` is 64-bit) and nothing
overflows where the algorithm expects: **every saturating op returns
garbage**, silently, while the tools run happily and produce plausible
PCM. The harness's doc comment carries the fix because it cost real time:
build with `Word32` as a 32-bit `int` — edit the typedef or build `-m32`.

The general lesson outranks the specific one: **a reference implementation
is itself a build artifact that can be wrong on your machine.** Before
trusting an oracle, make it prove itself — the ETSI encoder emits its own
locally-decoded synthesis (`synth_local.pcm`), so encoder and decoder can
be cross-checked against each other before either judges your port. A
conformance harness that skips this step can "fail" a correct decoder
against a broken reference and send you debugging working code — or worse,
"pass" a broken pair.

GopherTrunk's own side of that arithmetic lives in
`internal/voice/acelp/ops.go`: explicit `maxWord32`/`minWord32` saturation
in Go, where int32 is int32 on every platform — one of the quiet arguments
for the pure-Go port over binding the reference
([Part 5]({{ '/blog/deep-dives/from-spec-to-shipping-05-clean-room-rules/' | relative_url }})
takes up the clean-room side of that decision).

## Skip-guarded but reproducible

The ETSI sources and vectors are copyrighted and not committed, so the
test skips unless `GT_ETSI_SERIAL` and `GT_ETSI_REF` point at files
produced by the reference tools. That single design choice buys three
things at once:

- **CI stays green and honest** — no fixture of dubious licensing in the
  tree, no silently-degraded approximation of the check.
- **Anyone can reproduce the authoritative result** — the doc comment
  gives the exact `scoder`/`sdecoder` invocations; the claim "bit-exact
  against the reference" is a command away from re-verification, not
  folklore.
- **The harness doubles as a field instrument** — the same env-var pattern
  scales up to capture replays (`GT_TETRA_IQ`, `GT_TETRA_DMO_IQ`, …) that
  operators run against their own recordings, a practice
  [Part 11]({{ '/blog/deep-dives/from-spec-to-shipping-11-capture-driven-development/' | relative_url }})
  builds a whole workflow on.

The one discipline a skip-guard demands: someone must actually run it.
A skipped conformance test asserts nothing, so the result gets recorded
where the next engineer will find it — in GopherTrunk's case, the
project's institutional notes state the vocoder cleared two independent
conformance passes, with the harness names attached
([TETRA End to End Part 7]({{ '/blog/deep-dives/tetra-end-to-end-07-etsi-conformance/' | relative_url }})
walks through the runs).

## Bracketing: conformance at two layers

Here is the trap conformance sets for you: the vocoder passes bit-exact,
so when on-air voice produces silence, the vocoder is *ruled out* — and
the temptation is to conclude the whole voice path is fine. GopherTrunk
lived the counterexample. With the ACELP decoder conformance-clean, real
TETRA calls still yielded almost nothing, because the **class-2 CRC in the
TCH/S channel decode** (`internal/radio/tetra/tch.go`, one layer *above*
the vocoder) had been implemented as a generator-polynomial LFSR when the
real check is a fixed parity-check matrix — the reference's `TAB_CRC`
tables. Synthetic round-trips passed (both sides shared the wrong CRC —
[Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})'s
villain again), while every on-air burst was silently dropped. The story
is told in full in
[TETRA End to End Part 5]({{ '/blog/deep-dives/tetra-end-to-end-05-tchs-traffic-channel/' | relative_url }}).

The structural answer is a **second conformance pass at a different
layer**:

| | Bottom pass | Top pass |
|---|---|---|
| Harness | `TestETSIReferenceConformance` | `TestTETRAMultiSlotReplay` |
| Input | reference `serial.bin` bitstream | real cs16 IQ capture (`GT_TETRA_IQ`) |
| Oracle | ETSI reference decoder | the air: CC grant timeslots vs per-slot audio |
| Assertion | 0 PCM mismatches | CRC-valid speech clustered on granted slots |

The replay harness runs the receiver, the traffic extractor, the TCH/S
channel decode and the vocoder end to end, printing a per-slot activity
timeline to cross-check against the control channel's grants. With the
bottom pinned bit-exact and the top pinned against real air, a defect has
nowhere to live except *between* the two pinned surfaces — and the
diagnosis rule falls out for free: **when "voice doesn't decode" but the
vocoder unit tests pass, suspect the channel coding — CRC, interleave,
reorder — and validate the whole chain, not parts**
([Voice Coding Part 12]({{ '/blog/deep-dives/voice-coding-12-calibration-testing/' | relative_url }})
applies the same bracketing to the MBE stack).

### How that principle shaped the Go code

- **The exactness lives in the assertion, not the prose.** `mismatches != 0`
  fails the test; no tolerance parameter exists to loosen quietly.
- **Representation differences are absorbed by the harness.**
  `postProcessX2` mirrors the reference's output stage in the *test*, so
  the shipped decoder carries no reference-shaped warts.
- **Oracles are documented as build artifacts.** The Word32 note rides in
  the test's doc comment — the next person to build the reference on LP64
  hits the warning before the garbage.
- **Every conformance claim names its harness.** "Bit-exact" appears in
  the notes only alongside `TestETSIReferenceConformance` and
  `TestTETRAMultiSlotReplay` — a claim without a re-runnable check is
  treated as unverified, the standard
  [/learn/testing/]({{ '/learn/testing/' | relative_url }}) builds toward.

## Where this goes next

This harness runs a copyrighted reference *as a binary oracle* while
GopherTrunk ships a clean-room Go implementation — and that boundary is
load-bearing, legally and technically.
[Part 5]({{ '/blog/deep-dives/from-spec-to-shipping-05-clean-room-rules/' | relative_url }})
draws the line in practice: implementing from the spec, using other
people's decoders as validation oracles rather than source to translate,
and the licensing hygiene that keeps test-only dependencies out of the
shipped binary.

## FAQ

**What if my target codec has no reference implementation?**
Fall back one rung on Part 2's ladder: a proven decoder becomes a
behavioral reference (GopherTrunk uses mbelib/DSD-FME this way for
IMBE/AMBE, comparing octave-band energy rather than bits, since MBE
synthesis is floating-point). The assertion weakens from bit-identical to
measured-and-bounded — document that difference, because it changes what
a "pass" proves.

**Why not commit the reference vectors so CI runs the conformance test?**
Licensing — the ETSI sources and their outputs are copyrighted. The
skip-guard pattern is the honest compromise: CI runs everything it can
own, and the authoritative pass is reproducible by anyone holding the
reference, with the exact commands documented in the test.

**Is bit-identical ever the wrong goal?**
For floating-point algorithms, yes — platform rounding makes exactness
brittle, and a bounded-error criterion is correct. But for fixed-point
codecs, exact is *easier* to maintain than approximate: any nonzero
mismatch count is a bug with a frame number attached, and there is no
threshold to argue about when one appears.

**How do I debug a conformance failure with thousands of mismatches?**
Start at `firstBadFrame` — fixed-point decoders carry state, so the first
divergence is the real defect and everything after may be cascade. Then
bisect by stage: the reference tools print or can be made to print
intermediate parameters (LSPs, gains, codebook indices), so you can find
the first *parameter* that differs, not just the first sample.

**Does passing conformance mean the vocoder work is done?**
It means one layer is done. The class-2 CRC story is the standing proof
that a conformance-clean vocoder can sit inside a voice path that decodes
nothing — and the reverse trap, green synthetics above a broken layer, is
[Part 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})'s
whole subject. Bracket with an end-to-end pass, then let real captures
referee.

## Series navigation

**Part 4 of 14** · ←
[Part 3: Literal Vectors, Not Round-Trips]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
· Next →
[Part 5: Clean-Room Rules — Reading Without Copying]({{ '/blog/deep-dives/from-spec-to-shipping-05-clean-room-rules/' | relative_url }})
