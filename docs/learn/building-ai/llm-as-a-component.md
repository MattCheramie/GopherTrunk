---
slug: llm-as-a-component
title: The model as a probabilistic component
description: A language model is a new kind of software component — probabilistic instead of deterministic, sometimes confidently wrong, and slow, metered, and remote. Learn why you must design around it rather than call it like an ordinary function.
keywords: probabilistic component, non-deterministic model, LLM as a dependency, model output varies, hallucination, verify model output, latency and cost, rate limits, network dependency, designing around a model, unreliable dependency
level: beginner
status: full
prereq:
  - building-vs-using-ai
faq:
  - q: Why does the model give a different answer each time I run it?
    a: Because it is **probabilistic**, not deterministic. It predicts likely next words and samples from them, so the same input can produce different wording — or different content — on each call. This is normal behaviour, not a bug, and your code has to expect it.
  - q: "If the output looks right, can I trust it?"
    a: "No. A model can be confidently wrong — fluent, plausible text that is factually incorrect (a \"hallucination\"). Plausible is not the same as correct, so you verify model output before you act on it, the same way you'd check an unfamiliar colleague's work."
  - q: "How is calling a model different from calling a normal function?"
    a: "An ordinary function is deterministic, fast, free, and reliable. A model call is the opposite on every axis: the output varies, it takes real time, each call costs money, and it runs on a remote server that can be slow, rate-limit you, or be down. Treat it like a network dependency, not a local function."
  - q: How should I design around something this unreliable?
    a: Give clear instructions, constrain what the output can be, validate it before use, expect the wording to vary, handle slow or failed calls gracefully, and keep an eye on cost. In short, treat the model like a powerful but flaky component whose work you always review.
---

# The model as a probabilistic component

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A language model is a new kind of software component: **probabilistic** and
**non-deterministic** (the same input can give a different output), sometimes
confidently wrong, and slow, metered, and remote. The shift is to treat it like a
powerful but flaky remote component — an **unreliable dependency** you design
around — never like an ordinary function that returns the same answer every time.
Start with [building vs. using AI](/learn/building-ai/building-vs-using-ai/), and
for *why* the output varies see [how a model decides](/learn/ai-software-dev/how-models-decide/).
</div>

If you've written software before, you have a deep instinct that a function is
reliable: call it with the same input and you get the same output, instantly and for
free. A language model breaks that instinct. It's still a component you call from your
code, but it behaves nothing like the functions you're used to — and building with it
well starts by understanding exactly how it's different.

## A new kind of component

Think about an ordinary function or library you already use — say, one that sorts a
list or formats a date. You can count on it completely:

- **Deterministic** — the same input always gives the same output.
- **Fast** — it returns in microseconds.
- **Free** — calling it costs nothing.
- **Reliable** — it's right there in your process; it doesn't fail or disappear.

A language model is the opposite on every one of those axes. It's a component you
call, but a strange one: give it the same input twice and you may get two different
answers; it takes a noticeable moment to respond; every call costs money; and it lives
on a server across the network that can be slow, busy, or down.

| | Ordinary code | A language model |
|---|---|---|
| **Output** | Deterministic — same in, same out | Probabilistic — same in, maybe different out |
| **Speed** | Microseconds | Slow-ish — often a second or more |
| **Cost** | Free | Metered — you pay per call |
| **Reliability** | Always there, never fails | Remote — can be slow, rate-limited, or down |

None of this makes the model bad. It makes it *different*, and the whole craft of
building AI features is working with these properties instead of being surprised by
them.

## Probabilistic, not deterministic

The single biggest shift is this: a model's output is **probabilistic**. Under the
hood it predicts likely next words and then *samples* from those possibilities, so the
exact text it produces can vary from call to call — even with identical input. (The
[how a model decides](/learn/ai-software-dev/how-models-decide/) lesson unpacks the
sampling and "temperature" that drive this.)

The practical implication for your code: **don't assume you'll get an exact string.**
Ask the model to summarise something twice and you might get "The meeting is at 3pm" one
time and "Meeting scheduled for 3:00 PM" the next — both correct, worded differently. So
don't write brittle checks that demand one precise output, like testing whether the
result *equals* a specific sentence. Design for variance instead: look for the
information you need, not for one exact phrasing.

## Powerful but wrong sometimes

The second shift is harder to accept: a model can be **confidently wrong**. It will
produce fluent, authoritative-sounding text that is simply incorrect — a made-up fact,
a wrong number, a function that doesn't exist. This is often called a
**hallucination**, and it's not an occasional glitch; it's a property of how the model
works.

The trap is that the output *looks* right. It's well-formed and plausible, and plausible
is exactly what a model is built to produce — but **plausible is not the same as
correct**. That means you can never take the output on faith. You **verify** it before
acting on it: check facts against a real source, run the code, validate the values. Later
lessons go deep on this — [evaluating AI features](/learn/building-ai/evaluating-ai-features/)
and [handling hallucinations and failure](/learn/building-ai/handling-hallucinations-and-failure/)
— and the sibling path makes it a rule in
[verification and trust](/learn/ai-software-dev/verification-and-trust/).

## Slow, metered, and remote

The third shift is practical. A model call is not a local function — it's a request to
a service on someone else's servers, and that has real consequences:

- **Latency.** A response takes time, often a second or more. If your feature calls the
  model, it will feel slower than pure local code, and you have to design the experience
  around that wait.
- **Cost.** Each call is **metered** — you pay for it. Calls in a loop, or over lots of
  data, add up. [Cost, latency, and limits](/learn/building-ai/cost-latency-and-limits/)
  covers this in detail.
- **It can fail.** Being remote means it's a **network dependency**. The call can time
  out, the service can be busy and reject you (a rate limit, often an HTTP 429), or the
  provider can have an outage. Your code has to handle a call that doesn't come back.

Treat a model call the way you'd treat any network request: assume it might be slow or
fail, and have a plan for when it does.

## Designing around it

Put those three shifts together and a working stance falls out. You don't call a model
and hope; you build a small amount of structure around it:

- **Give clear instructions.** The more precisely you say what you want, the better and
  more consistent the output.
- **Constrain the output.** Ask for a specific, limited shape (a short answer, a set
  list of choices) so there's less room to wander.
- **Validate before you use it.** Check the output is well-formed and sensible before
  acting on it — never trust it blind.
- **Expect variance.** Assume the wording will change run to run; don't build on exact
  strings.
- **Handle latency and failure.** Plan for the wait, and for the call that times out or
  gets rate-limited.
- **Bound the cost.** Know roughly what each call costs and don't let calls multiply
  unchecked.

A useful mental model: the model is like a **brilliant but unreliable junior** on your
team. It's fast, knowledgeable, and tireless, and it produces impressive work — but it
sometimes gets things confidently wrong, so you give it clear direction and you always
review what it hands back before it goes anywhere important.

<div class="knowledge-check" data-quiz data-correct-msg="Right — models are probabilistic, so varying wording is expected. Design for variance rather than exact strings." markdown="0">
  <p class="knowledge-check__q">Quick check: your feature returns slightly different wording each run in production. Is that a bug?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Yes — the model is broken and needs to be replaced</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">No — models are probabilistic; design for variance</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Yes — a correct model always returns the exact same text</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A model is a **component you call**, but unlike ordinary code it's probabilistic,
  slow-ish, metered, and remote.
- **Probabilistic, not deterministic** — the same input can give different output, so
  don't rely on exact strings or brittle equality checks.
- **Confidently wrong sometimes** — output can be fluent but incorrect (a
  hallucination); plausible is not correct, so you verify before acting.
- **Slow, metered, remote** — expect latency and per-call cost, and handle timeouts and
  rate limits like any network dependency.
- **Design around it** — clear instructions, constrained output, validation, and a plan
  for variance, failure, and cost.
- Treat it like a **brilliant but unreliable junior** whose work you always review.

Next up: [how a model API call works](/learn/building-ai/how-a-model-api-works/).
