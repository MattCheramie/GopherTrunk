---
slug: home-servers
title: Home servers & self-hosting
description: A home server runs your own services on your own internet — an old PC, NAS, or Raspberry Pi. Learn the freedom, the hidden costs, and why it's a great first server.
keywords: home server, self-hosting, Raspberry Pi server, NAS, Plex Jellyfin, home automation, port forwarding, dynamic IP, residential internet, local hardware access
level: intermediate
status: full
faq:
  - q: "What is a home server?"
    a: "A **home server** is a computer you run in your own home to provide services — anything from an old desktop or a mini PC to a NAS box or a **Raspberry Pi**. Instead of renting space in a data center, you host your own apps, files, or media on hardware you own, connected through your home internet. The practice of running your own services this way is called **self-hosting**."
  - q: "What can I self-host at home?"
    a: "A lot. Common uses are a media server (**Plex** or **Jellyfin**), file storage and backups (a **NAS**), home automation, a dev/test environment, and self-hosted versions of apps you'd otherwise pay for. A home server is also ideal for an SDR setup like **GopherTrunk**, because the radio hardware plugs in locally — something a cloud VPS, with no USB ports, simply can't do."
  - q: "Why is a Raspberry Pi a good first home server?"
    a: "It's cheap, tiny, and sips power, so you can leave it running all day for pennies. It runs full Linux, so it teaches you real server skills, and it has **USB and GPIO ports** for attaching hardware locally. That makes it a forgiving, low-stakes way to learn self-hosting — and a natural home for a [single-board computer](/learn/intro-hardware/what-is-an-sbc/) project like a GopherTrunk box."
---
# Home servers & self-hosting

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**A home server is a machine in your home** — an old PC, a NAS, a mini PC, or a Raspberry Pi — running your own services. **Self-hosting buys ownership, privacy, and local hardware access** with no monthly fees. **The hidden costs are real** — your electricity, your uptime, residential internet quirks, and you as the entire IT department. **A Raspberry Pi is a great first server.**
</div>

So far the server has lived somewhere else — a shared host, a VPS, a dedicated box in a data center. But you can also run a server right at home, on hardware you own and internet you already pay for. It's how a lot of developers first truly *get* servers, and it unlocks things the cloud can't do. This lesson covers what self-hosting is, what people run, the freedom it gives you, and the costs nobody mentions until you hit them.

## What a home server is

A **home server** is simply a computer in your home that's left running to provide services to you and your devices. It doesn't have to be special hardware. Common choices are:

- **An old PC or laptop** — repurposed instead of recycled.
- **A NAS** (network-attached storage) — a dedicated box for files and backups.
- **A mini PC** — small, quiet, low-power, always on.
- **A Raspberry Pi** — credit-card-sized, cheap, and perfect for learning.

The practice of running your own apps and services on it — rather than renting them or relying on cloud providers — is called **self-hosting**. You own the machine, you own the data, and you decide what runs.

## What it's for

Home servers shine at jobs where ownership, privacy, or *local hardware* matter:

- **Media server** — stream your own library with **Plex** or **Jellyfin**.
- **File storage and backups** — a NAS that keeps your data at home, not in someone's cloud.
- **Home automation** — a hub coordinating lights, sensors, and switches locally.
- **Dev and test environment** — a sandbox to break things safely.
- **Self-hosted apps** — replacements for paid services you'd rather own.
- **An SDR / GopherTrunk box** — and this one is special. SDR hardware has to be *physically plugged in*, and a cloud VPS has no USB ports. A home server sits right where the antenna is, so you can attach the SDR dongle locally and run GopherTrunk's recorder and dashboard on the same machine. The radio's data enters through real hardware you can touch.

That last case captures the whole reason home servers exist alongside the cloud: some jobs need the machine to be *here*.

## The languages and stacks it runs

Like a VPS or a dedicated server, a home server runs **anything** — you control the operating system, usually Linux. Most self-hosted software is distributed as ready-to-run packages or **Docker containers**, so you spend less time writing code and more time configuring services. When you do build your own, Python, Node.js, Go, and shell scripts are common choices, and on a Raspberry Pi you can also reach the **GPIO pins** to talk to attached electronics — a capability no rented server has.

## Strengths and drawbacks

Self-hosting is genuinely empowering and genuinely more work. The honest balance:

| Strengths | Drawbacks |
|-----------|-----------|
| **Full ownership** — your hardware, your data | **Your electricity and uptime** — it's down when your power is |
| **No monthly fees** — pay once for the hardware | **Residential internet** — dynamic IPs, NAT, port forwarding, ISP terms |
| **Privacy** — data never leaves your home | **You are all of IT** — setup, updates, backups, repairs |
| **Learning** — real server skills, low stakes | **Security exposure** — anything you open to the internet is a target |
| **Local hardware access** — USB, GPIO, SDR dongles | **Reliability** — no data center's redundant power or cooling |

A few drawbacks deserve a closer look. **Residential internet** isn't built for serving: your IP address may change (a **dynamic IP**), your router hides you behind **NAT**, and reaching a service from outside usually means **port forwarding** — and some ISP terms of service frown on running public servers from home. And once you expose anything to the internet, *you* are responsible for keeping it secure. None of this is a dealbreaker for learning; it's just the reality the cloud was quietly handling for you.

## Why a Raspberry Pi is a great first server

If you want to try self-hosting, a **Raspberry Pi** is hard to beat as a starting point. It's inexpensive, it draws so little power you can leave it on around the clock for pennies, and it runs full Linux so the skills transfer directly to bigger servers. Crucially, it has USB and GPIO ports for attaching hardware locally — which is exactly what an SDR-based GopherTrunk box needs. It's a forgiving, low-cost way to learn, and it leads straight into the next module on single-board computers, which we'll explore starting with [what an SBC is](/learn/intro-hardware/what-is-an-sbc/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — a home server lets you plug in local hardware like an SDR dongle, which a cloud VPS can't do." markdown="0">
  <p class="knowledge-check__q">Quick check: What can a home server do that a cloud VPS cannot?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Run Linux and host websites</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Have physical hardware like an SDR dongle plugged directly into it via USB</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Provide redundant power and cooling for guaranteed uptime</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **A server in your home** — an old PC, NAS, mini PC, or Raspberry Pi running your own services.
- **Self-hosting buys ownership** — your hardware, your data, no monthly fees, real privacy.
- **Great for media, backups, automation, and SDR** — including a GopherTrunk box, because the radio plugs in locally.
- **Runs anything** — full OS control, often via Docker, plus GPIO for attached electronics.
- **The hidden costs are real** — electricity, uptime, dynamic IP/NAT/port forwarding, ISP terms, security, and being all of IT.
- **A Raspberry Pi is the ideal starter** — cheap, low-power, full Linux, and local hardware ports.

Next up: stepping away from servers to the machine you sit in front of every day, starting with the most capable of them. See [Desktop computers](/learn/intro-hardware/desktop-computers/).
