---
slug: project-structure
title: Structuring a Go project
description: "The cmd, internal, and package conventions that shape real Go repositories — where code goes and why, using GopherTrunk's own layout as the map."
keywords: go project structure, cmd directory, internal package, go project layout, package per responsibility, go module layout, organizing go code
level: intermediate
status: full
prereq:
  - packages-and-modules
faq:
  - q: What is the internal directory for in a Go project?
    a: "internal is a special directory the Go toolchain enforces: packages under internal/ can only be imported by code inside the same module. It's how a project keeps its implementation packages private, exposing a deliberate surface while forbidding outside code from depending on internals it might break."
  - q: Why put each command under cmd/?
    a: "A repository often ships more than one binary — the main app, a calibration tool, a test harness. Giving each its own package under cmd/ keeps every main() small and separate, while the real logic lives in importable packages under internal/ that all the commands can share."
---

# Structuring a Go project

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Real Go repositories follow a few conventions. Each executable gets a tiny `main`
package under **`cmd/`**. Private implementation packages live under **`internal/`**,
which the compiler forbids outside code from importing. Each package owns **one
responsibility** and is named for it. GopherTrunk's own tree is a textbook example —
`cmd/gophertrunk` plus `internal/dsp`, `internal/scanner`, `internal/sdr`, and more.
</div>

Where does code go? Go has strong conventions, and following them makes any project
navigable. This lesson maps them onto GopherTrunk's real layout, extending
[Packages & modules](/learn/programming-go/packages-and-modules/).

## cmd/ — one directory per binary

Every executable is a `package main` with a `main()` function, and each lives in its
own directory under `cmd/`. GopherTrunk ships several:

```
cmd/
  gophertrunk/      # the main scanner binary
  voice-calibrate/  # a calibration helper
  encodetest/       # a codec test harness
```

Each `main` stays thin — parse flags, wire things up, call into the real packages.
`go build ./cmd/gophertrunk` builds just that one.

## internal/ — enforced-private packages

The bulk of the code lives under `internal/`, and the toolchain gives that name a
superpower: **nothing outside the module can import it**. That lets the project expose
a deliberate surface while keeping implementation free to change:

```
internal/
  dsp/       # signal processing — filters, down-converters
  scanner/   # control-channel decode, trunking follow
  sdr/       # radio hardware drivers
  config/    # loading and validating configuration
  api/       # the HTTP/JSON surface
```

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 170" role="img" aria-label="A tree with the module root at top, cmd and internal beneath it, and package directories under each." xmlns="http://www.w3.org/2000/svg">
  <rect x="200" y="10" width="120" height="28" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="260" y="29" text-anchor="middle" font-size="11" fill="currentColor">module root</text>
  <rect x="90" y="70" width="120" height="28" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="150" y="89" text-anchor="middle" font-size="11" fill="currentColor">cmd/</text>
  <rect x="310" y="70" width="120" height="28" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="370" y="89" text-anchor="middle" font-size="11" fill="currentColor">internal/</text>
  <line x1="240" y1="38" x2="150" y2="70" stroke="currentColor" stroke-width="1.5"/>
  <line x1="280" y1="38" x2="370" y2="70" stroke="currentColor" stroke-width="1.5"/>
  <text x="150" y="130" text-anchor="middle" font-size="10" fill="currentColor">gophertrunk/  voice-calibrate/</text>
  <text x="370" y="125" text-anchor="middle" font-size="10" fill="currentColor">dsp/  scanner/</text>
  <text x="370" y="142" text-anchor="middle" font-size="10" fill="currentColor">sdr/  config/  api/</text>
  <line x1="150" y1="98" x2="150" y2="118" stroke="currentColor" stroke-width="1.5"/>
  <line x1="370" y1="98" x2="370" y2="113" stroke="currentColor" stroke-width="1.5"/>
</svg>
<figcaption>The conventional Go tree: thin binaries under cmd/, private implementation packages under internal/, each owning one concern.</figcaption>
</figure>

## One package, one responsibility

Notice the pattern in those names: each package is a single concern, named for what it
does — `dsp` does signal processing, `sdr` talks to hardware, `config` handles
configuration. This keeps dependencies flowing one way (a command imports `scanner`,
`scanner` imports `dsp` and `sdr`) and makes the whole tree readable.

| Directory | Holds | Importable by |
|-----------|-------|---------------|
| `cmd/<name>/` | One `main` per binary | (it's the top; imports others) |
| `internal/<pkg>/` | Private implementation | Only this module |
| `pkg/` (if present) | Deliberately public API | Anyone |
| module root | `go.mod`, `go.sum`, top-level docs | — |

## Applying it

When you start a project, resist one giant package. Put each binary under `cmd/`, put
shared logic in small `internal/` packages named by responsibility, and let the
`internal` boundary keep your public surface honest. Reading a codebase laid out this
way — which is exactly what the
[Reading a real Go codebase](/learn/programming-go/reading-real-go/) lesson does with
GopherTrunk — becomes a matter of following the imports.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the compiler blocks any import of internal/ from outside the module." markdown="0">
  <p class="knowledge-check__q">Quick check: what's special about a package under <em>internal/</em>?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It's compiled with extra optimizations</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Only code in the same module may import it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It must contain the main function</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Each binary is a thin **`package main`** under **`cmd/<name>/`**.
- **`internal/`** packages are private — only the same module can import them.
- Each package owns **one responsibility** and is named for it.
- GopherTrunk's `cmd/` + `internal/dsp`, `scanner`, `sdr`, `config` tree is the model.

Next up: the conventions that make Go code idiomatic — from naming to error handling.
