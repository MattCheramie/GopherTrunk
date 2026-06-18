---
slug: releases-and-tags
title: Tags, releases & semantic versioning
description: Mark versions with Git tags (lightweight vs annotated), publish GitHub Releases with notes and binaries, and apply semantic versioning so MAJOR.MINOR.PATCH means something.
keywords: git tag, annotated tag, lightweight tag, git push tags, github releases, semantic versioning, semver, major minor patch, pre-release, release notes
level: intermediate
status: full
prereq:
  - viewing-history
faq:
  - q: What is the difference between a lightweight and an annotated Git tag?
    a: A lightweight tag is just a name pointing at a commit, like a branch that never moves — it stores no extra information. An annotated tag is a full object in Git's database with a tagger name, date, message, and optional GPG signature. Use annotated tags for releases (git tag -a v1.2.0 -m "..."), because they record who tagged what and when; lightweight tags are fine for private, temporary markers.
  - q: Why doesn't git push send my tags?
    a: By default git push only pushes branch commits, not tags. You push a single tag with "git push origin v1.2.0", or all of them at once with "git push origin --tags". This separation is deliberate — it stops you from accidentally publishing local experimental tags. Remember the push step or your tag exists only on your machine.
  - q: How does semantic versioning decide which number to bump?
    a: Semantic versioning uses MAJOR.MINOR.PATCH. Bump PATCH for backwards-compatible bug fixes, MINOR for backwards-compatible new features, and MAJOR for breaking changes that require users to adjust their code. So 1.4.2 to 1.4.3 is a bug fix, 1.4.2 to 1.5.0 adds features safely, and 1.4.2 to 2.0.0 warns of incompatibility.
  - q: What is the difference between a Git tag and a GitHub Release?
    a: A Git tag is a plain Git concept — a named pointer to a commit that lives in any clone of the repo. A GitHub Release is a layer GitHub builds on top of a tag, adding a title, formatted release notes, downloadable binary assets, and a draft/pre-release status, all shown on the repository's Releases page. Every Release is attached to a tag, but not every tag has a Release.
---

# Tags, releases & semantic versioning

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **tag** is a permanent name for a specific commit — a bookmark for "this is version
1.2.0." Prefer **annotated** tags (`git tag -a v1.2.0 -m "..."`) over lightweight ones
for releases, and remember tags don't push by default (**`git push origin v1.2.0`** or
**`--tags`**). A **GitHub Release** builds on a tag, adding release **notes** (which
GitHub can **auto-generate**), downloadable **binary assets**, and **draft**/**pre-release**
flags. Name versions with **Semantic Versioning** — **MAJOR.MINOR.PATCH** — so the
number signals what changed. A tag push can even **trigger a release workflow** via
[GitHub Actions](/learn/git/github-actions/).
</div>

You can already [navigate history](/learn/git/viewing-history/) with hashes, but
`a1b2c3d` is a poor name for "the version we shipped to customers." **Tags** give
meaningful, permanent names to important commits, and **Releases** turn a tag into
something users can read and download. Let's start with the Git primitive.

## Git tags: lightweight vs annotated

A tag points at one commit and never moves — unlike a branch, it doesn't advance when
you commit. There are two kinds.

A **lightweight** tag is just a name on a commit, with no metadata:

```bash
$ git tag v1.2.0-quick
```

An **annotated** tag is a real object storing the tagger, a date, a message, and an
optional signature — the right choice for releases:

```bash
$ git tag -a v1.2.0 -m "Release 1.2.0 — dark mode and bug fixes"
```

| | Lightweight | Annotated |
|---|---|---|
| Stores message/author/date | No | Yes |
| Can be GPG-signed | No | Yes |
| Best for | Quick private markers | Releases |

Rule of thumb: if anyone other than you will see it, make it annotated.

## Listing, pushing, and deleting tags

List tags (optionally filtered) and inspect one:

```bash
$ git tag                 # list all tags
$ git tag -l "v1.2.*"     # filter by pattern
$ git show v1.2.0         # annotated tag's message + the commit
```

Tags are **not** pushed by `git push` alone. Push one explicitly, or push them all:

```bash
$ git push origin v1.2.0   # push a single tag
$ git push origin --tags   # push every local tag
```

Deleting takes two steps — local and remote are separate:

```bash
$ git tag -d v1.2.0                  # delete locally
$ git push origin --delete v1.2.0    # delete on the remote
```

To tag a commit other than the current one, append its hash:
`git tag -a v1.1.0 9fceb02`.

## Semantic Versioning: making the number mean something

A version like `2.4.1` is only useful if the digits follow a convention. **Semantic
Versioning (SemVer)** is the dominant one: **`MAJOR.MINOR.PATCH`**, where each part
signals a different kind of change.

| Part | Bump it when… | Example |
|------|---------------|---------|
| **MAJOR** | You make a **breaking** change | `1.4.2 → 2.0.0` |
| **MINOR** | You add **backwards-compatible** features | `1.4.2 → 1.5.0` |
| **PATCH** | You make **backwards-compatible** bug fixes | `1.4.2 → 1.4.3` |

When you bump MINOR you reset PATCH to 0; when you bump MAJOR you reset both. The
contract is a promise to your users: a PATCH or MINOR upgrade is safe, a MAJOR upgrade
might require changes on their side.

## Pre-release and build metadata

SemVer has two optional suffixes for versions that aren't a finished release. A
**pre-release** is appended with a hyphen and is considered *lower* than the matching
final release:

```text
1.0.0-alpha   <   1.0.0-beta   <   1.0.0-rc.1   <   1.0.0
```

**Build metadata** is appended with a plus sign and is ignored for ordering — it's just
a label:

```text
1.0.0+20260618        1.0.0-beta+exp.sha.5114f85
```

So `2.0.0-rc.1` means "release candidate for 2.0.0, not the real thing yet," which is
exactly what you'd mark as a **pre-release** on GitHub.

## GitHub Releases: tags users can read and download

A **GitHub Release** is the human-facing layer on top of a tag. From the repo's
**Releases** page, choose **Draft a new release**, pick (or create) a tag like `v1.2.0`,
then add:

- a **title** and formatted **release notes** in Markdown — describe what changed;
- **auto-generated notes**: click **Generate release notes** and GitHub drafts a
  changelog from the merged pull requests since the last tag;
- **binary assets**: upload compiled artifacts (installers, zips) that users download
  directly from the release;
- a status flag — mark it a **draft** (saved but not published) or a **pre-release**
  (visible but flagged as not production-ready).

The published release appears on the Releases page and as the repo's "Latest release"
badge, giving users a clear, downloadable version rather than a bare commit hash.

## Tying it together: tag-triggered release workflows

Tags and releases shine when automated. Because a tag push is an event,
[GitHub Actions](/learn/git/github-actions/) can run a workflow whenever you push a
version tag — building binaries and publishing the GitHub Release for you:

```yaml
name: Release
on:
  push:
    tags:
      - "v*"        # any tag starting with v
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: ./build.sh        # produce the binaries
      - uses: softprops/action-gh-release@v2
        with:
          generate_release_notes: true
          files: dist/*
```

Now your release process is: bump the version, `git tag -a v1.3.0 -m "..."`,
`git push origin v1.3.0`, and the workflow does the rest. (Unsure what a "ref" or
"artifact" is? The [glossary](/learn/git/glossary/) explains them.)

<div class="knowledge-check" data-quiz data-correct-msg="Right — a backwards-compatible new feature bumps MINOR." markdown="0">
  <p class="knowledge-check__q">Quick check: under SemVer, you add a new feature without breaking anything. Which part bumps?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">MAJOR (1.4.2 → 2.0.0)</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">MINOR (1.4.2 → 1.5.0)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">PATCH (1.4.2 → 1.4.3)</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **tag** permanently names a commit; prefer **annotated** (`git tag -a … -m …`) for
  releases.
- Tags don't push by default — use **`git push origin <tag>`** or **`--tags`**; delete
  locally and remotely separately.
- **Semantic Versioning** is **MAJOR.MINOR.PATCH**: breaking, feature, fix — with
  `-pre` and `+build` suffixes.
- A **GitHub Release** adds notes (auto-generatable), binary assets, and
  draft/pre-release flags on top of a tag.
- A **tag push can trigger** a release [workflow](/learn/git/github-actions/) to build
  and publish automatically.

Next up: a lighter tour of [Pages, Gists, Discussions & wikis](/learn/git/github-pages-and-more/).
