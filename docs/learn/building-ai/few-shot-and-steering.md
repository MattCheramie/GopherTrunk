---
slug: few-shot-and-steering
title: Few-shot examples & steering output
description: How to make a model's output consistent by showing it a handful of worked input-output examples — zero-shot vs few-shot prompting, in-context learning, and the other steering levers (format instructions, constraints, a role, and temperature).
keywords: few-shot prompting, zero-shot, in-context learning, examples, steering output, temperature, output format, classification prompt, prompt engineering, consistency
level: intermediate
status: full
prereq:
  - prompts-as-code
faq:
  - q: "What is the difference between zero-shot and few-shot prompting?"
    a: "Zero-shot means you give the model only instructions and let it answer. Few-shot means you also include a handful of worked input-output examples in the prompt, so the model matches their pattern instead of guessing at a format. Few-shot is the strongest, cheapest way to lock output that has to look a certain way."
  - q: "How many examples should I include?"
    a: "Usually two to five. Quality beats quantity: a few representative, correct, and diverse examples steer better than a dozen near-identical ones. Every example also costs tokens in the context window, so keep them short and stop adding once the output is stable."
  - q: "What does temperature do?"
    a: "Temperature controls how random the model's next-token choices are. Lower temperature makes output more deterministic and consistent — good when you want the same shape every time — while higher temperature makes it more varied. It is one steering lever among several; see how models decide."
  - q: Can a bad example hurt the output?
    a: A wrong or inconsistent example actively misleads the model, because it copies the pattern you showed rather than the one you meant. Check every example is correct and that they all follow the same format before you rely on them.
---

# Few-shot examples & steering output

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Zero-shot** gives the model instructions only; **few-shot** adds a handful of
worked **in-context examples** so it matches their pattern. Showing the model a
few input-output examples is the strongest, cheapest way to make output
**consistent** — far more reliable than asking it to "be consistent." Examples
are the sharpest of several **steering** levers, alongside format instructions,
constraints, a role, and temperature. Builds on
[prompts as code](/learn/building-ai/prompts-as-code/).
</div>

You have a prompt that works, but the output drifts — mostly the right shape,
occasionally a stray label or a reformatted answer that breaks your parser. The
fix is rarely a longer instruction. It's to *show* the model what you want
instead of only telling it. This lesson covers few-shot prompting and the other
levers you use to steer output toward the shape your code needs.

## Zero-shot vs few-shot

**Zero-shot** is the default: you write instructions and the model answers with
no examples to copy. It's fine when the task is common and the format is obvious
— "summarise this in one sentence" needs no demonstration.

**Few-shot** means you include a handful of worked examples — each an input
paired with the output you'd want for it — right in the prompt, before the real
input. The model reads the pattern in those examples and continues it. This is
called **in-context learning**: the model isn't retrained or fine-tuned, it just
picks up the pattern from what's in the context window for this one call. The
examples are gone the moment the request ends.

The shift from telling to showing is the whole idea. An instruction describes the
output; an example *is* the output. When the two could conflict — a described
format versus a demonstrated one — the demonstration usually wins, which is
exactly why examples steer so hard.

## When few-shot pays off

Few-shot earns its token cost when the output has to be *consistent*, not just
correct. The cases where it helps most:

- **Locking output format** — you need the same JSON keys, the same column order,
  the same delimiter every time so downstream code can parse it.
- **Tone and style** — matching a house voice, a terse log line, or a specific
  level of detail that's hard to describe but easy to show.
- **Tricky edge cases** — one example of the awkward input (an empty field, a
  negative number) teaches the model how you want it handled.
- **Classification label sets** — pinning the model to *your* exact labels rather
  than synonyms it invents.

That last one is the clearest win. Here's a short two-shot classification prompt
that fixes the label set by demonstration:

```text
Classify each radio log line as: ROUTINE, DEGRADED, or OUTAGE.
Reply with only the label.

Input: control channel locked, SNR 22 dB, all sites reporting
Label: ROUTINE

Input: site 3 dropped off the network, calls failing over to site 4
Label: DEGRADED

Input: no control channel decode for 90 seconds across all sites
Label:
```

The two examples do what a paragraph of instructions can't: they show the model
that "SNR 22 dB, all reporting" is `ROUTINE` and a failover is `DEGRADED`, so it
answers the third line with one of your three labels — not `WARNING`, not a
sentence.

## Other steering levers

Examples are the sharpest tool, but not the only one. Reach for these alongside
or instead of few-shot:

- **Explicit format instructions.** State the shape you want directly — "reply as
  a JSON object with keys `label` and `confidence`." Pair it with an example for
  the strongest effect.
- **Hard constraints.** Rule out the failure you keep seeing: "respond with only
  the JSON, no preamble, no code fence." Constraints remove options the model
  would otherwise take.
- **A role or persona.** "You are a terse network operator" sets a default voice
  and level of detail for the whole response without you specifying each detail.
- **Temperature.** This dial controls how random the model's choices are — lower
  means more deterministic and repeatable, higher means more varied. For output
  that must be consistent, turn it down. It's covered from the model's side in the
  sibling path's [how models decide](/learn/ai-software-dev/how-models-decide/).

These stack: a role, a format instruction, one or two examples, and a low
temperature together will steer far harder than any one alone. When you need a
*guaranteed* structure rather than a strongly-nudged one, there's a stronger tool
still — the API can constrain the output shape directly, which we cover in
[structured output](/learn/building-ai/structured-output/).

## Quality over quantity

More examples is not better. A few **representative, correct, and diverse**
examples steer better than a pile of near-identical ones — the model learns the
*range* of the task from variety, not from repetition. Three examples that cover
a routine case, an edge case, and an ambiguous case teach more than ten that all
look the same.

Two cautions. First, a **wrong example actively misleads**: the model copies the
pattern you showed, so an inconsistent or mislabelled example teaches the mistake.
Check every example before you trust it. Second, examples **cost tokens** — they
sit in the context window on every call, adding to latency and price, which is
exactly the budget you're managing in
[cost, latency & limits](/learn/building-ai/cost-latency-and-limits/). Keep them
short and drop any that isn't pulling its weight.

## Iterating on examples

Few-shot prompts get better the same way code does: by responding to failures you
actually see. Run the prompt on real inputs, watch where the output drifts, and
**add one example for each failure mode** — an edge case it fumbled, a label it
invented, a format it broke. Each example inoculates against that specific error.

Don't guess at examples in advance; you'll spend tokens defending against
problems that never happen. Add them in response to observed drift, and *test*
that the change helped rather than assuming it did — measuring whether an output
change is really an improvement is its own discipline, covered in
[evaluating AI features](/learn/building-ai/evaluating-ai-features/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — one or two examples showing the exact format is the cheapest reliable way to lock output shape." markdown="0">
  <p class="knowledge-check__q">Quick check: the model mostly nails the output format but occasionally drifts. What's the cheapest reliable fix?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Add "be consistent and always use the right format" to the instructions</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Add one or two examples showing the exact format (few-shot)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Raise the temperature so the model considers more options</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Zero-shot** is instructions only; **few-shot** adds worked input-output
  examples the model matches — a.k.a. **in-context learning**.
- Showing a few examples is the **strongest, cheapest** way to make output
  consistent — better than telling the model to be consistent.
- Few-shot pays off for **locking format, tone, edge cases, and label sets**;
  a short two-shot classification prompt pins the model to your exact labels.
- Other **steering** levers: explicit format instructions, hard constraints, a
  role, and **temperature** (lower = more consistent).
- **Quality over quantity** — a few correct, diverse examples beat many similar
  ones; wrong examples mislead, and every example costs tokens.
- **Iterate**: add an example per failure mode you observe, then test that it
  helped.

Next up: [getting usable output back](/learn/building-ai/parsing-and-validating-output/).
