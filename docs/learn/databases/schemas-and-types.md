---
slug: schemas-and-types
title: Schemas, columns & data types
description: A schema is the shape of your data — the columns, their types, and the rules. Learn what common data types to reach for, why deciding the shape up front saves you from a mess later, and how the database enforces it.
keywords: database schema, data types, columns, integer, text, varchar, boolean, timestamp, NULL, NOT NULL, CREATE TABLE, DDL, typed columns
level: beginner
status: full
prereq:
  - the-relational-model
faq:
  - q: "What is a schema, exactly?"
    a: "A schema is the defined shape of your data: which tables exist, what columns each has, the data type of every column, and the rules like 'this can't be empty.' It's the blueprint the database enforces on every row you store. In relational databases the schema is fixed up front, before you insert data."
  - q: "What does NULL mean?"
    a: "NULL is the database's marker for 'no value here' — unknown or missing, not zero and not an empty string. A column marked NOT NULL forbids it, forcing every row to supply a value. NULL has surprising behaviour in comparisons, so it's worth understanding early."
  - q: "Why not just make every column text?"
    a: "Because types are how the database catches bad data and works efficiently. A numeric column rejects 'banana', sorts and sums correctly, and stores compactly; a date column knows what 'before yesterday' means. Storing everything as text throws all of that away and pushes the checking onto you."
---

# Schemas, columns & data types

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **schema** is the shape of your data — which **columns** a table has, the **data
type** of each, and the rules like **NOT NULL**. In a relational database you
decide it *up front*, and the database enforces it on every row, rejecting anything
that doesn't fit. Picking sensible **types** (integer, text, boolean, timestamp)
lets the database catch bad data, store it compactly, and query it correctly —
building on [the relational model](/learn/databases/the-relational-model/).
</div>

A table is columns and rows. This lesson is about the columns: what a **schema**
is, the **types** you'll reach for, and why relational databases make you decide
the shape before you store a single row. That "decide up front" discipline feels
like a chore at first and turns out to be one of the model's quiet superpowers.

## A schema is the shape of your data

The **schema** is the full definition of your data's structure: the tables, their
columns, each column's **data type**, and the constraints on them. It's a blueprint
the database holds and enforces. Every row you insert is checked against it — try
to store text where a number belongs, or leave out a required field, and the
database refuses. Bad data bounces at the door instead of quietly rotting inside
your tables.

You write a schema with a `CREATE TABLE` statement. Here's the systems table from
the last lesson, defined properly:

```sql
CREATE TABLE systems (
    id           INTEGER      PRIMARY KEY,
    name         TEXT         NOT NULL,
    frequency_hz BIGINT       NOT NULL,
    protocol     TEXT         NOT NULL,
    active       BOOLEAN      NOT NULL DEFAULT TRUE,
    first_seen   TIMESTAMP    NOT NULL DEFAULT now()
);
```

Each line names a column, gives its type, and adds any rules. This is the shape
every row must obey. (Statements that define structure like this are called
**DDL** — data definition language — as opposed to the queries that read and
change data.)

## The types you'll actually use

Databases offer many types, but a handful cover most of what you'll store:

- **Integer** (`INTEGER`, `BIGINT`) — whole numbers: counts, IDs, a frequency in
  hertz. `BIGINT` holds bigger values than `INTEGER`.
- **Text** (`TEXT`, `VARCHAR(n)`) — strings: names, descriptions. `VARCHAR(n)` caps
  the length; `TEXT` is unbounded.
- **Boolean** (`BOOLEAN`) — true or false: is this system active?
- **Decimal / floating point** (`DECIMAL`, `REAL`) — fractional numbers. Use
  `DECIMAL` for money, where exact rounding matters; `REAL` for measurements where
  a tiny imprecision is fine.
- **Timestamp / date** (`TIMESTAMP`, `DATE`) — points in time: when a call was
  decoded. The database understands these, so "everything from the last hour" is a
  real, sortable query.

Choosing the *right* type isn't pedantry. A numeric column sorts and sums
correctly and stores compactly; a timestamp column knows what "before yesterday"
means; a boolean is one bit of truth, not the strings "yes"/"no"/"Y"/"true" you'd
have to reconcile forever. The type is the first line of data integrity.

## NULL: the "no value" marker

Every column also answers one question: can it be empty? The database's marker for
"no value here" is **NULL** — meaning unknown or missing, which is *not* the same
as zero, and *not* the same as an empty string. A `frequency_hz` of NULL means "we
don't know the frequency"; a `frequency_hz` of 0 means "the frequency is zero
hertz." Those are different facts, and NULL keeps them distinct.

You control whether NULL is allowed. A column marked **NOT NULL** must have a real
value in every row — the database rejects an insert that omits it. Marking the
columns that truly must be present as NOT NULL is one of the cheapest, highest-value
things you can do for data quality. (NULL also behaves oddly in comparisons —
`frequency_hz = NULL` is never true — which the
[filtering lesson](/learn/databases/filtering-and-sorting/) returns to.)

## Why decide the shape up front

Relational databases are **schema-on-write**: the shape is fixed before data goes
in, and the database enforces it on every write. That front-loading has real
payoffs:

- **Bad data can't get in.** Wrong type, missing required field, malformed value —
  rejected at write time, so your tables stay trustworthy.
- **Everyone knows the shape.** The schema is documentation the database can't let
  drift out of date. New code, new teammates, and your future self all read the
  same source of truth.
- **The database can optimise.** Knowing every row's exact layout lets it store,
  index, and query the data efficiently.

The tradeoff is rigidity: changing the shape later means a deliberate
[migration](/learn/databases/migrations/) rather than just writing a differently-
shaped record. Some non-relational stores flip this to **schema-on-read** — store
anything, sort out the shape when you read — which buys flexibility at the cost of
these guarantees. We weigh that tradeoff in Unit 4; for now, know that the
up-front schema is a feature, not red tape.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a schema is the fixed, enforced shape of your data: columns, their types, and rules like NOT NULL." markdown="0">
  <p class="knowledge-check__q">Quick check: what does a table's schema define?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The order rows are physically stored in on disk</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The columns, their data types, and the rules every row must obey</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The list of users allowed to read the table</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **schema** is the defined shape of your data: the tables, their **columns**,
  each column's **data type**, and its rules — written with `CREATE TABLE`.
- Common **types** are integer, text, boolean, decimal, and timestamp; picking the
  right one lets the database catch errors, store compactly, and query correctly.
- **NULL** marks a missing/unknown value — distinct from zero or empty string; a
  **NOT NULL** column forbids it and forces a real value.
- Relational databases are **schema-on-write**: the shape is fixed up front and
  enforced on every write, so bad data bounces at the door.
- Deciding the shape up front buys **data quality, shared documentation, and
  optimisation**, at the cost of needing a deliberate change to evolve it later.

Next up: [Keys & relationships](/learn/databases/keys-and-relationships/).
