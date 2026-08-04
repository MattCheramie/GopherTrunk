---
slug: the-relational-model
title: The relational model — tables, rows & columns
description: The idea that has run the data world for fifty years — organising data into tables of rows and columns. Learn what a relation, a row, and a column really are, and why this simple shape is so powerful and enduring.
keywords: relational model, tables, rows, columns, relation, tuple, attribute, relational database, Edgar Codd, structured data, records
level: beginner
status: full
prereq:
  - what-is-a-database
faq:
  - q: "Why is it called the 'relational' model if it's just tables?"
    a: "The name comes from the mathematical term relation, which is what a table technically is — a set of rows sharing the same columns. It is not, confusingly, about the relationships between tables; those (via keys) come later. A table is a relation; that's where the name comes from."
  - q: "What's the difference between a row and a column?"
    a: "A column is a named field that every row has — like 'frequency' or 'name' — with a fixed type. A row is one complete record: one system, one call, one user, holding a value for each column. Columns define the shape; rows are the actual data."
  - q: "Do all databases use tables?"
    a: "No — but the relational, table-based model is overwhelmingly the most common, and it's the foundation SQL is built on. Non-relational (NoSQL) stores use other shapes like documents or key-value pairs, which a later unit covers. Learn tables first; they're the default for good reason."
---

# The relational model — tables, rows & columns

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The **relational model** organises data into **tables** — grids of **rows** and
**columns**. Each **column** is a named, typed field every row shares; each **row**
is one complete record. A table is formally a **relation**, which is where the name
comes from. This deceptively simple shape has run the data world for fifty years
because it's easy to reason about, query, and validate — and it's what
[SQL](/learn/databases/what-is-sql/) is built on.
</div>

You now know a database stores data durably and lets you query it. But *how* is the
data arranged so that querying is even possible? For most databases you'll meet,
the answer is the **relational model**: everything lives in tables. It sounds almost
too plain to matter, yet this one idea has outlasted decades of technology churn.
This lesson is about why that shape is the right one.

## Tables: the one shape to learn

A **table** is a grid, exactly like a well-behaved spreadsheet. It has named
**columns** running across the top and **rows** stacked below, each row a single
record. Here's a table of radio systems:

| id | name       | frequency_hz | protocol |
|----|------------|--------------|----------|
| 1  | Metro P25  | 851012500    | P25      |
| 2  | County DMR | 453100000    | DMR      |
| 3  | Rail TETRA | 390000000    | TETRA    |

That's the whole model in one picture. A database is a collection of tables like
this. When people say "relational database," this grid is what they mean, and
learning to think in tables is most of what it takes to think in databases.

## Columns define the shape

A **column** is a named field that *every* row in the table has: `name`,
`frequency_hz`, `protocol`. A column has a fixed **data type** — text, a number, a
date — so the database knows what kind of value belongs there and can reject
anything that doesn't fit. The set of columns and their types is the table's
**schema**, its shape, which the [next lesson](/learn/databases/schemas-and-types/)
covers in full.

The key idea: columns are decided *up front* and apply to the whole table. Every
row has a `frequency_hz`; none has some extra field the others lack. That
uniformity is exactly what lets the database query the table efficiently — it knows
in advance what's there.

## Rows are the records

A **row** (also called a **record**) is one complete entry: one system, one call,
one user. In the table above, row 2 is the County DMR system — one value for each
column, together describing one real thing. Adding data means adding rows; the
columns stay put.

Two properties make rows well-behaved. First, each row should be **uniquely
identifiable** — that's the `id` column, a
[primary key](/learn/databases/keys-and-relationships/), so you can always point at
exactly one row. Second, **row order carries no meaning**. A table is a *set* of
rows; the database is free to store them however it likes, and if you want results
in a particular order you ask for it explicitly with
[ORDER BY](/learn/databases/filtering-and-sorting/). Don't ever rely on "the order
they were inserted."

## Why the name "relational"

Here's a nuance worth getting straight, because the name misleads almost everyone.
"Relational" does **not** refer to the relationships *between* tables (those come
from [keys](/learn/databases/keys-and-relationships/) in a later lesson). It comes
from mathematics: a **relation** is a set of rows that all share the same columns —
which is exactly what a table is. The model was formalised by Edgar Codd in 1970,
and its staying power comes from that rigor. Because a table is a precise
mathematical object, you can define exact operations on it — filter, combine,
summarise — and that's what SQL turns into a practical language.

You'll hear the formal words occasionally: a **relation** (table), a **tuple**
(row), an **attribute** (column). You don't need them day to day, but now they
won't throw you.

## Why this shape wins

Why has such a simple idea dominated for fifty years? Because tables are:

- **Easy to reason about.** A grid of rows and columns is something anyone can
  picture and talk about, from junior dev to DBA.
- **Uniform.** Every row has the same columns, so one query works across all of
  them — no special cases.
- **Composable.** Tables can be filtered, sorted, and *joined* to each other on
  shared values, letting you answer questions no single table holds — the power we
  reach in [joins](/learn/databases/joins/).
- **Enforceable.** Fixed columns and types let the database reject bad data before
  it's stored.

Newer database shapes exist and earn their place for particular jobs — we cover
them in Unit 4 — but the table remains the default, and for most applications it's
the right default.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a table is a relation: rows are records, columns are the shared, typed fields." markdown="0">
  <p class="knowledge-check__q">Quick check: in a table, what does a single row represent?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A named field, like 'frequency', that every record shares</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">One complete record — one system, call, or user — with a value per column</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The type rule that decides what values a column can hold</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The **relational model** stores data in **tables** — grids of **rows** and
  **columns** — and it's the default shape for most databases.
- A **column** is a named, typed field every row shares; the columns and their
  types make up the table's **schema**.
- A **row** (record) is one complete entry, one value per column; rows should be
  uniquely identifiable, and their stored **order carries no meaning**.
- "Relational" comes from the mathematical **relation** (a table), *not* from
  relationships between tables — those come later, from keys.
- Tables win because they're **easy to reason about, uniform, composable, and
  enforceable** — the foundation SQL and the rest of this module build on.

Next up: [Schemas, columns & data types](/learn/databases/schemas-and-types/).
