---
slug: scheduling-with-cron
title: Scheduling with cron & timers
description: How to run a script automatically on a schedule — the classic cron daemon, editing your jobs with crontab -e, the five-field cron syntax for minutes through days, the @reboot shortcut, where cron output goes, common environment gotchas, and systemd timers as the modern alternative.
keywords: cron, crontab, cron schedule, systemd timer, scheduled task, automation, @reboot, five field cron, cron job, crontab -e, scheduled script
level: intermediate
status: full
prereq:
  - first-shell-script
faq:
  - q: What is cron used for?
    a: "cron is the classic Linux scheduler: a background service that runs commands automatically at times you choose. You give it a schedule and a command, and it fires that command every night, every few minutes, at boot, or on whatever pattern you set. It is the standard way to automate backups, cleanups, and any script you would otherwise have to remember to run by hand."
  - q: How do I edit my cron jobs?
    a: "Run crontab -e to open your personal crontab in an editor, add or change lines, then save and exit. Each line is one scheduled job. Run crontab -l to list what you currently have scheduled without editing. Your jobs are stored per-user, so what you see is your own schedule, separate from other users and the system crontab."
  - q: What does the cron schedule 0 2 * * * mean?
    a: "The five fields are minute, hour, day-of-month, month, and day-of-week. 0 2 * * * means minute 0 of hour 2 on every day, every month, every weekday — in other words, 2:00 AM every day. A star in a field means every value of that field, so only the minute and hour are pinned here."
  - q: Why does my cron job work by hand but fail on schedule?
    a: "cron runs with a stripped-down environment and a minimal PATH, not the one from your interactive shell. Commands found by name at your prompt may not be found by cron, and variables you rely on may be unset. Use absolute paths to programs and files, set any variables the job needs inside the job or script, and redirect output to a log so you can see what actually happened."
---

# Scheduling with cron & timers

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**cron** is the classic Linux scheduler that runs commands automatically. Edit
your jobs with **`crontab -e`**; each line is **the five fields** — minute, hour,
day-of-month, month, day-of-week — followed by the command. Use it to run a
[script](/learn/linux-cli/first-shell-script/) nightly, hourly, or at boot with
`@reboot`. **systemd timers** are the modern alternative with richer logging and
dependencies. Either way, use absolute paths and send output to a log.
</div>

You have a script that does something useful — a backup, a cleanup, a health
check. Running it by hand every night gets old fast, and eventually you forget.
Scheduling hands that job to the machine: it runs on time, every time, whether or
not you remember. Linux gives you two ways to do this, and both are worth knowing.

## cron

**cron** is the original Unix scheduler and it is still everywhere. A background
service (the cron *daemon*) wakes up every minute, checks a table of jobs, and runs
any whose schedule matches the current time. You never talk to the daemon directly —
you just edit the table, called a **crontab**.

Two commands cover almost everything:

- `crontab -e` — **edit** your jobs. This opens your personal crontab in an editor;
  add lines, save, and exit. cron picks up the changes immediately.
- `crontab -l` — **list** your jobs, so you can see what is scheduled without editing.

Each non-comment line in the crontab is one job: a schedule followed by the command
to run.

## The five-field syntax

A cron line is **five time fields and then the command**. The fields, in order:

| Field | Meaning | Range |
| --- | --- | --- |
| 1 | minute | 0–59 |
| 2 | hour | 0–23 |
| 3 | day of month | 1–31 |
| 4 | month | 1–12 |
| 5 | day of week | 0–6 (0 = Sunday) |

A `*` in any field means **every value** of that field. You can also use lists
(`1,15`), ranges (`9-17`), and steps (`*/5` = every 5th value). A few examples:

```cron
# minute hour day-of-month month day-of-week  command
0 2 * * *      /home/pi/backup.sh        # 2:00 AM every day
*/5 * * * *    /home/pi/check.sh          # every 5 minutes
30 8 * * 1     /home/pi/weekly-report.sh  # 8:30 AM every Monday
0 0 1 * *      /home/pi/monthly.sh        # midnight on the 1st of each month
@reboot        /home/pi/startup.sh        # once, each time the machine boots
```

The trick to reading a line is to work field by field: pin the minute and hour
first, then decide which days. If you find yourself unsure what a pattern means,
translate each field out loud — "minute 0, hour 2, any day, any month, any
weekday" — rather than guessing. `@reboot` is a special shortcut that runs the job
once at startup instead of on a clock schedule.

## Where output goes

By default, cron **captures whatever your job prints** (both normal output and
errors) and tries to email it to you. On a typical home machine there is no mail
system, so that output is effectively **discarded** — which is why a broken job can
fail silently and leave you no clue.

The fix is to redirect output to a log file yourself, using the same
[pipes and redirection](/learn/linux-cli/pipes-and-redirection/) you already know:

```cron
0 2 * * *  /home/pi/backup.sh >> /home/pi/backup.log 2>&1
```

Here `>>` appends normal output to the log and `2>&1` sends errors to the same
place. Now every run leaves a record you can read afterwards.

## Common gotchas

The single biggest surprise: **cron does not run with your shell's environment.**
It uses a minimal set of variables and a short **PATH**, so a command that works
fine when you type it may not be found when cron runs it. To stay out of trouble:

- **Use absolute paths** for programs and files — `/usr/bin/rsync`, not `rsync`;
  `/home/pi/data`, not `~/data` or a relative path.
- **Set any variables the job needs explicitly**, inside the script or at the top
  of the crontab, rather than assuming they are inherited. See
  [environment variables](/learn/linux-cli/environment-variables/) for why the
  environment differs.
- **Test the command by hand first**, exactly as written in the crontab, before you
  schedule it. If it fails at the prompt it will certainly fail under cron.

Most "my cron job never runs" problems are really "my cron job ran and failed
because of PATH or a relative path" — the log from the previous section is how you
find that out.

## systemd timers

On modern Linux, **systemd timers** are the newer alternative to cron. Instead of
one crontab line, you write two small units: a **service** unit that says *what* to
run, and a **timer** unit that says *when* to run it. The timer triggers the
service on schedule.

That is more files than a cron line, but you get real advantages: output lands in
the system journal automatically (no manual log redirection), timers can depend on
other services or wait for the network, and a `Persistent=true` option can run a
missed job after the machine was off. Timers build directly on the
[services and systemd](/learn/linux-cli/services-and-systemd/) model, so if you are
already running something as a service, adding a timer to it is a natural next step.

For a quick personal job, cron is still the fastest thing to reach for. For
anything that needs solid logging or must coordinate with other services, timers
are usually the better home.

## A practical example

Say you want to clean up old capture files every night. You already have a script,
`/home/pi/cleanup.sh`, that deletes files older than a week. Schedule it with cron:

```cron
15 3 * * *  /home/pi/cleanup.sh >> /home/pi/cleanup.log 2>&1
```

That runs it at **3:15 AM every day**, off-peak, and appends every run's output to
a log. Before scheduling, run `/home/pi/cleanup.sh` by hand once to confirm it
behaves, then add the line with `crontab -e`. Check the log the next morning to
confirm it fired — and now the machine remembers so you do not have to.

<div class="knowledge-check" data-quiz data-correct-msg="Right — minute 0, hour 2, every day." markdown="0">
  <p class="knowledge-check__q">Quick check: what does the cron schedule <code>0 2 * * *</code> mean?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Run every 2 minutes</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Run every day at 2:00 AM</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Run on the 2nd of every month</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **cron** runs commands automatically; edit your jobs with **`crontab -e`** and
  list them with **`crontab -l`**.
- A cron line is **five fields** — minute, hour, day-of-month, month, day-of-week —
  then the command; `*` means "every".
- `@reboot` runs a job once at startup instead of on a clock.
- cron discards output by default — **redirect it to a log** so failures are visible.
- cron's **environment is minimal**: use absolute paths, set variables explicitly,
  and test by hand first.
- **systemd timers** are the modern alternative, with journal logging and service
  dependencies built in.

Next up: Module 6 puts it to work on real machines — [SSH & working remotely](/learn/linux-cli/ssh-and-remote/)
