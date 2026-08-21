---
slug: installing-gophertrunk-on-a-pi
title: Install GopherTrunk on a Pi
description: Get the ARM build of GopherTrunk running on a Raspberry Pi — download the right binary, write a first config, run the daemon in the foreground, open the web console, and hear your first decoded call on SBC hardware.
keywords: gophertrunk raspberry pi, install gophertrunk arm, arm64 binary, sdr scanner raspberry pi, trunking decoder pi, first config, web console
level: intermediate
status: full
prereq:
  - connecting-an-sdr
  - services-with-systemd
faq:
  - q: Does GopherTrunk run on a Raspberry Pi?
    a: Yes — GopherTrunk is a single Go binary and ships ARM Linux builds that run on Raspberry Pi OS and similar distributions. It runs headless, with its web console served over the network, which fits the Pi perfectly. A modern Pi decodes a trunked system comfortably; the following lessons cover fitting the workload to smaller boards.
  - q: Which GopherTrunk download do I need for a Pi?
    a: "The Linux ARM build matching your OS word size: on a 64-bit OS (uname -m reports aarch64) take the arm64 build; on a 32-bit OS (armv7l) take the 32-bit ARM build. Because it's a single static binary there is nothing else to install — copy it into /usr/local/bin, make it executable, and it runs."
  - q: Do I need a monitor on the Pi to use GopherTrunk?
    a: No. GopherTrunk is designed to run headless — you install and control it over SSH, and its interface is a web console you open from any browser on your network. The Pi itself never needs a display, keyboard, or desktop environment.
gophertrunk_links:
  - title: Linux install guide
    url: /install-linux.html
    note: the full install reference this lesson's fast path condenses.
  - title: GopherTrunk SBC build
    url: /gophertrunk-sbc-build/
    note: the site's dedicated single-board-computer build walkthrough.
---

# Install GopherTrunk on a Pi

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Everything converges: GopherTrunk ships as a **single ARM binary** — match it to
`uname -m` (**arm64** for `aarch64`), drop it in `/usr/local/bin`, and it runs
with nothing else to install. First run goes **foreground over SSH** with a
minimal config, so you can watch the daemon find the SDR and lock a control
channel; the interface is the **web console** on your LAN. Once decoding works,
promote it to the **systemd service** from Unit 3 — and the moment
`enable --now` returns, you own a headless scanner that survives reboots.
</div>

Unit 6 begins the build the module promised. The board is flashed, hardened,
networked; the dongle streams clean. This lesson adds the decoder and gets to the
first decoded call — deliberately in the foreground first, service second, so
every layer is proven before it's automated.

## Which binary, and where does it go?

The [Go-language](/learn/programming-go/why-go/) payoff: GopherTrunk is one
static binary — no runtime, no dependency tree. Two facts pick your download from
the [Linux install guide](/install-linux.html): it must be **Linux**, and it must
match your ARM flavour ([ARM and the SoC](/learn/embedded/arm-and-socs/)):

```bash
$ uname -m
aarch64            # → the arm64 Linux build
```

Grab the matching asset from the [GopherTrunk releases
page](https://github.com/MattCheramie/GopherTrunk/releases) and unpack it on the
Pi (substitute the current version number):

```bash
$ curl -LO https://github.com/MattCheramie/GopherTrunk/releases/download/<version>/gophertrunk-<version>-linux-arm64.tar.gz
$ tar xzf gophertrunk-<version>-linux-arm64.tar.gz
$ sudo install -m 0755 gophertrunk /usr/local/bin/
$ gophertrunk version
```

`gophertrunk version` printing a version string proves architecture and binary in
one stroke — `exec format error` here means the wrong architecture was
downloaded.

## What does a first config look like?

Start minimal: one system, its control channel, your dongle. Create
`/etc/gophertrunk/config.yaml` with the shape (values from your local system —
found via the [Scanning module's](/learn/scanning/programming-a-trunked-system/)
database-and-discovery methods):

```yaml
sdr:
  driver: rtlsdr
systems:
  - name: county-p25
    protocol: p25
    control_channels: [853.9625e6]
recordings:
  dir: /var/lib/gophertrunk/recordings
```

Config vocabulary — protocols, talkgroups, recording options — is the
[install guide](/install-linux.html)'s territory; the embedded-lesson point is
narrower: **start with the smallest config that can possibly decode**, so the
first run has the fewest moving parts.

## Why run in the foreground first?

Resist the urge to go straight to the service. In your SSH session (inside tmux,
[as always](/learn/embedded/remote-administration/)):

```bash
$ gophertrunk daemon -config /etc/gophertrunk/config.yaml
```

The log streams to your terminal — and narrates the whole chain you've built:
the SDR opening (Unit 4's driver work), tuning, control-channel hunting, then
the lock, and decoded grants scrolling by as calls happen. Now open the **web
console** from any browser in the house — `http://scanner.local:8080` (or the
pinned IP from [Networking your board](/learn/embedded/networking-your-board/))
— and watch call activity live. When a recording lands, listen to it in the
console: *that is the first decoded call on your own SBC hardware.*

Foreground-first pays off precisely when something's wrong: no control-channel
lock (antenna, frequency list, signal — the
[RF-side checklist](/learn/rf-sdr/finding-systems/)), SDR won't open (back to
[Connecting an SDR](/learn/embedded/connecting-an-sdr/) — is `rtl_test` still
clean?), or CPU pinned (next lesson's tuning). Each failure is visible at the
moment it happens, in a log you're already watching.

## How does it become an appliance?

Promote the proven foreground command into the Unit 3 pattern — this is exactly
the unit file [Services with systemd](/learn/embedded/services-with-systemd/)
wrote, `ExecStart` now pointing at a decoder you've watched work:

```bash
$ sudo adduser --system --group gophertrunk
$ sudo mkdir -p /var/lib/gophertrunk/recordings
$ sudo chown -R gophertrunk:gophertrunk /var/lib/gophertrunk /etc/gophertrunk
$ sudo systemctl enable --now gophertrunk
$ journalctl -u gophertrunk -f        # same narration, now in the journal
```

Then the graduation exam — the **power-cycle test** from
[Watchdogs &amp; recovery](/learn/embedded/watchdogs-and-recovery/): `sudo
reboot`, wait two minutes, open the console. Lock reacquired, calls flowing,
nobody logged in? Then you have an appliance, not a program. Add the daemon's
vitals to your [monitoring script](/learn/embedded/monitoring-your-board/)
(service active + a recording landed recently) and Unit 5's machinery is
guarding the new workload.

> Rule of thumb: foreground until it decodes, service once it's proven, reboot
> test before you call it done. Never automate a thing you haven't watched work.

<div class="knowledge-check" data-quiz data-correct-msg="Right — foreground-first makes every failure visible live, and only proven commands get automated." markdown="0">
  <p class="knowledge-check__q">Quick check: why run the daemon in the foreground before enabling the systemd service?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The service refuses to start until the program has run once manually</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Foreground mode decodes with higher audio quality</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">You watch the SDR open and the control channel lock live, so problems surface before anything is automated</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- GopherTrunk is a **single static ARM binary**: match `uname -m` (**arm64** for
  `aarch64`), install to `/usr/local/bin`, verify with `gophertrunk version`.
- Start from the **smallest config that can decode** — one system, one control
  channel, your dongle.
- **Foreground first** over SSH: watch the SDR open and the control channel
  lock; the **web console** on the LAN is the interface — the Pi stays headless.
- Promote to the **systemd service** (dedicated user, `enable --now`) only once
  decoding is proven, then pass the **reboot test**.
- Wire the new workload into Unit 5's **monitoring** — the appliance now guards
  itself.

Next up: [Tuning for small CPUs](/learn/embedded/tuning-for-small-cpus/).
