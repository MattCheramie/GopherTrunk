---
slug: key-value-and-document
title: Key-value & document stores
description: Two of the most common NoSQL shapes explained — the dictionary-like key-value store built for fast lookups by key, and the JSON-document store built for self-contained records with flexible schemas. Learn what each is good at and where each falls down.
keywords: key-value store, document store, document database, NoSQL, Redis, MongoDB, JSON documents, denormalization, schema flexibility, key lookup, embedding vs referencing
level: intermediate
status: full
prereq:
  - sql-vs-nosql
faq:
  - q: What's the difference between a key-value store and a document store?
    a: "A key-value store treats each value as an opaque blob you can only fetch or set by its exact key — it doesn't look inside the value. A document store keeps structured JSON-like documents it *can* look inside, so you can query and index on fields within a document. Document stores are essentially key-value stores that understand their values' contents."
  - q: When should I use a key-value store?
    a: "When your access pattern is 'give me the value for this exact key' and you don't need to query by anything else — caches, user sessions, feature flags, rate-limit counters. They're extremely fast and simple precisely because they do so little. If you need to search by the contents of the value, you've outgrown a pure key-value store."
  - q: Do document stores mean I don't have to design my data?
    a: "No. The database stops enforcing a schema, but your data still has one — it just lives in your application code now. You still decide what fields a document has and whether to embed related data or reference it. Flexibility shifts the modeling burden onto you rather than removing it."
---

# Key-value & document stores

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **key-value store** is a giant dictionary: set and get an opaque **value** by its
exact **key**, nothing more — blazing fast and dead simple, perfect for caches,
sessions, and counters. A **document store** keeps self-contained **JSON-like
documents** it can look *inside*, so you can query and index fields within them, with
a **flexible schema** per record. Both are NoSQL shapes from the
[SQL vs. NoSQL](/learn/databases/sql-vs-nosql/) lesson; both trade joins and enforced
structure for speed and flexibility, and both push the modeling work into your code.
</div>

The last lesson said "NoSQL" is really a family of specific stores. Here are the two
you'll meet most often. They sit at different points on the same spectrum: the
key-value store does the least possible and is fast because of it, and the document
store adds just enough structure to query inside its records. Understanding both, and
where each breaks down, is most of what you need to use them well.

## The key-value store: a dictionary at scale

A **key-value store** is exactly the data structure you already know from every
programming language — a dictionary, map, or hash — turned into a database. You **set**
a value under a key, and later **get** it back by that same key:

```
SET  session:abc123   {"user":42,"expires":1699999999}
GET  session:abc123
DEL  session:abc123
```

The defining property is that the value is **opaque**: the store doesn't know or care
what's inside it — a string, a number, a serialized blob — it just hands it back
whole. You can *only* find a value by its exact key. There's no "find all sessions for
user 42," because that would require looking inside the values, which a pure key-value
store doesn't do.

That limitation is also the source of its power. Doing so little means it can be
extraordinarily **fast** and **simple to scale** — spreading keys across many machines
is easy when each key is independent. This is why **Redis** and similar stores are the
default tool for:

- **Caching** — hold the result of an expensive query under a key (the whole next
  lesson).
- **Sessions** — a logged-in user's session data, fetched by session ID every request.
- **Counters and rate limits** — increment a number under a key, atomically.
- **Feature flags and simple config** — one lookup by name.

Many key-value stores keep data in **memory** for speed, which makes them fast but also
means you treat the data as transient unless the store is configured for durability.
The moment you find yourself wanting to query by anything *other* than the key, you've
outgrown it — and that's exactly the door a document store opens.

## The document store: values you can look inside

A **document store** keeps records as **documents** — typically JSON — and, crucially,
it *understands* those documents. It can query and index on fields *within* a document,
not just fetch by an outer key. Records live in **collections**, and each document is
meant to be **self-contained**: as much of what you need for one thing as makes sense,
in one place.

```json
{
  "_id": "call_0192",
  "system": { "name": "Metro P25", "wacn": 782336 },
  "talkgroup": "Fire Dispatch",
  "started_at": "2026-08-04T09:00:00Z",
  "frequencies": [851000000, 851012500],
  "recorded": true
}
```

Notice what's happening: the system's details are **embedded** right in the call
document rather than sitting in a separate table you'd join to. A document store leans
into this — it's built to hand you a whole self-contained record in one read, no join
required. You can also index and query on inner fields, like "all documents where
`talkgroup` is `Fire Dispatch`."

## Flexible schema — a gift and a bill

The headline feature is a **flexible schema**: two documents in the same collection can
have different fields. Add a new field to new documents without a migration; older ones
simply don't have it. For fast-moving or genuinely irregular data, that's a real
advantage.

But the schema doesn't vanish — it **moves into your application code**. The database no
longer guarantees every call document has a `started_at`, so *your code* must handle the
ones that don't. All the rules that
[constraints](/learn/databases/constraints-and-integrity/) enforced for you become your
responsibility. Flexibility isn't the absence of a schema; it's the schema being
unenforced. That's a fine trade when you want it and a quiet source of bugs when you
forget you took it.

## Embed or reference: the modeling decision

Document stores resurface a decision relational normalization made for you: when data
relates, do you **embed** it (copy it into the document) or **reference** it (store an
ID and look it up separately)?

- **Embed** when the data is owned by and read with its parent, and rarely changes on
  its own — a call's frequency list, an order's line items. One read gets everything.
- **Reference** when the data is shared, large, or changes independently — the system a
  thousand calls belong to. Embedding it into every call recreates exactly the
  [duplication](/learn/databases/normalization/) problem: update the system and you'd
  have to rewrite a thousand documents.

Document stores tolerate — even encourage — some **denormalization** for read speed,
but the update, insert, and delete anomalies from the normalization lesson are still
real. You've just taken on the job of managing them, because the store won't join the
data back for you or keep the copies in sync.

## Which one, and when

The two stores answer different questions:

- Use a **key-value store** when your access is purely **"value for this exact key"** and
  you need it fast and simple — caches, sessions, counters. Its whole virtue is doing
  one thing at speed.
- Use a **document store** when your data is naturally a set of **self-contained
  documents** you'll query by their fields, and a rigid table schema fights the shape of
  your data more than it helps.
- Use **neither** — stay relational — when your data is highly interconnected and you
  need joins, strong transactions across records, and enforced integrity. That's most
  business data, which is why relational is still the default.

Both of these are tools you'll often run *alongside* a relational database, not instead
of it — a document store for a flexible slice of data, a key-value cache in front of
your SQL. Speaking of which, that cache is the next lesson.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a key-value store only fetches by exact key and treats the value as opaque; a document store understands the value and can query fields inside it." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the key difference between a key-value store and a document store?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Key-value stores are relational; document stores are not</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A key-value store treats the value as opaque; a document store can query fields inside the value</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Document stores can't be queried at all, only fetched by ID</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **key-value store** is a dictionary at scale: **set/get an opaque value by its exact
  key**, nothing more — fast, simple, and easy to scale.
- It fits **caches, sessions, counters, and flags**; the moment you need to query by
  anything but the key, you've outgrown it.
- A **document store** keeps **self-contained JSON-like documents** it can look inside,
  so you query and index on **fields within** a document, and often **embed** related
  data instead of joining.
- A **flexible schema** doesn't remove the schema — it **moves it into your code**, along
  with the integrity rules a relational database used to enforce.
- **Embed** owned, co-read data; **reference** shared or independently-changing data — or
  you recreate the duplication anomalies from normalization.
- Both commonly run **alongside** a relational database, not instead of it.

Next up: [Time-series & analytical stores](/learn/databases/time-series-and-analytics/).
