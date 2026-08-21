---
slug: reading-error-messages
title: Reading error messages & stack traces
description: Errors and panics are structured evidence, not noise — how to read a Go panic and stack trace from the top, find the frame that matters, and follow wrapped errors to their origin.
keywords: go stack trace, reading panic go, error messages go, goroutine stack trace, index out of range, nil pointer dereference, error wrapping unwrap
level: beginner
status: full
---

# Reading error messages & stack traces

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An error message is **structured evidence** — the program's own account of
where and why it gave up. A Go **panic** prints the failure kind (with the
actual values involved), then a **stack trace**: the chain of calls, innermost
first, each with file and line. Read **top-down**: the first frames in *your*
code are where to look, and the frames below explain **how execution got
there**. Wrapped errors read the same way in miniature — outermost context
first, root cause last. The habit that changes everything: **read the words**,
slowly, before touching anything.
</div>

The most common beginner debugging mistake isn't a technique — it's a flinch.
An error fills the terminal, it looks like machine noise, and the eyes slide
off it into guessing. This lesson retrains the flinch: everything in that wall
of text was put there to help you, and reading it takes two minutes.

## Anatomy of a panic

Here's a representative Go crash, the kind a decoder bug produces:

```text
panic: runtime error: index out of range [4] with length 4

goroutine 17 [running]:
example/radio/decoder.frameSlot(...)
        /home/dev/gt/decoder/frames.go:88
example/radio/decoder.DecodeBurst(0xc0000b4000, 0x40, 0x40)
        /home/dev/gt/decoder/decode.go:41 +0x1c5
example/radio/scanner.processChunk(0xc0000a6120)
        /home/dev/gt/scanner/scanner.go:203 +0x8b
main.run()
        /home/dev/gt/cmd/scan/main.go:57 +0x2ea
```

Take it line by line:

- **The first line is the diagnosis**, and it's specific: not "error," but
  *index out of range*, with the values — the code asked for index `[4]` of
  something of length `4`. (Lengths count from 1, indexes from 0, so 4 is one
  past the end: this is an off-by-one, and the message just told you so.)
- **`goroutine 17 [running]`** — which concurrent thread of execution crashed.
  A real crash dump lists *every* goroutine; the one marked `[running]` at the
  top is the one that panicked, and it's the one to read first.
- **The frames, innermost first.** The top frame (`frames.go:88`) is where the
  panic physically happened. Each frame below is the caller of the one above:
  `DecodeBurst` called `frameSlot`, `processChunk` called `DecodeBurst`, and
  so on down to `main`. The trace is the answer to "how did we get here?"

## Which frame is *the* frame?

The top frame is where the failure **surfaced** — remember from
[What is a bug?](/learn/testing/what-is-a-bug/) that this is not necessarily
where the defect **lives**. The reading strategy:

1. Scan from the top for **the first frame in your own code**. Traces often
   start inside the standard library or a dependency — those frames are
   almost never the bug; they're your code's bad input detonating downstream.
2. Open that file at that line. Ask the message's question: *what is length 4
   here, and why did we ask for its element 4?*
3. If the values at that line came from elsewhere (a struct filled earlier, an
   argument), **walk down the trace** — each lower frame is a place the bad
   value may have been produced or should have been rejected.

> Rule of thumb: the trace tells you where the failure *surfaced* and the path
> that led there. Finding where the bad value was *born* is the actual hunt —
> and it's next lesson's job when reading alone isn't enough.

## Reading error values (the non-crash kind)

Go's everyday errors don't crash — they return, and good code **wraps** them
with context at each level (the
[error-handling lesson](/learn/programming-go/error-handling-patterns/) shows
the `%w` mechanics). The result reads like a miniature stack trace in one
line, outermost context first:

```text
start scanner: load config "site.yaml": parse talkgroups: line 14: invalid ID "12a4"
```

Read it right-to-left for the root cause (an invalid ID on line 14), and
left-to-right for the story of who cared. Everything you need — file, line,
offending value — is in the sentence, put there by a developer for exactly
this moment. When *you* return errors, honor the same contract:
`fmt.Errorf("parse talkgroups: %w", err)` costs seconds and pays whoever
debugs next — including you, reading your own log at 2 a.m., as the
[failure-message lesson](/learn/testing/anatomy-of-a-test/) already argued for
tests.

## The panic bestiary

Three runtime panics account for most Go crashes; each message names its bug
pattern:

| Message | It means | The usual defect |
|---------|----------|------------------|
| `index out of range [N] with length L` | Slice/array access past the end | Off-by-one, or an empty slice nobody checked |
| `invalid memory address or nil pointer dereference` | A method or field on a `nil` pointer | A constructor's error was ignored, a map lookup missed, a field never initialized |
| `assignment to entry in nil map` | Writing to a map that was declared but never `make`d | `var m map[string]int` instead of `m := make(map[string]int)` |

Meeting any of these, you already know the *kind* of mistake before opening
the file — the message did the classification for you.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the innermost frames are often library code your bad input crashed inside; the first of your own frames is where the investigation starts." markdown="0">
  <p class="knowledge-check__q">Quick check: the top three frames of a panic trace are inside the standard library. What does that usually mean?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Your code, a few frames down, handed the library something invalid — find the first frame in your own code</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The standard library has a bug — report it upstream</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The trace is corrupted and can't be used</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Error output is **evidence**: read the words, slowly, before changing
  anything.
- A panic prints the **diagnosis with values** first, then the stack —
  **innermost frame first**, each lower frame the caller of the one above.
- Start at the **first frame in your own code**; library frames at the top are
  usually your bad input detonating downstream.
- The trace shows where the failure **surfaced** — the defect may live down
  the trace, where the bad value was born.
- Wrapped errors read like one-line traces — root cause at the right end —
  so **wrap with context** in your own code too.

Next up: [Print debugging & logging](/learn/testing/print-debugging-and-logging/)
