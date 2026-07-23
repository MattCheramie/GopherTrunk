---
slug: what-is-a-container
title: What is a container?
description: Containers versus virtual machines, the isolation the kernel provides, and why shipping the whole environment solves the works-on-my-machine problem.
keywords: what is a container, containers vs vms, docker container, containerization, kernel namespaces, cgroups, isolation, oci image
level: beginner
status: full
prereq:
  - what-is-deployment
faq:
  - q: What is the difference between a container and a virtual machine?
    a: A virtual machine emulates a whole computer, running its own full operating system on top of a hypervisor — heavy but strongly isolated. A container shares the host's operating-system kernel and just isolates the application's files, processes, and network, making it far lighter and faster to start. For deploying most applications, containers hit the sweet spot.
  - q: Do containers include an operating system?
    a: A container image includes the userland files an application needs — libraries, a minimal base like Debian slim, your binary — but not a kernel. It borrows the host's kernel at runtime. That's why images are much smaller than VM disk images and why a container starts in a fraction of a second.
---

# What is a container?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **container** packages an application with everything it needs to run — libraries,
files, dependencies — and runs it **isolated** on a shared host **kernel**. Unlike a
**virtual machine**, it doesn't carry a whole operating system, so it's **light and
fast**. Containers solve "works on my machine" by shipping the *whole environment*, not
just the code.
</div>

Unit 1 named the "works on my machine" problem. Containers are the most popular
solution, and this lesson explains what they actually are.

## Ship the environment, not just the code

The insight behind containers is simple: instead of shipping your program and *hoping*
the target machine has the right libraries and settings, you ship your program
**together with** its libraries, dependencies, and environment, bundled into one unit.
That bundle — a **container** — runs the same on your laptop, a colleague's machine, and
a cloud server, because it *brings its whole world with it*.

## Containers vs virtual machines

Both isolate software, but at very different weights:

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 170" role="img" aria-label="Left: virtual machines each with a full guest OS on a hypervisor. Right: containers sharing one host OS kernel via a container runtime." xmlns="http://www.w3.org/2000/svg">
  <text x="130" y="14" text-anchor="middle" font-size="10" fill="currentColor">virtual machines</text>
  <g font-size="8" fill="currentColor" text-anchor="middle">
  <rect x="20" y="22" width="100" height="26" fill="none" stroke="currentColor"/><text x="70" y="39">App + Guest OS</text>
  <rect x="140" y="22" width="100" height="26" fill="none" stroke="currentColor"/><text x="190" y="39">App + Guest OS</text>
  <rect x="20" y="52" width="220" height="18" fill="none" stroke="currentColor"/><text x="130" y="65">Hypervisor</text>
  <rect x="20" y="74" width="220" height="16" fill="none" stroke="currentColor"/><text x="130" y="86">Host OS + Hardware</text>
  </g>
  <text x="390" y="14" text-anchor="middle" font-size="10" fill="currentColor">containers</text>
  <g font-size="8" fill="currentColor" text-anchor="middle">
  <rect x="285" y="22" width="66" height="20" fill="none" stroke="currentColor"/><text x="318" y="35">App</text>
  <rect x="357" y="22" width="66" height="20" fill="none" stroke="currentColor"/><text x="390" y="35">App</text>
  <rect x="429" y="22" width="66" height="20" fill="none" stroke="currentColor"/><text x="462" y="35">App</text>
  <rect x="285" y="46" width="210" height="18" fill="none" stroke="currentColor"/><text x="390" y="59">Container runtime</text>
  <rect x="285" y="68" width="210" height="22" fill="none" stroke="currentColor" stroke-width="1.6"/><text x="390" y="82">Shared Host OS kernel + Hardware</text>
  </g>
</svg>
<figcaption>VMs each carry a full guest OS; containers share the host kernel — so containers are far lighter and start almost instantly.</figcaption>
</figure>

| | Virtual machine | Container |
|--|-----------------|-----------|
| Contains | a full guest OS | just app + its dependencies |
| Isolation | very strong (own kernel) | strong (shared kernel) |
| Size | gigabytes | megabytes |
| Start time | tens of seconds | fraction of a second |

## How the isolation works

A container shares the host's **kernel** but the kernel gives each container its own
private view: **namespaces** make it see its own processes, filesystem, and network as
if it were alone, and **control groups (cgroups)** cap how much CPU and memory it can
use. So many containers safely coexist on one host without tripping over each other —
lighter than VMs but still well isolated.

## The vocabulary

Two words you'll use constantly:

- An **image** is the packaged, on-disk bundle — the read-only template (built from a
  [Dockerfile](/learn/deployment/writing-a-dockerfile/), shared via a
  [registry](/learn/deployment/images-and-registries/)).
- A **container** is a *running instance* of an image — like a program is a running
  instance of an executable.

**Docker** is the best-known tool for building images and running containers, and the
one we'll use. GopherTrunk ships a Dockerfile so you can run the whole scanner as a
container — the subject of the next lesson.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a container shares the host kernel, so it's far lighter than a VM." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the main difference between a container and a virtual machine?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A container runs faster because it uses less code</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A container shares the host OS kernel instead of carrying its own full OS</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A container cannot be isolated from other programs</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **container** bundles an app with its dependencies and runs it **isolated** on a
  shared host **kernel**.
- Versus a **VM**, it carries no guest OS — so it's **megabytes**, not gigabytes, and
  starts almost instantly.
- The kernel isolates each container with **namespaces** and **cgroups**.
- An **image** is the template; a **container** is a running instance of it. Docker is
  the common tool.

Next up: writing the recipe for an image — the Dockerfile.
