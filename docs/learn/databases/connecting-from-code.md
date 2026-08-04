---
slug: connecting-from-code
title: Connecting from your program
description: How a running program actually opens a link to a database — the driver that speaks the database's protocol, the connection string that says where and who, and where to keep the credentials so they never end up in your code.
keywords: database driver, connection string, DSN, credentials, host port user password, TCP connection, TLS, environment variable, connect from code, database client
level: intermediate
status: full
prereq:
  - what-is-sql
faq:
  - q: "What is a database driver?"
    a: "A **driver** is the library that lets your program speak a specific database's wire protocol. Postgres, MySQL, and SQLite each have their own on-the-wire format; the driver turns your query into bytes the server understands and turns the response back into values your code can read. You pick the driver that matches your database."
  - q: "What goes in a connection string?"
    a: "Where the database is and who is connecting: the host and port, the database name, a username and password, and options like TLS mode. It is often written as one URL-like string (a **DSN**, data source name). The credentials in it are secrets — load them from the environment, never hard-code them."
  - q: "Do I open a new connection for every query?"
    a: "No. Opening a connection is expensive — a TCP handshake, TLS, and authentication every time. Real apps open a small set once and reuse them, which is what a **connection pool** does. The next lesson covers pooling in full."
---

# Connecting from your program

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
To talk to a database, your program needs three things: a **driver** that speaks
the database's protocol, a **connection string** that says where the database is
and who is connecting, and **credentials** kept safely out of your source. The
driver turns your SQL into bytes on a socket and the reply back into values. Get
these right and every query in this module runs from real code. Next you will
learn why you should never open one connection per query — see
[connection pools](/learn/databases/connection-pooling/).
</div>

So far every query in this module has been something you might type into a
database shell. Real software doesn't type queries — it sends them from a running
program. This lesson is the bridge: how a process on your machine actually opens a
link to a database, authenticates, and starts sending SQL. There is less magic
here than you might expect, and knowing the pieces makes every "cannot connect"
error readable.

## The driver speaks the protocol

A database server doesn't accept plain text over a socket the way a web server
accepts HTTP. Each database has its own **wire protocol** — a binary format for
requests and responses. Postgres speaks one, MySQL another, SQLite doesn't use a
socket at all because it lives inside your process as a file.

The library that knows a given protocol is the **driver**. Your code calls a
normal function like "run this query"; the driver serialises that into the exact
bytes the server expects, reads the response, and hands you back rows and values.
You choose a driver to match your database — a Postgres driver won't talk to
MySQL. In most languages the standard library gives you a common *interface* and
you plug in the driver underneath, so your code looks the same across databases
even though the wire format differs.

## The connection string says where and who

To open a link, the driver needs to know two things: **where** the database is and
**who** is connecting. That information is usually packed into one
**connection string**, also called a **DSN** (data source name). It commonly looks
like a URL:

```
postgres://scanner:s3cret@db.internal:5432/gophertrunk?sslmode=require
```

Read left to right, that says: use the Postgres driver, log in as user `scanner`
with password `s3cret`, connect to host `db.internal` on port `5432`, use the
database named `gophertrunk`, and require an encrypted (**TLS**) connection. Some
drivers take the same information as separate key-value pairs instead:

```
host=db.internal port=5432 user=scanner password=s3cret dbname=gophertrunk sslmode=require
```

Either way the ingredients are the same — **host, port, database, user, password,
and options**. The options matter more than they look: `sslmode=require`, for
instance, is the difference between your password crossing the network encrypted
or in the clear.

## Credentials are secrets

A connection string contains a password, so it is a secret — treat it exactly like
an API key. **Never hard-code it in source, and never commit it to git.** A leaked
database credential is worse than a leaked API key: it can mean someone reading,
altering, or deleting all your data.

Load the connection string, or at least the password, from an **environment
variable** or a secret manager at runtime:

```bash
export DATABASE_URL="postgres://scanner:s3cret@db.internal:5432/gophertrunk?sslmode=require"
```

Then your program reads `DATABASE_URL` from its environment and passes it to the
driver. The value lives in your deployment's configuration, not your repository.
This is the same discipline the AI path teaches for API keys in
[your first API call](/learn/building-ai/your-first-api-call/), and the security
path treats as a rule in [secure coding](/learn/cybersecurity/secure-coding/).

## What "open" actually does

When your code opens a connection, a fair amount happens under the hood:

- **A network connection** is established to the host and port — a TCP handshake,
  and for a remote database a TLS handshake on top to encrypt the link.
- **Authentication** — the driver sends the username and password (or a token),
  and the server checks them and grants a session.
- **A session** is now open: a stateful channel over which you can send queries and
  read results until you close it or it drops.

All of that costs real time — often milliseconds, sometimes more across a slow
network. That cost is exactly why you don't want to pay it on every query, which
leads straight into pooling.

## When connecting fails

The failures here are common and, once you know the pieces, self-explanatory:

- **Connection refused / timeout** — nothing is listening at that host and port, or
  a firewall is blocking you. Check the host, the port, and network reach.
- **Authentication failed** — wrong username or password, or the user isn't allowed
  from your address. Check the credentials and the database's access rules.
- **Database does not exist** — the server is reachable and you authenticated, but
  the named database isn't there. Check the `dbname` part.
- **TLS / SSL errors** — the server requires encryption you didn't offer, or a
  certificate doesn't verify. Check the `sslmode` and certificate settings.

Each maps to one ingredient of the connection string, which is why reading the
string carefully is usually the fastest way to fix a connection problem.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the driver speaks the database's wire protocol, turning your query into bytes and the reply back into values." markdown="0">
  <p class="knowledge-check__q">Quick check: what is a database driver's job?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">To store your tables and indexes on disk</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">To speak the database's wire protocol — turning queries into bytes and replies into values</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">To write your SQL for you so you don't have to</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- To talk to a database from code you need a **driver**, a **connection string**,
  and **credentials** — and the driver must match your database.
- The driver speaks the database's **wire protocol**, turning your query into bytes
  on a socket and the response back into values.
- A **connection string** (DSN) carries **host, port, database, user, password, and
  options** like TLS mode — often as one URL-like string.
- The credentials are secrets: load them from an **environment variable** or secret
  manager, never hard-code or commit them.
- Opening a connection is real work — TCP, TLS, and authentication — which is why
  you reuse connections rather than open one per query.
- Most connection failures map cleanly to one part of the connection string: host,
  auth, database name, or TLS.

Next up: [connection pools](/learn/databases/connection-pooling/).
