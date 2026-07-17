---
slug: glossary
title: Glossary of Linux & command-line terms
description: Plain-language definitions of Linux and shell terms — kernel, distribution, shell, path, permission, process, pipe, PATH, script, cron, SSH, systemd, and more — each cross-linked to the lesson that explains it.
keywords: linux glossary, command line terms, bash glossary, shell, kernel, permissions, process, pipe, PATH, ssh, systemd, cron, sudo
level: beginner
status: full
lesson_standalone: true
---

# Glossary of Linux & command-line terms

Every term used across the [Linux & the Command Line](/learn/linux-cli/) path,
defined in plain language and linked to the lesson where it's explained in full.
Skim it as a refresher, or use your browser's find (Ctrl/Cmd-F) to jump to a word.
Terms are grouped by theme, roughly in the order the path introduces them.

## The shell & terminal

**Linux** — A free, open-source operating system built on a Unix-like design; it
runs most servers, Android, and single-board computers. See
[What is Linux?](/learn/linux-cli/what-is-linux/)

**Kernel** — The core of the OS that manages hardware and runs programs; "Linux" is
technically just the kernel. See [What is Linux?](/learn/linux-cli/what-is-linux/)

**Distribution (distro)** — A packaged, ready-to-use bundle of the Linux kernel plus
tools and software — Ubuntu, Debian, Fedora, Raspberry Pi OS. See
[What is Linux?](/learn/linux-cli/what-is-linux/)

**Shell** — The program that reads the commands you type and runs them (bash, zsh).
See [The shell explained](/learn/linux-cli/the-shell/)

**Terminal** — The window (or app) that a shell runs inside; the terminal is the
screen, the shell is the interpreter. See [The shell explained](/learn/linux-cli/the-shell/)

**Command, option, argument** — The three parts of what you type: the program name,
its flags (like `-l`), and what it acts on. See [The shell explained](/learn/linux-cli/the-shell/)

**man page** — The built-in manual for a command, opened with `man <command>`. See
[Your first commands](/learn/linux-cli/first-commands/)

**WSL (Windows Subsystem for Linux)** — Microsoft's way to run a real Linux distro
alongside Windows, the best way to get a Linux shell there. See
[Getting a shell](/learn/linux-cli/getting-a-shell/)

## The filesystem

**Root directory (/)** — The single top of the Linux filesystem tree; everything
hangs off it (no drive letters). See [The filesystem hierarchy](/learn/linux-cli/filesystem-layout/)

**Absolute vs. relative path** — An absolute path starts from `/`; a relative path
starts from where you currently are. See [The filesystem hierarchy](/learn/linux-cli/filesystem-layout/)

**Home directory (~)** — Your personal folder for files and settings, shorthand `~`.
See [The filesystem hierarchy](/learn/linux-cli/filesystem-layout/)

**Working directory** — The folder you are "in" right now, shown by `pwd`. See
[Your first commands](/learn/linux-cli/first-commands/)

**Dotfile** — A file whose name starts with `.`, hidden by default and often used
for configuration. See [Navigating & listing files](/learn/linux-cli/navigating-and-listing/)

**/etc, /var, /home** — Conventional folders: `/etc` for system config, `/var` for
logs and changing data, `/home` for user files. See
[The filesystem hierarchy](/learn/linux-cli/filesystem-layout/)

## Users, permissions & processes

**root (superuser)** — The all-powerful account (UID 0) that bypasses permissions;
used sparingly. See [Users, groups & root](/learn/linux-cli/users-and-groups/)

**User & group** — An account that owns files and runs programs, and a collection of
users that can share access. See [Users, groups & root](/learn/linux-cli/users-and-groups/)

**sudo** — Runs a single command with root privileges, safer than logging in as
root. See [sudo & becoming root safely](/learn/linux-cli/sudo-and-root/)

**Permissions (rwx)** — Read, write, and execute rights granted to the owner, group,
and everyone else. See [File permissions](/learn/linux-cli/permissions/)

**chmod / chown** — Commands that change a file's permission bits (`chmod`) and its
owner (`chown`). See [File permissions](/learn/linux-cli/permissions/)

**Process (PID)** — A running program, identified by a numeric process ID. See
[Processes & jobs](/learn/linux-cli/processes/)

**Signal / kill** — A message sent to a process to pause or stop it; `kill` sends one
(SIGKILL / `-9` forces). See [Processes & jobs](/learn/linux-cli/processes/)

**Package manager** — The tool (apt, dnf, pacman) that installs, updates, and removes
software from trusted repositories. See
[Installing software with a package manager](/learn/linux-cli/package-management/)

**Repository** — A trusted, signed source of software packages the package manager
installs from. See [Installing software with a package manager](/learn/linux-cli/package-management/)

## Command-line power

**stdin / stdout / stderr** — A command's standard input, standard output, and
standard error streams. See [Pipes & redirection](/learn/linux-cli/pipes-and-redirection/)

**Pipe ( | )** — Sends one command's output straight into the next command's input.
See [Pipes & redirection](/learn/linux-cli/pipes-and-redirection/)

**Redirection ( > >> < )** — Sends output to a file (`>` overwrite, `>>` append) or
reads input from one (`<`). See [Pipes & redirection](/learn/linux-cli/pipes-and-redirection/)

**grep** — Searches for text inside files or a stream, line by line. See
[Text-processing tools](/learn/linux-cli/text-tools/)

**Glob / wildcard** — A pattern like `*` or `?` the shell expands into matching
filenames before a command runs. See [Wildcards, globbing & expansion](/learn/linux-cli/wildcards-and-expansion/)

**Environment variable** — A named value the shell and programs read, such as `HOME`
or `EDITOR`. See [Environment variables & PATH](/learn/linux-cli/environment-variables/)

**PATH** — The list of directories the shell searches to find a command you type. See
[Environment variables & PATH](/learn/linux-cli/environment-variables/)

**Tab completion** — Pressing Tab to auto-complete a command, filename, or path. See
[History, tab completion & shortcuts](/learn/linux-cli/history-and-shortcuts/)

## Scripting

**Shell script** — A file of commands run as one program. See
[Your first shell script](/learn/linux-cli/first-shell-script/)

**Shebang (#!)** — The first line of a script that names the interpreter to run it
with, e.g. `#!/bin/bash`. See [Your first shell script](/learn/linux-cli/first-shell-script/)

**Argument ($1, $@)** — Values passed to a script on the command line, read as `$1`,
`$2`, and so on. See [Variables, arguments & input](/learn/linux-cli/variables-and-input/)

**Exit code (status)** — The number a command returns — `0` for success, non-zero for
failure — readable as `$?`. See [Functions, exit codes & error handling](/learn/linux-cli/functions-and-exit-codes/)

**Function** — A named, reusable block of commands inside a script. See
[Functions, exit codes & error handling](/learn/linux-cli/functions-and-exit-codes/)

**cron / crontab** — The scheduler that runs commands on a timetable, edited with
`crontab -e`. See [Scheduling with cron & timers](/learn/linux-cli/scheduling-with-cron/)

## Servers, services & remote

**SSH (secure shell)** — A secure way to log into and run commands on another machine
over the network. See [SSH & working remotely](/learn/linux-cli/ssh-and-remote/)

**SSH key** — A key pair used to log in over SSH without a password, more secure and
convenient. See [SSH & working remotely](/learn/linux-cli/ssh-and-remote/)

**scp / rsync** — Commands that copy files to and from a remote machine over SSH. See
[SSH & working remotely](/learn/linux-cli/ssh-and-remote/)

**Daemon / service** — A background program that runs without a terminal, managed by
a service manager. See [Services & systemd](/learn/linux-cli/services-and-systemd/)

**systemd / systemctl** — The init system that runs services, controlled with
`systemctl` (start, stop, enable at boot). See [Services & systemd](/learn/linux-cli/services-and-systemd/)

**journalctl** — Reads the systemd journal — a service's logs — with `-u` to filter
and `-f` to follow. See [Services & systemd](/learn/linux-cli/services-and-systemd/)

**df / du** — Report free disk space per filesystem (`df`) and how much space a
directory uses (`du`). See [Monitoring, disk & logs](/learn/linux-cli/monitoring-and-logs/)
