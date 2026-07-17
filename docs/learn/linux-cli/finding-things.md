---
slug: finding-things
title: Finding files & text
description: Locate files and search their contents from the command line — find for locating files by name, type, size, or age, and grep for searching text inside them, plus which and locate for tracking down commands and filenames fast.
keywords: find, grep, locate, which, search files linux, grep recursive, find by name, search text, grep -i, find -type, type command, updatedb
level: intermediate
status: full
prereq:
  - navigating-and-listing
faq:
  - q: "How do I find a file by name in Linux?"
    a: "Use find with a starting directory and -name, like find . -name \"*.log\" to find every .log file under the current directory. Quote the pattern so the shell doesn't expand the wildcard before find sees it. Add -type f to match only regular files."
  - q: "How do I search for text inside files?"
    a: "Use grep. grep \"error\" app.log prints every line in app.log containing error. Add -r to search a whole directory tree, -i to ignore case, and -n to show line numbers so you can jump straight to the match."
  - q: "What's the difference between find and grep?"
    a: "find locates files by their properties — name, type, size, age — and doesn't look inside them. grep searches for text inside files. You reach for find when you know something about the file itself, and grep when you know some text that's in it."
  - q: "What does which do?"
    a: "which tells you the full path of the program a command would run, by searching your PATH. which python might print /usr/bin/python. It's how you confirm you're running the binary you think you are when several versions exist."
---

# Finding files & text

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Two tools cover almost everything. **`find`** locates files by their
properties — `find <where> -name "pattern"` searches by name, and it can also
match by **type**, **size**, or **age**. **`grep`** searches for *text inside*
files — `grep "text" file`, or `grep -r` to sweep a whole tree. **`which`** tells
you which program a command actually runs. If moving between directories still
feels shaky, revisit [navigating & listing](/learn/linux-cli/navigating-and-listing/) first.
</div>

Sooner or later you'll know a file exists but not where it is, or you'll need every
line in a pile of logs that mentions a particular error. The command line has two
purpose-built tools for exactly this: one finds files, the other reads inside them.

## Finding files — find

`find` walks a directory tree and lists files matching the tests you give it. The
basic shape is a starting point followed by what to match:

```
find <where> -name "pattern"
```

`<where>` is the directory to start from — usually `.` for "here" — and `-name`
matches against the filename, wildcards included:

```
find . -name "*.log"
```

That prints every `.log` file under the current directory, however deeply nested.
Quote the pattern (`"*.log"`) so the shell hands the `*` to `find` instead of
expanding it first.

`find` matches on more than just names:

- **`-type f`** — only regular files; **`-type d`** — only directories.
- **`-size +10M`** — files larger than 10 megabytes (`-` for smaller).
- **`-mtime -1`** — modified in the last day.

Combine them freely. This finds regular files ending in `.cfile` that were changed
in the last two days:

```
find . -type f -name "*.cfile" -mtime -2
```

## Searching inside files — grep

Where `find` cares about the file, `grep` cares about the *contents*. Give it some
text and a file, and it prints every line that contains that text:

```
grep "error" app.log
```

A handful of flags do most of the everyday work:

- **`-r`** — search recursively through a whole directory tree, not one file.
- **`-i`** — ignore case, so `error` also matches `Error` and `ERROR`.
- **`-n`** — show the line number of each match.
- **`-l`** — print just the *filenames* that contain a match, not the lines.

So to find every case-insensitive mention of "timeout" anywhere under your config
directory, with line numbers:

```
grep -rin "timeout" /etc/gophertrunk/
```

And to list just *which* log files mention a failed decode — handy when you only
need the filenames:

```
grep -rl "decode failed" ./logs/
```

## Combining find and grep

`find` and `grep` are natural partners: find narrows down *which* files, grep reads
*inside* them. You can feed one into the other with a pipe, sending find's file list
to grep to search only those files. Piping is a topic of its own —
[pipes & redirection](/learn/linux-cli/pipes-and-redirection/) covers it properly.

For everyday use, though, you rarely need the pipe: **`grep -r`** already walks the
tree and searches contents in one step. Reach for the find-into-grep combination only
when you want find's richer file tests (size, age, type) to pick the files first.

## Locating commands — which & type

Sometimes the "file" you're hunting for is a command itself. **`which`** tells you the
full path of the program a command would run:

```
which gophertrunk
```

It searches your **PATH** — the list of directories the shell looks through for
programs — and prints the first match, which is exactly the binary that would run.
That's the quick way to confirm you're launching the version you expect when several
are installed. (PATH is worth understanding in its own right —
[environment variables](/learn/linux-cli/environment-variables/) explains it.) The
built-in **`type`** does much the same and also flags shell built-ins and aliases.

For finding files by name *fast*, there's also **`locate`**, which searches a
prebuilt database of filenames instead of walking the disk. It's near-instant but
only as current as its database, which a `updatedb` run refreshes — so a brand-new
file may not show up until then.

## Putting it to use

**"I misplaced a config file."** You know it ends in `.yaml` but not where it landed:

```
find ~ -name "*.yaml"
```

That lists every YAML file under your home directory; scan the paths for the one you
meant.

**"I need every line mentioning an error."** You've got a directory of logs and want
each error, with its file and line number:

```
grep -rin "error" ./logs/
```

Now you've got the file, the line number, and the text — enough to open the right
file and jump straight to the problem.

<div class="knowledge-check" data-quiz data-correct-msg="Right — grep searches text inside files; find locates files by their properties." markdown="0">
  <p class="knowledge-check__q">Quick check: you want every line that mentions "lock lost" across a folder of logs. Which tool searches for text INSIDE files?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">find</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">grep</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">which</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **`find`** locates files by their properties — name, **type**, **size**, **age** — and doesn't look inside them.
- **`grep`** searches for **text inside** files; `grep -r` sweeps a whole tree in one step.
- Useful grep flags: **`-i`** case-insensitive, **`-n`** line numbers, **`-l`** filenames only.
- **`which`** (and `type`) show which program a command runs, by searching your **PATH**.
- **`locate`** finds filenames fast from a database — quick, but only as fresh as its last `updatedb`.

Next up: Module 3 covers who can do what — [users, groups & root](/learn/linux-cli/users-and-groups/)
