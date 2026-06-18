---
slug: remotes
title: "Remotes: clone, fetch, pull & push"
description: Work with Git remotes — clone, fetch, pull, and push explained clearly. Learn origin, remote-tracking branches, upstream, pull --rebase, and deleting branches.
keywords: git remote, git clone, git fetch, git pull, git push, origin, remote tracking branch, git push -u, upstream, pull rebase, delete remote branch
level: intermediate
status: full
prereq:
  - branches
  - what-is-github
faq:
  - q: What is the difference between git fetch and git pull?
    a: git fetch only downloads new commits from the remote into your remote-tracking branches (like origin/main); it does not touch your working files or current branch. git pull does a fetch and then immediately merges (or rebases) those commits into your current branch. Fetch is the safe, look-before-you-leap option; pull is fetch plus integrate in one step.
  - q: What does origin mean in Git?
    a: origin is just the default name Git gives to the remote you cloned from. It is a convention, not a keyword — you could rename it or add remotes with other names. When you run git clone, Git automatically creates a remote called origin pointing at the URL you cloned, which is why so many commands mention origin.
  - q: What does git push -u origin main do?
    a: It pushes your main branch to the remote called origin and, thanks to -u (short for --set-upstream), records that your local main should track origin/main. After that, you can simply run git push and git pull with no arguments because Git knows which remote branch they correspond to. You only need -u the first time you push a new branch.
  - q: How do I delete a branch on the remote?
    a: Run git push origin --delete <branch>. This removes the branch from the remote only; your local copy of the branch is untouched. To also delete it locally, use git branch -d <branch>. Deleting the remote branch is common housekeeping after a pull request has been merged.
---

# Remotes: clone, fetch, pull & push

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **remote** is a named link to a repository hosted elsewhere; the default one is
called **`origin`**. **`git clone`** copies a remote down to start; **`git push`**
uploads your commits; and the pair people confuse most — **`git fetch`** only
*downloads* commits, while **`git pull`** *fetches then merges* into your branch.
Git tracks the remote's state in **remote-tracking branches** like `origin/main`.
The first time you push a new branch, use **`git push -u origin <branch>`** to set
its upstream. Prefer **`git pull --rebase`** for a tidy history, and delete remote
branches with **`git push origin --delete <branch>`**.
</div>

You met the idea of a [remote](/learn/git/what-is-github/) in the last lesson: a copy
of your repository living on a server. This lesson is the practical half — the
commands that move commits between your machine and that server. These four verbs
(clone, fetch, pull, push) are the daily vocabulary of working with GitHub.

## What a remote is, and the `origin` convention

A remote is simply a **named bookmark** for a repository URL. List your remotes with
`git remote -v` (the `-v` means verbose, showing the URLs):

```bash
$ git remote -v
origin  git@github.com:alice/myproject.git (fetch)
origin  git@github.com:alice/myproject.git (push)
```

By convention the remote you cloned from is named **`origin`**. There's nothing
magic about the name — it's just the default Git picks. You can add more remotes by
hand with `git remote add <name> <url>`:

```bash
$ git remote add backup git@gitlab.com:alice/myproject.git
$ git remote -v
backup  git@gitlab.com:alice/myproject.git (fetch)
backup  git@gitlab.com:alice/myproject.git (push)
origin  git@github.com:alice/myproject.git (fetch)
origin  git@github.com:alice/myproject.git (push)
```

Most projects have just `origin`. A second remote called `upstream` is common when
you're [forking](/learn/git/forking/) someone else's project — more on that later.

## Clone: starting from a remote

To get a local copy of a repository that already exists on GitHub, **clone** it.
Cloning downloads the full history *and* automatically sets up `origin` for you:

```bash
$ git clone git@github.com:alice/myproject.git
Cloning into 'myproject'...
remote: Enumerating objects: 412, done.
remote: Total 412 (delta 0), reused 412 (delta 0), pack-reused 0
Receiving objects: 100% (412/412), 1.83 MiB | 4.10 MiB/s, done.
Resolving deltas: 100% (210/210), done.
```

You now have a directory `myproject/` with the entire project and its history, an
`origin` remote pointing back at GitHub, and a local `main` already tracking
`origin/main`. From here, the other three verbs keep the two copies in sync.

## Fetch vs pull vs push — the crucial distinction

This is the part worth slowing down for. The three commands move data in different
directions and do different amounts of work.

| Command | Direction | What it does |
|---------|-----------|--------------|
| `git fetch` | remote → you | Downloads new commits, but **does not** change your branch or files |
| `git pull` | remote → you | Runs `fetch`, then **merges (or rebases)** into your current branch |
| `git push` | you → remote | Uploads your local commits to the remote |

**Fetch** is the cautious move. It updates your knowledge of the remote without
disturbing your work:

```bash
$ git fetch origin
remote: Enumerating objects: 8, done.
From github.com:alice/myproject
   a1b2c3d..e4f5a6b  main       -> origin/main
```

Notice it updated `origin/main`, not your `main`. You can now inspect what changed
before integrating it.

**Pull** does the fetch *and* the merge in one go:

```bash
$ git pull
Updating a1b2c3d..e4f5a6b
Fast-forward
 README.md | 4 ++--
 1 file changed, 2 insertions(+), 2 deletions(-)
```

**Push** sends your commits the other way:

```bash
$ git push
Enumerating objects: 5, done.
To github.com:alice/myproject.git
   e4f5a6b..c7d8e9f  main -> main
```

Rule of thumb: `fetch` to look, `pull` to look-and-integrate, `push` to share.

## Remote-tracking branches vs local branches

When you fetch, Git stores the remote's state in **remote-tracking branches** —
read-only pointers named like `origin/main`. These are Git's local cache of "where
the remote was last time we talked to it." They are distinct from your own `main`:

```text
   main          ← your local branch (you commit here)
   origin/main   ← remote-tracking branch (updated by fetch/pull)
```

You never commit directly to `origin/main`; it moves only when you fetch or pull.
This separation is what lets you compare your work against the server's:

```bash
$ git log main..origin/main   # commits on the remote you don't have yet
$ git log origin/main..main   # commits you have that aren't pushed yet
```

## Setting upstream and pulling with rebase

The first time you push a *new* branch, tell Git which remote branch it should
track using `-u` (short for `--set-upstream`):

```bash
$ git switch -c feature/login
$ git push -u origin feature/login
...
branch 'feature/login' set up to track 'origin/feature/login'.
```

After that, plain `git push` and `git pull` work with no arguments because Git
knows the pairing. You only need `-u` once per branch.

When you pull, the default is to *merge* the remote commits, which can create extra
"merge" commits. Many people prefer **`git pull --rebase`**, which replays your
local commits on top of the fetched ones for a straight-line history (you'll learn
the mechanics in [rebasing](/learn/git/rebasing/)):

```bash
$ git pull --rebase
Successfully rebased and updated refs/heads/main.
```

You can make rebase the default with `git config --global pull.rebase true`.

## Deleting remote branches and multiple remotes

Once a feature branch is merged, tidy up the remote copy with `--delete`:

```bash
$ git push origin --delete feature/login
To github.com:alice/myproject.git
 - [deleted]         feature/login
```

That removes the branch on the server only; delete your local copy separately with
`git branch -d feature/login`.

Finally, remember that a repo can have **multiple remotes**. The classic case is
contributing to a project you don't own: you fork it, clone *your* fork as `origin`,
and add the *original* project as a second remote called `upstream` so you can pull
in its updates. That whole pattern is the subject of the next lesson.

<div class="knowledge-check" data-quiz data-correct-msg="Right — fetch downloads only; pull fetches and integrates." markdown="0">
  <p class="knowledge-check__q">Quick check: you want to see what's new on the remote without changing your current branch. Which command?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">git pull</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">git fetch</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">git push</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **remote** is a named link to a hosted repo; the default is **`origin`**.
- **`git clone`** copies a remote down and sets up `origin` automatically.
- **`git fetch`** downloads only; **`git pull`** fetches *and* integrates; **`git push`** uploads.
- **Remote-tracking branches** like `origin/main` are Git's cache of the remote's state.
- Use **`git push -u origin <branch>`** the first time; prefer **`git pull --rebase`** for tidy history.
- Delete a remote branch with **`git push origin --delete <branch>`**; add extra remotes for forks.

Next up: keeping things secure with [SSH keys and tokens](/learn/git/authentication/).
