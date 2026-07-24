---
slug: deploying-to-cloud-and-vps
title: Deploying to the cloud & VPS
description: The hosting spectrum — a VPS you manage, managed platforms, and container services — and how to choose based on control, cost, and effort.
keywords: vps, cloud hosting, deploy to cloud, virtual private server, managed platform, paas, container hosting, self hosting, where to deploy
level: intermediate
status: full
prereq:
  - what-is-deployment
faq:
  - q: What is a VPS?
    a: A VPS (virtual private server) is a virtual machine you rent from a hosting provider — a small Linux server in the cloud that you control fully. You install and manage everything yourself, which gives maximum control at the cost of doing the operations work. It's a common, affordable way to run a service like GopherTrunk.
  - q: Should I use a managed platform or a plain server?
    a: A managed platform (PaaS) handles the servers, scaling, and much of the operations for you, so you focus on the app — easier, but with less control and usually higher cost. A plain server or VPS gives full control and lower cost but you run everything yourself. Choose based on how much operations work you want to own versus how much control you need.
---

# Deploying to the cloud & VPS

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Where you run a deployment is a spectrum trading **control** against **effort**: a
**VPS** or your own hardware gives full control but you run everything; a **managed
platform (PaaS)** or **container service** handles the servers for you at higher cost
and less control. Choose by how much operations work you want to own. A service like
GopherTrunk runs happily on a cheap VPS or even a Raspberry Pi.
</div>

You've built and containerized your app. Now — *where* does it run? This lesson maps the
hosting options and how to pick.

## The hosting spectrum

Every option trades how much you control against how much you have to operate:

| Option | You manage | Control | Effort | Good for |
|--------|-----------|---------|--------|----------|
| **Own hardware / SBC** | everything, incl. the machine | total | high | home labs, GopherTrunk on a Pi |
| **VPS** | the OS and up | high | medium | most small services |
| **Container service** | just the container | medium | low-medium | containerized apps that scale |
| **Managed platform (PaaS)** | just the code | low | low | teams who want zero ops |

Moving down the table, the provider takes on more of the work — and charges for it —
while you give up some control and flexibility.

## The VPS: the common middle ground

For most small deployments, a **VPS** — a Linux virtual machine you rent — is the
sweet spot. You get a real server you fully control, for a few dollars a month, and you
deploy onto it with the tools from this module: install your
[systemd service](/learn/deployment/services-and-systemd/) or run your
[container](/learn/deployment/docker-compose/), and you're live. You reach it over
[SSH](/learn/linux-cli/ssh-and-remote/) to set it up and manage it.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 70" role="img" aria-label="A horizontal spectrum from more control and effort on the left to less control and effort on the right, with hardware, VPS, container service, and PaaS marked." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="40" x2="490" y2="40" stroke="currentColor" stroke-width="1.5"/>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
  <circle cx="60" cy="40" r="3"/><text x="60" y="28">own hardware</text><text x="60" y="58">control &#8593;</text>
  <circle cx="200" cy="40" r="3"/><text x="200" y="28">VPS</text>
  <circle cx="340" cy="40" r="3"/><text x="340" y="28">container svc</text>
  <circle cx="470" cy="40" r="3"/><text x="470" y="28">PaaS</text><text x="470" y="58">effort &#8595;</text>
  </g>
</svg>
<figcaption>The hosting spectrum: more control and operations work on the left, more hands-off convenience on the right.</figcaption>
</figure>

## How to choose

Ask three questions:

- **How much control do you need?** Special hardware (like GopherTrunk's USB radio) or
  custom OS setup pushes you toward a VPS or your own machine — a fully managed platform
  usually can't attach a USB dongle.
- **How much operations work do you want?** More convenience costs money and control;
  less costs your time.
- **Does it need to scale?** A single small service is fine on one VPS; something that
  must grow to many instances leans toward container services or orchestration.

## GopherTrunk's angle

Because GopherTrunk talks to physical radio hardware, it typically runs on a machine
that *has* that hardware — a home server, a mini-PC, or a Raspberry Pi with the SDR
plugged in — rather than a hands-off cloud platform. That's a useful reminder: the right
place to deploy is driven by what the app *needs*, not by what's trendiest. Wherever it
runs, the [Linux CLI](/learn/linux-cli/running-gophertrunk-on-linux/) and this module's
tools are how you get it there.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a VPS gives you a full server you control, with medium operations effort." markdown="0">
  <p class="knowledge-check__q">Quick check: what do you get with a VPS?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A fully managed app platform with no server to run</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A rented Linux server you fully control and operate yourself</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A container registry</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Hosting is a spectrum trading **control** for **effort** — own hardware, VPS,
  container service, PaaS.
- A **VPS** is the common middle ground: a full server you control cheaply, managed over
  SSH.
- Choose by **control needed**, **ops effort wanted**, and whether it must **scale**.
- Let the app's needs decide — GopherTrunk runs where its **radio hardware** is.

Next up: keeping a deployment healthy over time — monitoring and updates.
