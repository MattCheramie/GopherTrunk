---
title: "TETRA End to End, Part 8: Going Soft — Soft-Decision TCH/S"
description: Why hard-decision decoding threw away roughly seventy percent of a marginal same-carrier TETRA call's traffic bursts, and how the soft path — receiver differentials carried in lockstep through the traffic extractor into a soft Viterbi — recovers them without touching the hard contract.
category: deep-dives
keywords: tetra soft decision, tch/s soft decode, llr viterbi decoding, soft depuncture rcpc, pi/4-dqpsk soft information, softsink differentials, tetra traffic extractor, soft decision coding gain, gophertrunk tetra
tags: [tetra-end-to-end, tetra, soft-decision, viterbi, dsp, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "TETRA End to End"
series_part: 8
---

*Part 8 of **TETRA End to End**, a 14-part deep dive into how GopherTrunk turns
one real 25 kHz TETRA carrier into clear recorded voice.
[Part 7]({{ '/blog/deep-dives/tetra-end-to-end-07-etsi-conformance/' | relative_url }})
closed the conformance loop — bit-identical PCM against the ETSI reference codec,
twice over — so the vocoder and the channel coding are proven. And yet a marginal
same-carrier call still came out short and garbled. The chain was correct; it was
also throwing away information at the very first decision it made. This part is
about keeping that information: the soft-decision TCH/S path, which carries the
demodulator's *confidence* all the way into the Viterbi decoder instead of
flattening it into bits at the slicer.*

> **TL;DR:** Hard-decision TCH/S decoding failed **~70%** of a marginal
> same-carrier call's bursts — every slicer decision discarded how *sure* the
> demod was. The soft path keeps that confidence: the receiver's **`SoftSink`**
> emits the complex π/4-DQPSK differential `s·conj(prev)` per symbol, the
> **`TrafficExtractor`** carries it in strict lockstep with the dibits
> (**`StashSoft`**, `softBuf`), `softType5FromDiffs` turns each differential into
> two per-bit LLRs, `framing.DescrambleTetraSoft` applies the colour-code sign
> flips, and **`tetra.DecodeTCHSSoft`** runs soft depuncture + soft Viterbi
> (`framing.DecodeRCPCTetraMotherSoft`) to the same class-2 CRC gate. The
> composer tries soft first and falls back to the hard `TCHSpeechFrames` when no
> soft info was stashed — the hard path is byte-identical to before.

**Key takeaways**

- **A hard slicer is an information shredder.** The differential's angle says
  *which* dibit; its magnitude and distance from the decision boundary say *how
  confident*. Hard decision keeps the first and burns the second — exactly the
  part a Viterbi decoder can spend.
- **The soft information is the differential itself.** For π/4-DQPSK the two
  on-air bits' LLRs are the imaginary and real parts of `s·conj(prev)` — no
  separate estimator, the demod already computed it.
- **Lockstep or nothing.** `softBuf` is either exactly parallel to the dibit
  buffer or empty; on any misalignment the extractor drops the soft path for
  that burst rather than decode with shifted LLRs. Misaligned soft data is worse
  than none.
- **The CRC is still the gate.** `TCHSpeechFramesSoft` returns nil on a class-2
  CRC failure exactly like the hard gate — soft decision recovers more real
  bursts; it never admits fake ones.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Soft emission | per-symbol complex differential, 1:1 with dibits | `internal/radio/tetra/receiver/receiver.go` (`Options.SoftSink`) |
| Soft carry | stash differentials for the next `Process` call | `internal/radio/tetra/traffic.go` (`TrafficExtractor.StashSoft`) |
| Lockstep buffer | `softBuf` strictly parallel to `buf`, or empty | `internal/radio/tetra/traffic.go` (`Process`, `softFrame`) |
| Diff → LLR | rotation-aware differential to per-bit LLRs | `internal/radio/tetra/process.go` (`softType5FromDiffs`) |
| Soft descramble | colour-code sign flips in the LLR domain | `internal/radio/framing/soft_tetra.go` (`DescrambleTetraSoft`) |
| Soft TCH/S decode | soft deinterleave → depuncture → Viterbi → CRC | `internal/radio/tetra/tch.go` (`DecodeTCHSSoft`) |
| Composer fallback | soft first, hard `TCHSpeechFrames` otherwise | `internal/voice/composer/tetra_voice.go` (`decodeTETRASpeech`) |
| Soft AACH rescue | soft RM(30,14) recovers a marginal usage marker | `internal/radio/tetra/traffic.go` (`usageOfSoft`) |

## In this post

- **What hard decision costs** — the ~70% figure and where it comes from.
- **The differential is the soft information** — LLRs for free from the demod.
- **Carrying LLRs in lockstep** — the stash bridge and its alignment contract.
- **The soft decode chain, step for step** — `DecodeTCHSSoft` mirrors the hard chain.
- **Fallback, CRC gates, and the AACH bonus** — where soft helps beyond speech.

## What hard decision costs

[Part 5]({{ '/blog/deep-dives/tetra-end-to-end-05-tchs-traffic-channel/' | relative_url }})
built the hard-decision TCH/S path: slice each burst's BKN1+BKN2, descramble,
deinterleave, depuncture, Viterbi, check the class-2 CRC. On a clean carrier it
works. On a *marginal* same-carrier call — the voice riding the same 25 kHz
carrier as the control channel, at the edge of the receiver's budget — it failed
roughly **70% of the call's bursts**, and the recordings came out short and
garbled. The vocoder was fine; Part 7 proved that. The bursts were real; the
training-sequence correlator found them. The losses happened inside the FEC.

The reason is the first decision the pipeline makes. The demodulator produces a
complex differential per symbol, and the slicer quantizes it to one of four
dibits. A symbol sitting dead-center in its decision region and a symbol
grazing the boundary produce the *same* dibit — the slicer reports the verdict
and destroys the confidence. A convolutional decoder is precisely the machine
that can spend that confidence: a Viterbi search weighing each received bit by
its reliability will happily overrule two shaky bits on the strength of twelve
solid ones. Feeding it hard bits forces every bit to count equally, and the
textbook cost of that is about **2 dB** of coding gain — which, at the margin
this call lived at, is the difference between 30% yield and a usable recording.
The general theory — LLRs, why the gain concentrates exactly at the margin —
lives in
[Weak-Signal Engineering Part 8]({{ '/blog/deep-dives/weak-signal-engineering-08-soft-decisions/' | relative_url }});
this post is the TETRA case that motivated it.

## The differential is the soft information

The elegant part of doing this for π/4-DQPSK is that the soft information costs
nothing to produce. The demod already forms `s·conj(prev)` for every symbol —
that *is* the differential decode from
[Part 1]({{ '/blog/deep-dives/tetra-end-to-end-01-pi4-dqpsk-carrier/' | relative_url }}).
And in that complex number, the two on-air bits' log-likelihood ratios are
simply the imaginary and real components. The receiver exposes it behind one
optional callback:

```go
// internal/radio/tetra/receiver/receiver.go (shape) — Options
// SoftSink, when non-nil, receives the complex π/4-DQPSK differential
// (s·conj(last)) for each symbol, aligned 1:1 with the dibits emitted
// to DibitSink and carrying the same baseIdx. It is the soft
// information for soft-decision channel decoding (the two on-air bits'
// LLRs are Im and Re of the differential). Emitted just before the
// matching DibitSink call. nil ⇒ no soft emission, zero overhead.
SoftSink func(diffs []complex64, baseIdx int)
```

`softType5FromDiffs` (in `process.go`) does the conversion, taking a rotation
parameter because the constellation can sit at any of four residual rotations —
its hard-slice is defined to equal the hard dibit path exactly, so the two
streams can never disagree about *which* bits, only about how much to trust
them. The convention throughout `framing/soft_tetra.go` is **LLR > 0 ⇒ bit 0**,
magnitude = reliability, and an exact 0.0 is an erasure — which is also what
soft depuncturing inserts for the bits the puncturing pattern never transmitted.
That is strictly more honest than the hard path, which has to *guess* a value
for punctured positions.

## Carrying LLRs in lockstep

The receiver's `DibitSink` contract predates all of this, and half the callers
(tests, hard-only paths) neither know nor care about soft data. So the soft
stream rides a stash bridge instead of a changed signature: `SoftSink` fires
just before the matching `DibitSink` call with the same `baseIdx`, the pipeline
stashes the differentials, and the extractor picks them up on its next
`Process`:

```go
// internal/radio/tetra/traffic.go (shape) — the lockstep contract
func (te *TrafficExtractor) StashSoft(diffs []complex64, baseIdx int) {
    te.pendingSoft = diffs
    te.pendingSoftBase = baseIdx
}

// Process: append the stashed differentials ONLY when they match this dibit
// block (same base + length) AND softBuf is already in lockstep with buf;
// otherwise drop the soft path (reset to empty) rather than risk misalignment.
if te.pendingSoft != nil && te.pendingSoftBase == baseIdx &&
    len(te.pendingSoft) == len(dibits) && len(te.softBuf) == len(te.buf) {
    te.softBuf = append(te.softBuf, te.pendingSoft...)
} else {
    te.softBuf = te.softBuf[:0]
}
```

The invariant is all-or-nothing: `softBuf` is either **exactly** `len(buf)` or
empty. Every trim that drops dibits from the rolling buffer drops the same
count of differentials, and any mismatch anywhere collapses the soft path to
empty for that stretch. That severity is deliberate. LLRs shifted by even one
symbol are confidently wrong about every bit — a decoder fed misaligned soft
data does worse than the hard path it was meant to improve. A burst whose soft
span is not fully covered simply decodes hard-only; `softFrame` returns nil and
nothing downstream notices. The same stash-bridge pattern carries the raw
pre-differential symbols (`StashSymbols` / `symBuf`) for the trained equalizer
we meet in Part 9 —
[Weak-Signal Engineering Part 9]({{ '/blog/deep-dives/weak-signal-engineering-09-parallel-buffers/' | relative_url }})
generalizes the whole parallel-buffer design.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="The hard and soft decode paths fork at the demodulator's differential output. The hard path slices each differential to a dibit and runs hard descramble, depuncture and Viterbi. The soft path keeps the complex differential, converts it to two per-bit LLRs, descrambles in the sign domain, and runs soft depuncture and soft Viterbi. Both paths end at the same class-2 CRC gate; the soft path recovers marginal bursts the hard path drops.">
  <rect x="8" y="86" width="128" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="72" y="106" text-anchor="middle" fill="currentColor" font-size="11">differential</text>
  <text x="72" y="120" text-anchor="middle" fill="var(--fg-muted)" font-size="9">s·conj(prev)</text>
  <line x1="136" y1="98" x2="170" y2="60" stroke="var(--fg-muted)"/><polygon points="166,58 178,54 172,66" fill="var(--fg-muted)"/>
  <line x1="136" y1="122" x2="170" y2="160" stroke="var(--accent)"/><polygon points="172,154 178,166 164,162" fill="var(--accent)"/>
  <rect x="180" y="26" width="120" height="48" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="240" y="46" text-anchor="middle" fill="var(--fg-muted)" font-size="10">hard slice</text>
  <text x="240" y="60" text-anchor="middle" fill="var(--fg-muted)" font-size="9">dibit, confidence lost</text>
  <rect x="180" y="146" width="120" height="48" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="240" y="166" text-anchor="middle" fill="var(--accent)" font-size="10">softType5FromDiffs</text>
  <text x="240" y="180" text-anchor="middle" fill="var(--fg-muted)" font-size="9">2 LLRs per symbol</text>
  <line x1="300" y1="50" x2="334" y2="50" stroke="var(--fg-muted)"/><polygon points="334,46 344,50 334,54" fill="var(--fg-muted)"/>
  <line x1="300" y1="170" x2="334" y2="170" stroke="var(--accent)"/><polygon points="334,166 344,170 334,174" fill="var(--accent)"/>
  <rect x="344" y="26" width="150" height="48" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="419" y="46" text-anchor="middle" fill="var(--fg-muted)" font-size="10">hard Viterbi</text>
  <text x="419" y="60" text-anchor="middle" fill="var(--fg-muted)" font-size="9">DecodeRCPCTetraMother</text>
  <rect x="344" y="146" width="150" height="48" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="419" y="166" text-anchor="middle" fill="var(--accent)" font-size="10">soft Viterbi</text>
  <text x="419" y="180" text-anchor="middle" fill="var(--fg-muted)" font-size="9">DecodeRCPCTetraMotherSoft</text>
  <line x1="494" y1="50" x2="540" y2="98" stroke="var(--fg-muted)"/><polygon points="536,96 546,104 532,106" fill="var(--fg-muted)"/>
  <line x1="494" y1="170" x2="540" y2="124" stroke="var(--accent)"/><polygon points="534,116 546,118 540,130" fill="var(--accent)"/>
  <rect x="548" y="86" width="124" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="610" y="106" text-anchor="middle" fill="currentColor" font-size="11">class-2 CRC</text>
  <text x="610" y="120" text-anchor="middle" fill="var(--fg-muted)" font-size="9">same gate for both</text>
  <text x="340" y="212" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the fork keeps confidence alive on the accent path — the ~2 dB the marginal call was missing</text>
</svg>
<figcaption>Hard and soft paths fork at the differential and rejoin at the class-2 CRC gate; only the soft path carries the demod's confidence into the Viterbi search.</figcaption>
</figure>

## The soft decode chain, step for step

`DecodeTCHSSoft` mirrors the hard `DecodeTCHS` from Part 5 step for step, in
the LLR domain. Same 24×18 deinterleave permutation, same split into class-0 /
class-1 / class-2 regions, same rate-8/12 and rate-8/18 depuncture geometry —
just twinned functions operating on `float32` reliabilities instead of bits:

```go
// internal/radio/tetra/tch.go (shape) — DecodeTCHSSoft
type3 := tchDeinterleaveSoft(type5LLR[:tchType3Bits])
class0 := type3[:tchClass0Bits]
c1 := type3[tchClass0Bits : tchClass0Bits+tchClass1Coded]
c2 := type3[tchClass0Bits+tchClass1Coded:]

m1 := framing.DepunctureRCPCTetraSoft(c1, framing.RCPCTetraPeriod23,
    framing.RCPCTetraPuncture23, 3*tchClass1Bits)
m2 := framing.DepunctureRCPCTetraSoft(c2, framing.RCPCTetraPeriod818,
    framing.RCPCTetraPuncture818, 3*(tchClass2Bits+tchCRCBits+tchTailBits))
conv, metric := framing.DecodeRCPCTetraMotherSoft(append(m1, m2...), tchConvIn)
// …class-2 CRC check is HARD and identical to DecodeTCHS
```

Two details are worth pausing on. First, the uncoded class-0 bits have no FEC
to spend reliability on, so `hardSliceLLR` just slices them — soft decision
only pays where a decoder exists to weigh evidence. Second, the class-2 CRC
check at the end is the *hard* check from
[Part 3]({{ '/blog/deep-dives/tetra-end-to-end-03-channel-coding-crc/' | relative_url }})
— the fixed parity-check matrix, not an LFSR — computed over the Viterbi's hard
output. Soft decision changes how hard the decoder fights for a burst; it does
not change what counts as winning. On the reporter's marginal capture that
combination took the same bursts the hard path dropped and recovered most of
them — and when Part 9's equalizer later stacked on top, the soft path is the
stream it multiplied.

### How that principle shaped the Go code

- **Opt-in at every layer.** `SoftSink` nil means zero overhead; no `StashSoft`
  means `softBuf` stays empty and the extractor is byte-identical to the
  pre-soft code. Every test that predates the feature still passes untouched.
- **Twinned functions, not flags.** `tchDeinterleaveSoft`,
  `DepunctureRCPCTetraSoft`, `DecodeRCPCTetraMotherSoft`, `DecodeTCHSSoft` —
  each hard function has a soft twin with the same geometry constants, so the
  two chains cannot drift apart structurally.
- **The scrambler moves into the sign domain.** Descrambling XORs a keystream
  bit; in LLRs that is a sign flip. `DescrambleTetraSoft` applies exactly the
  flips `DescrambleTetra` applies, keyed by the same extended colour code from
  [Part 4]({{ '/blog/deep-dives/tetra-end-to-end-04-scrambling-colour-codes/' | relative_url }}).

## Fallback, CRC gates, and the AACH bonus

The composer's voice chain (`decodeTETRASpeech` in
`internal/voice/composer/tetra_voice.go`) tries soft first and falls back:

```go
// internal/voice/composer/tetra_voice.go (shape)
if softType5 != nil {
    frames = tetra.TCHSpeechFramesSoft(softType5)
} else {
    frames = tetra.TCHSpeechFrames(frame)
}
```

The onBurst callback signature carries both — `frame []byte, softType5
[]float32` — so a burst whose soft span got dropped by the lockstep guard
degrades to the hard decode of that one burst, not a lost burst. And the soft
buffer earned a second job while it was there: the AACH usage marker — the
per-slot call identifier that demultiplexes concurrent same-carrier calls —
rides a small RM(30,14) block that frequently fails hard decode under load.
`usageOfSoft` re-decodes it from the same differentials, gated by
`aachSoftMaxDist = 6`: the soft maximum-likelihood codeword must sit within 6
bits of the hard-sliced word, so a rescued marker is a genuinely marginal burst
re-decided, never a low-confidence guess that could route another call's speech
into this recording. Once the LLRs are flowing, every marginal decode in the
burst becomes cheaper to save.

## Where this goes next

Soft decision recovered the bursts the noise was costing us — and exposed what
noise wasn't. The residual garble on the reporter's concurrent-load captures
had structure: a smeared constellation that no amount of per-bit confidence can
fix, because the *symbols themselves* were dragged off their positions by the
channel. [Part 9]({{ '/blog/deep-dives/tetra-end-to-end-09-equalizer-voice-path/' | relative_url }})
puts a blind equalizer between timing recovery and the differential decoder —
and explains why the obvious way to do that corrupts every dibit, and the
snapshot trick that doesn't.

## FAQ

**Where do the LLRs actually come from — is there a separate estimator?**
No. The demod already computes the complex differential `s·conj(prev)` for
every symbol; for π/4-DQPSK the two transmitted bits' LLRs are its imaginary
and real parts. `softType5FromDiffs` reshuffles and signs them per the
rotation; nothing new is measured.

**Why is the soft path allowed to silently fall back to hard?**
Because misaligned soft data is worse than none. The lockstep contract
(`softBuf` exactly parallel or empty) means any chunk boundary hiccup degrades
one burst to the hard decode instead of decoding it with shifted LLRs — which
would be confidently wrong about every bit and *raise* the error rate.

**Does soft decision ever pass a burst the hard CRC would reject?**
The gate is identical — the class-2 parity-check over the Viterbi's hard
output. Soft decision finds better codeword paths through the trellis, so more
*real* bursts reach the gate intact; a random or foreign burst still passes
only at the ~1/256 chance floor either way.

**How much did it actually recover?**
The marginal same-carrier call that motivated it was losing ~70% of its bursts
hard-only. Soft decision recovered the bulk of those, and it compounds with
Part 9's equalizer — the 410→778 CRC-valid figure across six captures is
measured on the soft stream.

**Why hard-slice the class-0 bits instead of keeping them soft?**
Class 0 is uncoded — there is no decoder downstream to spend the reliability.
An LLR only buys something when a code constrains which bit patterns are
possible; for uncoded bits the sign is all the information there is.

## Series navigation

**Part 8 of 14** · ←
[Part 7: Conformance — Bit-Identical Against the ETSI Reference]({{ '/blog/deep-dives/tetra-end-to-end-07-etsi-conformance/' | relative_url }})
· Next →
[Part 9: The Equalizer on the Voice Path]({{ '/blog/deep-dives/tetra-end-to-end-09-equalizer-voice-path/' | relative_url }})
