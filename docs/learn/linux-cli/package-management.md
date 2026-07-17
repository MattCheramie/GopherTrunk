---
slug: package-management
title: Installing software with a package manager
description: How Linux installs software — from trusted, signed repositories with a single command instead of hunting for downloads. Meet the main package-manager families (apt, dnf, pacman), the core install/remove/update operations, and where Snap and Flatpak fit in.
keywords: package manager, apt, dnf, pacman, install software linux, repository, dependencies, apt update, apt install, snap, flatpak, update packages
level: beginner
status: full
prereq:
  - sudo-and-root
faq:
  - q: "What is a package manager on Linux?"
    a: A package manager is the tool that installs, updates, and removes software on Linux. Instead of downloading installers yourself, you ask the package manager for a program by name and it fetches it — plus everything it depends on — from trusted, signed repositories. Common ones are apt, dnf, and pacman.
  - q: How do I install software on Linux?
    a: Use your distribution's package manager. On Debian, Ubuntu, or Raspberry Pi OS you run sudo apt update to refresh the list of available packages, then sudo apt install followed by the package name. Fedora uses dnf and Arch uses pacman, but the idea is the same — name the software and let the tool fetch it.
  - q: "What is the difference between apt, dnf, and pacman?"
    a: They are the package managers for different families of Linux distributions — apt for Debian, Ubuntu, and Raspberry Pi OS; dnf for Fedora and RHEL; pacman for Arch. They do the same job of installing and updating software from repositories, just with different commands. You use whichever one ships with your distribution.
  - q: What is a repository?
    a: A repository is an online collection of software packages that your package manager is configured to trust. Packages are cryptographically signed, so the manager can check they came from that source and were not tampered with. Adding a third-party repository is possible, but it means extending your trust to whoever runs it.
---

# Installing software with a package manager

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
On Linux you do not hunt the web for installers. A **package manager** fetches
software for you from trusted **repositories** — signed collections of programs —
and works out the **dependencies** each one needs. You name the software, run one
command (with [sudo](/learn/linux-cli/sudo-and-root/)), and it is installed, ready to
update or remove cleanly later. This is safer and simpler than downloading random
files, and it is how you will set up almost everything on a Linux machine.
</div>

Coming from Windows or macOS, installing software probably means visiting a website,
downloading an installer, and clicking through it. Linux flips that around: the system
already knows where to get trusted software, so you just ask for it by name. Once this
clicks, setting up a new machine becomes a list of one-line commands.

## Why a package manager

A **package manager** exists so you never have to trust — or manage — software by hand.
It gives you three things that a downloaded installer cannot:

- **Trusted, signed sources.** Packages come from repositories the system already trusts,
  and each one is cryptographically signed. The manager verifies that signature before
  installing, so you know the software is genuine and unmodified.
- **Automatic dependency resolution.** Most programs rely on other libraries and tools —
  their **dependencies**. The package manager works out the full list, fetches it, and
  installs everything in the right order. No chasing "missing library" errors.
- **Clean updates and removal.** Because the manager tracks exactly what it installed, it
  can update every program with one command and remove one completely, without leaving
  stray files behind.

That combination — trusted sources, dependencies handled, tidy updates — is why
installing from a package manager is both safer and simpler than running an installer you
found somewhere.

## The families

Different Linux distributions use different package managers, but they all do the same
job. The three you are most likely to meet:

- **apt** — used by **Debian, Ubuntu, and Raspberry Pi OS** (and their many relatives).
- **dnf** — used by **Fedora and RHEL** (Red Hat Enterprise Linux) and their relatives.
- **pacman** — used by **Arch** Linux and its relatives.

The idea is identical across all three: name the software, and the tool fetches it from a
repository with its dependencies. Only the commands differ. The first useful thing to know
about any Linux machine is **which package manager your distribution uses** — that tells
you which commands below apply. The examples here use apt, because it is the most common
on the beginner-friendly distributions.

## Core operations (apt examples)

Almost everything you do with apt is one of a handful of operations. Each changes the
system, so each needs [sudo](/learn/linux-cli/sudo-and-root/):

- **`sudo apt update`** — refresh the local list of what is available in the repositories.
  Run this first; it does not install anything, it just updates the catalogue.
- **`sudo apt install <package>`** — install a program and its dependencies.
- **`sudo apt remove <package>`** — remove a program you no longer want.
- **`sudo apt upgrade`** — update every installed package to its latest version.
- **`apt search <term>`** — find packages by keyword (no sudo needed; it only reads).

A typical first session looks like this:

```
$ sudo apt update
$ apt search sox
$ sudo apt install sox
$ sox --version
```

That refreshes the catalogue, searches for the audio tool `sox`, installs it, then
confirms it is there. The `update` step matters: without a fresh catalogue, `install` may
try to fetch a version the repository no longer has.

## Repositories & trust

Every package your manager offers comes from a **repository** it is configured to trust.
A fresh install already points at your distribution's official repositories, which is why
`apt install` "just works" for common software — it is all curated and signed.

You can add **third-party repositories** to reach software the official ones do not carry.
That is a normal thing to do, but it is a **trust decision**: you are telling your system
to accept and run signed software from someone new. Add only repositories you have reason
to trust, the same way you would think twice before running an installer from an unknown
website.

## Snap, Flatpak & others

Alongside the classic managers, newer **cross-distribution** formats have appeared —
mainly **Snap** and **Flatpak**. Instead of one package format per distribution, these
bundle an application together with its dependencies so the same package runs on Debian,
Fedora, Arch, and the rest.

You will meet them when a project ships only as a Snap or Flatpak, or when you want a
newer version of a desktop application than your distribution's repositories carry. They
trade a little extra disk space and startup time for that portability. For most
command-line and server work — including setting up GopherTrunk — the traditional package
manager is still what you will reach for first.

## In practice

Most of the time you will use a package manager to install the **build tools and
dependencies** other software needs. Compilers, libraries, and command-line utilities all
come from your repositories with a single `install` command — so when you set up something
like [GopherTrunk on Linux](/learn/linux-cli/running-gophertrunk-on-linux/), the "install
these prerequisites" step is just a short list of package names handed to apt, dnf, or
pacman.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the manager verifies a signed source and pulls in everything the program needs." markdown="0">
  <p class="knowledge-check__q">Quick check: why install software with a package manager instead of downloading a binary from a website?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It is the only way to run any program on Linux</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It comes from a trusted source and handles dependencies and updates for you</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It avoids ever needing sudo or root</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **package manager** installs software from trusted, signed **repositories** — no
  hunting for downloads.
- It resolves **dependencies** automatically and lets you update and remove software
  cleanly.
- Know your family: **apt** (Debian, Ubuntu, Raspberry Pi OS), **dnf** (Fedora/RHEL),
  **pacman** (Arch) — same idea, different commands.
- The core apt moves: `sudo apt update`, then `install`, `remove`, or `upgrade`; search
  with `apt search`.
- Adding a **third-party repository** is possible but is a trust decision.
- **Snap** and **Flatpak** are cross-distribution formats you will meet for some apps.

Next up: Module 4 unlocks the shell's real power — [pipes & redirection](/learn/linux-cli/pipes-and-redirection/)
