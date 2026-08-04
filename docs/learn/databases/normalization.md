---
slug: normalization
title: Normalization & avoiding duplication
description: Normalization is the practice of storing every fact in exactly one place so it can never disagree with itself. Learn the update, insert, and delete anomalies that duplication causes, and the first three normal forms that fix them.
keywords: normalization, database normalization, normal forms, first normal form, second normal form, third normal form, 3NF, data duplication, update anomaly, redundancy, functional dependency, denormalization
level: intermediate
status: full
prereq:
  - keys-and-relationships
  - the-relational-model
faq:
  - q: What problem does normalization actually solve?
    a: "Duplication. When the same fact is copied into many rows, the copies can drift out of sync, and you get update, insert, and delete anomalies. Normalization stores each fact once, so there is only one copy to keep correct."
  - q: How far should I normalize?
    a: "Third normal form (3NF) is the practical target for almost all everyday application data, and getting to it is mostly common sense once you can spot duplication. Higher normal forms exist but rarely change a design that is already in 3NF."
  - q: Is denormalization ever right?
    a: "Yes — deliberately, for performance, after you have a normalized design and a measured reason. Denormalization trades safe, single-copy data for faster reads, and you take on the job of keeping the copies in sync yourself. Normalize first; denormalize only when a real workload demands it."
---

# Normalization & avoiding duplication

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Normalization** is the discipline of storing every fact in **exactly one
place**. Duplicated data eventually **disagrees with itself** — the same customer
name spelled two ways, a price updated in one row but not another — which produces
**update, insert, and delete anomalies**. The fix is to split data into tables
until each fact lives once, linked back together with the **keys** from
[keys & relationships](/learn/databases/keys-and-relationships/). For everyday
application data, **third normal form (3NF)** is the target.
</div>

The previous unit gave you tables, keys, and SQL. This unit is about using them
*well* — turning a workable pile of columns into a sound design. The first and most
important idea is normalization, and its whole motivation fits in one sentence: a
fact stored twice is a fact that can eventually be wrong in one of the two places.

## The one-sentence rule

**Store each fact exactly once.** That is normalization in a nutshell. Everything
below is just the mechanics of achieving it. A customer's email address, a system's
control-channel frequency, a product's price — each of these is a single fact about
a single thing, and it should live in exactly one row of one table. When you need it
elsewhere, you *point* at it with a key rather than copying it.

The opposite of this is **redundancy**: the same fact physically written into many
rows. Redundancy feels harmless — even convenient — right up until two copies stop
agreeing, and then you have data you can no longer trust.

## What duplication does to you

Imagine a single flat table that records decoded calls, and jams the *system* details
into every call row:

| call_id | system_name | system_wacn | talkgroup | started_at |
|---|---|---|---|---|
| 1 | Metro P25 | 0xBEE00 | Fire Dispatch | 09:00 |
| 2 | Metro P25 | 0xBEE00 | EMS | 09:01 |
| 3 | Metro P25 | 0xBEE0**8** | Police | 09:02 |

The system's WACN is copied into every call. Look at row 3: someone fat-fingered it,
and now the database says Metro P25 has two different WACNs. Which is right? The table
can't tell you. This is the disease normalization cures, and it shows up as three
classic **anomalies**:

- **Update anomaly.** The system gets renamed. You must find and change *every* call
  row that mentions it. Miss one, and the data contradicts itself.
- **Insert anomaly.** You want to record a newly discovered system that has no calls
  yet. In this design you can't — there's no row to put it in without inventing a fake
  call.
- **Delete anomaly.** You purge old calls and accidentally delete the last row that
  mentioned a system. The system's details vanish with it, even though you only meant
  to remove call history.

Every one of these disappears the moment each fact lives in exactly one place.

## First normal form: one value per cell

**First normal form (1NF)** is the price of admission to the relational model: every
cell holds a **single, atomic value**, and there are no repeating groups. No
comma-separated lists stuffed into one column, no `talkgroup1`, `talkgroup2`,
`talkgroup3` columns marching across the table.

If a call can involve several talkgroups, you don't cram them into one cell:

```sql
-- NOT 1NF: a list hidden in a text column
talkgroups TEXT   -- "Fire Dispatch, EMS, Police"
```

You give each value its own row in a related table instead. A list in a cell is
invisible to the query planner — you can't index it, join on it, or filter it
cleanly — so 1NF is what makes the rest of SQL actually work on your data.

## Second and third normal form: facts belong to their key

Once every table has a **primary key**, the next two forms are both variations on a
single idea: **every non-key column must describe the whole key, and nothing but the
key.**

**Second normal form (2NF)** rules out a column that depends on only *part* of a
composite key. If a table's key is `(system_id, talkgroup_id)` but it also stores
`system_name`, that name depends on `system_id` alone — half the key — so it's
misplaced. It belongs in a `systems` table keyed by `system_id`.

**Third normal form (3NF)** rules out a column that depends on another *non-key*
column. If a `calls` table stores `system_id` and also `system_wacn`, the WACN really
depends on the system, not on the call — it's a fact *about the system* that has
leaked into the call row. Move it to `systems` and let the calls point at it:

```sql
CREATE TABLE systems (
    system_id   INTEGER PRIMARY KEY,
    name        TEXT NOT NULL,
    wacn        INTEGER NOT NULL      -- the WACN lives here, once
);

CREATE TABLE calls (
    call_id     INTEGER PRIMARY KEY,
    system_id   INTEGER NOT NULL REFERENCES systems(system_id),
    talkgroup   TEXT NOT NULL,
    started_at  TIMESTAMP NOT NULL
    -- no system_name, no wacn: point at systems instead
);
```

Now the WACN exists in one row. Rename the system, correct its WACN, or add a system
with no calls yet — each is a single change in one obvious place, and no anomaly is
possible.

## Getting to 3NF is mostly common sense

You don't need to recite the formal definitions to normalize well. The instinct is
this: whenever you notice a value being **repeated across rows**, ask "what is this a
fact *about*?" If it's a fact about something other than this row's identity, it wants
its own table, referenced by a key. A repeated system name is a fact about the system;
a repeated author name is a fact about the author. Pull it out, key it, point at it.

Third normal form is the sweet spot for nearly all everyday data. There are higher
forms (BCNF, 4NF, and beyond) that handle subtle key overlaps, but a clean 3NF design
is already free of the anomalies that bite real applications, and pushing further
rarely changes anything.

## When to denormalize on purpose

Normalization optimises for **correctness**; it can cost you **read speed**, because
answering a question may now require joining several tables back together. Sometimes,
for a hot query on a large table, that cost matters. **Denormalization** is the
deliberate choice to reintroduce some duplication — a cached count, a copied name — to
make reads faster.

The key word is *deliberate*. You normalize first, get a clean design, and denormalize
only where a **measured** workload demands it — and when you do, you accept the job of
keeping the duplicated copies in sync, usually with the tools from later lessons like
[transactions](/learn/databases/transactions-and-acid/) or application logic. A
duplicated fact you *chose*, know about, and maintain is fine. A duplicated fact you
stumbled into is the bug this whole lesson is about.

<div class="knowledge-check" data-quiz data-correct-msg="Right — storing a fact once means there's only ever one copy to keep correct, so it can't disagree with itself." markdown="0">
  <p class="knowledge-check__q">Quick check: what is normalization fundamentally trying to prevent?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Tables from ever growing beyond a few thousand rows</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The same fact being stored in two places where the copies can disagree</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Queries from ever needing to join more than one table</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Normalization** means storing every fact **exactly once**, so no two copies can
  drift apart.
- Duplication causes **update, insert, and delete anomalies** — data that contradicts
  itself, facts you can't record, and facts you lose by accident.
- **First normal form**: one atomic value per cell, no lists or repeating column
  groups.
- **Second and third normal form**: every non-key column must describe the **whole
  key and nothing but the key** — pull out facts that really belong to something else.
- **3NF** is the practical target for everyday data, and reaching it is mostly a habit
  of spotting repeated values and giving them their own keyed table.
- **Denormalize only on purpose** — for a measured performance need, accepting that
  you now maintain the duplicate yourself.

Next up: [Data modeling for a real app](/learn/databases/data-modeling/).
