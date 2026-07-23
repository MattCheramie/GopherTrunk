---
slug: channels
title: Channels
description: Channels are typed pipes that let goroutines communicate safely — sending, receiving, buffering, and closing them — Go's share-memory-by-communicating model of concurrency.
keywords: go channels, channel send receive, buffered channel, unbuffered channel, close channel, go concurrency, share memory by communicating
level: intermediate
status: full
prereq:
  - goroutines
faq:
  - q: What is the difference between a buffered and an unbuffered channel?
    a: An unbuffered channel has no storage — a send blocks until another goroutine is ready to receive, so the two rendezvous. A buffered channel (`make(chan T, n)`) holds up to n values, so a send only blocks when the buffer is full. Unbuffered channels synchronize two goroutines; buffered channels let a fast producer run ahead of a slower consumer up to the buffer size.
  - q: Do I always need channels for concurrency?
    a: No. Channels are the idiomatic way to pass data between goroutines, but sometimes plain shared state guarded by a mutex from the sync package is simpler — for example, a counter many goroutines increment. The next lesson covers when to reach for sync instead. The guideline is to use channels to move data and ownership, and mutexes to protect small pieces of shared state.
---

# Channels

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **channel** is a typed pipe that connects goroutines: one **sends** with `ch <- v`
and another **receives** with `v := <-ch`. An **unbuffered** channel makes sender
and receiver rendezvous; a **buffered** one holds a few values. Channels let
goroutines share data **without shared-memory bugs** — the heart of Go's concurrency
model.
</div>

Goroutines run independently; channels are how they talk. This lesson covers Go's
central concurrency tool.

## Creating and using a channel

```go
ch := make(chan int)   // a channel that carries ints

go func() {
    ch <- 42           // send 42 into the channel
}()

v := <-ch              // receive; v is 42
```

The arrow points the way the data flows: `ch <- v` sends *into* the channel,
`<-ch` receives *out of* it. Because this channel is **unbuffered**, the send blocks
until the receive happens — the two goroutines meet at that moment. That built-in
synchronization is exactly why you don't need locks to hand a value from one
goroutine to another.

## Buffered channels

Give `make` a size and the channel can hold values without an immediate receiver:

```go
results := make(chan []byte, 8)   // holds up to 8 before a send blocks
```

A buffered channel lets a fast producer stay ahead of a slower consumer — useful in
a signal pipeline where samples arrive in bursts. When the buffer is full, the next
send blocks until room opens up, which naturally applies backpressure.

## Ranging and closing

A sender can **close** a channel to signal "no more values," and a receiver can
`range` over it until it's drained:

```go
go func() {
    for _, frame := range frames {
        out <- frame
    }
    close(out)          // no more frames coming
}()

for frame := range out {   // loops until out is closed and empty
    process(frame)
}
```

Closing is a broadcast that the stream has ended — only the sender should close, and
only once.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 100" role="img" aria-label="Two goroutines connected by a channel: a producer sends values in one end and a consumer receives them from the other." xmlns="http://www.w3.org/2000/svg">
  <rect x="10" y="35" width="120" height="34" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="70" y="57" text-anchor="middle" font-size="12" fill="currentColor">producer</text>
  <rect x="180" y="40" width="160" height="24" rx="12" fill="none" stroke="currentColor" stroke-width="1.5" stroke-dasharray="5 3"/>
  <text x="260" y="56" text-anchor="middle" font-size="11" fill="currentColor">channel</text>
  <line x1="130" y1="52" x2="180" y2="52" stroke="currentColor" stroke-width="1.5" marker-end="url(#a2)"/>
  <line x1="340" y1="52" x2="390" y2="52" stroke="currentColor" stroke-width="1.5" marker-end="url(#a2)"/>
  <rect x="390" y="35" width="120" height="34" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="450" y="57" text-anchor="middle" font-size="12" fill="currentColor">consumer</text>
  <defs><marker id="a2" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A channel decouples producer from consumer — they share data through the pipe, not through shared variables.</figcaption>
</figure>

## Why this is safer

The classic concurrency bug is a **data race**: two goroutines touching the same
variable at once, one of them writing. Channels sidestep it by passing a value's
*ownership* along the pipe — at any moment, only one goroutine holds it. That's the
meaning of Go's motto, *share memory by communicating*. When channels aren't the
right fit, Go's `sync` package offers locks — the subject of the next lesson.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an unbuffered send waits for a receiver, synchronizing the two goroutines." markdown="0">
  <p class="knowledge-check__q">Quick check: on an <em>unbuffered</em> channel, what happens when you send a value?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It's stored until someone reads it later</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The send blocks until another goroutine receives it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It's dropped if no one is listening</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **channel** carries typed values between goroutines: `ch <- v` sends, `<-ch`
  receives.
- **Unbuffered** channels synchronize sender and receiver; **buffered** ones hold a
  few values and apply backpressure.
- **`close`** signals the end of a stream; `range` receives until it's drained.
- Channels prevent **data races** by passing ownership instead of sharing variables.

Next up: coordinating multiple channels with select, and when to reach for locks instead.
