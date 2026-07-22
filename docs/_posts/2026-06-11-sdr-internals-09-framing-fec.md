---
title: "SDR in Pure Go, Part 9: Framing & Forward Error Correction"
description: How GopherTrunk recovers clean data from noisy symbols using pure-Go forward error correction — Golay, BCH, Hamming, Reed-Solomon, BPTC, trellis, and Viterbi — built as deterministic, table-driven codecs.
category: deep-dives
tags: [sdr, go, fec, error-correction, viterbi, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "SDR Internals"
series_part: 9
---

*Part 9 of **SDR Internals**. Over the air, bits get flipped. This post is about
the forward-error-correction layer in `internal/radio/framing` that detects and
repairs those errors — and why it's written as pure, deterministic functions.*

## In this post

- What **forward error correction (FEC)** is and why every digital radio depends
  on it.
- The codes GopherTrunk implements in Go: **Golay, BCH, Hamming, Reed-Solomon,
  BPTC, trellis/Viterbi**, plus interleaving and scrambling.
- The **pure-function, table-driven** design that makes FEC fast and exhaustively
  testable.

## What framing and FEC do

A symbol stream straight off the timing loop has bit errors. Digital radio
survives this by adding **forward error correction**: structured redundancy that
lets the receiver detect *and repair* errors without asking for a retransmission.
([reference]({{ '/reference/forward-error-correction/' | relative_url }}))

<figure class="lab-figure">
<svg viewBox="0 0 660 116" width="660" height="116" role="img" aria-label="The framing and FEC stack: a symbol stream is scanned for a sync word that marks the frame boundary, the framed bits are assembled, the FEC decoder detects and repairs errors, and a clean payload emerges.">
  <rect x="8" y="36" width="112" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="64" y="56" text-anchor="middle" fill="currentColor" font-size="10">symbol stream</text>
  <text x="64" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="8">dibits, with errors</text>
  <line x1="120" y1="58" x2="138" y2="58" stroke="currentColor"/><polygon points="138,54 148,58 138,62" fill="currentColor"/>
  <rect x="148" y="36" width="104" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="200" y="56" text-anchor="middle" fill="currentColor" font-size="10">sync word</text>
  <text x="200" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="8">frame boundary</text>
  <line x1="252" y1="58" x2="270" y2="58" stroke="currentColor"/><polygon points="270,54 280,58 270,62" fill="currentColor"/>
  <rect x="280" y="36" width="104" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="332" y="60" text-anchor="middle" fill="currentColor" font-size="10">frame</text>
  <line x1="384" y1="58" x2="402" y2="58" stroke="currentColor"/><polygon points="402,54 412,58 402,62" fill="currentColor"/>
  <rect x="412" y="36" width="118" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="471" y="56" text-anchor="middle" fill="var(--accent)" font-size="10">FEC decode</text>
  <text x="471" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="8">detect + repair</text>
  <line x1="530" y1="58" x2="548" y2="58" stroke="currentColor"/><polygon points="548,54 558,58 548,62" fill="currentColor"/>
  <rect x="558" y="36" width="94" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="605" y="60" text-anchor="middle" fill="currentColor" font-size="10">payload</text>
</svg>
<figcaption>The framing stack: sync word &#8594; frame &#8594; FEC decode &#8594; clean payload &#8212; every protocol draws its codes from the shared <code>internal/radio/framing</code> toolbox.</figcaption>
</figure>

Each protocol layers several codes, plus
[interleaving]({{ '/reference/interleaving/' | relative_url }}) (spreads burst
errors out so the codes can handle them) and
[scrambling]({{ '/reference/scrambling/' | relative_url }}) (whitens the data).
GopherTrunk's `internal/radio/framing` package is the shared toolbox every
protocol draws from:

- [Golay(24,12)]({{ '/reference/golay-code/' | relative_url }}) — corrects up to
  3 errors (P25, DMR, M17 metadata).
- [Hamming]({{ '/reference/hamming-code/' | relative_url }}) variants — short
  single-error-correcting words.
- [BCH codes]({{ '/reference/bch-code/' | relative_url }}) — POCSAG, EDACS, DSC,
  MPT 1327.
- [Reed-Solomon]({{ '/reference/reed-solomon-code/' | relative_url }}) — P25
  Phase 2 and others.
- [BPTC]({{ '/reference/bptc/' | relative_url }}) — the DMR/P25 Phase 2 block
  product code.
- [Trellis-coded modulation]({{ '/reference/trellis-coded-modulation/' | relative_url }})
  decoded with the
  [Viterbi algorithm]({{ '/reference/viterbi-algorithm/' | relative_url }})
  (P25 Phase 1, YSF FICH).

## How GopherTrunk implements it in Go

FEC codecs are **pure functions over bits**: given the same input, they always
produce the same output, with no I/O and no shared state. That makes them the
most "ordinary Go" code in the whole project:

```go
// internal/radio/framing — Golay (shape)
func GolayEncode(data uint16) uint32       // 12 data bits -> 24-bit codeword
func GolayDecode(word uint32) (data uint16, ok bool) // repair up to 3 errors
```

Decoders use syndrome decoding — compute a syndrome, look up the matching error
pattern, flip the bad bits. Where a code is small, the error patterns are
**precomputed into a table** built once at package init, so decoding is a lookup,
not a search. The Viterbi decoder walks a trellis and traces back the
most-likely path; soft-decision variants take symbol confidences (LLRs) when the
demodulator can supply them.

<figure class="lab-figure">
<svg viewBox="0 0 660 170" width="660" height="170" role="img" aria-label="The syndrome-decoding flow for a Golay codeword: a noisy 24-bit word has its syndrome computed; the syndrome indexes a precomputed error-pattern table built at package init; the matching pattern is XORed to flip the bad bits, yielding 12 clean data bits and an ok flag.">
  <rect x="8" y="30" width="120" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="68" y="50" text-anchor="middle" fill="var(--accent)" font-size="10">noisy codeword</text>
  <text x="68" y="64" text-anchor="middle" fill="var(--fg-muted)" font-size="8">24 bits</text>
  <line x1="128" y1="53" x2="146" y2="53" stroke="currentColor"/><polygon points="146,49 156,53 146,57" fill="currentColor"/>
  <rect x="156" y="30" width="118" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="215" y="57" text-anchor="middle" fill="currentColor" font-size="10">compute syndrome</text>
  <line x1="274" y1="53" x2="292" y2="53" stroke="currentColor"/><polygon points="292,49 302,53 292,57" fill="currentColor"/>
  <rect x="302" y="30" width="126" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="365" y="52" text-anchor="middle" fill="currentColor" font-size="10">lookup error</text>
  <text x="365" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="8">pattern (O(1))</text>
  <line x1="428" y1="53" x2="446" y2="53" stroke="currentColor"/><polygon points="446,49 456,53 446,57" fill="currentColor"/>
  <rect x="456" y="30" width="110" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="511" y="52" text-anchor="middle" fill="currentColor" font-size="10">XOR flip</text>
  <text x="511" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="8">bad bits</text>
  <line x1="566" y1="53" x2="584" y2="53" stroke="currentColor"/><polygon points="584,49 594,53 584,57" fill="currentColor"/>
  <rect x="594" y="30" width="58" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="623" y="50" text-anchor="middle" fill="currentColor" font-size="9">data</text>
  <text x="623" y="64" text-anchor="middle" fill="var(--fg-muted)" font-size="8">12b &#183; ok</text>
  <rect x="302" y="118" width="126" height="34" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="365" y="134" text-anchor="middle" fill="var(--fg-muted)" font-size="9">syndrome&#8594;pattern</text>
  <text x="365" y="146" text-anchor="middle" fill="var(--fg-muted)" font-size="8">table built in init()</text>
  <line x1="365" y1="118" x2="365" y2="78" stroke="var(--fg-muted)"/><polygon points="361,86 365,78 369,86" fill="var(--fg-muted)"/>
</svg>
<figcaption>Syndrome decoding as a pure function: <code>GolayDecode</code> computes a syndrome, looks up the error pattern in a table precomputed at <code>init()</code>, and XORs it away &#8212; a lookup on the hot path, not a search.</figcaption>
</figure>

These pieces compose into a protocol's framer. A P25 Phase 1 voice frame, for
example, runs deinterleave → trellis/Viterbi → Hamming on the link-control words —
each a `framing` function called in sequence.

## The design principle: pure functions & table-driven determinism

FEC math has no business touching the network, the disk, or shared state. By
keeping every codec a **pure function** and pushing constants into **lookup
tables**, the package becomes both fast and trivially correct to test.

### How that principle shaped the Go code

- **No state, no surprises.** `GolayDecode(word)` depends only on `word`. There's
  nothing to mock, nothing to race, nothing to reset between calls — so these
  functions are safe to call from any goroutine.
- **Exhaustive, table-driven tests.** Because a short code's input space is small,
  tests can inject every single-bit and multi-bit error and assert the decoder
  recovers (or correctly rejects) it. Determinism makes 100% confidence
  achievable, not aspirational.
- **Tables built once.** Syndrome→error-pattern maps are computed in `init()` and
  reused, turning per-frame decoding into O(1) lookups on the hot path — the same
  "design once, run hot" idea from
  [Part 4]({{ '/blog/deep-dives/sdr-internals-04-dsp-foundations-filters-nco-agc/' | relative_url }}).
- **Allocation-free decoding.** Codecs work on fixed-width integers and
  caller-provided slices, so the FEC stage adds no garbage-collector pressure to
  the decode loop.

## Where this goes next

Coding theory is a whole field, and each of these codes could anchor its own
article — the Golay code's elegant structure, BPTC's two-dimensional product
construction, and Viterbi's dynamic-programming traceback especially. A future
**"Error Correction in Go"** series can derive them from scratch. Next, we see how
the protocols assemble these primitives into working decoders.

## FAQ

**What's the difference between error detection and correction?**
A CRC ([cyclic redundancy check]({{ '/reference/cyclic-redundancy-check/' | relative_url }}))
only *detects* that something is wrong. FEC codes like Golay and Reed-Solomon add
enough redundancy to *locate and repair* a bounded number of errors with no
retransmission.

**Why so many different codes?**
Each protocol's designers chose a code matched to its channel and bit budget —
short Hamming words for control fields, powerful Reed-Solomon/BPTC blocks for
voice. GopherTrunk implements whatever each protocol on the air actually uses.

**Does soft-decision decoding really help?**
Yes. Feeding the decoder symbol confidences instead of hard 0/1 bits can recover
frames at noticeably lower SNR, which is why the demod and FEC stages are designed
to pass soft values through where the code supports it.

## Series navigation

**Part 9 of 14** · ←
[Part 8]({{ '/blog/deep-dives/sdr-internals-08-equalization-diversity-fft/' | relative_url }})
· Next →
[Part 10: Protocol decoders as state machines]({{ '/blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/' | relative_url }})
