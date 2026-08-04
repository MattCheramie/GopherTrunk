---
slug: replication-and-scaling
title: Replication, sharding & scaling
description: When one database server isn't enough — read replicas that copy your data for more read capacity and failover, sharding that splits data across servers for write capacity, and the CAP tradeoff that shapes every distributed database.
keywords: replication, read replica, primary replica, failover, sharding, partitioning, horizontal scaling, vertical scaling, CAP theorem, replication lag, eventual consistency
level: advanced
status: full
prereq:
  - transactions-and-acid
faq:
  - q: "What's the difference between vertical and horizontal scaling?"
    a: "**Vertical scaling** means a bigger server — more CPU, RAM, and faster disks for the one database. It's simple and gets you a long way, but there's a ceiling and it's a single point of failure. **Horizontal scaling** means more servers — replicas and shards — which has no hard ceiling but adds real complexity. Most systems scale up first and out only when they must."
  - q: "What is a read replica?"
    a: "A copy of your database that continuously receives every change from the primary, kept nearly in sync. Your app sends writes to the primary and spreads reads across the replicas, multiplying read capacity. Replicas also serve as failover targets: if the primary dies, a replica can be promoted to take its place."
  - q: "What is the CAP theorem in one line?"
    a: "When a distributed database is partitioned — some nodes can't reach others — it must choose between staying **consistent** (refuse to serve possibly-stale data) or staying **available** (serve what it has, possibly stale). You can't have both during a partition, so every distributed store makes this tradeoff on purpose."
---

# Replication, sharding & scaling

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
When one server isn't enough you scale **up** (a bigger machine) until you must scale
**out** (more machines). **Replication** copies your data to **read replicas** —
multiplying read capacity and giving you a **failover** target — but writes still go
to one **primary**. **Sharding** splits the data itself across servers to scale
**writes**, at real cost in complexity. And the **CAP theorem** says a partitioned
distributed database must trade **consistency** against **availability**. Fast reads
also lean on a [caching layer](/learn/databases/caching-layers/) in front of all this.
</div>

For a long time, the answer to "my database is slow" is a better query or a bigger
machine, and this lesson's honest first message is that you should exhaust those
before reaching for anything here. But every single server has a ceiling, and past
it you need more than one. Spreading a database across machines buys capacity and
survivability — and introduces the hardest tradeoffs in all of data systems. This
lesson maps the terrain so you know which tool solves which problem.

## Scale up before you scale out

There are two directions to grow.

- **Vertical scaling (scale up)** — give the one server more resources: more CPU
  cores, more RAM, faster disks. It's the simplest lever, requires no application
  changes, and modern hardware takes you remarkably far. Its limits are a hard
  ceiling (the biggest machine you can buy) and that it's still a **single point of
  failure**.
- **Horizontal scaling (scale out)** — add more servers and spread the work across
  them. There's no fixed ceiling, but every distributed technique below adds
  operational and correctness complexity.

The seasoned instinct is **up first, out only when forced.** A single well-tuned
database with good indexes and a cache in front handles more than most projects will
ever need. Scale out because you've hit a real wall, not because it sounds impressive.

## Replication: copies for reads and failover

**Replication** keeps one or more **copies** of your database in sync with the
original. The common shape is **primary/replica** (also called leader/follower): one
**primary** accepts all the **writes**, and every change streams to one or more
**replicas** that stay nearly current.

That buys two big things:

- **Read capacity.** Most workloads read far more than they write. Send writes to the
  primary and spread reads across the replicas, and you multiply how many reads you
  can serve without touching write capacity.
- **Failover / availability.** A replica is a warm standby. If the primary dies, you
  **promote** a replica to become the new primary, and the system keeps running.
  Replicas can also live in another region for disaster resilience.

The catch is **replication lag.** A replica is a moment behind the primary, so a read
from a replica may miss a write that just landed — you write a value, immediately read
it from a replica, and it's not there yet. This is **eventual consistency**: replicas
converge to the same state, but not instantly. You design around it by routing
reads-that-must-see-your-own-writes to the primary, and sending everything else to
replicas.

## Sharding: splitting the data itself

Replication scales *reads* — but every write still funnels through the single
primary, so it doesn't help a write-bound system. For that you need **sharding**
(horizontal **partitioning**): splitting the data across multiple servers so each
**shard** holds a slice and handles the writes for that slice.

You split on a **shard key** — say, `system_id`, so all of one radio system's calls
live on one shard. Now writes spread across shards and total write capacity grows with
the number of shards.

The cost is steep, which is why sharding is a last resort:

- **Cross-shard queries are hard.** A query spanning shards must hit several servers
  and combine results; a join across shards may be impractical.
- **The shard key is a heavy commitment.** Choose it badly and one shard gets all the
  traffic (a **hot shard**), or rebalancing later means moving huge amounts of data.
- **Transactions across shards** lose the easy [ACID](/learn/databases/transactions-and-acid/)
  guarantees you get on one machine.

Reach for sharding only when a single primary genuinely can't keep up with writes and
you've already scaled up and cached.

## The CAP tradeoff

Once data lives on multiple machines, a law of distributed systems bites — the **CAP
theorem**. It says that when a **network partition** happens (some nodes can't reach
others, which *will* happen), a distributed database must choose between two goods:

- **Consistency** — every read sees the latest write, or an error. The system refuses
  to serve data it can't confirm is current.
- **Availability** — every request gets an answer, even if some nodes are unreachable —
  but that answer might be **stale**.

You cannot have both **during a partition**: either you reject requests to stay
correct, or you answer them and risk staleness. Every distributed store picks a side
on purpose — a bank ledger leans consistent, a social feed leans available — and that
choice, not a benchmark, is often what makes a database right or wrong for a job. It's
a big part of the [SQL vs. NoSQL](/learn/databases/sql-vs-nosql/) conversation.

<div class="knowledge-check" data-quiz data-correct-msg="Right — replication multiplies read capacity and gives a failover target, but writes still funnel through the single primary." markdown="0">
  <p class="knowledge-check__q">Quick check: what does adding read replicas primarily buy you?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">More write capacity, since writes spread across replicas</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">More read capacity and a failover target — writes still go to the one primary</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Stronger consistency, since every replica is always exactly in sync</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Scale **up** (a bigger server) before you scale **out** (more servers) — a single
  tuned database with a cache goes a very long way.
- **Replication** copies data to **read replicas**, multiplying **read** capacity and
  providing a **failover** target — but all **writes** still go to one **primary**.
- **Replication lag** means replicas are slightly behind (**eventual consistency**);
  route reads that must see their own writes to the primary.
- **Sharding** splits the data across servers on a **shard key** to scale **writes**,
  at heavy cost — cross-shard queries, hot shards, and lost cross-shard transactions.
- The **CAP theorem**: during a network partition a distributed database must trade
  **consistency** against **availability** — a deliberate choice per system.

Next up: [performance tuning & monitoring](/learn/databases/performance-and-monitoring/).
