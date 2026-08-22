---
slug: services-with-systemd
title: Services with systemd
description: Turn a program into a systemd service that starts at boot and restarts on failure — the unit file, systemctl enable and start, journalctl logs, and the pattern every appliance is built on.
keywords: systemd, unit file, systemctl enable, restart on failure, journalctl, ExecStart, WantedBy, service at boot, daemon, raspberry pi service
level: intermediate
status: full
prereq:
  - first-boot-and-ssh
  - users-and-updates
faq:
  - q: How do I make a program start automatically when a Raspberry Pi boots?
    a: "Write a small systemd unit file in /etc/systemd/system describing the program (its ExecStart command, the user to run as, and Restart=on-failure), then run sudo systemctl enable --now yourservice. enable registers it to start at every boot; --now also starts it immediately. From then on the program starts itself after every power cut and reboot with no login required."
  - q: What is the difference between systemctl enable and systemctl start?
    a: start runs the service right now, in this boot, and does nothing about the future. enable registers it to start automatically at boot but doesn't launch it this second. An appliance service wants both — enable --now is the shorthand that does the two together.
  - q: Where do a service's logs go?
    a: systemd captures everything the service prints to standard output and error into the journal. Read it with journalctl -u yourservice; add -f to follow live like tail -f, or -e to jump to the end. No log files need to be configured — though on an SBC you should bound the journal's size so it doesn't wear the SD card.
---

# Services with systemd

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The difference between "a program I run" and "an appliance" is a **service**:
something the OS itself starts at boot, supervises, and restarts when it dies. On
Linux that supervisor is **systemd**. You describe the program once in a **unit
file** (its `ExecStart` command, the **user** it runs as, `Restart=on-failure`),
then `systemctl enable --now` it — from that moment it survives every reboot and
power cut, unattended. Its output lands in the **journal**, read with
**`journalctl -u name`**. This one file is the load-bearing pattern of the whole
appliance.
</div>

Everything so far — headless boot, SSH, updates — has been preparing the stage for
this lesson. An appliance's defining behaviour is that its job starts *itself*:
power arrives, and a minute later the work is running, no login, no human. systemd
is how Linux delivers that promise.

## What is a service, and what does systemd do?

A **service** (traditionally a *daemon*) is a long-running background program with
no terminal attached. **systemd** is the init system — the first process the kernel
starts — and the manager of everything after it: it starts services in the right
order at boot, tracks their state, **restarts them when they crash**, and collects
their logs. You talk to it with `systemctl`:

```bash
$ systemctl status ssh          # is it running? since when? recent log lines
$ sudo systemctl restart ssh    # stop + start
$ sudo systemctl enable ssh     # start at every boot
```

The [Linux CLI module](/learn/linux-cli/services-and-systemd/) introduces these
commands broadly; here we go one level deeper and *write* a service, because your
appliance's daemon needs one.

## What's inside a unit file?

A **unit file** is a short INI-style description of how to run a program. Here is
the realistic shape of one for a decoder daemon — the same skeleton Unit 6 will use
for GopherTrunk:

```ini
# /etc/systemd/system/gophertrunk.service
[Unit]
Description=GopherTrunk trunking scanner daemon
After=network-online.target
Wants=network-online.target

[Service]
User=gophertrunk
ExecStart=/usr/local/bin/gophertrunk daemon -config /etc/gophertrunk/config.yaml
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Reading it line by line:

| Line | What it does |
|------|--------------|
| `After=` / `Wants=network-online.target` | Wait for the network — a daemon serving a web console shouldn't race it |
| `User=` | Run as a **dedicated unprivileged user**, not root — smallest possible blast radius |
| `ExecStart=` | The one command to run — absolute paths, no shell tricks |
| `Restart=on-failure` + `RestartSec=5` | The appliance magic: crash → wait 5 s → start again, forever |
| `WantedBy=multi-user.target` | Which boot stage `enable` hooks it into — normal non-graphical boot |

Create the service user (`sudo adduser --system --group gophertrunk`) so `User=`
has someone to be, then install the unit:

```bash
$ sudo systemctl daemon-reload              # re-read unit files
$ sudo systemctl enable --now gophertrunk   # boot-start + start now
$ systemctl status gophertrunk
● gophertrunk.service - GopherTrunk trunking scanner daemon
     Active: active (running) since Tue 2026-08-18 09:14:02 GMT; 2min ago
```

## Where did the program's output go?

Into the **journal** — systemd captures stdout and stderr automatically:

```bash
$ journalctl -u gophertrunk -e     # jump to the newest entries
$ journalctl -u gophertrunk -f     # follow live, like tail -f
$ journalctl -u gophertrunk --since "1 hour ago"
```

This is your window into a headless daemon's mind, and you'll live in it during
Unit 6. One SBC-specific caveat: the journal is disk writes, and unbounded logging
is exactly the load that wears SD cards — [SD-card wear](/learn/embedded/sd-card-wear/)
shows how to cap it (`SystemMaxUse=` in `journald.conf`).

## What does restart-on-failure buy — and not buy?

`Restart=on-failure` converts a crash from an outage into a blip: the daemon dies at
3am, systemd relaunches it five seconds later, and the appliance never noticed you
were asleep. What it does *not* buy: recovery from a **hung** process (running but
doing nothing — watchdogs handle that, in
[Watchdogs &amp; recovery](/learn/embedded/watchdogs-and-recovery/)), or a fix for a
program that crashes *instantly* every time (systemd will notice the crash-loop and
give up; the journal tells you why it's crashing). Restart is the safety net, not a
substitute for reading the logs.

> Rule of thumb: anything your appliance needs to be doing at 3am belongs in a unit
> file. If starting it requires you to log in and type something, it isn't an
> appliance yet.

<div class="knowledge-check" data-quiz data-correct-msg="Right — enable --now registers it for every boot and starts it immediately." markdown="0">
  <p class="knowledge-check__q">Quick check: what does <code>sudo systemctl enable --now gophertrunk</code> do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Starts the service now, but it will not survive a reboot</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Registers the service to start at every boot and also starts it immediately</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Compiles the unit file into the kernel so it can never be stopped</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **service** is a program the OS starts, supervises, and restarts —
  **systemd** is the supervisor, `systemctl` the steering wheel.
- A **unit file** describes the program once: `ExecStart`, a dedicated **User=**,
  `Restart=on-failure`, and network ordering.
- **`systemctl enable --now`** = start at every boot *and* start now — the
  appliance's core guarantee across reboots and power cuts.
- Logs land in the **journal** (`journalctl -u name -f`) — and on an SBC the
  journal's size should be bounded.
- Restart handles **crashes**; hung processes need **watchdogs** (Unit 5), and
  crash-loops need you to read the journal.

Next up: [Networking your board](/learn/embedded/networking-your-board/).
