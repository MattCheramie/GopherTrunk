---
slug: alerting-and-oncall
title: Alerting & on-call
description: "Turning metrics into pages that matter — good alert thresholds, avoiding alert fatigue, and what being on-call for a service actually means."
keywords: alerting, on-call, slo, alert thresholds, alert fatigue, paging, pagerduty, symptom alerts, runbook, error budget
level: intermediate
status: full
prereq:
  - observability
faq:
  - q: What is alert fatigue?
    a: "Alert fatigue is what happens when a system pages people too often, especially for things that don't need action. Responders start ignoring or silencing alerts, and eventually miss the real one buried among the noise. The cure is to alert only on symptoms that need a human right now, tune out the flappy and the harmless, and make every page actionable."
  - q: What is an SLO and how does it drive alerts?
    a: "A service level objective (SLO) is a target for how the service should behave from the user's view — for example, 99.9% of requests succeed. It gives you an error budget: the small fraction you're allowed to miss. You alert when you're burning that budget fast enough to breach the SLO, which ties pages to real user impact instead of arbitrary thresholds."
---

# Alerting & on-call

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **alert** turns a metric crossing a line into a notification — ideally a **page** only
when a human must act now. Good alerts fire on **user-visible symptoms** (errors,
latency, "it's down"), tie to an **SLO**, and each links a **runbook**. Too many alerts
cause **alert fatigue** — people tune out the noise and miss the real one. **On-call**
means being the person who responds when a page fires.
</div>

[Observability](/learn/deployment/observability/) gave you metrics; nobody watches a
dashboard at 3 a.m. **Alerting** is how the system watches for you and wakes someone only
when it must. The whole art is firing on the *right* things — enough to catch real
problems, few enough that people still listen.

## Alert on symptoms, not causes

The most useful alerts fire on what a **user would notice**: requests failing, responses
slow, the service unreachable. These "symptom" alerts catch every underlying cause,
including ones you never predicted. Alerting on internal causes — high CPU, a full-ish
disk — floods you with pages that often don't affect anyone. A Prometheus rule that
alerts on a user-visible symptom:

```yaml
groups:
  - name: gophertrunk
    rules:
      - alert: HighErrorRate
        expr: rate(http_request_errors_total[5m]) / rate(http_requests_total[5m]) > 0.05
        for: 10m           # sustained 10 min, not a one-second blip
        labels:
          severity: page
        annotations:
          summary: "Error rate above 5% for 10 minutes"
          runbook: "https://runbooks.example.com/gophertrunk/high-error-rate"
```

Two details matter: `for: 10m` waits for the problem to be *sustained* so a momentary
blip doesn't page anyone, and the `runbook` annotation links the fix.

## Tie thresholds to an SLO

Where do you draw the line? Anchor it to a **service level objective** — a target from the
user's perspective, like "99.9% of requests succeed." That leaves a small **error budget**
(the 0.1% you're allowed to miss). You page when you're burning that budget fast enough to
breach the SLO, which means every page corresponds to real user impact. Arbitrary
thresholds ("CPU > 80%") page you about things users never feel; SLO-based alerts page
you about things they do.

## Avoid alert fatigue

The fastest way to make alerting useless is too much of it. When pages fire constantly —
especially for things that need no action — people start silencing and ignoring them, and
the one real page gets lost in the noise. Keep alerts trustworthy:

- **Every page must be actionable** — if there's nothing to do, it's not a page.
- **Alert on symptoms, suppress the flapping** — use `for:` durations and sensible
  thresholds.
- **Route by severity** — a *page* wakes someone; a *ticket* or a chat message is enough
  for the non-urgent.

| Severity | Means | Delivery |
|----------|-------|----------|
| Page | User impact now, act immediately | Phone/pager, wakes someone |
| Ticket | Needs attention, not tonight | Issue tracker, next business day |
| Info | Worth knowing, no action | Chat channel, dashboard |

## What on-call actually means

**On-call** is being the designated responder for a window of time — you carry the pager
and answer when it fires. Healthy on-call has a few properties: a **rotation** so no one
person carries it forever, a **clear escalation** path when the first responder is stuck,
and every alert linking a **runbook** so the responder isn't debugging from a blank page.
A page should arrive with a symptom, a severity, and a link to *what to do* — which is the
subject of the next lesson.

<div class="knowledge-check" data-quiz data-correct-msg="Right — alert on user-visible symptoms so every page reflects real impact and stays actionable." markdown="0">
  <p class="knowledge-check__q">Quick check: what should a page-level alert fire on?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Any CPU spike above 80%</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A sustained user-visible symptom like a high error rate</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Every log line marked "warning"</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **alert** notifies when a metric crosses a line; a **page** should mean a human must
  act now.
- Alert on **user-visible symptoms**, not internal causes; use `for:` to require a
  sustained problem.
- Tie thresholds to an **SLO** and its error budget so pages reflect real user impact.
- Fight **alert fatigue**: every page actionable, route by severity, link a runbook.
- **On-call** is the responder rotation — with escalation and runbooks so pages are
  answerable.

Next up: what to do when a page fires — incident response and runbooks.
