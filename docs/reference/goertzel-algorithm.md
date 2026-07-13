---
slug: goertzel-algorithm
title: Goertzel algorithm
entry_type: algorithm
category: algorithms
description: The Goertzel algorithm evaluates a single DFT bin with a small recursive filter, giving SDRs a cheap way to detect specific tones like DTMF or CTCSS.
keywords: Goertzel algorithm, single-bin DFT, tone detection, DTMF decoding, CTCSS, second-order resonator, recursive filter, sub-audible tone, Gerald Goertzel
aka: [Goertzel algorithm, Goertzel filter]
autolink: true
infobox:
  - { label: Type, value: Single-bin DFT evaluator }
  - { label: Detects, value: One or a few specific tone frequencies }
  - { label: Complexity, value: O(N) per bin, no full FFT }
see_also: [discrete-fourier-transform, fast-fourier-transform, dtmf, iir-filter, energy-detection]
cite_urls:
  - https://en.wikipedia.org/wiki/Goertzel_algorithm
  - https://www.embedded.com/the-goertzel-algorithm/
---

The **Goertzel algorithm** computes the value of a single
[DFT](/reference/discrete-fourier-transform/) bin using a small recursive filter, so a
receiver can measure the energy at one chosen frequency without running a full
[FFT](/reference/fast-fourier-transform/).[^wiki] It behaves like a sharply tuned
second-order resonator that is run over a block of samples; when only a handful of target
tones matter, it is dramatically cheaper than transforming the whole spectrum and then
throwing most of it away.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A stream of samples feeds a two-tap recursive resonator with feedback coefficient two cosine omega, producing a single magnitude output for one target frequency." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <line x1="15" y1="60" x2="70" y2="60" stroke="currentColor" stroke-width="1.2" marker-end="url(#goar)"/><text x="40" y="52">x[n]</text>
    <circle cx="85" cy="60" r="12" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="85" y="63">+</text>
    <rect x="130" y="44" width="70" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="165" y="58">delay z^-1</text><text x="165" y="69">s[n-1]</text>
    <rect x="130" y="100" width="70" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="165" y="119">delay s[n-2]</text>
    <line x1="97" y1="60" x2="129" y2="60" stroke="currentColor" stroke-width="1.2" marker-end="url(#goar)"/>
    <line x1="200" y1="60" x2="260" y2="60" stroke="currentColor" stroke-width="1.2" marker-end="url(#goar)"/>
    <text x="335" y="52">magnitude at f0</text>
    <rect x="270" y="44" width="120" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="330" y="65">|X(f0)|^2</text>
    <path d="M165 76 V 100" fill="none" stroke="currentColor" stroke-width="1.1" marker-end="url(#goar)"/>
    <path d="M130 115 H 60 V 72" fill="none" stroke="currentColor" stroke-width="1.1" marker-end="url(#goar)"/>
    <text x="60" y="95">2cos&#969;</text>
  </g>
  <defs><marker id="goar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Goertzel runs a two-state resonator over the samples; after N inputs a short final step yields the magnitude at the one target frequency f0.</figcaption>
</figure>

## How it works

For a target bin *k* (frequency `f0 = k·fs/N`), define the coefficient `c = 2·cos(2πk/N)`.
The algorithm keeps two running state variables and, for each incoming sample `x[n]`,
updates them with a single recurrence:

`s[n] = x[n] + c·s[n-1] − s[n-2]`.

This is a two-pole [IIR-like](/reference/iir-filter/) resonator whose poles sit right on
the unit circle at the target frequency, so it accumulates energy there. After processing
all *N* samples, one short closing computation combines the two final states into the
complex DFT value `X[k]`, from which magnitude (and, if wanted, phase) is read. Only the
final step needs the complex twiddle factor; the inner loop is a real multiply-add,
making the per-sample cost tiny.

- **Cost.** Detecting *M* tones costs about `M·N` real multiply-adds, versus `N·log₂N`
  for a full FFT. When *M* is small — a few DTMF tones, one sub-audible squelch tone —
  Goertzel wins outright and needs no large buffer or bit-reversal.
- **Resolution and duration.** The effective bandwidth of the detector still follows
  `fs/N`, so a longer block gives a narrower, more selective response but a slower answer.
  Choosing *N* so the target lands exactly on a bin centre maximises the response and
  minimises leakage.
- **Off-grid tones.** Frequencies that fall between bins are still detected but with a
  slightly reduced, phase-dependent magnitude; a generalized Goertzel with a non-integer
  *k* recovers arbitrary frequencies exactly.

## In practice

Goertzel is the classic choice for **[DTMF](/reference/dtmf/)** decoding: the eight
keypad tones map to eight parallel Goertzel detectors, and a valid digit is declared when
exactly one row tone and one column tone exceed threshold with the right energy ratio.
The same pattern detects continuous control tones — CTCSS/"PL" sub-audible squelch tones,
selective-calling (SelCal) tones, and single-frequency signalling — where the receiver
only ever cares about a fixed, small list of frequencies. Because the inner loop is so
light, it is a staple on microcontrollers and DSPs that could not afford a streaming FFT.

## Relevance to SDR

In SDR pipelines Goertzel appears wherever a decoder must watch for a specific tone rather
than survey the band: sub-audible squelch-tone recognition on analog voice channels, MDC
and tone-signalling front ends, and quick presence/absence
[energy detection](/reference/energy-detection/) at known control frequencies. It
complements — rather than replaces — the FFT that drives full spectral displays.

GopherTrunk's primary targets are digital trunking protocols, which it decodes with
matched filtering and symbol recovery rather than tone banks, so a Goertzel stage is not
central to those chains. It remains the standard, well-understood tool for the analog
tone-detection tasks (CTCSS, DTMF) that sit alongside digital scanning, and is worth
reaching for any time only a few frequencies matter.

## Sources

[^wiki]: [Goertzel algorithm](https://en.wikipedia.org/wiki/Goertzel_algorithm) — Wikipedia, on the recursive single-bin DFT resonator and its use in tone detection.
