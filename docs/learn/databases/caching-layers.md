---
slug: caching-layers
title: Caching layers
description: A cache keeps hot data in fast memory in front of the database so repeated reads skip the slow lookup. Learn what a cache buys you, the cache-aside pattern, TTLs, and why cache invalidation is one of the genuinely hard problems in software.
keywords: caching, cache layer, Redis, cache-aside, TTL, cache invalidation, stale data, cache hit, cache miss, memcached, read-through cache, hot data
level: intermediate
status: full
prereq:
  - key-value-and-document
faq:
  - q: What is a cache, in one sentence?
    a: "A small, fast store — usually in memory — that holds copies of data you read often, sitting in front of your slower database so repeated reads can be answered from the cache instead of hitting the database every time."
  - q: What is the cache-aside pattern?
    a: "The most common caching approach: your app checks the cache first; on a hit it returns the cached value, and on a miss it reads from the database, stores the result in the cache, and returns it. The cache fills itself lazily with whatever data actually gets requested, and each entry usually has a TTL so it eventually refreshes."
  - q: Why is cache invalidation considered so hard?
    a: "Because a cache holds a *copy* of data that can change underneath it. The moment the database updates, the cached copy is stale, and knowing exactly when and how to remove or refresh every affected entry — without missing one or clearing too much — is genuinely tricky. A TTL bounds staleness by time; explicit invalidation on write is more precise but easy to get subtly wrong."
---

# Caching layers

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **cache** is a small, fast store — usually in-memory, like **Redis** — that holds
copies of frequently-read data **in front of** your database, so repeated reads are
answered from memory instead of a slower lookup. The common approach is
**cache-aside**: check the cache, and on a **miss** fetch from the database and store
the result. The catch — and one of the genuinely hard problems in software — is
**invalidation**: a cached copy goes **stale** the moment the underlying data changes.
A cache is itself a [key-value store](/learn/databases/key-value-and-document/) put to
a specific job.
</div>

This unit's last stop isn't a new *kind* of data — it's a technique that sits in front
of whatever database you already have. Caching is how you make a system that reads the
same data over and over dramatically faster, and it's nearly universal in production
apps. It also comes with one famously hard problem, which is worth understanding
clearly before you rely on one.

## What a cache buys you

Some data gets read far more than it changes. A system's configuration, a popular
talkgroup's label, a user's profile — read on nearly every request, updated rarely.
Hitting the database for each of those reads is wasteful: the same query, the same
answer, again and again, each one costing a round-trip and some database work.

A **cache** is a small, very fast store — almost always in **memory**, which is orders
of magnitude faster than disk — that sits **between your app and the database** and
holds copies of that hot data. When the data's already there (a **cache hit**), you
answer from memory and never touch the database. Two big wins follow:

- **Latency** — a memory lookup is far faster than a database query, so cached reads
  return almost instantly.
- **Load** — every hit is a query your database *didn't* have to run, so the database
  survives far more traffic before it strains.

**Redis** and **Memcached** are the usual tools, and it's no accident they're key-value
stores from the last lesson: fetch-by-key is exactly the fast, simple operation a cache
needs.

## Cache-aside: the standard pattern

The most common way to use a cache is **cache-aside** (also called lazy loading). Your
application code sits between the cache and the database and follows a simple dance:

```go
func getSystem(id string) (System, error) {
    // 1. Try the cache first.
    if v, ok := cache.Get("system:" + id); ok {
        return v, nil                       // cache hit
    }
    // 2. Miss — read from the database.
    sys, err := db.LoadSystem(id)
    if err != nil {
        return System{}, err
    }
    // 3. Store it in the cache for next time, then return.
    cache.Set("system:"+id, sys, 5*time.Minute)
    return sys, nil
}
```

On a **hit**, you return immediately. On a **miss**, you read from the database, *put
the result in the cache*, and return it — so the next request for the same key is a hit.
The cache fills itself **lazily** with exactly the data that actually gets requested;
data nobody reads never takes up space. This is the pattern you'll reach for most.

## TTL: letting entries expire

Notice the `5*time.Minute` above — that's a **TTL** (time to live). Each cached entry is
stamped with an expiry, and once it passes, the cache drops the entry; the next read
misses and refetches fresh data from the database. TTLs do two jobs at once:

- They **bound staleness**: no cached value is ever more than its TTL out of date, even
  if you do nothing else.
- They **bound memory**: entries that stop being read eventually expire and free space,
  and caches also **evict** old entries when full (commonly least-recently-used).

Choosing a TTL is a trade. Longer means more hits and less database load but more time
serving slightly-old data; shorter means fresher data but more misses. Data that can
tolerate being a few minutes old — most read-heavy display data — is the sweet spot for
a generous TTL.

## The hard problem: invalidation and staleness

Here's the rub, captured in a famous programmers' joke: *there are only two hard things
in computer science — cache invalidation and naming things.* A cache holds a **copy**.
The instant the real data changes in the database, that copy is **stale** — it says
something the source of truth no longer says. If you cached a talkgroup's label and
someone renames it, every cache hit now returns the old name until something fixes the
entry.

You have two levers, and both are imperfect:

- **Expire by TTL.** Simple and robust — the stale window is at most the TTL. But you
  *will* serve stale data for that window, so it only suits data that tolerates being a
  little old.
- **Invalidate on write.** When you update the database, also delete (or update) the
  affected cache entry, so the next read refetches fresh. More precise, but harder than
  it sounds: you must find *every* key that depends on the changed data, do it reliably
  even if the write and the invalidation can't be one atomic step, and avoid both
  missing an entry (stale forever) and clearing too much (needless misses).

The reason invalidation is genuinely hard is that it re-creates, in a new place, the
same **duplication** problem [normalization](/learn/databases/normalization/) warned
about: a fact now lives in two places — the database and the cache — and keeping them in
agreement is exactly the difficulty. A cache is deliberate, managed duplication for
speed, and the price is that you own the job of keeping the copy honest.

## Use a cache on purpose

A cache is a powerful tool, not a default to sprinkle everywhere. Add one when you have
a **measured** read-heavy hotspot — the same expensive reads dominating your load — and
data that can tolerate being briefly stale. Match the staleness lever to the data:
generous TTLs for display data, tight invalidation for things that must look fresh, and
no cache at all for data that changes as often as it's read. Done well, a caching layer
is one of the highest-leverage performance wins available; done carelessly, it's a
machine for serving wrong answers quickly. Get it right and you have the front of a
production data stack — which is exactly where the next unit picks up, wiring a database
into real running code.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the cache holds a copy, so when the source data changes the copy is stale, and knowing exactly what to refresh and when is the hard part." markdown="0">
  <p class="knowledge-check__q">Quick check: why is cache invalidation considered a genuinely hard problem?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Caches are too slow to update once data is written</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The cache holds a copy that goes stale when the source changes, and knowing exactly what to refresh, and when, is tricky</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Caches can only ever store data permanently and never remove it</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **cache** is a small, fast, usually in-memory store (like **Redis**) **in front of**
  your database, holding copies of frequently-read data.
- It cuts **latency** (memory beats disk) and **load** (every **hit** is a query the
  database didn't run).
- **Cache-aside** is the standard pattern: check the cache, and on a **miss** read the
  database, store the result, and return it — filling the cache lazily.
- A **TTL** expires each entry after a set time, bounding both **staleness** and memory
  use; longer TTLs mean more hits but staler data.
- **Invalidation is the hard part**: a cached copy goes **stale** when the source
  changes. Bound it with a TTL, or invalidate on write for precision — both are easy to
  get subtly wrong because a cache is deliberate duplication you must keep honest.
- Add a cache **on purpose**, for a measured read-heavy hotspot with staleness-tolerant
  data — not by default everywhere.

Next up: [Connecting from your program](/learn/databases/connecting-from-code/).
