---
slug: bisecting-history
title: Bisecting history
description: git bisect turns "it broke sometime this month" into the exact commit, in a dozen automatic steps — binary search over history, fully automated with a test script.
keywords: git bisect, git bisect run, find breaking commit, binary search git history, regression hunting, bisect automation, when did it break
level: intermediate
status: full
prereq:
  - reproducing-a-bug
---

# Bisecting history

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
When something **used to work**, version history holds the answer: some single
commit broke it. **`git bisect`** binary-searches history — mark a **good**
commit and a **bad** one, test the midpoint, halve the range, repeat — finding
the culprit among ~1,000 commits in **~10 tests**. With a script that exits
0/non-zero, **`git bisect run`** does the whole hunt **unattended**. The
prerequisites are the module's other disciplines: a **reproduction** you can
run per commit, and the small, always-building commits good practice produces.
</div>

Some bugs you diagnose forward, from the code. This lesson's tool diagnoses
*backward, from history* — and for one common situation it beats everything
else in this unit: the regression, where the only thing you know is that it
worked before and doesn't now.

## The insight: history is a sorted array

"It worked three weeks ago" plus "it's broken now" brackets the failure: in
the commit sequence between those two points, there's a **first bad commit**,
and every commit after it is bad too (good-good-good-bad-bad-bad). A sorted
boundary like that is binary-searchable: test the **middle** commit — if it's
good, the break is in the newer half; if bad, the older half. Either way the
suspect range just **halved**. Powers of two do the rest:

| Commits in range | Tests needed |
|------------------|--------------|
| 16 | 4 |
| 250 | 8 |
| 1,000 | ~10 |
| 16,000 | ~14 |

A month of a busy repo's history, narrowed to one commit, in a dozen checks.
And the payoff of finding it is huge: the diff of one commit is a
*shortlist of suspect lines*, plus the author, the message, and the intent —
often the diagnosis practically reads itself.

## Driving git bisect

Git automates the bookkeeping — the checkout choreography, the range
tracking — while you supply verdicts:

```bash
git bisect start
git bisect bad                  # current commit is broken
git bisect good v1.4.0          # this tag was fine

# git checks out the midpoint. Test it (run your reproduction), then:
git bisect good                 # ...if it worked
git bisect bad                  # ...if it failed
# repeat: git keeps halving until —

# 3f2a91c is the first bad commit
#   decoder: rework burst length handling

git bisect reset                # return to where you started
```

Each round: git checks out a commit, you run your
[reproduction](/learn/testing/reproducing-a-bug/), you pronounce `good` or
`bad`. That's the whole interface — plus `git bisect skip` for a commit that
won't build, and the [Git module's lesson](/learn/git/cherry-pick-bisect-reflog/)
for the surrounding mechanics.

## The killer feature: `git bisect run`

If your reproduction is a *command* — and after this module it should be: a
test is exactly that — the human leaves the loop entirely. Any script that
exits `0` for good and non-zero for bad can drive the search:

```bash
git bisect start
git bisect bad
git bisect good v1.4.0
git bisect run go test ./decoder/ -run TestDecodeBurst_TruncatedFrame
```

Git now checks out, runs, judges, and halves — unattended — and prints the
first bad commit a few minutes later. This is the module's ideas compounding:
the failing-first [regression test](/learn/testing/regression-tests/) you
wrote to pin the bug is *also* the oracle that finds the commit that caused
it. Write once, use as reproduction, bisect oracle, and permanent guard.

```bash
# The reproduction can be anything executable — a replay of a capture:
git bisect run ./scripts/check_capture_decodes.sh site_capture.raw
```

## What bisect quietly assumes — and repays

Bisect's preconditions are the very habits earlier lessons argued for, which
is why it doubles as an argument for them:

- **A runnable reproduction.** Bisect asks "good or bad?" dozens of times;
  only an on-demand check can answer. No reproduction, no bisect.
- **Commits that build.** A history where half the commits don't compile
  yields `skip` after `skip`, and wide skip-gaps can leave a range of
  candidates instead of one culprit. This is the concrete payoff of the
  [Git module's](/learn/git/best-practices/) "every commit should build"
  discipline.
- **Small, single-purpose commits.** Bisect names a *commit*; the diagnosis
  quality depends on its size. A 40-file "misc changes" culprit leaves you
  nearly where you started — a 3-file focused commit hands you the bug. (The
  same property that made [review](/learn/testing/code-review/) effective.)
- **A deterministic check.** A [flaky](/learn/testing/flaky-tests/) oracle
  poisons the search: one wrong verdict sends the binary search into the
  wrong half, and the final answer is confidently wrong. If the failure is
  probabilistic, make the script run the test N times and judge the batch.

> Rule of thumb: when the report says "it used to work," reach for bisect
> *before* reading code — ten minutes of mechanical search often replaces an
> afternoon of theorizing, and hands you the diff, the author, and the intent.

<div class="knowledge-check" data-quiz data-correct-msg="Right — binary search halves the range each test: 1,000 commits collapse in about ten checks, which is why bisect makes big histories searchable." markdown="0">
  <p class="knowledge-check__q">Quick check: roughly how many test runs does git bisect need to find one bad commit among 1,000?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">About 500 — half of them on average</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">About 10 — each test halves the remaining range</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">All 1,000 — each commit must be checked in order</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A regression means history contains a **first bad commit** — a sorted
  boundary, findable by **binary search**.
- **`git bisect`** runs the search: mark good and bad, judge midpoints, ~10
  tests per 1,000 commits.
- **`git bisect run <cmd>`** automates the verdicts — your failing regression
  test doubles as the oracle.
- The found commit's **diff, author, and message** turn "somewhere this month"
  into a near-complete diagnosis.
- Bisect repays the module's habits: runnable reproductions, deterministic
  checks, and **small commits that build**.

Next up: [Root-cause analysis](/learn/testing/root-cause-analysis/)
