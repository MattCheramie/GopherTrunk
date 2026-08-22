---
slug: root-cause-analysis
title: Root-cause analysis
description: Keep asking why until the answer would have prevented the bug — five whys, symptom fixes vs cause fixes, and turning each incident into defenses that outlast it.
keywords: root cause analysis, five whys, symptom vs cause, postmortem analysis, blameless postmortem, fix the class not the instance, defense in depth debugging
level: intermediate
status: full
prereq:
  - what-is-a-bug
  - reproducing-a-bug
---

# Root-cause analysis

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Finding *a* cause isn't the end of debugging — most bugs offer a **chain** of
causes, and where you stop determines what you fix. A **symptom fix** patches
where the failure surfaced and leaves the cause free to strike again;
**root-cause analysis** keeps asking **"why?"** down the chain (the **five
whys**) until an answer would have **prevented** the bug — then fixes there,
plus a **net** (a test, a check, a validation) at the layers above. The
strongest closing question: not "how do we fix this bug?" but "**what would
have caught this class of bug earlier — and its siblings still in the code?**"
</div>

You can now reproduce failures, read their evidence, instrument them, pause
them, and date them to a commit. This closing lesson of the unit is about the
judgment those tools serve: deciding *what actually gets fixed* — because the
first plausible fix is usually a bandage on the surfacing point, not a repair
of the cause.

## The chain, and where people stop

Run a realistic failure through repeated "why":

1. **The scanner crashed** mid-session. *Why?*
2. A panic: index out of range in the frame slicer. *Why?*
3. `DecodeBurst` sliced `raw[:frameLen]` with `frameLen` beyond the buffer.
   *Why?*
4. `frameLen` came from a length field in the received message, used
   unchecked. *Why unchecked?*
5. The decoder trusted the message to obey the spec — but this input came off
   the *air*, where noise corrupts bits; a corrupted length field is a
   *when*, not an *if*. *Why did no test catch it?*
6. Every test fed the decoder well-formed messages; nothing ever
   [fuzzed](/learn/testing/fuzzing/) it with corrupt input.

Each level supports a "fix." Stop at level 2 and you wrap the panic in a
recover — the corruption now propagates *silently* (arguably worse). Stop at
level 3 and you clamp that one slice — until the *same unchecked trust* in a
different field crashes a different line next month. Level 4 is the real
repair: **validate the length field against the buffer at the trust
boundary** — reject the frame, count it, move on. Levels 5–6 are the
*defenses*: a regression test with the corrupt frame, and a fuzz target so
the cousins of this bug die in CI instead of the field.

> Rule of thumb: you've reached root cause when the answer to "why?" would
> have **prevented** the bug — and fixing there kills the **class**, not just
> today's instance.

## Symptom fix vs cause fix

How to tell which one you're holding:

| | Symptom fix | Cause fix |
|--|------------|-----------|
| Where it acts | Where the failure **surfaced** | Where the bad state was **born** (or should have been rejected) |
| Its shape | Special-case guard at the crash site; "if weird, skip" | Validation at the boundary; corrected assumption; redesigned contract |
| Its future | The cause strikes again elsewhere | The class of bug is closed |
| Its tell | You can't explain *why* the bad value occurred, only that it did | The explanation reaches an assumption that was wrong, and you changed it |

That last row is the honest self-check, and it circles back to
[lesson 2's](/learn/testing/what-is-a-bug/) defect-vs-failure distinction: if
you can't narrate the full chain from defect to failure, you haven't found the
defect — you've found a place to hide the failure. Note the trap of
**plausibility**: the first explanation that *could* account for the symptom
gets adopted because searching is expensive and the explanation is
comfortable. The antidote is the module's standing discipline — a fix you
can't demonstrate with a [failing-first test](/learn/testing/regression-tests/)
against the *reproduced* symptom is a hypothesis, however plausible it feels.
GopherTrunk's debugging history is full of first-theory autopsies — "it must
be the load," "it must be encryption" — that a forced reproduction later
overturned; the discipline exists because plausible-and-wrong is the default
outcome of stopping early.

## Beyond the code: the five whys' real destination

Keep asking why past the code and you reach the conditions that *let* the
defect ship — and that's where the highest-leverage fixes live, because they
apply to every future bug:

- *Why did no test feed corrupt input?* → No fuzz targets on parse boundaries
  → **add fuzzing to CI** for every decoder entry point.
- *Why did review miss unchecked trust in radio data?* → No shared habit of
  flagging trust boundaries → **make "where does this value come from?" a
  standing review question**.
- *Why did the crash take a week to notice?* → Panics only in a terminal
  scrollback → **log and count decode rejections**, alert on crash loops.

Two disciplines keep this productive rather than theatrical. **Blameless**: at
every level, ask what *let* the mistake through, never who made it — the
five whys should surface missing nets, not names, or people stop surfacing
facts (the framing [lesson 1](/learn/testing/why-software-breaks/) opened
with). And **sibling hunting**: the moment you can name the class — "length
fields trusted from the air" — grep for its brothers *now*, while you
understand the pattern better than anyone ever will again. The best debuggers
fix three latent bugs per reported one this way.

<div class="knowledge-check" data-quiz data-correct-msg="Right — recovering at the crash site hides the symptom while the corrupt data flows on; the cause fix validates at the boundary where the untrusted value entered." markdown="0">
  <p class="knowledge-check__q">Quick check: a corrupted radio length field crashes a slice operation. Which change is the symptom fix?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Wrapping the slicing code in recover() so the panic no longer crashes the program</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Validating the length field against the buffer size where the frame enters the decoder</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Adding a fuzz target that feeds the decoder corrupted frames in CI</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Bugs have **cause chains**; where you stop asking "why?" decides whether you
  fix the **symptom** or the **cause**.
- **Five whys**: stop when the answer would have **prevented** the bug —
  usually a wrong assumption at a trust boundary.
- Symptom fixes hide failures and leave the class alive; cause fixes **close
  the class** — then add nets (tests, fuzzing, checks) at the layers above.
- Beware **plausible** explanations adopted without a reproduced,
  failing-first demonstration — plausible-and-wrong is the default.
- Go past the code: fix what let the bug **ship**, keep it **blameless**, and
  **hunt the siblings** while the pattern is fresh.

Next up: [The make vet test gate](/learn/testing/make-vet-test/)
