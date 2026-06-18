---
slug: undoing-changes
title: "Undoing changes: restore, reset & revert"
description: Undo Git changes safely — restore working edits, unstage files, reset HEAD soft/mixed/hard, revert shared commits, clean untracked files, and avoid destructive mistakes.
keywords: git restore, git reset, git revert, git reset hard, git reset soft, unstage, git clean, undo commit, git checkout, reflog
level: intermediate
status: full
prereq:
  - staging-and-commits
  - status-and-diffs
faq:
  - q: What is the difference between git reset and git revert?
    a: "git reset moves your branch pointer backward, rewriting history by discarding or unstaging commits — it's a local edit to where the branch points. git revert instead creates a brand-new commit that undoes a previous one, leaving the original in place. Use reset on local, unpushed work, and revert to undo a commit that others have already pulled, because revert doesn't rewrite shared history."
  - q: What do --soft, --mixed, and --hard do on git reset?
    a: "They differ in how much they undo. --soft moves only the branch pointer, keeping your index and working tree, so the undone changes stay staged. --mixed (the default) also resets the index, so changes become unstaged but remain on disk. --hard resets all three — pointer, index, and working tree — throwing away your changes entirely. Only --hard is destructive to your files."
  - q: How do I discard changes to a single file?
    a: "Use git restore <file> to throw away unstaged edits and bring the file back to its committed state. To unstage a file you've added but keep the edits, use git restore --staged <file>. The older git checkout -- <file> does the same discard, but restore is the modern, clearer command split out specifically for this purpose."
  - q: I ran git reset --hard and lost work — can I get it back?
    a: "Often yes. Git's reflog records where HEAD has pointed recently, so git reflog will show the commit you reset away from; you can return to it with git reset --hard <that-hash>. The reflog is a safety net for commits, but it can't recover uncommitted working-tree changes that were never staged or committed, so commit early and often."
---

# Undoing changes: restore, reset & revert

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Git has three undo tools for three jobs. **`git restore`** throws away working-tree
edits or **unstages** files. **`git reset`** moves your branch pointer back, with
**`--soft`**, **`--mixed`**, and **`--hard`** controlling whether your index and
working tree come along. **`git revert`** makes a *new* commit that undoes an old one
— the safe choice on **shared** branches because it doesn't rewrite history. Treat
**`reset --hard`** and any history rewrite as destructive, and lean on the **reflog**
as your safety net.
</div>

"How do I undo this?" is the question every Git user asks within their first week. The
trouble is that "undo" means several different things, and Git has a different tool for
each. Match the tool to the situation and undoing becomes routine instead of scary.

## Discard or unstage with git restore

`git restore` is the modern, purpose-built command for working-tree and staging undos.
It has two everyday forms:

```bash
# Throw away unstaged edits to a file (back to its committed state)
$ git restore README.md

# Unstage a file but keep the edits in your working tree
$ git restore --staged README.md
```

The first **discards** changes — it's irreversible for unstaged edits, so be sure. The
second only moves the file out of the index; your edits are untouched on disk. To
restore everything at once, use `git restore .`.

## The older checkout equivalent

Before `restore` arrived, the same discard was spelled `git checkout -- <file>`:

```bash
$ git checkout -- README.md   # legacy: same as git restore README.md
```

It still works, but `git checkout` was overloaded — it also switches branches, which
made it confusing and occasionally dangerous. Git split that role into `git restore`
(for files) and `git switch` (for branches). Prefer `restore` in new habits; recognise
`checkout --` when you see it in older guides.

## git reset and the three trees

`git reset` moves your current branch to point at an earlier commit. The flag decides
how far the undo reaches into Git's **three trees** — the branch pointer (HEAD), the
index, and the working tree.

| Mode | Moves HEAD | Resets index | Resets working tree | Net effect |
|------|:---------:|:------------:|:-------------------:|------------|
| `--soft` | yes | no | no | undone changes stay **staged** |
| `--mixed` *(default)* | yes | yes | no | undone changes become **unstaged** |
| `--hard` | yes | yes | yes | undone changes are **discarded** |

```bash
# Undo the last commit but keep everything staged (e.g. to re-commit)
$ git reset --soft HEAD~1

# Undo the last commit and unstage its changes (still on disk)
$ git reset --mixed HEAD~1

# Undo the last commit and throw the changes away entirely
$ git reset --hard HEAD~1
```

`HEAD~n` chooses how many commits to roll back. Only `--hard` touches your files, so
only `--hard` can lose uncommitted work — treat it with respect.

## Undoing a commit safely with git revert

`reset` rewrites history, which is fine locally but dangerous once a commit has been
pushed and pulled by others. `git revert` is the safe alternative: it creates a **new**
commit that applies the inverse of an existing one, leaving the original in place.

```bash
$ git revert a4d9e10
[main 7e2f9c0] Revert "Tighten parser error message"
 1 file changed, 1 insertion(+), 1 deletion(-)
```

Because nothing existing is altered, everyone's history stays consistent — perfect for
undoing a bad change on `main`. The trade-off is that the undone commit and its reversal
both remain visible in the log.

## "I want to…" → use…

A quick lookup for the common cases:

| I want to… | Use |
|------------|-----|
| Discard unstaged edits to a file | `git restore <file>` |
| Unstage a file, keep the edits | `git restore --staged <file>` |
| Undo my last commit, keep changes staged | `git reset --soft HEAD~1` |
| Undo my last commit, unstage changes | `git reset --mixed HEAD~1` |
| Undo my last commit and discard it | `git reset --hard HEAD~1` |
| Undo a commit others already have | `git revert <commit>` |
| Delete untracked files | `git clean -fd` |

## Cleaning untracked files

None of the above removes *untracked* files — `git restore` and `git reset` only act on
tracked content. For stray build output or scratch files, use `git clean`:

```bash
# Preview what would be removed (always do this first)
$ git clean -n
Would remove notes.txt
Would remove tmp/

# Actually remove untracked files (-f) and directories (-d)
$ git clean -fd
```

`-n` (dry run) is your friend, because `git clean` deletes files outright — they don't
go through the index or the trash.

## A word on destruction and the safety net

Two operations can genuinely lose work: **`git reset --hard`** and history rewrites such
as [rebasing](/learn/git/rebasing/). On a **shared** branch, rewriting history that
others have already pulled forces painful reconciliation on everyone — so never rewrite
published history.

Your safety net for committed work is the **reflog**, which records where `HEAD` has
been:

```bash
$ git reflog
a4d9e10 HEAD@{0}: reset: moving to HEAD~1
7e2f9c0 HEAD@{1}: commit: Tighten parser error message
$ git reset --hard 7e2f9c0   # jump back to the commit you reset away from
```

The reflog can rescue commits you thought were gone, but it cannot recover
working-tree edits that were never committed. The real lesson: commit often, and a
`--hard` mistake becomes recoverable. See the
[glossary](/learn/git/glossary/) for the reflog and three-tree terms.

<div class="knowledge-check" data-quiz data-correct-msg="Correct — revert adds a new inverse commit and never rewrites shared history." markdown="0">
  <p class="knowledge-check__q">Quick check: a bad commit is already on <code>main</code> and teammates have pulled it. How do you undo it safely?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong"><code>git reset --hard HEAD~1</code></button></li>
    <li><button type="button" class="quiz__option" data-answer="correct"><code>git revert &lt;commit&gt;</code></button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong"><code>git restore .</code></button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- `git restore <file>` discards working edits; `git restore --staged <file>` unstages.
- The legacy `git checkout -- <file>` does the same discard but is superseded.
- `git reset` moves the branch: `--soft` keeps changes staged, `--mixed` unstages them,
  `--hard` discards them entirely.
- `git revert` makes a new inverse commit — the safe undo for **shared** history.
- `git clean -fd` removes untracked files (preview with `-n` first).
- `reset --hard` and rebasing are destructive; the **reflog** can recover lost commits.

Next up: branching — working on many lines of development at once.
