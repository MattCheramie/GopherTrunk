---
slug: code-review
title: Code review
description: A second pair of eyes before code merges — what review catches that tests can't, how to review a change well, and how to receive review without bruises.
keywords: code review, pull request review, how to review code, review checklist, receiving code review, review culture, what review catches
level: beginner
status: full
---

# Code review

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Code review** puts a second human between a change and the main branch —
usually via a **pull request**. It catches what no machine can: wrong
**assumptions**, missing **cases**, designs that will hurt later, and tests
that don't test. Good reviewers **start with the tests**, ask questions rather
than issue verdicts, and match depth to **risk**; good authors keep changes
**small**, explain the *why*, and treat feedback as **about the code, not the
coder**. Review is also how a team teaches itself — knowledge moves in both
directions.
</div>

Machines have carried the unit so far: formatters settle style, linters catch
known-wrong patterns, tests verify chosen behaviors. This lesson is the layer
only a human can provide — and the practice that most shapes what working on a
software team actually feels like.

## What review catches that machines can't

Every mechanical check shares one blind spot: it verifies the code against
some *stated* expectation — the test's assertion, the linter's rule. None of
them can ask whether the expectation itself is right. Humans can:

- **Wrong assumptions.** "This assumes talkgroup IDs fit in 16 bits — the
  protocol allows 24." The code is internally consistent; a test written by
  the same author would share the same assumption. A second person brings a
  *second model of the world*, which is exactly what the
  [self-consistent trap](/learn/testing/the-self-consistent-synthetic-trap/)
  requires.
- **Missing cases.** "What happens when two calls start in the same instant?"
  — the reviewer's [breaker brain](/learn/testing/the-testing-mindset/) runs
  against the author's builder brain.
- **Tomorrow's problems.** This design works today and walls off next
  quarter's feature; this name will mislead the next reader. No test fails on
  a future cost.
- **Tests that don't test.** Assertion-free tests, doubles that assume the
  answer, a "regression test" that would pass against the broken code —
  review is where test *quality* gets checked.

There's a second product besides caught bugs: **knowledge transfer**. Review
spreads familiarity with the codebase in both directions — the junior learns
idioms from the senior's comments, the senior learns what's become confusing
from the junior's questions — and it's most teams' main defense against
one-person-knows-this-module fragility.

## Reviewing well

A working sequence for reviewing a change (on GitHub, a
[pull request](/learn/git/pull-requests/) — the Git module covers the
mechanics):

1. **Read the description first.** What problem does this claim to solve? A
   change you can't state the purpose of can't be reviewed, only skimmed.
2. **Read the tests next.** They're the author's most honest statement of what
   was verified — and their gaps ("no truncated-input case?") are your best
   comments.
3. **Then the code**, asking the review questions: does this do what it
   claims? What inputs break it? Is it clear to the next reader?
4. **Comment as questions where honest ones exist.** "What happens if `frames`
   is empty?" both respects that the author may know something you don't and
   teaches more than "add a nil check."
5. **Match depth to risk.** A typo fix needs a glance; new decoding logic in a
   core path deserves an hour. Uniform-depth review is depth-starved review.

> Rule of thumb: review the change the author *made*, not the change you
> *would have made*. "Different from my taste" is not a defect; "breaks on
> empty input" is.

## Being reviewed well

The other chair has skills too, and they're mostly about lowering the cost of
reviewing you:

- **Keep changes small and single-purpose.** A 200-line focused change gets a
  real review; a 2,000-line grab-bag gets a weary approval. Review quality is
  inversely proportional to diff size — and mixing refactors into
  behavior-change diffs (a thing GopherTrunk's contribution guidelines
  explicitly prohibit) makes both halves unreviewable.
- **Write the description you'd want to read**: the problem, the approach, and
  how you verified it — plus where you're *unsure*, which directs reviewer
  attention to exactly the right place.
- **Take feedback as information about the code**, not a grade on you. Half of
  review comments are misunderstandings — which are themselves findings: if
  the reviewer misread it, the next maintainer will too.
- **Say thanks when review catches something.** A culture where finding a bug
  pre-merge is a *win for both people* is a culture where reviews stay honest.

## Review is a gate, not a suggestion

Like every check in this unit, review only protects what it's *required* for.
Most teams enforce it mechanically: the platform blocks merging until an
approval lands — one of the [required checks](/learn/testing/required-checks/)
you'll meet at the end of this unit, alongside CI. The point isn't distrust of
any individual; it's the same humility the whole module runs on — *everyone's*
blind spots are real, so every change gets a second pair of eyes, no matter
whose it is.

<div class="knowledge-check" data-quiz data-correct-msg="Right — tests are the author's statement of what was verified, and their gaps are usually the most valuable review findings." markdown="0">
  <p class="knowledge-check__q">Quick check: why do experienced reviewers read a change's tests before its code?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Tests are shorter, so it feels like progress</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Tests reveal what the author actually verified — and what they didn't — which directs the whole review</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Test files are the only part machines can't check for style</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Review** catches what machines can't: wrong **assumptions**, missing
  cases, future costs, and **tests that don't test**.
- Its second product is **knowledge transfer** — the team teaching itself,
  both directions.
- Reviewers: description → **tests first** → code; question honestly; match
  **depth to risk**; review the change made, not the one you'd have made.
- Authors: **small single-purpose changes**, a real description, feedback
  taken as data about the code.
- Enforce it as a **gate** — a required approval, alongside CI — because
  optional checks protect nothing.

Next up: [Continuous integration](/learn/testing/continuous-integration/)
