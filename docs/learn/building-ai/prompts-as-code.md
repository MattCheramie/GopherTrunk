---
slug: prompts-as-code
title: Prompts as code — system prompts & templates
description: In a real feature a prompt is a versioned, testable artifact, not a throwaway chat message. Learn to split stable instructions from per-request data, build prompt templates, and manage prompts as they grow.
keywords: system prompt, prompt template, prompt engineering, instructions vs data, versioning prompts, roles, prompt management, prompt registry, output format, prompt files
level: intermediate
status: full
prereq:
  - how-a-model-api-works
faq:
  - q: What belongs in the system prompt versus the user message?
    a: "Put stable instructions in the system prompt — the persona, the rules, the output-format spec, anything that's the same on every call. Put the variable, per-request input in the user message. Keeping the two apart makes prompts easier to reuse and safer, because untrusted data never mixes into your instructions."
  - q: What is a prompt template?
    a: "A prompt template is a fixed scaffold of text with labelled slots that you fill in at runtime — the same skeleton every call, with only the variable parts substituted. Keeping the scaffold in one place (a constant or a dedicated prompt file) beats rebuilding the string with concatenation scattered across your code."
  - q: Why treat a prompt like source code?
    a: "Because in a feature the prompt decides how the model behaves, and it ships with your program. That makes it a real artifact that deserves version control, review, and testing. A tiny wording change can shift behaviour, so you want small diffs and a record of what changed and why."
  - q: How should I change a prompt that's already in production?
    a: "The same way you change code: make a small, deliberate edit, test the behaviour before and after, and keep a record of the change. Big rewrites make it impossible to tell which line moved the result, so prefer narrow diffs you can evaluate one at a time."
---
# Prompts as code — system prompts & templates

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
In a feature, a prompt isn't a throwaway chat message — it's a **versioned, testable
artifact** that ships with your code. Split a **system prompt** (stable instructions,
persona, rules, output format) from the **user** message (the per-request input): that's
the **instructions vs data** line, and it buys you clarity, reuse, and safety. Keep the
scaffold in a **prompt template** with named slots, not in scattered string
concatenation.
</div>

You've seen [how a model API call works](/learn/building-ai/how-a-model-api-works/): an
ordered list of messages, each with a role, in and generated text out. This lesson is
about the messages themselves — specifically, how to treat the prompts your program
sends as real engineering artifacts rather than text you improvise. If you've written
prompts by hand, the sibling lesson
[prompting for code](/learn/ai-software-dev/prompting-for-code/) covers the craft of a
single good request. Here we're one level up: the prompts baked *into* a feature.

## A prompt is now part of your program

When you chat with a model, a prompt is disposable — you type it, read the answer, move
on. When a prompt sits inside a feature, it's the opposite. It ships with your code. It
decides how the feature behaves for every user. Change one line and the behaviour of a
shipped product changes.

That makes a prompt a first-class artifact, and it deserves the same treatment as any
other load-bearing code: **version control** so you can see what changed, **review** so a
second person sees the change before it ships, and **testing** so you know the new
wording didn't quietly break something. A prompt stored as a literal buried in a function
is easy to tweak carelessly; a prompt you treat as code is one you change on purpose.
We'll come back to how you actually measure a prompt change in
[evaluating AI features](/learn/building-ai/evaluating-ai-features/).

## System vs user: separate instructions from data

The single most useful structural habit is to split **instructions** from **data**.

- The **system** prompt holds everything stable: the persona ("you are a careful radio-
  systems assistant"), the rules it must follow, and the exact output format you expect
  back. This is the same on every call.
- The **user** message holds the variable, per-request input — the one thing that differs
  from call to call, like the record to summarise or the question to answer.

Three reasons this split pays off. **Clarity**: the standing rules live in one place
instead of being re-typed into every request. **Reuse**: the same system prompt serves
thousands of different user inputs unchanged. And **safety**: untrusted input — anything
that came from a user, a file, or the web — does not belong in your instructions. If you
paste a user's text straight into the instruction block, text like "ignore your previous
rules" is read *as an instruction*. Keeping it in the user slot, clearly marked as data,
is the first line of defence. We'll return to feeding input in
[context and user input](/learn/building-ai/context-and-user-input/), and to the attack
itself in [prompt injection and security](/learn/building-ai/prompt-injection-and-security/).

## Prompt templates

Most feature prompts aren't a single fixed string — they're a **fixed scaffold with
slots** filled in at runtime. The scaffold (the instructions, the structure, the format
spec) stays constant; only the slots change per request. That scaffold is a **prompt
template**.

Keep the template in source — a constant, or a dedicated prompt file next to your code —
not assembled from string concatenation scattered across the codebase. One place to read,
review, and version beats hunting for the pieces.

```text
System template
---------------
You are a radio-systems assistant. Summarise the trunked-system record below
for a scanner operator. Be factual and concise. Do not invent fields.

Output exactly these lines:
  System: <name>
  Type:   <protocol>
  Sites:  <count>

Record:
{{record}}
```

At runtime you substitute the `{{record}}` slot with the actual data and send it. The
instructions never move; only the slot's contents do. Because the scaffold lives in one
named place, you can diff it, test it, and reuse it across every record you process.

## Writing a good instruction prompt

A template is only as good as the instructions inside it. The same rules that make a
hand-written request work apply here:

- **Be specific and state the task** — say exactly what the model should produce, not a
  vague gesture at it.
- **Give the constraints** — length, tone, what to include, what to leave out.
- **Pin the output format** — if your code parses the reply, the format is not optional;
  spell it out precisely (the example above fixes three exact lines).
- **Give the model a role** — a short persona ("you are a careful radio-systems
  assistant") steadies its behaviour.
- **Say what to do when unsure** — tell it to leave a field blank or say "unknown" rather
  than invent one. That single instruction prevents a lot of confident nonsense.

The craft of writing these instructions is covered in depth in
[prompting for code](/learn/ai-software-dev/prompting-for-code/) — the same skills apply
whether you type the prompt once or bake it into a feature.

## Iterate deliberately

Treat a prompt change exactly like a code change. A prompt is deceptively sensitive: a
tiny wording change — swapping "summarise" for "list", adding a single sentence — can
shift the model's behaviour more than you'd expect. That cuts both ways, so you change
prompts the way you change code:

- **Small diffs.** Change one thing at a time so you can tell what moved the result.
- **Test before and after.** Run the same inputs through the old and new prompt and
  compare. Don't ship on a hunch that it "feels better".
- **Keep a record.** The prompt is in version control, so the diff *is* the record of
  what changed and why.

A big rewrite that changes ten things at once tells you nothing about which change
helped. How you measure "better" rigorously is its own subject —
[evaluating AI features](/learn/building-ai/evaluating-ai-features/) picks that up.

## Managing prompts as they grow

One prompt is easy. A dozen, reused across features and edited by several people, is not.
As they multiply, treat them as managed assets:

- **Centralize** them so a prompt isn't duplicated in three functions that then drift
  apart.
- **Name and version** each one, so you can refer to "the summariser prompt, v3" and know
  exactly what that is.
- **Consider a prompt file or registry** — a dedicated location, or a small module, that
  holds the templates and hands them out. Your code asks for a prompt by name instead of
  carrying the text inline.

These often live alongside other **prompt and config files** that steer a model's
standing behaviour — the same idea as the config files covered in the sibling lesson on
[skills and config files](/learn/ai-software-dev/skills-and-config-files/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — stable instructions belong in the system prompt; the variable per-request input goes in the user message." markdown="0">
  <p class="knowledge-check__q">Quick check: where do stable instructions belong, and where does per-request data go?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Instructions in the system prompt; variable data in the user message</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Both in the user message, so the model sees them together</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Both in the system prompt, so the rules stay first</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- In a feature a prompt is a **versioned, testable artifact** that ships with your code —
  it gets version control, review, and testing.
- Split **instructions from data**: stable rules, persona, and output format go in the
  **system** prompt; the variable per-request input goes in the **user** message.
- That split buys **clarity, reuse, and safety** — untrusted data never sits in your
  instructions.
- A **prompt template** is a fixed scaffold with named slots, kept in source, not built
  from scattered string concatenation.
- **Iterate deliberately**: small diffs, test before and after, keep a record — a tiny
  wording change can shift behaviour.
- As prompts grow, **centralize, name, and version** them; consider a prompt file or
  registry.

Next up: [feeding the model context & user input](/learn/building-ai/context-and-user-input/).
