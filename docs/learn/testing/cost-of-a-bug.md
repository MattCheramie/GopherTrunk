---
slug: cost-of-a-bug
title: What does a bug cost?
description: The later a bug is found, the more it costs to fix — from seconds on your laptop to disasters in the field. The cost curve is the economic argument for everything else in this module.
keywords: cost of a bug, cost of defects, shift left testing, cost curve software, famous software bugs, ariane 5 bug, therac-25, why test early
level: beginner
status: full
prereq:
  - what-is-a-bug
---

# What does a bug cost?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The cost of a bug is dominated by **when it's found**, not how hard it was to
make. Caught while typing, it costs seconds; caught by a **test** on your
machine, minutes; caught in **code review** or **CI**, an hour of someone's day;
caught **in production**, it costs debugging under pressure, user trust, and
sometimes far worse. That steeply rising curve — often summarized as **shift
left**, find problems earlier — is the economic argument for the whole
discipline of testing.
</div>

Testing takes effort, so it's fair to ask what the effort buys. This lesson
answers with the single most useful economic fact in software engineering: an
identical defect has wildly different price tags depending on where along the
pipeline it's caught.

## The cost curve

Follow one defect — say, a wrong length check — through the places it could be
caught:

| Caught by | Time from defect to catch | Typical cost |
|-----------|--------------------------|--------------|
| Your editor / compiler | seconds | A red squiggle; you fix it without breaking flow |
| A unit test on your machine | minutes | Read one failure, fix one function |
| Code review | hours–a day | A reviewer's time, a re-push, some context reloading |
| CI on the pull request | hours–a day | Same, plus the queue's time |
| QA / a user's bug report | days–weeks | Reproduce from a vague description, re-learn code you've forgotten |
| Production incident | weeks–months later | Debug live under pressure, ship an emergency fix, repair data and trust |

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 190" role="img" aria-label="A curve rising steeply from left to right: cost of fixing a bug versus the stage where it is found, from editor through tests, review, CI, users, and production." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="160" x2="500" y2="160" stroke="currentColor" stroke-width="1.5"/>
  <line x1="40" y1="160" x2="40" y2="15" stroke="currentColor" stroke-width="1.5"/>
  <path d="M40 156 C 180 152, 300 140, 380 100 S 480 30, 495 20" fill="none" stroke="currentColor" stroke-width="2.5"/>
  <text x="18" y="90" font-size="12" fill="currentColor" transform="rotate(-90 18 90)">cost to fix</text>
  <text x="60" y="176" font-size="11" fill="currentColor">editor</text>
  <text x="130" y="176" font-size="11" fill="currentColor">tests</text>
  <text x="205" y="176" font-size="11" fill="currentColor">review</text>
  <text x="280" y="176" font-size="11" fill="currentColor">CI</text>
  <text x="340" y="176" font-size="11" fill="currentColor">users</text>
  <text x="420" y="176" font-size="11" fill="currentColor">production</text>
</svg>
<figcaption>The same defect gets more expensive at every stage it survives — the curve every testing practice exists to flatten.</figcaption>
</figure>

Two things drive the curve. First, **context**: the moment you write a defect,
everything about that code is loaded in your head; weeks later, you've paged it
all out and must reconstruct it. Second, **blast radius**: a defect on your
laptop affects you; a defect in production affects every user, every downstream
system, and any data it touched while nobody was looking.

## Famous waypoints on the curve

The far right of the curve is not hypothetical:

- **Ariane 5 (1996).** Europe's new rocket self-destructed 37 seconds after
  launch. A number too large for a 16-bit slot — in navigation code inherited
  from the older, slower Ariane 4, where the value could never get that big —
  crashed the guidance computer. Roughly half a billion dollars, from one unhandled
  conversion whose *assumption* stopped holding when the environment changed.
- **Therac-25 (1985–87).** A radiation-therapy machine delivered massive
  overdoses when operators typed quickly enough to trigger a race condition;
  patients died. The software had been "working" for years — the defect was
  latent, waiting for a rare timing.
- **Mars Climate Orbiter (1999).** One team's software produced thrust figures in
  pound-seconds, another consumed them as newton-seconds. Every piece worked in
  isolation; the *interaction* was wrong, and the spacecraft burned up in the
  Martian atmosphere. An integration test comparing real units across the
  boundary would have caught it on the ground.

You will probably never write rocket code. But scale the stakes down and the
shape survives perfectly: the bug caught by tonight's test run costs you a
coffee; the same bug caught by an annoyed user next month costs you an evening,
a reputation dent, and an apology.

## "Shift left" — the strategy the curve implies

Draw the pipeline left-to-right — write, test, review, integrate, release,
operate — and the curve says: **move discovery as far left as you can afford**.
That's the entire strategy of this module, unit by unit:

- **Unit tests** (Unit 2) catch defects minutes after they're written.
- **Linters and static analysis** (Unit 4) catch some *before the code even runs*.
- **Code review and CI** (Unit 4) catch what one person's blind spot let through,
  still before merge.
- **Regression tests** (Unit 3) keep a bug that cost you dearly once from ever
  charging you twice.

> Rule of thumb: the cheapest bug is the one caught before it leaves your
> machine. Every practice in this module is a way of buying catches further left.

There's a balance, of course — you can over-invest in testing trivial code, and
[code coverage](/learn/testing/code-coverage/) will have things to say about
chasing numbers for their own sake. But beginners almost universally err in the
other direction, paying production prices for laptop-priced bugs.

<div class="knowledge-check" data-quiz data-correct-msg="Right — when it's found is the dominant factor; the same defect gets orders of magnitude more expensive as it travels right." markdown="0">
  <p class="knowledge-check__q">Quick check: what most determines how much a given defect ends up costing?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">How late in the pipeline it's discovered</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">How many lines of code the fix eventually touches</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Whether it was a typo or a logic error</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A bug's cost is set mostly by **when it's found** — the cost curve rises
  steeply from editor to production.
- Rising cost is driven by **lost context** (you've forgotten the code) and
  **blast radius** (more people and data affected).
- Ariane 5, Therac-25, and Mars Climate Orbiter are the curve's far-right
  waypoints — latent defects and broken interactions at full price.
- **Shift left**: every practice in this module moves bug discovery earlier,
  where the same catch is orders of magnitude cheaper.
- The cheapest bug you'll ever fix is the one your own **test** catches tonight.

Next up: [What is testing?](/learn/testing/what-is-testing/)
