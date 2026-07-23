---
slug: control-flow
title: Control flow
description: "if, for, and switch — Go's deliberately small set of control structures, including the single loop keyword that replaces while, do-while, and for-each."
keywords: go control flow, go for loop, go if statement, go switch, go while loop, go range, switch fallthrough
level: beginner
status: full
prereq:
  - values-and-types
faq:
  - q: Does Go have a while loop?
    a: "No — and it doesn't need one. Go has exactly one loop keyword, for, and it covers every case: for cond {} is a while loop, for {} is an infinite loop, and for i := 0; i < n; i++ {} is the classic counted loop. One keyword, three shapes."
  - q: Does a Go switch fall through to the next case?
    a: "No. Unlike C, each Go case breaks automatically — there is no accidental fall-through. If you genuinely want the next case to run too, you add an explicit fallthrough statement, which is rare."
---

# Control flow

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Go keeps control flow tiny: **`if`**, **`for`**, and **`switch`**. There is no
`while` — the single **`for`** keyword does counted loops, condition loops, infinite
loops, and (with **`range`**) iteration over collections. A **`switch`** breaks after
each case by default, so there is no accidental fall-through. Small set, few
surprises.
</div>

This lesson covers every way Go changes the path a program takes. It builds on
[Values, variables & types](/learn/programming-go/values-and-types/) and sets up
[Functions & error handling](/learn/programming-go/functions-and-errors/), where
`if err != nil` becomes the most-used branch in the language.

## How does `if` work in Go?

An `if` needs no parentheses around its condition, and the braces are always
required:

```go
if snr < 6.0 {
    return errNoLock
}
```

Go adds one twist — an `if` can declare a variable scoped to the branch:

```go
if peak := findPeak(fft); peak.Power > threshold {
    tune(peak.FreqHz)   // peak is visible here...
}
// ...and gone here.
```

This keeps short-lived values out of the surrounding scope, which is why you'll see
`if err := do(); err != nil { ... }` everywhere in Go.

## The one loop: `for`

Go has no `while` and no `do`. Everything loops with `for`, in three shapes:

```go
for i := 0; i < 8; i++ {        // classic counted loop
    process(taps[i])
}

for scanning {                  // condition-only — this is Go's "while"
    scanning = step()
}

for {                           // no condition — loop forever
    sample := <-samples
    handle(sample)              // break or return to stop
}
```

To walk a slice, map, or channel, add `range`:

```go
for i, s := range symbols {     // index and value
    decode(i, s)
}
for freq := range channels {    // map: keys
    tune(freq)
}
```

`range` over a channel receives until it is closed — the pattern GopherTrunk's
sample pipeline uses to drain a stream of `complex64` values.

## What makes Go's `switch` different?

A `switch` is a cleaner chain of `if`/`else`. Crucially, each case **breaks
automatically** — no `break` needed, no accidental fall-through:

```go
switch mode {
case "P25":
    decodeP25(frame)
case "DMR", "NXDN":     // one case, several values
    decodeTDMA(frame)
default:
    log.Printf("unknown mode %q", mode)
}
```

You can also omit the value and use `switch` as a tidy if-chain:

```go
switch {
case snr > 20:
    quality = "excellent"
case snr > 8:
    quality = "usable"
default:
    quality = "no lock"
}
```

The table below sums up the whole toolkit.

| Construct | Shape | Use it for |
|-----------|-------|-----------|
| `if` / `else` | `if cond { }` | A one-off branch, often `if err != nil` |
| `for` (counted) | `for i := 0; i < n; i++` | A known number of iterations |
| `for` (condition) | `for cond { }` | Go's `while` loop |
| `for` (infinite) | `for { }` | Loop until `break`/`return` |
| `for range` | `for k, v := range x` | Walking slices, maps, channels |
| `switch` | `switch x { case … }` | Many branches on one value |

<div class="knowledge-check" data-quiz data-correct-msg="Right — Go's for is its while; there is no separate while keyword." markdown="0">
  <p class="knowledge-check__q">Quick check: how do you write a while loop in Go?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">With the while keyword</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">With for and a single condition: for cond { }</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">With a do…until block</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **`if`** needs no parentheses and can declare a branch-scoped variable.
- **`for`** is Go's only loop — counted, condition (its `while`), infinite, or
  `range`.
- **`range`** walks slices, maps, and channels.
- **`switch`** breaks after each case by default; `fallthrough` is opt-in and rare.

Next up: functions, multiple return values, and Go's `error` type.
