---
slug: dry-kiss-yagni
title: DRY, KISS, YAGNI & separation of concerns
description: Four heuristics that keep code lean — DRY, KISS, YAGNI and separation of concerns — plus when duplication is fine and abstraction is the wrong call.
keywords: DRY principle, KISS principle, YAGNI, separation of concerns, don't repeat yourself, premature abstraction, wrong abstraction, over-engineering, software design heuristics
level: beginner
status: full
faq:
  - q: "Does DRY mean I should never duplicate any code?"
    a: "No. DRY is about not duplicating *knowledge* — a single rule or fact that lives in one place. Two pieces of code that happen to look alike today but represent different ideas can safely stay separate. Forcing them together creates the \"wrong abstraction,\" which is harder to undo than duplication."
  - q: "How is KISS different from YAGNI?"
    a: "KISS says keep the solution to a real problem as simple as possible. YAGNI says don't build solutions to problems you don't yet have. KISS is about *how* you solve today's problem; YAGNI is about *whether* you should solve a speculative future one at all."
  - q: "What is separation of concerns in plain terms?"
    a: "It means each part of a program should deal with one concern — one area of responsibility — and not tangle itself with unrelated ones. For example, keep code that reads radio samples separate from code that decodes them and code that displays results, so each can change independently."
---
# DRY, KISS, YAGNI & separation of concerns

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**DRY** — every piece of knowledge has one home, but don't over-abstract. **KISS & YAGNI** — solve today's problem simply, skip tomorrow's imaginary one. **Separation of concerns** — one part, one responsibility.
</div>

Once your code is readable, the next question is structural: how do you keep a growing program from collapsing under its own weight? Over decades, the industry distilled a handful of heuristics that answer this — short slogans that pack hard-won judgement. The four in this lesson, **DRY, KISS, YAGNI, and separation of concerns**, work together to keep code lean and changeable. But heuristics are guidelines, not laws. Misapplied, each can do real damage, and the most interesting part of this lesson is knowing when *not* to follow them.

## DRY — Don't Repeat Yourself

DRY says that **every piece of knowledge should have a single, authoritative representation** in your system. If a business rule, a constant, or a calculation appears in five places, a change means finding and editing all five — and missing one is a bug.

The classic example is a magic number scattered around:

```go
// Repeated knowledge — change one and you must change them all.
buffer := make([]float64, 48000)        // sample rate
window := 48000 / 100                   // 10 ms window
if elapsedSamples > 48000*5 { ... }     // 5-second timeout

// DRY — the sample rate has one home.
const sampleRateHz = 48000
buffer := make([]float64, sampleRateHz)
window := sampleRateHz / 100
if elapsedSamples > sampleRateHz*5 { ... }
```

Now "the sample rate is 48 kHz" is stated once. DRY applies far beyond constants: a validation rule, an API endpoint format, a data schema — anything that encodes a fact your system relies on.

The crucial nuance: **DRY is about knowledge, not about characters.** Two snippets that look identical but represent *different* facts are not a DRY violation, and merging them is a mistake.

## The wrong-abstraction trap

The most common way DRY backfires is premature deduplication. You see two functions that look 90% alike, so you extract a shared helper with a flag or two to cover the differences. It feels clean. Then the two callers start evolving in different directions, you add another flag, then a conditional, then a special case — and soon the "shared" abstraction is a tangle that serves no one well.

Sandi Metz put it sharply: **"duplication is far cheaper than the wrong abstraction."** Duplicated code is easy to understand and easy to delete. A wrong abstraction has tendrils everywhere and is painful to unwind.

A safer rule of thumb than "never repeat":

- Don't abstract on the *first* duplication. Copy it.
- Wait until you see the pattern a third time and understand which parts truly vary.
- If an abstraction starts sprouting boolean flags and special cases, that's a signal it's wrong — consider inlining it back into separate copies.

Duplication is a smell worth watching, but it is a *cheaper* problem than over-abstraction.

## KISS — Keep It Simple

KISS reminds you that **the simplest design that solves the actual problem is almost always the best one.** Complexity is not a sign of sophistication; it's a cost you pay forever in comprehension, bugs, and onboarding.

Simplicity shows up as: fewer moving parts, fewer layers of indirection, fewer configuration options, and straightforward control flow. When you reach for a design pattern, a generic framework, or a configurable abstraction, ask whether the plain version would do. Often a function and a slice beat a class hierarchy and a plugin registry.

KISS doesn't mean "no abstractions" — some problems are genuinely complex and need structure. It means: match the complexity of the solution to the complexity of the problem, and not a notch more.

## YAGNI — You Aren't Gonna Need It

YAGNI targets a specific, seductive failure: building for an imagined future. "We might want to support multiple databases someday, so let me add an abstraction layer now." "Someday we'll have plugins, so let me build a registry." Most of those somedays never arrive, and the speculative machinery sits there as dead weight — extra code to read, test, and maintain, often guessing the future wrong.

The principle: **implement things when you actually need them, not when you merely foresee needing them.** This is hard because anticipating needs *feels* responsible. But YAGNI argues that the cost of adding a feature later (when you understand the real requirement) is almost always lower than the cost of carrying unused, speculative code now — and of getting the guess wrong.

YAGNI and KISS reinforce each other: skipping speculative features (YAGNI) keeps today's design simple (KISS). Together they're the antidote to over-engineering.

A quick contrast:

| Principle | Question it answers | Failure it prevents |
|-----------|--------------------|--------------------|
| DRY | Does this knowledge live in one place? | Inconsistent edits, drift |
| KISS | Is this the simplest solution to the real problem? | Needless complexity |
| YAGNI | Am I building for a need I actually have? | Speculative over-engineering |

## Separation of concerns

Separation of concerns (SoC) is the structural backbone behind the slogans: **each part of a program should be responsible for one concern and minimally entangled with others.** A "concern" is a distinct area of responsibility — input, business logic, storage, presentation.

Consider a small radio pipeline in GopherTrunk. One natural separation:

- **Acquisition** — pull raw IQ samples from the SDR hardware.
- **Decoding** — turn samples into demodulated frames.
- **Presentation** — show decoded messages in the UI.

If these are tangled into one function, changing the display forces you to understand sample acquisition, and swapping the hardware risks breaking the decoder. Kept separate, each can change, be tested, and be reasoned about on its own. You could replace the SDR with a file-based source without touching a line of decode logic.

Separation of concerns is *why* DRY, KISS, and YAGNI pay off: clean seams between responsibilities are what make code simple to keep simple, easy to deduplicate correctly, and cheap to extend later. It's also the conceptual bridge to the next lesson — [SOLID](/learn/intro-software-dev/solid/) is, in large part, a rigorous take on separating concerns in object-oriented designs.

<div class="knowledge-check" data-quiz data-correct-msg="Right — DRY is about deduplicating knowledge; identical-looking code for different ideas should stay separate." markdown="0">
  <p class="knowledge-check__q">Quick check: two functions look almost identical but encode different business rules. What does DRY advise?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Always merge them into one shared helper with flags</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Leave them separate — they represent different knowledge</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Delete one of them since they're nearly the same</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **DRY** — give each piece of *knowledge* one authoritative home; deduplicate facts, not coincidentally similar text.
- **Wrong abstraction** — duplication is cheaper than the wrong abstraction; wait for the pattern to prove itself before extracting.
- **KISS** — match solution complexity to the real problem; the simplest design that works usually wins.
- **YAGNI** — build for needs you have, not ones you imagine; speculative code is mostly dead weight.
- **Separation of concerns** — give each part one responsibility and clean seams, so it can change independently.

Next up: a deeper, object-oriented framework for one-job design — the [SOLID principles](/learn/intro-software-dev/solid/).
