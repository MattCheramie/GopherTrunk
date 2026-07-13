---
slug: eb-n0
title: Eb/N0 (energy per bit to noise density)
entry_type: term
category: rf-metrics
description: Eb/N0 is the ratio of energy per information bit to noise power spectral density, the normalized SNR that lets different modems and code rates be compared fairly.
keywords: Eb/N0, energy per bit, noise spectral density, EbNo, normalized SNR, digital communications, coding gain, Shannon limit, modem comparison
aka: [EbNo, Eb over N0, energy-per-bit to noise-density ratio]
autolink: true
infobox:
  - { label: Symbol, value: "Eb/N0" }
  - { label: Unit, value: Decibels (dB) }
  - { label: Relation, value: "Eb/N0 = (SNR)·(B/Rb)" }
see_also: [bit-error-rate, signal-to-noise-ratio, shannon-capacity, carrier-to-noise-ratio, spectral-efficiency]
cite_urls:
  - https://en.wikipedia.org/wiki/Eb/N0
  - https://en.wikipedia.org/wiki/Shannon%E2%80%93Hartley_theorem
---

**Eb/N0** (spoken "E-b over N-zero") is the ratio of the energy carried by one
information bit, Eb, to the noise [power spectral density](/reference/power-spectral-density/),
N0.[^wiki] It is a normalized, bandwidth-independent form of
[signal-to-noise ratio](/reference/signal-to-noise-ratio/) and is the metric of choice
for comparing digital modems, because it strips out the effects of bit rate, bandwidth,
and modulation order — leaving a fair basis on which to plot
[bit error rate](/reference/bit-error-rate/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A diagram relating SNR to Eb/N0 by dividing signal power over noise density by the bit rate, with the Shannon limit marked at minus 1.6 dB on an Eb/N0 axis." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="30" width="120" height="40" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="90" y="46" text-anchor="middle" font-size="9" fill="currentColor">SNR = S/(N0·B)</text>
  <text x="90" y="60" text-anchor="middle" font-size="9" fill="currentColor">(in-band)</text>
  <defs><marker id="ebar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="150" y1="50" x2="215" y2="50" stroke="currentColor" stroke-width="1.4" marker-end="url(#ebar)"/>
  <text x="182" y="42" text-anchor="middle" font-size="8" fill="currentColor">÷ (Rb/B)</text>
  <rect x="220" y="30" width="120" height="40" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="280" y="52" text-anchor="middle" font-size="9" fill="currentColor">Eb/N0</text>
  <line x1="60" y1="120" x2="420" y2="120" stroke="currentColor" stroke-opacity="0.6"/>
  <text x="380" y="135" font-size="9" fill="currentColor">Eb/N0 (dB) →</text>
  <line x1="90" y1="112" x2="90" y2="128" stroke="currentColor"/>
  <text x="72" y="150" font-size="8" fill="currentColor">−1.6 dB</text>
  <text x="72" y="162" font-size="8" fill="currentColor">Shannon limit</text>
</svg>
<figcaption>Eb/N0 is in-band SNR divided by spectral efficiency (bit rate over bandwidth); no system can transmit reliably below the −1.6 dB Shannon limit.</figcaption>
</figure>

## How it works

The link is Eb/N0 = SNR × (B / Rb), where B is the noise bandwidth and Rb the
information bit rate. Equivalently, since Eb = S/Rb and total noise N = N0·B, the
two SNR forms differ only by the ratio of bandwidth to bit rate — that is, by the
system's [spectral efficiency](/reference/spectral-efficiency/). Because Eb/N0 divides
signal power by the number of bits per second it carries, a slow, robust link and a
fast, dense link that achieve the same BER report the same Eb/N0, even though their
raw SNRs differ. That is exactly what makes it the fair comparison metric.

Every modulation-and-coding scheme has a characteristic BER-versus-Eb/N0 curve. The
required Eb/N0 for a target BER is the headline number quoted in link budgets and
standards. **Coding gain** is the reduction in required Eb/N0 that
[forward error correction](/reference/forward-error-correction/) buys at a given BER —
modern [turbo](/reference/turbo-code/) and [LDPC](/reference/ldpc-code/) codes operate
within a fraction of a dB of the theoretical limit.

## In practice

- The [Shannon–Hartley theorem](/reference/shannon-capacity/) sets an absolute floor:
  as spectral efficiency approaches zero, the minimum Eb/N0 for error-free
  communication converges to ln 2 ≈ −1.6 dB. No system, however cleverly coded, can
  operate reliably below it.
- Uncoded [BPSK](/reference/bpsk/) needs about 9.6 dB of Eb/N0 for a 10⁻⁵ BER; a good
  LDPC code reaches the same BER a few dB lower, and dense
  [QAM](/reference/quadrature-amplitude-modulation/) needs more.
- Deep-space and satellite links quote Eb/N0 directly because their power budgets are
  tight and every tenth of a dB of coding gain matters.

## Relevance to SDR

Eb/N0 is the currency of link-budget engineering and the reason different digital
voice systems have different range. [P25](/reference/p25-phase-1/) and
[DMR](/reference/dmr/) at 4800 symbols/s, [TETRA](/reference/tetra/) at higher gross
rates, and narrowband [NXDN](/reference/nxdn/) each demand a specific Eb/N0 for reliable
decoding, and comparing them meaningfully requires normalizing out their differing
rates and bandwidths — which is precisely what Eb/N0 does. For a decoder like
[GopherTrunk](/reference/software-defined-radio/), the practical takeaway is that a
signal's raw SNR must be interpreted against the mode's bit rate: a healthy SNR for a
slow control channel may be marginal for a faster
[voice channel](/reference/voice-channel/) sharing the same site.

## Sources

[^wiki]: [Eb/N0](https://en.wikipedia.org/wiki/Eb/N0) — Wikipedia, definition and relationship to SNR, spectral efficiency, and the Shannon limit.
