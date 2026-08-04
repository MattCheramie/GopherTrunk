---
slug: inserting-updating-deleting
title: Inserting, updating & deleting
description: The write side of SQL — INSERT adds rows, UPDATE changes them, DELETE removes them — and the WHERE clause you forget at your peril. Learn the syntax, the danger of a missing WHERE, and why real writes lean on transactions.
keywords: INSERT, UPDATE, DELETE, SQL writes, DML, WHERE clause, RETURNING, transaction, rollback, data manipulation, modifying data
level: beginner
status: full
prereq:
  - querying-with-select
  - filtering-and-sorting
faq:
  - q: "What happens if I run UPDATE or DELETE without a WHERE clause?"
    a: "It applies to every row in the table. UPDATE without WHERE changes all rows; DELETE without WHERE empties the whole table. There's no confirmation prompt — the database does exactly what you said. This is the single most common way people wreck data, so always write and check the WHERE first."
  - q: "Can I undo a DELETE?"
    a: "Only if it ran inside a transaction you haven't committed — then you can ROLLBACK. Once committed, the rows are gone and your only recovery is a backup. This is exactly why writes that matter run inside transactions and why tested backups exist."
  - q: "How do I get the id of a row I just inserted?"
    a: "Many databases support a RETURNING clause (INSERT ... RETURNING id) that hands back generated values from the new row. Where that's not available, the driver usually exposes a 'last insert id'. Either way you don't have to run a second query to find it."
---

# Inserting, updating & deleting

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The write side of SQL is three statements: **INSERT** adds rows, **UPDATE** changes
existing ones, **DELETE** removes them. UPDATE and DELETE take a **WHERE** clause that
decides *which* rows — and **forgetting it hits every row in the table**. Because a
write can go wrong, real applications wrap important ones in a **transaction** so they
can **ROLLBACK**. This is the counterpart to the reads you've learned since
[querying with SELECT](/learn/databases/querying-with-select/).
</div>

Everything so far has *read* data. Now you'll change it. The write statements —
**INSERT**, **UPDATE**, **DELETE** — are collectively called **DML** (data manipulation
language), and they're refreshingly simple in form. The catch is that they're
*powerful*: a single careless statement can rewrite or erase an entire table with no
undo. This lesson teaches the syntax and, just as importantly, the habits that keep you
from a very bad afternoon.

## INSERT: add rows

**`INSERT`** adds new rows. Name the columns you're supplying and give their values:

```sql
INSERT INTO systems (name, frequency_hz, protocol)
VALUES ('Harbor P25', 852000000, 'P25');
```

Columns you don't list take their defaults (or NULL) — that's how an auto-generated
`id` and a `first_seen DEFAULT now()` fill themselves in. Naming the columns explicitly
(rather than relying on their order) is the robust habit: it keeps working when someone
adds a column later. You can insert several rows at once by listing multiple value
tuples:

```sql
INSERT INTO systems (name, frequency_hz, protocol) VALUES
  ('County DMR', 453100000, 'DMR'),
  ('Rail TETRA', 390000000, 'TETRA');
```

Often you want the new row's generated id straight back. Many databases offer a
**`RETURNING`** clause for exactly that:

```sql
INSERT INTO systems (name, frequency_hz, protocol)
VALUES ('Harbor P25', 852000000, 'P25')
RETURNING id;
```

No second query needed — the new `id` comes back with the insert.

## UPDATE: change existing rows

**`UPDATE`** modifies rows already in the table. You set new column values and — this is
the important part — you say *which* rows with **`WHERE`**:

```sql
UPDATE systems
SET active = FALSE
WHERE id = 3;
```

That deactivates exactly system 3. The `WHERE` uses everything from the
[filtering lesson](/learn/databases/filtering-and-sorting/) — you can update all rows
matching any condition, and set several columns at once with commas.

## The WHERE you forget at your peril

Here is the most important sentence in this lesson. **UPDATE and DELETE without a WHERE
clause apply to every row in the table.** There's no "are you sure?" — the database does
precisely what you wrote:

```sql
UPDATE systems SET active = FALSE;   -- deactivates EVERY system
DELETE FROM systems;                 -- empties the ENTIRE table
```

This is the classic career-defining mistake, and it's made by experienced people, not
just beginners. Two habits defend against it:

- **Write the WHERE first.** Type the `WHERE` clause before the `SET` or even before
  `UPDATE`/`DELETE`, so a half-finished statement can't fire against everything.
- **SELECT before you write.** Run `SELECT * FROM systems WHERE id = 3;` first, confirm
  it returns exactly the rows you mean to change, then swap `SELECT *` for your UPDATE
  or DELETE with the *same* WHERE.

## DELETE: remove rows

**`DELETE`** removes rows matching its WHERE:

```sql
DELETE FROM calls
WHERE started_at < '2026-01-01';
```

That prunes old calls. As with UPDATE, the WHERE is what stands between "delete the old
ones" and "delete everything." Note that a DELETE can be blocked by
[referential integrity](/learn/databases/keys-and-relationships/): if other rows have a
foreign key pointing at the row you're deleting, the database may refuse (or cascade the
delete), which is the guardrail working as designed.

## Writes deserve transactions

Reads are safe; writes are not, and things go wrong — a bug computes the wrong values, a
multi-step change fails halfway. That's why important writes run inside a
**transaction**: a group of statements that either *all* take effect or *none* do. If
anything looks wrong before you finish, you **`ROLLBACK`** and it's as if nothing
happened:

```sql
BEGIN;
UPDATE accounts SET balance = balance - 100 WHERE id = 1;
UPDATE accounts SET balance = balance + 100 WHERE id = 2;
COMMIT;   -- or ROLLBACK to undo both
```

If the second update fails, you roll back and the first is undone too — you never leave
money removed from one account but not added to the other. Transactions and the
guarantees behind them (the "ACID" properties) are a full topic later in the module; for
now, know that once a DELETE is **committed**, it's gone, and your only recovery is a
[backup](/learn/deployment/backups-and-data/) — one more reason those exist.

<div class="knowledge-check" data-quiz data-correct-msg="Right — without a WHERE clause, UPDATE and DELETE apply to every row in the table." markdown="0">
  <p class="knowledge-check__q">Quick check: what does <code>DELETE FROM systems;</code> with no WHERE do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Nothing — a DELETE requires a WHERE clause to run</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Deletes every row in the table — no confirmation, no undo once committed</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Deletes only the most recently inserted row</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The write side of SQL is **INSERT** (add rows), **UPDATE** (change rows), and
  **DELETE** (remove rows) — collectively **DML**.
- **INSERT** names columns and supplies values; unlisted columns take defaults, and
  **RETURNING** hands back generated values like a new id.
- **UPDATE** and **DELETE** use a **WHERE** clause to choose rows — and **without one
  they hit every row in the table**, with no undo once committed.
- Defend against that with two habits: **write the WHERE first**, and **SELECT to
  preview** the rows before writing.
- A **DELETE** may be blocked or cascaded by **foreign-key** integrity — the guardrail
  working.
- Important writes run inside a **transaction** so you can **ROLLBACK**; once
  **committed**, recovery means a **backup**.

Next up: [Indexes & how queries get fast](/learn/databases/indexes/).
