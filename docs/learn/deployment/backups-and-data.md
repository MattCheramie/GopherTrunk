---
slug: backups-and-data
title: Backups & persistent data
description: "Where a service's data lives, how to back it up and restore it, and why stateful data needs a plan that stateless containers don't."
keywords: backups, persistent data, stateful vs stateless, backup volumes, database backup, restore testing, 3-2-1 backup rule, disaster recovery
level: intermediate
status: full
prereq:
  - container-networking-and-volumes
faq:
  - q: What is the 3-2-1 backup rule?
    a: "Keep at least 3 copies of your data, on 2 different types of media or storage, with 1 copy off-site. The three copies survive a single deletion or corruption, two media types survive a whole class of failure, and the off-site copy survives fire, theft, or a compromised host. It's the minimum bar for data you'd be upset to lose."
  - q: What's the difference between stateful and stateless?
    a: "A stateless service keeps no important data of its own — restart it or replace it and nothing is lost, because its state lives elsewhere. A stateful service owns data that must survive, like a database or GopherTrunk's recordings and call log. Stateless parts are easy to scale and redeploy; stateful parts are the ones you must back up and restore carefully."
---

# Backups & persistent data

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Split your system into **stateless** parts (lose nothing on restart) and **stateful**
parts (own data that must survive). Only the stateful data needs backing up — but it
needs it seriously. Back up the **volumes and databases**, follow the **3-2-1 rule**
(three copies, two media, one off-site), and — the step everyone skips — **test your
restores**. A backup you've never restored is a guess, not a backup.
</div>

[Networking & volumes](/learn/deployment/container-networking-and-volumes/) showed how a
[volume](/learn/deployment/container-networking-and-volumes/) keeps data alive when a
container is replaced. That protects you from an *upgrade*; it does not protect you from a
deleted volume, a corrupted database, or a dead disk. Backups do — and knowing what's
stateful tells you exactly what to back up.

## Stateless vs stateful: know what to protect

The first question is *what actually holds data*. Sort every part of your system into two
buckets:

| | Stateless | Stateful |
|---|-----------|----------|
| Loses data on restart? | No | Yes |
| Examples | The app binary, a reverse proxy, a stateless API | A database, GopherTrunk's `calls.db`, its recordings |
| Recovery | Redeploy the artifact | Restore from backup |
| Backup needed? | No — rebuild from the image | Yes — this is the data that matters |

The GopherTrunk **binary** is stateless: lose the container and you just pull the image
again. Its **`calls.db` and recordings** are stateful: lose those and the history is gone
forever. Your backup plan targets the stateful bucket and ignores the stateless one.

## Back up the volumes and databases

For a file-based store like GopherTrunk's, a backup is a consistent copy of the volume.
For a SQLite database, use the database's own backup command so you capture a coherent
snapshot rather than a file caught mid-write:

```bash
# SQLite call database — .backup takes a consistent snapshot even while in use
sqlite3 /var/lib/gophertrunk/calls.db ".backup '/backups/calls-$(date +%F).db'"

# Recordings directory — a plain compressed archive
tar czf /backups/recordings-$(date +%F).tar.gz -C /var/lib/gophertrunk recordings
```

For a server database like Postgres you'd use `pg_dump` instead — same idea, a
consistent dump rather than copying live files. Schedule it with a
[systemd timer or cron](/learn/linux-cli/services-and-systemd/) so it runs unattended.

## Follow the 3-2-1 rule

One copy on the same disk as the original is barely a backup — the disk that dies takes
both. The **3-2-1 rule** is the durable minimum:

- **3** copies of the data (the live one plus two backups),
- on **2** different media or storage types,
- with **1** copy **off-site** — another machine, another building, or object storage.

```bash
# push the local backup off-site to object storage
aws s3 sync /backups s3://my-gophertrunk-backups/ --storage-class STANDARD_IA
```

The off-site copy is the one that survives fire, theft, ransomware, or a fat-fingered
`rm -rf` on the host.

## Test your restores — really

This is the step that separates real backups from hopeful ones. A backup you have never
restored is an untested assumption. Practice the restore on a scratch host on a schedule:

```bash
# restore into a throwaway location and confirm it opens and has data
sqlite3 /tmp/restore-test.db ".restore '/backups/calls-2026-07-23.db'"
sqlite3 /tmp/restore-test.db "SELECT count(*) FROM calls;"
```

If that count looks right, your backup works. If the file won't open, you just found out
during a *drill* instead of during a *disaster*. Restore time is also worth measuring —
knowing it takes 20 minutes shapes how you handle an [incident](/learn/deployment/incident-response/).

## Retention: don't keep everything forever

Backups cost storage, so decide how long to keep them. A common shape is **grandfather-
father-son**: keep daily backups for a week, weekly for a month, monthly for a year.
Prune older ones automatically so [cost](/learn/deployment/cost-and-resources/) stays
bounded while you retain enough history to recover from a problem you didn't notice for a
while.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an untested backup is only an assumption; restoring proves it works." markdown="0">
  <p class="knowledge-check__q">Quick check: what is the step most often skipped that makes a backup trustworthy?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Compressing the backup file</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Actually restoring it and confirming the data is intact</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Renaming it with the date</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Split the system into **stateless** (rebuild from the image) and **stateful** (must be
  backed up) parts.
- Back up **volumes and databases** with the store's own consistent-snapshot tool, on a
  schedule.
- Follow **3-2-1**: three copies, two media, one off-site.
- **Test restores** on a schedule — an unrestored backup is a guess.
- Set a **retention** policy so backups don't grow without bound.

Next up: locking the whole host down — production hardening.
