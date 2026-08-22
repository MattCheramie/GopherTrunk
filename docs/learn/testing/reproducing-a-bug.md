---
slug: reproducing-a-bug
title: Reproducing a bug
description: A bug you can reproduce on demand is already half fixed — building minimal reproductions, shrinking them, and why "works on my machine" is a clue about the environment, not a verdict.
keywords: reproducing bugs, minimal reproduction, repro steps, works on my machine, deterministic reproduction, bug report reproduction, shrinking a test case
level: intermediate
status: full
faq:
  - q: Why is reproducing a bug considered half the work of fixing it?
    a: Because a reliable reproduction converts debugging from guesswork into experiment. Once you can trigger the failure on demand, you can bisect the cause, test hypotheses in minutes, and — crucially — know with certainty whether a candidate fix worked. Without reproduction you're changing code and hoping, and "the symptom didn't happen this time" is indistinguishable from "the bug is gone."
  - q: What is a minimal reproduction?
    a: The smallest input, code path, and environment that still triggers the bug — found by repeatedly cutting the original failing scenario in half and keeping whichever half still fails. Each removed element eliminates a suspect; what survives the shrinking is a shortlist of what actually matters, and often points straight at the defect.
  - q: What does "works on my machine" actually tell you?
    a: That the bug depends on something that differs between the two environments — a version, a file, a locale, timing, configuration, or data. Rather than closing the report, diff the environments; the difference that matters is part of the bug's trigger condition, and finding it is genuine progress toward the cause.
---

# Reproducing a bug

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Debugging starts with **reproduction**: making the failure happen **on
demand**. Until then you can't test hypotheses and you can't know a fix
worked — "didn't happen this time" proves nothing. The craft is **shrinking**:
cut the failing scenario in half repeatedly, keep the half that still fails,
until a **minimal reproduction** points at the cause. **"Works on my
machine"** is data, not a verdict — the bug's trigger lives in whatever
differs between machines. And a finished reproduction is a
**failing-first regression test** waiting to be committed.
</div>

Unit 5 begins where the safety nets end: something is broken anyway. Every
debugging technique in the next five lessons stands on the discipline in this
one — because until a failure happens when *you* decide, you're not debugging;
you're wishing.

## Why reproduction comes first

A bug you can trigger on demand transforms the whole problem:

- **You can experiment.** Form a hypothesis, change one thing, re-trigger, and
  observe — minutes per cycle. Without reproduction, each "test" is waiting
  days for the symptom to maybe recur in the wild.
- **You can verify a fix.** This is the decisive one. Reproduce → apply fix →
  fail to reproduce: that's evidence. No reproduction means "we changed
  something and the symptom hasn't been seen since" — which is
  indistinguishable from the bug still being there on its original, lazy
  schedule. This is precisely why GopherTrunk's fix policy says: *if you can't
  write a test that fails, you haven't reproduced the bug — keep digging,
  don't guess at a fix.*
- **You can hand it to others.** A reproduction script travels; "it crashed
  yesterday around noon" doesn't.

## The shrinking loop

Real failure reports arrive huge: a 40-minute session, a 2 GB input, a config
touching everything. The path from there to understanding is **binary-search
shrinking** — the same instinct as
[property-test shrinking](/learn/testing/property-based-testing/), done by
hand:

1. Confirm the full scenario fails.
2. **Cut something in half** — the input, the config, the steps, the elapsed
   time.
3. Still fails? The removed half was irrelevant — permanent progress. Passes
   now? The trigger lives in what you removed — put it back, cut elsewhere.
4. Repeat until nothing more can be removed.

Two habits make the loop fast: **change one variable per iteration** (change
two and a result blames neither), and **write down each result** — memory
lies, and by iteration fifteen "did plain FM audio also crash?" needs an
answer, not a feeling. What survives shrinking is a shortlist of what
*matters*: if a decoder crash survives shrinking a 40-minute capture down to
0.8 seconds and one truncated burst, you've practically named the defect
before opening the code.

Determinism is the other shrinking axis. Failures that come and go usually
hinge on timing, ordering, or randomness — pin each: seed the randomness (as
[property tests do](/learn/testing/property-based-testing/)), replay recorded
input instead of live input, force the suspicious ordering explicitly. Every
pinned variable either kills the mystery ("seeded → always fails" — it was the
random path) or eliminates a suspect.

## "Works on my machine" is a clue

The report reproduces for the user and not for you. The wrong response is a
verdict ("can't reproduce, closing"); the right one is noticing you've been
handed a fact: **the trigger depends on an environmental difference**. Diff
the environments like a suspect list — versions (program, OS, dependencies),
configuration and data files, locale and timezone, hardware, timing (their
machine slower? loaded?). The difference that matters, once found, is part of
the reproduction: "crashes only when the config lacks a `talkgroups` file" is
a *better* reproduction than one machine's mystery, and its discoverer
understands the bug better than anyone.

This is also why *reporters* are half the debugging team. The habits that make
your own reports reproducible — exact versions, exact input, exact steps,
expected vs actual — are the same list you wish every bug filed against *you*
included. For radio software the gold standard is the raw capture file: an IQ
recording turns "TETRA sounded garbled last night" into a file that fails the
same way every time, on any developer's desk. Which is exactly the machinery
[Unit 6 builds](/learn/testing/replay-integration-tests/).

> Rule of thumb: the endpoint of reproduction is not a memory — it's an
> artifact. A script, a fixture file, a test. If it isn't runnable, it isn't
> finished.

And the best artifact is a **test**: the minimal reproduction, expressed as a
[failing-first regression test](/learn/testing/regression-tests/), is
simultaneously your experiment rig while fixing and the permanent guard after.

<div class="knowledge-check" data-quiz data-correct-msg="Right — without a reproduction, a quiet symptom is indistinguishable from a bug on its own schedule; only reproduce → fix → fail-to-reproduce is evidence." markdown="0">
  <p class="knowledge-check__q">Quick check: why can't you trust a fix for a bug you never reproduced?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Unreproduced bugs are usually hardware problems, not software</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">"The symptom didn't happen this time" looks identical whether the fix worked or the bug is just waiting</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Fixes applied before reproduction don't compile cleanly</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Reproduce first**: on-demand failure is what turns debugging into
  experiment — and it's the only way to *know* a fix fixed.
- **Shrink by halves**, one variable at a time, writing results down; what
  survives names the trigger.
- **Pin nondeterminism** — seeds, recorded input, forced orderings — until the
  failure is reliable.
- **"Works on my machine"** means the trigger hides in an environmental
  difference; diff the environments, don't close the report.
- Finish with an **artifact** — ideally the minimal reproduction as a
  failing-first regression test.

Next up: [Reading error messages & stack traces](/learn/testing/reading-error-messages/)
