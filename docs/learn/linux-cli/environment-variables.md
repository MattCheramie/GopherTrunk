---
slug: environment-variables
title: Environment variables & PATH
description: Understand environment variables and PATH on Linux — setting shell variables with NAME=value, reading them with $NAME, exporting them to programs, how PATH decides which commands run, common variables like HOME and EDITOR, and making settings stick in ~/.bashrc.
keywords: environment variable, PATH, export, bashrc, shell variable, command not found, HOME, EDITOR, printenv, LANG, source
level: intermediate
status: full
prereq:
  - the-shell
faq:
  - q: "What is an environment variable in Linux?"
    a: "An environment variable is a named setting that the shell and the programs it launches can read. You set one with NAME=value and read it with $NAME. Once exported, it's passed to every program the shell starts, which is how tools discover things like your home directory, preferred editor, or language."
  - q: "How do I add a directory to my PATH?"
    a: "Append it to the existing PATH, keeping the old value with a colon separator, then export it: export PATH=\"$PATH:/new/dir\". Put that line in ~/.bashrc so it applies to every new shell. PATH is a colon-separated list of directories the shell searches to find the commands you type."
  - q: "Why do I get command not found?"
    a: "The shell looks for a command by searching each directory in your PATH, in order. If the program isn't in any of those directories, you get command not found even though the file exists. Either call it by its full path, or add its directory to PATH."
  - q: "What's the difference between a shell variable and an environment variable?"
    a: "A plain shell variable lives only in your current shell — programs you launch can't see it. Running export on it turns it into an environment variable, which is copied into every program the shell starts. Both are read the same way, with $NAME."
---

# Environment variables & PATH

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Variables hold settings the shell and programs read. Set one with
**`NAME=value`** and read it back with **`$NAME`**. **`export`** promotes a
variable so the programs your shell launches can see it too. The most important
one is **`PATH`** — the list of directories the shell searches to decide which
commands you can actually run. If the shell itself still feels unfamiliar, start
with [the shell](/learn/linux-cli/the-shell/).
</div>

Much of how Linux behaves is steered by a handful of named settings the shell
keeps in memory. Learn to read and set them and a lot of "why is it doing that?"
moments turn into a one-line answer.

## Shell variables

A variable is just a name with a value attached. You create one by writing the
name, an equals sign, and the value — **no spaces around the `=`**:

```
name=Matt
```

Read it back by putting a `$` in front of the name. `echo` prints the result:

```
echo $name
```

That prints `Matt`. The spaces rule matters: `name = Matt` (with spaces) is *not*
an assignment — the shell reads it as a command called `name`. Keep the `=` tight.

A variable made this way is a **shell variable**: it lives only in your *current*
shell. Close the terminal, or open a second one, and it's gone. Programs you launch
from this shell can't see it either — which is where `export` comes in.

## Environment variables & export

To make a variable visible to the programs your shell starts, **`export`** it:

```
export EDITOR=nano
```

Now any program the shell launches — a text editor hook, a build script, `git` —
can read `EDITOR` and act on it. That exported copy is an **environment variable**.
The difference is only visibility:

- **Shell variable** — lives in this shell only; child programs can't see it.
- **Environment variable** — exported, so it's copied into every program the shell
  launches.

You read both the same way, with `$NAME`. To see everything that's currently
exported, use **`env`** or **`printenv`**:

```
printenv
```

`printenv HOME` prints just one variable's value, which is handy when the full list
scrolls off the screen.

## PATH — how commands are found

When you type a command like `ls` or `gophertrunk`, the shell doesn't search your
whole disk for it. It looks through **`PATH`** — a colon-separated list of
directories — and runs the first matching program it finds:

```
echo $PATH
```

You'll see something like `/usr/local/bin:/usr/bin:/bin`. The shell checks each of
those directories, left to right, and stops at the first match.

This is exactly why **`command not found`** happens: the program may well exist, but
if it isn't in any PATH directory, the shell never sees it. The
[`which`](/learn/linux-cli/finding-things/) command shows you which directory a
command was found in — the same search the shell just did:

```
which gophertrunk
```

To make a program in a new directory runnable by name, add its directory to PATH,
keeping the existing value so you don't lose the standard directories:

```
export PATH="$PATH:$HOME/bin"
```

Now anything in `~/bin` runs just by typing its name.

## Common variables

A few environment variables show up constantly:

- **`HOME`** — the path to your home directory (what `~` expands to).
- **`USER`** — your login name.
- **`PWD`** — the directory you're currently in.
- **`SHELL`** — the path to your login shell, e.g. `/bin/bash`.
- **`EDITOR`** — the text editor programs open when they need one from you.
- **`LANG`** — your language and character-encoding setting, e.g. `en_US.UTF-8`.

You rarely set the first four yourself — the system fills them in — but reading them
is a quick way to answer "where am I / who am I / what's my editor?".

## Setting them for good

An `export` typed at the prompt lasts only for that **session** — close the shell and
it's forgotten. To make a variable stick, put the same line in a file the shell reads
when it starts, usually **`~/.bashrc`** (or **`~/.profile`**):

```
export EDITOR=nano
export PATH="$PATH:$HOME/bin"
```

New shells pick it up automatically. To apply the change to the shell you're already
in — without opening a new one — reload the file with **`source`**:

```
source ~/.bashrc
```

## A note on secrets

Environment variables are the standard place to hand a program a secret like an API
key. Instead of writing the key into your code — where it can leak into version
control or logs — you set it in the environment and have the program read it at run
time:

```
export API_KEY=sk-your-secret-here
```

The program reads `API_KEY` from its environment and never sees a hard-coded value.
This is exactly the pattern used when wiring up AI features — see
[your first API call](/learn/building-ai/your-first-api-call/), which reads its key
straight from an environment variable for this reason.

<div class="knowledge-check" data-quiz data-correct-msg="Right — PATH is the list of directories the shell searches to find a command." markdown="0">
  <p class="knowledge-check__q">Quick check: what does the PATH variable control?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">How much memory a program is allowed to use.</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Which directories the shell searches to find a command you type.</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The order files are listed by ls.</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Set a variable with **`NAME=value`** (no spaces around `=`) and read it with **`$NAME`**.
- A plain **shell variable** lives in your current shell; **`export`** turns it into an environment variable that child programs can see.
- List what's exported with **`env`** or **`printenv`**.
- **`PATH`** is a colon-separated list of directories the shell searches for commands — an empty or wrong PATH is what causes **`command not found`**.
- Common ones: **`HOME`**, **`USER`**, **`PWD`**, **`SHELL`**, **`EDITOR`**, **`LANG`**.
- Make a setting permanent by adding the `export` line to **`~/.bashrc`**, then reload with **`source`**.

Next up: [history, tab completion & shortcuts](/learn/linux-cli/history-and-shortcuts/)
