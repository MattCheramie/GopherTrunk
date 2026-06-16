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
