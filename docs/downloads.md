---
layout: default
title: Downloads
description: Install GopherTrunk on Linux, macOS, or Windows
---

# Download GopherTrunk

Each tagged release publishes signed-and-checksummed binaries to the
**[Releases page on GitHub][releases]**. Pick the bundle for your
platform below.

[releases]: https://github.com/MattCheramie/GopherTrunk/releases
[latest]: https://github.com/MattCheramie/GopherTrunk/releases/latest

<p align="center">
  <a href="{{ '/releases.html' | relative_url }}" class="btn">Latest release</a>
  <a href="https://github.com/MattCheramie/GopherTrunk/releases" class="btn">All releases</a>
</p>

## What ships in every release

Every tag produces a static, dependency-free binary built with
`CGO_ENABLED=0`. No `librtlsdr`, no `libusb`, no system libraries
required at runtime. The release artefact bundle is:

| File                                                   | Platform   | What it is |
| ------------------------------------------------------ | ---------- | ---------- |
| `gophertrunk-<ver>-windows-amd64-setup.exe`            | Windows 11 | Inno Setup installer — one-click install, Start Menu entry, signed binary inside |
| `gophertrunk-<ver>-windows-amd64.zip`                  | Windows 11 | Portable ZIP — same binary, no installer (extract anywhere) |
| `gophertrunk-<ver>-linux-amd64.tar.gz`                 | Linux      | Tarballed static binary + `README.md` + `LICENSE` + `config.example.yaml` |
| `SHA256SUMS`                                           | all        | SHA-256 checksums for every binary archive in the release |

## Linux

```sh
# 1. Fetch the tarball + checksum
VERSION=v0.99.0   # or v1.0.0, whatever's listed as the latest release
curl -L -o gophertrunk.tar.gz \
    https://github.com/MattCheramie/GopherTrunk/releases/download/${VERSION}/gophertrunk-${VERSION}-linux-amd64.tar.gz
curl -L -o SHA256SUMS \
    https://github.com/MattCheramie/GopherTrunk/releases/download/${VERSION}/SHA256SUMS

# 2. Verify the checksum (refuse to install on a hash mismatch)
sha256sum --ignore-missing -c SHA256SUMS

# 3. Extract + run
tar xzf gophertrunk.tar.gz
cd gophertrunk-${VERSION}-linux-amd64
cp config.example.yaml config.yaml          # edit before launch
./gophertrunk version                       # confirms ldflags landed
./gophertrunk run -config config.yaml
```

RTL-SDR access on Linux needs the udev rules + DVB-driver blacklist
documented in **[`hardware.md`]({{ '/hardware.html' | relative_url }})**.
For a systemd-managed install, copy
[`gophertrunk.service`](https://github.com/MattCheramie/GopherTrunk/blob/main/docs/gophertrunk.service)
to `/etc/systemd/system/` and follow the install header.

## Windows 11

Run the installer:

```powershell
# Or just double-click the .exe in Explorer.
.\gophertrunk-v0.99.0-windows-amd64-setup.exe
```

After install, complete the WinUSB driver swap via Zadig — see
**[`install-windows.md`]({{ '/install-windows.html' | relative_url }})**
for the full step-by-step (the installer's last page links there too).
The OS won't see your RTL-SDR until that swap is done.

## macOS

Tagged releases don't currently ship a pre-built macOS bundle —
the CI matrix builds + tests the macOS path on every PR via the
`usb-macos` job, but signing + notarisation are still operator
work. Build from source:

```sh
git clone https://github.com/MattCheramie/GopherTrunk.git
cd gophertrunk
make build
./bin/gophertrunk version
```

A pre-built macOS bundle lands once code-signing + notarisation
are wired into `release.yml` — track via
[issue #XX](https://github.com/MattCheramie/GopherTrunk/issues)
or open a new one if it doesn't exist yet.

## Build from source

For every platform, the full source build is:

```sh
git clone https://github.com/MattCheramie/GopherTrunk.git
cd gophertrunk
make build               # → ./bin/gophertrunk (static, CGO_ENABLED=0)
make test                # unit tests
make integration         # daemon end-to-end (no SDR required)
```

Requires Go 1.25+ — the project's `go.mod` pins the toolchain to
1.25.10 (closes the 23 stdlib CVEs in the bare 1.25.0). See
**[`CONTRIBUTING.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/CONTRIBUTING.md)**
for the full dev setup.

## Docker

The repository ships a multi-stage `Dockerfile` + `docker-compose.yml`
with RTL-SDR USB pass-through wired:

```sh
git clone https://github.com/MattCheramie/GopherTrunk.git
cd gophertrunk
docker compose up -d
```

See **[`hardening.md` §"Docker"]({{ '/hardening.html#docker' | relative_url }})**
for the USB pass-through + healthcheck + Prometheus scrape config.

## Verifying a build

After installing, the daemon binary self-reports its build
provenance:

```sh
$ ./gophertrunk version
v0.99.0 (sha=abc1234, built=2026-05-13T19:00:00Z)
```

The `sha=` value matches the commit on the [Releases page][releases]
that produced the binary; `built=` is the UTC timestamp the CI
runner produced the artefact. Both are injected at link time via
`-ldflags` — they're empty on `go run` / `go test` builds and on
binaries built without the `Makefile` / release workflow.

## What's in this release

The full per-release changelog lives at
**[`CHANGELOG.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/CHANGELOG.md)**.
Each tagged release also generates a GitHub-side release-notes
section automatically from the merged-PR history.

## Security

Found a vulnerability? Please follow the responsible-disclosure
process in
**[`SECURITY.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/SECURITY.md)** —
do not open a public issue. Use GitHub's private security advisory
workflow:

<https://github.com/MattCheramie/GopherTrunk/security/advisories/new>

## Older releases

Every prior tag stays on GitHub's [Releases page][releases]; binaries
remain downloadable indefinitely. Security fixes only back-port to
the most recent stable tag — older tags receive best-effort support.
