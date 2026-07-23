---
slug: idiomatic-go
title: Writing idiomatic Go
description: "The conventions that make Go code read the same everywhere — gofmt, accept interfaces and return structs, handle errors explicitly, and avoid cleverness."
keywords: idiomatic go, effective go, gofmt, accept interfaces return structs, go naming conventions, handle errors go, go best practices, go style
level: intermediate
status: full
prereq:
  - interfaces
faq:
  - q: What does "accept interfaces, return structs" mean?
    a: "A function should take the smallest interface it needs as a parameter, so any type with those methods can be passed in — maximum flexibility for callers. But it should return a concrete struct, so callers get the full type with all its methods and fields, not a narrowed-down interface. Be liberal in what you accept, specific in what you return."
  - q: Is there an official Go style guide?
    a: "gofmt is the style guide — it settles all formatting mechanically, so there are no arguments about braces or spacing. For the deeper conventions, Effective Go and the Go Code Review Comments wiki are the canonical references the whole community follows."
---

# Writing idiomatic Go

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Idiomatic Go is remarkably uniform because so much is settled for you. **`gofmt`**
formats every file the same way. Functions **accept interfaces and return structs**.
Errors are **handled explicitly**, never swallowed. Names are **short in short
scopes** and clear at package boundaries. The overriding value is **clarity over
cleverness** — code that reads plainly. *Effective Go* is the canonical guide.
</div>

Go code from different authors looks strikingly similar, and that's by design. This
final craft lesson gathers the conventions that make it so, drawing on everything from
[interfaces](/learn/programming-go/interfaces/) to
[error handling](/learn/programming-go/error-handling-patterns/).

## gofmt ends the style debate

There is one correct formatting, and `gofmt` (or `go fmt`) produces it — indentation,
alignment, import grouping, all mechanical. Nobody argues about braces in Go because
the tool decides. Run it on save; CI checks it. Formatting is simply not a thing you
think about.

## Accept interfaces, return structs

The guiding shape for APIs: take the **smallest interface** you need as a parameter,
return a **concrete type**:

```go
// Accepts any reader — file, socket, buffer.
func Decode(r io.Reader) (*Frame, error) { ... }

// Returns a concrete *Receiver, not an interface — callers get everything.
func NewReceiver(rate int) *Receiver { ... }
```

Accepting an interface makes the function flexible for callers; returning a struct
gives them the full type without guessing what interface to satisfy. This is the
single most Go-shaping habit for library code.

## Handle every error, explicitly

Idiomatic Go never ignores an error silently. Handle it, wrap it, or return it —
`_ = f()` on an error is a code smell:

```go
if err := cfg.Validate(); err != nil {
    return fmt.Errorf("invalid config: %w", err)
}
```

The verbosity is the point: every failure path is visible on the page, which is why Go
programs tend to fail informatively rather than crash mysteriously.

## Names: short scope, short name

Go names scale with scope. A loop index is `i`; a receiver is one or two letters; a
package-level exported function has a full, clear name. Long names in tiny scopes are
noise.

| Scope | Convention | Example |
|-------|-----------|---------|
| Loop / short block | Single letters | `i`, `r`, `s` |
| Method receiver | 1–2 letters, consistent | `func (r *Receiver)` |
| Package function/type | Clear, no stutter | `scanner.New`, not `scanner.NewScanner` |
| Exported identifier | CamelCase, documented | `Demodulate`, `FreqHz` |

Note the anti-stutter rule: it's `bytes.Buffer`, not `bytes.BytesBuffer` — the package
name already carries context.

## Clarity over cleverness

The through-line of every rule above is that Go optimizes for the reader. Prefer the
plain loop to the clever one-liner; prefer an obvious name to a witty one; prefer a
few more lines to a dense trick. Code is read far more than it's written, and Go's
culture — captured in **Effective Go** and the Go Code Review Comments — bakes that
into how the whole community writes. That shared taste is what lets you drop into
GopherTrunk, or any Go project, and read it in the
[next lesson](/learn/programming-go/reading-real-go/) as if you'd written it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — accept the smallest interface, return the concrete struct." markdown="0">
  <p class="knowledge-check__q">Quick check: what does "accept interfaces, return structs" advise?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Take concrete types in, return interfaces out</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Take the smallest interface you need, return the concrete type</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Never use interfaces in public APIs</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **`gofmt`** settles all formatting — style is never a debate.
- **Accept interfaces, return structs** for flexible, honest APIs.
- **Handle every error explicitly** — never swallow it.
- Names are **short in short scopes**, clear at boundaries, and avoid stutter.
- Above all, favour **clarity over cleverness** — see *Effective Go*.

Next up: putting it all together by reading a real Go codebase — GopherTrunk itself.
