---
slug: data-and-persistence
title: Data, state & persistence
description: The difference between data your program holds in memory and data that outlives it. Learn what persistence and durability really mean, why "just save it" is harder than it looks, and why a database is the reliable place for state that must survive.
keywords: data persistence, state, in-memory, volatile, durability, save data, RAM vs disk, ephemeral data, fsync, crash safety, persistent storage
level: beginner
status: full
prereq:
  - what-is-a-database
faq:
  - q: "What's the difference between memory and persistence?"
    a: "Data in memory (RAM) lives only while your program runs — quit or crash and it's gone. Persistent data is written to durable storage (disk) so it survives the program ending, a restart, or a power loss. Persistence is the act of moving state from the volatile place to the lasting one."
  - q: "If I write to a file, is my data safe?"
    a: "Not necessarily. The operating system often buffers a write in memory before it actually reaches the disk, so a crash in that window loses it. True durability means the data is confirmed on disk — which is one of the guarantees a database is built to give you, and hand-rolled file code usually isn't."
  - q: "Does persistent mean permanent?"
    a: "No. Persistent means it survives the program that created it — a restart, a crash. It can still be deleted, overwritten, or lost if the disk fails. That's why persistence and backups are different concerns; you need both."
---

# Data, state & persistence

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Data your program holds in memory is **volatile** — it vanishes when the program
stops. **Persistence** is writing that **state** somewhere durable so it outlives
the process; **durability** is the stronger promise that a confirmed write really
reached the disk and survives a crash. "Just save it" is deceptively hard, which
is exactly the work a database takes off your hands — building on
[what a database is](/learn/databases/what-is-a-database/).
</div>

The last lesson said a program eventually needs to *remember* something. This one
is about what "remember" actually means, because there are two very different
kinds of memory in a computer — one that forgets the moment you stop, and one that
holds on. Knowing which is which, and how to move data safely between them, is the
foundation everything else in this module sits on.

## State: the data a program holds right now

While your program runs, it holds **state** — the current values of everything
it's working with. The user who's logged in, the list of calls decoded so far, a
half-filled form. This state lives in **memory** (RAM): fast, close to the CPU,
and easy to change. It's where all the actual work happens.

But RAM has one defining property: it's **volatile**. When the program exits — on
purpose, from a crash, or because the power blinked — everything in memory is
gone, instantly and completely. There's no undo. A program that keeps its data
only in memory is a program that forgets everything the moment it stops. For a
calculator, fine. For anything that has to remember across runs, that's the whole
problem.

## Persistence: making state outlive the program

**Persistence** is the fix: writing state out to **durable storage** — a disk, an
SSD, a database — so it survives the process ending. Persistent data is data you
can still read tomorrow, after a restart, after the machine was off all night.
The move from volatile to durable is the act of *persisting*.

The two live on very different terms:

| | In-memory state | Persistent data |
|---|---|---|
| **Lives in** | RAM | Disk / SSD / database |
| **Survives program exit?** | No | Yes |
| **Speed** | Extremely fast | Slower |
| **Capacity** | Limited by RAM | Much larger |
| **On a crash** | Lost | Survives (if durably written) |

Almost every real application is a dance between the two: load persistent data
into memory to work with it, and write changes back out so they aren't lost. The
in-memory copy is the working set; the persistent copy is the source of truth.

## "Just save it" is harder than it sounds

Here's the trap. Writing data to a file *looks* like persistence — you called
`write`, the data's on disk, done. Except often it isn't. For speed, the operating
system usually **buffers** your write in memory and reports success before the
bytes actually hit the disk platter. If the machine loses power in that window,
the write you thought was safe is gone.

**Durability** is the stronger guarantee: once the system confirms a write, the
data is truly on disk and will survive a crash. Getting there means explicitly
flushing buffers to the physical device (the `fsync` system call), and ordering
writes so a crash mid-operation leaves recoverable data, not a corrupt heap. Doing
that correctly, for concurrent writers, is genuinely hard engineering.

```go
// Naive: looks saved, may not survive a crash mid-flight.
os.WriteFile("calls.log", data, 0644)

// The OS may buffer this. Real durability needs an explicit
// flush to the physical disk — and correct ordering if a crash
// hits partway through. A database does this for you, correctly.
```

This is one of the biggest reasons to reach for a database. A DBMS makes
durability a *promise*: when it confirms a write, that data is committed and will
be there after a crash. You get to think about your data, not about `fsync`
ordering.

## Persistence isn't the same as permanent

One more distinction that saves grief later: **persistent** means "survives the
program," not "safe forever." A persisted file can still be deleted, overwritten
by a bug, or lost when the disk itself dies. Guarding against *that* is a separate
job — copies kept elsewhere, tested restores — which the deployment path covers in
[backups & data](/learn/deployment/backups-and-data/). Persistence keeps your data
across a restart; backups keep it across a disaster. Real systems need both, and
confusing the two is how people lose data they thought was safe.

<div class="knowledge-check" data-quiz data-correct-msg="Right — in-memory state is volatile and vanishes on exit; persistence writes it to durable storage that survives." markdown="0">
  <p class="knowledge-check__q">Quick check: what happens to data your program holds only in memory when it crashes?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It's automatically saved to disk by the operating system</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It's lost — memory is volatile, so unpersisted state vanishes</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Nothing — memory survives a crash just like disk does</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A program's **state** lives in **memory (RAM)**, which is **volatile** — it
  vanishes when the program stops or crashes.
- **Persistence** writes state to **durable storage** so it outlives the process; a
  restart later can read it back.
- **Durability** is the stronger promise that a *confirmed* write actually reached
  the disk — the OS often buffers writes, so a naive file save can still be lost.
- Getting durability right (flushing, crash-safe ordering, concurrent writers) is
  hard — a **database** makes it a guarantee so you don't have to.
- **Persistent** is not the same as **permanent**: persisted data can still be
  deleted or lost, which is why persistence and **backups** are separate concerns.

Next up: [The relational model — tables, rows & columns](/learn/databases/the-relational-model/).
