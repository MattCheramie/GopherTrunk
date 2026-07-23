---
slug: structs-and-methods
title: Structs & methods
description: Group related data with Go structs, attach behaviour with methods, and understand pointer versus value receivers — Go's lightweight alternative to classes.
keywords: go structs, go methods, pointer receiver, value receiver, go struct fields, composition, go objects
level: intermediate
status: full
prereq:
  - functions-and-errors
faq:
  - q: Does Go have classes?
    a: No. Go has structs (bundles of named fields) and methods (functions attached to a type). Together they cover most of what classes do, but there is no inheritance. Instead, Go favours composition — embedding one struct inside another — and interfaces for shared behaviour.
  - q: When do I use a pointer receiver versus a value receiver?
    a: Use a pointer receiver (`func (r *Receiver)`) when the method needs to modify the struct, or when the struct is large and you want to avoid copying it. Use a value receiver when the method only reads and the struct is small. A common rule of thumb is to be consistent within a type — if any method needs a pointer receiver, use pointer receivers for all of them.
---

# Structs & methods

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **struct** groups related fields into one type; a **method** is a function
attached to a type via a **receiver**. A **pointer receiver** (`*T`) lets a method
modify the struct or avoid copying it; a **value receiver** (`T`) works on a copy.
Go has **no classes and no inheritance** — structs plus methods, composed together,
do the job.
</div>

Real programs need to bundle data — a radio channel has a frequency, a bandwidth, a
name. This lesson introduces structs to hold that data and methods to give it
behaviour.

## Defining a struct

```go
type Channel struct {
    Name      string
    FreqHz    float64
    Bandwidth float64
}
```

Create one and read its fields with dot notation:

```go
cc := Channel{Name: "P25 CC", FreqHz: 851.0125e6, Bandwidth: 12500}
fmt.Println(cc.Name, cc.FreqHz)
```

Field names starting with a **capital letter** are *exported* — visible outside the
package. Lowercase fields are private to the package (more in
[Packages & modules](/learn/programming-go/packages-and-modules/)).

## Attaching methods

A method is a function with a **receiver** — the type it belongs to, written before
the name:

```go
func (c Channel) Wavelength() float64 {
    return 3e8 / c.FreqHz
}

fmt.Println(cc.Wavelength())
```

Here `c Channel` is a **value receiver**: the method gets a copy of the channel and
can only read it.

## Pointer vs value receivers

When a method needs to *change* the struct, it takes a **pointer receiver**:

```go
func (c *Channel) Retune(freqHz float64) {
    c.FreqHz = freqHz   // modifies the original, not a copy
}
```

| Receiver | Written | Sees | Use when |
|----------|---------|------|----------|
| Value | `func (c Channel)` | a copy | read-only, small structs |
| Pointer | `func (c *Channel)` | the original | modifying, or large structs |

Call it the same way — `cc.Retune(852e6)` — Go takes the address for you. The rule
of thumb: if any method on a type needs a pointer receiver, give them all pointer
receivers for consistency.

## Composition instead of inheritance

Go has no subclasses. To reuse a struct, you **embed** it:

```go
type TrunkedChannel struct {
    Channel        // embedded — TrunkedChannel gets Channel's fields and methods
    SystemID int
}
```

A `TrunkedChannel` now has `Name`, `FreqHz`, and `Wavelength()` directly, plus its
own `SystemID`. This "has-a" composition is Go's answer to the "is-a" hierarchies of
class-based languages — simpler, and it pairs naturally with the interfaces you'll
meet next.

<div class="knowledge-check" data-quiz data-correct-msg="Right — modifying the struct requires a pointer receiver." markdown="0">
  <p class="knowledge-check__q">Quick check: a method needs to change a field of its struct. Which receiver does it need?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A value receiver (T)</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A pointer receiver (*T)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Either works identically</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **struct** bundles named **fields**; **capitalized** fields are exported.
- A **method** attaches behaviour to a type through a **receiver**.
- **Pointer receivers** modify the original or avoid copying; **value receivers**
  work on a copy.
- Go uses **composition (embedding)**, not inheritance, to reuse types.

Next up: interfaces — the idea that makes Go code flexible without class hierarchies.
