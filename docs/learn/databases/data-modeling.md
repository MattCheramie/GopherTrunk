---
slug: data-modeling
title: Data modeling for a real app
description: Data modeling is the craft of turning what an app does into a set of tables and relationships. Learn to find entities from the nouns, relationships from the verbs, and how to model one-to-many and many-to-many links before writing code.
keywords: data modeling, database design, entities, relationships, entity relationship, one-to-many, many-to-many, junction table, schema design, modeling data, ERD, database schema
level: intermediate
status: full
prereq:
  - keys-and-relationships
  - normalization
faq:
  - q: Where do I start when modeling a new app?
    a: "With the nouns. Write down the important things your app talks about — users, orders, systems, calls — and each distinct kind of thing usually becomes a table (an entity). Then look at the verbs connecting them to find the relationships. You can sketch this on paper before touching a database."
  - q: What's a junction table and when do I need one?
    a: "A junction (or join) table sits between two tables to model a many-to-many relationship — where each side can link to many of the other. It holds a foreign key to each side, turning one many-to-many link into two clean one-to-many links. You need one whenever both sides can have many of the other."
  - q: Should I model everything perfectly up front?
    a: "No. Model the core entities and relationships you understand now, keep it normalized, and expect it to evolve. Migrations exist precisely because schemas change — a good model is one that's clear today and easy to extend, not one that predicts every future need."
---

# Data modeling for a real app

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Data modeling** turns "what does my app do" into a set of **tables and
relationships** before you write code. The reliable method: **entities come from the
nouns**, **relationships come from the verbs**, and each relationship is
**one-to-one**, **one-to-many**, or **many-to-many** — the last of which needs a
**junction table**. It's the same keys-and-foreign-keys machinery from
[keys & relationships](/learn/databases/keys-and-relationships/), applied on purpose
to a real problem, and kept clean with
[normalization](/learn/databases/normalization/).
</div>

Normalization told you how to arrange columns *correctly*. Data modeling is the step
before that: deciding what the tables even are. It's the part of database work that
feels most like design rather than mechanics, and doing it well up front saves you
from fighting your own schema for months. The good news is there's a dependable
recipe.

## Start with the nouns

Write one or two plain sentences describing what your app does, then **underline the
nouns**. Each important, distinct kind of thing is an **entity**, and most entities
become a table.

> "A scanner decodes **calls** on trunked **systems**. Each call belongs to a
> **talkgroup** and may be recorded to an audio **file**."

The nouns — *call*, *system*, *talkgroup*, *file* — are your candidate tables. Give
each one a **primary key** to name its rows and the columns that are genuinely *its
own* attributes. A system has a name and a WACN; a call has a start time and a
duration. Attributes that turn out to belong to a *different* noun are exactly the
duplication normalization warns about — they go with their real owner.

## Find relationships in the verbs

Now look at the **verbs** connecting those nouns: a call *belongs to* a system, a
talkgroup *is heard on* a system, a call *is recorded to* a file. Each verb is a
**relationship**, and every relationship has a **cardinality** — how many of each side
connect to the other. There are three shapes, and telling them apart is the whole
game.

## One-to-many: the workhorse

A **one-to-many** relationship means one row on one side links to many rows on the
other, but each of those links back to just one. One system has many calls; each call
belongs to exactly one system. This is by far the most common shape.

You model it by putting a **foreign key on the "many" side**, pointing at the "one":

```sql
CREATE TABLE systems (
    system_id  INTEGER PRIMARY KEY,
    name       TEXT NOT NULL
);

CREATE TABLE calls (
    call_id    INTEGER PRIMARY KEY,
    system_id  INTEGER NOT NULL REFERENCES systems(system_id),  -- the "many" side
    talkgroup  TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL
);
```

The foreign key always lives on the many side. A quick test: ask "does *this* thing
belong to *one* of the other?" If a call belongs to one system, the call row carries
the `system_id`.

## Many-to-many: use a junction table

Sometimes both sides can have many of the other. A **talkgroup** might be patched
across several systems, and a system carries many talkgroups — that's a
**many-to-many** relationship, and you *cannot* model it with a single foreign key on
either side, because neither side has just one of the other.

The answer is a third table — a **junction table** (also called a join, link, or
bridge table) — that holds a foreign key to each side. Each row is one pairing:

```sql
CREATE TABLE talkgroups (
    talkgroup_id INTEGER PRIMARY KEY,
    label        TEXT NOT NULL
);

CREATE TABLE system_talkgroups (        -- the junction table
    system_id    INTEGER NOT NULL REFERENCES systems(system_id),
    talkgroup_id INTEGER NOT NULL REFERENCES talkgroups(talkgroup_id),
    PRIMARY KEY (system_id, talkgroup_id)
);
```

A junction table turns one messy many-to-many into two clean one-to-many
relationships — system-to-link and talkgroup-to-link — which is why it's the standard
solution. It's also the natural home for facts *about the pairing itself*, like when a
talkgroup was first heard on that system.

## One-to-one: less common, real enough

A **one-to-one** relationship links exactly one row to one row. You reach for it when
you want to split a table — say, to keep a large, rarely-read blob (a call's full audio
metadata) out of the hot main row, or to separate optional data. It's modeled with a
foreign key on one side that's also marked **unique**, so at most one match exists. Use
it sparingly; often a one-to-one is really just more columns on the same table.

## Sketch it, then check it against the questions

Before you commit, draw the entities as boxes and the relationships as lines — an
**entity-relationship diagram**, even a rough one on paper. Then pressure-test the
model by asking the real questions your app needs to answer: "show every call on this
system in the last hour," "list the talkgroups this system carries," "find the file
for this call." Each question should be answerable by following keys and joins you've
actually modeled. If a question forces an awkward workaround, the model is telling you
something's missing.

Finally, sanity-check against normalization: is any fact stored twice? If a system's
name appears in the `calls` table, back it out. A model that's clean here is a model
that stays trustworthy as data pours in.

## Model for change, not for prophecy

You will not get it perfectly right, and you don't need to. Model the entities and
relationships you understand *now*, keep them normalized, and lean on
[migrations](/learn/databases/migrations/) to evolve the schema as the app grows. A
good model is clear today and cheap to extend tomorrow — not an attempt to foresee
every feature. The recipe stays the same at every scale: nouns become entities, verbs
become relationships, and many-to-many gets a junction table.

<div class="knowledge-check" data-quiz data-correct-msg="Right — when both sides can have many of the other, you need a junction table holding a foreign key to each side." markdown="0">
  <p class="knowledge-check__q">Quick check: a talkgroup can be on many systems and a system carries many talkgroups. How do you model that?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Put a foreign key to systems on the talkgroups table</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Add a junction table with a foreign key to each side</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Store a comma-separated list of system IDs in a talkgroup column</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Data modeling** turns what an app does into **tables and relationships** before
  you write code.
- **Entities come from the nouns**; each distinct kind of thing becomes a table with
  its own primary key and its own attributes.
- **Relationships come from the verbs**, and each has a cardinality: one-to-one,
  one-to-many, or many-to-many.
- **One-to-many** — the common case — is a **foreign key on the "many" side**.
- **Many-to-many** needs a **junction table** holding a foreign key to each side, which
  also stores facts about the pairing.
- **Sketch the model, test it against the real questions**, check it for duplication,
  and expect to evolve it with migrations rather than predicting the future.

Next up: [Constraints & data integrity](/learn/databases/constraints-and-integrity/).
