---
slug: vectorization-simd
title: SIMD vectorization
entry_type: concept
category: sdr-programming
description: "SIMD vectorization applies one instruction to many samples at once (SSE, AVX, NEON), the main way CPUs accelerate DSP inner loops in SDR software."
keywords: SIMD, vectorization, SSE, AVX, AVX2, AVX-512, NEON, VOLK, single instruction multiple data, DSP acceleration, vector instructions, data parallelism
aka: [SIMD, vectorisation, vector processing]
autolink: true
infobox:
  - { label: Type, value: CPU data-parallel acceleration }
  - { label: Idea, value: One instruction over a vector of samples }
  - { label: ISAs, value: "SSE / AVX (x86), NEON (ARM)" }
see_also: [volk, multithreaded-dsp, gpu-dsp, vector-processor, hardware-acceleration]
cite_urls:
  - https://en.wikipedia.org/wiki/Single_instruction,_multiple_data
  - https://en.wikipedia.org/wiki/Advanced_Vector_Extensions
---

**SIMD vectorization** — Single Instruction, Multiple Data — is the technique of applying one
CPU instruction to several data elements simultaneously, packed side by side in a wide
register.[^wiki] Because digital signal processing hammers the same arithmetic across long
streams of samples, SIMD is the single most effective way to speed up DSP inner loops on a
general-purpose CPU, and it underpins how [SDR](/reference/software-defined-radio/) software
sustains multi-megasample-per-second rates in software rather than dedicated hardware.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A scalar add processes one pair of numbers per instruction while a SIMD add processes four pairs with a single instruction, giving four times the work per cycle." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="18" font-size="9" fill="currentColor">scalar: 1 add per instruction</text>
  <rect x="20" y="26" width="24" height="18" fill="none" stroke="currentColor"/><text x="32" y="39" font-size="9" fill="currentColor" text-anchor="middle">a</text>
  <text x="52" y="39" font-size="10" fill="currentColor">+</text>
  <rect x="62" y="26" width="24" height="18" fill="none" stroke="currentColor"/><text x="74" y="39" font-size="9" fill="currentColor" text-anchor="middle">b</text>
  <text x="94" y="39" font-size="10" fill="currentColor">=</text>
  <rect x="104" y="26" width="24" height="18" fill="none" stroke="currentColor"/>
  <text x="20" y="78" font-size="9" fill="currentColor">SIMD: 4 adds per instruction</text>
  <g>
    <rect x="20" y="86" width="96" height="18" fill="none" stroke="currentColor"/>
    <line x1="44" y1="86" x2="44" y2="104" stroke="currentColor" stroke-opacity="0.4"/><line x1="68" y1="86" x2="68" y2="104" stroke="currentColor" stroke-opacity="0.4"/><line x1="92" y1="86" x2="92" y2="104" stroke="currentColor" stroke-opacity="0.4"/>
  </g>
  <text x="126" y="99" font-size="10" fill="currentColor">+</text>
  <g>
    <rect x="138" y="86" width="96" height="18" fill="none" stroke="currentColor"/>
    <line x1="162" y1="86" x2="162" y2="104" stroke="currentColor" stroke-opacity="0.4"/><line x1="186" y1="86" x2="186" y2="104" stroke="currentColor" stroke-opacity="0.4"/><line x1="210" y1="86" x2="210" y2="104" stroke="currentColor" stroke-opacity="0.4"/>
  </g>
  <text x="244" y="99" font-size="10" fill="currentColor">=</text>
  <g>
    <rect x="256" y="86" width="96" height="18" fill="none" stroke="currentColor"/>
    <line x1="280" y1="86" x2="280" y2="104" stroke="currentColor" stroke-opacity="0.4"/><line x1="304" y1="86" x2="304" y2="104" stroke="currentColor" stroke-opacity="0.4"/><line x1="328" y1="86" x2="328" y2="104" stroke="currentColor" stroke-opacity="0.4"/>
  </g>
  <text x="230" y="132" font-size="8.5" fill="currentColor" text-anchor="middle">one instruction, one vector register — four sample lanes at once</text>
</svg>
<figcaption>A SIMD instruction packs several samples into one wide register and operates on all lanes together, so a four-lane add does the work of four scalar adds in a single instruction.</figcaption>
</figure>

## How it works

A modern CPU register can be 128, 256, or 512 bits wide. A 256-bit AVX register holds eight
32-bit floats; one `vaddps` instruction adds all eight lanes at once. DSP kernels are ideal
for this because their arithmetic is *data-parallel* — a FIR filter multiplies each of *N*
coefficients by a sample and sums, a mixer multiplies every sample by a rotating phasor, a
power detector squares and sums — the same operation, independent across samples. Packing
samples into vector lanes turns these loops into a handful of wide instructions.

The major instruction-set families are:

- **SSE** — 128-bit x86 vectors (4 floats), near-universal on desktop/server CPUs.
- **AVX / AVX2 / AVX-512** — 256- and 512-bit x86 vectors, with fused multiply-add (FMA)
  that does a multiply and an add in one instruction, exactly the FIR/complex-multiply
  pattern.
- **NEON** — 128-bit ARM vectors, the reason a Raspberry Pi or phone can run real-time SDR.

Getting the speedup is the hard part. Auto-vectorizing compilers help but often fail on
strided, complex-valued, or reduction-heavy code. Hand-written *intrinsics* give control but
are per-ISA: the same kernel needs an SSE, an AVX, and a NEON version, plus runtime dispatch
to pick the widest the host supports. Data must also be laid out contiguously and aligned so
whole vectors load in one go — interleaved complex I/Q, for instance, is often deinterleaved
so real and imaginary parts vectorise cleanly.

## In practice

Because writing and dispatching per-ISA kernels is tedious, most SDR stacks lean on a kernel
library. [VOLK](/reference/volk/) (the Vector-Optimized Library of Kernels, from the GNU Radio
project) is the canonical example: it ships many hand-tuned SIMD implementations of common DSP
primitives — complex multiply, dot product, magnitude, conversions — and at load time probes
the CPU and binds each call to the fastest available version. Application code calls one
portable function and transparently gets AVX-512 on a new Xeon or NEON on an ARM board. SIMD
is complementary to the other two acceleration axes: [multithreaded DSP](/reference/multithreaded-dsp/)
spreads *blocks* across cores, and [GPU DSP](/reference/gpu-dsp/) offloads massively parallel
work to a graphics processor; SIMD extracts parallelism *within* a single core's inner loop.

## Relevance to SDR

SIMD is what makes software radio viable on commodity CPUs. Down-conversion, filtering,
resampling, and demodulation all reduce to vectorizable multiply-accumulate loops, and at
several megasamples per second a scalar implementation would saturate a core while a
vectorized one leaves headroom for more channels. **GopherTrunk** is written in Go, which
does not expose SIMD intrinsics in portable code and leans on the compiler and scalar-clean
kernels rather than a VOLK-style hand-tuned library; its performance strategy is careful
data layout, avoiding allocation in hot loops, and concurrency across goroutines rather than
manual vectorization. That is an honest contrast worth understanding: the wider C/C++ SDR
ecosystem (GNU Radio, liquid-dsp, csdr) depends heavily on SIMD via VOLK, whereas a Go SDR
application trades some peak per-core throughput for simplicity and portability, recovering it
through parallelism and lean allocation instead. Either way, the concept — one instruction
over many samples — is the foundation of real-time DSP on a CPU.

## Sources

[^wiki]: [Single instruction, multiple data](https://en.wikipedia.org/wiki/Single_instruction,_multiple_data) — Wikipedia, on data-parallel vector execution and its use in signal processing.
