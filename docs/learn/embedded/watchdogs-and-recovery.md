---
slug: watchdogs-and-recovery
title: Watchdogs & recovery
description: Design for the crash you can't prevent — systemd restart policies for crashes, hardware watchdog timers for hangs, and the layered self-healing that lets a board recover with nobody home.
keywords: hardware watchdog, systemd watchdog, restart on failure, self-healing, hung system, RuntimeWatchdogSec, recovery design, crash loop, unattended recovery
level: intermediate
status: full
prereq:
  - services-with-systemd
---

# Watchdogs & recovery

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Appliance reliability is not preventing every failure — it's **recovering from
each failure class automatically**. Three layers: **`Restart=on-failure`** revives
a **crashed** service; a **service watchdog** (`WatchdogSec=`) catches a service
that's running but **hung**; and the SoC's **hardware watchdog** — a countdown
timer that reboots the board unless the OS keeps petting it — catches a hung
*kernel*, the failure software alone cannot escape. Add "**recovers by
power-cycle**" as a design test, and layer the escalation: process restart →
service restart → reboot.
</div>

You can now keep a board cool and its storage alive. This lesson accepts the
uncomfortable truth of 24/7 operation — *something will eventually wedge anyway*
— and engineers the machine to un-wedge itself, because nobody is home to pull
the plug.

## Why design for failure instead of against it?

Run any system long enough and you'll meet the long tail: a rare bug, a memory
leak, a USB hiccup that wedges a driver, a cosmic-ray bit flip (genuinely — RAM
is not immune). Preventing all of them is impossible; *recovering* from all of
them is a Tuesday. The reframe: classify failures by **what's still alive to
notice**, and give each class a supervisor:

| Failure class | What still works | Recovery layer |
|---------------|------------------|----------------|
| Service **crashes** | systemd sees the death | `Restart=on-failure` |
| Service **hangs** (runs, does nothing) | systemd runs; service silent | Service watchdog (`WatchdogSec=`) |
| **Kernel/system hangs** | Only the silicon | **Hardware watchdog** reboot |
| Won't boot / storage dead | Nothing on the board | You + the [backup image](/learn/embedded/backups-and-images/) |

The [systemd lesson](/learn/embedded/services-with-systemd/) built the first
layer. Now the other two.

## What catches a service that's running but dead inside?

The nastiest common failure is the **hang**: the process exists, systemd is
content, and nothing has happened for an hour — a deadlock, a stuck network
call, a wedged device. Two catches:

**A systemd service watchdog.** The unit declares `WatchdogSec=60`, and the
service must actively report liveness ("pet the dog") via systemd's notification
mechanism at least that often; miss the deadline and systemd kills and restarts
it. This needs the *program's* cooperation (daemons built for service use,
GopherTrunk included, can notify) — the strongest form, because liveness is
asserted from inside the work loop.

**An external health check.** For services without native support: a timer job
probes something observable — the web console answers, the log has advanced, a
recording appeared this hour — and restarts the service when the probe fails.
Cruder, universally applicable, and it tests what you *actually care about*: work
being done, not a process existing. ([Monitoring](/learn/embedded/monitoring-your-board/)
formalises health checks; cron basics are in
[Scheduling with cron](/learn/linux-cli/scheduling-with-cron/).)

## What catches a hung kernel — when software can't save software?

If the kernel itself wedges, no process — supervisor or otherwise — will run
again until power cycles. The escape is the **hardware watchdog**: a countdown
timer in the SoC, independent of the CPU's software. Once armed, it must be
reset ("fed") continuously; if the feeder ever stops — because the whole system
froze — the timer expires and the *silicon* yanks reset. The board reboots,
services `enable`d at boot come back, and the appliance heals from a failure
mode software alone cannot escape.

systemd feeds it for you — two lines:

```ini
# /etc/systemd/system.conf
RuntimeWatchdogSec=15
ShutdownWatchdogSec=10min
```

Reboot, and from then on: kernel healthy → fed every few seconds → nothing
happens. Kernel frozen for 15 s → hardware reboot. The second line guards
shutdown itself hanging. It's among the highest reliability-per-line settings on
an SBC.

> Rule of thumb: apply the **power-cycle test** to every part of the build —
> "if this board loses power right now and comes back, does everything resume
> without me?" Services enabled at boot, filesystems that mount, dongles that
> re-enumerate, no step that waits for a keyboard. A watchdog reboot is just a
> power-cycle you didn't schedule.

## When is auto-recovery the wrong move?

Honesty about the limits keeps the tool sharp:

- **Crash loops.** A service dying instantly every restart (bad config, missing
  device) will loop; systemd's `StartLimitBurst` backs off, and the fix is in
  the journal, not another retry.
- **Masked causes.** Recovery that always works becomes invisible — a daily
  mystery reboot deserves diagnosis (check `journalctl --list-boots`, watchdog
  events, [undervoltage flags](/learn/embedded/power-supplies/)), or you'll
  never notice the failing supply behind it. Recovery buys you *time* to debug,
  not exemption from it.
- **State corruption.** Rebooting through a failure assumes storage-safe
  workloads — journalled filesystems and appendable recordings (yes for this
  build); some databases need more care.

Layered escalation, monitoring that counts the saves, and a human who reads the
counts — that's the whole discipline.

<div class="knowledge-check" data-quiz data-correct-msg="Right — only an independent hardware timer can reboot a board whose kernel is frozen." markdown="0">
  <p class="knowledge-check__q">Quick check: why does a hung kernel need a hardware watchdog rather than a supervisor process?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Supervisor processes are too slow to notice kernel problems</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">When the kernel freezes no software runs at all — only an independent silicon timer can force the reset</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Hardware watchdogs can repair corrupted kernel files during reboot</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Reliability = **automatic recovery per failure class**, not the fantasy of
  preventing all failures.
- **Crashes** → `Restart=on-failure`; **hangs** → `WatchdogSec=` liveness or an
  external **health check** probing real work.
- A **hardware watchdog** (armed via `RuntimeWatchdogSec=`) reboots a frozen
  kernel — the failure software cannot self-escape.
- Design to the **power-cycle test**: power loss and return must resume
  everything unattended.
- Auto-recovery **masks causes** — count the saves in monitoring and debug the
  repeat offenders.

Next up: [Remote administration](/learn/embedded/remote-administration/).
