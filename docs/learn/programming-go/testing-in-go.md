---
slug: testing-in-go
title: Testing in Go
description: Go's built-in testing package, table-driven tests, and benchmarks — why testing is a first-class part of the language and how GopherTrunk gates every commit on it.
keywords: go testing, go test, testing package, table driven tests, go benchmarks, unit tests go, testing.T, go test race
level: intermediate
status: full
prereq:
  - functions-and-errors
faq:
  - q: Do I need a testing framework for Go?
    a: No. The standard library's testing package plus the go test command cover unit tests, table-driven tests, benchmarks, and coverage out of the box. Third-party assertion libraries exist, but idiomatic Go often skips them and just uses plain if checks with t.Errorf, keeping tests dependency-free.
  - q: What is a table-driven test?
    a: A test that lists its cases as a slice of structs — each with inputs and the expected output — then loops over them running the same assertion. It's the dominant Go testing style because adding a new case is one line, and each case can be named and reported independently with t.Run.
---

# Testing in Go

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Testing is built into Go: put tests in `_test.go` files, write functions like
`func TestX(t *testing.T)`, and run them with **`go test`**. The idiomatic style is
**table-driven** — a list of cases looped over. Benchmarks and the **race detector**
come free too. GopherTrunk gates every commit on **`make vet test`**.
</div>

In Go, testing isn't an add-on you bolt on later — it's part of the toolchain from
day one. This lesson shows the shape of a Go test and the style Go projects use.

## The anatomy of a test

Tests live beside the code they test, in files ending `_test.go`, and are just
functions taking `*testing.T`:

```go
// channel_test.go
package scanner

import "testing"

func TestWavelength(t *testing.T) {
    got := Channel{FreqHz: 300e6}.Wavelength()
    want := 1.0
    if got != want {
        t.Errorf("Wavelength() = %v, want %v", got, want)
    }
}
```

Run it with `go test ./...`. A test *fails* by calling `t.Errorf` (or `t.Fatalf` to
stop immediately); a test that returns without complaint passes. No assertion
library required.

## Table-driven tests

The dominant Go style lists cases in a table and loops:

```go
func TestChannelWidth(t *testing.T) {
    cases := []struct {
        name           string
        high, low, want float64
    }{
        {"12.5 kHz", 851.01875e6, 851.00625e6, 12500},
        {"zero width", 460e6, 460e6, 0},
    }
    for _, c := range cases {
        t.Run(c.name, func(t *testing.T) {
            if got := channelWidth(c.high, c.low); got != c.want {
                t.Errorf("got %v, want %v", got, c.want)
            }
        })
    }
}
```

Adding a case is one line, and `t.Run` gives each case its own name in the output.
This is exactly the pattern GopherTrunk uses throughout — including the regression
tests its [contribution rules](/learn/git/pull-requests/) require for every bug fix.

## Benchmarks and the race detector

The same package measures performance. A `func BenchmarkX(b *testing.B)` run with
`go test -bench=.` reports nanoseconds per operation — invaluable for DSP hot paths.
And `go test -race` re-runs your tests watching for the concurrency bugs from
[select & synchronization](/learn/programming-go/select-and-sync/).

## Testing as a gate

Because tests are so easy to run, projects wire them into a single gate. GopherTrunk's
is `make vet test` — `go vet` plus the whole suite, green before any commit. That one
habit is what lets a large codebase change quickly without breaking. You'll see how
CI enforces the same gate on every push in the
[Deployment module](/learn/deployment/ci-cd-pipelines/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — Go test files end in _test.go and use the testing package." markdown="0">
  <p class="knowledge-check__q">Quick check: where do Go unit tests live?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">In a separate tests/ folder</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">In files ending _test.go beside the code</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Inside the main function</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Tests live in **`_test.go`** files as `func TestX(t *testing.T)`, run with **`go
  test`**.
- The idiomatic style is **table-driven** — a list of cases looped with `t.Run`.
- **Benchmarks** and the **`-race`** detector come built in.
- Projects gate commits on tests — GopherTrunk uses **`make vet test`**.

Next up: a tour of the standard library that ships with Go.
