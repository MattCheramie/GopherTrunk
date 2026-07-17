---
slug: processes
title: Processes & jobs
description: What a Linux process is — a running program with a PID, a parent, and an owner — plus watching them with ps, top, and htop, running commands in the foreground vs. background with &, bg, fg, and jobs, and stopping one with Ctrl-C, kill, and signals.
keywords: linux process, PID, ps command, top, htop, kill, signal, background job, foreground, nohup, SIGTERM, SIGKILL
level: intermediate
status: full
prereq:
  - first-commands
faq:
  - q: How do I see what's running on Linux?
    a: "Use ps for a snapshot — ps aux lists every process with its owner, PID, and CPU/memory use. For a live, self-updating view, run top (installed everywhere) or htop (friendlier, colour-coded, if installed). These show which programs are running, how much they're using, and each one's PID so you can act on it."
  - q: How do I run a command in the background?
    a: "Append & to the command, e.g. long-task &, and the shell starts it in the background and hands your prompt straight back. If a command is already running, press Ctrl-Z to suspend it, then type bg to resume it in the background. Use jobs to list background work in the current shell and fg to pull one back to the foreground."
  - q: How do I stop a Linux process?
    a: "If it's the program running in your terminal right now, press Ctrl-C to interrupt it. For any other process, find its PID (with ps or top) and run kill <PID>, which politely asks it to shut down. If it ignores that, kill -9 <PID> forces it to stop immediately — a last resort, since the program gets no chance to clean up."
  - q: What's the difference between SIGTERM and SIGKILL?
    a: "SIGTERM (the default kill signal) asks a process to shut down and lets it tidy up first — close files, finish writing, exit cleanly. SIGKILL (kill -9) can't be caught or ignored; the kernel stops the process dead with no chance to clean up. Always try SIGTERM first and reach for SIGKILL only when a process is truly stuck."
---

# Processes & jobs

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Every running program is a **process** with a **PID** (a numeric ID). You watch
them with **ps** for a snapshot or **top / htop** for a live view. A command
normally holds your terminal, but you can send it to the **background** with `&`
and juggle **background jobs** with `bg`, `fg`, and `jobs`. To stop one you
**kill** it — really, you send it a **signal**: a gentle "please stop", or, as a
last resort, a forced one. First time in a terminal?
[Start with your first commands](/learn/linux-cli/first-commands/).
</div>

Open a terminal, run a program, and something invisible happens: Linux creates a
*process* to hold it. Understanding processes is what lets you see what your
machine is actually doing, run long tasks without tying up your terminal, and
stop something that has hung — skills you'll lean on the moment you run a
long-lived tool like GopherTrunk.

## What a process is

A **process** is simply a running program. When you launch anything — a command,
an editor, a background service — Linux loads it into memory and gives it a
**PID** (process ID), a unique number that identifies it while it runs. That PID
is the handle you use to watch or stop it later.

Every process also has:

- an **owner** — the [user](/learn/linux-cli/first-commands/) it runs as, which
  decides what it's allowed to touch;
- a **parent** — the process that started it (your shell is the parent of the
  commands you type), forming a tree back to the very first process the system
  started at boot.

Your machine runs *many* processes at once — often hundreds — even when it looks
idle. Most belong to the system and quietly keep it going; the ones you start
are just a handful near the top of that tree.

## Seeing what's running

The classic tool is **`ps`**. On its own it shows only *your* processes in the
current terminal, which is rarely what you want. The common incantation is:

```
ps aux
```

That lists **every** process on the system, one per line, with its owner, PID,
and how much **CPU** and **memory** it's using. It's a *snapshot* — a single
frozen moment — so run it again to see how things have changed.

For a **live** view that refreshes itself, use **`top`**:

```
top
```

`top` fills the screen with the busiest processes, updating every couple of
seconds, so you can watch which program is eating the CPU right now. Press `q` to
quit. Many people install **`htop`**, a friendlier, colour-coded version with
scrolling and mouse support — same idea, nicer to read.

Reading the columns doesn't need to be deep: the **%CPU** column shows the share
of a processor core a program is using, and **%MEM** (or the resident-memory
column) shows how much RAM it holds. A process pinned near 100% CPU or steadily
climbing in memory is usually the one worth investigating. We cover watching a
system properly — including logs — in
[monitoring & logs](/learn/linux-cli/monitoring-and-logs/).

## Foreground vs. background

By default a command runs in the **foreground**: it takes over your terminal, and
you don't get your prompt back until it finishes. That's fine for quick commands,
but useless for something long-running — you'd lose the terminal for the whole
time.

Add an **`&`** to the end and the command runs in the **background** instead:

```
long-task &
```

The shell starts it and hands your prompt straight back, so you can keep working
while it runs. A few controls tie this together:

- **Ctrl-Z** — suspend the foreground program (pauses it, gives you the prompt).
- **`bg`** — resume that suspended program *in the background*.
- **`fg`** — pull a background program back to the **foreground**.
- **`jobs`** — list the background work belonging to this shell.

One catch: background jobs are tied to your shell, so they normally die when you
log out. To let a task **survive logout**, start it with **`nohup`** (`nohup
long-task &`) or **`disown`** a job you've already backgrounded. That's a stop-gap
for a one-off, though — for anything you want running permanently, a service
manager is the right tool, which we get to below.

## Stopping a process

If the program hogging your terminal *right now* needs to go, press **Ctrl-C**.
That interrupts the foreground process and usually ends it cleanly.

For any *other* process — a background job, or something in a different terminal —
you use its PID:

```
kill <PID>
```

Despite the name, `kill` doesn't force anything by default; it **sends a signal**.
A signal is a small message the kernel delivers to a process. The two you'll
actually use:

- **SIGTERM** (what plain `kill` sends) — "please shut down". The process can
  catch it and tidy up first: finish writing, close files, exit cleanly.
- **SIGKILL** — `kill -9 <PID>` — the forced stop. It can't be caught or ignored;
  the kernel ends the process immediately, with no chance to clean up. It's the
  **last resort**, for when a process is well and truly stuck.

Always try plain `kill` first and only reach for `-9` if the process ignores it.

If you don't want to look up a PID, you can act by **name**: **`pkill firefox`**
or **`killall firefox`** signals every process with that name at once. Handy, but
double-check the name — you'll signal *all* of them.

<div class="knowledge-check" data-quiz data-correct-msg="Right — find the PID, send a gentle kill first, and only force it with -9 if it won't budge." markdown="0">
  <p class="knowledge-check__q">Quick check: a program is frozen and isn't responding. From another terminal, how do you stop it?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Unplug the machine and reboot</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Find its PID and kill it (kill -9 as a last resort)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Press Ctrl-C in the other terminal</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Why it matters

Most of what you'll run in earnest is *long-lived*: a web server, a database, a
decoder like **GopherTrunk** listening to the airwaves for hours. Each is just a
process — the same PIDs, signals, and views you've met here apply to all of them.

Starting such a tool with `&` and `nohup` works for a quick test, but it's
fragile: nothing restarts it if it crashes or the machine reboots. That's exactly
the job of a proper **service manager**, which supervises a process for you —
starting it at boot, restarting it on failure, and capturing its output. We build
that up in [services & systemd](/learn/linux-cli/services-and-systemd/).

## Recap

- A **process** is a running program with a **PID**, an **owner**, and a
  **parent**; many run at once.
- **`ps aux`** gives a snapshot; **`top`** and **`htop`** give a live view with
  CPU and memory columns.
- A command holds your terminal in the **foreground**; **`&`** runs it in the
  **background**, and `Ctrl-Z`, `bg`, `fg`, and `jobs` manage the switch.
- **`nohup`** / **`disown`** let a job survive logout — a stop-gap, not a
  real service.
- **Ctrl-C** stops the foreground program; **`kill <PID>`** signals another —
  **SIGTERM** asks nicely, **SIGKILL (`-9`)** forces it as a last resort.
- Long-running tools and daemons are just processes; a service manager supervises
  them for you.

Next up: [installing software with a package manager](/learn/linux-cli/package-management/)
