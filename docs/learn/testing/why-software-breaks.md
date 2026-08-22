---
slug: why-software-breaks
title: Why does software break?
description: Software fails because people encode assumptions — about inputs, environments, and each other's code. Learn how complexity, change, and edge cases turn small oversights into visible failures, and why that's normal rather than shameful.
keywords: why software breaks, software bugs causes, edge cases, software complexity, assumptions in code, why programs fail, software quality basics
level: beginner
status: full
faq:
  - q: Why does software have so many bugs?
    a: Because software is a huge pile of precise instructions written by people who cannot hold every detail in their heads at once. Every line encodes assumptions about inputs, timing, other code, and the environment. When any assumption turns out to be wrong — or becomes wrong later as the world changes — the program misbehaves. Bugs are not a sign of carelessness so much as an inevitable by-product of complexity, which is why engineers build systematic defenses like testing instead of relying on being careful.
  - q: Can software ever be completely bug-free?
    a: For tiny programs with mathematically verified behavior, nearly. For real-world software — with operating systems, networks, hardware, and users in the loop — no. The realistic goal is different, to find bugs early, keep fixed bugs from returning, and make the remaining failures visible and recoverable. That is what this module teaches.
  - q: Is it my fault when my code breaks?
    a: In the trivial sense yes, someone typed the wrong thing. But mature teams treat bugs as information about the system, not verdicts on the person. The useful question is never "who broke it?" but "what let this mistake reach users, and what net do we add so the next one is caught?" That mindset shift — from blame to defenses — is where quality engineering starts.
---

# Why does software break?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Software breaks because every line of code encodes **assumptions** — about inputs,
timing, other code, and the environment — and reality eventually violates some of
them. **Complexity** means nobody can hold the whole system in their head;
**change** means code that was correct yesterday can be wrong today; **edge cases**
are the rare inputs nobody thought to try. Bugs are therefore *normal*, and the
professional response is not "be more careful" but **systematic defenses** —
which is what this whole module is about.
</div>

This is lesson 1, and its job is to change how you feel about things breaking.
If you've ever watched a program fail and thought "I must be bad at this" — you're
not. You've met the fundamental condition of the craft. By the end of this lesson
you'll know the three forces that make failure inevitable, and why the engineers
you admire don't avoid bugs so much as *trap* them early.

## What is code, really?

A program is an enormous stack of exact instructions. A mid-sized project like
GopherTrunk runs to hundreds of thousands of lines, and every single one is a small
claim about the world: *this value will never be negative*, *this file exists*,
*this radio message always has 64 bits*, *nobody calls this function twice at
once*. Most of those claims are never written down — they live silently inside
the code as **assumptions**.

The program works exactly as long as every assumption holds. The moment one
doesn't — a user pastes an emoji into a numbers-only field, a network reply
arrives out of order, a radio sends a message a half-bit shorter than the spec
promised — the instructions keep executing, but now they're operating on a world
that doesn't match the one they were written for. That mismatch is where every
bug is born.

## Force one: complexity

No one can hold a whole system in their head. You can fully understand a
50-line function; you cannot fully understand 500,000 lines plus the operating
system, the network stack, and the hardware underneath. So developers work with a
**mental model** — a simplified picture of how everything behaves — and mental
models are always missing details.

Complexity also multiplies interactions. Ten components that each work perfectly
alone can still fail in combination, because component A's output is a legal value
that component B never expected. The bigger the system, the more of its behavior
lives in these *interactions between* parts rather than in any part — and
interactions are precisely what a single developer reading a single file cannot
see.

## Force two: change

Here's the uncomfortable part: **correct code becomes incorrect without anyone
touching it**. The code stays frozen; the world moves.

- A library you depend on releases a new version with subtly different behavior.
- A file format, API, or protocol evolves.
- Traffic grows 100× and a loop that was "fast enough" no longer is.
- A teammate edits a *different* file, changing an input your code silently
  relied on.

That last one is the everyday killer. Software is edited constantly, and every
edit risks invalidating assumptions elsewhere. A bug that appears in behavior
that *used to work* is called a **regression**, and regressions are so central to
this module that [an entire lesson](/learn/testing/regression-tests/) — and
GopherTrunk's entire fix policy — is built around preventing them.

## Force three: edge cases

Developers naturally test the **happy path** — the normal input, the expected
sequence. But real inputs include the weird ones: the empty list, the zero, the
negative number, the string with a quote in it, the leap day, the two events that
arrive in the same millisecond. These rare inputs are **edge cases**, and they're
where bugs concentrate, because they're exactly what nobody imagined while
writing.

Radio software is an edge-case generator. A decoder like GopherTrunk must handle
signals that are weak, distorted, cut off mid-message, or slightly out of spec —
because real transmitters produce all of those daily. Code that only works on a
clean, strong, textbook signal *looks* finished and isn't. "Works on the input I
tried" and "works" are different claims, and confusing them is the root mistake
this module trains out of you.

> Rule of thumb: if you haven't deliberately tried to break it, you don't know
> that it works — you only know it hasn't failed *yet*.

## So what do professionals do about it?

Not "be more careful." Care doesn't scale, and everyone's attention fails
sometimes. Instead, engineers build **layered defenses** that catch mistakes
mechanically:

| Defense | Catches | Covered in |
|---------|---------|------------|
| Automated tests | Wrong behavior, and regressions when code changes | Units 2–3 |
| Linters & static analysis | Suspicious code before it even runs | Unit 4 |
| Code review & CI | What one person's blind spot misses | Unit 4 |
| Methodical debugging | The bugs that get through anyway | Unit 5 |

Each layer is imperfect; stacked, they catch most problems before a user ever
sees them. The rest of this module walks each layer, ending with how GopherTrunk
combines all of them into a working discipline.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the code didn't change, but the world it made assumptions about did. That's why testing has to be continuous, not one-time." markdown="0">
  <p class="knowledge-check__q">Quick check: a function worked perfectly for a year, was never edited, and now fails. What's the most likely explanation?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">That's impossible — unchanged code can't stop working</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The compiler introduced random errors over time</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Something around it changed — an input, a dependency, or other code it relied on</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Every line of code encodes **assumptions**; bugs are what happens when reality
  violates one.
- **Complexity** means no one holds the whole system in their head, and failures
  hide in the interactions between parts.
- **Change** breaks unchanged code — a bug in behavior that used to work is a
  **regression**.
- **Edge cases** — the rare, weird inputs — are where bugs concentrate, because
  they're what nobody imagined.
- The professional answer is **layered, mechanical defenses** — tests, analysis,
  review, CI — not heroic carefulness.

Next up: [What is a bug?](/learn/testing/what-is-a-bug/)
