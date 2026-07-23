---
slug: incident-response
title: Incident response & runbooks
description: "What to do when production breaks — detecting, mitigating, and communicating during an incident, and the runbooks and postmortems that prevent the next one."
keywords: incident response, runbook, postmortem, blameless postmortem, severity, mitigation, on-call, incident commander, mttr, production outage
level: intermediate
status: full
prereq:
  - alerting-and-oncall
faq:
  - q: What is a blameless postmortem?
    a: "A blameless postmortem is a write-up after an incident that focuses on what in the system let the failure happen, not on who made a mistake. People act reasonably given the information and tools they had, so the fix is almost always a systemic one — a missing guardrail, a confusing interface, a gap in monitoring. Removing blame is what makes people honest, which is what makes the analysis useful."
  - q: What's the difference between mitigating and resolving an incident?
    a: "Mitigating means stopping the user impact as fast as possible — often by rolling back, failing over, or restarting — even if you don't yet understand the root cause. Resolving means actually fixing the underlying cause so it can't recur. During an incident you mitigate first to stop the bleeding; you resolve afterwards, calmly, once users are no longer affected."
---

# Incident response & runbooks

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **incident** is unplanned user impact. Responding to one is a repeatable flow:
**detect** it (usually via an [alert](/learn/deployment/alerting-and-oncall/)),
**mitigate** to stop the bleeding *before* you fully understand it, **communicate** with
users and teammates, then **resolve** the root cause afterwards. A **runbook** is the
pre-written playbook for a known failure; a **blameless postmortem** turns each incident
into a systemic fix.
</div>

[Alerting](/learn/deployment/alerting-and-oncall/) wakes someone when production breaks.
This lesson is what that someone *does*. The goal during an incident is not to be clever —
it's to reduce user impact fast, then learn from it so the same thing can't page you
twice.

## The response flow

<figure class="figure" markdown="0">
<svg viewBox="0 0 480 90" role="img" aria-label="Four stages: detect, mitigate, communicate, resolve, in sequence." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
  <rect x="8" y="30" width="96" height="30" rx="4" fill="none" stroke="currentColor"/><text x="56" y="49">detect</text>
  <rect x="132" y="30" width="96" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.7"/><text x="180" y="49">mitigate</text>
  <rect x="256" y="30" width="112" height="30" rx="4" fill="none" stroke="currentColor"/><text x="312" y="49">communicate</text>
  <rect x="396" y="30" width="80" height="30" rx="4" fill="none" stroke="currentColor"/><text x="436" y="49">resolve</text>
  </g>
  <g stroke="currentColor" fill="none"><line x1="104" y1="45" x2="132" y2="45"/><line x1="228" y1="45" x2="256" y2="45"/><line x1="368" y1="45" x2="396" y2="45"/></g>
</svg>
<figcaption>Detect the problem, mitigate to stop impact, communicate throughout, then resolve the root cause.</figcaption>
</figure>

- **Detect** — an alert fires, or a user reports it. The clock on user impact is running.
- **Mitigate** — stop the bleeding *before* diagnosing. Roll back the recent deploy, fail
  over, restart — whatever ends the impact fastest.
- **Communicate** — tell affected users and teammates what's happening and when you'll
  update next, even if the update is "still investigating."
- **Resolve** — once users are safe, calmly find and fix the root cause.

## Mitigate before you diagnose

The instinct to understand *why* before acting is the wrong one mid-incident. Every minute
spent debugging is a minute of user impact. If the incident started right after a deploy,
your fastest mitigation is almost always a [rollback](/learn/deployment/zero-downtime-deploys/)
to the last known-good [version](/learn/deployment/build-artifacts-and-versioning/):

```bash
# stop the bleeding first — roll back, understand later
docker compose up -d           # recreate on the previous known-good image
journalctl -u gophertrunk -n 200 --no-pager   # capture logs for the postmortem
```

Note the second command: grab the evidence *before* it rotates away, then move on.
Understanding is for the postmortem; right now you're ending impact.

## Severity sets the response

Not every incident is all-hands. A **severity** level scales the response to the impact:

| Severity | Impact | Response |
|----------|--------|----------|
| SEV1 | Full outage / data loss | All hands, incident commander, active comms |
| SEV2 | Major feature broken for many | On-call + owner, regular updates |
| SEV3 | Minor / degraded, workaround exists | On-call handles, ticket to follow up |

Declaring the severity early tells everyone how hard to pull the fire alarm — and stops a
SEV3 from consuming a team that a SEV1 will need.

## Runbooks: don't improvise the known

A **runbook** is a short, pre-written playbook for a *known* failure mode: the symptom,
how to confirm it, and the exact steps to mitigate. Every [alert](/learn/deployment/alerting-and-oncall/)
should link one, so a half-awake responder follows steps instead of inventing them. A
runbook for GopherTrunk losing its control channel might read: *confirm via `/api/v1/health`,
check the SDR is enumerated with `lsusb`, restart the service, escalate to the owner if it
doesn't recover in 10 minutes.* Boring, specific, and exactly what you want at 3 a.m.

## The blameless postmortem

After a significant incident, write it up — timeline, impact, what happened, and what will
stop it recurring. Make it **blameless**: focus on the *system* that let the failure
happen, not the person at the keyboard. People act reasonably with the information they
have, so the durable fixes are systemic — a missing health gate, a confusing config, an
[alert](/learn/deployment/alerting-and-oncall/) that fired too late. Blame makes people
hide facts; blamelessness makes them share the ones that prevent the next incident. Each
postmortem's action items feed back into the [next build](/learn/deployment/the-deployment-lifecycle/)
— closing the deployment loop.

<div class="knowledge-check" data-quiz data-correct-msg="Right — mitigate first to stop user impact; find the root cause afterwards." markdown="0">
  <p class="knowledge-check__q">Quick check: what should you do first when an incident starts?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Fully diagnose the root cause before touching anything</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Mitigate to stop user impact, even before you understand the cause</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Write the postmortem</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **incident** is unplanned user impact; respond with **detect → mitigate →
  communicate → resolve**.
- **Mitigate first** — roll back or restart to stop impact before diagnosing.
- **Severity** scales the response so effort matches impact.
- **Runbooks** give responders pre-written steps; link one from every alert.
- A **blameless postmortem** fixes the *system*, and its action items feed the next build.

Next up: keeping a deployment affordable — cost and resource management.
