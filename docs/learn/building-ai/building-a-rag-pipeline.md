---
slug: building-a-rag-pipeline
title: Building a RAG pipeline
description: A RAG feature is two phases wired into a pipeline — offline indexing that ingests, chunks, embeds, and stores your documents, and an online query path that retrieves the closest chunks, augments the prompt, and generates a grounded, cited answer.
keywords: RAG pipeline, ingest, chunk, embed, retrieve, augmented prompt, citations, indexing, retrieval, grounded answer, vector index, top-k
level: advanced
status: full
prereq:
  - embeddings-and-vector-search
faq:
  - q: "What are the two phases of a RAG pipeline?"
    a: "Indexing and query. Indexing runs offline, once or on a schedule: it ingests your documents, chunks them, embeds each chunk, and stores the vectors. The query path runs online, per request: it embeds the user's question, retrieves the closest chunks, augments the prompt, and generates the answer. Keeping the two separate is the whole design."
  - q: "What actually goes into the final prompt?"
    a: "Your system instructions, the retrieved chunks labeled as context, and the user's question. The model is told to answer only from the provided context. It's the same context-assembly job from the earlier lesson, with the context filled in automatically by retrieval instead of by hand."
  - q: "How does RAG reduce hallucination?"
    a: "By grounding. You retrieve real passages from your own corpus and instruct the model to answer only from them, and to say it doesn't know when the answer isn't there. The model is steered toward quoting supplied text rather than inventing from training, and citations let a user verify each claim."
  - q: "Why return citations?"
    a: "Because a grounded answer you can't check is still a leap of faith. Returning the source chunks — article, system, date — lets users confirm the answer against the original and builds trust in the feature. It's a small addition with an outsized payoff."
gophertrunk_links:
  - title: Architecture
    url: /architecture.html
    note: the decoders and reference material a GopherTrunk assistant would index.
---
# Building a RAG pipeline

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A RAG feature is two phases wired into a pipeline. **Offline indexing** —
**ingest / chunk / embed / store** your documents into a vector index, done once
or on a schedule. **Online query** — **retrieve / augment / generate**: embed the
question, pull the closest chunks, drop them into the prompt as labeled context,
and let the model answer from them. Builds on
[embeddings & vector search](/learn/building-ai/embeddings-and-vector-search/) and
[grounding & retrieval](/learn/building-ai/grounding-and-retrieval/).
</div>

You've seen the pieces — embeddings turn text into vectors, similarity search finds
the nearest ones, and grounding means answering from supplied passages. This lesson
assembles them into the standard shape of a **retrieval-augmented generation (RAG)**
feature: a pipeline with two clearly separated halves. Get the split right and the
rest is plumbing.

## Two phases

The single most useful idea in RAG is that it is **two phases, not one**.

- **Indexing** is **offline**. It runs once when you first build the feature, and
  again periodically as your documents change. It touches every document but never
  sees a user question. Its whole job is to leave behind a searchable index.
- **Query** is **online**. It runs **per request**, once for each question a user
  asks. It never re-reads the whole corpus — it consults the index the offline
  phase already built.

Blur the two together and everything gets slow and confusing. Keep them apart and
each half becomes simple: one prepares the haystack, the other pulls the needle.

## The indexing phase

Indexing turns a pile of raw documents into a vector index you can search. Four
steps, in order:

1. **Ingest.** Load the source material — articles, records, notes, whatever the
   feature answers about. This is where you normalise formats and strip boilerplate
   so only real content moves forward.
2. **Chunk.** Split each document into smaller passages. A whole document is usually
   too coarse to retrieve well, so you break it into paragraph-sized pieces. *How*
   you chunk turns out to matter a great deal — that's the
   [next lesson](/learn/building-ai/chunking-and-retrieval-quality/).
3. **Embed.** Run each chunk through an embedding model to get its vector, the same
   operation you'll later apply to the question so the two live in one space.
4. **Store.** Write each vector into a **vector index**, alongside **metadata** —
   the source document, a date, a title, a link. That metadata is what later lets
   you cite where an answer came from.

Do this once and you have a durable index. You only revisit it when the underlying
documents change.

## The query phase

Now a user asks something. The online path is also four steps:

1. **Embed** the user's query with the *same* embedding model used at index time.
2. **Retrieve** the **top-k** most similar chunks from the vector index — the handful
   whose vectors sit closest to the question's.
3. **Augment** the prompt: assemble your system instructions, the retrieved chunks
   as **labeled context**, and the user's question into one prompt. This is exactly
   the context-assembly job from
   [feeding the model context & user input](/learn/building-ai/context-and-user-input/) —
   retrieval just fills the context in for you.
4. **Generate** the answer. The model reads the assembled prompt and responds from
   the supplied passages.

The corpus is never held in context all at once. Only the few retrieved chunks ever
reach the model.

## Grounded-answer prompting

Retrieval gets the right passages in front of the model, but the *prompt* is what
makes it use them. Two instructions do most of the work:

> Answer only using the provided context. If the answer isn't in the context, say
> you don't know.

That single rule is the biggest lever you have against invented answers — it steers
the model to quote supplied text rather than fill gaps from training. Pair it with a
request to **cite the source chunks** it relied on. Together they are what turns a
plausible-sounding generator into a trustworthy one; more on the failure modes in
[handling hallucinations & failure](/learn/building-ai/handling-hallucinations-and-failure/).

## Citations & attribution

Because you stored metadata at index time, you can return the sources behind every
answer — the article title, the record, the date. Surfacing them lets a user open
the original and confirm the claim for themselves. That verifiability is a large
trust win for a small amount of work: it turns "trust me" into "here's where I got
it," and it makes wrong answers *catchable* instead of silent.

## A sketch

The whole feature in pseudo-code — two functions, one per phase:

```
# offline, run once and on document changes
def index(documents):
    for doc in documents:
        for chunk in chunk(doc):
            vector = embed(chunk)
            store(vector, text=chunk, meta=doc.source_and_date)

# online, run per request
def answer(query):
    q = embed(query)
    chunks = retrieve(top_k=5, near=q)          # nearest passages
    prompt = system_instructions + label(chunks) + query
    reply = generate(prompt)                     # grounded answer
    return reply, sources_of(chunks)             # answer + citations
```

`index()` prepares the haystack offline; `answer()` pulls the needle online. The
only thing they share is the vector store and the same `embed()` function.

## Where it breaks

A RAG answer is only ever as good as what retrieval hands the model. If the right
passage isn't in the top-k, no amount of clever prompting recovers it — the model
simply never sees it. That makes **retrieval quality** the pipeline's ceiling, and
it's why the [next lesson](/learn/building-ai/chunking-and-retrieval-quality/) is
entirely about chunking and retrieval. It's also why you can't eyeball whether a RAG
feature is good; you need to measure it, which is the job of
[evaluating AI features](/learn/building-ai/evaluating-ai-features/).

## A GopherTrunk example

Picture a "search my logged systems and the field guide" assistant. **Offline**, you
index two corpora: the reference articles (how trunking works, protocol notes) and
the user's own saved systems and site records. Each gets chunked, embedded, and
stored with metadata pointing back to its source. **Online**, when the user asks
"why won't the Mt Anakie control channel lock?", you embed that question, retrieve
the closest chunks — maybe a paragraph on control-channel SNR plus the user's own
notes on that site — augment the prompt with them, and generate an answer that cites
both. The user gets a grounded reply *and* a link to verify it, from a body of
material far too large to paste into any prompt.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the final prompt carries your instructions, the retrieved chunks as labeled context, and the user's question." markdown="0">
  <p class="knowledge-check__q">Quick check: in the query phase, what goes into the final prompt sent to the model?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The entire indexed corpus, embedded into one vector</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The user's question plus the retrieved chunks as labeled context</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Only the user's question — the model retrieves on its own</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A RAG feature is **two phases**: offline **indexing** and online **query**, wired
  into one pipeline.
- **Indexing** = **ingest → chunk → embed → store** your documents in a vector index
  with metadata, done once or on a schedule.
- **Query** = **embed → retrieve → augment → generate**: embed the question, pull the
  top-k chunks, build the prompt, answer from them.
- **Grounded-answer prompting** — "answer only from the context; say you don't know
  otherwise" — is your main defence against hallucination.
- **Citations** return the sources used, so users can verify the answer — a big trust
  win for little effort.
- The answer is only as good as **retrieval**, which is where the next lessons focus.

Next up: [chunking & retrieval quality](/learn/building-ai/chunking-and-retrieval-quality/).
