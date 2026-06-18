---
slug: what-are-patterns
title: What is a design pattern?
description: A design pattern is a named, reusable solution to a recurring design problem. Learn the GoF categories, how patterns become shared vocabulary, and anti-patterns.
keywords: design pattern, gang of four, GoF, creational structural behavioral, software design, anti-pattern, god object, golden hammer, Christopher Alexander
level: beginner
status: full
faq:
  - q: "What exactly is a design pattern?"
    a: "A **design pattern** is a named, reusable description of a solution to a problem that comes up again and again in software design. It is not finished code you paste in — it is a template for how to arrange classes, objects, and responsibilities so the design stays flexible. The pattern gives you a vocabulary word (\"use a factory here\") that captures a whole approach in one term."
  - q: "Where did design patterns come from?"
    a: "The idea was borrowed from architecture. **Christopher Alexander** wrote about patterns in building design in the 1970s. In 1994 four authors — the \"Gang of Four\" — published *Design Patterns*, cataloguing 23 reusable solutions for object-oriented software. That book made patterns a standard part of how developers talk about design."
  - q: "Can you use too many design patterns?"
    a: "Yes. Patterns solve specific problems, and forcing one where it is not needed adds indirection and complexity for no benefit — the \"golden hammer\" anti-pattern. A pattern earns its place when it removes a real pain (a hard-to-change design, duplicated logic, tight coupling). If the simple version already works, the simple version usually wins."
---
# What is a design pattern?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Pattern** — a named, reusable solution to a recurring design problem. **Vocabulary, not code** — patterns are templates and shared language, not copy-paste snippets. **Three GoF families** — creational, structural, and behavioral patterns cover how objects are made, composed, and made to cooperate.
</div>

You have written enough code by now to notice something: the same shapes keep reappearing. You need an object but you do not want the caller to know which exact class it is. You need to add behavior to something without rewriting it. You need one part of a program to react when another part changes. These problems show up in a scanner's decoder engine, a web server, and a video game alike. Over decades, the community noticed these recurring problems and wrote down the proven solutions. Those write-ups are **design patterns**. This lesson explains what a pattern is, where the idea came from, the three classic families, and the failure modes — including overusing patterns themselves.

## A pattern is a named, reusable solution

A **design pattern** is a description of a solution to a problem that recurs in software design, captured in a way you can reuse across many situations. Each pattern has four rough parts:

- **A name** — so you can refer to the whole idea in one or two words.
- **A problem** — the situation where the pattern applies.
- **A solution** — the general arrangement of classes and objects, not a specific implementation.
- **Consequences** — the trade-offs you accept by using it.

The key word is *general*. A pattern is not a library function or a code template you paste into your editor. It is closer to a recipe written at the level of "make a roux, then add liquid" rather than "use exactly 30 grams of this brand of flour." You still have to implement it in your own language, for your own types, in your own codebase. Two correct uses of the same pattern can look quite different in source code while sharing the same underlying structure.

This is why patterns survive language changes. The Factory idea works in Java, Go, Python, and Rust even though the syntax differs wildly. You are reusing the *design*, not the bytes.

## Patterns are vocabulary, not copy-paste code

The most underrated benefit of patterns is communication. When a teammate says "let's put a **facade** over the DSP chain so the UI only sees one call," everyone who knows the term instantly understands a whole design decision: a simple front hiding a complex subsystem, where to draw the boundary, and roughly what the interface looks like. That single word replaces a paragraph of explanation.

> A shared pattern vocabulary lets a team discuss design at a higher altitude — talking about responsibilities and relationships instead of lines of code.

This connects directly to ideas earlier in the path. Patterns are tools for managing [abstraction and coupling](/learn/intro-software-dev/abstraction-coupling/) — most of them exist to keep parts of a system from depending too tightly on each other. And many of them are concrete ways to honour the principles in [SOLID](/learn/intro-software-dev/solid/), especially the open/closed principle: design so you can extend behavior without editing existing code.

Because patterns are vocabulary, learning them is partly learning to *recognize* them in code you read. Much existing software already uses these structures, sometimes without naming them. Once you have the names, you start seeing factories and observers everywhere — in frameworks, standard libraries, and the GopherTrunk codebase.

## Where the idea came from

Patterns did not start in software. The architect **Christopher Alexander** introduced the concept in the late 1970s, in books like *A Pattern Language*, to describe recurring good solutions in building and town design — a window seat, a courtyard that catches light, the right width for a walkable street. Each was a named solution to a human problem, reusable across many buildings.

Software people borrowed the idea. In 1994, four authors — Erich Gamma, Richard Helm, Ralph Johnson, and John Vlissides, collectively nicknamed the **"Gang of Four" (GoF)** — published *Design Patterns: Elements of Reusable Object-Oriented Software*. It catalogued **23 patterns** for object-oriented design and gave each a name, a problem, a solution, and consequences. That book is the reason "design pattern" is now a standard phrase, and its 23 patterns are still the canonical core. Later work added concurrency patterns, enterprise patterns, and architectural patterns, but the GoF catalogue is where most people start.

## The three GoF categories

The Gang of Four sorted their 23 patterns into three families by *what kind of problem* each solves. You will spend the next three lessons on these in detail; here is the map.

| Category | The problem it addresses | Examples |
|----------|--------------------------|----------|
| **Creational** | How objects get *created*, without hard-coding the exact class | Factory Method, Builder, Singleton, Abstract Factory, Prototype |
| **Structural** | How objects are *composed* into larger structures | Adapter, Facade, Decorator, Proxy, Composite, Bridge |
| **Behavioral** | How objects *interact* and share responsibility | Observer, Strategy, State, Command, Iterator, Template Method |

A quick way to remember it: **creational** is about *making* things, **structural** is about *arranging* things, and **behavioral** is about *coordinating* things. A real radio decoder uses all three at once — a factory chooses which protocol decoder to build, an adapter wraps the hardware driver, and an observer notifies the UI when a call is decoded.

## Anti-patterns: the shapes to avoid

Patterns describe good recurring solutions. **Anti-patterns** are the opposite: recurring *bad* solutions that look tempting but cause pain. Naming them is just as useful, because it lets a team say "that's a god object" and agree to fix it. A few classics:

- **God object** — one class that knows and does everything. It accumulates responsibilities until nothing can change without touching it. The cure is splitting responsibilities (high cohesion, low coupling).
- **Spaghetti code** — control flow so tangled, with no clear structure, that you cannot follow it. Usually the result of growth without design.
- **Golden hammer** — "when all you have is a hammer, everything looks like a nail." Reaching for the same favourite tool or pattern for every problem, whether it fits or not.

The golden hammer is the one that bites pattern enthusiasts directly: once you learn the catalogue, there is a strong urge to use it. But every pattern adds indirection. A factory you do not need is just a confusing extra layer between the caller and the object. Patterns are medicine for specific diseases — applying them to a healthy design only adds side effects. The skill is not memorizing all 23; it is judging *when a real problem calls for one*, which is exactly the kind of judgement the [decision framework](/learn/intro-software-dev/decision-framework/) lesson trains.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a pattern is a reusable design template and a shared name, not finished source code." markdown="0">
  <p class="knowledge-check__q">Quick check: What is a design pattern, most precisely?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A code snippet you copy and paste into your project unchanged</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A named, reusable template for solving a recurring design problem</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A rule that every program must follow to compile</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **A pattern is a named, reusable solution** — a general template (name, problem, solution, consequences) for a recurring design problem, not finished code.
- **Vocabulary first** — patterns let a team discuss design at a higher level; "use a facade here" carries a whole decision in one word.
- **Origins** — Christopher Alexander brought patterns from architecture; the Gang of Four's 1994 book catalogued 23 for object-oriented software.
- **Three GoF families** — creational (making objects), structural (arranging objects), behavioral (coordinating objects).
- **Anti-patterns** — god object, spaghetti code, and the golden hammer are recurring *bad* solutions worth naming so you can avoid them.
- **Patterns can be overused** — every pattern adds indirection; use one only when a real design problem calls for it.

Next up: the creational patterns — Factory, Builder, and Singleton — and how they decide *how objects get made*. See [Creational patterns](/learn/intro-software-dev/creational-patterns/).
