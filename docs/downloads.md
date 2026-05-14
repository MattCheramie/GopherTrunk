---
layout: page
title: Downloads
description: Install GopherTrunk on Linux, macOS, or Windows
nav_group: Install
---

{%- assign latest = site.github.latest_release -%}
{%- if latest and latest.tag_name -%}
  {%- assign ver = latest.tag_name -%}
{%- else -%}
  {%- assign ver = "v0.1.0" -%}
{%- endif -%}
{%- assign rel_url = "https://github.com/MattCheramie/GopherTrunk/releases/download/" | append: ver -%}

# Download GopherTrunk

<p class="downloads-version">
  Latest release: <a href="https://github.com/MattCheramie/GopherTrunk/releases/tag/{{ ver }}"><strong>{{ ver }}</strong></a>
  · <a href="https://github.com/MattCheramie/GopherTrunk/releases">all releases</a>
  · <a href="{{ rel_url }}/SHA256SUMS"><code>SHA256SUMS</code></a>
  · <a href="#verify-your-download">verify</a>
</p>

<div class="download-cards">

  <div class="download-card" data-platform="linux">
    <h3>Linux</h3>
    <p class="download-card__lede">Tarballed static binary. No <code>librtlsdr</code> or <code>libusb</code> at runtime.</p>
    <div class="download-card__buttons">
      <a class="btn btn--primary" href="{{ rel_url }}/gophertrunk-{{ ver }}-linux-amd64.tar.gz">x86_64 (.tar.gz)</a>
      <a class="btn btn--primary" href="{{ rel_url }}/gophertrunk-{{ ver }}-linux-arm64.tar.gz">aarch64 (.tar.gz)</a>
    </div>
    <p class="download-card__note">Full walkthrough: <a href="{{ '/install-linux.html' | relative_url }}">Linux install guide</a>. aarch64 covers Raspberry Pi 4 / 5 + most modern ARM SBCs; RTL-SDR needs udev rules + DVB blacklist on first install.</p>
  </div>

  <div class="download-card" data-platform="macos">
    <h3>macOS</h3>
    <p class="download-card__lede">Static binary + sample config. Bundled with README and LICENSE.</p>
    <div class="download-card__buttons">
      <a class="btn btn--primary" href="{{ rel_url }}/gophertrunk-{{ ver }}-darwin-arm64.tar.gz">Apple Silicon (.tar.gz)</a>
      <a class="btn btn--primary" href="{{ rel_url }}/gophertrunk-{{ ver }}-darwin-amd64.tar.gz">Intel (.tar.gz)</a>
    </div>
    <p class="download-card__note">Full walkthrough: <a href="{{ '/install-macos.html' | relative_url }}">macOS install guide</a>. Builds are unsigned — right-click → Open the first time, or run <code>xattr -dr com.apple.quarantine gophertrunk</code>.</p>
  </div>

  <div class="download-card" data-platform="windows">
    <h3>Windows 11</h3>
    <p class="download-card__lede">One-click installer (x64), portable ZIPs for x64 and ARM.</p>
    <div class="download-card__buttons">
      <a class="btn btn--primary" href="{{ rel_url }}/gophertrunk-{{ ver }}-windows-amd64-setup.exe">x64 Installer (.exe)</a>
      <a class="btn btn--secondary" href="{{ rel_url }}/gophertrunk-{{ ver }}-windows-amd64.zip">x64 Portable (.zip)</a>
      <a class="btn btn--secondary" href="{{ rel_url }}/gophertrunk-{{ ver }}-windows-arm64.zip">ARM64 Portable (.zip)</a>
    </div>
    <p class="download-card__note">Full walkthrough: <a href="{{ '/install-windows.html' | relative_url }}">Windows install guide</a>. After install, swap the WinUSB driver via Zadig — the OS won't see your RTL-SDR until you do this once.</p>
  </div>

</div>

<p class="downloads-jump">
  Jump to: <a href="#linux">Linux quick start</a> · <a href="#macos">macOS quick start</a> · <a href="#windows-11">Windows quick start</a> · <a href="#verify-your-download">Verify</a> · <a href="#build-from-source">Build from source</a> · <a href="#docker">Docker</a>
</p>

[releases]: https://github.com/MattCheramie/GopherTrunk/releases
[latest]: https://github.com/MattCheramie/GopherTrunk/releases/latest

## Quick start by OS

### Linux

```sh
VERSION={{ ver }}
ARCH=$(uname -m)                            # x86_64 or aarch64 / arm64
case "$ARCH" in
  x86_64)         PKG=linux-amd64 ;;
  aarch64|arm64)  PKG=linux-arm64 ;;
esac
curl -L -o gophertrunk.tar.gz \
  https://github.com/MattCheramie/GopherTrunk/releases/download/${VERSION}/gophertrunk-${VERSION}-${PKG}.tar.gz
tar xzf gophertrunk.tar.gz
cd gophertrunk-${VERSION}-${PKG}
cp config.example.yaml config.yaml          # edit before launch
./gophertrunk version                       # confirms ldflags landed
./gophertrunk run -config config.yaml
```

For a systemd-managed install, copy [`gophertrunk.service`](https://github.com/MattCheramie/GopherTrunk/blob/main/docs/gophertrunk.service) to `/etc/systemd/system/` and follow the install header.

### macOS

```sh
VERSION={{ ver }}
ARCH=$(uname -m)                            # arm64 on Apple Silicon, x86_64 on Intel
case "$ARCH" in
  arm64)  PKG=darwin-arm64 ;;
  x86_64) PKG=darwin-amd64 ;;
esac
curl -L -o gophertrunk.tar.gz \
  https://github.com/MattCheramie/GopherTrunk/releases/download/${VERSION}/gophertrunk-${VERSION}-${PKG}.tar.gz
tar xzf gophertrunk.tar.gz
cd gophertrunk-${VERSION}-${PKG}
xattr -dr com.apple.quarantine gophertrunk  # bypass Gatekeeper (unsigned build)
cp config.example.yaml config.yaml
./gophertrunk version
./gophertrunk run -config config.yaml
```

RTL-SDR on macOS uses the bundled IOKit driver — no kext or driver swap required.

### Windows 11

Run the installer:

```powershell
# Or just double-click the .exe in Explorer.
.\gophertrunk-{{ ver }}-windows-amd64-setup.exe
```

After install, complete the WinUSB driver swap via Zadig — see **[`install-windows.md`]({{ '/install-windows.html' | relative_url }})** for the full step-by-step (the installer's last page links there too). The OS won't see your RTL-SDR until that swap is done.

## Verify your download

Every binary archive is SHA-256-checksummed. Refuse to install on a hash mismatch:

```sh
# Linux / macOS
curl -L -O {{ rel_url }}/SHA256SUMS
sha256sum --ignore-missing -c SHA256SUMS    # Linux
shasum -a 256 --ignore-missing -c SHA256SUMS  # macOS
```

```powershell
# Windows
$expected = (Get-Content SHA256SUMS | Select-String "windows-amd64-setup.exe").ToString().Split(" ")[0]
$actual = (Get-FileHash gophertrunk-{{ ver }}-windows-amd64-setup.exe -Algorithm SHA256).Hash.ToLower()
if ($actual -ne $expected) { throw "checksum mismatch" }
```

The daemon also self-reports its build provenance — every release pins the commit + build time at link time:

```sh
$ ./gophertrunk version
{{ ver }} (sha=abc1234, built=2026-05-13T19:00:00Z)
```

The `sha=` value matches the commit on the [release tag][releases]; `built=` is the UTC timestamp the CI runner produced the artefact. Both are injected via `-ldflags` and are empty on `go run` / `go test` builds.

## Build from source

For every platform, the full source build is:

```sh
git clone https://github.com/MattCheramie/GopherTrunk.git
cd gophertrunk
make build               # → ./bin/gophertrunk (static, CGO_ENABLED=0)
make test                # unit tests
make integration         # daemon end-to-end (no SDR required)
```

Requires Go 1.25+ — the project's `go.mod` pins the toolchain to 1.25.10 (closes the 23 stdlib CVEs in the bare 1.25.0). See **[`CONTRIBUTING.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/CONTRIBUTING.md)** for the full dev setup.

## Docker

The repository ships a multi-stage `Dockerfile` + `docker-compose.yml` with RTL-SDR USB pass-through wired:

```sh
git clone https://github.com/MattCheramie/GopherTrunk.git
cd gophertrunk
docker compose up -d
```

See **[`hardening.md` §"Docker"]({{ '/hardening.html#docker' | relative_url }})** for the USB pass-through + healthcheck + Prometheus scrape config.

## Security

Found a vulnerability? Please follow the responsible-disclosure process in **[`SECURITY.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/SECURITY.md)** — do not open a public issue. Use GitHub's private security advisory workflow:

<https://github.com/MattCheramie/GopherTrunk/security/advisories/new>

## Older releases

Every prior tag stays on GitHub's [Releases page][releases]; binaries remain downloadable indefinitely. Security fixes only back-port to the most recent stable tag — older tags receive best-effort support.
