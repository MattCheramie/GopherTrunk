---
slug: first-repository
title: Your first repository
description: Create your first Git repository with git init or git clone, learn what lives inside the .git directory, add a README, and read your first git status output.
keywords: git init, git clone, first git repository, create git repo, .git directory, working tree, git status, README, initialize repository, untracked files
level: beginner
status: full
prereq:
  - installing-git
faq:
  - q: What is the difference between git init and git clone?
    a: "`git init` creates a brand-new, empty repository in the current folder — use it to start a fresh project. `git clone <url>` copies an existing repository (and its full history) from somewhere else, such as GitHub — use it to start working on a project that already exists."
  - q: What is the .git directory?
    a: It's the hidden folder Git creates at the root of a repository to store everything it tracks — the object database, references (branches and tags), the HEAD pointer, and the repo's config. Your project files sit alongside it in the working tree. Delete `.git` and you delete the history, turning the folder back into ordinary files.
  - q: Why does git status call my new file "untracked"?
    a: Git only follows files you've told it to. A file you've created but never `git add`-ed is *untracked* — Git sees it but isn't recording its history yet. Staging it with `git add` brings it under version control, ready for your first commit.
  - q: Can I turn an existing folder of files into a repository?
    a: Yes. Run `git init` inside the folder; Git creates the `.git` directory without touching your files. They'll all show as untracked until you `git add` and commit them.
---

# Your first repository

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A Git project lives in a **repository**. Start a fresh one with **`git init`**, or
copy an existing one with **`git clone`**. The repository is just your normal files
(the **working tree**) plus a hidden **`.git`** directory that stores the object
database, **refs**, **HEAD**, and config. After creating a repo and adding a file
like a **README**, **`git status`** shows it as **untracked** — Git sees it but
isn't recording it yet. That sets up the next step: staging and committing.
</div>

You've got Git [installed and configured](/learn/git/installing-git/). Let's create
a repository and look inside it, using the
[three-area model](/learn/git/git-mental-model/) from the last lesson.

## init vs clone

There are two ways to get a repository, depending on whether the project already
exists:

| You want to… | Command | Result |
|--------------|---------|--------|
| Start a brand-new project | `git init` | Empty repo in the current folder |
| Work on an existing project | `git clone <url>` | Full copy, including history |

You'll use `git clone` constantly once you reach [Remotes](/learn/git/remotes/) and
[Forking](/learn/git/forking/). For now we'll start fresh with `git init`.

## Creating a repository with git init

Make a folder, move into it, and initialize:

```bash
$ mkdir hello-git
$ cd hello-git
$ git init
Initialized empty Git repository in /home/you/hello-git/.git/
```

That's it — `hello-git` is now a repository. The only thing that changed is a new
hidden `.git` folder; your working tree (currently empty) is otherwise untouched.

## What lives in .git?

You rarely edit anything in `.git` by hand, but knowing what's there demystifies
Git. A freshly initialized repo looks roughly like this:

```bash
$ ls .git
HEAD  config  description  hooks/  info/  objects/  refs/
```

| Item | What it holds |
|------|---------------|
| `objects/` | The object database — every blob, tree, and commit |
| `refs/` | Branch and tag pointers (e.g. `refs/heads/main`) |
| `HEAD` | A pointer to the branch you currently have checked out |
| `config` | This repository's **local** settings |
| `hooks/` | Optional scripts that run on Git events (see later) |

These map directly onto the [object model and pointers](/learn/git/git-mental-model/)
you just learned. Notice `objects/` is empty right now — you haven't committed
anything yet. Delete `.git` and you'd be left with plain files and no history, so
treat it with respect.

## Adding a README

Most projects open with a README that explains what they are. Create one:

```bash
$ echo "# Hello Git" > README.md
```

This is just an ordinary file in the working tree. Git has noticed it exists, but it
isn't tracking its history yet — let's confirm that.

## Reading your first git status

`git status` is the command you'll run more than any other. It reports where you
stand across the three areas:

```bash
$ git status
On branch main

No commits yet

Untracked files:
  (use "git add <file>..." to include in what will be committed)
        README.md

nothing added to commit but untracked files present (use "git add" to track)
```

Three things to read here:

- **On branch main** — your default branch, thanks to `init.defaultBranch main`.
- **No commits yet** — the history is empty; `objects/` has nothing in it.
- **Untracked files: README.md** — Git sees the file but isn't recording it. An
  **untracked** file is one Git has never been told to follow.

Helpfully, Git even tells you the next move: *use `git add` to track*. We'll explore
everything `git status` can show in
[Status & diffs](/learn/git/status-and-diffs/).

## What comes next

Right now `README.md` is sitting in your working tree, untracked. To bring it under
version control you'll **stage** it with `git add` and record a snapshot with
`git commit` — which is exactly the subject of the next lesson. Don't run those yet;
we'll do it step by step there so each part is clear.

If a term here was unfamiliar, the [glossary](/learn/git/glossary/) has short
definitions for all of them.

<div class="knowledge-check" data-quiz data-correct-msg="Correct — git init creates a new, empty repository." markdown="0">
  <p class="knowledge-check__q">Quick check: which command starts a brand-new, empty repository?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">git clone</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">git status</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">git init</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Get a repo with **`git init`** (new) or **`git clone`** (existing).
- A repository is your **working tree** plus a hidden **`.git`** directory.
- `.git` holds **`objects/`**, **`refs/`**, **`HEAD`**, and the local **config**.
- A newly created file shows as **untracked** until you stage it.
- **`git status`** is your constant compass across the three areas.

Next up: the staging area and your first commit.
