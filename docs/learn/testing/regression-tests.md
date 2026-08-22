---
slug: regression-tests
title: Regression tests & failing first
description: Every bug fix ships with a test that failed before the fix and passes after it — the failing-first discipline that stops bugs from coming back, and proves you actually understood the bug.
keywords: regression testing, failing first test, regression test go, bug fix testing, test that fails first, prevent regressions, red green testing
level: intermediate
status: full
prereq:
  - what-is-a-bug
  - your-first-go-test
---

# Regression tests & failing first

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **regression** is a bug in behavior that used to work; a **regression test**
pins a fixed bug's exact triggering input into the suite so the bug can never
silently return. The discipline that makes it real is **failing first**: write
the test *before* the fix and watch it **fail against the broken code** — proof
you reproduced the bug — then watch the fix turn it green. A test that never
failed proves nothing; a fix without a test is a bug on layaway.
</div>

This lesson is the hinge of the module. Everything before it taught you to
write tests; this one teaches the single practice that most changes how bugs
get fixed — and it's the rule GopherTrunk's contribution policy is built
around: *a bug fix is one narrow commit plus a regression test that fails
without the fix and passes with it.*

## Fixed bugs come back

It sounds paranoid until you've watched it happen. A bug is fixed in March.
In September, someone refactors the function — carefully, test suite green —
and reintroduces the same defect, because nothing in the code *says* why that
odd-looking length check exists. The bug returns; a user re-reports it; someone
re-diagnoses it from scratch. Every hour of the original debugging is spent
again.

The cause is structural, not personal: a fix encodes hard-won knowledge ("this
input occurs, and this is how it must be handled"), and code alone can't
protect that knowledge from future edits. A **regression test** can. It carries
the bug's exact triggering input and correct expected output — so any change
that resurrects the defect turns the suite red in minutes, with a test whose
name points straight at the original bug.

## Failing first: the proof you understood the bug

Order matters more than it looks. The discipline:

1. **Reproduce the bug as a test.** From the bug report, construct the input
   that triggers it, and write a test asserting the *correct* behavior.
2. **Run it against the unfixed code and watch it fail.** This is the crucial
   step, not ceremony. A red test here proves two things at once: you have
   *actually reproduced* the reported bug, and your test is *capable of
   detecting* it.
3. **Fix the code.**
4. **Watch the test pass** — and the rest of the suite stay green (the fix
   broke nothing else).
5. **Ship fix and test together**, one narrow commit.

Now run the logic in reverse to see why step 2 is load-bearing. Suppose you
skip it: write the fix first, then add a test that passes. What does that green
test prove? Maybe it exercises the bug's conditions — or maybe it misses them
entirely and would have passed against the broken code too. You cannot tell.
The only way to know a test detects a bug is to have *watched it do so*.

> Rule of thumb: **if you can't write a test that fails, you haven't reproduced
> the bug — keep digging instead of guessing at a fix.** A fix you can't
> demonstrate is a hypothesis wearing a fix's clothes.

That rule, verbatim, is GopherTrunk policy — grown out of real debugging
history where "obvious" fixes shipped for symptoms that later turned out to
still be live. The failing test is the receipt.

## What a regression test looks like

Often it's one new [table row](/learn/testing/table-driven-tests/). Suppose a
frequency parser crashed on a negative offset written like `"-812.5kHz"`:

```go
// One-line regression: the exact input from the bug report.
{"negative kHz offset (crashed before fix)", "-812.5kHz", -812_500, false},
```

For a subtler bug, a dedicated test with a comment telling the story:

```go
// TestParseOffset_NegativeKHz is a regression test: negative offsets with a
// kHz suffix used to lose their sign because the parser stripped '-' with
// the unit. Must stay: the sign matters when retuning below center.
func TestParseOffset_NegativeKHz(t *testing.T) {
    got, err := ParseOffset("-812.5kHz")
    if err != nil {
        t.Fatalf("ParseOffset(%q) error: %v", "-812.5kHz", err)
    }
    if got != -812_500 {
        t.Errorf("ParseOffset(%q) = %d, want %d", "-812.5kHz", got, -812_500)
    }
}
```

The comment is part of the test. September's refactorer, staring at a red
`TestParseOffset_NegativeKHz`, learns in ten seconds what March's debugger
spent a day discovering.

## The compounding effect

A suite that accretes one test per fixed bug becomes something more than a
checklist: it's the project's **scar tissue** — a machine-checked history of
every way the code has actually failed in the wild. Real-world bugs cluster in
the gnarly regions (parsing, boundaries, concurrency), so regression tests
concentrate coverage exactly where experience proved it's needed — no
[coverage metric](/learn/testing/code-coverage/) allocates tests that well.
Unit 6 closes the module by [walking this loop](/learn/testing/your-first-regression-test/)
on a realistic bug end to end; there you'll also meet the harder question this
lesson defers — what counts as *verified* when the failing-first test passes
but the original symptom might still be alive
([capture-gated verification](/learn/testing/capture-gated-verification/)).

<div class="knowledge-check" data-quiz data-correct-msg="Right — only a test you watched fail against the broken code is proven able to detect the bug; written after the fix, it might pass for the wrong reasons." markdown="0">
  <p class="knowledge-check__q">Quick check: why must the regression test be written and run <em>before</em> the fix?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Because tests written after a fix don't count toward coverage</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Because Go caches test results and would skip a new test otherwise</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Because watching it fail proves it reproduces the bug and can detect it — a test that never failed proves neither</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **regression** is a bug in behavior that used to work; fixed bugs *do* come
  back when nothing guards the fix.
- A **regression test** pins the bug's exact triggering input into the suite —
  often one table row — so resurrection turns the suite red.
- **Failing first**: red against the broken code proves you reproduced the bug
  *and* that the test can detect it; then the fix earns its green.
- **Can't make it fail? You haven't reproduced the bug** — keep digging, don't
  guess. Fix and test ship together, one narrow commit.
- Accumulated regression tests are the project's **scar tissue** — coverage
  concentrated exactly where reality proved it matters.

Next up: [Golden files & fixtures](/learn/testing/golden-files-and-fixtures/)
