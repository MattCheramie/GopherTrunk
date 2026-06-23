---
slug: prompting-for-code
title: Prompting for code
description: How to write a prompt that produces the code you actually wanted — a clear goal, explicit constraints, relevant context, examples, and a stated output format. Learn to plan before writing, decompose big asks, and iterate in place.
keywords: prompting for code, AI coding prompts, prompt engineering, constraints, context, system prompt, role prompt, iterative refinement, decomposition, plan before code, GopherTrunk
level: beginner
prereq:
  - the-spectrum
faq:
  - q: "Why does the model produce wrong or generic code even when I describe the task?"
    a: "Usually because the prompt left out something the model needs: the exact types or interface the code must fit, the library you want it to use, or the constraints that rule out a generic answer. The model fills gaps with the most common pattern from training, which may not match your project. Add the missing context — paste the interface, name the library, state the constraint — and the output gets specific fast."
  - q: "Should I ask the model to plan before it writes code?"
    a: "For anything non-trivial, yes. Asking it to outline an approach or explain the trade-offs first lets you catch a wrong direction before any code exists, and it tends to produce better code because the model 'reasons on paper' before committing. For a one-line change it's overkill — just ask for the change."
  - q: "Is it better to refine a result or start a new prompt?"
    a: "Refine in place when the result is close — say what's wrong and what to change, and the model keeps the parts that were right. Start fresh when the conversation has drifted, accumulated contradictions, or the model keeps reintroducing a mistake. A muddy thread is harder to steer than a clean one."
---
# Prompting for code

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**A good prompt has structure** — a clear goal, explicit constraints, relevant context, and a stated output format. **Plan before code on hard tasks** — ask the model to outline or explain its approach so you can catch a wrong direction early. **Iterate in place** — refine a close result instead of restarting, and decompose big asks into steps.
</div>

This is lesson 19 of the path, and the first of Module 5 — *directing how the model writes code*. The earlier modules covered how models work and how you reach them; the spectrum lesson laid out the range of ways to work with them, from autocomplete to agents. Now we get practical. The single biggest lever you have over the quality of AI-generated code is the prompt — the request you actually type. By the end of this lesson you'll know what separates a vague request that produces generic, wrong-shaped code from a precise one that produces code that drops straight into your project.

## What the model can and can't infer

Remember the [context window](/learn/ai-software-dev/context-windows/): the model only knows what it can see — your prompt, plus whatever files or history are in the context. It cannot read your mind, your project's conventions, or the function signature three files away unless you put it where the model can see it. A model that "doesn't get it" is almost never being dumb; it's being asked to guess.

When you leave a gap, the model fills it with the most statistically common pattern from its training. Ask vaguely for "a function to parse the data" and you'll get a generic parser for some imagined data shape — not the one that fits *your* types. The fix is to stop leaving gaps that matter. A good prompt is mostly an exercise in being explicit about things you've been carrying silently in your head.

## The anatomy of a good prompt

A strong coding prompt usually has five ingredients. You won't need all five every time, but for anything beyond a trivial edit, the more of these you supply, the closer the first result lands.

| Ingredient | What it answers | Example |
|---|---|---|
| **Goal** | What should the code *do*? | "Decode the talkgroup ID from a P25 control-channel frame." |
| **Constraints** | What rules must it obey? | Language, libraries (or "standard library only"), style, performance, error handling. |
| **Context** | What must it fit into? | The existing interface, types, struct fields, or surrounding code it has to integrate with. |
| **Examples** | What does right look like? | A sample input and the expected output, or a similar function already in the codebase. |
| **Output format** | How should the answer come back? | "Just the function, no explanation," or "explain your approach first, then the code." |

The two ingredients beginners most often skip are **context** and **constraints** — and they're the two that matter most for code. A model writing against an interface it can see will match your method names, types, and error conventions. A model guessing at them will invent plausible-but-wrong ones that you then have to rewrite.

### Plan or explain before writing — for hard tasks

For a genuinely tricky task, don't ask for code straight away. Ask the model to *plan* first: "Before writing anything, outline how you'd approach this and the trade-offs you see." This does two useful things. It lets you catch a wrong direction while it's still a paragraph, not a hundred lines you have to read and reject. And it tends to improve the code itself — letting the model lay out its reasoning before committing to an implementation produces more coherent results than demanding the answer cold. For a one-line fix, skip this; it's pure overhead. Reserve it for the prompts where being wrong is expensive.

### Decompose big asks into steps

"Build me a TETRA decoder" is not a prompt; it's a project. Large requests overwhelm the context and produce sprawling, hard-to-review output where one mistake early poisons everything after it. Break the work into steps you can review one at a time: first the frame-synchroniser, then the de-interleaver, then the channel decoder, wiring each into the last. You stay in control, each piece is reviewable, and the model isn't trying to hold the whole system in its head at once. This mirrors how you'd [decompose the problem](/learn/intro-software-dev/clean-code/) writing it yourself.

## Iterate in place, don't restart

When a result is *close* but not right, your instinct might be to rewrite the whole prompt and try again. Usually that's wasteful. The better move is to refine in place: tell the model specifically what's wrong and what to change, and let it keep the parts that were already correct.

"That's close. Use a `uint32` for the talkgroup field instead of `int`, and return an error instead of panicking on a short frame" is a far better follow-up than starting over — the model keeps the working structure and just adjusts the two things you named. Restarting throws away correct work and re-rolls the dice on it.

The exception is a thread that has gone *muddy* — full of contradictions, half-abandoned directions, or a mistake the model keeps reintroducing no matter what you say. When steering costs more than starting clean, start clean. We'll dig into managing that context deliberately in [Feeding the model the right context](/learn/ai-software-dev/providing-context/).

## Bad vs. good: a side-by-side

Here's the same task expressed two ways. The task: add a helper to GopherTrunk that turns a decoded talkgroup ID into a human-readable label.

```text
BAD
-----
write a function to get the talkgroup name


GOOD
-----
Write a Go method on the existing `*AliasTable` type that resolves a
talkgroup ID to a display label. It must satisfy this interface, which
already exists in internal/scanner/alias:

    type Resolver interface {
        // Label returns the human label for a talkgroup, and ok=false
        // if the ID is unknown.
        Label(tg uint32) (label string, ok bool)
    }

Constraints:
- Standard library only; no new dependencies.
- Lookups are hot (called per decoded frame), so it must be O(1) and
  must not allocate on the hit path.
- Unknown IDs return ("", false), never an error and never a panic.

The alias data is already loaded into the table's `byID map[uint32]string`
field. Match the existing error/return style in that package.

Output: just the method, no prose. Then a one-line note on any assumption
you had to make.
```

The bad prompt forces the model to invent everything: the language, the type it hangs off, the signature, what happens on a miss, whether allocation matters. You'll get *a* function, but almost certainly not one that fits. The good prompt removes every guess. It names the language and the exact interface, states the performance and error constraints, points at the field holding the data, and specifies the output format. The result drops into the package with little or no editing — and the model's "one-line note on assumptions" surfaces anything you forgot to pin down.

Notice what the good prompt is *not*: it isn't longer for the sake of it, and it isn't padded with politeness. Every line removes a specific ambiguity. That's the skill — not writing more, but writing the parts the model can't infer.

## A note on the system (role) prompt

Everything above is about the *one* request you're making right now. There's a second, quieter layer: the **system prompt** (sometimes called a role or custom-instructions prompt). This is standing text the tool sends ahead of every conversation — "you are a careful Go engineer; prefer the standard library; always write a test for bug fixes." It sets behaviour you don't want to retype each time.

The system prompt and a per-request prompt work together: standing rules live in the system prompt, the specific task lives in your message. You generally don't want project conventions cluttering every prompt — that's exactly what the next lesson is about. So we'll leave the details there.

<div class="knowledge-check" data-quiz data-correct-msg="Right — naming the exact interface and types the code must fit removes the guesswork that produces generic, wrong-shaped output." markdown="0">
  <p class="knowledge-check__q">Quick check: your prompt keeps producing code with the wrong function signatures and types. What's the most direct fix?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Add "please be accurate and careful" to the prompt</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Paste the exact interface and types the code must fit into the prompt</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Ask for a longer, more detailed answer each time</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Be explicit, not telepathic** — the model only knows what's in its context, so put the goal, constraints, and relevant types where it can see them.
- **Five ingredients** — goal, constraints, context, examples, and output format; supply the ones a non-trivial task needs.
- **Plan before code on hard tasks** — ask for an outline or explanation first so you catch a wrong direction before it becomes a hundred lines.
- **Decompose big asks** — break a project-sized request into reviewable steps instead of one sprawling prompt.
- **Iterate in place** — refine a close result by naming what to change; restart only when the thread has gone muddy.
- **System prompt sets standing behaviour** — per-request prompts carry the task; standing rules belong in a config the tool reads every time.

Next up: how to stop repeating your conventions in every prompt by giving the model standing instructions it reads automatically — see [Skills & AI config files](/learn/ai-software-dev/skills-and-config-files/).
