---
slug: backend-and-database
title: The back end & its database
description: Where the server meets the data layer — how a back end connects to a database, reads and writes it safely with parameterised queries, reuses connections with a pool, and keeps the boundary between web logic and data logic in the right place.
keywords: back end database, database connection, parameterised query, SQL injection, connection pool, ORM, data access layer, prepared statement, query, persistence, data layer
level: intermediate
status: full
prereq:
  - building-a-rest-api
faq:
  - q: How does the back end talk to the database?
    a: "It opens a connection to the database server, sends queries (usually SQL), and reads back rows. In practice it keeps a pool of open connections to reuse, and often uses a driver, query builder, or ORM library so it works with the database in the language's own types rather than raw wire protocol."
  - q: What is SQL injection and how do I prevent it?
    a: "SQL injection is when user input is concatenated into a query so that the input changes what the query does — a classic way attackers read or destroy data. You prevent it by using parameterised queries (placeholders bound to values), which keep the input as data, never executable SQL. Never build a query by string-concatenating user input."
  - q: Should the front end ever talk to the database directly?
    a: "Almost never. The database holds shared, sensitive data and its credentials are secrets, so it lives on the trusted server side. The browser talks to your API, and only the back end talks to the database — that boundary is what lets the server enforce rules and keep credentials hidden."
---

# The back end & its database

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The **database** is where a [back end](/learn/web-dev/what-a-backend-does/) keeps its
**shared, durable data**, and the server is the **only** thing that talks to it — never
the browser. The back end **connects** to the database, sends **queries**, and maps rows
into objects, reusing connections through a **[pool](/learn/databases/connection-pooling/)**
rather than opening one per request. The rule that matters most for safety is
**parameterised queries** — never build SQL by concatenating user input, or you invite
[SQL injection](/learn/databases/sql-injection/). Keep **web logic and data logic**
separate so the boundary stays clean. The data layer itself is the
[Databases module](/learn/databases/).
</div>

A [REST API](/learn/web-dev/building-a-rest-api/) is only useful if there's real data
behind it. That data lives in a **database**, and this lesson is where the server meets it:
how a handler goes from "the client asked for the calls" to actually reading them, and how
to do that **safely** and with the boundaries in the right place. It's the seam between
this module and the [Databases module](/learn/databases/), which covers the data layer in
depth; here we focus on the back end's side of the relationship.

## Only the server touches the database

First, a boundary. The database holds **shared, sensitive data**, and its credentials are
**secrets** — so it lives entirely on the trusted server side, and the **browser never
connects to it directly.** The flow is always:

```
Browser ──HTTP──▶ Back-end handler ──query──▶ Database
        ◀─JSON──                    ◀─rows──
```

The front end calls your API; the handler queries the database; the handler shapes the
result into JSON and returns it. This is the same trust argument from
[what a back end does](/learn/web-dev/what-a-backend-does/): if the browser could reach the
database, every rule you enforce and every secret you hold would be bypassable. The server
in the middle is what keeps the data layer both safe and abstract — clients see an API, not
tables.

## Connecting and querying

To use a database, the back end opens a **connection** to the database server, sends a
**query**, and reads back **rows**. Most relational databases speak **SQL**; the handler
sends a statement and maps the results into the language's own types.

```go
func listCalls(w http.ResponseWriter, r *http.Request) {
    rows, err := db.Query("SELECT id, talkgroup, seconds FROM calls ORDER BY id DESC")
    if err != nil { http.Error(w, "db error", 500); return }
    defer rows.Close()

    var calls []Call
    for rows.Next() {
        var c Call
        rows.Scan(&c.ID, &c.Talkgroup, &c.Seconds)   // row → struct
        calls = append(calls, c)
    }
    json.NewEncoder(w).Encode(calls)                 // structs → JSON
}
```

That's the whole loop of the data layer: query, scan rows into objects, return them. Writes
work the same way with `INSERT`, `UPDATE`, and `DELETE` statements — often mapping directly
onto the [REST verbs](/learn/web-dev/building-a-rest-api/) from the last lesson.

## Never concatenate user input: parameterise

Here is the single most important safety rule on the back end. **Never build a query by
gluing user input into a string.** Do this and an attacker can inject SQL that changes what
the query does — reading other users' data, or dropping your tables. That's
**[SQL injection](/learn/databases/sql-injection/)**, one of the oldest and most damaging
web vulnerabilities.

```go
// NEVER — user input becomes part of the SQL itself.
db.Query("SELECT * FROM calls WHERE talkgroup = '" + input + "'")

// ALWAYS — a placeholder; the driver binds the value as data, never as code.
db.Query("SELECT * FROM calls WHERE talkgroup = ?", input)
```

A **parameterised query** (also called a prepared statement) sends the SQL and the values
**separately**, so the input is always treated as data and can never become executable SQL —
no matter what the user types. This isn't an optimisation you add later; it's the default
you never deviate from. The [SQL injection lesson](/learn/databases/sql-injection/) shows
exactly how the attack works and why parameterisation stops it cold.

## Connection pooling

Opening a fresh database connection for every request is slow — each involves a network
handshake and authentication. Instead the back end keeps a **[connection
pool](/learn/databases/connection-pooling/)**: a set of open connections it **reuses**
across requests, handing one to a handler for the duration of a query and returning it
afterward. The pool caps how many connections exist at once, which also protects the
database from being overwhelmed under load. Most database libraries manage the pool for
you — you configure a size and borrow/return happens automatically — but knowing it's there
explains a lot of production behaviour, from latency to "too many connections" errors. The
[connection pooling lesson](/learn/databases/connection-pooling/) goes into how to size and
tune it.

## Keep the boundary clean

As an app grows, it pays to keep **web logic** and **data logic** in separate places rather
than scattering SQL through your handlers. A common shape is a **data-access layer** (a
repository or store): handlers deal with HTTP — parse the request, check auth, format JSON —
and call into functions like `store.ListCalls()` that own the queries.

```go
// Handler stays about HTTP; the store owns the SQL.
func listCalls(w http.ResponseWriter, r *http.Request) {
    calls, err := store.ListCalls(r.Context())   // data logic lives here
    if err != nil { http.Error(w, "db error", 500); return }
    json.NewEncoder(w).Encode(calls)
}
```

This separation keeps handlers readable, makes the data layer testable on its own, and
means a change to how data is stored doesn't ripple through your web code. Some teams use an
**ORM** (object-relational mapper) to generate much of this layer from data models — a
convenience with real trade-offs, covered in the [Databases module](/learn/databases/).
Whichever you choose, the principle holds: the database is the server's private
responsibility, accessed safely, behind a clean boundary — and everything the front end
knows about it comes through your [API](/learn/web-dev/building-a-rest-api/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — parameterised queries send SQL and values separately, so user input can never become executable SQL. That's the defense against SQL injection." markdown="0">
  <p class="knowledge-check__q">Quick check: how do you safely include user input in a database query?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Concatenate it into the SQL string after escaping a few quotes</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Use a parameterised query with a placeholder bound to the value</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Let the browser send the finished SQL so the server doesn't build it</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The **database** holds shared, durable data; **only the server** connects to it — the
  browser talks to your [API](/learn/web-dev/building-a-rest-api/), never the database.
- The back end **connects, queries, and maps rows** into objects; writes use
  `INSERT`/`UPDATE`/`DELETE`, often mirroring the REST verbs.
- **Never concatenate user input into SQL** — use **parameterised queries** to prevent
  [SQL injection](/learn/databases/sql-injection/); this is the non-negotiable default.
- Reuse connections with a **[connection pool](/learn/databases/connection-pooling/)**
  rather than opening one per request.
- Keep **web logic and data logic separate** — a data-access layer (or ORM) — so the
  boundary stays clean and testable.
- The data layer itself is the subject of the [Databases module](/learn/databases/).

Next up: [SSR vs. SPA vs. static](/learn/web-dev/ssr-spa-static/).
