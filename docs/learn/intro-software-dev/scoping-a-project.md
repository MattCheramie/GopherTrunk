---
slug: scoping-a-project
title: From idea to project
description: Turn a vague idea into a v1 you can finish — MVP thinking, ruthless scope-cutting, a short spec, a sensible repo, and milestones, with a worked SDR example.
keywords: project scoping, MVP, minimum viable product, cutting scope, writing a spec, README first, milestones, scope creep, solo project
level: intermediate
status: full
faq:
  - q: "What is an MVP?"
    a: "An **MVP**, or minimum viable product, is the smallest version of your idea that actually delivers value to a user. It is not a half-built mess — it is a complete, useful thing with a deliberately narrow scope. The point is to finish and ship something real quickly, learn from it, and expand from there rather than spending months on features no one has asked for yet."
  - q: "How do I stop scope from creeping?"
    a: "Write down what v1 **is and is not** before you start, and treat that list as a contract. When a tempting new idea appears, do not build it — add it to a \"later\" list. Most scope creep comes from saying yes in the moment. A written, deliberately small scope gives you something concrete to say no against, which is the only reliable defence when you are alone."
  - q: "Should I write a spec before coding?"
    a: "A short one, yes. A page or two — often just a README written first — that says what the thing does, who it is for, and what v1 includes, forces you to think before you type and gives you a target to aim at. It does not need to be formal or long. The act of writing it usually reveals that half your planned features are not actually needed for v1."
---
# From idea to project

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Define a finishable v1** — the smallest version that is genuinely useful. **Cut ruthlessly** — write down what v1 is *not*. **Spec first** — a short README and milestones beat diving straight into code.
</div>

The graveyard of solo projects is full of brilliant ideas that were never finished —
not because the developer lacked skill, but because they never defined a version 1
small enough to reach. Working alone, your scarcest resource is *finishing energy*,
and the surest way to run out of it is to aim at the whole grand vision at once. This
lesson is about the opposite discipline: shrinking an idea down to something you can
actually ship, writing just enough of a plan to aim at it, and ruthlessly defending
that boundary against your own enthusiasm. Get this right and the rest of the module —
quality, packaging, shipping — has something real to act on.

## Define a v1 you can actually finish

Every idea has a grand version: all the features, all the polish, all the
platforms. **That is not your first goal.** Your first goal is a **v1** — the smallest
version that is genuinely useful to one real person (even if that person is you). This
is **MVP thinking**: a minimum *viable* product is complete and useful, just narrow.

The test for whether something belongs in v1 is brutal and clarifying:

- *Without this, is the program still useful to someone?* If yes, it is not v1 — it is
  later.
- *Is this the one core thing the program is fundamentally for?* If yes, it is v1.

Almost everything fails the first test. That is the point. A v1 you can finish in
weeks and put in front of users teaches you more than a v3 you imagine for months and
never ship. **Shipping something small is progress; planning something huge is not.**

## Cut scope ruthlessly — and write down what you cut

The single most valuable artefact in early scoping is not a list of features to
build. It is a list of features you are **deliberately not building yet.** Writing
"v1 does NOT include X, Y, Z" turns vague restraint into a concrete decision you can
point at later.

This matters because of two classic, opposite traps:

- **The infinite-polish trap.** Endlessly refining what you have — another setting,
  another edge case — because finishing and shipping feels scarier than tinkering.
  Polish is a stalling tactic in disguise.
- **The rewrite-everything trap.** Convincing yourself the whole thing must be torn
  down and rebuilt "properly" before it is good enough to ship. It almost never must.
  Ship the imperfect thing, then improve it in place.

Both traps are forms of avoiding the finish line. A written, intentionally small scope
is your defence against both, because it tells you, in your own earlier words, when
you are done.

## Write the README first

Before you write code, write the **README** — the document that explains what the
thing is. Writing it *first* (sometimes called "readme-driven development") is a
forcing function: you cannot describe a product you have not actually thought through.

A first README needs only:

- **What it does**, in one or two sentences a stranger would understand.
- **Who it is for** and the problem it solves for them.
- **What v1 includes** — and a short "not yet" list of what it does not.
- A rough idea of **how someone will run it.**

```text
# Tinylog
A tiny command-line tool to log activity on one radio frequency to a CSV file.

v1: tune one frequency, detect activity, append timestamped rows.
Not yet: scanning multiple frequencies, audio recording, a GUI.
```

Half the time, writing this reveals that several of your "essential" features are
nowhere near essential. That realization, before you have written a line of code, is
worth an hour of typing many times over.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an MVP is the smallest complete, useful version, built to ship and learn from quickly rather than to be impressive." markdown="0">
  <p class="knowledge-check__q">Quick check: what is the goal of defining an MVP for your v1?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">To include every feature you can think of so users are never disappointed</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">To ship the smallest genuinely useful version quickly, then learn and expand from real use</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">To spend as long as possible perfecting it before anyone sees it</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Structure the repo and set milestones

A little structure up front keeps a small project from turning into a single
1,000-line file you are afraid to touch. You do not need elaborate architecture — just
sensible separation:

- A clear **entry point** (where the program starts).
- Logic grouped into a few **modules or packages** by responsibility, not crammed
  together.
- Tests living **alongside the code** they cover.
- The **README**, a license, and config at the top level where people expect them.

Then break the road to v1 into a few **milestones** — concrete, checkable steps, each
of which leaves the program in a working state:

1. It runs and prints something.
2. It does the core thing once, by hand.
3. It does the core thing reliably, with tests.
4. It is packaged so someone else can run it.

Milestones turn "build the whole thing" — which is paralysing — into "finish the next
small thing," which is doable. Each one is a little win that keeps momentum alive,
which for a solo dev is half the battle.

## Worked example: scoping a radio logger

Suppose you want to build "a software-defined-radio scanner" — monitor a whole band,
record audio, identify systems, the works. That is a wonderful v3 and a terrible v1.
Apply the discipline above and shrink it hard:

- **The grand vision:** a multi-frequency scanner with audio recording, a database of
  systems, a live waterfall display, and trunk-tracking.
- **The honest v1:** *a single-frequency activity logger.* Tune to **one** frequency,
  detect when there is activity, and append a timestamped row to a file. That is it.

Look at what that v1 deliberately does **not** include: no scanning across
frequencies, no audio capture, no display, no database. Each of those is a real
feature for later — and each would have stretched v1 from weeks into never. The single
frequency logger is finishable, genuinely useful (people want activity logs), and it
forces you to solve the actual hard part — talking to the radio and detecting a signal
on captured [IQ samples](/learn/rf-sdr/iq-data/) — without the distraction of ten
other features.

Once that logger works, ships, and you have used it, *then* you expand: a second
frequency, then a list, and you are on the road to a real scanner — built incrementally
on a foundation that already works, rather than imagined whole and never reached. If
you want to go deep on the radio side of such a project, the
[RF & SDR path](/learn/rf-sdr/) is the companion to this one. This incremental, ship-then-grow
path is exactly how a project like GopherTrunk gets from a single useful feature to a
full toolkit.

## Recap

- **Aim at a finishable v1** — the smallest version that is genuinely useful, not the
  grand vision.
- **MVP thinking** — small and complete beats large and unfinished; ship to learn.
- **Cut ruthlessly and write it down** — a "v1 does not include…" list is your defence
  against scope creep.
- **Avoid both traps** — infinite polish and total rewrites are both ways of dodging the
  finish line.
- **README first** — writing what it is before coding reveals which features are
  actually essential.
- **Milestones over mountains** — break the path into small, working steps that each
  leave the program runnable.

Next up: keeping that v1 solid without a team to review it in
[keeping quality high alone](/learn/intro-software-dev/quality-when-solo/).
