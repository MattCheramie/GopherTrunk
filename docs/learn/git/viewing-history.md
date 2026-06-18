---
slug: viewing-history
title: Viewing history with log & show
description: Explore Git history with log, show, and blame — oneline graphs, filtering by author, date, message, or content, custom formats, and HEAD navigation notation.
keywords: git log, git show, git blame, git log oneline, git log graph, git log pickaxe, git log grep, pretty format, HEAD notation, git history
level: beginner
status: full
prereq:
  - staging-and-commits
faq:
  - q: What is the difference between git log and git show?
    a: "git log lists commits, newest first, with whatever detail you ask for — one line each, full messages, diffs, or stats. git show focuses on a single object, usually one commit, and prints its full message plus the patch of exactly what that commit changed. Reach for log to browse history and show to inspect one specific commit."
  - q: How do I see a compact, visual commit graph?
    a: "git log --oneline --graph --all --decorate draws an ASCII graph of every branch. --oneline condenses each commit to a short hash and subject, --graph adds the branching lines, --all includes every branch and not just the current one, and --decorate labels commits with their branch and tag names."
  - q: What does HEAD~3 mean?
    a: "HEAD is your current commit. The tilde walks straight back through first parents — HEAD~1 is its parent, HEAD~3 is three commits back. The caret HEAD^ also means the parent, but it selects between parents of a merge — HEAD^1 is the first parent and HEAD^2 the second. For a linear history HEAD~1 and HEAD^ point to the same commit."
  - q: How can I find which commit introduced or removed a piece of text?
    a: "Use the pickaxe: git log -S\"someText\" lists commits where the number of occurrences of that string changed, so it pinpoints when a function or value was added or deleted. To search commit messages instead, use git log --grep=\"pattern\". To see who last touched each line of a file, use git blame."
---

# Viewing history with log & show

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**`git log`** lists commits newest-first, and its flags turn it from a wall of text
into a precise tool: **`--oneline`** for a compact list, **`--graph --all --decorate`**
for a visual branch map, and filters like **`--author`**, **`--since`**, **`--grep`**,
and the **`-S` pickaxe** to find exactly the commits you want. **`git show`** prints
one commit's message and full diff, and **`git blame`** tells you who last changed
each line. Navigate the past with **`HEAD~n`** and **`HEAD^`** notation.
</div>

A repository is a story told in commits. Once you can query that story — by author,
by date, by message, by the actual lines of code that changed — debugging and code
archaeology stop being guesswork.

## The most useful git log options

Bare `git log` prints every commit in full, which is rarely what you want. These
flags do the heavy lifting:

| Option | What it does |
|--------|--------------|
| `--oneline` | one line per commit: short hash + subject |
| `--graph` | draw ASCII lines showing branch and merge structure |
| `--all` | include every branch, not just the current one |
| `--decorate` | label commits with branch and tag names |
| `-p` | show the full patch (diff) of each commit |
| `--stat` | show a per-file change summary for each commit |
| `-n 5` | limit output to the 5 most recent commits |

The combination worth committing to muscle memory is:

```bash
$ git log --oneline --graph --all --decorate
* a4d9e10 (HEAD -> main) Tighten parser error message
* 8b1c3f2 (origin/main) Add README install section
| * 3f7b2c1 (feature/retry) Add retry backoff
|/
* c1d0a99 Initial commit
```

That single command shows where every branch points and how they relate.

## Filtering history

`git log` becomes a search engine once you add filters. They combine freely.

```bash
# Commits by a given author, since a date
$ git log --author="Matt" --since="2 weeks ago" --oneline

# Commits whose message matches a pattern
$ git log --grep="fix" --oneline

# Commits in a date window
$ git log --since="2026-01-01" --until="2026-03-31" --oneline
```

| Filter | Selects commits… |
|--------|------------------|
| `--author="name"` | by a matching author |
| `--grep="text"` | whose **message** contains the text |
| `--since=` / `--until=` | within a date range (accepts "2 weeks ago") |
| `-S"text"` | where the **count** of that text in the code changed |
| `-G"regex"` | whose diff matches a regular expression |

The last two are the **pickaxe**, and they're underused. `git log -S"parseTimeout"`
finds the exact commits that added or removed that identifier — far faster than
scrolling through diffs by hand.

```bash
$ git log -S"parseTimeout" --oneline
a4d9e10 Tighten parser error message
c1d0a99 Initial commit
```

## Custom output with --pretty=format

When the built-in formats don't fit, `--pretty=format:` lets you build your own line
from placeholders.

```bash
$ git log --pretty=format:"%h %an %ar — %s" -n 3
a4d9e10 Matt 2 hours ago — Tighten parser error message
8b1c3f2 Matt 1 day ago — Add README install section
c1d0a99 Matt 3 days ago — Initial commit
```

Handy placeholders: `%h` short hash, `%H` full hash, `%an` author name, `%ae` author
email, `%ar` author date relative, `%ad` author date, `%s` subject. This is the basis
of the pretty aliases many people add to their config.

## Inspecting one commit with git show

Where `log` browses, `git show` zooms in on a single object — by default the commit
at `HEAD`, or any commit you name:

```bash
$ git show 8b1c3f2
commit 8b1c3f2d4e... 
Author: Matt <matt@example.com>
Date:   Mon Jun 16 09:14:22 2026 -0500

    Add README install section

diff --git a/README.md b/README.md
@@ -4,3 +4,6 @@
 ## Install
+
+    go install ./...
```

It prints the full message followed by the exact patch that commit applied. You can
also point it at a file *at* a commit — `git show 8b1c3f2:README.md` prints that file
as it existed in that commit, without touching your working copy.

## Who wrote this line? git blame

`git blame <file>` annotates every line with the commit, author, and date that last
changed it.

```bash
$ git blame README.md
^c1d0a99 (Matt 2026-06-13 10:02:11 -0500 1) # GopherTrunk
a4d9e10  (Matt 2026-06-18 14:31:07 -0500 2) A small but mighty tool.
8b1c3f2  (Matt 2026-06-16 09:14:22 -0500 4) ## Install
```

Read each row as: short hash, author, timestamp, line number, then the line itself.
A `^` before a hash marks the repository's first ("boundary") commit. Once you find
the commit responsible, feed its hash to `git show` to see the full change in context.

## Navigating with HEAD notation

Many commands take a commit reference, and you rarely want to type a hash. Git's
relative notation walks the history for you:

| Notation | Meaning |
|----------|---------|
| `HEAD` | the commit you're currently on |
| `HEAD~1`, `HEAD^` | the parent (one commit back) |
| `HEAD~3` | three commits back along first parents |
| `HEAD^2` | the **second** parent of a merge commit |

The tilde `~n` always walks back through first parents, so `HEAD~3` is "three
generations up." The caret `^n` picks *which* parent at a merge — relevant only on
merge commits, where `HEAD^1` is the branch you were on and `HEAD^2` is the one you
merged in. For the meaning of commits, parents, and branches, see
[Git's mental model](/learn/git/git-mental-model/) and the
[glossary](/learn/git/glossary/).

<div class="knowledge-check" data-quiz data-correct-msg="Correct — the -S pickaxe finds commits where the count of that text changed." markdown="0">
  <p class="knowledge-check__q">Quick check: which option finds the commit that introduced the string <code>parseTimeout</code> into the code?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct"><code>git log -S"parseTimeout"</code></button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong"><code>git log --grep="parseTimeout"</code></button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong"><code>git log --author="parseTimeout"</code></button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- `git log` lists commits; `--oneline --graph --all --decorate` is the go-to visual.
- Filter with `--author`, `--since`/`--until`, `--grep` (messages), and `-S`/`-G`
  (the **pickaxe**, which searches the diffs).
- `--pretty=format:` builds custom output from placeholders like `%h`, `%an`, `%s`.
- `git show <commit>` prints one commit's message and full patch.
- `git blame <file>` shows who last touched each line.
- `HEAD~n` walks back through parents; `HEAD^n` selects a parent at a merge.

Next up: keeping noise out of your repository with `.gitignore`.
