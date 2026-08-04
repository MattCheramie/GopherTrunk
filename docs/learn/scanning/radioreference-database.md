---
slug: radioreference-database
title: RadioReference & system databases
description: How the scanning community's shared databases — RadioReference chief among them — tell you what's on the air near you, from conventional frequencies to full trunked-system profiles, and how to read a system page without getting lost.
keywords: RadioReference, scanner database, RRDB, trunked system database, control channel frequency, talkgroup list, county frequencies, system profile, scanner programming, FCC ULS
level: beginner
status: full
faq:
  - q: What is RadioReference?
    a: RadioReference.com is the largest community-maintained database of radio systems in North America. Volunteers document conventional frequencies, trunked-system parameters, talkgroups, and site details, organised mostly by county and agency. It is where nearly every scanner hobbyist starts when they want to know what is on the air near them.
  - q: Do I have to pay to use it?
    a: The core frequency and talkgroup data is free to browse. A modest paid membership unlocks bulk export, the mobile apps, and some convenience features, but you can find a control channel and a talkgroup list without spending anything. Most people join once they are hooked and want faster programming.
  - q: Is everything in the database?
    a: No. Databases are only as complete as their volunteers, so new systems, recently rebanded sites, low-traffic business licences, and anything private may be missing or out of date. When a database comes up empty, that is exactly when the searching, discovery, and identification skills in the rest of this unit earn their keep.
---

# RadioReference & system databases

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Before you tune a single frequency, look up what is already documented. **RadioReference**
is the scanning community's shared map — a free-to-browse database of **conventional
frequencies**, **trunked-system parameters**, and **talkgroups**, organised by county and
agency. A system page hands you the **control-channel frequency** and **system type** you need
to start following a trunked system. Databases are never complete, though, so treat them as a
head start, not the whole story — the [searching](/learn/scanning/searching-and-discovery/) and
[identifying](/learn/scanning/identifying-unknown-signals/) lessons cover what they miss.
</div>

The fastest way to fill a scanner with things worth hearing is to stand on the work others
have already done. Somewhere, a local hobbyist has probably already logged the frequencies,
identified the systems, and named the talkgroups for your area — and posted them to a shared
database. This lesson is about finding that work and reading it, so you spend your air time
listening instead of rediscovering what is already known.

## Why a shared database exists

Radio scanning has always been a community hobby. Long before the web, listeners traded
printed frequency lists and club newsletters; today that collective knowledge lives in online
databases that anyone can search. The largest by far is **RadioReference.com** (often just
"the RRDB"), which documents tens of thousands of systems across North America, with sister
projects and regional databases covering other parts of the world.

The value is not any single frequency — it is the **structure**. A good database doesn't just
tell you that 154.265 MHz is active; it tells you it's the county fire dispatch, what tone it
uses, and which neighbouring channels belong to the same agency. For trunked systems the
payoff is bigger still: a single page can hand you the control channel, the system type, every
site, and a full talkgroup list.

## What's in a database

Most databases organise around a few core record types. Learning to recognise them makes any
system page readable:

- **Conventional frequencies** — a plain list of fixed channels: frequency, mode (FM, digital),
  any subaudible tone or colour code, and a plain-language description like "PD Dispatch."
  These are the park-on-a-channel systems from the
  [conventional-vs-trunked](/learn/scanning/conventional-vs-trunked-recap/) recap.
- **Trunked systems** — a richer profile: the system name and type (P25, DMR, and so on), its
  **system identifiers**, a list of **sites** with their **control-channel frequencies**, and
  the **talkgroups** that ride on it.
- **Talkgroups** — the logical channels within a trunked system, each with an ID, an alpha tag,
  and a description. This is the list you'll turn into [scan
  lists](/learn/scanning/talkgroups-and-scan-lists/) later.
- **Licence data** — many databases surface the official FCC (or national regulator) licence
  records, which show who is legally assigned a frequency even when no one has logged it on the
  air yet.

## Reading a trunked system page

A trunked-system page is the one that intimidates newcomers, because it packs the most in.
Read it top to bottom and it settles into a clear shape. The **header** names the system and
its type — the single most important line, because it tells you which decoder to use. Below
that, a **sites** table lists each transmitter site with its control-channel frequency (and
often alternates); this is what you point your receiver at, and it's the same value the
[finding the control channel](/learn/digital-trunking/finding-the-control-channel/) lesson
teaches you to confirm on the air.

Further down sits the **talkgroups** table — the actual conversations. Each row has a numeric
ID, a short **alpha tag**, and a description, so you can decide what's worth hearing before a
single call comes in. You don't need to understand every field to get started: the system type
and one control-channel frequency are enough to begin, and the talkgroups tell you what you'll
hear once you're locked.

## Databases are a starting point, not gospel

The great strength of a community database — that volunteers maintain it — is also its limit.
Coverage is uneven: a well-documented metro system might have every talkgroup named, while the
county next door is a stub. Records go stale when a system rebands, adds a site, or is replaced,
and encrypted or newly built systems may never appear at all.

So treat a database as a strong first draft. When the page is complete, you've saved hours;
when it's thin or empty, that's your cue to switch to the discovery skills in the rest of this
unit — searching a band, watching the [waterfall](/learn/rf-sdr/fft-and-waterfall/), and
identifying what you find. The most rewarding catches are often the ones no database listed.

## Give back what you learn

The databases stay useful only because listeners contribute. When you confirm a frequency,
identify an unlisted talkgroup, or catch a new site, submitting that back keeps the shared map
alive for the next person. You don't have to — but a hobby built on shared knowledge works
best when the people it helps also feed it. The
[frequency records](/learn/scanning/frequency-records/) lesson shows how to keep your own logs
in a shape that makes contributing painless.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the system type and a control-channel frequency are the two things a database gives you to start following a trunked system." markdown="0">
  <p class="knowledge-check__q">Quick check: from a trunked system's database page, what two things do you most need to start following it?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The number of talkgroups and the licence expiry date</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The system type and a control-channel frequency</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The tower's height and the transmit power</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **RadioReference** and similar databases are the scanning community's shared map — the
  fastest way to learn what's on the air near you.
- Records come in a few types: **conventional frequencies**, **trunked systems**, **talkgroups**,
  and official **licence data**.
- A trunked-system page hands you the **system type** and a **control-channel frequency** — the
  two things you need to start following it.
- Databases are **never complete**: coverage is uneven and records go stale, so treat them as a
  starting point.
- When a page is thin, switch to **searching and identifying**; when you learn something new,
  **give it back** so the map stays current.

Next up: [band plans &amp; where services live](/learn/scanning/band-plans/).
