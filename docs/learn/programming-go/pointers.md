---
slug: pointers
title: Pointers
description: "What a pointer is, why Go has them without pointer arithmetic, and how they let functions modify values and avoid copying large structs — safely and simply."
keywords: go pointers, pointer vs value, nil pointer, address of operator, dereference go, pointer to struct, go pass by value
level: intermediate
status: full
prereq:
  - values-and-types
faq:
  - q: Does Go have pointer arithmetic like C?
    a: "No. You can take a pointer with & and follow it with *, but you cannot add to or subtract from an address. That single restriction removes an entire class of memory-corruption bugs while keeping the useful part of pointers: sharing and mutating a value."
  - q: When should a function take a pointer instead of a value?
    a: "Take a pointer when the function must modify the caller's value, or when the value is a large struct you don't want to copy on every call. For small values that only get read, passing by value is simpler and just as fast."
---

# Pointers

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **pointer** holds the *address* of a value. `&x` takes the address; `*p`
**dereferences** it to reach the value. Go has pointers but **no pointer
arithmetic**, so you get sharing and mutation without the classic memory bugs. Use
them to let a function **modify its caller's value** or to **avoid copying** a big
struct. A pointer to nothing is **`nil`**.
</div>

Go passes everything by value — a function gets its own copy of each argument. This
lesson explains the escape hatch: pointers, which let a function reach back and
change the original.

## What is a pointer?

A pointer is a value whose contents are the memory address of another value. Two
operators do all the work:

```go
snr := 12.5
p := &snr        // p is a *float64 — the address of snr
fmt.Println(*p)  // 12.5 — dereference to read the value
*p = 9.0         // write through the pointer
fmt.Println(snr) // 9 — snr itself changed
```

`&` means "address of"; `*` in front of a pointer means "the value it points at."
`*float64` is read as "pointer to float64."

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A pointer variable p holding an arrow that points at a separate box holding the value 9.0." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="40" width="120" height="40" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="80" y="30" text-anchor="middle" font-size="12" fill="currentColor">p (*float64)</text>
  <text x="80" y="65" text-anchor="middle" font-size="12" fill="currentColor">0x40c0</text>
  <rect x="300" y="40" width="120" height="40" rx="6" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="360" y="30" text-anchor="middle" font-size="12" fill="currentColor">snr (float64)</text>
  <text x="360" y="65" text-anchor="middle" font-size="12" fill="currentColor">9.0</text>
  <line x1="140" y1="60" x2="300" y2="60" stroke="currentColor" stroke-width="1.5" marker-end="url(#pa)"/>
  <defs><marker id="pa" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A pointer stores an address; dereferencing with <strong>*p</strong> follows the arrow to the value.</figcaption>
</figure>

## Why do pointers matter? Modifying the caller

Because arguments are copied, a plain function can't change what the caller passed:

```go
func reset(x int)  { x = 0 }        // changes only the copy
func clear(p *int) { *p = 0 }       // changes the caller's value

n := 5
reset(n); fmt.Println(n)   // 5 — unchanged
clear(&n); fmt.Println(n)  // 0 — cleared through the pointer
```

This is why methods that mutate a struct use a **pointer receiver** — the subject of
[Structs & methods](/learn/programming-go/structs-and-methods/).

## Avoiding copies of big structs

A `Receiver` in GopherTrunk might hold filter state, buffers, and AGC settings.
Passing it by value copies all of that on every call. Passing `*Receiver` copies only
the address:

```go
func (r *Receiver) Step(sample complex64) {
    r.agc.Apply(&sample)   // mutates r's state in place
}
```

Rule of thumb: small, read-only values go by value; things you must mutate, or large
structs, go by pointer.

## The nil pointer

A pointer that points at nothing is `nil` — the zero value for any pointer type.
Dereferencing a `nil` pointer panics, so guard it when it might be unset:

```go
if cfg.Filter != nil {
    cfg.Filter.Apply(buf)
}
```

Go has no pointer arithmetic, so a pointer is always either `nil` or a valid address
of a real value — never a hand-computed offset into memory. That is what makes Go
pointers safe to use everywhere.

<div class="knowledge-check" data-quiz data-correct-msg="Right — & takes an address, * follows it to the value." markdown="0">
  <p class="knowledge-check__q">Quick check: what does the <em>*</em> operator do in front of a pointer?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Takes the address of a value</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Dereferences it — reaches the value it points at</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Advances the pointer to the next address</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **pointer** holds a value's address; **`&`** takes it, **`*`** dereferences it.
- Go has **no pointer arithmetic** — a pointer is always `nil` or a valid address.
- Use pointers to **modify the caller's value** or to **avoid copying** big structs.
- The zero pointer is **`nil`**; dereferencing `nil` panics, so guard it.

Next up: grouping data with structs and attaching behaviour with methods.
