---
slug: select-and-sync
title: select & synchronization
description: Coordinate multiple channels with Go's select statement, and use the sync package — mutexes, WaitGroups, and context — for the times channels aren't the right tool.
keywords: go select, select statement, sync package, mutex, waitgroup, context, goroutine coordination, data race, go concurrency patterns
level: advanced
status: full
prereq:
  - channels
faq:
  - q: What does the select statement do?
    a: "`select` waits on several channel operations at once and proceeds with whichever is ready first. It's how a goroutine can, say, read from a data channel but also watch a cancellation channel — whichever fires first wins. With a `default` case, select can also try channels without blocking at all."
  - q: What is a WaitGroup for?
    a: A `sync.WaitGroup` lets one goroutine wait for a group of others to finish. You call Add to count the goroutines, each one calls Done as it completes, and Wait blocks until the count reaches zero. It's the standard way to make main wait for background work before the program exits.
---

# select & synchronization

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**`select`** waits on several channels at once and acts on whichever is ready —
the key to handling multiple streams and cancellation. When channels aren't the
right fit, the **`sync`** package provides a **`Mutex`** to guard shared state and a
**`WaitGroup`** to wait for goroutines to finish, and **`context`** carries
cancellation across a whole call tree.
</div>

Real concurrent programs juggle several channels and need a clean way to stop. This
lesson covers `select` and the coordination tools that round out Go's concurrency
model.

## select: waiting on many channels

`select` is like a `switch`, but its cases are channel operations. It blocks until
one is ready, then runs that case:

```go
select {
case frame := <-frames:
    process(frame)          // data arrived
case <-quit:
    return                  // asked to stop
}
```

This pattern — "handle data, but also watch for a stop signal" — is everywhere in
long-running services. Add a `default` case and `select` won't block at all,
letting you poll a channel and move on if nothing's waiting.

## The sync package: when locks are simpler

Channels shine for moving data, but sometimes you just have shared state many
goroutines touch — a counter, a map, a cache. A **`sync.Mutex`** guards it:

```go
var mu sync.Mutex
count := 0

func record() {
    mu.Lock()
    count++          // only one goroutine in here at a time
    mu.Unlock()
}
```

`Lock`/`Unlock` ensure only one goroutine is in the critical section at once,
preventing the [data race](/learn/programming-go/channels/) you'd otherwise get.

## WaitGroup: waiting for goroutines to finish

If `main` returns while goroutines are still running, they're cut off. A
**`sync.WaitGroup`** waits for them:

```go
var wg sync.WaitGroup
for _, ch := range channels {
    wg.Add(1)
    go func(c Channel) {
        defer wg.Done()
        decode(c)
    }(ch)
}
wg.Wait()   // blocks until every decode finishes
```

## context: cancellation across a call tree

Long-running work needs a way to be told "stop now" — a user cancels, a timeout
fires. Go's **`context`** carries that signal down through every function and
goroutine involved:

```go
func run(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return          // cancelled or timed out
        case s := <-samples:
            process(s)
        }
    }
}
```

A single `cancel()` at the top propagates to every goroutine watching `ctx.Done()`.
GopherTrunk uses exactly this shape to shut a scanner down cleanly.

## Detecting mistakes

Concurrency bugs are subtle, so Go ships a **race detector**: run your tests with
`go test -race` and it flags any unsynchronized shared access at runtime. Make it a
habit — it catches the bugs that are hardest to find by reading.

<div class="knowledge-check" data-quiz data-correct-msg="Right — select proceeds with whichever channel operation is ready first." markdown="0">
  <p class="knowledge-check__q">Quick check: a goroutine must read incoming data but also stop when a quit channel fires. What does it use?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">A select statement over both channels</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Two separate goroutines that never coordinate</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A tight loop polling with time.Sleep</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **`select`** waits on multiple channels and acts on whichever is ready — ideal for
  data-plus-cancellation.
- **`sync.Mutex`** guards shared state; **`sync.WaitGroup`** waits for goroutines to
  finish.
- **`context`** propagates cancellation and timeouts across a whole call tree.
- **`go test -race`** catches unsynchronized access — run it often.

Next up: how Go organizes code into packages and modules.
