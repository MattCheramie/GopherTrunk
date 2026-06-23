---
slug: what-is-ai-for-coding
title: What is AI, for a developer?
description: For a developer, "AI" today means the Large Language Model — a next-token predictor trained on text and code. Learn what that is, why it's powerful for programming, and why it's fallible.
keywords: AI for coding, large language model, LLM, next-token prediction, machine learning, narrow AI, AGI, generative AI, AI coding assistant, statistical patterns
level: beginner
status: full
faq:
  - q: "Is an AI coding assistant the same as artificial general intelligence (AGI)?"
    a: "No. Today's coding tools are **narrow AI** — they do one kind of task (predicting likely text and code) very well. AGI, a hypothetical system that matches human flexibility across any task, does not exist. Treat current tools as powerful pattern engines, not thinking colleagues."
  - q: "Does the model actually understand the code it writes?"
    a: "Not in the human sense. A Large Language Model captures the **statistical patterns** of how code is written, so its output often looks and behaves like understanding. But it is predicting a plausible continuation, not reasoning about meaning, which is exactly why it can produce confident, well-formatted answers that are wrong."
  - q: "If it's just predicting text, why is it so good at code?"
    a: "Because code is unusually **patterned** text — strict grammar, repeated idioms, conventional names — and there is an enormous amount of public code to learn from. Predicting the next token is a much easier job when the material is this regular, which is why coding is one of the strongest use cases for these models."
---
# What is AI, for a developer?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**AI here means the LLM** — for coding, "AI" today is the Large Language Model, learned from data rather than hand-written rules. **It's a next-token predictor** — it models the statistical patterns of text and code, not human-style understanding. **Powerful but fallible** — code is highly patterned text, so prediction works well, but the model produces the *plausible* continuation, not the guaranteed *correct* one.
</div>

This is lesson 1 of the path. The word "AI" gets stretched to cover everything from chess engines to science-fiction robots, so before we talk about tools, providers, or how to use them well, it's worth being precise about what "AI" means for someone writing software in 2026. By the end of this lesson you'll know what kind of system you're actually talking to when you use a coding assistant, the single core idea everything else in this path builds on, and why the same tool can be astonishingly helpful one minute and confidently wrong the next.

## Two ways to make a computer "smart"

For most of computing history, getting a machine to do something clever meant **writing the rules yourself**. A programmer studies the problem, works out the logic, and encodes it as explicit instructions — the kind of step-by-step program we met in [What is software?](/learn/intro-software-dev/what-is-software/). If you want a program to detect a control channel in a radio signal, you write code that says exactly what a control channel looks like and how to find it. The intelligence lives in the rules a human wrote.

**Machine learning** flips that around. Instead of hand-writing the rules, you show a program many examples and let it *learn* the patterns on its own. You don't tell it the rule; you give it data and a training process, and it builds an internal model that captures the regularities in that data. The intelligence is learned from examples rather than dictated.

| Approach | Who supplies the logic | Good when |
|----------|------------------------|-----------|
| **Hand-written rules** | A human, explicitly in code | The rules are known, precise, and stable |
| **Machine learning** | The model, learned from data | The pattern is fuzzy, huge, or hard to write down |

Almost everything people call "AI" today is machine learning. The chess engine, the spam filter, the photo tagger, and your coding assistant are all programs that learned patterns from data rather than running rules a person spelled out by hand.

## The dominant tool for coding: the LLM

Within machine learning, the tool that has transformed software development is the **Large Language Model**, or **LLM**. "Large" refers to the sheer size of the model and the amount of text it was trained on; "language model" means its job is to model language — to predict what text is likely to come next. We'll see exactly how one is built in [How AI models are trained](/learn/ai-software-dev/how-models-are-trained/).

When you ask a coding assistant to write a function, explain an error, or refactor a file, an LLM is doing the work underneath. The chat window, the editor autocomplete, and the command-line agent are all different *interfaces* onto the same kind of model. Get the LLM right in your head and the rest of this path falls into place.

## Narrow AI, not the robot from the movies

There is a lot of noise about **artificial general intelligence (AGI)** — a hypothetical system that could match a human across any task, learn anything, and reason about the world. That system does not exist today, and nothing in this path requires it.

What you actually have is **narrow AI** (also called task AI): a system that is very good at one *kind* of task and has no capability outside it. An LLM is narrow in this sense. It is extraordinarily good at producing plausible text and code, and that is genuinely useful — but it is not a general mind. It does not have goals, it does not know things the way you do, and it does not "want" your program to work. Keeping that framing honest will make you a far better user of these tools than any amount of hype.

## The core idea: a next-token predictor

Here is the single insight everything else builds on. **An LLM is a next-token predictor.** A *token* is a small chunk of text — roughly a word, part of a word, or a piece of punctuation (we cover tokens properly in [How a model decides](/learn/ai-software-dev/how-models-decide/)). Given the text so far, the model's one and only job is to estimate which token is likely to come next, pick one, and repeat.

That's it. There is no separate "understanding" step and no internal checklist of true facts. The model was trained on an enormous amount of text and code, and from that it absorbed the *statistical patterns* of language: which words tend to follow which, how a Go function is shaped, what an error message usually looks like, how a comment is phrased. When it answers you, it is replaying those patterns to produce a likely continuation of your prompt.

This is a humbler description than "the AI understands your code," and it's the accurate one. The model is modelling language, not meaning. It can be wrong in ways a person who truly understood the problem never would be — and it can also be right in ways that feel uncanny, because the patterns it learned are deep and broad.

## Why this is powerful for code

If an LLM just predicts plausible text, why is it so good at programming specifically? Because **code is unusually patterned text**.

Natural language is messy and ambiguous. Code, by contrast, has strict grammar, a limited vocabulary of keywords, heavily repeated idioms, and strong conventions for names and structure — the kind of regularity we toured in [A tour of programming languages](/learn/intro-software-dev/language-tour/). Predicting the next token is far easier when the material is this regular. On top of that, there is a vast amount of public code to learn from, covering common libraries, patterns, and bug fixes.

The result is a tool that can draft a function, translate code from one language to another, write tests, or explain an unfamiliar snippet — often well, and always instantly.

A concrete GopherTrunk example: imagine you open the down-converter code in `internal/scanner/ccdecoder/ddc.go` and you're new to digital signal processing. You don't understand what the filter taps or the decimation factor are doing. You can paste that chunk into a coding assistant and ask, "Explain what this Go code does, step by step." The model has seen enough DSP code and enough plain-English explanations of it to produce a clear walkthrough — naming the pattern, relating it to standard filtering, and pointing out what each variable is for. That is the everyday superpower: turning unfamiliar patterned text into something you can read.

## Why it's fallible

The same mechanism that makes LLMs powerful is the source of their biggest weakness. The model predicts the *plausible* continuation, not the *correct* one — and plausible is not the same as true.

Ask it about a function that doesn't exist and it may cheerfully invent one, because an invented call can look exactly like a real one. Ask about a library and it may describe a version that never shipped, because that description fits the pattern of how such things are written. These confident, well-formatted mistakes are called **hallucinations**, and they are not bugs to be patched away — they fall straight out of "predict the likely next token." We'll see why in the [next lesson](/learn/ai-software-dev/how-models-are-trained/), and what to do about it in [Verification and trust](/learn/ai-software-dev/verification-and-trust/).

For that GopherTrunk DDC explanation, the model's account might be 90% right and quietly wrong about one constant. It looked authoritative because looking authoritative is what it's good at. Your job is to treat its output as a strong draft to verify, never as a citation.

## What this path covers

Now that you know what an LLM is, the rest of this path builds out from it: how models are [trained](/learn/ai-software-dev/how-models-are-trained/) and how they [decide](/learn/ai-software-dev/how-models-decide/) what to say; the [context window](/learn/ai-software-dev/context-windows/) that bounds what they can see; the [types of models](/learn/ai-software-dev/types-of-models/) and the [provider landscape](/learn/ai-software-dev/the-provider-landscape/); the tools you'll actually use, from the chat app to IDE plugins to command-line agents; and the skills — prompting, context, verification, and judgment — that turn a fallible predictor into a reliable assistant.

<div class="knowledge-check" data-quiz data-correct-msg="Right — at its core an LLM predicts the next token from learned statistical patterns, not from human-style understanding." markdown="0">
  <p class="knowledge-check__q">Quick check: What is the core thing a Large Language Model actually does?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It reasons about your problem the way a human expert would</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It predicts a likely next token from statistical patterns it learned from text and code</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It looks up the correct answer in a verified database of facts</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **AI for coding means the LLM** — the dominant tool is the Large Language Model, a machine-learning system that learned patterns from data rather than running hand-written rules.
- **Narrow, not general** — today's tools are narrow task AI; AGI does not exist, and treating the tool as a pattern engine keeps your expectations honest.
- **Next-token prediction** — an LLM's one job is to estimate the likely next token and repeat, modelling the statistical patterns of language rather than understanding meaning.
- **Powerful for code** — code is highly patterned text with plenty of examples to learn from, so prediction works unusually well there.
- **Fallible by design** — the model produces the plausible continuation, not the guaranteed correct one, which is exactly why it hallucinates and why you verify its output.

Next up: where all those patterns come from — the pipeline that turns raw text and code into a model that follows your instructions. See [How AI models are trained](/learn/ai-software-dev/how-models-are-trained/).
