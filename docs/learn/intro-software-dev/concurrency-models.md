---
slug: concurrency-models
title: Concurrency & parallelism
description: Concurrency vs parallelism, OS threads and locks, async/await event loops, Go's goroutines and channels, the actor model, and how data races happen.
keywords: concurrency, parallelism, threads, async await, goroutines, channels, CSP, actor model, data race
level: intermediate
status: full
faq:
  - q: "What is the difference between concurrency and parallelism?"
    a: "**Concurrency** is *dealing with* many tasks at once — structuring a program so tasks can make progress independently, even on a single core by interleaving. **Parallelism** is *doing* many things at literally the same instant, which requires multiple cores. Concurrency is about structure; parallelism is about execution."
  - q: "What is a data race?"
    a: "A **data race** happens when two threads access the same memory at the same time and at least one is writing, with no synchronisation. The result is unpredictable — corrupted values or crashes that appear randomly. Locks, channels and the actor model are different strategies for avoiding them."
  - q: "How are goroutines different from OS threads?"
    a: "**Goroutines** are lightweight tasks managed by Go's runtime, not the operating system. They start with tiny stacks and the runtime multiplexes many of them onto a few OS threads, so you can run hundreds of thousands cheaply. They communicate over **channels** rather than sharing memory directly."
---
# Concurrency & parallelism

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Concurrency vs parallelism** — structuring tasks vs running them simultaneously. **Models** — threads+locks, async/await, channels, actors. **Data races** — the bug every model tries to prevent.
</div>

Modern programs rarely do one thing at a time. They wait on a network, read from a
device, and crunch numbers — ideally without each task blocking the others, and
ideally using all the CPU cores you paid for. The trouble is that doing several
things at once is where some of the nastiest, hardest-to-reproduce bugs live.
Languages offer different **concurrency models** to make this tractable, and
choosing one is a major part of a language's character. We start by untangling two
words that get used interchangeably but are not the same.

## Concurrency vs parallelism

- **Concurrency** is about *structure*: organising a program so that multiple tasks
  can be in progress and make independent progress. Even on a single CPU core, a
  concurrent program can interleave tasks — work on task A, pause it while it waits
  for I/O, switch to task B. It is about *dealing with* many things at once.
- **Parallelism** is about *execution*: literally running multiple computations at
  the same instant, which requires multiple cores. It is about *doing* many things
  at once.

The two are related but independent. You can have concurrency without parallelism
(one core juggling tasks), and the goal of a good concurrency model is to make it
easy to express tasks that the runtime can *also* run in parallel when cores are
available. As Rob Pike put it: concurrency is a way to structure things; if it
works, parallelism may be a free bonus.

## Threads and locks

The oldest model is **OS threads**: the operating system gives your process
multiple threads of execution that share the same memory and can run on different
cores. It is powerful and parallel by nature, but sharing memory is exactly where
the danger lies.

When two threads touch the same data and at least one writes, you have a
**data race**, and the result is undefined — a value half-updated, a counter that
loses increments, a crash that appears once a week. The traditional fix is a
**lock** (mutex): only one thread may hold the lock and touch the shared data at a
time.

```text
thread A:  lock → read/modify shared counter → unlock
thread B:  ........ wait for lock ........ → lock → ... → unlock
```

Locks work but introduce their own hazards: **deadlock** (two threads each waiting
on a lock the other holds), forgotten locks, and performance loss from contention.
"Shared memory plus locks" is correct in principle and treacherous in practice,
which motivated every model that follows.

## Async/await event loops

For workloads dominated by **waiting** — network requests, disk reads — you often
do not need parallelism at all; you need to not block while waiting.
**Async/await** runs many tasks concurrently on a single thread using an **event
loop**. When a task hits an `await` on something slow, it yields control back to
the loop, which runs other ready tasks; when the slow thing completes, the task
resumes.

```python
async def fetch(url):
    data = await http_get(url)   # yields to the loop while waiting
    return parse(data)
```

JavaScript (Node.js), Python (`asyncio`), C# and Rust all offer this. It is
excellent for **I/O-bound** work — thousands of concurrent connections on one
thread — but does little for **CPU-bound** work, since a long computation still
hogs the single thread and blocks everything else.

## Goroutines and channels (CSP)

**Go** popularised a model based on **Communicating Sequential Processes (CSP)**.
Instead of sharing memory and guarding it with locks, you run lightweight
**goroutines** and have them **communicate over channels**. The slogan: *"Do not
communicate by sharing memory; share memory by communicating."*

- A **goroutine** is a tiny task the Go runtime schedules onto OS threads; they cost
  almost nothing, so you can have hundreds of thousands.
- A **channel** is a typed pipe; one goroutine sends values, another receives. The
  channel handles synchronisation, so there is no explicit lock and no data race on
  the data that flows through it.

```go
samples := make(chan float64)      // a typed channel
go produce(samples)                // one goroutine sends
go consume(samples)                // another receives
```

This makes concurrent *pipelines* natural to express, which is why Go is so common
in networked and streaming systems.

## The actor model

The **actor model** (made famous by **Erlang**, and used by Elixir and Akka) goes
further: the unit of concurrency is an **actor** — an isolated process with its own
private state that **shares nothing**. Actors communicate only by sending each
other **messages**, processed one at a time from a mailbox.

Because no state is shared, **data races are structurally impossible**, and
because actors are isolated, one can crash and be restarted without taking down
the others. Erlang built telecom systems with famous uptime on exactly this "let
it crash and supervise" philosophy. The trade-off is message-passing overhead and
a different way of thinking about program structure.

<div class="knowledge-check" data-quiz data-correct-msg="Right — concurrency is structuring tasks to progress independently; parallelism is running them at the same instant on multiple cores." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the difference between concurrency and parallelism?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">They mean the same thing — both require multiple CPU cores</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Concurrency is structuring tasks to progress independently; parallelism is running them simultaneously on multiple cores</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Concurrency needs multiple cores; parallelism works on one core</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Choosing a model, and the costs you can't escape

No concurrency model is universally best; each suits a different workload, and the
right question is what your program spends its time doing.

- **I/O-bound** work — waiting on networks, disks, devices — favours async/await or
  lightweight tasks like goroutines. You have many things waiting and few things
  computing, so cheap concurrency matters more than raw parallelism.
- **CPU-bound** work — heavy number-crunching like DSP — favours real parallelism
  across threads or cores, because the bottleneck is computation, not waiting.
- **Fault-tolerant, long-running systems** favour the actor model, where isolation
  lets parts crash and restart independently.

Whatever you choose, some costs are inherent. Splitting work across tasks adds
**coordination overhead** — passing messages, acquiring locks, scheduling. Beyond
a point, adding more parallelism yields diminishing returns because the sequential
and coordination portions dominate (this is **Amdahl's law** in spirit: a program's
speed-up is capped by the fraction that cannot be parallelised). And concurrency
bugs — races, deadlocks, subtle ordering issues — are among the hardest to
reproduce and fix, because they depend on timing that changes from run to run. The
appeal of channels and actors is precisely that they make whole categories of these
bugs impossible by design, trading a little overhead for a lot of certainty.

## A streaming radio pipeline

Concurrency is not academic for radio software — it is the whole architecture. A
live capture from a software-defined radio naturally splits into stages that must
run *at the same time*, each feeding the next:

- a **capture thread** pulling raw [IQ samples](/learn/rf-sdr/iq-data/) off the
  device as fast as they arrive, never blocking;
- a **DSP thread** filtering and demodulating those samples;
- a **decode thread** turning the demodulated signal into packets, audio or text.

These form a **pipeline**, and channels (or queues) are the perfect glue: the
capture stage sends buffers down a channel, the DSP stage receives, processes, and
sends results onward to the decode stage. Each stage runs concurrently, the
runtime can spread them across cores in parallel, and the channels handle
hand-off without a single shared-memory data race. If one stage falls behind, a
bounded channel provides natural back-pressure. We build out this exact pattern in
[concurrency and pipelines](/learn/intro-software-dev/concurrency-and-pipelines/).

## Recap

- **Concurrency vs parallelism** — concurrency structures independent tasks;
  parallelism runs them at the same instant on multiple cores.
- **Threads and locks** — powerful and parallel, but shared memory invites data
  races and deadlocks.
- **Async/await** — single-threaded event loops excel at I/O-bound work, not
  CPU-bound work.
- **Goroutines and channels** — Go's CSP model communicates instead of sharing
  memory, making pipelines natural.
- **Actor model** — isolated, share-nothing actors pass messages, so data races are
  impossible by construction.
- **Data races are the enemy** — every model is, at heart, a strategy for avoiding
  unsynchronised shared writes.

Next up: how a language's design shapes its attack surface — [language-level
security](/learn/intro-software-dev/language-security/).
