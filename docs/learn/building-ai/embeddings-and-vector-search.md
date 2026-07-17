---
slug: embeddings-and-vector-search
title: Embeddings & vector search
description: An embedding model turns text into a vector so you can search by meaning instead of exact words. Learn how embeddings work, how nearest-neighbour search retrieves the closest documents, and where vector databases and indexes fit in.
keywords: embeddings, vector search, semantic search, vector database, cosine similarity, nearest neighbor, pgvector, FAISS, embedding model, top-k retrieval, ANN
level: intermediate
status: full
prereq:
  - grounding-and-retrieval
faq:
  - q: "What is an embedding?"
    a: "An **embedding** is a **vector** — a list of numbers — that a model produces from a piece of text so that texts with similar meaning sit close together in that numeric space. Comparing vectors lets you find related text by meaning rather than by matching exact words."
  - q: "How is vector search different from keyword search?"
    a: "Keyword search (like BM25) matches the actual words in your query against the words in each document. **Vector search** compares meanings, so it can find a relevant passage that shares no words with your query. Many systems combine both, which is called hybrid search."
  - q: "Do I need a special database for this?"
    a: "Not necessarily. You can store vectors in a general database with an extension such as pgvector in Postgres, in a library like FAISS, or in a dedicated vector database. What they share is a fast approximate-nearest-neighbour index so you can retrieve the closest vectors without comparing every one."
  - q: "Can I mix embedding models?"
    a: "No — use the **same embedding model** for your documents and your queries, because different models place text in different, incompatible spaces. If you switch models, you have to re-embed everything so the old and new vectors are comparable."
---

# Embeddings & vector search

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **embedding** turns a piece of text into a **vector** — a list of numbers
positioned so that similar meanings land close together. Compare vectors and you
get **semantic search**: retrieval by meaning, not exact keywords. The vectors
live in a **vector database** built for fast nearest-neighbour lookup. This is the
engine under [grounding & retrieval](/learn/building-ai/grounding-and-retrieval/) —
the mechanism that actually finds the right material to hand a model.
</div>

The last lesson said retrieval means finding the relevant material and putting it in
front of the model. This lesson opens the box and shows *how* that finding happens.
The trick is to convert text into numbers in a way that captures meaning, so that
"finding relevant text" becomes "finding nearby points."

## What an embedding is

An **embedding model** takes a chunk of text and outputs a **vector** — a fixed-length
list of numbers. The numbers aren't random: the model is trained so that texts with
similar *meaning* produce vectors that are close together, and unrelated texts produce
vectors that are far apart.

Think of each vector as a point in space. Two sentences that say the same thing in
different words land near each other; a sentence about something else lands far away.
You never read the numbers yourself — they're only useful for measuring distance
between texts. That single property, *meaning becomes proximity*, is what everything
else in this lesson is built on.

## Searching by meaning

Here's the move. You **embed your documents once**, ahead of time, and store the
vectors. Then, when a question comes in, you **embed the query** with the same model
and look for the document vectors closest to it. A common way to measure "close" is
**cosine similarity**, which compares the direction two vectors point. The closest
documents are your results.

Because closeness tracks meaning, this returns relevant passages **even when they share
no words with the query** — ask about "losing signal lock" and it can surface a passage
about "the decoder dropping sync." That's the payoff over **keyword search** (such as
**BM25**), which matches the literal words and would miss it.

Neither approach is strictly better, so many systems run **hybrid search**: combine the
keyword score and the vector score so exact terms and overall meaning both count.

## Vector databases & indexes

Comparing your query to every stored vector one by one works for a few thousand
documents but gets slow fast. A **vector database** solves this by building an index for
**approximate-nearest-neighbour (ANN)** search — a structure that finds the closest
vectors quickly without checking all of them, trading a tiny bit of accuracy for a big
speed-up.

You have options, and they're all the same idea underneath:

- **pgvector** — an extension that adds vector columns and ANN indexes to Postgres, so
  your vectors live next to your regular data.
- **FAISS** — a library for building an in-process vector index, good when you want
  vectors in your own application rather than a separate service.
- **Dedicated vector databases** — standalone services built specifically for storing
  and querying vectors at scale.

Whatever you pick, a query asks for the **top-k** results — the *k* nearest vectors,
say the top 5 — which become the material you pass along to the model.

## The workflow

Two phases, and it helps to keep them separate. **Indexing** happens offline, once (or
whenever documents change). **Querying** happens online, every time someone asks.

```text
INDEX  (offline, once)
  document ─▶ chunk into passages ─▶ embed each chunk ─▶ store vectors in the index

QUERY  (online, per request)
  question ─▶ embed the question ─▶ search the index ─▶ return top-k nearest chunks
```

The indexing phase is the expensive, slow part, and you pay it up front. The query
phase is cheap: one embedding call and one fast index lookup. Both phases must use the
**same embedding model**, or the query vector and the document vectors won't live in
the same space.

## Practical notes

- **The embedding model is separate and cheap.** Embeddings come from a different,
  much smaller and cheaper model than the chat model that writes the answer — see the
  sibling lesson on [types of models](/learn/ai-software-dev/types-of-models/). You call
  it a lot (every chunk, every query), so its low cost matters.
- **Budget for cost and storage.** Embedding a large corpus is many model calls, and
  every vector takes space. Neither is huge, but both scale with your document count, so
  plan for them.
- **Keep the model consistent.** Use the same embedding model for documents and queries.
  Mixing models compares points from incompatible spaces and returns nonsense.
- **Re-embed when you change models.** Switching to a newer embedding model means
  re-embedding your whole corpus so old and new vectors are comparable again.

## Limits

Semantic search is powerful, but it isn't the right tool for everything. Because it
matches meaning rather than exact strings, it can **miss exact-match needs** — a
specific part number, an error code, a talkgroup ID — where you want the literal token,
not something "similar." **Hybrid search** helps here by keeping a keyword path
alongside the vector path.

And even perfect search can only return what you fed it. If your documents were chopped
into unhelpful pieces, retrieval surfaces unhelpful pieces. Chunking quality is its own
topic — we cover it in [chunking & retrieval quality](/learn/building-ai/chunking-and-retrieval-quality/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — embeddings place similar meanings close together, so search returns results by meaning rather than exact words." markdown="0">
  <p class="knowledge-check__q">Quick check: Embeddings let you search by...</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Exact keyword matches only</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Meaning / semantic similarity, not exact keywords</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The date a document was created</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **embedding model** turns text into a **vector** so that similar meanings sit
  close together.
- **Semantic search** embeds the query and returns the nearest document vectors
  (e.g. by **cosine similarity**) — results by meaning, not exact keywords.
- A **vector database** (pgvector, FAISS, or a dedicated service) stores vectors and
  runs fast **approximate-nearest-neighbour** search to fetch the **top-k** matches.
- **Index offline** (chunk, embed, store); **query online** (embed, search, return
  top-k) — always with the same embedding model on both sides.
- Semantic search can miss exact-match needs, so **hybrid search** keeps a keyword path
  too, and retrieval quality still depends on how you chunked.

Next up: [building a RAG pipeline](/learn/building-ai/building-a-rag-pipeline/).
