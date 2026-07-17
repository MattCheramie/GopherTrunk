---
slug: evaluating-ai-features
title: Evaluating AI features
description: You can't assert one correct string against a probabilistic feature, so you evaluate it against a curated test set and scoring criteria — before and after every prompt, model, or retrieval change — to measure quality and catch regressions.
keywords: LLM evaluation, evals, test set, LLM as judge, regression testing, quality metrics, groundedness, pass rate, offline evaluation, online signals, scoring outputs
level: advanced
status: full
prereq:
  - llm-as-a-component
faq:
  - q: "What is an eval?"
    a: "An eval is a curated set of representative inputs plus a way to score the outputs against expected behaviour. You run it on every prompt, model, or retrieval change to measure quality and catch regressions — the AI-feature equivalent of a test suite."
  - q: "Why can't I just write a normal unit test for an AI feature?"
    a: "Because the output is **probabilistic** — the same input can produce different valid wording each time. An equality assertion on one fixed string would fail on perfectly good answers. You score against criteria across a test set instead of asserting one correct output."
  - q: "What is LLM-as-a-judge?"
    a: "It's using a model to grade outputs against written criteria — handy when correctness is about meaning, not an exact string. It has biases and limits, so keep the rubric simple, watch for judge failure modes, and spot-check its verdicts against human review."
  - q: "What's the difference between offline and online evals?"
    a: "Offline evals run your test set before you ship, so a change is measured before it reaches users. Online signals come after shipping — A/B tests, thumbs up/down, sampling real traffic — and catch what your test set didn't. You want both."
gophertrunk_links:
---

# Evaluating AI features

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A probabilistic feature has no single correct output, so you can't assert one fixed
string against it. Instead you build **evals**: a curated **test set** of representative
inputs plus a way to score outputs against criteria. You run it **before and after every
change** — prompt, model, or retrieval — to measure quality and catch **regressions**.
Deterministic checks where you can, **LLM-as-judge** or human review where you can't.
Start from [the model as a component](/learn/building-ai/llm-as-a-component/); this is
the [building-ai](/learn/building-ai/) sibling of
[verifying AI-written code](/learn/ai-software-dev/verification-and-trust/).
</div>

You wouldn't ship a change to a decoder without running the regression suite. An AI
feature needs the same safety net — but the old shape of test doesn't fit, because the
output isn't one fixed answer. This lesson is about the discipline that replaces it.

## Why ordinary tests don't fit

A normal unit test asserts that an input produces one exact output: `f(2) == 4`. That
works because the function is deterministic. A [model call is not](/learn/building-ai/llm-as-a-component/):
the same prompt can return different — but equally valid — wording each time, and a
good answer can be phrased a hundred ways.

So an equality assertion on the output is the wrong tool. It fails on answers that are
perfectly correct just because the words differ, and it tells you nothing about the
answers that are subtly wrong. Worse, the thing you most want to catch is invisible to a
single assertion: a prompt tweak or a model upgrade that fixes one case can **silently
regress ten others** you weren't looking at. You need a way to measure quality across
many cases at once.

## Evals = test set + scoring

An **eval** is two things: a curated **test set** of representative inputs, and a **way
to score** the outputs. That's it. The test set stands in for the real inputs your
feature will see; the scoring turns "looks right" into a number you can compare.

The point is to run it on **every change that can move quality** — a new prompt, a
different model, a tweak to your retrieval or chunking. You get two things from each run:
a **quality measurement** (how good is it now?) and a **regression check** (did this
change break cases that used to pass?). Without an eval, both of those are guesswork, and
a probabilistic feature gives you plenty of room to guess wrong.

## Ways to score

There's no single scoring method — pick the cheapest one that actually captures what
"correct" means for the case, and reserve the expensive methods for the cases that need
them.

- **Deterministic / structural checks.** Where the answer has a checkable property, just
  check it in code: valid JSON, contains a required value, matches a pattern, falls in a
  range. Cheap, fast, and reliable — prefer these whenever the criterion allows.
- **Reference-based.** Compare against a known-good answer — exact match for constrained
  outputs, or a similarity measure where near-enough counts.
- **Rubric / criteria.** Write down what a good answer must do (covers the key fact, no
  invented details, right tone) and score against that list.
- **LLM-as-judge.** Use a model to grade an output against your rubric — useful when
  correctness is about meaning, not an exact string. It has real biases and limits (it
  can favour longer or more confident answers, and it can be wrong), so keep the rubric
  **simple**, and **spot-check** the judge against human review rather than trusting it
  blind.
- **Human review.** For the hard, high-stakes, or ambiguous cases, a person still scores
  best. Use it deliberately on the slice that matters, not on everything.

## Building an eval set

A good eval set is grown, not written once. Start by gathering **real inputs** from
actual usage, add the **edge cases** you know are tricky, and — most valuable of all —
fold in **every past failure**: each bug or bad output becomes a permanent case so the
same mistake can't come back unnoticed.

For each case, define the **expected behaviour or criteria** — not necessarily one exact
answer, but what a correct response must and must not do. Then keep the whole set
**version-controlled beside the prompts** it exercises, the same way you'd keep
[prompts as code](/learn/building-ai/prompts-as-code/). The eval and the prompt change
together, review together, and ship together.

## Metrics that matter

The headline number is usually **pass rate** or accuracy — the fraction of cases that
meet their criteria. But a single number hides too much; track the metrics that fit the
task alongside it:

- **Groundedness / faithfulness** for retrieval features — does the answer stick to the
  sources it was given, or does it invent? Essential once you're
  [building a RAG pipeline](/learn/building-ai/building-a-rag-pipeline/), and closely tied
  to your [retrieval quality](/learn/building-ai/chunking-and-retrieval-quality/).
- **Format validity** — does the output parse and match the schema you require?
- **Refusal / safety** — does it decline what it should, and not over-refuse what it
  shouldn't?

Break these out per category or input type so a regression in one slice can't hide behind
a healthy overall average.

## Offline and online

Evals come in two moments. **Offline evals** run your test set **before you ship** — in
development and in CI — so a change is measured against known cases before any user sees
it. That's where regressions get caught early and cheaply.

But your test set can't contain every real input, so pair it with **online signals**
after shipping: A/B tests between versions, thumbs up/down from users, and sampling real
traffic to score later. When online turns up a failure your offline set missed, fold that
case back into the set. Wiring up those signals is the job of
[observability and monitoring](/learn/building-ai/observability-and-monitoring/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — the output is probabilistic, so you score against a test set and criteria, not one fixed string." markdown="0">
  <p class="knowledge-check__q">Quick check: why can't you test an LLM feature with a simple equality assertion on the output?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Because model outputs are always too long to compare</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It's probabilistic — you evaluate against a test set and criteria, not one fixed string</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Because assertions can't run inside a CI pipeline</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A probabilistic feature has **no single correct output**, so equality assertions don't
  fit — you evaluate against a test set and criteria.
- An **eval** is a curated **test set** plus a way to **score** outputs; run it on every
  prompt, model, or retrieval change.
- **Score cheaply where you can** (structural and reference checks) and reserve
  **LLM-as-judge** or human review for meaning-based cases — spot-check the judge.
- **Grow the eval set** from real inputs, edge cases, and every past failure; keep it
  version-controlled beside the prompts.
- Track **pass rate** plus task-specific metrics — groundedness, format validity,
  refusal/safety — broken out by slice.
- Run **offline** before shipping and watch **online signals** after; feed new failures
  back into the set.

Next up: [handling hallucinations & failure](/learn/building-ai/handling-hallucinations-and-failure/).
