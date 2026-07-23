---
slug: container-networking-and-volumes
title: Container networking & volumes
description: "How containers get an IP, talk to each other by name, publish ports, and persist data — the networking and storage model beneath a Compose file."
keywords: docker networking, docker volumes, bridge network, container dns, publish ports, named volume, bind mount, data persistence, docker network
level: intermediate
status: full
prereq:
  - docker-compose
faq:
  - q: How do containers talk to each other?
    a: "Containers on the same Docker network can reach each other by service name — Docker runs an internal DNS so the name 'db' resolves to that container's IP. You don't hard-code IP addresses; you use the service names from your Compose file. Containers on different networks can't see each other unless you connect them, which is how you isolate a database from the outside world."
  - q: What is the difference between a named volume and a bind mount?
    a: "A bind mount maps a specific host directory into the container — you control the exact path, good for config files and code you edit. A named volume is managed by Docker in its own storage area, referenced by a name — better for data like databases where you don't care about the host path, just that it persists. Both survive the container being removed."
---

# Container networking & volumes

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Every container joins a **network** — usually a **bridge** — where Docker gives it an IP
and lets containers reach each other by **service name** through built-in DNS.
**Publishing a port** maps a container port to the host so the outside world can reach
it. A container's own filesystem is temporary; a **volume** — named or a bind mount —
persists data on the host so it survives the container being replaced.
</div>

The [Compose lesson](/learn/deployment/docker-compose/) showed `ports:` and `volumes:`
lines without explaining the machinery under them. This lesson is that machinery: how
containers get on the network, find each other, expose ports, and keep data alive.

## Every container joins a network

When Docker starts a container it attaches it to a **network** and hands it a private IP.
By default that's a **bridge network** — a virtual switch on the host. Compose creates
one network per project automatically, so all the services in a `docker-compose.yml`
land on the same bridge and can talk to each other. Containers on *different* networks
can't see one another, which is how you keep, say, a database off the public side.

```bash
docker network ls                    # list networks
docker network inspect gophertrunk_default   # see who's attached
```

## Containers find each other by name

The key trick: on a Docker network, you never hard-code IP addresses. Docker runs an
internal **DNS** so a container can reach another by its **service name**. If your
Compose file has a `db` service, the app container just connects to the host `db`:

```yaml
services:
  app:
    environment:
      DATABASE_URL: "postgres://db:5432/app"   # "db" resolves to the db container
  db:
    image: postgres:16
```

Docker resolves `db` to that container's current IP, even if it changes on restart.
Names are stable; IPs are not — so you always use names.

## Publishing ports exposes a container to the host

Container-to-container traffic stays on the internal network. To let something *outside*
reach a container — your browser, another machine — you **publish** a port, mapping a
host port to a container port. GopherTrunk publishes both its ports but binds them to
localhost only:

```yaml
ports:
  - "127.0.0.1:8080:8080"    # host 127.0.0.1:8080 -> container 8080
  - "127.0.0.1:50051:50051"  # gRPC, localhost only
```

The `127.0.0.1:` prefix is the important detail: without it, `- "8080:8080"` binds to
**every** host interface, exposing the API to the whole network. Binding to `127.0.0.1`
keeps it reachable only from the host itself — you then put a
[reverse proxy](/learn/deployment/reverse-proxies-and-tls/) in front to expose it safely.

<figure class="figure" markdown="0">
<svg viewBox="0 0 480 150" role="img" aria-label="A host with a published port forwarding to a container, and two containers on a bridge network talking by name." xmlns="http://www.w3.org/2000/svg">
  <rect x="6" y="6" width="468" height="138" rx="6" fill="none" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="16" y="22" font-size="9" fill="currentColor">host</text>
  <g font-size="9" fill="currentColor" text-anchor="middle">
  <rect x="20" y="55" width="90" height="28" rx="4" fill="none" stroke="currentColor"/><text x="65" y="73">127.0.0.1:8080</text>
  <rect x="200" y="35" width="110" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.7"/><text x="255" y="54">app</text>
  <rect x="200" y="95" width="110" height="30" rx="4" fill="none" stroke="currentColor"/><text x="255" y="114">db</text>
  </g>
  <rect x="180" y="20" width="270" height="118" rx="6" fill="none" stroke="currentColor" stroke-dasharray="4 3" stroke-opacity="0.6"/>
  <text x="440" y="34" font-size="8" fill="currentColor" text-anchor="end">bridge network</text>
  <g stroke="currentColor" fill="none"><line x1="110" y1="69" x2="200" y2="55"/><line x1="255" y1="65" x2="255" y2="95"/></g>
  <text x="268" y="84" font-size="8" fill="currentColor">"db"</text>
</svg>
<figcaption>A published port forwards from the host to a container; on the bridge network, containers reach each other by service name.</figcaption>
</figure>

## Volumes: data that outlives the container

A container's writable filesystem is **ephemeral** — remove the container (which happens
on every upgrade) and everything it wrote is gone. A **volume** maps storage that lives
on the host, so data survives. There are two kinds:

| | Bind mount | Named volume |
|---|-----------|--------------|
| You write | `./recordings:/var/lib/...` | `recdata:/var/lib/...` |
| Location | A host path you choose | Docker-managed storage |
| Best for | Config, code, files you edit | Databases, data you don't hand-edit |

GopherTrunk uses **bind mounts** so the operator can see and edit the files directly:

```yaml
volumes:
  - ./config.yaml:/etc/gophertrunk/config.yaml:ro   # read-only config
  - ./recordings:/var/lib/gophertrunk/recordings    # recordings persist
  - ./calls.db:/var/lib/gophertrunk/calls.db        # call database persists
```

The `:ro` on the config makes it **read-only** inside the container — the app can read
its config but can't overwrite it. Pull a new image, recreate the container, and the
recordings and database are still there because they live on the host, not in the
container. That persistence is the whole subject of
[backups & data](/learn/deployment/backups-and-data/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — Docker's internal DNS resolves service names to container IPs on the same network." markdown="0">
  <p class="knowledge-check__q">Quick check: how does one container reach another on the same Docker network?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">By its hard-coded IP address</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">By its service name, which Docker's internal DNS resolves</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Only through the host's published ports</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Containers join a **bridge network** and get a private IP; Compose makes one per
  project.
- On a network, containers reach each other by **service name** via Docker's DNS — never
  hard-code IPs.
- **Publishing a port** maps a container port to the host; binding to `127.0.0.1` keeps
  it off the public network.
- A **volume** (bind mount or named) persists data on the host so it survives the
  container being replaced; `:ro` makes a mount read-only.

Next up: giving a container exactly the access it needs — container security.
