---
slug: sd-card-wear
title: SD-card wear
description: Flash cells wear out with every write — why constant logging kills cheap SD cards, how to measure and reduce a system's write load, and the tiered defences that make storage last for years.
keywords: sd card wear, flash write endurance, wear leveling, journald SystemMaxUse, log2ram, tmpfs, reduce writes raspberry pi, sd card corruption, iostat
level: intermediate
status: full
prereq:
  - storage-and-sd-cards
  - services-with-systemd
---

# SD-card wear

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Flash cells endure a finite number of **write cycles**, and an always-on Linux box
writes *constantly* — logs above all. Cheap cards have weak **wear-levelling** and
no spare area, so the steady drip kills them in months. The defence is tiered:
**measure** the write load (`iostat`), **bound the journal**
(`SystemMaxUse=`), move scratch data to **RAM** (`tmpfs`, log2ram), keep
deliberate data (recordings) on media built for it — **high-endurance card or
SSD** — and cap growth with **retention policies**. Then let the backup image
make the eventual failure boring.
</div>

Heat taxes performance; wear kills the whole system — the most common way an SBC
appliance actually dies. The good news: write load is measurable and very
compressible, and the defences stack.

## Why do writes destroy flash?

A flash cell stores bits as trapped charge, and each **program/erase cycle**
slightly damages the insulating layer — after thousands of cycles the cell no
longer holds data. Controllers fight back with **wear-levelling** (spreading
writes across all cells) and spare capacity to retire failing blocks; quality
SSDs do this well, cheap SD cards barely
([Storage &amp; SD cards](/learn/embedded/storage-and-sd-cards/) was the buying
half of this story). Worse, flash erases in large blocks, so a tiny write can
cost a full block cycle — **write amplification** — and a stream of tiny log
lines is close to the worst case. The endgame is quiet: slowdowns, then files
that won't write, then a card stuck read-only, then no boot.

## Where do all the writes come from?

On an appliance, four suspects in rough order:

- **Logs.** The journal, plus every chatty service, syncing line after line to
  disk around the clock. The classic card-killer — a misbehaving service logging
  an error ten times a second is a wear catastrophe *and* invisible until you
  look.
- **Application data.** For the scanner: recordings and the call database —
  *deliberate* writes you want, sized by your retention policy.
- **Housekeeping.** Package updates, temp files — modest.
- **Swap.** Memory pressure turning RAM shortage into disk writes; on a
  well-sized board it should be near-idle.

Measure before optimising — `iostat` (from `sysstat`) shows the running total:

```bash
$ sudo apt install sysstat
$ iostat -h
Device     tps    kB_read/s    kB_wrtn/s    kB_read    kB_wrtn
mmcblk0   4.1        1.2k        58.3k       412.3M      19.8G
```

`kB_wrtn` since boot, divided by uptime, is your write rate. Tens of GB per day
on a cheap card is a countdown timer; a few hundred MB is years of life. Find the
culprits with a before/after diff or by watching which files grow (`du -sh
/var/log/*`).

## How do you cut the write load?

Tiered, cheapest first:

**Bound the journal.** systemd's journal happily grows to gigabytes; cap it:

```ini
# /etc/systemd/journald.conf
[Journal]
SystemMaxUse=64M
```

(`sudo systemctl restart systemd-journald` after.) You keep recent logs — which
is all an appliance needs, since [monitoring](/learn/embedded/monitoring-your-board/)
carries the long-term signal — at a fixed wear cost. Also calm any service
logging at debug verbosity in production.

**Move scratch to RAM.** A `tmpfs` is a filesystem living in memory — contents
vanish at reboot, wear cost zero. `/tmp` is often tmpfs already (check with
`df -h /tmp`); tools like **log2ram** go further, collecting logs in RAM and
flushing to card hourly — turning thousands of tiny writes into one big one
(write amplification working *for* you).

**Put deliberate data on real media.** Recordings and the database are the writes
you *want*. Give them a **high-endurance card** at minimum — or better, move the
data directory (or the whole system) to an **SSD**, which ends the wear
conversation. And bound them with **retention policies** — recordings pruned
after N days, database vacuumed — so growth is a flat line, not a ramp.

> Rule of thumb: an appliance's steady-state write rate should be something you
> have *measured once and can say out loud*. "About 300 MB a day, mostly
> recordings" is engineering; "no idea" is a countdown.

## What's the failure-day plan?

Even managed wear ends somewhere, so the last defence is attitude: the card is a
**consumable**. Watch for early warnings — filesystem remounting read-only,
I/O errors in `dmesg`, mysterious slowdowns — and keep the
[backup image](/learn/embedded/backups-and-images/) current, so failure day is:
new card, re-flash, restore, coffee still warm. Wear management buys *years*;
the backup buys *indifference*. An appliance wants both.

<div class="knowledge-check" data-quiz data-correct-msg="Right — continuous small log writes are the classic wear load, and bounding/redirecting them is the first fix." markdown="0">
  <p class="knowledge-check__q">Quick check: which write source most typically kills a cheap SD card in an always-on box?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Reading configuration files at every boot</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Continuous small log writes trickling to the card around the clock</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The monthly apt upgrade downloading packages</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Flash endures finite **write cycles**; cheap cards' weak **wear-levelling**
  plus **write amplification** turns constant small writes into early death.
- **Logs are the classic killer**; recordings and databases are deliberate load
  to place and bound properly.
- **Measure first** (`iostat`, `du`) — a write rate you can say out loud.
- Cut in tiers: **`SystemMaxUse=`** on the journal, **tmpfs/log2ram** for
  scratch, **high-endurance card or SSD** for real data, **retention policies**
  for growth.
- Treat the card as a **consumable**: heed read-only remounts and `dmesg` I/O
  errors, and keep the **backup image** current.

Next up: [Watchdogs &amp; recovery](/learn/embedded/watchdogs-and-recovery/).
