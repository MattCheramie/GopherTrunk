---
slug: glossary
title: Glossary of database terms
description: Plain-language definitions of every term in the Databases & Data module — database, schema, SQL, primary and foreign key, join, index, normalization, transaction, ACID, NoSQL, vector store, ORM, connection pool, migration, replica, backup, and more — each linked to the lesson that explains it.
keywords: database glossary, SQL terms, primary key, foreign key, join, index, normalization, transaction, ACID, NoSQL, ORM, connection pool, replication, backup, vector database
level: beginner
status: full
lesson_standalone: true
---

# Glossary of database terms

Every term used across the [Databases &amp; Data](/learn/databases/) module, defined
in plain language and linked to the lesson where it's explained in full. Skim it as a
refresher, or use your browser's find (Ctrl/Cmd-F) to jump to a word. Terms are grouped
by theme, roughly in the order the module introduces them.

## Foundations

**Database** — An organised store of data that many readers and writers can query
reliably, safely, and fast, instead of hand-rolling files. See
[What a database is](/learn/databases/what-is-a-database/)

**Persistence** — Data outliving the program that created it, by being written to
durable storage rather than held only in memory. See
[Data, state & persistence](/learn/databases/data-and-persistence/)

**State (in-memory)** — Data a program holds while it runs, which vanishes when the
process stops — the opposite of persisted data. See
[Data, state & persistence](/learn/databases/data-and-persistence/)

**Durability** — The guarantee that once data is saved it survives crashes and power
loss, not just that it was written. See
[Data, state & persistence](/learn/databases/data-and-persistence/)

**Relational model** — Organising data into **tables** of **rows** and **columns**, the
shape that has run the data world for fifty years. See
[The relational model](/learn/databases/the-relational-model/)

**Table / row / column** — A table is a named collection of rows (records); each row is
one item; each column is one field with a fixed type across all rows. See
[The relational model](/learn/databases/the-relational-model/)

**Schema** — The shape of your data: the tables, their columns, the types, and the
rules — decided up front to save a mess later. See
[Schemas, columns & data types](/learn/databases/schemas-and-types/)

**Data type** — The kind of value a column holds — integer, text, timestamp, boolean —
which the database enforces on every row. See
[Schemas, columns & data types](/learn/databases/schemas-and-types/)

**Primary key** — A column (or set) that uniquely names each row in a table, so any row
can be found and referenced unambiguously. See
[Keys & relationships](/learn/databases/keys-and-relationships/)

**Foreign key** — A column that points at another table's primary key, linking rows
across tables and modeling a relationship between things. See
[Keys & relationships](/learn/databases/keys-and-relationships/)

## SQL, the language of data

**SQL** — The declarative language nearly every relational database speaks: you describe
the data you want, and the database works out how to fetch it. See
[What SQL is](/learn/databases/what-is-sql/)

**Declarative** — Saying *what* you want, not *how* to get it — the defining trait of
SQL, which leaves the execution strategy to the database. See
[What SQL is](/learn/databases/what-is-sql/)

**SELECT** — The statement that reads data, asking for specific columns and rows — the
single most-used statement in all of software. See
[Querying with SELECT](/learn/databases/querying-with-select/)

**WHERE** — The clause that filters a query to only the rows matching a condition,
instead of returning the whole table. See
[Filtering, sorting & limiting](/learn/databases/filtering-and-sorting/)

**ORDER BY / LIMIT** — `ORDER BY` sorts the result rows; `LIMIT` caps how many come
back — together, "the newest 50" without dragging the whole table. See
[Filtering, sorting & limiting](/learn/databases/filtering-and-sorting/)

**JOIN** — Combining rows from two tables on a shared key — the heart of relational
power. An **inner join** keeps only matched rows; an **outer (left) join** keeps
unmatched rows too. See [Joining tables](/learn/databases/joins/)

**Aggregation** — Turning many rows into a summary value with functions like `COUNT`,
`SUM`, and `AVG` — the tools behind every dashboard number. See
[Aggregation & GROUP BY](/learn/databases/aggregation-and-grouping/)

**GROUP BY / HAVING** — `GROUP BY` collapses rows into groups to aggregate each one;
`HAVING` filters those groups after aggregating (as `WHERE` filters rows before). See
[Aggregation & GROUP BY](/learn/databases/aggregation-and-grouping/)

**INSERT / UPDATE / DELETE** — The write side of SQL: add new rows, change existing
ones, or remove them — with a `WHERE` clause you forget at your peril. See
[Inserting, updating & deleting](/learn/databases/inserting-updating-deleting/)

**Index** — A secondary structure, like a book's index, that lets the database jump
straight to matching rows instead of scanning the whole table — the difference between
a millisecond and a minute. See [Indexes & how queries get fast](/learn/databases/indexes/)

## Designing data well

**Normalization** — Structuring tables so each fact is stored in exactly one place,
avoiding the duplication that eventually betrays you. See
[Normalization & avoiding duplication](/learn/databases/normalization/)

**Normal form** — One of a graded series of rules (1NF, 2NF, 3NF…) describing how far a
schema removes duplication and update anomalies. See
[Normalization & avoiding duplication](/learn/databases/normalization/)

**Data modeling** — The craft of going from "what does my app do" to a set of tables and
relationships, before writing any code. See
[Data modeling for a real app](/learn/databases/data-modeling/)

**Constraint** — A rule the database enforces on data — `NOT NULL`, `UNIQUE`, `CHECK`,
and foreign keys — that keeps bad data out no matter what the app does. See
[Constraints & data integrity](/learn/databases/constraints-and-integrity/)

**Data integrity** — The property that the data in a database stays valid and consistent
with its rules, upheld by constraints. See
[Constraints & data integrity](/learn/databases/constraints-and-integrity/)

**Transaction** — A group of changes treated as one all-or-nothing unit: either every
change applies or none does. See [Transactions & ACID](/learn/databases/transactions-and-acid/)

**ACID** — The four guarantees behind a transaction — **Atomicity** (all or nothing),
**Consistency** (rules always hold), **Isolation** (concurrent transactions don't
corrupt each other), **Durability** (committed changes survive). See
[Transactions & ACID](/learn/databases/transactions-and-acid/)

**Migration** — A versioned, repeatable change to a live database's schema, letting you
evolve its shape safely without losing data. See
[Schema migrations](/learn/databases/migrations/)

## Beyond relational

**NoSQL** — An umbrella for non-relational stores that trade some of SQL's structure and
guarantees for a different shape, scale, or flexibility. See
[SQL vs. NoSQL](/learn/databases/sql-vs-nosql/)

**Key-value store** — A dictionary-like database that stores values behind unique keys,
built for fast lookups by known key. See
[Key-value & document stores](/learn/databases/key-value-and-document/)

**Document store** — A NoSQL database that stores self-contained, often JSON documents
that can vary in shape, fetched whole. See
[Key-value & document stores](/learn/databases/key-value-and-document/)

**Time-series database** — A store tuned for when-it-happened data — metrics, events,
sensor readings — written in time order and queried by range. See
[Time-series & analytical stores](/learn/databases/time-series-and-analytics/)

**Column store / analytical (OLAP)** — A database that stores data by column for fast
aggregation over huge tables — where your metrics and analytics live. See
[Time-series & analytical stores](/learn/databases/time-series-and-analytics/)

**Vector database** — A store that indexes **embeddings** (vectors) to search by meaning
— nearest-neighbour lookups — the store behind AI retrieval and RAG. See
[Vector databases & similarity search](/learn/databases/vector-databases/)

**Embedding** — A list of numbers representing the meaning of a piece of data, so
similar meanings sit close together in vector space. See
[Vector databases & similarity search](/learn/databases/vector-databases/) and
[Embeddings & vector search](/learn/building-ai/embeddings-and-vector-search/)

**Similarity search** — Finding the items whose vectors are nearest to a query vector —
searching by meaning instead of exact keywords. See
[Vector databases & similarity search](/learn/databases/vector-databases/)

**Cache / caching layer** — Fast in-memory storage in front of the database that holds
hot data so repeated reads skip the slower database. See
[Caching layers](/learn/databases/caching-layers/)

**Cache invalidation** — Keeping cached data fresh — deciding when a cached value is
stale and must be refreshed — the classic hard problem of caching. See
[Caching layers](/learn/databases/caching-layers/)

## Using a database from code

**Driver** — The library that speaks a specific database's wire protocol, turning your
query into bytes on a socket and the reply back into values. See
[Connecting from your program](/learn/databases/connecting-from-code/)

**Connection string / DSN** — The string that says where a database is and who is
connecting — host, port, database, user, password, and options like TLS. See
[Connecting from your program](/learn/databases/connecting-from-code/)

**Credentials** — The username and password (or token) that authenticate a connection —
secrets to load from the environment, never hard-code. See
[Connecting from your program](/learn/databases/connecting-from-code/)

**Connection pool** — A managed set of open connections the app borrows and returns per
query, avoiding per-query setup cost and capping load on the database. See
[Connection pools](/learn/databases/connection-pooling/)

**Pool exhaustion** — The failure where every pooled connection is busy at once, so new
queries wait and may time out — usually a sign of slow queries or a too-small pool. See
[Connection pools](/learn/databases/connection-pooling/)

**ORM (object-relational mapper)** — A library that maps table rows to objects in your
code and generates the SQL to load and save them, cutting boilerplate. See
[ORMs vs. raw SQL](/learn/databases/orms-vs-raw-sql/)

**Raw SQL** — Writing query text by hand and mapping the results yourself, for full
control and no hidden queries. See [ORMs vs. raw SQL](/learn/databases/orms-vs-raw-sql/)

**N+1 problem** — The ORM trap where a loop lazily loads a related row per item, firing
one query plus one per row instead of a single join. See
[ORMs vs. raw SQL](/learn/databases/orms-vs-raw-sql/)

**SQL injection** — A vulnerability where untrusted input is glued into a query and gets
executed as SQL, letting an attacker read, alter, or destroy data. See
[SQL injection & querying safely](/learn/databases/sql-injection/)

**Parameterised query / prepared statement** — A query with placeholders whose values
are sent separately, so input is always bound as data and can never become SQL — the
fix for injection. See [SQL injection & querying safely](/learn/databases/sql-injection/)

**database/sql** — Go's standard-library interface to SQL databases; you register a
driver and run parameterised queries against a pooled `sql.DB`. See
[Talking to a database from Go](/learn/databases/databases-in-go/)

**sql.DB** — In Go, a **connection pool** (not a single connection), safe for concurrent
use, opened once and shared for the program's life. See
[Talking to a database from Go](/learn/databases/databases-in-go/)

**Scan** — In Go, copying the columns of a result row into your variables, row by row,
in a `rows.Next()` loop. See [Talking to a database from Go](/learn/databases/databases-in-go/)

## Running a database in production

**Backup** — A copy of your data you can restore from after failure — the operational
floor beneath everything, worthless until you've actually restored it. See
[Backups & recovery](/learn/databases/backups-and-recovery/)

**Point-in-time recovery (PITR)** — Restoring to *any* moment by replaying a change log
on top of a base backup — including the instant before a mistake. See
[Backups & recovery](/learn/databases/backups-and-recovery/)

**RPO / RTO** — Recovery **point** objective (how much recent data you can afford to
lose) and recovery **time** objective (how long you can be down) — the targets that
shape a backup strategy. See [Backups & recovery](/learn/databases/backups-and-recovery/)

**Replication** — Keeping copies of a database in sync with the original, for read
capacity and failover. See [Replication, sharding & scaling](/learn/databases/replication-and-scaling/)

**Primary / read replica** — The **primary** takes all writes; **replicas** receive every
change and serve reads (and stand by for failover). Writes still funnel through the one
primary. See [Replication, sharding & scaling](/learn/databases/replication-and-scaling/)

**Replication lag** — The small delay before a replica reflects a write, so a replica
read can miss a just-written value — **eventual consistency**. See
[Replication, sharding & scaling](/learn/databases/replication-and-scaling/)

**Sharding** — Splitting the data itself across servers on a **shard key** to scale
**writes**, at real cost in cross-shard queries and lost cross-shard transactions. See
[Replication, sharding & scaling](/learn/databases/replication-and-scaling/)

**Vertical / horizontal scaling** — Scaling **up** (a bigger server) versus **out** (more
servers). Scale up first; scale out only when forced. See
[Replication, sharding & scaling](/learn/databases/replication-and-scaling/)

**CAP theorem** — During a network partition, a distributed database must trade
**consistency** (never serve stale data) against **availability** (always answer) — you
can't have both. See [Replication, sharding & scaling](/learn/databases/replication-and-scaling/)

**EXPLAIN / query plan** — `EXPLAIN` prints the **query plan** — the strategy the
database will use to run a query — so you can see *why* it's slow. See
[Performance tuning & monitoring](/learn/databases/performance-and-monitoring/)

**Full table scan (sequential scan)** — The database reading every row to satisfy a
query, usually because an index is missing — the most common cause of slowness. See
[Performance tuning & monitoring](/learn/databases/performance-and-monitoring/)

**p95 / p99 latency** — The slowest 5% and 1% of query times — the tail that averages
hide and your unhappiest users feel; the metric to watch. See
[Performance tuning & monitoring](/learn/databases/performance-and-monitoring/)

**Slow-query log** — A record of every query slower than a threshold, used to find which
query to optimise before guessing. See
[Performance tuning & monitoring](/learn/databases/performance-and-monitoring/)

**Managed database** — A database a cloud provider or service (Postgres/MySQL, Supabase,
Neon…) runs for you — handling backups, patching, and failover — versus self-hosting.
See [Choosing a database](/learn/databases/choosing-a-database/)

**Choose boring technology** — The principle that for something as important as your
data you pick the mature, well-understood option (often Postgres or SQLite) and diverge
only for a concrete need. See [Choosing a database](/learn/databases/choosing-a-database/)

**Retention sweep** — A periodic job that deletes data older than a configured window to
bound a long-running database's growth — as GopherTrunk does with old calls. See
[Data in GopherTrunk](/learn/databases/data-in-gophertrunk/)
