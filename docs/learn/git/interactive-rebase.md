---
slug: interactive-rebase
title: Rewriting history with interactive rebase
description: Use git rebase -i to squash, reword, reorder, edit, and drop commits — clean up a feature branch with the todo list, --autosquash, and safe force-pushing.
keywords: git rebase -i, interactive rebase, squash commits, reword commit, edit commit, fixup, autosquash, git commit --fixup, force-with-lease, rewrite history
level: advanced
status: full
prereq:
  - rebasing
faq:
  - q: What does git rebase -i do?
    a: Interactive rebase opens a "todo list" of the commits in the range you give it (for example git rebase -i HEAD~5 for the last five). You edit that list to choose what happens to each commit — pick keeps it, reword changes its message, edit pauses so you can amend its content, squash and fixup combine it into the previous commit, and drop deletes it. You can also reorder lines to reorder commits. Saving and closing the editor replays the commits according to your instructions, giving each a new hash.
  - q: What is the difference between squash and fixup?
    a: Both combine a commit into the one above it in the todo list. squash keeps the commit's message and lets you edit the combined message in an editor; fixup discards the commit's message entirely and keeps only the previous commit's message. Use fixup for "oops, typo" commits whose messages add nothing, and squash when each message has something worth merging into the final text.
  - q: How does --autosquash work?
    a: When you commit a fix with git commit --fixup=<sha>, Git writes a special message ("fixup! <subject>") that points at the target commit. Later, running git rebase -i --autosquash automatically positions that fixup commit directly below its target and marks it as a fixup, so you don't have to reorder the todo list by hand. Set rebase.autosquash to true to make it the default.
  - q: Is it safe to rebase commits I've already pushed?
    a: Only if the branch is yours and you understand you'll have to force-push. Interactive rebase rewrites history by creating new commits with new hashes, so the golden rule still applies — never rewrite history that others have built on. For a personal feature branch behind a pull request, rebase then push with --force-with-lease, which refuses to clobber a teammate's commits. If something goes wrong, the reflog lets you recover the pre-rebase state.
---

# Rewriting history with interactive rebase

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**`git rebase -i HEAD~n`** opens a **todo list** of recent commits that you edit to
reshape history. Each line gets a command: **`pick`** (keep), **`reword`** (change the
message), **`edit`** (pause to amend content), **`squash`**/**`fixup`** (combine into the
commit above), or **`drop`** (delete) — and you can **reorder lines** to reorder commits.
Squash a string of WIP commits into one clean commit, fix a typo in an old message, or
mark fixes with **`git commit --fixup=<sha>`** and let **`--autosquash`** file them
automatically. Every rewrite makes **new hashes**, so the [golden
rule](/learn/git/rebasing/) holds — only on history you haven't shared — and the
[reflog](/learn/git/cherry-pick-bisect-reflog/) is your undo. Push with
**`--force-with-lease`**.
</div>

Plain [rebasing](/learn/git/rebasing/) moves a branch onto a new base. *Interactive*
rebase rewrites the branch's own commits: squashing, reordering, rewording, splitting,
and dropping. It's how a messy sequence of "wip", "fix", "actually fix" commits becomes
the tidy, reviewable story you'd be proud to put in front of a teammate.

## The todo list

Start an interactive rebase by naming how far back to go. `HEAD~5` means "the last five
commits":

```bash
$ git rebase -i HEAD~5
```

Git opens your editor with a **todo list** — one line per commit, *oldest at the top*,
each prefixed with the default command `pick`:

```text
pick a1b2c3d Add login form
pick e4f5g6h wip
pick i7j8k9l fix typo
pick m0n1o2p wip more
pick q3r4s5t Add tests for login

# Rebase 9f8e7d6..q3r4s5t onto 9f8e7d6 (5 commands)
#
# Commands:
# p, pick   = use commit
# r, reword = use commit, but edit the commit message
# e, edit   = use commit, but stop for amending
# s, squash = use commit, but meld into previous commit
# f, fixup  = like "squash", but discard this commit's log message
# d, drop   = remove commit
```

You change history by editing this text. Replace `pick` with another command, or
**reorder the lines** to reorder the commits. When you save and close the editor, Git
replays the commits top to bottom following your instructions. Get cold feet at any point
with `git rebase --abort`.

| Command | Short | Effect |
|---------|-------|--------|
| `pick` | `p` | Keep the commit as-is |
| `reword` | `r` | Keep the changes, edit the message |
| `edit` | `e` | Pause here so you can amend the commit's content |
| `squash` | `s` | Meld into the commit above; combine both messages |
| `fixup` | `f` | Meld into the commit above; discard this message |
| `drop` | `d` | Delete the commit entirely |

## Squashing WIP commits into one

The classic use: collapse a trail of work-in-progress commits into a single, coherent
commit. Change the first commit's command to `pick` (or leave it) and mark the rest
`squash` or `fixup`:

```text
pick a1b2c3d Add login form
fixup e4f5g6h wip
fixup i7j8k9l fix typo
fixup m0n1o2p wip more
pick q3r4s5t Add tests for login
```

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 176" role="img" aria-label="An interactive rebase todo list on the left holds five commits: one pick followed by three fixups, then another pick. An arrow labelled rebase dash i leads to the result on the right, where the three fixups have been folded into the commit above them, leaving just two clean commits." xmlns="http://www.w3.org/2000/svg">
  <text x="18" y="20" font-size="9" fill="currentColor" font-weight="600">todo list (before)</text>
  <g font-size="8.5">
    <rect x="16" y="30" width="198" height="18" rx="3" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.4"/>
    <rect x="22" y="33" width="38" height="12" rx="2" fill="currentColor" fill-opacity="0.20" stroke="currentColor" stroke-width="0.9"/><text x="41" y="42" text-anchor="middle" fill="currentColor" font-weight="600">pick</text>
    <text x="66" y="42" fill="currentColor">Add login form</text>
    <rect x="16" y="52" width="198" height="18" rx="3" fill="currentColor" fill-opacity="0.03" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.3" stroke-dasharray="3 2"/>
    <rect x="30" y="55" width="38" height="12" rx="2" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="0.9" stroke-dasharray="3 2"/><text x="49" y="64" text-anchor="middle" fill="currentColor">fixup</text>
    <text x="74" y="64" fill="currentColor" fill-opacity="0.6">wip</text>
    <rect x="16" y="74" width="198" height="18" rx="3" fill="currentColor" fill-opacity="0.03" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.3" stroke-dasharray="3 2"/>
    <rect x="30" y="77" width="38" height="12" rx="2" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="0.9" stroke-dasharray="3 2"/><text x="49" y="86" text-anchor="middle" fill="currentColor">fixup</text>
    <text x="74" y="86" fill="currentColor" fill-opacity="0.6">fix typo</text>
    <rect x="16" y="96" width="198" height="18" rx="3" fill="currentColor" fill-opacity="0.03" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.3" stroke-dasharray="3 2"/>
    <rect x="30" y="99" width="38" height="12" rx="2" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="0.9" stroke-dasharray="3 2"/><text x="49" y="108" text-anchor="middle" fill="currentColor">fixup</text>
    <text x="74" y="108" fill="currentColor" fill-opacity="0.6">wip more</text>
    <rect x="16" y="118" width="198" height="18" rx="3" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.4"/>
    <rect x="22" y="121" width="38" height="12" rx="2" fill="currentColor" fill-opacity="0.20" stroke="currentColor" stroke-width="0.9"/><text x="41" y="130" text-anchor="middle" fill="currentColor" font-weight="600">pick</text>
    <text x="66" y="130" fill="currentColor">Add tests for login</text>
  </g>
  <line x1="222" y1="83" x2="288" y2="83" stroke="currentColor" stroke-width="1.4" fill="none" marker-end="url(#ireb_ar)"/>
  <text x="255" y="76" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.9">rebase -i</text>
  <text x="298" y="20" font-size="9" fill="currentColor" font-weight="600">clean history (after)</text>
  <g font-size="8.5">
    <rect x="296" y="56" width="158" height="26" rx="4" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/><text x="375" y="72" text-anchor="middle" fill="currentColor" font-weight="600">Add login form</text>
    <rect x="296" y="102" width="158" height="26" rx="4" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/><text x="375" y="118" text-anchor="middle" fill="currentColor" font-weight="600">Add tests for login</text>
  </g>
  <text x="235" y="162" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">five messy commits → two clean ones · each survivor gets a new hash</text>
  <defs><marker id="ireb_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Marking the three throwaway commits <strong>fixup</strong> melds each into the <strong>pick</strong> above it and discards its message. The five-line todo collapses to two coherent commits — the tidy, reviewable story you'd put in front of a teammate. (Use <strong>squash</strong> instead when a message is worth keeping.)</figcaption>
</figure>

Here the three throwaway commits melt into "Add login form" and their useless messages
vanish. Use `squash` instead of `fixup` for any commit whose message is worth keeping —
Git then opens an editor so you can write one combined message:

```bash
$ git rebase -i HEAD~5
[detached HEAD 7c8d9e0] Add login form
 3 files changed, 84 insertions(+)
Successfully rebased and updated refs/heads/feature.
```

Five commits became two, with clean messages. Each surviving commit has a **new hash** —
that's the rewrite at work.

## Rewording a message

To fix a typo or sharpen the wording of a past commit *without* touching its content,
mark it `reword`:

```text
pick a1b2c3d Add login form
reword e4f5g6h Add sesion timeout
pick q3r4s5t Add tests for login
```

When you save, Git replays up to that commit and reopens your editor on just its message,
where you correct `sesion` to `session`. This is the right tool for one stale message;
for the most recent commit only, plain `git commit --amend` is faster.

## Editing an old commit's content

Sometimes you need to change *what's in* an old commit — drop a debug line, split it, add
a forgotten file. Mark it `edit`:

```text
pick a1b2c3d Add login form
edit e4f5g6h Add session timeout
pick q3r4s5t Add tests for login
```

Git replays up to that commit and then **pauses**, leaving you on it:

```bash
$ git rebase -i HEAD~3
Stopped at e4f5g6h...  Add session timeout
You can amend the commit now, with

  git commit --amend

Once you are satisfied with your changes, run

  git rebase --continue
```

Now make your edits, stage them, fold them into the paused commit with `--amend`, then
resume:

```bash
$ git add timeout.js
$ git commit --amend --no-edit
$ git rebase --continue
```

(To *split* a commit instead, run `git reset HEAD^` at the pause to unstage its changes,
then make two smaller commits before continuing — see
[undoing changes](/learn/git/undoing-changes/).)

## --autosquash and git commit --fixup

Manually reordering fixups in the todo list gets old. Git can do it for you. When you spot
a problem in an earlier commit, stage the fix and commit it *targeting* that commit:

```bash
$ git add login.js
$ git commit --fixup=a1b2c3d
[feature 5d6e7f8] fixup! Add login form
```

The `fixup!` prefix tags this commit as belonging to `a1b2c3d`. Later, run the rebase with
`--autosquash`:

```bash
$ git rebase -i --autosquash HEAD~6
```

Git automatically moves the `fixup!` commit directly beneath its target and pre-marks it
`fixup` in the todo list — you just confirm. Make it permanent so every interactive
rebase autosquashes:

```bash
$ git config --global rebase.autosquash true
```

There's also `git commit --squash=<sha>` for the case where you want the message merged
rather than discarded.

## The golden rule still applies — and the reflog saves you

Interactive rebase is history rewriting, so the [golden rule from
rebasing](/learn/git/rebasing/) is non-negotiable: **never rewrite commits others have
built on.** Reshape a personal feature branch all you like; once it's shared, coordinate
or leave it be. When you've rewritten a branch you already pushed for a [pull
request](/learn/git/pull-requests/), force-push *safely*:

```bash
$ git push --force-with-lease
```

`--force-with-lease` refuses to overwrite the remote if a teammate pushed since your last
fetch — unlike plain `--force`, which clobbers unconditionally.

And if a rebase goes sideways? Nothing is truly lost. The
[reflog](/learn/git/cherry-pick-bisect-reflog/) records every position `HEAD` has held,
so you can jump back to the pre-rebase commit:

```bash
$ git reflog
7c8d9e0 HEAD@{0}: rebase (finish): returning to refs/heads/feature
9f8e7d6 HEAD@{1}: rebase (start): checkout main
4a5b6c7 HEAD@{2}: commit: Add tests for login   ← before the rebase
$ git reset --hard HEAD@{2}
```

That recoverability is what makes experimenting with rebase safe on your own branches.

<div class="knowledge-check" data-quiz data-correct-msg="Right — fixup melds the commit into the one above and throws its message away." markdown="0">
  <p class="knowledge-check__q">Quick check: in a rebase todo list, which command combines a commit into the one above it and discards its message?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">fixup</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">reword</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">edit</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **`git rebase -i HEAD~n`** opens a **todo list** you edit to reshape recent commits.
- Commands: **`pick`**, **`reword`**, **`edit`**, **`squash`**, **`fixup`**, **`drop`** — and **reorder lines** to reorder commits.
- **`squash`/`fixup`** collapse WIP commits into one; **`reword`** fixes a message; **`edit`** pauses to amend old content.
- **`git commit --fixup=<sha>`** plus **`--autosquash`** files fixes automatically.
- Rewriting makes **new hashes**: obey the **golden rule**, push with **`--force-with-lease`**, and lean on the **[reflog](/learn/git/cherry-pick-bisect-reflog/)** to recover.

Next up: copying single commits, hunting bugs by bisection, and the reflog safety net — [cherry-pick, bisect & the reflog](/learn/git/cherry-pick-bisect-reflog/).
