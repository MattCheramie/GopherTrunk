---
title: "Running It For Real, Part 12: Docker & RTL-SDR USB Pass-Through"
description: Containerizing the pure-Go daemon and passing a real RTL-SDR through the container boundary — the multi-stage zero-CGO build, the three things a dongle needs inside a container (host udev, DVB blacklist, device mapping), and why least-privilege beats --privileged.
category: deep-dives
keywords: docker rtl-sdr, usb passthrough container, udev rules docker, multi-stage go build, cgo disabled sdr, device mapping compose, dac_override capability, usbdevfs, gophertrunk running it for real
tags: [running-it-for-real, docker, usb, deployment, hardening, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Running It For Real"
series_part: 12
---

*Part 12 of **Running It For Real**. Everything so far assumed the daemon runs
wherever you launched it — a shell on your laptop. A real 24/7 service usually
runs in a container, restarted by the orchestrator, isolated from the host. The
catch is that a scanner is not a stateless web app: it needs a physical radio, and
a container by default sees no USB. This post is about crossing that boundary
cleanly — building the daemon as a tiny static image, then giving exactly one
RTL-SDR to the container without handing it the whole host. The pure-Go design
makes the first half almost trivial; the USB half is where the real operating
knowledge lives. See the
[Containers & Deployment]({{ '/learn/deployment/' | relative_url }}) module for
the first-principles version.*

> **TL;DR:** The `Dockerfile` is a two-stage, zero-CGO build — stage one compiles
> the daemon with `CGO_ENABLED=0` (no C toolchain, no `librtlsdr`/`libusb`), stage
> two carries just the binary on `debian:bookworm-slim` as a non-root user. The
> hard part is USB: a dongle needs **three** things to work inside a container —
> host **udev rules** granting a non-root group access to the device node, a **DVB
> blacklist** so the kernel doesn't claim it first, and **container privileges**
> that map the device (`devices:` / `--device`), join the right group
> (`group_add`), and add just `DAC_OVERRIDE` so the non-root user can open the node.
> The shipped `docker-compose.yml` does all of it — no `--privileged` required.

**Key takeaways**

- **Zero CGO makes the image tiny and the build boring.** No C libraries to install
  or link, so the runtime stage is a slim base plus one static binary and
  `ca-certificates`.
- **USB access is the host's job first.** The udev rule and DVB blacklist are host
  configuration; no container flag can substitute for them.
- **Map one device, not the whole bus.** `--device /dev/bus/usb/<bus>/<dev>` gives
  the container exactly one dongle; `/dev/bus/usb` (or `--privileged`) is the
  over-broad shortcut to avoid.
- **Least privilege still works.** `cap_drop: ALL` plus `cap_add: DAC_OVERRIDE` and
  a `group_add` is enough for a non-root container user to claim the radio — you
  never need `--privileged`.

## Cheat sheet

| Piece | Where it lives | What it does |
|---|---|---|
| Build stage | `Dockerfile` (stage 1) | `CGO_ENABLED=0 go build`, trimmed + stripped |
| Runtime stage | `Dockerfile` (stage 2) | slim base, non-root `gopher`, `ca-certificates` |
| Host udev rule | `/etc/udev/rules.d/20-rtlsdr.rules` | grants a group access to the device node |
| DVB blacklist | `/etc/modprobe.d/blacklist-dvb_usb_rtl28xxu.conf` | stops the kernel claiming the dongle |
| Device mapping | `docker-compose.yml` `devices:` | maps `/dev/bus/usb/<bus>/<dev>` in |
| Group + cap | `group_add: 46`, `cap_add: DAC_OVERRIDE` | lets the non-root user open the node |
| Liveness | compose `healthcheck` → `/api/v1/health` | orchestrator restarts a dead daemon |

## In this post

- **The zero-CGO image** — why the build is two short stages and a slim base.
- **Why a container sees no radio** — the boundary, stated plainly.
- **The three requirements** — udev, DVB blacklist, container privileges.
- **Least privilege over `--privileged`** — the capability that's actually needed.
- **Verifying the dongle crossed** — the one command that proves it worked.

## The zero-CGO image

Most SDR software drags a C toolchain and `libusb`/`librtlsdr` into its container,
which makes the image large and the build fragile. GopherTrunk's pure-Go design
means neither. The `Dockerfile` is two stages: build, then carry the binary.

```dockerfile
# Dockerfile (shape) — stage 1: pure-Go build, no C toolchain
FROM golang:1.25-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download          # cache deps before the source
COPY . .
ARG VERSION=docker
ENV CGO_ENABLED=0
RUN go build -trimpath \
        -ldflags "-s -w -X .../internal/version.Version=${VERSION}" \
        -o /out/gophertrunk ./cmd/gophertrunk

# stage 2: carry only the binary on a slim base
FROM debian:bookworm-slim AS runtime
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
 && rm -rf /var/lib/apt/lists/*
RUN useradd --system --create-home --shell /usr/sbin/nologin gopher
USER gopher
COPY --from=builder /out/gophertrunk /usr/local/bin/gophertrunk
ENTRYPOINT ["/usr/local/bin/gophertrunk"]
CMD ["run", "-config", "/etc/gophertrunk/config.yaml"]
```

Three operating properties fall out of `CGO_ENABLED=0`. The runtime image needs
**no SDR system libraries** — the daemon talks to the dongle through a pure-Go
USBDEVFS backend, so there's nothing to `apt-get install` but `ca-certificates`
(for the outbound HTTPS the broadcast feeds need). The build **caches cleanly** —
deps download in their own layer before the source is copied, so a code change
doesn't re-fetch the module cache. And the daemon runs as a **non-root user**
(`gopher`) by default, which matters the moment we talk about USB permissions:
the container user has to be *granted* device access, it doesn't have it for free.

## Why a container sees no radio

Here's the boundary, stated plainly: a container gets its own device namespace,
and by default that namespace contains no USB devices at all. The daemon inside
can enumerate all it likes and find nothing, because `/dev/bus/usb` simply isn't
there. So "run it in Docker" for a scanner is really "run it in Docker *and* thread
one physical dongle across the isolation boundary" — and that thread has three
strands, two of which are host configuration that no container flag can replace.

<figure class="lab-figure">
<svg viewBox="0 0 660 210" width="660" height="210" role="img" aria-label="A layered diagram of USB pass-through. At the bottom, the physical RTL-SDR dongle. Above it on the host: a udev rule sets the device node mode and group, and a DVB blacklist stops the kernel driver claiming the dongle. Above that, the container boundary, crossed by three things: the device mapping brings the node in, group_add joins the container user to the host group, and the DAC_OVERRIDE capability lets the non-root user open the node. At the top, the daemon's pure-Go USBDEVFS backend reads and writes the device node directly.">
  <rect x="200" y="176" width="260" height="26" rx="6" fill="none" stroke="currentColor"/>
  <text x="330" y="193" text-anchor="middle" fill="currentColor" font-size="10">RTL-SDR dongle (/dev/bus/usb/003/002)</text>
  <rect x="40" y="120" width="580" height="44" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="120" y="138" text-anchor="middle" fill="var(--fg-muted)" font-size="10">HOST</text>
  <text x="330" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="9">udev rule → node MODE/GROUP</text>
  <text x="330" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="9">DVB blacklist → kernel doesn't claim it</text>
  <line x1="330" y1="176" x2="330" y2="164" stroke="currentColor"/><polygon points="326,164 330,156 334,164" fill="currentColor"/>
  <rect x="40" y="34" width="580" height="72" rx="8" fill="none" stroke="var(--accent)"/>
  <text x="120" y="52" text-anchor="middle" fill="var(--accent)" font-size="10">CONTAINER</text>
  <rect x="90" y="60" width="140" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="160" y="80" text-anchor="middle" fill="currentColor" font-size="9">devices: mapping</text>
  <rect x="260" y="60" width="140" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="330" y="80" text-anchor="middle" fill="currentColor" font-size="9">group_add: 46</text>
  <rect x="430" y="60" width="140" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="500" y="76" text-anchor="middle" fill="currentColor" font-size="9">cap_add:</text>
  <text x="500" y="88" text-anchor="middle" fill="currentColor" font-size="9">DAC_OVERRIDE</text>
  <line x1="330" y1="120" x2="330" y2="106" stroke="var(--accent)"/><polygon points="326,106 330,98 334,106" fill="var(--accent)"/>
  <text x="330" y="24" text-anchor="middle" fill="var(--accent)" font-size="10">daemon · pure-Go USBDEVFS backend</text>
</svg>
<figcaption>Three strands cross the boundary — the device mapping, the group, and one capability — but they only work on top of host configuration: the udev rule and the DVB blacklist.</figcaption>
</figure>

## The three requirements

The [hardening guide]({{ '/hardening.html' | relative_url }}) and the
[Linux install guide]({{ '/install-linux.html' | relative_url }}) spell these out;
here they are as one operating checklist.

**1. Host udev rules.** The device node has to be openable by the group the
container user will join. On the host:

```
# /etc/udev/rules.d/20-rtlsdr.rules
SUBSYSTEM=="usb", ATTRS{idVendor}=="0bda", ATTRS{idProduct}=="2838", \
    MODE="0660", GROUP="plugdev"
```

then `sudo udevadm control --reload && sudo udevadm trigger` and replug the
dongle. This is *host* config — it sets the permissions on the node before it's
ever mapped into a container.

**2. DVB blacklist.** An RTL-SDR presents as a DVB-T TV tuner, and the kernel's
`dvb_usb_rtl28xxu` driver will grab it on plug-in. Stop that:

```
# /etc/modprobe.d/blacklist-dvb_usb_rtl28xxu.conf
blacklist dvb_usb_rtl28xxu
```

The daemon auto-detaches the DVB driver at open time, so this is belt-and-braces —
but one less moving part, and on a headless box you want the device clean before
the container ever touches it.

**3. Container privileges.** Now, and only now, the container flags matter. The
shipped `docker-compose.yml` maps exactly one device, joins the host group, and
adds exactly one capability:

```yaml
# docker-compose.yml (shape)
services:
  gophertrunk:
    restart: unless-stopped
    devices:
      - "/dev/bus/usb/003/002:/dev/bus/usb/003/002"  # match your lsusb
    group_add:
      - "46"          # plugdev GID on Debian — check getent group plugdev
    cap_drop:
      - ALL
    cap_add:
      - DAC_OVERRIDE  # let the non-root user open the device node
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--spider", "http://127.0.0.1:8080/api/v1/health"]
      interval: 30s
```

`devices:` maps the specific node in; `group_add: 46` puts the container's non-root
user in the same `plugdev` group your udev rule granted; `cap_add: DAC_OVERRIDE`
lets that user open the node. That's the complete recipe.

### How that principle shaped the Go code

- **The USBDEVFS backend needs no kernel driver.** The daemon reads and writes
  `/dev/bus/usb/<bus>/<dev>` directly, which is exactly why mapping *that node* — not
  installing a driver in the container — is all it takes. The pure-Go transport and
  the container recipe are two sides of the same design choice.
- **Non-root by default forces the permission story to be explicit.** Because the
  image runs as `gopher`, not root, you can't accidentally rely on root's blanket
  device access; the udev group + `DAC_OVERRIDE` path is the only way in, and it's
  the least-privilege one.
- **`ca-certificates` is the only runtime dependency.** It's there for the outbound
  HTTPS the broadcast feeds (Parts 9–11) need — the SDR path pulls in nothing.

## Least privilege over `--privileged`

The temptation, when a dongle won't show up, is to reach for `--privileged` or to
map all of `/dev/bus/usb`. Both work, and both are wrong for a service that runs
for months: `--privileged` hands the container the entire host device space and
most capabilities, turning a scanner into a much larger blast radius than a scanner
needs. The recipe above proves you don't have to. A `cap_drop: ALL` container with
a single `DAC_OVERRIDE` and one mapped node can claim the radio and do nothing
else — the daemon inside can't touch another device, can't load a module, can't
escalate. The broader `/dev/bus/usb` mapping is only for cases where the dongle's
bus/device path shifts on every replug and you can't pin it; even then, prefer a
udev `SYMLINK` and map the symlink over opening the whole bus.

One more operating wrinkle worth pre-loading: USB **autosuspend**. On a headless
host the kernel may power-manage an idle-looking dongle mid-stream, which the
daemon sees as `usb: device disconnected`. Pin it on the host with
`echo on > /sys/bus/usb/devices/<id>/power/control` (or a udev `TEST` rule) so a
long-running container stream isn't cut by power management it can't see.

## Verifying the dongle crossed

After `docker compose up -d`, one command settles whether all three strands
landed — you run the daemon's own SDR enumeration *inside* the container:

```bash
docker exec gophertrunk gophertrunk sdr list
# should print the dongle: index, serial, tuner type, supported gains
curl -s http://localhost:8080/api/v1/health   # daemon serving
```

If `sdr list` shows the dongle, the whole chain is good: host permissions, device
mapping, group, and capability. If it prints nothing, work the layers in order —
check `dmesg` on the host for a DVB driver claim (blacklist didn't take),
`ls -l /dev/bus/usb/...` from *inside* the container (mapping or udev permission
wrong), and `getent group plugdev` against your `group_add` GID (group mismatch).
The compose `healthcheck` closes the loop for the orchestrator: it hits
`/api/v1/health` every 30 seconds, so a daemon that wedges gets restarted by
`restart: unless-stopped` without you watching — the containerized equivalent of
the watchdog we reach in the finale.

## Where this goes next

Containers are one way to run a service; a native system daemon is the other.
[Part 13]({{ '/blog/deep-dives/running-it-for-real-13-systemd-windows/' | relative_url }})
takes the same daemon to a hardened systemd unit — the sandboxing directives that
box in a native process the way a container isolates a containerized one,
including the `DeviceAllow` line that's the systemd analogue of everything we just
did — and to the Windows service, installer, and launcher for operators not on
Linux at all.

## FAQ

**Do I need `libusb` or `librtlsdr` in the image?**
No. The build is `CGO_ENABLED=0` and the daemon talks to the dongle through a
pure-Go USBDEVFS backend, so the runtime stage carries only the static binary and
`ca-certificates`. That's the whole reason the image is small and the USB recipe is
"map a device node," not "install a driver."

**Why isn't `--privileged` the simplest answer?**
It works but it's the wrong trade for a months-long service — it grants the container
the entire host device space and nearly every capability. A `cap_drop: ALL` container
with one mapped node, a `group_add`, and `DAC_OVERRIDE` claims exactly one radio and
nothing else. Least privilege costs three lines of config and bounds the blast radius.

**My container's `sdr list` is empty — where do I start?**
Layer by layer. `dmesg` on the host for a DVB driver claim (blacklist), `ls -l
/dev/bus/usb/...` inside the container (device mapping + udev mode), and `getent group
plugdev` against your `group_add` GID (group match). One of those three is almost
always the culprit; the udev rule and blacklist are host config no flag can replace.

**The stream drops after a while with `device disconnected` — why?**
USB autosuspend. The kernel power-manages a dongle it thinks is idle, which the daemon
sees as a disconnect. Pin power on the host with `echo on >
/sys/bus/usb/devices/<id>/power/control` or a udev rule so a long-running container
stream isn't cut by power management inside a namespace it can't reach.

**How does the orchestrator know the daemon is alive?**
The compose `healthcheck` polls `/api/v1/health` every 30 seconds; a failing probe
plus `restart: unless-stopped` gets the container restarted automatically. That
endpoint reports more than "process up" — attached SDR count, active calls, DB
connectivity — which the finale (Part 14) uses to distinguish "running" from
"actually working."

## Series navigation

**Part 12 of 14** · ←
[Part 11: Grant Webhooks & External Integrations]({{ '/blog/deep-dives/running-it-for-real-11-grant-webhooks/' | relative_url }})
· Next →
[Part 13: systemd Hardening & the Windows Installer]({{ '/blog/deep-dives/running-it-for-real-13-systemd-windows/' | relative_url }})
