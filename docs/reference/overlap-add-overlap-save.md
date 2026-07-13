---
slug: overlap-add-overlap-save
title: Overlap-add & overlap-save
entry_type: algorithm
category: algorithms
description: Overlap-add and overlap-save perform fast FIR filtering by convolving signal blocks with an FFT, stitching the blocks so long streams filter efficiently in SDRs.
keywords: overlap-add, overlap-save, fast convolution, block convolution, FFT filtering, FIR filtering, circular convolution, overlap-discard, channelization
aka: [overlap-add, overlap-save, overlap-discard, fast convolution]
autolink: true
infobox:
  - { label: Type, value: Fast block convolution }
  - { label: Implements, value: Long FIR filtering via FFT }
  - { label: Benefit, value: O(N log N) vs O(N*M) direct }
see_also: [fast-fourier-transform, fir-filter, discrete-fourier-transform, channelizer, digital-filter]
cite_urls:
  - https://en.wikipedia.org/wiki/Overlap%E2%80%93add_method
  - https://en.wikipedia.org/wiki/Overlap%E2%80%93save_method
---

**Overlap-add** and **overlap-save** are two block-processing methods that carry out
[FIR filtering](/reference/fir-filter/) — the convolution of a long input stream with a
filter's impulse response — efficiently by doing the work in the frequency domain with an
[FFT](/reference/fast-fourier-transform/).[^oa][^os] Multiplying two spectra is equivalent to
convolving in time, so for a long filter it is far cheaper to FFT a block, multiply by the
filter's stored frequency response, and inverse-FFT than to compute the direct
sample-by-sample sum. Both methods exist to solve the same problem: an FFT gives *circular*
convolution, but filtering needs *linear* convolution, and the two differ at the block edges.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="Top: overlap-add zero-pads each input block, filters it, and sums the tails of adjacent output blocks. Bottom: overlap-save overlaps input blocks and discards the corrupted leading samples of each output." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="60" y="16">overlap-add</text>
    <rect x="20" y="24" width="90" height="16" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <rect x="20" y="46" width="60" height="14" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <rect x="80" y="46" width="30" height="14" fill="none" stroke="currentColor" stroke-dasharray="3 2" stroke-width="1.1"/>
    <rect x="80" y="64" width="60" height="14" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <text x="200" y="45">tails summed</text>
    <text x="95" y="90">&#8595; add overlapping tails &#8595;</text>
    <line x1="20" y1="98" x2="200" y2="98" stroke="currentColor" stroke-width="1"/>

    <text x="65" y="122">overlap-save</text>
    <rect x="20" y="128" width="40" height="14" fill="none" stroke="currentColor" stroke-dasharray="3 2" stroke-width="1.1"/>
    <rect x="60" y="128" width="70" height="14" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <rect x="90" y="150" width="40" height="14" fill="none" stroke="currentColor" stroke-dasharray="3 2" stroke-width="1.1"/>
    <rect x="130" y="150" width="70" height="14" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <text x="300" y="136">dashed = discarded</text>
    <text x="300" y="158">solid = kept output</text>
  </g>
</svg>
<figcaption>Overlap-add zero-pads blocks and sums the wrap-around tails; overlap-save overlaps input blocks and discards the wrap-corrupted leading samples of each result.</figcaption>
</figure>

## How it works

A length-*M* FIR filter convolving with a length-*N* block produces `N+M-1` output samples,
but an *L*-point FFT (with `L ≥ N+M-1`) only holds *L*. Circular convolution wraps the extra
`M-1` samples back onto the start of the block, corrupting it. The two methods handle that
wrap differently:

- **Overlap-add.** Cut the input into *non-overlapping* blocks of length *N*, zero-pad each
  to length `L = N+M-1`, filter it (FFT, multiply, inverse-FFT). Each filtered block is
  `N+M-1` long, so its last `M-1` samples spill into the next block's time span. Lay the
  blocks end to end and **add** the overlapping tails. The sum reconstructs exact linear
  convolution.
- **Overlap-save** (overlap-discard). Let the input blocks *overlap* by `M-1` samples. Filter
  each *L*-point block with the FFT; the first `M-1` output samples are the corrupted
  circular-wrap region, so simply **discard** them and keep the rest. No addition is needed —
  the overlap supplies the history each block needs.

Both give bit-for-bit the same result as a direct FIR; they only differ in bookkeeping.
Overlap-add sums tails and zero-pads; overlap-save discards a prefix and overlaps input.

## In practice

The speed win is large when the filter is long: direct convolution costs `O(N·M)` per block
while the FFT route costs `O(L log L)`, so once *M* is more than a few dozen taps the FFT
method dominates. The block/FFT size *L* is tuned to balance FFT efficiency against latency —
bigger blocks amortise the FFT better but delay the output more. Overlap-save is often
marginally leaner because it avoids the add step, while overlap-add composes naturally when
outputs are accumulated from several sources.

The same machinery generalises to **channelization**: an FFT-based
[channelizer](/reference/channelizer/) or fast-convolution filter bank splits a wideband
capture into many narrow channels at once by filtering blocks in the frequency domain and
selecting sub-bands, reusing exactly the overlap bookkeeping described here.

## Relevance to SDR

Fast convolution is a core efficiency tool in SDR. Filtering a wideband
[I/Q](/reference/iq-data/) stream with a sharp, long FIR — for channel selection,
pulse-shaping, or matched filtering — is often cheapest via overlap-save block convolution,
and wideband channelizers that peel dozens of trunking channels out of one capture are built
on the same block-FFT structure. Efficient [digital filtering](/reference/digital-filter/) of
this kind is what lets a general-purpose CPU keep up with megahertz-wide SDR front ends in
real time.

GopherTrunk's decode chain leans on direct/polyphase time-domain filtering and per-channel
down-conversion for its trunking work, but the overlap methods are the standard way to make
long-FIR and channelizing operations fast, and any FFT-based wideband filtering follows this
block-and-stitch pattern.

## Sources

[^oa]: [Overlap–add method](https://en.wikipedia.org/wiki/Overlap%E2%80%93add_method) — Wikipedia, on zero-padded block convolution with tail summation.
[^os]: [Overlap–save method](https://en.wikipedia.org/wiki/Overlap%E2%80%93save_method) — Wikipedia, on overlapping input blocks with discard of the wrap-corrupted output.
