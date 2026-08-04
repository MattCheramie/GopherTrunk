---
slug: databases-in-go
title: Talking to a database from Go
description: The Go way to use a database — the database/sql package and its driver model, opening a pooled handle, running parameterised queries, scanning rows, prepared statements, and transactions — the patterns a service like GopherTrunk uses.
keywords: Go database/sql, Go database driver, sql.DB, QueryContext, Scan rows, prepared statement, parameterised query Go, sql.Tx transaction, connection pool Go, pgx
level: advanced
status: full
prereq:
  - sql-injection
faq:
  - q: "Do I need an ORM to use a database in Go?"
    a: "No. Go's standard library ships `database/sql`, a clean interface for running SQL directly, and idiomatic Go leans toward using it (or a thin helper) rather than a heavy ORM. You get parameterised queries, a built-in connection pool, and full control over the SQL, with a small amount of row-scanning boilerplate."
  - q: "Is sql.DB a single connection?"
    a: "No — and this trips people up. An `sql.DB` is a **pool** of connections managed for you, safe for concurrent use by many goroutines. You open one per database at startup and share it for the life of the program; you do not open one per request."
  - q: "How do I avoid SQL injection in Go?"
    a: "Pass values as query arguments, never format them into the SQL string. `db.QueryContext(ctx, \"... WHERE id = $1\", id)` sends the value as a parameter, so it can never be parsed as SQL. Using `fmt.Sprintf` to build a query with user input is the bug to avoid."
---

# Talking to a database from Go

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Go talks to databases through the standard library's **`database/sql`** package: you
register a **driver**, open a single **`sql.DB`** — which is a **connection pool**, not
one connection — and run **parameterised queries** whose values are passed as
arguments, never formatted into the string. You **scan** result rows into variables,
use **prepared statements** for repeated queries, and wrap multi-step writes in a
**transaction**. This is the concrete, idiomatic version of everything in this unit,
and the shape a service like [GopherTrunk](/learn/databases/data-in-gophertrunk/) uses.
</div>

Everything in this unit — connecting, pooling, avoiding injection, choosing between
raw SQL and an ORM — comes together in real code. Go is a good language to see it in,
because its standard library takes a clear position: it gives you a thin, safe SQL
interface rather than an ORM, so the concepts you've just learned are right on the
surface. If you're new to the language, the [Programming in Go](/learn/programming-go/)
path covers the basics; here we focus on the database.

## The database/sql model

Go's `database/sql` package defines a **generic interface** to SQL databases, and the
actual database-specific code lives in a **driver** you import separately. You write
against `database/sql`; the driver underneath speaks Postgres, MySQL, SQLite, or
whatever you chose. The driver registers itself via a blank import:

```go
import (
    "database/sql"

    _ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver
)
```

The underscore means "import for its side effects only" — you never call the driver
directly, you just make it available to `database/sql` by name. Swap the driver and
your query code barely changes.

## Open a pooled handle

You open the database once, at startup, and keep the handle for the program's life:

```go
db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
if err != nil {
    log.Fatalf("open db: %v", err)
}
defer db.Close()

// Open() doesn't actually connect — verify with a ping.
if err := db.PingContext(ctx); err != nil {
    log.Fatalf("connect db: %v", err)
}

db.SetMaxOpenConns(20)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(time.Hour)
```

The crucial thing to understand: **`sql.DB` is a connection pool, not a connection.**
It is safe to share across every goroutine, it opens and reuses connections for you,
and the `SetMax*` calls are exactly the [pool knobs](/learn/databases/connection-pooling/)
from earlier. You do **not** open a new `sql.DB` per request — that would defeat the
pool entirely. Note too that `sql.Open` is lazy; it validates arguments but doesn't
connect, so a `Ping` confirms the credentials and reachability up front.

## Querying safely

Reading rows is a query, then a loop that **scans** each row into variables:

```go
rows, err := db.QueryContext(ctx,
    `SELECT id, started_at, talkgroup
     FROM calls
     WHERE system_id = $1 AND started_at > $2
     ORDER BY started_at DESC
     LIMIT 50`,
    systemID, since,
)
if err != nil {
    return nil, err
}
defer rows.Close()

var calls []Call
for rows.Next() {
    var c Call
    if err := rows.Scan(&c.ID, &c.StartedAt, &c.Talkgroup); err != nil {
        return nil, err
    }
    calls = append(calls, c)
}
return calls, rows.Err() // check Err() after the loop, too
```

The `$1` and `$2` are **placeholders**, and `systemID` and `since` are passed as
**arguments** — they travel to the database as data, so this is injection-proof by
construction. This is the whole lesson of [SQL injection](/learn/databases/sql-injection/)
in one idiom: never build the query with `fmt.Sprintf`, always pass values as
parameters. `QueryRowContext` is the variant for a query you expect to return a
single row.

## Prepared statements

If you run the same query many times — say, inserting every decoded call — you can
**prepare** it once so the database parses and plans it a single time, then execute it
repeatedly with different values:

```go
stmt, err := db.PrepareContext(ctx,
    `INSERT INTO calls (system_id, talkgroup, started_at) VALUES ($1, $2, $3)`)
if err != nil {
    return err
}
defer stmt.Close()

for _, c := range batch {
    if _, err := stmt.ExecContext(ctx, c.SystemID, c.Talkgroup, c.StartedAt); err != nil {
        return err
    }
}
```

Prepared statements are still fully parameterised — the same injection safety — and
they save the parse-and-plan cost on each execution. `ExecContext` is what you use
for statements that don't return rows (`INSERT`, `UPDATE`, `DELETE`).

## Transactions

When several writes must succeed or fail together, wrap them in a **transaction** so
the change is all-or-nothing, exactly the [ACID](/learn/databases/transactions-and-acid/)
guarantee:

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback() // no-op if we've already committed

if _, err := tx.ExecContext(ctx,
    `UPDATE systems SET last_seen = $1 WHERE id = $2`, now, systemID); err != nil {
    return err // deferred Rollback undoes any partial work
}
if _, err := tx.ExecContext(ctx,
    `INSERT INTO calls (system_id, talkgroup, started_at) VALUES ($1, $2, $3)`,
    systemID, tg, now); err != nil {
    return err
}
return tx.Commit()
```

The pattern is idiomatic Go: `defer tx.Rollback()` right after `BeginTx`, so any
early return unwinds the transaction, and a successful `Commit()` at the end makes the
rollback a harmless no-op. All statements on `tx` run on the **same connection**, which
is what makes them one atomic unit.

## A few Go-specific habits

- **Always pass a `context.Context`** (the `...Context` methods). It carries deadlines
  and cancellation, so a slow query dies with its request instead of hanging a pooled
  connection.
- **Always `defer rows.Close()`** and check `rows.Err()` after the loop — a leaked
  `rows` holds its connection out of the pool.
- **`sql.ErrNoRows`** is how `QueryRow(...).Scan` signals "no row found"; handle it as
  a normal case, not an error.
- **Prefer the standard library or a thin helper.** Idiomatic Go tends toward
  `database/sql` (or a light layer like `sqlx`) over a full ORM — the
  [raw-SQL end of the tradeoff](/learn/databases/orms-vs-raw-sql/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — an sql.DB is a connection pool safe for concurrent use, opened once and shared, not a single connection per request." markdown="0">
  <p class="knowledge-check__q">Quick check: what is a Go <code>sql.DB</code>?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A single database connection you open per request</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A connection pool, safe for concurrent use, opened once and shared for the program's life</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">An ORM that maps Go structs to tables automatically</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Go uses the standard library's **`database/sql`** with a separately imported
  **driver** — you write generic SQL code and swap the driver underneath.
- **`sql.Open` returns an `sql.DB`, which is a connection pool**, not a single
  connection — open it once, share it, and tune it with `SetMaxOpenConns` and friends.
- Run **parameterised queries** with placeholders (`$1`) and pass values as arguments;
  never build SQL with `fmt.Sprintf` — that's injection-proof by construction.
- **Scan** result rows into variables in a `rows.Next()` loop, always `defer
  rows.Close()`, and check `rows.Err()`.
- Use **prepared statements** for repeated queries and **transactions** (`BeginTx` /
  `Commit` with a deferred `Rollback`) for all-or-nothing writes.
- Pass a **`context.Context`** to every call, and lean on the standard library rather
  than reaching first for an ORM.

Next up: [backups & recovery](/learn/databases/backups-and-recovery/).
