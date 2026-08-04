---
layout: learn-hub
learn_module: databases
permalink: /learn/databases/
title: Databases & Data — where your application's data lives, from a first table to production
description: A free, structured module on databases and data — the relational model and SQL, designing and modeling data, NoSQL and vector stores, using a database from your own code, and running one in production, with GopherTrunk's own data as the worked example.
keywords: learn databases, SQL tutorial, relational model, database design, normalization, joins, indexes, transactions, ACID, NoSQL, vector database, ORM, migrations, database in Go, choosing a database, Postgres, SQLite, Supabase
---

Almost every program you build eventually needs to **remember something** — a
user, a setting, a log of the calls it decoded. A **database** is where that data
lives: structured so you can store it reliably and get it back fast, even with many
readers and writers at once. Learning databases is one of the highest-leverage
things a developer can do, because the same handful of ideas — tables, keys, SQL,
transactions — show up in almost every application ever built.

**Who this is for.** Anyone who can write a little code (or wants to) and keeps
bumping into the question of *where does the data go?* You don't need any database
background — this starts from why plain files aren't enough and builds up from
there. It pairs naturally with the
[Intro to Software Dev]({{ '/learn/intro-software-dev/' | relative_url }}),
[Programming in Go]({{ '/learn/programming-go/' | relative_url }}), and
[Building AI Into Software]({{ '/learn/building-ai/' | relative_url }}) modules.

**How the module works.** Six units take you from a first table to a database
running in production. The early units build the core ideas — the relational model,
**SQL**, and how to design data well. The middle unit reaches beyond relational to
the **NoSQL, time-series, and vector stores** that modern apps lean on, including
the vector search behind AI retrieval. The last two units are the working
developer's part: talking to a database from your own **code** (safely, in Go), and
the operational reality of backups, scaling, and monitoring. Examples lean on real
software, including how a scanner like [GopherTrunk]({{ '/' | relative_url }}) stores
the systems and calls it decodes. Mark lessons complete as you go — your progress is
saved in your browser. New here?
**[Start with lesson 1: What a database is](/learn/databases/what-is-a-database/)**
