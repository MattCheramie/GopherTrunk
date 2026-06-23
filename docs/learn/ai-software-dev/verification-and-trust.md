---
slug: verification-and-trust
title: Verifying AI-written code
description: AI writes plausible code, not guaranteed-correct code. Learn the verification disciplines — read it, test it, run it, lint and type-check it, watch for hallucinated APIs and subtle logic bugs — and why heavier delegation raises the stakes on verification.
keywords: verifying AI code, trust but verify, AI hallucination, hallucinated API, code review, testing AI code, type checker, linter, subtle logic bugs, over-delegation, plausible code, AI code safety
level: intermediate
prereq:
  - the-spectrum
faq:
  - q: "Can I trust code an AI writes for me?"
    a: "Trust it the way you'd trust a confident but fallible colleague: read it, test it, and run it before you rely on it. AI produces **plausible** code — code that looks right — which is not the same as code that *is* right. The discipline is 'trust but verify': the model drafts, you confirm."
  - q: "What is a hallucinated API?"
    a: "It's when a model invents a function, method, library, or option that doesn't actually exist but looks completely real because it fits the patterns the model has seen. The code reads naturally and may even compile-by-resembling-something-real, then fails because the thing it calls was never real. Checking calls against the actual documentation catches these."
  - q: "If I read every line, doesn't that cancel out the time the AI saves?"
    a: "No — reading and understanding is far faster than writing from scratch, and it's where the real value lands. The AI does the tedious drafting; you do the cheaper, higher-leverage work of confirming it's correct. Skipping the review doesn't save time, it just defers the cost to a harder-to-find bug later."
---
# Verifying AI-written code

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Plausible is not correct** — AI writes code that *looks* right, which is a different claim from code that *is* right. **Verify deliberately** — read it, test it, run it, and lean on type-checkers and linters before you trust it. **Never outsource your judgement** — the more you delegate, the more the verification matters.
</div>

This is lesson 25 of the path, and it's the spine of everything this course has tried to teach about working safely. You've chosen a [model](/learn/ai-software-dev/choosing-model-and-provider/) and a [method](/learn/ai-software-dev/choosing-your-method/); now comes the discipline that keeps both from hurting you. The single most important idea in AI-assisted development is this: **a model produces plausible code, not guaranteed-correct code.** By the end of this lesson you'll understand why that's true, the concrete habits that turn plausible into trusted, and why heavier delegation makes those habits more important, not less.

## Why AI writes plausible code, not correct code

Recall how a model works: it predicts the next token, over and over, from patterns learned across enormous amounts of text and code. It is extraordinarily good at producing output that *resembles* correct code — code with the right shape, idioms, and structure. But resemblance is not correctness. The model has no compiler, no test run, and no ground truth inside it; it is generating what *looks like* the right continuation, not verifying that the continuation works.

That's why AI code can be confidently, fluently wrong. It won't hedge; it will hand you a clean-looking function with a real bug in it and no warning sign. The fluency is exactly what makes it dangerous — wrong code that *looked* wrong would be easy to catch. The job of verification is to supply the checking the model can't.

This isn't a knock on the tools. A plausible draft is genuinely useful; it's a head start, not a finished product. The mistake is treating the draft as the destination.

## Trust but verify: the core disciplines

"Trust but verify" is the whole stance in three words. Trust the model enough to let it draft; verify enough that nothing reaches your project unconfirmed. Here are the disciplines, roughly in order of how routinely you should apply them.

**Read and understand it before you merge.** This is non-negotiable. If you can't explain what a block of AI-written code does and why, you're not ready to keep it. Reading is fast and high-leverage — far cheaper than writing from scratch, and the place bugs are most often caught. Code you don't understand is a liability whether a human or a model wrote it ([clean code](/learn/intro-software-dev/clean-code/) is partly about making code readable for exactly this reason).

**Lean on tests.** Tests are how you turn "looks right" into "behaves right." Have the model write tests alongside the code, then *read the tests too* — a model can write a test that passes for the wrong reason. Better still, write or sketch the test yourself so it encodes what you actually want, then let the AI make it pass. The whole logic of [testing](/learn/intro-software-dev/testing/) — checking behaviour against expectations — is your best defence against plausible-but-wrong code.

**Actually run it.** Code that compiles is not code that works. Run it on real inputs, including the awkward edge cases, and watch what it actually does. Plenty of plausible code falls over the first time it meets reality.

**Use type-checkers and linters.** These are cheap, automatic verifiers that catch a whole class of errors for free — type mismatches, unused variables, suspicious patterns, undefined names. They won't catch a logic bug, but they'll catch many of the mechanical mistakes that slip into generated code, and they cost you nothing once set up. (See [robustness and errors](/learn/intro-software-dev/robustness-and-errors/) for why catching failures early pays off.)

**Watch for hallucinated APIs and subtle logic bugs.** Two failure modes deserve their own attention, below.

## The two failure modes to hunt for

### Hallucinated APIs

A **hallucinated API** is a function, method, library, or option the model invents — it doesn't exist, but it looks completely real because it matches the patterns the model has seen elsewhere. The model "knows" that libraries like this usually have a method named something like this, so it confidently writes the call. The give-away is often that the name is *too* convenient: exactly the function you wished existed.

Hallucinated APIs are usually easy to catch once you look, because the code fails or won't resolve. The trap is not looking — accepting the call because it reads naturally. The fix: check unfamiliar calls against the *actual* documentation, not the model's say-so.

### Subtle logic bugs

Far more dangerous, because they hide. The code runs, the types check, nothing errors — and the logic is quietly wrong. An off-by-one in a loop, a flipped comparison, a swapped sign, a misread requirement. These survive everything except a test that pins the expected behaviour and a careful read by someone who understands the intent. This is precisely why "read it and test it" sits at the top of the list: the worst bugs are the ones that don't announce themselves.

| Failure mode | How it shows up | Best defence |
|---|---|---|
| **Hallucinated API** | A call to something that doesn't exist; fails to resolve or run | Check against real documentation |
| **Subtle logic bug** | Runs cleanly, types check, behaviour is silently wrong | Tests that pin expected behaviour + understanding the intent |

## A GopherTrunk example: nothing trusted until the tests pass

Make this concrete in [GopherTrunk](/learn/intro-software-dev/what-is-software/). Suppose you ask an AI to write or modify a piece of DSP code — say, a down-converter or a part of a control-channel decoder. The output will look entirely plausible: correct-shaped Go, sensible filter math, the right function names. None of that earns trust on its own. Signal-processing code is exactly where a subtle logic bug hides best — a sign error or an off-by-one in a filter doesn't crash, it just quietly degrades the decode, and you might not notice until a capture fails to lock.

So in this project the rule is concrete: AI-written DSP code is not trusted until it passes the existing regression suite — `make vet test` must be green before any commit. And the project's discipline goes further: a bug fix should come with a **failing-first regression test** — a test that fails *without* the change and passes *with* it. That rule exists because it's the only proof that the fix addresses the real problem rather than a plausible-looking guess. An AI's confidence that its fix is correct counts for nothing; a test that failed first and now passes counts for everything. The model can write that test, but you decide it genuinely captures the bug.

## Why over-delegation raises the stakes

Recall the [delegation spectrum](/learn/ai-software-dev/the-spectrum/): from completing a single line you're typing, all the way to handing an [agent](/learn/ai-software-dev/agentic-and-vibe-coding/) a whole feature it builds across many files. As you slide toward the autonomous end, two things grow at once: how much the AI produces between your checkpoints, and how much you didn't write yourself. Both make verification *more* important, not less.

When you accept one completed line, the surface to verify is tiny and you saw it appear. When an agent returns a 600-line diff across eight files, the surface is large, you didn't watch it form, and the plausible-but-wrong code has more places to hide. The temptation at high delegation is to trust the volume because reviewing it is tedious — which is exactly backwards. The more you delegate, the more disciplined the verification must be: review the whole diff, run the full test suite, and don't let "it's a lot of code and it looks fine" stand in for "I checked it."

The principle underneath all of it: **never outsource your judgement.** Delegate the typing, the drafting, the tedium — but the decision that code is correct and safe to keep stays with you. That decision is the one thing the model can't make, because it can only tell you what's plausible.

<div class="knowledge-check" data-quiz data-correct-msg="Right — models generate plausible-looking code, so you must read, test, and run it before trusting it." markdown="0">
  <p class="knowledge-check__q">Quick check: why must you verify AI-written code rather than trust it as-is?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Because AI code never compiles without manual fixes</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Because a model produces plausible-looking code, which is not the same as guaranteed-correct code</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Because AI tools are legally required to be reviewed by a human</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Plausible ≠ correct** — a model predicts what looks like the right code; it has no built-in check that the code actually works.
- **Read it first** — if you can't explain what AI-written code does, you're not ready to merge it.
- **Test and run it** — pin expected behaviour with tests you understand, and actually execute the code on real inputs and edge cases.
- **Automate the cheap checks** — type-checkers and linters catch mechanical errors for free.
- **Hunt the two failure modes** — hallucinated APIs (caught by docs) and subtle logic bugs (caught by tests and understanding).
- **More delegation, more verification** — a large autonomous diff hides more, so review it harder; never outsource the judgement that code is safe to keep.

Next up: where your prompts and code actually go, who might see them, and the licensing and ethics questions that come with all of this. See [Security, privacy & ethics](/learn/ai-software-dev/security-privacy-ethics/).
