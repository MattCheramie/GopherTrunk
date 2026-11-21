---
title: "The Operator's Cookbook Vol. 2, Part 10: The Phone Rig — GopherTrunk in Termux"
description: "A complete recipe for running GopherTrunk on an Android phone in Termux: which tarball to download and why it is a GOOS=android build, an rtl_tcp driver app in front of the dongle, the first commands, wake locks, and how to read a drop warning. It states plainly what reporters have run on phones and what nobody has."
category: tutorials
keywords: gophertrunk termux android, rtl-sdr android termux sdr, sigsys bad system call termux go, faccessat2 seccomp android go binary, rtl_tcp driver app android, sdr scanner on a phone, termux wake lock sdr, goos android cross compile ndk, gophertrunk phone rig
tags: [operator-cookbook, cookbook-volume-2, android, termux, rtl-tcp, portable]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook, Volume 2"
series_part: 10
---

*Part 10 of **The Operator's Cookbook, Volume 2**, a 14-part run of
complete, copy-paste GopherTrunk builds — one working rig per part — built
on what landed after Volume 1.
[Part 9]({{ '/blog/tutorials/cookbook-volume-2-09-hold-avoid-priority/' | relative_url }})
drove a desk rig like a scanner. This part takes the desk away: an old
Android phone, Termux, an RTL-SDR on an OTG cable. It is the least
powerful rig in either volume and the only one that fits in a jacket, and
most of the recipe is about the two things a phone does that a PC does
not — refuse to run a Linux binary, and quietly stop giving you CPU.*

> **TL;DR:** The Termux tarballs (`…-termux-arm64` and `…-termux-armv7`;
> `uname -m` tells you which) are native Android executables built
> `GOOS=android`, because a static Linux binary is killed with `SIGSYS`
> by Android's seccomp filter the first time Go calls `faccessat2`.
> Termux cannot open a USB dongle, so an rtl_tcp driver app owns it and
> GopherTrunk connects with `-rtltcp 127.0.0.1:1234`. `power`, `wmbus`
> and `capture` are the phone's tools; the daemon takes the same dongle
> under `sdr.rtl_tcp`, without live audio. Run `termux-wake-lock`, keep
> the screen on, and read the drop warning: it states what share of real
> time the decoder got. Status: reporters have run `power` and Wireless
> M-Bus walks on Galaxy S5-class phones
> ([#1230](https://github.com/MattCheramie/GopherTrunk/issues/1230),
> [#1256](https://github.com/MattCheramie/GopherTrunk/issues/1256)); the
> binaries cannot run in CI, and the daemon on a phone has no field
> report.

**Key takeaways**

- **The phone build is an Android program, not a Linux one.** Same Go
  source, different target: `GOOS=android`, linked against the NDK. The
  Linux build dies before `main` on many phones.
- **Termux never touches USB.** A driver app serves the dongle as rtl_tcp
  on loopback; a rooted phone is the one exception.
- **Dropped IQ on a phone is usually the power manager.** The warning
  separates a process starved of CPU from a decoder that is simply too
  slow, and the two have different fixes.
- **Give a phone jobs with slack in them.** A sweep with a time budget and
  a burst decoder tolerate a throttled CPU; a continuous multi-channel
  decode does not.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Which tarball | 64-bit or 32-bit Android userland | `uname -m`: `aarch64` is arm64; `armv7l` or `armv8l` is armv7 |
| Reaching the dongle | a driver app owns USB and serves rtl_tcp | `-rtltcp 127.0.0.1:1234` |
| Band survey | sweeps a range to CSV on a time budget | `gophertrunk power -rtltcp` |
| Meter telegrams | decodes Wireless M-Bus at 868.95 MHz | `gophertrunk wmbus -rtltcp` |
| Record now, decode at home | writes IQ and a metadata sidecar | `gophertrunk capture -rtltcp` |
| Staying awake | stops Android throttling the process | `termux-wake-lock`, screen on, battery exemption |
| Dongle fell off the bus | waits for it and carries on | `-reconnect` on `wmbus` (on by default) |
| The daemon | an rtl_tcp device in the SDR pool | `sdr.rtl_tcp[]` with `audio.enabled: false` |

## In this post

- **What you're building** — the phone's signal path, and why the build is
  different.
- **The config** — install, first commands, and a daemon config.
- **First run — what healthy looks like** — real output, then what has
  actually run on a phone.
- **When it doesn't work** — symptom → cause → fix.
- **Variations** — root, a remote dongle, a Pluto over WLAN, building it.

## What you're building

A pocket receiver for the jobs that happen away from the desk: a band
survey from a hilltop, a walk past the utility meters, a capture of a
signal you can only reach on foot.

Two facts shape everything. First, an Android app cannot hand a USB
device to a Termux program. The dongle is therefore reached the way a
remote SDR is: an **rtl_tcp driver app** claims the USB device and serves
it on a local port, and GopherTrunk connects to that port — the same
[rtl_tcp]({{ '/reference/rtl-tcp/' | relative_url }}) path Volume 1 used
for [radios far away]({{ '/blog/tutorials/operator-cookbook-08-remote-radios/' | relative_url }}),
with "far" shortened to loopback.

Second, a Linux binary is not an Android binary, even a fully static one.
Android runs every app, Termux included, under a seccomp filter that
kills the process with `SIGSYS: bad system call` on a blocked system
call. `faccessat2` is on that list, and Go's Linux build makes that call
whenever it looks up an installed program — which a clipboard library did
at start-up, for `termux-clipboard-set`. So the v1.2.3 Termux builds,
which were Linux binaries, crashed before `main` on exactly the phones
that had the Termux:API package installed. Releases since are built by
`scripts/build-android.sh`: `GOOS=android`, cgo against the Android NDK,
API level 21 (Android 5.0), and `-tags nolibasound` because Android has
no ALSA. `scripts/check-android.sh` then fails the build unless the
binary's interpreter is `/system/bin/linker` or `linker64`, it links only
Android system libraries, and it carries no `syscall.faccessat2` symbol.

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="The phone rig's signal path: an RTL-SDR on a USB OTG adapter is owned by an rtl_tcp driver app on the Android phone, which serves it on 127.0.0.1 port 1234; the GopherTrunk Android build running in Termux connects to that port and produces a sweep CSV, telegrams and IQ files, or the web console; on a rooted phone only, GopherTrunk can open the dongle directly without the driver app">
  <rect x="8" y="92" width="86" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="51" y="116" text-anchor="middle" fill="currentColor" font-size="10">RTL-SDR</text>
  <line x1="94" y1="112" x2="148" y2="112" stroke="currentColor"/>
  <text x="121" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="8">USB OTG</text>
  <rect x="134" y="14" width="538" height="204" rx="8" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="403" y="30" text-anchor="middle" fill="var(--fg-muted)" font-size="10">Android phone (5.0 or later)</text>
  <rect x="148" y="84" width="150" height="56" rx="4" fill="none" stroke="currentColor"/>
  <text x="223" y="108" text-anchor="middle" fill="currentColor" font-size="10">rtl_tcp driver app</text>
  <text x="223" y="124" text-anchor="middle" fill="var(--fg-muted)" font-size="9">owns the USB device</text>
  <line x1="298" y1="112" x2="368" y2="112" stroke="var(--accent)"/>
  <text x="333" y="104" text-anchor="middle" fill="var(--accent)" font-size="9">127.0.0.1:1234</text>
  <rect x="368" y="44" width="150" height="136" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="443" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Termux</text>
  <text x="443" y="86" text-anchor="middle" fill="var(--accent)" font-size="10">gophertrunk</text>
  <text x="443" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="9">GOOS=android build</text>
  <text x="443" y="126" text-anchor="middle" fill="currentColor" font-size="9">power · wmbus · capture</text>
  <text x="443" y="142" text-anchor="middle" fill="currentColor" font-size="9">run (the daemon)</text>
  <text x="443" y="166" text-anchor="middle" fill="var(--fg-muted)" font-size="9">no sound hardware</text>
  <line x1="518" y1="112" x2="546" y2="58" stroke="currentColor"/>
  <line x1="518" y1="112" x2="546" y2="112" stroke="currentColor"/>
  <line x1="518" y1="112" x2="546" y2="166" stroke="currentColor"/>
  <rect x="546" y="42" width="118" height="32" rx="4" fill="none" stroke="currentColor"/>
  <text x="605" y="62" text-anchor="middle" fill="currentColor" font-size="9">sweep CSV</text>
  <rect x="546" y="96" width="118" height="32" rx="4" fill="none" stroke="currentColor"/>
  <text x="605" y="116" text-anchor="middle" fill="currentColor" font-size="9">telegrams, IQ files</text>
  <rect x="546" y="150" width="118" height="32" rx="4" fill="none" stroke="currentColor"/>
  <text x="605" y="170" text-anchor="middle" fill="currentColor" font-size="9">web console :8080</text>
  <polyline points="51,132 51,204 443,204 443,180" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="247" y="198" text-anchor="middle" fill="var(--fg-muted)" font-size="8">rooted phones only: direct USB, no driver app</text>
</svg>
<figcaption>On a phone the dongle is a network radio whose network is loopback: the driver app holds USB, and the Android build of GopherTrunk is one rtl_tcp client.</figcaption>
</figure>

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| An Android phone | $0 | Android 5.0 or later; the one in the drawer is the point |
| RTL-SDR and antenna | ~$35 | the same dongle as every other part |
| USB OTG adapter | a few dollars | the phone powers the dongle through it |
| [Termux](https://termux.dev) | $0 | the terminal the binary runs in |
| An rtl_tcp driver app | $0 | for example the [Rtl-sdr driver](https://f-droid.org/en/packages/marto.rtl_tcp_andro/) on F-Droid; a rooted phone can do without |

## The config

There is no config file for the first half of this recipe: the one-shot
tools read none. In Termux:

```sh
uname -m
pkg install wget
V=v1.4.2
ARCH=arm64
wget "https://github.com/MattCheramie/GopherTrunk/releases/download/${V}/gophertrunk-${V}-termux-${ARCH}.tar.gz"
tar xzf "gophertrunk-${V}-termux-${ARCH}.tar.gz"
cd "gophertrunk-${V}-termux-${ARCH}"
./gophertrunk version
```

Set `ARCH` from the first line's answer: `aarch64` means `arm64`;
`armv7l` or `armv8l` means `armv7`. `armv8l` is a 64-bit CPU running a
32-bit Android, common on phones from before about 2017 — the Android
build decides, not the chip, and the arm64 binary on such a phone fails
with `cannot execute: required file not found` because there is no 64-bit
loader to start it. Set `V` to the newest release tag (v1.4.2 as this is
written), and pass over the `v0.0.0` pre-release: it is the v1.4.2 code
under a wrong label.

Now the radio. Open the driver app, plug the dongle into the OTG adapter,
grant the app USB access and start its server; by default it listens on
`127.0.0.1:1234`. Then, one command at a time — an rtl_tcp server serves a
single client:

```sh
termux-wake-lock
./gophertrunk power -rtltcp 127.0.0.1:1234 -f 88M:108M:10k -i 30 -out fm.csv
./gophertrunk wmbus -rtltcp 127.0.0.1:1234 -log walk.log
./gophertrunk capture -rtltcp 127.0.0.1:1234 -freq 868950000 \
  -sample-rate 1600000 -seconds 30 -format u8 -out meters.cu8
```

The first is a band survey
([Part 11]({{ '/blog/tutorials/cookbook-volume-2-11-band-survey/' | relative_url }})),
the second a meter walk
([Part 12]({{ '/blog/tutorials/cookbook-volume-2-12-meter-walk/' | relative_url }})),
the third a recording to decode at home, where nothing is in a hurry.
`-log` appends everything `wmbus` prints to a file as it happens, which
matters when the phone is in a pocket.

The daemon is the same dongle under `sdr.rtl_tcp`. This config logs one
P25 control channel — lock, grants, the web console's activity views —
and nothing more, because one dongle behind one rtl_tcp session is one
tuner:

```yaml
log:
  level: info

storage:
  path: "../data/calls.db"

recordings:
  dir: "../recordings"

audio:
  enabled: false                # Termux has no access to the sound hardware

api:
  http_addr: "127.0.0.1:8080"   # the web console, in the phone's browser

sdr:
  sample_rate: 2_400_000        # lower it if the daemon warns of dropped IQ
  rtl_tcp:
    - addr: "127.0.0.1:1234"    # the rtl_tcp driver app
      serial: "phone-dongle"
      role: control
      gain: "auto"

trunking:
  systems:
    - name: "Metro-P25"
      protocol: p25
      control_channels: [857_262_500]
```

```sh
./gophertrunk run -headless -config config.yaml
```

## First run — what healthy looks like

`./gophertrunk version` prints the version and nothing else. The first
real command shows the Terms of Service and asks once:

```text
Do you accept the GopherTrunk Terms of Service? [yes/no]:
```

`power` then prints its plan and gets on with it:

```text
power: rtl_tcp 127.0.0.1:1234, 88000000–108000000 Hz in 12 hops of 192 × 9375.0 Hz bins (256-point FFT), 30s per sweep
```

`wmbus` announces itself, then prints a progress line every 30 seconds:

```text
wmbus: listening on rtl_tcp 127.0.0.1:1234 at 868.950 MHz, 1600000 S/s (Ctrl-C to stop)
wmbus: listening 5m0s (14:02:39): 3 frames (1 CRC-valid, 2 CRC-failed), 5 access codes in 41 bursts, 0 dropped chunks, ...
```

The number to watch is the last: **dropped chunks must stay at 0**. When it
does not, the warning tells you which of two different problems you have.
This one is the phone, not the decoder:

```text
wmbus: WARNING — the SDR has dropped ... IQ chunk(s) (... since the last line): in the last 3m44s the decoder got through only 30s of IQ (13% of real time), so about ... of air went unheard. Something is starving GopherTrunk of CPU; on a phone that is usually the screen turning off or Termux going to the background. Keep the screen on, run termux-wake-lock before starting, and exempt Termux from battery optimisation.
```

USB delivered every sample; the process was not allowed to run. Every
such line also rings the terminal bell, which Termux turns into a
vibration. The daemon, for its part, should show:

```text
INF rtltcp: connected addr=127.0.0.1:1234 tuner=... gain_count=...
INF control channel locked nac=... freq=857262500 ...
WRN no voice SDR available; voice grants will be dropped — add a role: voice device, or a role: wideband device with voice_taps (see docs/hardware.md)
```

The WARN is expected on this rig: it is the daemon noticing that it has a
control tuner and nothing to follow calls with.

**What has actually run on a phone.** The Android binaries cannot be
executed on a CI host, so `check-android.sh` verifies the file, not its
behaviour; a phone is the only real test. Reporters have supplied these:

- **The failure and the fix.** The v1.2.3 builds died with `SIGSYS` on a
  Galaxy S5 running LineageOS and on an Android 14 phone; a Galaxy S5 Neo
  without Termux:API ran them, which is what pointed at the lookup.
- **`power`.** Measured in #1230: a Galaxy S5 kept up at 2.4 MS/s from its
  fourth hop on; an S5 Neo managed about 38 % and needs a lower `-rate`.
- **`wmbus`.** Several walks on a Galaxy S5 with later releases, v1.4.0
  among them, with the dongle opened directly as root. They are where the
  drop warning's wording, the `-reconnect` behaviour and the bell came
  from. On those runs the diagnostics box read `cpus=2` on a four-core
  phone: Android had taken cores offline.

Not confirmed on a phone: the daemon in any configuration; host-name
resolution in the Android build (use a numeric address); and the burst
gate's CPU saving, which was measured on a desktop.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| A bad-system-call crash the moment it starts | a v1.2.3 Termux build, which was a Linux binary | download any later release |
| "cannot execute: required file not found" | the arm64 build on a 32-bit Android | take the armv7 tarball; check with `uname -m` |
| Permission denied opening the USB device | Termux is not allowed to open it | use the driver app and `-rtltcp`, or run as root |
| The server accepted the connection but sent no header | another client holds the rtl_tcp server, or it lost its dongle | stop the other client and restart the driver app |
| Drop warning with a percentage of real time | Android is throttling the process | `termux-wake-lock`, keep the screen on, exempt Termux from battery optimisation |
| Drop warning that says the decoder is not keeping up | the CPU is too slow for the job, awake or not | record with `gophertrunk capture` and decode the file later |
| A note that the FFT cannot keep up | a sweep on a slow CPU; hops are cut at their time budget | lower the rate, for example `-rate 1024000` |
| "still waiting for the SDR", then a request to replug | the dongle left the USB bus; an OTG port that cut power stays off | unplug and replug it; a `wmbus` run carries on by itself |
| Termux shows that the process completed with signal 9 | Android killed it outright | take the wake lock first; `-log` keeps what was printed |
| Ctrl-C does nothing under su | the process has no controlling terminal | type `q` and Enter |

Why the reporter's dongle left the bus was never established. The notes
name the instrument, not the answer: as root, `dmesg` around the
disconnect — an over-current or VBUS line points at the phone's OTG
supply, a descriptor-read error at the dongle.

### How this recipe shapes operator practice

- **Take the wake lock before the first command, not after the first
  warning.** The minutes a throttled phone loses are not recoverable.
- **Always log to a file.** A phone in a pocket has no scrollback worth
  the name, and a killed process leaves only what `-log` wrote.
- **When in doubt, capture.** A recording decoded at home has no
  real-time limit and no power manager;
  [Field Notebook Part 7]({{ '/blog/tutorials/field-notebook-07-overruns-host-drops/' | relative_url }})
  explains why a drop is a downstream signal.

## Variations

- **A rooted phone.** As root GopherTrunk opens the dongle itself, exactly
  as on a PC: leave out `-rtltcp`. Force-stop the driver app first — only
  one program can hold the dongle.
- **A dongle somewhere else.** `-rtltcp` takes any rtl_tcp server, so the
  phone can be the screen for a Raspberry Pi at the antenna. Use its
  numeric address.
- **A Pluto over WLAN.** `power` and `capture` take `-iio` with a board's
  address, and `gophertrunk iio probe` has been confirmed from Termux
  against a PlutoSky
  ([Part 13]({{ '/blog/tutorials/cookbook-volume-2-13-pluto-native/' | relative_url }})).
- **Build it yourself.** `make termux-build` produces both binaries under
  `dist/`; it needs the Android NDK, with `ANDROID_NDK_HOME` pointing at
  it.

The reference is the [Termux doc page]({{ '/termux.html' | relative_url }}).

## Where this goes next

The first command in this part was a sweep, and it deserves a part of its
own. [Part 11]({{ '/blog/tutorials/cookbook-volume-2-11-band-survey/' | relative_url }})
is `gophertrunk power` in full: the CSV and its honest `samples` column,
the spike at every hop's centre and how it is removed, and how a survey
turns into a list of frequencies worth hunting.

## FAQ

**Can I run GopherTrunk on an Android phone?**
Yes, in Termux, on Android 5.0 or later. Download the Termux tarball for
your phone, serve the RTL-SDR with an rtl_tcp driver app, and point
GopherTrunk at it with `-rtltcp 127.0.0.1:1234`. The one-shot tools are
the ones reporters have run; live audio is not available.

**Why does a Go program crash with SIGSYS bad system call in Termux?**
Android's seccomp filter kills a process that calls `faccessat2`, and a Go
binary built for `GOOS=linux` calls it when it looks up an installed
program. A binary built for `GOOS=android` never does. The fix is the
right build, not a setting.

**Which GopherTrunk Termux download do I need, arm64 or armv7?**
Run `uname -m` in Termux. `aarch64` takes the arm64 tarball; `armv7l` and
`armv8l` take armv7, because `armv8l` means a 32-bit Android on a 64-bit
CPU and has no 64-bit loader.

**Why does GopherTrunk drop IQ when the phone's screen turns off?**
Android throttles background processes, so the decoder gets a fraction of
real time while the dongle keeps delivering samples. The drop warning
prints that fraction. `termux-wake-lock`, a lit screen and a battery
optimisation exemption for Termux are the remedies.

**Is the Android build verified?**
Partly. Reporters have run `power` and Wireless M-Bus walks on Galaxy
S5-class phones, and those runs shaped the tool. The binaries cannot be
run in CI, the daemon has no report from a phone, and each release's real
test is someone's handset.

## Series navigation

**Part 10 of 14** · ←
[Part 9: Hold, Avoid and Priority — Driving It Like a Scanner]({{ '/blog/tutorials/cookbook-volume-2-09-hold-avoid-priority/' | relative_url }})
· Next →
[Part 11: The Band Survey — gophertrunk power]({{ '/blog/tutorials/cookbook-volume-2-11-band-survey/' | relative_url }})
