---
slug: services-and-systemd
title: Services & systemd
description: What a Linux service (daemon) is and how systemd runs one — starting it at boot, restarting it on failure, and collecting its logs — plus the everyday systemctl commands, reading logs with journalctl, and the anatomy of a simple .service unit file.
keywords: systemd, service, daemon, systemctl, journalctl, unit file, enable at boot, background service, restart on failure, init system, ExecStart, WantedBy
level: advanced
status: full
prereq:
  - processes
faq:
  - q: What is systemd?
    a: "systemd is the init system on most modern Linux distributions — the first process the kernel starts (PID 1) and the manager that brings up and supervises everything else. It runs background programs as services: starting them at boot, restarting them if they crash, tracking their state, and collecting their logs. You control it with the systemctl command."
  - q: How do I start a service at boot?
    a: "Run systemctl enable <service> to make it start automatically on every boot, and systemctl start <service> to start it right now. These are separate: enable only sets the boot-time behaviour and start only affects the current session. Use systemctl enable --now <service> to do both at once, and systemctl disable <service> to stop it starting at boot."
  - q: How do I see a service's logs?
    a: "Use journalctl -u <service> to read everything that service has written to the system journal. Add -f to follow new lines live as they arrive (like tail -f), or -e to jump straight to the most recent entries. systemd captures a service's standard output and error automatically, so you don't need to set up log files yourself."
  - q: What's the difference between systemctl start and enable?
    a: "start launches the service now, in the current boot, and does nothing about the future — reboot and it's gone unless something else starts it. enable registers the service to start automatically at boot but does not start it this moment. You usually want both: enable so it survives reboots, start (or enable --now) so it's running immediately."
---

# Services & systemd

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **service** (or **daemon**) is a background program with no terminal, kept
alive by a service manager. On modern Linux that manager is **systemd**, which
you drive with **systemctl** (`status`, `start`, `stop`, `restart`, and
crucially `enable` = **start at boot**). Its logs live in the journal, read with
**journalctl**. systemd is how modern Linux runs background programs reliably —
starting them at boot, restarting them on failure, and collecting their logs.
New to background programs?
[Start with processes & jobs](/learn/linux-cli/processes/).
</div>

You've met the [process](/learn/linux-cli/processes/): a running program you can
start, watch, and stop by hand. That's fine for a quick task, but anything you
want running *permanently* — a web server, a database, a decoder like
**GopherTrunk** — needs more. It should come back after a reboot, restart itself
if it crashes, and keep its logs somewhere you can find them. That's the job of a
**service manager**.

## What a service is

A **service** is a background program — a **daemon** — that runs without a
terminal attached. Nobody is sitting there watching its output; it just runs,
quietly, for as long as the machine is up. The web servers, print spoolers, and
schedulers that keep a system working are all daemons.

The difference from a [bare process](/learn/linux-cli/processes/) you started by
hand is *supervision*. When you launch something with `&` and `nohup`, nothing is
looking after it: if it crashes, it stays dead; if the machine reboots, it never
comes back. A service, by contrast, is managed — something is responsible for
keeping it alive, and that something is the service manager.

## systemd & systemctl

On most modern distributions the service manager is **systemd**. It's the
**init system** — the very first process the kernel starts (PID 1) — and the
parent of everything else on the machine. When the system boots, systemd is what
brings the services up in order; while it runs, it watches them.

You talk to systemd through one command: **`systemctl`**. The everyday verbs take
a service name:

```
systemctl status  gophertrunk    # is it running? recent log lines
systemctl start   gophertrunk    # start it now
systemctl stop    gophertrunk    # stop it now
systemctl restart gophertrunk    # stop then start (after a config change)
```

The most important pair, though, is **`enable`** versus **`start`**, because they
sound alike but do different things:

- **`systemctl start <name>`** — run it **now**, in the current boot. Reboot and
  it's gone unless something starts it again.
- **`systemctl enable <name>`** — register it to **start automatically at boot**.
  This does *not* start it this moment.

You almost always want both. Run `systemctl enable --now <name>` to enable *and*
start in one go, and `systemctl disable <name>` to stop it launching at boot.

## Reading logs — journalctl

A daemon has no terminal, so where does its output go? systemd captures each
service's standard output and error into the **journal**, a central log you read
with **`journalctl`**:

```
journalctl -u gophertrunk        # everything this service has logged
journalctl -u gophertrunk -f     # follow new lines live, like tail -f
journalctl -u gophertrunk -e     # jump to the most recent entries
```

`-u` picks one **unit** (service), `-f` **follows** the log as it grows, and `-e`
jumps to the **end**. Because systemd collects this for you, there are no log
files to set up or rotate by hand. We go deeper on watching a running system in
[monitoring, disk & logs](/learn/linux-cli/monitoring-and-logs/).

## A simple unit file

systemd learns about a service from a **unit file**: a small text file, ending in
`.service`, that describes how to run the program. Here's a complete one for a
made-up tool of your own:

```ini
[Unit]
Description=My background tool
After=network.target

[Service]
ExecStart=/usr/local/bin/mytool --config /etc/mytool.conf
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

Three sections, each doing one job:

- **`[Unit]`** — metadata and ordering. `Description` names it; `After` says what
  should come up first (here, the network).
- **`[Service]`** — how to run it. **`ExecStart`** is the exact command to launch,
  and **`Restart=on-failure`** tells systemd to relaunch it if it exits with an
  error — the supervision a bare process never gets.
- **`[Install]`** — what `enable` hooks into. **`WantedBy=multi-user.target`**
  means "start this once the system reaches normal multi-user operation", i.e. at
  boot.

Drop the file in `/etc/systemd/system/mytool.service`, then run **`systemctl
daemon-reload`** so systemd reads your changes — you must do this after creating
or editing a unit file. From there, `systemctl enable --now mytool` and it's a
live, supervised service.

## Why bother

Everything a long-running tool needs, systemd hands you for free:

- it **restarts on crash**, so a transient failure doesn't take the service down
  for good;
- it **starts on boot**, so a power blip or reboot doesn't need you at the
  keyboard;
- it keeps **centralized logs** in the journal, one command away;
- and it gives you **clean start / stop / restart** control instead of hunting
  PIDs.

That's exactly the shape of running **GopherTrunk** unattended — a decoder that
should sit by the antenna for days, survive reboots, and keep its logs. We put
all of this together in
[running GopherTrunk on Linux](/learn/linux-cli/running-gophertrunk-on-linux/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — enable registers it to start at every boot; start only affects the current session." markdown="0">
  <p class="knowledge-check__q">Quick check: which command makes a service start automatically at every boot?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">systemctl start &lt;service&gt;</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">systemctl enable &lt;service&gt;</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">journalctl -u &lt;service&gt;</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **service** (**daemon**) is a background program with no terminal, kept alive
  by a service manager — unlike a bare process you started by hand.
- **systemd** is the **init system** (PID 1) on most modern distros; you drive it
  with **`systemctl`**: `status`, `start`, `stop`, `restart`.
- **`enable`** = start at boot; **`start`** = start now. Use `enable --now` for
  both.
- **`journalctl -u <service>`** reads a service's logs; `-f` follows live, `-e`
  jumps to the end.
- A **`.service`** unit file has `[Unit]`, `[Service]`, and `[Install]` sections;
  `ExecStart`, `Restart=on-failure`, and `WantedBy=multi-user.target` are the key
  lines. Run `systemctl daemon-reload` after editing.
- Restarts, boot-time start, central logs, and clean control are why you run a
  long-lived tool as a service.

Next up: [monitoring, disk & logs](/learn/linux-cli/monitoring-and-logs/)
