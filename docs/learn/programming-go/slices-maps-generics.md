---
slug: slices-maps-generics
title: Slices, maps & generics
description: Go's core collections — slices and maps, how they grow and alias — plus type parameters (generics) for writing reusable, type-safe code.
keywords: go slices, go maps, go generics, type parameters, append, make, slice aliasing, go collections
level: intermediate
status: full
prereq:
  - values-and-types
faq:
  - q: What is the difference between an array and a slice in Go?
    a: An array has a fixed length that is part of its type ([4]int), so it rarely appears directly. A slice is a flexible, growable view into an underlying array — it has a length and a capacity and is what you use in practice. You create slices with literals or make, and grow them with append.
  - q: When did Go get generics, and do I need them?
    a: Generics arrived in Go 1.18 (2022). You don't need them for everyday code — slices and maps already work with any type — but they let you write one function or data structure that works across many types safely, instead of duplicating it per type or falling back to the empty interface. Reach for them when you'd otherwise copy-paste the same logic for int, string, and float64.
---

# Slices, maps & generics

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **slice** is Go's growable list — a view into an array you extend with
**`append`**. A **map** is a key/value lookup table. Both are created with **`make`**
and are used constantly. **Generics** (type parameters) let you write one function
that works for many types safely — added in Go 1.18.
</div>

Almost every program stores collections of things. This lesson covers Go's two core
containers and the generics that make reusable code type-safe.

## Slices: the everyday list

A **slice** holds an ordered, growable sequence:

```go
var samples []float64            // an empty slice
samples = append(samples, 0.5)   // grow it
samples = append(samples, 0.7, 0.9)

first := samples[0]              // index from 0
window := samples[1:3]           // a sub-slice: elements 1 and 2
```

`append` returns a (possibly new) slice, so you always assign its result back. A
slice is a lightweight view over a backing array, which is why passing one to a
function is cheap — but also why two slices can share storage.

## A slice aliasing gotcha

Because a slice points into a shared array, sub-slices can *alias* each other:

```go
a := []int{1, 2, 3, 4}
b := a[0:2]      // b shares storage with a
b[0] = 99        // this also changes a[0]!
```

This is efficient but occasionally surprising — when you need an independent copy,
use the built-in `copy`. Keeping this in mind matters in signal code, where large
sample buffers are sliced constantly.

## Maps: key/value lookup

A **map** associates keys with values:

```go
names := make(map[int]string)
names[101] = "Dispatch"          // talkgroup 101
names[102] = "Fireground"

n, ok := names[101]              // n = "Dispatch", ok = true
_, ok = names[999]               // ok = false: key absent
```

The two-value form (`value, ok`) tells you whether the key was present — the idiom
for "look it up, and tell me if it exists." Maps are unordered; range over one and
the order is deliberately randomized.

## Generics: one function, many types

Before generics, a function that summed a slice had to be written once per numeric
type or fall back to the untyped empty interface. **Type parameters** fix that:

```go
func Max[T int | float64](xs []T) T {
    m := xs[0]
    for _, x := range xs[1:] {
        if x > m {
            m = x
        }
    }
    return m
}

Max([]int{3, 1, 4})        // works
Max([]float64{2.7, 1.8})   // also works, type-safe
```

The `[T int | float64]` says "T can be either type," and the compiler checks each
call. Use generics when you'd otherwise duplicate logic across types; for most
day-to-day code, plain slices and maps are enough.

<div class="knowledge-check" data-quiz data-correct-msg="Right — append returns the grown slice, so you assign it back." markdown="0">
  <p class="knowledge-check__q">Quick check: how do you add an element to a slice <code>s</code>?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">s.add(x)</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">s = append(s, x)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">s[len(s)] = x</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **slice** is a growable view into an array; extend it with **`append`** and
  assign the result back.
- Slices can **alias** shared storage — use `copy` when you need independence.
- A **map** is key/value lookup; the `value, ok` form reports whether a key exists.
- **Generics** (type parameters) write one type-safe function for many types.

Next up: testing, a first-class part of the language.
