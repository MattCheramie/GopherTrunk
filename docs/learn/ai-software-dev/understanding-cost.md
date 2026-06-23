---
slug: understanding-cost
title: Understanding the cost
description: How token-based AI pricing works — input and output priced separately, what drives a bill up, prompt caching, subscription versus API economics, the hardware cost of local models, and a clearly-fictional worked estimate.
keywords: AI cost, token pricing, input tokens output tokens, cost per million tokens, prompt caching, model tier cost, agent cost, subscription vs API pricing, local model cost, estimating AI spend, LLM pricing
level: intermediate
status: full
prereq:
  - usage-limits-and-tiers
faq:
  - q: "Why is output usually more expensive than input?"
    a: "Generating tokens is the costly part. **Input** tokens are read in a single pass to set up the model's state, but each **output** token is produced one at a time, each requiring a full run through the model. Because generation is sequential and compute-heavy, providers typically price output at a higher rate per token than input — often several times higher."
  - q: "What is prompt caching and when does it save money?"
    a: "**Prompt caching** lets the provider store the processed form of a large, stable chunk of context — a long system prompt, a big reference file, a codebase summary — so it isn't reprocessed on every call. Subsequent requests that reuse that context are billed at a steep discount for the cached part. It pays off when you send the *same* large context across many calls, which is exactly what agents and long sessions do."
  - q: "Is a subscription or pay-per-token API cheaper for coding?"
    a: "It depends on your pattern. A **subscription** is a flat, predictable fee that wins for steady daily interactive use. **Pay-per-token API** scales precisely — cheaper for light or occasional use, but it can climb fast under heavy automation. Light steady use favours a subscription; bursty or low-volume use often favours the API. There's no universal winner."
---
# Understanding the cost

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Priced per token, input and output separately** — output usually costs more, because generating is the expensive part. **Four things drive the bill** — context size, output length, model tier, and number of calls. **Caching, model choice and the right billing model** keep it in check — and a local model just trades tokens for hardware and electricity.
</div>

This is lesson 10 of the path, and the last in Module 2. You know the providers, the models, the interfaces and the limits; now: where does the money actually go? Cost is where careless AI use stings, and where a little understanding saves a lot. By the end you'll understand token-based pricing, exactly what pushes a bill up, how prompt caching cuts the cost of large stable context, when a subscription beats pay-per-token (and vice versa), what local models trade cost for, and how to make a rough estimate. As with everything in this module, the *mechanism* is durable but the *numbers* are not — every figure below is explicitly illustrative, and you should check the provider's pricing page for real rates.

## Token-based pricing

Most providers charge by the **token** — the chunk of text models read and write, roughly a few characters each (a word is often one or two tokens; see [how models decide](/learn/ai-software-dev/how-models-decide/)). Prices are quoted **per million tokens**, and the key wrinkle is that **input and output are priced separately**:

- **Input tokens** are everything you send: your prompt, the conversation history, attached files — the whole context the model reads.
- **Output tokens** are everything the model generates back.

**Output usually costs more than input** — often several times more — because generating tokens is the expensive part: each output token is produced one at a time, each a full pass through the model, whereas input is read in one go. So the cost of a single call is roughly:

```text
cost ≈ (input_tokens  × input_rate_per_token)
     + (output_tokens × output_rate_per_token)

where output_rate is typically several times the input_rate
```

That formula is the whole game. Everything that follows is about which term grows and how to keep it down.

## What drives the cost up

Four levers move your bill, and they multiply rather than add up gently.

- **Context size.** Big files and long histories mean lots of input tokens *on every call*. Pasting a whole 5,000-line [GopherTrunk](/learn/intro-software-dev/what-is-software/) DSP module into the context, then asking ten follow-up questions, re-sends that mass of tokens ten times. Trim context to what the task needs — a discipline we develop in [Providing context](/learn/ai-software-dev/providing-context/).
- **Output length.** Asking for a full file rewrite costs more than asking for the three lines that changed, and at the higher output rate. Request the smallest useful output.
- **Model tier.** A frontier flagship can cost *far* more per token than the fast small tier — sometimes an order of magnitude. Routing routine work to the cheap tier and reserving the flagship for hard problems (the difficulty-routing from [Coding models compared](/learn/ai-software-dev/coding-models-compared/)) is the biggest single cost lever you have.
- **Number of calls.** This is the one that surprises people. An [agent](/learn/ai-software-dev/agentic-and-vibe-coding/) doesn't make one call — it *loops*: read files, think, edit, run tests, read results, think again. A single "fix this bug" task can be dozens of calls, each carrying the growing context. Autonomy is powerful and can get pricey fast, which is why agent runs deserve a watchful eye on spend.

## Prompt caching

When you send the *same* large context repeatedly — a long system prompt, a big reference document, a codebase summary — reprocessing it on every call is wasted money. **Prompt caching** fixes this: the provider stores the processed form of that stable chunk, and subsequent calls that reuse it are billed at a steep discount for the cached portion (you still pay full price for the new part of each request).

Caching shines in exactly the workloads that otherwise hurt: long agent sessions and chats that carry the same big context across many turns. The catch is that the cached context must be *stable* — change it and the cache is invalidated and rebuilt. So structure your requests with the unchanging bulk (instructions, reference files) up front and the varying part last, to keep the cache working for you. Exact discounts and cache lifetimes vary by provider; check their docs.

## Subscription vs API economics

This builds on the metering split from [Usage limits & tiers](/learn/ai-software-dev/usage-limits-and-tiers/), now through a cost lens.

| | Subscription | Pay-per-token API |
|---|---|---|
| **You pay** | A flat monthly fee | Per token, input and output separately |
| **Predictability** | High — same bill every month | Variable — tracks usage |
| **Best for** | Steady, heavy, interactive daily use | Light/occasional use, or automation you're willing to meter |
| **Risk** | Paying for a plan you underuse | A runaway agent quietly running up spend |
| **Needs monitoring** | Little — you're capped | Yes — set budgets and alerts |

The rule of thumb: **flat subscriptions win when your interactive use is steady and would otherwise be expensive per-token; the API wins when your use is light, occasional, or automated** — provided you monitor it. Many developers run both: a subscription for daily hands-on coding, an API key for scripts and agents.

## Local and open-weight models: a different trade

[Open-weight models](/learn/ai-software-dev/the-provider-landscape/) you run yourself have *no per-token charge at all* — but the cost doesn't vanish, it changes shape. You pay it up front and ongoing as **hardware** (a capable GPU or a lot of memory) and **electricity** (running that hardware), plus your own time to set up and maintain the model. For high, steady volume that can work out cheaper than per-token API pricing, and it buys privacy and offline use as a bonus. For light or occasional use, the hardware sits idle and a hosted model is far cheaper. The decision is the classic fixed-cost-versus-variable-cost trade.

## Estimating your spend

You can sanity-check a workload before you run it. Estimate the tokens, multiply by the rates, and remember that an agent multiplies the call count. Here is a single worked example — **the rates below are invented round numbers for arithmetic only, not real prices; check the provider's pricing page for current figures.**

```text
ILLUSTRATIVE ONLY — fictional rates
  Suppose a model costs:
    $3 per million input tokens
    $15 per million output tokens   (output pricier, as usual)

  One coding question:
    input:  8,000 tokens (your code + question + history)
    output: 2,000 tokens (the model's answer)

  input cost  = 8,000  / 1,000,000 × $3   = $0.024
  output cost = 2,000  / 1,000,000 × $15  = $0.030
  ----------------------------------------------------
  one call    ≈ $0.054   (about five cents)

  Now an agent that loops 30 times on a bug fix,
  each loop carrying ~8,000 input tokens:
    ≈ 30 × $0.054  ≈ $1.62 for the single task
```

The lesson of the arithmetic isn't the dollar figure — it's the *shape*. A single question is pennies; an agent that loops thirty times is the same pennies multiplied by thirty, with output (at the higher rate) doing much of the damage on verbose answers. That's why context-trimming, choosing the cheap tier for routine work, caching stable context, and watching agent loops all matter — and why a flat subscription can be reassuring precisely because it removes this arithmetic from your day.

<div class="knowledge-check" data-quiz data-correct-msg="Right — output is generated one token at a time, each a full pass through the model, so it's the costly part and is usually priced higher than input." markdown="0">
  <p class="knowledge-check__q">Quick check: in token-based pricing, why is output usually billed at a higher rate than input?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Output tokens are longer than input tokens</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Generating output is sequential and compute-heavy — each token is a full pass through the model</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Providers charge extra to discourage long answers</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Token-based pricing** — charged per million tokens, with input and output priced separately and output usually costing several times more.
- **The cost formula** — a call costs roughly input tokens × input rate plus output tokens × output rate.
- **Four cost drivers** — context size, output length, model tier, and number of calls; agents loop, so calls multiply fast.
- **Prompt caching** — reuse large stable context at a discount by keeping the unchanging bulk fixed and up front.
- **Subscription vs API** — flat fees suit steady heavy interactive use; pay-per-token suits light or automated use, with monitoring.
- **Local models** — trade per-token cost for hardware and electricity, a fixed-cost bet that pays off at high steady volume.

Next up: Module 3 opens by actually *using* the most accessible interface — the provider's chat app — in [The provider's app](/learn/ai-software-dev/the-provider-app/).
