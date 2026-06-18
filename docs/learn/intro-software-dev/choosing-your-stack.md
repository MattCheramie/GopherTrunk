---
slug: choosing-your-stack
title: Choosing your stack
description: How a solo developer picks a primary language, a small set of dependable libraries and tools, and favours boring, maintainable tech with easy distribution.
keywords: choosing a stack, solo developer stack, boring technology, single binary, fewer dependencies, maintainable tech, Go distribution, technology choice
level: intermediate
status: full
faq:
  - q: "How should a solo developer choose a programming language?"
    a: "Pick the language you are **most productive in** and can maintain alone for years — usually one you already know well, with a large standard library, good documentation, and a healthy ecosystem. Novelty is a tax you pay every day you work alone. A language that compiles to a **single binary**, like Go, removes a whole category of distribution and ops headaches when you have no operations team."
  - q: "What is \"boring technology\" and why does it matter?"
    a: "**Boring technology** is mature, widely used, well-documented tooling whose failure modes are understood and searchable. As a solo dev your scarcest resource is attention, and boring tech spends almost none of it — answers already exist on the internet, the edges are mapped, and you are not the one debugging a brand-new framework at midnight. Save your novelty budget for the part that is actually your idea."
  - q: "Should I minimise dependencies?"
    a: "Mostly yes. Every dependency is something you must understand, update, and trust to keep working. **Fewer moving parts** means less that can break while you are not looking. Pull in a library when it solves a genuinely hard problem (cryptography, parsing, DSP), but resist adding one to save ten lines of code you could own outright."
---
# Choosing your stack

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Productive primary language** — pick what you can maintain alone for years. **Boring tech** — mature and well-documented beats new and exciting. **Minimise moving parts** — fewer dependencies, easy single-binary distribution.
</div>

Module 6 gave you a framework for choosing a language on its technical merits. As a
solo developer you apply that framework through one extra lens: **what can one person
realistically maintain?** A team can afford a sprawling stack because it has people
to babysit each part. You cannot. Your stack has to be small enough to hold in one
head, boring enough to never surprise you, and easy enough to ship that distribution
is not its own second project. This lesson is about making those choices on purpose,
using the [decision framework](/learn/intro-software-dev/decision-framework/) you
already have — just weighted heavily toward maintainability.

## Pick one primary language you are productive in

Your primary language is the single most consequential choice in your stack, because
you will spend more time in it than anywhere else. The instinct to learn something
shiny for the project is usually a mistake. Optimise for **throughput over years**,
not excitement over a weekend:

- **Fluency.** A language you already know well lets you spend your attention on the
  *problem*, not on remembering syntax. As a beginner-to-intermediate solo dev, this
  alone outweighs most theoretical advantages of a "better" language.
- **A large standard library.** The more the language ships with — HTTP, JSON, file
  handling, testing — the fewer third-party dependencies you must manage.
- **Strong documentation and a healthy community.** When you hit a wall at midnight,
  the answer needs to already exist somewhere searchable.
- **Good tooling out of the box.** A built-in formatter, test runner and build tool
  mean you configure less and ship more.

You can absolutely have a second language for a specific job — a shell script here,
some SQL there — but resist running your project across several languages of equal
weight. Each one is a separate set of tools, idioms, and gotchas to keep current.

## Favour boring technology

There is a famous idea among small teams: spend your **innovation tokens** wisely.
You only get a few; every genuinely novel, unproven piece of your stack costs one,
because you will be the one mapping its sharp edges in production. Everything else
should be **boring** — mature, popular, and thoroughly documented.

Boring is not an insult. Boring means:

- The failure modes are **known and searchable**, so you are debugging a solved
  problem, not pioneering.
- The library is **still maintained** and will not be abandoned next year, leaving you
  holding it.
- The hiring-yourself problem is solved — *you* can come back in a year and still
  understand it.

Spend your one or two innovation tokens on the part of the project that is genuinely
your idea — the clever algorithm, the unusual feature. Make everything around it
deliberately dull.

## Keep the moving parts to a minimum

Every component in your stack — a database, a message queue, a build tool, a
dependency — is a thing that can break, needs updating, and must be understood. Alone,
the cost of each one is fully yours. So the default answer to "should I add this?" is
**no, unless it earns its place.**

A useful test before adding anything:

- Does it solve a problem that is genuinely **hard to do myself** (cryptography,
  parsing, signal processing)? If yes, take the dependency — do not roll your own
  crypto.
- Or am I adding it to save a few lines I could just **write and own**? If so, skip it.

| Choice | Heavier (more to maintain) | Lighter (solo-friendly) |
| --- | --- | --- |
| Data storage | Separate database server | A local file or embedded DB |
| Many small libraries | Dozens of dependencies | A handful you actually understand |
| Deployment | Orchestrated services | One process you run |

Fewer parts is not just less work — it is fewer 3am surprises. The most maintainable
system is the one with the least in it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — boring, mature technology has known, searchable failure modes, so a solo dev spends almost no attention on it." markdown="0">
  <p class="knowledge-check__q">Quick check: why do solo developers favour "boring" technology?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Because new technology is always slower than old technology</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Because mature, well-documented tools have known failure modes and cost almost no attention to run</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Because employers only ever ask about old technologies</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Distribution should be part of the choice

Here is a factor teams rarely weigh but solo devs must: **how hard is it to get your
program onto a stranger's machine?** With no operations team, anything that requires
the user to install a runtime, set up dependencies, or configure an environment is
support burden you will personally absorb — every email, every "it doesn't work on my
laptop."

This is where a language that compiles to a **single static binary** is a quiet
superpower. **Go** is the standard example: `go build` produces one self-contained
executable with no runtime to install, and you can **cross-compile** it for Linux,
macOS and Windows from your own machine. The user downloads one file and runs it. No
"install Python 3.11 first," no dependency hell, no version conflicts.

Compare the two distribution stories:

```text
Interpreted stack:  user installs runtime → installs deps → hopes versions match → runs
Single binary:      user downloads one file → runs
```

For a tool aimed at non-technical users — exactly the case for radio software a
scanner hobbyist downloads from the [downloads](/downloads.html) page — that
difference is enormous. The fewer steps between "I want this" and "it's running," the
fewer people give up. We unpack the full distribution strategy in
[packaging and distributing your software](/learn/intro-software-dev/packaging-and-distribution/);
for now, just let "how will I ship it?" be a real input to your stack choice, not an
afterthought.

## A worked stack for a small radio tool

Suppose you are building a modest software-defined-radio utility — capture some
samples, decode a signal, log the result. A maintainable solo stack might be:

- **Language:** Go — you are fluent, the standard library covers most of it, and the
  single-binary output makes shipping to hobbyists trivial.
- **Dependencies:** one well-maintained library for talking to the SDR device, and the
  standard library for everything else. That is the whole list.
- **Storage:** a plain log file or a small embedded database — no separate server to
  run.
- **Tooling:** the language's built-in formatter, linter and test runner; Git for
  version control.

Notice what is *not* there: no microservices, no exotic framework, no database to
administer. Every piece is something one person can hold in their head and maintain
for years. That restraint is the whole point. If you want to see how these choices
play out in a real radio codebase, the GopherTrunk
[architecture](/architecture.html) page is a good companion read.

## Recap

- **One productive primary language** — fluency, a big standard library, and good
  docs beat novelty when you work alone.
- **Boring technology** — mature, searchable, maintained tools cost almost no
  attention; spend your innovation tokens on your actual idea.
- **Minimise moving parts** — every dependency and service is yours to maintain, so
  add them only when they earn it.
- **Distribution is a stack choice** — favour tech (like Go's single binary) that gets
  your program running in one step.
- **Hold it in one head** — the best solo stack is the smallest one that does the job.

Next up: turning that stack into a fast, comfortable place to work in
[your environment and daily workflow](/learn/intro-software-dev/dev-environment-workflow/).
