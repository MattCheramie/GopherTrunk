---
slug: constraints-and-integrity
title: Constraints & data integrity
description: Constraints let the database enforce your data's rules no matter what code writes to it. Learn NOT NULL, UNIQUE, CHECK, primary-key, and foreign-key constraints, and why the database is a safer place for a rule than your application.
keywords: constraints, data integrity, NOT NULL, UNIQUE constraint, CHECK constraint, foreign key constraint, referential integrity, primary key, database rules, validation, cascade
level: intermediate
status: full
prereq:
  - keys-and-relationships
  - data-modeling
faq:
  - q: Why enforce a rule in the database if my app already checks it?
    a: "Because the database is the one place every write must pass through. App checks can be bypassed — by another service, a bug, a manual query, or a second app hitting the same database. A constraint is enforced no matter what code does the writing, so it can't be skipped by accident."
  - q: What's the difference between a primary key and a UNIQUE constraint?
    a: "Both guarantee no duplicate values. A primary key is the row's official identifier — one per table, and it can't be null. A UNIQUE constraint enforces uniqueness on some other column (like an email), can be applied to several columns, and typically does allow nulls. Use the primary key for identity, UNIQUE for other must-be-distinct fields."
  - q: What does a foreign key with ON DELETE CASCADE do?
    a: "It tells the database that when a parent row is deleted, its child rows should be deleted too — automatically. It's one way to keep referential integrity when removing data. The alternatives are to block the delete (RESTRICT) or set the child's reference to null (SET NULL); which you want depends on what the relationship means."
---

# Constraints & data integrity

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Constraints** are rules you declare *in the schema* so the **database itself**
enforces them on every write — not your app, which can be bypassed. The core set is
**NOT NULL** (a value is required), **UNIQUE** (no duplicates), **CHECK** (a value
must satisfy a condition), **PRIMARY KEY** (unique identity), and **FOREIGN KEY**
(a reference must point at a row that exists). Together they guarantee **data
integrity** — that the data in your tables is always valid — building directly on
[keys & relationships](/learn/databases/keys-and-relationships/).
</div>

You've modeled your tables and normalized them. But a schema that merely *allows* good
data isn't the same as one that *guarantees* it. Constraints are how you move a rule
out of hopeful application code and into the database, where it's enforced on every
single write, forever, no matter who or what is doing the writing.

## The rule that matters: put integrity where writes can't dodge it

Here's the core argument. Your application validates input — good. But your database
is very likely written to by more than just that one code path: a second service, a
background job, a data-migration script, a developer running SQL by hand at 2am, or a
future rewrite in another language. Every one of those can *skip* your app's checks.
Not one of them can skip a **constraint**, because a constraint lives in the table and
the database refuses any write that violates it.

So the principle is: **the database is the last line of defense, and the only one
every write must cross.** App-level validation is still worth having — it gives users
friendly, immediate errors — but it's a convenience on top of the constraint, not a
replacement for it. This is the same defense-in-depth thinking as
[secure coding](/learn/cybersecurity/secure-coding/): don't trust that callers behaved.

## NOT NULL: a value is required

The simplest constraint. **NOT NULL** says this column must always have a value —
inserting or updating a row without one is rejected. `NULL` in SQL means "unknown or
absent," and it's a frequent source of bugs (it isn't equal to anything, not even
another `NULL`). Marking the columns that *must* be present keeps those surprises out
of the data entirely.

```sql
CREATE TABLE calls (
    call_id    INTEGER PRIMARY KEY,
    system_id  INTEGER NOT NULL,     -- every call must belong to a system
    started_at TIMESTAMP NOT NULL,   -- every call has a start time
    notes      TEXT                  -- optional: null is allowed here
);
```

Decide for each column whether absence is meaningful. A call with no start time is
nonsense — `NOT NULL`. A call with no notes is fine — leave it nullable.

## UNIQUE: no duplicates

A **UNIQUE** constraint forbids two rows from sharing the same value in a column (or
combination of columns). It's how you enforce that emails, usernames, or a system's
external ID are distinct:

```sql
CREATE TABLE users (
    user_id INTEGER PRIMARY KEY,
    email   TEXT NOT NULL UNIQUE      -- no two users share an email
);
```

This overlaps with the **primary key**, which is also unique — but they play different
roles. The primary key is the row's one official **identity**, one per table, never
null. A `UNIQUE` constraint enforces distinctness on *other* fields, can be declared on
several columns at once (a composite uniqueness rule), and usually permits nulls. Use
the primary key for identity; use `UNIQUE` for every other "these can't repeat" rule.

## CHECK: the value has to make sense

A **CHECK** constraint enforces an arbitrary condition on a row's values — anything you
can write as a boolean expression. It's how you stop impossible data at the door:

```sql
CREATE TABLE calls (
    call_id     INTEGER PRIMARY KEY,
    duration_ms INTEGER NOT NULL CHECK (duration_ms >= 0),
    frequency   BIGINT  NOT NULL CHECK (frequency BETWEEN 25000000 AND 1300000000)
);
```

A negative duration or a frequency outside the radio spectrum you support is now
simply *impossible to store*. `CHECK` is the constraint people reach for least and
benefit from most: it encodes your domain's real limits into the schema, so a bug that
tries to write garbage fails loudly instead of quietly corrupting a row.

## FOREIGN KEY: references must point at something real

A **foreign key** constraint enforces **referential integrity** — the guarantee that a
reference actually points at a row that exists. If `calls.system_id` is a foreign key
into `systems`, the database will reject any call whose `system_id` has no matching
system, and it will refuse to delete a system that still has calls pointing at it
(unless you tell it otherwise). No more **orphan rows** referring to things that were
never there or have since vanished.

```sql
CREATE TABLE calls (
    call_id   INTEGER PRIMARY KEY,
    system_id INTEGER NOT NULL REFERENCES systems(system_id)
);
```

You control what happens to children when a parent is deleted with a **referential
action**:

- **ON DELETE RESTRICT** (often the default) — block the delete while children exist.
  Safe: you must clean up children first.
- **ON DELETE CASCADE** — delete the children automatically with the parent. Convenient
  for truly owned data (delete a system, delete its calls), dangerous if you didn't
  mean it.
- **ON DELETE SET NULL** — keep the child but null out its reference. For optional links
  where the child outlives the parent.

Which you choose is a statement about what the relationship *means*, so choose it
deliberately rather than accepting the default by accident.

## Integrity is a property you declare once

The payoff of all this is that **data integrity stops being a thing you remember to do
and becomes a property the schema holds automatically.** Once the constraints are in
place, invalid data can't enter through any door. Bad writes fail with a clear error
at the moment they happen — right where the bug is — instead of surfacing as
mysterious wrong answers weeks later. When you combine constraints with the all-or-
nothing guarantee of the next lesson, you get a database that keeps itself correct even
under concurrent, partial, and failing writes.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a constraint is enforced by the database on every write, so no code path, script, or second app can bypass it." markdown="0">
  <p class="knowledge-check__q">Quick check: your app validates input, so why also add a constraint in the database?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Constraints make queries run faster than app-side checks</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Every write must pass the database, but app checks can be bypassed by other code, scripts, or services</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The database can validate data that application code is unable to check</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Constraints** move a rule into the schema so the **database enforces it on every
  write** — the one path no code can skip.
- **NOT NULL** requires a value; use it for every column where absence would be
  nonsense.
- **UNIQUE** forbids duplicates on non-identity columns like email; the **primary key**
  is the row's single official, non-null identity.
- **CHECK** enforces any condition on a row's values, encoding your domain's real limits
  so impossible data can't be stored.
- **FOREIGN KEY** enforces referential integrity — references must point at real rows —
  with **RESTRICT / CASCADE / SET NULL** deciding what happens to children on delete.
- App validation is a friendly extra; the constraint is the guarantee. Together they
  make **data integrity** a property the schema holds by itself.

Next up: [Transactions & ACID](/learn/databases/transactions-and-acid/).
