---
title: "TETRA End to End, Part 7: Conformance — Bit-Identical Against the ETSI Reference"
description: "How GopherTrunk proves its TETRA voice path — two independent conformance passes against the ETSI EN 300 395-2 reference codec, one demanding sample-for-sample PCM equality on a shared bitstream and one replaying real IQ through the whole chain, plus the 64-bit Word32 build trap that makes the reference tools lie."
category: deep-dives
keywords: etsi reference codec, en 300 395-2 conformance, bit exact vocoder test, tetra acelp validation, word32 lp64 trap, scoder sdecoder, skip guarded test harness, whole chain validation, tetra replay capture, gophertrunk tetra
tags: [tetra-end-to-end, tetra, conformance, testing, acelp, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "TETRA End to End"
series_part: 7
---

*Part 7 of **TETRA End to End**, a 14-part deep dive into how GopherTrunk turns
one real 25 kHz TETRA carrier into clear recorded voice.
[Part 6]({{ '/blog/deep-dives/tetra-end-to-end-06-clean-room-acelp/' | relative_url }})
closed with a promise: the pure-Go ACELP decoder is bit-exact against the ETSI
reference, and that claim has proof. This part is the proof — and the method
behind it, because the method is the real deliverable. This series' villain is
the test that validates its own bugs; the antidote is an **external
reference**, and TETRA voice has two independent ones: the ETSI reference C
codec, and the air itself. Between them sits a build trap that makes the
reference tools produce garbage on every modern 64-bit machine, and a
diagnostic rule this repo now treats as standing guidance.*

> **TL;DR:** Two independent conformance passes, both reproducible.
> **Pass one** (`internal/voice/acelp/etsi_reference_test.go`,
> `TestETSIReferenceConformance`): feed the *same* 137-bit-per-frame bitstream
> to the ETSI `sdecoder` and to GopherTrunk's decoder and demand **zero
> mismatched samples** — the decoder is fixed-point, so a faithful port
> matches exactly. **Pass two**
> (`cmd/gophertrunk/tetra_multislot_replay_test.go`,
> `TestTETRAMultiSlotReplay`): replay a real cs16 IQ capture through DDC →
> receiver → extractor → TCH/S → ACELP and correlate per-slot audio against
> the control channel's grant timeslots. Both are **skip-guarded** behind env
> vars because the ETSI vectors are copyrighted and captures are large.
> The trap: the reference sources assume a 32-bit `Word32` (`typedef long`) —
> on an LP64 host every saturating op silently returns garbage, so build the
> tools with a 32-bit `Word32` (or `-m32`) before trusting a byte they emit.
> The rule the passes enforce: **validate the whole chain against the
> reference, not just parts** — when voice doesn't decode but the vocoder's
> unit tests pass, suspect the channel coding.

**Key takeaways**

- **Bit-exactness is a binary verdict, and that's its value.** Fixed-point
  codecs permit `mismatches == 0` as the pass condition — no perceptual
  scoring, no tolerance tuning, no argument. One wrong saturating op fails
  loudly at a specific frame.
- **Two passes catch different lies.** The bitstream pass pins the vocoder in
  isolation; the IQ replay pins everything *upstream* of it. Part 3's CRC bug
  passed the first kind of scrutiny and only the second kind caught it.
- **The reference itself can be built wrong.** ETSI's fixed-point basic ops
  predate LP64; an unmodified 64-bit build produces confidently wrong
  vectors. Verify your reference before you verify against your reference.
- **Skip-guarded is not second-class.** Harnesses gated on `GT_ETSI_SERIAL` /
  `GT_TETRA_IQ` keep copyrighted vectors and bulky captures out of the repo
  while keeping the *procedure* executable by anyone — the reproducibility
  lives in the test file, not the test data.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Bitstream conformance | same serial file → GT vs ETSI PCM, 0 mismatches | `internal/voice/acelp/etsi_reference_test.go` (`TestETSIReferenceConformance`) |
| Serial format | int16 LE, 138 words/frame: BFI + 137 bits | `etsi_reference_test.go` (doc comment) |
| Level matching | reference `Post_Process` saturating ×2 | `etsi_reference_test.go` (`postProcessX2`) |
| Whole-chain replay | real cs16 IQ → per-slot audio vs grants | `cmd/gophertrunk/tetra_multislot_replay_test.go` (`TestTETRAMultiSlotReplay`) |
| Replay knobs | `GT_TETRA_IQ`, `GT_TETRA_IQ_RATE`, `GT_TETRA_COLOUR`, `GT_TETRA_OUT` | same file |
| Reference build trap | `Word32` must be 32-bit; LP64 `long` breaks saturation | `etsi_reference_test.go` (build note) |

## In this post

- **Why conformance needed two passes** — what each can and cannot see.
- **Pass one: the shared bitstream** — serial format, the ×2, and zero as the bar.
- **The Word32 trap** — when the reference implementation isn't the reference.
- **Pass two: the air** — replaying a capture through everything at once.
- **The diagnostic rule** — chain validation as standing guidance.

## Why conformance needed two passes

A voice path is a chain: demod → burst extraction → descramble → channel
decode → vocoder. Part 3 told the story of a bug that lived in the middle —
the class-2 CRC — while the pieces on either side of it were provably fine.
That episode fixed more than a CRC; it fixed the *testing model*. A unit test
per stage, each validated against its own encoder, proves only that each
stage agrees with itself. What's needed is external anchoring at two scopes:
the most complex single component (the vocoder) pinned against an independent
implementation, and the chain as a whole pinned against the only artifact
that exercises every stage with real-world conventions at once — an off-air
capture. Neither pass substitutes for the other. The bitstream pass would
never notice a scrambler seed bug (it enters below the scrambler); the replay
pass can't tell you *which* stage broke, only that one did. Together they
triangulate.

## Pass one: the shared bitstream

The ETSI distribution ships a fixed-point encoder and decoder. The harness
runs them once to produce two files, then holds GopherTrunk to their output:

```
scoder   in.pcm     serial.bin  synth_local.pcm   # ETSI reference encoder
sdecoder serial.bin ref_out.pcm                   # ETSI reference decoder
```

`serial.bin` is the shared bitstream — int16 little-endian, **138 words per
frame**: word 0 is the bad-frame indicator, words 1..137 are the coded speech
bits, exactly the format the reference's `Bits2prm_Tetra` reads and
GopherTrunk's `Decoder.Decode` consumes. `ref_out.pcm` is the reference
decoder's 240-samples-per-frame output. The test walks both in lockstep:

```go
// internal/voice/acelp/etsi_reference_test.go (shape)
dec := NewDecoder()
for f := 0; f < nFrames; f++ {
    bfi := serial[base] != 0
    /* … unpack 137 bits … */
    out := dec.Decode(bits, bfi)
    for i, s := range out {
        got := postProcessX2(s) // reference Post_Process: saturating ×2
        if got != ref[f*pcmPerFrame+i] {
            mismatches++ /* … track maxAbs, firstBadFrame … */
        }
    }
}
// pass condition: mismatches == 0
```

The bar is **zero**. Not "SNR above X," not "perceptually transparent" —
identical int16s, every sample of every frame, BFI frames included. That bar
is only reachable because the codec is defined in saturating fixed-point
arithmetic (Part 6's point): determinism is the spec. And it's why the pass
condition doubles as a diagnostic — `firstBadFrame` and `maxAbsDiff` in the
failure log localise a divergence to the frame where some ported operator
first disagreed, which during development repeatedly turned "the decoder is
wrong somewhere" into "look at this one subframe's gain path." Note the test
applies the `postProcessX2` itself: GopherTrunk's raw `Decoder` omits the
reference's output doubling (the registry adapter applies it in production),
so the harness levels the two before comparing.

The vectors are not committed — ETSI's sources and outputs are copyrighted —
so the test skips unless `GT_ETSI_SERIAL` and `GT_ETSI_REF` point at files you
built yourself. The procedure is the artifact under version control.

## The Word32 trap

Which brings us to the trap, preserved as a build note in the test's doc
comment because it burned real hours: the ETSI basic operators assume
`Word32` is 32 bits, via `typedef long` — true under ILP32, false on every
LP64 Linux/macOS host, where `long` is 64 bits. The consequence is not a
crash. Saturation tests like "does this exceed the 32-bit range" simply never
fire, so **every saturating operation returns unsaturated garbage** and the
reference tools emit confidently wrong PCM. Diff GopherTrunk against *that*
and you'd conclude the Go port is broken — or worse, "fix" the Go port into
agreement with a miscompiled reference, manufacturing a self-consistent pair
of wrong decoders. The villain, one meta-level up. The fix is mechanical
(edit the typedef to `int`, or build `-m32`); the lesson is not: **a
reference implementation is only a reference once you've verified the build
assumptions it was written under.** Trust, then verify — in that order,
applied to the yardstick itself.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="The two conformance passes. Pass one: a shared serial bitstream from the ETSI encoder fans out to the ETSI reference decoder and to GopherTrunk's decoder, whose PCM outputs are compared for zero mismatches. Pass two: a real cs16 IQ capture flows through the full GopherTrunk chain of DDC, receiver, extractor, TCH/S and ACELP, and the per-slot audio is correlated against the control channel's grant timeslots.">
  <text x="20" y="24" fill="var(--fg-muted)" font-size="11" font-weight="bold">Pass 1 — shared bitstream (vocoder in isolation)</text>
  <rect x="20" y="36" width="110" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="75" y="53" text-anchor="middle" fill="currentColor" font-size="10">serial.bin</text>
  <text x="75" y="67" text-anchor="middle" fill="var(--fg-muted)" font-size="9">BFI + 137 bits/frame</text>
  <line x1="130" y1="46" x2="196" y2="40" stroke="currentColor"/><polygon points="188,36 202,40 190,46" fill="currentColor"/>
  <line x1="130" y1="66" x2="196" y2="72" stroke="currentColor"/><polygon points="190,66 202,72 188,76" fill="currentColor"/>
  <rect x="202" y="24" width="128" height="32" rx="6" fill="none" stroke="currentColor"/>
  <text x="266" y="44" text-anchor="middle" fill="currentColor" font-size="10">ETSI sdecoder (32-bit!)</text>
  <rect x="202" y="62" width="128" height="32" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="266" y="82" text-anchor="middle" fill="var(--accent)" font-size="10">acelp.Decoder + ×2</text>
  <line x1="330" y1="40" x2="392" y2="52" stroke="currentColor"/><polygon points="384,50 398,55 386,58" fill="currentColor"/>
  <line x1="330" y1="78" x2="392" y2="66" stroke="currentColor"/><polygon points="386,60 398,63 384,68" fill="currentColor"/>
  <rect x="398" y="42" width="120" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="458" y="57" text-anchor="middle" fill="var(--accent)" font-size="10">compare PCM</text>
  <text x="458" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="9">mismatches must be 0</text>
  <text x="20" y="122" fill="var(--fg-muted)" font-size="11" font-weight="bold">Pass 2 — real capture (the whole chain)</text>
  <rect x="20" y="134" width="90" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="65" y="151" text-anchor="middle" fill="currentColor" font-size="10">cs16 IQ</text>
  <text x="65" y="165" text-anchor="middle" fill="var(--fg-muted)" font-size="9">GT_TETRA_IQ</text>
  <line x1="110" y1="154" x2="130" y2="154" stroke="currentColor"/><polygon points="130,150 139,154 130,158" fill="currentColor"/>
  <rect x="139" y="134" width="300" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="289" y="151" text-anchor="middle" fill="var(--accent)" font-size="10">DDC → receiver → extractor → TCH/S → ACELP</text>
  <text x="289" y="165" text-anchor="middle" fill="var(--fg-muted)" font-size="9">every stage, real-air conventions</text>
  <line x1="439" y1="154" x2="459" y2="154" stroke="currentColor"/><polygon points="459,150 468,154 459,158" fill="currentColor"/>
  <rect x="468" y="134" width="190" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="563" y="151" text-anchor="middle" fill="currentColor" font-size="10">per-slot audio timeline</text>
  <text x="563" y="165" text-anchor="middle" fill="var(--fg-muted)" font-size="9">correlated vs grant timeslots</text>
  <text x="340" y="204" text-anchor="middle" fill="var(--fg-muted)" font-size="10">pass 1 pins the vocoder; pass 2 pins everything upstream of it — the CRC bug of Part 3 was only visible to pass 2</text>
</svg>
<figcaption>Two external anchors: sample-exact PCM equality on a shared bitstream, and grant-correlated audio from a real capture through the whole chain.</figcaption>
</figure>

## Pass two: the air

The second pass is Part 5's replay harness wearing its conformance hat.
`TestTETRAMultiSlotReplay` takes a real cs16 capture of a TETRA voice carrier
(`GT_TETRA_IQ` + `GT_TETRA_IQ_RATE`), pushes it through the *production*
chain — the same `ccdecoder.NewDownconverter`, the same receiver options, the
same `TrafficExtractor` and `TCHSpeechFrames` and ACELP decode the daemon
runs — and emits per-slot WAVs plus activity timelines. The conformance
content is the correlation step: decoded speech must cluster on the timeslots
the control channel granted, at the times it granted them, with CRC yields
well off the 1/256 chance floor. A capture is the one test vector nobody on
the implementation side could have accidentally shaped, which is why this
pass caught what the unit tests could not: the class-2 CRC bug (all slots at
the chance floor), and later the slot-anchor shift (activity on the wrong
timeslots). It has since become the standing A/B instrument — the
soft-decision work of
[Part 8]({{ '/blog/deep-dives/tetra-end-to-end-08-soft-decision-tchs/' | relative_url }})
and the equalizer work after it are all scored as CRC-yield deltas on this
same harness, against the same captures.

## The diagnostic rule

The two passes compose into a decision procedure that is now standing
guidance in the repo, verbatim: *when "voice doesn't decode" but the vocoder
unit tests pass, suspect the channel coding (CRC / interleave / reorder), not
the vocoder — validate the whole chain against the reference, not just
parts.* The reasoning generalises. Pass one is cheap, deterministic, and
absolute — once it holds, the vocoder is effectively eliminated as a suspect
for any field symptom. So a field symptom with a green pass one *localises
itself* to the chain between demod and vocoder. That inversion — conformance
tests as suspect-elimination for future debugging, not just release gating —
is what the harnesses actually bought. Every subsequent TETRA investigation
in this series (soft decision, equalizers, DMO's false "encryption") started
from "the vocoder is proven; where in the chain is the loss?" and was faster
for it.

### How that principle shaped the Go code

- **Formats are documented at the test site.** The 138-word serial layout and
  the Word32 build note live in the test's doc comment — the next person to
  regenerate vectors gets the traps handed to them with the procedure.
- **The comparison is production-honest.** The harness applies the same
  `Post_Process` ×2 the registry adapter applies, so pass one certifies the
  audio path as shipped, not a lab variant.
- **One harness, many questions.** The replay test grew counters
  (`trafficMarkedCRC`, soft-path yields, per-marker clustering) instead of
  spawning sibling tests — every voice-path change since is an A/B on the
  same instrument, so numbers stay comparable across months.

## Where this goes next

The chain is proven end to end — on a *clean* capture. The next problem is
the marginal one: a same-carrier call where the hard-decision path threw away
~70% of the bursts that were actually recoverable.
[Part 8]({{ '/blog/deep-dives/tetra-end-to-end-08-soft-decision-tchs/' | relative_url }})
follows the soft path — the receiver's `SoftSink` differentials through soft
depuncture and Viterbi to `DecodeTCHSSoft` — and the ~2 dB that turned
short, garbled recordings into complete ones.

## FAQ

**Why aren't the conformance vectors committed to the repo?**
Because the ETSI reference sources and their outputs are ETSI-copyrighted.
The harness is committed; the vectors are regenerated locally from the ETSI
distribution by anyone who wants to run it. Reproducibility of the
*procedure* is the goal — the test file documents the exact commands.

**How would I know if I hit the Word32 trap?**
The reference tools build clean and run without error — the tell is
downstream: `synth_local.pcm` sounds wrong (harsh, clipped-then-not), and
GopherTrunk "fails" conformance with enormous `maxAbsDiff` from frame 0.
If a supposedly bit-exact port diverges immediately and hugely, audit the
reference build before the port.

**Does pass one cover the encoder too?**
No — GopherTrunk only decodes (it's a receiver), so only `sdecoder`'s side is
ported and pinned. The ETSI `scoder` is used solely to manufacture realistic
serial files from PCM for the comparison.

**What counts as passing pass two?**
No single number — it's a correlation judgment made against the capture's own
control channel: CRC-valid speech clustered on granted timeslots at granted
times, yields far above the 1/256 floor, and per-usage-marker activity mapping
one-to-one onto physical slots. The harness prints all of it; the operator's
capture is the ground truth.

**Is bit-exactness overkill when the output is lossy codec speech anyway?**
It's the opposite of overkill — it's the cheapest strong claim available.
Perceptual similarity requires listeners and thresholds; sample equality
requires `==`. And only the exact form has the elimination power the
diagnostic rule depends on: "close" leaves the vocoder a suspect forever.

## Series navigation

**Part 7 of 14** · ←
[Part 6: A Clean-Room ACELP Vocoder in Pure Go]({{ '/blog/deep-dives/tetra-end-to-end-06-clean-room-acelp/' | relative_url }})
· Next →
[Part 8: Going Soft — Soft-Decision TCH/S]({{ '/blog/deep-dives/tetra-end-to-end-08-soft-decision-tchs/' | relative_url }})
