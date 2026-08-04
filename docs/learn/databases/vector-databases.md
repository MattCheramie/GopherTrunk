---
slug: vector-databases
title: Vector databases & similarity search
description: The database behind modern AI retrieval — it stores embeddings as high-dimensional vectors and finds the nearest ones, so you can search by meaning instead of exact keywords. Learn how vectors, distance, and approximate nearest-neighbor search power RAG.
keywords: vector database, similarity search, embeddings, nearest neighbor, ANN, semantic search, RAG, retrieval augmented generation, cosine similarity, vector index, pgvector, high-dimensional vectors
level: intermediate
status: full
prereq:
  - sql-vs-nosql
faq:
  - q: What does a vector database actually store?
    a: "Embeddings — lists of numbers (vectors) that a model produces to represent the *meaning* of a piece of text, an image, or other data. Similar meanings get vectors that sit close together in space. The database stores these vectors alongside a reference to the original item, and its core job is finding the vectors nearest to a query vector."
  - q: How is similarity search different from a normal WHERE clause?
    a: "A WHERE clause matches exact or pattern-based conditions — this word appears, this value equals that. Similarity search matches by *closeness in meaning*: it finds the items whose vectors are nearest to your query's vector, so 'radio signal fading' can match a document about 'weak reception' with no shared keywords. It's search by meaning rather than by literal text."
  - q: Why not just store vectors in a regular table?
    a: "You can store them, but finding the nearest vectors by brute force means comparing your query against every stored vector — far too slow at scale. Vector databases (and vector extensions like pgvector) add a specialized index for approximate nearest-neighbor search that finds the closest matches quickly without checking every one."
---

# Vector databases & similarity search

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **vector database** stores **embeddings** — lists of numbers that capture the
*meaning* of text or other data — and its core job is **similarity search**: given a
query vector, find the stored vectors **nearest** to it. That lets you search by
**meaning instead of keywords**, and it's the retrieval engine behind **RAG**
(retrieval-augmented generation). It does this fast with an **approximate
nearest-neighbor** index. The vectors themselves come from the models covered in
[embeddings & vector search](/learn/building-ai/embeddings-and-vector-search/); this
lesson is about the *store* that holds and searches them.
</div>

Every store so far — relational, key-value, document, time-series — finds data by
**matching**: this key, this value, this condition. A vector database does something
different in kind. It finds data by **meaning**, returning the items most *similar* to
what you asked for even when they share no words with your query. It's the database
that appeared alongside the current wave of AI, and understanding it demystifies a lot
of how those systems work.

## Embeddings: meaning as coordinates

The idea rests on **embeddings**. A model can turn a piece of text (or an image, or
audio) into a **vector** — a fixed-length list of numbers, often hundreds or thousands
of them:

```
"weak radio reception"  →  [0.021, -0.44, 0.13, ..., 0.08]   (e.g. 768 numbers)
```

The magic is that the model arranges these numbers so that **similar meanings land
close together** in this high-dimensional space. "Weak radio reception" and "signal
fading" produce vectors that sit near each other; "chocolate cake recipe" lands far
away. The vector *is* a numerical fingerprint of meaning. (How models produce these is
the [embeddings & vector search](/learn/building-ai/embeddings-and-vector-search/)
lesson in the AI path — here we take the vectors as given.)

Once meaning is coordinates, "find things that mean something similar" becomes a
geometry problem: **find the nearest points**.

## Similarity search: nearest instead of equal

A vector database stores each embedding alongside a reference to its original item (the
document, the paragraph, the record). Its defining operation is: given a **query
vector**, return the **k nearest** stored vectors — a **k-nearest-neighbor** search.

"Nearest" needs a definition of distance. The common measures are **cosine similarity**
(the angle between two vectors — are they pointing the same way?) and **Euclidean
distance** (straight-line distance between the points). The details matter less than the
concept: the database ranks stored vectors by how close they are to your query and hands
back the top few.

Contrast that with a `WHERE` clause. SQL filtering asks "does this row *equal* or
*contain* this exact thing?" Vector search asks "which rows are *most like* this?" — a
ranked, fuzzy, meaning-based match with no requirement of shared words. That's why it's
called **semantic search**: search by sense, not by string.

## Why it needs a special database

Here's the catch that makes it a database problem rather than a one-liner. Finding the
true nearest vectors by brute force means comparing your query against **every** stored
vector and sorting — fine for a thousand vectors, hopeless for ten million. Each
comparison is real arithmetic across hundreds of dimensions.

So vector databases (Pinecone, Weaviate, Milvus, Qdrant, and the **pgvector** extension
that adds this to Postgres) build a specialized **index** for **approximate
nearest-neighbor (ANN)** search. Instead of checking every vector, the index cleverly
narrows the search to a promising neighborhood and returns *almost certainly* the
nearest matches, thousands of times faster. The word **approximate** is the trade: you
give up a tiny chance of missing the exact closest match in exchange for speed that
makes the whole thing usable at scale. For search-by-meaning, near-perfect and instant
beats perfect and unusable.

## RAG: what this is all for

The headline use is **retrieval-augmented generation (RAG)** — the standard way to give
a language model access to *your* documents. A model on its own only knows its training
data; RAG lets it answer from a knowledge base you control. The flow:

1. **Index once.** Split your documents into chunks, embed each chunk into a vector, and
   store the vectors in the vector database.
2. **Embed the question.** When a user asks something, turn *their question* into a
   vector with the same model.
3. **Retrieve.** Ask the vector database for the chunks whose vectors are nearest the
   question's vector — the passages most *relevant in meaning*.
4. **Generate.** Hand those retrieved chunks to the language model as context, and it
   answers grounded in your actual documents.

The vector database is step 3 — the retrieval engine. It's why a chatbot can answer from
your company's manuals, or why "how do I fix a fading signal" can surface a
troubleshooting note that never uses the word "fading." The
[building AI](/learn/building-ai/) path covers RAG as a whole; the piece it leans on
here is exactly this store.

## Where it fits alongside everything else

A vector database rarely replaces your other databases — it **joins** them. The source
of truth for your documents might be relational or a document store; the vector database
holds their embeddings for meaning-based retrieval; a cache sits in front of it all. In
fact, with an extension like **pgvector** you can add similarity search *to* a
relational database you already run, keeping vectors right next to the rows they
describe. That mirrors the whole unit's lesson: reach for the store whose shape matches
the job — and "search by meaning" is a job that wanted its own shape.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a vector database finds the stored embeddings nearest to a query vector, matching by meaning rather than by exact words." markdown="0">
  <p class="knowledge-check__q">Quick check: what does a vector database primarily do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Store rows and fetch them by an exact primary key, like a relational table</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Store embeddings and find the ones nearest a query vector — search by meaning, not exact words</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Compress time-stamped metrics for dashboards over time</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **vector database** stores **embeddings** — vectors of numbers that encode the
  *meaning* of text or other data, with similar meanings landing close together.
- Its core operation is **similarity search**: find the **k nearest** stored vectors to a
  query vector, ranked by **cosine** or **Euclidean** distance.
- This is **search by meaning** (semantic search), fundamentally different from a
  `WHERE` clause's exact or pattern matching.
- Brute-force nearest-neighbor is too slow at scale, so these stores use an
  **approximate nearest-neighbor (ANN)** index — trading a tiny bit of exactness for
  huge speed.
- It's the retrieval engine behind **RAG**: embed and store your documents, embed the
  question, retrieve the nearest chunks, and feed them to the model.
- It usually runs **alongside** your other databases (or as the **pgvector** extension
  inside Postgres), not instead of them.

Next up: [Caching layers](/learn/databases/caching-layers/).
