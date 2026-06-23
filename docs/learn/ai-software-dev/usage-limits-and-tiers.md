---
slug: usage-limits-and-tiers
title: Usage limits & tiers
description: How AI usage limits really work — rate limits, daily and monthly caps, consumer message caps, and usage tiers that grow with your account — why they exist, how subscription and API metering differ, and practical strategies when you hit a limit.
keywords: AI usage limits, rate limits, tokens per minute, requests per minute, message caps, usage tiers, subscription metering, API metering, free tier limits, context window limit, exponential backoff, handling rate limits
level: intermediate
status: full
prereq:
  - interfaces-and-access
faq:
  - q: "Why do AI providers impose rate limits at all?"
    a: "Because model serving runs on shared, finite, expensive hardware. **Rate limits** keep one user from starving everyone else, enforce fairness across customers, and blunt abuse like scraping or runaway scripts. They're a capacity-and-fairness mechanism, not a punishment — and most raise automatically as your account matures."
  - q: "What's the difference between a rate limit and a usage cap?"
    a: "A **rate limit** controls how *fast* you can go — requests or tokens per minute — and resets continuously. A **cap** controls how *much* you can do in total — messages per day, or spend per month — and resets on a calendar boundary. You can be well under your monthly cap and still get throttled by a per-minute rate limit, and vice versa."
  - q: "What should I do when I keep hitting rate limits?"
    a: "First, **back off and retry** — wait, ideally with exponential backoff, since limits reset quickly. Beyond that: **batch** small requests together, route routine work to a **smaller/cheaper model** with looser limits, **spread load** over time instead of bursting, and trim oversized context. If you've genuinely outgrown your tier, upgrading or moving up a usage tier raises the ceiling."
---
# Usage limits & tiers

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Two kinds of ceiling** — rate limits cap how *fast* you go; caps limit how *much* you do per day or month. **Tiers grow with your account** — limits rise as your account ages and spends. **Hit a wall? Back off, batch, downsize the model, and spread load** — most limits reset quickly.
</div>

This is lesson 9 of the path. Whichever [interface](/learn/ai-software-dev/interfaces-and-access/) you use, the provider meters your usage — and sooner or later you'll bump into a limit mid-task and wonder what just happened. This lesson demystifies that. By the end you'll understand the kinds of limits that exist, why they're there, how subscription metering differs from API metering, why the context window is itself a per-request limit, what free tiers constrain, and the practical moves that get you unstuck. As ever in this module, the *mechanisms* here are durable; any specific number you see quoted anywhere is a snapshot — check the provider's live limits page for current figures.

## How limits actually work

Limits come in two flavours, and confusing them is the usual source of frustration.

**Rate limits** cap how *fast* you can send work, and reset continuously (per minute or per second). The two you'll meet most:

- **Requests per minute (RPM)** — how many separate calls you may make in a minute.
- **Tokens per minute (TPM)** — how many tokens (the chunks of text models read and write — see [how models decide](/learn/ai-software-dev/how-models-decide/)) may flow in a minute, counting input *and* output.

You can blow the TPM limit with a single huge request even while well under your RPM, because a long document is a lot of tokens at once.

**Caps** limit how *much* you can do over a longer window and reset on a calendar boundary:

- **Daily / monthly caps** — a ceiling on total usage or total spend in a period.
- **Message caps on consumer subscriptions** — many chat-app plans let you send only so many messages in a rolling window, sometimes more for the cheap model than the frontier one.

**Usage tiers** then sit on top: providers raise your limits automatically as your account matures — as it ages, builds a payment history, or spends more over time. A brand-new account starts conservative; a long-standing, paying one earns far more headroom. Moving up a tier is the main lever for a higher ceiling once you've outgrown the starting limits.

| Limit type | Controls | Resets | Where you meet it |
|---|---|---|---|
| **Requests per minute (RPM)** | How many calls per minute | Continuously | API, agents making many calls |
| **Tokens per minute (TPM)** | How much text per minute (in + out) | Continuously | API, large documents or contexts |
| **Daily / monthly cap** | Total usage or spend per period | Calendar boundary | API budgets, plan allowances |
| **Message cap** | Messages per window on a subscription | Rolling window | Consumer chat apps |
| **Usage tier** | Raises the ceilings above | As account ages/spends | All of the above |

## Why limits exist

Limits aren't arbitrary gatekeeping. Three honest reasons:

- **Shared, finite capacity.** Models run on expensive GPUs serving everyone at once. Limits ration that shared resource so it doesn't collapse under load.
- **Fairness.** Without ceilings, a few heavy users could starve everyone else. Limits spread capacity across the customer base.
- **Abuse prevention.** Caps blunt scraping, spam, and runaway scripts — including the accidental kind, where a buggy loop would otherwise rack up a fortune before you noticed.

Read this way, a limit is a guardrail as much as a gate: it protects the service *and* protects you from your own runaway agent.

## Subscription metering vs API metering

The two billing models meter you completely differently, and each wins in different situations.

**Subscription metering** — a **flat monthly fee** for access, usually through a consumer app or plan. You pay the same whether you use it lightly or heavily, up to **soft or hard caps** (a soft cap slows or nudges you; a hard cap stops you until reset). The appeal is **predictability and simplicity**: one known bill, no per-request math. The limits are message- or usage-based rather than per-token. This suits steady, interactive, hard-to-predict daily use.

**API metering** — **pay per token**, billed by exactly how much you send and receive (the subject of the next lesson, [Understanding the cost](/learn/ai-software-dev/understanding-cost/)). The appeal is that it **scales precisely** — you pay for what you use and nothing more, and you can drive enormous volume if you're willing to pay for it. The catch is that it **needs monitoring**: an agent in a loop can quietly run up real money, so you'll want budgets, alerts and rate-limit handling. This suits automation, products, and bursty or high-volume workloads.

Neither is "cheaper" in general — it depends on your pattern. Light, steady interactive use often favours a flat subscription; spiky or automated workloads often favour pay-per-token. We weigh the economics in detail next lesson.

## The context window is a per-request limit

One limit lives *inside* each request rather than across time: the **context window**, the maximum amount of text (your prompt plus the conversation history plus any files you've attached, all measured in tokens) a model can consider at once. Covered in depth in [Context windows](/learn/ai-software-dev/context-windows/), it's worth flagging here because it behaves like a hard cap per call. Overflow it and the model can't see the excess — older messages get dropped or the request is rejected. So a giant file or a very long chat isn't just slower and costlier; past a point it simply won't fit. Managing what goes into that window is a skill we return to in [Providing context](/learn/ai-software-dev/providing-context/).

## Free tiers and their constraints

Most providers offer a **free tier** so you can try a model before paying. It's genuinely useful for learning, but expect tight constraints: low rate limits, small daily caps, sometimes only the smaller models, and often weaker (or different) data-handling terms — some free tiers may use your inputs to improve models, which matters for private code (see [Security, privacy & ethics](/learn/ai-software-dev/security-privacy-ethics/)). A free tier is a test drive, not a workshop. Read its terms before you feed it anything sensitive.

## Strategies when you hit a limit

Hitting a limit is routine, not a failure. The standard moves:

- **Back off and retry.** Rate limits reset fast, so wait and try again — ideally with **exponential backoff**: wait a short moment, then double the wait on each retry. This avoids hammering the service the instant it's ready and is the single most useful reflex. Many tools and SDKs do it for you.
- **Batch.** Combine several small requests into one where you can, so you make fewer calls under the RPM limit (mind the TPM limit, since the combined request is bigger).
- **Use a smaller / cheaper model.** Route routine work to the fast small tier, which usually has looser limits and frees frontier capacity for the hard problems — exactly the difficulty-routing from [Coding models compared](/learn/ai-software-dev/coding-models-compared/).
- **Spread the load.** Pace work over time instead of firing it all in one burst, smoothing your usage under per-minute limits.
- **Trim context.** Send only the code that matters rather than the whole repository, easing both token limits and cost.

If you've tried these and still hit walls constantly, you've probably outgrown your tier — upgrading the plan or moving up a usage tier raises the ceiling.

Here's exponential backoff as pseudocode, the pattern worth internalising:

```text
wait = 1 second
repeat:
    response = call_model(request)
    if response is OK:           return response
    if response is "rate limited":
        sleep(wait)
        wait = wait * 2          # 1s, 2s, 4s, 8s, ...
    else:
        raise the error          # a real failure, not throttling
```

<div class="knowledge-check" data-quiz data-correct-msg="Right — exponential backoff waits and doubles the delay on each retry, letting a fast-resetting rate limit clear without hammering the service." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the first thing to try when you hit a rate limit?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Immediately resend the same request as fast as possible</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Back off and retry with an increasing wait (exponential backoff)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Switch to a larger, more capable model</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Rate limits vs caps** — rate limits (RPM, TPM) cap how fast you go and reset continuously; caps limit how much you do per day or month and reset on a calendar boundary.
- **Tiers grow with you** — providers raise limits automatically as your account ages and spends, so a mature account has far more headroom.
- **Why limits exist** — shared finite capacity, fairness across users, and abuse prevention, including protecting you from a runaway loop.
- **Two metering models** — flat-fee subscriptions are simple and predictable; pay-per-token API scales precisely but needs monitoring.
- **Context is a per-request limit** — the window caps how much text one call can consider, behaving like a hard per-call ceiling.
- **When you hit a wall** — back off with exponential retry, batch, drop to a cheaper model, spread load, and trim context before upgrading.

Next up: where the money actually goes — input and output token pricing, what drives a bill up, and how to estimate spend — in [Understanding the cost](/learn/ai-software-dev/understanding-cost/).
