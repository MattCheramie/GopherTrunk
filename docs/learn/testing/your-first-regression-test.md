---
slug: your-first-regression-test
title: Write your first regression test
description: Walk the full failing-first loop on a real bug — reproduce it, write the test that fails, fix the code, watch it pass, and ship both together in one narrow commit.
keywords: write a regression test, failing first walkthrough, red green refactor bug fix, go regression test tutorial, bug fix with test, practice regression testing
level: advanced
status: full
prereq:
  - regression-tests
  - reproducing-a-bug
---

# Write your first regression test

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The capstone: run the **failing-first loop** end to end on a concrete bug —
**reproduce** the report as a test, watch it **fail red** against the broken
code, make the **narrow fix**, watch it **pass green** with the rest of the
suite, then ship **fix and test in one commit** whose message tells the story.
Along the way, every discipline from the module gets used in anger:
shrinking, boundary thinking, root-cause honesty, and knowing what you may —
and may not — claim as **verified** afterward.
</div>

Thirty-three lessons of ideas; one lesson of doing. Work through this with
your hands — running the loop once completely is how it becomes reflex.

## The bug report

Your incoming issue, in typical field condition — real, vague, radio-flavored:

> **Scan list skips the last talkgroup.** I have 8 talkgroups configured.
> The scanner cycles through the first 7 fine, but TG 4521 — the last one in
> my file — never gets checked. Worked fine when I had 5.

The (buggy) code it points into:

```go
// NextTalkgroup returns the next talkgroup to check, cycling the scan list.
func (s *ScanList) NextTalkgroup() int {
    s.pos++
    if s.pos >= len(s.groups)-1 {   // the bug lives here
        s.pos = 0
    }
    return s.groups[s.pos]
}
```

Resist reading for the bug yet — that's builder brain volunteering. The loop
starts elsewhere.

## Step 1 — reproduce as a test (and shrink)

Turn the report's *claim* into a checkable prediction, applying the
[shrinking discipline](/learn/testing/reproducing-a-bug/): the report says 8
talkgroups, but if the last element never appears, 3 — the smallest list
where "middle" and "last" differ — should show it too. The prediction:
cycling from the start must visit **every** talkgroup:

```go
func TestNextTalkgroup_VisitsEveryGroup(t *testing.T) {
    s := NewScanList([]int{100, 200, 4521}) // arrange: shrunk from the report's 8

    seen := map[int]bool{}
    for i := 0; i < 6; i++ {                // act: two full cycles
        seen[s.NextTalkgroup()] = true
    }

    for _, tg := range []int{100, 200, 4521} { // assert: nobody skipped
        if !seen[tg] {
            t.Errorf("talkgroup %d never visited in two full cycles; visited: %v", tg, seen)
        }
    }
}
```

Note the test asserts the *reported symptom* — "a talkgroup is never
visited" — not any theory about the code, so it stays honest even if your
diagnosis turns out wrong: the
[trap-lesson](/learn/testing/the-self-consistent-synthetic-trap/) hygiene of
taking expected values from the *requirement*, never the implementation.

## Step 2 — watch it fail

```text
--- FAIL: TestNextTalkgroup_VisitsEveryGroup (0.00s)
    scanlist_test.go:14: talkgroup 4521 never visited in two full cycles; visited: map[100:true 200:true]
```

Savor this red: it's the [receipt](/learn/testing/regression-tests/) — proof
you've reproduced the reported bug (last element skipped, matching the
report) and that this test can detect it. Green here would be a finding too:
your reproduction is wrong or the report is subtler than it reads, and per
the standing rule you'd **keep digging, not guess at a fix.**

## Step 3 — diagnose and fix, narrowly

*Now* read the code, with the failure in hand. The wrap check `s.pos >=
len(s.groups)-1` wraps when `pos` reaches `len-1` — the last valid index —
so the last element is never reachable. A classic off-by-one at a
[boundary](/learn/testing/the-testing-mindset/); the report's "worked with 5"
was a red herring — it never worked, nobody noticed. (A quick
[five-whys](/learn/testing/root-cause-analysis/): why did no test catch it?
The old tests checked *cycling happens*, not *every element is visited* — an
assertion gap your new test closes.)

```go
    if s.pos >= len(s.groups) {   // wrap after the last index, not at it
        s.pos = 0
    }
```

The fix is one comparison. **Keep it that size.** "Cleaning up while you're
in there" — renames, restructuring — breaks the narrow-commit discipline:
reviewers and future [bisecters](/learn/testing/bisecting-history/) need this
commit to mean one thing. (GopherTrunk's contribution rules say it outright:
refactors travel separately from behavior changes.)

## Step 4 — green, and the full gate

```bash
go test -run TestNextTalkgroup -v   # the new test: now passes
make vet test                        # the whole gate: nothing else broke
```

Both matter: the first confirms the fix, and the
[always-green gate](/learn/testing/make-vet-test/) confirms it didn't buy
this behavior at the cost of another — the complete evidence pair.

## Step 5 — ship both together

One commit, fix plus test, with a message that tells the next archaeologist
the story ([commit craft](/learn/git/staging-and-commits/) from the Git
module):

```text
scanlist: fix off-by-one that skipped the last talkgroup

NextTalkgroup wrapped at len-1, so the final entry was never
visited. Regression test cycles a 3-entry list twice and asserts
every talkgroup is seen; it fails against the previous code.
```

Then close the loop honestly, the
[capture-gated way](/learn/testing/capture-gated-verification/): for a pure
logic bug, the failing-first test plus the reporter confirming TG 4521 now
gets checked *is* verification. Had the symptom lived in signal handling, the
bar would be higher — the reporter's capture replayed clean — and until then
the issue stays open and the PR says `Refs`, not `Closes`.

<div class="knowledge-check" data-quiz data-correct-msg="Right — green-on-first-run means the test doesn't reproduce the reported bug; the rule is to keep digging until you have a red that matches the symptom." markdown="0">
  <p class="knowledge-check__q">Quick check: you write the reproduction test and it passes against the supposedly buggy code. What now?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Proceed with the fix — the test will guard it anyway</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Stop: you haven't reproduced the bug — refine the test or your understanding until it fails for the reported reason</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Delete the test and close the issue as not reproducible</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The loop: **reproduce as a test → red → narrow fix → green → whole suite
  green → one commit, fix + test.**
- Assert the **symptom**, with expectations from the requirement — not from
  the implementation or your theory of it.
- The first **red is the receipt**; a green reproduction means keep digging,
  never guess.
- Keep fixes **narrow** and claims **honest**: verification standards scale
  with how far the symptom sits from your test.

That's the module: you came in able to hope code works; you leave able to
*show* it. From here, three natural directions: deepen the Go-side craft in
[Testing in Go](/learn/programming-go/testing-in-go/) and the
[Programming in Go module](/learn/programming-go/); wire your new gates into
[pull requests, checks, and automation](/learn/git/) in the Git module; or
follow quality into production — pipelines, monitoring, and release safety —
in the [Deployment module](/learn/deployment/). And keep the
[glossary](/learn/testing/glossary/) handy whenever a term goes fuzzy.
