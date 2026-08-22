---
slug: capture-gated-verification
title: Capture-gated verification
description: A fix isn't verified until it works against real captured data — why GopherTrunk refuses to close a bug on a green synthetic test alone, and what "verified" is allowed to mean.
keywords: capture gated verification, verified fix, closing bugs policy, real data validation, regression verification, reporter confirmation, issue closing discipline
level: advanced
status: full
prereq:
  - regression-tests
  - the-self-consistent-synthetic-trap
---

# Capture-gated verification

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Closing a bug is a **claim that the reported problem is gone** — and
GopherTrunk's policy is to refuse that claim until it's **earned twice over**:
a **failing-first regression test** now passes, **and** the original symptom
is shown gone against **real captured data** (or confirmed by the reporter).
One green alone isn't enough, because the synthetic test can share the bug's
own wrong assumption — the previous lesson's trap, applied to *fixes*. Until
both halves land, the issue **stays open** with an honest status note. The
principle generalizes: **"verified" means the real-world symptom died, not
that your model of it did.**
</div>

The last lesson showed how tests lie when both sides share an assumption.
This one is the policy GopherTrunk built so that *fix claims* can't lie the
same way — a discipline that grew directly out of debugging history where
"fixed" turned out to mean "our model of it stopped failing."

## What closing a bug actually claims

File this under words meaning things. Marking an issue *fixed* doesn't claim
"we changed code thoughtfully" or "a test we wrote now passes." It claims
**the problem the reporter experienced no longer occurs**. That's a statement
about the world, not about your code — and the module has already shown the
two ways a green test can fail to support it:

- The test reproduces *your model* of the bug, not the bug — pass the model,
  miss the symptom. ([Reproduction](/learn/testing/reproducing-a-bug/)
  lesson's core warning.)
- The test is [self-consistent](/learn/testing/the-self-consistent-synthetic-trap/)
  with the fix — both built on the same misunderstanding, both move together.

GopherTrunk learned this the expensive way: bugs closed on plausible,
green-tested fixes whose symptom was still live in the field — including one
closed *twice* on fixes that touched the wrong code path entirely (the fix
repaired one down-conversion path while the reporter's replay symptom lived
in a parallel one). Each premature close costs more than the bug: the
reporter re-reports, trust erodes, and the *next* investigator starts from
"supposedly fixed," the worst possible prior.

## The two-part gate

So the project's standing policy makes "verified" a conjunction. A fix may be
called verified when:

1. **A failing-first regression test passes.** The
   [Unit 3 discipline](/learn/testing/regression-tests/), strictly applied:
   the test failed against the unfixed code — proving it detects *something*
   real — and passes with the fix.
2. **The original symptom is shown gone against reality**: the fix is run
   against the **reporter's own captures** (or the reporter confirms on
   their live system) and the reported behavior — the garble, the lost sync,
   the missing audio — no longer occurs.

Part 1 without part 2 is the trap lesson's loophole: your test might encode
your theory of the bug, and your theory might be wrong even when your code
change is locally sensible. Part 2 without part 1 is
[reproduction's](/learn/testing/reproducing-a-bug/) loophole in reverse — the
symptom vanished this time, but nothing guards it and nothing proves your fix
(rather than luck) killed it. Together they close both: the capture anchors
the claim to reality, the regression test pins it there permanently.

The [replay machinery](/learn/testing/replay-integration-tests/) is what
makes part 2 *practical*: "run against the reporter's reality" would be
wishful without capture files, and it's a command with them. Notice the whole
Unit 6 stack interlocking — replay makes real data testable, independence
makes tests honest, and this policy makes *closure* honest.

## Until verified: stay open, say why

The policy's other half is what happens when you *can't* verify — no capture
obtainable, symptom not yet reproduced, reporter gone quiet:

- **The issue stays open.** An unverified fix can merge — as an improvement,
  a probable fix, an experiment — but the *claim* waits. In pull requests
  this shows up as a deliberately weak linking word: reference the issue
  without auto-closing it (`Refs #N` rather than `Closes #N` on GitHub), so
  a merge can't quietly assert what nobody verified.
- **A status comment says exactly where things stand**: what was found, what
  was changed, what's still needed ("needs the reporter's capture files to
  confirm"). Honest state beats optimistic state — the next person to touch
  the issue inherits facts, not vibes.
- **Follow-ups get addressed as themselves.** When a reporter comes back
  with "still happening," the answer engages the *new* evidence — not a
  re-posting of the original fix rationale. A symptom that survives the fix
  is a fresh fact, and per
  [root-cause discipline](/learn/testing/root-cause-analysis/), it usually
  means the theory was incomplete.

GopherTrunk enforces the spirit mechanically where it can — closing an issue
as completed triggers a confirmation step, a
[required-checks-style](/learn/testing/required-checks/) speed bump on
exactly the action that historically went wrong. (Closing as *not planned* or
*duplicate* isn't gated: those close conversations, not symptoms.)

> Rule of thumb: your fix is an hypothesis until reality — a capture, a
> reporter, a field run — has voted on it. Merge hypotheses freely; *claim*
> only verdicts.

The transferable policy for any project: define what "verified" means
*before* the pressure to close arrives, make it require evidence from
outside the fixer's own model, and make unverified-but-improved a normal,
sayable state.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the green test may encode the same wrong theory as the fix; only the reporter's real data (or confirmation) shows the actual symptom is gone." markdown="0">
  <p class="knowledge-check__q">Quick check: a bug-fix PR includes a regression test that failed before the fix and passes after. Under capture-gated verification, why isn't that alone enough to close the issue?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Because regression tests are too slow to trust in CI</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Because policy requires two reviewers on every bug fix</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Because the test may reproduce the fixer's theory of the bug rather than the reporter's symptom — reality hasn't voted yet</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Closing a bug claims **the reported problem is gone** — a fact about the
  world, which green tests alone can't establish.
- The gate is a **conjunction**: failing-first regression test passes **and**
  the symptom is shown gone against **real captures** or by the reporter.
- Each half closes the other's loophole: the capture defeats
  self-consistency; the test makes the verification **permanent**.
- Until both land: **stay open**, link weakly (`Refs`, not `Closes`), post
  honest status, and answer follow-ups on their **new** evidence.
- Portable principle: **verified = reality voted** — define it before
  closing pressure arrives, and gate the claim, not the code.

Next up: [Write your first regression test](/learn/testing/your-first-regression-test/)
