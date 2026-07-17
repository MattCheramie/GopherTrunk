---
slug: cost-latency-and-limits
title: Cost, latency & rate limits by design
description: Every model call costs money, takes real time, and is throttled by the provider. Learn to treat token cost, latency, and rate limits as first-class design constraints — with practical levers to stay inside all three.
keywords: token cost, latency, rate limits, HTTP 429, tokens per minute, throughput, backoff, batching, budget, streaming, max output tokens, model choice
level: intermediate
status: full
prereq:
  - how-a-model-api-works
faq:
  - q: Why does a bigger prompt cost more even when the answer is short?
    a: Because you're billed for **input tokens** as well as output. Every character you send — the system prompt, the conversation history, retrieved context, few-shot examples — is read on every call and priced. A short answer keeps the output term small, but the input term still grows with the prompt, so trimming context is one of the cheapest ways to save money.
  - q: "What is HTTP 429 and what do I do about it?"
    a: "A 429 is the provider telling you you've exceeded a rate limit — too many requests or too many tokens in the current window. The first fix is retry-with-backoff: wait a moment and try again, increasing the wait each time. If you hit it constantly, spread the load out, batch less aggressively, or raise your tier for a higher cap."
  - q: What makes one call slower than another?
    a: Output length dominates, because the model generates one token at a time — a long answer simply takes more steps than a short one. Larger and reasoning-heavy models are slower per token too, and the input has to be read before generation starts. If a feature feels sluggish, cap the output and consider a smaller model before anything else.
  - q: "Do cost and latency pull in the same direction?"
    a: "Often, yes. Long outputs and large models cost more and take longer, so many levers help both at once: capping output tokens, trimming context, and choosing the smallest model that works all cut the bill and the wait together. Streaming is the exception — it doesn't reduce latency, it just hides it by showing text as it arrives."
---

# Cost, latency & rate limits by design

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Every model call spends three things: **tokens** (money), **latency** (time),
and a slice of your **rate limits** (throughput). None of them is an
afterthought — they are design constraints you plan for from day one. Bigger
prompts and longer outputs cost more and run slower; at scale the provider
throttles you with HTTP 429. Build with these limits in mind and your feature
stays cheap, responsive, and reliable. Start from
[how a model API call works](/learn/building-ai/how-a-model-api-works/), and for
the pricing model in depth see the sibling
[understanding the cost](/learn/ai-software-dev/understanding-cost/).
</div>

A model call looks like an ordinary function call in your code, but it quietly
draws down three budgets at once: a money budget, a time budget, and a
throughput budget the provider enforces. Ignore them and your prototype works
beautifully — right up until real traffic makes it expensive, sluggish, and
throttled all at once. The good news is that the same handful of design choices
keep all three in check, and it's far easier to build them in early than to
retrofit them under load.

## Cost: you pay per token

Providers bill by the **token** — the small chunk of text a model reads and
writes — and they price **input** and **output** tokens separately. Everything
you send is input: the system prompt, the user's message, the conversation
history, any retrieved context or few-shot examples. Everything the model
generates back is output, and output usually costs more per token, because
generating is the expensive part.

Two things push the bill up. A **bigger prompt** — long context, a RAG
pipeline that stuffs in retrieved passages, a pile of examples — means more
input tokens *on every call*. A **longer output** means more of the pricier
output tokens. So a rough cost estimate is just tokens times price:

```text
cost ≈ (input_tokens  × input_rate)
     + (output_tokens × output_rate)      (output_rate is usually higher)
```

That formula is the whole game — the sibling
[understanding the cost](/learn/ai-software-dev/understanding-cost/) lesson works
through real drivers and prompt caching. What matters here is the shape: to
spend less, send less and ask for less.

## Latency: calls take real time

A model call is not instant. It travels to a remote server and comes back, and
the round trip runs from a few hundred milliseconds to many seconds. Three
things make it slower: **larger** models take more time per token, **reasoning**
models that "think" before answering add more still, and — most of all —
**output length dominates**, because the model produces the answer one token at
a time. A 1,000-token answer is roughly ten times the generation work of a
100-token one.

This is a user-experience problem as much as an engineering one. A feature that
freezes for eight seconds feels broken even when it's working perfectly. You
design around the wait — the biggest tool for that is
[streaming](/learn/building-ai/streaming/), which shows text as it's generated so
the user sees progress immediately instead of staring at a spinner.

## Rate limits

Providers don't let you send unlimited traffic. They cap you on two axes at
once — **requests per minute** and **tokens per minute** — and the ceilings rise
as you move up their usage **tiers**. For a single developer testing by hand you
may never notice. At scale, with real traffic or a batch job, you *will* hit
them.

When you do, the call fails with **HTTP 429** ("Too Many Requests"). It isn't an
error in your code — it's the provider saying "slow down." The sibling
[usage limits & tiers](/learn/ai-software-dev/usage-limits-and-tiers/) lesson
covers how tiers and quotas work; the point for design is that 429 is a normal,
expected condition you must handle, not an edge case you can hope to avoid.

## Designing within the budget

The same levers tend to help cost, latency, and rate limits together. Reach for
these in roughly this order:

- **Cap max output tokens.** Set a ceiling on the response length. It bounds the
  most expensive, slowest part of the call and stops a runaway answer.
- **Trim the context.** Send only what the task needs. Every token you cut is
  input you don't pay for, on every call, and less to read before generation
  starts.
- **Pick the smallest model that works.** A fast, small model is cheaper *and*
  quicker; reserve the big one for genuinely hard work. This is important enough
  to earn its own lessons —
  [choosing a model for a feature](/learn/building-ai/choosing-a-model-for-a-feature/)
  and [optimizing cost and latency](/learn/building-ai/optimizing-cost-and-latency/).
- **Cache repeated work.** If you send the same large, stable context over and
  over, caching lets you avoid reprocessing it every time.
- **Batch and spread load.** Group work where the provider supports it, and
  space requests out so you stay under the per-minute caps instead of slamming
  into them.
- **Stream for perceived speed.** Streaming doesn't make the call faster, but
  showing tokens as they arrive makes the wait feel far shorter.
- **Retry with backoff.** On a 429 or a transient 5xx, wait and try again,
  lengthening the wait each attempt. This is the single most important piece of
  reliability code around a model call.

## A worked estimate

You can size a request before you ship it — no real prices needed, just the
token math. Walk through one call and add up the input:

```text
ONE REQUEST — token budget (illustrative)
  system prompt            ~   400 tokens   (fixed instructions)
  user input               ~   200 tokens   (the question)
  retrieved context (RAG)  ~ 3,000 tokens   (passages you looked up)
  ------------------------------------------
  input total              ~ 3,600 tokens   (billed every call, read before output)

  output (the answer)      ~   600 tokens   (generated one at a time — priced higher)
```

Reading the budget tells you where to cut. The retrieved context is 80% of the
input — so a tighter retrieval step (fewer, better passages) saves more than
anything else, and cuts latency too. The output is small here but is the pricier
rate and the slowest part, so a max-output cap protects the worst case. And that
whole ~4,200-token call counts against your tokens-per-minute limit, so a
hundred of them a minute is where 429s begin. The same three constraints, all
visible in one estimate.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a 429 is a rate limit. The first fix is retry-with-backoff, and at scale you spread load or raise your tier." markdown="0">
  <p class="knowledge-check__q">Quick check: at scale your feature starts returning HTTP 429. What is it, and what's the first fix?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A bug in your request — fix the JSON and it goes away</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A rate limit — add retry-with-backoff and/or spread load / raise your tier</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The model hallucinating — lower the temperature</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Every call spends three budgets at once: **tokens** (cost), **latency**
  (time), and your **rate limits** (throughput) — design for all three early.
- **Cost is per token**, input and output priced separately; bigger prompts and
  longer outputs both push it up, and output usually costs more.
- **Latency is real** — output length dominates because tokens are generated one
  at a time; larger and reasoning models are slower.
- **Rate limits** cap requests and tokens per minute by tier; at scale you'll hit
  them and get **HTTP 429**.
- **Levers** that help all three: cap output, trim context, pick the smallest
  model that works, cache, batch, stream, and retry with backoff.

Next up: Module 2 starts turning prompts into code — [prompts as code](/learn/building-ai/prompts-as-code/).
