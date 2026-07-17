---
slug: what-is-linux
title: What is Linux?
description: Linux is the free, open-source operating system that quietly runs most servers, Android phones, and single-board computers. Learn what an OS does, the difference between the kernel and a full system, and how distributions package it up.
keywords: Linux, operating system, kernel, distribution, open source, Ubuntu, Debian, Linux servers, GNU, Raspberry Pi OS
level: beginner
status: full
faq:
  - q: "What is Linux, simply put?"
    a: "Linux is a free, open-source **operating system** — the software that manages a computer's hardware and runs your programs, the same job Windows and macOS do. It powers most of the world's servers, every Android phone, and countless single-board computers like the Raspberry Pi."
  - q: "Is Linux the same as the kernel?"
    a: "Strictly, **Linux** is just the **kernel** — the core that talks to the hardware. A usable system adds the GNU userland, a shell, and everyday tools on top, which is why the whole thing is often called GNU/Linux. In casual use, people say Linux to mean the entire operating system."
  - q: "What is a Linux distribution?"
    a: "A **distribution** (or distro) is a ready-to-use bundle of the Linux kernel plus a chosen set of software, tools, and default settings. Ubuntu, Debian, Fedora, Arch, and Raspberry Pi OS are all distributions of the same kernel, packaged differently for different users."
  - q: "Do I need to learn Linux to use GopherTrunk?"
    a: "No, but it helps. GopherTrunk runs well on Linux — including a Raspberry Pi — and most of its documentation assumes a Linux command line. Learning the basics of the shell, which this path teaches, makes running and troubleshooting it far easier."
---
# What is Linux?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Linux is an operating system** — the software that manages a computer's hardware and runs its programs, the same job Windows and macOS do. Strictly it's the **kernel**, the core that talks to the hardware; a full, usable system wraps that in tools and a shell, and a **distribution** packages the whole lot up. It's **open source** and free, and it quietly runs most servers, Android phones, and [single-board computers](/learn/intro-hardware/what-is-an-sbc/).
</div>

If you've decided to learn the command line, Linux is where that journey almost always begins. You may never have chosen Linux on purpose, yet you use it every day — it's inside your phone, behind the websites you visit, and running on the small computers hobbyists love. This lesson explains what Linux actually is, so the rest of the path has a firm footing.

## An operating system, briefly

Every computer needs an **operating system** (OS). It's the layer of software that sits between the bare hardware and the programs you actually want to run. When you open an app, the OS finds it on storage, loads it into memory, and gives it a slice of the processor. When that app wants to save a file, read the network, or draw on the screen, it asks the OS, which manages the hardware on its behalf.

Without an OS, every program would have to know how to drive every piece of hardware itself — an impossible job. The OS handles all of that once, so programs can share the machine peacefully. Windows is an operating system. So is macOS. And so is **Linux** — it does the same fundamental job, just with a different design and a very different history.

## Kernel vs. the whole system

Here's a subtlety that trips up newcomers: the word *Linux* technically refers only to the **kernel**. The kernel is the innermost core of the operating system — the part that talks directly to the hardware, manages memory, schedules programs, and controls access to devices. It's essential, but on its own it's not something you can sit down and use.

A system you can actually work with adds a lot more on top: the **GNU userland** (a large collection of standard commands and libraries), a **shell** (the program that reads your typed commands), a text editor, a way to install software, and dozens of everyday tools. Because so much of that surrounding software comes from the GNU project, many people call the complete system **GNU/Linux** to give both halves their due. In casual conversation, though, almost everyone just says "Linux" to mean the whole thing — kernel plus everything around it.

## Distributions

If the kernel and the GNU tools are ingredients, a **distribution** — or **distro** — is the finished meal. A distribution takes the same Linux kernel, bundles it with a selection of software, sets sensible defaults, and hands you something you can install and boot. The kernel underneath is largely the same; what differs is the packaging, the default tools, and the philosophy.

A few you'll hear about constantly:

- **Ubuntu** — friendly and popular, a common first Linux for desktops and servers.
- **Debian** — rock-stable and community-run; Ubuntu is built on top of it.
- **Fedora** — modern and fast-moving, favoured by many developers.
- **Arch** — minimal and do-it-yourself, for people who want to build up from nothing.
- **Raspberry Pi OS** — Debian tuned for the [Raspberry Pi](/learn/intro-hardware/raspberry-pi-and-family/).

A distro decides which programs are installed by default, how you install more, how updates work, and how the desktop (if any) looks. For a beginner, the choice matters less than it seems: pick a mainstream one like Ubuntu, Debian, or Raspberry Pi OS, and the skills you learn carry over to all the others. The command line, in particular, is nearly identical everywhere.

## Where Linux runs

Linux is easy to overlook because it rarely shows its face, but it is genuinely everywhere:

- **Servers and the cloud** — the vast majority of web servers and cloud infrastructure run Linux. When you load a website, you're almost certainly talking to a Linux machine.
- **Android phones** — Android is built on the Linux kernel, so billions of pockets carry it.
- **Embedded devices** — routers, smart TVs, cars, and appliances often run a small Linux inside.
- **Single-board computers** — the [Raspberry Pi and its family](/learn/intro-hardware/raspberry-pi-and-family/) run Linux, which is why they're so capable.
- **Supercomputers** — essentially all of the world's fastest machines run Linux.

This ubiquity is exactly why developers and hobbyists learn it. If you want to run a [home server](/learn/intro-hardware/home-servers/), rent a machine in the cloud, or build a project on a small board, you'll be doing it on Linux. Learn it once and the knowledge follows you across all of those worlds.

## Unix heritage & cousins

Linux didn't appear from nowhere. It was written to work like **Unix**, a family of operating systems from the 1970s whose design ideas — small tools that do one job well, a shared file layout, a powerful command line — still shape Linux today. Linux is a fresh, open-source implementation of those ideas rather than a direct descendant, but the family resemblance is strong.

That heritage is why skills transfer so well. **macOS** is also Unix-like under the hood, so much of what you learn here works in a Mac terminal too. And if you're on **Windows**, you don't miss out: the Windows Subsystem for Linux (WSL) runs a real Linux environment right alongside Windows, and we'll cover it when we look at [getting a shell](/learn/linux-cli/getting-a-shell/). Whatever computer you have, there's a way in.

## You talk to it through a shell

Linux has desktops with windows and a mouse, but its real power — and the reason this path exists — is the **command line**. You drive Linux by typing commands into a program called a **shell**, which reads what you type, runs it, and shows the result. It looks something like this:

```console
$ uname -o
GNU/Linux
$ echo "Hello from Linux"
Hello from Linux
```

That prompt is where the rest of this path lives. We'll meet the shell properly in [the shell](/learn/linux-cli/the-shell/) and build up from there, command by command. And it pays off directly here: **GopherTrunk** runs great on Linux — including on a Raspberry Pi tucked next to your antenna — and knowing the command line is what makes [running GopherTrunk on Linux](/learn/linux-cli/running-gophertrunk-on-linux/) straightforward. Learn to talk to Linux, and you can run it anywhere.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a distribution bundles the same kernel with a chosen set of software and tools." markdown="0">
  <p class="knowledge-check__q">Quick check: What is a Linux "distribution"?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A different, incompatible operating system unrelated to Linux</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A packaged bundle of the Linux kernel plus software and tools (Ubuntu, Fedora, …)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The core part of Linux that talks directly to the hardware</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Linux is an operating system** — it manages the hardware and runs your programs, like Windows or macOS.
- **Strictly, "Linux" is the kernel** — the core that talks to hardware; a full system adds the GNU userland, a shell, and tools, hence "GNU/Linux".
- **A distribution packages it all up** — Ubuntu, Debian, Fedora, Arch, and Raspberry Pi OS share the same kernel, bundled differently.
- **It runs almost everywhere** — most servers, the cloud, Android, embedded gear, single-board computers, and supercomputers.
- **It shares Unix roots** — so skills carry to macOS, and Windows can run Linux through WSL.
- **You drive it through a shell** — the command line, which is exactly what this path teaches.

Next up: [why the command line?](/learn/linux-cli/why-the-command-line/)
