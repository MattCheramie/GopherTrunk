---
slug: docker-compose
title: Multi-container apps with Compose
description: Describe an app and its dependencies as one file — services, networks, and volumes — and bring the whole stack up with a single command, using GopherTrunk's real compose file.
keywords: docker compose, docker-compose.yml, compose services, docker volumes, docker networks, multi-container, compose up, container orchestration
level: intermediate
status: full
prereq:
  - images-and-registries
faq:
  - q: What is Docker Compose for?
    a: Compose describes one or more containers, plus their networks and storage, in a single YAML file, so you can start the whole set with one command instead of many long docker run lines. It's ideal for an application and its dependencies — a service, a database, a cache — running together on one host.
  - q: What is a volume in Docker?
    a: A container's own filesystem is temporary — delete the container and its data is gone. A volume maps a directory into the container that persists on the host, so data like a database file or recordings survives restarts and upgrades. GopherTrunk uses volumes for its config, recordings, and call database.
---

# Multi-container apps with Compose

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Docker Compose** describes an app — its **services**, **networks**, and persistent
**volumes** — in one `docker-compose.yml` file, so `docker compose up` starts the whole
stack. **Volumes** keep data alive across restarts. GopherTrunk ships a real compose
file that maps its config, recordings, database, USB device, and ports.
</div>

Running one container by hand is fine; running an app with its config, storage, and
device access every time is tedious. **Compose** captures all of it in a file.

## One file for the whole stack

A single `docker run` command with all its flags gets unwieldy fast. Compose moves that
into a declarative `docker-compose.yml`, then you manage everything with:

```bash
docker compose up -d      # start the whole stack in the background
docker compose logs -f    # follow its logs
docker compose down       # stop and remove it
```

## GopherTrunk's compose file, annotated

Here's the core of GopherTrunk's actual compose file:

```yaml
services:
  gophertrunk:
    image: ghcr.io/mattcheramie/gophertrunk:latest
    build:                       # build locally from the repo Dockerfile
      context: .
      args:
        VERSION: "compose-dev"
    container_name: gophertrunk
    restart: unless-stopped      # restart on crash or reboot

    ports:
      - "127.0.0.1:8080:8080"    # HTTP API, bound to localhost only
      - "127.0.0.1:50051:50051"  # gRPC

    volumes:
      - ./config.yaml:/etc/gophertrunk/config.yaml:ro
      - ./recordings:/var/lib/gophertrunk/recordings
      - ./calls.db:/var/lib/gophertrunk/calls.db
```

Every key maps to a concept from this module:

- **`image`** / **`build`** — use a [registry image](/learn/deployment/images-and-registries/)
  or build locally from the [Dockerfile](/learn/deployment/writing-a-dockerfile/).
- **`restart: unless-stopped`** — process management, so the container comes back after
  a crash or reboot (the [systemd lesson](/learn/deployment/services-and-systemd/)
  does the same for a plain binary).
- **`ports`** — map container ports to the host; binding to `127.0.0.1` keeps them off
  the public network (a [security](/learn/deployment/secrets-and-configuration/)
  default).
- **`volumes`** — persist the config (read-only), recordings, and call database on the
  host so they survive the container being replaced.

## Volumes: where your data lives

The volumes line is the one to internalize. A container's own filesystem vanishes when
the container is removed — which happens on every upgrade. **Volumes** map host
directories into the container so the data that matters *outlives* any single container.
GopherTrunk keeps its recordings and `calls.db` in volumes for exactly this reason: pull
a new image, recreate the container, and your data is still there.

## Hardware and permissions

Because GopherTrunk talks to a USB radio, its compose file also passes the device
through (`devices:`), grants the right group (`group_add`), and drops all Linux
capabilities except the one needed to open the device (`cap_drop: ALL` + `cap_add:
DAC_OVERRIDE`) — a nice example of giving a container *exactly* the access it needs and
nothing more. You'll see this same least-privilege spirit in the
[systemd unit](/learn/deployment/services-and-systemd/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — a volume persists data on the host so it survives the container being replaced." markdown="0">
  <p class="knowledge-check__q">Quick check: why does GopherTrunk store its recordings and database in volumes?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">To make the image smaller</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">So the data survives when the container is replaced on upgrade</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Because containers cannot write files otherwise</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Compose** describes services, networks, and volumes in one file; `docker compose
  up` runs the stack.
- **`restart: unless-stopped`** brings containers back after crashes and reboots.
- **Volumes** persist data on the host so it **survives** container replacement.
- GopherTrunk's compose maps its config, recordings, database, ports, and USB device —
  with least-privilege capabilities.

Next up: Unit 3 — running a plain binary as a managed service with systemd.
