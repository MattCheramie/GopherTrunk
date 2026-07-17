---
slug: wildcards-and-expansion
title: Wildcards, globbing & expansion
description: How the shell expands patterns before a command runs — glob wildcards (* ? []), brace expansion {a,b} and {1..5}, quoting to switch expansion off, tilde and command substitution, and why rm plus a wildcard deserves an echo first.
keywords: wildcard, glob, globbing, asterisk, brace expansion, shell expansion, quoting, tilde, command substitution, star pattern, single quotes, double quotes
level: intermediate
status: full
prereq:
  - first-commands
faq:
  - q: What is the difference between a wildcard and a glob?
    a: "They are the same idea under two names. A wildcard is a character like * or ? that stands in for other characters; globbing is the shell's act of matching those wildcard patterns against real filenames and replacing the pattern with the list of files it finds. So * is a wildcard, and expanding *.txt into the matching files is globbing."
  - q: Why does my command act on the wrong files with a wildcard?
    a: "Because the shell expands the pattern before the command ever runs, and it expands against whatever is in the current directory. If you are not where you think you are, or the pattern matches more than you expected, the command receives a different file list than you pictured. Run echo yourpattern or ls yourpattern first to see exactly what it expands to."
  - q: How do I stop the shell from expanding a wildcard?
    a: "Quote it. Single quotes 'like this' are fully literal — nothing inside is expanded. Double quotes \"like this\" block globbing and brace expansion but still allow variable expansion, so \"$HOME\" is filled in but \"*.txt\" stays literal. A backslash escapes a single character, so \\* is a literal asterisk."
  - q: What does cp file{,.bak} do?
    a: "It is brace expansion: {,.bak} expands to two words, the empty string and .bak, so the shell rewrites the line as cp file file.bak before cp runs. That makes a quick backup copy of file named file.bak. The braces are a text trick in the shell and have nothing to do with the cp command itself."
---

# Wildcards, globbing & expansion

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Patterns let one command act on many files. **Glob wildcards** — **`*`**, **`?`**,
**`[]`** — match filenames; **brace expansion** **`{}`** generates lists of words.
The crucial part: **the shell expands them first**, rewriting your line *before*
the command sees it. `rm *.log` becomes `rm a.log b.log …` — `rm` never knew a `*`
was involved. New to commands? Start with
[your first commands](/learn/linux-cli/first-commands/).
</div>

Typing out twenty filenames by hand is misery, and it is exactly the kind of
tedium the shell exists to spare you. Instead you describe the files with a
**pattern** and let the shell fill in the rest. Understanding *who* does the
filling in — the shell, not the command — is the one idea that turns wildcards
from a source of surprises into a reliable tool.

## Globbing

A **glob** is a filename pattern built from a few wildcard characters. The shell
matches the pattern against the files in a directory and replaces it with the
list of names that match.

- **`*`** matches **any run of characters** (including none).
- **`?`** matches **exactly one** character.
- **`[abc]`** matches **one character from the set**; ranges like **`[0-9]`**
  work too.

```console
$ ls
img1.png  img2.png  imgX.png  notes.txt  report-01.log  report-02.log

$ ls *.txt
notes.txt

$ ls report-??.log
report-01.log  report-02.log

$ ls img[0-9].png
img1.png  img2.png
```

`*.txt` matches every name ending in `.txt`. `report-??.log` matches a name with
exactly two characters where the `??` sit — so `report-01.log` matches but
`report-1.log` would not. `img[0-9].png` matches `img1.png` and `img2.png` but
skips `imgX.png`, because `X` is not in the range `0-9`.

## Brace expansion

**Braces** are a different tool that happens to look similar. Instead of matching
existing files, `{}` **generates** a list of words by stamping out each option:

```console
$ echo file.{txt,md,log}
file.txt file.md file.log

$ echo {1..5}
1 2 3 4 5
```

A comma-separated `{a,b,c}` expands to each item in turn; a range `{1..5}`
counts. The classic trick is a one-line backup, where `{,.bak}` expands to the
empty string and `.bak`:

```console
$ cp report.log{,.bak}
```

The shell rewrites that as `cp report.log report.log.bak`, copying the file to a
`.bak` twin in a single short command. Braces also save repetition when making
directories:

```console
$ mkdir -p project/{src,docs,tests}
```

That creates `project/src`, `project/docs`, and `project/tests` in one line.
Unlike globs, brace expansion does **not** care whether the names already exist —
it is pure text generation.

## The shell expands before the command runs

Here is the mental model that explains almost every wildcard surprise: **the
shell expands patterns first, then hands the finished words to the command.**

When you type `rm *.log`, `rm` never sees the `*`. The shell looks in the current
directory, finds the matching files, and rewrites the line as:

```console
$ rm *.log        # what you typed
$ rm a.log b.log c.log   # what rm actually receives
```

The command is blissfully unaware a pattern was ever involved — it just gets a
list of names. This is why globbing works the same for *every* command: it is the
shell's job, done once, before anything else happens.

One consequence to know: **if a glob matches nothing**, the shell (in its default
setting) leaves the pattern untouched and passes the literal text through. So
`ls *.xyz` with no matching files runs `ls *.xyz` — and `ls` complains it cannot
find a file literally named `*.xyz`. That confusing error is really telling you
"nothing matched."

## Quoting to stop expansion

Sometimes you want the special characters to stay literal. **Quoting** switches
expansion off — with three levels of strictness:

- **`"double quotes"`** allow variable expansion but **not** globbing or braces,
  so `"$HOME"` is filled in while `"*.txt"` stays a literal star.
- **`'single quotes'`** are fully literal — **nothing** inside is expanded.
- **`\`** (backslash) escapes the single character that follows it.

```console
$ echo "*.txt"
*.txt

$ echo '$HOME'
$HOME

$ echo "$HOME"
/home/you
```

That last case is why quoting matters in practice. Wrapping a variable as
**`"$var"`** lets its value expand but protects it from being re-split or
re-globbed by the shell — essential when a value might contain spaces or special
characters. Variables get their own lesson in
[environment variables & PATH](/learn/linux-cli/environment-variables/).

## Other expansions (a taste)

The shell does several more substitutions on the same line, before the command
runs. Two you will meet constantly:

- **`~`** expands to your **home directory**, so `~/notes` becomes
  `/home/you/notes`.
- **`$(command)`** is **command substitution** — it runs the inner command and
  drops its output right into the line: `echo "today is $(date +%F)"`.

You do not need to master these yet; just recognise that `~` and `$( )` are the
shell rewriting your line, exactly like globs and braces do.

## Care with rm + wildcards

Because the shell expands the pattern *before* `rm` runs, and because `rm` has no
undo, a wildcard mistake is unforgiving. The habit that saves you is to **look
before you delete**: preview the expansion with `echo` or `ls` first.

```console
$ echo *.log       # or: ls *.log
a.log b.log c.log
$ rm *.log         # only now, once you've seen the list
```

A stray space is the classic trap — `rm * .log` (with a space) expands `*` to
**everything** in the directory and then tries to remove `.log` too. Previewing
with `echo` would have shown the danger instantly. For safe copying, moving, and
deleting in general, see
[working with files](/learn/linux-cli/working-with-files/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — the shell expands the * into a filename list before rm ever runs." markdown="0">
  <p class="knowledge-check__q">Quick check: in <code>rm *.log</code>, what expands the <code>*</code> into a list of files?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The rm command, after it starts</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The shell, before rm runs</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The Linux kernel, at delete time</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Globs** match filenames: **`*`** any characters, **`?`** one character,
  **`[…]`** a set or range.
- **Brace expansion** **`{a,b}`** and **`{1..5}`** *generates* words — the classic
  `cp file{,.bak}` makes a backup.
- **The shell expands first**, then passes the finished words to the command — the
  command never sees the pattern.
- If a glob **matches nothing**, the literal pattern is passed through (that odd
  error means "no match").
- **Quote to stop expansion**: `'single'` is literal, `"double"` still allows
  `"$var"`, and `\` escapes one character.
- With **`rm` and wildcards**, `echo` or `ls` the pattern first — expansion has no
  undo.

Next up: [environment variables & PATH](/learn/linux-cli/environment-variables/)
