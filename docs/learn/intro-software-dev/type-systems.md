---
slug: type-systems
title: Type systems & safety
description: Static vs dynamic, strong vs weak typing, type inference and gradual typing — what type checking catches before run time, what it costs, and where it earns its keep.
keywords: type system, static typing, dynamic typing, strong typing, type inference, gradual typing, TypeScript, type safety
level: intermediate
status: full
faq:
  - q: "What is the difference between static and dynamic typing?"
    a: "With **static typing** (Rust, Go, Java) types are checked at compile time, before the program runs, so whole classes of bugs are caught early. With **dynamic typing** (Python, Ruby, JavaScript) types are checked at run time, which is more flexible but lets type errors slip through until that line actually executes."
  - q: "Is strong typing the same as static typing?"
    a: "No. **Static vs dynamic** is *when* types are checked; **strong vs weak** is *how strictly* the language enforces them. Python is dynamically typed but strongly typed — it refuses to add a string to an integer. JavaScript is dynamic and weaker — it silently coerces types."
  - q: "What is gradual typing?"
    a: "**Gradual typing** lets you add optional type annotations to a dynamically-typed language and check them with a separate tool. TypeScript adds types to JavaScript; Python type hints checked by mypy do the same for Python. You get many compile-time guarantees without giving up the language's flexibility."
---
# Type systems & safety

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Static vs dynamic** — *when* types are checked. **Strong vs weak** — *how strictly* they're enforced. **Gradual typing** — add type safety to a dynamic language incrementally.
</div>

Every value in a program has a **type** — integer, string, list, a `Receiver`
object. A **type system** is the set of rules a language uses to track those types
and decide which operations are allowed. Type systems are one of the deepest
dividing lines between languages, and the debates around them are really debates
about a trade-off: how much **safety** you want guaranteed up front versus how
much **flexibility and brevity** you are willing to trade for it. This lesson
gives you the vocabulary to reason about that trade-off instead of arguing by
reflex.

## Static vs dynamic typing

The first axis is *when* the language checks types.

**Static typing** checks types at **compile time**, before the program runs. The
compiler knows the type of every variable and rejects code that misuses them. C,
C++, Rust, Go, Java and C# are statically typed.

**Dynamic typing** checks types at **run time**, as each operation executes. A
variable can hold a string now and a number later; the language only complains
when an actual operation is invalid. Python, Ruby and JavaScript are dynamically
typed.

```python
# Dynamic (Python): this is fine until the line runs
def gain(x):
    return x * 2      # works for numbers... or strings, surprisingly
gain("oops")          # returns "oopsoops" — no error, maybe not what you meant
```

```go
// Static (Go): the compiler rejects the mismatch before you can run it
func gain(x float64) float64 { return x * 2 }
gain("oops") // compile error: cannot use "oops" as float64
```

The static-typing payoff is that **whole classes of bugs are caught at compile
time** — passing the wrong type, calling a method that does not exist, returning
the wrong shape of data. The dynamic-typing payoff is **flexibility and less
ceremony**: quick scripts, duck typing, and code that adapts at run time.

## Strong vs weak typing

A separate axis — often confused with the first — is *how strictly* the language
enforces types once it knows them.

**Strong typing** refuses to silently mix incompatible types. **Weak typing**
performs implicit conversions ("coercion") to make operations work, sometimes
surprisingly.

- **Python** is *dynamically* but *strongly* typed: `"3" + 5` raises an error
  rather than guessing.
- **JavaScript** is *dynamically* and *weakly* typed: `"3" + 5` yields `"35"`
  (it coerces the number to a string), and `"3" * 5` yields `15`.

So the two axes are independent. You can have static+strong (Rust), dynamic+strong
(Python), dynamic+weak (JavaScript), and static-but-looser (C, which lets you cast
fairly freely). "Strong" and "static" both push toward safety, but they are not
the same property.

| | Static | Dynamic |
|---|---|---|
| **Strong** | Rust, Go, Java | Python, Ruby |
| **Weak(er)** | C, C++ | JavaScript, PHP |

## Type inference

A common objection to static typing is the **ceremony** — writing the type of
everything. **Type inference** removes most of it: the compiler *deduces* types
from context, so you get static checking with dynamic-looking brevity.

```rust
let count = 0;              // Rust infers i32
let names = vec!["a", "b"]; // infers Vec<&str>
```

Rust, Go (`:=`), Swift, Kotlin and modern C++ (`auto`) all infer types heavily.
You annotate at the boundaries — function signatures, public APIs — and let the
compiler fill in the rest. This is a big reason modern statically-typed languages
feel far less verbose than 1990s Java.

<div class="knowledge-check" data-quiz data-correct-msg="Right — static is about *when* types are checked (compile time); strong is about *how strictly* they're enforced. They're independent axes." markdown="0">
  <p class="knowledge-check__q">Quick check: Python rejects <code>"3" + 5</code> at run time. What does that make it?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Statically and weakly typed</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Dynamically and strongly typed</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Statically and strongly typed</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## What checking catches — and what it costs

A good type system pays for itself by catching mistakes for free, every time you
compile:

- **Wrong-type arguments** — you cannot pass a string where a count is expected.
- **Null/None misuse** — languages like Rust and Kotlin make "no value" a distinct,
  checked type so you cannot forget to handle it.
- **Refactoring safety** — rename a field and the compiler lists every site that
  must change.
- **Self-documenting APIs** — the signature tells you what goes in and what comes
  out.

But it is not free:

- **Ceremony** — even with inference, you write more annotations than in a dynamic
  language.
- **Rigidity** — some genuinely dynamic patterns become awkward to express, and you
  occasionally fight the type checker to express something it cannot model.
- **Slower prototyping** — you must make the types line up before you can run
  anything.

The right answer depends on the project's size and lifetime. A throwaway script
barely benefits; a large, long-lived, multi-author system benefits enormously,
which is why such systems tend toward static typing over time. The
[decision framework](/learn/intro-software-dev/decision-framework/) covers how to
weigh this for a real project.

## Gradual typing

You do not have to choose all-or-nothing. **Gradual typing** lets you add
*optional* type annotations to a dynamic language and check them with a separate
tool, so untyped and typed code coexist.

- **TypeScript** adds a static type layer over JavaScript; you opt in file by file,
  and a type checker catches errors before the code ships, even though it still
  runs as plain JavaScript.
- **Python type hints** (`def gain(x: float) -> float:`) are checked by tools like
  **mypy** or **pyright**, while the interpreter ignores them at run time.

Gradual typing has become the default for large dynamic codebases: keep the
flexibility for quick work, add guarantees where the code is critical or shared.

## Generics and richer type systems

The conversation does not stop at "static or dynamic". Statically-typed languages
also differ in how *expressive* their type systems are — how much of a program's
meaning you can encode in types so the compiler enforces it for you.

- **Generics (parametric polymorphism)** let you write code once that works for many
  types while staying fully type-checked — a `List<T>` is a list of *some specific*
  type, not a list of "anything". Java, C#, Rust, Swift and (since 1.18) Go all
  have generics.
- **Sum types / enums with data** (Rust's `enum`, Swift's `enum`, Haskell's
  algebraic data types) let you say "a value is exactly one of these cases", and the
  compiler forces you to handle every case — invaluable for modelling protocol
  messages or parser states.
- **Option/Result types** replace `null` and exception-based error handling with
  values the type system makes you unwrap, so "I forgot to handle the error case"
  becomes a compile error.

The general principle is **"make illegal states unrepresentable"**: design your
types so that a value that should not exist cannot even be constructed. The more a
type system lets you express, the more bugs become compile errors instead of run
time surprises — at the cost of more upfront design thinking. This is why the
expressiveness of a language's type system, not just whether it has one, is part of
the [decision framework](/learn/intro-software-dev/decision-framework/) for serious
projects.

## A units bug a type system can prevent

In radio software you constantly juggle quantities that are *just numbers* to the
computer but mean very different things: a frequency in hertz, a **sample rate**
in samples-per-second, a gain in dB, a count of samples. A plain `float` lets you
accidentally pass a sample rate where a frequency belongs, and nothing complains —
the bug surfaces later as garbled audio.

A strong, static type system lets you wrap each quantity in its own type:

```rust
struct Hertz(f64);
struct SampleRate(f64);

fn tune(freq: Hertz) { /* ... */ }

let sr = SampleRate(2_048_000.0);
tune(sr); // compile error: expected Hertz, found SampleRate
```

Now the mistake is impossible to compile, not merely unlikely. This is the
type system catching a **whole class of unit-confusion bugs** — exactly the kind
that are maddening to debug at run time when all you see is noise where signal
should be.

## Recap

- **Static vs dynamic** — whether types are checked at compile time or run time.
- **Strong vs weak** — how strictly the language enforces types; independent of the
  static/dynamic axis.
- **Type inference** — gives static safety without spelling out every type.
- **Checking catches bugs early** — wrong types, null misuse, broken refactors — but
  costs ceremony and some rigidity.
- **Gradual typing** — TypeScript and Python hints add optional, checkable types to
  dynamic languages.
- **Types encode meaning** — wrapping hertz and sample-rate in distinct types makes
  unit-confusion bugs un-compilable.

Next up: how programs get and release the memory they use — [manual management,
garbage collection and ownership](/learn/intro-software-dev/memory-management/).
