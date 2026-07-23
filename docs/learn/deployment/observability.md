---
slug: observability
title: Observability — metrics, logs & traces
description: "The three pillars that let you understand a running system — metrics, logs, and traces — and how they answer is it up, what happened, and why is it slow."
keywords: observability, metrics, logs, traces, three pillars, prometheus, grafana, metrics endpoint, monitoring, distributed tracing
level: intermediate
status: full
prereq:
  - logging-and-health-checks
faq:
  - q: What are the three pillars of observability?
    a: "Metrics, logs, and traces. Metrics are cheap numeric time-series that answer 'is it up and how much?' — request rate, error rate, memory. Logs are timestamped event records that answer 'what exactly happened?' Traces follow one request across services to answer 'where did the time go?' Together they let you understand a system's behaviour from the outside without attaching a debugger."
  - q: What is the difference between monitoring and observability?
    a: "Monitoring watches for known problems — you decide in advance what to measure and alert on. Observability is the broader property of being able to ask new questions of a running system after the fact, including ones you didn't anticipate. Good metrics, logs, and traces give you observability; dashboards and alerts built on them are monitoring."
---

# Observability — metrics, logs & traces

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Observability is being able to understand a running system from the outside. It rests on
**three pillars**: **metrics** (cheap numbers over time — "is it up?"), **logs**
(timestamped events — "what happened?"), and **traces** (one request's path across
services — "why is it slow?"). GopherTrunk exposes a **`/metrics`** endpoint that
Prometheus scrapes and Grafana graphs.
</div>

[Logging & health checks](/learn/deployment/logging-and-health-checks/) gave you a service
that can *say* it's healthy. Observability is the wider skill of *understanding* it —
answering questions you didn't script in advance. Three kinds of signal do that, and each
answers a different question.

## The three pillars

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 110" role="img" aria-label="Three columns: metrics answering is it up, logs answering what happened, traces answering why is it slow." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9.5" fill="currentColor" text-anchor="middle">
  <rect x="14" y="16" width="130" height="80" rx="5" fill="none" stroke="currentColor"/><text x="79" y="38">metrics</text><text x="79" y="60" font-size="8">numbers over time</text><text x="79" y="80" font-size="8">"is it up?"</text>
  <rect x="170" y="16" width="130" height="80" rx="5" fill="none" stroke="currentColor"/><text x="235" y="38">logs</text><text x="235" y="60" font-size="8">event records</text><text x="235" y="80" font-size="8">"what happened?"</text>
  <rect x="326" y="16" width="130" height="80" rx="5" fill="none" stroke="currentColor"/><text x="391" y="38">traces</text><text x="391" y="60" font-size="8">a request's path</text><text x="391" y="80" font-size="8">"why is it slow?"</text>
  </g>
</svg>
<figcaption>Each pillar answers a different question; together they let you understand behaviour without a debugger.</figcaption>
</figure>

## Metrics: is it up, and how much?

**Metrics** are cheap numeric measurements sampled over time — request rate, error count,
response latency, memory in use. They're compact enough to keep for months, which makes
them perfect for dashboards, trends, and [alerts](/learn/deployment/alerting-and-oncall/).
GopherTrunk exposes them in Prometheus format at **`/metrics`** — its compose file even
labels the container for a Prometheus sidecar to find:

```yaml
labels:
  - "prometheus.scrape=true"
  - "prometheus.port=8080"
  - "prometheus.path=/metrics"
```

**Prometheus** periodically *scrapes* that endpoint and stores the numbers as time-series;
**Grafana** graphs them into dashboards:

```text
# a Prometheus scrape target
scrape_configs:
  - job_name: gophertrunk
    static_configs:
      - targets: ["127.0.0.1:8080"]
    metrics_path: /metrics
```

A metric like `http_requests_total` climbing while `http_request_errors_total` stays flat
tells you at a glance the service is healthy under load — no log-reading required.

## Logs: what exactly happened?

Metrics tell you *that* errors rose; **logs** tell you *what* they were. A log line is a
timestamped record of a discrete event, ideally [structured](/learn/deployment/logging-and-health-checks/)
so you can filter it:

```text
{"time":"2026-07-23T14:02:11Z","level":"error","msg":"decode failed","tg":52198,"freq":851012500}
```

When a metric spikes, logs are where you go to read the story of a specific failure — the
talkgroup, the frequency, the error. Ship them somewhere searchable (the
[journal](/learn/linux-cli/monitoring-and-logs/), or a log aggregator) so you can query
across time rather than tailing one file.

## Traces: where did the time go?

In a system of several services, a single user request hops between them, and "why is it
slow?" means "which hop ate the time?" A **trace** follows one request end to end,
recording how long each step took as nested **spans**. For a single-service app like a
lone GopherTrunk instance, traces matter little — there's one process. They earn their
keep once a request fans out across services, which is exactly the world
[orchestration](/learn/deployment/container-orchestration/) and
[scaling](/learn/deployment/scaling-and-load-balancing/) create.

## Pick the pillar for the question

| The question | The pillar |
|--------------|-----------|
| Is the service up? Is error rate rising? | Metrics |
| What was *that specific* failure? | Logs |
| Which service made this request slow? | Traces |
| Should I get paged right now? | Metrics → [alerts](/learn/deployment/alerting-and-oncall/) |

You don't need all three from day one — metrics plus good logs cover most single-service
deployments, and GopherTrunk gives you both out of the box. Add traces when you have
multiple services and "which one is slow?" becomes a real question.

<div class="knowledge-check" data-quiz data-correct-msg="Right — metrics are cheap numbers over time, ideal for dashboards and alerts on whether it's up." markdown="0">
  <p class="knowledge-check__q">Quick check: which pillar best answers "is the service up and how much load is it under?"</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Metrics</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Traces</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A one-off debugger session</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Observability rests on three pillars: **metrics**, **logs**, and **traces**.
- **Metrics** are cheap numbers over time — "is it up?" — and drive dashboards and alerts;
  GopherTrunk exposes **`/metrics`** for Prometheus, graphed by Grafana.
- **Logs** are timestamped events — "what happened?" — best kept structured and
  searchable.
- **Traces** follow one request across services — "why is it slow?" — and matter most in
  multi-service systems.

Next up: turning metrics into pages that matter — alerting and on-call.
