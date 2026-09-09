---
title: "The Operator's Cookbook, Part 11: The Closet Appliance — Pi, systemd & Docker"
description: Turn a GopherTrunk rig into a headless 24/7 appliance — a Raspberry Pi or mini-PC under a hardened systemd unit, Docker with USB pass-through, honest LAN auth posture, sdr doctor preflight, runtime heartbeats and USB watchdogs, and the graceful shutdown that finally takes milliseconds.
category: tutorials
keywords: raspberry pi sdr scanner 24/7, gophertrunk systemd service, docker rtl-sdr usb passthrough, headless police scanner setup, sdr daemon hardening linux, rtl-sdr udev rules, scanner appliance raspberry pi, sdr watchdog usb reset, gophertrunk cookbook
tags: [operator-cookbook, raspberry-pi, systemd, docker, headless, operations]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook"
series_part: 11
---

*Part 11 of **The Operator's Cookbook**, a 14-part series of complete,
copy-paste GopherTrunk builds — one working rig per part, antenna to browser.
[Part 10]({{ '/blog/tutorials/operator-cookbook-10-archival-rig/' | relative_url }})
made your recordings folder an archive worth keeping; this part gives it a
home that never sleeps. The closet appliance is the rig most operators end
up with: a small board in a closet, no monitor, started by systemd at boot,
still decoding months later. The single Go binary makes this easy — the
discipline is everything *around* the binary: supervision, permissions,
watchdogs, and knowing healthy from a log line, because a log line is all
you'll see.*

> **TL;DR:** One static Go binary + the example unit at
> [`docs/gophertrunk.service`](https://github.com/MattCheramie/GopherTrunk/blob/main/docs/gophertrunk.service)
> makes a hardened appliance: `DynamicUser=true`, `ProtectSystem=strict`,
> `DeviceAllow=char-usb_device rwm` for the pure-Go USB driver,
> `Restart=on-failure`, logs in `journalctl -u gophertrunk -f`. Preflight
> with `gophertrunk sdr doctor`, watch health via `runtime: heartbeat` lines
> and the USB watchdog (`sdr.watchdog_interval_ms`) that re-acquires
> vanished dongles by serial. Docker works too — the repo's
> `docker-compose.yml` maps the dongle in via `devices:`. And
> `systemctl restart` is now milliseconds, not a silent 30 s stall — that
> teardown bug is fixed and warns if it ever recurs.

**Key takeaways**

- **An appliance is a binary plus supervision.** GopherTrunk is one static
  executable — no libusb, no librtlsdr, no ALSA — so the "install" is
  copying a file, and everything hard lives in the systemd unit.
- **Hardening is shipped, not homework.** The example unit runs as a dynamic
  user with a read-only filesystem view, explicit `ReadWritePaths`, and a
  scoped USB `DeviceAllow` — you edit paths and serials, not policy.
- **Headless health is three signals.** `runtime: heartbeat` proves the
  process alive and bounded, the SDR watchdog narrates dongles leaving and
  rejoining the bus, and the decode lines prove the radio side.
- **Shutdown speed is a correctness feature.** A stop that takes 30 silent
  seconds gets blamed on ghosts. The streaming handlers now exit on signal;
  a blown window logs a WARN naming itself.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Which board | Pi 4/5 class SBC or mini-PC sizing | [SBC build list]({{ '/gophertrunk-sbc-build/' | relative_url }}), [best SBC guide]({{ '/best-single-board-computer-for-gophertrunk/' | relative_url }}) |
| Install & udev | binary, config, USB permissions | [Linux install]({{ '/install-linux.html' | relative_url }}), [hardening guide]({{ '/hardening.html' | relative_url }}) |
| Supervision | hardened unit, restart-on-failure | `docs/gophertrunk.service`, [systemd deep dive]({{ '/blog/deep-dives/running-it-for-real-13-systemd-windows/' | relative_url }}) |
| Containers | USB pass-through via `devices:` | repo `docker-compose.yml`, [Docker deep dive]({{ '/blog/deep-dives/running-it-for-real-12-docker-usb/' | relative_url }}) |
| Preflight | who owns the dongle before the daemon runs | `gophertrunk sdr doctor`, [deep dive]({{ '/blog/deep-dives/running-it-for-real-07-sdr-doctor-preflight/' | relative_url }}) |
| Health & watchdogs | heartbeat line, USB re-acquire, memory cap | `diagnostics.heartbeat_seconds`, `sdr.watchdog_interval_ms`, `diagnostics.memory_limit_mb` |
| Auth on the LAN | token-gated mutations off-loopback | `api.auth`, [auth posture]({{ '/blog/deep-dives/running-it-for-real-02-auth-posture/' | relative_url }}) |

## In this post

- **What you're building** — a supervised, hardened, self-reporting box.
- **The shopping list** — the board, and the power supply not to cheap out on.
- **The config** — the appliance-specific keys, atop any earlier recipe.
- **The systemd unit** — install commands and what the hardening lines do.
- **The Docker variation** — USB pass-through that actually works.
- **First run & when it doesn't work** — headless health, undervoltage, USB, thermal.

## What you're building

Take any rig from Parts 1–10 and remove the human. The appliance boots,
starts GopherTrunk under systemd, restarts itself on failure, and answers
two remote surfaces: the web console, and `journalctl` over SSH. The
[laptop-to-service deep dive]({{ '/blog/deep-dives/running-it-for-real-01-laptop-to-service/' | relative_url }})
tells the *why* of each piece;
[staying up]({{ '/blog/deep-dives/running-it-for-real-14-staying-up/' | relative_url }})
covers the long-run failure modes. This recipe is the assembly.

<figure class="lab-figure">
<svg viewBox="0 0 680 240" width="680" height="240" role="img" aria-label="Block diagram of the closet appliance: a Raspberry Pi runs systemd supervising the gophertrunk daemon, which owns an RTL-SDR over USB guarded by a re-acquiring watchdog, emits heartbeat and decode lines into the journal, and serves the token-gated web console over the LAN">
  <rect x="10" y="16" width="400" height="208" rx="8" fill="none" stroke="var(--fg-muted)" stroke-dasharray="5 4"/>
  <text x="210" y="34" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the closet — Raspberry Pi / mini-PC, headless</text>
  <rect x="28" y="52" width="120" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="88" y="69" text-anchor="middle" fill="currentColor" font-size="10">systemd</text>
  <text x="88" y="83" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Restart=on-failure</text>
  <line x1="88" y1="92" x2="88" y2="120" stroke="currentColor"/>
  <polygon points="84,114 88,122 92,114" fill="currentColor"/>
  <rect x="28" y="122" width="120" height="44" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="88" y="140" text-anchor="middle" fill="var(--accent)" font-size="10">gophertrunk run</text>
  <text x="88" y="155" text-anchor="middle" fill="var(--fg-muted)" font-size="9">DynamicUser, strict FS</text>
  <rect x="196" y="122" width="96" height="44" rx="4" fill="none" stroke="currentColor"/>
  <text x="244" y="140" text-anchor="middle" fill="currentColor" font-size="10">RTL-SDR</text>
  <text x="244" y="155" text-anchor="middle" fill="var(--fg-muted)" font-size="9">/dev/bus/usb</text>
  <line x1="148" y1="144" x2="196" y2="144" stroke="currentColor"/>
  <text x="172" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="9">USB</text>
  <path d="M 196 170 C 172 190 156 178 150 160" fill="none" stroke="var(--fg-muted)"/>
  <text x="216" y="186" fill="var(--fg-muted)" font-size="9">watchdog: re-acquire by serial</text>
  <rect x="196" y="52" width="196" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="294" y="69" text-anchor="middle" fill="currentColor" font-size="10">journald</text>
  <text x="294" y="83" text-anchor="middle" fill="var(--fg-muted)" font-size="9">heartbeat · watchdog · decode lines</text>
  <line x1="118" y1="122" x2="216" y2="92" stroke="var(--fg-muted)"/>
  <rect x="470" y="52" width="196" height="52" rx="4" fill="none" stroke="currentColor"/>
  <text x="568" y="72" text-anchor="middle" fill="currentColor" font-size="10">laptop: journalctl -f over SSH</text>
  <text x="568" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="9">the only console this box has</text>
  <rect x="470" y="132" width="196" height="52" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="568" y="152" text-anchor="middle" fill="var(--accent)" font-size="10">web console :8080 on the LAN</text>
  <text x="568" y="170" text-anchor="middle" fill="var(--fg-muted)" font-size="9">mutations gated by bearer token</text>
  <line x1="410" y1="76" x2="470" y2="76" stroke="currentColor" stroke-dasharray="3 3"/>
  <line x1="410" y1="158" x2="470" y2="158" stroke="var(--accent)"/>
  <text x="340" y="234" text-anchor="middle" fill="var(--fg-muted)" font-size="10">no monitor, no keyboard: supervision, permissions and log lines do the operator's job</text>
</svg>
<figcaption>The appliance in one picture: systemd supervises, the watchdog guards the USB port, and everything you'd normally see on a screen arrives as journal lines and web panels.</figcaption>
</figure>

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| [Raspberry Pi]({{ '/reference/raspberry-pi/' | relative_url }}) 4/5 (or any x86 mini-PC) | ~$60–100 | a Pi 4 handles a single-system starter rig; multi-system and TETRA builds want a Pi 5 or mini-PC — see the [SBC guide]({{ '/best-single-board-computer-for-gophertrunk/' | relative_url }}) |
| Official-grade power supply | ~$10 | the most skipped, most load-bearing item: undervoltage is the classic "random USB weirdness" cause |
| Storage | varies | boot SD plus the Part 10 external disk if you archive |
| Case with airflow | ~$10 | closets are warm; passive fins or a quiet fan |

Dongles and antenna carry over. The
[SBC build list]({{ '/gophertrunk-sbc-build/' | relative_url }}) and the
[Pi scanner guide]({{ '/raspberry-pi-sdr-scanner/' | relative_url }}) cover
specific models.

## The config

Whatever system config you run, the appliance adds one theme: the box is on
the LAN and nobody is watching it. Keys verified against
`config.example.yaml`:

```yaml
log:
  level: info

api:
  http_addr: "0.0.0.0:8080"        # reachable from the LAN, not just loopback
  auth:
    mode: "auto"                   # require a token off-loopback
    token_file: "/etc/gophertrunk/api-token"

diagnostics:
  heartbeat_seconds: 60            # periodic runtime health line (0 = default 60)
  memory_limit_mb: 0               # 0 auto-derives ~70% of RAM as a GC soft cap

sdr:
  watchdog_interval_ms: 30000      # USB-disconnect watchdog; re-acquires by serial
```

Three decisions. **`auth.mode: "auto"`** is the honest LAN posture: token
required on non-loopback binds, loopback exempt. Set it *explicitly* — per
the [hardening guide]({{ '/hardening.html' | relative_url }}), an empty
`mode` now resolves to `disabled` for closed-LAN convenience, with a loud
startup warning on a non-loopback bind. `token_file` keeps the secret out
of `config.yaml` and is re-read per request, so rotation needs no restart.
**`memory_limit_mb: 0`** auto-derives a GC soft cap near 70% of physical
RAM, so the daemon bounds its footprint instead of meeting the OOM-killer.
**`heartbeat_seconds`** makes silence impossible: a periodic
`runtime: heartbeat` line means a stopped log is itself a diagnosis.

## The systemd unit

The repo ships a complete hardened unit. Install, straight from its header:

```sh
sudo install -m 0644 docs/gophertrunk.service /etc/systemd/system/gophertrunk.service
sudo install -m 0755 bin/gophertrunk /usr/local/bin/gophertrunk
sudo install -d -m 0755 /etc/gophertrunk
sudo install -m 0640 config.example.yaml /etc/gophertrunk/config.yaml
sudo systemctl daemon-reload && sudo systemctl enable --now gophertrunk
```

(Edit `/etc/gophertrunk/config.yaml` before the last line — and note the
`0640` mode: the Part 9 key-hygiene rule, enforced.) The lines worth
understanding rather than cargo-culting:

| Unit line | Why it's there |
|---|---|
| `Restart=on-failure` / `RestartSec=5` | crashes come back in five seconds; clean stops stay stopped |
| `DynamicUser=true`, `NoNewPrivileges=true` | no account to manage, no escalation path |
| `ProtectSystem=strict` + `ReadWritePaths=` + `StateDirectory=` | filesystem read-only except where recordings and state live — point `ReadWritePaths` at your Part 10 archive dirs |
| `DeviceAllow=char-usb_device rwm`, `SupplementaryGroups=plugdev` | the pure-Go USBDEVFS backend needs `/dev/bus/usb/*` read/write — no kernel driver, no libusb ([why]({{ '/blog/deep-dives/rf-front-end-01-why-pure-go-drivers/' | relative_url }})) |
| `EnvironmentFile=-/etc/gophertrunk/env` (optional) | secrets outside the unit and the config |

Logs land in the journal — `journalctl -u gophertrunk -f` is your new
console. The udev rule granting non-root dongle access lives in the
[Linux install guide]({{ '/install-linux.html' | relative_url }}); apply it,
then preflight the USB story from a shell:

```sh
gophertrunk sdr doctor        # read-only: which driver owns each known SDR
gophertrunk sdr list --probe  # enumeration + per-device gain ladders
```

`sdr doctor` walks the known RTL-SDR and HackRF VID/PIDs and reports which
kernel (or Windows) driver is bound to each, with an actionable next step —
read-only, safe alongside a live daemon
([preflight deep dive]({{ '/blog/deep-dives/running-it-for-real-07-sdr-doctor-preflight/' | relative_url }})).

## The Docker variation

The repo ships a multi-stage `Dockerfile` and a `docker-compose.yml` whose
`devices:` entry maps the dongle in:

```yaml
    devices:
      - "/dev/bus/usb/003/002:/dev/bus/usb/003/002"
```

Match the path to `lsusb` on the host, pair it with the host-side udev rule
so a non-root container user can open the node, then smoke-test with
`docker exec gophertrunk gophertrunk sdr list`. The
[Docker deep dive]({{ '/blog/deep-dives/running-it-for-real-12-docker-usb/' | relative_url }})
explains the trap: the mapped path names a *specific bus/device number*,
which changes when the dongle re-enumerates — the reason systemd-on-metal
is this recipe's default.

## First run — what healthy looks like

`sudo systemctl start gophertrunk`, then watch the journal. You'll see
Part 1's trio (`api: listening`, `control channel locked`,
`recorder: call started`), plus the appliance's own pulse once a minute:

```
INF runtime: heartbeat uptime=1h0m0s goroutines=142 heap_alloc_mb=88 heap_sys_mb=143 sys_mb=189 next_gc_mb=112 num_gc=412
```

Read it like a vital sign: climbing goroutines or heap across hours is a
leak, a frozen heartbeat on a live process is a hang, and the last line
before an abrupt cut pins the pre-kill footprint. When a dongle drops off
the bus, the watchdog narrates the recovery:

```
WRN sdr: watchdog: device missing from USB enumerate serial=00000001
INF sdr: watchdog: device reappeared; reacquiring serial=00000001
```

Finally, test the thing appliances do most: restart.
`sudo systemctl restart gophertrunk` should log
`gophertrunk shutdown initiated` → `gophertrunk shutdown complete` in well
under a second. That speed is recent work: shutdown used to park for a full
30 s whenever any SSE or live-audio client was attached — nothing told the
streaming handlers to exit while the HTTP server politely waited, and the
timeout was miscounted as a clean exit. Now a stop signal reaches every
long-lived handler, the 30 s is only a cap, and blowing it logs
`api: graceful shutdown window expired — a streaming handler did not exit`.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| Random USB errors, dongle re-enumerates under load | undervoltage — the classic Pi failure | Use a proper supply; check the Pi kernel log for low-voltage warnings; a powered hub helps when a bias-tee LNA browns the board out |
| `sdr: watchdog: device missing from USB enumerate` repeating | flaky cable/port, or power sag | The watchdog re-acquires by serial automatically; if it cycles, treat it as hardware — the [dongle-off-the-bus postmortem]({{ '/blog/solution-postmortem/from-the-issue-tracker-18-the-stall-that-wasnt/' | relative_url }}) is the war story |
| `ccdecoder: decode can't keep up with real time` | CPU-starved board, or thermal throttling | Reduce taps/systems, add cooling, or step up the board; this WARN is the CPU signal, distinct from any RF problem |
| Decodes for days, then killed with no log line | memory pressure — OOM-killer | The auto `memory_limit_mb` guards this; on tiny boards set it explicitly and watch heartbeat `heap_sys_mb` trends |
| `api: auth disabled — mutation endpoints are not authenticated` at startup | empty `auth.mode` on a `0.0.0.0` bind | Set `mode: "auto"` or `"required"` with a `token_file` ([hardening]({{ '/hardening.html' | relative_url }})) |
| Daemon starts, no radio, no error | device-node permissions under the hardened unit | Check the udev rule + `DeviceAllow`/`SupplementaryGroups`; `gophertrunk sdr doctor` tells you who owns the dongle |
| `systemctl stop` takes ~30 s | a streaming handler missed the stop signal (the fixed bug's shape) | The journal WARN above fires at the cap — report it with the log; it's a bug, not a config problem |
| Web console unreachable from the LAN | daemon bound to loopback | `api.http_addr: "0.0.0.0:8080"` — then the auth row above, in that order |

### How this recipe shapes operator practice

- **Preflight, then trust.** `sdr doctor` and `sdr list --probe` before
  `systemctl enable` retires "it doesn't work headless" almost entirely.
- **Make silence impossible.** Heartbeat plus watchdog means every failure
  leaves a journal trail; a box that can fail silently will.
- **Restart as a test.** A fast, clean `systemctl restart` exercises the
  whole teardown path — do it once after setup, while you're still watching.

## Variations

- **The TUI closet.** SSH in and run
  [`gophertrunk tui`]({{ '/tui.html' | relative_url }}) — the full cockpit in
  a terminal, no browser required.
- **Docker Compose stack.** The compose file pairs naturally with a reverse
  proxy for [TLS]({{ '/blog/deep-dives/running-it-for-real-03-tls-reverse-proxy/' | relative_url }})
  when the console must leave the LAN.
- **Split appliance.** Pi at the antenna as a Part 8 `rtl_tcp` / SoapyRemote
  source, decode on a stronger box — closet for RF, desk for DSP.
- **Windows service.** The same daemon runs as a Windows service via the
  installer; the [systemd & Windows deep dive]({{ '/blog/deep-dives/running-it-for-real-13-systemd-windows/' | relative_url }})
  covers both supervisors.

## Where this goes next

The appliance is running; now it can get *better at radio*. [Part
12]({{ '/blog/tutorials/operator-cookbook-12-diversity-mrc/' | relative_url }})
is the advanced RF build: two antennas on one coherent front end, MRC
diversity combining, and the honest accounting of when a second antenna
buys decode margin — and when it just buys a second feedline.

## FAQ

**Can a Raspberry Pi run GopherTrunk 24/7?**
Yes — a Pi 4 comfortably runs a single-system trunking rig with recording
and the web console; a Pi 5 or mini-PC covers multi-system builds. The
binary is pure Go with no native SDR libraries, so the ARM64 build runs
as-is; sizing lives in the [SBC guide]({{ '/best-single-board-computer-for-gophertrunk/' | relative_url }}).

**Do I need to install librtlsdr or libusb on the appliance?**
No. GopherTrunk's drivers speak USB directly through the kernel's USBDEVFS —
the only host requirements are the udev rule granting device-node access
and, under systemd hardening, the `DeviceAllow` line the shipped unit
already carries.

**How do I pass an RTL-SDR into Docker?**
Map the device node with `devices:` in `docker-compose.yml` (or `--device`),
matched to `lsusb`, with a host udev rule so the container user can open it;
verify with `docker exec gophertrunk gophertrunk sdr list`. Bus/device
numbers change on re-enumeration — map `/dev/bus/usb` broadly if your
dongle power-cycles.

**How do I know a headless scanner is still healthy?**
Three journal signals: the periodic `runtime: heartbeat` line (alive,
memory bounded), no `sdr: watchdog: device missing` cycles (USB stable),
and the ordinary rhythm of `control channel locked` and
`recorder: call started/ended`. The web Dashboard shows the same facts
graphically from any LAN machine.

**Why does my daemon warn about auth at startup?**
It's bound to a non-loopback address with auth effectively off — the
closed-LAN default. A prompt, not an error: set `api.auth.mode: "auto"` or
`"required"` plus a `token_file` and the warning goes away with the
exposure ([auth-posture deep dive]({{ '/blog/deep-dives/running-it-for-real-02-auth-posture/' | relative_url }})).

## Series navigation

**Part 11 of 14** · ←
[Part 10: The Archival Rig — FLAC, Retention & the Call Log]({{ '/blog/tutorials/operator-cookbook-10-archival-rig/' | relative_url }})
· Next →
[Part 12: Two Antennas, One Signal — A Diversity Build]({{ '/blog/tutorials/operator-cookbook-12-diversity-mrc/' | relative_url }})
