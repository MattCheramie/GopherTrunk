---
slug: monitoring-and-logs
title: Monitoring, disk & logs
description: How to tell whether a Linux machine is healthy and where to look when it isn't — checking free space with df and du, memory and load with free, uptime, top and htop, and reading logs under /var/log and the systemd journal with journalctl.
keywords: df, du, free, htop, top, disk space, load average, /var/log, journalctl, logs, uptime, monitoring
level: intermediate
status: full
prereq:
  - processes
faq:
  - q: How do I check free disk space on Linux?
    a: "Run df -h. It lists every mounted filesystem with its size, how much is used, how much is free, and the percentage full, all in human-readable units (G, M). To find what's eating a particular directory, use du -sh <dir> for its total, or du -sh * inside a folder to size each item."
  - q: How do I check memory and CPU load on Linux?
    a: "free -h shows total, used, and available memory in readable units. uptime prints the load average — roughly how many processes were waiting to run over the last 1, 5, and 15 minutes. For a live, per-process view of CPU and memory, run top (everywhere) or htop (friendlier, if installed)."
  - q: Where are logs stored on Linux?
    a: "Two places. Traditional services write plain-text logs under /var/log — you read them with tail, less, or grep. Systemd-managed services log to the systemd journal instead, which you read with journalctl. Many modern systems use both, so check the journal if /var/log looks empty."
  - q: How do I watch a log update live?
    a: "For a text log, tail -f /var/log/somefile follows it and prints new lines as they arrive. For a systemd service, journalctl -u <service> -f does the same from the journal. Press Ctrl-C to stop following."
---

# Monitoring, disk & logs

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A handful of commands tell you whether a machine is healthy and where to look when
it isn't. **df / du** answer "am I out of disk?"; **free / htop** answer "am I out
of memory or CPU?"; **/var/log** and **journalctl** hold the **logs** that explain
what actually went wrong. Learn these and a misbehaving box stops being a mystery.
You'll want [processes](/learn/linux-cli/processes/) under your belt first.
</div>

Servers don't usually fail loudly. A recording stops, a decode goes quiet, a service
won't start — and the machine looks fine until you ask it the right question. These are
the questions.

## Disk space — df & du

A full disk is one of the most common causes of failure on a long-running box: once
there's no room left, programs **can't write** — logs stop, recordings truncate, and
services fall over with confusing errors that have nothing obviously to do with disk.

`df -h` shows free space per filesystem:

```
$ df -h
Filesystem      Size  Used Avail Use% Mounted on
/dev/root        59G   48G  8.2G  86% /
/dev/sda1       466G  410G   56G  89% /mnt/captures
```

The `Use%` column is what you scan first. Anything near 100% is a problem waiting to
happen. When a filesystem is filling up, find the culprit with `du` (disk usage):

```
$ du -sh /mnt/captures        # total size of one directory
410G    /mnt/captures
$ du -sh /mnt/captures/*      # size of each item inside it
380G    /mnt/captures/2026-07
 30G    /mnt/captures/old
```

`df` tells you a filesystem is full; `du` tells you *what's* filling it so you know what
to delete or move.

## Memory & load — free, uptime, htop

`free -h` shows memory at a glance:

```
$ free -h
               total        used        free      available
Mem:           7.6Gi       2.1Gi       3.9Gi       5.2Gi
Swap:          1.0Gi          0B       1.0Gi
```

The **available** column is the honest one — memory the system can hand out right now.
If it's near zero and swap is filling, the machine is under memory pressure and will
slow down.

`uptime` prints how long the box has been up and its **load average**:

```
$ uptime
 14:22:01 up 6 days,  3:11,  2 users,  load average: 0.42, 0.55, 0.61
```

The three numbers are the average number of processes waiting to run over the last 1, 5,
and 15 minutes. As a rough rule, compare them to your CPU core count: a load of 4.0 on a
4-core machine means it's fully busy; well above that means work is queuing up.

For a live, per-process view of who's using the CPU and memory, run `top` (installed
everywhere) or `htop` (colour-coded and friendlier, if installed). These are the same
tools you met in [processes](/learn/linux-cli/processes/) — here you're using them to
spot a runaway program.

## Logs live in two places

When something breaks, the logs are where the reason is written down. On a modern Linux
box they live in two places:

- **Text files under `/var/log`** — traditional services write plain-text logs here.
  Read them with `less`, search with `grep`, or watch one live with `tail -f`:

  ```
  $ tail -f /var/log/syslog
  ```

  (Opening and paging files is covered in
  [viewing & editing](/learn/linux-cli/viewing-and-editing/).)

- **The systemd journal** — services managed by systemd log to a binary journal instead,
  which you read with `journalctl`. Filter to one service with `-u`, and follow live with
  `-f`:

  ```
  $ journalctl -u gophertrunk -f
  ```

  (More on systemd services in
  [services & systemd](/learn/linux-cli/services-and-systemd/).)

Many systems use both, so if `/var/log` looks empty for a service, check the journal —
and vice versa.

## Reading logs to diagnose

Logs can be huge, so you don't read them front to back. The routine:

1. **Start at the end.** The most recent lines are usually the ones that matter —
   `tail -n 50 <file>`, or `journalctl -u <service> -e` to jump to the end.
2. **Search for trouble words.** `grep -i error /var/log/syslog`, or
   `journalctl -u <service> -p err` to show only error-priority entries.
3. **Correlate timestamps.** Note *when* the symptom appeared, then look at what the log
   was doing at that moment. The line just *before* the failure is often the real cause.

A short worked flow — "why did my service stop?":

```
$ journalctl -u gophertrunk -e          # jump to the latest entries
... Jul 17 14:05:12  writing capture to /mnt/captures/2026-07/...
... Jul 17 14:05:12  write failed: no space left on device
... Jul 17 14:05:12  shutting down
```

The last lines name the cause directly: the disk filled, the write failed, the service
gave up. That points you straight back to `df -h` — the loop closes.

## Applied to GopherTrunk

- **Check disk before a long recording.** A multi-hour capture can be tens of gigabytes;
  run `df -h` on the target filesystem first so you don't fill it mid-record and lose the
  tail of the capture.
- **Watch CPU on a Pi.** On a Raspberry Pi the decoder can saturate a core. Keep `htop`
  open while you tune a new setup — if one core is pinned at 100% and audio stutters, the
  board is the bottleneck, not the radio.
- **Read the daemon's logs when a decode misbehaves.** If GopherTrunk runs as a service,
  `journalctl -u gophertrunk -f` shows what it's doing in real time; if it logs to a file,
  `tail -f` that file instead. Either way the "why" is usually right there. Setting the
  daemon up is covered in
  [running GopherTrunk on Linux](/learn/linux-cli/running-gophertrunk-on-linux/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — df -h lists free space per filesystem in human-readable units." markdown="0">
  <p class="knowledge-check__q">Quick check: which command shows free disk space per filesystem?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">free -h</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">df -h</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">journalctl -f</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **`df -h`** shows free space per filesystem; **`du -sh <dir>`** shows what's using it.
- A **full disk** breaks writes and is a top cause of quiet failures — check it early.
- **`free -h`** shows memory, **`uptime`** shows load average, **`top` / `htop`** give a
  live per-process view.
- Logs live in **two places**: text files under **`/var/log`** and the systemd journal via
  **`journalctl`** — watch either live with `tail -f` or `journalctl -f`.
- To diagnose, **start at the end**, **search for error/warn**, and **correlate timestamps**.

Next up: [running GopherTrunk on Linux](/learn/linux-cli/running-gophertrunk-on-linux/)
