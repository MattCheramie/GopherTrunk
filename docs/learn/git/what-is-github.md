---
slug: what-is-github
title: What is GitHub?
description: GitHub explained for beginners — a hosting and collaboration platform built on top of Git. What a remote is, the headline features, and the alternatives.
keywords: what is github, github vs git, git hosting, remote repository, github features, gitlab, bitbucket, codeberg, gitea, create github account
level: beginner
status: full
prereq:
  - what-is-version-control
faq:
  - q: What is the difference between Git and GitHub?
    a: Git is the version control tool that runs on your computer and tracks changes to your files. GitHub is a website that hosts Git repositories online and adds collaboration features on top — pull requests, issue tracking, automation, and more. You can use Git with no GitHub account at all, and GitHub is just one of several places to host a Git repository. In short, Git is the engine; GitHub is one of the garages you can park it in.
  - q: Do I need GitHub to use Git?
    a: No. Git is fully functional on its own and many people use it entirely locally or with a self-hosted server. GitHub becomes useful when you want an off-machine backup, want to collaborate with others, or want to publish your work. It also provides extras like automated testing, project boards, and free static web hosting that pure Git does not.
  - q: Is GitHub free?
    a: Yes for most people. GitHub offers free public and private repositories, free collaborators, and a generous allowance of automation minutes and storage. Paid plans add more automation minutes, advanced security features, and organisation-level controls, but you can do everything in this learning path on a free account.
  - q: What is a remote repository?
    a: A remote is a copy of your repository that lives somewhere other than your computer — usually on a server like GitHub. You push your local commits up to the remote to back them up and share them, and pull other people's commits down. The remote and your local clone are both complete repositories; they simply synchronise with each other.
---

# What is GitHub?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Git** is the version control tool on your computer; **GitHub** is a website that
**hosts** Git repositories and adds **collaboration** on top. The thing GitHub hosts
is a **remote** — a copy of your repo living on a server that you push to and pull
from. GitHub's headline features are **repositories, pull requests, issues, Actions,
Pages, and Discussions**. It isn't the only option — **GitLab, Bitbucket, Codeberg,
and self-hosted Gitea** all host Git too. Sign up, create a repo, and the rest of
this module shows you how to connect your local work to it.
</div>

You've seen that [Git](/learn/git/what-is-version-control/) tracks your project's
history on your own machine. That's powerful on its own, but most people eventually
want to back their work up somewhere safe, share it, or collaborate. That's where
GitHub comes in. This lesson draws a crisp line between the two and tours what GitHub
actually offers.

## Git is the tool; GitHub is a place to host it

This is the single most important distinction in the whole module, so let's nail it.

- **Git** is a program that runs locally. It records commits, manages branches, and
  knows nothing about the internet. You can use Git forever without an account
  anywhere.
- **GitHub** is a commercial web service that stores Git repositories on its servers
  and wraps them in a friendly interface plus a pile of collaboration features.

A useful analogy: Git is like the camera that takes the photos; GitHub is like a
photo-sharing site where you upload albums so friends can view and comment. The
camera works without the website, and you could upload to a different site instead.

Because GitHub speaks plain Git underneath, everything you learned locally still
applies. You're just adding a server into the picture.

## What a remote is, conceptually

When your repository lives only on your laptop, it has one copy. The moment you put
a copy on GitHub, you have a second, networked copy called a **remote**.

Both copies are *complete* repositories — each has the full history, not a thin
mirror. They stay in sync through two operations you'll meet in detail in
[Remotes: clone, fetch, pull & push](/learn/git/remotes/):

- **push** — upload your new local commits to the remote.
- **pull** — download new commits from the remote into your local copy.

```text
   your laptop                       GitHub (the remote)
  ┌───────────┐    push  ───────►   ┌───────────┐
  │ local repo│                     │  origin   │
  └───────────┘   ◄───────  pull    └───────────┘
```

That's the heart of it: a remote is just another repository, kept somewhere else,
that you synchronise with. (If "repository" or "commit" feels fuzzy, the
[glossary](/learn/git/glossary/) is one click away.)

## The headline features at a glance

GitHub's value is everything it layers on top of plain Git hosting. Here's the
tour, with pointers to the lessons that cover each one.

| Feature | What it does | Learn more |
|---------|--------------|------------|
| **Repositories** | The hosted home for your project's code and history | [First repository](/learn/git/first-repository/) |
| **Pull requests** | Propose, review, and discuss merging one branch into another | [Pull requests](/learn/git/pull-requests/) |
| **Issues** | Track bugs, tasks, and ideas; organise with projects | [Issues & projects](/learn/git/issues-and-projects/) |
| **Actions** | Run automated tests and workflows on every push | [GitHub Actions](/learn/git/github-actions/) |
| **Pages** | Publish a free static website straight from a repo | [GitHub Pages & more](/learn/git/github-pages-and-more/) |
| **Discussions** | Forum-style conversations separate from issues | — |

You don't need all of these on day one. Most beginners start with just
repositories and gradually adopt the rest as collaboration grows.

## GitHub is not the only option

GitHub is the most popular host, but it's far from the only one. Since they all
speak the same Git protocol, the commands you'll learn work against any of them —
only the website and account differ.

- **GitLab** — a full-featured competitor with strong built-in CI/CD; available as
  a hosted service or self-hosted.
- **Bitbucket** — Atlassian's offering, popular with teams already using Jira.
- **Codeberg** — a community-run, non-profit host built on the open-source Forgejo
  software.
- **Gitea / Forgejo** — lightweight software you can **self-host** on your own
  server for full control.

The skills transfer freely. Learn the workflow on GitHub and you can walk into any
of these tomorrow.

## Creating an account and a repository

Getting started on GitHub is quick. At a high level:

1. Go to **github.com** and sign up for a free account, choosing a username (this
   becomes part of your repo URLs, e.g. `github.com/<you>/<repo>`).
2. Verify your email and, when prompted, enable **two-factor authentication** —
   GitHub now requires it for most accounts, and it's good practice anyway.
3. Click the **+** in the top-right corner and choose **New repository**.
4. Give it a name, an optional description, pick **public** or **private**, and
   optionally let GitHub add a `README`, a `.gitignore`, and a licence.
5. Click **Create repository**. GitHub then shows you the exact commands to connect
   an existing local project or to clone the new one down.

That last screen is your bridge from the website back to the command line — and
connecting the two is exactly what the next lessons cover, starting with
authentication and remotes.

<div class="knowledge-check" data-quiz data-correct-msg="Right — Git is the tool, GitHub is one place to host it." markdown="0">
  <p class="knowledge-check__q">Quick check: which statement best describes the relationship between Git and GitHub?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Git is the version control tool; GitHub is a platform that hosts Git repos and adds collaboration</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">GitHub is a newer version of Git that replaces it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">You must have a GitHub account before Git will work</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Git** is the local version control tool; **GitHub** is a website that hosts Git
  repositories and adds collaboration.
- A **remote** is a complete copy of your repo on a server, kept in sync via **push**
  and **pull**.
- GitHub's headline features are **repos, pull requests, issues, Actions, Pages, and
  Discussions**.
- **GitLab, Bitbucket, Codeberg, and Gitea** are alternatives that speak the same Git.
- Signing up and creating a repo takes minutes — and 2FA is required.

Next up: connecting your local work to a remote with [clone, fetch, pull & push](/learn/git/remotes/).
