---
slug: shipping-your-first-ai-feature
title: Shipping your first AI feature
description: The capstone — thread the whole path together into one real, launchable AI feature. Scope it narrowly, walk it through evals, prompting, RAG, guardrails, and monitoring, and ship it behind a feature flag with a human in the loop.
keywords: ship AI feature, worked example, launch checklist, MVP, feature flag, human in the loop, RAG assistant, first AI feature, guardrails, rollback, iterate on evals
level: intermediate
status: full
prereq:
  - building-a-rag-pipeline
  - evaluating-ai-features
faq:
  - q: "What's the best first AI feature to ship?"
    a: "One narrow, well-scoped feature with an eval set and a human in the loop — not a broad, autonomous one. A tightly scoped assistant that suggests and lets the user confirm is easy to measure, cheap to run, and safe to roll back. Ambition can come later, once evals prove the feature earns it."
  - q: "Do I have to build all of this before I ship?"
    a: "No. You ship the smallest version that clears your launch checklist — evals passing, guardrails on, cost bounded, privacy reviewed, monitoring on, and a feature flag with a rollback. The full pipeline is the map of what a mature feature grows into, not a gate you must clear on day one."
  - q: "Why ship behind a feature flag?"
    a: "A flag lets you release to a small slice of users, watch real behaviour against your monitoring, and turn the feature off instantly if something goes wrong — no redeploy. It turns a launch from an irreversible event into a dial you control, which is exactly what a probabilistic feature needs."
  - q: "What do I do after launch?"
    a: "Watch the feature in production and feed what you learn back in. Every real failure and every piece of user feedback becomes a new eval case, so the same mistake can't return unnoticed. You improve steadily, widening scope only as the evals justify it."
gophertrunk_links:
  - title: Architecture
    url: /architecture.html
    note: the components a GopherTrunk assistant would sit beside and draw its reference material from.
  - title: Web UI
    url: /web.html
    note: where a Field Guide assistant would live for the user — the surface you'd put behind a flag.
---

# Shipping your first AI feature

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
This is the capstone: put the whole path together to build and launch **one real AI
feature** — small, measured, and safe. Three rules carry it. Keep the **scope narrow** —
one well-defined job, not a broad autonomous helper. Wire in **evals and a
human-in-the-loop** so quality is measured and nothing high-stakes acts on the model's
word alone. And **ship behind a flag** with a rollback, so launch is a dial you control,
not a leap. Everything before this lesson was a piece; here they become a product. Start
from [building AI into software](/learn/building-ai/building-vs-using-ai/).
</div>

You've met every part on its own — the model as a component, prompts, retrieval, evals,
guardrails, cost, monitoring. This lesson threads them into one worked example so you can
see the whole thing move together, and gives you a checklist to actually launch. The goal
isn't a grand system. It's one narrow feature that works, that you can measure, and that
you can turn off in a hurry.

## Pick a narrow feature

The first decision is the most important, and it's about *restraint*. Pick a single,
well-bounded job the model can do well, not a sprawling assistant that promises
everything.

For a GopherTrunk-like app, two good candidates:

- **A "Field Guide assistant"** that answers questions about radio systems from the
  site's reference articles and the user's own saved systems — a
  [RAG](/learn/building-ai/building-a-rag-pipeline/) feature grounded in a corpus you
  control.
- **An "explain this control-channel activity in plain English"** helper that takes a
  slice of decoded activity and describes what's happening for a newcomer.

Pick **one** and scope it tightly. "Answer questions about the systems in the reference
library and the user's saved list, with citations, and say *I don't know* when the answer
isn't there" is a shippable feature. "An AI that knows everything about radio" is not. A
narrow feature is one you can evaluate, ground, and reason about — the width comes later.

## Walk the whole pipeline

With the feature chosen, thread the path's lessons in order. Each is a link in the chain;
the feature is what happens when you connect them.

1. **Define success and build an eval set.** Before writing a prompt, decide what a good
   answer looks like and collect representative inputs to measure it —
   [evaluating AI features](/learn/building-ai/evaluating-ai-features/). This is your
   safety net for every change that follows.
2. **Choose a model.** Match capability, cost, and latency to the job rather than reaching
   for the biggest option — [choosing a model for a feature](/learn/building-ai/choosing-a-model-for-a-feature/).
3. **Write the prompt.** Give clear [system instructions](/learn/building-ai/prompts-as-code/)
   and keep the user's input and your retrieved facts as clearly
   [labeled context](/learn/building-ai/context-and-user-input/), never blurred together.
4. **Ground it, and structure the output.** Feed the model real passages with
   [RAG](/learn/building-ai/building-a-rag-pipeline/) and ask for
   [structured output](/learn/building-ai/structured-output/) where a downstream step needs
   to parse the answer or its citations.
5. **Handle failure and hallucination.** Assume some answers will be confidently wrong and
   some calls won't return; reduce, catch, and fail safe, keeping a human in the loop for
   anything high-stakes — [handling hallucinations & failure](/learn/building-ai/handling-hallucinations-and-failure/).
6. **Secure it, and mind privacy.** Treat retrieved and user text as untrusted against
   [prompt injection](/learn/building-ai/prompt-injection-and-security/), and be deliberate
   about what data leaves your system — [privacy, data & compliance](/learn/building-ai/privacy-data-and-compliance/).
7. **Control cost and latency, and stream.** Keep spend and response time bounded, and
   [stream](/learn/building-ai/streaming/) the answer so it feels fast —
   [optimizing cost & latency](/learn/building-ai/optimizing-cost-and-latency/).
8. **Add observability.** Log calls, cost, latency, and quality signals so you can see the
   feature's behaviour in production — [observability & monitoring](/learn/building-ai/observability-and-monitoring/).

You don't build all eight to their fullest before launch. You build the smallest version
of each that clears the checklist below, and let the rest grow with the feature.

## Start small and human-in-the-loop

Ship the feature **assistive**, not autonomous. It *suggests* — an answer, an explanation,
a draft — and the user confirms, edits, or ignores it. A Field Guide assistant that
proposes a grounded, cited answer the user reads and judges is worlds safer than one that
silently changes their saved systems.

This isn't a limitation, it's the design. A human-in-the-loop feature can be wrong without
being harmful, because a person is between the model and any consequence. You expand its
autonomy only as your [evals](/learn/building-ai/evaluating-ai-features/) prove it has
earned more — never on optimism.

## A launch checklist

Before you flip the feature on, run down the list. Everything on it comes from an earlier
lesson; the checklist just makes sure nothing was skipped.

- **Evals pass.** Your offline test set meets its criteria, and you've measured this
  version — not the last one.
- **Guardrails on.** Output is validated, unverified answers can't trigger high-stakes
  actions, and a human confirms anything irreversible.
- **Cost bounded.** Spend and token use are capped, with a cheaper fallback path.
- **Privacy reviewed.** You know what data the feature sends where, and you're sending only
  what's needed.
- **Monitoring and alerting on.** Cost, latency, error rate, and a quality signal are
  logged, with an alert if they go bad.
- **Ship behind a feature flag with a rollback.** Release to a small slice first, and keep
  a switch that turns the feature off instantly if it misbehaves.

If any line isn't true yet, that's your next task — not a reason to skip it.

## Iterate

Launch is the start of the loop, not the end. Once the feature is live, the most valuable
data you have is how it actually behaves.

Every production failure and every piece of user feedback becomes a **new eval case**, the
same way a fixed bug becomes a regression test. Your test set grows from real usage, your
prompts and retrieval improve against it, and the same mistake can't quietly come back. A
narrow feature that improves steadily beats a broad one that ships once and rots. Widen
the scope only when the evals say the feature can carry it.

## Where to go next

You've shipped one real feature — the rest is craft that compounds. Deepen the wider
discipline of AI engineering: better retrieval, tighter evals, sharper prompts, and the
operational maturity to run features at scale. Keep the [glossary](/learn/building-ai/glossary/)
handy as you go — the vocabulary is the shared map. And remember there's a legal side to
shipping software: licensing of your dependencies, your model's terms, and what you owe
your users. The [Software Licensing](/learn/software-licensing/) path covers that ground.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a narrow, well-scoped feature with evals and a human in the loop is measurable, safe, and easy to roll back. Breadth and autonomy are earned later." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the best first AI feature to ship?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A broad, autonomous assistant that handles as many tasks as possible</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">One narrow, well-scoped feature with evals and a human in the loop — not a broad, autonomous one</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Whatever ships fastest, evals and monitoring can come after launch</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The capstone move is to **thread the whole path into one real feature** — small,
  measured, and safe — rather than build a grand system.
- **Pick a narrow, well-scoped job** (a Field Guide RAG assistant, a plain-English
  explainer) you can evaluate and ground.
- **Walk the pipeline in order**: success and evals, model, prompt and labeled context,
  RAG and structured output, failure handling, security and privacy, cost and streaming,
  observability.
- **Start assistive and human-in-the-loop** — it suggests, the user confirms — and expand
  autonomy only as evals justify it.
- **Run the launch checklist**: evals pass, guardrails on, cost bounded, privacy reviewed,
  monitoring on, shipped behind a flag with a rollback.
- **Iterate** — production failures and feedback become new eval cases, so the feature
  improves steadily instead of drifting.

Next up: keep the [glossary](/learn/building-ai/glossary/) close, and when you're ready to release, the [Software Licensing](/learn/software-licensing/) path covers the legal side of shipping.
