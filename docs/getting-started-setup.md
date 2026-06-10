---
layout: page
title: Get started
description: Download GopherTrunk and run your first scan from the web interface — no command line required
nav_group: Getting Started
---

# Get started

This is the simple, click-along path: download GopherTrunk, start it, and drive
everything from the **web interface** in your browser — no command line, no Git,
no editing config files by hand. New to the project? Read
[What is GopherTrunk?](getting-started.html) first for the big picture.

> Prefer the terminal, or running on a headless server / Raspberry Pi? The
> per-OS install guides cover the command-line path:
> [Linux](install-linux.html) · [macOS](install-macos.html) ·
> [Windows](install-windows.html).

## 1. Get a radio

You need an SDR dongle plugged into a USB port. An **RTL-SDR** (around $30) is
the easiest and cheapest place to start. Connect an antenna suited to the band
you want to listen to. Device-by-device advice is in the
[Hardware guide](hardware.html).

## 2. Download GopherTrunk

Go to the **[Downloads page](downloads.html)** and click the button for your
computer:

- **Windows** — click **x64 Installer (.exe)**, then run the downloaded file
  (double-click it) and follow the wizard, just like any other program. It adds
  a **GopherTrunk** shortcut to your Start Menu.
- **macOS** — click the **Apple Silicon** (newer Macs) or **Intel** button,
  open the downloaded file, and drag GopherTrunk out. The first time you open
  it, **right-click the app and choose "Open"** to get past the security prompt.
- **Linux** — download the file for your machine and unpack it.

That's the whole install — GopherTrunk is one self-contained program.

## 3. Windows only: turn on your radio driver

Windows needs a one-time driver step before it can see an RTL-SDR. The installer
bundles a tool that does this for you:

1. Open the Start Menu → **GopherTrunk** → **"Install RTL-SDR driver (Zadig)"**.
2. In the window that opens, pick your dongle from the list and click the
   install button.

Do this once per dongle. (Full screenshots are in the
[Windows install guide](install-windows.html).) macOS and Linux don't need this.

## 4. Start GopherTrunk and open the web interface

Launch GopherTrunk (on Windows, use the Start Menu shortcut). It asks how you
want to drive it — **choose Web**. Your browser opens to the GopherTrunk
console, where everything from here on happens with clicks.

The program keeps running in the background while you use the browser. To stop
it later, close the GopherTrunk window (or press `Ctrl-C` in its window).

## 5. Add a system to listen to — with the Config Builder

GopherTrunk needs to know *which* radio system to follow. The friendliest way to
set this up is the **Config Builder** — a guided, browser-based editor. You never
touch a config file.

Open it from the web console by clicking the **🛠 Config Builder** button (top of
the page) — it opens in a new tab. Inside, you can:

- **Browse RadioReference** for your county/agency and pull a system straight in,
  or **import a PDF/CSV** you downloaded from
  [RadioReference](https://www.radioreference.com/) — the builder fills in the
  frequencies and talkgroups for you.
- **Build a system by hand** — add a system, pick its protocol (for example P25
  or DMR), and type in its control-channel frequency.

The builder checks your settings as you go and **saves** them when you're done.
More detail is in the [Import guide](import.html).

> Quick alternative: the web console's own **Settings** page can also add a
> system inline, and small tweaks made anywhere in the console save
> automatically and mostly apply without a restart — see
> [Live edits](live-edits.html).

## 6. Watch your first call

Back on the dashboard, you're looking for this to happen on its own:

1. The control channel **locks** (GopherTrunk has tuned in to the system).
2. A **call** appears in the active-calls list as people talk.
3. The call is **recorded** and shows up in your history.

The [Web console guide](web.html) walks through what every panel shows.

## 7. Where to go next

From here, the guides pick up where this page leaves off and walk you through
the rest of GopherTrunk, one step at a time:

- **[Everyday basics](guide-basics.html)** — drive the console, listen and play
  back calls, tidy up your talkgroups, and share a feed. **Start here.**
- **[Going further](guide-intermediate.html)** — multiple dongles, paging alerts,
  alias files, discovering unknown systems, and reaching the console from your
  phone.
- **[Advanced](guide-advanced.html)** — the terminal cockpit, signal scopes,
  offline analysis, every receiver, and the APIs.

In a hurry to reach a specific feature? Jump straight to
**[Hunt](hunt.html)** (discover an unknown system),
**[Hardening](hardening.html)** (passwords/encryption for network access), or
**[Opt-in features](opt-in-features.html)** (paging, aircraft, boats, and more).

## If something's not working

- **GopherTrunk can't find your dongle** — on Windows, re-run the Zadig driver
  step (#3); on any system, check the [Hardware guide](hardware.html).
- **No sound when playing a call** — make sure your computer's audio output is
  selected and turned up.
- **Nothing is decoding** — double-check the control-channel frequency and
  protocol you entered for the system in step 5.
