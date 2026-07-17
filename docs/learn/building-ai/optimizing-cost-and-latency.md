---
slug: optimizing-cost-and-latency
title: Optimizing cost & latency
description: Practical levers to make an AI feature cheaper and faster without wrecking quality — measuring first, trimming tokens, caching, right-sizing and routing models, parallelizing work, and streaming for perceived speed.
keywords: prompt caching, response cache, semantic caching, model routing, model cascade, batching, reduce tokens, latency optimization, cost optimization, streaming, right-size model, precompute embeddings
level: advanced
status: full
prereq:
  - cost-latency-and-limits
faq:
  - q: "What's the single cheapest big win when many requests are identical?"
    a: "Response caching. If the same request comes in repeatedly, serve the stored answer and skip the model call entirely — it costs nothing and returns instantly. For requests that only *share* a large prefix rather than being identical, provider prompt caching gives you the discount instead."
  - q: "Does streaming make a feature cheaper?"
    a: "No. Streaming sends output tokens as they are generated, so the user sees words appear almost immediately, but the total tokens billed are exactly the same. It reduces *perceived* latency, not cost. It is one of the cheapest UX wins available, but treat it as separate from the levers that actually cut spend."
  - q: "How do I optimize without quietly degrading quality?"
    a: "Re-run your evals after every change. Cost, quality, and latency form a triangle — pulling one usually tugs the others. An optimization that saves money but drops your eval score below the bar is a regression, not a win, so measure the score, not just the bill."
---

# Optimizing cost & latency

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Four families of levers make an AI feature cheap and fast without wrecking quality:
**caching** (skip repeated work), **right-size and route** (use the smallest model that
passes, escalate only when needed), **trim tokens** (send and generate less), and
**stream** (transform how fast it *feels*). Always **measure first** and **re-run your
evals after** — cost, quality, and latency are a triangle, and you can't optimize what
you haven't measured.
</div>

An AI feature that works in a demo can still be too slow or too expensive to ship. The
good news is that most features leave a lot on the table: a few targeted changes often
cut cost and latency by large factors without users noticing any drop in quality. This
lesson is the toolbox. It builds directly on
[Cost, latency & limits](/learn/building-ai/cost-latency-and-limits/), and pairs with the
coding-focused sibling [Understanding the cost](/learn/ai-software-dev/understanding-cost/).

## Measure first

Optimizing before you measure is guessing. The cost and latency of an AI feature are
almost never spread evenly — a handful of call sites usually dominate the bill, and one
slow step usually dominates the wait. Instrument the feature so you can see per-call
token counts, latency, and spend broken down by call site (this is the job of
[observability & monitoring](/learn/building-ai/observability-and-monitoring/)).

Then optimize the biggest items, in order. Halving a call that is 5% of your cost is
noise; halving the one that is 60% of it is the whole game. Let the numbers pick your
targets rather than your intuition.

## Trim what you send and generate

Every token is billed and every token takes time, so the cheapest token is the one you
never send. Look for tokens you can cut without hurting the result:

- **Shorter prompts.** Tighten verbose instructions and drop restated context. Long
  system prompts are re-sent on every call.
- **Leaner retrieval context.** If you use RAG, feed the model the few passages that
  matter, not everything the search returned — retrieval quality beats retrieval
  quantity (see [chunking & retrieval quality](/learn/building-ai/chunking-and-retrieval-quality/)).
- **Fewer, shorter examples.** Trim few-shot examples to the minimum that holds quality.
- **Cap max output tokens.** Output is the slow, pricey part — generated one token at a
  time at the higher rate — so ask for the smallest useful answer and set an explicit
  ceiling on output length.

## Caching

Caching is the highest-leverage cost lever because it lets you *skip work entirely*.
There are three flavours, and they solve different problems:

- **Response caching.** If a request is byte-for-byte identical to one you've served,
  return the stored answer and make no model call at all. Perfect for repeated identical
  queries — it costs nothing and is instant.
- **Provider prompt caching.** When many requests share a large, stable *prefix* — a long
  system prompt, a big reference document — the provider stores its processed form and
  bills the reused part at a steep discount. Keep the unchanging bulk up front and the
  varying part last so the cache keeps hitting.
- **Semantic caching.** For requests that are *near*-duplicates rather than identical,
  match on meaning (via embeddings) and reuse a previous answer when a new query is close
  enough. Powerful, but set the similarity threshold carefully — too loose and you serve
  a stale or wrong answer.

## Right-size and route

The model tier is often the single biggest cost lever. Don't default every call to the
flagship. Instead:

- **Right-size.** Use the smallest, cheapest model that *passes your evals* for the task.
  Many routine jobs — classification, extraction, short rewrites — are handled fine by a
  fast small model at a fraction of the price. Picking per feature is the subject of
  [choosing a model for a feature](/learn/building-ai/choosing-a-model-for-a-feature/).
- **Cascade / route.** Try a cheap model first, and *escalate* to a stronger one only
  when the cheap one's answer isn't good enough — detected by a confidence signal, a
  validation check, or a lightweight judge. Most traffic is handled cheaply; only the
  hard minority pays the flagship rate.

## Parallelize and precompute

Latency isn't only about making each call faster — it's about not waiting when you don't
have to.

- **Parallelize.** If a request needs several *independent* model calls, fire them
  concurrently rather than one after another. The wall-clock latency drops toward the
  slowest single call instead of their sum.
- **Precompute.** Work that doesn't depend on the live request can be done ahead of time.
  Compute embeddings for your document corpus offline, generate summaries in a batch job —
  so the live path only does the small, request-specific part.
- **Go async.** If the user needn't watch the result appear, push the work to a background
  job and notify them when it's done, instead of holding a request open.

## Stream for perceived speed

Streaming sends output tokens to the user as they are generated, so the first words
appear almost immediately instead of after the whole answer is complete. It does **not**
reduce cost or total generation time — the same tokens are billed — but it transforms the
*felt* latency, because a response that starts in 200 ms reads as fast even if it finishes
in five seconds. For any interactive feature it's one of the cheapest wins available; see
[streaming](/learn/building-ai/streaming/) for how it works.

## Don't trade away quality blindly

Every lever here has a failure mode: trim too much context and the answer gets worse,
route too aggressively and the cheap model flubs hard cases, cache too loosely and you
serve stale answers. So make optimization a measured loop, not a one-way cut. **Re-run
your [evals](/learn/building-ai/evaluating-ai-features/) after each change** and keep the
optimization only if the quality score stays above your bar. Cost, quality, and latency
form a triangle — pull one corner and the others move — and the only way to know you've
made a good trade is to watch all three.

<div class="knowledge-check" data-quiz data-correct-msg="Right — identical requests can skip the model entirely with response caching, and shared prefixes get the prompt-caching discount." markdown="0">
  <p class="knowledge-check__q">Quick check: many requests to your feature are identical. What's the cheapest big win?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Switch every call to the flagship model</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Cache responses (and use prompt caching for shared prefixes)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Turn on streaming to cut the token bill</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Measure first** — instrument cost and latency per call site and optimize the biggest
  items, not your guesses.
- **Trim tokens** — shorter prompts, leaner RAG context, fewer examples, and a cap on
  output, which is the slow and pricey part.
- **Cache** — response caching for identical requests, prompt caching for shared prefixes,
  semantic caching for near-duplicates.
- **Right-size and route** — use the smallest model that passes your evals, and cascade to
  a stronger one only when needed.
- **Parallelize, precompute, and stream** — run independent calls concurrently, do work
  offline where you can, and stream to cut *perceived* latency.
- **Re-run evals after every change** — it's a cost/quality/latency triangle, so verify
  you haven't traded quality away.

Next up: [privacy, data & compliance](/learn/building-ai/privacy-data-and-compliance/).
