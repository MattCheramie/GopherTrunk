---
slug: context-windows
title: The context window, in detail
description: The context window is everything a model sees at once — system prompt, history, files, and your message — measured in tokens and capped by a hard limit. Learn how attention, "lost in the middle", and context rot affect quality, and why context is both your biggest quality lever and a direct cost driver.
keywords: context window, context length, tokens, attention, lost in the middle, context rot, context degradation, prompt, truncation, token limit, quadratic attention, context management
level: intermediate
prereq:
  - how-models-decide
status: full
faq:
  - q: "What exactly counts toward the context window?"
    a: "Everything the model processes in one go: the hidden **system prompt**, the **conversation history** so far, any **files or attachments** you've included, and your **current message** — plus the room reserved for the model's reply. It's all measured in tokens, and input and output together must fit under the model's limit."
  - q: "Is a bigger context window always better?"
    a: "No. A larger window lets you include more, but **relevant beats big**: stuffing in unneeded files dilutes attention, can bury the important parts in the middle where recall is weakest, and costs more because you pay per token. Use the space deliberately rather than filling it."
  - q: "What happens when I exceed the context window?"
    a: "The limit is hard. Tools handle overflow by **truncating** — usually dropping or summarizing the oldest content — so the model effectively *forgets* the earliest parts of the conversation. That's why a long session can lose track of something you said near the start; the fix is to prune, summarize, or start fresh."
---
# The context window, in detail

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Context is everything the model sees at once** — system prompt, history, files, and your message, all measured in tokens. **The window is a hard limit** — input plus output must fit, and overflow means truncation and forgetting. **It's a lever and a cost** — the right context is your biggest control on output quality, and every token is something you pay for, so relevant beats big.
</div>

This is lesson 4 of the path, and the one to slow down on. Almost every frustration and almost every great result with a coding model traces back to *context* — what the model could see when it answered. In [the last lesson](/learn/ai-software-dev/how-models-decide/) we watched the model generate token by token; this lesson explains the space those tokens live in. By the end you'll know precisely what "context" includes, why there's a hard ceiling on it, how the model's attention behaves across long inputs, and why managing context well is the single highest-leverage skill in this whole path.

## What "context" actually is

**Context** is everything the model processes in a single pass to produce its next reply. It is not just your latest message. It's the whole bundle:

| Part of context | What it is | Who put it there |
|-----------------|------------|------------------|
| **System prompt** | Hidden instructions setting the model's role and rules | The tool/provider |
| **Conversation history** | Every earlier message, yours and the model's | The session so far |
| **Files & attachments** | Code, docs, images, or data you've included | You |
| **Current message** | What you just typed | You |

All of it is concatenated into one long sequence of tokens and fed to the model at once. The model has no memory between calls beyond this — it doesn't "remember" your last session unless that information is in the context this time. Anything outside the context window might as well not exist to the model. This is why two people can get wildly different answers to the same question: their context differed.

## The window is a hard limit

The **context window** is the maximum number of tokens a model can handle in one pass — and it counts *both* the input (everything above) *and* the output the model is about to generate. It is a hard architectural limit, not a guideline. As a rough illustration, models today range from tens of thousands of tokens up to a million or more, and the numbers keep climbing — so don't memorize a figure; check the provider's docs for any specific model.

What happens when you hit the ceiling? The model can't simply read more, so tools handle overflow by **truncating** the context — typically dropping or summarizing the oldest messages to make room. The visible symptom is *forgetting*: in a long chat the model loses track of a decision you made near the start, because that decision scrolled out of the window. It's not being careless; the tokens are literally gone from what it can see. Remember too that a long input *and* a long requested output have to share the same budget — ask for a huge file rewrite at the end of a packed session and you can run out of room for the answer.

## Attention: how the model relates tokens

Why does length matter so much beyond the hard cap? The answer is **attention**, the mechanism at the core of modern LLMs. As the model processes the sequence, attention lets every token "look at" every other token and weigh how relevant each one is to what it's currently deciding. That's how a token at the end of your prompt can be informed by a constraint you stated at the beginning — the model relates them directly.

This is powerful, but it isn't free. Because every token attends to every other token, the work grows roughly **quadratically** with the length of the context: double the tokens and you do something closer to four times the relating, not twice. (Providers use optimizations that soften this in practice, but the scaling pressure is real.) That cost is one reason longer context is slower and more expensive, and it's a hint that more tokens are not automatically better.

## "Lost in the middle"

Attention reaching everywhere doesn't mean it reaches *evenly*. A well-documented effect, often called **"lost in the middle,"** is that models recall information placed at the **start** and **end** of a long context more reliably than information buried in the **middle**. If the one critical detail is sitting halfway down a giant pasted file, the model is measurably more likely to overlook it than if it's near the top or the bottom.

The practical takeaway: position matters. Put the instruction you most want followed, and the most important reference material, where attention is strongest — typically the very end (closest to the question) and the beginning. Don't assume that because something is *in* the context, it will be *used* equally.

## Context degradation, or "context rot"

There's a slower-acting cousin of the problem. In very long sessions — many turns, lots of accumulated history — quality can quietly drift even before you hit the hard limit. This is sometimes called **context degradation** or **"context rot."** Stale instructions, half-abandoned approaches, superseded code, and contradictory back-and-forth pile up, and the model is now attending to a noisy mess that no longer cleanly represents what you want. Answers get vaguer, the model re-raises settled questions, or it clings to an early wrong turn.

The cure is hygiene: when a session has gotten long and muddy, **start fresh** with a clean prompt that states the current goal and includes only the still-relevant material. A short, sharp context usually beats a long, polluted one.

## The dual nature: lever and cost

Here is the idea to carry out of this lesson. Context has two faces, and they pull in opposite directions.

It is your **single biggest lever on output quality.** A coding model with the right files in front of it gives a right answer; the same model guessing blind gives a plausible wrong one. If you ask GopherTrunk's assistant to "add a unit test for the down-converter" with nothing else, it will invent function names and signatures — pure hallucination territory, as we saw in [How a model decides](/learn/ai-software-dev/how-models-decide/). Paste in the actual `internal/scanner/ccdecoder/ddc.go` and the relevant test file, and it can write a test that matches your real API, your real naming, and your real edge cases. Same model, completely different result — the difference is entirely context.

It is also a **direct cost driver.** You pay per token (see [Understanding cost](/learn/ai-software-dev/understanding-cost/)), every token in context is processed on every turn, and attention's quadratic pressure means long contexts are slower and pricier. Dumping your whole repository into the window isn't just wasteful — by burying the relevant parts in the middle, it can make answers *worse*.

So the guiding principle is **relevant beats big.** Practical habits:

- **Prune** — include the files and snippets that bear on the task, not everything nearby.
- **Summarize** — replace a long settled discussion with a short statement of the conclusion.
- **Start fresh** — when a session is long or has drifted, open a clean one with just what matters now.
- **Position deliberately** — put the most important material and instruction near the end, where recall is strongest.

We'll turn these habits into concrete technique in [Providing context](/learn/ai-software-dev/providing-context/) later in the path. For now, internalize the mindset: you are curating what the model sees, and that curation is most of the job.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the window is a hard token limit on input plus output, and overflow forces truncation, so the model forgets the oldest content." markdown="0">
  <p class="knowledge-check__q">Quick check: What happens when a conversation exceeds the model's context window?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The model automatically reads the extra tokens in a second pass</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The context is truncated — usually the oldest content is dropped or summarized — so the model effectively forgets it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The model permanently stores the overflow in long-term memory</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Context is everything seen at once** — system prompt, conversation history, files and attachments, and your current message, all as one token sequence.
- **The window is a hard limit** — input plus output must fit under the model's token cap, and overflow forces truncation and forgetting.
- **Attention has limits** — every token can relate to every other, but compute grows roughly quadratically with length, and recall is weaker for content in the middle ("lost in the middle").
- **Context rot** — very long, messy sessions degrade quality even below the hard limit, so starting fresh often beats pressing on.
- **Lever and cost** — the right context is your biggest control on quality and a direct cost driver, so relevant beats big: prune, summarize, position deliberately, and start fresh.

Next up: not every AI model is a chat LLM — some reason, some turn text into vectors, some read images. See [The types of AI models](/learn/ai-software-dev/types-of-models/).
