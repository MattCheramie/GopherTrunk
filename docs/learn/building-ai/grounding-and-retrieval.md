---
slug: grounding-and-retrieval
title: Why retrieval — grounding the model in your data
description: A model doesn't know your documents, product, or records, and will confidently invent answers about them. Learn how grounding and retrieval-augmented generation (RAG) feed the model your real facts at query time so it answers from truth, not memory.
keywords: retrieval augmented generation, RAG, grounding, hallucination, knowledge base, source of truth, context injection, up to date answers, vector search, question answering
level: intermediate
status: full
prereq:
  - prompts-as-code
faq:
  - q: What is retrieval-augmented generation (RAG)?
    a: "RAG is a pattern where, for each question, the system retrieves the most relevant pieces of your own data and adds them to the prompt before the model generates an answer. The model answers from those retrieved facts instead of from its training memory, which keeps answers current and specific to your content."
  - q: Why does retrieval reduce hallucination?
    a: "A model with no relevant facts in front of it will still produce a confident, fluent answer — that guess is a hallucination. Retrieval puts the real facts into the prompt, so the model has something true to answer from and can be told to say it doesn't know when the facts aren't there."
  - q: When should I not bother with retrieval?
    a: "Skip it for general tasks the model already handles from training — summarising text you paste in, rewriting, translation, or answering common-knowledge questions. Retrieval earns its place when answers must come from your specific documents, product, or records that the model was never trained on."
  - q: Is retrieval the same as retraining the model?
    a: "No. Retraining or fine-tuning changes the model's weights and is slow and expensive. Retrieval leaves the model untouched and simply feeds it fresh facts at query time, so you update answers by updating your data — not by rebuilding the model."
---
# Why retrieval — grounding the model in your data

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A model doesn't know your data — it answers from training plus whatever is in the prompt, so ask about your own documents and it will **hallucinate**, inventing a confident answer that sounds right. **Grounding** fixes this by putting the real facts into the prompt. **Retrieval (RAG)** automates grounding: for each question it fetches the most relevant pieces of your data and adds them before the model answers, so replies stay current, specific, and traceable to a source.
</div>

You've written [prompts as code](/learn/building-ai/prompts-as-code/) and can shape how the model behaves. But shaping behaviour doesn't give the model *facts* it never had. The moment a feature needs to answer from your documents, your product, or your records, you hit a wall that better prompting alone can't get past. Retrieval is how you get over it.

## The knowledge problem

A model knows two things and nothing else: the patterns it absorbed during training, and whatever you put in the prompt. Training data is broad but **general and stale** — it's a snapshot of public text, frozen at some cutoff, and it never contained your internal wiki, your customer records, or last week's release notes.

So when you ask about *your* material, the model has no source to draw from. It doesn't go quiet — that's not how a [language model as a component](/learn/building-ai/llm-as-a-component/) works. It produces the most plausible-sounding continuation it can, which means it **guesses**, and a fluent guess about facts it doesn't have is a hallucination. The answer reads as authoritative and is simply made up.

You can't prompt your way out of this. No instruction like "only tell the truth" helps, because the model has no truth to tell — the facts aren't in front of it.

## Grounding: give it the facts

**Grounding** is the fix, and it's exactly as simple as it sounds: put the facts the model needs *into the prompt*, so it answers from them instead of from memory. Paste the relevant policy paragraph, the product spec, the record — and now the question isn't "what do you recall about this?" but "here are the facts; answer from them."

A grounded model stops guessing because it no longer has to. Given the right passage, it reads and reasons over text you supplied, which is something models are genuinely good at. The remaining problem is purely mechanical: **how do you know which facts to paste** when your data is thousands of documents and every question needs a different slice?

## Retrieval-augmented generation (RAG)

You automate the grounding step. **Retrieval-augmented generation (RAG)** means: for each incoming question, *retrieve* the most relevant pieces of your data, add them to the prompt, then let the model *generate* the answer from them. That's the whole idea, and the name spells out the two halves — retrieval (find the facts) augmenting generation (produce the answer).

Instead of you hand-picking passages, a retrieval step searches your data for the chunks most relevant to *this* question and injects them automatically. The model still just answers from what's in the prompt — the win is that the right facts get there on their own, every time, at whatever scale your data reaches.

## Why it's the workhorse

RAG is the default way real features ground themselves in private data, and it's worth being clear about why it wins so often:

- **Current.** When your data changes, you update the data — not the model. New facts are available to the next question immediately, with no rebuild.
- **Private.** Your documents stay in your store as your **source of truth**. You retrieve from them at query time rather than baking them into a model, so the data stays yours and under your control.
- **Citable.** Because you know which chunks you retrieved, you can return them as **sources** — the answer comes with a trail back to the document it came from.
- **Grounded.** Feeding real facts in cuts hallucination directly: the model has something true to answer from, and you can instruct it to defer when it doesn't.

And it's far **cheaper and faster** than the alternative of retraining the model on your data every time that data changes — a trade-off we'll weigh in full in [retrieval vs. long context vs. fine-tuning](/learn/building-ai/rag-vs-long-context-vs-finetuning/).

## The basic shape

Every RAG feature, however elaborate, has the same skeleton:

1. **Query** — the user's question arrives.
2. **Retrieve** — search your data for the chunks most relevant to that question.
3. **Inject** — add those chunks to the prompt as clearly [labelled context](/learn/building-ai/context-and-user-input/), kept separate from the question itself.
4. **Generate** — the model answers *from the provided context, or says it doesn't know* when the context doesn't cover it.

That last instruction matters: grounding gives the model the option to admit the facts aren't there, which is exactly the behaviour you want over made-up ones. Building each of these steps for real — chunking, the retrieval index, the prompt assembly — is the job of [building a RAG pipeline](/learn/building-ai/building-a-rag-pipeline/) later in the path.

## When you need it (and when you don't)

Reach for retrieval when the answer must come from **your** content:

- A support assistant answering from your help docs and policies.
- Question-answering over an internal wiki, codebase, or knowledge base.
- An assistant that must cite the record or document behind each answer.

Skip it when the task doesn't depend on private facts. Summarising text you already pasted in, rewriting a paragraph, translating, or answering general-knowledge questions are things the model handles from training alone — bolting retrieval onto them just adds cost and moving parts for no gain. Retrieval is the answer to "the model doesn't know *my* data," not a tax on every call.

<div class="knowledge-check" data-quiz data-correct-msg="Right — retrieval fetches your current, specific facts into the prompt at query time, so the model answers from truth instead of guessing." markdown="0">
  <p class="knowledge-check__q">Quick check: RAG mainly improves answers because it...</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Retrains the model on your data before every question</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Feeds the model your current, specific facts at query time, reducing hallucination</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Makes the model's training data larger and more general</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **A model doesn't know your data** — it answers from general, stale training plus the prompt, and asked about your material it will **hallucinate** a confident guess.
- **Grounding** means putting the needed facts into the prompt so the model answers from them, not from memory.
- **Retrieval-augmented generation (RAG)** automates grounding: retrieve the most relevant chunks per question, inject them, then generate.
- **It's the workhorse** because it stays current, keeps your data as your source of truth, can cite sources, cuts hallucination, and is far cheaper than retraining.
- **The shape** is query → retrieve → inject as labelled context → answer from the context or say you don't know.
- **Use it** for Q&A, support, and assistants over your own content; **skip it** for general tasks the model already handles.

Next up: [embeddings & vector search](/learn/building-ai/embeddings-and-vector-search/).
