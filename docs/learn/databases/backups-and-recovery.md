---
slug: backups-and-recovery
title: Backups & recovery
description: The single most important operational habit for a database — backups you have actually restored, the difference between a dump and point-in-time recovery, and why an untested backup is not a backup at all.
keywords: database backup, restore, point-in-time recovery, PITR, logical dump, physical backup, WAL, RPO RTO, tested backups, disaster recovery
level: intermediate
status: full
prereq:
  - transactions-and-acid
faq:
  - q: "What's the difference between a backup and point-in-time recovery?"
    a: "A plain **backup** is a snapshot of your data at one moment — restore it and you're back to how things were then, losing everything since. **Point-in-time recovery** (PITR) combines a base backup with a continuous log of every change, so you can restore to *any* moment — say, the second before an accidental `DELETE` — instead of only to the last snapshot."
  - q: "Why do people say an untested backup isn't a backup?"
    a: "Because a backup you've never restored is a guess, not a guarantee. Backups fail silently — a broken cron job, a corrupt file, a missing table — and you find out only when you try to restore during a real emergency, which is the worst possible time. Restoring regularly to a scratch environment is the only way to know it works."
  - q: "What are RPO and RTO?"
    a: "Two targets that shape your backup strategy. **RPO** (recovery point objective) is how much recent data you can afford to lose — an hour, a minute, none. **RTO** (recovery time objective) is how long you can afford to be down while restoring. Tighter targets cost more; you pick them deliberately per system."
---

# Backups & recovery

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Backups are the most important operational habit a database has, and the rule that
matters most is that **an untested backup is not a backup** — you must actually
restore it, regularly, before you need it. Know the difference between a **snapshot**
(restore to one moment) and **point-in-time recovery** (restore to any moment), and
set your **RPO** and **RTO** — how much data and downtime you can tolerate — on
purpose. This is the database side of [backups & data in deployment](/learn/deployment/backups-and-data/).
</div>

Every other lesson in this unit is about keeping a database fast and available.
This one is about the day it goes wrong — a disk fails, a bad migration corrupts a
table, someone runs a `DELETE` without a `WHERE`. On that day, the only thing
standing between you and permanent data loss is a backup you can actually restore.
It is unglamorous work, which is exactly why it's so often neglected until it's
too late. Get it right and everything else is recoverable.

## Backups are the floor

A running database can fail in ways that no amount of good code prevents: hardware
dies, a region goes dark, a deploy ships a destructive bug, or a person makes a
mistake. Transactions and constraints keep your data *consistent*; they do nothing
if the data is *gone*. Backups are the floor beneath everything — the guarantee that
however bad the failure, you can get back to a known-good state.

Because they're the floor, they deserve more care than any single feature. A lost
week of decoded calls is an annoyance; a lost *database* with no backup is often the
end of a project. Treat backups as the non-negotiable baseline, not an optimisation.

## Snapshots vs. point-in-time recovery

There are two broad shapes of backup, and the difference determines how much you can
lose.

- **A snapshot / dump** captures the whole database at one moment. Restore it and you
  return to exactly that moment — but everything that happened *after* the snapshot is
  lost. If you snapshot nightly and the database dies at 5pm, you lose the day. A
  **logical dump** exports data as SQL statements (portable, slower to restore); a
  **physical backup** copies the underlying files (faster, tied to the engine).

- **Point-in-time recovery (PITR)** goes further. You keep a base backup *plus* a
  continuous record of every change the database makes — the **write-ahead log** (WAL)
  or equivalent. To recover, you restore the base and then replay the log up to a
  chosen instant. That lets you rewind to *any* moment — crucially, to the second
  **before** a mistake — instead of only to the last snapshot.

```bash
# Logical dump of a Postgres database (a snapshot in time)
pg_dump --format=custom gophertrunk > gophertrunk-2026-08-04.dump

# Restore it into a fresh database
pg_restore --dbname=gophertrunk_restored gophertrunk-2026-08-04.dump
```

Snapshots are simple and often enough for a small project; PITR is what you want when
losing even an hour of data is unacceptable.

## RPO and RTO

How much backup machinery you need isn't a matter of taste — it follows from two
numbers you choose per system:

- **RPO — recovery point objective.** How much recent data can you afford to lose?
  Nightly snapshots mean an RPO of up to a day. Continuous WAL archiving pushes it
  toward seconds. A smaller RPO costs more.
- **RTO — recovery time objective.** How long can you be down while you restore? A
  huge database restored from a logical dump can take hours; a physical snapshot or a
  standby replica can cut that to minutes.

Deciding these on purpose keeps you honest. "We back up nightly and a full restore
takes three hours" is a real RPO and RTO — you just have to confirm the business can
live with them before the outage, not during it.

## The rule that matters most: test the restore

Here is the lesson that costs people their data: **a backup you have never restored is
not a backup — it's a hope.** Backups fail silently in every imaginable way. A cron
job stops running after a server move. A dump completes but excludes a schema. The
files are written to a disk that's also failing. The compression is corrupt. You
cannot tell any of this from the fact that a backup *file exists*.

The only proof is to **restore it, regularly, into a scratch environment** and check
the data is all there and usable. Automate a periodic restore-and-verify so a broken
backup surfaces on an ordinary Tuesday, not during a 3am incident. Many teams that
"had backups" discovered during a real disaster that the backups had been unusable
for months. Restore drills are what separate a real recovery plan from a false sense
of safety.

## Where and how many copies

A backup on the same server as the database dies with the server. The durable habit
is captured by the old **3-2-1 rule**: keep **3** copies of the data, on **2**
different kinds of media or systems, with **1** copy off-site (a different region or
provider). Encrypt backups — they contain all your data, so a stolen backup is a full
breach — and control who can access and delete them, since an attacker or a buggy
script that can wipe your backups can turn a recoverable incident into a permanent
loss. The deployment path treats the operational side of this in
[backups & data](/learn/deployment/backups-and-data/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — a backup you've never restored is only a hope; failures are silent, so you must regularly restore and verify." markdown="0">
  <p class="knowledge-check__q">Quick check: why is "an untested backup is not a backup"?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Because backups always work as long as the file exists</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Because backups fail silently, and only actually restoring one proves it works</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Because you must encrypt a backup before it counts</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Backups are the **floor** beneath a database — transactions keep data consistent,
  but only a backup brings it back once it's gone.
- A **snapshot / dump** restores to one moment (losing everything after);
  **point-in-time recovery** replays a change log to restore to *any* moment, including
  just before a mistake.
- Set **RPO** (how much data you can lose) and **RTO** (how long you can be down) on
  purpose — they drive how much backup machinery you need.
- The rule that matters most: **test the restore.** An unrestored backup is a hope;
  failures are silent, so restore and verify regularly in a scratch environment.
- Keep multiple copies, at least one **off-site** (3-2-1), **encrypt** them, and guard
  who can delete them.

Next up: [replication, sharding & scaling](/learn/databases/replication-and-scaling/).
