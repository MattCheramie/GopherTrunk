---
slug: channel-estimation
title: Channel estimation
entry_type: term
category: estimation-array
description: Channel estimation measures the complex gain, phase, and delay a radio channel applies to a signal — from known training symbols or blindly from statistics — so a receiver can invert or exploit the channel.
keywords: channel estimation, channel estimate, least squares, pilot symbols, training sequence, midamble, channel state information, CSI, blind estimation, equalizer training
aka: [channel estimate, CSI estimation]
autolink: true
infobox:
  - { label: Type, value: Receiver estimation step }
  - { label: Estimates, value: Complex channel gain(s) h }
  - { label: From, value: Training symbols, pilots, or signal statistics }
  - { label: Feeds, value: Equalizers, MRC, IRC, coherent demod }
see_also: [tetra-training-sequences, adaptive-filter, lms-algorithm, maximal-ratio-combining, interference-rejection-combining, mmse-equalizer, coherence]
cite_urls:
  - https://en.wikipedia.org/wiki/Channel_state_information
  - https://en.wikipedia.org/wiki/Least_squares
---

**Channel estimation** is the receiver's measurement of what the radio channel did to the
signal — the complex gain, phase rotation, delay spread, or full impulse response between
transmitter and receiver — so that later stages can undo it or exploit it.[^wiki] Almost
every coherent technique stands on a channel estimate: an
[equalizer](/reference/adaptive-filter/) needs one to invert
[ISI](/reference/intersymbol-interference/),
[maximal-ratio combining](/reference/maximal-ratio-combining/) needs one per branch to
co-phase and weight, and
[interference rejection combining](/reference/interference-rejection-combining/) needs one
just to define what the interference *is*.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A transmitted burst containing known training symbols passes through an unknown channel; the receiver compares the received training region against the known symbols to solve for the channel estimate h." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="46" width="150" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <rect x="72" y="46" width="44" height="26" fill="currentColor" fill-opacity="0.2"/>
  <text x="94" y="63" font-size="8" fill="currentColor" text-anchor="middle">known</text>
  <text x="95" y="88" font-size="8.5" fill="currentColor" text-anchor="middle">burst with training symbols</text>
  <line x1="175" y1="59" x2="225" y2="59" stroke="currentColor" stroke-width="1.2" marker-end="url(#cear)"/>
  <rect x="228" y="42" width="66" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3" stroke-dasharray="4 3"/>
  <text x="261" y="63" font-size="9" fill="currentColor" text-anchor="middle">channel h?</text>
  <line x1="297" y1="59" x2="347" y2="59" stroke="currentColor" stroke-width="1.2" marker-end="url(#cear)"/>
  <rect x="350" y="42" width="90" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="395" y="58" font-size="8.5" fill="currentColor" text-anchor="middle">solve h from</text>
  <text x="395" y="70" font-size="8.5" fill="currentColor" text-anchor="middle">received vs known</text>
  <defs><marker id="cear" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Because the receiver knows what the training region should have been, the difference between sent and received solves directly for the channel.</figcaption>
</figure>

## How it works

The workhorse is **least squares against known symbols**. If the receiver knows the
transmitted training block `s` and observes `x = h·s + n`, the LS estimate is
`ĥ = Σ x·conj(s) / Σ |s|²` — a correlation against the known pattern, normalised. Its
accuracy improves with training length N (the estimate's phase error shrinks roughly as
`1/√(SNR·N)`), which is the quantitative reason standards spend airtime on pilots: GSM and
[TETRA](/reference/tetra/) put a known midamble
([training sequence](/reference/tetra-training-sequences/)) in the middle of every burst,
OFDM systems scatter pilot subcarriers across the grid, and P25/DMR receivers can treat
their [frame sync words](/reference/frame-synchronization/) as free training. An **MMSE**
refinement folds in noise statistics; interpolation extends pilot estimates across time and
frequency as the channel changes.

When no training exists, **blind estimation** falls back on statistics — the constant
modulus of a PSK constellation (as the [CMA equalizer](/reference/cma-equalizer/) does),
cyclostationarity, or cross-correlation between diversity branches. Blind estimates are
cheaper in airtime but structurally weaker: they converge slower, leave ambiguities a
statistic cannot see (a blind estimate cannot recover absolute phase), and — critically —
they are **contaminated by interference**, because a correlation cannot tell the wanted
signal from a co-channel one and returns a power-weighted blend of both channels. That
single limitation is why blind IRC fails while trained IRC works.

## In practice

Three habits separate estimates that hold up from ones that mislead:

- **Remove DC before correlating.** Two receiver branches share LO leakage; an uncentred
  cross-correlation on independent noise plus a common DC term reports near-perfect
  [coherence](/reference/coherence/) and "estimates" the ratio of the DC offsets —
  confidently measuring nothing.
- **Gate on the estimate's own error, not on signal level.** The projected phase error
  `√((1−ρ²)/(2Nρ²))` from measured coherence ρ and window length N says whether an estimate
  is trustworthy, independent of gain staging or bandwidth.
- **Track when the channel moves.** A one-shot estimate is only right while the hardware
  holds still; independent PLLs, tuner retunes, and fading all demand re-estimation with
  smoothing.

## Relevance to SDR

In GopherTrunk the same LS machinery appears at two scales. Per diversity *branch*, the
SoapyRemote MRC combiner estimates one complex gain per ~2 ms window against a reference
branch (`internal/dsp/diversity/tracking.go`), coherence-gated and smoothed — a wideband,
frequency-flat channel estimate. Per *burst*, the TETRA traffic extractor trains a short
FIR channel inverse on the burst's known midamble
([snapshot equalization](/reference/snapshot-equalization/) via `SnapshotLMS`), a
frequency-selective estimate valid for one burst. Both illustrate the same trade: training
symbols buy an estimate that interference and noise cannot silently corrupt, and where no
training exists, every downstream conclusion inherits the blind estimate's blind spots.

## Sources

[^wiki]: [Channel state information](https://en.wikipedia.org/wiki/Channel_state_information) — Wikipedia, on estimating channel properties from training-based and blind methods and their use in coherent receivers.
