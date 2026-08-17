---
title: "Weak-Signal Engineering, Part 8: Soft Decisions — LLRs Through Depuncture & Viterbi"
description: Why hard-slicing a marginal symbol throws away the information the FEC needs most, how GopherTrunk carries per-bit log-likelihood ratios through descramble, deinterleave, depuncture, and a correlation-metric Viterbi, and the measured TETRA outcome that justified the whole soft path.
category: deep-dives
keywords: soft decision decoding, log likelihood ratio, soft viterbi decoder, depuncturing erasures, rcpc soft decode, tetra tch/s soft, coding gain weak signal, llr sign convention, gophertrunk weak-signal engineering
tags: [weak-signal-engineering, soft-decision, viterbi, llr, fec, tetra, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Weak-Signal Engineering"
series_part: 8
---

*Part 8 of **Weak-Signal Engineering**, a 14-part series on decoding the
marginal regime — where the receiver locks but only a fraction of frames
survive. [Part 7]({{ '/blog/deep-dives/weak-signal-engineering-07-trained-lms/' | relative_url }})
trained an LMS equalizer on the TETRA midamble and re-derived cleaner symbols
from the known reference. Both equalizer parts ended the same way: with better
*symbols*. This part is about what happens after the symbol — the moment the
receiver decides a bit. Hard-slice it and you discard exactly the information a
convolutional decoder is built to exploit. Carry the confidence instead, and
bursts that failed every hard CRC start passing. On the marginal same-carrier
calls that motivated this work, hard decoding was dropping ~70% of the speech
bursts; the soft path is how they came back.*

> **TL;DR:** A hard slicer collapses each received symbol to a bit and throws
> away *how sure it was*. GopherTrunk instead carries a **log-likelihood ratio**
> per bit — sign is the decision, magnitude is the confidence, **0 is an
> erasure** — from the demodulator's π/4-DQPSK differential (the receiver's
> `SoftSink`; the two on-air bits' LLRs are the Im and Re of `s·conj(last)`)
> all the way into the FEC. Every hard channel-coding primitive has a soft
> mirror in `internal/radio/framing/soft_tetra.go`: `DescrambleTetraSoft` flips
> LLR signs, `DepunctureRCPCTetraSoft` fills punctured positions with 0.0
> erasures, and `DecodeRCPCTetraMotherSoft` runs the same 16-state K=5 trellis
> with a correlation branch metric `L·(2g−1)` instead of Hamming distance.
> `tetra.DecodeTCHSSoft` chains them for voice; the composer prefers it
> whenever LLRs exist and falls back to the hard gate when they don't.

**Key takeaways**

- **The bits the FEC needs help with are the bits the slicer is least sure
  about.** Hard decisions weight a near-threshold symbol exactly as heavily as
  a full-quieting one; the Viterbi then defends wrong-but-confident-looking
  bits. LLRs let weak evidence count weakly.
- **An erasure is the limiting case, and puncturing already forces it on you.**
  A punctured position was never transmitted; the soft depuncture writes 0.0 —
  literally "no information" — and the trellis metric skips it for free. Hard
  decoders need a sentinel; soft decoders get erasures by construction.
- **The soft Viterbi is the same trellis, cheaper to reason about.** Same
  states, same polynomials, same traceback as the hard
  `DecodeRCPCTetraMother` — only the branch cost changes, so conformance
  arguments carry over.
- **The measured verdict, not the textbook one, closed the case.** The
  classic framing says soft decisions buy ~2 dB of coding gain; what mattered
  here is that a marginal call failing ~70% of its bursts under hard decoding
  produced short, garbled recordings — and decoded as real speech soft.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| LLR convention | sign = bit, magnitude = confidence, 0 = erasure | `internal/radio/framing/soft_tetra.go` (package doc) |
| Soft source | per-symbol differential `s·conj(last)` → LLR pair | `internal/radio/tetra/receiver/receiver.go` (`Options.SoftSink`) |
| Soft descramble | PN bit 1 ⇒ flip the LLR sign | `soft_tetra.go` (`DescrambleTetraSoft`) |
| Soft depuncture | punctured positions filled with 0.0 erasures | `soft_tetra.go` (`DepunctureRCPCTetraSoft`) |
| Soft Viterbi | correlation metric on the K=5 R=1/3 trellis | `soft_tetra.go` (`DecodeRCPCTetraMotherSoft`) |
| Voice chain | deinterleave → depuncture → Viterbi → CRC | `internal/radio/tetra/tch.go` (`DecodeTCHSSoft`, `TCHSpeechFramesSoft`) |
| Preference & fallback | soft when LLRs exist, hard otherwise | `internal/voice/composer/tetra_voice.go` (`decodeTETRASpeech`) |

## In this post

- **What a hard slicer throws away** — confidence, and why the FEC wanted it.
- **The LLR contract** — one convention, held end to end.
- **Erasures and soft depuncture** — puncturing's natural home.
- **The correlation-metric Viterbi** — same trellis, different cost.
- **The measured outcome** — ~70% of a marginal call's bursts, recovered.

## What a hard slicer throws away

Picture two received symbols on a marginal TETRA carrier. One lands dead centre
of its decision region — the demodulator would bet the house on it. The other
lands a hair from the boundary — noise alone could have put it on either side.
A hard slicer emits the same thing for both: one confident-looking bit. From
that moment on, every downstream stage treats the coin-flip bit and the
certain bit as equals.

Now hand those bits to a convolutional decoder. The Viterbi algorithm is a
maximum-likelihood sequence estimator: it picks the codeword closest to what
was received. With hard bits, "closest" is Hamming distance, and a wrong
near-threshold bit costs the correct path exactly as much as a wrong
full-confidence bit would — the decoder ends up defending evidence that was
never really there. That is precisely backwards. The bits most likely to be
wrong are the ones the FEC most needs to override, and they arrive stripped
of the one attribute — low confidence — that would have let it.

This is the reliability-limited regime, distinct from the ISI-limited regime
of [Part 3]({{ '/blog/deep-dives/weak-signal-engineering-03-isi-linear-channel/' | relative_url }}):
the symbols are not smeared by the channel, they are simply noisy, and no
equalizer can help. The lever here is keeping the confidence attached to each
bit. (The two levers stack — the equalized symbols from
[Part 7]({{ '/blog/deep-dives/weak-signal-engineering-07-trained-lms/' | relative_url }})
feed *re-derived LLRs*, not re-sliced bits, into this same path.)

## The LLR contract

GopherTrunk's soft information is a per-bit **log-likelihood ratio** (see the
[reference page]({{ '/reference/log-likelihood-ratio/' | relative_url }})) held
to one convention across every soft primitive, stated once in
`internal/radio/framing/soft_tetra.go`: **LLR > 0 ⇒ bit 0, LLR < 0 ⇒ bit 1,
and magnitude is reliability — 0 is an erasure, no information at all.** The
soft Viterbi's surviving path is scale-invariant, so the LLRs never need
normalising; any consistent scaling works.

Where do LLRs come from? For π/4-DQPSK they fall out of the demodulator
almost for free. The receiver's `SoftSink` emits the complex differential
`s·conj(last)` for every symbol, aligned one-to-one with the hard dibits and
carrying the same `baseIdx` — and the two on-air bits' LLRs are simply the
imaginary and real parts of that differential. A symbol near a decision
boundary produces a small component; a clean symbol produces a large one. The
demodulator always knew the confidence; the hard path just never wrote it down.

Even the scrambler respects the convention. XOR-ing a bit with the scrambling
PN sequence becomes a conditional sign flip of its LLR:

```go
// internal/radio/framing/soft_tetra.go (shape) — DescrambleTetraSoft
s := NewScramblerTetra(colourCode)
out := make([]float32, len(in))
for i, v := range in {
    if s.Next() == 1 {
        out[i] = -v // PN bit 1 inverts the bit, hence the LLR sign
    } else {
        out[i] = v
    }
}
```

Same LFSR as the hard `DescrambleTetra`, same colour-code seeding — the
magnitude (the confidence) passes through untouched, because scrambling
changes what the bit *is*, not how sure we are of it.

## Erasures and soft depuncture

TETRA's traffic channel uses a rate-compatible punctured convolutional code:
a K=5 rate-1/3 mother code with positions deleted before transmission to hit
the target rate. The receiver must re-inflate the stream to mother length
before the trellis — and the punctured positions were *never sent*, so there
is genuinely nothing to put there.

The hard decoder needs a special sentinel value for this. The soft decoder
gets it for free, because "no information" already has a number:

```go
// internal/radio/framing/soft_tetra.go (shape) — DepunctureRCPCTetraSoft
out := make([]float32, motherLen) // zero-valued = erasure
for j := 1; j <= len(punctured); j++ {
    blockIdx := (j - 1) / t
    offset := j - t*blockIdx
    k := period*blockIdx + puncture[offset-1]
    if k-1 < motherLen {
        out[k-1] = punctured[j-1]
    }
}
```

A 0.0 LLR contributes exactly zero to every branch metric, so a punctured
position neither helps nor hurts any path — which is the correct treatment of
a bit that never existed. The same mechanism handles a received bit the demod
had *no* confidence in: as its LLR magnitude shrinks toward zero it smoothly
becomes an erasure. Erasures are not a special case in the soft domain; they
are the limiting case of low confidence, and the code reflects that by having
no erasure branch at all.

## The correlation-metric Viterbi

The soft trellis decoder is deliberately the *same machine* as the hard one —
same 16 states, same generator polynomials (`g1=in^d1^d2^d3^d4`,
`g2=in^d1^d3^d4`, `g3=in^d2^d4`), same forced-to-zero tail and traceback. The
only change is the branch cost: instead of counting Hamming mismatches, each
expected mother bit `g` is correlated against its received LLR `L` as
`L·(2g−1)` — a match lowers the path metric, a mismatch raises it by an amount
proportional to how confident the demod was, and an erasure contributes
nothing:

```go
// internal/radio/framing/soft_tetra.go (shape) — DecodeRCPCTetraMotherSoft
for input := 0; input < 2; input++ {
    g1 := (input ^ d1 ^ d2 ^ d3 ^ d4) & 1
    g2 := (input ^ d1 ^ d3 ^ d4) & 1
    g3 := (input ^ d2 ^ d4) & 1
    cost := pm[cur]
    cost += rxG1 * float32(2*g1-1)
    cost += rxG2 * float32(2*g2-1)
    cost += rxG3 * float32(2*g3-1)
    /* … keep the minimum-cost survivor per next state … */
}
```

The minimum-cost path is the maximum-likelihood sequence, weighted by the
demodulator's actual confidence in every bit. `tetra.DecodeTCHSSoft` chains
the pieces for a voice burst, mirroring the hard chain step for step in the
LLR domain: soft deinterleave → split the class-0/class-1/class-2 regions →
soft depuncture each into the mother stream → soft Viterbi → the same hard
class-2 CRC check as before. The uncoded class-0 bits carry no FEC, so their
LLRs are hard-sliced straight through — softness only pays where there is a
decoder to spend it.

That final CRC matters for the series' standing rule
([Part 2]({{ '/blog/deep-dives/weak-signal-engineering-02-metrics-that-lie/' | relative_url }})):
the verdict on whether soft decoding *helped* is CRC-valid frames per
opportunity, judged by exactly the same gate the hard path uses.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="Two number lines of received symbol values. The top line shows hard slicing: a single threshold splits the axis into bit 0 and bit 1, and two received samples on opposite sides — one far from the threshold, one touching it — both emit full-confidence bits. The bottom line shows the LLR view: confidence grows with distance from zero, the far sample carries a large LLR, the near sample carries a tiny LLR close to the erasure point at zero, so the Viterbi can override it.">
  <text x="12" y="26" fill="currentColor" font-size="10">hard slicer: every bit looks equally certain</text>
  <line x1="60" y1="60" x2="620" y2="60" stroke="var(--fg-muted)"/>
  <line x1="340" y1="42" x2="340" y2="78" stroke="currentColor"/>
  <text x="340" y="36" text-anchor="middle" fill="currentColor" font-size="9">threshold</text>
  <text x="180" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="9">bit 0</text>
  <text x="500" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="9">bit 1</text>
  <circle cx="130" cy="60" r="5" fill="currentColor"/>
  <text x="130" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="9">clean symbol → "0"</text>
  <circle cx="332" cy="60" r="5" fill="currentColor"/>
  <text x="300" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="9">coin flip → also "0"</text>
  <text x="12" y="122" fill="currentColor" font-size="10">LLR: confidence rides along, zero is an erasure</text>
  <line x1="60" y1="164" x2="620" y2="164" stroke="var(--fg-muted)"/>
  <rect x="60" y="150" width="280" height="28" fill="var(--accent)" opacity="0.25"/>
  <rect x="340" y="150" width="280" height="28" fill="var(--fg-muted)" opacity="0.18"/>
  <line x1="340" y1="146" x2="340" y2="182" stroke="currentColor"/>
  <text x="340" y="198" text-anchor="middle" fill="currentColor" font-size="9">LLR = 0 (erasure)</text>
  <circle cx="130" cy="164" r="5" fill="var(--accent)"/>
  <text x="130" y="142" text-anchor="middle" fill="var(--accent)" font-size="9">large +LLR: trust it</text>
  <circle cx="332" cy="164" r="5" fill="var(--accent)"/>
  <text x="270" y="142" text-anchor="middle" fill="var(--accent)" font-size="9">tiny +LLR: FEC may override</text>
  <text x="120" y="216" fill="var(--fg-muted)" font-size="9">LLR &gt; 0 ⇒ bit 0</text>
  <text x="480" y="216" fill="var(--fg-muted)" font-size="9">LLR &lt; 0 ⇒ bit 1</text>
</svg>
<figcaption>A hard threshold makes a coin-flip symbol indistinguishable from a certain one; the LLR keeps the distance to the boundary, and zero — the erasure — is just its limiting case.</figcaption>
</figure>

## The measured outcome

The textbook framing says soft-decision decoding is worth roughly 2 dB of
coding gain over hard decisions in Gaussian noise — meaningful, but abstract.
What landed the change was a concrete failure: on a marginal same-carrier
TETRA call, the hard-decision TCH/S gate was failing **about 70% of the
call's bursts**, and the recordings came out short and garbled. Same RF, same
symbols, LLRs carried instead of bits: the bursts passed the class-2 CRC and
the call decoded as continuous speech. `TestSoftDecisionLocksWhereHardFails`
(`internal/radio/tetra/receiver/soft_e2e_test.go`) pins the effect end to end
at the receiver level, and its companion `TestSoftDecisionCleanStillLocks`
pins the no-harm side.

The composer's preference logic is the last piece, and it encodes the safety
rule this series keeps returning to — the soft path must be additive:

```go
// internal/voice/composer/tetra_voice.go (shape) — decodeTETRASpeech
// Prefer soft-decision TCH/S when the extractor supplied the burst's LLRs:
// the soft Viterbi's ~2 dB coding gain recovers real speech bursts the
// hard-decision gate drops on a marginal same-carrier signal. Fall back to
// the hard gate when no soft info is available (never worse than before).
var frames [][]byte
if softType5 != nil {
    frames = tetra.TCHSpeechFramesSoft(softType5)
} else {
    frames = tetra.TCHSpeechFrames(frame)
}
```

No LLRs stashed ⇒ the hard path runs untouched, byte for byte. How that
opt-in property is engineered — the parallel buffers, the stash bridge, and
the tests that pin "no sinks wired ⇒ identical output" — is the next part.
For the TETRA-side case narrative of this same lever, see the concurrent
[TETRA End to End, Part 8]({{ '/blog/deep-dives/tetra-end-to-end-08-soft-decision-tchs/' | relative_url }}).

### How that principle shaped the Go code

- **Every soft primitive mirrors a hard one, by name.** `DescrambleTetraSoft`,
  `BlockDeinterleaveTetraSoft`, `DepunctureRCPCTetraSoft`,
  `DecodeRCPCTetraMotherSoft` — same file layout, same index math, same
  constants. A conformance argument made once for the hard chain transfers.
- **One convention, zero normalisation.** Sign/magnitude/erasure is stated in
  one package doc and relied on everywhere; because the Viterbi survivor is
  scale-invariant, no stage rescales LLRs and no stage can get the scaling
  wrong.
- **The verdict gate did not move.** Soft or hard, a burst becomes speech only
  by passing the identical class-2 CRC — so the yield comparison between the
  two paths is honest by construction.

## Where this goes next

Soft decisions only work if the LLRs survive the trip from demodulator to
decoder — through a burst extractor that was designed around hard dibits.
[Part 9]({{ '/blog/deep-dives/weak-signal-engineering-09-parallel-buffers/' | relative_url }})
covers the architecture that made it landable: `SoftSink` and `SymbolSink`
buffers carried strictly parallel to the dibit buffer, a stash bridge keyed by
`baseIdx`, and the byte-identical opt-out property that let risky DSP ship
without endangering a single working configuration.

## FAQ

**Where do the LLRs physically come from — does the demodulator run twice?**
No. The π/4-DQPSK differential `s·conj(last)` is computed once; the hard path
slices it into a dibit and the `SoftSink` hands the same complex value out as
a pair of LLRs (Im and Re). The soft information was always present in the
demodulator — the change is refusing to discard it.

**Is soft decoding slower?**
The trellis work is the same order — 16 states, two branches per state, three
metric terms per branch, float adds instead of integer XOR/popcount. The cost
that actually shows up is carrying the parallel float32 buffers, and that is
only paid when a caller wires the sinks (Part 9).

**Why does class 0 get hard-sliced even on the soft path?**
Class-0 bits are uncoded — there is no decoder downstream to spend their
confidence. An LLR only earns its keep where a trellis or an erasure-aware
stage can weigh it; for an uncoded bit the sign is all there is.

**Does soft decoding fix an ISI-smeared channel?**
No. If the constellation is smeared by multipath or band-edge group delay,
the LLRs faithfully report high confidence in *wrong* bits. Equalize first
([Parts 4–7]({{ '/blog/deep-dives/weak-signal-engineering-04-blind-cma/' | relative_url }})),
then let soft decisions handle the residual noise. The levers stack; they do
not substitute.

**Is ~2 dB worth the engineering?**
On the yield cliff of [Part 1]({{ '/blog/deep-dives/weak-signal-engineering-01-marginal-regime/' | relative_url }}),
2 dB is the difference between a call that decodes and one that doesn't —
the marginal regime is exactly where small dB moves produce large yield moves.
The 70%-of-bursts recovery is that cliff, measured.

## Series navigation

**Part 8 of 14** · ←
[Part 7: Trained Equalization — LMS on the Midamble]({{ '/blog/deep-dives/weak-signal-engineering-07-trained-lms/' | relative_url }})
· Next →
[Part 9: Parallel Buffers — SymbolSink, SoftSink & Opt-In Soft Paths]({{ '/blog/deep-dives/weak-signal-engineering-09-parallel-buffers/' | relative_url }})
