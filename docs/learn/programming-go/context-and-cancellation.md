---
slug: context-and-cancellation
title: Context & cancellation
description: "The context package — how one cancel signal or timeout propagates through every goroutine in a call tree to shut work down cleanly, the first argument everywhere in Go."
keywords: go context, context.Context, WithCancel, WithTimeout, ctx.Done, cancellation go, context first argument, graceful shutdown go
level: advanced
status: full
prereq:
  - channels
faq:
  - q: Why is ctx the first argument to so many Go functions?
    a: "By convention a Context is passed as the first parameter, named ctx, so a cancel signal or deadline flows explicitly down the whole call tree. Making it the first argument means every function that might block or do I/O can be told to stop, and readers can see at a glance that a call is cancellable."
  - q: What does ctx.Done() actually give me?
    a: "It returns a channel that is closed when the context is cancelled or its deadline passes. You select on it: when the receive succeeds (because the channel closed), it's time to stop work and return ctx.Err(). It's the standard way a goroutine learns it should shut down."
---

# Context & cancellation

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **`context.Context`** carries a cancellation signal and deadline down a call tree.
**`WithCancel`** and **`WithTimeout`** derive a child context plus a `cancel`
function; calling `cancel` (or hitting the timeout) closes **`ctx.Done()`**, a
channel every goroutine can **`select`** on to stop cleanly. By convention `ctx` is
the **first argument** of any cancellable function.
</div>

Long-running programs need a way to say "stop now" — on shutdown, a timeout, or a
user's Ctrl-C. This lesson covers Go's answer, `context`, which builds directly on
[channels](/learn/programming-go/channels/) and
[select](/learn/programming-go/select-and-sync/).

## Creating a cancellable context

You start from a root (`context.Background()`) and derive children that can be
cancelled:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()          // always call cancel to release resources

go scan(ctx)            // pass ctx down to the work
// ...later, to stop everything:
cancel()
```

`WithTimeout` and `WithDeadline` do the same but also cancel automatically after a
duration:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

## Reacting to cancellation

The heart of it is `ctx.Done()` — a channel that closes when the context is
cancelled. A worker `select`s on it alongside its normal work:

```go
func scan(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()          // Canceled or DeadlineExceeded
        case sample := <-samples:
            process(sample)
        }
    }
}
```

The moment `cancel()` runs (or the timeout fires), the `<-ctx.Done()` case wins and
the goroutine returns. One signal, cleanly propagated.

## One signal, a whole tree

Because each derived context is a child of its parent, cancelling the parent cancels
every descendant. A single Ctrl-C at the top can shut down every goroutine in a
GopherTrunk pipeline — the SDR reader, the demodulators, the decoder — without any of
them knowing about each other.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 150" role="img" aria-label="A root context cancelling and the signal propagating to three child goroutines below it." xmlns="http://www.w3.org/2000/svg">
  <rect x="200" y="15" width="120" height="30" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="260" y="35" text-anchor="middle" font-size="11" fill="currentColor">root ctx — cancel()</text>
  <rect x="30" y="105" width="120" height="30" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="90" y="125" text-anchor="middle" font-size="11" fill="currentColor">reader</text>
  <rect x="200" y="105" width="120" height="30" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="260" y="125" text-anchor="middle" font-size="11" fill="currentColor">demod</text>
  <rect x="370" y="105" width="120" height="30" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="430" y="125" text-anchor="middle" font-size="11" fill="currentColor">decoder</text>
  <line x1="245" y1="45" x2="90" y2="105" stroke="currentColor" stroke-width="1.5" marker-end="url(#ca)"/>
  <line x1="260" y1="45" x2="260" y2="105" stroke="currentColor" stroke-width="1.5" marker-end="url(#ca)"/>
  <line x1="275" y1="45" x2="430" y2="105" stroke="currentColor" stroke-width="1.5" marker-end="url(#ca)"/>
  <defs><marker id="ca" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Cancelling the root context closes Done() for every child, so one signal stops the whole tree.</figcaption>
</figure>

## Rules of the road

| Do | Don't |
|----|-------|
| Pass `ctx` as the **first** argument | Store a Context inside a struct |
| Always call the returned `cancel` (defer it) | Pass a `nil` context |
| Return `ctx.Err()` when Done fires | Ignore cancellation in a long loop |
| Use `WithTimeout` for network/I/O deadlines | Use context to pass optional arguments |

Follow these and cancellation becomes a habit rather than an afterthought — essential
for the concurrent shapes in the next lesson.

<div class="knowledge-check" data-quiz data-correct-msg="Right — Done() closes on cancel or timeout, and select picks it up." markdown="0">
  <p class="knowledge-check__q">Quick check: how does a goroutine learn its context was cancelled?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It polls ctx.Canceled every loop</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The channel from ctx.Done() closes, and a select on it fires</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The runtime kills the goroutine automatically</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **`Context`** carries a cancel signal and deadline down a call tree.
- **`WithCancel`**/**`WithTimeout`** return a child ctx and a **`cancel`** — always
  call cancel.
- **`ctx.Done()`** is a channel you **`select`** on; return **`ctx.Err()`** when it
  closes.
- Pass **`ctx` first**; cancelling a parent cancels every child.

Next up: the reusable shapes of concurrent code — worker pools, pipelines, and
fan-out/fan-in.
