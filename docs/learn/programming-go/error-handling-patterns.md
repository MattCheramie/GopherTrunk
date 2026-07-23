---
slug: error-handling-patterns
title: Error-handling patterns
description: "Beyond if err != nil — wrapping errors with %w, sentinel errors, errors.Is and errors.As, and custom error types for programs that fail informatively."
keywords: go error handling, fmt.Errorf %w, errors.Is, errors.As, sentinel error, wrapped error, custom error type go, error wrapping
level: advanced
status: full
prereq:
  - functions-and-errors
faq:
  - q: What does %w do in fmt.Errorf?
    a: "It wraps an underlying error inside a new one while preserving the original for later inspection. fmt.Errorf(\"open capture: %w\", err) adds context to the message and keeps err reachable, so errors.Is and errors.As can still find it further up the stack."
  - q: When do I use errors.Is versus errors.As?
    a: "Use errors.Is to test whether an error chain contains a specific sentinel value, like errors.Is(err, io.EOF). Use errors.As to pull a specific error type out of the chain so you can read its fields, like errors.As(err, &pathErr). Is asks which value; As asks which type."
---

# Error-handling patterns

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Go returns errors as values, and real programs add structure. **Wrap** an error with
**`fmt.Errorf("...: %w", err)`** to add context while keeping the original. Compare
against a **sentinel** with **`errors.Is`**, and extract a **custom error type** with
**`errors.As`**. Define your own error types when callers need more than a message.
These patterns turn `if err != nil` into errors that are useful, not just present.
</div>

The [functions & errors](/learn/programming-go/functions-and-errors/) lesson
introduced the `error` interface. This one covers the patterns that make errors carry
real information as they travel up a call stack.

## Wrapping with %w

As an error bubbles up, each layer should add context — where it happened — without
discarding the cause. The `%w` verb wraps:

```go
func loadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("load config %s: %w", path, err)
    }
    // ...
}
```

The message becomes `load config scan.json: open scan.json: no such file or
directory` — a breadcrumb trail — and the wrapped error stays reachable underneath.
Use `%w` (wrap) when callers may inspect the cause; use `%v` (format) when you only
want the text.

## Sentinel errors and errors.Is

A **sentinel** is a named error value that callers can test against:

```go
var ErrNoLock = errors.New("decoder: no signal lock")

func decode(f Frame) error {
    if f.SNR < 6 {
        return ErrNoLock
    }
    return nil
}

if errors.Is(err, ErrNoLock) {   // matches even through wrapping
    retry()
}
```

`errors.Is` walks the whole wrapped chain, so a deeply wrapped `ErrNoLock` is still
found. That is why you test with `errors.Is` rather than `err == ErrNoLock`.

## Custom error types and errors.As

When callers need *data* about the failure — not just its identity — define a type
that implements `error`:

```go
type DecodeError struct {
    FreqHz float64
    Symbol int
}

func (e *DecodeError) Error() string {
    return fmt.Sprintf("decode failed at %.3f MHz, symbol %d", e.FreqHz/1e6, e.Symbol)
}
```

Pull it back out with `errors.As`, which finds the first error of that type in the
chain and assigns it:

```go
var de *DecodeError
if errors.As(err, &de) {
    log.Printf("bad symbol %d at %.3f MHz", de.Symbol, de.FreqHz/1e6)
}
```

| Tool | Question it answers | Example |
|------|--------------------|---------|
| `fmt.Errorf("...: %w", err)` | How do I add context? | `open capture: %w` |
| sentinel + `errors.Is` | Is this *that* error? | `errors.Is(err, io.EOF)` |
| custom type + `errors.As` | What are the details? | `errors.As(err, &de)` |

## When to wrap, when to handle

Not every error needs wrapping. A rule of thumb: **handle** an error where you can do
something about it (retry, default, log-and-continue), and **wrap-and-return** it
everywhere else so the layer that can act still has the context it needs. Wrapping
once per boundary keeps messages informative without turning them into noise.

<div class="knowledge-check" data-quiz data-correct-msg="Right — %w wraps the cause and keeps it reachable for errors.Is/As." markdown="0">
  <p class="knowledge-check__q">Quick check: what does wrapping with <em>%w</em> preserve that <em>%v</em> does not?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The stack trace</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The underlying error, so errors.Is/As can still find it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The line number where it occurred</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Wrap** with **`fmt.Errorf("...: %w", err)`** to add context and keep the cause.
- Define a **sentinel** and test it with **`errors.Is`** (works through wrapping).
- Define a **custom error type** and extract it with **`errors.As`** for details.
- **Handle** where you can act; **wrap-and-return** everywhere else.

Next up: proving your code works with Go's built-in testing package.
