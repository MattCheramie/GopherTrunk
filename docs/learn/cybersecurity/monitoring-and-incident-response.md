---
slug: monitoring-and-incident-response
title: Monitoring & incident response
description: Why prevention isn't enough — how logging, centralized monitoring, and intrusion detection let you notice trouble, and how the incident response cycle (prepare, identify, contain, eradicate, recover, learn) plus tested backups get you back on your feet after a breach.
keywords: monitoring, logging, SIEM, intrusion detection, incident response, containment, recovery, backups, detection, alerting, ransomware recovery, defensive security
level: advanced
status: full
prereq:
  - defense-in-depth
faq:
  - q: Why isn't prevention enough on its own?
    a: "Because every preventive control eventually fails — a patch lands late, a password gets phished, a new attack slips past. Prevention is only half the job; the other half is noticing when it fails and being able to recover. Detection and response are what keep a failure from becoming a disaster."
  - q: What is a SIEM?
    a: "A SIEM (security information and event management system) collects logs from many sources — servers, network gear, and applications — into one place, then correlates and alerts on anomalies and known-bad signs. Centralizing logs matters because you can't investigate an incident from records that were scattered, overwritten, or never kept."
  - q: What's the first thing to do during an active incident?
    a: "Contain it. Once you've confirmed an intrusion is real and live, the first priority is limiting the spread and damage — isolating affected systems — before you dig into exactly what happened. Investigation matters, but it comes after you've stopped the bleeding."
  - q: Why do backups need to be tested and offline?
    a: "A backup you've never restored from is only a guess that it works, and ransomware deliberately hunts down and encrypts backups it can reach. Offline or immutable copies survive an attacker who owns your network, and a rehearsed restore is the difference between recovering in hours and discovering your recovery plan doesn't work when you need it most."
---

# Monitoring & incident response

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
You can't prevent everything, so you plan to **detect + respond**. That rests on
**logging** — you can't investigate what you never recorded — feeding centralized
monitoring that alerts on trouble. When an incident is confirmed, **contain first**:
stop the spread before you investigate. And tested **backups** are what let you
recover from ransomware or destruction. This is the "detect and respond" half of
[defense in depth](/learn/cybersecurity/defense-in-depth/) made concrete.
</div>

Earlier lessons stacked up controls to keep attackers out. This one accepts that some
of them will fail anyway, and asks the next question: when a control fails, how do you
**notice**, and how do you **get back on your feet**? That's detection and response —
the unglamorous half of security that decides whether a breach is a bad afternoon or a
company-ending event.

## Why detection matters

Prevention eventually fails. Not because your controls are bad, but because no set of
controls is perfect and attackers only need one gap. If you've internalised
[assume breach](/learn/cybersecurity/defense-in-depth/), the logical next step is to
accept that something will get through — and to make sure you find out when it does.

The reason this matters so much is the gap between **breach and discovery**. An attacker
who gets in doesn't do all their damage in the first minute; they explore, escalate,
and spread. Every day they go unnoticed is another day to reach more systems and steal
more data. Studies of real breaches routinely find intruders sitting undetected for
weeks or months. That dwell time is where the damage compounds — which is exactly why
detection is half the job, not an afterthought.

## Logging and monitoring

Everything downstream depends on one habit: **recording what happens**. You cannot
investigate, alert on, or even prove an incident from data you never captured. So the
foundation is collecting logs from every layer:

- **Systems** — logins, privilege changes, process starts, and errors on each host.
  (This is the everyday material of [monitoring & logs](/learn/linux-cli/monitoring-and-logs/).)
- **Network** — connections, DNS lookups, and traffic to and from the outside world.
- **Applications** — what your own software records about who did what, which is part
  of [observability & monitoring](/learn/building-ai/observability-and-monitoring/).

Scattered logs are nearly useless in an incident — you can't correlate a suspicious
login on one box with an odd connection on another if they live in ten different places
that each roll over and overwrite themselves. So you **centralize** them, classically
into a **SIEM** (security information and event management system): one place that
gathers logs, keeps them safely out of an attacker's reach, and **alerts on anomalies
and known-bad signs**. Good alerting is the difference between logs you read after the
disaster and a system that taps you on the shoulder while there's still time to act.

## Detection

Centralized logs are the raw material; **detection** is turning them into a signal that
something is wrong. A few of the workhorses:

- **Intrusion detection** — systems that watch network or host activity for patterns
  that match known attacks or that simply look wrong for your environment.
- **Unusual logins or traffic** — a login from a new country at 3 a.m., an account
  suddenly touching servers it never uses, a machine quietly uploading gigabytes at
  night. None of these prove an attack, but each is worth a look.
- **Integrity checks** — comparing files against known-good
  [hashes](/learn/cybersecurity/hashing-and-integrity/) so that tampering with a system
  binary or a config file gets flagged, even when the change itself looks routine.

The aim isn't to catch everything — nothing does. It's to raise a flag early enough
that response has time to work.

## The incident response cycle

When an alert turns out to be real, you don't want to be inventing a process on the
spot. Teams work a repeatable cycle, usually drawn as six stages:

1. **Prepare** — before anything happens: have a plan, know who to call, and make sure
   your logging and backups actually exist.
2. **Identify** — confirm it's a real incident and understand its scope: what's
   affected, and how badly.
3. **Contain** — limit the spread and damage. Isolate affected machines, cut attacker
   access, and stop it from getting worse.
4. **Eradicate** — remove the cause: kick the attacker out, remove malware, close the
   hole they came through.
5. **Recover** — restore clean systems and data, verify they're healthy, and return to
   normal operations.
6. **Learn** — review what happened and fix the gaps so the next one is caught sooner.

One point is worth stating loudly because people get it wrong under pressure: the
**first move on a live incident is containment**, not investigation. It is tempting to
dig into exactly how the attacker got in while they are still inside — but every minute
you spend investigating is a minute they spend spreading. Stop the bleeding first;
understand it fully afterward.

## Backups and recovery

Recovery is only as good as your backups, and backups are where a lot of otherwise-solid
plans quietly fail. Two properties matter most:

- **Tested** — a backup you have never restored from is a hope, not a plan. Rehearse the
  restore, so you learn it's broken on a calm Tuesday and not during a crisis.
- **Offline or immutable** — [ransomware](/learn/cybersecurity/malware-and-endpoints/)
  specifically seeks out and encrypts every backup it can reach, and an attacker with
  the run of your network will delete the ones they can. A copy that's offline, or that
  cannot be altered once written, survives them.

This is what makes recovery from ransomware or outright destruction possible: not paying
a ransom, but restoring from copies the attacker never touched. A recovery plan you've
never rehearsed will fail at the exact moment you're relying on it — so rehearse it.

## Learn from it

The cycle ends where it began, on purpose. Every incident is expensive; the least you
can do is get a lesson out of it. After the dust settles, run an honest review: how did
they get in, why didn't detection fire sooner, and what would have contained it faster?

Then **close the gap**, **add detection** for the thing you missed, and **update the
plan** so the next response is smoother. A team that does this turns each incident into
a permanent improvement — which is the whole point of the "learn" stage, and the reason
mature security programs get harder to breach over time rather than repeating the same
mistakes.

<div class="knowledge-check" data-quiz data-correct-msg="Right — contain first, investigate after. Every minute spent investigating is a minute the attacker spreads." markdown="0">
  <p class="knowledge-check__q">Quick check: you've confirmed an active intrusion. What's the FIRST priority?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Fully investigate exactly how they got in before touching anything</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Contain it — limit the spread and damage before anything else</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Wipe and rebuild every machine immediately, backups or not</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Prevention eventually fails, so **detection and response** are half of security — the
  gap between breach and discovery is where damage compounds.
- **Logging** from systems, network, and applications is the foundation: you can't
  investigate what you never recorded.
- **Centralize** logs (a SIEM) and **alert** on anomalies and known-bad signs; add
  intrusion detection and integrity checks that flag tampering.
- The incident response cycle runs **prepare → identify → contain → eradicate → recover
  → learn** — and on a live incident, **contain first**.
- **Tested, offline or immutable backups** are what let you recover from ransomware or
  destruction; an unrehearsed plan fails when you need it.
- Every incident becomes a lesson: close the gap, add detection, and update the plan.

Next up: [privacy & data protection](/learn/cybersecurity/privacy-and-data-protection/)
