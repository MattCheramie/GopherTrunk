---
slug: tetra-rcpc-code
title: TETRA RCPC code
entry_type: algorithm
category: error-correction
description: TETRA's rate-compatible punctured convolutional codes protect its speech and signalling channels — a K=5 rate-1/3 mother code for traffic and a K=5 rate-1/4 code for signalling — with per-channel puncturing patterns that trade coding strength for throughput while sharing one decoder.
keywords: TETRA RCPC, rate-compatible punctured convolutional, K=5 mother code, puncturing, Viterbi decode, rate 2/3 8/18 8/17, EN 300 395-2 5.4.3, EN 300 392-2 8.2.3.1
aka: [TETRA RCPC, "rate-compatible punctured convolutional code", "TETRA convolutional code"]
autolink: true
infobox:
  - { label: TCH mother, value: "K=5, rate 1/3" }
  - { label: Sig mother, value: "K=5, rate 1/4" }
  - { label: Decoder, value: 16-state Viterbi }
  - { label: Spec, value: EN 300 395-2 §5.4.3 / EN 300 392-2 §8.2.3.1 }
see_also: [convolutional-code, puncturing, viterbi-algorithm, forward-error-correction, tetra-block-interleaver, tetra-tchs-speech-coding, tetra-logical-channels, tetra, soft-decision]
cite_urls:
  - https://en.wikipedia.org/wiki/Convolutional_code
  - https://en.wikipedia.org/wiki/Punctured_code
---

**TETRA RCPC codes** are the rate-compatible punctured [convolutional codes](/reference/convolutional-code/)
that carry forward error correction on every [TETRA](/reference/tetra/) logical channel.[^conv] TETRA uses two
16-state (constraint length K=5) mother codes: a **rate-1/3** code for the speech traffic channel
(EN 300 395-2 §5.4.3) and a **rate-1/4** code for the signalling channels (EN 300 392-2 §8.2.3.1). A single
low-rate mother code is [punctured](/reference/puncturing/) — some of its output bits are deleted — to reach
each channel's target rate, and the decoder fills the deleted positions with erasures and runs one common
[Viterbi](/reference/viterbi-algorithm/) survivor search.[^punc]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 130" role="img" aria-label="Information bits enter a K=5 shift-register convolutional encoder that emits three output streams; a puncturing stage deletes selected bits to raise the code rate; the receiver reinserts erasures and a Viterbi decoder recovers the information bits." xmlns="http://www.w3.org/2000/svg">
  <text x="18" y="50" font-size="8" fill="currentColor">info</text>
  <rect x="46" y="32" width="96" height="34" rx="3" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/>
  <text x="94" y="47" text-anchor="middle" font-size="8" fill="currentColor">K=5 encoder</text>
  <text x="94" y="59" text-anchor="middle" font-size="8" fill="currentColor">rate 1/3 or 1/4</text>
  <rect x="172" y="32" width="90" height="34" rx="3" fill="currentColor" fill-opacity="0.20" stroke="currentColor" stroke-width="1.1"/>
  <text x="217" y="47" text-anchor="middle" font-size="8" fill="currentColor">puncture</text>
  <text x="217" y="59" text-anchor="middle" font-size="8" fill="currentColor">delete bits</text>
  <rect x="292" y="32" width="96" height="34" rx="3" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/>
  <text x="340" y="47" text-anchor="middle" font-size="8" fill="currentColor">Viterbi</text>
  <text x="340" y="59" text-anchor="middle" font-size="8" fill="currentColor">+ erasures</text>
  <text x="410" y="50" font-size="8" fill="currentColor">info</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <path d="M32 49 L46 49"/><path d="M142 49 L172 49"/><path d="M262 49 L292 49"/><path d="M388 49 L406 49"/>
  </g>
  <text x="217" y="94" text-anchor="middle" font-size="7.5" fill="currentColor">one low-rate mother code + per-channel puncture pattern = many usable rates</text>
</svg>
<figcaption>A single low-rate mother code is punctured to each channel's target rate; the receiver reinserts erasure marks at the punctured positions and one Viterbi decoder recovers the bits regardless of which pattern was used.</figcaption>
</figure>

## The mother codes

Both mother codes share the same 16-state structure but differ in output count and polynomials. GopherTrunk
stores them as the per-stage XOR taps derived from the generator polynomials:

```go
// Traffic (TCH) mother, K=5 rate 1/3 — EN 300 395-2 §5.4.3.
//   G1 = 0x1F, G2 = 0x1B, G3 = 0x15
g1 = bit ^ d1 ^ d2 ^ d3 ^ d4   // 1 + D + D^2 + D^3 + D^4
g2 = bit ^ d1 ^ d3 ^ d4        // 1 + D + D^3 + D^4
g3 = bit ^ d2 ^ d4             // 1 + D^2 + D^4

// Signalling mother, K=5 rate 1/4 — EN 300 392-2 §8.2.3.1.
//   G1 = 0x13, G2 = 0x1D, G3 = 0x17, G4 = 0x1B
```

Each block is flushed with K−1 = 4 zero tail bits so the Viterbi survivor is forced to terminal state 0. The
rate-1/3 code protects speech; the rate-1/4 code, one output stream stronger, protects the BSCH, SCH/HD, BNCH,
STCH, SCH/HU, and SCH/F signalling channels.

## Rate compatibility and puncturing

**Rate-compatible** means the higher-rate patterns are *subsets* of the lower-rate ones: a bit kept at a
strong (low) rate is also kept at every weaker (higher) rate, so one encoder and one decoder serve the whole
family and a channel can even change rate mid-stream without swapping codecs. Puncturing follows the
§5.4.3.2 formula — for a period `t` and pattern `puncture[]`, the kept mother-code index is
`k = period·((j−1) div t) + puncture[(j−1) mod t]` (1-indexed). The traffic channel's class-1 bits use the
**rate-2/3** pattern `{1, 2, 4}` (period 6); its class-2 bits use the **rate-8/18** pattern
`{1,2,3,4,5,7,8,10,11}` (period 12); and under frame-stealing the **rate-8/17** pattern (period 24) applies.
The signalling code carries its own family — a rate-2/3 pattern `{1, 2, 5}` used by every standard signalling
channel, plus stronger and special 292/432 and 148/432 rates.

## Relevance to SDR

`internal/radio/framing/rcpc_tetra.go` and `rcpc_tetra_sig.go` implement the two mother codes, their
hard-decision 16-state Viterbi decoders, and the puncture/depuncture helpers, with the puncture patterns
exported as spec-verbatim tables. Depuncturing fills a mother-length buffer with a `DepunctureMark` erasure
at each deleted position, so the Viterbi metric simply skips those. Soft-decision variants exist so the
receiver's per-symbol confidences flow through, roughly doubling traffic yield on marginal captures. Chained
after the [block interleaver](/reference/tetra-block-interleaver/) and [scrambler](/reference/tetra-scrambler/),
these codes are what let a real TETRA channel survive a fading mobile path.

## Sources

[^conv]: [Convolutional code](https://en.wikipedia.org/wiki/Convolutional_code) — Wikipedia, on shift-register codes and their generator polynomials.
[^punc]: [Punctured code](https://en.wikipedia.org/wiki/Punctured_code) — Wikipedia, on deleting output bits to raise a code's rate while sharing one decoder.
