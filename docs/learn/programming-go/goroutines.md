---
slug: goroutines
title: Goroutines
description: Goroutines are Go's lightweight concurrency primitive — what they are, how they differ from operating-system threads, and why a program like GopherTrunk can run thousands of them at once.
keywords: goroutines, go concurrency, go routine, lightweight threads, go scheduler, concurrent programming, go keyword
level: intermediate
status: full
prereq:
  - interfaces
faq:
  - q: What is the difference between a goroutine and a thread?
    a: A goroutine is a function managed by Go's own runtime scheduler, not directly by the operating system. Goroutines start with a tiny stack (a few kilobytes) that grows as needed, so you can run tens of thousands of them; OS threads are heavier, so you typically have far fewer. Go multiplexes many goroutines onto a small pool of real threads for you.
  - q: Is concurrency the same as parallelism?
    a: No. Concurrency is structuring a program as independently running tasks; parallelism is those tasks literally executing at the same instant on multiple CPU cores. Goroutines give you concurrency; whether they run in parallel depends on how many cores are available. Go handles the mapping automatically.
---

# Goroutines

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **goroutine** is a function launched with the **`go`** keyword that runs
concurrently with the rest of your program. Goroutines are **lightweight** — managed
by Go's runtime, starting at a few kilobytes — so a program can run thousands. This
is Go's headline feature, and it's why a real-time engine like GopherTrunk can juggle
many signal streams at once.
</div>

Everything so far has run top to bottom, one line at a time. This lesson introduces
concurrency: doing several things at once, the Go way.

## Launching a goroutine

Put `go` in front of a function call and it runs concurrently — the calling code
does not wait for it:

```go
go processSamples(stream)   // runs alongside everything below
fmt.Println("kicked off processing")
```

`processSamples` now runs independently while `main` carries on. That one keyword is
the entire syntax for starting concurrent work.

## Why goroutines are cheap

A goroutine is **not** an operating-system thread. Go's runtime keeps a small pool
of real threads and schedules many goroutines onto them, each starting with a tiny
stack that grows only if needed.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 150" role="img" aria-label="Many goroutines multiplexed by the Go scheduler onto a few operating-system threads." xmlns="http://www.w3.org/2000/svg">
  <text x="60" y="20" text-anchor="middle" font-size="12" fill="currentColor">goroutines</text>
  <circle cx="20" cy="45" r="9" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <circle cx="45" cy="45" r="9" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <circle cx="70" cy="45" r="9" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <circle cx="95" cy="45" r="9" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <circle cx="20" cy="70" r="9" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <circle cx="45" cy="70" r="9" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <rect x="200" y="45" width="120" height="34" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="260" y="67" text-anchor="middle" font-size="12" fill="currentColor">Go scheduler</text>
  <line x1="110" y1="57" x2="200" y2="60" stroke="currentColor" stroke-opacity="0.5" stroke-width="1"/>
  <line x1="320" y1="62" x2="400" y2="45" stroke="currentColor" stroke-opacity="0.5" stroke-width="1"/>
  <line x1="320" y1="62" x2="400" y2="80" stroke="currentColor" stroke-opacity="0.5" stroke-width="1"/>
  <rect x="400" y="35" width="100" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="450" y="51" text-anchor="middle" font-size="11" fill="currentColor">OS thread</text>
  <rect x="400" y="70" width="100" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="450" y="86" text-anchor="middle" font-size="11" fill="currentColor">OS thread</text>
</svg>
<figcaption>The runtime multiplexes many cheap goroutines onto a few real threads — so thousands of concurrent tasks cost far less than thousands of threads.</figcaption>
</figure>

Because they are so cheap, the idiomatic Go move is to launch a goroutine per unit
of concurrent work rather than carefully pooling a scarce resource. GopherTrunk uses
this to run capture, decoding, and per-channel processing side by side.

## The catch: goroutines need coordination

A goroutine runs on its own, which raises two questions this unit answers next:

- **How do goroutines share results safely?** Two goroutines touching the same
  variable at once is a *data race* — a real bug. The answer is
  [channels](/learn/programming-go/channels/), Go's built-in way to pass data
  between goroutines.
- **How does `main` wait for them?** If `main` returns, the program exits and any
  running goroutines are cut off. You need a way to coordinate — covered in
  [select & synchronization](/learn/programming-go/select-and-sync/).

Go's motto captures the philosophy: *don't communicate by sharing memory; share
memory by communicating.* That's the channel model, and it's next.

<div class="knowledge-check" data-quiz data-correct-msg="Right — goroutines are managed by Go's runtime, far lighter than OS threads." markdown="0">
  <p class="knowledge-check__q">Quick check: why can a Go program run thousands of goroutines at once?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Each goroutine is a dedicated OS thread</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">They're lightweight and multiplexed by the runtime onto a few threads</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Only one runs at a time, so they're free</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **goroutine** is launched with the **`go`** keyword and runs concurrently.
- Goroutines are **lightweight**, scheduled by Go's runtime onto a few OS threads.
- You can run **thousands** cheaply — the idiomatic way to structure concurrent work.
- They need **coordination**: channels to share data, and synchronization to wait.

Next up: channels, the safe way for goroutines to talk to each other.
