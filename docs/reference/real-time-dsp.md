---
slug: real-time-dsp
title: Real-time DSP
entry_type: concept
category: sdr-programming
description: Real-time DSP is signal processing that must keep pace with an incoming sample stream, finishing each block before the next arrives, so an SDR meets the hard deadline its sample rate imposes.
keywords: real-time DSP, real time signal processing, streaming DSP, sample rate deadline, throughput, deadline, latency, keeping up, overrun, block processing, soft real time
aka: [real-time DSP, streaming DSP, real-time signal processing]
autolink: true
infobox:
  - { label: Type, value: Deadline-bound signal processing }
  - { label: Deadline, value: Set by the sample rate }
  - { label: Fails as, value: Overruns / dropped samples }
see_also: [overruns-underruns, dsp-latency, multithreaded-dsp, sample-buffer, ring-buffer, back-pressure]
cite_urls:
  - https://en.wikipedia.org/wiki/Real-time_computing
  - https://www.dspguide.com/ch28.htm
---

**Real-time DSP** is digital signal processing that must keep up with a continuous sample stream — finishing the work for each block of samples before the next block arrives — so the system never falls permanently behind the rate at which data is produced.[^wiki] For a software radio the deadline is set by the [sample rate](/reference/sample-rate/): at 2.4 MS/s the pipeline has, on average, well under a microsecond of compute budget per sample, and missing that budget for too long means dropped data, not merely a slow answer.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A timeline divided into equal sample-block windows; processing bars fit inside each window when keeping up, but one bar overruns its window boundary and the buffer level rises, marked as falling behind." xmlns="http://www.w3.org/2000/svg">
  <g font-size="11" fill="currentColor">
    <g stroke="currentColor" stroke-opacity="0.4"><line x1="10" y1="20" x2="10" y2="86"/><line x1="120" y1="20" x2="120" y2="86"/><line x1="230" y1="20" x2="230" y2="86"/><line x1="340" y1="20" x2="340" y2="86"/><line x1="450" y1="20" x2="450" y2="86"/></g>
    <text x="150" y="16" font-size="10">each window = one block's worth of samples</text>
    <rect x="14" y="40" width="88" height="20" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/>
    <rect x="124" y="40" width="96" height="20" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/>
    <rect x="234" y="40" width="150" height="20" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/>
    <text x="252" y="55" font-size="9">over deadline →</text>
    <path d="M340 66 L360 66 L360 78 L376 78" fill="none" stroke="currentColor" stroke-dasharray="3 2"/>
    <text x="300" y="104" font-size="10">buffer backs up, then overruns</text>
    <text x="14" y="104" font-size="10">keeping up</text>
  </g>
</svg>
<figcaption>As long as each block finishes inside its window the system holds real time; a block that overruns its window pushes the backlog up until the buffer overflows.</figcaption>
</figure>

## How it works

Real-time DSP is organized around **throughput first, then latency**. The unbreakable requirement is that the *average* processing rate meet or exceed the sample rate — if it doesn't, the backlog in the [sample buffer](/reference/sample-buffer/) grows without bound until it overflows into an [overrun](/reference/overruns-underruns/). A [ring buffer](/reference/ring-buffer/) between the source and the DSP absorbs short-term jitter: a block that runs a little long is fine as long as later blocks catch up and the buffer drains again. What is fatal is a *sustained* deficit.

Most SDR work is **soft** real time: an occasional missed deadline degrades quality (a lost decode) but is tolerable, unlike a hard real-time controller where a single miss is a failure. To hold the deadline, real-time DSP code avoids anything with unpredictable timing on the hot path:

- **No blocking I/O** in the processing loop — disk writes, network sends, and logging move to separate threads fed through queues.
- **No unbounded allocation** — buffers are pre-allocated and reused so memory management doesn't inject pauses (a particular concern under a garbage collector).
- **Bounded, predictable kernels** — fixed-size FFTs, filters, and loops whose cost per block is known, so the budget can actually be reasoned about.

## In practice

Two levers relieve a pipeline that can't keep up. The first is **doing less work**: decimate early so later stages run at a lower rate, choose the smallest sample rate the signal needs, and pick efficient algorithms. The second is **parallelism**: [multithreaded DSP](/reference/multithreaded-dsp/) splits stages across cores so a slow stage doesn't stall the whole chain, at the cost of added [latency](/reference/dsp-latency/) and synchronization. Deep buffers trade latency for robustness against jitter; shallow buffers give tight latency but leave no slack. The right balance depends on whether a human is listening live or a decoder is merely logging.

## Relevance to SDR

Every live receiver is a real-time DSP system: tune, filter, demodulate, and decode must all clear before the ADC's next batch lands. Offline replay from a file is *not* real-time — the file source will wait — which is precisely why file-based development is easier and reproducible, and why a bug that only appears live often points at timing or resource contention rather than the math.

**GopherTrunk** is a real, pure-Go SDR application built around this discipline. It streams IQ through buffers the decoders drain independently, sizes those buffers to ride out scheduling jitter, and exposes a dropped-sample counter so a real-time miss surfaces as a visible [overrun](/reference/overruns-underruns/) instead of a silent glitch. Being written in Go means GT also has to keep allocation off the hot sample path so garbage collection doesn't threaten the deadline. Notably, its decode chain is *rate-invariant* — it normalizes each protocol to a fixed channel rate regardless of capture rate — so the real-time budget is governed by that channel rate and the CPU, and the offline `.cfile` path can reproduce a live decode without any real-time deadline at all.

## Sources

[^wiki]: [Real-time computing](https://en.wikipedia.org/wiki/Real-time_computing) — Wikipedia, on hard vs soft real-time deadlines and deadline-bound processing.
