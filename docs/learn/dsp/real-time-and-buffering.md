---
slug: real-time-and-buffering
title: Real-time processing & buffering
description: Block processing, ring buffers, latency, and keeping up with the sample stream — the engineering that makes DSP run live instead of offline.
keywords: real-time dsp, block processing, ring buffer, circular buffer, latency vs throughput, sample rate keep up, overrun, buffering
level: advanced
status: full
prereq:
  - decimation-and-resampling
faq:
  - q: Why is DSP done in blocks instead of one sample at a time?
    a: "Processing a whole block of samples at once amortizes per-call overhead, keeps data in cache, and lets algorithms like the FFT work on a batch, so it is far more efficient than handling single samples. The tradeoff is latency: the system must wait to collect a full block before it can start, so bigger blocks mean higher throughput but longer delay before results appear."
  - q: What is a ring buffer and why is it used in real-time DSP?
    a: "A ring buffer is a fixed-size array treated as if its ends were joined, with a write pointer where new samples arrive and a read pointer where the processor consumes them. It lets a fast producer and a slower consumer run at their own paces without copying or reallocating memory, absorbing short bursts. If the producer laps the consumer the buffer overruns and samples are lost — the classic real-time failure."
---

# Real-time processing & buffering

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Live DSP must consume samples **as fast as they arrive** — forever. It does this in
**blocks** (efficient, but each block adds **latency**), passed between producer and
consumer through a **ring buffer** that absorbs timing jitter. The governing rule:
**average throughput must meet or beat the sample rate**, or the buffer **overruns** and
samples are lost. This is the engineering that turns the algorithms of this module into a
scanner that keeps up.
</div>

Every prior lesson assumed the samples were simply *there*. This one is about the
relentless clock behind them: an SDR delivers millions of samples per second, every
second, and the pipeline must never fall behind. It builds on
[decimation](/learn/dsp/decimation-and-resampling/) — the first tool for coping with the
firehose.

## The firehose problem

A capture at 2.4 MS/s of [complex I/Q](/learn/dsp/complex-signals-and-iq/) is 2.4 million
samples arriving *every second*, and they never stop. Unlike an offline file you can
process at leisure, a live stream imposes a hard deadline: on average you must finish each
second's work within that second. The single most important defence is to
[decimate](/learn/dsp/decimation-and-resampling/) to a narrow channel rate as early as
possible, shrinking the stream before the expensive stages ever see it.

## Block processing

You rarely process one sample at a time. Instead samples are gathered into **blocks** (a
few hundred to a few thousand) and handled together. Blocks amortize function-call and
loop overhead, keep data in cache, and suit batch algorithms like the
[FFT](/learn/dsp/the-fft/). The cost is **latency**: the system must wait for a block to
fill before it can start, so results always trail real time by at least one block.

```text
bigger blocks  -> higher throughput, higher latency
smaller blocks -> lower latency, more per-block overhead
```

Choosing block size is choosing a point on that curve — small enough to feel responsive,
large enough to stay efficient.

## Ring buffers: decoupling producer and consumer

The SDR (producer) and the DSP (consumer) don't run in perfect lockstep — the OS
schedules them unevenly, and processing time varies block to block. A **ring buffer** (a
circular buffer) sits between them to absorb that jitter. It's a fixed array with a
**write pointer** where new samples land and a **read pointer** where they're consumed;
both wrap around the end back to the start.

<figure class="figure" markdown="0">
<svg viewBox="0 0 300 170" role="img" aria-label="A ring buffer drawn as a circle of cells with a write pointer where a producer adds samples and a read pointer where a consumer removes them, the arc between them marked as buffered samples." xmlns="http://www.w3.org/2000/svg">
  <circle cx="150" cy="85" r="60" fill="none" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-opacity="0.3">
    <line x1="150" y1="25" x2="150" y2="35"/><line x1="210" y1="85" x2="200" y2="85"/>
    <line x1="150" y1="145" x2="150" y2="135"/><line x1="90" y1="85" x2="100" y2="85"/>
  </g>
  <path d="M150 25 A 60 60 0 0 1 210 85" fill="none" stroke="currentColor" stroke-width="4" stroke-opacity="0.5"/>
  <line x1="150" y1="85" x2="150" y2="25" stroke="currentColor" stroke-width="1.5"/>
  <text x="150" y="18" text-anchor="middle" font-size="9" fill="currentColor">write (producer)</text>
  <line x1="150" y1="85" x2="210" y2="85" stroke="currentColor" stroke-width="1.5"/>
  <text x="216" y="88" font-size="9" fill="currentColor">read</text>
  <text x="185" y="45" font-size="8" fill="currentColor">buffered</text>
</svg>
<figcaption>A ring buffer: the producer writes ahead, the consumer reads behind, and the arc between them is the cushion that absorbs timing jitter.</figcaption>
</figure>

The gap between the pointers is slack that soaks up short bursts. In GopherTrunk this
decoupling is often expressed with **Go channels** between concurrent stages — the same
producer/consumer pattern with the buffer built in.

## Overruns: the real-time failure

The buffer has a limit. If the consumer is persistently too slow, the write pointer
catches the read pointer — an **overrun** — and incoming samples are dropped. Dropped
samples are gaps in the signal, and gaps break [symbol recovery](/learn/dsp/clock-and-symbol-recovery/)
and [framing](/learn/dsp/error-correction-and-framing/) downstream. A buffer can hide a
*momentary* slowness, but it cannot fix a consumer whose **average** speed is below the
sample rate — that only ends one way. The fixes are all about average throughput: decimate
early, keep the hot loops tight, and mind the number format, which the next lesson takes up.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a ring buffer absorbs momentary jitter, but average throughput must still meet the sample rate." markdown="0">
  <p class="knowledge-check__q">Quick check: a ring buffer prevents dropped samples only if…</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">it is made large enough, no matter how slow the consumer is</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">the consumer's average throughput meets or beats the sample rate</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">the block size is set to exactly one sample</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Live DSP faces a hard deadline: keep up with a never-ending **sample stream**.
- **Block processing** is efficient but adds **latency** — a throughput-vs-delay tradeoff.
- A **ring buffer** decouples producer and consumer, absorbing timing **jitter**.
- Buffers hide momentary slowness only; sustained under-speed causes **overruns** and lost samples.

Next up: the walkthrough tying every stage together — DSP in GopherTrunk.
