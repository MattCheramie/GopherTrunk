---
slug: formatters-and-style
title: Formatters & style
description: gofmt ends formatting debates by making one style automatic — why mechanical formatting is a quality tool, not a cosmetic one, and how it keeps reviews and diffs about substance.
keywords: gofmt, go formatting, goimports, code style go, automatic formatting, format on save, code review noise, diff noise
level: beginner
status: full
---

# Formatters & style

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**gofmt** rewrites Go source into the **one canonical style** — automatically,
identically, for everyone. That's a quality tool, not cosmetics: uniform
formatting makes **diffs show only real changes**, frees **code review** for
substance, removes a whole category of team argument, and can even expose bugs
that misleading indentation hides. **goimports** adds import management on
top. The habit is **format on save**; the payoff compounds across every review
and every diff for the life of the project.
</div>

The shortest lesson in the module, about the tool that ended one of
programming's oldest arguments. The interesting part isn't how to run it —
it's why a *formatter* belongs in a module about *quality*.

## One style, by fiat

Before Go, every team negotiated formatting: tabs or spaces, brace placement,
line lengths — recurring arguments consuming real engineering hours, settling
nothing, because the answers are genuinely arbitrary. Go's designers ended the
argument by removing it: **gofmt** ships with the toolchain, rewrites code
into one canonical style, and the entire ecosystem adopted it. Ask "what's the
standard style?" and the answer is "whatever gofmt outputs."

```bash
gofmt -l .      # list files that differ from canonical form
gofmt -w .      # rewrite them in place
goimports -w .  # same, plus add/remove/sort import statements
```

In practice nobody runs these by hand: every Go-aware editor formats on save.
The style being *nobody's preference* is precisely the feature — there's
nothing to argue about, ever again.

## Why this is a quality tool

**Diffs show only real change.** Version control diffs are how changes get
reviewed, and how history gets [bisected](/learn/testing/bisecting-history/)
when hunting a regression. In an unformatted codebase, editors quietly
re-indent and diffs fill with noise — a two-line logic change buried in eighty
whitespace lines. Under gofmt, formatting never changes after the fact, so
**every changed line in a diff is a changed behavior or a changed thought**.
That property pays out on every review and every archaeology dig for the
project's lifetime.

**Review attention is finite.** [Code review](/learn/testing/code-review/) —
next lesson — works because a human reads a change closely. Every comment
spent on "indentation is off here" is attention not spent on "this length
check is wrong." Mechanical style enforcement means humans never spend review
budget on what a machine settles for free.

**Misleading layout hides real bugs.** The classic:

```go
if err != nil
    log.Printf("retune failed: %v", err)
    return err   // looks conditional; runs ALWAYS
```

In brace-optional languages, indentation-vs-reality mismatches have caused
famous security bugs (Apple's `goto fail` among them). Go's grammar plus gofmt
close the gap from both ends: braces are mandatory, and formatting always
reflects actual structure — code can't *look* different from what it does.

> Rule of thumb: if a style question can be settled by a machine, let the
> machine settle it, and spend the humans on questions that need judgment.

## Enforce it like any other check

Formatting only delivers these benefits if it's *universal*, so mature
projects verify it mechanically rather than trusting habit: a
[CI](/learn/testing/continuous-integration/) step fails when `gofmt -l` names
any file. That sounds strict and costs nothing — everyone's editor already
complies — while guaranteeing the noise-free-diff property holds for the whole
history. It's the same philosophy as the rest of this unit: hygiene that's
enforced mechanically stays enforced; hygiene that relies on memory decays.

The same idea now exists nearly everywhere — Prettier for JavaScript, Black
for Python, rustfmt for Rust — all following gofmt's insight that a
*non-configurable* formatter is the valuable kind. Configuration would just
relocate the argument.

<div class="knowledge-check" data-quiz data-correct-msg="Right — when formatting is canonical and constant, diffs contain only genuine changes, which sharpens review and history archaeology alike." markdown="0">
  <p class="knowledge-check__q">Quick check: how does universal gofmt improve code <em>review</em>?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It makes the code run faster, so reviewers wait less</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Diffs contain only real changes and reviewers never spend attention on style nits</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It automatically fixes logic errors it finds while reformatting</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **gofmt** makes one canonical style automatic — the argument isn't won, it's
  **removed**.
- Quality payoffs: **noise-free diffs**, review attention spent on
  **substance**, and layout that can't misrepresent structure.
- **goimports** adds import add/remove/sort on top of gofmt.
- Make it invisible (**format on save**) and universal (**CI check**) —
  mechanical hygiene stays enforced; remembered hygiene decays.
- The principle generalizes: machine-settleable questions go to machines;
  human judgment goes where it's needed — like the next lesson.

Next up: [Code review](/learn/testing/code-review/)
