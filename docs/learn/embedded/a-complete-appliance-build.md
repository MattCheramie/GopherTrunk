---
slug: a-complete-appliance-build
title: A complete appliance build
description: The full GopherTrunk scanner appliance recipe end to end — hardware list, OS and hardening, dongle proof, daemon service, tuning, monitoring, and backups — with the acceptance tests that prove each layer.
keywords: raspberry pi scanner build, complete sdr appliance, gophertrunk pi recipe, build checklist, 24/7 scanner, headless scanner appliance, acceptance test
level: advanced
status: full
prereq:
  - installing-gophertrunk-on-a-pi
  - tuning-for-small-cpus
  - appliance-networking
gophertrunk_links:
  - title: GopherTrunk SBC build
    url: /gophertrunk-sbc-build/
    note: the site's companion walkthrough of this build, kept current.
  - title: Raspberry Pi SDR scanner
    url: /raspberry-pi-sdr-scanner/
    note: the classic project page this module's path leads to.
---

# A complete appliance build

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
This is the whole module as **one recipe**: a hardware list with reasons, then
seven build phases — **flash → harden → prove the dongle → daemon foreground →
service → tune → monitor &amp; back up** — each ending in an **acceptance test**
that must pass before the next phase starts. The finished state: a Pi in a
closet decoding trunked radio **24/7**, console on every device in the house,
invisible to the internet, self-healing, monitored, and restorable in ten
minutes. Build it in order; the order *is* the method.
</div>

Thirty lessons of ideas become one afternoon of build. Nothing here is new —
that's the point: every step links back to the lesson that taught it, and each
phase ends with a test that proves the layer before you stack the next. Follow
it as a checklist the first time; after that, it's simply how you build
appliances.

## What goes in the box?

The shopping list, each line carrying its unit's reasoning:

| Part | Spec | Why (lesson) |
|------|------|--------------|
| Board | Current full-size Pi, 4 GB+ | CPU + USB headroom ([Picking a board](/learn/embedded/picking-a-board/)) |
| Storage | High-endurance SD 64 GB+, or SSD | Continuous writes ([Storage](/learn/embedded/storage-and-sd-cards/)) |
| Power | Official/name-brand supply | Undervoltage is the classic gremlin ([Power](/learn/embedded/power-supplies/)) |
| Case | Vented or thermal metal, mountable | Sustained load, closet ambient ([Cooling](/learn/embedded/cases-and-cooling/)) |
| SDR | Name-brand RTL-SDR (TCXO) | The radio ([Connecting an SDR](/learn/embedded/connecting-an-sdr/), [hardware guide](/hardware.html)) |
| Network | Ethernet cable to the router | Reliability + RF silence ([Networking](/learn/embedded/networking-your-board/)) |
| Antenna | Suited to your band, metres of coax | Distance from the board's hash ([Gotchas](/learn/embedded/usb-sdr-gotchas/), [antennas](/learn/rf-sdr/antennas/)) |

## Phases 1–2: a hardened headless base

**Flash** ([lesson](/learn/embedded/flashing-an-os-image/)): Raspberry Pi OS
Lite 64-bit via Imager — hostname `scanner`, your user, SSH on. Boot on
Ethernet. **Find and enter** ([lesson](/learn/embedded/first-boot-and-ssh/)):
`ssh matt@scanner.local`, health tour, note the IP.
**Harden** ([lesson](/learn/embedded/users-and-updates/)): full upgrade,
unattended-upgrades on. Pin a **DHCP reservation**
([lesson](/learn/embedded/networking-your-board/)). Push **SSH keys**, disable
password login — new-door-before-old-door
([Remote administration](/learn/embedded/remote-administration/)). Arm the
**hardware watchdog** (`RuntimeWatchdogSec=15` —
[Watchdogs](/learn/embedded/watchdogs-and-recovery/)) and **cap the journal**
(`SystemMaxUse=64M` — [SD-card wear](/learn/embedded/sd-card-wear/)).

> **Acceptance test:** reboot; the board returns on the same address, key-only
> SSH, `vcgencmd get_throttled` reads `0x0`.

## Phase 3: a proven radio

Blacklist the DVB module, `apt install rtl-sdr`, dongle straight into the board
([Connecting an SDR](/learn/embedded/connecting-an-sdr/)). Antenna on its coax,
**metres from the board** ([Gotchas](/learn/embedded/usb-sdr-gotchas/)).

> **Acceptance test:** two minutes of `rtl_test` with zero lost-sample lines,
> and `get_throttled` still clean under the dongle's load.

## Phases 4–5: the daemon, foreground then service

Install the **arm64 binary**, write the minimal one-system config, and run
**foreground in tmux** until the control channel locks and a call records
([Install GopherTrunk on a Pi](/learn/embedded/installing-gophertrunk-on-a-pi/)).
Open `http://scanner.local:8080` from the sofa. Then promote: dedicated user,
the [unit file](/learn/embedded/services-with-systemd/) with
`Restart=on-failure`, `enable --now`, retention policy set in config.

> **Acceptance test:** the **power-cycle test** — pull the plug, wait two
> minutes, console up, lock reacquired, calls flowing, nobody logged in.

## Phase 6: tuned to its budget

During the busy hour: `htop`, load average, journal grep for *can't keep up*
([Tuning for small CPUs](/learn/embedded/tuning-for-small-cpus/)). Trim
captured bandwidth and concurrency until sustained CPU sits **≤50%** with
throttling flags clean.

> **Acceptance test:** a full busy hour with no overrun warnings, no dropped
> samples, temperature equilibrium under ~70 °C.

## Phase 7: watched and recoverable

The [health-check script](/learn/embedded/monitoring-your-board/) on a timer —
vitals logged, thresholds alerting, weekly heartbeat. **Golden image** of the
finished card, config/data **rsync on schedule** to another machine, and the
**restore rehearsed once** on a spare card
([Backups &amp; images](/learn/embedded/backups-and-images/)). Access stays
LAN-only; remote listening via VPN or tunnel
([Appliance networking](/learn/embedded/appliance-networking/)).

> **Acceptance test:** kill the daemon (`sudo systemctl kill gophertrunk`) —
> it self-restarts and the alert arrives. Boot the spare card — the clone
> decodes. Now, and only now, put it in the closet.

## Where do you go from here?

You've built the thing this module exists to teach — and picked up the whole
embedded discipline on the way: headless Linux, services, buses, reliability
engineering, and a working scanner as proof. Three roads onward:

- **Become a better operator** of what you built: the
  [Scanning &amp; Monitoring module](/learn/scanning/) — finding systems,
  scan lists, feeds, and the craft of listening.
- **Understand the signal path** beneath the daemon: the
  [RF &amp; SDR module](/learn/rf-sdr/) from antenna to decoded bits.
- **Scale the ops skills up**: the [Deployment module](/learn/deployment/) —
  containers, CI, and fleet-grade monitoring; your appliance habits transfer
  directly.

And the build itself keeps evolving — the
[SBC build guide](/gophertrunk-sbc-build/) tracks hardware and config as boards
change. Keep the [glossary](/learn/embedded/glossary/) handy as a reference.

<div class="knowledge-check" data-quiz data-correct-msg="Right — each phase's acceptance test proves its layer, so failures are always localized to the newest layer." markdown="0">
  <p class="knowledge-check__q">Quick check: why does the recipe demand an acceptance test at the end of every phase?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Because the software refuses to install until tests are recorded</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Each test proves its layer works, so any later failure is localized to the newest layer instead of the whole stack</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">To make the build take longer and feel more professional</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The build is **seven phases** — flash, harden, prove the dongle, foreground,
  service, tune, monitor &amp; back up — each gated by an **acceptance test**.
- The hardware list is the module's units made shoppable: headroom, endurance,
  clean power, cooling, antenna distance.
- The two decisive gates: **`rtl_test` clean** before any decoder, and the
  **power-cycle test** before calling it an appliance.
- Finished state: **24/7 decoding, LAN-only console, self-healing, monitored,
  ten-minute restore** — an embedded system in the full sense of lesson 1.
- The path continues in [Scanning &amp; Monitoring](/learn/scanning/),
  [RF &amp; SDR](/learn/rf-sdr/), and [Deployment](/learn/deployment/).

Next up: you've completed the module — take the operator's road with
[Scanning &amp; Monitoring](/learn/scanning/), or level up your ops with
[Deployment](/learn/deployment/).
