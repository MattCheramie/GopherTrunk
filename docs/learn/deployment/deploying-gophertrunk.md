---
slug: deploying-gophertrunk
title: Deploying GopherTrunk end to end
description: A complete worked example — container image, docker-compose stack, and a systemd service — taking GopherTrunk from a repo to a running scanner on a server.
keywords: deploy gophertrunk, gophertrunk docker, gophertrunk systemd, sdr deployment, end to end deployment, docker compose deploy, run gophertrunk server
level: advanced
status: full
prereq:
  - docker-compose
  - services-and-systemd
  - secrets-and-configuration
faq:
  - q: Do I need Docker to run GopherTrunk in production?
    a: No — you can run it either way. Docker plus docker-compose is the quickest path and keeps everything contained, but GopherTrunk also ships a systemd unit so you can run the plain binary as a managed service. This lesson shows both, since which you choose is a matter of preference and environment.
  - q: Why does deploying GopherTrunk involve USB device access?
    a: GopherTrunk reads I/Q samples from a physical SDR dongle over USB, so the running service — whether a container or a systemd unit — needs permission to open that USB device. Both GopherTrunk's compose file and its systemd unit include the device pass-through and group settings that grant exactly that access and nothing more.
---

# Deploying GopherTrunk end to end

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
This capstone deploys GopherTrunk for real, two ways: as a **docker-compose** stack, and
as a **systemd** service running the binary. Both pull together everything in the module
— a versioned [artifact](/learn/deployment/build-artifacts-and-versioning/), a
[config file](/learn/deployment/environments-and-config/), USB **device access**,
persistent [volumes](/learn/deployment/docker-compose/), a
[health check](/learn/deployment/logging-and-health-checks/), and
[updates with rollback](/learn/deployment/monitoring-and-updates/).
</div>

Time to put it all together. This lesson walks GopherTrunk from a fresh repo to a
running scanner on a server — the whole module, applied.

## What we're deploying

GopherTrunk is a long-running daemon that reads a USB SDR dongle, decodes trunked
radio, and serves a web API and gRPC interface. A real deployment therefore needs: the
binary or image, a config file, access to the USB device, somewhere to persist
recordings and the call database, and process management so it survives crashes and
reboots. We'll do it with containers first, then the systemd alternative.

## Path A — docker-compose

**1. Get the repo and build the image.** The [compose file](/learn/deployment/docker-compose/)
builds locally from the repo [Dockerfile](/learn/deployment/writing-a-dockerfile/):

```bash
git clone https://github.com/mattcheramie/gophertrunk
cd gophertrunk
docker compose build          # builds the image from the Dockerfile
```

**2. Provide config and find your dongle.** Copy the example config and edit it, then
find the USB device to pass through:

```bash
cp config.example.yaml config.yaml   # then edit device serials, talkgroups, etc.
lsusb                                # e.g. "Bus 003 Device 002: ID 0bda:2838"
```

Set the matching `devices:` path in `docker-compose.yml` (e.g.
`/dev/bus/usb/003/002`), and make sure `group_add` matches the group your host's udev
rules grant (commonly `plugdev`, GID 46 on Debian).

**3. Start the stack.**

```bash
docker compose up -d          # start in the background
docker compose logs -f        # watch it come up
```

The compose file already sets `restart: unless-stopped` (survives crashes/reboots),
maps the config read-only, persists `recordings/` and `calls.db` in
[volumes](/learn/deployment/docker-compose/), binds the API to `127.0.0.1` only, and
drops all Linux capabilities except `DAC_OVERRIDE` for the USB device.

**4. Verify health.**

```bash
curl http://127.0.0.1:8080/api/v1/health    # -> {"status":"ok"}
```

That's the same [health endpoint](/learn/deployment/logging-and-health-checks/) Docker
polls automatically every 30 seconds.

## Path B — systemd (the binary)

Prefer running the plain [binary](/learn/programming-go/hello-go/) as a
[service](/learn/deployment/services-and-systemd/)? GopherTrunk ships a hardened unit:

```bash
# build the binary (or download a release artifact)
go build -o bin/gophertrunk ./cmd/gophertrunk

# install the unit, binary, and config
sudo install -m 0644 docs/gophertrunk.service /etc/systemd/system/gophertrunk.service
sudo install -m 0755 bin/gophertrunk /usr/local/bin/gophertrunk
sudo install -d -m 0755 /etc/gophertrunk
sudo install -m 0640 config.example.yaml /etc/gophertrunk/config.yaml
# edit /etc/gophertrunk/config.yaml

sudo systemctl daemon-reload
sudo systemctl enable --now gophertrunk      # start now and on every boot
```

The unit handles the rest: `Restart=on-failure` for crash resilience, `DeviceAllow` +
`SupplementaryGroups=plugdev` for USB access, and a stack of
[hardening](/learn/cybersecurity/hardening-systems/) directives (`DynamicUser`,
`ProtectSystem=strict`, `NoNewPrivileges`) that sandbox the service. Tail its logs with:

```bash
journalctl -u gophertrunk -f
```

## Handling the API token (a secret)

If you enable the API's auth, keep the token out of `config.yaml`. Put it in a
locked-down file and reference it — the [secrets](/learn/deployment/secrets-and-configuration/)
pattern GopherTrunk supports:

```ini
# in the systemd unit:
EnvironmentFile=-/etc/gophertrunk/env
# and in config.yaml: api.auth.token_file: /etc/gophertrunk/token
```

## Updating and rolling back

To move to a new [version](/learn/deployment/build-artifacts-and-versioning/):

```bash
# containers:
docker compose pull && docker compose up -d
# systemd: install the new binary, then
sudo systemctl restart gophertrunk
```

Watch the health check and logs after the update. If the new version misbehaves,
[roll back](/learn/deployment/monitoring-and-updates/) by redeploying the previous
image tag or reinstalling the previous binary — the old artifact still exists.

## You've deployed it

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 100" role="img" aria-label="An SDR dongle connects over USB to a server running GopherTrunk as a container or systemd service, which serves an API to a user on localhost." xmlns="http://www.w3.org/2000/svg">
  <rect x="10" y="40" width="70" height="28" rx="4" fill="none" stroke="currentColor"/><text x="45" y="58" text-anchor="middle" font-size="8.5" fill="currentColor">SDR dongle</text>
  <line x1="80" y1="54" x2="120" y2="54" stroke="currentColor" stroke-width="1.4"/><text x="100" y="48" text-anchor="middle" font-size="7" fill="currentColor">USB</text>
  <rect x="120" y="24" width="180" height="60" rx="5" fill="none" stroke="currentColor" stroke-width="1.7"/><text x="210" y="42" text-anchor="middle" font-size="9" fill="currentColor">server</text>
  <rect x="140" y="50" width="140" height="24" rx="4" fill="none" stroke="currentColor"/><text x="210" y="66" text-anchor="middle" font-size="8" fill="currentColor">GopherTrunk (container/systemd)</text>
  <line x1="300" y1="54" x2="360" y2="54" stroke="currentColor" stroke-width="1.4"/><text x="330" y="48" text-anchor="middle" font-size="7" fill="currentColor">:8080</text>
  <rect x="360" y="40" width="70" height="28" rx="4" fill="none" stroke="currentColor"/><text x="395" y="58" text-anchor="middle" font-size="8.5" fill="currentColor">web API</text>
</svg>
<figcaption>The finished deployment: a dongle feeds GopherTrunk on a server, managed by Docker or systemd, serving its API.</figcaption>
</figure>

That's a complete, production-shaped deployment — build, configure, run, secure,
monitor, update — using every idea in this module on a real application.

<div class="knowledge-check" data-quiz data-correct-msg="Right — GopherTrunk can run as a docker-compose stack or a systemd-managed binary." markdown="0">
  <p class="knowledge-check__q">Quick check: what two ways does this lesson deploy GopherTrunk?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Only as a Docker container</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">As a docker-compose stack, or as a systemd-managed binary</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Only on a managed cloud platform</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- GopherTrunk deploys as a **docker-compose** stack (`build`, config, USB device,
  volumes, health check) or a **systemd** service running the binary.
- Both grant **least-privilege USB access** and provide **crash/reboot resilience**.
- Keep the API token as a **secret** in a separate file, not in `config.yaml`.
- **Update** by pulling/reinstalling the new version and watching health; **roll back**
  to the previous artifact if needed.

That's the module — from source to a running, monitored, updatable service. Keep the
[glossary](/learn/deployment/glossary/) handy as a reference.
