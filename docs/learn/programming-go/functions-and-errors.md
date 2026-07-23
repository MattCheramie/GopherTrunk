---
slug: functions-and-errors
title: Functions & error handling
description: Go functions, multiple return values, and the error interface — why Go returns errors as ordinary values instead of throwing exceptions, and the if err != nil pattern you'll see everywhere.
keywords: go functions, multiple return values, go error handling, error interface, if err nil, go errors, named returns, defer
level: beginner
status: full
prereq:
  - values-and-types
faq:
  - q: Why doesn't Go have exceptions?
    a: Go's designers felt exceptions hide the error paths and make control flow hard to follow. Instead, a function that can fail returns an error value alongside its result, and the caller decides what to do right there. The code is more explicit — you can see every place an error is handled — at the cost of a little more typing.
  - q: What does if err != nil mean?
    a: It's the standard Go idiom for checking whether the function you just called failed. Functions return an `error` that is `nil` on success and non-nil on failure, so `if err != nil { ... }` means "if something went wrong, handle it here" — usually by returning the error up to the caller or logging it.
---

# Functions & error handling

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Go functions can return **multiple values**, and the idiom is to return a result
plus an **`error`**. There are **no exceptions**: a function that can fail hands
back an `error` that is **`nil`** on success, and the caller checks it with **`if
err != nil`**. This makes every failure path visible in the code.
</div>

Functions are where Go's philosophy shows most clearly. This lesson covers how you
write them and the error-handling style that pervades every Go program you'll ever
read.

## Writing a function

```go
func channelWidth(highHz, lowHz float64) float64 {
    return highHz - lowHz
}
```

Parameters come with their types; the return type comes after the parentheses. When
consecutive parameters share a type you can write it once (`highHz, lowHz float64`).

## Multiple return values

A Go function can return several values at once — used constantly to return "the
answer, and whether it worked":

```go
func tune(freqHz float64) (float64, error) {
    if freqHz <= 0 {
        return 0, fmt.Errorf("invalid frequency: %v", freqHz)
    }
    return freqHz, nil
}
```

The second return value is an **`error`**. On success it is `nil`; on failure it
carries a message.

## The error pattern

Because errors are just values, the caller handles them immediately:

```go
freq, err := tune(-5)
if err != nil {
    return err        // pass the problem up to whoever called us
}
// safe to use freq here — we know tune succeeded
```

You will see `if err != nil` on a huge fraction of Go lines. It can feel repetitive,
but it means the error paths are right there in front of you — nothing is hidden in
an invisible exception jumping up the stack.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 120" role="img" aria-label="A function returning two values, result and error, with the caller branching on whether error is nil." xmlns="http://www.w3.org/2000/svg">
  <rect x="10" y="45" width="130" height="34" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="75" y="67" text-anchor="middle" font-size="12" fill="currentColor">tune(freq)</text>
  <line x1="140" y1="62" x2="210" y2="62" stroke="currentColor" stroke-width="1.5"/>
  <text x="250" y="52" text-anchor="middle" font-size="12" fill="currentColor">result, err</text>
  <line x1="290" y1="62" x2="350" y2="40" stroke="currentColor" stroke-width="1.5"/>
  <line x1="290" y1="62" x2="350" y2="88" stroke="currentColor" stroke-width="1.5"/>
  <text x="430" y="44" text-anchor="middle" font-size="12" fill="currentColor">err == nil -> use result</text>
  <text x="430" y="94" text-anchor="middle" font-size="12" fill="currentColor">err != nil -> handle it</text>
</svg>
<figcaption>Every fallible call forks the same way: success uses the result; failure handles the error.</figcaption>
</figure>

## defer: cleanup that always runs

`defer` schedules a call to run when the surrounding function returns, no matter how
it returns. It's the standard way to release resources:

```go
f, err := os.Open("capture.cfile")
if err != nil {
    return err
}
defer f.Close()   // runs when this function exits, success or error
```

This keeps "open" and "close" next to each other and guarantees the file is closed
even on an early error return.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a nil error means the call succeeded." markdown="0">
  <p class="knowledge-check__q">Quick check: a function returns <code>(value, error)</code>. What does an <code>error</code> of <code>nil</code> mean?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">The call succeeded</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The call failed</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The value is empty</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Go functions declare parameter and return types and can **return multiple
  values**.
- Fallible functions return an **`error`**; it is **`nil`** on success.
- The **`if err != nil`** idiom handles failures explicitly — Go has **no
  exceptions**.
- **`defer`** guarantees cleanup runs when a function returns.

Next up: bundling data together with structs and giving it behaviour with methods.
