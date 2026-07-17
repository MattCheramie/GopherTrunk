---
slug: navigating-and-listing
title: Navigating & listing files
description: Move around the Linux filesystem with cd and see what's there with ls — absolute vs relative paths, going home and up, the ls flags that matter (-l, -a, -h, -t, -R), hidden dotfiles, the tree command, and tab completion.
keywords: cd, ls, ls -l, ls -a, hidden files, dotfiles, tree, tab completion, list directory, relative path, absolute path, pwd
level: beginner
status: full
prereq:
  - filesystem-layout
faq:
  - q: "How do I go back to the previous directory in Linux?"
    a: "Run cd - (that's cd followed by a hyphen). It jumps back to whatever directory you were in before the last cd, and prints the path so you can confirm. Plain cd with no argument takes you to your home directory instead."
  - q: "Which ls flag shows hidden files?"
    a: "Use ls -a (all). Files whose names start with a dot are hidden from a plain ls; the -a flag lists them too, including the . and .. entries for the current and parent directories."
  - q: "What does ls -l show that plain ls doesn't?"
    a: "ls -l is the long format: one file per line with permissions, owner, group, size, and modification date alongside the name. Combine it with -h (ls -lh) to get sizes as K/M/G instead of raw bytes."
  - q: "What's the difference between an absolute and a relative path?"
    a: "An absolute path starts from the root of the filesystem with a leading slash, like /home/you/captures — it means the same thing from anywhere. A relative path is measured from where you are now, like captures or ../logs, so it depends on your current directory."
---

# Navigating & listing files

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Two commands do most of the work. **`cd`** moves you around — `cd <path>` to go
somewhere, plain `cd` home, `cd ..` up one level, `cd -` back to where you were.
**`ls`** shows what's there — add **`-l`** for full detail, **`-a`** to reveal
hidden **dotfiles**, **`-h`** for readable sizes. Press **Tab** to auto-complete
paths so you type less and misspell nothing. If the directory names here are
unfamiliar, read [the filesystem layout](/learn/linux-cli/filesystem-layout/) first.
</div>

Working from the command line comes down to a simple loop: move to where you want
to be, look at what's there, act on it. This lesson covers the first two — moving and
looking. They're the commands you'll run more than any others, so a little fluency
here pays off every session.

## Moving with cd

`cd` stands for "change directory." Give it a path and it takes you there:

- `cd <path>` — go to that directory. The path can be **absolute** (starts with `/`,
  like `/home/you/captures`, and means the same thing from anywhere) or **relative**
  (measured from where you are now, like `captures` or `logs/old`).
- `cd` — with no argument, go to your **home** directory.
- `cd ~` — the `~` is shorthand for your home directory, so this also takes you home;
  `cd ~/captures` means the `captures` folder inside your home.
- `cd ..` — go **up** one level, to the parent of the current directory.
- `cd -` — go back to the **previous** directory you were in (handy for hopping
  between two places).

Not sure where you landed? Run `pwd` ("print working directory") and it tells you the
absolute path of where you are now.

## Listing with ls

Plain `ls` prints the names of files and directories in the current folder. That's
often all you need, but a handful of flags unlock the detail:

- **`-l`** — long format: one item per line with permissions, owner, size, and date.
  (The permission and owner columns get their own lesson —
  see [file permissions](/learn/linux-cli/permissions/).)
- **`-a`** — all: include hidden **dotfiles** (see below).
- **`-h`** — human-readable sizes, so you see `4.0K` or `2.1M` instead of raw byte
  counts. Only meaningful alongside `-l`.
- **`-t`** — sort by modification **time**, newest first (great for "what did I just
  create?").
- **`-R`** — recursive: list this directory and everything inside it, all the way down.

Flags combine into one word. `ls -lah` gives you the long format, hidden files, and
human-readable sizes all at once:

```console
$ ls -lah ~/captures
total 48M
drwxr-xr-x  2 you you 4.0K Jul 17 09:14 .
drwxr-xr-x 18 you you 4.0K Jul 17 09:02 ..
-rw-r--r--  1 you you  24M Jul 17 09:14 control.cfile
-rw-r--r--  1 you you  24M Jul 16 22:31 voice.cfile
-rw-r--r--  1 you you   61 Jul 16 22:30 .notes
```

## Hidden files (dotfiles)

Any file whose name starts with a dot — like `.notes` or `.bashrc` — is a **dotfile**,
and a plain `ls` hides it on purpose. There's nothing secret about them; the dot just
keeps clutter out of everyday listings. This is where a lot of configuration lives:
your home directory is typically full of dotfiles and dot-directories that programs
use to store their settings. Add `-a` whenever you need to see them.

## Seeing the shape of a tree

`ls -R` will list nested directories, but the output is a wall of text. The **`tree`**
command draws the same thing as a visual outline instead:

```console
$ tree ~/captures
/home/you/captures
├── control.cfile
├── voice.cfile
└── old
    └── 2026-06.cfile
```

It's not always installed by default, but it's a small package worth adding — one
glance shows you the shape of a directory that `ls` would only hint at.

## Tab completion

You rarely have to type a full path. Start typing a file or command name and press
**Tab** — the shell completes it for you. If more than one thing matches, press Tab
again to see the choices, then type another letter or two and Tab once more. It means
fewer typos and far less typing, especially with long capture filenames. There's more
to it — recalling past commands, editing a line quickly — in
[history & shortcuts](/learn/linux-cli/history-and-shortcuts/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — -a lists all entries, including the hidden dotfiles." markdown="0">
  <p class="knowledge-check__q">Quick check: which ls flag reveals hidden "dotfiles"?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">-l</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">-a</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">-h</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- `cd` moves you: `cd <path>` (absolute or relative), plain `cd` or `cd ~` home,
  `cd ..` up, `cd -` back to the previous directory.
- `pwd` confirms exactly where you are.
- `ls` lists what's there; `-l` adds detail, `-a` reveals hidden dotfiles, `-h`
  makes sizes readable, `-t` sorts by time, `-R` recurses.
- Flags stack — `ls -lah` is the everyday combination.
- **Dotfiles** (names starting with `.`) are hidden by default and hold most config.
- Press **Tab** to auto-complete paths and commands — fewer typos, faster.

Next up: [creating, copying & moving files](/learn/linux-cli/working-with-files/)
