---
slug: performance-and-monitoring
title: Performance tuning & monitoring
description: How to find the slow query before your users do — reading a query plan with EXPLAIN, the metrics that actually matter, the usual suspects behind a sluggish database, and why you measure before you optimise.
keywords: query performance, EXPLAIN, query plan, slow query log, sequential scan, missing index, database metrics, p99 latency, monitoring, N+1, measure before optimising
level: advanced
status: full
prereq:
  - indexes
faq:
  - q: "What does EXPLAIN do?"
    a: "`EXPLAIN` shows the **query plan** — the step-by-step strategy the database will use to run a query: which indexes it uses (or doesn't), how it joins tables, and how many rows it expects at each step. `EXPLAIN ANALYZE` actually runs the query and reports real timings. It's the first tool you reach for when a query is slow, because it tells you *why*."
  - q: "What's the single most common cause of a slow query?"
    a: "A **missing index** causing a full table scan — the database reading every row to find the few it needs. `EXPLAIN` reveals it as a sequential scan over a large table. Adding the right index usually turns a query from seconds to milliseconds, which is why indexes are the first thing to check."
  - q: "Which metrics should I actually watch?"
    a: "Query latency (especially the **p95/p99**, not just the average), throughput (queries per second), error rate, connection-pool usage, cache hit ratio, and replication lag if you have replicas. Averages hide the slow tail; the p99 is what your unhappiest users feel."
---

# Performance tuning & monitoring

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Finding a slow database is a measurement problem, not a guessing game. **`EXPLAIN`**
shows the **query plan** — how the database will actually run a query — and usually
points straight at the culprit, most often a **missing index** causing a full table
scan. Watch the metrics that matter (**p95/p99 latency**, throughput, errors, pool
usage), catch slow queries with a **slow-query log**, and always **measure before you
optimise**. The biggest wins are almost always [indexes](/learn/databases/indexes/)
and killing [N+1 queries](/learn/databases/orms-vs-raw-sql/).
</div>

A database rarely gets slow all at once. It creeps: a table grows, a query that was
fine at a thousand rows crawls at ten million, and one day the dashboard times out.
The skill here is diagnosis — knowing how to ask the database *why* a query is slow
rather than guessing and adding random indexes. This lesson is the toolkit: read the
plan, watch the right numbers, and fix the usual suspects, in that order.

## Measure before you optimise

The first rule is discipline: **don't optimise what you haven't measured.** It is
astonishingly easy to spend a day speeding up a query that runs once a week while the
real problem — a query firing ten thousand times a minute — sits untouched. Optimising
on a hunch usually adds complexity and indexes you don't need, and sometimes makes
things *slower*.

So start by finding *which* query is actually the problem. A **slow-query log** records
every query that takes longer than a threshold, and database statistics views show
which queries consume the most total time (frequency × duration). Let the data point
you at the one query worth your afternoon, then optimise that one.

## Reading a query plan with EXPLAIN

Once you have the offending query, ask the database how it intends to run it. Every
relational database has an **`EXPLAIN`** command that prints the **query plan** — the
strategy the planner chose:

```sql
EXPLAIN ANALYZE
SELECT c.id, c.started_at, t.name
FROM calls c
JOIN talkgroups t ON t.id = c.talkgroup_id
WHERE c.system_id = 7 AND c.started_at > '2026-08-01'
ORDER BY c.started_at DESC
LIMIT 50;
```

`EXPLAIN` shows the plan; `EXPLAIN ANALYZE` actually runs it and reports real
timings. Reading a plan, you're looking for a few tells:

- **Sequential scan (full table scan)** on a big table — the database reading *every*
  row. Fine on a tiny table, a disaster on a huge one. Usually the smoking gun.
- **Index scan** — the database using an index to jump to the rows it needs. What you
  want on your filter and join columns.
- **Row estimates far off from reality** — the planner expected 10 rows and hit a
  million, a sign its statistics are stale.
- **An expensive sort or join** high in the plan — sometimes fixed by an index that
  provides the order, sometimes by rewriting the query.

The plan turns "it's slow" into "it's doing a sequential scan over ten million rows
because there's no index on `system_id`" — a problem you can actually fix.

## The usual suspects

Most database slowness comes from a short list of causes, and knowing them shortcuts
diagnosis:

- **Missing indexes.** By far the most common. A filter or join on an unindexed column
  forces a full scan. Add the right [index](/learn/databases/indexes/) and the query
  often drops from seconds to milliseconds.
- **N+1 queries.** Not one slow query but a flood of small ones — the
  [ORM trap](/learn/databases/orms-vs-raw-sql/) of looping and lazily loading a related
  row per item. Replace the loop with a single join.
- **`SELECT *` and over-fetching.** Pulling every column and every row when you need a
  few. Select only what you use and always `LIMIT` large result sets.
- **Too many indexes.** Indexes speed reads but slow **writes**, since every insert or
  update must maintain them. An over-indexed table is slow to write; indexes have a
  cost, so add them deliberately.
- **Lock contention.** Long transactions holding locks block others. Keep
  transactions short.

## The metrics that matter

You can't watch a database by staring at it; you instrument it and watch trends. The
metrics worth a dashboard and alerts:

- **Query latency — and the *tail*.** Not just the average but the **p95 and p99**: the
  slowest 5% and 1% of queries. Averages hide a slow tail, and the p99 is exactly what
  your most frustrated users experience.
- **Throughput** — queries per second, so you can see load and correlate spikes with
  slowness.
- **Error rate** — failed queries, timeouts, deadlocks.
- **Connection-pool usage** — how often the [pool](/learn/databases/connection-pooling/)
  is saturated and queries wait.
- **Cache hit ratio** — how much is served from memory versus disk; a falling ratio
  often precedes a slowdown.
- **Replication lag** — if you run [replicas](/learn/databases/replication-and-scaling/),
  how far behind they are.

The goal is to see the problem in a graph before it's an incident — the database
version of the observability the AI path describes in
[observability & monitoring](/learn/building-ai/observability-and-monitoring/).

## A tuning loop

Put it together into a repeatable loop: **measure** to find the worst query, **explain**
it to learn why, **change one thing** (add an index, rewrite the query, kill an N+1),
**measure again** to confirm it helped, and stop when it's fast enough. One change at a
time, always verified — the same measure-change-verify discipline as any good
performance work.

<div class="knowledge-check" data-quiz data-correct-msg="Right — EXPLAIN shows the query plan, revealing things like a full table scan from a missing index, so you fix the real cause instead of guessing." markdown="0">
  <p class="knowledge-check__q">Quick check: a query is slow. What's the first thing to reach for?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Add indexes to every column in the table and see if it helps</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Run EXPLAIN to see the query plan and find out why it's slow</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Move the database to a bigger server immediately</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Measure before you optimise** — use a **slow-query log** and stats views to find
  the query that actually matters, then fix that one.
- **`EXPLAIN`** shows the **query plan**; `EXPLAIN ANALYZE` runs it with real timings.
  Look for **sequential (full table) scans**, bad row estimates, and expensive sorts.
- The usual suspects: **missing indexes** (the big one), **N+1 queries**, `SELECT *`
  over-fetching, **too many indexes** slowing writes, and lock contention.
- Watch the metrics that matter — latency at the **p95/p99 tail**, throughput, errors,
  pool usage, cache hit ratio, replication lag — to catch trouble in a graph, not an
  outage.
- Work a **measure → explain → change one thing → measure again** loop, and stop when
  it's fast enough.

Next up: [choosing a database](/learn/databases/choosing-a-database/).
