---
slug: services-and-systemd
title: Services & systemd
description: Turn a program into a managed service that starts on boot and restarts on failure, using the systemd unit GopherTrunk ships as the worked example.
keywords: systemd, systemd service, systemd unit, service management, start on boot, restart on failure, systemctl, linux service
level: intermediate
status: full
prereq:
  - build-artifacts-and-versioning
faq:
  - q: What is systemd?
    a: systemd is the init and service manager on most modern Linux systems. It starts services at boot, restarts them if they crash, manages their dependencies and ordering, and collects their logs. You describe a service in a unit file and control it with systemctl commands like start, stop, enable, and status.
  - q: How is running under systemd different from just running a binary?
    a: Running a binary in your terminal stops the moment you log out or the program crashes. A systemd service runs in the background, starts automatically on boot, restarts on failure, logs to the journal, and can be locked down with security settings — everything a long-running production service needs.
---

# Services & systemd

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **service** is a program that runs in the background, starts on boot, and restarts if
it crashes. On Linux, **systemd** manages services: you describe one in a **unit file**
and control it with **`systemctl`**. GopherTrunk ships a real unit that starts the
scanner on boot, restarts it on failure, and locks it down with hardening — the
non-container way to run in production.
</div>

Containers are one way to run in production; a plain binary managed by **systemd** is
the other. This lesson turns a binary into a proper service.

## From a binary to a service

Run `./gophertrunk` in your terminal and it dies when you log out or it crashes. A
production service needs to:

- start automatically **on boot**,
- **restart** itself if it fails,
- run in the **background** independent of any login,
- **log** somewhere you can read later.

On Linux, **systemd** provides all of this. You describe your service once in a **unit
file** and systemd handles the rest. (The [Linux CLI module](/learn/linux-cli/services-and-systemd/)
covers systemd basics; here we focus on deploying with it.)

## GopherTrunk's unit file

Here's the heart of the unit GopherTrunk ships:

```ini
[Unit]
Description=GopherTrunk digital trunking scanner
After=network-online.target
Wants=network-online.target

[Service]
Type=exec
ExecStart=/usr/local/bin/gophertrunk run -config /etc/gophertrunk/config.yaml
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Reading it top to bottom:

- **`[Unit]`** — a description and ordering: start *after* the network is online.
- **`ExecStart`** — the exact command to run, pointing at the installed binary and its
  [config file](/learn/deployment/environments-and-config/).
- **`Restart=on-failure`** with **`RestartSec=5`** — if it crashes, wait 5 seconds and
  restart. This is your crash resilience.
- **`WantedBy=multi-user.target`** — enable it and it starts on every boot.

## Installing and controlling it

Install the unit and binary, then manage the service with `systemctl`:

```bash
sudo install -m 0644 docs/gophertrunk.service /etc/systemd/system/gophertrunk.service
sudo install -m 0755 bin/gophertrunk /usr/local/bin/gophertrunk
sudo systemctl daemon-reload
sudo systemctl enable --now gophertrunk   # start now AND on every boot
sudo systemctl status gophertrunk         # check it's running
```

`enable --now` is the key line: it starts the service immediately *and* wires it to
start on boot.

## Locking it down

A production service should run with the least privilege it needs. GopherTrunk's real
unit adds a raft of **hardening** directives — a taste:

```ini
DynamicUser=true            # run as an unprivileged, ephemeral user
ProtectSystem=strict        # the filesystem is read-only except...
ReadWritePaths=/var/lib/gophertrunk
NoNewPrivileges=true        # can never gain new privileges
ProtectKernelModules=true
RestrictRealtime=true
```

These tell the kernel to run the scanner in a tightly confined sandbox — it can't write
outside its data directory, can't escalate privileges, can't touch kernel internals.
It's the same least-privilege instinct as the container's dropped capabilities, applied
to a bare-metal service, and it connects directly to the
[cybersecurity](/learn/cybersecurity/hardening-systems/) module.

<div class="knowledge-check" data-quiz data-correct-msg="Right — enable --now starts the service now and on every future boot." markdown="0">
  <p class="knowledge-check__q">Quick check: which systemd setting makes a service restart itself after it crashes?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">WantedBy=multi-user.target</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Restart=on-failure</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Description=</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **service** runs in the background, starts on boot, and restarts on failure.
- **systemd** manages services via a **unit file**; **`systemctl`** controls them.
- GopherTrunk's unit sets **`ExecStart`**, **`Restart=on-failure`**, and
  **`WantedBy=multi-user.target`** — plus **hardening** for least privilege.
- **`systemctl enable --now`** starts a service immediately and on every boot.

Next up: how a running service tells you it's healthy — logging and health checks.
