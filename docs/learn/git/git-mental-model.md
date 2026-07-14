---
slug: git-mental-model
title: "How Git thinks: snapshots, not diffs"
description: The mental model behind Git — the three areas, the blob/tree/commit object model, content hashing, the commit DAG, and how branches, tags, and HEAD are just pointers.
keywords: Git mental model, snapshots not diffs, working tree, staging area, index, Git object model, blob tree commit, SHA-1 hash, Git DAG, HEAD branch pointer, git cat-file
level: beginner
status: full
prereq:
  - what-is-version-control
faq:
  - q: Does Git store the differences between versions?
    a: Conceptually, no — each commit references a complete snapshot of your project's tree at that moment. Git is clever about storage (it deduplicates unchanged files and later compresses objects into packfiles using deltas), but the model you reason about is full snapshots, not a chain of diffs. Diffs are *computed* on demand when you ask to compare two snapshots.
  - q: What are the three areas in Git?
    a: The **working tree** is the files you edit on disk. The **staging area** (also called the index) is where you assemble the exact set of changes for your next commit. The **repository** is the `.git` directory holding the committed history. Files move working tree → staging → repository as you `git add` then `git commit`.
  - q: What is a SHA in Git?
    a: Every object Git stores (blob, tree, or commit) is named by a hash of its content — historically SHA-1, now optionally SHA-256. Identical content always produces the same hash, so Git deduplicates automatically and can detect corruption. A commit's hash is the 40-character ID you see in `git log`.
  - q: Are branches expensive to create in Git?
    a: No. A branch is simply a small file containing the hash of one commit — a movable pointer. Creating one writes 41 bytes; it does not copy your files. This is why branching in Git is effectively free and why teams branch so freely.
---

# How Git thinks: snapshots, not diffs

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Git models your project as three areas — the **working tree**, the **staging area**
(index), and the **repository**. Internally it stores three object types:
**blobs** (file contents), **trees** (directories), and **commits** (a full
**snapshot** plus its parent). Every object is named by a **content hash (SHA)**, so
identical content is stored once and corruption is detectable. Commits link to their
parents to form a **DAG**, and **branches, tags, and HEAD are just pointers** into
it. Git stores *snapshots, not diffs* — which is what makes it fast and safe.
</div>

You don't need to know Git's internals to use it, but a short tour of how it
*thinks* makes every later command feel obvious instead of magical. This builds on
[Why version control?](/learn/git/what-is-version-control/).

## The three areas

Working with Git means moving changes through three places:

```text
  working tree   ── git add ──▶   staging area   ── git commit ──▶  repository
  (files you      (the index:        (.git: the
   edit on disk)   your next          committed
                   snapshot draft)    history)
```

- **Working tree** — the actual files in your project folder, where you make edits.
- **Staging area (index)** — a draft of your *next* snapshot. You choose exactly
  what goes in it with `git add`, which lets you commit some changes and leave
  others for later.
- **Repository** — the `.git` directory, where committed snapshots live permanently.

The next two lessons make this concrete:
[Your first repository](/learn/git/first-repository/) and
[The staging area & commits](/learn/git/staging-and-commits/).

## Snapshots, not diffs

Many older systems store a file as an original plus a chain of diffs. Git takes a
different view: **each commit records a complete snapshot** of what every tracked
file looked like at that moment.

That sounds wasteful, but it isn't. If a file doesn't change between commits, Git
stores it **once** and points both snapshots at the same content (this is the hash
trick, below). Later, Git compresses objects into *packfiles* that do use deltas for
storage efficiency — but the model you reason about stays "full snapshots." Diffs
are *computed* when you ask for them, as you'll see in
[Status & diffs](/learn/git/status-and-diffs/).

## The object model: blob, tree, commit

Under the hood Git stores just three kinds of object:

| Object | Holds | Roughly equals |
|--------|-------|----------------|
| **blob** | the raw bytes of one file | a file's *contents* (no name) |
| **tree** | a list of names → blobs/trees | a *directory* listing |
| **commit** | a tree + parent(s) + author + message | a *snapshot* of the whole project |

A commit points at one top-level tree, which points at sub-trees and blobs — together
capturing the entire project at that instant. Note that a blob stores *content only*;
the filename lives in the tree that references it.

## Content hashing (the SHA)

Every object is named by a **hash of its own content**. You can watch Git compute
one:

```bash
$ echo "hello git" | git hash-object --stdin
8d0e41234f24b6da002d962a26c2495ea16a425f
```

And you can read any stored object back with `git cat-file`:

```bash
$ git cat-file -p HEAD
tree 9f3a...c1
parent 4b2e...90
author Ada Lovelace <ada@example.com> 1718668800 +0000
committer Ada Lovelace <ada@example.com> 1718668800 +0000

Add project README
```

Two consequences fall out of content-addressing:

- **Deduplication** — identical content yields the same hash, so it's stored once.
- **Integrity** — change a single byte and the hash changes, so silent corruption
  is detectable. The commit hash effectively verifies its entire history.

(Git historically used SHA-1 and now supports SHA-256; the principle is identical.)

## Commits form a DAG

Each commit records its **parent** — the commit that came before it. Follow those
parent links and you walk back through history. Because a merge commit has *two*
parents, history forms a **directed acyclic graph (DAG)**, not a simple line:

```text
A ── B ── C ── D        (main)
       \         \
        E ── F ─── G     (a branch, merged back at G)
```

This structure is what makes [branches](/learn/git/branches/),
[merging](/learn/git/merging/), and [viewing history](/learn/git/viewing-history/)
possible.

## Branches, tags, and HEAD are just pointers

Here's the payoff. A **branch** is nothing but a small file holding the hash of one
commit — a movable pointer. A **tag** is a pointer that doesn't move (used for
[releases](/learn/git/releases-and-tags/)). **HEAD** is a pointer to *the branch you
currently have checked out*.

```text
HEAD ─▶ main ─▶ D            "I'm on main, whose latest commit is D"
        feature ─▶ G
```

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 210" role="img" aria-label="A commit graph: commits A through D form the main line and commits E and F branch off commit B, each commit linked to its parent by an arrow. Small rounded pills labelled HEAD, main, feature, and a v1.0 tag point at individual commits, showing that branches, tags, and HEAD are just movable labels attached to commits." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.4" fill="none">
    <line x1="138" y1="120" x2="72" y2="120" marker-end="url(#mm_ar)"/>
    <line x1="238" y1="120" x2="162" y2="120" marker-end="url(#mm_ar)"/>
    <line x1="338" y1="120" x2="262" y2="120" marker-end="url(#mm_ar)"/>
    <line x1="242" y1="64" x2="160" y2="110" marker-end="url(#mm_ar)"/>
    <line x1="338" y1="56" x2="262" y2="56" marker-end="url(#mm_ar)"/>
  </g>
  <g fill="currentColor" font-size="9" text-anchor="middle">
    <circle cx="60" cy="120" r="12" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="60" y="123">A</text>
    <circle cx="150" cy="120" r="12" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="150" y="123">B</text>
    <circle cx="250" cy="120" r="12" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="250" y="123">C</text>
    <circle cx="350" cy="120" r="12" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="350" y="123">D</text>
    <circle cx="250" cy="56" r="12" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="250" y="59">E</text>
    <circle cx="350" cy="56" r="12" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="350" y="59">F</text>
  </g>
  <g font-size="8.5" text-anchor="middle" font-weight="600">
    <line x1="350" y1="44" x2="350" y2="36" stroke="currentColor" stroke-width="1" stroke-opacity="0.6"/>
    <rect x="318" y="18" width="64" height="18" rx="4" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/><text x="350" y="30" fill="currentColor">feature</text>
    <line x1="350" y1="132" x2="350" y2="150" stroke="currentColor" stroke-width="1" stroke-opacity="0.6"/>
    <rect x="318" y="150" width="64" height="18" rx="4" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/><text x="350" y="162" fill="currentColor">main</text>
    <rect x="236" y="150" width="58" height="18" rx="4" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/><text x="265" y="162" fill="currentColor">HEAD</text>
    <line x1="294" y1="159" x2="314" y2="159" stroke="currentColor" stroke-width="1.2" marker-end="url(#mm_ar)"/>
    <line x1="150" y1="132" x2="150" y2="150" stroke="currentColor" stroke-width="1" stroke-opacity="0.6" stroke-dasharray="3 2"/>
    <rect x="118" y="150" width="64" height="18" rx="4" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1" stroke-dasharray="3 2"/><text x="150" y="162" fill="currentColor">tag v1.0</text>
  </g>
  <text x="230" y="192" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">every arrow points at a parent · every pill is just a label pointing at a commit</text>
  <defs><marker id="mm_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Commits link to their parents to form the graph (a merge commit would point at two). Everything else — <strong>main</strong>, <strong>feature</strong>, the <strong>v1.0</strong> tag, and <strong>HEAD</strong> — is just a lightweight label pointing at one commit. HEAD points at the branch you're on; making a commit slides that branch's pill forward.</figcaption>
</figure>

Make a commit and Git creates the new snapshot, then nudges the current branch
pointer forward to it. That's why creating a branch is instant: it writes a 41-byte
file, not a copy of your project. These pointers are called **refs**, and you'll see
them again in the [glossary](/learn/git/glossary/).

<div class="knowledge-check" data-quiz data-correct-msg="Correct — a branch is just a movable pointer to a commit." markdown="0">
  <p class="knowledge-check__q">Quick check: in Git's model, what is a branch?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">A movable pointer to a single commit</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A full copy of every file in the project</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A stored list of diffs since the last commit</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Git has three areas: **working tree → staging area (index) → repository**.
- Each commit stores a **full snapshot**, not a diff; Git deduplicates and compresses
  behind the scenes.
- The object model is **blob** (file contents), **tree** (directory), **commit**
  (snapshot + parent).
- Every object is named by a **content hash (SHA)**, giving deduplication and
  integrity.
- Commits link via parents into a **DAG**; **branches, tags, and HEAD are just
  pointers** into it.

Next up: creating your very first repository.
