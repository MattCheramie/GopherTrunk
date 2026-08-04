---
slug: orms-vs-raw-sql
title: ORMs vs. raw SQL
description: The perennial choice when your code talks to a database — let an ORM map objects to rows and generate SQL for you, or write the SQL by hand — what each approach buys you, what each costs, and how to decide (or blend the two).
keywords: ORM, object-relational mapping, raw SQL, query builder, N+1 problem, leaky abstraction, data mapper, active record, SQL by hand, ORM tradeoffs
level: intermediate
status: full
prereq:
  - querying-with-select
faq:
  - q: "What does an ORM actually do?"
    a: "An **ORM** (object-relational mapper) maps rows in a table to objects in your code, so you work with `user.name` instead of columns and result sets, and it generates the SQL to load and save them. It handles the boilerplate of turning query results into typed objects and back, plus relationships, migrations, and more depending on the library."
  - q: "Is raw SQL faster than an ORM?"
    a: "Often, at the extremes. An ORM's generated SQL is usually fine for ordinary queries, but for complex or performance-critical ones a hand-written query gives you full control and no surprises. The bigger cost of an ORM isn't raw speed — it's losing visibility into what SQL actually runs, which is where problems like N+1 hide."
  - q: "Do I have to pick one for the whole project?"
    a: "No, and most mature codebases don't. A common pattern is to use an ORM or a lightweight mapper for the routine create-read-update-delete work, and drop to raw SQL for the handful of complex reports and hot-path queries where control matters. The two coexist against the same database."
---

# ORMs vs. raw SQL

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
When your code talks to a database you can let an **ORM** map rows to objects and
generate the SQL for you, or write the **raw SQL** yourself. An ORM cuts boilerplate
and keeps you in your language; raw SQL gives full control and no hidden queries.
The real cost of an ORM is **losing sight of the SQL that actually runs** — the
source of traps like the **N+1 problem**. Most codebases **blend** the two. Whichever
you choose, keep queries safe with [parameterised queries](/learn/databases/sql-injection/).
</div>

Once your program is connected and pooled, you have to decide *how* to express
queries in code. There are two schools. One says: work with objects in your
language and let a library translate to SQL. The other says: write the SQL yourself,
it's a perfectly good language. This is one of the oldest debates in software, and
the honest answer is that both are right, for different queries. This lesson gives
you the tradeoffs so you can choose deliberately instead of by habit.

## What an ORM gives you

An **ORM** — object-relational mapper — bridges two worlds: the tables-and-rows of a
database and the objects-and-fields of your program. You define that a `Call` object
corresponds to the `calls` table, and the ORM lets you write code like:

```
call = Call.find(42)
call.talkgroup = "Fireground"
call.save()
```

Behind those three lines the ORM generates a `SELECT ... WHERE id = 42`, tracks that
you changed a field, and generates the matching `UPDATE`. You never wrote SQL, never
mapped columns to fields by hand, and never assembled a result set into an object.

That's the pitch, and it's a strong one:

- **Less boilerplate.** The tedious work of reading rows into typed objects and
  writing objects back is automated.
- **One language.** You stay in your program's types and idioms instead of switching
  to SQL strings.
- **Safety by default.** Good ORMs parameterise every value, so the most common
  injection mistakes don't happen.
- **Extras.** Many bundle migrations, relationship loading, validation, and
  connection handling.

For ordinary create-read-update-delete work over a well-shaped schema, an ORM
removes a great deal of repetitive code.

## What raw SQL gives you

Writing **raw SQL** means composing the query text yourself and mapping the results
into your own structures:

```sql
SELECT c.id, c.started_at, t.name AS talkgroup
FROM calls c
JOIN talkgroups t ON t.id = c.talkgroup_id
WHERE c.system_id = $1 AND c.started_at > $2
ORDER BY c.started_at DESC
LIMIT 50;
```

What you get in return is **control and clarity**:

- **Exactly the query you intend.** No generated SQL, no surprises about what hit the
  database.
- **The full power of SQL.** Complex joins, window functions, CTEs, and
  database-specific features an ORM may not express well.
- **Predictable performance.** You can read the query, run `EXPLAIN` on it, and tune
  it directly — see [performance & monitoring](/learn/databases/performance-and-monitoring/).
- **No extra abstraction to learn or fight.** SQL is the interface, and you already
  know it from this module.

The cost is that you write and maintain the mapping code yourself, and you have to
be disciplined about parameterising every value.

## The leaky abstraction problem

The deepest issue with ORMs is that the database doesn't go away — it's just hidden.
An ORM is a **leaky abstraction**: most of the time it lets you ignore SQL, but the
moment performance matters, the SQL underneath leaks back through and you have to
understand it anyway.

The classic example is the **N+1 problem**. You load a list of 100 calls, then loop
over them printing each call's talkgroup name. If the ORM loads each talkgroup
lazily, that innocent loop fires **1 query for the list plus 100 more** — one per
call — where a single join would have done. The code looks clean; the database is
drowning. You only see it if you look at the SQL the ORM emits.

This is the real tradeoff. An ORM's convenience comes from hiding the SQL, and
hidden SQL is SQL you can't reason about until it hurts. Raw SQL is more work up
front but nothing is hidden.

## Choosing — and blending

You rarely have to pick one for everything, and you usually shouldn't. A common,
pragmatic split:

- **Use an ORM or a light mapper for the routine 90%** — the straightforward inserts,
  lookups, and updates where boilerplate is the only real cost.
- **Drop to raw SQL for the hard 10%** — complex reports, aggregations, and the
  performance-critical queries on your hot path, where control beats convenience.

Between the two sits the **query builder** — a library that helps you assemble SQL
programmatically (composable `where` and `join` calls) while still producing plain
SQL you can see. It gives some of the ORM's ergonomics without hiding the query.

Whatever you choose, two rules don't bend: **always parameterise values** to prevent
injection, and **stay able to see the SQL** so you can debug and tune it. An ORM that
lets you log its generated queries keeps that door open; one that hides them
entirely is a liability. Go's own standard library leans toward the raw-SQL end, as
the [talking to a database from Go](/learn/databases/databases-in-go/) lesson shows.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the N+1 problem: a loop that lazily loads a related row per item fires one query plus one per row, where a single join would do." markdown="0">
  <p class="knowledge-check__q">Quick check: which problem is a classic hazard of ORMs hiding the SQL they run?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">SQL injection from unescaped input</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The N+1 problem — a loop firing one query per row instead of a single join</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Connection pool exhaustion under heavy load</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **ORM** maps rows to objects and generates SQL for you — cutting boilerplate,
  keeping you in your language, and parameterising values by default.
- **Raw SQL** means writing queries by hand for full control, the complete power of
  SQL, and predictable, inspectable performance.
- An ORM is a **leaky abstraction**: the database is hidden, not gone, and the SQL
  leaks back through when performance matters.
- The **N+1 problem** — a loop that lazily loads a related row per item — is the
  classic ORM trap, invisible until you look at the emitted SQL.
- Most codebases **blend** the two: an ORM or mapper for routine work, raw SQL for
  complex and hot-path queries, with a query builder as a middle ground.
- Non-negotiables either way: **parameterise every value**, and **stay able to see
  the SQL**.

Next up: [SQL injection & querying safely](/learn/databases/sql-injection/).
