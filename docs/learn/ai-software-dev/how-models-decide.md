---
slug: how-models-decide
title: "How a model decides: tokens & prediction"
description: Inside the moment of generation — tokens and tokenization, the autoregressive loop, and sampling settings like temperature and top-p. Learn why the same prompt yields different code and why a model can invent an API that doesn't exist.
keywords: tokens, tokenization, autoregressive, next-token prediction, temperature, top-p, sampling, greedy decoding, probability distribution, streaming, stop tokens, hallucination
level: beginner
prereq:
  - how-models-are-trained
status: full
faq:
  - q: "Why does the same prompt give me different code each time?"
    a: "Because generation usually involves **sampling** — the model produces a probability distribution over possible next tokens and picks one with some randomness, controlled by a setting called *temperature*. Higher temperature means more variety; even at the lowest setting, small numerical differences mean output is near-deterministic but not guaranteed identical."
  - q: "What is a token, and why should I care about token counts?"
    a: "A **token** is a subword chunk of text — a short word might be one token, a longer one several, and whitespace and code symbols tokenize their own way. You care because limits and cost are measured in tokens, not characters or lines, so the same idea written verbosely costs more. The [context window](/learn/ai-software-dev/context-windows/) lesson covers the limit side in detail."
  - q: "Why does the model invent functions or APIs that don't exist?"
    a: "Because it generates a *plausible* sequence of tokens, and a fabricated function call can look exactly like a real one token-for-token. The model has no built-in check that the sequence corresponds to something real, so a confident, well-formed — but invented — API is a natural output. Always verify calls against real documentation."
---
# How a model decides: tokens & prediction

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Tokens are the unit** — models read and write in subword tokens, and limits and cost are counted in tokens, not characters. **The autoregressive loop** — the model outputs a probability distribution, picks one token, appends it, and repeats. **Sampling controls variety** — temperature and top-p decide how random the pick is, which is why the same prompt can yield different code and why a model can confidently invent an API.
</div>

This is lesson 3 of the path. We've established that an LLM predicts the next token and that training shaped which tokens it finds likely. This lesson opens up the actual moment of generation: what a token *is*, how the model turns its predictions into the exact text you see, and the dials that control that process. By the end you'll understand why the same question can produce different answers, why token counts govern your limits and cost, and — concretely — why a model can hand you a Go function that calls something which simply doesn't exist.

## Tokens and tokenization

The model doesn't see characters or whole words. Before any text reaches it, a step called **tokenization** chops the text into **tokens** — subword units drawn from a fixed vocabulary the model learned during training.

A common short word might be a single token. A longer or rarer word gets split into several. Whitespace, punctuation, and code symbols tokenize their own way, and a leading space is often part of the token. As a rough rule of thumb for English prose, a token averages around four characters, so a token is a bit shorter than a typical word — but code, with its braces, identifiers, and indentation, tokenizes differently and often less efficiently. Don't rely on the exact ratio; providers publish tokenizers and counting tools for precise numbers.

Here's the intuition with a small Go snippet. Visually it's a few lines, but to the model it's a sequence of tokens — keywords, names, punctuation, and the whitespace between them:

```go
func decode(samples []complex64) int {
    return len(samples)
}
```

Why care? Because everything that's *measured* about a model is measured in tokens: the limit on how much it can handle at once (the [context window](/learn/ai-software-dev/context-windows/)) and what you [pay](/learn/ai-software-dev/understanding-cost/) for usage. The same idea written verbosely uses more tokens than written tersely, and a wall of generated code can be a lot of tokens. Thinking in tokens, not lines, is a habit worth building early.

## The autoregressive loop

With the input tokenized, generation runs as a tight loop. The word for it is **autoregressive** — the model feeds its own output back in as it goes. Each step looks like this:

1. The model reads all the tokens so far (your prompt plus whatever it has generated).
2. It produces a **probability distribution** over its entire vocabulary — a number for every possible next token saying how likely each one is.
3. It **picks one token** from that distribution.
4. It **appends** that token to the sequence and goes back to step 1.

So a paragraph of output is this loop run hundreds of times, one token at a time, each new token chosen in light of everything before it. Crucially, the model commits to each token before generating the next — it can't go back and revise an earlier word once it's out. That one-directional, one-token-at-a-time nature explains a lot of LLM behaviour, including why a single early misstep can send an answer down the wrong path.

## Sampling: how the token gets picked

Step 3 — *picks one token* — is where you have control. Several strategies exist for turning the probability distribution into an actual choice.

| Strategy | How it picks | Effect |
|----------|--------------|--------|
| **Greedy** | Always the single most likely token | Repetitive, predictable, can feel flat |
| **Temperature** | Scales the distribution before sampling | Higher = more random and creative; lower = safer |
| **Top-p (nucleus)** | Sample only from the smallest set of tokens whose probabilities add up to *p* | Trims the unlikely long tail, keeps sensible variety |

**Temperature** is the dial you'll hear about most. Think of it as a randomness knob. At a high temperature the model is more willing to pick less-likely tokens, giving more varied and "creative" output — useful for brainstorming, riskier for code. At a low temperature it sticks close to the most probable tokens, giving safer, more focused output. **Top-p** works alongside it by restricting choices to the most probable cluster of tokens, cutting off the weird long tail entirely.

Setting the temperature to its lowest, near-zero value approximates **greedy** decoding: the model nearly always takes the top token, so output is *near-deterministic*. Note "near": floating-point math and infrastructure details mean even temperature 0 is not guaranteed to produce byte-for-byte identical results every time. For coding work you'll usually want a lower temperature — you want correct and consistent far more than you want surprising.

## Why the same prompt yields different code

This is the everyday consequence: because picking the next token involves randomness (unless temperature is pinned low), running the same prompt twice can produce different answers. Ask a coding assistant to "write a function to detect a control channel" twice and you may get two valid but different implementations — different variable names, a different structure, a different edge case handled.

That's not a malfunction. It's sampling doing its job. For a GopherTrunk task where you want repeatable output — say, regenerating a test you'll commit — prefer a low temperature and, where the tool allows, fix the setting. For exploration, where you *want* to see alternatives, a higher temperature is a feature.

## Why a model can invent an API that doesn't exist

Now the mechanism behind a hallucination we've flagged twice, seen from the generation side. The model is assembling a *plausible token sequence*. A real function call and an invented one can look identical token by token — `samples.Demodulate()` is just as grammatical and just as likely-looking whether or not that method actually exists in the codebase or the library.

At no point in the loop does the model consult a list of real, callable functions and check the sequence against it. It only knows what's *probable* given the patterns it learned, and a confidently-named method on a plausible type is highly probable text. So it can hand you code that compiles in your head and falls over the moment you build it, because the API was never real. This connects straight back to the hallucination story from [the last lesson](/learn/ai-software-dev/how-models-are-trained/): a plausible token sequence is not the same as a real one. The practical defense — verify every unfamiliar call against actual docs or by compiling it — is exactly the discipline we build in [Verification and trust](/learn/ai-software-dev/verification-and-trust/).

## Streaming and stop conditions

Two last mechanics you'll notice in real tools. **Streaming** is why text appears token by token as the model writes rather than all at once: each token is sent to your screen as soon as it's generated, which is just the autoregressive loop made visible. It makes tools feel responsive and lets you cancel early if the answer is going wrong.

Generation has to end somehow, and that's handled by **stop conditions**. The model can emit a special end-of-sequence token signalling "I'm done," the tool can impose a maximum number of output tokens (a hard cap that can cut a long answer off mid-sentence), or you can supply explicit *stop sequences* — strings that, when produced, halt generation. If you ever see an answer end abruptly, a token limit or stop condition is usually why.

<div class="knowledge-check" data-quiz data-correct-msg="Right — at each step the model produces a probability distribution, samples one token, appends it, and repeats." markdown="0">
  <p class="knowledge-check__q">Quick check: How does a model generate a multi-word answer?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It writes the whole answer at once from a fixed lookup table</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It produces a probability distribution over the vocabulary, picks one token, appends it, and repeats the loop</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It edits an answer back and forth until every word is provably correct</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Tokens** — models read and write in subword tokens, and code, whitespace, and punctuation each tokenize their own way; limits and cost are counted in tokens.
- **Autoregressive loop** — the model outputs a probability distribution, picks one token, appends it, and repeats, committing to each token before the next.
- **Sampling** — greedy, temperature, and top-p decide how the next token is chosen; temperature is the randomness dial, and low (near-0) is near-deterministic but not guaranteed identical.
- **Variability** — sampling randomness is why the same prompt can give different code, useful for exploration and worth pinning low for repeatable work.
- **Invented APIs** — a plausible token sequence isn't a real one, and the model never checks against real APIs, so it can confidently emit calls that don't exist.
- **Streaming and stops** — output appears token by token, and generation ends on an end-of-sequence token, an output cap, or a stop sequence.

Next up: the model can only decide based on what it can see at once — so what exactly can it see, and how big is that window? See [The context window, in detail](/learn/ai-software-dev/context-windows/).
