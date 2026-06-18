---
slug: docs-and-maintenance
title: Documentation, tech debt & maintenance
description: The kinds of documentation, writing for future readers, technical debt and how to pay it down, refactoring, deprecation, bus factor, and why most of a program's life is maintenance.
keywords: software documentation, README, API docs, technical debt, refactoring, deprecation, software maintenance, bus factor, writing for maintainers, architecture docs
level: intermediate
status: full
faq:
  - q: "Isn't taking on technical debt always a bad idea?"
    a: "No — deliberate, well-understood debt can be a smart trade. Shipping a quick-but-imperfect solution to hit a deadline or validate an idea is reasonable, as long as you know you're borrowing and plan to repay. The dangerous kind is accidental or ignored debt that compounds silently until the codebase grinds to a halt. Like financial debt, it's a tool when managed and a trap when forgotten."
  - q: "What is the bus factor?"
    a: "The bus factor is the number of people who'd have to be hit by a bus (or just leave) before a project is in serious trouble because nobody else understands it. A bus factor of one means a single person holds critical knowledge in their head alone. Documentation, code review, and shared ownership raise the bus factor, making the project resilient to any one person leaving."
  - q: "How is refactoring different from rewriting?"
    a: "Refactoring changes a program's internal structure without changing its external behavior — you improve the code while it keeps doing exactly what it did, ideally with tests proving so. A rewrite throws the existing code out and builds anew, which is far riskier because you lose all the accumulated bug fixes and edge-case handling. Most of the time, steady refactoring beats a dramatic rewrite."
---
# Documentation, tech debt & maintenance

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Maintenance is the main act** — most of a program's life is spent being maintained, not first written. **Docs serve future readers** — including you, six months from now. **Tech debt is a tool** — deliberate and repaid, it's fine; ignored, it compounds until the code seizes.
</div>

There's a myth that software is "done" when it first ships. In reality, that's the starting line. Most of a program's life — and most of the total effort spent on it — happens *after* the initial version: fixing bugs, adapting to new requirements, keeping dependencies current, and helping the next person understand it. This final lesson of the module is about that long tail: the documentation that makes code survivable, the technical debt that accumulates as you go, and the maintenance reality that defines a project's real cost.

## The kinds of documentation

"Documentation" isn't one thing. Different readers need different documents, and good projects layer several:

- **READMEs** — the front door. What is this, what problem does it solve, how do I install and run it, where do I go next. The first thing anyone sees; often the only thing they read.
- **API / reference docs** — the precise contract of every public function, type, and endpoint: what it takes, what it returns, what it guarantees. Frequently generated from comments in the code so they stay close to the source.
- **Architecture docs** — the big picture: how the major components fit together and *why* the design is the way it is. These capture the reasoning future maintainers can't reverse-engineer from code alone.
- **Tutorials and reference material** — task-oriented guides that walk a newcomer from zero to working, distinct from dry reference. *This very site is an example* — a structured learning path is documentation in the tutorial-and-reference tradition, designed to take a reader from punch cards to shipping their own software.

A useful frame (the "Diátaxis" idea) is that tutorials, how-to guides, reference, and explanation each serve a different need, and mixing them muddies all four. You don't need every kind for every project, but knowing which kind you're writing keeps it focused.

## Writing for future readers

The single most important audience for documentation is the **future reader** — most often *you*, months from now, with every assumption forgotten. Code shows *what* it does; it rarely captures *why* a decision was made, what alternatives were rejected, or what constraint forced an ugly workaround. That "why" is exactly what evaporates from memory and exactly what a maintainer most needs.

This connects directly to writing [readable code](/learn/intro-software-dev/clean-code/): clear names and structure are the first layer of documentation, and prose fills the gaps that code can't express. The test of good documentation isn't how thorough it looks — it's whether someone unfamiliar can get unstuck without finding you and asking. Write down the non-obvious, keep it near the code so it doesn't rot, and delete docs that have gone stale rather than letting them mislead.

<div class="knowledge-check" data-quiz data-correct-msg="Right — comments and docs are most valuable for the 'why' that code can't express on its own." markdown="0">
  <p class="knowledge-check__q">Quick check: what should documentation most focus on capturing?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A line-by-line restatement of what every line of code does</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The 'why' — intent, trade-offs, and constraints the code can't show</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Nothing — good code never needs any documentation</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Technical debt

**Technical debt** is the implied future cost of choosing a quick or easy solution now instead of a better one that would take longer. The metaphor, coined by Ward Cunningham, is deliberate: like financial debt, you get something now (speed) and pay interest later (every future change in that area is slower and riskier).

Crucially, debt isn't automatically bad. **Deliberate, strategic debt** is a legitimate tool — ship a rough version to hit a deadline or validate whether an idea even works, *knowing* you've borrowed and intending to repay. The trouble is the other kinds:

- **Reckless debt** — cutting corners with no awareness, leaving a mess by accident.
- **Ignored debt** — debt you took on knowingly but never circle back to pay down.

Untended debt **compounds.** Each shortcut makes the next change a little harder, until a once-nimble codebase becomes one where every small task is a slog and developers fear touching anything. The skill is treating debt like a loan: take it on consciously, track it, and *pay it down* — usually by refactoring — before the interest swallows your velocity.

## Refactoring and deprecation

**Refactoring** is improving the internal structure of code *without changing what it does externally.* You rename for clarity, extract a tangled function into smaller ones, remove duplication — and the program behaves identically throughout. This is how you pay down debt safely, and it's exactly where [your test suite](/learn/intro-software-dev/testing/) earns its keep: tests give you the confidence to restructure aggressively, because they'll scream if behavior changes. Refactoring without tests is just hoping. Note the contrast with a *rewrite*, which discards the code entirely and forfeits years of accumulated bug fixes and edge-case handling — almost always riskier than steady, test-backed refactoring.

**Deprecation** is how you retire something gracefully. When an API or feature needs to go but people depend on it, you don't yank it overnight — you mark it deprecated, document the replacement, give users time and warnings to migrate, and only then remove it. Good deprecation respects the people building on your work and is part of the maturity of any long-lived project or library.

## Maintenance is most of the life

Step back and the dominant fact of software is this: **the initial build is a small fraction of the total.** A program that lives for years spends those years being maintained — adapting to new operating systems, patching security holes, fixing bugs found in the wild, adding features, and tracking dependency changes. Studies of software cost have long put maintenance at well over half — often the large majority — of total lifetime effort.

This reframes everything in the module. The point of [readable code](/learn/intro-software-dev/clean-code/), tests, version control, and CI isn't bureaucracy — it's that they make the *long, dominant maintenance phase* survivable. Code optimized only for getting written fast becomes a liability the moment someone has to change it, which is approximately always.

Take GopherTrunk as a concrete case. Radio protocols evolve, new signal types appear, operating systems shift under your feet, and dependencies need updating. A decoder written without docs or golden-file tests might work the day it ships and become unmaintainable a year later, when the original author has forgotten the protocol's quirks and no test pins down correct behavior. Invest in maintainability and that same decoder stays changeable for as long as people need it.

## Bus factor

A blunt but useful measure of project health is the **bus factor**: how many people would have to be hit by a bus (or simply leave) before the project is in serious trouble because the knowledge left with them. A bus factor of **one** — a single person holding critical understanding in their head alone — is a fragile project, one resignation away from a crisis.

Almost everything in this module raises the bus factor. Documentation externalizes knowledge from heads into shared, durable form. [Code review](/learn/intro-software-dev/version-control-collab/) spreads understanding across the team as a side effect. Version control records the *why* behind changes. Tests encode expected behavior so it isn't tribal lore. Raising the bus factor is what turns a clever individual's project into a resilient, lasting one — and it's the quiet through-line of every practice in this module.

## Recap

- **Documentation has kinds** — READMEs, API/reference, architecture docs, and tutorials/reference (like this site); each serves a different reader.
- **Write for future readers** — capture the *why* code can't express; the test is whether someone can get unstuck without asking you.
- **Tech debt is a tool** — deliberate, tracked, repaid debt is fine; reckless or ignored debt compounds until the code seizes up.
- **Refactor with tests** — improve structure without changing behavior; far safer than a rewrite, and tests make it fearless.
- **Deprecate gracefully** — retire features with warnings, a migration path, and time, respecting those who depend on you.
- **Maintenance dominates** — most of a program's life and cost is maintenance, which is why every practice in this module exists; raising the bus factor keeps the project alive.

Next up: with the systems behind shipping understood, the next module asks a sharper question — [why language choice matters](/learn/intro-software-dev/why-language-choice-matters/) when you start a project of your own.
