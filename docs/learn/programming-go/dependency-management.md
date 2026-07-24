---
slug: dependency-management
title: Dependency management
description: "How go.mod and go.sum pin versions, how go get and go mod tidy manage dependencies, and why reproducible builds and a small dependency tree matter."
keywords: go modules, go.mod, go.sum, go get, go mod tidy, semantic versioning, go dependencies, vendoring, reproducible build
level: intermediate
status: full
prereq:
  - packages-and-modules
faq:
  - q: What's the difference between go.mod and go.sum?
    a: "go.mod lists your module's path, its Go version, and the versions of the dependencies you require — it's the human-readable declaration. go.sum records a cryptographic checksum for every module version, so the toolchain can verify that what it downloads is byte-for-byte what you locked. go.mod says what; go.sum proves it hasn't changed."
  - q: What does go mod tidy do?
    a: "It reconciles go.mod and go.sum with what your code actually imports — adding any dependency you use but haven't declared, and removing any you declared but no longer import. Running it keeps the module files honest, which is why it's a common pre-commit step."
---

# Dependency management

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A module's dependencies live in **`go.mod`** (what versions you require) and
**`go.sum`** (checksums that prove those versions haven't changed). **`go get`** adds
or upgrades a dependency; **`go mod tidy`** syncs the files to what your code actually
imports. Versions follow **semantic versioning**, and the checksums give you
**reproducible builds** — everyone compiles the exact same code.
</div>

Every non-trivial program uses code someone else wrote. This lesson covers how Go
tracks that code, building on
[Packages & modules](/learn/programming-go/packages-and-modules/).

## go.mod and go.sum

A module is a tree of packages with a `go.mod` at its root. It declares the module
path, the Go version, and each dependency's version:

```
module github.com/example/gophertrunk

go 1.22

require (
    github.com/some/dsp v1.4.2
    golang.org/x/sync v0.7.0
)
```

Alongside it, `go.sum` stores a checksum for every module version the build uses.
When the toolchain downloads a dependency, it verifies the bytes against `go.sum` —
so a tampered or changed release fails the build instead of silently slipping in.

## Adding and updating dependencies

You rarely edit `go.mod` by hand. The commands do it:

| Command | What it does |
|---------|-------------|
| `go get github.com/some/dsp@v1.5.0` | Add or move to a specific version |
| `go get -u ./...` | Upgrade dependencies to newer minor/patch versions |
| `go mod tidy` | Add missing and drop unused dependencies |
| `go mod download` | Fetch everything `go.mod` lists into the cache |
| `go mod verify` | Check the cache matches `go.sum` |

Just importing a package and running `go mod tidy` is the usual way to pick up a new
dependency — the tool finds the import, resolves a version, and records both files.

## Semantic versioning

Go leans hard on **semver** — `vMAJOR.MINOR.PATCH`. A patch or minor bump promises
backward compatibility; a **major** bump (v2+) is a breaking change and, by Go's
import-compatibility rule, even changes the import path (`.../dsp/v2`). This is what
lets `go get -u` safely pull bug fixes without breaking your build.

## Why a small dependency tree matters

Every dependency is code you now ship, trust, and must keep updated. Go's culture —
and GopherTrunk's — favours a **lean** tree: reach for the
[standard library](/learn/programming-go/the-standard-library/) first, add a
dependency only when it earns its place. Fewer dependencies means faster builds, a
smaller attack surface, and less breakage when you upgrade. For fully offline or
audited builds, `go mod vendor` copies dependencies into a `vendor/` directory that
the build uses instead of the network.

<div class="knowledge-check" data-quiz data-correct-msg="Right — go.sum holds checksums that verify each dependency's bytes." markdown="0">
  <p class="knowledge-check__q">Quick check: what is the job of the go.sum file?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It lists which packages your code imports</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It records checksums so downloaded dependencies can be verified</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It stores your module's own version number</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **`go.mod`** declares required dependency versions; **`go.sum`** stores checksums to
  verify them.
- **`go get`** adds/upgrades; **`go mod tidy`** syncs the files to real imports.
- Versions follow **semantic versioning**; a major bump changes the import path.
- Prefer a **small** dependency tree; `go mod vendor` supports offline builds.

Next up: error-handling patterns beyond the basic `if err != nil`.
