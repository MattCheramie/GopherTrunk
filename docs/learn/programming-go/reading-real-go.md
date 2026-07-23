---
slug: reading-real-go
title: Reading a real Go codebase
description: A guided tour of how a working SDR engine like GopherTrunk is structured in Go — packages, interfaces, and goroutines you can now recognize and read with confidence.
keywords: reading go code, go codebase structure, go project layout, internal packages, go interfaces in practice, gophertrunk source, learn go by reading
level: advanced
status: full
prereq:
  - packages-and-modules
  - interfaces
  - goroutines
faq:
  - q: What is the best way to learn to read a large Go codebase?
    a: "Start at the entry point (the main package under cmd/), follow the top-level flow, and lean on the fact that Gos structure is explicit — package boundaries, capitalized public APIs, and small interfaces at seams. Because every Go project is formatted and organized similarly, the skills transfer: once you can read one, you can read most."
  - q: Why do Go projects put code under an internal directory?
    a: A package under internal/ can only be imported within the same module, so it's private implementation the outside world can't depend on. This lets maintainers reorganize internals freely without breaking anyone. GopherTrunk keeps its engine under internal/ for exactly this reason.
---

# Reading a real Go codebase

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Real Go projects are laid out predictably: an entry point under **`cmd/`**, private
guts under **`internal/`**, packages split by responsibility, **small interfaces** at
the seams, and **goroutines plus channels** carrying work between stages. Everything
in this module shows up in GopherTrunk — this lesson is where the pieces click into a
whole you can read.
</div>

You've met every ingredient. This lesson assembles them by touring how a real Go
program — GopherTrunk, an SDR trunking scanner — is put together, so you can open the
source and actually follow it.

## The standard project shape

Most Go projects, GopherTrunk included, follow a familiar layout:

```text
cmd/            entry points — the main package(s) you build into binaries
internal/       the engine — private packages, one per responsibility
  dsp/          signal processing (filters, mixing, resampling)
  scanner/      following control channels and voice
  sdr/          talking to the radio hardware
proto/          data definitions shared across the program
go.mod          the module and its pinned dependencies
```

`cmd/` holds the `package main` and `func main()` from
[lesson 2](/learn/programming-go/hello-go/); `internal/` holds the
[packages](/learn/programming-go/packages-and-modules/) that can only be imported
within the project. Just reading the directory names tells you where things live.

## Following the flow

Open the entry point and you can trace the whole program:

1. **`main`** parses config and starts the engine — often launching long-running
   work as [goroutines](/learn/programming-go/goroutines/).
2. The **`sdr`** package reads samples off the radio and sends them down a
   [channel](/learn/programming-go/channels/) — a stream of `complex64` I/Q values.
3. The **`dsp`** package receives that stream, then tunes, filters, and demodulates
   it (the whole [DSP module](/learn/dsp/) is this stage in depth).
4. The **`scanner`** package interprets the resulting bits — control-channel
   messages, voice frames — and drives what the user sees.

Each stage is a package with a small public API; the stages hand data to each other
over channels, exactly the *share-memory-by-communicating* model from Unit 3.

## Interfaces at the seams

Where two packages meet, you'll usually find a small
[interface](/learn/programming-go/interfaces/). A demodulator, a sample source, or a
sink is defined by the one or two methods it must have — so the scanner can be tested
with a fake source, and a new demodulator can drop in without touching its callers.
Spotting these interfaces is the key to understanding *why* the code is split the way
it is.

## Reading tips that always work

- **Start at `main`** and follow the calls outward; don't read files alphabetically.
- **Capitals are the API.** An exported (capitalized) name is meant to be used by
  other packages; lowercase is internal detail you can skim.
- **Follow the channels.** Data flowing over a channel marks the boundary between
  concurrent stages.
- **Run the tests.** `_test.go` files are worked examples of how each package is
  meant to be used — often the fastest way to understand an API.

Because Go is formatted and organized so consistently, these habits carry from
GopherTrunk to almost any Go project you'll ever open.

<div class="knowledge-check" data-quiz data-correct-msg="Right — cmd/ holds the main package(s) you build into binaries." markdown="0">
  <p class="knowledge-check__q">Quick check: in a typical Go project, where do you find the program's entry point?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">In the internal/ directory</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">In a main package under cmd/</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">In go.mod</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Real Go projects share a shape: **`cmd/`** entry points, **`internal/`** engine,
  packages by responsibility.
- Follow the program from **`main`** outward; **capitalized** names are the public
  API.
- Stages hand work over **channels**; **small interfaces** sit at package seams.
- These reading habits transfer from GopherTrunk to nearly any Go codebase.

Next up: where to go from here to keep growing as a Go developer.
