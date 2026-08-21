---
slug: your-first-go-test
title: Your first Go test
description: Write a _test.go file, run go test, and read a failure — the complete edit-test loop with nothing but Go's standard library. A hands-on walkthrough from empty directory to green.
keywords: go test tutorial, first go test, testing package go, _test.go, go test command, testing.T, run go tests, go test verbose
level: beginner
status: full
prereq:
  - anatomy-of-a-test
---

# Your first Go test

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Go tests need **no framework**: put a function named **`TestXxx(t *testing.T)`**
in a file ending **`_test.go`**, and **`go test`** finds and runs it. Failures
are reported through **`t.Errorf`**/`t.Fatalf`; a silent run means green. The
loop — edit, `go test`, read the failure, fix — takes seconds, and this lesson
walks it end-to-end, including the crucial step beginners skip: **watching your
test fail** so you know it can.
</div>

Time to stop reading and type. This lesson builds a tiny piece of
radio-flavored code and its first test, using nothing but the standard library —
the same tooling GopherTrunk's entire suite runs on. If Go isn't installed yet,
detour through the [Go module's setup lesson](/learn/programming-go/hello-go/)
first.

## Step 1 — the code under test

Make a directory, initialize a module, and create a function worth testing.
Radio frequencies are usually stored as integer hertz and shown as megahertz,
so:

```bash
mkdir freqfmt && cd freqfmt
go mod init example/freqfmt
```

```go
// freq.go
package freqfmt

import "fmt"

// FormatMHz renders integer hertz as a megahertz string, e.g. 851012500 -> "851.0125 MHz".
func FormatMHz(hz int64) string {
    return fmt.Sprintf("%.4f MHz", float64(hz)/1e6)
}
```

## Step 2 — the test file

Test code lives beside the code, in a file whose name ends `_test.go`. Inside,
any function shaped `func TestXxx(t *testing.T)` is a test:

```go
// freq_test.go
package freqfmt

import "testing"

func TestFormatMHz_TypicalFrequency(t *testing.T) {
    got := FormatMHz(851_012_500)

    want := "851.0125 MHz"
    if got != want {
        t.Errorf("FormatMHz(851012500) = %q, want %q", got, want)
    }
}
```

That's a complete test: arrange (the literal input), act (one call), assert
(got vs want). No framework, no imports beyond `testing` — the `go` tool itself
is the runner.

## Step 3 — run it

```bash
go test
```

```text
PASS
ok      example/freqfmt 0.002s
```

Quiet output is the point: green says almost nothing, by design. Add `-v` to see
each test named as it runs, and `./...` to run every package under the current
directory — the form you'll use in real projects:

```bash
go test -v ./...
```

## Step 4 — break it on purpose

Here's the step that separates a real test from a decoration. **Make the code
wrong and watch the test catch it.** Change `1e6` to `1e5` in `freq.go`, then:

```text
--- FAIL: TestFormatMHz_TypicalFrequency (0.00s)
    freq_test.go:10: FormatMHz(851012500) = "8510.1250 MHz", want "851.0125 MHz"
FAIL
```

Read the failure like a report, because it is one: the file and line, the call
and its input, what came back, what was expected. Now revert the sabotage and
watch it go green again. You've just proven the test *can* fail — a test that
passes no matter what the code does is worse than none, because it manufactures
false confidence. This pass-fail-pass check is a habit worth keeping for every
test you ever write, and in [Unit 6](/learn/testing/regression-tests/) it
becomes a formal rule: GopherTrunk doesn't accept a bug-fix test unless it
*failed first* against the unfixed code.

## Step 5 — grow the suite at the boundaries

One test covers the happy path. The
[testing mindset](/learn/testing/the-testing-mindset/) says go hunt the edges —
zero, and something enormous:

```go
func TestFormatMHz_Zero(t *testing.T) {
    if got := FormatMHz(0); got != "0.0000 MHz" {
        t.Errorf("FormatMHz(0) = %q, want %q", got, "0.0000 MHz")
    }
}

func TestFormatMHz_GigahertzRange(t *testing.T) {
    if got := FormatMHz(1_200_000_000); got != "1200.0000 MHz" {
        t.Errorf("FormatMHz(1200000000) = %q, want %q", got, "1200.0000 MHz")
    }
}
```

Three tests, three named behaviors, all running in milliseconds. This is the
loop working software is built in: small edit, `go test`, green, repeat — with
any mistake surfacing seconds after you make it, while the cost curve is still
at its floor.

## The commands you'll actually use

| Command | What it does |
|---------|--------------|
| `go test` | Run the current package's tests |
| `go test ./...` | Run every package below the current directory |
| `go test -v` | Verbose: name each test as it runs |
| `go test -run TestFormatMHz_Zero` | Run only tests matching a name pattern |
| `go test -count=1 ./...` | Bypass Go's test result cache |

That `-run` flag is the day-to-day workhorse in a big repo: while iterating on
one decoder, GopherTrunk developers run a single package —
`go test ./internal/scanner/ccdecoder/...` — and save the full sweep for
[the commit gate](/learn/testing/make-vet-test/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — deliberately breaking the code proves the test can actually detect a failure, so its green means something." markdown="0">
  <p class="knowledge-check__q">Quick check: why break your code on purpose after writing a new test?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">To prove the test can fail — a test that passes regardless is false confidence</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">To practice reading Go compiler errors</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">To make the coverage number go up</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A Go test is a **`TestXxx(t *testing.T)`** function in a **`_test.go`** file —
  no framework, `go test` is the runner.
- Failures speak through **`t.Errorf`** with input/got/want; a quiet run is
  green.
- **Break the code on purpose** once per new test — pass-fail-pass proves the
  test can detect anything at all.
- Grow suites toward the **boundaries**: zero, empty, huge.
- `go test ./...`, `-v`, `-run`: the three flags that carry you through a real
  repo.

Next up: [Table-driven tests](/learn/testing/table-driven-tests/)
