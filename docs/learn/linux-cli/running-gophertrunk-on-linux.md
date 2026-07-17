---
slug: running-gophertrunk-on-linux
title: Running GopherTrunk on Linux
description: The capstone walkthrough — install prerequisites, download or build GopherTrunk, grant your user access to the USB SDR, run it from the shell, and wrap it in a systemd service so it survives reboots on a headless Linux box or Raspberry Pi.
keywords: run gophertrunk linux, raspberry pi sdr, install gophertrunk, systemd service, headless scanner, usb permissions, SDR on linux, gophertrunk raspberry pi, udev rule sdr, journalctl logs
level: intermediate
status: full
prereq:
  - services-and-systemd
  - package-management
faq:
  - q: "How do I run GopherTrunk on a Raspberry Pi?"
    a: "Flash Raspberry Pi OS and bring the Pi up headless over SSH, install any dependencies with **apt**, download or build the GopherTrunk binary, plug in your USB SDR and grant your user access to it, then run GopherTrunk from the shell. Once it works, wrap it in a **systemd service** so it starts at boot and restarts on failure. See the getting-started page for the exact commands."
  - q: "Why can't GopherTrunk see my SDR on Linux?"
    a: "Almost always a **USB permissions** problem. The SDR appears as a device under `/dev`, and a normal user often can't open it until they're added to the right group or a **udev rule** grants access. Add the rule (or the group membership), unplug and replug the dongle, then try again — you should not need to run the scanner as root."
  - q: "How do I keep GopherTrunk running after a reboot?"
    a: "Set it up as a **systemd service** and `enable` it. systemd starts the service automatically at boot, restarts it if it crashes, and captures its logs in the journal — no one has to log in to start it by hand."
gophertrunk_links:
  - title: Downloads
    url: /downloads.html
    note: grab the prebuilt binary for your platform (or the source to build).
  - title: Getting started & setup
    url: /getting-started-setup.html
    note: the exact install, configuration, and first-run steps.
---

# Running GopherTrunk on Linux

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
This is the payoff. Three moves get you there: **install the dependencies**,
**grant your user access to the USB SDR**, and **run GopherTrunk as a
[systemd service](/learn/linux-cli/services-and-systemd/)** so it stays alive
across reboots. Everything the whole path taught you — the shell, permissions,
processes, SSH, and services — now goes to work getting GopherTrunk running on a
Linux box or a Raspberry Pi and keeping it there.
</div>

You've done the groundwork. Now you point it all at one real job: a running
scanner. This lesson is the map, not a line-by-line install — for the exact
commands and flags, we send you to the GopherTrunk pages, because those stay
current and made-up flags don't.

## The plan

The shape of the job is the same on any machine. You **install the
prerequisites**, **get GopherTrunk** onto the box, **plug in the SDR and grant
access** to it, **run it** to confirm it decodes, and finally **make it a
service** so it comes back on its own. You can do every step over
[SSH on a headless Pi](/learn/linux-cli/ssh-and-remote/) — no monitor, no
keyboard, just a terminal on your laptop talking to a board by the antenna.

## Install prerequisites

GopherTrunk needs a few supporting libraries and tools present on the system.
Install them with your distribution's
[package manager](/learn/linux-cli/package-management/) — on Raspberry Pi OS
(and Debian/Ubuntu) that's `apt`:

```bash
sudo apt update
sudo apt install <the packages listed on the setup page>
```

The setup page names exactly what to install for your platform. The point of
this step is simply that supporting software is installed the ordinary Linux
way, not hunted down by hand.

## Get GopherTrunk

Two routes, both familiar by now:

- **Download the binary.** Grab the prebuilt release for your platform from the
  [downloads page](/downloads.html), then make it runnable with
  [`chmod +x`](/learn/linux-cli/permissions/):

  ```bash
  chmod +x ./gophertrunk
  ```

- **Build from source.** Clone the repository and compile it, if you'd rather
  track the latest code or your platform has no prebuilt binary.

Either way, the [getting-started page](/getting-started-setup.html) has the exact
steps — follow those rather than guessing at version numbers or paths.

## Plug in the SDR & USB permissions

Your SDR is a **USB device**. When you plug it in, Linux exposes it as a device
node under `/dev`, and — this trips up almost everyone — a normal user usually
**can't open it yet**. The kernel guards raw USB devices, so out of the box only
root has access.

The clean fix is *not* to run the scanner as root. Instead, grant your own user
access the standard way: add yourself to the group that owns the device, or
install a **udev rule** that hands the dongle to your user when it's plugged in.
This is exactly the [permissions](/learn/linux-cli/permissions/) and
[sudo / root](/learn/linux-cli/sudo-and-root/) thinking from earlier in the path —
give the least privilege that gets the job done. After adding the rule or group,
unplug and replug the SDR so the new access takes effect. The
[hardware page](/hardware.html) lists supported SDRs and any device-specific notes.

## Run it

With the dongle accessible, run GopherTrunk straight from the shell and watch
what it prints:

```bash
./gophertrunk <options from the getting-started page>
```

Read its output the way you'd read any long-running program's — is it finding the
control channel, is it decoding, are there errors scrolling past? This is a
[foreground process](/learn/linux-cli/processes/) for now; `Ctrl-C` stops it.
Watching the live logs is how you'll confirm it's healthy — see
[monitoring & logs](/learn/linux-cli/monitoring-and-logs/). On a Pi you're
typically doing all of this [over SSH](/learn/linux-cli/ssh-and-remote/), so the
output is streaming to your laptop from the board across the room.

## Keep it running — a systemd service

Running it by hand is fine for a first test, but a real scanner should start at
boot and restart itself if it ever dies. That's a job for
[systemd](/learn/linux-cli/services-and-systemd/). Write a small unit file
describing how to launch GopherTrunk — here's the generic shape:

```ini
[Unit]
Description=GopherTrunk scanner
After=network.target

[Service]
ExecStart=/path/to/gophertrunk <your options>
Restart=on-failure
User=youruser

[Install]
WantedBy=multi-user.target
```

Drop that in the system's unit directory, then enable and start it:

```bash
sudo systemctl enable --now gophertrunk.service
```

`enable` wires it to start at boot; `--now` starts it immediately. Check that it
came up cleanly, and read its logs, with:

```bash
systemctl status gophertrunk.service
journalctl -u gophertrunk.service -f
```

Treat the unit above as a sketch — fill in the real binary path and options from
the getting-started page.

## Watch its health

A scanner that runs for months needs the occasional glance, and everything you
need is on the box already. Keep an eye on **disk space** if GopherTrunk is
writing recordings — a full disk stops a lot of things ungracefully. Watch
**CPU** on a small Pi, since decoding is real work and a struggling board shows
up as dropped or garbled audio. And when a decode misbehaves, the
[logs](/learn/linux-cli/monitoring-and-logs/) are the first place to look —
`journalctl` for the service, plus GopherTrunk's own output, usually tell you
whether it's a signal problem or a software one.

## Where to go next

You can now run and maintain a real, always-on Linux service — which is most of
what running a home server or an embedded project comes down to. A **Raspberry Pi
sitting at the base of the antenna** is a classic GopherTrunk setup: small, quiet,
low-power, and reachable over SSH from anywhere in the house. If that's your plan,
the hardware side is worth a read — start with
[Raspberry Pi & family](/learn/intro-hardware/raspberry-pi-and-family/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — a systemd service, enabled to start at boot, brings GopherTrunk back automatically after a reboot and restarts it on failure." markdown="0">
  <p class="knowledge-check__q">Quick check: You want GopherTrunk to come back automatically after a reboot. What do you set up?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A shell script you run by hand each time you log in</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A systemd service, enabled to start at boot</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A udev rule for the USB device</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Install prerequisites** with the package manager (`apt` on Raspberry Pi OS).
- **Get GopherTrunk** — download the binary and `chmod +x` it, or build from
  source; follow the getting-started page for the exact steps.
- **Grant USB access** to your user with a group or udev rule — don't run the
  scanner as root.
- **Run it** from the shell first and watch the logs to confirm it decodes.
- **Make it a systemd service**, `enable` it, and check `journalctl` — now it
  survives reboots and restarts on failure.
- **Watch its health** — disk space, CPU on a small Pi, and the logs.

Next up: keep the [glossary](/learn/linux-cli/glossary/) handy — and if you're setting up a Pi by the antenna, the [Intro to Hardware](/learn/intro-hardware/) path covers the board itself.
