---
slug: indexes
title: Indexes & how queries get fast
description: Why the same query can take a millisecond or a minute — indexes, what they cost, and the mental model of a book's index that explains them. Learn how a full table scan differs from an index lookup, when to add an index, and the price you pay on writes.
keywords: database index, full table scan, index lookup, B-tree, query performance, EXPLAIN, index cost, write overhead, primary key index, when to index
level: intermediate
status: full
prereq:
  - filtering-and-sorting
  - keys-and-relationships
faq:
  - q: "What actually is an index?"
    a: "A separate, sorted data structure — usually a B-tree — that maps the values in a column to the rows that hold them, so the database can jump straight to matching rows instead of scanning the whole table. It's exactly like the index at the back of a book: a pre-sorted lookup that saves you reading every page."
  - q: "If indexes make queries faster, why not index every column?"
    a: "Because indexes aren't free. Each one takes storage and must be updated on every insert, update, and delete — so more indexes mean slower writes and more disk. Index the columns you actually filter, join, or sort on; skip the rest. It's a deliberate read-speed-for-write-cost trade."
  - q: "How do I know if a query is using an index?"
    a: "Run it through EXPLAIN, which shows the query plan the database chose. If you see a sequential or full table scan on a big table where you expected a lookup, you're likely missing an index — or the query is written so the index can't be used."
---

# Indexes & how queries get fast

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **index** is a separate sorted structure (usually a **B-tree**) that lets the
database jump straight to matching rows instead of a **full table scan** — the
difference between a millisecond and a minute. It's the back-of-the-book index made
literal. But indexes **cost**: storage, plus slower **writes**, because every insert,
update, and delete must maintain them. So you index the columns you **filter, join, or
sort** on — not every column. Use **EXPLAIN** to see what the database actually does.
</div>

You can now write queries that read, filter, join, and summarise. This final lesson of
the unit answers a question that decides whether those queries are usable at scale: why
does the *same* query take a millisecond on one table and a minute on another? The
answer, almost always, is **indexes** — and understanding them is the leap from queries
that work to queries that work *fast*.

## The full table scan

Picture the calls table with ten million rows, and this query:

```sql
SELECT * FROM calls WHERE system_id = 3;
```

Without help, the database has exactly one option: read *every* row and check each
one's `system_id`. That's a **full table scan** (or sequential scan) — ten million
comparisons to find maybe a few hundred matches. It works, and it's fine on small
tables. On a big one it's slow, and it gets slower as the table grows, because the work
is proportional to the *whole* table, not to the number of matches.

## The book-index mental model

Now think about finding every mention of "downconverter" in a 900-page book. You could
read all 900 pages (the full scan). Or you could flip to the **index** at the back — an
alphabetical list of terms with the pages they appear on — find the word in seconds, and
turn straight to those pages.

A database **index** is exactly this. It's a separate structure that keeps a column's
values in *sorted* order, each paired with a pointer to the row that holds it. With an
index on `system_id`, the database doesn't scan ten million rows — it navigates the
sorted index straight to `system_id = 3` and follows the pointers to just those rows. The
lookup cost grows with the *logarithm* of the table size, not the size itself, so it
stays fast even as the table balloons.

```sql
CREATE INDEX idx_calls_system ON calls (system_id);
```

That one statement can turn a minute-long scan into an instant lookup. Under the hood
most indexes are a **B-tree**, a structure built for exactly this: staying balanced and
sorted so any value is reachable in a few steps regardless of size.

## What indexes speed up

Indexes help precisely the operations that need to *locate* rows by a column's value:

- **WHERE filters** — `WHERE system_id = 3` or `WHERE started_at > '2026-08-01'` jump to
  matches instead of scanning.
- **[Joins](/learn/databases/joins/)** — matching a foreign key to a primary key is a
  lookup per row; an index on the join key is what keeps joins fast.
- **ORDER BY** — because an index is already sorted, the database can often read rows in
  order straight from it, skipping a separate sort step.

This is why **primary keys are automatically indexed** — you look rows up by their key
constantly, so the database indexes it for you. Foreign keys usually deserve an index
too, precisely because you join on them.

## Indexes aren't free

If indexes are so good, why not index every column? Because they have a real **cost**,
and it's the crux of using them well:

- **Storage.** An index is extra data on disk — a copy of the column's values plus
  pointers. Many indexes can rival the size of the table itself.
- **Slower writes.** This is the big one. Every **INSERT**, **UPDATE**, and **DELETE**
  must *also* update every index on the table to keep it sorted and correct. Five indexes
  mean each write does roughly five times the index maintenance. Indexes trade *write*
  speed for *read* speed.

So indexing is a deliberate trade, not a free win. Index the columns you actually
**filter, join, or sort** on in real queries; leave the rest unindexed. A table that's
written far more than it's read might want *fewer* indexes; a read-heavy reporting table
might want several. Match the indexes to how the data is actually used.

## Seeing what the database does

You don't have to guess whether an index is helping — the database will tell you. Prefix
a query with **`EXPLAIN`** and it shows the **query plan**: the strategy the
[planner](/learn/databases/what-is-sql/) chose, including whether it's doing an index
lookup or a full scan.

```sql
EXPLAIN SELECT * FROM calls WHERE system_id = 3;
```

If you see a sequential scan over a large table where you expected a lookup, you're
probably missing an index — or the query is written in a way that prevents one being used
(wrapping the column in a function, or a `LIKE '%foo'` with a leading wildcard, can both
defeat an index). Reading query plans is the core skill of database performance work, and
the module's later
[performance & monitoring](/learn/databases/performance-and-monitoring/) lesson goes deeper. For now, the
essential mental model is the one that carries you a long way: an index is a book's index,
it makes lookups fast, and it isn't free.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an index is a sorted lookup structure that skips the full scan, at the cost of storage and slower writes." markdown="0">
  <p class="knowledge-check__q">Quick check: why not just put an index on every column?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Databases only allow one index per table</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Each index costs storage and slows writes, since every write must update it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Extra indexes make SELECT queries slower, not faster</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Without help, a filter runs a **full table scan** — reading every row — which grows
  slower as the table grows.
- An **index** is a separate **sorted** structure (usually a **B-tree**) that lets the
  database jump straight to matching rows, like a book's back-of-index.
- Indexes speed up **WHERE filters**, **joins** on keys, and **ORDER BY**; **primary
  keys** are indexed automatically.
- Indexes **cost** storage and, crucially, slow every **INSERT/UPDATE/DELETE**, since
  each write must maintain them — a read-speed-for-write-cost trade.
- Index the columns you **filter, join, or sort** on; don't index everything.
- Use **EXPLAIN** to see the query plan and confirm whether an index is actually being
  used.

Next up: [Normalization & avoiding duplication](/learn/databases/normalization/).
