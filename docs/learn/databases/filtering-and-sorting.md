---
slug: filtering-and-sorting
title: Filtering, sorting & limiting
description: WHERE, ORDER BY, and LIMIT narrow a result set to the rows that matter, in the order you want, without dragging back the whole table. Learn to filter with conditions, combine them, handle NULL correctly, sort, and page through results.
keywords: WHERE, ORDER BY, LIMIT, filtering, sorting, SQL conditions, AND OR, IN, LIKE, BETWEEN, NULL, IS NULL, pagination, OFFSET
level: beginner
status: full
prereq:
  - querying-with-select
faq:
  - q: "Why can't I use = NULL to find empty values?"
    a: "Because NULL means 'unknown', and unknown compared to anything — even another NULL — is itself unknown, never true. So frequency_hz = NULL matches nothing. You must use IS NULL (or IS NOT NULL), which is the special syntax for testing the absence of a value."
  - q: "Does ORDER BY change the table's stored order?"
    a: "No. Rows in a table have no inherent order; ORDER BY sorts only the result set of that one query. If you want results sorted, you must say so every time — never rely on the order rows happen to come back without an ORDER BY."
  - q: "What's the point of LIMIT?"
    a: "LIMIT caps how many rows come back, so you fetch just the first page or the top few rather than an entire large table. Paired with ORDER BY (and OFFSET) it's how apps show 'the 20 most recent calls' without dragging back millions of rows."
---

# Filtering, sorting & limiting

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Three clauses turn a whole-table SELECT into a precise question. **WHERE** keeps only
the rows matching a condition, **ORDER BY** sorts the result, and **LIMIT** caps how
many rows come back. Together they let you ask for "the 20 newest P25 calls" without
dragging back the whole table. Watch out for **NULL** — test it with **IS NULL**, not
`= NULL` — building on [querying with SELECT](/learn/databases/querying-with-select/).
</div>

A bare SELECT returns *every* row. Real questions are narrower: not "all calls" but
"today's calls on this talkgroup, newest first, top twenty." This lesson adds the three
clauses that get you there — **WHERE** to filter, **ORDER BY** to sort, **LIMIT** to
cap — the trio behind almost every list you've ever seen in an app.

## WHERE: keep only matching rows

**`WHERE`** filters the rows: the database keeps only those for which your condition is
true, and discards the rest before you ever see them.

```sql
SELECT name, frequency_hz
FROM systems
WHERE protocol = 'P25';
```

Only P25 systems come back. The condition can use the comparisons you'd expect: `=`,
`!=` (or `<>`), `<`, `>`, `<=`, `>=`. And a few SQL-specific operators earn their keep:

- **`IN`** — matches any value in a list: `WHERE protocol IN ('P25', 'DMR')`.
- **`BETWEEN`** — an inclusive range: `WHERE frequency_hz BETWEEN 851000000 AND 860000000`.
- **`LIKE`** — pattern matching on text, with `%` as a wildcard:
  `WHERE name LIKE 'Metro%'` matches anything starting "Metro".

Filtering in the database, not in your application, is the whole point. Let the
database return 20 matching rows rather than shipping a million to your program to sift
through — it's dramatically faster and it's what [indexes](/learn/databases/indexes/)
are built to accelerate.

## Combining conditions

Real filters stack up. Join conditions with **`AND`** (all must hold) and **`OR`** (any
may hold), and group them with parentheses to keep the logic unambiguous:

```sql
SELECT name, frequency_hz
FROM systems
WHERE protocol = 'P25'
  AND (frequency_hz > 851000000 OR active = FALSE);
```

Those parentheses matter. `AND` binds tighter than `OR`, so without them the meaning
shifts. When a filter mixes AND and OR, parenthesise the intent explicitly — it's the
difference between the query you meant and the one you typed.

## NULL needs special handling

Here's the classic trap. To find systems with no recorded frequency, you'd reach for
`WHERE frequency_hz = NULL` — and get *zero rows*, always, even when NULLs exist.

The reason goes back to what [NULL means](/learn/databases/schemas-and-types/):
"unknown." Comparing unknown to anything — even to another unknown — yields *unknown*,
never true, so the row is never kept. SQL gives you dedicated syntax for the absence of
a value:

```sql
SELECT name FROM systems WHERE frequency_hz IS NULL;      -- missing
SELECT name FROM systems WHERE frequency_hz IS NOT NULL;  -- present
```

Use **`IS NULL`** and **`IS NOT NULL`** whenever you're testing presence. This one
gotcha catches nearly everyone once; now it won't catch you.

## ORDER BY: sort the result

Table rows have no inherent order — the database may return them however it likes.
**`ORDER BY`** sorts the *result set* by one or more columns:

```sql
SELECT name, frequency_hz
FROM systems
ORDER BY frequency_hz DESC, name ASC;
```

`ASC` sorts ascending (the default), `DESC` descending. List several columns to break
ties: here, by frequency high-to-low, and equal frequencies alphabetically by name.
The crucial rule: **if you want a particular order, you must ask for it.** Without an
ORDER BY, don't assume rows come back in insertion order, or any order at all — that's
undefined and *will* eventually surprise you.

## LIMIT: just the first page

**`LIMIT`** caps how many rows return, so you fetch the top few instead of everything:

```sql
SELECT name, started_at
FROM calls
ORDER BY started_at DESC
LIMIT 20;
```

That's "the 20 most recent calls" — the pattern behind virtually every feed and
top-N list. LIMIT paired with ORDER BY is what makes it meaningful: the *newest*
twenty, not an arbitrary twenty. To page further, add **`OFFSET`** to skip rows —
`LIMIT 20 OFFSET 20` is the second page. That's the basic recipe for **pagination**,
letting an app show huge tables one screen at a time without ever loading the whole
thing.

<div class="knowledge-check" data-quiz data-correct-msg="Right — NULL means 'unknown', so = NULL is never true; use IS NULL to test for a missing value." markdown="0">
  <p class="knowledge-check__q">Quick check: how do you find rows where a column has no value?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">WHERE column = NULL</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">WHERE column IS NULL</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">WHERE column = '' — an empty string always matches NULL</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **WHERE** filters rows to those matching a condition, using `=`, `<`, `>`, and
  SQL operators like **IN**, **BETWEEN**, and **LIKE** — done in the database, not your
  app.
- Combine conditions with **AND**/**OR**, and use parentheses to make mixed logic
  unambiguous.
- **NULL** means "unknown", so `= NULL` is never true — test presence with **IS NULL**
  and **IS NOT NULL**.
- **ORDER BY** sorts the result set (`ASC`/`DESC`); rows have no inherent order, so ask
  for the order you want every time.
- **LIMIT** caps returned rows for top-N lists; with **OFFSET** it gives you
  **pagination** through a large table.

Next up: [Joining tables](/learn/databases/joins/).
