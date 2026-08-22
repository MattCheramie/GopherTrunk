---
slug: monitoring-your-board
title: Monitoring your board
description: Temperature, load, disk, and service health — a lightweight monitoring script, trends over snapshots, and alerts that tell you about trouble before it becomes an outage.
keywords: monitor raspberry pi, health check script, disk space alert, temperature monitoring, load average, systemctl is-active, cron monitoring, lightweight monitoring sbc
level: intermediate
status: full
prereq:
  - services-with-systemd
  - thermal-throttling
---

# Monitoring your board

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An unattended board fails silently unless something is watching. Watch the **four
vitals** — **temperature**, **load**, **disk space**, and **service health** —
plus the work itself (is it actually decoding?). The right weight for one
appliance is a **small script on a timer** that logs the vitals (a **trend**
beats any snapshot — problems announce themselves as drift) and **alerts** you
only when a threshold breaks. The goal is a machine that tells *you* about
trouble days before it becomes an outage — never the other way round.
</div>

Unit 5's finale turns the tables: instead of you remembering to check the board,
the board reports to you. Everything here composes tools you already have —
the vitals commands, cron, SSH — into a habit that catches every slow-burning
failure this unit described.

## What are the four vitals — and the fifth that outranks them?

| Vital | Command | Failing pattern it catches |
|-------|---------|---------------------------|
| **Temperature** | `vcgencmd measure_temp` | Summer, dust, dead fan → [throttling](/learn/embedded/thermal-throttling/) |
| **Load** | `cat /proc/loadavg` | Creeping CPU → falling behind real time |
| **Disk** | `df -h /` | Recordings/logs filling the card → crash-on-full |
| **Service** | `systemctl is-active gophertrunk` | Crash-loops, failed starts |

Each maps to a slow failure from this unit — and each gives days of warning if
anything is looking. The fifth check outranks them all: **is the work
happening?** A service can be `active` while decoding nothing (the hang class
from [Watchdogs &amp; recovery](/learn/embedded/watchdogs-and-recovery/)). Probe
the *output*: has a recording landed recently, has the log advanced, does the
web console answer? For GopherTrunk, "no CC lock for an hour on a busy system"
is a better alarm than any CPU graph.

## What does a right-sized monitor look like?

For one appliance, a dozen lines of shell on a systemd timer (or cron —
[Scheduling with cron](/learn/linux-cli/scheduling-with-cron/)):

```bash
#!/bin/bash
# /usr/local/bin/health-check — run every 5 minutes
temp=$(vcgencmd measure_temp | grep -o '[0-9.]*')
load=$(cut -d' ' -f1 /proc/loadavg)
disk=$(df --output=pcent / | tail -1 | tr -dc '0-9')
svc=$(systemctl is-active gophertrunk)

echo "$(date -Is) temp=$temp load=$load disk=${disk}% svc=$svc" >> /var/log/health.log

[ "${temp%.*}" -ge 75 ] && alert "temp ${temp}C"
[ "$disk" -ge 90 ]      && alert "disk ${disk}%"
[ "$svc" != active ]    && alert "service $svc"
```

(`alert` is whatever reaches you — the next section.) Two design points hiding
in those lines. **Log every run, alert on few**: the log builds the trend; the
thresholds guard the cliff edges. And **thresholds sit below the cliff** — alert
at 75 °C, not the 80 °C soft limit; at 90% disk, not 100% — so the alert *is*
the head start. One flourish worth adding on a Pi: `vcgencmd get_throttled`,
so undervoltage/throttling-since-boot flags land in the log too.

## Why do trends beat snapshots?

A snapshot says 68 °C — fine? A trend says 55 °C in April, 61 °C in June, 68 °C
in August — *that's a line pointing somewhere*, and you can meet it with a
vacuum cleaner before it meets you with throttling. The same goes for disk
(creeping % = retention policy leak), load (each config change's cost, visible),
and restart counts (the masked failures
[watchdogs](/learn/embedded/watchdogs-and-recovery/) warned about). The
`health.log` from the script *is* the trend — greppable, plottable, and cheap.
Give it the [journal-cap treatment](/learn/embedded/sd-card-wear/) (logrotate or
a size check) so the monitor doesn't become a wear source itself.

Weekly ritual, two minutes over SSH: skim the log's last thousand lines, note
drift, done. (GopherTrunk's web console shows the *radio* side of the same
story — decode health, call activity — the application-level dashboard atop
these system vitals.)

## How should alerts reach you — and how often?

An alert that fires daily gets ignored weekly — **alert fatigue** is the failure
mode of monitoring itself. Principles:

- **Alert on actionable states only.** "Disk 91%" → you'll prune or fix
  retention. "Load is 1.9" → so what? Log it, don't page it.
- **Pick a channel you actually see**: email (`msmtp` makes `mail` work with any
  mailbox), a phone-push service, or a chat webhook — anything reachable from a
  shell one-liner.
- **Rate-limit repeats** (a touch-file per alert type: notify once, re-arm when
  cleared) so a stuck state sends one message, not 288 a day.
- **Watch the watcher.** A silent monitor looks exactly like a healthy system. A
  weekly "all well" heartbeat message closes the loop — no news becomes *bad*
  news after seven quiet days.

> Rule of thumb: every alert should name the action you'll take. If you can't
> finish the sentence "when this fires I will…", it's a log line, not an alert.

For fleets and richer dashboards, real monitoring stacks exist — the
[Deployment module](/learn/deployment/monitoring-and-updates/) tours them. For
one appliance, the script above catches what matters.

<div class="knowledge-check" data-quiz data-correct-msg="Right — slow failures reveal themselves as drift, which only a logged trend can show." markdown="0">
  <p class="knowledge-check__q">Quick check: why log the vitals every run instead of only checking when something feels wrong?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Because the commands only work when run on a schedule</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Because slow failures show up as drift in a trend long before any single snapshot looks alarming</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Because logs make the board run faster</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Watch the **four vitals** — temperature, load, disk, service — plus the one
  that outranks them: **is the work actually happening?**
- Right-sized for one appliance: a **script on a timer** — log every run, alert
  on thresholds set **below the cliff**.
- **Trends beat snapshots**: drift is the early warning for every slow failure
  in this unit.
- Fight **alert fatigue**: actionable alerts only, rate-limited, on a channel
  you see — and a **heartbeat** so silence can't hide a dead monitor.
- With monitoring in place, Unit 5's promise holds: the board tells you about
  trouble before trouble becomes an outage.

Next up: [Install GopherTrunk on a Pi](/learn/embedded/installing-gophertrunk-on-a-pi/).
