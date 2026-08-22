---
slug: users-and-updates
title: Users, permissions & updates
description: Basic security hygiene for an always-on board — no default credentials, your own user with sudo, apt updates on a schedule, and the minimum-exposure mindset for a device that lives on your network.
keywords: raspberry pi security, change default password, sudo user, apt update upgrade, unattended-upgrades, linux permissions, patching, network hygiene
level: beginner
status: full
prereq:
  - first-boot-and-ssh
---

# Users, permissions & updates

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An always-on, network-attached box deserves server hygiene from day one: **no
default credentials** anywhere, a **personal user** that borrows root through
**sudo** rather than logging in as root, and **regular updates** via `apt` — with
security patches automated by **unattended-upgrades**, because an appliance's
updates must not depend on you remembering. The mindset is **minimum exposure**:
run only the services you need, listening only where you need them, on your LAN
only. Boring, and the entire difference between an appliance and a liability.
</div>

Your board is on the network for good now — which makes it, in the eyes of every
automated scanner on the internet and every misbehaving device on your LAN, a
server. This lesson is the short list of habits that keep a 24/7 box trustworthy.

## Why does a hobby box need security hygiene?

Because attackers don't check what a device is *for*. Botnets scan relentlessly for
anything answering SSH with factory credentials — embedded devices are their
favourite prey precisely because nobody watches them. A compromised scanner Pi isn't
a scanner problem: it's a machine *inside your home network*, running whatever
someone else likes. The defence is not sophistication; it's the absence of the three
classic sins — default passwords, needless exposure, and stale software.

## Who should be able to log in?

Modern images have you create a user while flashing — the pattern to keep:

- **One personal user** (yours), with a strong password you don't reuse.
- **Root never logs in directly.** The root account stays disabled for login;
  administrative power comes via **sudo**, which grants your user root *per command*
  and logs each use:

```bash
$ sudo systemctl restart gophertrunk   # elevated, logged
$ whoami                               # still just you
matt
```

This is standard Linux practice — [sudo &amp; root](/learn/linux-cli/sudo-and-root/)
and [users &amp; groups](/learn/linux-cli/users-and-groups/) cover the mechanics. On
an appliance it earns its keep the day something goes wrong: an audit trail of who
ran what, and no all-powerful account with a guessable name for bots to hammer. In
[Remote administration](/learn/embedded/remote-administration/) you'll go further —
SSH keys instead of passwords, and password login switched off entirely.

Permissions follow the same logic downstream: the GopherTrunk daemon will run as its
**own unprivileged user** (Unit 3's [systemd lesson](/learn/embedded/services-with-systemd/)
sets this up), so even a compromised or crashing service can touch only its own files.

## How do updates work, and how often?

Debian-family systems update from **package repositories** with `apt`:

```bash
$ sudo apt update          # refresh the list of available versions
$ sudo apt full-upgrade    # install them
$ sudo apt autoremove      # sweep up unneeded leftovers
```

Run this on a schedule you'll actually keep — monthly is a fine floor for a home
appliance, plus whenever you're logged in anyway. Kernel and firmware updates want a
reboot to take effect; an appliance whose services start at boot
([next lesson](/learn/embedded/services-with-systemd/)) makes rebooting a non-event.

For the updates that matter most, remove yourself from the loop.
**unattended-upgrades** installs *security* patches automatically:

```bash
$ sudo apt install unattended-upgrades
$ sudo dpkg-reconfigure -plow unattended-upgrades   # answer Yes
```

The default configuration applies security updates only — the low-risk, high-value
slice — leaving bigger version jumps for your attended sessions. For a box designed
to be forgotten, automated security patching isn't optional polish; it's the
patch-day you'd otherwise skip. ([Package management](/learn/linux-cli/package-management/)
covers apt in depth.)

> Rule of thumb: an appliance should stay reasonably patched *even if you ignore it
> for three months*. If your update plan depends on your memory, it isn't a plan.

## What does "minimum exposure" mean in practice?

Every service listening on a port is surface area. The Lite image starts almost
nothing — keep it that way:

- **Install only what the job needs.** Every extra package is patch burden and
  potential exposure.
- **Know what's listening.** `ss -tlnp` lists listening TCP ports; on a fresh
  appliance expect SSH (22) and, later, GopherTrunk's web console. Anything you
  can't explain, investigate.
- **LAN only.** Nothing on this box should be reachable from the internet — no
  router port-forwards "just to check it from work." Unit 6's
  [Appliance networking &amp; access](/learn/embedded/appliance-networking/) shows
  the safe patterns (and the [Networking module](/learn/networking/firewalls/)
  covers firewalls when you want them).

```bash
$ ss -tlnp
State   Local Address:Port   Process
LISTEN  0.0.0.0:22           sshd
```

<div class="knowledge-check" data-quiz data-correct-msg="Right — security updates must land without you; unattended-upgrades does exactly that." markdown="0">
  <p class="knowledge-check__q">Quick check: why set up unattended-upgrades on an appliance instead of relying on manual updates?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Manual apt commands are disabled on headless systems</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It upgrades the board to new OS releases automatically</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Security patches keep arriving even during the months you forget the box exists</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A networked always-on box is a **server**; bots assume so even if you don't. No
  **default credentials**, ever.
- Use a **personal user + sudo**; root never logs in, services get their **own
  unprivileged users**.
- Update with **`apt update && apt full-upgrade`** on a schedule, and automate
  security patches with **unattended-upgrades** — the appliance must self-patch.
- Practice **minimum exposure**: minimal packages, know your listening ports
  (`ss -tlnp`), and keep everything **LAN-only**.
- These habits cost minutes once — and are the whole difference between an
  appliance and a liability.

Next up: [Services with systemd](/learn/embedded/services-with-systemd/).
