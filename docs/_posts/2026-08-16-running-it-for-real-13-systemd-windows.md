---
title: "Running It For Real, Part 13: systemd Hardening & the Windows Installer"
description: A hardened systemd unit that sandboxes the daemon down to one USB device and a state directory, and the Inno Setup Windows installer that separates program from data, seeds a config, bundles the Zadig driver, and preserves your captures on uninstall.
category: deep-dives
keywords: systemd hardening, systemd sandboxing, deviceallow usb, protectsystem strict, inno setup installer, windows service sdr, zadig winusb, data root separation, gophertrunk running it for real
tags: [running-it-for-real, systemd, windows, hardening, deployment, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Running It For Real"
series_part: 13
---

*Part 13 of **Running It For Real**. Part 12 isolated the daemon in a container.
The two native-OS deployments — a Linux systemd service and a Windows install —
want the same result by different means. On Linux, systemd can sandbox a native
process nearly as tightly as a container: give it one USB device, one writable
directory, and nothing else. On Windows, the work is different in kind — an
installer that separates the program from the operator's irreplaceable captures,
seeds a working config, bundles the driver, and — the part that separates a good
installer from a hostile one — never eats your recordings on uninstall. This post
is both, because "runs natively" is two very different engineering problems that
share one goal: least surprise, least privilege.*

> **TL;DR:** The shipped systemd unit is `Type=exec` with `Restart=on-failure`,
> wrapped in a sandbox: `DynamicUser=true`, `ProtectSystem=strict`,
> `ProtectHome=true`, a `StateDirectory` for the only writable path, and the full
> `NoNewPrivileges` / `ProtectKernel*` / `Restrict*` set — punctured by exactly one
> hole, `DeviceAllow=char-usb_device rwm` plus `SupplementaryGroups=plugdev`, so the
> pure-Go USBDEVFS backend can reach the dongle. The Windows side is an Inno Setup
> script that installs the program to Program Files but keeps **all operator files
> in a separate data root**, seeds a starter `config.yaml` that survives reinstalls,
> bundles the Zadig WinUSB driver, and on uninstall **preserves recordings, iq, and
> exports** while offering to clear the rest.

**Key takeaways**

- **systemd sandboxes a native process like a container does.** `ProtectSystem`,
  `ProtectHome`, and a `StateDirectory` leave the daemon one writable path and a
  read-only rest-of-system.
- **One deliberate hole.** `DeviceAllow=char-usb_device rwm` is the systemd analogue
  of `--device` — the single capability the whole sandbox exists to grant safely.
- **The Windows installer separates program from data.** The `.exe` lands in
  Program Files; config, recordings, IQ, and exports live under a user-chosen data
  root that reinstalls don't touch.
- **Uninstall never destroys captures.** Recordings, IQ, and exports are preserved
  unconditionally; only config/data/logs/web are offered for deletion, defaulting to
  keep.

## Cheat sheet

| Piece | Where it lives | What it does |
|---|---|---|
| Service unit | `docs/gophertrunk.service` | `Type=exec`, restart-on-failure |
| Filesystem sandbox | `ProtectSystem=strict`, `StateDirectory=gophertrunk` | one writable path, read-only rest |
| The USB hole | `DeviceAllow=char-usb_device rwm` + `SupplementaryGroups=plugdev` | reach `/dev/bus/usb/*` |
| Secret injection | `EnvironmentFile=-/etc/gophertrunk/env` | bearer token out of `config.yaml` |
| Windows installer | `installer/windows/gophertrunk.iss` | Inno Setup, admin, x64 |
| Data root | `{code:DataDir}` (default `Documents\GopherTrunk`) | config/recordings/iq/exports/… |
| Driver | bundled `zadig.exe` | one-time WinUSB bind per dongle |

## In this post

- **The unit, minus the sandbox** — restart policy and secret injection.
- **The sandbox** — what systemd takes away, and the one thing it leaves.
- **The USB hole** — the single `DeviceAllow` the whole unit is built around.
- **The Windows installer** — program vs data, and the config that survives.
- **A humane uninstall** — why your captures outlive the software.

## The unit, minus the sandbox

Strip the hardening away and the systemd unit is unremarkable — which is the
point. It runs the daemon, restarts it if it dies, and pulls a secret out of an
environment file instead of `config.yaml`:

```ini
# docs/gophertrunk.service (shape)
[Service]
Type=exec
ExecStart=/usr/local/bin/gophertrunk run -config /etc/gophertrunk/config.yaml
Restart=on-failure
RestartSec=5

# Pull a bearer token out of an env file so it's not committed in config.yaml;
# reference it from config.yaml as api.auth.token_file.
# EnvironmentFile=-/etc/gophertrunk/env
```

`Type=exec` means systemd considers the service started once the binary has
`exec`'d — simpler and more honest than `forking`, since the daemon never
daemonizes itself. `Restart=on-failure` with `RestartSec=5` is the base-level
supervisor: a crash gets a fresh process five seconds later, the native analogue
of the container's `restart: unless-stopped`. And the commented `EnvironmentFile`
is the pattern we've pointed at all series for every credential — the Broadcastify
key, the Icecast password, the API bearer token belong in an env file
`0640`-owned by root, not in a config that gets read and copied everywhere.

## The sandbox: what systemd takes away

The rest of the `[Service]` block is a container-grade sandbox built from systemd
directives. The design is *deny by default, permit one path*:

```ini
# docs/gophertrunk.service (shape) — the sandbox
DynamicUser=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/gophertrunk /etc/gophertrunk/last_cc_cache.json
StateDirectory=gophertrunk
StateDirectoryMode=0750
NoNewPrivileges=true
ProtectKernelModules=true
ProtectKernelTunables=true
ProtectControlGroups=true
RestrictRealtime=true
RestrictNamespaces=true
RestrictSUIDSGID=true
SystemCallArchitectures=native
```

`DynamicUser=true` runs the daemon as an ephemeral, unprivileged user that exists
only for the service's lifetime — no account to manage, no home to leak into.
`ProtectSystem=strict` mounts the entire filesystem read-only, and `ProtectHome=true`
hides `/home`, `/root`, and `/run/user` entirely. Against that read-only backdrop,
`StateDirectory=gophertrunk` creates and owns exactly one writable path
(`/var/lib/gophertrunk`) at mode `0750`, and `ReadWritePaths` adds the one extra
file the daemon needs to persist (the last-control-channel cache). Everything else
— `NoNewPrivileges`, the three `ProtectKernel*`/`ProtectControlGroups` lines, the
`Restrict*` set, native-only syscalls — closes escalation and lateral-movement
paths a compromised process might otherwise reach. The result is a native daemon
that can write to one directory, read a read-only system, and touch one class of
device. That last clause is the interesting one.

<figure class="lab-figure">
<svg viewBox="0 0 660 178" width="660" height="178" role="img" aria-label="The systemd sandbox as concentric restriction. The outer region is the whole host, marked read-only by ProtectSystem strict with ProtectHome hiding home directories. Inside sits the daemon process, which has exactly two openings: a single writable StateDirectory at /var/lib/gophertrunk, and one DeviceAllow hole to char-usb_device that reaches the RTL-SDR. Everything else — kernel modules, tunables, namespaces, SUID — is blocked.">
  <rect x="20" y="20" width="620" height="140" rx="10" fill="none" stroke="var(--fg-muted)"/>
  <text x="150" y="40" text-anchor="middle" fill="var(--fg-muted)" font-size="10">host filesystem — read-only (ProtectSystem=strict, ProtectHome)</text>
  <rect x="230" y="58" width="200" height="80" rx="8" fill="none" stroke="var(--accent)"/>
  <text x="330" y="84" text-anchor="middle" fill="var(--accent)" font-size="11">gophertrunk</text>
  <text x="330" y="100" text-anchor="middle" fill="var(--fg-muted)" font-size="9">DynamicUser</text>
  <text x="330" y="114" text-anchor="middle" fill="var(--fg-muted)" font-size="9">NoNewPrivileges</text>
  <rect x="60" y="74" width="130" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="125" y="94" text-anchor="middle" fill="currentColor" font-size="9">StateDirectory</text>
  <text x="125" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="8">/var/lib/gophertrunk (rw)</text>
  <line x1="230" y1="98" x2="192" y2="98" stroke="currentColor"/><polygon points="192,94 184,98 192,102" fill="currentColor"/>
  <rect x="470" y="74" width="130" height="48" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="535" y="94" text-anchor="middle" fill="var(--accent)" font-size="9">DeviceAllow</text>
  <text x="535" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="8">char-usb_device rwm</text>
  <line x1="430" y1="98" x2="468" y2="98" stroke="var(--accent)"/><polygon points="468,94 476,98 468,102" fill="var(--accent)"/>
  <text x="330" y="152" text-anchor="middle" fill="var(--fg-muted)" font-size="9">blocked: kernel modules · tunables · namespaces · SUID · non-native syscalls</text>
</svg>
<figcaption>Deny by default, permit two paths: one writable state directory, one class of USB device. Everything a compromised process would reach for is closed.</figcaption>
</figure>

## The USB hole the whole unit is built around

A fully-locked sandbox that can't see the radio is useless, so the unit punches
exactly one deliberate hole:

```ini
# docs/gophertrunk.service (shape) — the one device permission
# The pure-Go USBDEVFS backend needs read+write on /dev/bus/usb/<bus>/<dev>;
# libsystemd routes this through cgroup device controllers.
DeviceAllow=char-usb_device rwm
SupplementaryGroups=plugdev
```

This is the systemd analogue of Part 12's `--device` plus `group_add`.
`DeviceAllow=char-usb_device rwm` grants the sandbox read/write/mknod on USB
character devices through the cgroup device controller; `SupplementaryGroups=plugdev`
puts the `DynamicUser` in the group the host's udev rule granted, so the node is
actually openable. The comment names the reason precisely: the pure-Go USBDEVFS
backend talks to `/dev/bus/usb/<bus>/<dev>` directly, so it needs read+write on the
node, and this is the one line that lets it through the otherwise-total device
lockdown. Operators who want tighter scope replace this with an explicit
`DeviceAllow` per enumerated dongle; the class-wide form is the trusted-host
default. There's a parallel, commented pair for audio (`char-alsa` +
`SupplementaryGroups=audio`) that stays off unless `audio.enabled` is set — the
same opt-in posture as Part 8. The unit even documents its own sharp edge:
`PrivateUsers=false` is required today because the USB transport uses udev's
netlink monitor, a thing to tighten once that dependency is gone.

## The Windows installer: program vs data

Windows has no systemd, and the operating problem shifts from *sandboxing a
process* to *installing software a non-technical operator won't regret*. The
[Inno Setup](https://jrsoftware.org/isinfo.php) script's central decision is to
**split the program from the operator's files**:

```pascal
{ installer/windows/gophertrunk.iss (shape) — two roots }
[Setup]
DefaultDirName={autopf}\GopherTrunk   ; program → Program Files
PrivilegesRequired=admin
ArchitecturesAllowed=x64compatible

[Dirs]                                 ; data → user-chosen root
Name: "{code:DataDir}\config"
Name: "{code:DataDir}\recordings"
Name: "{code:DataDir}\iq"
Name: "{code:DataDir}\exports"
Name: "{code:DataDir}\data"
Name: "{code:DataDir}\logs"
Name: "{code:DataDir}\web"
```

The `.exe` installs to Program Files (`{autopf}`), where it belongs. But every file
the operator *creates or cares about* — config, voice recordings, raw IQ, CSV/PDF
exports, the SQLite database, logs, and the bundled web consoles — lives under a
**separate data root** the installer prompts for, defaulting to
`Documents\GopherTrunk` because that's the spot a non-admin user can always write
to. The shipped `config.example.yaml` uses config-relative paths (`../recordings`,
`../iq`, …), so once the seeded `config.yaml` lands in `<DataRoot>\config` the
daemon writes everything into the sibling folders automatically.

Two registry values make the daemon discoverable without a `-config` flag:
`GOPHERTRUNK_CONFIG` points at the seeded config, and `GOPHERTRUNK_HOME` at the
data root — and because the installer sets `ChangesEnvironment=yes`, Inno
broadcasts `WM_SETTINGCHANGE` so a freshly-opened shell sees them. The config
itself is seeded with `onlyifdoesntexist uninsneveruninstall`, so **a reinstall
never overwrites operator edits and an uninstall never deletes the config file**.
The installer also bundles `zadig.exe` — the WinUSB driver tool — so the operator
binds their RTL-SDR's driver in one click rather than chasing a download, and drops
Start-menu shortcuts for the three web consoles (standard, Signal Lab, Config
Builder).

### How that principle shaped the installer

- **Config-relative paths make one data-root prompt enough.** Because every default
  path in `config.yaml` is relative to the config file, the installer asks for *one*
  folder and every other file follows — no per-folder prompts, no absolute paths
  baked into a machine-specific config.
- **The data root is bridged into the uninstaller via the registry.** Inno's `[Code]`
  state from the install run doesn't survive into the uninstall run, so the data root
  is persisted to `HKLM\Software\GopherTrunk\Install\DataDir` — the only durable way
  the uninstaller can find the operator's files to (optionally) clean them.
- **Every env var and PATH edit is reversible.** `GOPHERTRUNK_CONFIG` /
  `GOPHERTRUNK_HOME` carry `uninsdeletevalue`, and the PATH entry is stripped by an
  explicit uninstall step that sandwiches the match in semicolons so it never chops a
  path that's a prefix of another.

## A humane uninstall

The single most operator-respecting line in the whole installer is what it *won't*
delete. On uninstall, the script strips its PATH entry, then asks whether to remove
config, database, logs, and web console — and no matter what the operator answers,
it **never touches recordings, IQ, or exports**:

```pascal
{ installer/windows/gophertrunk.iss (shape) — WipeManagedData }
{ Deletes only Setup-managed folders; the operator's irreplaceable
  captures (recordings, iq, exports) are DELIBERATELY preserved. }
DelTree(Root + 'config', True, True, True);
DelTree(Root + 'data',   True, True, True);
DelTree(Root + 'logs',   True, True, True);
DelTree(Root + 'web',    True, True, True);
```

The uninstall prompt defaults to **No** (`MB_DEFBUTTON2`), and even a Yes leaves
the capture folders standing. This is the desktop equivalent of the container's
bind-mounted volumes: the software is disposable, the data is not. An operator who
ran GopherTrunk for six months and then uninstalls it still has every recording —
which is exactly the promise a 24/7 receiver should keep.

## Where this goes next

We've now hardened *where* the daemon runs on every surface — container, systemd,
Windows. The finale,
[Part 14]({{ '/blog/deep-dives/running-it-for-real-14-staying-up/' | relative_url }}),
is about *staying* up: the health endpoints an orchestrator or a monitor polls, the
heartbeat that makes a stop never silent, the USB watchdog that reacquires a dropped
dongle without a restart, and what months of uninterrupted uptime actually look
like — the operational payoff the whole series has been building toward.

## FAQ

**Do I have to use all the systemd hardening directives?**
No — the unit's comment says most operators leave them alone but to remove any that
block your specifics. The load-bearing ones are `ProtectSystem=strict` +
`StateDirectory` (the writable-path sandbox) and the `DeviceAllow` + `SupplementaryGroups`
pair (USB access). Drop a `Restrict*` line if it conflicts with your host, but keep the
device permission or the daemon can't see the radio.

**Why `DeviceAllow=char-usb_device rwm` instead of a specific device?**
It's the trusted-host default — a class-wide grant that works regardless of which bus
the dongle enumerates on. For tighter scope, replace it with an explicit `DeviceAllow`
line per enumerated RTL-SDR. Either way `SupplementaryGroups=plugdev` is needed so the
`DynamicUser` is in the group your udev rule granted.

**Where does the Windows installer put my recordings?**
Under the data root you choose at install time (default `Documents\GopherTrunk`), in a
`recordings` subfolder — separate from the program in Program Files. Config, IQ, exports,
database, and logs are siblings under the same root, so everything you care about lives
in one place you pick and can back up.

**Will reinstalling or upgrading overwrite my config?**
No. The seeded `config.yaml` uses `onlyifdoesntexist`, so an existing config is never
overwritten by a reinstall, and `uninsneveruninstall` means an uninstall leaves it in
place too. Your edits survive both.

**What does uninstall delete?**
Only Setup-managed folders — config, data, logs, web — and only if you say yes to the
prompt (which defaults to No). Your recordings, IQ captures, and exports are preserved
unconditionally. The uninstaller also strips its PATH entry and its environment
variables, but never your captures.

## Series navigation

**Part 13 of 14** · ←
[Part 12: Docker & RTL-SDR USB Pass-Through]({{ '/blog/deep-dives/running-it-for-real-12-docker-usb/' | relative_url }})
· Next →
[Part 14: Staying Up — Health, Watchdogs & the Ops Payoff]({{ '/blog/deep-dives/running-it-for-real-14-staying-up/' | relative_url }})
