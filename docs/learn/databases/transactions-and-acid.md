---
slug: transactions-and-acid
title: Transactions & ACID
description: A transaction groups several database changes into one all-or-nothing unit that either fully happens or not at all. Learn the ACID guarantees — atomicity, consistency, isolation, durability — and why moving money or updating two tables needs them.
keywords: transactions, ACID, atomicity, consistency, isolation, durability, commit, rollback, database transaction, all or nothing, concurrency, isolation levels
level: intermediate
status: full
prereq:
  - inserting-updating-deleting
  - constraints-and-integrity
faq:
  - q: What is a transaction, in one sentence?
    a: "A group of database operations treated as a single all-or-nothing unit: either every operation in the group takes effect (commit) or none of them do (rollback), so the data is never left half-changed."
  - q: What do the letters in ACID stand for?
    a: "Atomicity (all-or-nothing), Consistency (the database moves from one valid state to another, honoring constraints), Isolation (concurrent transactions don't see each other's half-done work), and Durability (once committed, the change survives a crash). Together they're the guarantees a relational database makes about transactions."
  - q: When do I actually need a transaction?
    a: "Whenever a single logical change touches more than one row or table and a partial result would be wrong — transferring money between two accounts, creating an order and its line items, or updating a record and a counter that must stay in sync. If a half-completed version of the change would corrupt your data, wrap it in a transaction."
---

# Transactions & ACID

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **transaction** groups several changes into one **all-or-nothing** unit: it either
**commits** entirely or **rolls back** entirely, so the data is never left
half-changed. The guarantees behind it are **ACID** — **Atomicity** (all or
nothing), **Consistency** (constraints stay satisfied), **Isolation** (concurrent
transactions don't see each other's unfinished work), and **Durability** (a commit
survives a crash). You need one whenever a single logical change spans more than one
write and a partial result would corrupt your data. It works hand in hand with the
[constraints](/learn/databases/constraints-and-integrity/) from the last lesson.
</div>

Constraints keep any single row valid. But real changes often span several rows or
tables at once — and between the first write and the last, a crash, an error, or
another user can catch your data mid-change, in a state that's briefly *wrong*.
Transactions are the mechanism that makes a multi-step change behave like a single
indivisible one.

## The canonical example: moving money

The textbook case, for good reason. Transfer 100 units from account A to B, and it
takes two writes:

```sql
UPDATE accounts SET balance = balance - 100 WHERE id = 'A';
UPDATE accounts SET balance = balance + 100 WHERE id = 'B';
```

Now imagine the process crashes, the connection drops, or an error fires *between*
those two statements. Money has left A and never arrived at B. It has simply vanished.
No constraint catches this, because each row is individually valid — A's balance is a
real number, B's is a real number — yet the *system* is corrupt. The problem isn't any
one write; it's that the two writes must happen **together or not at all**.

## All-or-nothing: BEGIN, COMMIT, ROLLBACK

A transaction wraps the group so the database treats it as one unit. You open it with
`BEGIN`, do your writes, and finish with `COMMIT` to make them all permanent — or
`ROLLBACK` to throw them all away:

```sql
BEGIN;
    UPDATE accounts SET balance = balance - 100 WHERE id = 'A';
    UPDATE accounts SET balance = balance + 100 WHERE id = 'B';
COMMIT;
```

If anything goes wrong before `COMMIT` — a crash, an error, an explicit `ROLLBACK` —
the database undoes *everything* since `BEGIN`, as if the transaction never ran. There
is no state in which A was debited but B wasn't. That's the promise: the change is
**atomic**, one indivisible step from the outside, no matter how many writes it takes
inside.

## ACID: the four guarantees

Transactions are backed by four properties, abbreviated **ACID**. They're the
guarantees a relational database makes, and each one earns its letter.

- **Atomicity** — all or nothing. Every operation in the transaction takes effect, or
  none does. This is the money-transfer guarantee: no half-done changes survive.
- **Consistency** — the transaction moves the database from one **valid state to
  another**. Every constraint, foreign key, and check still holds after commit; if a
  transaction would violate one, it's rejected whole. Transactions and constraints
  reinforce each other here.
- **Isolation** — concurrent transactions don't step on each other. While your transfer
  is mid-flight, other transactions don't see A debited-but-B-not; they see the state
  either fully before or fully after. It's *as if* transactions ran one at a time, even
  when they overlap.
- **Durability** — once the database says `COMMIT` succeeded, the change is permanent.
  It's written to durable storage and will survive a power loss or crash the instant
  after. A confirmed commit is a kept promise.

## Isolation is where concurrency lives

Atomicity and durability are about a single transaction surviving failure; **isolation**
is about many transactions running *at the same time* without corrupting each other. If
two people transfer money involving account A simultaneously, isolation stops their
reads and writes from interleaving into a wrong total — for instance, both reading the
old balance and both subtracting from it, losing one of the deductions.

Perfect isolation is expensive, so databases offer **isolation levels** — a dial
trading strictness for speed. Lower levels allow certain anomalies (a transaction
seeing another's uncommitted or newly-inserted rows) in exchange for more concurrency;
the strictest level, **serializable**, makes transactions behave exactly as if they ran
one after another. Most applications run at a sensible middle default and only reach for
serializable on the operations that truly can't tolerate a race. You don't need the
details yet — just the idea that isolation is a setting, and that concurrency bugs live
where it's set too loose.

## When you need one — and when you don't

Reach for a transaction whenever a **single logical change touches more than one row or
table and a partial result would be wrong**:

- Transferring a value between two rows (the money case).
- Creating a parent and its children together — an order and its line items, a call and
  its audio-file record — so you never have one without the other.
- Updating a row and a running counter or summary that must stay in agreement.
- Any sequence where step two depends on step one having really happened.

You *don't* need to think about it for a single `INSERT`, `UPDATE`, or `DELETE`: those
are already atomic on their own — the database runs each statement as its own tiny
transaction. Transactions matter precisely when *one* meaningful change is made of
*several* statements. Wrapping those in `BEGIN`/`COMMIT` is what keeps a database
correct not just at rest, but through failure and concurrency — the reason relational
databases are trusted with data that has to be right.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the two updates must both happen or both not, or money vanishes; a transaction makes them one atomic unit." markdown="0">
  <p class="knowledge-check__q">Quick check: why does transferring money between two accounts need a transaction?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Because updating a balance is too slow to do without one</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Because the debit and credit must both happen or both not — a crash between them would lose money</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Because balances can't be negative and only a transaction can check that</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **transaction** groups multiple changes into one **all-or-nothing** unit: `BEGIN`,
  then `COMMIT` to keep everything or `ROLLBACK` to discard everything.
- It prevents **partial results** — the classic case is money debited from one account
  but never credited to another after a crash.
- **ACID** names the four guarantees: **Atomicity** (all or nothing), **Consistency**
  (constraints stay satisfied), **Isolation** (concurrent transactions don't see each
  other's unfinished work), **Durability** (a commit survives a crash).
- **Isolation levels** dial strictness against speed; **serializable** is strictest, and
  concurrency bugs appear where isolation is set too loose.
- Use a transaction whenever **one logical change spans several writes** and a partial
  version would be wrong; single statements are already atomic.

Next up: [Schema migrations & evolving a database](/learn/databases/migrations/).
