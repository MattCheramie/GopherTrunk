---
slug: one-model-vs-many
title: One model, or a combination?
description: Should you stick with a single AI model and provider or combine several? Weigh simplicity against best-tool-per-task, learn how multi-model setups work — routers, per-task selection, fallback chains — and the real costs of juggling providers.
keywords: one model vs many, multi-model setup, model router, OpenRouter, fallback chain, per-task model selection, single provider, local model, embedding model, AI provider strategy, GopherTrunk
level: intermediate
prereq:
  - providing-context
  - the-provider-landscape
faq:
  - q: "Should a beginner use one AI model or several?"
    a: "Start with one. A single model and provider means one bill, one set of quirks to learn, and far less tooling to set up, which lets you focus on the actual work. Add a second model only when a *specific* need appears — a recurring task that's too expensive on the frontier model, or private code that shouldn't leave your machine. Most people arrive at a combination by accident, one justified addition at a time."
  - q: "How do multi-model setups actually work in practice?"
    a: "Three common patterns. A **router or aggregator** (like OpenRouter) gives you one account and API to reach many models, switching by changing a string. **Per-task selection** means you deliberately pick the model per job — a small fast one for trivial edits, a frontier one for hard problems. A **fallback chain** tries one model and automatically falls back to another if it's unavailable or refuses. They combine freely."
  - q: "Does my conversation carry over when I switch models?"
    a: "Not automatically across providers. Context lives in the request you send, so switching models mid-task means re-sending the relevant history to the new model — and each provider counts and prices tokens differently. This is one of the real costs of juggling: the model changes, but the context doesn't follow it for free."
---
# One model, or a combination?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**One model is simpler** — one bill, one set of quirks, consistent behaviour, less tooling. **A combination buys best-tool-per-task** — cheap-and-fast for trivial edits, a frontier model for hard problems, local for private code, a specialist for retrieval. **Start with one, add as needs appear** — routers, per-task selection, and fallback chains make a combination work, at the cost of more to manage.
</div>

This is lesson 22 of the path. We've covered how to prompt, how to give the model standing instructions, and how to feed it context. A question sits underneath all of that: *which* model is doing the work — one you commit to, or several you mix? The [provider landscape](/learn/ai-software-dev/the-provider-landscape/) lesson mapped who makes the models; this one is about how many of them *you* should actually use. By the end you'll understand the honest trade-off between simplicity and best-fit, how real multi-model setups are wired, and why the sensible path for almost everyone is "start with one."

## The case for one model and provider

Committing to a single model and provider has under-rated advantages, and for most people most of the time it's the right call.

- **Simplicity.** One account, one set of credentials, one tool configuration. Nothing to route, nothing to keep in sync.
- **One bill.** A single, predictable billing relationship instead of reconciling spend across several providers and tracking which one charged for what.
- **You learn its quirks.** Every model has a personality — how it likes prompts phrased, where it tends to over-engineer, what it's reliably good and bad at. Sticking with one lets you build that intuition, and it pays off in better prompts and fewer surprises.
- **Consistent behaviour.** The same model gives you the same style and conventions across the whole project, so your code stays coherent and your config file is tuned to one target.
- **Less tooling.** No router to operate, no fallback logic to maintain, no per-task selection to think about. The setup gets out of your way.

Familiarity compounds. A model you know well, prompted by someone who knows it well, often beats a "better" model used clumsily.

## The case for a combination

The argument for several models is simple: no single model is best at everything, and the cheapest-capable tool for each job isn't always the same one. Matching the model to the task can save money and improve results at once.

- **A cheap, fast small model for simple edits.** Renaming, boilerplate, a quick docstring — using a frontier model for these is like couriering a postcard. A small fast model does them in less time and at a fraction of the cost.
- **A frontier model for hard problems.** Novel design, a gnarly concurrency bug, an unfamiliar algorithm — this is where the strongest model earns its price.
- **A local or open-weight model for private code.** When code can't leave your network, a model you [run yourself](/learn/ai-software-dev/the-provider-landscape/) keeps everything on-premise, even if it trails the hosted frontier a little.
- **A specialised embedding model for retrieval.** The [RAG](/learn/ai-software-dev/providing-context/) pipeline doesn't use a chat model at all for its retrieval step — it uses a dedicated embedding model. So even a "one model" setup is often quietly two: one to generate, one to embed.

| Task | Reasonable choice | Why |
|---|---|---|
| Rename, boilerplate, docstrings | Small, fast model | Cheap and quick; the task is easy |
| Novel design, hard bug | Frontier model | Worth the cost when difficulty is high |
| Sensitive / private code | Local or open-weight model | Code never leaves your machine |
| Retrieval over a big corpus | Embedding model | Purpose-built for similarity search |

## How multi-model setups actually work

If you do combine models, three patterns do most of the work, and they layer freely.

**Routers and aggregators.** An aggregator such as **OpenRouter** sits in front of many providers and exposes them through one account and one API. You switch models by changing a single identifier in the request — no new credentials, one consolidated bill. This is the lowest-friction way to have many models on tap, at the cost of putting a middleman between you and the provider.

**Per-task selection.** You — or your tool — pick the model deliberately per job. Many coding tools now expose a model dropdown precisely so you can drop to a fast cheap model for routine edits and switch up to a frontier model when a problem gets hard. The selection can be manual or driven by simple rules ("use the cheap model unless the task touches more than N files").

**Fallback chains.** You order models by preference and the system tries them in turn: if the first is unavailable, rate-limited, or refuses, it automatically falls back to the next. This buys resilience — your workflow doesn't stall because one provider had a bad afternoon.

## The costs of juggling

A combination is not free, and the costs are easy to underestimate.

- **Complexity.** More accounts, keys, configuration, and moving parts — more that can break and more to keep updated.
- **Context doesn't transfer between providers.** As we saw with [context windows](/learn/ai-software-dev/context-windows/), the conversation lives in the request, not in the model. Switch providers mid-task and you must re-send the relevant history to the new model; it doesn't follow for free, and each provider tokenises and prices it differently.
- **More to manage.** Several bills to watch, several sets of [usage limits](/learn/ai-software-dev/usage-limits-and-tiers/) and quirks to track, and the cognitive load of deciding which model to use when. That decision overhead is itself a cost.

None of these is disqualifying, but together they explain why piling on models from day one usually backfires.

## The common path: start with one, add as needs appear

The pattern that works for most developers: **begin with a single model and provider**, learn it well, and ship. Add a second only when a *specific, recurring* need makes the case for itself — a routine task that's needlessly expensive on the frontier model, a body of private code that can't go to a hosted API, a retrieval feature that needs an embedding model. Each addition should solve a problem you actually have, not a hypothetical one.

For a project like GopherTrunk this falls out naturally. A frontier model handles the genuinely hard work — reasoning about an unfamiliar DSP algorithm or a subtle decoder bug. If you later find yourself spending real money on trivial edits, a small fast model for those is an easy, justified addition. And if some captures are sensitive enough that they can't leave the machine, a local model for that slice is worth the setup. You grow into a combination one good reason at a time; you don't start there.

How to make that first choice — and how to decide when a second model is genuinely justified — is exactly the decision framework we build in the next module.

<div class="knowledge-check" data-quiz data-correct-msg="Right — start with one model to keep things simple, and add others only when a specific recurring need makes the case." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the recommended path for most developers choosing how many models to use?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Wire up a router with several models on day one to be safe</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Start with one model and add others only when a specific recurring need appears</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Always use a different model for every single task from the start</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **One model is simpler** — one bill, one set of quirks to learn, consistent behaviour, and far less tooling to maintain.
- **A combination buys fit** — cheap-and-fast for trivial edits, a frontier model for hard problems, local for private code, a specialist embedding model for retrieval.
- **Three setup patterns** — routers/aggregators like OpenRouter, deliberate per-task selection, and fallback chains, layered freely.
- **Juggling has real costs** — more complexity to manage, and context doesn't transfer between providers, so switching means re-sending history.
- **Start with one** — learn it well, then add a second model only when a specific recurring need justifies it.
- **Even "one" is often two** — a RAG pipeline pairs a generation model with a dedicated embedding model.

Next up: turning all of this into a concrete decision — how to actually pick the model and provider for your situation — in [Choosing a model & provider](/learn/ai-software-dev/choosing-model-and-provider/).
