---
slug: functions-and-exit-codes
title: Functions, exit codes & error handling
description: How to write reusable shell functions, read a command's exit code with $?, react to failure with if and && / ||, and make scripts fail loudly with set -euo pipefail and trap for cleanup — turning a throwaway script into one you can trust.
keywords: bash function, exit code, exit status, $?, set -e, error handling, pipefail, trap, robust scripts, return status, set -u, unset variables
level: advanced
status: full
prereq:
  - conditionals-and-loops
faq:
  - q: What is the difference between return and exit in a shell script?
    a: "return ends a function and hands back a status code to whoever called it, leaving the script running. exit ends the whole script (and the shell it runs in) with the status you give it. Use return N inside a function to report success or failure; use exit N to stop the script for good."
  - q: What does $? mean in bash?
    a: "It is the exit status of the most recently finished command. 0 means success and any non-zero value means some kind of failure. Read it immediately, because the very next command overwrites it — capture it in a variable like status=$? if you need it later."
  - q: What does set -euo pipefail actually do?
    a: "It bundles three safety switches. set -e stops the script the moment any command fails, set -u turns a reference to an unset variable into an error instead of an empty string, and set -o pipefail makes a pipeline fail if any command in it fails, not just the last one. Together they make a script stop at the first real problem instead of charging on with bad data."
  - q: Why does my script keep going after a command fails?
    a: "By default the shell runs the next line regardless of whether the last one worked — a failed command just sets a non-zero exit code and execution continues. Add set -e (usually as part of set -euo pipefail) near the top so an unhandled failure stops the script, or check the exit code yourself with if or && / ||."
---

# Functions, exit codes & error handling

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Two ideas turn a fragile script into a dependable one. **Functions** let you name
a block of commands and reuse it, keeping scripts short and DRY. **Exit codes**
tell you whether each command worked — `0` is success, non-zero is failure, and
you read the last one from **`$?`**. Cap it off with the safety header
**`set -euo pipefail`** so failures stop the script instead of silently
corrupting the run. New to `if` and loops? Start with
[conditionals & loops](/learn/linux-cli/conditionals-and-loops/).
</div>

Anyone can string a few commands into a file and call it a script. The difference
between that and something you'd trust to run unattended — on a schedule, against
real data — comes down to two habits: pulling repeated work into **functions**,
and taking **exit codes** seriously so the script knows when something went wrong.

## Functions

A **function** is a named block of commands you can call as many times as you
like. Define one with a name, parentheses, and a body in braces:

```bash
greet() {
  echo "Hello, $1 — today is $(date +%F)"
}

greet Matt
greet World
```

Arguments arrive inside the function as **`$1`**, **`$2`**, and so on — exactly
like a script's own arguments, but scoped to the call. You **call** a function
just by writing its name followed by any arguments; no special syntax needed.

A function reports back in one of two ways. Use **`return N`** to hand back a
status code (an integer, same as exit codes below), or **`echo`** to produce
output the caller can capture:

```bash
disk_free() {
  df -h / | awk 'NR==2 {print $4}'
}

free=$(disk_free)
echo "Free space: $free"
```

Pulling repeated logic into a function keeps a script **DRY** — "don't repeat
yourself." Fix a bug once, in the function, instead of in five copied blocks.

## Exit codes

Every command returns an **exit status** when it finishes: a number from `0` to
`255`. The convention is simple and universal:

- **`0`** means **success**.
- **any non-zero value** means **failure** (different numbers can signal
  different kinds of failure).

The shell stores the status of the **last** command in the special variable
**`$?`**. Read it right away — the next command overwrites it:

```console
$ ls /etc/hostname
/etc/hostname
$ echo $?
0
$ ls /nope
ls: cannot access '/nope': No such file or directory
$ echo $?
2
```

Your own script sets its status with **`exit N`**. `exit 0` says "all good";
`exit 1` (or any non-zero) tells whatever called your script that something went
wrong — which is how schedulers and other scripts know to react.

## Reacting to failure

Because every command yields a pass/fail status, you can branch on it. An **`if`**
tests a command's exit code directly — no `$?` needed:

```bash
if ping -c1 gophertrunk.local; then
  echo "host is up"
else
  echo "host is unreachable"
fi
```

For quick one-liners, chain with **`&&`** ("and — run the next only if this
succeeded") and **`||`** ("or — run the next only if this failed"):

```bash
mkdir -p /var/log/gt && echo "log dir ready"
systemctl restart gophertrunk || echo "restart failed" >&2
```

A common idiom packs both together — do a thing, report either way:

```bash
cmd && echo ok || echo failed
```

## Fail loudly, not silently

By default the shell shrugs off failures: a command fails, sets a non-zero code,
and the script marches on to the next line — often making a mess with bad data.
Three switches fix that, and you set them near the top of the script:

- **`set -e`** — **exit** the moment any unhandled command fails.
- **`set -u`** — treat a reference to an **unset variable** as an error, instead
  of silently substituting an empty string (which turns `rm -rf "$dir/"` into
  `rm -rf /` when `$dir` is a typo).
- **`set -o pipefail`** — make a **pipeline** fail if *any* command in it fails,
  not just the last one. Without it, `false | true` reports success.

You'll almost always see them combined into one header line:

```bash
#!/usr/bin/env bash
set -euo pipefail
```

That single line prevents a whole class of nasty surprises: half-finished runs,
commands acting on empty variables, and pipe failures hidden behind a successful
final stage. It's the first thing to add to any script you mean to keep.

## trap for cleanup

Sometimes a script leaves something behind — a temp file, a lock — that must be
removed **however** the script ends, success or failure. **`trap`** registers a
command to run when the script exits:

```bash
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

# ... work with "$tmp" freely ...
```

The `EXIT` trap fires on normal exit *and* when `set -e` aborts the script, so the
temp file is cleaned up either way — no stray files left when something breaks.

## A robust example

Putting it together: the safety header, a function, an explicit exit-code check,
and a cleanup trap.

```bash
#!/usr/bin/env bash
set -euo pipefail

tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

fetch_channels() {
  local host=$1
  curl -fsS "http://$host/channels.json" > "$tmp"
}

if fetch_channels "gophertrunk.local"; then
  echo "Fetched $(wc -l < "$tmp") lines of channel data"
else
  echo "Could not reach the scanner" >&2
  exit 1
fi
```

`curl -f` makes a failed HTTP request return non-zero, so the `if` actually
notices; the trap tidies the temp file whether the fetch works or not.

<div class="knowledge-check" data-quiz data-correct-msg="Right — 0 means the command succeeded." markdown="0">
  <p class="knowledge-check__q">Quick check: a command finishes with exit code 0. What does that mean?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Success</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">An error occurred</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The command was skipped</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Functions** — `name() { … }` — name reusable blocks; arguments are `$1`,
  `$2`; `return N` reports a status, `echo` produces output. Keep scripts DRY.
- Every command has an **exit code**: `0` success, non-zero failure; the last one
  lives in **`$?`**, and your script sets its own with `exit N`.
- **React to failure** with `if`, or chain with `&&` / `||`
  (`cmd && echo ok || echo failed`).
- **`set -euo pipefail`** makes a script fail loudly: stop on error, error on
  unset variables, and catch failures anywhere in a pipe.
- **`trap '…' EXIT`** runs cleanup however the script ends — the reliable way to
  remove temp files and release locks.

Next up: [scheduling with cron & timers](/learn/linux-cli/scheduling-with-cron/)
