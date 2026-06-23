---
slug: how-models-are-trained
title: How AI models are trained
description: AI coding models are built in stages — pretraining on a huge corpus, fine-tuning to follow instructions, and preference tuning to be helpful and honest. Learn the pipeline and what it means for knowledge cutoffs, bias, and hallucination.
keywords: how LLMs are trained, pretraining, fine-tuning, RLHF, instruction tuning, training data, knowledge cutoff, hallucination, alignment, open weights, model training pipeline
level: beginner
prereq:
  - what-is-ai-for-coding
status: full
faq:
  - q: "Why doesn't the model know about a library version that just came out?"
    a: "Because of the **knowledge cutoff**: a model's training data ends on some date, so anything released after that — a new library version, a renamed API, a fresh framework — is simply absent from what it learned. The model will still answer confidently, often by guessing based on older patterns, so always check current docs for anything recent."
  - q: "Why do models make things up (hallucinate)?"
    a: "Training rewards fluent, confident, helpful-sounding continuations, and nothing in the basic pipeline rewards the model for saying \"I don't know\" or for actually checking a fact. So when it lacks the real answer it produces a **plausible** one anyway, because plausible-and-confident is what it was optimized to generate."
  - q: "What's the difference between open and closed models?"
    a: "**Closed** (or proprietary) models are trained and run by a company that exposes them through an API or app but doesn't release the trained parameters — Anthropic's Claude, OpenAI's GPT, and Google's Gemini are examples. **Open-weight** models, like Llama, Qwen, DeepSeek, and Mistral, publish their trained parameters so you can download and run them yourself. Both are trained with broadly the same pipeline."
---
# How AI models are trained

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**A multi-stage pipeline** — models are pretrained on a huge corpus, then fine-tuned to follow instructions, then preference-tuned to be helpful, honest, and harmless. **Data shapes the model** — coverage, quality, and recency of the training data decide what the model knows and the biases it inherits. **Cutoffs and hallucination are built in** — training ends on a date and rewards confident fluency, which is why models miss new releases and sometimes make things up.
</div>

This is lesson 2 of the path. In [the last lesson](/learn/ai-software-dev/what-is-ai-for-coding/) we said an LLM is a next-token predictor that learned the statistical patterns of text and code. This lesson explains *how* it learned them. Understanding the training pipeline isn't academic — it directly explains three things you'll feel every day as a developer: why a model doesn't know about a brand-new library, why it sometimes reflects skewed assumptions, and why it confidently invents answers. By the end you'll be able to reason about a model's behaviour from how it was made.

## The training pipeline at a glance

A modern coding model isn't trained in one shot. It's built in stages, each one shaping it further. The four stages below run roughly in order, and each depends on the one before it.

| Stage | What happens | What it gives the model |
|-------|--------------|--------------------------|
| **1. Pretraining** | Predict the next token over a massive corpus | Grammar, facts, and coding patterns |
| **2. Training data** | Curating and weighting that corpus | The boundaries of what it knows |
| **3. Instruction tuning** | Fine-tune on instruction → response examples | The habit of following requests |
| **4. Preference tuning** | Adjust toward responses humans prefer | Being helpful, honest, and harmless |

## Stage 1: Pretraining

**Pretraining** is the heart of it. The model is shown an enormous amount of text and code and given one repetitive task: predict the next token. Hide the next token, let the model guess, compare the guess to the real token, and nudge the model's billions of internal parameters to make the right answer slightly more likely. Do that over and over, across a corpus far larger than any person could read in a lifetime.

Nobody hand-labels grammar rules or teaches it the syntax of Go. All of that is absorbed *implicitly* from the prediction task. To predict the next token in a Go function well, the model has to internalize how Go functions are structured; to continue a sentence sensibly, it has to capture facts and relationships. The patterns we relied on in lesson 1 — that code is regular and predictable — are exactly what pretraining soaks up. The output of this stage is a "base model": fluent and knowledgeable, but not yet good at *following instructions*. Ask a raw base model a question and it might just continue your text with more questions, because that's a plausible continuation.

## Stage 2: The training data, and why it matters

Pretraining is only as good as what you feed it, so the corpus matters enormously. Three properties decide what you get.

- **Coverage** — what topics, languages, and libraries appear at all. If little Go DSP code exists in the data, the model will be weaker at it than at, say, web JavaScript, which is everywhere.
- **Quality** — clean, correct, well-structured examples teach better patterns than buggy or low-quality ones. Garbage in, garbage patterns out.
- **Recency** — the data was collected up to some point in time. Everything after that is invisible to the model.

That last point has a name and a direct consequence for you, so it gets its own section below. The first two explain why a model can be brilliant at popular, well-documented tasks and shaky at niche or specialized ones.

## Stage 3: Instruction tuning

A base model that just continues text isn't much use as an assistant. **Instruction tuning** (a form of *supervised fine-tuning*) fixes that. The model is trained further on a curated set of examples that pair an instruction with a good response — "Write a function that does X" followed by a quality answer; "Explain this error" followed by a clear explanation.

This is still next-token prediction, but now over examples of *being helpful on request*. After this stage the model has learned the format of assistance: when you ask for something, produce the thing. This is what turns a raw language model into something that behaves like the coding assistant you actually talk to.

## Stage 4: Preference tuning and alignment

The final common stage shapes the model's *style and behaviour* toward what people prefer. The best-known technique is **RLHF** — reinforcement learning from human feedback. The idea: show humans (or a model trained to imitate them) several candidate responses, have them rank which is better, and tune the model toward the preferred ones. Related approaches go by other names, but the goal is the same.

This stage is often described with the shorthand **helpful, honest, and harmless** — the model is pushed to actually address the request, avoid stating things it has reason to doubt, and decline genuinely harmful asks. This is also called **alignment**: bringing the model's behaviour into line with what users and providers want. It's why a modern assistant is polite, stays roughly on task, and refuses obviously dangerous requests.

## What a developer actually feels

The pipeline above isn't trivia. Three of its properties show up constantly in real work.

### Knowledge cutoff

Because the training data stops being collected at some point, every model has a **knowledge cutoff** — a date beyond which it has learned nothing. As a rough illustration: if a model's data ends in one year and a popular library ships a major rewrite the next year, the model won't know the new API. It will still answer — usually by pattern-matching from the older version — and may invent a call that no longer exists. The fix is structural, not a prompt trick: for anything recent, check the library's current docs, or give the model the up-to-date material directly (we cover that in [Providing context](/learn/ai-software-dev/providing-context/)). Always treat the cutoff as real and never assume the model knows about this week's release.

### Inherited bias

A model learns the patterns in its data, including the skewed ones. If the training code overwhelmingly uses one framework, one style, or one set of assumptions, the model will lean that way even when it isn't the best fit for your project. This isn't malice; it's a faithful reflection of what it saw. Notice it, and steer explicitly when your context differs from the internet's average.

### Why models hallucinate

Now the deep reason behind the hallucinations we flagged in lesson 1. Across the whole pipeline, the model is rewarded for producing fluent, confident, helpful-sounding continuations. At no point in the basic process is it rewarded for having actually *checked* anything, or penalized cleanly for being smoothly wrong rather than honestly uncertain. There is no built-in "I verified this" step.

So when the model lacks the real answer — because of the cutoff, thin coverage, or an obscure detail — it doesn't stop. It generates the most plausible-looking answer it can, in the same confident tone it uses when it's right. That's a **hallucination**: a fluent, well-formatted, completely fabricated answer. Providers work hard to reduce it, and techniques like giving the model real source material help a lot, but it can't be fully trained away because it's a side effect of optimizing for plausible, confident text. This is the single most important reason to [verify](/learn/ai-software-dev/verification-and-trust/) what a model tells you.

## Scale, cost, and open vs closed

Two practical notes round out the picture. First, **scale and cost**: pretraining a frontier model consumes a vast amount of computation and money, which is why only a handful of organizations train the largest models from scratch — though fine-tuning an existing model is far cheaper and within reach of many teams.

Second, **open vs closed**. *Closed* (proprietary) models — such as Anthropic's Claude, OpenAI's GPT, and Google's Gemini — are trained and operated by a company that exposes them through an API or app without releasing the trained parameters. *Open-weight* models — such as Llama, Qwen, DeepSeek, and Mistral — publish their parameters so you can download and run them yourself. Both families are built with broadly the same pipeline; the difference is who holds the weights and where the model runs, which we'll weigh up in [The provider landscape](/learn/ai-software-dev/the-provider-landscape/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — training rewards confident, fluent, helpful continuations, with no built-in check that the answer is true." markdown="0">
  <p class="knowledge-check__q">Quick check: Why do models hallucinate — confidently produce wrong answers?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">They are deliberately programmed to lie when they're unsure</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Training rewards fluent, confident, helpful-sounding continuations, with no built-in step that checks the answer is actually true</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Their training data is always completely empty of real facts</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Pretraining** — predicting the next token over a huge corpus is how the model implicitly absorbs grammar, facts, and coding patterns.
- **Data is destiny** — coverage, quality, and recency of the training data set the boundaries of what the model knows and the biases it carries.
- **Instruction and preference tuning** — fine-tuning teaches the model to follow instructions, and preference tuning (RLHF) makes it helpful, honest, and harmless.
- **Knowledge cutoff** — training data ends on a date, so the model may not know a brand-new library version and will guess confidently instead.
- **Hallucination is structural** — the model is optimized for fluent, confident continuations with no built-in verification, so it sometimes invents plausible falsehoods.
- **Open vs closed** — closed models are run by a provider behind an API; open-weight models publish their parameters to download and run yourself.

Next up: zoom in on the moment of generation — what a token is, how the model turns probabilities into the exact words it gives you, and why the same prompt can yield different code. See [How a model decides: tokens & prediction](/learn/ai-software-dev/how-models-decide/).
