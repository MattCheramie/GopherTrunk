---
slug: handling-hallucinations-and-failure
title: Handling hallucinations & failure
description: A model will sometimes be confidently wrong, and the service behind it will sometimes be slow or down. Learn to plan for both — reduce and catch bad output, fail safe with fallbacks and human review, and stay resilient when the call fails.
keywords: hallucination, failure handling, fallback, guardrails, human in the loop, grounding, retries, graceful degradation, circuit breaker, operational resilience
level: intermediate
status: full
prereq:
  - llm-as-a-component
faq:
  - q: "What is the difference between content failure and operational failure?"
    a: "Content failure is when the call succeeds but the answer is wrong or hallucinated — it looks fine, so nothing errors. Operational failure is when the call itself doesn't return — a timeout, a rate limit (HTTP 429), or an outage. They need completely different handling, and a real feature plans for both."
  - q: "How do I stop a model from hallucinating?"
    a: "You can't stop it entirely, but you can cut it sharply. Ground the model in retrieved facts, instruct it to say it doesn't know rather than guess, ask for citations, lower the temperature, and keep each task narrow and well-specified. Then catch what slips through by validating the output before you act on it."
  - q: "When do I need a human in the loop?"
    a: "Whenever an action is high-stakes or hard to undo — spending money, deleting data, sending an irreversible message. For those, the model proposes and a person confirms. Cheap, reversible actions can run automatically; irreversible ones should not act on unverified output."
  - q: "What should happen when the model call fails?"
    a: "Fail safe rather than crash. Retry transient errors with backoff, set a timeout so a slow call can't hang your feature, and have a fallback ready — a cached or default response, a simpler model, or an honest \"I'm not sure right now.\" A failed call should degrade the experience, not break it."
---

# Handling hallucinations & failure

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A model will sometimes be **confidently wrong** — a **hallucination** that looks
completely fine — and the service behind it will sometimes be slow or down. A real
feature plans for all three: wrong, slow, and unavailable. Reduce bad output and add
**guardrails** to catch it, **fail safe** with fallbacks instead of crashing, and put a
**human-in-the-loop** before anything high-stakes or irreversible. Start from
[the model as a probabilistic component](/learn/building-ai/llm-as-a-component/).
</div>

Once you accept that a [model is a probabilistic component](/learn/building-ai/llm-as-a-component/),
the next job is engineering. The model *will* hand you a wrong answer that looks right,
and the call *will* sometimes fail to come back at all. Hoping it won't isn't a plan.
This lesson is about the structure you build around the model so that when it misbehaves
— and it will — your feature degrades gracefully instead of doing something wrong or
falling over.

## Two kinds of failure

There are two completely different ways an AI feature fails, and you have to handle both.

- **Content failure.** The call succeeds and returns text, but the *content* is wrong —
  a made-up fact, a bad number, an invented API. This is the dangerous one because
  nothing errors: the output is fluent and plausible, so it sails straight through unless
  you check it.
- **Operational failure.** The call itself doesn't come back — it times out, the service
  rate-limits you (an HTTP 429), or the provider has an outage. This is the ordinary kind
  of failure any network dependency has, and it announces itself with an error.

They need different defences. Content failure is fought with grounding, validation, and
review; operational failure is fought with retries, timeouts, and fallbacks. A feature
that handles only one of these is only half-built.

## Reducing hallucination

The cheapest bad output to handle is the one that never happens. You can't eliminate
hallucination, but a few habits cut it sharply:

- **Ground the model in real data.** Instead of asking the model what it "knows," give it
  the relevant facts in the prompt and ask it to answer *from those*. This is the whole
  point of [grounding and retrieval](/learn/building-ai/grounding-and-retrieval/) — the
  single biggest lever on accuracy.
- **Give it an out.** Instruct it explicitly: "If the answer isn't in the context, say you
  don't know." A model told it may decline is far less likely to invent something.
- **Ask for citations.** Requiring the model to point at its source both improves grounding
  and gives you something to check.
- **Lower the temperature.** For factual, structured tasks, less randomness means fewer
  creative wrong turns.
- **Keep tasks narrow.** A small, well-specified job has less room to wander than a vague,
  sprawling one. Break big asks into checkable steps.

## Catching bad output

Reduction isn't proof. Assume some wrong output gets through and build a net for it before
it reaches anything that matters.

- **Validate structure and values.** If you asked for JSON, confirm it parses and has the
  right fields; if you asked for a number, confirm it's in a sane range. See
  [parsing and validating output](/learn/building-ai/parsing-and-validating-output/).
- **Run sanity and consistency checks.** Does the answer contradict the input? Is a total
  the sum of its parts? Cheap logic catches a lot.
- **Add guardrail checks.** Screen for the specific failure modes that matter to your
  feature — off-topic replies, unsafe content, values that violate a business rule.
- **Never auto-act on unverified output.** If the model's answer triggers an action —
  especially through [tool use and function calling](/learn/building-ai/tool-use-and-function-calling/)
  — verify it *before* the action fires, not after.

## Failing safe

When something is wrong or missing, the feature should fail *safe* — degrade to a
sensible state rather than break or do the wrong thing.

- **Fallbacks and graceful degradation.** Have a lesser-but-fine answer ready: a cached
  result, a sensible default, a plain "I'm not sure — here's what I can tell you," or a
  handoff to a non-AI path. A narrower correct answer beats a confident wrong one.
- **Human-in-the-loop for high stakes.** When an action is expensive or irreversible —
  spending money, deleting records, sending something you can't unsend — the model
  *proposes* and a person *confirms*. Automate the cheap, reversible actions; gate the
  ones you can't take back.

The rule of thumb: the higher the cost of being wrong, the more the design should refuse
to act on the model's word alone.

## Operational resilience

For the calls that don't come back, borrow the standard playbook for any flaky network
dependency:

- **Retry with backoff.** Transient errors and 429s often clear on a second try; retry a
  few times with increasing delays rather than hammering the service.
- **Set timeouts.** A slow call shouldn't be allowed to hang your whole feature. Cap how
  long you'll wait and treat the overrun as a failure.
- **Use a circuit breaker.** If the service is clearly down, stop calling it for a while
  and serve the fallback — don't pile requests onto a dead dependency.
- **Keep a fallback model or provider.** A cheaper or alternate model can carry the load
  when your first choice is unavailable ([choosing a model for a feature](/learn/building-ai/choosing-a-model-for-a-feature/)).
- **Serve cached or default responses.** When all else fails, a stored or default answer
  keeps the feature alive.

Costs and limits shape all of this — see [cost, latency, and limits](/learn/building-ai/cost-latency-and-limits/).

## UX of uncertainty

How you *present* uncertainty is part of handling it. The interface should be honest that
an AI produced the answer and that it might be wrong.

- **Show sources.** Let people see what the answer was grounded in so they can judge it.
- **Express uncertainty.** When confidence is low, say so — a hedge is more trustworthy
  than false certainty.
- **Make bad answers easy to report.** A one-click "this was wrong" both helps the user in
  the moment and feeds your [evaluations](/learn/building-ai/evaluating-ai-features/) and
  [monitoring](/learn/building-ai/observability-and-monitoring/), turning real failures
  into the data that makes the feature better.

<div class="knowledge-check" data-quiz data-correct-msg="Right — for high-stakes, irreversible actions, ground and verify the output and require a human to confirm before acting. That's failing safe." markdown="0">
  <p class="knowledge-check__q">Quick check: a feature could take an expensive, irreversible action based on the model's output. What's the safest design?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Trust the output — the model is usually right</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Ground and verify the output, and require human confirmation before acting — fail safe</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Add a retry so the call never fails</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Plan for all three failure shapes: **confidently wrong**, **slow**, and **down**.
- **Content vs. operational failure** are different problems — wrong output that looks
  fine, versus a call that never returns — and you must handle both.
- **Reduce hallucination** by grounding in real data, giving the model an out, asking for
  citations, lowering temperature, and keeping tasks narrow.
- **Catch what's left** with structure and value validation, sanity checks, and guardrails
  — and never auto-act on unverified output.
- **Fail safe**: fall back and degrade gracefully, and put a human in the loop before
  high-stakes or irreversible actions.
- **Stay resilient** with retries and backoff, timeouts, circuit breakers, a fallback
  model, and cached defaults — and be honest about uncertainty in the UX.

Next up: [prompt injection & data safety](/learn/building-ai/prompt-injection-and-security/).
