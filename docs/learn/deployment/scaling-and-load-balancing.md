---
slug: scaling-and-load-balancing
title: Scaling & load balancing
description: "Scaling up vs out, stateless vs stateful services, and how a load balancer spreads traffic across instances — the mechanics of handling more users."
keywords: scaling, load balancing, vertical scaling, horizontal scaling, stateless service, sticky sessions, round robin, load balancer algorithms, scale out
level: advanced
status: full
prereq:
  - container-orchestration
faq:
  - q: What's the difference between vertical and horizontal scaling?
    a: "Vertical scaling (scaling up) means giving one machine more resources — more CPU, more memory. It's simple but capped by the biggest machine you can buy, and that machine is a single point of failure. Horizontal scaling (scaling out) means running more copies of the service behind a load balancer. It scales far further and survives one instance dying, but only works if the service is stateless."
  - q: Why does horizontal scaling need stateless services?
    a: "If each instance holds its own important state — a user's session in local memory, say — then which instance a request lands on matters, and you can't freely spread traffic. A stateless instance keeps no such state locally (it lives in a shared database or cache), so any instance can serve any request. That's what lets a load balancer treat all instances as interchangeable and scale them freely."
---

# Scaling & load balancing

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Two ways to handle more load: **scale up** (a bigger machine) or **scale out** (more
copies). Scaling up is simple but capped and single-point-of-failure; scaling out goes
far further and survives failures — but only if the service is **stateless**, so any
instance can serve any request. A **load balancer** spreads traffic across the instances
by an algorithm like round-robin or least-connections.
</div>

[Orchestration](/learn/deployment/container-orchestration/) can run many replicas — but
running them only helps if traffic is *spread* across them and the service is *built* to
be spread. This lesson is the mechanics of both: the two directions you can scale, why
statelessness is the enabler, and how a load balancer divides the work.

## Scale up or scale out

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="Scaling up shown as one growing box, scaling out shown as several identical boxes behind a load balancer." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
  <text x="80" y="16">scale up</text>
  <rect x="55" y="26" width="50" height="26" rx="3" fill="none" stroke="currentColor"/>
  <rect x="45" y="60" width="70" height="52" rx="3" fill="none" stroke="currentColor" stroke-width="1.7"/><text x="80" y="90" font-size="8">bigger box</text>
  <text x="320" y="16">scale out</text>
  <rect x="245" y="30" width="150" height="22" rx="3" fill="none" stroke="currentColor"/><text x="320" y="45" font-size="8">load balancer</text>
  <rect x="245" y="78" width="40" height="30" rx="3" fill="none" stroke="currentColor"/>
  <rect x="300" y="78" width="40" height="30" rx="3" fill="none" stroke="currentColor"/>
  <rect x="355" y="78" width="40" height="30" rx="3" fill="none" stroke="currentColor"/>
  </g>
  <g stroke="currentColor" fill="none"><line x1="300" y1="52" x2="265" y2="78"/><line x1="320" y1="52" x2="320" y2="78"/><line x1="340" y1="52" x2="375" y2="78"/></g>
</svg>
<figcaption>Scaling up grows one machine; scaling out adds identical instances behind a load balancer.</figcaption>
</figure>

- **Vertical (up)** — give one machine more CPU and memory. Dead simple, no code changes,
  but capped by the largest machine available, and that one box is a single point of
  failure. A great *first* move.
- **Horizontal (out)** — run more copies behind a load balancer. Scales far past any
  single machine, and losing one instance just means the rest carry on. But it demands a
  stateless service.

Start by scaling up — it's free of architectural change. Scale out when you hit the
ceiling of one machine or need to survive an instance dying.

## Statelessness is the enabler

Horizontal scaling works only if **any instance can serve any request**. That's true when
the instance keeps no important state of its own — the state lives in a shared
[database](/learn/deployment/backups-and-data/) or cache that all instances read. Then
the load balancer can send request 1 to instance A and request 2 to instance B and
nothing breaks.

Put session data in one instance's local memory and you've broken that: a user's next
request might land on a different instance that's never heard of them. The fix is to push
state *out* of the instances into a shared store, making the instances interchangeable —
the same [stateless-vs-stateful](/learn/deployment/backups-and-data/) split that decided
what to back up.

## The load balancer and its algorithms

A **load balancer** sits in front of the instances (often the same
[reverse proxy](/learn/deployment/reverse-proxies-and-tls/) you already run) and picks
which one gets each request. Common algorithms:

| Algorithm | Picks the instance… | Good when |
|-----------|---------------------|-----------|
| Round-robin | Next one in rotation | Requests are roughly equal cost |
| Least-connections | With the fewest active requests | Request durations vary a lot |
| IP hash | Determined by client IP | You need a client pinned to one instance |
| Weighted | Proportional to capacity | Instances have different sizes |

An nginx upstream spreading traffic across three instances:

```text
upstream app {
    least_conn;                 # send each request to the least-busy instance
    server 127.0.0.1:8081;
    server 127.0.0.1:8082;
    server 127.0.0.1:8083;
}
server {
    location / { proxy_pass http://app; }
}
```

The balancer also does **health checking**: an instance that fails its
[health check](/learn/deployment/logging-and-health-checks/) is pulled from rotation until
it recovers, so traffic only goes to instances that can serve it.

## Sticky sessions: the escape hatch

Sometimes you can't fully remove per-instance state and need a given client to keep
hitting the same instance. **Sticky sessions** (session affinity) pin a client to one
instance, usually via a cookie or IP hash. It's a pragmatic patch, but treat it as a
smell: it re-introduces the coupling that statelessness removed, weakening both scaling
and failover. Prefer shared state; use stickiness only when you must.

<div class="knowledge-check" data-quiz data-correct-msg="Right — stateless instances are interchangeable, so any one can serve any request." markdown="0">
  <p class="knowledge-check__q">Quick check: what makes horizontal scaling possible?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Giving each instance more CPU and memory</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Stateless instances, so any one can serve any request</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Storing each user's session in one instance's local memory</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Scale up** (bigger machine) is simple but capped and a single point of failure;
  **scale out** (more copies) goes further and survives failures.
- Scaling out needs **stateless** services — state pushed to a shared store so instances
  are interchangeable.
- A **load balancer** spreads traffic by round-robin, least-connections, and other
  algorithms, and drops unhealthy instances via health checks.
- **Sticky sessions** pin a client to one instance — a pragmatic patch, but prefer shared
  state.

Next up: replacing a running version without dropping a request — zero-downtime deploys.
