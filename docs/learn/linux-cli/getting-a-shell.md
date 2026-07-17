---
slug: getting-a-shell
title: Getting a shell
description: How to actually open a command-line shell on whatever computer you already own — the Terminal app on Linux and macOS, WSL on Windows, and SSH into a server or Raspberry Pi — plus safe ways to practice without risk.
keywords: open terminal, WSL, Windows Subsystem for Linux, macOS terminal, SSH, Raspberry Pi, virtual machine, cloud shell, zsh, Git Bash, practice Linux safely
level: beginner
status: full
prereq:
  - the-shell
faq:
  - q: How do I open a terminal on my computer?
    a: "On Linux or macOS, open the app called Terminal — it is already installed. On Windows, install WSL (Windows Subsystem for Linux) and open your Linux distribution, which gives you a real Unix shell. Once the black-and-text window is open and showing a prompt, you are ready to type commands."
  - q: Do I need Linux to follow this path?
    a: "No. macOS is Unix-like, so almost every command in this path works there unchanged. On Windows you install WSL, which runs a genuine Linux distribution alongside Windows. You do not need to wipe your machine or dual-boot to learn the shell."
  - q: Is it safe to practice these commands?
    a: "The beginner commands in this path only list, read, and move around — they will not harm your machine if you follow along. If you want a completely isolated sandbox, use a virtual machine, a container, a spare Raspberry Pi, or a free cloud shell, and experiment there with nothing important at stake."
  - q: How do I get a shell on a Raspberry Pi or a server?
    a: "You reach those over the network with SSH. The machine runs its own operating system — a Pi runs Raspberry Pi OS, a full Linux — and SSH gives you a shell on it from the terminal on your own computer. That is covered later in this path."
gophertrunk_links:
  - title: Getting started & setup
    url: /getting-started-setup.html
    note: once you have a shell, this walks through installing and running GopherTrunk.
---

# Getting a shell

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The [shell](/learn/linux-cli/the-shell/) lives inside a **terminal** — a window that
shows a prompt and waits for commands. On Linux and macOS the **Terminal** app is
already installed; open it and you are ready. On Windows, the best option is
**WSL** (Windows Subsystem for Linux), a real Linux running alongside Windows. To
work on a server or a Raspberry Pi you connect over the network with **SSH**. And
you can practise safely in a virtual machine, a container, or a cloud shell.
</div>

The [previous lesson](/learn/linux-cli/the-shell/) explained what a shell *is*. This
one is purely practical: how to open one on whatever computer you happen to have in
front of you. Every machine below reaches the same kind of Unix shell — only the way
you open the window differs.

## On Linux

You already have everything you need. Open your applications menu and look for the app
called **Terminal** (some desktops call it "Console" or "Terminal Emulator"). Launch
it and you will see a **prompt** — a short line of text ending in `$` waiting for you
to type. That is the shell. You are ready; skip ahead to the next lesson whenever you
like.

## On macOS

macOS also ships with a **Terminal** app — find it in *Applications → Utilities*, or
press `Cmd + Space` and type "Terminal". It opens a shell called **zsh**, which behaves
almost exactly like the Linux shell this path teaches.

macOS is **Unix-like** under the hood, so nearly everything in this path works there
unchanged — the same commands, the same navigation, the same ideas. The one wrinkle to
know about: macOS inherits its command-line tools from **BSD**, while most Linux systems
use the **GNU** versions. A few commands take slightly different flags as a result (for
example, some options to `ls` or `sed` differ). It rarely matters for the basics, and
we will flag it when it does.

## On Windows

Native Windows shells — **PowerShell** and **CMD** — are genuinely useful, but they are
*different* shells with different commands. This path targets a **Unix** shell, so on
Windows you want a way to run one of those instead.

The best option by far is **WSL** (Windows Subsystem for Linux). WSL installs a real
Linux distribution that runs **alongside** Windows — no dual-boot, no wiping anything.
You open it like any other app and get a genuine Linux shell, the same one a Linux user
sees. In a recent Windows, you can usually install it by opening PowerShell and running
a single command to enable WSL, then launching the Linux distribution it installs.

If WSL is not an option for you, two alternatives give you a Unix-style shell:

- **Git Bash** — a lightweight shell that comes with Git for Windows. Good enough for
  learning the basics, though it is not a full Linux.
- **A virtual machine** — software like VirtualBox runs a complete Linux inside a window
  on your Windows desktop. Heavier to set up, but a full, isolated system.

For following this path, **WSL first, Git Bash or a VM as fallbacks**.

## On a server or a Raspberry Pi

Small computers like a [Raspberry Pi](/learn/intro-hardware/raspberry-pi-and-family/) or
a rented server often have **no screen or keyboard** of their own. You reach them over
the network with **SSH** — a tool that opens a shell on the remote machine from the
terminal on your own computer. You type locally; the commands run over there.

A Raspberry Pi is not a toy in this respect: it runs **Raspberry Pi OS**, a full Linux,
so the shell on a Pi is the same shell you have been learning. (If the idea of a whole
computer on one small board is new, see
[what is an SBC](/learn/intro-hardware/what-is-an-sbc/).) SSH itself is a later lesson in
this path — for now, just know that "getting a shell" on a remote box means connecting
to it with SSH rather than opening a local app.

## Practicing safely

If you are nervous about typing commands into the computer you rely on every day, give
yourself a sandbox. Any of these lets you experiment freely and, if something goes
wrong, throw it away and start fresh:

- **A virtual machine** — a whole Linux running in a window, isolated from your real
  system.
- **A container** — a lightweight, disposable Linux environment you can reset in seconds.
- **A spare Raspberry Pi** — cheap, real hardware you can reflash if you break it.
- **A cloud shell** — a free Linux shell in your browser, hosted by a cloud provider;
  nothing to install at all.

That said, the beginner commands ahead only look around, read, and move between folders.
Follow along as written and they **will not harm your machine** — the sandbox is for
peace of mind and for the day you start experimenting on your own.

<div class="knowledge-check" data-quiz data-correct-msg="Right — WSL runs a real Linux distribution alongside Windows, giving you a genuine Unix shell." markdown="0">
  <p class="knowledge-check__q">Quick check: on Windows, the best way to get a real Linux shell for this path is…</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Use PowerShell — it is the same as a Unix shell</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">WSL (Windows Subsystem for Linux)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">You cannot run a Unix shell on Windows at all</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The shell lives inside a **terminal** window that shows a prompt.
- **Linux and macOS** already have a **Terminal** app — open it and you are ready.
- **macOS** runs **zsh** and is Unix-like; watch for the odd **BSD-vs-GNU** flag difference.
- On **Windows**, install **WSL** for a real Linux shell (Git Bash or a VM as fallbacks).
- Reach a **server or Raspberry Pi** over the network with **SSH**.
- Practise safely in a **VM, container, spare Pi, or cloud shell** — the basics won't hurt your machine either way.

Next up: [your first commands](/learn/linux-cli/first-commands/)
