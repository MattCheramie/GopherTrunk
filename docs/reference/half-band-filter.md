---
slug: half-band-filter
title: Half-band filter
entry_type: algorithm
category: filtering-multirate
description: A half-band filter is an FIR whose passband and stopband are symmetric about a quarter of the sample rate, leaving nearly half its taps zero — ideal for cheap 2:1 decimation.
keywords: half-band filter, halfband, FIR filter, 2:1 decimation, interpolation, multirate, zero taps, efficient decimation chain, SDR filter
aka: [halfband filter, half-band FIR]
autolink: true
infobox:
  - { label: Type, value: Symmetric FIR (multirate) }
  - { label: Special property, value: ~½ of taps are exactly zero }
  - { label: Best for, value: 2:1 decimation / interpolation }
see_also: [fir-filter, decimation, polyphase-filter-bank, cic-filter, resampler]
cite_urls:
  - https://en.wikipedia.org/wiki/Half-band_filter
  - https://en.wikipedia.org/wiki/Finite_impulse_response
---

A **half-band filter** is a linear-phase [FIR filter](/reference/fir-filter/)
whose frequency response is antisymmetric about one quarter of the sample rate
(*f_s*/4), so its passband and stopband are mirror images and — the useful
consequence — **every other tap is exactly zero**.[^wiki] That structural zero
means a half-band filter does about half the work of an ordinary FIR of the same
length, which makes it the standard building block for efficient 2:1
[decimation](/reference/decimation/) and interpolation stages in
[software radio](/reference/software-defined-radio/) front-ends.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Impulse response of a half-band filter: a large centre tap, non-zero taps at odd offsets, and zero-valued taps at every even offset except the centre." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <line x1="30" y1="110" x2="440" y2="110" stroke="currentColor" stroke-width="1"/>
    <text x="235" y="128">tap index n (even taps = 0 except centre)</text>
    <line x1="235" y1="20" x2="235" y2="110" stroke="currentColor" stroke-width="2"/><text x="235" y="14">h[0]</text>
    <line x1="205" y1="70" x2="205" y2="110" stroke="currentColor" stroke-width="1.6"/>
    <line x1="265" y1="70" x2="265" y2="110" stroke="currentColor" stroke-width="1.6"/>
    <line x1="175" y1="92" x2="175" y2="110" stroke="currentColor" stroke-width="1.6"/>
    <line x1="295" y1="92" x2="295" y2="110" stroke="currentColor" stroke-width="1.6"/>
    <line x1="145" y1="100" x2="145" y2="110" stroke="currentColor" stroke-width="1.6"/>
    <line x1="325" y1="100" x2="325" y2="110" stroke="currentColor" stroke-width="1.6"/>
    <circle cx="115" cy="110" r="2.4" fill="none" stroke="currentColor"/>
    <circle cx="355" cy="110" r="2.4" fill="none" stroke="currentColor"/>
    <circle cx="205" cy="110" r="0" /><text x="150" y="70">non-zero at odd n</text>
    <text x="392" y="86">○ = zero tap</text>
  </g>
</svg>
<figcaption>A half-band impulse response: a dominant centre tap, tapering non-zero taps at odd offsets, and forced-zero taps at every even offset — so nearly half the multiplies vanish.</figcaption>
</figure>

## How it works

Design a symmetric low-pass FIR with its −6 dB cutoff exactly at *f_s*/4 and equal
passband/stopband ripple. The half-band constraint forces the impulse response to
satisfy *h*[*n*] = 0 for all even *n* except the centre tap, which equals 0.5.
Intuitively, the response has odd symmetry about the (0.5, *f_s*/4) point: whatever
it passes below *f_s*/4 it must reject by the same amount above it, and that pins
the even taps to zero.

Two things follow directly:

- **Half the arithmetic.** With ~*N*/2 of the *N* taps zero, you skip those
  multiplies entirely. Combined with FIR coefficient symmetry (fold-and-add the
  mirror taps), a half-band filter needs roughly *N*/4 multiplies per output.
- **Natural fit for 2:1.** Its cutoff at *f_s*/4 is exactly the new Nyquist edge
  after halving the rate, so it is the correct anti-alias filter for a
  decimate-by-2 step (and the correct anti-image filter for an interpolate-by-2
  step). Because the zero taps land on the samples you are about to discard,
  polyphase form skips them too.

## In practice

Large rate changes are built as a **cascade of half-band stages**, each halving the
rate. Early stages (still at a high rate, where most energy is far from the band of
interest) can use short, gentle half-bands; the final stage nearest the signal uses
a longer, sharper one. This "decreasing-length" chain keeps the total multiply-rate
tiny compared with one big filter at the input rate. Half-bands pair well with a
[CIC filter](/reference/cic-filter/) doing the bulk, coarse decimation and a
[polyphase](/reference/polyphase-filter-bank/) half-band cleaning up the passband
droop and providing the last 2:1. Their main limitation is that the fixed *f_s*/4
transition band offers no free choice of cutoff — you get 2:1 or you cascade.

## Relevance to SDR

Half-band chains are ubiquitous in SDR hardware and software: RTL-SDR and other
receivers decimate their raw ADC stream toward the channel rate through cascaded
half-bands, and virtually every SDR toolkit ships a half-band decimator. In a
trunking receiver they help bring a multi-megahertz capture down to the
48 kHz-class channel rate used by C4FM/π-4-DQPSK decoders. GopherTrunk's multirate
[decimation](/reference/decimation/) path relies on the same principle — do the
sharpest filtering only at the lowest rate — even where the exact stages are
general FIR/CIC rather than strictly half-band.

## Sources

[^wiki]: [Half-band filter](https://en.wikipedia.org/wiki/Half-band_filter) — Wikipedia, on the *f_s*/4-symmetric FIR with alternate zero taps used for efficient 2:1 rate change.
