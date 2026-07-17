---
slug: observability-and-monitoring
title: Observability & monitoring
description: Why you can't fix what you can't see — logging every model call, tracing multi-step agent and RAG flows, tracking cost and latency metrics, running online evals, and catching quality drift in production.
keywords: LLM observability, monitoring, logging prompts, tracing, cost metrics, latency, quality drift, online evals, user feedback, alerting, production AI
level: advanced
status: full
prereq:
  - evaluating-ai-features
faq:
  - q: "What is LLM observability?"
    a: "It's the practice of instrumenting an AI system so you can see what it actually did in production — the inputs, outputs, model version, token counts, latency, cost, and errors of every model call. Traditional apps are deterministic and easy to reason about; a model's behaviour shifts with the input, the prompt, and even the provider's version, so you can't infer what happened from the code alone. Observability makes that invisible behaviour visible."
  - q: "How is monitoring different from pre-launch evaluation?"
    a: "Pre-launch evals test a fixed set of cases before you ship, giving you a snapshot of quality on that day. Monitoring watches the live system continuously as real inputs and real users flow through it. Both matter — evals gate the release, monitoring catches what the evals couldn't foresee, including drift and inputs you never thought to test."
  - q: "What is quality drift?"
    a: "Quality drift is a gradual decline in how well a shipped feature performs, even though nothing in your code changed. It happens because the world moves — user inputs shift over time, and the provider may update the model underneath you. A feature that scored well at launch can quietly degrade, which is why online evals and user feedback matter long after release."
---
# Observability & monitoring

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Log** every model call, **trace** multi-step flows, watch **metrics**, and hunt
for **quality drift**. You can't fix what you can't see, so instrument every model
call — inputs, outputs, tokens, latency, cost, and errors. A model's behaviour isn't
visible in your code the way ordinary logic is; the only way to know what it did is to
record it. This is where [evaluating AI features](/learn/building-ai/evaluating-ai-features/)
continues once the feature is live.
</div>

Ordinary software is deterministic: read the code and you can reason about what it
does. An AI feature isn't like that. The same code produces different outputs depending
on the input, the prompt, and the exact model version the provider is serving that day.
You cannot infer what happened from the source — you have to record what actually went
through the system. Observability is how you turn an opaque box into something you can
debug, measure, and improve.

## Log the right things

The foundation is logging every model call with enough detail to reconstruct it later.
For each call, capture:

- **The prompt and inputs** — the full text or structured input sent to the model.
- **The output** — exactly what came back, before any post-processing.
- **The model and version** — which model answered, including the version string.
- **Token counts** — input and output tokens, the basis of both cost and latency.
- **Latency** — how long the call took, end to end.
- **Cost** — derived from token counts and the model's pricing.
- **Errors** — timeouts, refusals, rate limits, and malformed output.

Log all of this **while respecting privacy**. Prompts and outputs routinely contain
personal information, so redact or hash PII before it lands in your logs, and keep
retention short. What you record about your users is a compliance question as much as a
technical one — see [privacy, data & compliance](/learn/building-ai/privacy-data-and-compliance/).

## Trace multi-step flows

A single log line is enough for a single call. But agents and
[RAG](/learn/building-ai/building-a-rag-pipeline/) systems aren't single calls — they're
chains of them. An [agent](/learn/building-ai/what-is-an-ai-agent/) might reason, call a
tool, read the result, reason again, and call another. A RAG pipeline retrieves context,
then feeds it to the model.

**Tracing** ties those steps together under one request so you can see the whole flow.
Capture each step, every tool call and its result, and the retrieved context that was
actually put in front of the model. When an answer comes out wrong, the trace tells you
*why* — the model reasoned fine but a tool returned bad data, or the retriever pulled the
wrong document. Without the trace you only see the bad answer, not the step that caused
it.

## Metrics & dashboards

Logs and traces explain individual requests; **metrics** show you the system in
aggregate. Track:

- **Latency** — typical and worst-case response time.
- **Cost** — spend per request and total, so a runaway bill can't surprise you.
- **Throughput** — requests per second the system is handling.
- **Error and refusal rates** — how often calls fail or the model declines.
- **Token usage** — the driver behind both cost and latency.

Put these on a dashboard and watch the **trends**, not just the current number. Set
**budgets and alerts** so a spike — cost doubling overnight, latency creeping up — reaches
you before your users feel it.

## Quality in production

Pre-launch evals give you a snapshot, and a snapshot isn't enough. Once the feature is
live, real users send inputs you never imagined, and the model underneath you may change.
Keep measuring quality after launch:

- **Online evals** — run automated quality checks against real production traffic, not
  just your fixed test set.
- **User feedback** — a simple thumbs up/down on each answer is a cheap, honest signal.
- **Human review** — sample a slice of outputs and read them; nothing replaces a person
  spotting a subtly wrong answer.
- **Quality drift** — watch for slow decline as inputs shift and the provider updates the
  model, even when your own code hasn't changed.

This is [evaluation](/learn/building-ai/evaluating-ai-features/) that never stops. The
feature that scored well the day you shipped it can quietly get worse, and only ongoing
measurement catches it.

## Alerting

Metrics you never look at won't save you. Wire up **alerts** that fire on the things that
signal trouble: a **cost spike**, a **latency spike**, a jump in **error rates**, or a rash
of **guardrail hits** where the model tries to do something it shouldn't. A good alert
catches a bad deploy or a silent provider-side change within minutes instead of days —
long before it becomes a support flood or a budget overrun.

## Close the loop

Observability isn't just defence. Every production failure is raw material. A wrong answer
you spot in the logs becomes a **new eval case**; a class of failures becomes a **prompt
fix** or a guardrail. Feed what you see in production back into your test set and your
prompts, and the system gets steadily better. Observability is the sensor that drives the
whole improvement cycle — without it you're flying blind, and with it every mistake teaches
you something.

<div class="knowledge-check" data-quiz data-correct-msg="Right — pre-launch tests are a snapshot; only ongoing monitoring catches slow drift in the live system." markdown="0">
  <p class="knowledge-check__q">Quick check: a shipped feature's answer quality slowly degrades over weeks, though nothing in your code changed. What catches it?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A more thorough round of one-time pre-launch testing</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Ongoing monitoring — online evals and user feedback, not one-time pre-launch testing</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Lowering the model's temperature setting</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **You can't fix what you can't see** — a model's behaviour isn't visible in your code, so instrument every call.
- **Log the right things** — prompt, output, model and version, tokens, latency, cost, and errors, with PII redacted.
- **Trace multi-step flows** — capture each step, tool call, and retrieved context so you can debug *why* an answer went wrong.
- **Watch metrics** — latency, cost, throughput, error/refusal rates, and token usage; alert on spikes and set budgets.
- **Quality doesn't stop at launch** — run online evals, collect user feedback, sample for human review, and watch for drift.
- **Close the loop** — turn production failures into new eval cases and prompt fixes.

Next up: [shipping your first AI feature](/learn/building-ai/shipping-your-first-ai-feature/).
