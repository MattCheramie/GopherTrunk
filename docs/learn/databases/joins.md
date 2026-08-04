---
slug: joins
title: Joining tables
description: Joins are the heart of relational power — combining rows from two tables on a shared key. Learn how an inner join works, the difference between inner and outer joins, and why splitting data across tables and rejoining it is the whole point of the relational model.
keywords: SQL join, inner join, outer join, left join, join key, combining tables, ON clause, relational, foreign key join, join condition
level: intermediate
status: full
prereq:
  - keys-and-relationships
  - querying-with-select
faq:
  - q: "What's the difference between an inner join and a left join?"
    a: "An inner join returns only rows that have a match in both tables — unmatched rows from either side are dropped. A left join returns every row from the left table, matched or not, filling the right side with NULLs where there's no match. Use inner when you only want matched pairs; use left when you want all of one side regardless."
  - q: "What do I join two tables on?"
    a: "On a shared value — almost always a foreign key matching a primary key. To join calls to their systems you match calls.system_id to systems.id in the ON clause. That shared key is the thread the relational model uses to reconnect data it split across tables."
  - q: "Why split data into separate tables if I just have to join them back?"
    a: "Because splitting keeps each fact in one place — no duplication, no update anomalies — which is the goal of normalization. Joins are cheap and fast (especially with indexes on the keys), so you get clean storage and can still assemble any combined view you need on demand."
---

# Joining tables

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **join** combines rows from two tables by matching a **shared key** — usually a
foreign key meeting a primary key. An **inner join** returns only rows with a match on
both sides; an **outer join** (like a **LEFT JOIN**) keeps all rows from one side,
filling missing matches with **NULL**. Joins are why you can split data across tables
for cleanliness and still assemble any combined view — the payoff of
[keys & relationships](/learn/databases/keys-and-relationships/).
</div>

You split data into separate tables — systems here, calls there — linked by
[keys](/learn/databases/keys-and-relationships/). That keeps each fact in one place,
but it raises an obvious question: how do you get a call *and* the name of its system
in one result? Neither table holds both. The answer, and arguably the single most
important operation in relational databases, is the **join**. This lesson is where the
relational model earns its keep.

## Why you need a join

Recall the shape: `calls` has a `system_id` foreign key pointing at `systems.id`. To
show "each recent call with its system's name," you need columns from both tables at
once. Querying `calls` alone gives you a `system_id` — a bare number — not the human
name. A join brings the two tables together on that key so a single result row carries
data from both.

## The inner join

An **inner join** matches rows from two tables on a condition and returns the combined
rows where the match holds:

```sql
SELECT calls.started_at, systems.name, systems.protocol
FROM calls
JOIN systems ON calls.system_id = systems.id;
```

Read the **`ON`** clause as the rule for pairing rows: for each call, find the system
whose `id` equals that call's `system_id`, and stitch them into one wide row. The
result has columns from both tables — the call's timestamp beside its system's name and
protocol.

Two details worth noting. First, prefix columns with their table (`calls.started_at`,
`systems.name`) when a name could be ambiguous — both tables have an `id`, so you must
say which. Second, the join key is the foreign-key/primary-key pair; matching on the
relationship you designed is the overwhelmingly common case, and
[indexing those keys](/learn/databases/indexes/) is what keeps joins fast.

The word **inner** matters: an inner join keeps only rows that have a match on *both*
sides. A call whose `system_id` points nowhere (if that were allowed), or a system with
no calls, simply doesn't appear. Matched pairs only.

## Outer joins keep the unmatched

Sometimes dropping the unmatched rows is exactly wrong. "List every system and how many
calls it had" must include systems with *zero* calls — but an inner join would silently
omit them. That's what an **outer join** is for.

The most common is the **LEFT JOIN**: keep *every* row from the left (first) table,
matched or not, and fill the right side's columns with **NULL** where there's no match.

```sql
SELECT systems.name, calls.started_at
FROM systems
LEFT JOIN calls ON calls.system_id = systems.id;
```

Now a system with no calls still appears once, with `calls.started_at` as NULL. The
mental model:

- **INNER JOIN** — only rows matched on both sides.
- **LEFT JOIN** — all left rows; right side NULL when unmatched.
- **RIGHT JOIN** — the mirror image (all right rows); less common, since you can flip
  the tables and use LEFT.
- **FULL OUTER JOIN** — all rows from both sides, NULLs wherever a match is missing.

Choosing between them is really one question: *when a row on one side has no partner,
do I still want it?* Yes means outer; no means inner.

## Joining more than two tables

Joins chain. To reach across the [join table](/learn/databases/keys-and-relationships/)
of a many-to-many, you join through it — calls to `call_talkgroups` to `talkgroups`:

```sql
SELECT calls.started_at, talkgroups.name
FROM calls
JOIN call_talkgroups ON call_talkgroups.call_id = calls.id
JOIN talkgroups      ON talkgroups.id = call_talkgroups.talkgroup_id;
```

Each `JOIN ... ON` adds another table to the growing combined row. This is how a query
assembles a rich result from a well-split schema — three normalized tables becoming one
readable answer.

## Splitting and rejoining is the point

It can feel circular: split data into tables, then write joins to glue it back. But
that's precisely the relational bargain, and it's a good one. Splitting keeps every
fact in exactly one place — no duplicated system names to drift out of sync, the
discipline of [normalization](/learn/databases/normalization/). Joins then let you
reassemble *any* combined view on demand, cheaply, especially with the join keys
indexed. You store data cleanly and read it flexibly. That combination — clean writes,
flexible reads — is the whole reason the relational model has lasted, and the join is
the hinge it turns on.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an inner join returns only matched rows; a left join keeps all left rows, NULLing the unmatched right side." markdown="0">
  <p class="knowledge-check__q">Quick check: you want every system listed, even ones with no calls. Which join?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">An inner join — it always includes every row from both tables</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A left join from systems — it keeps all systems, NULLing calls where there are none</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">No join is needed; one table already has both</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **join** combines rows from two tables on a **shared key** — typically a foreign
  key matching a primary key — named in the **`ON`** clause.
- An **inner join** returns only rows with a match on **both** sides; unmatched rows are
  dropped.
- An **outer join** keeps unmatched rows: a **LEFT JOIN** keeps every left row and fills
  the right side with **NULL** when there's no match.
- Choose inner vs. outer by asking whether you still want a row that has no partner.
- Joins **chain** across multiple tables, including through a join table for
  many-to-many relationships.
- Splitting data for clean storage and **rejoining** it on demand is the core relational
  bargain — clean writes, flexible reads.

Next up: [Aggregation & GROUP BY](/learn/databases/aggregation-and-grouping/).
