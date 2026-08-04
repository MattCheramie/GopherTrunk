---
slug: time-series-and-analytics
title: Time-series & analytical stores
description: Databases tuned for two special jobs — time-series stores for a relentless stream of timestamped measurements, and column-oriented analytical stores for crunching huge tables. Learn why your metrics live in one and why analytics doesn't run on your app's database.
keywords: time-series database, analytical database, column store, columnar, OLTP vs OLAP, metrics, InfluxDB, TimescaleDB, data warehouse, aggregation, downsampling, row vs column storage
level: intermediate
status: full
prereq:
  - sql-vs-nosql
  - aggregation-and-grouping
faq:
  - q: What makes a time-series database different from a normal table with a timestamp?
    a: "Time-series data has a specific shape — a huge, append-only stream of timestamped measurements that you almost always query by time range and aggregate. Time-series databases optimize hard for exactly that: fast writes of new points, storage that compresses old data, and built-in time-bucketing and downsampling. You *can* store it in a regular table, but a purpose-built store handles the volume and the time-window queries far better."
  - q: What is a column store and why is it faster for analytics?
    a: "A column store keeps each column's values together on disk instead of each row's values together. Analytical queries usually read a few columns across millions of rows — a sum of one column, say — so reading just those columns, tightly compressed, is dramatically faster than reading whole rows. Row stores win for fetching whole records; column stores win for aggregating across many."
  - q: Why not run analytics on my application's main database?
    a: "Because the workloads conflict. Your app database (OLTP) is tuned for many small, fast reads and writes of individual rows; big analytical scans (OLAP) hammer it and slow down real users. The usual pattern is to copy data into a separate analytical store or warehouse and run the heavy queries there, keeping the two workloads apart."
---

# Time-series & analytical stores

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Two special jobs get their own databases. A **time-series store** is built for a
relentless, append-only stream of **timestamped measurements** — metrics, sensor
readings, decoded-call events — queried by **time range** and aggregated. An
**analytical (column) store** keeps data **column by column** instead of row by row,
which makes crunching a few columns across **millions of rows** vastly faster. The
theme is **OLTP vs. OLAP**: your app's database serves many small transactions;
analytics is a different workload that belongs in a different store, leaning on the
[aggregation](/learn/databases/aggregation-and-grouping/) you already know.
</div>

Key-value and document stores changed the *shape* of your records. This lesson changes
what the database is *optimized for*. Some data isn't a special shape so much as a
special *pattern of use* — a firehose of timestamped points, or an enormous table you
summarize rather than read row by row — and two families of database exist because
that pattern breaks a normal relational store's assumptions.

## Time-series data: a firehose of timestamps

**Time-series data** is measurements stamped with *when they happened*, arriving
continuously and rarely changing once written. Server CPU every second, temperature
every minute, a decoded-call event every time GopherTrunk hears one — all the same
shape: `(timestamp, what, value)`, forever appended, almost never updated.

```
2026-08-04T09:00:00Z  system=metro_p25  calls_active=3
2026-08-04T09:00:01Z  system=metro_p25  calls_active=4
2026-08-04T09:00:02Z  system=metro_p25  calls_active=4
```

You almost always query it the same way too: **over a time range, aggregated** — "average
active calls per minute over the last hour," "peak in the last day." You rarely ask for
one exact point by ID the way you'd fetch one user.

## Why time-series stores exist

You *can* put this in a regular table with a timestamp column, and for modest volumes
that's fine. But at scale the pattern strains a general-purpose database in ways a
**time-series database** (InfluxDB, TimescaleDB, Prometheus, and others) is built to
handle:

- **Write volume.** Points arrive constantly — thousands per second is normal. These
  stores optimize hard for fast, append-only writes.
- **Compression.** Old points are highly repetitive and ordered by time, so they
  compress enormously. A time-series store squeezes months of history into a fraction of
  the space.
- **Time-window queries and downsampling.** Bucketing by minute/hour/day and rolling old
  fine-grained data up into coarser summaries (**downsampling** — keep per-second for a
  day, per-minute for a month) are first-class, built-in operations.
- **Retention.** Automatically expiring data older than N days, because you rarely need
  per-second detail from last year.

This is why **your metrics and monitoring live in one**. Any dashboard of graphs over
time — the subject of [performance & monitoring](/learn/databases/performance-and-monitoring/)
later — is almost certainly backed by a time-series store.

## The other axis: analytical (OLAP) workloads

The second special job is **analytics**: questions that scan huge amounts of data to
produce a summary. "Total call-seconds per system per day across the last year," "the
ten busiest talkgroups this month." These read *many* rows but usually only a *few*
columns, and they don't need to be instant — they need to chew through volume.

This is the classic split between two workload types:

- **OLTP** (Online Transaction Processing) — your application's database. Many small,
  fast reads and writes of individual rows: fetch this user, insert this call, update
  this balance. Tuned for low-latency single-row work.
- **OLAP** (Online Analytical Processing) — analytics and reporting. A few big queries
  that scan and aggregate millions of rows. Tuned for throughput over huge scans.

They pull a database in opposite directions, which is why serious systems keep them
apart.

## Row stores vs. column stores

The technical heart of analytical databases is **how data is laid out on disk**. A
normal (OLTP) database is **row-oriented**: it stores all of row 1's columns together,
then all of row 2's, and so on. That's perfect for "give me this whole call" — one
seek grabs the entire record.

An analytical database is often **column-oriented**: it stores all of the `duration`
values together, all of the `system_id` values together, column by column. Now consider
`SELECT system_id, SUM(duration) ... GROUP BY system_id` over ten million calls. A
column store reads *only* the two columns it needs — tightly packed and heavily
compressed — and skips every other column entirely. A row store would drag every full
row off disk just to use two fields of each.

```
Row store:     [call1: id,sys,tg,dur,ts][call2: id,sys,tg,dur,ts]...   ← reads everything
Column store:  [all system_id][all duration][all timestamp]...          ← reads two columns
```

So the rule of thumb: **row stores win at fetching whole records; column stores win at
aggregating a few columns across many rows.** It's the same data — just organized for
the opposite question.

## Keep the workloads apart

The practical consequence ties it together: **don't run heavy analytics on your live
application database.** A big OLAP scan competes with your users' small OLTP queries for
the same resources and can grind real traffic to a crawl. The standard pattern is to
move data out — copy or stream it into a separate **analytical store** or **data
warehouse** (column-oriented, OLAP-tuned) and run the heavy queries there, where they
can't hurt production.

The takeaway of the whole lesson is that "database" isn't one thing tuned one way. The
*shape* and *pattern* of your data pick the tool: a firehose of timestamps wants a
time-series store, a mountain of rows you summarize wants a column store, and your
transactional app data stays where it is. Recognizing which workload you're looking at
is half of [choosing a database](/learn/databases/choosing-a-database/) well.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a column store reads only the few columns a query touches, tightly compressed, instead of dragging whole rows off disk." markdown="0">
  <p class="knowledge-check__q">Quick check: why is a column store faster for a query that sums one column across millions of rows?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It keeps the whole table in memory at all times</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It stores each column together, so it reads only the columns the query needs, not whole rows</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It skips aggregation entirely and precomputes every possible answer</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Time-series data** is an append-only stream of **timestamped measurements**, queried
  by **time range** and aggregated — metrics, sensors, decoded-call events.
- **Time-series databases** optimize for high write volume, heavy compression of old
  data, built-in **time-bucketing / downsampling**, and automatic retention — which is
  why your **metrics live in one**.
- **OLTP** (your app's database) does many small, fast single-row reads and writes;
  **OLAP** (analytics) does a few huge scans that aggregate millions of rows.
- **Row stores** keep each row together and win at fetching whole records; **column
  stores** keep each column together and win at aggregating a few columns across many
  rows.
- **Keep the workloads apart** — copy data into a separate analytical store/warehouse so
  heavy queries don't starve your live application.

Next up: [Vector databases & similarity search](/learn/databases/vector-databases/).
