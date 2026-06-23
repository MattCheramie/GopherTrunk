---
slug: getting-started-today
title: Getting started today
description: Turn the whole path into action — pick one real task this week, choose a gentle starting method, write a small project config file, set a budget, and verify everything. Then a look ahead to a future path on building software with AI embedded inside it.
keywords: getting started with AI coding, AI coding first steps, starter plan, config file, set a budget, verify AI code, build good habits, embedded AI, building AI features, learning path next steps
level: advanced
prereq:
  - choosing-model-and-provider
  - choosing-your-method
  - verification-and-trust
  - security-privacy-ethics
faq:
  - q: "What's the easiest way to actually start using AI for coding?"
    a: "Pick **one real, small task** you have this week and do it with AI instead of reading about it. Use the gentlest method — the chat app or an IDE integration — set a small budget, and verify the result before keeping it. One finished real task teaches more than ten tutorials."
  - q: "Do I really need a config file for my project?"
    a: "Not on day one, but it pays off fast. A short [config file](/learn/ai-software-dev/skills-and-config-files/) that states your project's conventions, build and test commands, and key constraints means the AI starts every session already oriented, instead of you re-explaining each time. A few lines is enough to begin."
  - q: "What comes after this path?"
    a: "This path was about *using* AI to write software. A future path will cover *building* software that has AI embedded inside it — putting LLMs, embeddings, and small or local models into your own applications as features. Until then, the [glossary](/learn/ai-software-dev/glossary/) is a good place to consolidate the terms you've learned."
---
# Getting started today

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Start with one real task** — pick something small you actually need this week and do it with AI. **Set up gently and safely** — a soft starting method, a short project config file, a budget, and verification on everything. **Then look ahead** — this path was about using AI to write software; a future path covers building software with AI embedded inside it.
</div>

This is lesson 27 — the final lesson of the path. You've learned how models work, how to reach them, the techniques for working with them, and how to choose, verify, and use them responsibly. Knowledge that stays unused fades, so this lesson is about *action*: a concrete plan to start this week, a worked end-to-end example, the habits that make you steadily better, and finally a bridge to where this all goes next. By the end you'll have a checklist you can run today, not someday.

## A concrete starting plan

Don't try to adopt everything at once. Run this five-step plan on one real task.

**1. Pick one real task this week.** Not a tutorial, not a toy — something you actually need done. Small and well-scoped is ideal: add a small feature, write tests for an existing function, fix a known bug, or get an unfamiliar piece of code explained. A real task gives you real feedback that a contrived exercise can't.

**2. Choose a gentle starting method.** The [provider's chat app](/learn/ai-software-dev/the-provider-app/) or an [IDE integration](/learn/ai-software-dev/ide-integration/) is the softest on-ramp — low cost, easy to undo, and you see every change before it lands. Save [agents](/learn/ai-software-dev/agentic-cli-tools/) for once you're comfortable reviewing changes you didn't type. Match the method to your task as [Choosing your method](/learn/ai-software-dev/choosing-your-method/) laid out, but when in doubt, start small.

**3. Write a small config file for your project.** A short [config file](/learn/ai-software-dev/skills-and-config-files/) tells the AI, once, what it needs to know every session: the project's conventions, how to build and test, and the constraints it must respect. A few lines beats re-explaining your project in every prompt.

**4. Set a budget or limit.** Decide up front what you're willing to spend, and use your provider's spending caps or a cheaper [model tier](/learn/ai-software-dev/usage-limits-and-tiers/) as a default. A budget turns cost from a worry into a setting (see [Understanding cost](/learn/ai-software-dev/understanding-cost/)).

**5. Verify everything.** This is the rule that makes all of the above safe: read it, test it, run it, before you trust it. Everything in [Verifying AI-written code](/learn/ai-software-dev/verification-and-trust/) applies to your very first task. Plausible is not correct — confirm it.

## A worked end-to-end example

Make it concrete with [GopherTrunk](/learn/intro-software-dev/what-is-software/). Suppose your real task this week is small: add a regression test for an existing function in the decoder — say, a helper in the control-channel decoder that doesn't yet have direct coverage.

- **Set up.** Open your editor with AI integration, and make sure the project has a short config file noting the build and test commands (`make vet test`), the language (Go), and the project's conventions — including its [failing-first regression-test rule](/learn/intro-software-dev/testing/).
- **Plan in chat.** Ask the model to explain what the function does and where its untested edge cases are. Read its explanation against the actual code — this is also how you learn the codebase.
- **Draft the test.** Ask the AI to write a table-driven test covering those edge cases. It produces plausible Go.
- **Verify.** Read the test — does it actually assert the behaviour you want, or does it pass for the wrong reason? Then run it. Confirm it exercises the real function, and sanity-check that it would *fail* if the function were broken (the failing-first idea). Run `make vet test` to be sure nothing else regressed.
- **Keep or refine.** If it's correct and you understand it, keep it. If not, tell the AI what's wrong and iterate. You stay in control of the decision to merge.

That's the entire loop in miniature: plan, draft, verify, decide. Every larger task is this same loop at a bigger scale, which is exactly why heavier delegation demands heavier verification.

## Habits to build for getting steadily better

A few habits compound over time:

| Habit | Why it pays off |
|---|---|
| **Always read what you keep** | Keeps your judgement and skills sharp, and catches bugs early |
| **Let tests carry the trust** | Turns "looks right" into "behaves right," every time |
| **Keep a budget and a default cheap model** | Cost stays predictable; you escalate only when it's worth it |
| **Refine your config file as you go** | Each session starts better-oriented than the last |
| **Match method and model to the task** | Avoids overkill and overspend; the right tool for the job |
| **Iterate, don't accept blindly** | A wrong first draft is normal — steering it is the skill |
| **Keep your setup portable** | Cheap to switch when a better model or tool appears |

The throughline: the AI does the drafting and the tedium, and you keep the judgement. Done consistently, that combination makes you faster *and* better, instead of faster and dependent.

## The bridge: from using AI to building with AI

Step back and notice what this whole path has been about. Every lesson — from how models are [trained](/learn/ai-software-dev/how-models-are-trained/) and [decide](/learn/ai-software-dev/how-models-decide/), through the [techniques](/learn/ai-software-dev/the-spectrum/) and the [choices](/learn/ai-software-dev/choosing-model-and-provider/) — has been about *using AI as a tool to write software*. The AI sat beside you and helped you build ordinary programs.

There's a whole other direction this can go, and it's where a future learning path will take you: **building software that has AI embedded inside it.** Instead of using a model to help you write a feature, you put the model *into* the feature — making the AI a part of the application your users run. Recall the [types of models](/learn/ai-software-dev/types-of-models/) you met early on; that's the raw material for this next step. That future path will cover things like:

- **LLMs inside your app** — calling a model from your own code to summarise, classify, converse, or generate, as a feature your users interact with.
- **Embeddings** — turning text into vectors so your application can search by meaning, recommend, and feed relevant context to a model (the foundation of retrieval-augmented generation).
- **Small and local models** — running compact or open-weight models inside your product for cost, privacy, or offline use, rather than always calling a hosted API.

That's a different craft — software engineering *with AI as a component* rather than AI as your assistant — and it builds directly on the understanding you've assembled here. The mechanics you learned (tokens, context windows, what models can and can't do, where data goes) are exactly what you need to build responsibly with embedded AI.

For now, the most valuable thing you can do is the smallest: pick that one real task and run the five-step plan today. The understanding you've built only becomes skill when you use it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the best way to start is to do one small, real task with AI now, verifying the result, rather than waiting for the perfect setup." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the recommended way to actually start using AI for coding?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Wait until you've assembled the perfect model-and-tool setup before touching real code</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Pick one small, real task this week, use a gentle method, set a budget, and verify the result</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Hand an agent your largest project and let it run unsupervised</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Start with one real task** — something small you actually need this week, done with AI rather than read about.
- **Begin gently** — the chat app or an IDE integration is the softest, safest on-ramp; save agents for later.
- **Set up for success** — a short project config file, a budget or cheap-model default, and verification on everything you keep.
- **Run the loop** — plan, draft, verify, decide; every larger task is the same loop at a bigger scale.
- **Build compounding habits** — read what you keep, lean on tests, keep your setup portable and your judgement engaged.
- **Look ahead** — this path was about *using* AI to write software; a future path covers *building* software with AI embedded inside it.

Next up: this is the final lesson, so there's no next lesson to point to. Consolidate the terms you've learned in the [glossary](/learn/ai-software-dev/glossary/) — and keep an eye out for the coming path on building software with embedded AI, where you'll put LLMs, embeddings, and local models inside your own applications.
