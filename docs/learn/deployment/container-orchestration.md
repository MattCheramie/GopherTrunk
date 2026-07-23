---
slug: container-orchestration
title: Container orchestration
description: "What Kubernetes and its cousins actually do — scheduling, self-healing, and scaling many containers — and how to tell when you truly need one."
keywords: container orchestration, kubernetes, k8s, scheduling, self-healing, autoscaling, docker swarm, nomad, when to use kubernetes, orchestrator
level: advanced
status: full
prereq:
  - deploying-to-cloud-and-vps
faq:
  - q: What does a container orchestrator do?
    a: "An orchestrator runs containers across a pool of machines for you. You declare what you want — 'run 5 copies of this image, keep them healthy, expose them on this port' — and it decides which machine each copy runs on, restarts any that die, reschedules them if a machine fails, and scales the count up or down. Kubernetes is the dominant one; Docker Swarm and Nomad are simpler alternatives."
  - q: Do I need Kubernetes for my app?
    a: "Usually not. Kubernetes earns its considerable complexity when you run many services across many machines, need automatic scaling and self-healing across nodes, and have a team to operate it. A single service on a single host — like one GopherTrunk instance — is far better served by docker-compose or a systemd unit. Reaching for Kubernetes too early buys you a second full-time system to run."
---

# Container orchestration

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **orchestrator** like **Kubernetes** runs containers across a *fleet* of machines. You
declare the desired state — how many replicas, which ports, how much CPU — and it handles
**scheduling** (which machine runs what), **self-healing** (restart or reschedule what
dies), and **scaling** (add or remove replicas). It's powerful and complex — and for one
service on one host, you almost certainly **don't need it**.
</div>

[Compose](/learn/deployment/docker-compose/) runs several containers on *one* host. When
you outgrow one host — many services, many machines, automatic scaling — you reach for an
**orchestrator**. This lesson explains what it does and, just as importantly, when *not*
to reach for it.

## From one host to a fleet

Compose is declarative but single-host: it can't move a container to another machine when
one dies, or spread ten replicas across five servers. An orchestrator manages a **cluster**
of machines as one pool. You hand it a desired state and it continuously works to make
reality match:

- **Scheduling** — decide which machine has room to run each container.
- **Self-healing** — a container crashes, the orchestrator restarts it; a whole machine
  dies, it reschedules that machine's containers elsewhere.
- **Scaling** — change "5 replicas" to "20" and it starts fifteen more, spread across the
  cluster; automatic scaling ties the count to load.
- **Service discovery & load balancing** — replicas come and go, and the orchestrator
  keeps a stable address that spreads traffic across whichever are currently healthy.

## Kubernetes in one breath

Kubernetes is the dominant orchestrator. You describe workloads declaratively — much like
IaC — and its control loop reconciles reality toward your spec:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 5                # keep 5 copies running, always
  template:
    spec:
      containers:
        - name: web
          image: ghcr.io/example/web:1.4.2
          resources:
            limits:
              cpu: "500m"     # right-sizing, from the cost lesson
              memory: "256Mi"
```

Apply that and Kubernetes ensures five copies exist across the cluster forever — kill
one and a replacement appears in seconds. It's [zero-downtime
deploys](/learn/deployment/zero-downtime-deploys/), [self-healing](/learn/deployment/logging-and-health-checks/),
and [scaling](/learn/deployment/scaling-and-load-balancing/) built into the platform.

## The cost: a whole system to operate

That power isn't free. A Kubernetes cluster is itself a distributed system — a control
plane, networking overlays, ingress controllers, storage plugins — that you must install,
secure, upgrade, and debug. It adds concepts (pods, services, ingresses, config maps,
operators) and failure modes you didn't have before. Run it and you've taken on a second
full-time system alongside your app.

## When you actually need one

The honest test is scale and team, not fashion:

| You have… | Reach for |
|-----------|-----------|
| One service on one host | A [systemd unit](/learn/deployment/services-and-systemd/) or [Compose](/learn/deployment/docker-compose/) |
| A few services on one host | [Compose](/learn/deployment/docker-compose/) |
| Many services, many machines, needing autoscaling & self-healing across nodes | Kubernetes (or a managed one) |
| Kubernetes' model but less overhead | Docker Swarm or Nomad |

A single **GopherTrunk** instance talking to a USB radio on one box is squarely in the
top row — Compose or a systemd unit is the *right* tool, and Kubernetes would be pure
overhead (it also can't magically share one physical SDR across nodes). Match the tool to
the actual problem; orchestration is a solution to a fleet-sized problem, and using it on
a single host adds cost with no benefit.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a single service on a single host is better served by Compose or systemd than by Kubernetes." markdown="0">
  <p class="knowledge-check__q">Quick check: when does a single GopherTrunk instance on one host need Kubernetes?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Always — Kubernetes is required for any container in production</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It doesn't — Compose or a systemd unit is the right fit</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Only when the container image is larger than 1 GB</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **orchestrator** runs containers across a **fleet**, handling scheduling,
  self-healing, and scaling.
- **Kubernetes** is the dominant one; you declare desired state and its control loop
  reconciles reality to it.
- It brings real **operational cost** — a distributed system you must run alongside your
  app.
- Reach for it only at **fleet scale with a team**; one service on one host wants
  **Compose or systemd** instead.

Next up: the mechanics of handling more users — scaling and load balancing.
