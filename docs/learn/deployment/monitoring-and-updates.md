---
slug: monitoring-and-updates
title: Monitoring & updates
description: Keeping a deployment healthy over time — metrics and alerts, rolling out new versions safely, and rolling back when a release goes wrong.
keywords: monitoring, metrics, alerts, prometheus, rolling update, rollback, deploy new version, uptime, observability, safe deployment
level: intermediate
status: full
prereq:
  - logging-and-health-checks
  - build-artifacts-and-versioning
faq:
  - q: What is the difference between logs, metrics, and alerts?
    a: Logs are a detailed record of individual events. Metrics are numbers measured over time — request rate, error count, memory use — good for spotting trends. Alerts are automatic notifications when a metric crosses a threshold, like errors spiking or the service going down, so a human finds out without watching dashboards.
  - q: How do you update a deployed service safely?
    a: Deploy the new versioned artifact, watch its health check and metrics, and keep the previous version available. If the new one misbehaves, roll back by redeploying the previous artifact. Doing updates in a way that lets you quickly return to a known-good version is what makes them safe.
---

# Monitoring & updates

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A deployment isn't done when it starts — you have to keep it healthy. **Metrics** track
trends over time and **alerts** notify you when something breaks. Ship new versions as
**updates** while watching health, and keep the previous **versioned artifact** so you
can **roll back** instantly if a release goes wrong. Safe updates are all about being
able to undo them.
</div>

This lesson is about the long tail of deployment: watching a running service and
changing it without breaking it.

## From health checks to metrics and alerts

A [health check](/learn/deployment/logging-and-health-checks/) tells you up or down.
**Metrics** go further — numbers sampled over time that reveal *trends*:

- request rate and error rate
- latency (how long responses take)
- memory and CPU use
- app-specific gauges (for GopherTrunk, active channels, decode rate)

A tool like **Prometheus** scrapes these from a `/metrics` endpoint (GopherTrunk labels
its [compose service](/learn/deployment/docker-compose/) for exactly this) and stores
them so you can graph them. **Alerts** sit on top: define a rule — "error rate above 5%"
or "service down for 2 minutes" — and get notified automatically, so you learn about
problems from a page, not from an angry user.

## Rolling out a new version

When a new [version](/learn/deployment/build-artifacts-and-versioning/) is ready,
updating a container deployment is a pull-and-recreate:

```bash
docker compose pull        # fetch the new image
docker compose up -d       # recreate the container with it
```

For a systemd binary, you install the new binary and restart the service. Either way,
right after the update you **watch the health check and metrics** — the update isn't
"done" until the new version proves healthy under real traffic.

## Rolling back

Here's why [immutable, versioned artifacts](/learn/deployment/build-artifacts-and-versioning/)
matter so much: if the new version misbehaves, you don't debug in production under
pressure — you **roll back** to the previous known-good version, which still exists:

```bash
docker compose pull gophertrunk:1.4.1   # the previous good version
docker compose up -d
```

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 90" role="img" aria-label="Version 1.4.1 running, then deploy 1.4.2, health check fails, then roll back to 1.4.1." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
  <rect x="10" y="34" width="80" height="28" rx="4" fill="none" stroke="currentColor"/><text x="50" y="52">run 1.4.1</text>
  <rect x="130" y="34" width="90" height="28" rx="4" fill="none" stroke="currentColor"/><text x="175" y="52">deploy 1.4.2</text>
  <rect x="260" y="34" width="95" height="28" rx="4" fill="none" stroke="currentColor" stroke-dasharray="3 2"/><text x="307" y="52">health fails</text>
  <rect x="395" y="34" width="110" height="28" rx="4" fill="none" stroke="currentColor" stroke-width="1.7"/><text x="450" y="52">roll back 1.4.1</text>
  </g>
  <g stroke="currentColor"><line x1="90" y1="48" x2="130" y2="48"/><line x1="220" y1="48" x2="260" y2="48"/><line x1="355" y1="48" x2="395" y2="48"/></g>
</svg>
<figcaption>Because the previous versioned artifact still exists, a bad release is undone by redeploying it — no live debugging.</figcaption>
</figure>

## Update safely, sleep well

Good update habits:

- **Update one thing at a time** so if something breaks, you know what caused it.
- **Watch after every deploy** — health, metrics, logs — for long enough to trust it.
- **Keep the last known-good version** ready to redeploy.
- **Automate it** through your [CI/CD pipeline](/learn/deployment/ci-cd-pipelines/) so
  updates are consistent, not improvised.

<div class="knowledge-check" data-quiz data-correct-msg="Right — keeping the previous versioned artifact lets you roll back instantly." markdown="0">
  <p class="knowledge-check__q">Quick check: what makes rolling back a bad release possible?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Editing the running container by hand</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The previous versioned artifact still exists to redeploy</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Deleting the logs</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Keep a deployment healthy with **metrics** (trends over time) and **alerts**
  (automatic notification).
- Ship new versions as **updates** — pull and recreate — then **watch** health and
  metrics.
- Keep the previous **versioned artifact** so you can **roll back** instantly.
- Update one thing at a time, watch after each deploy, and automate through CI/CD.

Next up: Unit 5 — deploy GopherTrunk end to end, using everything you've learned.
