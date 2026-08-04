---
slug: connection-pooling
title: Connection pools
description: Opening a database connection is expensive, so real applications keep a small pool of connections open and hand them out for each query — what a pool is, why it exists, and how to size one without starving the database or your app.
keywords: connection pool, database pooling, max connections, pool size, connection reuse, idle connection, connection lifetime, pgbouncer, pool exhaustion, sizing a pool
level: intermediate
status: full
prereq:
  - connecting-from-code
faq:
  - q: "Why not just open a connection for each query?"
    a: "Because opening one is slow — a TCP handshake, TLS, and authentication every time — and the database can only handle so many open connections at once. Under load, open-per-query either crawls or overwhelms the server. A **pool** opens a handful once and reuses them, so a query borrows a ready connection instead of paying the setup cost."
  - q: "How big should the pool be?"
    a: "Smaller than you think. A pool of tens, not thousands. The database has a hard cap on total connections, and past a point more connections mean more contention, not more throughput. Size the pool to your database's limit divided across all your app instances, and tune from measurements."
  - q: "What happens when the pool is empty?"
    a: "A query that needs a connection waits for one to be returned, up to a timeout. If every connection is busy and none free up in time, the query fails with a pool-exhaustion or timeout error — usually a sign the pool is too small, queries are too slow, or connections are being held too long."
---

# Connection pools

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Opening a database connection is expensive, so applications keep a **pool** of them
open and lend one out per query, returning it when done. A pool trades a little
memory for a lot of speed and protects the database from being swamped by too many
connections. The main knobs are **pool size**, **idle timeout**, and **connection
lifetime**, and the main failure is **pool exhaustion** — all connections busy at
once. Sizing it well starts from your database's own
[connection limits](/learn/databases/connecting-from-code/).
</div>

The previous lesson showed how much work opening a connection is: a network
handshake, TLS, and authentication, all before a single query runs. Paying that on
every query would be absurd — like re-negotiating a phone contract before each call.
So real applications don't. They open a small set of connections once, keep them
alive, and reuse them. That set is a **connection pool**, and understanding it is
the difference between an app that stays fast under load and one that falls over.

## What a pool is

A **connection pool** is a managed collection of open database connections that your
application holds ready. When code needs to run a query it **borrows** a connection
from the pool, uses it, and **returns** it — the connection stays open for the next
borrower rather than being torn down.

Think of it like a pool of shared bikes at a station. Bikes are expensive to build,
so you don't manufacture one each time someone wants to ride; you keep a rack of
them, riders take one and bring it back, and the rack serves far more trips than it
has bikes. A connection pool works the same way: a handful of connections serves a
stream of queries, because no single query holds one for long.

In most stacks you never manage this by hand. You configure a pool once at startup
and the library hands out connections behind your normal "run this query" calls.

## Why a pool, not a connection per query

Two problems make open-per-query a bad idea, and the pool solves both.

- **Setup cost.** Every open pays for TCP, TLS, and auth — often several
  milliseconds. Under hundreds of queries a second, that overhead dominates. A
  pooled connection has already paid it once, so borrowing is nearly free.
- **The database's own limit.** A database server can only keep so many connections
  open at once — each one costs it memory and a process or thread. Open a fresh
  connection per concurrent request and a traffic spike can blow straight past that
  cap, at which point the server rejects *everyone*. A pool bounds how many
  connections you ever open, so your app can't overwhelm the database.

The second point surprises people: a pool isn't only a speed optimisation, it's a
**safety limit**. It's the throttle that keeps your app a well-behaved client.

## Sizing the pool

The tempting instinct — "more connections, more speed" — is wrong past a small
number. A database does real work per connection, and beyond the point where it can
keep every connection busy, extra ones just add contention and context-switching.
Throughput goes *down*, not up.

The right size is usually **surprisingly small** — often in the tens. Two limits box
it in:

- **The database's max connections.** This is a hard ceiling on the server. Your
  pool — across *every* app instance — must fit under it. Ten app instances each
  with a pool of 20 means 200 connections; the database must allow at least that.
- **How many queries are truly running at once.** A connection is only useful while
  it's executing a query. If your queries are fast, a few connections churn through
  an enormous number of them.

Start from the database's limit, divide it across your instances with headroom to
spare, and then **tune from measurements** — watch how often the pool is full and
how long queries wait, and adjust. Guessing large is the classic mistake.

## The knobs that matter

Beyond raw size, a pool exposes a few settings worth knowing:

- **Max open / max pool size** — the ceiling on total connections. The safety limit
  above.
- **Max idle** — how many connections to keep open when things are quiet. Too low and
  you re-open constantly on the next burst; too high and you hold connections you
  aren't using.
- **Idle timeout** — close a connection that's been unused for this long, so a quiet
  period releases resources back to the database.
- **Max lifetime** — retire and replace a connection after this long, even if it's
  healthy. This lets load rebalance after a failover and sidesteps connections that
  slowly go stale.

## Pool exhaustion

The signature failure is **pool exhaustion**: every connection is checked out and a
new query has nowhere to go. It waits for one to be returned, and if none frees up
within the timeout, it fails.

Exhaustion almost always points at one of three things: the pool is **too small** for
the load, queries are **too slow** (each holds its connection longer), or code is
**holding connections it isn't using** — the classic bug of borrowing a connection,
starting a slow non-database task, and only then returning it. The fix is rarely
"make the pool huge"; it's usually to speed up the queries or shorten how long each
connection is held.

For very high connection counts — many app instances, or serverless functions that
each want a pool — a dedicated **external pooler** (like PgBouncer in front of
Postgres) sits between your apps and the database and multiplexes many client
connections onto a few real ones.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a pool reuses a small set of already-open connections, avoiding the setup cost and bounding how many you ever open." markdown="0">
  <p class="knowledge-check__q">Quick check: why do applications use a connection pool?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">To make queries return more rows at once</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">To reuse a small set of open connections, avoiding per-query setup cost and capping load on the database</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">To encrypt the connection so credentials stay safe</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Opening a connection is expensive, so apps keep a **pool** of open connections and
  **borrow and return** one per query instead of opening a fresh one each time.
- A pool avoids per-query **setup cost** and, just as important, **caps** how many
  connections your app opens — protecting the database from being swamped.
- The right pool size is usually **small** (tens): past the point the database can
  keep them busy, more connections *reduce* throughput.
- Size it from the database's **max connections** divided across your app instances,
  then tune from measurements.
- Know the knobs — **max size, max idle, idle timeout, max lifetime** — and their
  tradeoffs.
- **Pool exhaustion** (all connections busy) usually means the pool is too small,
  queries are too slow, or connections are held too long — not that the pool needs
  to be enormous.

Next up: [ORMs vs. raw SQL](/learn/databases/orms-vs-raw-sql/).
