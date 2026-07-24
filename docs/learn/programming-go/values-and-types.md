---
slug: values-and-types
title: Values, variables & types
description: Go's built-in types, variable declarations, zero values, and static typing — why a small, strict set of types makes programs easier to reason about and safer to change.
keywords: go types, go variables, go var, short variable declaration, zero value, static typing, int string bool, go constants
level: beginner
status: full
prereq:
  - hello-go
faq:
  - q: What is a zero value in Go?
    a: Every variable in Go is usable the moment it is declared, even if you didn't assign it — it gets a sensible "zero value" for its type. Numbers start at 0, booleans at false, strings at "" (empty), and pointers and other reference types at nil. This removes a whole category of uninitialized-variable bugs.
  - q: When should I use := versus var?
    a: Use the short form `x := value` inside functions when you're declaring and assigning at once and Go can infer the type — it's the common case. Use `var x Type` when you need a variable without an initial value, want to state the type explicitly, or are declaring a package-level variable, where `:=` isn't allowed.
---

# Values, variables & types

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Go is **statically typed**: every value has a type fixed at compile time. You
declare variables with **`var`** or the short **`:=`** form, and an unassigned
variable starts at its type's **zero value** (0, `false`, `""`, or `nil`). The
built-in types are few and predictable, which is exactly what makes Go code easy
to read and safe to change.
</div>

Before functions and structs, you need the raw material: values and the types that
describe them. This lesson covers Go's building-block types and the two ways to
make a variable.

## The built-in types

Go keeps its type set small and explicit:

| Type | Holds | Example |
|------|-------|---------|
| `bool` | true or false | `true` |
| `int`, `int64`, `uint8` … | whole numbers (sized variants too) | `460025000` |
| `float64` | decimal numbers | `851.0125` |
| `string` | immutable text | `"control channel"` |
| `byte` | one 8-bit value (alias for `uint8`) | raw sample data |

Sized integer types like `uint8` and `int16` matter in signal work — a stream of
radio samples is often bytes or 16-bit integers, and choosing the right width
controls exactly how much memory a buffer uses.

## Declaring variables

Two forms cover almost everything:

```go
var frequency float64 = 851.0125   // explicit type and value
var count int                       // no value -> zero value (0)

name := "P25 control channel"       // short form: type inferred as string
locked := true                      // inferred as bool
```

The short `:=` form only works **inside functions**, where it is the everyday
choice. `var` is for package-level variables and for when you want a zero-valued
variable to fill in later.

## Zero values: nothing is uninitialized

In many languages a variable you forget to set holds garbage. In Go it holds a
defined **zero value**:

```text
int      -> 0
float64  -> 0.0
bool     -> false
string   -> ""   (empty string)
pointer  -> nil
```

That means you can declare `var total int` and immediately start adding to it — no
initialization ceremony, no undefined behaviour.

## Constants and readability

Values that never change become **constants** with `const`. They make code
self-documenting:

```go
const sampleRate = 2_400_000   // 2.4 MS/s; underscores are just for reading
```

## Static typing is a feature

Because types are checked when you compile, a whole class of mistakes never reaches
a running program. Assign a string to an `int`, or pass the wrong type to a
function, and the build fails with a clear message instead of crashing later. When
you are processing millions of samples a second, "the compiler already proved this
is type-correct" is a comforting guarantee.

<div class="knowledge-check" data-quiz data-correct-msg="Correct — an unassigned int starts at its zero value, 0." markdown="0">
  <p class="knowledge-check__q">Quick check: you write <code>var n int</code> and never assign it. What is <code>n</code>?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">undefined / garbage</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">0</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">nil</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Go is **statically typed** — every value's type is known at compile time.
- Declare with **`var`** (explicit) or **`:=`** (inferred, inside functions).
- Unassigned variables take a defined **zero value**, never garbage.
- **`const`** names fixed values; sized integer types control memory precisely.

Next up: functions, and the distinctive way Go handles errors.
