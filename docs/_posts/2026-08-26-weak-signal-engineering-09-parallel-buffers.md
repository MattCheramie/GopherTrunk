---
title: "Weak-Signal Engineering, Part 9: Parallel Buffers — SymbolSink, SoftSink & Opt-In Soft Paths"
description: The architecture pattern that let equalizers and soft-decision FEC land safely — LLR differentials and raw symbols carried strictly parallel to the hard dibit buffer, a stash bridge keyed by base index, and a byte-identical opt-out pinned by failing-first tests.
category: deep-dives
keywords: opt-in dsp architecture, parallel soft buffer, symbol sink, soft sink, byte-identical fallback, stash bridge, tetra traffic extractor, safe dsp refactoring, gophertrunk weak-signal engineering
tags: [weak-signal-engineering, architecture, soft-decision, dsp, tetra, testing, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Weak-Signal Engineering"
series_part: 9
---

*Part 9 of **Weak-Signal Engineering**, a 14-part series on decoding the
marginal regime — where the receiver locks but only a fraction of frames
survive. [Part 8]({{ '/blog/deep-dives/weak-signal-engineering-08-soft-decisions/' | relative_url }})
carried per-bit LLRs through depuncture and a correlation-metric Viterbi and
recovered the ~70% of marginal bursts the hard gate was dropping. But that
post glossed over a harder question: how do you thread new soft and equalized
data through a burst extractor that a fleet of working configurations already
depends on — without risking any of them? This is the engineering-process
part of the series: the parallel-buffer pattern, the stash bridge, the
fallback ladder, and the one property that made every risky DSP change in
Parts 4–8 landable: **no sinks wired ⇒ byte-identical legacy behaviour.***

> **TL;DR:** New DSP information travels in **parallel buffers, never in
> modified ones**. The TETRA receiver's `Options.SoftSink` (complex
> differentials — the LLR source) and `Options.SymbolSink` (raw
> pre-differential symbols — the equalizer's training domain) fire just before
> the matching `DibitSink` call, aligned 1:1 and keyed by the same `baseIdx`.
> The `tetra.TrafficExtractor` stashes them (`StashSoft` / `StashSymbols`)
> into `softBuf` / `symBuf` held **strictly parallel** to its hard dibit
> buffer, and `softFrame` walks a fallback ladder: equalized differentials →
> raw differentials → nil (hard-only). A nil sink costs zero; a caller that
> never stashes is **byte-identical** to the pre-soft code — a property pinned
> by regression tests, not asserted in a comment.

**Key takeaways**

- **Additive, not intrusive.** The hard `DibitSink` contract never changed
  through four generations of DSP upgrades. Soft LLRs and raw symbols ride
  beside the dibits, so every existing caller compiles, runs, and produces the
  same bytes.
- **The two sinks exist because the two domains are different.** LLRs live in
  the differential domain (right for the FEC); a linear channel is only a
  convolution in the raw-symbol domain (right for a trained equalizer —
  [Part 3]({{ '/blog/deep-dives/weak-signal-engineering-03-isi-linear-channel/' | relative_url }})'s
  fact, made structural).
- **Fallback is a ladder, and every rung is total.** Equalized-soft → soft →
  hard: a burst missing its symbol span falls to raw LLRs; a burst missing
  LLRs falls to the hard frame; nothing errors, nothing is silently worse
  than the old code.
- **"Byte-identical opt-out" is a testable claim.** Tests literally run the
  extractor with and without sinks and compare outputs — which is what lets a
  blind equalizer or a soft Viterbi land on a Tuesday without an on-air
  regression risk.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Soft source | differentials `s·conj(last)`, 1:1 with dibits | `internal/radio/tetra/receiver/receiver.go` (`Options.SoftSink`) |
| Symbol source | raw post-timing/AFC symbols, 1:1 with dibits | `receiver.go` (`Options.SymbolSink`) |
| Stash bridge | hold one call's soft/symbols for the next `Process` | `internal/radio/tetra/traffic.go` (`StashSoft`, `StashSymbols`) |
| Parallel buffers | `softBuf`/`symBuf` strictly parallel to `buf` | `traffic.go` (`TrafficExtractor`) |
| Fallback ladder | equalized diffs → raw diffs → nil | `traffic.go` (`softFrame`, `equalizedBurstDiffs`, `rawBurstDiffs`) |
| Composer rung | soft frames when LLRs exist, else hard | `internal/voice/composer/tetra_voice.go` (`decodeTETRASpeech`) |
| Opt-out pin | no sinks ⇒ identical output | `traffic_lms_test.go` (`TestTrafficExtractorSoftUnchangedWithoutEqualizer`), `dmo_equalizer_test.go` (`TestExtractDMBurstsEqualizedNoSymbolsUnchanged`) |

## In this post

- **The constraint** — risky DSP over a working fleet.
- **Two sinks, one dibit contract** — why the receiver grew callbacks, not flags.
- **The stash bridge** — how `baseIdx` keeps three buffers in lockstep.
- **The fallback ladder** — total functions all the way down.
- **Byte-identical opt-out** — the property, and the tests that pin it.

## The constraint: risky DSP over a working fleet

Every lever in this series so far — blind CMA
([Part 4]({{ '/blog/deep-dives/weak-signal-engineering-04-blind-cma/' | relative_url }})),
frozen snapshots ([Part 5]({{ '/blog/deep-dives/weak-signal-engineering-05-snapshot-trick/' | relative_url }})),
trained LMS ([Part 7]({{ '/blog/deep-dives/weak-signal-engineering-07-trained-lms/' | relative_url }})),
soft FEC ([Part 8]({{ '/blog/deep-dives/weak-signal-engineering-08-soft-decisions/' | relative_url }})) —
is exactly the kind of change that breaks working systems. Adaptive filters
have failure modes that only appear on air. Soft paths double the data flowing
through a burst extractor. And the extractor in question,
`tetra.TrafficExtractor`, sits in the live voice path of every TETRA
configuration in the field.

The classic responses are both bad. Fork the extractor into a "v2" and you
maintain two decoders that drift apart. Thread the new data through the
existing types — change `DibitSink` to carry a struct of dibits-plus-LLRs —
and every caller changes, every test fixture changes, and the diff that lands
a 2 dB decoding improvement also touches thirty files that had nothing wrong
with them. GopherTrunk took a third route, and it is the reason four
generations of demod-side upgrades landed without a single change to the hard
contract: **new information travels beside the old, in buffers that may
simply not exist.**

## Two sinks, one dibit contract

The receiver's options grew two optional callbacks. Neither changes what
`DibitSink` receives; both are documented as zero-overhead when nil:

```go
// internal/radio/tetra/receiver/receiver.go (shape) — Options
// SoftSink, when non-nil, receives the complex π/4-DQPSK differential
// (s·conj(last)) for each symbol, aligned 1:1 with the dibits emitted
// to DibitSink and carrying the same baseIdx. … Emitted just before the
// matching DibitSink call. nil ⇒ no soft emission, zero overhead.
SoftSink func(diffs []complex64, baseIdx int)
// SymbolSink, when non-nil, receives the RAW post-timing/AFC/equalizer
// complex symbols (before the differential decode) … Unlike the SoftSink
// differential (a nonlinear product s·conj(last), in which the channel is
// no longer a clean convolution), the symbol stream is where a linear
// channel IS a convolution — so it is the input a training-sequence
// equalizer … must train on and equalize per burst.
SymbolSink func(symbols []complex64, baseIdx int)
```

Why two? Because the two consumers need different domains, and the difference
is mathematical, not organisational. The soft Viterbi wants the differential —
that is where the on-air bits' LLRs live. But the trained LMS equalizer of
Part 7 *cannot* work there: the differential `s·conj(prev)` is a nonlinear
product of two channel-affected symbols, and a linear channel stops being a
clean convolution the moment you form it. The equalizer must see the raw
symbols. One stream cannot serve both, so both are emitted — each 1:1 with
the dibits, each tagged with the same `baseIdx`, each skippable.

The wiring at a call site is one closure per sink, as in the control-channel
pipeline:

```go
// internal/scanner/ccdecoder/pipelines.go (shape) — newTETRAPipeline
rx := tetrarx.New(tetrarx.Options{
    SampleRateHz: opts.SampleRateHz,
    DibitSink: func(dibits []uint8, baseIdx int) {
        opts.tapDibits(dibits, baseIdx)
        cc.Process(dibits, baseIdx)
    },
    // Soft differentials for soft-decision channel decoding, stashed
    // just before the matching DibitSink → Process call.
    SoftSink: func(diffs []complex64, baseIdx int) {
        cc.StashSoft(diffs, baseIdx)
    },
    /* … ClockMode, EnableAFC, EnableChannelFilter, EnableEqualizer … */
})
```

## The stash bridge

The sinks fire *before* the matching `DibitSink` call, and the extractor's
`Process` runs *inside* that call — so the soft data for a chunk of dibits
must be parked somewhere until the dibits arrive. That is the stash bridge:

```go
// internal/radio/tetra/traffic.go (shape) — the stash half
func (te *TrafficExtractor) StashSoft(diffs []complex64, baseIdx int) {
    te.pendingSoft = diffs
    te.pendingSoftBase = baseIdx
}

func (te *TrafficExtractor) StashSymbols(syms []complex64, baseIdx int) {
    te.pendingSym = syms
    te.pendingSymBase = baseIdx
}
```

Each stash holds exactly one pending chunk, keyed by the `baseIdx` the next
`Process` call will deliver — and `Process` consumes it once, appending to
`softBuf`/`symBuf` in lockstep with the hard `buf`. The invariant the whole
design rests on is blunt: `len(softBuf)` is either **zero** (a caller that
never stashes) or **exactly `len(buf)`** — and the same for `symBuf`. There
is no partially-soft state. Both parallel buffers share `buf`'s base offset
(`bufBase`) and are trimmed together, so an index into the dibit buffer is an
index into its soft twin, forever. Everything runs on the receiver's single
`Process` goroutine, so the bridge needs no locks — sequencing does the work.

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="Block diagram of the parallel-buffer pattern. The receiver emits three aligned streams per chunk: SymbolSink raw symbols, SoftSink differentials, and DibitSink hard dibits, all sharing one baseIdx. The two optional streams pass through StashSymbols and StashSoft into symBuf and softBuf, held parallel to the hard dibit buffer buf inside the TrafficExtractor. A fallback ladder on the right reads equalized differentials first, then raw differentials, then falls to hard-only decode.">
  <rect x="8" y="76" width="110" height="60" rx="6" fill="none" stroke="currentColor"/>
  <text x="63" y="100" text-anchor="middle" fill="currentColor" font-size="10">receiver</text>
  <text x="63" y="114" text-anchor="middle" fill="var(--fg-muted)" font-size="8">one Process goroutine</text>
  <line x1="118" y1="90" x2="180" y2="52" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <line x1="118" y1="106" x2="180" y2="106" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <line x1="118" y1="122" x2="180" y2="160" stroke="currentColor"/>
  <text x="149" y="44" fill="var(--fg-muted)" font-size="8">SymbolSink (opt)</text>
  <text x="132" y="99" fill="var(--fg-muted)" font-size="8">SoftSink (opt)</text>
  <text x="138" y="156" fill="currentColor" font-size="8">DibitSink</text>
  <rect x="182" y="34" width="120" height="34" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="242" y="55" text-anchor="middle" fill="var(--fg-muted)" font-size="9">StashSymbols</text>
  <rect x="182" y="88" width="120" height="34" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="242" y="109" text-anchor="middle" fill="var(--fg-muted)" font-size="9">StashSoft</text>
  <rect x="330" y="24" width="150" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="405" y="45" text-anchor="middle" fill="var(--accent)" font-size="10">symBuf []complex64</text>
  <text x="405" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="8">len 0 or len(buf)</text>
  <rect x="330" y="86" width="150" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="405" y="107" text-anchor="middle" fill="var(--accent)" font-size="10">softBuf []complex64</text>
  <text x="405" y="124" text-anchor="middle" fill="var(--fg-muted)" font-size="8">len 0 or len(buf)</text>
  <rect x="330" y="148" width="150" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="405" y="169" text-anchor="middle" fill="currentColor" font-size="10">buf (hard dibits)</text>
  <text x="405" y="186" text-anchor="middle" fill="var(--fg-muted)" font-size="8">the unchanged contract</text>
  <line x1="302" y1="51" x2="330" y2="51" stroke="var(--fg-muted)"/>
  <line x1="302" y1="105" x2="330" y2="105" stroke="var(--fg-muted)"/>
  <line x1="248" y1="160" x2="330" y2="171" stroke="currentColor"/>
  <rect x="516" y="76" width="156" height="96" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="594" y="96" text-anchor="middle" fill="var(--accent)" font-size="10">softFrame ladder</text>
  <text x="594" y="114" text-anchor="middle" fill="var(--fg-muted)" font-size="9">1. equalizedBurstDiffs</text>
  <text x="594" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="9">2. rawBurstDiffs</text>
  <text x="594" y="146" text-anchor="middle" fill="var(--fg-muted)" font-size="9">3. nil → hard-only</text>
  <line x1="480" y1="112" x2="516" y2="112" stroke="var(--accent)"/>
  <text x="340" y="222" text-anchor="middle" fill="var(--fg-muted)" font-size="10">no sinks wired ⇒ softBuf and symBuf stay empty ⇒ byte-identical to the pre-soft extractor</text>
</svg>
<figcaption>Three streams, one index space: the optional symbol and soft buffers ride strictly parallel to the hard dibit buffer, and the fallback ladder degrades one rung at a time.</figcaption>
</figure>

## The fallback ladder

When a burst is ready to emit, `softFrame` builds its 432-LLR type-5 stream —
or declines to, one rung at a time:

```go
// internal/radio/tetra/traffic.go (shape) — softFrame
func (te *TrafficExtractor) softFrame(L int) []float32 {
    if len(te.softBuf) != len(te.buf) {
        return nil // no LLRs were ever stashed: hard-only caller
    }
    // Prefer the training-sequence-equalized differentials when the
    // equalizer is enabled and this burst's raw symbol span is fully
    // buffered; otherwise use the receiver's raw differentials.
    diffs := te.equalizedBurstDiffs(L)
    if diffs == nil {
        diffs = te.rawBurstDiffs(L)
    }
    if diffs == nil {
        return nil
    }
    llr := softType5FromDiffs(diffs, 0)
    if te.colourCode != 0 {
        llr = framing.DescrambleTetraSoft(llr, te.colourCode)
    }
    return llr
}
```

Read the rungs. `equalizedBurstDiffs` returns nil whenever
`EnableLMSEqualizer` was never called, no symbols were stashed, or this
particular burst's symbol span (including the FIR warm-up) is not fully
buffered — so the equalizer is a no-op unless *both* halves of its opt-in are
in play. `rawBurstDiffs` returns nil when the soft buffer doesn't cover the
burst. And a nil `softFrame` simply means the burst is emitted hard-only —
`onBurst` always carries the hard frame beside the (possibly nil) soft one,
and the composer's `decodeTETRASpeech` completes the ladder: soft frames when
`softType5 != nil`, the hard `TCHSpeechFrames` gate otherwise. Every rung is
a total function; a degraded input degrades the output by one rung, never to
an error and never below the pre-soft baseline.

## Byte-identical opt-out — the property, and its tests

The claim "a caller that never stashes is unchanged" is cheap to write in a
doc comment and worth little there. What makes it load-bearing is that it is
*pinned*: `TestTrafficExtractorSoftUnchangedWithoutEqualizer`
(`internal/radio/tetra/traffic_lms_test.go`) runs the extractor over the same
burst stream with and without the new machinery and requires identical soft
output, and `TestExtractDMBurstsEqualizedNoSymbolsUnchanged`
(`dmo_equalizer_test.go`) does the same for the Direct Mode path — alongside
the improvement pins (`TestTrafficExtractorLMSRecoversMultipathBurst`:
synthetic multipath through the real extractor, raw 13% → 0% payload
bit-error) and the no-harm pins (`TestTrafficExtractorLMSNoHarmOnCleanChannel`).

That triple — *improves the bad case, doesn't touch the clean case, is
byte-identical when off* — is the shape of every safely-landable DSP change
in this series, and the parallel-buffer pattern is what makes the third leg
provable rather than hoped-for. Compare the alternative: had LLRs been woven
into the dibit type, "off" would not be a state the type system could even
express, and the only evidence of safety would be a full re-run of every
on-air configuration.

### How that principle shaped the Go code

- **nil is the feature flag.** No config knob decides whether soft decoding
  happens — wiring a sink does. The zero value of the system is the legacy
  system, so the safest state is also the default state.
- **Invariants over checks.** `softBuf` is empty or exactly parallel — the
  extractor maintains that in one place (`Process`) instead of length-checking
  at every use, and `softFrame`'s single guard is the whole enforcement.
- **The ladder lives in one function.** Fallback priority is not scattered
  across call sites; `softFrame` is the only place that ranks equalized over
  raw over nothing, so the policy is readable — and changeable — in one diff.

## Where this goes next

Everything so far has squeezed more out of one antenna. The next two parts
add a second one.
[Part 10]({{ '/blog/deep-dives/weak-signal-engineering-10-mrc-calibration/' | relative_url }})
opens the diversity pair: maximal-ratio combining, why one wideband complex
gain is both the opportunity and the caveat, and the coherence-gated
calibration — `CrossStats`, `|rho| = γ/(1+γ)`, and the DC-removal detail that
turns out to be load-bearing — that decides when combining can be trusted at
all.

## FAQ

**Why sinks and stashes instead of returning richer values from the receiver?**
The receiver's emission points and the extractor's consumption points are on
opposite sides of an existing callback boundary crossed by many other callers.
Callbacks that default to nil extend that boundary without moving it; a richer
return type would move it for everyone, including the callers that want
nothing new.

**What stops the soft buffer drifting out of alignment with the dibits?**
Sequencing plus a hard invariant. Sinks fire immediately before their matching
`DibitSink` call on the same goroutine, the stash is consumed exactly once by
the next `Process`, both buffers share the same base offset and trims, and
`softFrame` refuses to operate unless `len(softBuf) == len(buf)`. Misalignment
degrades to hard-only decode; it cannot silently mis-map LLRs.

**Does an unwired sink really cost nothing?**
Yes — a nil callback is never invoked, no soft or symbol slices are
allocated, and the parallel buffers stay empty. The cost exists only for
callers that opted in, which is what lets the control-channel, voice, and DMO
paths each choose independently.

**Why does the composer get its own fallback rung?**
Because the extractor can produce a hard frame with no soft twin (early
bursts, trimmed windows). The composer deciding "soft if present, hard
otherwise" per burst means one degraded burst degrades alone rather than
switching the whole call's mode.

**Could this pattern carry other data — say, per-burst quality metrics?**
That is exactly its shape: anything aligned 1:1 with the dibits and keyed by
`baseIdx` can ride a third parallel buffer without touching the hard
contract. The pattern's cost is one invariant per buffer; its payoff is that
"off" remains the identity function.

## Series navigation

**Part 9 of 14** · ←
[Part 8: Soft Decisions — LLRs Through Depuncture & Viterbi]({{ '/blog/deep-dives/weak-signal-engineering-08-soft-decisions/' | relative_url }})
· Next →
[Part 10: Diversity I — MRC & Coherence-Gated Calibration]({{ '/blog/deep-dives/weak-signal-engineering-10-mrc-calibration/' | relative_url }})
