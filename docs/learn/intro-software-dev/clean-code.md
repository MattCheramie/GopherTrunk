---
slug: clean-code
title: Writing readable code
description: Code is read far more than it is written. Learn naming, small functions, why-not-what comments, consistent style, and the real cost of cleverness.
keywords: clean code, readable code, code readability, good naming, intention-revealing names, small functions, code comments, code style, maintainable code
level: beginner
status: full
faq:
  - q: "Why does readability matter more than writing code quickly?"
    a: "Because code is read far more often than it is written. A line you type once may be read dozens of times during reviews, debugging, and future changes. Optimizing for the reader saves vastly more time over a project's life than the few seconds saved by terse code."
  - q: "Should I add comments to all my code?"
    a: "No. Aim for code that explains itself through good names and structure, and reserve comments for the *why* — intent, trade-offs, and non-obvious constraints. Comments that merely restate what the code already says tend to drift out of date and add noise."
  - q: "Is clever, compact code a sign of skill?"
    a: "Rarely. The harder skill is making complex logic look simple and obvious. Cleverness that the next reader (often you, months later) has to decode is a liability. Prefer the boring, clear version unless a measured performance need forces otherwise."
---
# Writing readable code

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Read over write** — code is read far more than it's written, so optimize for the reader. **Names carry meaning** — intention-revealing names beat clever abbreviations. **Comments explain why** — the code already shows the *what*.
</div>

Most beginners assume the hard part of programming is getting a program to run. It isn't. The hard part is keeping it understandable once it works — by your teammates, by future maintainers, and by *you* six months from now when you've forgotten every assumption you made. A program is written once but read many times: during review, during debugging, every time someone extends it. That asymmetry is the single most important fact about writing code, and clean code is simply the discipline of optimizing for the reader instead of the writer.

This lesson covers the everyday habits that make code readable: naming things well, keeping functions small and focused, commenting intent rather than mechanics, formatting consistently, and resisting the temptation to be clever. None of these require talent — just attention.

## Code is read far more than it is written

When you write a line of code, you do it once. That line then gets read again and again: by a reviewer approving your change, by a colleague hunting a bug nearby, by a future contributor adding a feature, and by you, long after you've lost the context. Studies and experience both point the same way — developers spend far more time reading code than writing it.

The practical conclusion: **a few seconds saved while typing is never worth minutes lost while reading.** Terse, dense, "efficient-to-type" code is a false economy. Clear code that takes slightly longer to write pays for itself many times over. Every clean-code habit below flows from this one principle.

## Good names are intention-revealing

Naming is the highest-leverage readability skill. A good name tells the reader *what something is* and *why it exists* without forcing them to read the surrounding code. A bad name forces a detour into the implementation just to understand a single line.

The rules are simple:

- **Reveal intent.** A name should answer why this exists and how it's used.
- **Avoid cryptic abbreviations.** `sr` could be sample rate, success rate, or string. `sampleRateHz` cannot be misread.
- **Encode units and meaning** when it prevents bugs: `timeoutMs`, `frequencyHz`, `bufferSizeBytes`.
- **Be consistent.** Pick one word per concept — don't mix `fetch`, `get`, and `retrieve` for the same idea.
- **Avoid noise words** like `data`, `info`, `manager`, `temp` that add length without meaning.

Here is the difference in a signal-processing context:

```go
// Before: what is sr? what is n? what does this loop do?
func proc(b []float64, sr int, n int) []float64 {
    out := make([]float64, 0)
    for i := 0; i < len(b); i += n {
        out = append(out, b[i])
    }
    return out
}

// After: the signature alone tells the story.
func downsample(samples []float64, sampleRateHz int, decimationFactor int) []float64 {
    decimated := make([]float64, 0)
    for i := 0; i < len(samples); i += decimationFactor {
        decimated = append(decimated, samples[i])
    }
    return decimated
}
```

The logic is identical; the second version is readable at a glance. `sampleRateHz` beats `sr` every time — and notice that once the names are good, the function barely needs a comment at all.

## Small, focused functions

A function should do one thing and do it at one level of abstraction. When a function tries to validate input, transform data, and write a result all at once, the reader has to hold all three concerns in their head simultaneously. Splitting it lets each part be named, understood, and tested on its own.

Signs a function is doing too much:

- You'd struggle to name it without using "and."
- It has many parameters or deeply nested conditionals.
- You have to scroll to see all of it.
- Sections are separated by blank lines and a comment — those are usually functions waiting to be extracted.

Small functions also turn comments into names. Instead of `// now normalize the gain`, you call `normalizeGain(samples)`. The name documents the step, and the documentation can never go stale because it *is* the code.

There's no magic line count, but a useful instinct: if a function no longer fits comfortably on your screen, it's probably hiding a smaller function inside it.

## Comments explain *why*, not *what*

Good code already says *what* it does — that's what the names and structure are for. Comments earn their place when they explain things the code *can't*: the reason behind a choice, a non-obvious constraint, a workaround for an external bug, or a trade-off you deliberately made.

```go
// Bad — restates the obvious, will rot when the code changes.
i++ // increment i

// Good — explains a why the code cannot show.
// The receiver hardware emits a spurious DC spike at bin 0; skip it
// so the FFT peak detector doesn't lock onto a non-signal.
spectrum[0] = 0
```

The danger of *what*-comments is that they drift. Code changes; the comment beside it often doesn't, and now it actively lies to the reader. A wrong comment is worse than no comment. So the rule is: if a comment just narrates the next line, delete it and improve the name instead. Save comments for intent, gotchas, and links to the reasoning.

## Consistent formatting and style

Consistency lets readers stop noticing the layout and start seeing the logic. When indentation, brace placement, naming case, and import ordering are uniform across a codebase, differences become meaningful — a difference signals a *real* difference, not just a stylistic whim.

The good news is you rarely have to think about this manually. Modern languages ship formatters and linters that enforce a single style automatically:

- **Go** has `gofmt` — one canonical format, no debate.
- Many ecosystems use tools like Prettier, Black, or `rustfmt`.
- Linters (`golangci-lint`, ESLint, and friends) catch suspicious patterns beyond formatting.

Adopt the tools, run them on save or in CI, and spend your attention on substance instead of spacing. A consistent style is also a courtesy: it makes diffs small and reviews focused on behavior rather than cosmetics.

## The cost of cleverness

There is a seductive pull toward clever code — a one-liner that chains five operations, a bitwise trick, an obscure language feature that saves three lines. It feels like craftsmanship. Usually it's a trap.

> "Everyone knows that debugging is twice as hard as writing a program in the first place. So if you're as clever as you can be when you write it, how will you ever debug it?" — Brian Kernighan

Clever code shifts cost from the writer (who understood it in the moment) to every future reader (who has to reverse-engineer it). The genuinely hard skill is making complicated logic *look* simple — choosing the boring, explicit version that anyone can follow. Reserve cleverness for the rare spot where profiling proves you need it, and when you do, wrap it in a clear name and a *why*-comment explaining the trade-off.

<div class="knowledge-check" data-quiz data-correct-msg="Right — code is read far more often than written, so the reader's time is what you're optimizing." markdown="0">
  <p class="knowledge-check__q">Quick check: what is the core reason clean code matters?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It makes programs run faster at runtime</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Code is read far more often than it is written</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Compilers reject code that isn't well formatted</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Read over write** — code is read far more than written, so optimize every choice for the future reader.
- **Names reveal intent** — `sampleRateHz` beats `sr`; avoid cryptic abbreviations and noise words.
- **Functions stay small** — one job, one level of abstraction; extract steps into well-named helpers.
- **Comments explain why** — let names show the *what*; reserve comments for intent and gotchas.
- **Style is consistent** — lean on formatters and linters so attention goes to logic, not layout.
- **Cleverness has a cost** — the boring, clear version usually wins; clever code is hard to debug.

Next up: with readable code as the foundation, we turn to the design heuristics that keep it lean — [DRY, KISS, YAGNI and separation of concerns](/learn/intro-software-dev/dry-kiss-yagni/).
