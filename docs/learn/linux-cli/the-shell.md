---
slug: the-shell
title: The shell explained
description: What the shell actually is — the program that reads what you type and runs it — the terminal-vs-shell distinction, how to read the prompt, and the command-options-arguments pattern behind every command you'll ever type.
keywords: shell, bash, zsh, terminal, prompt, command, argument, option, flag, shell vs terminal, command line syntax, built-in commands
level: beginner
status: full
prereq:
  - why-the-command-line
faq:
  - q: "What is a shell in Linux?"
    a: "The shell is the program that reads the commands you type and runs them. It sits between you and the operating system — you type a line, the shell figures out what you meant, launches the right program (or runs a built-in), and shows you the result. Bash and zsh are the two you'll meet most."
  - q: "What is the difference between a terminal and a shell?"
    a: "The terminal is the window (or app) that shows text and takes your keystrokes; the shell is the interpreter running inside it that actually reads and executes your commands. One is the screen, the other is the brain. You can run the same shell in many different terminals."
  - q: "What do the parts of a command mean?"
    a: "Almost every command follows the pattern command [options] [arguments]. The command is the program to run, options (also called flags, like -l) change how it behaves, and arguments are what it acts on, like a filename or path. In ls -l /home, ls is the command, -l is the option, and /home is the argument."
  - q: "What is the difference between bash and zsh?"
    a: "Both are shells, and for everything in this course they behave the same way. Bash is the default on most Linux systems; zsh is the default on modern macOS. Zsh adds nicer tab-completion and prompts, but the commands and patterns you learn here work in either."
---

# The shell explained

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The **shell** is the program that reads what you type and runs it. It shows you a
**prompt**, waits for a line, and hands the result back. Almost everything you type
follows one pattern — a **command**, then **options** that change how it behaves, then
**arguments** it acts on. Learn that pattern once and every command reads the same way.
</div>

In the [last lesson](/learn/linux-cli/why-the-command-line/) we said the command line
gives you speed, precision, and remote control. This lesson introduces the thing that
actually *does* that work: the shell. Understanding it turns a wall of cryptic text into
a small set of predictable pieces.

## Terminal vs. shell

These two words get used interchangeably, but they're different things.

The **terminal** is the *window* — the app on your screen that displays text and passes
your keystrokes along. On Linux it might be called GNOME Terminal or Konsole; on macOS,
Terminal or iTerm.

The **shell** is the *interpreter* running inside that window. It reads each line you
type, works out what you meant, runs the right program, and prints the result. Common
shells are **bash**, **zsh**, and **sh**.

A useful picture: the terminal is the phone, the shell is the person you're talking to.
The same shell can run inside many different terminals, and one terminal can run
different shells — they're separate jobs.

## The prompt

When a shell is ready for a command, it shows a **prompt** — a bit of text ending in a
symbol, with the cursor waiting after it. A typical prompt packs in some useful context:

```text
matt@raspberrypi:~/gophertrunk$
```

Read left to right, that's the **user** (`matt`), the **host** (`raspberrypi`), and the
**current directory** (`~/gophertrunk`, where `~` means your home folder). Prompts vary
between systems, but most show some version of *who* and *where* you are.

The final symbol follows a widespread convention:

- **`$`** — you're a normal, everyday user.
- **`#`** — you're **root**, the all-powerful administrator, where a typo can wreck the
  system. Seeing `#` is a signal to slow down (more on this in
  [sudo & becoming root safely](/learn/linux-cli/sudo-and-root/)).

Throughout this course, example commands start with `$` to show they're typed at the
prompt. You don't type the `$` yourself — just the part after it.

## Anatomy of a command

Here's the single most useful idea in this whole path. Almost every command follows the
same shape:

```text
command [options] [arguments]
```

The square brackets just mean "optional" — plenty of commands work with none. Let's
break down a real one:

```text
$ ls -l /home
```

- **`ls`** is the **command** — the program to run (here, "list files").
- **`-l`** is an **option** (also called a **flag**) — it *changes how the command
  behaves*, in this case asking for a long, detailed listing.
- **`/home`** is an **argument** — *what the command acts on*, here the directory to
  list.

Change the argument and `ls` lists somewhere else; change the option and it lists the
same place differently. That separation — *what to do*, *how to do it*, *what to do it
to* — is the same for nearly every command you'll meet.

A few details that come up constantly:

- **Short vs. long options.** Many options have a terse form and a readable form that do
  the same thing: `-a` and `--all`, or `-h` and `--help`. Short options can often be
  bundled, so `ls -la` is the same as `ls -l -a`.
- **Options that take a value.** Some options expect their own argument right after
  them, like `head -n 5 file.txt` (show the first `5` lines).
- **Spaces separate everything.** The shell splits your line on spaces, so a filename
  *with* a space needs quoting — `cat "my notes.txt"` — or the shell reads it as two
  arguments.

## Built-ins vs. programs

Not every command is a separate program. A handful are **built into the shell itself** —
`cd` (change directory) is the classic example, because changing *the shell's own*
directory is something only the shell can do.

Most commands, though, are **separate programs** living as files on disk — `ls`, `cat`,
`grep` and the rest. When you type one, the shell searches a list of directories called
your **PATH** to find the matching program, then runs it. That's why you can type `ls`
instead of its full location: it's on the PATH. We'll unpack PATH properly in
[environment variables](/learn/linux-cli/environment-variables/), and you'll meet your
first real commands in [your first commands](/learn/linux-cli/first-commands/).

## Common shells

You'll almost certainly be using one of these:

- **bash** — the default on most Linux distributions and the one most tutorials assume.
- **zsh** — the default on modern macOS, with fancier completion and prompts.
- **sh** — a smaller, older shell that scripts often target for maximum portability.

The good news: for everything in this course they're broadly compatible. The prompt, the
command pattern, and the everyday commands all work the same way. When a difference
actually matters, we'll call it out.

<div class="knowledge-check" data-quiz data-correct-msg="Right — -l is the option; it changes how ls behaves." markdown="0">
  <p class="knowledge-check__q">Quick check: in <code>ls -l /home</code>, which part is the option (flag)?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">ls</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">-l</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">/home</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The **terminal** is the window; the **shell** (bash, zsh, sh) is the interpreter
  inside it that reads and runs your commands.
- The **prompt** shows who and where you are, ending in `$` for a normal user or `#` for
  root.
- Nearly every command follows **`command [options] [arguments]`** — what to run, how,
  and on what.
- Options come in **short** (`-a`) and **long** (`--all`) forms, and some take a value.
- A few commands (like `cd`) are **built into the shell**; most are **programs** the
  shell finds on your **PATH**.

Next up: [getting a shell](/learn/linux-cli/getting-a-shell/)
