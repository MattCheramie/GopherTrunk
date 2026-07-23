---
slug: cost-and-resources
title: Cost & resource management
description: "Right-sizing CPU and memory, understanding what cloud resources cost, and keeping a deployment affordable without starving it."
keywords: cloud cost, resource limits, right-sizing, cpu limits, memory limits, cloud pricing, over-provisioning, under-provisioning, reserved instances, cost management
level: intermediate
status: full
prereq:
  - deploying-to-cloud-and-vps
faq:
  - q: What does right-sizing mean?
    a: "Right-sizing means matching the CPU and memory you allocate to what the workload actually uses — not far more (wasted money) and not far less (throttling and crashes). You measure real usage over time, then set requests and limits a sensible margin above the observed peak. It's the core discipline of keeping a deployment both affordable and stable."
  - q: Why set CPU and memory limits on a container?
    a: "Limits cap how much a container can consume so one misbehaving service can't starve everything else on the host. A memory limit also makes failures predictable — a leaking container is killed and restarted instead of taking the whole machine down. Without limits, a single runaway process can bring down every other service sharing the box."
---

# Cost & resource management

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Every deployment costs money and consumes finite CPU and memory. **Right-sizing** means
matching what you allocate to what the workload actually uses — measure first, then set
**limits** a margin above the real peak. **Over-provisioning** wastes money;
**under-provisioning** causes throttling and crashes. Cloud **pricing models**
(on-demand, reserved, spot) trade commitment for discount. Limits also protect the host:
one greedy container can't starve the rest.
</div>

You've deployed to a [cloud or VPS](/learn/deployment/deploying-to-cloud-and-vps/) — which
sends a bill and hands you a fixed pool of CPU and memory. This lesson keeps that bill sane
and that pool from being exhausted, without starving the service into instability. It's the
practical end of operating: affordable *and* stable.

## Measure before you size

You can't right-size what you haven't measured. Watch real usage over a representative
period — a busy day, not an idle minute — using the [metrics](/learn/deployment/observability/)
you already collect and basic host tools:

```bash
docker stats gophertrunk        # live CPU / memory for the container
free -h                         # host memory headroom
```

The number that matters is the **sustained peak** under real load, plus a margin. Size to
that, not to the average (you'll throttle at peak) and not to the worst case imaginable
(you'll pay for headroom you never touch).

## Set requests and limits

Once you know the real usage, cap it. In Compose:

```yaml
services:
  gophertrunk:
    deploy:
      resources:
        limits:
          cpus: "1.0"           # never use more than 1 core
          memory: 512M          # killed & restarted if it exceeds this
        reservations:
          memory: 256M          # guaranteed this much
```

GopherTrunk's shipped [systemd unit](/learn/deployment/production-hardening/) offers the
same as commented defaults — `MemoryMax=1G`, `TasksMax=256` — described as *conservative
for a 2-SDR setup*. Limits do double duty: they keep cost predictable **and** they protect
the host, since a limited container can't consume the whole machine and take its neighbours
down with it.

## Over- and under-provisioning

Right-sizing lives between two failure modes:

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 96" role="img" aria-label="A spectrum from under-provisioned through right-sized to over-provisioned." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="52" x2="450" y2="52" stroke="currentColor" stroke-opacity="0.4"/>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
  <circle cx="70" cy="52" r="5" fill="none" stroke="currentColor"/><text x="70" y="34">under</text><text x="70" y="76" font-size="7.5">throttles, crashes</text>
  <circle cx="235" cy="52" r="6" fill="currentColor"/><text x="235" y="34">right-sized</text><text x="235" y="76" font-size="7.5">stable + affordable</text>
  <circle cx="400" cy="52" r="5" fill="none" stroke="currentColor"/><text x="400" y="34">over</text><text x="400" y="76" font-size="7.5">wasted spend</text>
  </g>
</svg>
<figcaption>Under-provisioning starves the service; over-provisioning burns money; right-sizing sits in the middle.</figcaption>
</figure>

- **Under-provisioning** — too little CPU throttles the service and too little memory gets
  it killed. Cheap on paper, expensive in outages.
- **Over-provisioning** — a giant instance running at 5% utilisation. Stable, but you're
  paying for idle capacity every hour. It's the more common and quieter waste.

## Cloud pricing models

Cloud providers price the same capacity differently depending on how much you commit:

| Model | You pay | Trade-off |
|-------|---------|-----------|
| On-demand | Full rate, no commitment | Flexible; most expensive per hour |
| Reserved / committed | Discount for a 1–3 year commitment | Cheaper for steady, predictable load |
| Spot / preemptible | Deep discount, can be reclaimed anytime | Cheapest; only for interruptible work |

A steady service like a always-on GopherTrunk instance suits **reserved** pricing or a
flat-rate [VPS](/learn/deployment/deploying-to-cloud-and-vps/); bursty batch jobs that can
tolerate interruption suit **spot**. Match the pricing model to the workload's shape and
the same compute costs far less.

## Keep it honest over time

Costs drift: a service grows, an instance was sized for a spike that's long gone, a forgotten
test box bills forever. Revisit sizing periodically against fresh metrics, delete what you
don't use, and set a billing alert so a surprise bill pages you like any other
[symptom](/learn/deployment/alerting-and-oncall/). Cost is just another signal to observe
and keep in bounds.

<div class="knowledge-check" data-quiz data-correct-msg="Right — right-sizing matches allocation to real measured usage plus a margin." markdown="0">
  <p class="knowledge-check__q">Quick check: what is right-sizing?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Always buying the largest instance for safety</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Matching CPU and memory to real measured usage plus a sensible margin</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Removing all limits so the app can use everything</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Measure** real usage (sustained peak) before sizing anything.
- Set **limits and reservations** to cap cost and protect the host from a greedy
  container — GopherTrunk's unit ships conservative defaults.
- **Under-provisioning** throttles and crashes; **over-provisioning** wastes spend —
  right-sizing sits between.
- Match the **pricing model** (on-demand, reserved, spot) to the workload's shape.
- Revisit sizing over time and alert on the bill like any other signal.

Next up: putting it all together — deploying GopherTrunk end to end.
