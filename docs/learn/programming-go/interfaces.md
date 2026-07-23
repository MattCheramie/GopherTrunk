---
slug: interfaces
title: Interfaces & composition
description: Small interfaces, implicit satisfaction, and composition over inheritance — the Go idea that lets unrelated types share behaviour without a class hierarchy.
keywords: go interfaces, implicit interface, duck typing go, io reader writer, composition over inheritance, go polymorphism, empty interface
level: intermediate
status: full
prereq:
  - structs-and-methods
faq:
  - q: How does a type implement an interface in Go?
    a: Implicitly. There is no "implements" keyword. If a type has all the methods an interface lists, it satisfies that interface automatically. This means you can write an interface for types that already exist — even types in libraries you don't control — as long as they have the right methods.
  - q: Why are Go interfaces usually small?
    a: Small interfaces (often one method, like io.Reader's Read) are easy to satisfy and easy to reuse. A one-method interface can be implemented by countless types, so functions that accept it work with all of them. Go's standard library is full of these tiny, composable interfaces, and idiomatic Go follows the same habit.
---

# Interfaces & composition

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **interface** is a set of method signatures. In Go a type satisfies an interface
**implicitly** — just by having those methods, with no `implements` keyword. Small,
one-method interfaces like **`io.Reader`** are the norm, letting unrelated types be
used interchangeably. Interfaces are how Go gets flexibility and polymorphism
**without inheritance**.
</div>

Interfaces are the single most important idea for reading real Go code. This lesson
explains what they are and why Go's "implicit" twist makes them so powerful.

## What an interface is

An interface names behaviour, not data:

```go
type Demodulator interface {
    Demodulate(samples []complex128) []byte
}
```

Anything with a `Demodulate(samples []complex128) []byte` method *is* a
`Demodulator`. A function can now accept the interface and work with any concrete
type that fits:

```go
func decode(d Demodulator, samples []complex128) []byte {
    return d.Demodulate(samples)
}
```

Pass a `C4FMDemod`, an `FMDemod`, or a test fake — `decode` doesn't care, as long as
it has the method.

## Implicit satisfaction

Here is the Go twist: a type never *declares* that it implements an interface. If it
has the methods, it satisfies the interface — automatically.

```go
type C4FMDemod struct{}

func (C4FMDemod) Demodulate(s []complex128) []byte { /* ... */ return nil }
// C4FMDemod is now a Demodulator, with no extra ceremony.
```

This lets you define an interface *after the fact* to describe types you didn't
write — including ones from libraries. It keeps packages loosely coupled: the
demodulator doesn't need to import the interface, and the interface doesn't need to
know about the demodulator.

## Keep interfaces small

The most reused interface in Go has one method:

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}
```

`io.Reader` is satisfied by files, network connections, byte buffers, gzip streams,
and more — so any function that reads from an `io.Reader` works with all of them,
including an SDR sample source. The lesson from the standard library: **small
interfaces compose**. A one-method interface is easy to satisfy and endlessly
reusable; a giant interface is neither.

## Composition, one more time

Interfaces embed too. `io.ReadWriter` is simply:

```go
type ReadWriter interface {
    Reader
    Writer
}
```

Between struct embedding (last lesson) and interface embedding (here), Go builds
everything out of small parts combined — never deep inheritance trees. When you read
GopherTrunk in [Reading a real Go codebase](/learn/programming-go/reading-real-go/),
this pattern of small interfaces at package boundaries is everywhere.

<div class="knowledge-check" data-quiz data-correct-msg="Right — in Go, having the methods is enough; satisfaction is implicit." markdown="0">
  <p class="knowledge-check__q">Quick check: how does a Go type declare that it implements an interface?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">With an "implements" keyword</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It doesn't — having the required methods is enough</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">By registering with the interface at startup</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **interface** is a set of method signatures describing behaviour.
- Go satisfies interfaces **implicitly** — no `implements`, just having the methods.
- **Small interfaces** (like `io.Reader`) are the norm and compose beautifully.
- Interfaces give Go **polymorphism without inheritance**.

Next up: Go's headline feature — concurrency with goroutines.
