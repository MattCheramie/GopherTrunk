---
slug: migrations
title: Schema migrations & evolving a database
description: A migration is a versioned, ordered change to a database's schema that you run to move it from one shape to the next. Learn why schema changes belong in version control, how up and down migrations work, and how to change a live database without losing data.
keywords: migrations, schema migration, database migration, schema evolution, version control database, up migration, down migration, migration tool, ALTER TABLE, zero downtime migration, backwards compatible
level: intermediate
status: full
prereq:
  - schemas-and-types
  - data-modeling
faq:
  - q: Why not just change the schema by hand when I need to?
    a: "Because a hand-run ALTER TABLE leaves no record of what changed, can't be reproduced on another environment, and drifts out of sync between your laptop, staging, and production. A migration is a checked-in file that any environment can apply in the same order, so every database ends up with the identical, known schema."
  - q: What are up and down migrations?
    a: "The up migration applies a change — add a column, create a table. The down migration reverses it — drop that column, drop the table — so you can roll back if something goes wrong. Each migration is a matched pair, and the tool tracks which ones have run so it applies each exactly once."
  - q: How do I change a schema on a live app without downtime?
    a: "Make the change in backwards-compatible steps. Add new columns as nullable or with a default, deploy code that writes both old and new, backfill existing rows, then switch reads over and finally remove the old column in a later migration. Each step leaves the running app working against the schema as it is at that moment."
---

# Schema migrations & evolving a database

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **migration** is a versioned, ordered change to your database's schema, kept as a
file **in version control** and applied by a tool that records which migrations have
already run. Migrations turn schema changes into something **reproducible** across
every environment, **reviewable** like code, and **reversible** via paired **up** and
**down** steps. On a live database you change the schema in **backwards-compatible**
stages so the running app never breaks. This is how the design from
[data modeling](/learn/databases/data-modeling/) keeps evolving safely.
</div>

Your [schema](/learn/databases/schemas-and-types/) is never truly finished. Features
get added, columns get renamed, tables get split — the shape of your data changes for
as long as the app lives. The question is *how* you make those changes: as risky,
undocumented, one-off surgery, or as ordinary, version-controlled steps. Migrations
are the second answer, and they're one of the habits that separates a hobby database
from a production one.

## The problem with changing schemas by hand

Suppose you `ALTER TABLE` on your laptop to add a column. It works. But now your
database and everyone else's have quietly diverged: your teammate's copy doesn't have
the column, staging doesn't, production doesn't. There's no record of *what* you
changed or *why*, no way to reproduce it reliably, and no way to review it before it
hits real data. Multiply that across months and a team, and your environments drift
into subtly different shapes — the source of "works on my machine" bugs that are
miserable to track down.

The fix is to treat a schema change exactly like a code change: **write it down, check
it in, and apply it the same way everywhere.**

## A migration is a checked-in, ordered change

A **migration** is a small file describing one schema change, stored in your
repository alongside your code. Each has a version — usually a timestamp or sequence
number — that fixes its place in an **ordered** list. A migration **tool** keeps a
little table in the database recording which migrations have already run, so when you
say "migrate," it applies exactly the ones that haven't, in order, and stops.

```sql
-- 20260804_add_recorded_flag.up.sql
ALTER TABLE calls ADD COLUMN recorded BOOLEAN NOT NULL DEFAULT false;
```

Because every environment applies the *same files in the same order*, they all
converge on the *same* schema. New developer joining? They run the migrations from
zero and get a database identical to production's shape. That reproducibility is the
whole point.

## Up and down: applying and reversing

Most migrations come as a matched pair. The **up** migration makes the change; the
**down** migration undoes it, so you can **roll back** if a deploy goes wrong:

```sql
-- up: apply the change
ALTER TABLE calls ADD COLUMN recorded BOOLEAN NOT NULL DEFAULT false;

-- down: reverse it
ALTER TABLE calls DROP COLUMN recorded;
```

The tool tracks a "current version" and moves you forward or backward through the
ordered list. Forward applies pending ups; rolling back one step runs the matching
down. Not every change is cleanly reversible — a down that drops a column throws away
the data in it — so a good down migration is a *best effort* to restore the previous
shape, and you lean on [backups](/learn/deployment/backups-and-data/) for the cases a
rollback can't recover.

## The dangerous part: changing a live database

Adding a table nobody uses yet is easy. The hard migrations change data or columns
that a **running application** is actively reading and writing. The trap is a change
that's fine *after* it finishes but breaks the app *during* the moment old code meets
new schema, or new code meets old schema.

The rule is to make every change **backwards-compatible** at each step, so at no point
is the deployed code out of step with the schema as it exists right then. A rename —
which looks trivial — is the classic example of what *not* to do in one shot, because
the instant you rename a column, the old still-running code that references the old
name breaks.

## The expand-and-contract pattern

The standard way to make a breaking change safely is to split it into small,
individually-safe steps — often called **expand and contract**. To rename `label` to
`talkgroup_label`:

1. **Expand.** Add the new `talkgroup_label` column (nullable, or with a default). The
   old column still exists; nothing reading `label` breaks.
2. **Backfill and dual-write.** Deploy code that writes *both* columns, and run a
   migration to copy existing `label` values into `talkgroup_label`. Now both are
   populated and correct.
3. **Migrate reads.** Deploy code that reads the new column. The old one is now unused
   but still present, so a rollback is still safe.
4. **Contract.** In a *later* migration, once you're confident, drop the old `label`
   column.

Each step leaves a database and an app that agree with each other, so you can deploy —
and roll back — at any point without a broken window. The same shape works for
splitting a table, changing a type, or adding a `NOT NULL` constraint to existing data
(add nullable, backfill, then enforce). It's more steps than a raw `ALTER`, and that's
exactly why it's safe.

## Migrations are part of your deploy

The last habit: migrations run as a **step in your deployment**, in order, before or
alongside the code that depends on them — never as a manual afterthought someone might
forget. Combined with backups and a rollback plan, that makes evolving a live database
a routine, low-drama event instead of a held-breath one. Your schema stays a living,
version-controlled artifact that any environment can rebuild exactly, which is the
foundation everything operational later in this module — replication, backups,
scaling — is built on.

<div class="knowledge-check" data-quiz data-correct-msg="Right — checked-in, ordered migrations mean every environment applies the same changes and converges on the same schema, reviewably and reversibly." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the main reason to use migration files instead of running ALTER TABLE by hand?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">ALTER TABLE is slower than a migration tool</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Checked-in, ordered migrations are reproducible across every environment and reviewable like code</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Migration files let you skip writing SQL entirely</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **migration** is a versioned, ordered schema change kept **in version control** and
  applied by a tool that records which have already run.
- Migrations make schema changes **reproducible** across environments, **reviewable**
  like code, and **reversible** through paired **up** and **down** steps.
- **Up** applies a change; **down** reverses it — though some downs can't recover data,
  which is what backups are for.
- Changing a **live** database is the dangerous part: keep every step
  **backwards-compatible** so the running app never meets a schema it doesn't expect.
- The **expand-and-contract** pattern splits a breaking change (like a rename) into
  add, backfill, dual-write, switch reads, then drop — each step individually safe.
- Run migrations **as part of your deploy**, in order, not as a manual afterthought.

Next up: [SQL vs. NoSQL](/learn/databases/sql-vs-nosql/).
