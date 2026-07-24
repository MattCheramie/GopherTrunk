---
slug: concurrency-patterns
title: Concurrency patterns
description: "Worker pools, pipelines, and fan-out/fan-in — the reusable shapes for structuring concurrent Go work, the way a real streaming sample pipeline is built."
keywords: go concurrency patterns, worker pool go, pipeline pattern go, fan-out fan-in, go channels pattern, goroutine pool, streaming pipeline go
level: advanced
status: full
prereq:
  - context-and-cancellation
faq:
  - q: What is a worker pool?
    a: "A fixed number of goroutines all reading jobs from one channel and writing results to another. It caps concurrency at a chosen size so you don't spawn an unbounded number of goroutines, and it spreads work evenly because whichever worker is free grabs the next job."
  - q: What is fan-out/fan-in?
    a: "Fan-out starts several goroutines reading from the same input channel to parallelize a slow stage. Fan-in merges their several output channels back into one. Together they let one busy stage of a pipeline run on many cores while the rest stays a single stream."
---

# Concurrency patterns

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Three shapes cover most concurrent Go. A **pipeline** chains stages, each a goroutine
reading one channel and writing the next. A **worker pool** runs a fixed number of
goroutines off one job channel to cap concurrency. **Fan-out/fan-in** parallelizes a
slow stage across many goroutines, then merges the results. All three are just
[goroutines](/learn/programming-go/goroutines/) and
[channels](/learn/programming-go/channels/) arranged deliberately.
</div>

Once you have goroutines, channels, and context, real programs assemble them into a
few recurring structures. This lesson shows the shapes GopherTrunk's sample pipeline
is built from.

## The pipeline

A pipeline is a chain of stages connected by channels. Each stage receives on its
input, does one job, and sends on its output — then closes it:

```go
func demod(in <-chan complex64) <-chan byte {
    out := make(chan byte)
    go func() {
        defer close(out)
        for s := range in {
            out <- symbol(s)
        }
    }()
    return out
}

symbols := demod(samples)   // stages compose: decode(demod(samples))
```

Each stage runs concurrently, so raw samples, demodulated symbols, and decoded frames
all flow at once — exactly how an SDR engine keeps up with a live radio.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 90" role="img" aria-label="Three pipeline stages connected by channels: samples flow into demod, then decode, then frames out." xmlns="http://www.w3.org/2000/svg">
  <rect x="10" y="30" width="110" height="34" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="65" y="51" text-anchor="middle" font-size="11" fill="currentColor">source</text>
  <rect x="175" y="30" width="110" height="34" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="230" y="51" text-anchor="middle" font-size="11" fill="currentColor">demod</text>
  <rect x="340" y="30" width="110" height="34" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="395" y="51" text-anchor="middle" font-size="11" fill="currentColor">decode</text>
  <line x1="120" y1="47" x2="175" y2="47" stroke="currentColor" stroke-width="1.5" marker-end="url(#pp)"/>
  <line x1="285" y1="47" x2="340" y2="47" stroke="currentColor" stroke-width="1.5" marker-end="url(#pp)"/>
  <line x1="450" y1="47" x2="505" y2="47" stroke="currentColor" stroke-width="1.5" marker-end="url(#pp)"/>
  <defs><marker id="pp" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A pipeline: each stage is a goroutine that reads one channel and writes the next; all stages run at once.</figcaption>
</figure>

## The worker pool

When one stage is the bottleneck, run several copies of it — but a *fixed* number, so
concurrency stays bounded:

```go
jobs := make(chan Frame)
results := make(chan Decoded)

for i := 0; i < runtime.NumCPU(); i++ {   // N workers
    go func() {
        for f := range jobs {
            results <- decode(f)          // whichever worker is free takes the job
        }
    }()
}
```

The pool self-balances: fast jobs free a worker to grab the next one immediately. Cap
the count at something sensible (often `runtime.NumCPU()`) rather than spawning one
goroutine per job.

## Fan-out and fan-in

Fan-out is starting several workers on the *same* input channel; fan-in merges their
outputs back into one, typically with a `sync.WaitGroup`:

```go
func fanIn(cs ...<-chan Decoded) <-chan Decoded {
    out := make(chan Decoded)
    var wg sync.WaitGroup
    for _, c := range cs {
        wg.Add(1)
        go func(c <-chan Decoded) {
            defer wg.Done()
            for v := range c {
                out <- v
            }
        }(c)
    }
    go func() { wg.Wait(); close(out) }()   // close once all inputs drain
    return out
}
```

| Pattern | Shape | Use it when |
|---------|-------|-------------|
| Pipeline | stage → channel → stage | Work is a series of transforms |
| Worker pool | N goroutines, one job channel | You must cap concurrency |
| Fan-out/fan-in | split to many, merge to one | One stage is the bottleneck |

Thread a `context` (previous lesson) through every stage so a single cancel tears the
whole structure down cleanly.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a pool is a fixed set of goroutines sharing one job channel." markdown="0">
  <p class="knowledge-check__q">Quick check: what makes a worker pool different from spawning a goroutine per job?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">It uses a fixed number of goroutines, bounding concurrency</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It runs jobs strictly in order</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It avoids using channels entirely</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **pipeline** chains goroutine stages with channels; each closes its output when
  done.
- A **worker pool** runs a fixed number of goroutines off one job channel to cap
  concurrency.
- **Fan-out/fan-in** parallelizes a slow stage, then merges with a `WaitGroup`.
- Thread a **`context`** through every stage for clean shutdown.

Next up: how Go organizes code into packages and modules.
