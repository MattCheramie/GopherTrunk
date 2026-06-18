---
slug: quality-when-solo
title: Keeping quality high alone
description: Use tests, linters, type checks and CI as the teammate you don't have — automated guardrails plus pre-commit hooks and self code review, kept realistically light.
keywords: solo quality, automated tests, continuous integration, pre-commit hooks, self code review, linters, type checks, guardrails, sustainable process
level: intermediate
status: full
faq:
  - q: "How does a solo developer review code with no one else?"
    a: "Two ways. First, you make **machines** do the mechanical review — tests, linters, type checks and CI catch the bugs and slips a human reviewer would. Second, you create distance from yourself: open a **pull request to your own project** and read it as a critic, and **sleep on** big changes so you see them with fresh, less attached eyes the next day. Distance is what a second reviewer really provides, and you can manufacture some of it deliberately."
  - q: "Is CI worth it for a one-person project?"
    a: "Usually yes, and it is cheap to set up. **Continuous integration** runs your tests and checks automatically on every push, so you find out within minutes if something broke — even on a machine that is not yours, catching \"works on my machine\" problems. It is the closest thing a solo dev has to a tireless teammate who checks every change without ever getting bored or distracted."
  - q: "How much process does a solo project actually need?"
    a: "Enough to catch real mistakes, and no more. A formatter, a linter, tests on the important parts, and CI cover most of the value. You do not need heavy ceremony, sign-off gates, or 100% test coverage. The goal is guardrails that protect you cheaply, not bureaucracy that slows you down — process should feel like a safety net, not a tax."
---
# Keeping quality high alone

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Automation is your reviewer** — tests, linters, type checks and CI catch what no human can. **Manufacture distance** — PRs to yourself and sleeping on changes. **Just enough process** — guardrails, not bureaucracy.
</div>

On a team, quality is partly someone else's job — a reviewer who spots your mistake, a
QA engineer who finds the crash, a CI server someone else maintained. Alone, all of
that is gone, and the temptation is to skip it and just "be careful." But you cannot
be careful at midnight on the hundredth change, and you have no second pair of eyes to
catch you when you slip. The solo answer is not heroic discipline — it is **building
guardrails that do the reviewing for you**, automatically, every time, forever. This
lesson is about assembling that automated teammate, plus a couple of human tricks to
review your own work, while keeping the whole thing realistically light.

## Automation is the teammate you don't have

Think of each automated check as a colleague who reviews every single change without
ever getting tired, bored or distracted. Four of them carry most of the weight:

- **Tests** — code that checks your code, proving the important behaviour still works
  after every change. They are how you refactor without fear and the closest thing to
  a QA engineer you have. The [testing lesson](/learn/intro-software-dev/testing/)
  goes deep on writing them well.
- **Linters** — flag likely bugs and risky patterns the moment you write them: unused
  values, ignored errors, suspicious comparisons.
- **Type checks** — in a typed language (or with optional typing added on), the
  compiler or checker catches whole categories of mistakes — wrong shapes, null
  surprises — before the program even runs.
- **Continuous integration (CI)** — runs all of the above automatically on every push,
  on a clean machine that is not yours, so nothing slips through and "works on my
  machine" gets caught.

The shift in mindset is this: you do not have to *remember* to check things or *be
sharp enough* to catch them. You set the guardrails up once, and from then on the
machine reviews every change for free. That is how a solo dev keeps quality high
without superhuman vigilance.

## Catch problems before they land: pre-commit hooks

A **pre-commit hook** runs checks automatically the instant you try to commit, and
refuses the commit if they fail. It is the earliest possible guardrail — problems are
stopped before they ever enter your history.

A sensible solo hook does the cheap, fast things:

```text
on git commit:
  format the code   →  if it changed anything, fix it
  run the linter    →  block on errors
  run quick tests   →  block on failures
```

Keep hooks **fast** — a hook that takes 30 seconds trains you to bypass it, which
defeats the purpose. Put the slow, thorough stuff (the full test suite,
cross-platform builds) in CI, where it can take its time after the push. Fast checks
locally, thorough checks remotely: that split keeps both your loop tight and your main
branch trustworthy.

## Manufacture the distance a reviewer gives you

The deepest value of a code reviewer is not their cleverness — it is their
**distance**. They did not just write the code, so they see it fresh, without the
author's blind spots and emotional attachment. You can deliberately recreate some of
that distance:

- **Pull requests to yourself.** Even alone, push your change to a branch and open a
  PR against your own main. Then read the diff *as a reviewer*, not as the author —
  the side-by-side view and the act of switching roles surfaces things you glide past
  in your editor. CI runs on the PR too, so you get the checks before you merge. (The
  [version control & collaboration lesson](/learn/intro-software-dev/version-control-collab/)
  covers the PR mechanics.)
- **Sleep on big changes.** The code that looked obviously correct at 2am often looks
  obviously wrong at 9am. For anything significant, let it sit overnight and reread it
  with fresh eyes before merging. Tomorrow-you is the second reviewer.
- **Rubber-duck it.** Explain the tricky change out loud, as if to someone else.
  Articulating it forces the gaps into the open.

None of these fully replace a real second person — but together they recover a
meaningful slice of what review provides, at zero cost.

<div class="knowledge-check" data-quiz data-correct-msg="Right — automated checks act as a tireless reviewer, catching bugs and slips on every change without a second person." markdown="0">
  <p class="knowledge-check__q">Quick check: how do tests, linters and CI help a solo developer most?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">They make the program run faster for end users</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">They act as a tireless automated reviewer, catching mistakes on every change with no second person</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">They remove the need to ever test the program by hand or think about design</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Keep it realistic — guardrails, not bureaucracy

It is easy to over-correct and bury a tiny project under team-sized process. Resist.
The goal is a safety net that protects you cheaply, not ceremony that slows you to a
crawl. A right-sized solo setup is roughly:

- **Formatter and linter** on save and on commit — near-zero cost, constant payoff.
- **Tests on the parts that matter** — the core logic, the tricky algorithm, the bits
  that would be embarrassing or dangerous to get wrong. You do **not** need to test
  every trivial getter, and you do not need 100% coverage.
- **CI on every push** — set up once, runs forever.
- **Self-review and sleeping on it** for anything non-trivial.

| Worth it for a solo dev | Probably overkill |
| --- | --- |
| Format + lint automatically | Mandatory multi-stage sign-off |
| Test the core and the risky bits | Chasing 100% coverage |
| CI runs tests on every push | Heavy release-approval ceremony |
| Self-PR + sleep on big changes | A formal QA team process |

Match the process to the stakes. A weekend utility needs less than software people pay
for or rely on for safety. The skill is dialing in *just enough* — enough to sleep
well, not so much that you stop shipping.

## A radio-tool example

Imagine your single-frequency logger from the previous lesson. The guardrails that
keep it solid are modest but real: a **test** that feeds it a known captured sample and
asserts it detects the signal at the right moment; a **linter** that nags you about the
error you forgot to handle when the radio device disconnects; and **CI** that builds
and tests it on Linux, macOS and Windows on every push — catching the path bug that
only shows up on Windows, which you would never have found on your own machine. That is
three small investments standing in for an entire QA team, and they let you change the
decoder with confidence instead of crossing your fingers.

## Recap

- **Automation is your reviewer** — tests, linters, type checks and CI catch on every
  change what a human reviewer would, tirelessly.
- **Pre-commit hooks** — stop problems before they enter your history; keep them fast.
- **Fast local, thorough remote** — quick checks on commit, the heavy suite in CI.
- **Manufacture distance** — PRs to yourself and sleeping on big changes recover much
  of what a second reviewer provides.
- **Right-size the process** — guardrails that protect you cheaply, not bureaucracy;
  skip 100% coverage and sign-off gates.
- **Match process to stakes** — a hobby tool needs less than software people rely on.

Next up: turning your solid v1 into something other people can install in
[packaging and distributing your software](/learn/intro-software-dev/packaging-and-distribution/).
