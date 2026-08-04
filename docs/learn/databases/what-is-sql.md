---
slug: what-is-sql
title: What SQL is
description: SQL is the declarative language for talking to a relational database — you describe the data you want, not how to fetch it, and the database works out the rest. Learn what makes SQL declarative, its main statement families, and why it's worth learning once.
keywords: SQL, structured query language, declarative language, relational database, query, DDL, DML, SELECT, standard, dialects, database language
level: beginner
status: full
prereq:
  - the-relational-model
faq:
  - q: "What does 'declarative' mean for SQL?"
    a: "You describe the result you want — which rows, which columns, in what order — and leave it to the database to figure out how to produce it efficiently. You don't write the loops that scan tables or the logic that uses indexes; the database's query planner does that. You say what, it decides how."
  - q: "Is SQL the same on every database?"
    a: "Mostly. There's an ANSI SQL standard that the core — SELECT, INSERT, JOIN, WHERE — follows everywhere, so skills transfer across Postgres, MySQL, SQLite, and others. But each database has its own dialect with extra functions and small differences, so the edges vary even though the middle is shared."
  - q: "Do I really need to learn SQL if I use an ORM?"
    a: "Yes. An ORM (object-relational mapper) generates SQL for you, but it's a leaky abstraction — when a query is slow or behaves oddly, you have to read the SQL underneath to understand why. SQL is the lingua franca of data; learning it once pays off across every database and every tool."
---

# What SQL is

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**SQL** (Structured Query Language) is how you talk to a relational database. It's
**declarative**: you *describe* the data you want — which rows, which columns, in
what order — and the database's **query planner** figures out *how* to fetch it.
Learn it once and it transfers everywhere, because a shared **standard** underlies
Postgres, MySQL, SQLite, and the rest (with small **dialect** differences). SQL is
what the [relational model](/learn/databases/the-relational-model/) is queried with,
and the rest of this unit is SQL in practice.
</div>

You've got tables, columns, and keys. Now you need a way to *ask questions* of them
— and that language, for essentially every relational database, is **SQL**. It's one
of the highest-leverage things a developer can learn: a single language that's stayed
relevant for fifty years and works across nearly every database you'll ever touch.
This lesson is the orientation; the lessons after it are SQL in your hands.

## Say what you want, not how to get it

The defining trait of SQL is that it's **declarative**. In most programming you write
*imperative* code: step-by-step instructions — open the file, loop over the rows,
check each one, collect the matches. SQL flips that. You state the *result you want*
and stay silent on the mechanics:

```sql
SELECT name, frequency_hz
FROM systems
WHERE protocol = 'P25'
ORDER BY frequency_hz;
```

Read it almost like English: "give me the name and frequency of every system whose
protocol is P25, sorted by frequency." Nowhere did you say *how* — no loop, no
index lookup, no decision about what order to scan the table. You described the
destination; the database plots the route.

## The query planner does the "how"

Who works out the how? A part of the database called the **query planner** (or
optimiser). It takes your declarative request and decides on an efficient execution
strategy — whether to use an [index](/learn/databases/indexes/) or scan the table,
what order to combine tables in a join, and so on. For the same query it might pick
different strategies as your data grows, all without you rewriting a line.

This is the deal SQL offers: you give up fine control over the mechanics, and in
return the database handles the hard, data-dependent optimisation for you. It's
usually a great trade — the planner is better at it than hand-written loops would be,
and your query stays readable. When it *doesn't* choose well, you can inspect its plan
and nudge it, which the [performance lesson](/learn/databases/indexes/) touches on.

## The families of statements

SQL isn't only for reading. Its statements fall into a few groups, and you'll spend
this unit meeting them:

- **Querying (reading)** — `SELECT`, the workhorse you'll use most, covered next in
  [querying with SELECT](/learn/databases/querying-with-select/). Everything about
  filtering, sorting, joining, and aggregating hangs off it.
- **Changing data** — `INSERT`, `UPDATE`, `DELETE`: adding rows, modifying them, and
  removing them. This is **DML** (data manipulation language), covered in
  [inserting, updating & deleting](/learn/databases/inserting-updating-deleting/).
- **Defining structure** — `CREATE TABLE`, `ALTER TABLE`, `DROP`: the schema
  statements you met earlier. This is **DDL** (data definition language).

Reading data is where you'll live day to day, so `SELECT` gets the lion's share of
attention. But it's the same language throughout — one grammar for asking, changing,
and shaping.

## One language, many dialects

Here's what makes SQL such a good investment: it's largely **standard**. There's an
ANSI SQL standard, and the core — `SELECT`, `WHERE`, `JOIN`, `INSERT` — works
essentially the same across Postgres, MySQL, SQLite, SQL Server, and Oracle. Learn it
on one and you can query all of them.

Each database also has its own **dialect**: extra built-in functions, slightly
different syntax for dates or JSON, its own way to auto-generate keys. So the *edges*
differ even though the *middle* is shared. In practice you learn the common core once
(that's this unit) and pick up a specific database's quirks as you need them. The
transferable skill is real — SQL fluency moves with you across jobs, stacks, and
decades.

## Why it's worth learning properly

You might reach a database through an **ORM** — a library that maps your program's
objects to rows and writes the SQL for you (we compare the approaches in
[ORMs vs. raw SQL](/learn/databases/orms-vs-raw-sql/) later). Tempting to think you can
then skip SQL. You can't, really. An ORM is a convenience over SQL, not a replacement
for understanding it: the moment a query is slow, returns the wrong rows, or does
something surprising, you're reading the generated SQL to find out why. SQL is the
ground truth underneath every relational tool. Learn it once; use it forever.

<div class="knowledge-check" data-quiz data-correct-msg="Right — SQL is declarative: you describe the result you want and the database's planner works out how to fetch it." markdown="0">
  <p class="knowledge-check__q">Quick check: what does it mean that SQL is a declarative language?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">You write step-by-step loops telling the database how to scan each table</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">You describe the result you want; the database works out how to produce it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It only works on one specific database and doesn't transfer</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **SQL** (Structured Query Language) is the language for talking to a relational
  database — the way you ask questions of your tables.
- It's **declarative**: you describe the *result* you want, not the step-by-step *how*
  of fetching it.
- The database's **query planner** turns your request into an efficient execution
  strategy, choosing indexes and join order for you.
- SQL statements group into **querying** (`SELECT`), **changing data** (`INSERT`,
  `UPDATE`, `DELETE` — DML), and **defining structure** (`CREATE TABLE` — DDL).
- The core is a shared **standard** across databases, with per-database **dialect**
  differences at the edges — so the skill transfers widely.
- Even with an **ORM**, SQL is the ground truth worth learning once.

Next up: [Querying with SELECT](/learn/databases/querying-with-select/).
