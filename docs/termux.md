---
layout: page
title: Android (Termux)
description: Run the GopherTrunk binary on an Android phone in Termux, with the dongle reached through an rtl_tcp driver app
nav_group: Reference
---

# Android (Termux)

GopherTrunk publishes static Linux ARM binaries that run inside
[Termux](https://termux.dev) on Android. A phone cannot hand its USB dongle
straight to a Termux program, so the dongle is reached the way a remote SDR
is: an **rtl_tcp driver app** owns the USB device and serves it on a local
port, and GopherTrunk connects to that port.

> **Status:** the Termux builds are new and have been checked for static
> linking only, not yet run on a phone. Reports from a real device are
> welcome on [issue #1230](https://github.com/MattCheramie/GopherTrunk/issues/1230).

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
(under `dist/`).

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

The daemon works the same way, with the dongle listed under `sdr.rtl_tcp`
(see [Remote rtl_tcp SDRs](hardware.html#remote-rtl_tcp-sdrs)).

## Limits

- **Use an IP address, not a host name.** A static Go binary looks up DNS
  servers in `/etc/resolv.conf`, which Android does not have, so a LAN or
  internet host name may fail to resolve. `127.0.0.1` or a numeric LAN
  address always works.
- **No live audio.** Android does not give Termux programs access to the
  sound hardware. With `audio.enabled: true` the daemon logs that the audio
  backend failed and carries on without sound; set `audio.enabled: false` to
  skip that. Recordings, the web console and the CLI tools are unaffected.
- **CPU.** An older phone may struggle at 2.4 MS/s. If GopherTrunk warns
  that it is dropping IQ, lower `-rate` (or `sdr.sample_rate`).
- **Why a separate build:** the regular Linux binary loads ALSA's
  `libasound` through glibc's `libdl` for live audio. Android uses a
  different C library with neither, so that binary does not start there.
  The Termux build is compiled with `-tags nolibasound`, which leaves that
  out and makes the binary fully static.
