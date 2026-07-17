---
slug: chunking-and-retrieval-quality
title: Chunking & retrieval quality
description: Why RAG lives or dies on retrieval — how chunking, metadata filtering, re-ranking, hybrid search, and query rewriting decide whether the right passage reaches the model, and how to measure retrieval on its own.
keywords: chunking, chunk size, overlap, retrieval quality, re-ranking, hybrid search, metadata filtering, query rewriting, recall, RAG, cross-encoder, BM25
level: advanced
status: full
prereq:
  - building-a-rag-pipeline
faq:
  - q: "What is the best chunk size for RAG?"
    a: "There's no single right number — it depends on your content and embedding model. Too large and each chunk carries noise that dilutes relevance and costs more; too small and you cut ideas in half and lose context. Split along natural boundaries (sections, paragraphs) where you can, add a little overlap so ideas aren't severed at the seam, and tune by measuring retrieval, not by guessing."
  - q: "What is re-ranking in RAG?"
    a: "Re-ranking is a second pass: you retrieve a larger candidate set with fast vector search, then score each candidate against the query with a stronger model — usually a cross-encoder — and keep only the top few. It's one of the cheapest ways to raise retrieval quality, because the first-pass search only has to get the right passage *somewhere* in the candidates, not first."
  - q: "What is hybrid search?"
    a: "Hybrid search combines keyword search (like BM25) with vector similarity search and merges the results. Vector search captures meaning but can miss exact tokens — an error code, a part number, a rare identifier — that keyword search nails. Running both and blending the scores catches passages either method alone would drop."
  - q: "How do I know if my RAG problem is retrieval or the model?"
    a: "Measure retrieval separately. For a set of questions with known answers, check whether the correct chunk appears in the top-k retrieved passages (recall). If it doesn't, no prompt or model change will help — fix retrieval. If it does and the answer is still wrong, the problem is in the prompt or generation step."
---

# Chunking & retrieval quality

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A RAG system's answers are only as good as its **retrieval quality**: if the right
passage never reaches the model, no amount of prompting or model choice can save the
answer. **Chunking** — how you split your documents — dominates that quality more than
almost anything else. And a handful of techniques on top — **re-ranking**, hybrid
search, metadata filtering, query rewriting — turn a mediocre retriever into a good one.
Build the [pipeline](/learn/building-ai/building-a-rag-pipeline/) first; this lesson is
about making it actually retrieve the right thing.
</div>

You've built a [RAG pipeline](/learn/building-ai/building-a-rag-pipeline/): documents
embedded into a vector store, questions embedded and matched, nearest chunks pasted into
the prompt. It works — sometimes. Other times it confidently answers from the wrong
passage, or misses an answer that's plainly in your docs. This lesson is about closing
that gap, and nearly all of it comes down to one idea: **retrieval is the part you have
to get right**.

## Garbage in, garbage out

The generator can only work with what retrieval hands it. If the correct passage isn't
in the retrieved chunks, the model is answering from thin air — and it will often do so
fluently and wrongly. This is the single most important thing to internalise about RAG:
**most RAG failures are retrieval failures, not model failures.** Teams reach for a
bigger model when their bot gives bad answers, but the bigger model was never shown the
right text. Fix what reaches it first, and the answers usually follow.

## Chunking

How you split documents into chunks shapes everything downstream, because a chunk is the
unit that gets embedded, retrieved, and pasted. Get it wrong and no later technique fully
recovers.

The two failure modes pull in opposite directions:

- **Chunks too large.** A big chunk covers several topics, so its embedding is a blurry
  average of all of them — it matches many queries weakly and none strongly. It also
  drags noise into the prompt and costs more tokens per retrieved passage.
- **Chunks too small.** A tiny chunk loses the context that gave it meaning. A sentence
  that says "this raises the noise floor" is useless if the chunk doesn't say *what*
  does.

Common strategies land between those extremes:

- **By structure** — split along sections, headings, or paragraphs so each chunk is one
  coherent idea. Usually the best default when documents have structure.
- **Fixed size with overlap** — cut every _N_ tokens, but let consecutive chunks share a
  little text at the seams. The **overlap** keeps an idea from being severed exactly at a
  boundary, so a passage split across two chunks still appears whole in at least one.
- **Semantic / structure-aware** — detect topic shifts (or use the document's own markup)
  and cut there, so boundaries fall between ideas rather than through them.

There's no universal chunk size. Match it to your content and, crucially, tune it by
*measuring* retrieval rather than eyeballing the chunks.

## Metadata & filtering

Every chunk can carry **metadata** alongside its text: source document, date, section,
author, content type. Storing that lets you **filter** — narrowing the search space by
metadata either before or after the vector search — so similarity is computed only over
the passages that could plausibly be right.

The payoff is sharper results. "Only this product's docs," "only pages updated this
year," "only troubleshooting sections" — each filter removes passages that might score
well on similarity yet be obviously irrelevant to the question. A well-chosen filter often
does more for quality than a better embedding model, because it eliminates whole classes
of wrong answers before ranking even begins.

## Re-ranking

Vector search is fast but coarse: it's good at getting the right passage *somewhere* in
the top results, less good at putting it first. **Re-ranking** exploits that. Retrieve a
larger candidate set — say the top 50 — then score each candidate against the query with a
stronger, slower model, typically a **cross-encoder** that reads the query and passage
together instead of comparing two independent embeddings. Keep the top few of *that*
ranking to send to the generator.

Because the cross-encoder only runs on the shortlist, not the whole corpus, the cost is
modest — and the quality gain is often the largest of any single change you can make.
First-pass retrieval only has to achieve good *recall* (the answer is in the 50);
re-ranking supplies the *precision* (the answer rises to the top 3).

## Hybrid search & query rewriting

Vector search matches on meaning, which is exactly what you want — until the query hinges
on an exact token. Error codes, part numbers, function names, and rare identifiers can be
semantically "close" to many things and embed poorly. **Hybrid search** runs a keyword
search (classically **BM25**) alongside the vector search and merges the two result lists,
so exact-term matches and meaning matches both get a vote. It reliably catches passages
either method alone would miss.

**Query rewriting** works on the other end — the question itself. A user's raw phrasing is
often a poor retrieval query: too terse, ambiguous, or full of pronouns referring to
earlier turns. Before retrieving, you can have a model rewrite or expand it — resolve
"it" to the thing meant, add synonyms, or split a compound question into parts — so the
text you actually embed and search is closer to how the answer is written in your docs.

## Measuring retrieval

You cannot improve what you don't measure, and the key move is to **evaluate retrieval
separately from answer quality**. Build a small set of questions with known answers, and
for each, record which chunk *should* be retrieved. Then ask one question of your
retriever: did the correct chunk appear in the top-_k_ results? The fraction that do is
your **recall** — the ceiling on how well the whole system can possibly do.

Splitting the metric this way tells you *where* to spend effort. Low retrieval recall
means the generator never had a chance — go fix chunking, filtering, or ranking. High
recall but bad answers means retrieval is doing its job and the problem is the prompt or
the model, which is where end-to-end
[evaluation](/learn/building-ai/evaluating-ai-features/) takes over. Without the split, you
can spend days tuning prompts for a problem that lives entirely in retrieval.

## Diagnosing

When a RAG answer is wrong, resist the urge to blame the model. Work the checklist in
order:

1. **Was the right chunk retrieved at all?** Log and inspect the actual retrieved
   passages for the failing query. This is nearly always where the fault is.
2. **If not, why?** Bad chunk boundaries (answer split across two chunks), a filter that
   excluded it, or a query that embedded poorly — a hybrid or rewritten query often fixes
   the last one.
3. **If the right chunk *was* retrieved but ranked low,** add or tune re-ranking.
4. **Only if the right chunk was retrieved and near the top,** look at the prompt and the
   model — this is the [component](/learn/building-ai/llm-as-a-component/) doing its job on
   what it was given, and blaming it first wastes the most time.

<div class="knowledge-check" data-quiz data-correct-msg="Right — inspect the retrieved chunks first; most RAG failures are retrieval failures, and the model can only use what it's handed." markdown="0">
  <p class="knowledge-check__q">Quick check: your RAG bot answers wrong even though the docs clearly contain the answer. First thing to check?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Retrieval — is the correct chunk actually being retrieved?</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Swap in a larger, more capable generation model</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Raise the temperature so the model explores more answers</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Retrieval is the ceiling.** The generator can only use what retrieval hands it — most
  RAG failures are retrieval failures, not model failures.
- **Chunking dominates.** Too large brings noise and diluted relevance; too small loses
  context. Split along structure, add overlap, and tune by measurement.
- **Filter with metadata.** Attach source, date, section, and type, then narrow the search
  before or after vector similarity to cut whole classes of wrong answers.
- **Re-rank and go hybrid.** Retrieve a wide candidate set and re-rank with a cross-encoder
  for precision; combine keyword (BM25) and vector search to catch exact terms.
- **Rewrite the query** before retrieving so the text you search matches how answers are
  written.
- **Measure retrieval on its own.** Recall in the top-_k_ tells you whether to fix
  retrieval or the prompt and model.

Next up: [RAG vs. long context vs. fine-tuning](/learn/building-ai/rag-vs-long-context-vs-finetuning/).
