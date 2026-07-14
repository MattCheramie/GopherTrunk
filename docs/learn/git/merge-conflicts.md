---
slug: merge-conflicts
title: Resolving merge conflicts
description: How to read and resolve Git merge conflicts calmly — understand the conflict markers, edit, stage, and commit, or abort cleanly with git merge --abort.
keywords: merge conflict, resolve merge conflict, conflict markers, git merge abort, mergetool, HEAD markers, git status conflict, rerere
level: intermediate
status: full
prereq:
  - merging
faq:
  - q: Why do merge conflicts happen?
    a: A conflict happens when two branches change the same lines of the same file in incompatible ways, or when one side edits a file the other deleted. Git can combine changes that touch different parts of a file automatically, but when both sides edit the very same region it cannot know which version you want, so it stops and asks you to decide.
  - q: What do the conflict markers mean?
    a: Git inserts three markers into the conflicted file. Everything between <<<<<<< HEAD and ======= is the version from your current branch. Everything between ======= and >>>>>>> branch-name is the version from the branch you are merging in. You resolve the conflict by editing the region to the final form you want and deleting all three marker lines.
  - q: How do I undo a merge that went wrong?
    a: While a merge is still in progress and unresolved, run git merge --abort. This returns your working directory and index to exactly the state they were in before you started the merge, discarding the half-finished result so you can try again.
  - q: What is git rerere?
    a: rerere stands for "reuse recorded resolution." When enabled, Git remembers how you resolved a particular conflict and replays that same resolution automatically if the identical conflict appears again later — handy on long-lived branches or repeated rebases where the same clash keeps coming back.
---

# Resolving merge conflicts

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **merge conflict** happens when two branches change the **same lines** in
incompatible ways — Git can't choose for you, so it pauses and marks the spot. The
markers **`<<<<<<< HEAD`**, **`=======`**, and **`>>>>>>> branch`** wrap *your* version
and *their* version. The fix is unhurried: **edit** the file to the form you want,
**`git add`** it, then **`git commit`** (or `--continue`). Lost or want a do-over?
**`git merge --abort`** returns you to before the merge. **`git status`** is your map
throughout, **`git mergetool`** opens a visual helper, and **`rerere`** can remember
resolutions for repeat conflicts.
</div>

Conflicts feel alarming the first time, but they're routine and recoverable — Git never
silently picks a side or destroys your work. A conflict is just Git saying "you two
disagreed here; tell me what you meant." This lesson gives you a calm, repeatable
process. It builds directly on [merging](/learn/git/merging/).

## Why conflicts happen

Git merges changes line by line. If one branch edits the top of a file and another edits
the bottom, Git stitches both in with no fuss. A conflict arises only when both branches
touch the **same region** in different ways — or when one side edits a file the other
deleted. Git has no way to know which intent should win, so rather than guess, it stops:

```bash
$ git merge feature/login
Auto-merging config.yml
CONFLICT (content): Merge conflict in config.yml
Automatic merge failed; fix conflicts and then commit the result.
```

The merge is now *paused*, not failed forever. Your job is to finish it.

## Reading the conflict markers

Open the conflicted file and you'll see Git has inserted three marker lines around the
disputed region:

```text
timeout: 30
<<<<<<< HEAD
retries: 5
backoff: exponential
=======
retries: 3
=======
>>>>>>> feature/login
log_level: info
```

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 208" role="img" aria-label="A conflicted file with Git's three conflict markers. Between the marker less-than signs HEAD and the equals-sign divider is the current branch's version, bracketed and labelled ours. Between the divider and the greater-than signs feature-slash-login marker is the incoming branch's version, bracketed and labelled theirs. Lines outside the markers are unchanged context." xmlns="http://www.w3.org/2000/svg">
  <rect x="16" y="28" width="272" height="164" rx="5" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-width="1" stroke-opacity="0.5"/>
  <g font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="10" fill="currentColor">
    <text x="28" y="48" fill-opacity="0.5">timeout: 30</text>
    <text x="28" y="67" font-weight="600">&lt;&lt;&lt;&lt;&lt;&lt;&lt; HEAD</text>
    <text x="28" y="86">retries: 5</text>
    <text x="28" y="105">backoff: exponential</text>
    <text x="28" y="124" font-weight="600">=======</text>
    <text x="28" y="143">retries: 3</text>
    <text x="28" y="162" font-weight="600">&gt;&gt;&gt;&gt;&gt;&gt;&gt; feature/login</text>
    <text x="28" y="181" fill-opacity="0.5">log_level: info</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <path d="M306 76 H298 V112 H306"/>
    <path d="M306 133 H298 V151 H306"/>
  </g>
  <g fill="currentColor">
    <text x="312" y="90" font-size="9" font-weight="600">ours</text>
    <text x="312" y="102" font-size="7.5" fill-opacity="0.85">HEAD — your branch</text>
    <text x="312" y="144" font-size="9" font-weight="600">theirs</text>
    <text x="312" y="156" font-size="7.5" fill-opacity="0.85">the incoming branch</text>
  </g>
  <text x="152" y="205" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">delete all three marker lines and keep the text you actually want</text>
</svg>
<figcaption>Git wraps the disputed region in three markers. Everything from <strong>&lt;&lt;&lt;&lt;&lt;&lt;&lt; HEAD</strong> to <strong>=======</strong> is <em>your</em> current branch's version; everything from there to <strong>&gt;&gt;&gt;&gt;&gt;&gt;&gt; feature/login</strong> is <em>theirs</em>. Lines outside the markers merged cleanly. Resolve by editing the region to its final form and deleting all three marker lines.</figcaption>
</figure>

Read it like this:

| Section | Meaning |
|---------|---------|
| `<<<<<<< HEAD` to `=======` | Your current branch's version (where HEAD is) |
| `=======` to `>>>>>>> feature/login` | The incoming branch's version |

Everything *outside* the markers merged cleanly and needs no attention. You only have to
decide what the region *between* the markers should become.

## A calm step-by-step resolve

1. **Find the conflicts.** Run `git status` — it lists every file needing attention
   under "Unmerged paths."
2. **Edit each file.** Replace the entire marked region with the final, correct content.
   Keep one side, keep the other, or write a blend — whatever is actually right. Then
   **delete all three marker lines**. The result for the example above might be:

   ```text
   timeout: 30
   retries: 5
   backoff: exponential
   log_level: info
   ```

3. **Stage the resolved file** to tell Git "this one is sorted":

   ```bash
   $ git add config.yml
   ```

4. **Finish the merge.** Once every conflict is staged, commit:

   ```bash
   $ git commit
   ```

   Git pre-fills a sensible merge message; just save and close. (During a rebase or
   cherry-pick instead of a plain merge, you'd run `git rebase --continue` or
   `git cherry-pick --continue` here rather than `git commit`.)

That's the whole loop: **edit → add → commit**. No magic.

## git status is your map

At any point during a conflict, `git status` tells you exactly where you stand:

```bash
$ git status
On branch main
You have unmerged paths.
  (fix conflicts and run "git commit")
  (use "git merge --abort" to abort the merge)

Unmerged paths:
  (use "git add <file>..." to mark resolution)
        both modified:   config.yml
```

"Unmerged paths" shrinks as you `git add` each resolved file. When the list is empty,
you're ready to commit. Lean on this command constantly — it removes all guesswork.

## Aborting: the safety hatch

If a merge is bigger or messier than expected, you don't have to push through. While the
merge is still in progress, abandon it entirely:

```bash
$ git merge --abort
```

This rewinds your working tree and index to exactly how they were *before* you ran
`git merge`. Nothing is lost; you simply get a clean slate to try again — perhaps after
talking to whoever made the other changes.

## Mergetools and rerere

For large or fiddly conflicts, a side-by-side visual editor beats hand-editing markers.
**`git mergetool`** launches whichever tool you've configured (VS Code, Meld, KDiff3,
vimdiff, and others) and walks you through each conflict:

```bash
$ git mergetool
Merging:
config.yml
Normal merge conflict for 'config.yml':
  {local}: modified file
  {remote}: modified file
```

You still finish with `git add` (the tool may stage for you) and `git commit`.

If you keep hitting the *same* conflict — common on long-lived branches or repeated
rebases — enable **`rerere`** ("reuse recorded resolution"):

```bash
$ git config --global rerere.enabled true
```

Git then records how you resolved each conflict and replays that resolution
automatically the next time the identical clash appears, so you only solve it once.
(See the [glossary](/learn/git/glossary/) for any term that's still fuzzy.)

<div class="knowledge-check" data-quiz data-correct-msg="Right — content between <<<<<<< HEAD and ======= is your current branch's version." markdown="0">
  <p class="knowledge-check__q">Quick check: in a conflict, what is the text between <code>&lt;&lt;&lt;&lt;&lt;&lt;&lt; HEAD</code> and <code>=======</code>?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The version from the branch you're merging in</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The version from your current branch</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The common ancestor before either change</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **conflict** means both branches changed the same lines; Git pauses and marks them.
- Markers wrap **your** version (`<<<<<<< HEAD` … `=======`) and **theirs** (`=======` … `>>>>>>> branch`).
- Resolve with the loop **edit → `git add` → `git commit`** (or `--continue` mid-rebase).
- **`git status`** lists unmerged paths; **`git merge --abort`** safely undoes the whole merge.
- **`git mergetool`** gives a visual editor; **`rerere`** remembers repeated resolutions.

Next up: replaying commits onto a new base for a [linear history with rebasing](/learn/git/rebasing/).
