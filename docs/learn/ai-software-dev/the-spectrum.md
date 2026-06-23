---
slug: the-spectrum
title: Choosing where on the spectrum to work
description: Completion, chat, agentic, and vibe coding form one spectrum of delegation — more speed for less understanding and more risk. Learn how to choose per task by stakes, familiarity, and reversibility.
keywords: spectrum of delegation, choosing AI coding mode, AI coding workflow, stakes and risk, reversibility, version control safety net, completion vs chat vs agentic, vibe coding, verification, supervision
level: intermediate
prereq:
  - code-completion
  - chat-assisted-coding
  - agentic-and-vibe-coding
faq:
  - q: "How do I decide which AI coding mode to use?"
    a: "Weigh three things per task: the **stakes** (what happens if it's wrong), your **familiarity** with the code, and the **reversibility** of the change. High stakes, unfamiliar code, or hard-to-undo changes push you toward less delegation and more review. Low stakes, familiar code, and easily reverted changes let you delegate more."
  - q: "Is more delegation always better because it's faster?"
    a: "No. More delegation buys speed but costs understanding and adds risk. The right amount is the *most* delegation the task can safely tolerate — which for a throwaway script is a lot and for a production security fix is very little. Speed is only a win if the result is correct."
  - q: "Does any mode let me skip verifying the result?"
    a: "No — and that's the throughline of all four modes. Whether you accepted one line of ghost text or a whole agent-built feature, the work isn't done until you've verified it does what you intended. [Verification](/learn/ai-software-dev/verification-and-trust/) is the constant; only the *amount* you delegate before it changes."
---
# Choosing where on the spectrum to work

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**One spectrum** — completion, chat, agentic, and vibe coding are points on a single line of delegation. **The trade-off** — more delegation buys speed but costs understanding and adds risk. **Choose per task** — pick a point by stakes, familiarity, and reversibility, and verify the result no matter where you land.
</div>

This is lesson 18 of the path, and it ties the last three together. We've met four ways of working with a model; this lesson shows they're not four separate tools but one continuous **spectrum of delegation**. By the end you'll be able to look at any task and decide *how much* to hand over — and you'll see why the one thing that never changes, wherever you land, is the need to verify.

## Four modes, one spectrum

Line the modes up and a pattern appears immediately. The thing that varies from one to the next is *how much of the work you hand to the model* — how much you delegate.

- **[Completion](/learn/ai-software-dev/code-completion/)** — *you drive, the AI assists token by token.* You write the code; the model finishes lines. Maximum control, minimum delegation.
- **[Chat](/learn/ai-software-dev/chat-assisted-coding/)** — *you drive, the AI writes chunks you review.* You direct each step and review every answer, but the model produces larger pieces.
- **[Agentic](/learn/ai-software-dev/agentic-and-vibe-coding/)** — *the AI drives, you supervise.* You set a goal; the model plans, edits, and runs, while you watch and steer.
- **[Vibe coding](/learn/ai-software-dev/agentic-and-vibe-coding/)** — *the AI drives, you barely read it.* You describe what you want and accept the result largely without reviewing the code.

That's a smooth gradient from "your hands on every keystroke" to "your hands off entirely," not four unrelated boxes. You can even move along it within a single session: vibe-code a rough prototype, then drop back to chat to harden the part that matters, then use completion as you polish it line by line.

## The core trade-off

Moving up the spectrum always trades the same currencies. It's worth stating bluntly because it's the heart of every choice you'll make:

> **More delegation buys more speed, but it costs understanding and adds risk.**

Hand over more and you go faster — you're describing instead of typing. But you also understand less of what ended up in your codebase, and you've reviewed less of it, so more can be wrong without you noticing. Hand over less and you're slower, but you understand every line and you've checked it as you went.

Neither end is "correct." A senior engineer hand-typing boilerplate they could have completed in a keystroke is wasting the tool; someone vibe-coding a payment system is misusing it. The skill isn't picking a favourite mode — it's choosing the right point on the spectrum *for the task in front of you*.

## How to choose: three questions

Three questions decide how far up the spectrum a task can safely go.

**1. What are the stakes?** What happens if the code is wrong? A throwaway script that misformats some output costs you a rerun. A bug in GopherTrunk's core demodulation path silently corrupts every decode. High stakes mean you need understanding and review, which means *less* delegation. Low stakes free you to delegate more.

**2. How familiar are you with the code?** When you know the area well, you can review an agent's output fast and catch its mistakes, so heavier delegation stays safe — your judgment is the backstop. In code you don't understand, you can't tell a correct change from a plausible-looking wrong one, so delegating heavily means flying blind. Counterintuitively, *unfamiliar, high-stakes* code is where you should delegate *least*, even though it's tempting to let the model handle what you don't know.

**3. How reversible is the change?** This is your safety net, and it's mostly made of [version control](/learn/intro-software-dev/version-control-collab/). A change on a fresh branch that you can revert with one command is cheap to get wrong — so you can delegate more and recover if it goes sideways. A change that's hard to undo — a database migration, a release that's already shipped, an irreversible side effect — demands caution and review regardless of how routine it looks. Before you let an agent loose, commit your work, so the worst case is "throw away the branch," not "lose what I had."

Run a task through those three and the right amount of delegation usually falls out. Adding a unit test to familiar code on a branch? Delegate freely. Touching the security boundary of a service in code you've never read? Drive it yourself.

## A mode-selection table

Pulling it together:

| Mode | Delegation | Risk | Good-fit tasks |
|------|-----------|------|----------------|
| **Completion** | Lowest — you write, AI finishes lines | Lowest | Boilerplate, repetitive patterns, the obvious next line, while writing code you understand |
| **Chat** | Low–medium — AI writes chunks you review | Low | Explanations, refactors, tests, focused fixes, anything you want to discuss and check |
| **Agentic** | High — AI plans, edits, runs; you supervise | Medium–high | Multi-file features, scaffolding, "make the tests pass," routine work in familiar areas on a branch |
| **Vibe coding** | Highest — you barely read the code | High | Prototypes, throwaway scripts, learning, low-stakes exploration |

Read the table as guidance, not law. The "good-fit" column shifts with the three questions above: a refactor in code you know on a safe branch can be agentic; the same refactor in a fragile, unfamiliar, hard-to-revert module belongs in chat or under your own hands.

## The throughline: every mode ends in verification

Here's the one thing the spectrum does *not* change. Whether you accepted a single line of ghost text or let an agent build a whole feature, the work isn't finished until you've confirmed it does what you intended. **Verification is the constant; only the amount you delegate before it varies.**

What changes across the spectrum is *how* you verify and how much there is to check. With completion you verify a line as you read it. With chat you verify each answer before using it. With agentic coding you verify a larger result — run the tests, read the diff, exercise the behaviour. With vibe coding, the temptation is to skip verification entirely because it "seems to work" — and that's precisely where the spectrum bites back, because the unread code carries the most unverified risk.

This deserves a lesson of its own, and it gets one — [Verification & trust](/learn/ai-software-dev/verification-and-trust/) — but the principle belongs here, at the moment you choose your mode: *decide how you'll verify before you decide how much to delegate.* If a task's verification is cheap and reliable (a fast test suite, an easily-checked output), you can delegate more, because mistakes will surface. If verification is hard, delegate less, because mistakes will hide. The model can write the code in any mode; making sure it's *right* is always your job.

<div class="knowledge-check" data-quiz data-correct-msg="Right — more delegation trades understanding and adds risk in exchange for speed." markdown="0">
  <p class="knowledge-check__q">Quick check: As you move toward more delegation on the spectrum, what is the trade-off?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">You gain both more speed and more understanding at no cost</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">You gain speed but lose understanding and take on more risk</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">You lose speed but eliminate the need to verify the result</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **One spectrum** — completion, chat, agentic, and vibe coding are points on a single line from "you drive" to "the AI drives."
- **The trade-off** — more delegation buys speed but costs understanding and adds risk; neither end is universally right.
- **Stakes** — the higher the cost of being wrong, the less you should delegate and the more you should review.
- **Familiarity** — delegate more in code you know and can check fast; delegate least in unfamiliar, high-stakes code.
- **Reversibility** — version control is your safety net, so easily-reverted changes can carry more delegation than irreversible ones.
- **Verification is constant** — every mode ends in confirming the result does what you intended; only how much you delegate beforehand changes.

Next up: now that you know *how much* to hand the model, learn how to ask for it well — the craft of writing prompts that get good code — see [Prompting for code](/learn/ai-software-dev/prompting-for-code/).
