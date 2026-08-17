---
title: "TETRA End to End, Part 6: A Clean-Room ACELP Vocoder in Pure Go"
description: Inside GopherTrunk's pure-Go TETRA speech decoder — how a 137-bit ACELP frame becomes 240 samples of 8 kHz audio through LSP dequantisation, adaptive and algebraic codebooks, and a bit-exact fixed-point synthesis filter, and what clean-room means when the target is sample-for-sample agreement with a reference codec.
category: deep-dives
keywords: tetra acelp decoder, acelp vocoder go, en 300 395-2 speech codec, lsp dequantisation, algebraic codebook, adaptive codebook pitch, fixed point speech synthesis, 137 bit speech frame, bad frame indicator concealment, gophertrunk tetra
tags: [tetra-end-to-end, tetra, acelp, vocoder, voice, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "TETRA End to End"
series_part: 6
---

*Part 6 of **TETRA End to End**, a 14-part deep dive into how GopherTrunk turns
one real 25 kHz TETRA carrier into clear recorded voice.
[Part 5]({{ '/blog/deep-dives/tetra-end-to-end-05-tchs-traffic-channel/' | relative_url }})
ended holding CRC-valid 137-bit speech frames — the traffic channel's whole
output. Every other digital voice protocol GopherTrunk decodes speaks some MBE
dialect, rendered by the vocoder family the
[Voice Coding series]({{ '/blog/deep-dives/voice-coding-01-what-is-a-vocoder/' | relative_url }})
dissected. TETRA speaks something else entirely: ACELP, the algebraic
code-excited linear prediction family that also underlies GSM-EFR and G.729.
GopherTrunk's implementation is pure Go, fixed-point, and bit-exact against
the ETSI reference decoder — this part is how it works and what "clean-room"
means when the acceptance test is sample-for-sample equality.*

> **TL;DR:** `internal/voice/acelp` decodes one **137-bit** EN 300 395-2
> speech frame into **240 samples of 8 kHz PCM** (30 ms). The frame is 23
> quantiser indices (`bits2prm`): three split-VQ LSP indices dequantised by
> `dLsp334` into the LPC filter, then four 60-sample subframes, each with a
> pitch lag (20..143, 1/3-sample resolution), a 4-pulse **algebraic codebook**
> index (`dD4i60`), and a 6-bit energy index (`decEner`, MA-predicted in the
> log2 domain). Excitation = adaptive + algebraic vectors scaled by the
> decoded gains, pushed through the `synFilt` LPC synthesis filter. All of it
> runs on ported 16/32-bit **saturating fixed-point ops** (`ops.go`), because
> bit-exactness lives or dies on overflow behaviour. The registry adapter
> (`VocoderName = "tetra-acelp"`) applies the reference's `Post_Process`
> saturating ×2 — omit it and the audio sits 6 dB under reference. Erased
> frames (BFI) repeat the previous LSPs and parameters with decayed gains.

**Key takeaways**

- **ACELP is a source-filter model, not a spectral model.** MBE transmits a
  description of the spectrum; ACELP transmits a recipe for the *excitation* —
  pitch echo plus four algebraic pulses — and filters it through a
  transmitted vocal-tract model. That's why its bit classes (Part 5) map so
  cleanly to perceptual damage.
- **Fixed-point is the spec, not an optimisation.** The reference codec
  defines behaviour in saturating 16/32-bit arithmetic; matching it
  sample-for-sample means porting the arithmetic, overflow flags and all.
  Floating point would sound fine and match nothing.
- **The decoder is honest about erasures.** A missing or CRC-failed frame
  (BFI) doesn't glitch: LSPs and parameters repeat, energies decay — the
  concealment the CRC gate of Part 5 relies on.
- **One saturating ×2 is part of the codec.** The reference's `Post_Process`
  doubles every output sample with saturation; GopherTrunk applies it in the
  vocoder adapter so recordings match reference level exactly.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Frame unpack | 137 bits → BFI + 23 indices (`bitWidths`) | `internal/voice/acelp/params.go` (`bits2prm`) |
| LSP dequantise | 3 split-VQ indices (8+9+9 bits) → 10 LSPs | `params.go` (`dLsp334`), `lsp_tables.go` |
| LPC interpolation | per-subframe filter from old/new LSPs | `internal/voice/acelp/lpc.go` (`intLpc4`, `lspAz`) |
| Adaptive codebook | pitch prediction at 1/3-sample resolution | `internal/voice/acelp/codebook.go` (`predLt`, `inter32_1_3`) |
| Algebraic codebook | 4 pulses from index/sign/shift + noise filter | `codebook.go` (`dD4i60`, `algebraicPulses`) |
| Gain decode | 6-bit energy VQ, log2-domain MA prediction | `internal/voice/acelp/gain.go` (`enerPredictor.decEner`) |
| Synthesis | excitation → speech through 1/A(z) | `internal/voice/acelp/syn.go` (`synFilt`) |
| Registry adapter | `"tetra-acelp"`, 18-byte frames, Post_Process ×2 | `internal/voice/acelp/vocoder.go` (`NewVocoder`) |

## In this post

- **A different animal from MBE** — what ACELP transmits and why.
- **Unpacking 137 bits** — the parameter layout of a frame.
- **The excitation** — adaptive pitch, algebraic pulses, decoded gains.
- **Synthesis in saturating fixed point** — why the basic ops are the codec.
- **Erasures, the adapter, and what clean-room means here** — BFI concealment, the ×2, and provenance.

## A different animal from MBE

The [MBE model]({{ '/blog/deep-dives/voice-coding-02-the-mbe-model/' | relative_url }})
that serves P25, DMR and NXDN transmits a *description of the output*: band
energies, voicing decisions, a fundamental. ACELP transmits a *procedure for
the input*: here is a filter approximating the vocal tract (the LPC
coefficients, sent as line spectral pairs), and here is how to build the
signal you should push through it — take an echo of what you excited the
filter with a pitch period ago, add a handful of sharp pulses at coded
positions, scale both. "Analysis-by-synthesis" means the *encoder* ran this
exact decoder in a loop to pick the codebook entries whose output best matched
the microphone; the decoder just replays the winning recipe. One consequence
paid off in Part 5: parameters differ wildly in sensitivity — a wrong LSP or
gain index corrupts the filter or energy trajectory for a whole frame (class
2), while a mislocated excitation pulse is a click at worst (class 0).

## Unpacking 137 bits

`bits2prm` splits a frame into 23 indices whose widths are the codec's anatomy
in one array:

```go
// internal/voice/acelp/params.go (shape)
var bitWidths = [numParams]int{
    8, 9, 9,        // split-VQ LSP indices (dico1..3)
    8, 14, 1, 1, 6, // subframe 1: pitch, code, sign, shift, energy
    5, 14, 1, 1, 6, // subframe 2
    5, 14, 1, 1, 6, // subframe 3
    5, 14, 1, 1, 6, // subframe 4
}
```

26 bits of LSPs describe the vocal tract for the whole 30 ms; the remaining
111 are four subframes of excitation recipe. The first subframe spends 8 bits
on pitch to encode an absolute lag (20..143 samples with 1/3 fractions);
subframes 2–4 spend 5, coding a delta around subframe 1's lag — pitch moves
slowly, so relative coding is nearly free. `dLsp334` dequantises the three LSP
indices against their codebooks and — with `intLpc4` — interpolates between
last frame's LSPs and this frame's so the filter glides rather than steps at
subframe boundaries.

## The excitation: pitch echo plus four pulses

Per subframe, the decoder builds 60 samples of excitation from two sources.
The **adaptive codebook** is simply the recent excitation history replayed at
the decoded lag: `predLt` reaches back `T0` samples (interpolating with
`inter32_1_3` when the lag has a 1/3 fraction) — that recursion is what makes
voiced speech periodic. The **algebraic codebook** adds the innovation — four
pulses whose positions come from the 14-bit index, with a global sign and
shift:

```go
// internal/voice/acelp/decoder.go (shape) — one subframe
predLt(d.oldExc, excOff+iSubfr, int(T0), int(t0Frac), subfrLen) // adaptive vector

ap3 := pondAi(aSub, d.fGamma3) // A(z/0.75)
ap4 := pondAi(aSub, d.fGamma4) // A(z/0.85)
/* … F[] = impulse response of ap3/ap4 weighting filter, + 0.8·pitch echo … */
dD4i60(prm[base+sfCode], prm[base+sfSign], prm[base+sfShift], zeroF, 64, code)

gainPit, gainCode := d.ener.decEner(prm[base+sfEnergy], bfi, aSub, excSub, code, subfrLen)
for i := 0; i < subfrLen; i++ {
    L := lMult0(d.oldExc[excOff+iSubfr+i], gainPit)
    L = lMac0(L, code[i], gainCode)
    d.oldExc[excOff+iSubfr+i] = int16(lShrR(L, 12)) // Q12 → Q0
}
synFilt(aSub, d.oldExc[excOff+iSubfr:], synth[iSubfr:iSubfr+subfrLen], subfrLen, d.memSyn[:], true)
```

Two refinements distinguish this from textbook CELP. The pulses aren't placed
as raw impulses: `dD4i60` convolves them with `F[]`, a perceptual noise-shaping
filter built per subframe from the bandwidth-expanded LPC (γ = 0.75/0.85) plus
a fixed 0.8-gain pitch contribution — the innovation arrives pre-shaped to
hide under the signal. And the gains aren't transmitted directly: `decEner`
decodes one 6-bit index against an MA *prediction* of the pitch and code
energies from the previous subframe, in the log2 domain — the codec bets that
energy is predictable and only sends the correction. (That inter-subframe
predictor is decoder state, which is why frame decoding is order-dependent and
the conformance test of Part 7 must run whole files, not cherry-picked
frames.)

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="ACELP decoder block diagram: the adaptive codebook replays excitation history at the decoded pitch lag and the algebraic codebook produces four shaped pulses; each is scaled by its decoded gain and summed into the excitation, which feeds both the LPC synthesis filter producing 240 PCM samples and, recursively, back into the excitation history.">
  <rect x="10" y="30" width="150" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="85" y="48" text-anchor="middle" fill="currentColor" font-size="10">adaptive codebook</text>
  <text x="85" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="9">excitation @ lag T0 (1/3 res)</text>
  <rect x="10" y="130" width="150" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="85" y="148" text-anchor="middle" fill="currentColor" font-size="10">algebraic codebook</text>
  <text x="85" y="162" text-anchor="middle" fill="var(--fg-muted)" font-size="9">4 pulses, noise-shaped (dD4i60)</text>
  <text x="216" y="45" text-anchor="middle" fill="var(--fg-muted)" font-size="9">× gain_pit</text>
  <line x1="160" y1="52" x2="268" y2="94" stroke="currentColor"/><polygon points="260,92 274,97 264,84" fill="currentColor"/>
  <text x="216" y="168" text-anchor="middle" fill="var(--fg-muted)" font-size="9">× gain_code</text>
  <line x1="160" y1="152" x2="268" y2="110" stroke="currentColor"/><polygon points="264,120 274,107 260,112" fill="currentColor"/>
  <circle cx="290" cy="102" r="14" fill="none" stroke="var(--accent)"/>
  <text x="290" y="106" text-anchor="middle" fill="var(--accent)" font-size="12">+</text>
  <line x1="304" y1="102" x2="352" y2="102" stroke="currentColor"/><polygon points="352,98 361,102 352,106" fill="currentColor"/>
  <rect x="361" y="80" width="128" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="425" y="98" text-anchor="middle" fill="var(--accent)" font-size="10">LPC synthesis 1/A(z)</text>
  <text x="425" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">synFilt, interpolated LSPs</text>
  <line x1="489" y1="102" x2="537" y2="102" stroke="currentColor"/><polygon points="537,98 546,102 537,106" fill="currentColor"/>
  <rect x="546" y="80" width="126" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="609" y="98" text-anchor="middle" fill="currentColor" font-size="10">240 samples / 30 ms</text>
  <text x="609" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">8 kHz PCM (+ sat. ×2)</text>
  <path d="M 290 116 Q 290 196 85 186 Q 30 182 24 78" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <polygon points="20,86 24,72 30,85" fill="var(--fg-muted)"/>
  <text x="290" y="200" text-anchor="middle" fill="var(--fg-muted)" font-size="9">excitation history feeds next subframe's adaptive codebook</text>
</svg>
<figcaption>The ACELP recipe: pitch echo plus shaped pulses, gain-scaled, summed, filtered — and written back into the history the next subframe will echo.</figcaption>
</figure>

## Synthesis in saturating fixed point

The least glamorous file is the most load-bearing. `ops.go` ports the
reference's basic operators — `addOp`, `sature`, `lMult0`, `lShrR`, the full
16/32-bit saturating family, overflow flag included — and every block above is
built exclusively from them. This is not retro affectation: the reference
codec *defines* the decoder as sequences of these operations, and legal
outputs are whatever they produce, saturation artifacts and all. A float
implementation would sound indistinguishable and never match a single sample;
matching sample-for-sample (Part 7's test is literally `mismatches != 0` →
fail) requires the arithmetic to be the spec. The same discipline showed up on
the C side of the fence with teeth: the reference tools themselves miscompute
on LP64 hosts unless `Word32` is forced to 32 bits — every saturating op
silently returns garbage — a trap Part 7 documents because it cost real time.

## Erasures, the adapter, and what "clean-room" means here

**Erasures.** When Part 5's CRC gate drops a burst, the vocoder still owes the
recording 30 ms. On BFI the decoder repeats the previous frame's LSPs and
quantiser parameters (`oldParm`, `oldT0`) while `decEner` decays the predicted
energies — a fading echo of the last good frame instead of a click or a hole.
A too-short input frame is treated as BFI automatically.

**The adapter.** `vocoder.go` bridges to the
[vocoder registry]({{ '/blog/deep-dives/voice-coding-01-what-is-a-vocoder/' | relative_url }}):
`VocoderName = "tetra-acelp"`, `FrameSize()` of 18 bytes (137 bits packed
MSB-first), registered in `voice.DefaultRegistry` at init so the recorder's
`tetra → tetra-acelp` mapping resolves it by name. Its `Decode` applies the
one post-step the raw decoder omits — the reference's `Post_Process`, a
saturating multiply-by-two on every sample. That's not a tweak; skip it and
GopherTrunk's output sits exactly 6 dB below the reference decoder's, which
both fails conformance and makes TETRA recordings quiet next to every other
protocol's.

**Clean-room.** The phrase earns definition. This is a from-scratch Go
implementation — no C compiled in, no cgo, no copied source — written to the
EN 300 395-2 spec with the ETSI reference codec's *behaviour* as the
acceptance target; the quantiser tables are the numeric constants the standard
itself publishes. The package documentation is candid about provenance and
posture: the reference implementation is ETSI-copyrighted, the algorithm
family historically patent-encumbered (the core patents largely expired), and
so the decoder lives in its own package behind an explicit vocoder
registration, with the shipping posture documented in `docs/vocoders.md`
rather than assumed. Honesty here is cheap; discovering a licensing problem
after shipping is not.

## Where this goes next

"Bit-exact against the reference" has been asserted three times now without
proof. [Part 7]({{ '/blog/deep-dives/tetra-end-to-end-07-etsi-conformance/' | relative_url }})
supplies it: the two independent conformance passes — same bitstream into the
ETSI reference decoder and GopherTrunk's, demanding identical PCM; and a real
IQ capture through the whole chain, grant-correlated — plus the LP64 `Word32`
trap in the reference build, and the chain-validation lesson that closes the
loop on Part 3's CRC story.

## FAQ

**Why does TETRA use ACELP when everything else GopherTrunk decodes uses MBE?**
Different lineages: the US land-mobile ecosystem standardised on DVSI's MBE
family, while ETSI drew on the European telecom tradition that produced
GSM-EFR and G.729 — ACELP was its state of the art at TETRA's design time, and
at 4.567 kbit/s of speech data it fits TETRA's slot budget with room for the
Part 5 protection.

**Is 137 bits per 30 ms really enough for speech?**
It's ~4.57 kbit/s, and yes — because so little of it is waveform. 26 bits
model the vocal tract, and the rest is the excitation recipe the
analysis-by-synthesis encoder chose. The result is characteristic codec
speech: intelligible, speaker-recognisable, and unmistakably synthetic.

**What sample rate and width does the decoder emit?**
Signed 16-bit PCM at 8 kHz, 240 samples per frame. The recorder resamples or
containers it the same way as every other vocoder's output — the registry
interface hides the codec entirely.

**How does concealment interact with the CRC gate upstream?**
The gate turns "wrong bits" into "missing frame," and the decoder turns
"missing frame" into a decayed repeat of the last good one. That pairing is
deliberate: a wrong class-2 parameter can sound far worse than 30 ms of faded
echo, so the system prefers erasure over corruption.

**Could the vocoder be validated without the ETSI reference?**
Only weakly — round-trip tests through GopherTrunk's own encoder would be
Part 3's self-consistent trap wearing a codec costume. Sample-exact comparison
against an independently built decoder is the strong form, and it's the entire
subject of the next part.

## Series navigation

**Part 6 of 14** · ←
[Part 5: TCH/S — From Traffic Burst to Speech Frame]({{ '/blog/deep-dives/tetra-end-to-end-05-tchs-traffic-channel/' | relative_url }})
· Next →
[Part 7: Conformance — Bit-Identical Against the ETSI Reference]({{ '/blog/deep-dives/tetra-end-to-end-07-etsi-conformance/' | relative_url }})
