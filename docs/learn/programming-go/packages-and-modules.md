---
slug: packages-and-modules
title: Packages & modules
description: How Go organizes code into packages and modules — imports, the go.mod file, and how visibility is controlled by capitalization rather than keywords.
keywords: go packages, go modules, go.mod, go imports, exported identifiers, package visibility, go get, dependency management
level: intermediate
status: full
prereq:
  - interfaces
faq:
  - q: What is the difference between a package and a module?
    a: A package is a single directory of Go files that are compiled together and imported as a unit. A module is a collection of packages versioned together, defined by a go.mod file at its root. You import packages; you version and distribute modules. A small project is one module containing several packages.
  - q: How does Go decide what is public?
    a: By capitalization, not keywords. An identifier — a function, type, field, or variable — whose name starts with an uppercase letter is exported and visible to other packages. A lowercase name is private to its own package. There is no public or private keyword; the first letter says it all.
---

# Packages & modules

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **package** is a directory of Go files compiled together; a **module** is a
versioned collection of packages defined by a **`go.mod`** file. You **import**
packages by path, and **capitalization controls visibility** — an `Uppercase` name
is exported, a `lowercase` name is private to its package. No public/private
keywords needed.
</div>

Once a program grows past one file, you need structure. This lesson covers how Go
organizes code and manages the libraries it depends on.

## Packages: the unit of code

Every Go file starts with a `package` line, and all files in a directory belong to
the same package. You pull one package into another with `import`:

```go
package scanner

import (
    "fmt"
    "github.com/mattcheramie/gophertrunk/internal/dsp"
)
```

Standard-library packages are imported by short name (`fmt`, `os`, `net`);
third-party and internal packages by their full path. GopherTrunk is organized this
way — `internal/dsp`, `internal/scanner`, and so on — each a package with a clear
job.

## Visibility by capitalization

Go has no `public` or `private` keyword. Instead, the **first letter** of a name
decides:

```go
func Decode(...)   // exported — other packages can call it
func decode(...)   // unexported — visible only inside this package
```

The same rule applies to types, struct fields, constants, and variables. It makes
the public surface of a package obvious at a glance: capitals are the API, lowercase
is the implementation.

## Modules: versioned dependencies

A **module** is the unit you version and share. Its `go.mod` file names the module
and pins every dependency:

```text
module github.com/mattcheramie/gophertrunk

go 1.22

require (
    github.com/some/library v1.4.2
)
```

When you import a package you don't yet have, `go get` or `go mod tidy` fetches it
and records the exact version, and `go.sum` locks its checksum. A fresh checkout
therefore builds with *identical* dependencies for everyone — no "works on my
machine" surprises. That reproducibility is what makes Go projects easy to
[build in CI](/learn/deployment/ci-cd-pipelines/) and to
[ship as one binary](/learn/programming-go/hello-go/).

## The internal directory

A package under a directory named `internal/` can only be imported by code in the
same module. GopherTrunk uses this deliberately: its guts live under `internal/`, so
they're private implementation the rest of the world can't depend on, leaving the
maintainers free to change them.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an uppercase initial means the name is exported." markdown="0">
  <p class="knowledge-check__q">Quick check: a function named <code>decode</code> (lowercase d) is…</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">private to its own package</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">exported to every other package</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">a syntax error — names must be capitalized</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **package** is a directory of files compiled together; you **import** it by path.
- A **module** (`go.mod`) versions a set of packages and pins dependencies for
  reproducible builds.
- **Capitalization** controls visibility — `Uppercase` is exported, `lowercase` is
  private.
- The **`internal/`** directory hides packages from outside the module.

Next up: Go's core collections — slices and maps — plus generics.
