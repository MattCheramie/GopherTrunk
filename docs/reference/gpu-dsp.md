---
slug: gpu-dsp
title: GPU DSP
entry_type: concept
category: sdr-app-building
description: "GPU DSP is running signal-processing work — FFTs, filtering, channelization — on a graphics processor's many parallel cores to gain throughput over a CPU."
keywords: GPU DSP, GPU signal processing, CUDA DSP, OpenCL FFT, GPU FFT, parallel filtering, channelizer on GPU, GPGPU radio, wideband DSP acceleration, cuFFT
aka: [GPU DSP, GPU signal processing, GPU-accelerated DSP]
autolink: true
infobox:
  - { label: Type, value: Parallel-compute DSP approach }
  - { label: Idea, value: Thousands of cores over sample blocks }
  - { label: Used in, value: "Wideband FFT, channelizers, RFML" }
see_also: [graphics-processing-unit, cuda, gpgpu, fast-fourier-transform, vectorization-simd, embedded-sdr]
cite_urls:
  - https://en.wikipedia.org/wiki/General-purpose_computing_on_graphics_processing_units
  - https://developer.nvidia.com/cufft
---

**GPU DSP** is the practice of running digital signal processing on a
[graphics processing unit](/reference/graphics-processing-unit/) instead of, or alongside, a
CPU, exploiting the GPU's thousands of small cores to process many samples at once.[^gpgpu]
Radio DSP is full of operations that apply the same arithmetic to huge blocks of independent
samples — [FFTs](/reference/fast-fourier-transform/), FIR filtering, mixing, channelization,
correlation — and that "same operation, many data" shape is exactly what a GPU is built for.
It is the large-scale cousin of [SIMD vectorization](/reference/vectorization-simd/) on the
CPU: where SIMD widens one core to a handful of lanes, a GPU spreads the work across
thousands.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A block of IQ samples is copied to GPU memory, processed in parallel by a grid of many small cores running the same FFT and filter kernel, then results are copied back to the host." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="gpar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="8" y="54" width="70" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="43" y="66">IQ block</text><text x="43" y="77">(host)</text>
    <rect x="150" y="30" width="160" height="86" rx="6" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="230" y="24">GPU: same kernel, many cores</text>
    <rect x="382" y="54" width="70" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="417" y="66">results</text><text x="417" y="77">(host)</text>
  </g>
  <g fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="0.8">
    <rect x="160" y="42" width="16" height="16"/><rect x="180" y="42" width="16" height="16"/><rect x="200" y="42" width="16" height="16"/><rect x="220" y="42" width="16" height="16"/><rect x="240" y="42" width="16" height="16"/><rect x="260" y="42" width="16" height="16"/><rect x="280" y="42" width="16" height="16"/>
    <rect x="160" y="62" width="16" height="16"/><rect x="180" y="62" width="16" height="16"/><rect x="200" y="62" width="16" height="16"/><rect x="220" y="62" width="16" height="16"/><rect x="240" y="62" width="16" height="16"/><rect x="260" y="62" width="16" height="16"/><rect x="280" y="62" width="16" height="16"/>
    <rect x="160" y="82" width="16" height="16"/><rect x="180" y="82" width="16" height="16"/><rect x="200" y="82" width="16" height="16"/><rect x="220" y="82" width="16" height="16"/><rect x="240" y="82" width="16" height="16"/><rect x="260" y="82" width="16" height="16"/><rect x="280" y="82" width="16" height="16"/>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="78" y1="69" x2="148" y2="69" marker-end="url(#gpar)"/>
    <line x1="310" y1="69" x2="380" y2="69" marker-end="url(#gpar)"/>
  </g>
</svg>
<figcaption>GPU DSP copies a block of samples to the device, runs one kernel across a grid of many cores in parallel, and copies the results back — throughput comes from breadth, not clock speed.</figcaption>
</figure>

## How it works

A GPU program (a **kernel**) is written once and launched across a grid of thousands of
threads; each thread handles one sample, one FFT bin, or one filter output. The programming
model is [CUDA](/reference/cuda/) on NVIDIA hardware or the vendor-neutral OpenCL/SYCL
elsewhere — both are forms of [general-purpose GPU computing](/reference/gpgpu/). Vendor
libraries do the heavy lifting: cuFFT and clFFT compute batched FFTs, and a channelizer is
often expressed as a large batch of small transforms plus a polyphase filter, which maps
beautifully onto the GPU's batching model.

The catch is **data movement**. Samples must be copied across the PCIe bus into GPU memory
and results copied back, and each kernel launch has fixed overhead. GPU DSP therefore only
wins when the block of work is large enough that compute dwarfs transfer time — wide
bandwidths, long FFTs, many channels, or deep filter banks. For a single narrowband channel
the copies cost more than they save, and a CPU with [SIMD](/reference/vectorization-simd/)
finishes first. Good GPU pipelines hide the transfers by overlapping copy and compute
(streaming) and by keeping intermediate results resident on the device across successive
stages.

## In practice

GPU acceleration pays off at the extremes of scale: real-time processing of tens or hundreds
of MHz, spectrum-monitoring systems that FFT enormous bands continuously, phased-array and
radar back ends, and RF machine-learning training where the same tensor math the GPU was
designed for is the workload. It is far less common — and often counterproductive — on the
small, power-constrained computers that host most scanners, where the PCIe copy overhead and
extra watts outweigh the gains and no discrete GPU is even present.

## Relevance to SDR

GPU DSP shows up wherever bandwidth or model size is large: research SDR platforms,
massive-MIMO and cellular test beds, wideband signal-intelligence receivers, and the
training side of [RF machine learning](/reference/rf-machine-learning/). It is a scaling
tool, not a default — most decoding of a single voice or trunking channel needs only a few
percent of one CPU core.

**GopherTrunk does not use the GPU.** GopherTrunk is a pure-Go decoder whose DSP —
down-conversion, filtering, timing and carrier recovery, symbol slicing, framing — runs on
the CPU, and it is designed to stay light enough to run comfortably on small
[single-board computers](/reference/single-board-computer/) and other
[embedded SDR](/reference/embedded-sdr/) hosts that have no discrete GPU at all. Its
performance strategy is efficient per-channel CPU code, not offloading, and it does not
implement CUDA/OpenCL kernels or any GPU-based machine learning. For the wideband,
many-channel or model-training regimes where GPUs genuinely help, that work belongs to
specialized frameworks; GopherTrunk deliberately keeps its footprint CPU-only and portable.

## Sources

[^gpgpu]: [General-purpose computing on graphics processing units](https://en.wikipedia.org/wiki/General-purpose_computing_on_graphics_processing_units) — Wikipedia, on running non-graphics parallel workloads on GPUs. See also [NVIDIA cuFFT](https://developer.nvidia.com/cufft) for batched GPU FFTs, the workhorse primitive of GPU DSP.
