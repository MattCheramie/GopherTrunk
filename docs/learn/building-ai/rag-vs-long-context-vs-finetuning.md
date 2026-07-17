---
slug: rag-vs-long-context-vs-finetuning
title: RAG vs. long context vs. fine-tuning
description: Three ways to give a model knowledge or behaviour it didn't ship with — retrieval, a big prompt, or further training. Learn what each changes, what it costs, and how to pick by what changes and how often.
keywords: RAG vs fine-tuning, long context, fine-tuning, knowledge injection, when to fine-tune, context window, model customization, retrieval augmented generation, RAG vs long context, teach a model new facts
level: advanced
status: full
prereq:
  - grounding-and-retrieval
faq:
  - q: Does fine-tuning teach a model new facts?
    a: "Not reliably. Fine-tuning is good at shaping *behaviour* — style, tone, format, following a house pattern — but it's a poor way to inject facts. Facts get baked in fuzzily, are hard to update, and the model still hallucinates around the edges. When the goal is \"know this specific, current information,\" retrieval (RAG) is the right tool, not fine-tuning."
  - q: "When should I put everything in the prompt instead of building RAG?"
    a: "When the data is small enough to fit the context window and you don't need to scale — a single document, a policy page, one contract. Long context needs no pipeline, so it's the simplest option for one-off, whole-document tasks. Once the corpus is large, always changing, or queried constantly, the per-call cost and the \"lost in the middle\" effect push you toward RAG."
  - q: Can I combine these approaches?
    a: "Yes, and real systems often do. A common pattern is to fine-tune a model for a consistent output format or domain tone, then use RAG to feed it current, citable facts at query time. They solve different problems — behaviour versus knowledge — so they stack cleanly."
  - q: "Why not just fine-tune on all our docs so the model \"knows\" them?"
    a: "Because it bakes a snapshot into the weights: the moment a doc changes you'd have to retrain, the model blends facts together imprecisely, and it can't cite a source. RAG keeps your docs in a store you can update any time and hands the model the exact passage to answer from. \"Teach it our docs\" is almost always a retrieval problem."
---

# RAG vs. long context vs. fine-tuning

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Three ways to give a model knowledge or behaviour it didn't ship with. **RAG**
retrieves the right facts at query time. **Long context** stuffs everything into
one big prompt. **Fine-tuning** further-trains the model on your data. They aren't
rivals so much as tools for different jobs — pick by **what changes and how often**.
Facts that move → RAG. A small, fixed body of text → long context. A consistent
behaviour or format → fine-tune. This lesson builds on
[grounding & retrieval](/learn/building-ai/grounding-and-retrieval/).
</div>

Once you've grounded a model in your own data with [retrieval](/learn/building-ai/grounding-and-retrieval/),
a natural question follows: is RAG always the answer? Sometimes you could just
paste the data into the prompt. Sometimes people reach for fine-tuning. These
three approaches get mixed up constantly, and choosing the wrong one wastes
weeks. The good news is that they line up cleanly once you ask the right question.

## The three options

- **RAG (retrieval-augmented generation)** — keep your knowledge in a store, and
  at query time fetch the few relevant passages and put *those* in the prompt.
  The model answers from material you handed it.
- **Long context** — skip the pipeline and put *everything* into one large prompt.
  The whole document, the whole policy, the whole record set — if it fits the
  [context window](/learn/ai-software-dev/context-windows/), the model can use it.
- **Fine-tuning** — take an existing model and train it further on your own
  examples, nudging its parameters so its default behaviour shifts toward what
  you want.

The quickest way to keep them straight: RAG and long context change *what's in the
prompt*; fine-tuning changes *the model itself*.

## RAG

**RAG shines when your knowledge is large, changing, and fact-heavy** — support
Q&A, docs search, anything where the right answer lives in a corpus that grows and
updates. Because you fetch fresh material every query, answers stay **current**;
because you know which passage you pulled, they can be **citable**; and because you
only put a handful of relevant chunks in the prompt, each query is **cheaper** than
shipping the whole corpus.

The biggest win is updatability: change a document in your store and the next query
sees it — **no retraining**. The cost is engineering. You own a pipeline
([building a RAG pipeline](/learn/building-ai/building-a-rag-pipeline/)) and its
answers are only as good as its retrieval — get the wrong chunks and the model
answers from the wrong material ([chunking & retrieval quality](/learn/building-ai/chunking-and-retrieval-quality/)).

## Long context

**Long context is the simplest option when the data is small enough to fit.** There's
no pipeline, no store, no retrieval step — you just include the material in the prompt
and ask. For a single document, one contract, or a page of policy, that's often all
you need, and there's nothing to break.

The limits show up as the data grows. Every call re-sends the whole payload, so
**cost and latency climb** with the amount of text. Models also suffer from **"lost in
the middle"** — a fact buried in the centre of a very long prompt is used less
reliably than one near the start or end. And it simply **doesn't scale**: you can't
fit a large, ever-growing corpus into a window, no matter how big the window gets.
Reach for long context for one-off, whole-document tasks; reach for RAG when the body
of knowledge is bigger than a prompt. (See the sibling lesson on
[context windows](/learn/ai-software-dev/context-windows/) for how big "big" really is.)

## Fine-tuning

**Fine-tuning changes behaviour, not knowledge.** By training the model further on
your examples, you shift its defaults — its **style, tone, and output format**, or a
domain-specific pattern it should follow every time without being told. If you need
the model to always answer in a particular voice or emit a particular structure,
fine-tuning teaches that habit.

What it is **not** is a reliable way to inject facts. Training bakes information into
the weights fuzzily: facts blend together, you **can't update** one without retraining,
there's no source to cite, and the model still **hallucinates** around anything it only
half-absorbed. Fine-tuning also has real costs the other two don't — you need a
curated dataset and the effort to train and evaluate. It's the same machinery covered
in [how AI models are trained](/learn/ai-software-dev/how-models-are-trained/), just
applied to your own data on top of a finished model. Use it to shape *how* the model
responds, and let RAG handle *what* it knows.

## A decision guide

The choice comes down to what you're actually trying to change, and how often it moves.

| You need… | Best default | Why |
|-----------|--------------|-----|
| Facts that change or grow | **RAG** | Current, citable, updatable without retraining |
| Small, fixed data or a one-off task | **Long context** | No pipeline; just include it in the prompt |
| Consistent behaviour, format, or tone | **Fine-tuning** | Bakes the *habit* into the model |

And they combine. A very common production shape is to **fine-tune for format** —
so every answer comes out in your house structure — while using **RAG for facts** —
so the content is current and grounded. They solve different problems, so stacking
them is normal, not a compromise.

## Common mistake

The classic misstep is reaching for fine-tuning to "teach the model our docs." It
feels like the powerful, permanent option — but it bakes a snapshot into the weights,
can't be updated when a doc changes, blends facts imprecisely, and still hallucinates.
"Get the model to answer from our knowledge" is a **retrieval** problem almost every
time. Fine-tune when you want to change *how* it behaves; use RAG when you want to
change *what* it knows.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a large, frequently-updated knowledge base is exactly what retrieval is for: fetch current facts per query, no retraining." markdown="0">
  <p class="knowledge-check__q">Quick check: You need answers from a large, frequently-updated knowledge base. Best default?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Fine-tune the model on the whole knowledge base</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">RAG — retrieve current facts at query time</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Paste the entire knowledge base into every prompt</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Three ways to add knowledge or behaviour: **RAG** (retrieve at query time),
  **long context** (put it all in the prompt), **fine-tuning** (train the model further).
- **RAG** fits large, changing, fact-heavy knowledge — current, citable, cheaper per
  query, updatable without retraining; the cost is pipeline and retrieval quality.
- **Long context** is simplest for small, fixed, one-off data, but per-call cost,
  "lost in the middle," and no scaling rule it out for big corpora.
- **Fine-tuning** changes behaviour, style, and format — it is **not** a reliable way
  to inject facts.
- Pick by **what changes and how often**; facts that move → RAG, small fixed data →
  long context, consistent behaviour → fine-tune — and they **combine**.
- The common mistake: fine-tuning to "teach it our docs" when RAG is the right tool.

Next up: Module 5 lets the model act — [agents: letting the model act in a loop](/learn/building-ai/what-is-an-ai-agent/).
