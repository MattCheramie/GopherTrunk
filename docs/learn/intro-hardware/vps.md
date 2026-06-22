---
slug: vps
title: Virtual private servers (VPS)
description: A VPS is your own root-access slice of a server, carved out by virtualization. Learn what it runs, how hypervisors split a machine, and the sysadmin duty it brings.
keywords: VPS, virtual private server, virtualization, hypervisor, root access, cloud server, DigitalOcean Linode Hetzner, self-managed server, sysadmin responsibility
level: intermediate
status: full
faq:
  - q: "What is a VPS?"
    a: "A **virtual private server** is an isolated virtual machine running on shared physical hardware, with its own operating system and full **root access**. Virtualization software splits one powerful physical server into several independent virtual ones, each behaving like a standalone computer. You get the freedom of your own machine — install anything, run anything — without paying for a whole physical box."
  - q: "How is a VPS different from shared hosting?"
    a: "Shared hosting hands you a managed slot on a server you can't control. A VPS hands you **root** on your own virtual machine — you choose the operating system, install any software, and run long-lived processes. The catch is that everything the shared host used to do for you (security, updates, backups, configuration) is now *your* job. You trade convenience for control."
  - q: "Can I run a GopherTrunk recorder on a VPS?"
    a: "You can run the *software* — the dashboard, the processing, the storage — but not the radio. A cloud VPS is a virtual machine in a data center with **no USB ports**, so you can't plug in an SDR dongle. The signal has to enter somewhere physical. This is a clean illustration that some workloads need hardware physically attached, which points toward a **[home server](/learn/intro-hardware/home-servers/)** for the radio end."
---
# Virtual private servers (VPS)

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**A VPS is your own virtual machine** carved out of shared physical hardware by virtualization, with full root access. **You can run anything** because you control the operating system. **The responsibility is yours too** — security, updates, and backups are now your job. **Performance can vary** because the underlying hardware is still shared with other tenants.
</div>

When shared hosting runs out of room, the natural next step is a VPS — a machine that *behaves* like it's all yours, even though it's a slice of a bigger one. It's the workhorse of the modern internet: most web apps, APIs, and small services you use are running on something like it. This lesson explains how virtualization carves one physical machine into many, what a VPS lets you do, and the new duty that arrives the moment you get root.

## What a VPS is and how virtualization works

A **virtual private server** is an isolated, self-contained virtual computer running on a physical server it shares with others. To you it looks and behaves like a dedicated machine: it boots its own operating system, has its own memory and disk, and gives you **root access** to do whatever you like.

The trick that makes this possible is **virtualization**. A layer of software called a **hypervisor** runs on the physical server and divides its resources — CPU cores, memory, storage — into several walled-off virtual machines. Each VPS thinks it has its own hardware; the hypervisor keeps them isolated so one tenant can't see or disturb another. (A lighter-weight cousin, **containers**, share the host's kernel instead of emulating whole machines, but the idea is the same: slice one box into many isolated environments.)

The result is the best of both worlds for the price: the independence of your own server without the cost of buying and housing a physical one.

## Who provides them and what they're for

Plenty of providers rent VPS instances by the hour or month — **DigitalOcean**, **Linode**, **Hetzner**, and the bigger clouds like **AWS EC2** among them. At a generic level they all offer the same thing: pick a size, pick an operating system, and a virtual machine appears in minutes.

What people run on them covers most of the internet's day-to-day work:

- **Web applications and APIs** — the back end behind a site or a mobile app.
- **Bots and small services** — chat bots, schedulers, scrapers, webhooks.
- **Databases** — a PostgreSQL or MySQL instance for your own apps.
- **Dashboards and internal tools** — anything that needs to run continuously and be reachable.

If a job needs to *stay running* and *be online*, and doesn't need physical hardware attached, a VPS is usually the right home for it.

## The languages and stacks it runs

Here the answer is simple: **anything**. Because you control the operating system, the question isn't "what does the host allow?" but "what do you want to install?" Python, Node.js, Go, Rust, Java, Ruby, PHP, a database, a message queue, your own compiled binaries — if it runs on Linux (or Windows, if you choose that), it runs on a VPS. This is the whole point of getting root: the environment is yours to shape.

## Strengths, drawbacks, and the GopherTrunk catch

The freedom of a VPS comes bundled with the duties of running a server. The balance looks like this:

| Strengths | Drawbacks |
|-----------|-----------|
| **Full control** — root access, any OS, any software | **You're the sysadmin** — security, patching, backups are on you |
| **Affordable** — far cheaper than a physical server | **Shared hardware** — performance can vary with neighbors |
| **Scalable** — resize or add instances on demand | **Easy to misconfigure** — an open port can mean a breach |
| **Snapshots** — save and restore the whole machine | **No physical access** — no USB, no plugging in hardware |

That last drawback is worth dwelling on. You *could* run the **GopherTrunk** dashboard, processing, and storage on a VPS — it's just software. But a cloud VPS is a virtual machine in a data center with **no USB ports**, so there's nowhere to plug in an SDR dongle. The radio signal has to enter the system through physical hardware, and a VPS has none. It's a perfect teaching point: control over the *software* environment doesn't give you the *physical* I/O some jobs need. For the radio end of GopherTrunk, you want a machine you can physically touch — which is where home servers come in.

## With control comes responsibility

The single biggest difference from shared hosting isn't technical, it's a shift in who's accountable. On shared hosting the provider keeps the machine patched, watched, and backed up. On a VPS, *you* do. An unpatched service, a weak password, or an exposed database is now your problem to prevent and your problem to clean up. None of this is hard to learn, but it's real work, and underrating it is the classic beginner mistake when moving up from shared hosting.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a VPS gives you root on a virtual machine, but security, updates, and backups become your responsibility." markdown="0">
  <p class="knowledge-check__q">Quick check: What do you gain — and take on — when you move from shared hosting to a VPS?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A fully managed server where the host still handles all maintenance</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Root access to run anything, plus the responsibility for security, updates, and backups</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A whole physical machine with no other tenants sharing the hardware</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **A VPS is your own virtual machine** — an isolated slice of a physical server with full root access.
- **Virtualization makes it work** — a hypervisor splits one machine into many walled-off virtual ones; containers do a lighter version.
- **Providers are plentiful** — DigitalOcean, Linode, Hetzner, AWS EC2, and others rent them by the hour or month.
- **It runs anything** — you control the OS, so any language or stack is fair game.
- **Control brings responsibility** — security, patching, and backups become your job.
- **No physical I/O** — a cloud VPS has no USB ports, so it can host GopherTrunk's software but not its SDR hardware.

Next up: when shared hardware isn't enough and you want an entire physical machine to yourself. See [Dedicated servers](/learn/intro-hardware/dedicated-servers/).
