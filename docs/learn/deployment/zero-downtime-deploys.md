---
slug: zero-downtime-deploys
title: Zero-downtime deploys
description: "Health-gated rollouts, draining connections, and rollback — how to replace a running version without dropping a single request."
keywords: zero downtime deployment, connection draining, graceful shutdown, health gated rollout, rollback, rolling update, blue-green, no downtime deploy
level: advanced
status: full
prereq:
  - scaling-and-load-balancing
faq:
  - q: How do you deploy without downtime?
    a: "Bring the new version up alongside the old, wait until it reports healthy before sending it traffic, then drain the old version — let its in-flight requests finish while the load balancer stops sending it new ones — and only then stop it. Rolling and blue-green strategies both do this. On a single instance you can't fully avoid a gap, but a fast graceful restart keeps it to a moment."
  - q: What is connection draining?
    a: "Draining means telling the load balancer to stop sending an instance new requests while letting the requests it's already handling finish. Without draining, stopping an old instance kills its in-flight requests and users see errors. With draining, the old instance quietly finishes its work and then exits cleanly, so no request is dropped during the swap."
---

# Zero-downtime deploys

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Replacing a running version without dropping requests has three moving parts: bring the
new version up and **gate it on a health check** before it takes traffic; **drain**
connections from the old version so its in-flight requests finish; and keep the ability
to **roll back** instantly if the new version misbehaves. Multi-instance setups do this
seamlessly; a single instance uses a brief graceful restart.
</div>

[Release strategies](/learn/deployment/release-strategies/) named the patterns — rolling,
blue-green, canary. This lesson is the *mechanics* that make any of them lose zero
requests: health gating, draining, and rollback. They lean directly on the
[health checks](/learn/deployment/logging-and-health-checks/) and
[load balancer](/learn/deployment/scaling-and-load-balancing/) you already have.

## Gate the rollout on health

The cardinal rule: **never send traffic to a version that hasn't proven it's ready.** A
new instance starts, but it might still be loading config, opening its database, or
warming caches. So the [load balancer](/learn/deployment/scaling-and-load-balancing/)
waits until the instance passes its [health check](/learn/deployment/logging-and-health-checks/)
— GopherTrunk exposes exactly this endpoint — before routing anything to it:

```bash
# poll the new instance until it's healthy, then it joins rotation
until curl -fsS http://127.0.0.1:8081/api/v1/health >/dev/null; do sleep 1; done
echo "new instance healthy — adding to the load balancer"
```

If the new version never goes healthy, it never gets traffic, and users stay on the good
old version. The health check is the gate that makes a rollout safe.

## Drain the old version

Once the new version is serving, retire the old one *gracefully*. **Draining** tells the
load balancer to stop sending the old instance new requests while letting its current
ones finish. The app cooperates by handling the shutdown signal — finishing in-flight
work before exiting rather than dying mid-request:

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 96" role="img" aria-label="Three phases: load balancer sends to old and new, then stops new requests to old while it finishes in-flight ones, then old exits." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
  <text x="78" y="14">1. both serve</text>
  <rect x="40" y="24" width="76" height="24" rx="3" fill="none" stroke="currentColor"/><text x="78" y="39">old + new</text>
  <text x="235" y="14">2. drain old</text>
  <rect x="190" y="24" width="90" height="24" rx="3" fill="none" stroke="currentColor" stroke-dasharray="4 3"/><text x="235" y="35">old finishes</text><text x="235" y="45" font-size="7">in-flight only</text>
  <text x="400" y="14">3. old exits</text>
  <rect x="360" y="24" width="80" height="24" rx="3" fill="none" stroke="currentColor" stroke-width="1.7"/><text x="400" y="39">new only</text>
  </g>
  <g stroke="currentColor" fill="none"><line x1="116" y1="36" x2="190" y2="36"/><line x1="280" y1="36" x2="360" y2="36"/></g>
</svg>
<figcaption>Draining lets the old version finish its in-flight requests before it stops, so nothing is dropped.</figcaption>
</figure>

A well-behaved server catches `SIGTERM`, stops accepting new connections, finishes what's
in flight, then exits — "graceful shutdown." Orchestrators and load balancers give it a
grace period to do so.

## Always be able to roll back

Even a health-gated, drained rollout can ship a subtle bug that only shows under real
traffic. So keep the previous [versioned artifact](/learn/deployment/build-artifacts-and-versioning/)
one command away:

```bash
# roll straight back to the known-good version
docker compose pull && docker tag ghcr.io/example/app:1.4.1 ghcr.io/example/app:current
docker compose up -d              # recreate on the previous image
```

This is why you version and keep images — rollback is just deploying the *last* good one.
[Blue-green](/learn/deployment/release-strategies/) makes rollback instant (flip back to
blue); rolling makes it a reverse rollout. Either way, "how do I undo this?" must have an
answer *before* you deploy.

## The single-instance reality

All of the above assumes multiple instances. One GopherTrunk instance bound to one USB
radio can't run two copies of itself — the radio is a single physical resource. So the
honest answer for a single instance is a **brief graceful restart**: `docker compose up -d`
recreates the container on the new image, and `systemctl restart` does the same for the
systemd unit, each a moment's gap while the process swaps. That's a perfectly fine
tradeoff for a single-user scanner — [recreate](/learn/deployment/release-strategies/)
with a short, honest downtime beats pretending you have a fleet you don't.

<div class="knowledge-check" data-quiz data-correct-msg="Right — draining lets in-flight requests finish before the old instance stops, dropping none." markdown="0">
  <p class="knowledge-check__q">Quick check: what does connection draining accomplish during a deploy?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It speeds up the new version's startup</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It lets the old instance finish in-flight requests before it stops, dropping none</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It encrypts traffic between instances</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Gate** the rollout on a [health check](/learn/deployment/logging-and-health-checks/) —
  no traffic until the new version proves it's ready.
- **Drain** the old version: stop new requests to it, let in-flight ones finish, then
  exit gracefully on `SIGTERM`.
- Keep **rollback** one command away by versioning and retaining images.
- A **single instance** can't be truly zero-downtime; a fast graceful restart (`compose
  up -d`, `systemctl restart`) is the honest, fine choice.

Next up: keeping a deployment healthy over time — monitoring and updates.
