---
slug: volk
title: VOLK
entry_type: technology
category: sdr-frameworks
description: "VOLK is the Vector-Optimized Library of Kernels: a library of hand-tuned SIMD math routines for SDR that picks the fastest implementation for the host CPU at run time."
keywords: VOLK, Vector-Optimized Library of Kernels, SIMD, SSE, AVX, NEON, GNU Radio, kernel dispatch, volk_profile, vectorized DSP
aka: [VOLK, Vector-Optimized Library of Kernels]
autolink: true
infobox:
  - { label: Type, value: SIMD DSP kernel library }
  - { label: Idea, value: One math call, fastest CPU path picked at run time }
  - { label: Origin, value: GNU Radio project }
see_also: [vectorization-simd, gnuradio, liquid-dsp, fir-filter, fast-fourier-transform, benchmarking-dsp]
cite_urls:
  - https://www.libvolk.org/
  - https://github.com/gnuradio/volk
---

**VOLK** — the **Vector-Optimized Library of Kernels** — is a library of hand-tuned,
[SIMD](/reference/vectorization-simd/)-accelerated math routines for
software-defined-radio DSP, together with a dispatcher that selects the fastest available
implementation for the CPU it is running on.[^volk] It grew out of the
[GNU Radio](/reference/gnuradio/) project to solve a specific problem: the inner loops of a
radio — multiply-accumulate, magnitude, phase, format conversion — run billions of times, and
writing them once per instruction set is both tedious and brittle. VOLK centralizes those
kernels so every application shares one optimized, portable copy.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A single VOLK function call dispatching at run time to one of several instruction-set implementations — generic C, SSE, AVX, or NEON — chosen for the host CPU." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="vkar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="150" y="12" width="160" height="28" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="30" text-anchor="middle" font-size="8" fill="currentColor">volk_32fc_x2_multiply_32fc(...)</text>
  <rect x="150" y="58" width="160" height="24" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="230" y="74" text-anchor="middle" font-size="8" fill="currentColor">run-time dispatcher</text>
  <g font-size="7.5" fill="currentColor" text-anchor="middle">
    <rect x="20" y="108" width="86" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="63" y="124">generic C</text>
    <rect x="130" y="108" width="70" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="165" y="124">SSE</text>
    <rect x="224" y="108" width="70" height="26" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="259" y="124">AVX</text>
    <rect x="318" y="108" width="86" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="361" y="124">NEON (ARM)</text>
  </g>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <line x1="230" y1="40" x2="230" y2="58" marker-end="url(#vkar)"/>
    <line x1="200" y1="82" x2="70" y2="106" marker-end="url(#vkar)"/>
    <line x1="222" y1="82" x2="168" y2="106" marker-end="url(#vkar)"/>
    <line x1="242" y1="82" x2="259" y2="106" marker-end="url(#vkar)"/>
    <line x1="262" y1="82" x2="355" y2="106" marker-end="url(#vkar)"/>
  </g>
  <text x="360" y="98" text-anchor="middle" font-size="7.5" fill="currentColor">chosen for this CPU</text>
</svg>
<figcaption>One VOLK call maps to many machine-specific implementations; the dispatcher picks the fastest the host CPU supports, so the same source runs optimally on x86 and ARM alike.</figcaption>
</figure>

## How it works

VOLK is organized around **kernels**: named vector operations such as "multiply two complex
float arrays," "compute the magnitude of a complex vector," or "convert 16-bit integers to
floats." Each kernel ships with several *protokernels* — independent implementations of the
same math targeting different instruction sets: a portable **generic** C version plus tuned
variants for SSE, AVX/AVX2/AVX-512 on x86 and NEON on ARM. The public API is a plain C
function per kernel; the caller never sees the variants.

Selection happens by CPU-feature detection. At load time VOLK reads the processor's capability
flags and binds each kernel's function pointer to the best protokernel the machine actually
supports, so a binary compiled once runs the AVX path on a modern desktop and falls back to
NEON or generic C elsewhere — no recompilation, no `#ifdef` maze in the caller. To resolve ties
where several variants are viable, the `volk_profile` tool benchmarks every protokernel on the
real hardware and writes a small config file recording the empirically fastest choice per kernel.

Two implementation details make VOLK effective in practice:

- **Alignment awareness** — each kernel provides aligned and unaligned entry points, because
  SIMD loads are fastest when data sits on 16- or 32-byte boundaries; VOLK exposes an aligned
  allocator so callers can get that speed.
- **Correctness harness** — a QA suite checks every protokernel against the generic reference so
  the AVX and NEON paths produce the same numbers, which matters when a decoder's bit decisions
  depend on them.

## Relevance to SDR

VOLK is the numerical engine under a great deal of open SDR software. Inside
[GNU Radio](/reference/gnuradio/), the hot loops of FIR filters, the frequency-translating
filter, magnitude and AGC blocks, and sample-format converters call VOLK, which is a large part
of why GNU Radio sustains high sample rates on commodity CPUs. Because it is a standalone C
library, projects outside GNU Radio link it directly to accelerate their own
[FIR filters](/reference/fir-filter/), correlators, and
[FFT](/reference/fast-fourier-transform/) front ends. It is, in effect, the community's shared
answer to "make this DSP inner loop fast on whatever CPU the user has," a concern it shares with
math-heavy libraries like [liquid-dsp](/reference/liquid-dsp/).

**GopherTrunk** does not use VOLK. GopherTrunk is written in Go, not C, and its DSP inner loops
are Go code that leans on the Go compiler and, where it matters, on Go's own facilities rather
than a C SIMD library — keeping the project a single dependency-free static binary. VOLK is
still directly relevant as the reference model for the problem GopherTrunk must also solve:
[SIMD](/reference/vectorization-simd/)-friendly, cache-aware, run-time-portable inner loops.
Where GNU Radio reaches for VOLK, GopherTrunk relies on careful Go and rate-invariant design to
keep real-time decoding within budget on the same commodity hardware.

## Sources

[^volk]: [libvolk.org](https://www.libvolk.org/) — the VOLK project site and documentation, describing kernels and protokernels, run-time CPU dispatch, `volk_profile`, aligned allocation, and the QA harness.
