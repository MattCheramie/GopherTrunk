---
slug: logging-and-health-checks
title: Logging & health checks
description: How a running service tells you it's healthy — structured logs, health endpoints, and the signals an orchestrator uses to restart what's broken.
keywords: logging, structured logs, health check, health endpoint, liveness, readiness, journald, monitoring, observability
level: intermediate
status: full
prereq:
  - services-and-systemd
faq:
  - q: What is a health check endpoint?
    a: It's a small HTTP endpoint — often /health — that a service exposes to report whether it's working. A monitoring tool or orchestrator polls it periodically; a healthy response means keep running, a failure or timeout means the service is stuck and should be restarted or taken out of rotation.
  - q: What are structured logs?
    a: Structured logs record each event as machine-readable fields (like key/value pairs or JSON) rather than free-form text. That makes them searchable and filterable — you can pull every error for a given call or timeframe — instead of grepping unstructured lines. They're far easier to work with once a system is running at scale.
---

# Logging & health checks

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A running service has to tell you it's alive. **Logs** record what it's doing —
**structured** logs (fields, not just text) are searchable. A **health check** endpoint
lets a monitor poll "are you OK?"; a bad answer triggers a **restart** or removal from
rotation. Together they're how you *know* a deployment is healthy without watching it.
</div>

Once a service runs unattended, you can't watch it by staring at a terminal. Logging
and health checks are how it reports its own state.

## Logs: the running narrative

Logs are the diary of a running service — what it did, what went wrong, when. Under
systemd, a service's output is captured by the **journal**, which you read with:

```bash
journalctl -u gophertrunk -f      # follow GopherTrunk's live logs
```

For a container, the equivalent is `docker logs -f gophertrunk`. Either way, the logs
are your first stop when something misbehaves.

**Structured** logs — events recorded as fields rather than free-form sentences — are
much easier to work with as a system grows: you can filter for every error, every event
on a given channel, every entry in a time window, instead of grepping prose. (The
[Linux CLI module](/learn/linux-cli/monitoring-and-logs/) covers reading logs in depth.)

## Health checks: a heartbeat you can poll

A log tells you what happened; a **health check** answers "are you working *right
now*?" The service exposes a small endpoint that returns OK when healthy:

```bash
$ curl http://127.0.0.1:8080/api/v1/health
{"status":"ok"}
```

GopherTrunk exposes exactly this at `/api/v1/health`, and its
[docker-compose](/learn/deployment/docker-compose/) file wires Docker to poll it:

```yaml
healthcheck:
  test: ["CMD", "wget", "--quiet", "--spider", "http://127.0.0.1:8080/api/v1/health"]
  interval: 30s
  timeout: 5s
  retries: 3
```

Every 30 seconds Docker checks the endpoint; three failures in a row mark the container
unhealthy, and a restart policy or orchestrator can then act.

## Liveness vs readiness

Two subtly different questions a health check can answer:

| Check | Asks | If it fails |
|-------|------|-------------|
| **Liveness** | Is the process alive and not stuck? | restart it |
| **Readiness** | Is it ready to serve requests yet? | hold traffic until ready |

Small deployments often use a single health endpoint for both; larger systems separate
them so a service that's *starting up* isn't killed for not being ready yet.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 100" role="img" aria-label="A monitor repeatedly polling a service's health endpoint; a healthy reply keeps it running, a failed reply triggers a restart." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="35" width="90" height="30" rx="5" fill="none" stroke="currentColor"/><text x="65" y="54" text-anchor="middle" font-size="9" fill="currentColor">monitor</text>
  <line x1="110" y1="44" x2="200" y2="44" stroke="currentColor" stroke-width="1.4" marker-end="url(#h1)"/><text x="155" y="38" font-size="8" fill="currentColor">GET /health</text>
  <rect x="200" y="35" width="90" height="30" rx="5" fill="none" stroke="currentColor"/><text x="245" y="54" text-anchor="middle" font-size="9" fill="currentColor">service</text>
  <line x1="200" y1="58" x2="110" y2="58" stroke="currentColor" stroke-width="1.4" marker-end="url(#h1)"/><text x="155" y="72" font-size="8" fill="currentColor">ok / fail</text>
  <line x1="290" y1="50" x2="360" y2="35" stroke="currentColor" stroke-width="1.2" stroke-dasharray="3 2" marker-end="url(#h1)"/>
  <text x="430" y="40" font-size="8" fill="currentColor">fail x3 &#8594; restart</text>
  <defs><marker id="h1" markerWidth="7" markerHeight="7" refX="5" refY="3" orient="auto"><path d="M0 0 L5 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A monitor polls the health endpoint; repeated failures trigger an automatic restart or traffic removal.</figcaption>
</figure>

## Metrics, briefly

Beyond a yes/no health check, many services also expose **metrics** — numeric gauges
and counters (requests handled, errors, memory) at an endpoint like `/metrics` for a
tool such as Prometheus to scrape. GopherTrunk labels its compose service for exactly
this. Metrics turn "is it up?" into "how is it *doing*?" — the subject of
[monitoring & updates](/learn/deployment/monitoring-and-updates/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — a health endpoint lets a monitor detect a stuck service and restart it." markdown="0">
  <p class="knowledge-check__q">Quick check: what happens when a service repeatedly fails its health check?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Its logs are deleted</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It's restarted or taken out of rotation</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Its version number increases</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Logs** are a service's running narrative — read them via `journalctl` or `docker
  logs`; **structured** logs are searchable.
- A **health check** endpoint lets a monitor poll "are you OK?" — GopherTrunk exposes
  `/api/v1/health`.
- **Liveness** (alive?) and **readiness** (ready to serve?) are distinct questions.
- **Metrics** at `/metrics` answer "how is it doing?" beyond a simple up/down.

Next up: keeping credentials safe — secrets and configuration management.
