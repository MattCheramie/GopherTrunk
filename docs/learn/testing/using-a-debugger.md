---
slug: using-a-debugger
title: Using a debugger
description: Pause a live program, inspect its state, step line by line — Delve for Go, the core vocabulary of breakpoints and stepping, and an honest guide to when a debugger beats prints.
keywords: delve go debugger, dlv debug, breakpoints go, step into step over, debugging go tests, conditional breakpoint, inspect variables debugger
level: intermediate
status: full
prereq:
  - print-debugging-and-logging
---

# Using a debugger

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **debugger** pauses a live program at a **breakpoint** and lets you inspect
**everything** — variables, the call stack, other goroutines — then advance
line by line with **next / step / continue**. Go's debugger is **Delve**
(`dlv`), and every major editor front-ends it. Its edge over prints: no
guessing what to print — the whole state is *there*, and you can chase
surprises immediately without another edit-rerun cycle. Its limits: timing
bugs, long time-scales, and remote machines — where
**[logging](/learn/testing/print-debugging-and-logging/)** stays king.
</div>

Prints answer the questions you thought to ask. This lesson's tool answers the
follow-ups too — while the program stands frozen mid-mistake, waiting for you
to look around.

## The core idea: stop time

A debugger runs your program under supervision. You mark a line as a
**breakpoint**; when execution reaches it, the program **freezes** — not
crashed, just paused — and the debugger hands you the controls:

- **Inspect any variable** in scope — not just ones you pre-chose to print.
- **Walk the call stack** — the same
  [frames you learned to read](/learn/testing/reading-error-messages/) in a
  panic, but *live*: click into any frame and inspect *its* locals.
- **Advance deliberately**: `next` (run this line, stop at the following),
  `step` (descend *into* the function being called), `continue` (run to the
  next breakpoint).

The experience that sells it: you're paused where the bad value appears, and
instead of adding another print and re-running, you just… look at the other
seventeen variables, and the caller's, until the surprise is found. Each
"wait, what's *that*?" costs seconds, not a rebuild cycle.

## Delve in five minutes

Go's debugger is **Delve**. Editors (VS Code, GoLand) wrap it in a UI —
breakpoints as gutter clicks — and the
[Go module's debugging lesson](/learn/programming-go/debugging-go/) covers
that setup; here's the raw CLI, worth seeing once so the UI never feels like
magic:

```bash
go install github.com/go-delve/delve/cmd/dlv@latest

dlv debug ./cmd/scan          # build + run under the debugger
dlv test ./decoder/            # debug a TEST — see below
```

A session against a misbehaving decode:

```text
(dlv) break decoder.DecodeBurst      # breakpoint at a function
(dlv) continue                       # run until it's hit
> decoder.DecodeBurst() ./decoder/decode.go:37

(dlv) print len(raw)                 # inspect anything in scope
12
(dlv) locals                         # all local variables
(dlv) next                           # advance one line
(dlv) stack                          # where am I? (live call stack)
(dlv) continue                       # onward to the next hit
```

Two power moves worth learning early. **`dlv test`** debugs a test — combined
with a [failing-first regression test](/learn/testing/regression-tests/), it
drops you *inside a reproduced bug* with two commands, which is the highest
debugging standard of living there is. And **conditional breakpoints** solve
the loop problem — a breakpoint in a hot decode loop hit 4,800 times a second
is unusable until you attach the condition that describes your suspect:

```text
(dlv) break decode.go:52
(dlv) condition 1 frame.Talkgroup == 4521 && !frame.CRCOK
```

Now the program runs at (near) full speed and stops only in the interesting
case — the debugger equivalent of a well-aimed hypothesis print.

## Debugger or prints? An honest decision table

| Situation | Reach for |
|-----------|-----------|
| Rich state to explore, unclear what's relevant yet | **Debugger** — everything inspectable, no guessing what to print |
| Bug reproduced inside a test | **Debugger** (`dlv test`) — frozen at the crime scene |
| Timing/concurrency suspect | **Prints/logs** — pausing *changes* timing; a frozen goroutine hides the race the others were losing |
| Behavior over minutes or hours (a slow drift, a leak) | **Logs** — you can't single-step an afternoon; you read its recording |
| Field machine you can't attach to | **Logs** — [the only witness](/learn/testing/print-debugging-and-logging/) |
| One known value to check at one known point | Either — a print may honestly be faster than launching a session |

The pattern behind the rows: a debugger examines **a moment** in depth; logs
record **a history** in breadth. Deep-narrow versus shallow-wide — which is
why real-time signal-processing code like GopherTrunk's leans logs-first (a
paused DSP pipeline is no longer real-time, and the interesting bugs live in
the flow), while a wrong-answer bug in a pure function is Delve's home game.

> Rule of thumb: debugger for *what is wrong with this state*; logs for *how
> did we get here over time*. Master both; choose per bug.

<div class="knowledge-check" data-quiz data-correct-msg="Right — pausing perturbs timing and freezes the interplay between goroutines, so races often vanish under a debugger; logging (and -race) serve those bugs better." markdown="0">
  <p class="knowledge-check__q">Quick check: which bug is a debugger LEAST suited to?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">A race between two goroutines that only fails under production timing</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A pure function returning the wrong value inside a failing test</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A wrong branch taken on one specific rare input</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A debugger **pauses** a live program at **breakpoints** and exposes all
  state — variables, live **stack**, goroutines.
- Core moves: `break`, `continue`, `next`, `step`, `print`, `stack` — that
  vocabulary is 90% of daily use, in any front-end.
- **Delve** (`dlv`) is Go's debugger; **`dlv test`** + a failing regression
  test drops you inside a reproduced bug.
- **Conditional breakpoints** make hot loops debuggable — stop only when your
  suspect condition holds.
- Debuggers examine a **moment**; logs record a **history** — timing bugs,
  long time-scales, and remote failures stay logs-first.

Next up: [Bisecting history](/learn/testing/bisecting-history/)
