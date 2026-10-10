---
layout: page
title: Android (Termux)
description: Run the GopherTrunk binary on an Android phone in Termux, with the dongle reached through an rtl_tcp driver app
nav_group: Reference
---

# Android (Termux)

GopherTrunk publishes Android ARM binaries that run inside
[Termux](https://termux.dev) on Android 5.0 or later. A phone cannot hand its USB dongle
straight to a Termux program, so the dongle is reached the way a remote SDR
is: an **rtl_tcp driver app** owns the USB device and serves it on a local
port, and GopherTrunk connects to that port.

> **Status:** the v1.2.3 Termux builds were Linux binaries and crashed with
> `SIGSYS: bad system call` on most phones (see *Why a separate build* below).
> Releases after v1.2.3 are native Android executables. Reports from a real
> device are welcome on
> [issue #1230](https://github.com/MattCheramie/GopherTrunk/issues/1230).

## 1. Pick the right download

Each [release](https://github.com/MattCheramie/GopherTrunk/releases) carries
two Termux tarballs. In Termux, run `uname -m`:

| `uname -m` | Download |
|---|---|
| `aarch64` | `gophertrunk-<version>-termux-arm64.tar.gz` |
| `armv7l` or `armv8l` | `gophertrunk-<version>-termux-armv7.tar.gz` |

`armv8l` means a 64-bit CPU running a 32-bit Android, as on many phones from
before about 2017. Those phones need the **armv7** build.

```sh
pkg install wget
wget https://github.com/MattCheramie/GopherTrunk/releases/download/<version>/gophertrunk-<version>-termux-armv7.tar.gz
tar xzf gophertrunk-<version>-termux-armv7.tar.gz
cd gophertrunk-<version>-termux-armv7
./gophertrunk version
```

The `make termux-build` target produces the same two binaries from source
(under `dist/`). It needs the
[Android NDK](https://developer.android.com/ndk/downloads): point
`ANDROID_NDK_HOME` at it.

## 2. Serve the dongle with rtl_tcp

Install an rtl_tcp driver app, for example the
[Rtl-sdr driver](https://f-droid.org/en/packages/marto.rtl_tcp_andro/) from
F-Droid. Plug in the dongle through a USB OTG adapter, start the driver's
rtl_tcp server, and grant it USB access. By default it listens on
`127.0.0.1:1234`. Check the app for the address and port it uses.

## 3. Run it

A power sweep ([full reference](power-sweep.html)):

```sh
./gophertrunk power -rtltcp 127.0.0.1:1234 -f 88M:108M:10k -i 30 -out fm.csv
```

Wireless M-Bus meters at 868.95 MHz ([full reference](wmbus.html)):

```sh
./gophertrunk wmbus -rtltcp 127.0.0.1:1234
```

The daemon works the same way, with the dongle listed under `sdr.rtl_tcp`
(see [Remote rtl_tcp SDRs](hardware.html#remote-rtl_tcp-sdrs)).

## Limits

- **Prefer an IP address.** The Android build resolves host names through
  Android's own resolver, so a LAN or internet host name should work, but
  this has not been checked on a phone yet. `127.0.0.1` or a numeric LAN
  address always works.
- **No live audio.** Android does not give Termux programs access to the
  sound hardware. With `audio.enabled: true` the daemon logs that the audio
  backend failed and carries on without sound; set `audio.enabled: false` to
  skip that. Recordings, the web console and the CLI tools are unaffected.
- **CPU.** An older phone may struggle at 2.4 MS/s. If GopherTrunk warns
  that it is dropping IQ, lower `-rate` (or `sdr.sample_rate`).
- **Why a separate build:** the regular Linux binary cannot start on
  Android, for two reasons.
  - It loads ALSA's `libasound` through glibc's `libdl` for live audio.
    Android uses a different C library with neither. The Termux build uses
    `-tags nolibasound`, which leaves that out.
  - Android runs every app, Termux included, under a sandbox filter that
    kills the process with `SIGSYS: bad system call` when it makes a
    blocked system call. One of them, `faccessat2`, is what Go's Linux build
    uses whenever it looks up an installed program, and the clipboard
    library does that at start-up for `termux-clipboard-set`. The v1.2.3
    Termux builds were Linux binaries, so they crashed on any phone with the
    Termux:API package installed. The Termux builds are now native Android
    executables (`GOOS=android`), where Go never makes that call.
