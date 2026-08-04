---
slug: monitoring-and-analytics
title: Monitoring & analytics
description: Knowing what your web app actually does once real users touch it — error tracking, uptime and health checks, structured logs, and privacy-respecting analytics — so you learn about problems before your users tell you.
keywords: monitoring, analytics, error tracking, uptime, health check, logging, structured logs, observability, alerting, privacy-respecting analytics, real user monitoring
level: intermediate
status: full
prereq:
  - deploying-a-web-app
faq:
  - q: "What's the difference between monitoring and analytics?"
    a: "**Monitoring** watches the *health* of your app — is it up, is it erroring, is it slow — so you can keep it running. **Analytics** watches *user behaviour* — which pages people visit, where they drop off, what they do — so you can improve the product. They overlap in tooling but answer different questions: monitoring keeps you out of trouble; analytics helps you decide what to build."
  - q: "Why not just check the logs when someone reports a problem?"
    a: "Because by then real users have already been affected, and a reactive check only finds problems people bother to report — many just leave. Good monitoring is **proactive**: automated health checks, error tracking, and alerts tell *you* about a broken deploy or a spike in errors within minutes, often before a single user complains. You want to find issues before they find you."
  - q: "Can I do analytics without invading users' privacy?"
    a: "Yes. **Privacy-respecting analytics** measures aggregate behaviour — page views, referrers, broad device types — without tracking individuals across sites or storing personal data. It answers *what's happening on my site* without building profiles of people, which is usually all you actually need and keeps you clear of the heavier consent and compliance burden."
---

# Monitoring & analytics

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Once your app is [deployed](/learn/web-dev/deploying-a-web-app/), you need to know
what it does *in the wild*. **Monitoring** watches the app's **health** — uptime,
errors, latency — so you learn about problems **before your users do**, through
**health checks**, **error tracking**, structured **logs**, and **alerts**.
**Analytics** watches **user behaviour** to guide the product, ideally with
**privacy-respecting** tools that measure aggregates instead of profiling people. The
throughline: shipping isn't the end — running it well means measuring it continuously.
</div>

Deploying gets your app onto the internet; it doesn't tell you whether it's *working*.
The moment real users, real devices, and real networks are involved, things break in
ways your laptop never showed you — and the worst way to find out is a user emailing
"your site is down." This lesson is about closing that gap: watching a live app so you
learn what it's doing before anyone has to tell you.

## Why you can't just watch the logs

A running app is a black box unless you deliberately make it observable. Waiting for
users to report problems is a losing strategy for two reasons: by the time they
report, people have already had a bad experience, and most unhappy users simply leave
without a word. The goal is to be **proactive** — to have the app tell *you* the
moment something's wrong. That splits into two related jobs: keeping it healthy
(**monitoring**) and understanding how it's used (**analytics**).

## Monitoring: is it healthy?

**Monitoring** answers *is the app up, working, and fast?* The pieces build on each
other:

- **Uptime & health checks.** A tiny endpoint — often `/health` — that returns "OK"
  when the app is alive. An external service pings it constantly and alerts you if it
  stops answering, so a crashed process or bad deploy is caught in minutes.

  ```http
  GET /health   ->   200 OK   {"status":"ok","db":"connected"}
  ```

- **Error tracking.** When code throws in production, an error tracker captures it —
  the message, stack trace, and context — and groups repeats, so you see *"this bug
  hit 400 users since the 3pm deploy"* instead of nothing. Server errors matter, and
  so do errors in the user's [browser JavaScript](/learn/web-dev/javascript-in-the-browser/),
  which you'd otherwise never see.

- **Structured logs.** Logs are the app's diary. Written as **structured** records
  (consistent fields you can search and filter) rather than free-form text, they let
  you reconstruct what happened around an incident. Log meaningful events and errors —
  never secrets or personal data.

- **Alerting.** Data no one looks at helps no one. Alerts turn signals into action:
  notify a human when errors spike, latency climbs, or the health check fails — but
  tune them, because alerts that fire constantly get ignored.

Together these are the app-level side of **observability**, the same discipline the
[deployment module](/learn/deployment/what-is-deployment/) applies to infrastructure.

## Analytics: how is it used?

**Analytics** answers a different question: *what are users actually doing?* Which
pages they visit, where they arrive from, where they abandon a flow, which feature
gets used. This guides *product* decisions — what to build, fix, or cut — in a way no
amount of monitoring can, because a page can be perfectly healthy and still useless.

Analytics also overlaps with performance: **real-user monitoring** collects the
[Core Web Vitals](/learn/web-dev/performance-and-web-vitals/) from actual visitors,
which is the field data that ultimately decides whether your site *feels* fast.

## Respecting privacy

Analytics has a bad reputation because it's often done invasively — tracking
individuals across the web and hoarding personal data. You rarely need any of that.
**Privacy-respecting analytics** measures **aggregate** behaviour — counts of page
views, referrers, broad device categories — without building profiles of people or
following them off your site. It answers *what's happening on my site* while sidestepping
the heavier consent and [compliance](/learn/web-dev/web-security-essentials/) burden,
and it's usually all the insight you actually need. Collect the minimum that answers
your question, and treat any personal data with the same care as everything else in
this module.

<div class="knowledge-check" data-quiz data-correct-msg="Right — monitoring watches the app's health so you catch problems proactively, before users report them." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the main point of monitoring a deployed app?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">To make the app load faster for every user automatically</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">To learn about health problems proactively, before users have to report them</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">To replace the need for automated tests before deploying</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Deploying isn't the end — a live app needs watching, because problems appear with
  real users, devices, and networks that your laptop never surfaced.
- **Monitoring** watches **health**: **uptime/health checks**, **error tracking**,
  structured **logs**, and **alerting** so you catch issues *proactively*, before
  users report them.
- **Analytics** watches **user behaviour** — pages, drop-off, feature use — to guide
  product decisions a healthy-but-useless page would otherwise hide.
- **Real-user monitoring** collects Core Web Vitals from actual visitors, the field
  data that decides whether the site truly feels fast.
- Prefer **privacy-respecting analytics** that measures aggregates over tools that
  profile individuals — collect the minimum that answers your question.

Next up: [choosing your web stack](/learn/web-dev/choosing-a-web-stack/).
