---
slug: polyphase-filter-bank
title: Polyphase filter bank (PFB)
entry_type: algorithm
category: filtering-multirate
description: A polyphase filter bank splits one FIR filter into parallel sub-filters so decimation, interpolation, and channelization run at the low output rate for huge efficiency gains.
keywords: polyphase filter bank, PFB, polyphase decomposition, polyphase channelizer, efficient decimation, interpolation, Noble identity, FIR filter, DFT filter bank, SDR channelizer
aka: [PFB, polyphase channelizer, polyphase decomposition, DFT filter bank]
autolink: true
infobox:
  - { label: Type, value: Multirate FIR structure }
  - { label: Enables, value: Cheap decimation/interpolation & N-channel channelizing }
  - { label: Savings, value: ~N× fewer multiplies vs naive filter-then-decimate }
see_also: [channelizer, fir-filter, decimation, resampler, cic-filter, digital-down-converter]
cite_urls:
  - https://en.wikipedia.org/wiki/Polyphase_matrix
  - https://en.wikipedia.org/wiki/Filter_bank
---

A **polyphase filter bank (PFB)** is a way of factoring a single
[FIR filter](/reference/fir-filter/) into a set of parallel sub-filters — its
*polyphase branches* — so that [decimation](/reference/decimation/),
interpolation, and [channelization](/reference/channelizer/) can be performed at
the low sample rate instead of the high one.[^wiki] The rearrangement is exact
(it computes the same output as the direct filter) but moves every multiply to
the slow side of the resampler, cutting arithmetic by roughly the resampling
factor and, when paired with a [DFT/FFT](/reference/fast-fourier-transform/),
producing many frequency channels in one shared pass.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="An input stream is commutated into M polyphase sub-filter branches whose outputs feed a DFT block that emits M separate channel outputs." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="pfbar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <line x1="12" y1="90" x2="60" y2="90" stroke="currentColor" stroke-width="1.2"/><text x="30" y="82">x[n]</text>
    <circle cx="70" cy="90" r="9" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="70" y="93">↻</text>
    <rect x="110" y="18" width="80" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="150" y="35">E0(z)</text>
    <rect x="110" y="58" width="80" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="150" y="75">E1(z)</text>
    <rect x="110" y="98" width="80" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="150" y="115">…</text>
    <rect x="110" y="138" width="80" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="150" y="155">E_{M-1}(z)</text>
    <path d="M79 86 L110 31" fill="none" stroke="currentColor" stroke-width="1" marker-end="url(#pfbar)"/>
    <path d="M79 88 L110 71" fill="none" stroke="currentColor" stroke-width="1" marker-end="url(#pfbar)"/>
    <path d="M79 92 L110 111" fill="none" stroke="currentColor" stroke-width="1" marker-end="url(#pfbar)"/>
    <path d="M79 94 L110 151" fill="none" stroke="currentColor" stroke-width="1" marker-end="url(#pfbar)"/>
    <rect x="270" y="30" width="60" height="120" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="300" y="86">DFT</text><text x="300" y="98">/ FFT</text>
    <path d="M190 31 L270 45" fill="none" stroke="currentColor" stroke-width="1" marker-end="url(#pfbar)"/>
    <path d="M190 71 L270 75" fill="none" stroke="currentColor" stroke-width="1" marker-end="url(#pfbar)"/>
    <path d="M190 151 L270 135" fill="none" stroke="currentColor" stroke-width="1" marker-end="url(#pfbar)"/>
    <line x1="330" y1="50" x2="380" y2="50" stroke="currentColor" stroke-width="1.1" marker-end="url(#pfbar)"/><text x="405" y="53">chan 0</text>
    <line x1="330" y1="90" x2="380" y2="90" stroke="currentColor" stroke-width="1.1" marker-end="url(#pfbar)"/><text x="405" y="93">chan 1</text>
    <line x1="330" y1="130" x2="380" y2="130" stroke="currentColor" stroke-width="1.1" marker-end="url(#pfbar)"/><text x="405" y="133">chan M-1</text>
  </g>
</svg>
<figcaption>The prototype low-pass filter is split into M polyphase branches; a DFT across the branch outputs shifts each into its own channel, yielding M channels from one filter pass.</figcaption>
</figure>

## How it works

Take a length-*NM* FIR prototype filter and decimate its output by *M*. Naively
you would compute every output sample of the filter and then throw away *M*−1 of
every *M* — most of the multiplies are wasted. Polyphase decomposition splits the
tap set into *M* interleaved subsequences, the branches *E₀(z), E₁(z), … E_{M-1}(z)*,
where branch *k* holds taps *k, k+M, k+2M, …*. The **Noble identities** let you
push the decimator *before* each branch filter, so every branch runs at the
output rate and consumes one input sample per branch through an input *commutator*
that hands successive samples to successive branches.

The payoff is twofold:

- **Efficiency.** Each input sample touches exactly one branch, and each branch
  runs at 1/*M* of the input rate. The total multiply-rate is that of the full
  filter evaluated once at the *low* rate — an ~*M*× saving over filter-then-decimate.
- **Channelization for free.** If, instead of summing the branch outputs, you feed
  them into an *M*-point DFT, each DFT output bin is a critically- or
  over-sampled channel centred on a different multiple of the channel spacing. One
  shared prototype filter plus one [FFT](/reference/fast-fourier-transform/) then
  yields *M* down-converted, filtered channels simultaneously — the **polyphase
  channelizer**.

Interpolation is the transpose: the DFT (or IDFT) spreads a symbol across the
branches, an output commutator interleaves branch outputs, and the prototype
filter's images are suppressed at the high rate. The same machinery, run with a
non-integer commutator step, becomes an arbitrary [resampler](/reference/resampler/).

## Variants

- **PFB channelizer** — the DFT-coupled bank above, the workhorse of wideband SDR
  receivers that must split one capture into dozens or hundreds of channels.
- **Oversampled / M:2M banks** — advance the commutator by *M*/2 so adjacent
  channels overlap, avoiding scalloping loss at channel edges for signals that
  straddle a boundary.
- **Polyphase arbitrary resampler** — a bank of branches indexed by a fractional
  accumulator gives rational or near-continuous rate change, a common alternative
  to a [CIC](/reference/cic-filter/)-plus-FIR chain.

## Relevance to SDR

Wideband trunking receivers routinely digitize several megahertz and must peel out
many 12.5 kHz or 25 kHz voice/control channels at once; a PFB channelizer does this
far more cheaply than an independent [digital down-converter](/reference/digital-down-converter/)
per channel. The structure appears throughout radio astronomy correlators, LTE/5G
base-station front-ends, and general-purpose SDR frameworks (GNU Radio's PFB blocks,
for example). GopherTrunk's tuning path uses discrete down-converters and decimating
FIR/CIC stages rather than a full DFT channelizer, but the polyphase idea — do the
filtering at the decimated rate — is the same efficiency principle its multirate
[decimation](/reference/decimation/) stages rely on.

## Sources

[^wiki]: [Polyphase matrix](https://en.wikipedia.org/wiki/Polyphase_matrix) — Wikipedia, on polyphase decomposition of filters for efficient multirate processing, and [Filter bank](https://en.wikipedia.org/wiki/Filter_bank) for the DFT-coupled channelizer.
