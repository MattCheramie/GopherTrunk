---
slug: overruns-underruns
title: Overruns & underruns
entry_type: concept
category: sdr-programming
description: An overrun is samples dropped when a receiver's consumer can't keep up with the ADC, and an underrun is a starved transmitter or sink; both are the timing faults an SDR program must detect and survive.
keywords: overrun, underrun, overflow, underflow, dropped samples, sample drops, O U, buffer overflow, real-time, sample rate keep up, flow control
aka: [overrun, underrun, overflow, underflow, "O/U"]
autolink: true
infobox:
  - { label: Type, value: Real-time timing fault }
  - { label: Overrun, value: Consumer too slow, samples dropped }
  - { label: Underrun, value: Producer starved, gap emitted }
see_also: [sample-buffer, real-time-dsp, back-pressure, ring-buffer, dsp-latency]
cite_urls:
  - https://en.wikipedia.org/wiki/Buffer_underrun
  - https://files.ettus.com/manual/page_general.html
---

**An overrun** (overflow) is what happens when an SDR delivers samples faster than the program can consume them and the buffer fills, forcing samples to be discarded; **an underrun** (underflow) is the mirror image — a consumer that needs samples finds the buffer empty and is left with a gap.[^wiki] Both are failures to meet a hard [real-time](/reference/real-time-dsp/) deadline set by the sample rate, and handling them is a defining concern of writing SDR software rather than an edge case.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="Two timelines. Top: a fast producer fills a buffer past its capacity, marked O for overrun with dropped samples. Bottom: a slow producer leaves a buffer empty when the consumer reads, marked U for underrun with a gap." xmlns="http://www.w3.org/2000/svg">
  <g font-size="12" fill="currentColor">
    <text x="0" y="14">Overrun — consumer too slow</text>
    <rect x="0" y="24" width="300" height="26" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
    <rect x="0" y="24" width="300" height="26" fill="currentColor" fill-opacity="0.22"/>
    <path d="M300 24 h40 v26 h-40" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.6"/>
    <text x="346" y="41" font-weight="bold">O</text>
    <text x="300" y="66" font-size="10">dropped samples</text>
    <text x="0" y="112">Underrun — producer too slow</text>
    <rect x="0" y="122" width="300" height="26" fill="none" stroke="currentColor" stroke-opacity="0.5"/>
    <rect x="0" y="122" width="150" height="26" fill="currentColor" fill-opacity="0.22"/>
    <rect x="150" y="122" width="150" height="26" fill="none" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.6"/>
    <text x="205" y="139" font-size="10">empty — gap</text>
    <text x="316" y="139" font-weight="bold">U</text>
  </g>
</svg>
<figcaption>Overrun: the buffer overflows and excess samples are dropped. Underrun: the buffer runs dry and a gap is read where data should be.</figcaption>
</figure>

## How it works

A streaming radio runs on a fixed clock: at 2.4 MS/s the ADC produces 2.4 million IQ pairs every second whether or not anyone is ready for them. Those samples land in a [sample buffer](/reference/sample-buffer/) — usually a [ring buffer](/reference/ring-buffer/) — that the DSP drains. The buffer papers over short-term jitter, but its capacity is finite. If the average consumption rate ever dips below the average production rate, the level trends upward until it hits the ceiling. At that point something must give, and the only options are to drop new samples, overwrite old ones, or stall the producer.

- **Overrun (receive side).** The demodulator, disk writer, or network sink can't keep up; the buffer is full when fresh samples arrive, so those samples are lost. Because the ADC can't be told to pause, a dropped sample is gone for good, and the discontinuity looks to downstream DSP like an instantaneous jump — corrupting the timing loop, breaking symbol sync, and dropping decodes.
- **Underrun (transmit or output side).** A sink that must emit on a clock — a transmitter feeding a DAC, or an audio device — finds nothing buffered and emits silence or repeats, producing an audible glitch or an RF gap.

By long convention (Ettus/UHD, GNU Radio, PortAudio) these events are surfaced as a single character on stderr — **`O`** for an overrun, **`U`** for an underrun — a terse signal that the pipeline is losing the real-time race.[^uhd]

## In practice

The root cause is almost never the buffer; it is that some consumer stage is, on average, too slow. Enlarging the ring only buys time before the same overrun recurs. Durable fixes attack the throughput deficit: move blocking work (disk, network, logging) off the hot path onto its own thread, lower the sample rate to what the decode actually needs, use more efficient kernels, or apply [back-pressure](/reference/back-pressure/) so a slow stage throttles the source instead of silently dropping. Crucially, an SDR program should **count and report** drops rather than hide them: a decode that quietly degrades under load is far harder to diagnose than one that prints an overrun counter.

## Relevance to SDR

Every live SDR receiver faces overruns; they are the price of a producer you cannot pause. Real deployments hit them when a CPU is shared, a USB bus is contended, or a demod momentarily spikes. Offline replay from a file has no such deadline — the file source can wait — which is exactly why file-based testing is reproducible where live capture is not.

**GopherTrunk** treats drops as first-class. Its drivers maintain an overrun/dropped counter (the airspy reaper, for instance, records IQ chunks it had to discard) so a live overrun is observable rather than a silent decode failure. Because GT's decode chain is [rate-invariant](/reference/sample-rate/) — it normalizes to a per-protocol channel rate regardless of capture rate — the practical defense is keeping the consumer fast enough to drain the ring, and preferring offline `.cfile` replay when a problem needs to be reproduced deterministically.

## Sources

[^wiki]: [Buffer underrun](https://en.wikipedia.org/wiki/Buffer_underrun) — Wikipedia, on buffer underflow and overflow, dropped data, and real-time streaming.
[^uhd]: [UHD general notes — overflow/underflow (O/U)](https://files.ettus.com/manual/page_general.html) — Ettus Research, on the `O` and `U` indicators for host-side overruns and underruns.
