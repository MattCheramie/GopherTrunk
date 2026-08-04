---
slug: data-in-gophertrunk
title: Data in GopherTrunk
description: A worked example that ties the whole module together — how a trunking scanner like GopherTrunk models systems, talkgroups, units, and decoded calls into tables, stores them in embedded SQLite, indexes them for search and playback, and sweeps old data away.
keywords: GopherTrunk data model, SQLite call log, scanner database, systems talkgroups calls schema, foreign keys, indexes, retention sweep, worked example, embedded database
level: advanced
status: full
prereq:
  - choosing-a-database
---

# Data in GopherTrunk

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
This is the module's worked example: [GopherTrunk](/) decodes radio traffic into a
stream of **calls**, and it stores them in an **embedded SQLite** database. The domain
maps cleanly onto the relational model — **systems**, **talkgroups**, **units**, and
**calls** become tables joined by **foreign keys** — with **indexes** for the searches
the UI runs and a **retention sweep** to bound growth. Every idea from this module —
schema, keys, joins, indexes, transactions, and choosing the right database — shows up
here in one real system. See the [architecture](/architecture.html) for the full picture.
</div>

We'll finish where the module started — with a real program that needs to remember
things — and watch every concept line up against it. [GopherTrunk](/) is a software
radio scanner: it tunes trunked radio systems, decodes the control channel, follows
voice calls, and records them. That produces a steady stream of events that have to be
stored so you can search, replay, and analyse them later. This is the data problem, and
it's a good one because it's neither trivial nor exotic — exactly the kind most software
faces.

## The domain, as things and relationships

Before any SQL, the [data-modeling](/learn/databases/data-modeling/) step: what are the
*things*, and how do they relate? Listening to the scanner, they fall out naturally:

- A **system** — a trunked radio network (a county's P25 system, say), with a name and
  identifiers.
- A **talkgroup** — a logical channel within a system (Fire Dispatch, Police North).
  Each talkgroup **belongs to** one system.
- A **unit** — an individual radio (a **radio ID**, or RID) heard on the system.
- A **call** — a single voice transmission: which system and talkgroup, which unit
  keyed up, when it started and stopped, the frequency, whether it was encrypted, and a
  path to the recorded audio.

The relationships are plain English: a system **has many** talkgroups; a talkgroup
**has many** calls; a call is **made by** a unit. That "has many / belongs to" language
is the tell that this is [relational](/learn/databases/the-relational-model/) data —
tables joined by [keys](/learn/databases/keys-and-relationships/).

## The schema

Each thing becomes a table, each relationship a **foreign key**. Here is the shape,
trimmed to the essentials:

```sql
CREATE TABLE systems (
    id          INTEGER PRIMARY KEY,
    name        TEXT    NOT NULL,
    protocol    TEXT    NOT NULL          -- 'P25', 'DMR', 'NXDN', ...
);

CREATE TABLE talkgroups (
    id          INTEGER PRIMARY KEY,
    system_id   INTEGER NOT NULL REFERENCES systems(id),
    tgid        INTEGER NOT NULL,          -- the on-air talkgroup number
    label       TEXT,
    UNIQUE (system_id, tgid)               -- a TGID is unique within its system
);

CREATE TABLE calls (
    id            INTEGER PRIMARY KEY,
    system_id     INTEGER NOT NULL REFERENCES systems(id),
    talkgroup_id  INTEGER          REFERENCES talkgroups(id),
    unit_rid      INTEGER,                 -- the radio that keyed up
    started_at    INTEGER NOT NULL,        -- unix time
    ended_at      INTEGER,
    frequency_hz  INTEGER NOT NULL,
    encrypted     INTEGER NOT NULL DEFAULT 0,
    audio_path    TEXT
);
```

Notice how much the schema enforces on its own, exactly as the
[constraints](/learn/databases/constraints-and-integrity/) lesson argued: `NOT NULL`
means a call can never exist without a start time or a frequency, `REFERENCES` means a
call can't point at a system that doesn't exist, and the `UNIQUE (system_id, tgid)`
means the same talkgroup number can't be registered twice for one system. The database
refuses bad data regardless of any bug in the decoder feeding it. The design is
[normalized](/learn/databases/normalization/): a system's name lives in exactly one row,
so renaming it is a single update, not a hunt through every call.

## Querying it

The web dashboard's questions become [joins](/learn/databases/joins/) and
[filters](/learn/databases/filtering-and-sorting/). "Show the last 50 calls on this
system, newest first, with talkgroup labels":

```sql
SELECT c.started_at, t.label, c.unit_rid, c.frequency_hz, c.audio_path
FROM calls c
LEFT JOIN talkgroups t ON t.id = c.talkgroup_id
WHERE c.system_id = ?
ORDER BY c.started_at DESC
LIMIT 50;
```

It's a `LEFT JOIN` because a call might be heard before its talkgroup is known, and we
still want to show it. "How busy was each talkgroup today?" is
[aggregation](/learn/databases/aggregation-and-grouping/):

```sql
SELECT t.label, COUNT(*) AS calls, SUM(c.ended_at - c.started_at) AS air_seconds
FROM calls c
JOIN talkgroups t ON t.id = c.talkgroup_id
WHERE c.started_at > ?
GROUP BY t.label
ORDER BY calls DESC;
```

Every dashboard number in the UI is a query of this kind — the same `SELECT`, `JOIN`,
`WHERE`, `GROUP BY` you've built up all module.

## Making it fast — and safe from Go

A scanner running for weeks accumulates a lot of calls, and the common queries all
filter by system and sort by time. So the [indexes](/learn/databases/indexes/) follow
the access patterns:

```sql
CREATE INDEX idx_calls_system_time ON calls (system_id, started_at DESC);
CREATE INDEX idx_calls_talkgroup   ON calls (talkgroup_id);
```

With those, "last 50 calls on system 7" is an index lookup, not a scan over every call
ever recorded — the difference between an instant page and a spinner. The storage layer
writes from Go using [`database/sql`](/learn/databases/databases-in-go/) with
**parameterised queries** throughout, so an operator-supplied search term is bound as a
value and can never become SQL — [injection](/learn/databases/sql-injection/) closed by
construction:

```go
rows, err := db.QueryContext(ctx,
    `SELECT started_at, unit_rid, audio_path
     FROM calls WHERE system_id = ? AND started_at > ?
     ORDER BY started_at DESC LIMIT ?`,
    systemID, since, limit)
```

Recording a completed call — insert the row, update the talkgroup's last-heard time —
happens in a [transaction](/learn/databases/transactions-and-acid/) so a crash can never
leave a half-written call.

## Why SQLite, and keeping it bounded

GopherTrunk [chooses](/learn/databases/choosing-a-database/) **embedded SQLite** for its
call log, and the reasoning is a clean instance of that lesson. The scanner is often a
single machine — a mini-PC or a Raspberry Pi in a closet — with one writer (the decode
engine) and light read traffic (an operator's dashboard). There's no need for a separate
database server to install, secure, and keep running; SQLite is a single file in the
process, which means zero operational overhead and trivial backups (copy the file). It's
the right default for exactly this shape of workload.

Left alone, that file would grow forever, so a **retention sweep** runs periodically to
delete calls (and their audio) older than a configured window — bounding the database
the way any long-running system must. Backing it up is as simple as the
[backups](/learn/databases/backups-and-recovery/) lesson's copy-and-verify, and if a
deployment ever outgrows one machine, the same relational model ports to Postgres with
the [migrations](/learn/databases/migrations/) discipline. The
[architecture overview](/architecture.html) shows the storage layer in the context of
the whole decode pipeline.

That's the module in one system: a real domain modeled as tables, tied together by keys,
queried with joins and aggregation, kept fast with indexes and safe with constraints and
parameterised Go, stored in a database chosen to fit — and bounded so it runs for years.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a call belongs to one system and one talkgroup via foreign keys, so a call can never reference a system that doesn't exist and shared facts live in one place." markdown="0">
  <p class="knowledge-check__q">Quick check: in GopherTrunk's schema, how is a call linked to its system and talkgroup?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">By copying the full system and talkgroup names into every call row</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">By foreign keys — the call stores the system and talkgroup ids, joined back when needed</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">By storing all calls for a system inside a single JSON blob</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- GopherTrunk's domain — **systems, talkgroups, units, calls** — is naturally
  [relational](/learn/databases/the-relational-model/), modeled as tables joined by
  **foreign keys**.
- The **schema** lets the database enforce the rules — `NOT NULL`, `REFERENCES`, and
  `UNIQUE` [constraints](/learn/databases/constraints-and-integrity/) keep bad data out
  — and is [normalized](/learn/databases/normalization/) so each fact lives once.
- The dashboard's questions are ordinary **joins, filters, and aggregations**, made fast
  with [indexes](/learn/databases/indexes/) that follow the access patterns.
- Writes go through Go's `database/sql` with **parameterised queries** and
  **transactions** — [injection](/learn/databases/sql-injection/)-safe and atomic.
- **Embedded SQLite** is the deliberate [choice](/learn/databases/choosing-a-database/)
  for a single-machine, single-writer workload, with a **retention sweep** to bound
  growth and a trivial file-copy backup.

Next up: keep the [glossary](/learn/databases/glossary/) handy, and see how it all ships in [Containers &amp; Deployment](/learn/deployment/).
