---
slug: what-is-deployment
title: What is deployment?
description: The journey from source code to a running service, what "it works on my machine" really costs, and the moving parts every deployment has to handle.
keywords: what is deployment, software deployment, deploy application, works on my machine, deployment basics, ship software, production
level: beginner
status: full
faq:
  - q: What does deploying software mean?
    a: Deploying means taking software you've written and getting it running somewhere it can do its job — a server, a cloud host, a small computer at home — reliably and repeatably. It covers building the program, moving it to the target, configuring it for that environment, starting it, and keeping it running.
  - q: Why is deployment considered hard?
    a: Because the machine you develop on differs from where the software runs — different libraries, settings, file paths, and secrets. Bridging that gap so the program behaves identically everywhere, survives crashes and reboots, and can be updated safely is a discipline of its own. The tools in this module exist to make that bridge reliable.
---

# What is deployment?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Deployment** is getting software from your machine to somewhere it runs reliably for
real. It has to solve the same moving parts every time — a **build**, its
**configuration**, a **runtime environment**, **process management**, **networking**,
**secrets**, and **monitoring**. The whole point is to kill "**it works on my
machine**" by making the software behave the same everywhere.
</div>

This is lesson 1 of the deployment path. Writing code is one half of shipping
software; this module is the other half. By the end of it you'll be able to take a
program like GopherTrunk and run it as a real service on a server.

## From source to a running service

On your machine, running a program is easy — you built it, the libraries are there,
the config points at your files. Deployment is doing that *somewhere else*, reliably:

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 90" role="img" aria-label="Flow: source code, build, artifact, configure, run on a server as a service." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
  <rect x="6" y="30" width="70" height="32" rx="4" fill="none" stroke="currentColor"/><text x="41" y="50">source</text>
  <rect x="100" y="30" width="70" height="32" rx="4" fill="none" stroke="currentColor"/><text x="135" y="50">build</text>
  <rect x="194" y="30" width="78" height="32" rx="4" fill="none" stroke="currentColor"/><text x="233" y="50">artifact</text>
  <rect x="296" y="30" width="80" height="32" rx="4" fill="none" stroke="currentColor"/><text x="336" y="50">configure</text>
  <rect x="400" y="30" width="110" height="32" rx="4" fill="none" stroke="currentColor" stroke-width="1.8"/><text x="455" y="50">run as a service</text>
  </g>
  <g stroke="currentColor"><line x1="76" y1="46" x2="100" y2="46"/><line x1="170" y1="46" x2="194" y2="46"/><line x1="272" y1="46" x2="296" y2="46"/><line x1="376" y1="46" x2="400" y2="46"/></g>
</svg>
<figcaption>Deployment is this whole pipeline — and this module is a lesson or two on each box.</figcaption>
</figure>

## The "works on my machine" problem

The classic failure is software that runs perfectly for its author and breaks
everywhere else. The causes are always the same handful of differences:

- A **library** or tool that's installed on your laptop but not the server.
- A **setting** or file path hard-coded to your environment.
- A **secret** (an API key, a password) that lived in your head or your shell.
- A different **operating system** or version.

Every tool in this module exists to erase one of these differences —
[containers](/learn/deployment/what-is-a-container/) bundle the libraries,
[configuration](/learn/deployment/environments-and-config/) externalizes the settings,
[secret management](/learn/deployment/secrets-and-configuration/) handles the keys.

## The moving parts of any deployment

However you deploy, the same responsibilities show up:

| Concern | The question it answers |
|---------|-------------------------|
| Build & artifact | What exactly are we shipping? |
| Configuration | How does it behave in *this* environment? |
| Runtime environment | What does it need installed to run? |
| Process management | Who starts it, and restarts it if it crashes? |
| Networking | How do users and other services reach it? |
| Secrets | Where do passwords and keys come from safely? |
| Monitoring | How do we know it's healthy? |

Each has a lesson (or a few) in this module. GopherTrunk touches all of them: it's a
long-running daemon that talks to USB radio hardware, serves a web API, records to
disk, and must survive reboots — a realistic, complete example we'll use throughout,
finishing with a full [end-to-end deploy](/learn/deployment/deploying-gophertrunk/).

## Why get good at this

Software that only runs on your laptop helps no one. The ability to reliably ship and
operate what you build is what turns a project into a *product* — and it's a skill that
transfers to every language and stack. Pair it with the
[Linux command line](/learn/linux-cli/) and [Git](/learn/git/) modules and you have the
full toolkit.

<div class="knowledge-check" data-quiz data-correct-msg="Right — deployment is about running software reliably somewhere other than your own machine." markdown="0">
  <p class="knowledge-check__q">Quick check: what problem is most deployment tooling designed to solve?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Writing the code faster</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Making software behave the same everywhere, not just on the author's machine</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Reducing the number of features</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Deployment** is getting software running reliably somewhere other than your machine.
- The enemy is "**it works on my machine**" — caused by library, config, secret, and OS
  differences.
- Every deployment handles the same **moving parts**: build, config, runtime, process
  management, networking, secrets, monitoring.
- GopherTrunk is the worked example we build toward deploying end to end.

Next up: environments and configuration — running one build in many places.
