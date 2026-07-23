---
slug: debugging-go
title: Debugging Go
description: "Finding bugs with print debugging, the delve debugger, the race detector, and static analysis — the toolkit for when a test goes red and you need to know why."
keywords: go debugging, delve, dlv, go test -race, race detector, go vet, staticcheck, print debugging go, go debugger
level: intermediate
status: full
prereq:
  - testing-in-go
faq:
  - q: What is the Go race detector and when should I use it?
    a: "It's a runtime tool you enable with go test -race (or go run -race). It instruments memory access and reports when two goroutines touch the same variable concurrently with at least one write — a data race. Run it whenever you change concurrent code; races are timing-dependent and may not show up otherwise."
  - q: Do I need Delve, or is print debugging enough?
    a: "Both have their place. A quick fmt.Println or log line is often the fastest way to see a value. Delve (dlv) earns its keep when you need to pause execution, step line by line, inspect goroutines, and set conditional breakpoints — especially in code that's hard to reproduce with prints alone."
---

# Debugging Go

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
When a test goes red, Go gives you a layered toolkit. **Print debugging** (`fmt` /
`log`) is the fast first move. **Delve** (**`dlv`**) is the real debugger — breakpoints,
stepping, inspecting goroutines. **`go test -race`** catches data races. And static
analysis — **`go vet`** and **staticcheck** — flags whole classes of bugs before you
ever run the code.
</div>

Every programmer spends more time finding bugs than writing them. This lesson covers
Go's debugging tools, which pair naturally with the
[testing](/learn/programming-go/testing-in-go/) habit of reproducing a bug as a
failing test first.

## Start with print debugging

The fastest way to see what a program is doing is often to print it:

```go
log.Printf("demod: snr=%.1f symbols=%d locked=%v", snr, n, locked)
```

Use `log` over `fmt.Println` for anything you might keep — it timestamps and can be
turned off. The `%+v` verb prints a struct with field names, and `%#v` prints it as
Go source, both invaluable for inspecting a value:

```go
log.Printf("%+v", cfg)   // {Gain:20 FreqHz:4.60025e+08 Active:true}
```

## Delve, the real debugger

When prints aren't enough, `dlv` lets you stop time. Debug a test and step through it:

```
$ dlv test ./internal/scanner
(dlv) break Demodulate       # set a breakpoint
(dlv) continue               # run to it
(dlv) print snr              # inspect a variable
(dlv) next                   # step one line
(dlv) goroutines             # list all goroutines
```

Breakpoints, single-stepping, and goroutine inspection make it the tool for bugs you
can't pin down with prints — a wedged goroutine, a value that's wrong only sometimes.

## The race detector

Concurrency bugs are the ones prints hide, because adding a print changes the timing.
The race detector finds them properly:

```
$ go test -race ./...
==================
WARNING: DATA RACE
Write at 0x00c0000b4010 by goroutine 7:
  ...
Previous read at 0x00c0000b4010 by goroutine 6:
  ...
```

It reports the exact two accesses and their goroutines. Run `-race` whenever you touch
[channels](/learn/programming-go/channels/) or shared state — it's how GopherTrunk
keeps its multi-goroutine sample pipeline honest.

## Static analysis catches bugs before you run

Some bugs never need to run to be found. The table lists Go's analysis layers.

| Tool | Catches |
|------|---------|
| `go vet` | Printf format mistakes, unreachable code, bad struct tags |
| `go test -race` | Data races between goroutines |
| `staticcheck` | Dead code, misused APIs, needless conversions |
| `gofmt` / `go vet` in CI | Style and correctness gates on every push |

GopherTrunk's `make vet test` gate runs `go vet` plus the full suite before any
commit — so an entire category of mistakes is caught automatically, not in review.

<div class="knowledge-check" data-quiz data-correct-msg="Right — -race instruments memory access to catch concurrent read/write bugs." markdown="0">
  <p class="knowledge-check__q">Quick check: which tool finds two goroutines touching the same variable at once?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">gofmt</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">go test -race, the race detector</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">go build</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Print debugging** with `log` and `%+v`/`%#v` is the fast first move.
- **Delve** (`dlv`) gives breakpoints, stepping, and goroutine inspection.
- **`go test -race`** catches data races that timing hides.
- **`go vet`** and **staticcheck** catch bug classes before the code runs.

Next up: a tour of the batteries the standard library ships with.
