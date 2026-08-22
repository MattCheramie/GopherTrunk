---
slug: what-is-a-bug
title: What is a bug?
description: Mistake, defect, failure — the three-stage life of a bug, from a wrong line of code to a wrong behavior a user sees. Understanding the chain shapes how you hunt bugs and how you talk about them.
keywords: what is a bug, software defect, error vs fault vs failure, bug definition, software fault, debugging basics, defect vs failure
level: beginner
status: full
prereq:
  - why-software-breaks
---

# What is a bug?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
"Bug" hides three distinct things. A **mistake** is the human slip — a wrong idea
at the keyboard. It leaves a **defect** (or *fault*) in the code — a wrong line
that sits there silently. When that line finally runs on the input that exposes
it, you get a **failure** — the wrong behavior someone actually observes. One
defect can cause many failures, a defect can hide for years, and the failure you
see is often far from the defect that caused it. Debugging is walking that chain
*backwards*.
</div>

Everyone says "bug" for all of it — the typo, the broken code, the crash. That's
fine in conversation, but when you're *hunting* one, the distinctions start doing
real work. This lesson gives you the vocabulary that the rest of the module (and
every incident discussion you'll ever join) is built on.

## The three-stage chain

A bug has a life cycle, and each stage has a name:

| Stage | What it is | Where it lives | Example |
|-------|-----------|----------------|---------|
| **Mistake** | A human error while thinking or typing | In someone's head, briefly | "I assumed the list is never empty" |
| **Defect** (fault) | The wrong code the mistake produced | In the source, silently | `frames[0]` with no length check |
| **Failure** | Observable wrong behavior | At runtime, visibly | The decoder panics on a silent channel |

The mistake happens once and is gone. The defect persists in the code — possibly
for years — doing nothing until conditions expose it. The failure is the only
part anyone *sees*.

## Why a defect can hide for years

A defect only causes a failure when the program actually executes that code *and*
the input violates the assumption. `frames[0]` is perfectly fine on every call
where the slice has elements. If empty input is rare — say, it only happens when
a radio channel goes quiet at the exact moment a call starts — the defect can sit
in production for years looking like solid, proven code.

This is why "this code has worked forever" is weak evidence of correctness. It
means the defect-exposing input hasn't arrived *yet*. It's also why testing
deliberately feeds code the rare inputs — the
[edge cases](/learn/testing/why-software-breaks/) from lesson 1 — instead of
waiting for the world to supply them at the worst moment.

## Why the failure is far from the defect

Here's the part that makes debugging genuinely hard: the failure often appears
nowhere near the defect. Suppose a parsing function has a defect that, for one
rare message type, writes a garbage value into a struct field. Nothing crashes.
The struct is stored. Minutes later, a completely different part of the program
reads that field, divides by it, and crashes.

The stack trace points at the division. The defect is in the parser, minutes and
thousands of lines away. Data flowed the corruption from one to the other.
Debugging (Unit 5 of this module) is exactly this: starting from the failure you
can see and tracing backwards to the defect you can't — which is why
[root-cause analysis](/learn/testing/root-cause-analysis/) keeps asking "but what
*wrote* that value?"

> Rule of thumb: the line that crashed is where the failure *surfaced*, not
> necessarily where the defect *lives*. Fix the crash site alone and you've
> bandaged a symptom.

## One defect, many failures — and the reverse

The mapping between defects and failures is many-to-many, and both directions
matter in practice:

- **One defect → many failures.** A single wrong length check in a decoder can
  produce garbled audio on one system, a crash on another, and silent missing
  calls on a third. Bug trackers fill with three "different" reports that are one
  fix. Recognizing that duplicate-looking-different pattern saves enormous effort.
- **One failure → several defects.** Sometimes the crash only happens when *two*
  wrong things line up — a bad value *and* a missing validation that should have
  rejected it. Fixing either one makes the failure vanish, but fixing only the
  first leaves the missing net for the next bad value to sail through.

## Where does the word "bug" come from?

Engineers were calling mechanical faults "bugs" back in Thomas Edison's day, but
the famous story is from 1947: operators of the Harvard Mark II computer found a
literal moth trapped in a relay, taped it into the logbook, and wrote "first
actual case of bug being found." The word stuck for the whole chain —
mistake, defect, and failure alike — and you'll use it that way too. Just keep
the three stages straight in your head when precision matters: bug *reports*
describe failures; bug *fixes* remove defects; and blameless teams focus on the
system that let the mistake through, not the person who made it.

<div class="knowledge-check" data-quiz data-correct-msg="Exactly — the defect was always there in the code; the failure only appears when the exposing input finally shows up." markdown="0">
  <p class="knowledge-check__q">Quick check: a program has run flawlessly for two years, then suddenly crashes on an unusual input. When did the defect enter the system?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">When the code was written — it sat latent until this input exposed it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">At the moment of the crash — before that, the code was correct</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It appeared gradually as the code aged</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **mistake** (human) creates a **defect** in code, which later causes a
  **failure** someone observes — three stages, one word: "bug."
- A defect can **hide for years**; "it's always worked" only means the exposing
  input hasn't arrived yet.
- The failure often surfaces **far from the defect** — debugging is tracing the
  chain backwards.
- Defects and failures map **many-to-many**: one defect can look like several
  bugs, and one failure can need two fixes.
- Bug reports describe failures; bug fixes remove defects; healthy teams blame
  systems, not people.

Next up: [What does a bug cost?](/learn/testing/cost-of-a-bug/)
