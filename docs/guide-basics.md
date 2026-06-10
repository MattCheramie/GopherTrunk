---
layout: page
title: Basics
description: Everyday GopherTrunk — drive the web console, listen to calls, tidy up your talkgroups, and share a feed
nav_group: Getting Started
---

# Everyday basics

You've got GopherTrunk running and your first call in the history list. This
page is the next step: the everyday skills for *operating* a scanner you've
already set up — all from the **web console**, no command line, no config files.

> **Before you start.** This page assumes you've finished
> [Get started](getting-started-setup.html) and have one system recording. If a
> control channel isn't locking and calls aren't appearing, go back there first.

## A tour of the console

Everything you do lives in the browser console the daemon opens for you. The
panels along the top (or the bottom on a phone) are different views of the same
running scanner. The five you'll use most:

- **Dashboard** — your home base. Live event feed, the audio controls, and an
  at-a-glance health readout.
- **Active** — every call happening *right now*, with a live timer. This is the
  panel to watch when a system is busy.
- **History** — the searchable call log. Filter by talkgroup, time, or system to
  find a call you missed.
- **Systems** — the trunked systems you've added and whether each one is locked.
- **Talkgroups** — the channels within a system: dispatch, tactical, fireground,
  and so on.

The full panel-by-panel reference is the [Web console guide](web.html).

## Listening and playback

GopherTrunk records calls whether or not your speakers are on, so there are two
ways to listen:

- **Live.** The **audio cockpit** at the top of the Dashboard plays calls as they
  happen. Set the volume, mute, or toggle recording from there. (On a phone or
  tablet you tap once to enable audio — a browser rule, not a GopherTrunk one.)
- **From history.** Open any past call in the **History** panel and play it back.
  Every recording is saved with its talkgroup and timestamp, so you can always
  go back and re-listen.

## Tidy up your talkgroups

Out of the box GopherTrunk follows *everything* a system carries. Once you know
which channels you actually care about, the **Talkgroups** panel lets you shape
what you hear:

- **Scan / lockout** — lock out a talkgroup to skip it entirely (data channels,
  test ranges, the chatty ones you don't want).
- **Priority** — mark the important channels so they're easy to spot.
- **Record / mute** — *record* keeps writing a talkgroup's calls to disk;
  *mute* keeps it recording but silences it in the live player. Handy for a busy
  channel you want logged but not blaring.

Changes you make here save on their own and mostly take effect without a
restart — see [Live edits](live-edits.html).

## Bookmarks and Radio IDs

Two more panels round out day-to-day use:

- **[Bookmarks](bookmarks.html)** — save interesting frequencies and jump back to
  them later.
- **[Radio IDs](radio-ids.html)** — see *which radio* is talking, not just which
  talkgroup. GopherTrunk tracks each radio's recent activity, and you can attach
  a friendly name (alias) to the ones you recognise.

## Scan more than one system

You're not limited to a single system. Add more from the Config Builder (the
same guided editor from [Get started](getting-started-setup.html)), and
GopherTrunk will hunt across them. The **scan mode** decides how much it follows:

- **All** — follow every call on every system (the default).
- **List** — follow only the talkgroups you've put on your scan list, so a busy
  system doesn't drown out the channels you came for.

## Share your feed

Want others to hear what you hear? From the **Settings** panel you can stream
your calls out to the popular aggregators — **Broadcastify Calls**,
**RdioScanner**, or **OpenMHz** — by pasting in the credentials they give you.
GopherTrunk handles the encoding and upload. (The finer controls — per-system
and per-talkgroup filtering, live Icecast/ShoutCast mountpoints — are covered in
the [next guide](guide-intermediate.html).)

## Where to go next

You can now run GopherTrunk as a scanner day to day. When you're ready to go
beyond the console — multiple dongles, paging alerts, discovering unknown
systems, and opening the console to your phone — keep going:

**Next up → [Going further (Intermediate)](guide-intermediate.html)**
