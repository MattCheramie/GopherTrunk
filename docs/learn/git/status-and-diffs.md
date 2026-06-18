---
slug: status-and-diffs
title: Reading status & diffs
description: Read git status and git diff with confidence — short status, untracked vs staged, working-tree vs index diffs, hunk headers, and comparing commits and branches.
keywords: git status, git diff, git diff staged, git diff cached, hunk header, short status, untracked files, working tree, index, git diff stat
level: beginner
status: full
prereq:
  - staging-and-commits
faq:
  - q: What is the difference between git diff and git diff --staged?
    a: "Plain git diff shows changes in your working tree that are not yet staged — that is, the difference between your files on disk and the index. git diff --staged (also spelled --cached) shows what you have already staged compared with the last commit (HEAD). Use the first to review edits before adding them, and the second to review exactly what your next commit will contain."
  - q: Why does git status say a file is both staged and modified?
    a: A file appears in both lists when you stage a version of it and then edit it again before committing. The staged snapshot is frozen in the index, while your newer edits sit in the working tree. git diff shows the unstaged part and git diff --staged shows the staged part. Run git add again to fold the latest edits into the staged snapshot.
  - q: How do I read a hunk header like @@ -12,7 +12,8 @@?
    a: The hunk header marks where a change lives. The part after the minus sign refers to the old file (start at line 12, spanning 7 lines) and the part after the plus sign refers to the new file (start at line 12, spanning 8 lines). Lines beginning with a minus were removed, lines with a plus were added, and unprefixed lines are unchanged context.
  - q: How do I see the difference between two branches?
    a: "Use git diff branchA..branchB to see what changed from branchA to branchB. To see only the names and a summary of changed files, add --stat. You can also diff any two commits the same way, for example git diff HEAD~3..HEAD to review the last three commits' combined effect."
---

# Reading status & diffs

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**`git status`** tells you *which* files changed and where they sit — **untracked**,
**modified** in the working tree, or **staged** in the index. **`git diff`** tells
you *what* changed, line by line. Plain `git diff` compares the working tree to the
**index**; `git diff --staged` compares the index to the last commit; `git diff HEAD`
shows everything at once. Learn to read a **hunk header** and the `+`/`-` lines and
you can review any change — even between two branches with `git diff A..B`.
</div>

These two commands are the ones you'll run more than any other. Together they answer
the only two questions that matter before a commit: *what have I changed, and is it
ready?* Spend a few minutes here and you'll stop committing surprises.

## What does git status tell me?

`git status` reports the state of your working tree relative to the **index** (the
staging area) and the last commit. It groups files into three buckets.

```bash
$ git status
On branch main
Your branch is ahead of 'origin/main' by 1 commit.
  (use "git push" to publish your local commits)

Changes to be committed:
  (use "git restore --staged <file>..." to unstage)
        modified:   src/parser.go

Changes not staged for commit:
  (use "git add <file>..." to update what will be committed)
  (use "git restore <file>..." to discard changes in working directory)
        modified:   README.md

Untracked files:
  (use "git add <file>..." to include in what will be committed)
        notes.txt
```

The first line names your **branch**, and the second compares it to its remote —
"ahead by 1 commit" means you have a local commit you haven't pushed. Then come the
three states:

| State | Meaning | How it got there |
|-------|---------|------------------|
| Changes to be committed | **staged** in the index | you ran `git add` |
| Changes not staged | **modified** on disk only | you edited a tracked file |
| Untracked files | Git has never seen this file | you created it |

If a file shows up under *both* "to be committed" and "not staged," you staged it and
then edited it again — the staged snapshot and the working copy now differ.

## The short status format

The full output is friendly but verbose. `git status -s` (short) packs the same
information into two columns:

```bash
$ git status -s
M  src/parser.go
 M README.md
?? notes.txt
```

The **left column is the index** (staged) and the **right column is the working
tree** (unstaged). So `M ` (M then space) means staged-modified, ` M` (space then M)
means modified-but-unstaged, and `??` means untracked. A file edited in both places
shows `MM`. Add `git status -sb` to keep the branch line at the top:

```bash
$ git status -sb
## main...origin/main [ahead 1]
M  src/parser.go
```

## git diff: working tree vs index

Plain `git diff` with no arguments shows only your **unstaged** changes — the gap
between the files on disk and what's in the index.

```bash
$ git diff
diff --git a/README.md b/README.md
index 8b1c3f2..a4d9e10 100644
--- a/README.md
+++ b/README.md
@@ -1,4 +1,4 @@
 # GopherTrunk
-A small tool.
+A small but mighty tool.

 ## Install
```

The `a/` version is the index side and `b/` is the working-tree side. Lines starting
with `-` were removed, lines with `+` were added, and unprefixed lines are unchanged
**context** shown so you can locate the change.

## git diff --staged and git diff HEAD

Because plain `git diff` ignores staged changes, you need other forms to see them:

| Command | Compares | Answers |
|---------|----------|---------|
| `git diff` | working tree ↔ index | "what haven't I staged yet?" |
| `git diff --staged` | index ↔ HEAD | "what will my next commit contain?" |
| `git diff HEAD` | working tree ↔ HEAD | "what changed in total since the last commit?" |

`--staged` and `--cached` are exact synonyms — use whichever you remember.
`git diff --staged` is the one to run right before you commit: it shows precisely
what is about to be recorded, nothing more and nothing less.

## How to read a hunk header

Each block of changes is a **hunk**, introduced by a header wrapped in `@@`:

```
@@ -12,7 +12,8 @@ func parse(line string) {
```

Read it as two ranges. The `-12,7` describes the **old** file: starting at line 12,
spanning 7 lines. The `+12,8` describes the **new** file: starting at line 12,
spanning 8 lines. Here the new version is one line longer, so a line was added. The
text after the final `@@` is the **function context** — the nearest enclosing
function or section, included to orient you. Inside the hunk:

- a `-` line existed before and was removed,
- a `+` line is new,
- a line with no prefix is unchanged context anchoring the change.

## Diffing commits and branches

`git diff` isn't limited to your working tree — give it two references and it shows
the difference between them.

```bash
$ git diff HEAD~3..HEAD --stat
 src/parser.go | 24 ++++++++++++++++--------
 README.md     |  2 +-
 2 files changed, 17 insertions(+), 9 deletions(-)
```

`A..B` reads as "what changed going from A to B." Common uses:

- `git diff main..feature` — what your feature branch adds over `main`.
- `git diff HEAD~1` — the most recent commit's changes (against your working tree).
- `git diff v1.0..v1.1` — everything between two tagged releases.

Two flags make big diffs readable. `--stat` (above) summarises files and line counts
instead of full text. `--word-diff` highlights changed *words* rather than whole
lines, which is gold for prose:

```bash
$ git diff --word-diff README.md
A small [-tool.-]{+but mighty tool.+}
```

The `[-...-]` marks removed words and `{+...+}` marks added words. For the underlying
model of working tree, index, and HEAD that all of this rests on, see
[Git's mental model](/learn/git/git-mental-model/) and the
[glossary](/learn/git/glossary/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — --staged compares the index to HEAD, so it's exactly your next commit." markdown="0">
  <p class="knowledge-check__q">Quick check: which command shows exactly what your next commit will contain?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong"><code>git diff</code></button></li>
    <li><button type="button" class="quiz__option" data-answer="correct"><code>git diff --staged</code></button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong"><code>git status</code></button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- `git status` shows *which* files are **untracked**, **modified**, or **staged**;
  `-s` gives a compact two-column view (left = index, right = working tree).
- Plain `git diff` shows **unstaged** changes (working tree vs index).
- `git diff --staged` shows **staged** changes (index vs HEAD) — your next commit.
- `git diff HEAD` shows everything since the last commit.
- A **hunk header** `@@ -a,b +c,d @@` maps old and new line ranges; `-`/`+` mark
  removed and added lines.
- `git diff A..B`, plus `--stat` and `--word-diff`, compare commits and branches.

Next up: reading the project's story with `git log`, `git show`, and `git blame`.
