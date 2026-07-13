---
slug: shannon-capacity
title: Shannon capacity
entry_type: term
category: rf-fundamentals
description: Shannon capacity is the maximum error-free data rate of a channel, C = B·log2(1+SNR) bits per second, set by its bandwidth and signal-to-noise ratio.
keywords: Shannon capacity, channel capacity, Shannon-Hartley theorem, C equals B log2 1 plus SNR, information theory, spectral efficiency, bits per second per hertz, Claude Shannon
aka: [Shannon capacity, channel capacity, Shannon limit, Shannon-Hartley theorem]
autolink: true
infobox:
  - { label: Type, value: Information-theoretic limit }
  - { label: Formula, value: "C = B·log2(1 + SNR)  bits/s" }
  - { label: Due to, value: "Claude Shannon (1948)" }
see_also: [claude-shannon, signal-to-noise-ratio, spectral-efficiency, forward-error-correction, bandwidth, eb-n0]
cite_urls:
  - https://en.wikipedia.org/wiki/Shannon%E2%80%93Hartley_theorem
  - https://en.wikipedia.org/wiki/Channel_capacity
---

**Shannon capacity** is the highest rate at which information can be sent over a channel with an
arbitrarily small error probability: for a bandlimited channel with additive Gaussian noise,
**C = B · log₂(1 + SNR)** bits per second, where *B* is the [bandwidth](/reference/bandwidth/) in
hertz and **SNR** is the linear [signal-to-noise ratio](/reference/signal-to-noise-ratio/).[^wiki]
Proved by [Claude Shannon](/reference/claude-shannon/) in 1948, it is a hard ceiling — no code or
modulation can exceed it — and the benchmark against which every real radio link is judged.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A curve of channel capacity versus signal-to-noise ratio: capacity rises steeply then logarithmically, with the formula C equals B log base two of one plus SNR labelling the curve." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor" stroke="none">
    <line x1="45" y1="20" x2="45" y2="140" stroke="currentColor" stroke-opacity="0.5"/>
    <line x1="45" y1="140" x2="435" y2="140" stroke="currentColor" stroke-opacity="0.5"/>
    <text x="8" y="80" >C</text>
    <text x="410" y="158">SNR (dB)</text>
    <path d="M45 140 C 120 90 200 62 300 46 C 360 37 400 32 435 30" fill="none" stroke="currentColor"/>
    <text x="250" y="70" font-style="italic">C = B·log₂(1 + SNR)</text>
    <text x="60" y="130" fill-opacity="0.7">near 0 dB: ~1 bit/s/Hz</text>
    <text x="300" y="90" fill-opacity="0.7">+3 dB SNR ≈ +1 bit/s/Hz</text>
  </g>
</svg>
<figcaption>Capacity grows only logarithmically with SNR: at high SNR each extra 3 dB buys roughly one more bit per second per hertz.</figcaption>
</figure>

## How it works

Two resources bound how much information a channel can carry: how *wide* it is and how far the signal
stands above the noise. Shannon's theorem combines them. The **bandwidth** *B* sets how many
independent symbols per second the channel supports (via the Nyquist rate), and the **SNR** sets how
many distinguishable levels each symbol can reliably carry — the `log₂(1 + SNR)` factor is the number
of bits those levels encode. Their product is the capacity in bits per second.

Dividing through by bandwidth gives the **[spectral efficiency](/reference/spectral-efficiency/)**,
*C/B = log₂(1 + SNR)* bits per second per hertz — the ceiling on how many bits each hertz of spectrum
can deliver. Two behaviours follow. At **high SNR** capacity grows only *logarithmically*: every
doubling of SNR (+3 dB) adds roughly one bit/s/Hz, so pushing rate up by brute-force power hits
diminishing returns. At **low SNR** capacity is nearly linear in SNR and you can trade bandwidth for
power — spreading a weak signal over more hertz still conveys the bits, which is the principle behind
spread-spectrum and deep-space links.

Crucially, Shannon proved capacity is *achievable* — codes exist that approach *C* with vanishing
error — but the proof is non-constructive. The gap between the limit and real systems is what decades
of [forward error correction](/reference/forward-error-correction/) work has closed:
[turbo codes](/reference/turbo-code/) and [LDPC codes](/reference/ldpc-code/) now operate within a
fraction of a dB of the Shannon limit.

## In practice

The theorem reframes engineering as a budget. Given a required data rate you can ask what combination
of bandwidth and SNR meets it; given fixed bandwidth and power you know the maximum rate worth
attempting. It also defines a related floor, the minimum energy per bit to noise density
**[Eb/N0](/reference/eb-n0/)** of about −1.6 dB, below which reliable communication is impossible at
any bandwidth. Every modulation-and-coding scheme in a modern standard is, in effect, a chosen point
on the Shannon curve trading spectral efficiency against robustness.

## Relevance to SDR

Shannon capacity explains the fundamental limits of the signals an [SDR](/reference/software-defined-radio/)
receives. A trunking or cellular waveform's choice of modulation order and code rate is a point on
this curve, and the [SNR](/reference/signal-to-noise-ratio/) at your antenna decides whether the link
can support that rate — below the required SNR, no amount of receiver cleverness recovers the data,
because the transmitter already sent at a rate the channel cannot sustain at your noise level. This is
the theoretical backing for why improving SNR (better antenna, [LNA](/reference/low-noise-amplifier/),
lower [noise figure](/reference/noise-figure/)) is the lever that turns a failing decode into a
working one.

**GopherTrunk** does not compute capacity, but the principle underlies its whole decode chain:
recovering bits from a waveform is only possible when the received SNR sits above what that waveform's
coding and modulation require, exactly the boundary Shannon's formula draws.

## Sources

[^wiki]: [Shannon–Hartley theorem](https://en.wikipedia.org/wiki/Shannon%E2%80%93Hartley_theorem) — Wikipedia, the C = B·log₂(1+SNR) capacity formula and its assumptions.
[^cap]: [Channel capacity](https://en.wikipedia.org/wiki/Channel_capacity) — Wikipedia, the information-theoretic definition and achievability of capacity.
