---
slug: first-commands
title: Your first commands
description: A gentle first tour of the Linux command line — pwd to see where you are, ls to see what's there, cd to move around, echo to print, and how to ask any command for help with --help and man so you never have to memorize.
keywords: pwd, ls, cd, echo, man, help, first linux commands, terminal basics, command line for beginners, current directory, --help, tldr
level: beginner
status: full
prereq:
  - getting-a-shell
faq:
  - q: What are the first Linux commands to learn?
    a: "Start with four: pwd shows the directory you are in, ls lists what is there, cd moves you to another directory, and echo prints text. Together they let you move around and look about safely. Add man and --help and you can figure out every other command as you meet it."
  - q: How do I know where I am in the terminal?
    a: "Type pwd (print working directory) and press Enter. It prints the full path of the directory you are currently in — your shell's current location in the filesystem. Every command you run without a path acts here, so pwd is worth checking whenever you feel lost."
  - q: How do I get help for a Linux command?
    a: "Two ways. Add --help after the command name for a quick summary of its options, for example ls --help. For the full manual, run man followed by the command name, such as man ls, and press q to quit. Looking things up beats memorizing — every command can explain itself."
  - q: Why does nothing happen when I type a command?
    a: "Usually a typo. Linux is case-sensitive and literal: LS is not ls, and a misspelled command name simply is not found. Check your spelling, watch for stray spaces, and remember that many commands print nothing at all when they succeed — silence is often success, not failure."
---

# Your first commands

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Four commands get you moving: **pwd** (where am I?), **ls** (what's here?),
**cd** (move around), and **echo** (print text). None of them change anything,
so they are safe to experiment with. Just as important is knowing how to ask
for help: **`command --help`** for a quick summary and **`man command`** for the
full manual. You never have to memorize — you look things up. New to the
terminal? Make sure you have one open first:
[getting a shell](/learn/linux-cli/getting-a-shell/).
</div>

You have a shell open and a blinking cursor waiting. This lesson is your first
walk around it. Every command here only *looks* — nothing you type below changes
a file or breaks anything — so type freely, make mistakes, and get a feel for
the rhythm of command, Enter, read the result.

## Where am I? — pwd

The shell always has a **current directory**: the folder it is "standing in"
right now. To see it, type:

```console
$ pwd
/home/you
```

**pwd** stands for *print working directory*, and that is exactly what it does —
it prints the full path of where you currently are. Whenever you feel lost, `pwd`
is the first thing to reach for. It answers "where am I?" in one word.

## What's here? — ls

Now that you know where you are, look at what is around you. **ls** (short for
*list*) lists the contents of the current directory:

```console
$ ls
Desktop  Documents  Downloads  notes.txt  photos
```

That is the plain view — just names. Commands often take **options** that change
their behaviour, and `ls` has a useful one, `-l` (a lowercase L), which shows a
longer, detailed listing with sizes, dates, and permissions:

```console
$ ls -l
-rw-r--r-- 1 you you  184 Jul 17 09:14 notes.txt
```

Don't worry about decoding that yet — we take `ls` and its options apart properly
in the next module, in
[navigating and listing](/learn/linux-cli/navigating-and-listing/). For now, just
know that `ls` shows you what's there and `ls -l` shows more about it.

## Moving around — cd

To change which directory you are standing in, use **cd** (*change directory*).
Give it the name of somewhere to go:

```console
$ cd Documents
$ pwd
/home/you/Documents
```

A few forms are worth knowing from day one:

- **`cd somewhere`** — move into a directory by name or path.
- **`cd ..`** — move *up* one level, to the parent directory. The `..` always
  means "the directory above this one."
- **`cd ~`** (or just `cd` on its own) — go **home**, back to your own personal
  directory.

Notice how `cd` and `pwd` work together: `cd` changes the current directory, and
`pwd` confirms where you landed. That "current directory" idea — your position in
a big tree of folders — is the backbone of the next module,
[the filesystem hierarchy](/learn/linux-cli/filesystem-layout/).

## Printing — echo

**echo** simply prints whatever you give it back to the screen:

```console
$ echo Hello there
Hello there
```

On its own that seems almost pointless, but `echo` turns out to be one of the
most-used commands of all — it is how scripts show messages, and how you peek at
the value stored in a **variable**:

```console
$ echo $HOME
/home/you
```

The `$HOME` there is a variable holding the path to your home directory, and
`echo` prints its contents. Variables and how the shell fills them in are a whole
topic of their own, covered later in
[variables and input](/learn/linux-cli/variables-and-input/). For now, treat
`echo` as your way to print text and check values.

## Getting help

Here is the most important skill in this whole path: you do not memorize
commands, you **look them up**. Every command can explain itself, two ways.

Add **`--help`** after a command name for a quick summary of what it does and the
options it takes:

```console
$ ls --help
```

For the full manual — longer, with every detail — use **man** (*manual*)
followed by the command name:

```console
$ man ls
```

The manual opens in a pager you scroll with the arrow keys; press **q** to quit
and return to the prompt. If the standard manual pages feel dense, try **tldr**,
a friendlier community tool that shows a handful of practical examples instead of
exhaustive detail (you may need to install it first). Whichever you use, the
habit is the same: when you meet an unfamiliar command, ask *it*.

## A first session

Here is everything above strung together into one short session — where am I,
what's here, move somewhere, look again, print a message:

```console
$ pwd
/home/you
$ ls
Desktop  Documents  Downloads  notes.txt  photos
$ cd Documents
$ ls
budget.txt  letters  report.pdf
$ echo All done looking around
All done looking around
```

That is a real, useful loop you will run constantly: orient yourself, look,
move, look again.

## Note on case & typos

Linux is **case-sensitive** and completely literal. `ls` is a command; `LS` and
`Ls` are not — it will simply report that they are not found. Commands do exactly
what you type, nothing more and nothing less, so a stray space or a misspelled
name means it can't do what you *meant*.

If you press Enter and nothing seems to happen, that is usually one of two things:
a typo the shell couldn't match, or a command that succeeded silently. Many Linux
commands print nothing at all when they work — **silence is often success**. When
in doubt, check your spelling and try again.

<div class="knowledge-check" data-quiz data-correct-msg="Right — pwd prints the working directory, the folder your shell is currently in." markdown="0">
  <p class="knowledge-check__q">Quick check: which command prints the directory you're currently in?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">ls</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">pwd</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">echo</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **pwd** prints your **current directory** — reach for it whenever you feel lost.
- **ls** lists what's in the current directory; **ls -l** shows more detail.
- **cd** moves you around: by name, `cd ..` up, `cd ~` (or `cd`) home.
- **echo** prints text and the values of variables — used constantly in scripts.
- Ask any command for help with **`--help`** or **`man command`** (press **q** to
  quit); **tldr** is a friendlier option.
- Linux is **case-sensitive** and literal — if nothing happens, check your spelling.

Next up: Module 2 explores the filesystem — [the filesystem hierarchy](/learn/linux-cli/filesystem-layout/)
