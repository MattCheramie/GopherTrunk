---
slug: alerting-on-calls
title: Alerting on the calls you care about
description: Let the setup watch for you — triggers and alerts on specific talkgroups, unit IDs, or activity patterns so you're notified the moment something happens, instead of babysitting a scanner for the one call that matters.
keywords: scanner alerts, talkgroup alert, call trigger, priority talkgroup, unit ID alert, notification, alerting, scanner notification, keyword alert, monitoring trigger
level: intermediate
status: full
prereq:
  - talkgroups-and-scan-lists
faq:
  - q: What can I set an alert on?
    a: Anything your setup knows about a call from the control channel — most usefully a specific talkgroup or set of talkgroups, a particular source unit ID, or a pattern like "any activity on a talkgroup that's normally quiet." Since scanner audio isn't transcribed by default, alerts fire on metadata (who and where), not on spoken words, unless you add speech-to-text.
  - q: How does an alert reach me?
    a: However you wire it up — a sound or pop-up on the monitoring machine, a push notification to your phone, an email or chat message, or triggering a recording and a log entry. The alert is just an action fired when a call matches your rule; where it goes is your choice.
  - q: How do I avoid alert fatigue?
    a: Alert narrowly. An alert on a busy dispatch talkgroup fires constantly and you quickly learn to ignore it, which defeats the purpose. Reserve alerts for the genuinely notable — a tac channel that only lights up during an incident, a specific unit, or unusual activity — and let everything else flow into the log silently.
---

# Alerting on the calls you care about

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
You can't watch a scanner all day, so let it watch for you. An **alert** is an
**action fired when a call matches a rule** you set — a specific
**talkgroup**, a **source unit**, or an unusual activity pattern. It can pop up on
screen, push to your phone, or kick off a recording. The craft is **alerting
narrowly**: a rule that fires constantly becomes noise you ignore, so reserve alerts
for the genuinely notable and let routine traffic flow into the
[log](/learn/scanning/logging-and-recording/) silently.
</div>

The whole promise of a monitoring setup is that it does the waiting for you. You care
about a handful of things — a tac channel that only lights up during a working
incident, a particular unit, the first sign of a big call-out — but they happen at
unpredictable times, buried in hours of routine chatter. Alerting is how you get
notified the instant one occurs, without babysitting the radio. This lesson covers
what to alert on, how the alert reaches you, and how to keep alerts trustworthy.

## What you can alert on

An alert fires on what your setup already knows about a call from the control
channel, so the natural triggers are all **metadata**:

- **A talkgroup, or a set of them.** The most common alert — "tell me the moment
  County Fire Tac 2 keys up." You already grouped talkgroups when you built your
  [scan lists](/learn/scanning/talkgroups-and-scan-lists/); an alert is a scan-list
  entry with a notification attached.
- **A source unit ID.** Follow one radio wherever it goes — useful for tracking a
  specific unit across an event, regardless of which talkgroup it's on.
- **An activity pattern.** "Any traffic at all on a talkgroup that's normally
  silent," or "more than N calls a minute on this system" — a quiet channel suddenly
  going busy is often the earliest sign something is happening.

What you *can't* alert on out of the box is spoken words: scanner audio isn't
transcribed by default, so alerts fire on **who and where**, not on **what was
said** — unless you bolt on speech-to-text, which is a much heavier add-on and beyond
a normal setup.

## How the alert reaches you

An alert is just an **action** wired to a matching rule, and where that action goes
is entirely your choice. Common destinations:

- **On the machine** — a sound, a pop-up, or a highlighted row so a call you care
  about jumps out of the scrolling log.
- **To your phone** — a push notification, so you learn about the incident whether
  you're at the desk or across town.
- **To a channel** — an email, a chat message, or a webhook into another tool, handy
  if a group of people watches the same system.
- **A recording or priority action** — the match itself triggers a
  [recording](/learn/scanning/logging-and-recording/), bumps the call to priority, or
  jumps the receiver to it.

You can fire several at once: highlight it on screen, push to your phone, *and* start
recording. The alert engine matches; the actions are yours to compose.

## Priority: alerting's real-time cousin

Alerting has a close relative you already met — **priority**. A priority talkgroup
interrupts whatever you're monitoring the instant it becomes active, so the receiver
jumps to the important call in real time. Think of priority as an alert whose action
is "listen to this *now*," while a notification alert's action is "tell me it
happened." They complement each other: priority handles the call you must hear live,
notifications handle the ones you want to know about but can catch up on later. Both
draw on the same matching rules you built into your scan lists.

## Keep alerts trustworthy: alert narrowly

The failure mode of alerting is **alert fatigue**. Put an alert on a busy dispatch
talkgroup and it fires every thirty seconds; within a day you've trained yourself to
ignore it, and now it's worse than useless — it's noise that hides the real thing.
An alert you ignore is an alert that isn't working.

So alert **narrowly and deliberately**. Reserve alerts for events that are genuinely
notable *and* genuinely infrequent: the tac channel that's silent until an incident,
the specific unit, the quiet talkgroup that suddenly wakes up. Everything routine
should flow into the log without a peep. The test is simple — if an alert fires and
your honest reaction is "so what," it's the wrong alert. Tune it down until every
alert that fires is one you're glad to have gotten.

## Building it up over time

Good alerting isn't designed once; it accretes. You start with one or two obvious
rules, notice you keep missing a certain kind of call, add a rule for it, then prune
one that cries wolf. Over weeks your alert set converges on the small handful of
things that reliably mean "pay attention," and the rest of the traffic keeps filling
the [searchable log](/learn/scanning/metadata-and-tagging/) quietly in the
background. That combination — loud about the few things that matter, silent about
everything else — is what makes an unattended setup genuinely useful, and it's the
mindset the [monitoring-post](/learn/scanning/building-a-monitoring-post/) lesson
builds a full station around.

<div class="knowledge-check" data-quiz data-correct-msg="Right — alerting on a constantly-busy talkgroup causes alert fatigue, so you learn to ignore it and miss the calls that matter." markdown="0">
  <p class="knowledge-check__q">Quick check: why should you avoid setting an alert on a busy dispatch talkgroup?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Busy talkgroups can't be alerted on — only quiet ones can</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It fires constantly, so you learn to ignore it and miss the calls that matter</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Alerts on busy talkgroups are illegal in most places</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **alert** is an action fired when a call matches a rule — a **talkgroup**, a
  **unit ID**, or an unusual **activity pattern**.
- Alerts fire on **metadata** (who and where), not on spoken words, unless you add
  speech-to-text.
- The action can go anywhere — on-screen, a phone push, a message, or triggering a
  recording — and you can fire several at once.
- **Priority** is alerting's real-time cousin: its action is "listen now," where a
  notification's action is "tell me it happened."
- **Alert narrowly** to avoid alert fatigue — reserve alerts for the notable and
  infrequent, and let routine traffic flow into the log silently.

Next up: [building an always-on monitoring post](/learn/scanning/building-a-monitoring-post/).
