---
slug: sql-vs-nosql
title: SQL vs. NoSQL
description: NoSQL is the umbrella for databases that dropped the rigid relational table for a different data model and different tradeoffs. Learn what NoSQL actually means, the flexibility-for-guarantees trade it makes, and how to tell when a non-relational store fits a problem better than tables.
keywords: SQL vs NoSQL, NoSQL, relational vs non-relational, schema flexibility, eventual consistency, horizontal scaling, key-value, document database, BASE, CAP theorem, when to use NoSQL
level: intermediate
status: full
prereq:
  - the-relational-model
  - transactions-and-acid
faq:
  - q: What does NoSQL actually mean?
    a: "It's an umbrella term for databases that don't use the rigid relational table-and-SQL model — key-value stores, document stores, wide-column, graph, and more. The name is misleading: it means 'not only SQL,' not 'no SQL,' and NoSQL databases vary wildly among themselves. What they share is having dropped some relational assumption in exchange for a different data model or scaling story."
  - q: Is NoSQL faster or more scalable than SQL?
    a: "Sometimes, for specific shapes of work — many NoSQL stores are built to scale horizontally across many machines and to serve simple lookups very fast. But it's not a free upgrade: they usually give up rich joins, strict schemas, or strong transactional guarantees to get there. It's a different set of tradeoffs, not a strictly better database."
  - q: How do I choose between them?
    a: "Start relational unless you have a concrete reason not to — the guarantees, joins, and mature tooling of SQL fit most applications. Reach for a specific NoSQL store when your data or access pattern genuinely matches what it's built for: a document store for self-contained documents, a key-value store for a cache or session data, a time-series store for metrics. Choose the store for the job, not the label."
---

# SQL vs. NoSQL

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**NoSQL** is an umbrella for databases that dropped the rigid **relational**
table-and-SQL model for a different one — **key-value**, **document**, wide-column,
graph. The name means "not only SQL," and these stores differ enormously among
themselves. The common thread is a **trade**: they relax something relational
databases guarantee — strict schema, rich joins, or strong
[ACID](/learn/databases/transactions-and-acid/) — usually to gain schema flexibility
or the ability to scale across many machines. The right question is never "SQL or
NoSQL" in the abstract but **which store fits this job**.
</div>

You've spent this module deep in the relational model — tables, keys, SQL, ACID. It
has run the data world for fifty years because its guarantees are genuinely valuable.
But it isn't the only way to store data, and this unit surveys the alternatives. The
place to start is understanding what "NoSQL" even means, because it's one of the most
misunderstood terms in software.

## "NoSQL" is a terrible name for a real idea

Taken literally, "NoSQL" sounds like a rejection of SQL. It isn't. The term is best
read as **"not only SQL"** — a loose label for the wave of databases that, starting
around 2009, chose *not* to be a relational store with a SQL front end. It groups
together things with almost nothing in common:

- **Key-value stores** — a giant dictionary: one key, one blob of value.
- **Document stores** — collections of self-contained JSON-like documents.
- **Wide-column stores** — rows with flexible, sparse sets of columns, built for scale.
- **Graph databases** — nodes and edges, built for richly interconnected data.

Calling these one thing is like calling every vehicle that isn't a sedan "NotSedan."
So the first correction: **there is no single "NoSQL database"** to compare against
SQL. There are specific stores with specific shapes, and the useful comparisons are
always against one of them — which the next lessons make.

## What relational databases give you

To see what NoSQL trades away, name what you're trading. A mature relational database
hands you, out of the box:

- **A strict schema** — every row has the same typed columns, with
  [constraints](/learn/databases/constraints-and-integrity/) keeping data valid.
- **Joins** — combine data from many tables on shared keys, cheaply and expressively.
- **Strong transactions (ACID)** — all-or-nothing changes with real consistency and
  isolation.
- **A declarative query language** — SQL, understood everywhere, with decades of tools.

These aren't small things. For most applications they're exactly what you want, which
is why "start relational" is sound default advice.

## What NoSQL trades to get flexibility and scale

NoSQL stores relax one or more of those in exchange for something else. The two things
they're usually buying are **schema flexibility** and **horizontal scale**.

**Schema flexibility.** A document store lets each record have a different shape — no
migration to add a field, just write it. That's liberating for fast-moving or
irregular data, and it moves the burden of consistency from the database into your
code. The schema doesn't disappear; it just stops being enforced for you.

**Horizontal scale.** Relational databases traditionally scale *up* (a bigger server)
and are harder to spread across many machines, partly *because* joins and strict
transactions are expensive to coordinate across a network. Many NoSQL stores were built
from the start to **shard** data across many commodity machines — to scale *out*. To
make that work cheaply, they often loosen the strongest guarantees.

## Strong vs. eventual consistency

The deepest part of the trade is **consistency**. A single relational node gives you
**strong consistency**: once a write commits, every subsequent read sees it. Spreading
data across many machines makes that expensive, so some distributed stores offer
**eventual consistency** instead: a write propagates to the copies *over time*, and for
a brief window different replicas can return different answers, until they converge.

This is the heart of the famous **CAP theorem**: when a network partition splits your
machines, a distributed store can stay **consistent** or stay **available**, but not
both — it has to choose. Relational databases traditionally lean toward consistency;
many NoSQL stores lean toward availability and accept eventual consistency. Neither is
wrong; they suit different problems. A bank ledger wants strong consistency; a "last
seen online" timestamp is perfectly happy being eventually consistent. This distinction
is why NoSQL guarantees are sometimes summarized as **BASE** (basically available, soft
state, eventually consistent) in contrast to ACID.

## Choosing: match the store to the job

Put it together and the decision rule is simple to state:

- **Default to relational.** Unless you have a concrete reason otherwise, a SQL database
  (Postgres, SQLite, MySQL) is the safe, powerful, well-understood choice, and its
  guarantees save you from bugs you'd otherwise write yourself.
- **Reach for a specific NoSQL store when your data or access pattern truly matches
  it.** A document store when your data is naturally self-contained documents; a
  key-value store for a cache, session, or feature flag; a time-series store for
  metrics; a vector store for similarity search. Each of those is a lesson in this
  unit.
- **You can use both.** Real systems commonly run a relational database as the source of
  truth *and* a NoSQL store for a job it's better at — a cache in front, a search index
  beside. It's not either/or.

The mistake is choosing by hype or by label. The right frame is the one the whole
[choosing a database](/learn/databases/choosing-a-database/) lesson later develops:
what does *this* data look like, how will you *query* it, and what must you
*guarantee*? Answer those, and the store picks itself.

<div class="knowledge-check" data-quiz data-correct-msg="Right — NoSQL means 'not only SQL,' a diverse family that trades relational guarantees for flexibility or scale; it's not a single faster database." markdown="0">
  <p class="knowledge-check__q">Quick check: which best describes what "NoSQL" means?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A single, newer database that is strictly faster than SQL databases</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">An umbrella for diverse non-relational stores that trade some relational guarantees for flexibility or scale</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Any database that doesn't let you write queries at all</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **NoSQL** means "not only SQL" — an umbrella over **key-value, document, wide-column,
  and graph** stores that have little in common except *not* being relational.
- There's no single "NoSQL database" to compare with SQL; useful comparisons are always
  against a *specific* store.
- Relational databases give you a **strict schema, joins, strong ACID transactions, and
  SQL** — genuinely valuable defaults.
- NoSQL stores **trade** some of those for **schema flexibility** or **horizontal
  scale**, often accepting **eventual** instead of strong consistency (the **CAP**
  tradeoff).
- **Default to relational**, reach for a specific NoSQL store when the data and access
  pattern truly fit it, and freely **use both** in one system.

Next up: [Key-value & document stores](/learn/databases/key-value-and-document/).
