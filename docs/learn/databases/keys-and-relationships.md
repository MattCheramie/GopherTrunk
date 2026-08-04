---
slug: keys-and-relationships
title: Keys & relationships
description: Primary keys name each row and foreign keys link tables together. Learn how a database models the relationships between your things — one-to-many and many-to-many — and why keys are what make the relational model relational in practice.
keywords: primary key, foreign key, keys, relationships, one-to-many, many-to-many, join table, referential integrity, surrogate key, natural key, relational database
level: beginner
status: full
prereq:
  - the-relational-model
  - schemas-and-types
faq:
  - q: "What's the difference between a primary key and a foreign key?"
    a: "A primary key uniquely identifies each row within its own table — no two rows share it. A foreign key is a column in one table that points at another table's primary key, linking the two. Primary keys name rows; foreign keys connect them."
  - q: "Should a primary key be meaningful data or just a number?"
    a: "Usually just a number (a surrogate key) — an auto-generated id with no meaning of its own. It never has to change, can't collide, and doesn't leak business meaning. Natural keys (like an email) work but tie your row's identity to data that might change, so surrogate keys are the safer default."
  - q: "How do I link two tables where both sides can have many of the other?"
    a: "With a join table — a third table whose rows each pair one key from each side. A call can hit many talkgroups and a talkgroup has many calls, so a call_talkgroups table with one row per pairing models the many-to-many cleanly."
---

# Keys & relationships

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **primary key** uniquely names each row in a table; a **foreign key** is a column
that points at another table's primary key, **linking** the two. Together they model
the **relationships** between your things — **one-to-many** (a system has many
calls) and **many-to-many** (via a **join table**). Keys are what let you split data
across tables without losing the connections, and they set up
[joins](/learn/databases/joins/) later — building on
[schemas & types](/learn/databases/schemas-and-types/).
</div>

Real data isn't one table — it's several, connected. You have systems *and* the
calls on them, users *and* their settings. The relational model splits these into
separate tables, and **keys** are the threads that stitch them back together. This
lesson is about those threads: the **primary key** that names a row and the
**foreign key** that links to it. Get keys right and the rest of relational
databases falls into place.

## Primary keys: naming each row

A **primary key** is a column (or set of columns) whose value **uniquely
identifies** each row — no two rows in the table may share it, and it can't be
NULL. It's the row's name, the handle you use to point at exactly one record. In
the systems table, `id` is the primary key: system 2 is *the* County DMR system,
unambiguously.

You have two flavours to choose from:

- A **natural key** is real data that happens to be unique — an email address, a
  system's callsign. It's meaningful, but it ties the row's identity to data that
  might change or turn out not to be as unique as you thought.
- A **surrogate key** is a made-up identifier with no meaning of its own — usually
  an auto-incrementing integer or a UUID. It never has to change and can't collide.

Most of the time, reach for a surrogate key. An auto-generated `id` never forces
you to update every reference just because someone changed their email:

```sql
CREATE TABLE talkgroups (
    id       INTEGER PRIMARY KEY,   -- surrogate key
    name     TEXT    NOT NULL,
    tg_number INTEGER NOT NULL
);
```

## Foreign keys: linking tables

A **foreign key** is a column in one table that holds the **primary key value of a
row in another table** — a pointer across the gap. Say each decoded call belongs to
one system. The calls table carries a `system_id` foreign key referencing
`systems.id`:

```sql
CREATE TABLE calls (
    id         INTEGER PRIMARY KEY,
    system_id  INTEGER NOT NULL REFERENCES systems(id),
    started_at TIMESTAMP NOT NULL,
    duration_s REAL
);
```

That `REFERENCES systems(id)` does real work. It declares the link *and* enforces
**referential integrity**: the database now refuses to store a call whose
`system_id` doesn't match a real system, and can stop you deleting a system that
still has calls. The relationship isn't just a convention you hope everyone
follows — the database guarantees it. (This is one of several
[constraints](/learn/databases/schemas-and-types/) the database enforces for you.)

## One-to-many: the everyday relationship

The calls-and-systems link above is a **one-to-many** relationship: one system has
many calls, but each call belongs to one system. This is by far the most common
shape in real schemas — a user has many orders, a system has many talkgroups, a
post has many comments.

The rule of thumb: the foreign key lives on the **"many" side**. Each call points
at its one system, so `system_id` sits on the calls table. You never need a list of
call-ids on the system row; to find a system's calls you just ask for every call
whose `system_id` matches — which is exactly what a [join](/learn/databases/joins/)
does.

## Many-to-many: the join table

Some relationships go both ways. A call might be heard on several talkgroups, and a
talkgroup certainly carries many calls — **many-to-many**. A single foreign key
can't express this; you can't fit a list of talkgroups in one column and stay
relational.

The answer is a third table, a **join table** (or junction table), whose every row
pairs one key from each side:

```sql
CREATE TABLE call_talkgroups (
    call_id      INTEGER NOT NULL REFERENCES calls(id),
    talkgroup_id INTEGER NOT NULL REFERENCES talkgroups(id),
    PRIMARY KEY (call_id, talkgroup_id)
);
```

Each row is one pairing: "call 42 involved talkgroup 7." Want a call's talkgroups?
Find the rows with that `call_id`. Want a talkgroup's calls? Find the rows with that
`talkgroup_id`. The many-to-many becomes two clean one-to-many links through the
middle table — the standard, boring, correct way to model it.

## Why keys matter

Splitting data across tables is what keeps it clean — no duplication, each fact in
one place, a design goal we formalise in
[normalization](/learn/databases/normalization/). But splitting only works if you
can reliably *reconnect* the pieces, and keys are that mechanism. Primary keys give
every row a stable name; foreign keys turn those names into enforced links. Without
them you'd have tables that can't refer to each other — a filing cabinet of drawers
with no cross-references. With them, you have a genuine relational database.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a foreign key holds another table's primary key, enforcing the link between the two." markdown="0">
  <p class="knowledge-check__q">Quick check: what is a foreign key?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A password that unlocks a protected table</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A column holding the primary key value of a row in another table, linking them</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The column a table is physically sorted by on disk</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **primary key** uniquely identifies each row in its table — no duplicates, no
  NULL; it's the row's stable name.
- Prefer a **surrogate key** (an auto-generated id) over a **natural key**, so a
  row's identity never has to change.
- A **foreign key** holds another table's primary key value, linking the two and
  enforcing **referential integrity** — no orphan references.
- **One-to-many** is the common case; the foreign key lives on the **many** side (a
  call points at its system).
- **Many-to-many** needs a **join table** with one row per pairing, turning it into
  two one-to-many links.
- Keys are what let you split data cleanly and still reconnect it — the foundation
  for **joins**.

Next up: [What SQL is](/learn/databases/what-is-sql/).
