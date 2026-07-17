---
slug: history-and-shortcuts
title: History, tab completion & shortcuts
description: Work faster at the shell — recall past commands with the history list and the up arrow, search them instantly with Ctrl-R reverse search, let Tab complete commands and paths, use line-editing shortcuts, and save favourite commands as aliases.
keywords: shell history, tab completion, Ctrl-R, keyboard shortcuts, aliases, bashrc, reverse search, productivity, command line, line editing
level: beginner
status: full
prereq:
  - first-commands
faq:
  - q: "How do I re-run a previous command in the terminal?"
    a: "Press the up arrow to step back through recent commands, one per press, and Enter to run the one you want. For the command you just ran, type !! and Enter. The history command lists your recent commands with numbers, and !123 re-runs the one numbered 123. Recalling a command is faster and safer than retyping it."
  - q: "What does Ctrl-R do in the terminal?"
    a: "Ctrl-R starts a reverse search of your command history. Type any fragment of a past command and the shell shows the most recent match as you type. Press Ctrl-R again to step to older matches, Enter to run the one shown, or the left/right arrow to edit it first. It is the single biggest speed-up at the prompt."
  - q: "What is tab completion?"
    a: "Tab completion lets the shell finish what you are typing. Press Tab and it completes a command name, filename, or path as far as it can. If more than one thing matches, press Tab twice to list the options. It saves keystrokes and, because the shell fills in the exact name, it prevents typos in long paths."
  - q: "How do I make a command alias that lasts?"
    a: "An alias gives a short name to a longer command, for example alias ll='ls -lah'. Typed at the prompt it lasts only for that session. To keep it, add the same line to the ~/.bashrc file in your home directory, which the shell reads each time it starts, so your aliases are there every session."
---

# History, tab completion & shortcuts

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The shell remembers everything you type, so you rarely have to type it twice.
Recall past commands with the **history** list and the arrow keys, or jump
straight to one with **Ctrl-R** reverse search. Let **Tab completion** finish
command names and paths for you — fewer keystrokes, fewer typos. And save the
long commands you run often as **aliases**. A handful of these turn slow,
error-prone typing into fast, confident work at the prompt. New here? Start with
[your first commands](/learn/linux-cli/first-commands/).
</div>

Nobody who is quick at the command line is a fast typist — they just retype
almost nothing. The shell keeps a record of what you've run and offers several
ways to bring it back, complete it, or edit it in place. Learn a few of these and
the terminal stops feeling like a typing test.

## Command history

Every command you run is saved to a **history** list. The simplest way to reach
it is the **up arrow**: press it to recall the previous command, press it again
to go further back, one command per press. The **down arrow** moves forward
again. When you land on the one you want, press Enter to run it — or edit it
first.

To see the list itself, run **history**:

```console
$ history
  1  pwd
  2  ls -l
  3  cd Documents
  4  echo hello
```

Each command has a number, and a couple of shortcuts use them:

- **`!!`** re-runs the **last** command exactly. Handy after a command fails for
  lack of permission — `sudo !!` re-runs it as root.
- **`!n`** re-runs the command with number *n* from the list, so `!2` here
  re-runs `ls -l`.

The habit worth building: before you retype something, ask whether you can
**recall** it instead. Re-running is faster and avoids fresh typos.

## Reverse search — Ctrl-R

Arrow keys are fine for something a few lines back, but not for a command you ran
twenty commands ago. For that, press **Ctrl-R**. This starts a **reverse
search**: type any fragment of the command you want, and the shell shows the most
recent match as you type.

```console
(reverse-i-search)`ssh': ssh matt@scanner.local
```

Here, typing just `ssh` found the last `ssh` command. From there:

- Press **Ctrl-R** again to step to the **next older** match, and again to keep
  cycling back through matches.
- Press **Enter** to run the command shown.
- Press the **left or right arrow** to drop the match onto the prompt so you can
  edit it before running.
- Press **Ctrl-C** to give up and clear the line.

Once this is in your fingers it becomes the single biggest speed-up at the
prompt — you stop remembering commands and start remembering *fragments* of them.

## Tab completion

The **Tab** key asks the shell to finish what you're typing. It works on command
names, filenames, and paths. Start typing and press Tab:

```console
$ cd Doc<Tab>
$ cd Documents/
```

The shell filled in the rest of `Documents` for you. If there is only one match,
Tab completes it fully. If **several** things match, a single Tab completes as far
as they share, and pressing **Tab twice** lists all the options so you can see
what's available:

```console
$ cd D<Tab><Tab>
Desktop/    Documents/    Downloads/
```

Beyond saving keystrokes, Tab completion prevents typos: because the shell fills
in the **exact** name, long or awkward filenames come out right every time. If
Tab does nothing, there's usually no match — often a sign you're not where you
think you are, or the name is spelled differently.

## Line-editing shortcuts

You don't have to hold down the arrow keys to fix the start of a long line. The
shell understands a set of editing shortcuts that jump and delete by position:

- **Ctrl-A** — jump to the **start** of the line.
- **Ctrl-E** — jump to the **end** of the line.
- **Ctrl-U** — delete from the cursor **to the start** of the line.
- **Ctrl-K** — delete from the cursor **to the end** of the line.
- **Ctrl-W** — delete the **word** before the cursor.
- **Ctrl-L** — **clear** the screen (same as `clear`), keeping your line intact.
- **Ctrl-C** — **cancel** the current line or stop a running command.
- **Ctrl-D** — signal **end of input**; on an empty prompt it logs out of the shell.
- **Ctrl-Z** — **suspend** the running program and drop back to the prompt (more
  on this in [processes](/learn/linux-cli/processes/)).

You won't use all of these at once. Ctrl-A, Ctrl-E, and Ctrl-U alone save a lot
of arrow-key holding when you need to fix the front of a long command.

## Aliases

When you find yourself typing the same long command over and over, give it a
short name with an **alias**:

```console
$ alias ll='ls -lah'
$ ll
```

Now `ll` runs `ls -lah` — a long, detailed listing — in two keystrokes. You can
alias anything you type often. There's one catch: an alias typed at the prompt
lasts only for the **current session**. Close the terminal and it's gone.

To make aliases **persist**, put them in **~/.bashrc**, a file in your home
directory that the shell reads every time it starts. Add a line like the one
above to that file, and `ll` is waiting for you in every new session. Editing
`~/.bashrc` is also where you'll set up your shell environment later — see
[environment variables](/learn/linux-cli/environment-variables/).

## Build the muscle memory

None of this helps until it's automatic, and you can't drill ten shortcuts at
once. Pick **two or three** — say the up arrow, Ctrl-R, and Tab — and use them
deliberately until your fingers reach for them without thought. Then add a couple
more. Within a week the fast way *is* your only way, and retyping full commands
will feel as slow as it actually is.

<div class="knowledge-check" data-quiz data-correct-msg="Right — Ctrl-R searches your history by fragment, the fastest way back to an old command." markdown="0">
  <p class="knowledge-check__q">Quick check: you typed a long command 20 lines ago and remember part of it. What's the fastest way to re-run it?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Hold the up arrow until it scrolls past</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Ctrl-R and type the fragment (reverse search)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Retype the whole command carefully</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The shell saves every command; **up/down arrows** and the **history** command
  bring them back, and **`!!`** / **`!n`** re-run them.
- **Ctrl-R** searches history by fragment — press it again to cycle, Enter to
  run — the single biggest speed-up at the prompt.
- **Tab** completes commands, filenames, and paths; **double-Tab** lists the
  options, and the exact name means fewer typos.
- **Line-editing shortcuts** (Ctrl-A/E, Ctrl-U/K, Ctrl-W, Ctrl-L, Ctrl-C) move
  and delete without the arrow keys.
- **Aliases** name a long command; put them in **~/.bashrc** so they persist.
- Pick two or three and use them until they're automatic.

Next up: Module 5 turns commands into programs — [your first shell script](/learn/linux-cli/first-shell-script/)
