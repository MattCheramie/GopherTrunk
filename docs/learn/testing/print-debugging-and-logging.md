---
slug: print-debugging-and-logging
title: Print debugging & logging
description: The humble print statement, done well — hypothesis-driven printing that localizes bugs fast, and how structured logging turns one-off debugging into a permanent diagnostic instrument.
keywords: print debugging, logging go, slog structured logging, debug logging levels, printf debugging technique, log vs print, diagnostic logging
level: beginner
status: full
prereq:
  - reading-error-messages
---

# Print debugging & logging

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Print debugging** — adding statements that show what the program is actually
doing — is legitimate, universal, and sharpest when each print tests a
**hypothesis** rather than spraying output everywhere. Print **values and
locations**, then **bisect** the pipeline: find the stage where good data
turns bad. **Logging** is the grown-up sibling: permanent, **leveled**
(debug/info/warn/error), and **structured** (key-value, machine-queryable, as
with Go's `slog`) — a diagnostic instrument you install once and use for every
future bug, including the ones that only happen in the field overnight.
</div>

Debuggers get the glamour, but ask working engineers what they actually reach
for first and most will admit it: prints. Rightly so — used with discipline
it's fast, works everywhere (including places a debugger can't go), and leads
naturally to the logging that keeps paying after the bug is fixed.

## Print debugging without the mess

The difference between flailing and technique is one word: **hypothesis**. You
[reproduced the failure](/learn/testing/reproducing-a-bug/); you have a theory
about where things go wrong; the print exists to make the program *answer*.

```go
func DecodeBurst(raw []byte) (Frame, error) {
    fmt.Printf("DEBUG DecodeBurst: len=%d first=%x\n", len(raw), raw[:min(8, len(raw))])
    ...
}
```

Rules that keep it sharp:

- **Print values, not greetings.** `"got here"` answers one narrow question.
  `len=12 first=5d00c2` answers the question you actually have — *what is the
  data at this point?* — and usually several you didn't know you had.
- **Include the location** (`DecodeBurst:`) — five prints of bare numbers
  interleaved from a pipeline are unreadable.
- **Bisect with prints.** Data flows through stages: capture → filter → sync →
  decode → audio. Print the data where it's known-good and where it's
  known-bad, then keep splitting the interval — the same halving loop as
  [reproduction shrinking](/learn/testing/reproducing-a-bug/), applied to
  *space in the pipeline* instead of the input. Three or four prints localize
  a bug to one stage; that stage gets the close reading.
- **Remove them when done** — with one exception worth its own section below.
  A codebase silted with dead `DEBUG` prints is noise the next debugger pays
  for. (`git diff` before committing is the net; you set that habit in the
  [Git module](/learn/git/status-and-diffs/).)

One honest caveat: in concurrent code, a print changes timing, and can make a
[race-driven flake](/learn/testing/flaky-tests/) hide while you're looking —
if a bug vanishes when printed, that's *itself* strong evidence you're chasing
a race, and `-race` is the better instrument.

## From prints to logging: make the instrument permanent

Here's the promotion question: that print at the decoder boundary — inputs
arriving, frames produced, errors hit — would have answered *last* month's bug
too, and will answer next month's. Deleting it after each hunt and rewriting
it next time is waste. **Logging** is print debugging institutionalized: the
useful observations, kept, controlled by configuration instead of editing.

Two upgrades turn prints into logs:

**Levels** make verbosity a runtime decision:

| Level | Meaning | On by default? |
|-------|---------|----------------|
| `debug` | Diagnostic detail — the promoted prints | No — flipped on when hunting |
| `info` | Normal, notable operation ("locked control channel") | Yes |
| `warn` | Wrong-ish but surviving ("dropped 3 chunks") | Yes |
| `error` | An operation failed | Yes |

**Structure** makes logs queryable. Go's standard `slog`
([covered in the Go module](/learn/programming-go/logging-and-slog/)) attaches
key-value pairs instead of burying values in prose:

```go
slog.Debug("burst decoded", "len", len(raw), "talkgroup", f.Talkgroup, "crc_ok", f.CRCOK)
```

Prose logs are read; structured logs are *interrogated* — "show every
`crc_ok=false` in the hour before the failure, by talkgroup" is a filter, not
an evening. That difference decides investigations.

## Why logging is the field's only witness

The bugs that matter most are the ones you *can't* attach anything to: they
happen on a user's machine, overnight, in the tenth hour of a run — precisely
the [hard-to-reproduce kind](/learn/testing/reproducing-a-bug/). For those,
whatever the program logged **is the entire evidence**. This is standard
operating reality for long-running software like a trunking scanner: when an
operator reports "it lost the control channel around 3 a.m.," the
investigation is a debug-level log showing what the decoder observed — signal
quality decaying, resyncs firing, the watchdog tripping — minutes before the
symptom. Good logging is the difference between reading that story and
shrugging.

> Rule of thumb: log at the **boundaries** — inputs arriving, outputs leaving,
> decisions made, errors met. When something is puzzling enough to print once,
> it's usually worth logging forever at `debug`.

<div class="knowledge-check" data-quiz data-correct-msg="Right — printing known-good and known-bad points and halving the interval localizes the corruption to one stage in a handful of prints." markdown="0">
  <p class="knowledge-check__q">Quick check: data enters a five-stage pipeline correct and exits wrong. What's the print-debugging move?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Add "got here" prints to every function in all five stages</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Rewrite the pipeline stage you trust least and see if the symptom changes</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Print the data mid-pipeline and bisect — keep halving the good-to-bad interval until one stage is left</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Print debugging is legitimate — done as **hypothesis testing**: print
  **values with locations**, not "got here."
- **Bisect the pipeline**: localize where good data turns bad in a handful of
  prints.
- Clean up prints afterward — but **promote** the durably useful ones into
  **leveled, structured logs** (`slog`).
- Structured logs are **interrogated**, not read — and for field failures the
  log is the **only witness** there is.
- Log at **boundaries**: inputs, outputs, decisions, errors.

Next up: [Using a debugger](/learn/testing/using-a-debugger/)
