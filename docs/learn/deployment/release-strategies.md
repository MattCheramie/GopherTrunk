---
slug: release-strategies
title: Release strategies
description: "Recreate, rolling, blue-green, and canary releases — the patterns for pushing a new version to users without downtime or a risky big-bang cutover."
keywords: release strategies, rolling release, blue-green deployment, canary release, recreate deployment, zero downtime release, deployment patterns
level: intermediate
status: full
prereq:
  - build-artifacts-and-versioning
faq:
  - q: What is a canary release?
    a: "A canary release sends the new version to a small slice of traffic first — say 5% of users — while everyone else stays on the old version. You watch the canary's error rate and latency; if it's healthy you widen it gradually to 100%, and if it misbehaves you pull it back having exposed only a few users. The name comes from the canary miners took underground to detect danger early."
  - q: What is the difference between blue-green and rolling?
    a: "A rolling release replaces instances a few at a time, so old and new run side by side during the rollout. Blue-green runs two complete environments — blue (current) and green (new) — and flips all traffic at once when green is verified. Rolling is gradual and cheap; blue-green is instant to switch and instant to roll back but needs double the capacity."
---

# Release strategies

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **release strategy** is how you swap a new version in for the old one. **Recreate**
stops everything then starts the new version (simple, but a gap of downtime).
**Rolling** replaces instances a few at a time. **Blue-green** runs two full
environments and flips traffic instantly. **Canary** sends a small slice of traffic to
the new version first and widens it as confidence grows. The tradeoff is always
downtime and risk versus cost and complexity.
</div>

You've built a versioned [artifact](/learn/deployment/build-artifacts-and-versioning/)
and it passed CI. Now you have to put it in front of users who are already using the old
version. *How* you make that swap is a release strategy — and the choice decides whether
users notice, and what happens when the new version is broken.

## Recreate: stop the old, start the new

The simplest strategy: shut the old version down, then start the new one.

```bash
docker compose down          # stop the old version
docker compose up -d         # start the new one
```

Between those two commands the service is **down** — every request fails. That's fine
for a hobby scanner, a nightly batch job, or anything where a few seconds of gap costs
nothing. It's the wrong choice for anything users hit constantly. Recreate is honest
about its one weakness: a visible outage window.

## Rolling: replace a few at a time

If you run several instances behind a
[load balancer](/learn/networking/proxies-and-load-balancers/), you don't have to take
them all down at once. A **rolling** release updates them in batches: drain and replace
instance 1, wait for it to report healthy, then instance 2, and so on. Old and new
versions serve traffic side by side during the rollout, so there's no outage — but for a
while your users are split across two versions, which the new version must tolerate.

## Blue-green: two environments, one switch

**Blue-green** keeps two complete environments. *Blue* is live and serving all traffic;
*green* is the new version, fully deployed but receiving nothing. You test green in
place, then flip the [proxy](/learn/deployment/reverse-proxies-and-tls/) to send all
traffic to green in one move. If green misbehaves, flip straight back to blue — rollback
is instant because blue never went away. The cost is real: you need double the capacity
during the switch.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A load balancer routing all traffic to a blue environment, with a green environment standing ready, and a dashed arrow showing the switch." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
  <rect x="180" y="8" width="100" height="26" rx="4" fill="none" stroke="currentColor"/><text x="230" y="25">load balancer</text>
  <rect x="60" y="90" width="120" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.7"/><text x="120" y="109">BLUE (live v1)</text>
  <rect x="280" y="90" width="120" height="30" rx="4" fill="none" stroke="currentColor" stroke-dasharray="4 3"/><text x="340" y="109">GREEN (new v2)</text>
  </g>
  <g stroke="currentColor" fill="none"><line x1="215" y1="34" x2="120" y2="90"/><line x1="245" y1="34" x2="340" y2="90" stroke-dasharray="4 3"/></g>
  <text x="300" y="60" font-size="8" fill="currentColor">flip</text>
</svg>
<figcaption>Blue-green: all traffic hits blue; when green is verified, the load balancer flips to it in one switch.</figcaption>
</figure>

## Canary: test on a slice of real traffic

A **canary** release is a rolling release that pauses at the start to *measure*. You
route a small fraction of traffic — 1%, then 5%, then 25% — to the new version while
watching its [metrics](/learn/deployment/observability/). If error rate and latency stay
healthy, you widen the split until the new version has 100%. If the canary looks sick,
you pull it back, having exposed only a handful of users to the bad version. It's the
safest strategy for high-traffic services and the one that leans hardest on good
[observability](/learn/deployment/observability/) — you can only trust a canary you can
measure.

## Choosing one

| Strategy | Downtime | Rollback | Extra capacity | Good for |
|----------|----------|----------|----------------|----------|
| Recreate | Yes, a gap | Redeploy old | None | Hobby apps, batch jobs, single instance |
| Rolling | None | Slow (roll back through batches) | Small | Many stateless instances |
| Blue-green | None | Instant (flip back) | Double | Fast rollback matters most |
| Canary | None | Instant (stop widening) | Small | High traffic, want to limit blast radius |

For a single GopherTrunk instance on one host, a brief
[recreate](/learn/deployment/zero-downtime-deploys/) (`docker compose up -d` recreates
the container) is entirely reasonable — the fancy strategies earn their complexity only
once you're running many instances for many users.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a canary exposes only a small traffic slice to the new version first." markdown="0">
  <p class="knowledge-check__q">Quick check: which strategy sends a small fraction of real traffic to the new version before widening?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Recreate</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Canary</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Blue-green</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Recreate** is simplest but has a downtime gap — fine for single instances and jobs.
- **Rolling** replaces instances a few at a time with no outage.
- **Blue-green** runs two environments and flips traffic instantly, at double capacity.
- **Canary** routes a small traffic slice to the new version first, widening as metrics
  stay healthy.
- The tradeoff is always **downtime and risk** versus **cost and complexity**.

Next up: what a container actually is, and why it solves "works on my machine".
