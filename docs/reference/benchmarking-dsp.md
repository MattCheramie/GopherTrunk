---
slug: benchmarking-dsp
title: Benchmarking DSP
entry_type: concept
category: sdr-app-building
description: "Benchmarking DSP measures signal-processing throughput in samples per second and profiles where CPU time is spent, to confirm a pipeline keeps up with the sample rate in real time."
keywords: benchmarking DSP, DSP throughput, samples per second, Msps, profiling, hotspot, flame graph, real-time budget, microbenchmark, cycles per sample, SDR performance
aka: ["DSP benchmarking", "DSP profiling", "throughput measurement"]
autolink: true
infobox:
  - { label: Type, value: "Performance-measurement practice" }
  - { label: Measures, value: "Throughput (Msps), CPU time per stage" }
  - { label: Goal, value: "Meet the real-time sample-rate budget" }
see_also: [real-time-dsp, vectorization-simd, volk, multithreaded-dsp, cache-memory]
cite_urls:
  - https://en.wikipedia.org/wiki/Profiling_(computer_programming)
  - https://en.wikipedia.org/wiki/Benchmark_(computing)
---

**Benchmarking DSP** is the practice of measuring how fast a signal-processing
pipeline runs — its throughput in samples per second — and profiling *where* the
CPU time goes, so you can confirm the code keeps up with the incoming sample rate
and find the stages worth optimizing.[^bench] In software radio the hard
constraint is simple: to process a stream arriving at, say, 2.4 Msps in real time,
the pipeline must consume more than 2.4 million samples every second, forever, or
it falls behind and drops data.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A horizontal bar profile shows CPU time spent per DSP stage, with the FIR filter as the dominant hotspot, next to a real-time budget line the total must stay under." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor">
    <text x="10" y="18" font-size="9">CPU time per stage (profile)</text>
    <g text-anchor="end">
      <text x="86" y="42">downconvert</text><text x="86" y="66">FIR filter</text><text x="86" y="90">demod</text><text x="86" y="114">framing</text>
    </g>
    <rect x="94" y="33" width="90" height="14" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="0.8"/>
    <rect x="94" y="57" width="220" height="14" fill="currentColor" fill-opacity="0.5" stroke="currentColor" stroke-width="0.8"/><text x="320" y="68" font-size="8">hotspot</text>
    <rect x="94" y="81" width="70" height="14" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="0.8"/>
    <rect x="94" y="105" width="30" height="14" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="0.8"/>
    <line x1="360" y1="26" x2="360" y2="126" stroke="currentColor" stroke-width="1.3" stroke-dasharray="4 3"/>
    <text x="366" y="70" font-size="8">real-time</text><text x="366" y="82" font-size="8">budget</text>
  </g>
</svg>
<figcaption>A profile attributes CPU time to each stage; total time per sample must stay under the real-time budget set by the sample rate.</figcaption>
</figure>

## How it works

Benchmarking answers two different questions with two different tools.
**Microbenchmarks** time a single function or block in isolation — run a
[FIR filter](/reference/fir-filter/) over a fixed buffer many times and report
samples per second or nanoseconds per sample. This gives a clean, comparable
number for one operation and is ideal for checking whether an optimization
actually helped. **Profiling** runs the whole pipeline and attributes elapsed CPU
time to each function, typically by statistical sampling of the call stack, so you
can see which stage dominates — usually visualized as a flame graph or a sorted
list of hotspots.[^prof]

The guiding principle is Amdahl's law in practice: optimizing a stage that uses 3%
of the time can never yield more than a 3% speedup, so you profile *first* and
optimize the tall bar. In DSP that tall bar is almost always the tight per-sample
inner loops — filtering, mixing, FFTs — because they touch every one of millions
of samples per second, while the framing and protocol logic runs comparatively
rarely.

Useful derived metrics:

- **Throughput (Msps)** — samples processed per second; compare against the
  required input rate to get headroom.
- **Cycles or nanoseconds per sample** — normalizes throughput so results are
  comparable across sample rates and machines.
- **Real-time factor** — how many times faster than real time an offline run
  completes; a replay that finishes 20× faster than real time has ample margin.

## In practice

Benchmarks must be run on representative input and a warmed-up machine, with
enough iterations to swamp measurement noise, and ideally pinned so a background
process or CPU frequency scaling does not corrupt the number. Because results are
comparative, the highest-value use is regression detection: record a baseline and
fail CI if throughput drops, catching a change that quietly halves performance
before it ships.

Optimization then follows the hotspot. The classic DSP levers are
[SIMD vectorization](/reference/vectorization-simd/) (processing many samples per
instruction), better memory access patterns to stay in
[cache](/reference/cache-memory/), and [multithreading](/reference/multithreaded-dsp/)
to spread stages across cores. Libraries like [VOLK](/reference/volk/) exist
precisely to supply hand-tuned, CPU-dispatched kernels for the common per-sample
operations so applications inherit the speed without writing assembly.

The honest caution is to keep the benchmark faithful to reality: a microbenchmark
that fits entirely in L1 cache can vastly overstate throughput compared to a real
pipeline that is memory-bound, so end-to-end throughput on real data is the number
that actually predicts whether the radio keeps up.

## Relevance to SDR

Meeting the sample-rate budget is the defining performance problem of
[real-time DSP](/reference/real-time-dsp/), and benchmarking is how you prove you
meet it with margin to spare for bursts and scheduling jitter. It is what tells you
whether a decoder will run on a Raspberry Pi or needs a desktop, and where to spend
effort if it does not.

**GopherTrunk** is a pure-Go application, so it benchmarks with Go's built-in
benchmarking and profiling tooling rather than relying on a
[VOLK](/reference/volk/)/GNU Radio stack, and its file-[replay](/reference/file-source-sink/)
path doubles as a natural throughput harness: replaying a capture flat-out and
measuring the real-time factor shows how much headroom the decode chain has above
the live sample rate. Because the decode chain normalizes to a fixed per-protocol
channel rate, the steady-state DSP cost is bounded regardless of capture rate,
which keeps the performance budget predictable. GT does not use a GPU or hand-tuned
SIMD kernel library; it leans on Go's compiler and clean per-sample loops, and
relates the heavier vectorization approaches to the wider ecosystem.

## Sources

[^bench]: [Benchmark (computing)](https://en.wikipedia.org/wiki/Benchmark_(computing)) — Wikipedia, on measuring and comparing the performance of software components.
[^prof]: [Profiling (computer programming)](https://en.wikipedia.org/wiki/Profiling_(computer_programming)) — Wikipedia, on attributing run time to code to locate hotspots worth optimizing.
