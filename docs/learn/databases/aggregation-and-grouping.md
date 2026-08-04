---
slug: aggregation-and-grouping
title: Aggregation & GROUP BY
description: Turning many rows into a summary — counts, sums, and averages — with GROUP BY and HAVING, the tools behind every dashboard number. Learn aggregate functions, how grouping partitions rows, and why HAVING is not the same as WHERE.
keywords: aggregation, GROUP BY, HAVING, COUNT, SUM, AVG, MIN, MAX, aggregate functions, summary, dashboard, grouping rows
level: intermediate
status: full
prereq:
  - filtering-and-sorting
faq:
  - q: "What's the difference between WHERE and HAVING?"
    a: "WHERE filters individual rows before they're grouped; HAVING filters the groups after aggregation. Use WHERE to pick which rows count (only P25 systems), and HAVING to pick which group results survive (only systems with more than 100 calls). WHERE can't reference an aggregate like COUNT; HAVING can."
  - q: "Do I always need GROUP BY to use COUNT or SUM?"
    a: "No. Without GROUP BY, an aggregate collapses the whole result into a single number — COUNT(*) returns the total row count. Add GROUP BY when you want one summary row per category instead of one for the entire table."
  - q: "Why does the database complain about a column not in GROUP BY?"
    a: "Because once you group, each output row stands for many input rows, so a plain column has many possible values and no single answer. Every column in the SELECT must either be in the GROUP BY or wrapped in an aggregate that reduces the many values to one."
---

# Aggregation & GROUP BY

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Aggregate functions** — **COUNT**, **SUM**, **AVG**, **MIN**, **MAX** — collapse
many rows into one summary number. **GROUP BY** partitions rows into groups and
produces one summary row per group (calls *per system*). **HAVING** then filters those
groups — and it's *not* the same as **WHERE**, which filters rows *before* grouping.
These are the tools behind every dashboard total, building on
[filtering & sorting](/learn/databases/filtering-and-sorting/).
</div>

So far every query returned rows more or less as they're stored — filtered, sorted,
joined, but still individual rows. Often what you actually want is a *summary*: not
every call, but how many; not each duration, but the average. That's **aggregation** —
turning many rows into a single computed answer — and with **GROUP BY** you get one
answer per category. Every "1,204 calls today" and "avg duration 8.3s" on a dashboard is
this lesson.

## Aggregate functions collapse rows

An **aggregate function** takes many rows and returns one value. The five you'll use
constantly:

- **`COUNT`** — how many rows. `COUNT(*)` counts all rows; `COUNT(column)` counts rows
  where that column isn't NULL.
- **`SUM`** — the total of a numeric column.
- **`AVG`** — the average.
- **`MIN`** / **`MAX`** — the smallest and largest value.

Used on their own, they reduce the whole table to a single row:

```sql
SELECT COUNT(*) AS total_calls,
       AVG(duration_s) AS avg_duration,
       MAX(duration_s) AS longest
FROM calls;
```

One row comes back: the total number of calls, their average duration, and the longest.
The individual rows vanish into the summary.

## GROUP BY: a summary per category

A single grand total is often too coarse. You want calls *per system*, average duration
*per talkgroup*. **`GROUP BY`** partitions the rows into groups that share a value, and
the aggregate runs *once per group*, yielding one summary row each:

```sql
SELECT system_id,
       COUNT(*) AS call_count,
       AVG(duration_s) AS avg_duration
FROM calls
GROUP BY system_id;
```

Instead of one row for the whole table, you get one row per `system_id` — its call
count and average duration. That's the classic dashboard shape: a metric broken down by
category. Group by more than one column to break down by combinations (system *and*
day, say).

## The golden rule of GROUP BY

There's one rule that trips up every beginner, so internalise it now: **every column in
the SELECT must either appear in the GROUP BY or be inside an aggregate.**

Why? Once you group by `system_id`, each output row represents *many* input rows.
Asking for a plain `started_at` alongside it is incoherent — which of the many
timestamps in that group should it show? There's no single answer, so the database
rejects it. The column has to be either the thing you grouped on (one value per group,
fine) or reduced by an aggregate like `MAX(started_at)` (many values, collapsed to one).
When the database complains that a column "must appear in the GROUP BY clause," this is
what it's protecting you from.

## HAVING: filter the groups

You already have [WHERE](/learn/databases/filtering-and-sorting/) to filter rows. But
what if you want to filter on the *aggregate result* — say, only systems with more than
100 calls? WHERE can't do it: WHERE runs *before* grouping, when `COUNT(*)` doesn't
exist yet. That's what **`HAVING`** is for — it filters the groups *after* aggregation:

```sql
SELECT system_id, COUNT(*) AS call_count
FROM calls
WHERE started_at >= '2026-08-01'    -- filter rows first
GROUP BY system_id
HAVING COUNT(*) > 100               -- then filter groups
ORDER BY call_count DESC;
```

Notice both clauses working together. **WHERE picks which rows count** (only calls since
August 1st). **HAVING picks which group results survive** (only systems that ended up
with more than 100). The order the database applies them is the order to think in:
filter rows, form groups, aggregate, filter groups, sort. Keeping WHERE and HAVING
straight — rows before, groups after — is most of getting aggregation right.

## Aggregation with joins

Aggregation and [joins](/learn/databases/joins/) combine naturally, and it's where
summaries get readable. Join first to bring in the names, then group and count:

```sql
SELECT systems.name, COUNT(calls.id) AS call_count
FROM systems
LEFT JOIN calls ON calls.system_id = systems.id
GROUP BY systems.name
ORDER BY call_count DESC;
```

The LEFT JOIN ensures systems with zero calls still appear (with a count of 0 — note
`COUNT(calls.id)` counts non-NULL matches, so an unmatched system counts 0, not 1).
This one query — join, group, count, sort — is the shape of a huge fraction of every
report and dashboard you'll ever build.

<div class="knowledge-check" data-quiz data-correct-msg="Right — WHERE filters rows before grouping; HAVING filters the groups after aggregation." markdown="0">
  <p class="knowledge-check__q">Quick check: you want only systems with more than 100 calls. Which clause?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">WHERE COUNT(*) > 100 — WHERE can filter on aggregates</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">HAVING COUNT(*) > 100 — HAVING filters groups after aggregation</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">LIMIT 100 — it keeps only groups over 100</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Aggregate functions** — **COUNT**, **SUM**, **AVG**, **MIN**, **MAX** — collapse
  many rows into one summary value.
- Without grouping, an aggregate reduces the whole result to a single number.
- **GROUP BY** partitions rows into groups and produces one summary row per group — the
  metric-by-category shape of every dashboard.
- The golden rule: every SELECT column must be in the **GROUP BY** or inside an
  **aggregate**, because each output row stands for many input rows.
- **WHERE** filters rows *before* grouping; **HAVING** filters groups *after*
  aggregation — they are not interchangeable.
- Aggregation combines with **joins** (join to get names, then group and count) to
  produce readable reports.

Next up: [Inserting, updating & deleting](/learn/databases/inserting-updating-deleting/).
