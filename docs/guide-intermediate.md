---
layout: page
title: Intermediate
description: Going further with GopherTrunk — the config file, multiple dongles, alias files, paging alerts, system discovery, and network access
nav_group: Getting Started
---

# Going further

The console gets you a long way, but GopherTrunk has more under the hood. This
guide is for when you're ready to open the **config file** and a **terminal** —
to run more than one dongle, decode paging, discover a system you don't have
details for, and reach the console from your phone.

> **Before you start.** This builds on [Everyday basics](guide-basics.html) — you
> should already be comfortable driving the web console and shaping talkgroups.

## The config file

Everything GopherTrunk does is described by one file, `config.yaml`. You have
two ways to edit it, and you can mix them freely:

- **Config Builder** — the guided, browser-based editor you met in
  [Get started](getting-started-setup.html). Best for adding systems and
  talkgroups without learning the YAML layout. It can also pull systems in from a
  RadioReference PDF or CSV — see [Import (PDF / CSV)](import.html).
- **By hand** — open `config.yaml` in any text editor. Once you know the layout,
  this is the fastest way to reach the settings the builder doesn't surface.

Many edits apply **without a restart**. On Linux and macOS you can also send the
daemon a `SIGHUP` to reload the file on the fly, and small tweaks made in the
console write straight back to the file — both covered in
[Live edits](live-edits.html).

## Run more than one dongle

A single dongle follows one control channel and records one call at a time. Add
a second (or third) dongle and GopherTrunk records **several calls at once** and
keeps watching the control channel while it does. Each dongle gets a **role**:

- **control** — stays parked on the control channel.
- **voice** — tunes to call frequencies as grants come in.
- **auto** — let GopherTrunk decide based on the pool.

On a system with two timeslots (DMR), a voice dongle can capture both slots as
separate recordings. Getting clean decodes also means **calibrating** each
dongle — its frequency error (PPM) and gain. The
[Hardware guide](hardware.html) covers dongle-by-dongle gain and PPM advice; if
you're on Windows, the [Windows user guide](user-guide-windows.html) walks
through the driver and device steps.

### One dongle, many channels

If you'd rather cover several carriers with a *single* dongle, the **wideband
channelizer** digitally slices one wide chunk of spectrum into multiple receivers
— you can even mix protocols inside the same window. It's a more advanced
allocation; the [Architecture reference](architecture.html) explains how the
tuner pool and channelizer fit together.

## Name your channels and radios

GopherTrunk reads **alias files** — CSV or JSON — that map raw numbers to
friendly names:

- **Talkgroup aliases** turn `1001` into "City Dispatch", and carry the scan,
  lockout, priority, and stream flags per channel.
- **Radio ID (RID) aliases** name individual radios. Paired with the live
  affiliation tracker, you can follow exactly who's on the air — see
  [Radio IDs](radio-ids.html).

RadioReference exports drop straight in through the [Import](import.html)
workflow, so you rarely type these by hand.

## Paging and tone alerts

GopherTrunk decodes more than trunked voice:

- **Paging** — turn on the [POCSAG](pocsag.html) and FLEX decoders to log pager
  traffic, including a wideband mode that catches several pagers on one dongle.
- **Tone-out alerts** — detect Motorola **Quick Call II** two-tone pages (the
  fire/EMS dispatch tones) and raise an alert, scoped to the systems and
  talkgroups you choose.

These and the other opt-in receivers — along with exactly which defaults apply
— are catalogued in [Opt-in features](opt-in-features.html).

## Discover an unknown system

Don't have the control-channel frequency or protocol for a system near you?
**Hunt** scans the band, finds trunked control channels, identifies the protocol
and system IDs, and hands you a ready-to-use config block (it can even prepare a
RadioReference submission). Full workflow in [Hunt](hunt.html).

## Reach the console from your phone

So far the console has been on the same machine as the daemon. To open it from a
laptop, tablet, or phone elsewhere on your network — the classic "headless
Raspberry Pi in the closet, watch from the couch" setup — you'll bind the daemon
to your network and **turn on authentication** so only you can reach it. Do this
deliberately: [Hardening](hardening.html) covers passwords, tokens, trusted
networks, and TLS.

## Where to go next

You can now run a multi-dongle setup, decode paging, name everything, find
systems on your own, and operate over the network. The last guide opens up the
power-user surface — the terminal cockpit, signal scopes, offline analysis, the
other receivers (aircraft, boats, packet), and the APIs:

**Next up → [Advanced & power-user features](guide-advanced.html)**
