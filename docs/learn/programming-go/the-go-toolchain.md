---
slug: the-go-toolchain
title: The Go toolchain
description: A tour of Go's batteries-included command-line tools — go build, run, test, fmt, and vet — the zero-configuration toolchain that formats, checks, tests, and builds your code.
keywords: go toolchain, go build, go test, gofmt, go vet, go tools, go command, go run
level: beginner
status: full
prereq:
  - hello-go
faq:
  - q: Do I need a build system like Make for Go?
    a: Usually not. The `go` command builds, tests, formats, and manages dependencies on its own, with no separate build file. Larger projects sometimes add a Makefile to bundle common commands (GopherTrunk has one), but that is a convenience wrapper around the same `go` tools, not a requirement.
  - q: What is the difference between gofmt and go vet?
    a: "`gofmt` is about *style* — it rewrites your code into Go's one official layout. `go vet` is about *correctness* — it inspects your code for suspicious constructs that compile but are probably bugs, like a format string that doesn't match its arguments. Formatting is cosmetic; vet catches real mistakes."
---

# The Go toolchain

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The single **`go`** command is a whole toolchain: **`go build`** and **`go run`**
compile, **`go test`** runs tests, **`gofmt`** enforces one official style, **`go
vet`** flags likely bugs, and **`go mod`** manages dependencies — all with **zero
configuration**. Learning these five commands is most of what day-to-day Go work
needs.
</div>

Most languages leave formatting, testing, and dependency management to third-party
tools you have to choose and configure. Go ships them in the box. This lesson is a
quick tour so the commands feel familiar before you meet the language features
they support.

## The commands you'll use every day

| Command | What it does |
|---------|--------------|
| `go run main.go` | Compile and immediately run — for iterating |
| `go build` | Compile to a keepable binary |
| `go test ./...` | Run every test in the module |
| `gofmt -w .` | Reformat files into the one official style |
| `go vet ./...` | Report suspicious, likely-buggy code |
| `go mod tidy` | Add missing and remove unused dependencies |

The `./...` you see on some of these means "this directory and everything below
it" — a handy way to act on a whole project at once.

## One style, settled

Run `gofmt` and arguments about tabs, spaces, and brace placement simply end —
there is one correct layout and the tool produces it. Because every Go project is
formatted the same way, code you have never seen still reads like your own. Most
editors run `gofmt` (or its cousin `goimports`, which also fixes import lines) every
time you save.

## Catching bugs before they run

`go vet` reads your code looking for mistakes the compiler accepts but that are
almost certainly wrong — a `Printf` whose placeholders don't match its arguments, a
lock copied by value, an unreachable branch. It is fast and worth running
constantly. GopherTrunk's own build gate is literally `make vet test`: vet plus the
full test suite must pass before any commit. You will meet the testing half in
[Testing in Go](/learn/programming-go/testing-in-go/).

## Modules and dependencies

A Go project is a **module**, defined by a `go.mod` file at its root that names the
module and lists its dependencies with exact versions. When you import a package you
don't have, `go mod tidy` fetches it and records the version, so a fresh checkout
builds identically for everyone. You'll go deeper in
[Packages & modules](/learn/programming-go/packages-and-modules/).

```text
$ go mod tidy      # sync dependencies with what the code imports
$ go build ./...   # build everything
$ go vet ./...     # check everything
$ go test ./...    # test everything
```

Those four lines are, essentially, a full Go CI pipeline — which is why wiring Go
into [continuous integration](/learn/deployment/ci-cd-pipelines/) is so painless.

<div class="knowledge-check" data-quiz data-correct-msg="Right — vet looks for likely bugs; gofmt only reformats." markdown="0">
  <p class="knowledge-check__q">Quick check: which tool would warn you that a <code>Printf</code> call's placeholders don't match its arguments?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">gofmt</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">go vet</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">go mod tidy</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The one **`go`** command builds, runs, tests, formats, vets, and manages
  dependencies — **no separate build system required**.
- **`gofmt`** gives every Go project the same style; **`go vet`** flags likely bugs.
- A project is a **module** (`go.mod`) with pinned dependency versions.
- `build`, `vet`, and `test` over `./...` act on the whole project at once.

Next up: the values and types that Go programs are built from.
