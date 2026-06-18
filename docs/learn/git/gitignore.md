---
slug: gitignore
title: .gitignore & tracking the right files
description: Keep secrets, build artifacts, and OS cruft out of Git with .gitignore — pattern syntax, precedence, global ignores, the already-tracked gotcha, and check-ignore.
keywords: gitignore, git ignore, ignore patterns, git rm cached, core.excludesFile, global gitignore, git check-ignore, info exclude, gitignore templates
level: beginner
status: full
prereq:
  - first-repository
faq:
  - q: Why isn't my .gitignore working?
    a: "The most common reason is that the file is already tracked. .gitignore only stops Git from picking up files it isn't already following — once a file has been committed, ignore rules are silent on it. Untrack it while keeping it on disk with git rm --cached <file>, then commit. If the file is genuinely untracked, double-check the pattern with git check-ignore -v <file>, which tells you which rule, if any, matches."
  - q: What is the difference between a trailing slash and a leading slash in a pattern?
    a: "A trailing slash like build/ matches only directories, so Git ignores the folder and everything inside it. A leading slash like /config.yml anchors the pattern to the directory containing the .gitignore file, so it matches only that top-level file and not config.yml in subfolders. Without a leading slash, a pattern matches at any depth."
  - q: How do I ignore files for just myself without changing the shared .gitignore?
    a: "You have two options. For a single repository, add patterns to .git/info/exclude, which is local and never committed. For every repository on your machine, set a global ignore file with git config --global core.excludesFile ~/.gitignore_global and list patterns there. Both are ideal for editor and OS files that are personal to your setup."
  - q: Can I un-ignore a file inside an ignored directory?
    a: "Negation with a leading ! re-includes a previously matched path, but with a catch — Git will not look inside a directory it has already excluded. So to keep one file inside an ignored folder you must not ignore the folder wholesale; instead ignore its contents and then negate, for example logs/* followed by !logs/.gitkeep."
---

# .gitignore & tracking the right files

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **`.gitignore`** file tells Git which paths to leave untracked — keeping **secrets**,
**build artifacts**, and **OS cruft** out of your history. Patterns use globs, where
**`*`** matches within a path segment, **`**`** spans directories, a **leading `/`**
anchors to the file's folder, a **trailing `/`** matches only directories, and a
leading **`!`** re-includes. The big gotcha: `.gitignore` does **not** affect files
already tracked — fix that with **`git rm --cached`**. Use **`git check-ignore -v`**
to debug which rule matched.
</div>

Not everything in your project belongs in version control. Compiled binaries,
dependency folders, editor settings, and especially credentials should stay out.
`.gitignore` is how you draw that line, and getting its rules right saves you from
both clutter and security incidents.

## Why ignore files at all?

Committing the wrong files causes real problems. Three categories almost always
belong in `.gitignore`:

| Category | Examples | Why |
|----------|----------|-----|
| **Secrets** | `.env`, `*.pem`, `credentials.json` | leaking keys is a security incident |
| **Build artifacts** | `dist/`, `*.o`, `node_modules/` | regenerated, bloat history |
| **OS / editor cruft** | `.DS_Store`, `Thumbs.db`, `.idea/` | personal, not part of the project |

Build artifacts also cause needless merge conflicts, and dependency folders can be
huge. A clean repository contains source, not the things derived from it.

## Pattern syntax

A `.gitignore` is a plain list of patterns, one per line. The grammar is small:

```bash
# Lines starting with # are comments

# Ignore a specific file anywhere in the tree
secret.txt

# Ignore by extension (* matches within one path segment)
*.log

# Anchor to this directory only (leading slash)
/config.local.yml

# Ignore a whole directory and its contents (trailing slash)
build/

# Match across directories with a double star
docs/**/*.tmp

# Re-include an exception with !
*.log
!important.log
```

The rules in plain words:

- **`*`** matches any run of characters *except* a slash — it stays within one path
  segment.
- **`**`** matches across directory boundaries, so `a/**/b` matches `a/b`, `a/x/b`,
  and deeper.
- A **leading `/`** anchors the pattern to the directory containing the `.gitignore`.
- A **trailing `/`** restricts the match to directories.
- A leading **`!`** **negates** a previous match, re-including the path.
- A leading **`#`** is a comment; blank lines are ignored.

## Precedence and per-directory files

A repository can have many `.gitignore` files — one at the root and others in
subdirectories. A pattern's reach is the directory it lives in and everything below.
When patterns compete, **the last matching rule wins**, and rules in a deeper
directory override those higher up. This is why negation order matters:

```bash
logs/*
!logs/.gitkeep
```

The first line ignores everything in `logs/`; the second re-includes the placeholder.
Reverse the order and the negation would be overwritten and have no effect. Note also
that Git will not descend into a directory it has already excluded wholesale, so to
keep one file inside, ignore the *contents* (`logs/*`) rather than the folder
(`logs/`).

## The already-tracked gotcha

This trips up nearly everyone: **`.gitignore` only applies to untracked files.** If a
file is already committed, adding it to `.gitignore` does nothing — Git keeps tracking
it. The fix is to stop tracking it while leaving it on disk:

```bash
$ git rm --cached config.local.yml
rm 'config.local.yml'
$ git commit -m "Stop tracking local config"
```

`git rm --cached` removes the file from the index but not from your working tree. For
a whole directory, add `-r`. After this commit, the matching `.gitignore` rule finally
takes effect and the file stays out for everyone who pulls the change.

## Ignoring without touching the shared file

Sometimes you want to ignore files only for yourself — your editor's folder, a scratch
directory. Two mechanisms keep these out of the committed `.gitignore`:

| Where | Scope | Set with |
|-------|-------|----------|
| `.git/info/exclude` | this repo, local only | edit the file directly |
| global ignore file | every repo on your machine | `core.excludesFile` |

```bash
$ git config --global core.excludesFile ~/.gitignore_global
# then add patterns like .DS_Store and .idea/ to ~/.gitignore_global
```

Neither is committed, so they're perfect for OS and editor cruft that shouldn't be
imposed on collaborators.

## Debugging with check-ignore

When a file is mysteriously ignored — or stubbornly *not* ignored — `git check-ignore -v`
tells you exactly which rule, in which file, is responsible:

```bash
$ git check-ignore -v build/app.bin
.gitignore:7:build/    build/app.bin
```

The output is `file:line:pattern` followed by the path. No output means no rule
matches. You don't have to write every pattern by hand, either: **gitignore.io** and
GitHub's [gitignore templates](/learn/git/glossary/) repository generate sensible
starter files for any language or framework. For where `.gitignore` fits in the
overall workflow, revisit [Staging & commits](/learn/git/staging-and-commits/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — ignore rules don't apply to tracked files, so you must untrack it first." markdown="0">
  <p class="knowledge-check__q">Quick check: you added <code>config.yml</code> to <code>.gitignore</code> but Git still tracks it. What fixes it?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Run <code>git rm --cached config.yml</code> and commit</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Delete <code>config.yml</code> from disk and re-create it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Add a second <code>.gitignore</code> in the same folder</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- `.gitignore` keeps **secrets**, **build artifacts**, and **OS cruft** untracked.
- Patterns: `*` within a segment, `**` across directories, leading `/` anchors,
  trailing `/` means directory-only, leading `!` re-includes, `#` comments.
- The **last matching rule wins**, and deeper `.gitignore` files override higher ones.
- Ignore rules **don't apply to already-tracked files** — untrack with
  `git rm --cached`.
- Keep personal patterns out of the shared file via `.git/info/exclude` or a global
  `core.excludesFile`.
- Debug with `git check-ignore -v`; generate starters with gitignore.io or templates.

Next up: undoing changes safely with restore, reset, and revert.
