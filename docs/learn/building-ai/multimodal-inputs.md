---
slug: multimodal-inputs
title: Multimodal — images, audio & documents
description: Some models accept more than text — images, audio, and documents alongside your words. Learn what multimodal input unlocks, how you send it, what it costs in tokens, and why extracted values still need checking.
keywords: multimodal, vision model, image input, audio input, OCR, document understanding, speech to text, image tokens, PDF input, extract fields
level: intermediate
status: full
prereq:
  - how-a-model-api-works
faq:
  - q: "What does multimodal mean?"
    a: "It means a model can accept inputs other than text — most commonly images, and increasingly audio and documents — alongside your words. The output is usually still text (or structured data), but the model can now see or hear what you send, not just read it."
  - q: "Do images and audio cost tokens?"
    a: "Yes. An image or an audio clip is converted into tokens the model reads, so it counts toward your input cost and the context window just like text does. A large or high-resolution image can use a surprising number of tokens, so it is a real design constraint, not a free add-on."
  - q: "Can I trust text a model extracts from a photo or scan?"
    a: "Treat it as a probabilistic draft, not a source of truth. Multimodal reading is genuinely useful, but quality varies with fine print, low resolution, and accents, and the model can still be confidently wrong. Verify extracted values before you rely on them."
---

# Multimodal — images, audio & documents

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Some models are **multimodal** — they accept **images, audio, and documents**
alongside text, not just words. That unlocks features like image understanding,
OCR, and transcription from the same
[model API](/learn/building-ai/how-a-model-api-works/) you already know. The
output is usually still text, and it is still probabilistic — so treat extracted
values as a draft to verify, not a fact. Not every model can do this; see the
sibling [types of models](/learn/ai-software-dev/types-of-models/).
</div>

So far this path has treated a model as a thing you send text to and get text
back from. But many models now accept more than text, and that opens a set of
features you can't build with words alone. This lesson covers what those inputs
are, how you send them, and where the sharp edges are.

## Beyond text

A **multimodal** model takes inputs beyond plain text. The most common and most
mature is **images** — a photo, a screenshot, a chart, a scan. Increasingly,
models also accept **audio** (a voice clip or recording) and whole
**documents** such as a PDF, handed over as a file rather than pasted-in text.

What comes *back* is usually still text — a description, an answer, a
transcript — or, when you ask for it, [structured
output](/learn/building-ai/structured-output/) your program can use directly.
The extra modality is on the *input* side; the model reads or hears more, then
responds in the familiar way.

## What you can build

Concretely, multimodal input lets you:

- **Describe or analyze an image** — hand the model a photo and ask what it
  shows, or what looks wrong with it.
- **Extract text or fields from a photo or scan (OCR)** — pull the numbers off a
  receipt, the readings off a display, or the fields off a form.
- **Answer questions about a chart or document** — "what's the trend here?" over
  a graph, or "what does clause 4 say?" over a PDF.
- **Transcribe audio** — turn a spoken clip into text you can store or search.
- **Reason over audio or a document** — summarize a recording, or compare two
  scanned pages, rather than just converting them.

Each of these would be hard or impossible with a text-only model; the input
modality is what makes the feature possible.

## How you send them

You send non-text inputs as **parts of a message**. Instead of a message whose
content is one string, you build a message whose content is a list: some text,
plus an image, audio clip, or file. An image is typically provided either as
**data** (the bytes, encoded) or as a **URL** the provider can fetch; audio and
documents work the same way, as attached parts.

The important thing to internalize is that these parts **consume tokens and cost
money** too. An image is converted into tokens the model reads, and a large or
high-resolution one can use many of them; audio and documents are no different.
So the [cost, latency, and limits](/learn/building-ai/cost-latency-and-limits/)
you learned to design around apply here as well — a picture is not a free
addition to a prompt.

## Limits & cautions

A few things to keep straight before you lean on this:

- **Not every model is multimodal.** Support is a property of the specific model
  you call, not a given. Check the sibling [types of
  models](/learn/ai-software-dev/types-of-models/) lesson for where multimodal
  and vision fit.
- **Quality varies with the input.** Fine print, low resolution, poor lighting,
  handwriting, and unfamiliar accents all degrade results. Clean inputs read far
  better than messy ones.
- **The output is still probabilistic.** A model reading an image is the same
  [probabilistic component](/learn/building-ai/llm-as-a-component/) as one
  reading text — it can be confidently wrong. A misread digit looks exactly like
  a correct one.
- **Verify before trusting extracted values.** Anything you pull out — a total,
  a serial number, a field — should go through the same [parsing and
  validation](/learn/building-ai/parsing-and-validating-output/) as any other
  model output before your code acts on it.

## A GopherTrunk example

Imagine feeding a model a **screenshot of a waterfall or spectrum display**, or
a **constellation plot**, and asking it to describe the features it can see or
to draft a plain-English explanation of what's on screen. That's a genuinely
handy assistive aid — a way to get a second, verbal read on a picture — and it
plays to a multimodal model's strengths.

But frame it honestly: it is an *assistant*, not an *authoritative decoder*. The
real decoding is done by GopherTrunk's DSP, which measures the signal precisely.
A model glancing at a screenshot might call a carrier "clean" that a proper
measurement shows is degraded. Use the description to orient yourself, then trust
the instrument for the numbers.

<div class="knowledge-check" data-quiz data-correct-msg="Right — multimodal input means the model can accept non-text inputs like images or audio, not just read text." markdown="0">
  <p class="knowledge-check__q">Quick check: what does "multimodal input" mean?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The model always replies with images and audio as well as text</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Accept non-text inputs like images or audio, not just text</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The model runs on your own machine instead of a provider's</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **multimodal** model accepts inputs beyond text — images most maturely, and
  increasingly audio and documents.
- The **output is usually still text** (or structured data); the extra modality
  is on the input side.
- It unlocks real features: describing images, **OCR**, answering questions over
  charts and documents, and transcribing audio.
- You send non-text inputs as **parts of a message**, and they **cost tokens**
  like everything else you send.
- Not every model supports it, quality varies with the input, and the output is
  still probabilistic — **verify extracted values** before trusting them.

Next up: Module 4 grounds the model in your own data — [why retrieval](/learn/building-ai/grounding-and-retrieval/).
