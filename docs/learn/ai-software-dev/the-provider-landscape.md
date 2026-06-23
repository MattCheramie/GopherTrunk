---
slug: the-provider-landscape
title: The provider & model landscape
description: A fair, vendor-neutral map of the major AI providers and model families — Anthropic, OpenAI, Google, Meta and others — plus the big proprietary-vs-open-weight split and how to choose in a field that ships a new "best" model every few months.
keywords: AI providers, LLM model families, Anthropic Claude, OpenAI GPT, Google Gemini, Meta Llama, open-weight models, proprietary models, OpenRouter, AWS Bedrock, Google Vertex, Azure OpenAI, choosing an AI model
level: beginner
status: full
prereq:
  - types-of-models
faq:
  - q: "What is the difference between a proprietary and an open-weight model?"
    a: "A **proprietary** (or hosted) model lives on the provider's servers — you reach it through an API or a subscription and never see the underlying weights. An **open-weight** model is one whose trained parameters you can download and run yourself, on your own hardware or a rented server. Proprietary models tend to lead the capability frontier; open-weight models give you control, privacy and offline use, usually a step or two behind the very best closed models."
  - q: "Which AI provider is the best for coding?"
    a: "There's no durable answer, and any specific claim goes stale fast. The frontier families from **Anthropic, OpenAI and Google** all produce strong coding models, and open-weight options keep closing the gap. Choose on *fit* — the interfaces you'll use, your budget, privacy needs and ecosystem — rather than chasing whichever model topped a leaderboard last month. We turn this into a real decision in [Choosing a model & provider](/learn/ai-software-dev/choosing-model-and-provider/)."
  - q: "Do I have to pick just one provider?"
    a: "No. Aggregators like **OpenRouter** and cloud platforms such as AWS Bedrock, Google Vertex AI and Azure let you reach many models through one account or API. Many developers mix providers — a frontier model for hard problems, a cheap fast one for routine edits. We cover using one model versus several in [One model vs. many](/learn/ai-software-dev/one-model-vs-many/)."
---
# The provider & model landscape

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**A handful of providers** — Anthropic, OpenAI, Google, Meta and several others ship the model families you'll actually use. **The big split is proprietary vs open-weight** — hosted frontier models you can't control, versus downloadable models you can run yourself. **Choose on fit, not the leaderboard** — capabilities converge and the "best" model changes constantly, so pick by ecosystem and need.
</div>

This is lesson 6 of the path, and the first of Module 2 — the map of *who makes these models, what they cost, and how you reach them*. The previous module was about how a model works on the inside; now we zoom out to the marketplace around it. By the end of this lesson you'll know the major providers and their flagship model families, the one dividing line that matters most (whose computer the model runs on), the routes that let you reach many models at once, and a sane way to think about a field that announces a new "best" model every few months.

One honest caveat up front, and it runs through this whole module: **specifics here age fast.** Model names, version numbers, prices and rankings change on a timescale of weeks. So this lesson names *families* rather than this-week's-release, and everything concrete is offered as a snapshot, not a fact to memorise. When you need a current figure, the only reliable source is the provider's own live documentation.

## The major providers and their families

A model family is a lineage of related models from one provider — usually offered in a few sizes and refreshed over time. Here are the families you'll meet most often, described neutrally.

- **Anthropic — the Claude family.** Reached through an API and through Anthropic's own apps and developer tools. Claude is known for coding, long-context work and following instructions carefully.
- **OpenAI — the GPT family and the o-series.** The widely used GPT models plus a line of "reasoning" models (the o-series) that spend extra compute thinking before answering. Many people first meet these through the **ChatGPT** consumer product, which is the app built on top of those models.
- **Google — the Gemini family.** Google's flagship multimodal models, available through its developer platform and consumer apps, and notable for large context windows and tight integration with Google's cloud.
- **Meta — the Llama family.** Meta's models are **open-weight**: you can download the weights and run them yourself. Llama has anchored much of the open ecosystem and the tooling built around it.

Beyond the largest players, several other providers ship capable families worth knowing by name:

- **Mistral** (France) — efficient models, including open-weight releases.
- **DeepSeek** (China) — models that drew attention for strong reasoning and coding at low cost, with open-weight releases.
- **Alibaba — the Qwen family** — a broad, frequently updated open-weight lineup.
- **xAI — the Grok family** — proprietary models tied to xAI's products.
- **Cohere** — models aimed at enterprise use cases like retrieval and search.

This list is not exhaustive and it *will* shift. Treat it as "the neighbourhoods of the city," not a ranking. No family here is uniformly best; each has releases that lead on some task and trail on others.

## The big dividing line: proprietary vs open-weight

If you remember one distinction from this lesson, make it this one. It shapes cost, privacy, control and capability more than any benchmark does.

**Proprietary (hosted) models** run on the provider's servers. You send your request to their API or use their app, the model runs on their hardware, and you get an answer back. You never possess the **weights** — the trained numbers that *are* the model. The frontier of capability almost always lives here, because training and serving these models takes enormous, expensive infrastructure. The trade-off: you depend on the provider for availability, pricing and policy, and your data leaves your machine to be processed.

**Open-weight models** publish their trained weights for download. You (or a host you trust) run them on your own hardware. This buys **control and privacy** — the model can run fully offline, your code never leaves your network, and no provider can change the price or retire the model out from under you. The costs are real too: the very best open-weight models typically trail the closed frontier by a step, and you need capable hardware (a strong GPU or a lot of memory) plus the know-how to run and update them.

A word on "open." *Open-weight* means you get the weights; it does not necessarily mean open-source in the full sense (training data and code, an OSI-approved licence). Some releases come with usage restrictions. Read the licence before you build on one.

| | Proprietary / hosted | Open-weight |
|---|---|---|
| **Where it runs** | Provider's servers | Your hardware (or a host you choose) |
| **How you reach it** | API or subscription | Download and self-host |
| **Capability** | Usually leads the frontier | Often a step behind, closing fast |
| **Privacy / control** | Data leaves your machine; provider sets terms | Runs offline; you control everything |
| **Cost shape** | Pay per use or monthly fee | Hardware + electricity + your time |
| **Examples (illustrative)** | Claude, GPT/o-series, Gemini, Grok | Llama, Qwen, DeepSeek, Mistral releases |

For local, hands-on work — say, decoding sensitive radio captures with [GopherTrunk](/learn/intro-software-dev/what-is-software/) on an air-gapped machine — an open-weight model you run yourself may matter more than a few points of capability. For the hardest novel problem at 2 a.m., a frontier hosted model often earns its keep. Many developers keep both within reach.

## Aggregators and cloud routes

You don't always go straight to a provider. Two kinds of intermediary let you reach many models through one door.

**Aggregators** sit in front of multiple providers and expose them through a single account and API. **OpenRouter** is the best-known example: one key, one billing relationship, and the ability to switch between dozens of models — proprietary and open-weight — by changing a single string in your request. That makes comparison and fallback easy, at the cost of adding a middleman between you and the provider.

**Cloud platforms** offer models inside the big cloud ecosystems, which is convenient if your project already lives there:

- **AWS Bedrock** — many model families, including third-party ones, served within Amazon's cloud.
- **Google Vertex AI** — Google's platform, the enterprise route to Gemini and other models.
- **Azure** — Microsoft's cloud, a common enterprise route to OpenAI's models and others.

The appeal of the cloud route is governance and integration: billing, security, data-residency and access controls handled the same way as the rest of your infrastructure. The trade-off is a bit more setup and sometimes a lag before the very newest model versions arrive, compared with going to the provider directly.

## How to think about a churning field

Every few months, someone ships a model that tops some chart, and the headlines declare a new king. It is genuinely hard to keep up — and mostly, you shouldn't try to. Three observations make the churn manageable.

**Capabilities converge.** When one provider unlocks a capability — longer context, better reasoning, tool use — the others tend to follow within a release or two. The frontier moves *together*. So the gap between "the best model" and "a very good model" is usually smaller than the marketing suggests, and it narrows further every quarter.

**Leaderboards churn — and don't measure your work.** A model can lead a public benchmark and still be a poor fit for your codebase, your latency budget, or your privacy rules. A leaderboard is a single number on someone else's test; your project is the only benchmark that counts. We'll look hard at reading benchmarks skeptically in the next lesson, [Coding models compared](/learn/ai-software-dev/coding-models-compared/).

**So choose on fit and ecosystem, not on one score.** The questions that actually decide it are durable: Which interface do I want to work in — an app, an [IDE](/learn/intro-software-dev/what-is-software/), the terminal? What's my budget? Do I need the model to run offline? Does it plug into the tools and cloud I already use? Those answers change slowly even as the models change fast. Pick a family and a route that fit your work, and treat swapping models as the cheap, routine operation it has become — especially if you go through an aggregator.

<div class="knowledge-check" data-quiz data-correct-msg="Right — open-weight models publish their trained weights so you can download and run them on your own hardware." markdown="0">
  <p class="knowledge-check__q">Quick check: what most clearly distinguishes an open-weight model from a proprietary one?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It always scores higher on public benchmarks</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Its trained weights are downloadable, so you can run it on your own hardware</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It can only be reached through a paid monthly subscription</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Provider families** — Anthropic's Claude, OpenAI's GPT and o-series (behind ChatGPT), Google's Gemini and Meta's open-weight Llama lead, alongside Mistral, DeepSeek, Qwen, Grok and Cohere.
- **Proprietary vs open-weight** — hosted frontier models you reach by API or subscription versus downloadable models you run yourself for control and privacy.
- **Aggregators and cloud routes** — OpenRouter and clouds like AWS Bedrock, Google Vertex and Azure let you reach many models through one account.
- **Capabilities converge** — the frontier moves together, so the gap between "best" and "very good" is smaller and shrinking.
- **Choose on fit** — pick by interface, budget, privacy and ecosystem, not by whichever model led a leaderboard this month.
- **Specifics age fast** — treat every model name, price and ranking here as a snapshot, and check the provider's live docs for current numbers.

Next up: the axes that actually tell coding models apart — and how to read a benchmark without being fooled by it — in [Coding models compared](/learn/ai-software-dev/coding-models-compared/).
