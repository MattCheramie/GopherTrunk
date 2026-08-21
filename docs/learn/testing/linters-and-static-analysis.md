---
slug: linters-and-static-analysis
title: Linters & static analysis
description: Tools that read your code without running it — go vet, staticcheck, and golangci-lint — and the bug classes they catch for free, before a single test executes.
keywords: go vet, staticcheck, golangci-lint, static analysis go, linter go, printf vet check, shadowed variable, unreachable code
level: intermediate
status: full
faq:
  - q: What's the difference between a linter and a test?
    a: A test runs your code on chosen inputs and checks the results; a linter reads the source without running it and flags patterns that are wrong or suspicious — a Printf with mismatched arguments, an error checked against the wrong variable, an unused result. Tests verify behavior you thought to check; linters catch whole classes of mistakes across all code, including code no test touches.
  - q: Is go vet the same as the Go compiler's error checking?
    a: No. The compiler rejects code that isn't valid Go. go vet accepts valid Go and looks for code that compiles but is probably wrong — like fmt.Printf("%d", "hello"), which is legal and broken. Vet ships with the toolchain, is fast, and has almost no false positives, which is why projects like GopherTrunk run it before every commit.
  - q: Which linters should a Go project actually run?
    a: Start with go vet — free, fast, near-zero noise. Add staticcheck for a much deeper catalog of correctness and simplification checks. Projects wanting one tool that bundles many linters with per-repo configuration typically adopt golangci-lint. The main discipline is to keep the enabled set signal-heavy — a noisy linter gets ignored, and an ignored linter catches nothing.
---

# Linters & static analysis

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Static analysis** finds bugs by **reading code without running it**. Go's
built-in **`go vet`** catches legal-but-wrong patterns — mismatched `Printf`
arguments, copied locks, unreachable code — with near-zero false positives;
**staticcheck** and bundlers like **golangci-lint** go deeper. Because analysis
covers **all** code, not just tested paths, it's the cheapest layer in the
defense stack: catches at the far-left of the **cost curve**, before anything
executes. That's why GopherTrunk's commit gate is `make vet test` — vet runs
*with* the tests, every time.
</div>

Unit 4 turns from writing tests to the machinery around them. First tool in the
belt: programs that inspect your source the way a sharp-eyed reviewer would —
except they read every line, every time, and never get tired.

## Finding bugs without running anything

A test needs the code to run, an input chosen, an assertion written. Static
analysis needs none of that — it examines the source and flags patterns that
are provably or probably wrong. Consider:

```go
fmt.Printf("decoded %d frames on talkgroup %s\n", count)
```

Valid Go: it compiles. Also definitely broken: two verbs, one argument, and
`%s` will meet an `int` if you "fix" it carelessly. No compiler error, and a
test would only catch it if some assertion inspected this exact log line —
which no test does. `go vet` flags it instantly, because the mistake is visible
*in the text of the program itself*.

That's the defining strength: **coverage without tests**. Your
[coverage report](/learn/testing/code-coverage/) shows red regions no test
executes; the linter reads those too. It checks the error path nobody wrote a
test for and the log statement nobody asserts on — for the classes of bug it
knows, it checks *everything*, at the cost-curve's absolute far left.

## What go vet catches

`go vet` ships with the toolchain and runs in seconds:

```bash
go vet ./...
```

A sampler of its checks, each a real-world bug class:

| Check | The mistake it catches |
|-------|------------------------|
| `printf` | Format verbs that don't match arguments in number or type |
| `copylocks` | Copying a struct containing a `sync.Mutex` — silently forks the lock |
| `loopclosure` | Goroutine capturing a loop variable that changes under it |
| `unreachable` | Code after an unconditional return — usually a misplaced brace |
| `nilfunc` | Comparing a function against nil in a way that's always true |
| `shadow` (opt-in) | An inner `err :=` shadowing the outer `err` — checks the wrong error |

Vet's design philosophy is why it's trusted enough to gate commits: it only
reports patterns that are *almost certainly* wrong. When vet speaks, you fix;
you don't debate.

## Beyond vet: staticcheck and friends

**staticcheck** applies hundreds of deeper checks — subtler correctness bugs
(a `time.Duration` multiplied wrongly, an impossible comparison, a misused
`append`), plus simplifications and deprecated-API warnings. **golangci-lint**
bundles vet, staticcheck, and dozens of other linters behind one command and a
per-repo config file, which is how most sizable Go projects run linting in
[CI](/learn/testing/continuous-integration/).

The discipline that keeps any of this useful is **signal management**. A linter
that cries wolf gets ignored, and an ignored linter catches nothing. Enable
checks whose findings you'll actually fix; when a specific finding is a
considered false positive, suppress it *narrowly and visibly* (`//nolint:`
with a reason) rather than disabling the check globally. Warnings that scroll
by unfixed are worse than no linter — they train everyone to skim past the day
a real bug appears in the list.

> Rule of thumb: the linter's output should normally be *empty*. Zero warnings
> is a state worth defending, because it makes the first new warning
> unmissable.

## Where linting sits in the defense stack

Each layer catches what the previous can't, at rising cost:

1. **Compiler** — invalid code. Free, instant, mandatory.
2. **Linters / static analysis** — legal-but-wrong patterns, across *all*
   code. Seconds, no tests required.
3. **Tests** — wrong *behavior* on chosen inputs. Requires writing them.
4. **Review, CI** — what all the machines missed.

GopherTrunk wires layer 2 directly into the everyday loop: the standing rule
is that **`make vet test`** — vet *plus* the unit suite, one command — must be
green before any commit. Vet isn't a special occasional deep-clean; it's part
of what "the build is green" *means*. The full anatomy of that gate is a
[Unit 6 lesson](/learn/testing/make-vet-test/). Formatting, the other
machine-enforced hygiene layer, is next.

<div class="knowledge-check" data-quiz data-correct-msg="Right — analysis reads all the source, so it checks code no test executes; that's its edge over testing for the bug classes it knows." markdown="0">
  <p class="knowledge-check__q">Quick check: what can a linter check that your test suite structurally cannot?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Code that no test ever executes — analysis reads every line, tested or not</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Whether functions return correct results for specific inputs</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">How the program behaves under production load</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Static analysis** finds bugs by reading source — no execution, no inputs,
  no assertions needed.
- **`go vet`**: toolchain-standard, fast, near-zero false positives — printf
  mismatches, copied locks, loop-closure captures, unreachable code.
- **staticcheck** / **golangci-lint** extend the catalog; keep the enabled set
  signal-heavy and suppressions narrow.
- Linters check **all** code, including the never-tested paths — the cheapest
  catches on the whole cost curve.
- GopherTrunk's gate is **`make vet test`**: vet runs with the tests, before
  every commit, always.

Next up: [Formatters & style](/learn/testing/formatters-and-style/)
