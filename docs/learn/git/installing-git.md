---
slug: installing-git
title: Installing & configuring Git
description: How to install Git on Linux, macOS, and Windows, verify the version, and do the essential first-time setup — name, email, default branch, editor, and config scopes.
keywords: install Git, configure Git, git config, git --version, install Git Linux, install Git macOS, install Git Windows, init.defaultBranch, git config global, core.autocrlf
level: beginner
status: full
prereq:
  - what-is-version-control
faq:
  - q: How do I check whether Git is already installed?
    a: Run `git --version` in a terminal. If Git is installed you'll see something like `git version 2.45.2`. If you get a "command not found" error, Git isn't installed yet — follow the section for your operating system below.
  - q: What does git config --global do?
    a: It writes a setting to your per-user config file (`~/.gitconfig`), so it applies to every repository for your user account. The most important globals to set first are `user.name` and `user.email`, which Git stamps onto every commit you make.
  - q: Do I have to set a default branch name?
    a: No, but setting `init.defaultBranch main` is recommended so every new repository starts on a branch called `main`, matching the convention used by GitHub and most projects today. Without it, older Git versions may default to `master`.
  - q: What is core.autocrlf about?
    a: It controls how Git handles line endings between Windows (CRLF) and Unix (LF). On Windows, `git config --global core.autocrlf true` converts to LF on commit and back to CRLF on checkout. On macOS/Linux, `input` keeps LF in the repo while leaving your files alone. It prevents noisy whitespace-only diffs in mixed-OS teams.
---

# Installing & configuring Git

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Install Git with your platform's package manager, then confirm it with
**`git --version`**. Do the one-time setup that stamps your commits and sets sane
defaults: **`user.name`**, **`user.email`**, **`core.editor`**,
**`init.defaultBranch main`**, and a **line-ending** option. Settings live at three
**scopes** — system, global (per-user), and local (per-repo) — with the most
specific one winning. Use **`git config --list --show-origin`** to see exactly
where each value comes from.
</div>

You learned *why* version control matters in
[the previous lesson](/learn/git/what-is-version-control/). Now let's get Git
installed and configured so the rest of the path runs smoothly.

## Installing on Linux

Use your distribution's package manager.

```bash
# Debian / Ubuntu and derivatives
sudo apt update
sudo apt install git

# Fedora / RHEL / CentOS Stream
sudo dnf install git
```

That installs a recent Git from the official repositories — fine for everything in
this path.

## Installing on macOS

The quickest route is the **Xcode Command Line Tools**, which include Git:

```bash
xcode-select --install
```

For a newer Git and easy updates, install via [Homebrew](https://brew.sh):

```bash
brew install git
```

Homebrew's Git usually leads the system one by several versions, which is handy if
you want the latest features.

## Installing on Windows

The standard package is **Git for Windows**, which bundles Git plus *Git Bash*, a
Unix-like terminal that lets you follow along with the commands in this path.
Download it from [git-scm.com](https://git-scm.com/download/win), or install from a
terminal with **winget**:

```bash
winget install --id Git.Git -e
```

During the installer, the defaults are reasonable; the one choice worth noting is
the line-ending option, covered below.

## Verifying the install

Open a fresh terminal and run:

```bash
$ git --version
git version 2.45.2
```

Any 2.x version is fine for this path. If you instead see "command not found,"
close and reopen the terminal (so it picks up the new `PATH`), then try again.

## First-time configuration

Git stamps every commit with a name and email, so set those before your first
commit. The `--global` flag stores them once for your whole user account:

```bash
git config --global user.name "Ada Lovelace"
git config --global user.email "ada@example.com"
```

A few more settings make life smoother:

```bash
# Which editor opens for commit messages, rebases, etc.
git config --global core.editor "nano"        # or "vim", "code --wait"

# New repos start on a branch called main
git config --global init.defaultBranch main

# Make `git pull` rebase by default for a linear history (optional, opinionated)
git config --global pull.rebase true
```

The `pull.rebase` choice is a matter of taste; we explain the trade-off in
[Rebasing](/learn/git/rebasing/). If you're unsure, leave it unset for now.

**Line endings.** Windows uses CRLF and Unix uses LF, which can produce noisy,
whitespace-only diffs in mixed teams. Set `core.autocrlf` to keep things tidy:

```bash
git config --global core.autocrlf true     # Windows
git config --global core.autocrlf input    # macOS / Linux
```

## Config scopes: system, global, local

The same key can be set at three levels. The **most specific scope wins**.

| Scope | Flag | File | Applies to |
|-------|------|------|------------|
| System | `--system` | `/etc/gitconfig` | Every user on the machine |
| Global | `--global` | `~/.gitconfig` | Your user, all your repos |
| Local | `--local` | `<repo>/.git/config` | One repository only |

A common use of **local** scope is a different email for a work repository:

```bash
cd work-project
git config --local user.email "ada@company.com"
```

## Inspecting your configuration

To see every effective setting *and the file it came from*, run:

```bash
$ git config --list --show-origin
file:/etc/gitconfig        core.editor=nano
file:~/.gitconfig          user.name=Ada Lovelace
file:~/.gitconfig          user.email=ada@example.com
file:~/.gitconfig          init.defaultBranch=main
```

To read a single value, name the key: `git config user.email`. This
`--show-origin` view is the fastest way to debug a surprising setting — it tells
you precisely which file to edit.

<div class="knowledge-check" data-quiz data-correct-msg="Correct — local (per-repo) config overrides global and system." markdown="0">
  <p class="knowledge-check__q">Quick check: if the same setting is defined at system, global, and local scope, which value wins?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">System</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Local (the per-repository value)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Whichever was set most recently</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Install Git with **apt/dnf** (Linux), **Xcode CLT or Homebrew** (macOS), or
  **Git for Windows / winget** (Windows).
- Verify with **`git --version`**.
- Set **`user.name`** and **`user.email`** before your first commit, plus
  **`core.editor`**, **`init.defaultBranch main`**, and a line-ending option.
- Config has three **scopes** — system, global, local — and the most specific wins.
- Use **`git config --list --show-origin`** to see where each value lives.

Next up: how Git actually thinks about your project — snapshots, not diffs.
