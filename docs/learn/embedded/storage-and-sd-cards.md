---
slug: storage-and-sd-cards
title: Storage & SD cards
description: SD cards, USB drives, and SSDs for single-board computers — speed classes, endurance ratings, why storage is the most common SBC failure, and which tier to buy for an always-on appliance.
keywords: SD card for Raspberry Pi, A1 A2 speed class, high endurance SD card, SSD boot Raspberry Pi, SBC storage, SD card failure, NVMe SBC, storage tiers
level: beginner
status: full
prereq:
  - picking-a-board
---

# Storage & SD cards

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Storage is the **most common failure point** of an SBC build. **SD cards** are slow
and wear out under constant writing; buy a **name-brand A2-class** card — or better,
a **"high endurance"** card built for continuous recording. The real upgrade is
booting from an **SSD** (USB or NVMe): faster in every way and dramatically more
durable. Whatever you choose, capacity is cheap — **buy bigger than you need** — and
assume the medium *will* eventually fail: the defence is the backup image, not the
perfect card.
</div>

The board you picked last lesson stores its entire life — OS, configuration,
recordings, logs — on one small piece of flash. This lesson is about choosing that
piece well, because a cheap card is the single most likely thing to take your
appliance down.

## Why is storage the weak link?

Flash memory wears: each cell survives a limited number of **write cycles**, and an
always-on Linux system writes constantly — logs, databases, temporary files, and (for
a scanner) a steady stream of audio recordings. A quality SSD manages this with
sophisticated wear-levelling across abundant spare cells. A bargain SD card has the
cheapest possible controller and no spare capacity to speak of. The typical death is
not dramatic: the system gets slow, then read-only, then unbootable — often months in,
long after you stopped watching. [SD-card wear](/learn/embedded/sd-card-wear/) in
Unit 5 covers reducing the write load; this lesson is about buying hardware that can
take it.

## How do you read an SD card label?

SD cards carry a fistful of overlapping ratings. The ones that matter:

| Marking | Means | Why you care |
|---------|-------|--------------|
| **A1 / A2** | "Application class" — random read/write performance | The best predictor of how fast Linux *feels*; buy A2 |
| **U1 / U3, V30…** | Sustained sequential write speed | Matters for video recording; less predictive for OS use |
| **"High endurance"** | Built (with wear headroom) for continuous recording, e.g. dashcams | The right product for a 24/7 appliance |
| Capacity | 32/64/128 GB… | Bigger cards also last longer — more cells to spread writes over |

Two buying rules beat all the ratings. **Buy a name brand from a reputable seller** —
counterfeit cards (fake capacity, dismal speed) are rampant on marketplaces. And
**bigger is better even if you don't need the space**, because wear-levelling across
more cells means each cell is written less often.

> Rule of thumb: for an always-on appliance, a 64 GB+ name-brand *high endurance*
> card is the floor. The few extra dollars are the cheapest reliability you will
> ever buy.

## When should you step up to an SSD?

An **SSD** — attached over USB 3, or on boards that support it, **NVMe** (a direct,
faster connection) — changes the storage story entirely:

- **Endurance**: real controllers, real spare area, wear ratings measured in written
  terabytes. Effectively removes the wear worry.
- **Speed**: boots, package updates, and database work go from sluggish to snappy;
  random I/O is often 10× an SD card's.
- **Reliability**: failure modes are rarer and more graceful.

The costs are modest: a bit more money, a bit more power draw (budget for it — next
lesson), and slightly more setup, since you configure the board to boot from USB/NVMe.
For a build whose whole point is running unattended for months while *writing
recordings continuously*, an SSD is the sensible default when the board supports it;
the high-endurance SD card is the budget path that also works.

## How much capacity does a scanner appliance need?

Sketch the arithmetic rather than guessing. The OS and GopherTrunk fit in well under
10 GB. Recordings dominate: compressed per-call audio is small individually, but a
busy system recording around the clock accumulates gigabytes per month, and call-log
databases grow alongside. A retention policy (delete recordings older than N days —
the [Scanning module](/learn/scanning/building-a-monitoring-post/) discusses this)
caps the total. With 64–128 GB and a sane retention policy you will run out of
interest before space. Log growth is the other quiet consumer —
[SD-card wear](/learn/embedded/sd-card-wear/) shows how to keep journals bounded.

## What's the mindset — durable media or disposable media?

Here is the reframe that separates frustrated beginners from calm operators: **assume
the storage will fail eventually, and make that boring.** You'll build this properly
in [Backups &amp; images](/learn/embedded/backups-and-images/): keep an image of your
configured system, keep configs exported, and a dead card becomes a ten-minute
re-flash instead of a lost weekend. Buy quality media to make failure *rare*; keep
backups to make it *irrelevant*. Do both.

<div class="knowledge-check" data-quiz data-correct-msg="Right — high-endurance cards are built for exactly the continuous-write life of an appliance." markdown="0">
  <p class="knowledge-check__q">Quick check: which SD card is the best fit for a 24/7 recording appliance?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The cheapest 32 GB card available, since all cards are the same inside</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A name-brand high-endurance card, sized larger than strictly needed</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Any card marked U1, since sequential video speed is what Linux needs</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Storage is the **most common SBC failure point**: flash cells wear out under an
  always-on system's constant writes.
- Read card labels for **A2** (random I/O) and **high endurance** (continuous-write
  headroom); buy **name-brand** and **bigger than needed**.
- An **SSD over USB 3 or NVMe** is the real upgrade — faster and dramatically more
  durable; the sensible default for an appliance when the board supports it.
- Size capacity by arithmetic: OS is small, **recordings plus retention policy**
  decide the number.
- Mindset: quality media makes failure **rare**; the **backup image** makes it
  **irrelevant**. Do both.

Next up: [Power supplies](/learn/embedded/power-supplies/).
