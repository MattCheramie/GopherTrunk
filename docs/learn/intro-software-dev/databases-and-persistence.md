---
slug: databases-and-persistence
title: Storing data — files & databases
description: Why programs write data down instead of keeping it only in memory, and how the choice runs from a plain file through embedded SQLite to full relational and NoSQL databases — with a guide to picking the simplest fit.
keywords: data persistence, files, database, SQLite, relational database, SQL, NoSQL, key-value store, document database, time-series, ACID, transactions
level: intermediate
status: full
prereq:
  - what-is-software
faq:
  - q: "What's the difference between a file and a database?"
    a: "A file is a plain stream of bytes on disk — your program decides what the bytes mean and reads or writes the whole thing. A database adds structure and machinery on top: it indexes your data so you can query it quickly, coordinates multiple writers safely, and enforces rules about what valid data looks like. A file is the raw storage; a database is storage plus a query engine and safety guarantees."
  - q: "When should I use SQLite versus a full database server?"
    a: "Use SQLite when the data lives with one application on one machine — a desktop app, a phone app, an embedded device. It's a database inside a single file with no server to install or run. Reach for a client-server database like PostgreSQL or MySQL when many clients need to share the same data at once, the data outgrows one machine, or you need separate access control and backups managed centrally. Most single-user tools never need more than SQLite."
  - q: "What is SQL versus NoSQL?"
    a: "SQL databases store data in tables of rows and columns with a fixed schema, and you query them with the SQL language; they excel when data is structured and relationships matter. \"NoSQL\" is an umbrella for everything else — key-value stores, document stores, time-series and search engines — that trade some of the relational guarantees for flexibility, scale, or a shape that fits a specific job better. Neither is strictly better; they suit different problems."
  - q: "Do I need a database at all?"
    a: "Often no. If your data is small, structured simply, and only one part of your program writes it, a plain file or a bit of SQLite is plenty — and far less to manage than a database server. Add a real database when you actually hit the limits: lots of data, complex queries, many simultaneous writers, or a need for transactions. Start with the simplest store that works and grow only when a concrete need appears."
gophertrunk_links:
  - title: Bookmarks
    url: /bookmarks.html
    note: GopherTrunk saves your systems and bookmarks to a small local store so they survive a restart.
---

# Storing data — files & databases

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Memory is temporary — when a program exits, everything in RAM is gone. **Persistence** means writing data down so it survives a restart. The simplest store is a **file**; the most powerful is a database. A **relational (SQL)** database organises data into tables and lets you query and update it safely; **NoSQL** stores trade some of those guarantees for flexibility or scale. Match the tool to the data — and [start simple](/learn/intro-software-dev/what-is-software/).
</div>

A running program keeps its working data in memory, which is fast but forgetful: close the program, lose the power, or crash, and it's all gone. Any software that needs to *remember* something between runs — your settings, your saved work, a log of what happened — has to write that data somewhere durable. This lesson walks from the humblest option, a plain file, up through databases, and ends on how to choose the simplest store that does the job.

## Why persistence

Computer memory (RAM) is **volatile**: it holds data only while the program runs and the power is on. **Persistence** is the property of data surviving beyond that — it's still there after a restart, a crash, or a reboot, because it was written to durable storage like a disk or SSD.

Almost every useful program persists *something*:

- **Configuration** — the settings a user picked, so they don't re-enter them every launch.
- **User data** — documents, saved records, the actual content people create.
- **Logs** — a record of what happened, for debugging and auditing.
- **Recordings and captures** — larger blobs of output the program produced.

The question is never *whether* to persist, but *where* and *how* — and that's a spectrum from a single file to a full database server.

## Files: the simplest store

A **file** is the most basic durable store: a named stream of bytes on disk. Your program writes bytes and reads them back; what those bytes mean is entirely up to you.

Files come in a few flavours:

- **Plain text** — human-readable notes, logs, simple lists.
- **Structured text formats** — **JSON**, **CSV**, and **YAML** give text a shape that both people and programs can parse. A config file in YAML or a data export in CSV is readable *and* machine-friendly.
- **Binary** — compact, non-human-readable formats for images, audio, or a program's own efficient encoding.

Files are perfect for configuration and modest amounts of data. They're simple, portable, and need no extra software. But they have limits that show up as data grows:

- **Concurrent writes** — if two parts of a program (or two programs) write the same file at once, they can corrupt it. Files have no built-in coordination.
- **Querying** — to find one record you often have to read and scan the whole file; there's no index.
- **Growth** — rewriting a large file to change one value gets slow and wasteful.

When you hit those walls, it's time for a database.

## Embedded databases (SQLite)

An **embedded database** gives you a database's power without running a separate server. The classic example is **SQLite**: an entire relational database that lives in a **single file**, driven by a small library linked directly into your program. There's nothing to install, start, or administer — you open the file and query it.

This is the sweet spot for desktop and embedded applications, and it's why SQLite is quietly one of the most deployed pieces of software on Earth. Your phone uses it. Your web browser uses it. Countless apps store their local data in a SQLite file because it offers real querying, structure, and safe concurrent access from within one program — all with the operational simplicity of "it's just a file."

If a plain file is starting to feel cramped but you don't have a server-shaped problem, SQLite is very often the right next step.

## Relational databases & SQL

The **relational** model is the dominant way to structure data. It organises information into **tables**, each with named **columns** and any number of **rows**:

- A **table** holds one kind of thing (say, `systems`).
- Each **column** is a field with a type (a name, a frequency, a timestamp).
- Each **row** is one record.
- **Relationships** link tables — a row in one table can reference rows in another, so data isn't duplicated.
- A **schema** defines this structure up front and enforces it, rejecting data that doesn't fit.

You talk to a relational database with **SQL** (Structured Query Language), a declarative language for asking questions and making changes: *select* the rows that match some condition, *insert* new ones, *update* or *delete* existing ones. You describe *what* you want and the database figures out *how* to fetch it efficiently, using indexes it maintains behind the scenes.

Relational databases also provide **transactions** and the **ACID** guarantees — Atomicity, Consistency, Isolation, Durability. In plain terms: a group of changes either all happen or none do (no half-finished updates), the data stays valid, concurrent users don't trip over each other, and once the database says a change is saved it really is on disk. That's what makes them trustworthy for money, records, and anything you can't afford to corrupt.

**PostgreSQL** and **MySQL** are the best-known **client-server** relational databases: a database process runs somewhere and many clients connect to it over the network. That architecture is what you want when lots of users or services share the same data, when the data outgrows a single machine, or when backups and access control need central management.

## NoSQL & specialized stores

The relational model isn't the best fit for every problem, and **NoSQL** databases fill the gaps. It's an umbrella term — "not only SQL" — covering several families:

- **Key-value stores** (like **Redis**) map a key straight to a value, extremely fast; great for caches and session data.
- **Document stores** (like **MongoDB**) hold flexible, self-describing documents (often JSON-shaped) without a fixed schema; handy when records vary in structure.
- **Time-series** databases are tuned for data stamped with a time — metrics, sensor readings, prices — and the queries that slice it by time.
- **Search engines** index text so you can find documents by content rather than by exact key.

The common trade is **flexibility versus guarantees**. NoSQL stores often relax the strict schema or the full ACID promises of a relational database in exchange for speed, horizontal scale, or a data shape that matches the job perfectly. That's a good deal when you genuinely need it — and an unnecessary complication when a plain relational table would have done. Choose a specialized store because your data has a specialized shape, not because it sounds modern.

## Choosing where data lives

There's no single right answer, but a few questions point you at the simplest fit:

- **How much data?** Kilobytes → a file. Gigabytes with fast lookups → a database.
- **How structured?** Free-form notes → a file. Rows, columns, and relationships → relational.
- **How many writers?** One part of one program → a file or SQLite. Many clients at once → a client-server database.
- **Do you need queries or transactions?** "Find all records where…" or all-or-nothing updates → a database earns its keep.
- **Does it need a server?** Only if the data is genuinely shared across machines or users.

The guiding principle is [YAGNI — "you aren't gonna need it"](/learn/intro-software-dev/dry-kiss-yagni/): start with the simplest store that works, and reach for something bigger only when a concrete limit forces you to. A file that becomes SQLite that one day becomes PostgreSQL is a healthy, needs-driven progression. Standing up a database cluster for a config file is over-engineering you'll have to maintain forever.

## A worked example

Look at how a program like GopherTrunk spreads its data across the right stores instead of forcing everything into one:

- **Configuration** lives in a **file** — settings a person reads and edits, small and stable. A structured text format is perfect; no database needed.
- **Saved systems and [bookmarks](/bookmarks.html)** go in a **small local store** — structured records you want to look up and update, tied to one installation. An embedded database like SQLite fits: queryable and safe, but still just a local file with no server to run.
- **Recordings** are written as **files on disk** — large binary blobs of captured audio or signal data. These are big and streamed sequentially; a filesystem handles them far better than any database, so each recording is its own file and the database (if any) just stores the path.

Each kind of data lands in the store that matches its shape and access pattern — the simplest thing that does the job, not one heavyweight database swallowing everything.

<div class="knowledge-check" data-quiz data-correct-msg="Right — SQLite is an embedded database: real querying and structure, in a single file, with no server to run." markdown="0">
  <p class="knowledge-check__q">Quick check: a single-user desktop app needs structured, queryable local storage with no server to manage. Best first choice?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A PostgreSQL client-server database cluster</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">SQLite (an embedded database)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Keep everything in memory and hope it isn't lost</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Persistence** means data survives a restart — memory forgets, disk remembers, so real programs write data down.
- **Files** are the simplest store: plain text, structured formats (JSON/CSV/YAML), or binary; great for config and modest data, but weak at concurrent writes, querying, and growth.
- **SQLite** is a full database in a single file with no server — the sweet spot for desktop and embedded apps.
- **Relational (SQL)** databases organise data into tables with a schema, queried with SQL and protected by ACID transactions; **PostgreSQL** and **MySQL** scale that to many shared clients.
- **NoSQL** stores — key-value, document, time-series, search — trade some relational guarantees for flexibility, scale, or a better-fitting shape.
- Choose by need: **start simple** (file or SQLite) and grow only when data size, queries, writers, or sharing actually demand it.

Next up: how programs talk to each other and the world beyond the disk — [APIs & talking to other software](/learn/intro-software-dev/apis-and-integration/).
