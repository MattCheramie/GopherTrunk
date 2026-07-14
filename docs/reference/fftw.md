---
slug: fftw
title: FFTW
entry_type: technology
category: sdr-frameworks
description: "FFTW (the Fastest Fourier Transform in the West) is a portable C library that computes the discrete Fourier transform at near-optimal speed by planning the transform for the host machine."
keywords: FFTW, Fastest Fourier Transform in the West, FFT library, discrete Fourier transform, wisdom, planner, codelet, MIT, spectral analysis, Frigo Johnson
aka: [FFTW, Fastest Fourier Transform in the West]
autolink: true
infobox:
  - { label: Type, value: FFT computation library (C) }
  - { label: Idea, value: Plan the transform for this machine }
  - { label: Origin, value: MIT (Frigo & Johnson) }
see_also: [fast-fourier-transform, discrete-fourier-transform, power-spectral-density, liquid-dsp, gnuradio, window-function]
cite_urls:
  - https://www.fftw.org/
  - https://en.wikipedia.org/wiki/FFTW
---

**FFTW** — the **Fastest Fourier Transform in the West** — is a widely used, portable C
library for computing the [discrete Fourier transform](/reference/discrete-fourier-transform/)
and its inverse, in one or more dimensions, at near-optimal speed.[^fftw] Developed at MIT by
Matteo Frigo and Steven G. Johnson, it earned its name and reputation by matching or beating
hand-tuned, vendor-specific FFT code while remaining fully portable — it adapts to the machine
rather than being rewritten for each one. It is the default high-performance
[FFT](/reference/fast-fourier-transform/) engine in an enormous range of scientific and
signal-processing software.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="FFTW's two-phase model: a planner measures candidate algorithms on the host and stores wisdom, then the executor runs the chosen fast plan repeatedly on incoming data." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="ftar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="14" y="24" width="150" height="44" rx="6" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="89" y="42" text-anchor="middle" font-size="8.5" fill="currentColor">PLAN (once)</text>
  <text x="89" y="55" text-anchor="middle" font-size="7.5" fill="currentColor">measure candidates</text>
  <rect x="14" y="90" width="150" height="30" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/>
  <text x="89" y="109" text-anchor="middle" font-size="7.5" fill="currentColor">wisdom (saved plan)</text>
  <rect x="292" y="24" width="154" height="44" rx="6" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/>
  <text x="369" y="42" text-anchor="middle" font-size="8.5" fill="currentColor">EXECUTE (many)</text>
  <text x="369" y="55" text-anchor="middle" font-size="7.5" fill="currentColor">run the fast plan</text>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="164" y1="46" x2="290" y2="46" marker-end="url(#ftar)"/>
    <line x1="89" y1="68" x2="89" y2="88" marker-end="url(#ftar)"/>
    <line x1="130" y1="105" x2="291" y2="60" marker-end="url(#ftar)"/>
  </g>
  <text x="227" y="38" text-anchor="middle" font-size="7.5" fill="currentColor">chosen plan</text>
  <text x="230" y="134" text-anchor="middle" font-size="9" fill="currentColor">plan once, transform many — reuse the tuned plan on each buffer</text>
</svg>
<figcaption>FFTW splits work into planning and execution: a planner finds the fastest algorithm for the exact transform on this machine and saves it as "wisdom," then the executor reuses that plan on every buffer.</figcaption>
</figure>

## How it works

FFTW's central idea is to separate **planning** from **execution**. Before transforming data,
the caller asks FFTW to create a *plan* for a specific transform — a given size, dimensionality,
direction, and data layout. The **planner** explores many ways to decompose that transform,
composing small optimized code fragments called **codelets** (generated automatically by FFTW's
own code generator) into a full algorithm, and — in its measuring modes — actually times the
candidates on the host to find the fastest combination. Once built, the plan is executed
repeatedly on real data; the expensive search happens once and the fast path runs on every
buffer thereafter.

Several design choices follow from this:

- **Machine adaptation, not machine-specific source.** FFTW discovers the best algorithm at run
  time, so a single portable build performs well across CPUs, cache sizes, and SIMD capabilities.
  Its executor uses SSE/AVX and NEON where present.
- **Arbitrary sizes.** It is not limited to powers of two; it factors composite lengths and has
  dedicated algorithms for prime sizes, so real, complex, and multidimensional transforms of
  almost any length are handled efficiently.
- **Wisdom.** The result of planning can be exported to disk as *wisdom* and reloaded later, so a
  program need not re-measure on every startup — planning cost is paid once and amortized across
  runs.

The API is a small set of `plan`, `execute`, and `destroy` calls, with variants for
real-to-complex, complex-to-complex, and multidimensional transforms.

## Relevance to SDR

The FFT is the workhorse of software radio, so a fast FFT library matters everywhere spectra are
computed. FFTW is the engine behind the [power spectral density](/reference/power-spectral-density/)
estimates, waterfalls, and spectrograms in countless SDR tools, and behind the frequency-domain
filtering (overlap-add / overlap-save) and channelizers that process wideband captures. It also
accelerates transforms inside larger DSP libraries such as [liquid-dsp](/reference/liquid-dsp/),
which uses FFTW when it is available. [GNU Radio](/reference/gnuradio/) and its ecosystem lean on
fast FFTs for exactly these visualization and filtering tasks. When a receiver needs to turn a
buffer of [IQ](/reference/iq-data/) samples into a spectrum many times a second, FFTW is the usual
reason it can keep up.

**GopherTrunk** does not link FFTW, because it is a pure-Go program with no C dependencies and
ships as one static binary; where it needs an FFT it uses a Go implementation. FFTW remains highly
relevant as context, though: it is the reference standard for FFT performance, the benchmark
against which other transforms (including Go ones) are judged, and the concrete example of the
"plan once, run many" pattern that any real-time spectral pipeline — GopherTrunk's included —
must adopt to avoid recomputing setup work on every buffer.

## Sources

[^fftw]: [FFTW](https://www.fftw.org/) — the official project site and documentation, describing the planner/executor split, codelets and the code generator, wisdom, arbitrary transform sizes, and SIMD support.
