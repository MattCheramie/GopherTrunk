---
slug: backups-and-images
title: Backups & images
description: An appliance you can re-flash in ten minutes is an appliance you never fear losing — full SD-card image backups, config exports with rsync, what to back up versus regenerate, and testing the restore.
keywords: sd card image backup, dd backup, rsync backup, config backup, restore test, golden image, raspberry pi backup, appliance recovery
level: intermediate
status: full
prereq:
  - flashing-an-os-image
  - remote-administration
---

# Backups & images

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Storage fails, cards corrupt, experiments go wrong — the appliance answer is a
**restore path measured in minutes**. Two complementary layers: a **full image**
(a byte-for-byte snapshot of the whole card — your "golden image" of the working
system) and ongoing **config + data exports** (rsync of the files that change).
Know your **regenerate vs restore** split — the OS and packages can be rebuilt
from any fresh image, but *your configuration and data* exist nowhere else. And
one law above all: **an untested backup is a hope, not a backup** — prove the
restore once.
</div>

Unit 5 has armoured the board against heat, wear, and hangs. The last layer
assumes total loss anyway — dead card, botched upgrade, lightning — and makes it
cost ten minutes. This is the cheapest insurance in the whole build.

## What's actually irreplaceable on the card?

Sort the card's contents by how you'd get them back:

| Category | Examples | Recovery without a backup |
|----------|----------|---------------------------|
| **OS + packages** | Raspberry Pi OS, rtl-sdr, the GopherTrunk binary | Re-flash + reinstall — tedious but certain |
| **Configuration** | `/etc/gophertrunk/config.yaml`, unit files, journald/watchdog tweaks, SSH setup | **Gone.** Hours of accumulated decisions |
| **Data** | Recordings, call database, labels | **Gone.** Irreplaceable history |

That middle row is the sleeper: a working appliance embodies dozens of small
decisions — the tuned sample rate, the talkgroup labels, the journal cap, the
static-address choice — made once and forgotten. Losing the *system* costs an
evening; losing the *decisions* costs the weeks it took to make them.

## How do you take a full image?

The **full image** is the inverse of [flashing](/learn/embedded/flashing-an-os-image/):
read the whole card into a file. With the card in your PC's reader:

```bash
$ sudo dd if=/dev/sdX bs=4M status=progress | gzip > scanner-2026-08-21.img.gz
```

Restoring is flashing that file like any OS image. Notes that matter:

- **When:** at milestones — the appliance first fully working, and after any
  significant reconfiguration. This is your **golden image**; it doesn't need to
  be frequent, it needs to *exist*.
- **Cold copies are clean copies.** Imaging a *running* system risks catching
  files mid-write. Cleanest: shut down and pull the card. The compromise —
  imaging a mostly-idle running board with the daemon stopped — mostly works;
  the journalled filesystem usually forgives. For the golden image, do it cold.
- **Compress** (as above) — the empty space squeezes to almost nothing.
- Tools like **rpi-clone** can mirror a running system to a second card/SSD,
  giving you a hot-spare card ready to swap — a lovely appliance trick.

## How do you keep the changing files flowing out?

The image is a milestone; config and data change weekly. For those, **pull
copies off the board on a schedule** — small, fast, no downtime, powered by
[last lesson's](/learn/embedded/remote-administration/) rsync habit, run *from*
your PC or home server:

```bash
$ rsync -av scanner.local:/etc/gophertrunk/ backups/scanner/etc-gophertrunk/
$ rsync -av scanner.local:/etc/systemd/system/gophertrunk.service backups/scanner/
$ rsync -av scanner.local:/var/lib/gophertrunk/ backups/scanner/data/
```

Schedule it (cron on the PC/server —
[Scheduling with cron](/learn/linux-cli/scheduling-with-cron/)), and keep dated
copies rather than overwriting one — yesterday's config is precious precisely
when today's edit broke something. Recordings deserve a policy decision: archive
them all off-board, or accept the retention window as the archive
([SD-card wear](/learn/embedded/sd-card-wear/) set those caps). The full
backup discipline — 3-2-1 copies, offsite — lives in the
[Deployment module](/learn/deployment/backups-and-data/); for a home appliance,
*any scheduled copy on a second machine* clears the bar that matters.

## Why must you rehearse the restore?

Because every backup story ends one of two ways, and you choose now: the restore
works, or you discover — on failure day, under stress — that the image was
truncated, the rsync missed a directory, or the restored config referenced a
path that doesn't exist. Run the drill once, calmly:

1. Flash the golden image to a **spare card** (never overwrite the working one).
2. Boot the board on it. Confirm SSH, service up, console reachable.
3. Apply the latest config export over it; confirm the daemon decodes.
4. Write down the minutes it took and any surprise. Fix the surprises.

Now "the card died" is a documented procedure with a known duration — ten-ish
minutes of calm — and, as a bonus, risky experiments lose their terror: you can
always get back to golden.

> Rule of thumb: you don't have a backup until you have restored from it once.
> Everything before that is optimism with a filename.

<div class="knowledge-check" data-quiz data-correct-msg="Right — configuration and data exist nowhere else; the OS can always be rebuilt from a fresh image." markdown="0">
  <p class="knowledge-check__q">Quick check: which loss is truly unrecoverable without a backup?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The operating system — it can never be reinstalled once corrupted</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The GopherTrunk binary — each download is unique to the board</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Your configuration and recorded data — everything else can be regenerated from fresh images and downloads</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Aim for a **restore path in minutes**: a **golden image** of the working card
  plus scheduled **config/data exports**.
- Sort by **regenerate vs restore** — OS and packages rebuild; *configuration
  and data are irreplaceable* and are the real backup targets.
- Image at **milestones**, cold for cleanliness, compressed; rsync the changing
  files on a schedule, keeping **dated copies** on another machine.
- **Rehearse the restore** on a spare card — timed, documented, surprises fixed.
- A proven restore makes storage failure boring *and* makes experimentation
  safe: you can always get back to golden.

Next up: [Monitoring your board](/learn/embedded/monitoring-your-board/).
