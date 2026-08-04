---
slug: querying-with-select
title: Querying with SELECT
description: SELECT is the single most-used statement in all of software. Learn how to ask a database for exactly the columns and rows you need, why SELECT * is a habit to break, and how to rename and compute columns in your results.
keywords: SELECT, SQL query, columns, SELECT star, FROM, aliases, computed columns, DISTINCT, projection, reading data
level: beginner
status: full
prereq:
  - what-is-sql
faq:
  - q: "What's wrong with SELECT *?"
    a: "SELECT * grabs every column, which is handy for quick exploration but a poor habit in real code. It fetches data you don't need (slower, more memory), and it breaks silently when someone adds or reorders columns. Naming the columns you actually want is clearer, faster, and stable against schema changes."
  - q: "Does SELECT change the data in the table?"
    a: "No. SELECT only reads — it produces a result set to look at and never modifies the stored rows. The statements that change data are INSERT, UPDATE, and DELETE, covered later. You can run any SELECT freely without fear of altering anything."
  - q: "What does DISTINCT do?"
    a: "DISTINCT removes duplicate rows from the result, so you get each unique combination once. SELECT DISTINCT protocol FROM systems gives you the set of protocols in use — one of each — rather than one entry per system."
---

# Querying with SELECT

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**SELECT** is how you read data from a database — it names the **columns** you want
**FROM** a table and returns a **result set**. Name the columns you need rather than
reaching for **`SELECT *`**, which is convenient for exploring but fragile in real
code. You can **rename** columns with aliases, **compute** new ones, and drop
duplicates with **DISTINCT**. It's the foundation every later query builds on, and it
only reads — building on [what SQL is](/learn/databases/what-is-sql/).
</div>

`SELECT` is, without much exaggeration, the most-used statement in all of software.
Every dashboard number, every list on a screen, every report starts as a SELECT. The
good news is the basic form is simple and reads almost like English. This lesson gets
you fluent in that basic form; the next lessons pile on filtering, sorting, joining,
and grouping — but they're all *SELECT with more clauses*.

## The basic shape

A SELECT names the columns you want and the table to get them from:

```sql
SELECT name, frequency_hz, protocol
FROM systems;
```

Two clauses: `SELECT` lists the columns, `FROM` names the table. The result is a
**result set** — itself a table of rows and columns, containing just what you asked
for. Run it and you get every row of `systems`, but only those three columns.

That "only those columns" part has a name: **projection**. You're projecting the
table down to the columns you care about. A table might have twenty columns; if you
need three, you ask for three.

## Break the SELECT * habit

You'll constantly see (and be tempted to write) `SELECT *`, where the `*` means
"every column":

```sql
SELECT * FROM systems;
```

For poking around at a table interactively, that's fine — it's the fastest way to see
what's there. But in real application code it's a habit worth breaking, for concrete
reasons:

- **It fetches data you don't need.** Pulling twenty columns to use three wastes
  bandwidth and memory, and can defeat an [index](/learn/databases/indexes/) that
  would otherwise satisfy the query from the index alone.
- **It's fragile.** Add, remove, or reorder a column and `SELECT *` silently returns a
  different shape — code that assumed a column position or count now breaks in
  surprising ways.
- **It hides intent.** Naming the columns documents exactly what this query depends
  on. The next reader (often you) sees precisely what's used.

Name the columns you want. It's a little more typing now and a lot less debugging
later.

## Renaming and computing columns

The `SELECT` list isn't limited to bare column names. You can give a column a new name
in the result with **`AS`** (an **alias**), and you can compute values on the fly:

```sql
SELECT
    name AS system_name,
    frequency_hz / 1000000.0 AS frequency_mhz
FROM systems;
```

Here `frequency_mhz` is a **computed column** — the result set carries a value that
isn't stored anywhere, calculated per row as the query runs. Aliases are especially
useful for computed columns (which otherwise get an ugly auto-generated name) and for
giving results friendly labels an application can rely on. The stored table is
untouched; you're just shaping the *output*.

## DISTINCT: unique rows only

Sometimes you want the *set* of values, not one per row. **`DISTINCT`** collapses
duplicate rows in the result to one each:

```sql
SELECT DISTINCT protocol
FROM systems;
```

If a hundred systems use only P25, DMR, and TETRA, this returns three rows, not a
hundred. It's the quick way to answer "what distinct values appear in this column?"
Keep in mind DISTINCT has to compare rows to weed out duplicates, so it's not free on
large results — but for exactly this question it's the right tool. (When you want
*counts* per value rather than just the list, that's
[aggregation and GROUP BY](/learn/databases/aggregation-and-grouping/), a couple of
lessons on.)

## SELECT only reads

One reassuring fact to lock in: **SELECT never changes your data**. It reads and
returns a result set, and the stored rows are exactly as they were. You can run any
SELECT — on production, even — without fear of altering anything. The statements that
*modify* data are `INSERT`, `UPDATE`, and `DELETE`, and they come later in
[inserting, updating & deleting](/learn/databases/inserting-updating-deleting/). This
read-only safety is why exploring a database with SELECT is a great way to learn one:
you genuinely can't break anything by looking.

<div class="knowledge-check" data-quiz data-correct-msg="Right — name the columns you need; SELECT * is handy for exploring but fragile and wasteful in real code." markdown="0">
  <p class="knowledge-check__q">Quick check: why prefer naming columns over <code>SELECT *</code> in application code?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">SELECT * doesn't actually work on most databases</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It fetches only what you need and won't silently break when columns change</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Naming columns lets SELECT modify the table, which SELECT * can't</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **SELECT** reads data: it names the **columns** you want **FROM** a table and
  returns a **result set** — itself a table of rows and columns.
- Choosing specific columns is **projection**; prefer it over **`SELECT *`**, which is
  fine for exploring but wasteful and fragile in real code.
- **Aliases** (`AS`) rename columns in the result, and you can add **computed columns**
  calculated per row without changing stored data.
- **DISTINCT** removes duplicate rows, giving you the unique set of values.
- **SELECT only reads** — it never modifies stored data, so you can run it freely.

Next up: [Filtering, sorting & limiting](/learn/databases/filtering-and-sorting/).
