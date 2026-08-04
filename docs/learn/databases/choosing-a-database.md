---
slug: choosing-a-database
title: Choosing a database
description: A practical framework for picking the right database for a project — SQLite, Postgres, MySQL, a document store, or a managed service like Supabase — starting from your data shape and access patterns rather than hype.
keywords: choosing a database, SQLite, PostgreSQL, MySQL, document database, managed database, Supabase, boring technology, database selection, default to Postgres
level: intermediate
status: full
prereq:
  - sql-vs-nosql
faq:
  - q: "What database should I default to?"
    a: "For most applications, **Postgres** — a mature, free, relational database that does an enormous amount well, from ordinary tables to JSON documents and vector search. Defaulting to Postgres and only diverging when you have a concrete reason is a strategy that rarely goes wrong. For a small local or embedded app, **SQLite** is the even simpler default."
  - q: "When is SQLite actually the right choice?"
    a: "More often than people think. SQLite is a full SQL database that lives in a single file inside your process — no server to run. It's ideal for local apps, desktop and mobile software, tests, prototypes, and even modest web apps with light write concurrency. You reach for a client-server database when many machines must write concurrently at scale."
  - q: "Should I pick the newest, most exciting database?"
    a: "Usually not. Databases hold your most important asset — your data — and maturity, stability, and a deep pool of documentation and hiring matter more than novelty. 'Choose boring technology' is genuinely good advice here: pick the well-understood option unless a specific need forces something specialised."
---

# Choosing a database

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Choosing a database starts from your **data shape and access patterns**, not from
hype. For most projects the right default is **Postgres** (or **SQLite** for a small,
local, or embedded app) — mature, relational, and capable far beyond plain tables.
Reach for a specialised store only when a concrete need — a document model, extreme
scale, time-series, vectors — forces it. **Choose boring technology**: your data is
too important to bet on novelty. This ties together everything from
[SQL vs. NoSQL](/learn/databases/sql-vs-nosql/) onward.
</div>

By now you've met relational databases, key-value and document stores, time-series
and column stores, vector databases, and caches. The natural question is: for *my*
project, which one? The honest answer is less exciting than the marketing suggests —
most projects are best served by a well-understood relational database, and the skill
is knowing the few situations that genuinely call for something else. This lesson is a
framework for deciding, and a strong default so you're not choosing from a blank page.

## Start from your data, not the tool

The wrong way to choose a database is to pick a technology you've heard is fast or
web-scale and then bend your problem to fit it. The right way is to look at your data
first and let it point you:

- **What shape is the data?** Rows with clear relationships between things (users have
  calls, calls belong to systems)? That's relational. Self-contained documents that
  vary in shape? Maybe a document store. Just values behind keys? Key-value.
- **How will you read it?** Rich queries, joins, and reports across the data pull
  strongly toward SQL. A single known-key lookup is happy in almost anything.
- **How much, and how fast will it grow?** Thousands of rows and a hobbyist's traffic
  need nothing exotic. Billions of writes a day is a different conversation.
- **What consistency do you need?** Money and bookings want strong transactional
  guarantees; a metrics feed can tolerate eventual consistency.

Answer those and the field narrows itself. The [data modeling](/learn/databases/data-modeling/)
and [SQL vs. NoSQL](/learn/databases/sql-vs-nosql/) lessons are the deeper version of
this step.

## A strong default: Postgres (or SQLite)

If you take one thing from this lesson: **default to Postgres, and diverge only for a
reason.** Postgres is a free, mature, relational database that quietly does a
staggering amount — solid SQL and transactions, JSON columns when you want document-ish
flexibility, full-text search, and even vector search through an extension. It scales
far past where most projects ever reach, and it has decades of documentation, tooling,
and people who know it.

For a smaller footprint, **SQLite** is the even simpler default. It's a complete SQL
database in a single file, embedded in your process with no server to run or secure —
perfect for local apps, desktop and mobile software, tests, and prototypes, and
capable of running real web apps with modest write concurrency. GopherTrunk itself
leans on SQLite for exactly these reasons, as the next lesson shows.

Between them, these two cover the overwhelming majority of projects. Starting from one
of them is rarely a decision you'll regret.

## When to reach for something else

Diverge from the default when you have a **concrete, specific need** that a specialised
store serves markedly better — the kinds of jobs Unit 4 covered:

- **A genuine document model** — deeply nested, schema-varying documents you always
  fetch whole — can fit a [document store](/learn/databases/key-value-and-document/).
- **Massive time-stamped data** — metrics, sensor readings, events — belongs in a
  [time-series or column store](/learn/databases/time-series-and-analytics/) tuned for it.
- **Search by meaning** for AI retrieval calls for a
  [vector database](/learn/databases/vector-databases/) or a vector-capable Postgres.
- **A hot, simple, high-throughput cache or key-value need** suits Redis or a
  [caching layer](/learn/databases/caching-layers/).
- **Truly extreme write scale** beyond one machine is where distributed and NoSQL
  systems and [sharding](/learn/databases/replication-and-scaling/) earn their
  complexity.

Notice these are *needs*, not preferences. "I might go web-scale someday" is not one of
them; you can migrate when the need is real and measured.

## Run it yourself or use a managed service

Separate from *which* database is *who operates it*. Running a production database well
— backups, upgrades, replication, monitoring, security — is real, ongoing work. A
**managed database** (a cloud provider's Postgres/MySQL, or a service like Supabase,
Neon, or PlanetScale) hands that operational burden to a vendor: they handle patching,
backups, and failover, and you pay for it and give up some low-level control.

For most teams, especially small ones, a managed service is the right call — your time
is better spent on your product than on database administration. Run your own only when
cost, control, or specific requirements justify the operational load. The tradeoff is
the same managed-versus-self one that runs through all of
[deployment](/learn/deployment/).

## Choose boring technology

The thread through all of this: **your data is your most important and least
replaceable asset, so choose conservatively.** A database's maturity, stability,
documentation, and the number of people who know it matter more than benchmarks or
novelty. The newest database with the best landing page is the one you'll be debugging
alone at 3am with no Stack Overflow answers. "Choose boring technology" is not
cynicism — it's how you keep your data safe and your future self sane. Pick the
well-understood option, and let a specific, demonstrated need be the only thing that
moves you off it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — start from your data shape and access patterns, default to a mature relational database like Postgres, and diverge only for a concrete need." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the soundest way to choose a database for a new project?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Pick the newest, most hyped database so you're ready for web scale</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Start from your data shape and access patterns; default to a mature option like Postgres and diverge only for a concrete need</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Always use whichever NoSQL store is fastest in benchmarks</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Choose from your **data shape, access patterns, scale, and consistency needs** —
  not from hype or a technology you want to try.
- **Default to Postgres** — mature, relational, and capable far beyond tables — or
  **SQLite** for small, local, or embedded apps; together they fit most projects.
- Reach for a **specialised store** (document, time-series, vector, cache, distributed)
  only when a **concrete need** genuinely calls for it — not "someday" scale.
- Decide separately whether to **self-host or use a managed service**; for most teams a
  managed database is the better use of time.
- **Choose boring technology** — your data is too important to bet on novelty; pick the
  well-understood option and diverge only for a demonstrated reason.

Next up: [data in GopherTrunk](/learn/databases/data-in-gophertrunk/).
