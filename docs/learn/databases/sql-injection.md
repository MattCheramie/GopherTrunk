---
slug: sql-injection
title: SQL injection & querying safely
description: The classic and still-common vulnerability where user input is glued into a query and becomes SQL — how it happens, the damage it does, and why parameterised queries (not manual escaping) end it for good.
keywords: SQL injection, parameterised query, prepared statement, query parameters, string concatenation, input validation, escaping, least privilege, OWASP, querying safely
level: intermediate
status: full
prereq:
  - querying-with-select
faq:
  - q: "What is SQL injection in one sentence?"
    a: "It's when untrusted input gets treated as part of your SQL command instead of as mere data, letting an attacker change what the query does — read, alter, or destroy data they shouldn't reach. It happens whenever you build a query by gluing user input into the query string."
  - q: "How do parameterised queries fix it?"
    a: "They send the SQL and the values to the database *separately*. The query text with placeholders is fixed and parsed first; your values are then bound to the placeholders as pure data that can never be reinterpreted as SQL. Because the structure is locked before the values arrive, injected SQL is just a string, not a command."
  - q: "Isn't escaping the input enough?"
    a: "No — manual escaping is fragile and easy to get wrong across encodings, quoting styles, and edge cases, and one missed spot is a hole. Parameterised queries remove the class of bug entirely rather than trying to sanitise your way around it. Use them everywhere and don't hand-escape."
---

# SQL injection & querying safely

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**SQL injection** happens when untrusted input is glued into a query string and
gets executed as **SQL** rather than treated as data — the most famous database
vulnerability, and still common. The fix is not to escape input by hand but to use
**parameterised queries** (prepared statements), which send the query and the values
to the database separately so input can never become command. Pair that with
**least privilege** and validation. This is the database face of
[secure coding](/learn/cybersecurity/secure-coding/).
</div>

Every lesson so far assumed the SQL you send is the SQL you meant to send. This one
is about what happens when an attacker gets to write part of it for you. SQL
injection has been near the top of every vulnerability list for over two decades,
not because it's clever but because it's easy to introduce and devastating when it
lands. The good news is that it has a complete, boring fix — and once you use it
everywhere, this entire class of bug goes away.

## How injection happens

Injection comes from one mistake: **building a query by gluing user input into the
query text.** Imagine a search box, and code that assembles the query like this:

```
query = "SELECT * FROM users WHERE name = '" + userInput + "'"
```

If someone types `Alice`, you get the query you expected. But the input is being
treated as part of the *command*, not just data, and an attacker knows it. Type:

```
' OR '1'='1
```

and the query becomes:

```sql
SELECT * FROM users WHERE name = '' OR '1'='1'
```

`'1'='1'` is always true, so this returns **every user**. Worse inputs terminate the
statement and add their own — `'; DROP TABLE users; --` — turning a search into a
delete. The database faithfully runs whatever SQL it receives; it has no way to know
part of that SQL came from a stranger. That confusion between **code and data** is
the whole vulnerability.

## What an attacker can do

Once input can become SQL, the blast radius is large. Depending on the query and the
database account's permissions, an attacker may:

- **Read data they shouldn't** — dump entire tables, including password hashes and
  personal data.
- **Bypass authentication** — the `OR '1'='1'` trick can make a login check pass
  without a valid password.
- **Alter or destroy data** — `UPDATE` values, or `DELETE`/`DROP` whole tables.
- **Escalate** — on some setups, reach the operating system or other databases on
  the server.

This is why injection is treated as critical: it can compromise **confidentiality,
integrity, and availability** all at once — the whole [CIA triad](/learn/cybersecurity/cia-triad/).

## Parameterised queries end it

The fix is to stop building queries by concatenation and use **parameterised
queries**, also called **prepared statements**. You write the SQL with **placeholders**
where values go, and pass the values separately:

```sql
SELECT * FROM users WHERE name = ?
```

```
db.query("SELECT * FROM users WHERE name = ?", userInput)
```

The database receives the query text and the value on **separate channels**. It
parses the SQL *first* — with the placeholder, so the statement's structure is fixed
— and only then binds your value into the placeholder as pure **data**. Now if
someone submits `' OR '1'='1`, that entire string is treated as one literal name to
search for; there is no name equal to that string, so it matches nothing. The
injected SQL is never parsed as SQL, because parsing already happened before the
value arrived. The **structure of the query is locked before the value is seen** —
that's the whole mechanism, and it's airtight.

Different databases spell the placeholder differently — `?`, `$1`, `:name` — but the
principle is identical everywhere.

## Why not just escape?

A tempting alternative is to **sanitise** the input yourself — strip or escape quotes
and dangerous characters. Resist it. Manual escaping is a losing game: quoting rules
differ across databases, character encodings introduce edge cases, and a single
place you forget is a single hole an attacker needs. You are trying to out-clever a
problem that parameterisation removes entirely. **Use parameterised queries
everywhere and don't hand-escape.** The only inputs you can't parameterise —
occasionally a table or column *name* — must be checked against a fixed allow-list,
never concatenated raw.

## Defence in depth

Parameterised queries are the cure, but a couple of habits back them up so a single
slip isn't catastrophic:

- **Least privilege.** The database account your app uses should have only the
  permissions it needs. An app that only reads and inserts calls shouldn't be able
  to `DROP TABLE`. If an injection ever does land, least privilege bounds the damage.
- **Validate input anyway.** Checking that a value is the expected type, length, and
  format is good practice — it catches bad data and shrinks the attack surface. It is
  a complement to parameterisation, never a replacement for it.
- **Least data exposed.** Don't return more than the caller needs; verbose errors and
  over-broad queries hand attackers information.

These are the same layered ideas the security path calls
[defense in depth](/learn/cybersecurity/defense-in-depth/), applied to the database.
Go's standard library makes the safe path the default, as the next lesson shows.

<div class="knowledge-check" data-quiz data-correct-msg="Right — parameterised queries send SQL and values separately, so input is always bound as data and can never be parsed as command." markdown="0">
  <p class="knowledge-check__q">Quick check: what actually prevents SQL injection?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Carefully escaping quotes in user input before concatenating it</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Parameterised queries — sending the SQL and the values separately so input is always data</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Hiding error messages so attackers can't see the query</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **SQL injection** happens when untrusted input is glued into a query string and
  executed as **SQL** instead of treated as data — a confusion of **code and data**.
- The damage is severe: reading private data, bypassing login, altering or destroying
  data, sometimes reaching the host.
- **Parameterised queries** (prepared statements) fix it by sending the SQL and the
  values separately — the query structure is **locked before the value is seen**, so
  input can never become command.
- **Don't hand-escape** input; manual sanitisation is fragile and one miss is a hole.
  Parameterise everywhere; allow-list the rare non-value inputs.
- Back it with **least privilege** on the database account and **input validation** —
  defence in depth so one slip isn't fatal.

Next up: [talking to a database from Go](/learn/databases/databases-in-go/).
