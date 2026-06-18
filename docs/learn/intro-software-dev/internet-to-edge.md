---
slug: internet-to-edge
title: PCs, the internet & the age of cheap compute
description: From mainframes to minicomputers, PCs, the web, cloud, mobile, and edge — how cheap, abundant compute and open source made software-defined radio practical.
keywords: history of computing, mainframe, minicomputer, personal computer, client server, the web, cloud computing, edge computing, open source, software-defined radio
level: beginner
status: full
faq:
  - q: "What is the difference between cloud and edge computing?"
    a: "**Cloud computing** runs your software in large, remote data centers you rent over the internet, scaling up on demand. **Edge computing** pushes work back out to small devices near where the data is — sensors, phones, an SDR plugged into a laptop. They're complementary: the edge handles real-time, local work, and the cloud handles heavy storage and large-scale processing."
  - q: "How did cheap computing make software-defined radio possible?"
    a: "Software radio needs an ordinary computer fast enough to process a stream of radio samples in real time. Decades of falling chip prices put that much power in any laptop, while cheap analog-to-digital hardware digitizes the signal. Add free, open-source DSP libraries and a thirty-dollar dongle, and a setup that once needed specialized lab gear now runs on a personal machine."
  - q: "Why does open source matter for software development?"
    a: "**Open source** means software whose code is published for anyone to read, use, and improve. It lets developers build on shared, peer-reviewed foundations instead of reinventing everything, which dramatically speeds up progress. Whole fields — including software-defined radio — exist as practical hobbies today largely because their core libraries are free and open."
---
# PCs, the internet & the age of cheap compute

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Compute spread outward** — from shared mainframes to personal PCs to phones to tiny edge devices. **The network changed everything** — client/server, the web, and the cloud connected those machines. **Cheap compute plus open source** — made software-defined radio a laptop hobby instead of lab work.
</div>

In the last lesson we met the people who built programming in labs and on mainframes. This final lesson of the module follows what happened when computing left the lab — getting smaller, cheaper, and more connected until it reached everyone, everywhere. That same arc is the reason you can do real radio work on a personal laptop today, so it's a fitting bridge from history into the practical modules ahead. It builds on [the pioneers](/learn/intro-software-dev/the-pioneers/) and the whole hardware story before it.

## Mainframes: computing as a shared resource

The first commercial computers were **mainframes** — room-sized, hugely expensive machines that an entire organization shared. A company or university might own one, and everyone's jobs queued up to run on it. You didn't sit at a computer; you submitted work to *the* computer and waited.

This shaped early software culture: computing time was precious and rationed, programs were submitted on punch cards or terminals, and access was a privilege. The mainframe model — many users, one powerful central machine — is worth remembering, because computing keeps swinging between centralized and personal, and we'll see that pendulum return.

## Minicomputers: smaller, closer, cheaper

By the 1960s and 70s, the **minicomputer** shrank that model. Still not personal, but small and cheap enough that a single department or lab could own one outright rather than sharing a corporate mainframe. Machines like Digital Equipment Corporation's PDP series put real computing within reach of researchers and engineers — and, not coincidentally, it was on this class of machine that Unix and C were born. Minicomputers were the proving ground where the falling cost of hardware first started to democratize who could compute.

## The personal computer

The microprocessor — a whole CPU on one chip, from [the hardware lesson](/learn/intro-software-dev/computing-hardware-story/) — made the next leap possible: the **personal computer**. Through the late 1970s and 1980s, machines small and affordable enough for one person arrived on desks at home and at work. Computing stopped being something you shared and became something you *owned*.

This was a genuine inversion of the mainframe model. Instead of many people queuing for one big machine, each person had their own. An entire software industry sprang up to fill those machines with applications, and for the first time ordinary people — not just trained operators — wrote and ran their own programs.

## Networks: client/server and the web

Personal machines were powerful, but isolated. The next era connected them.

- **Client/server.** Networks let a personal machine (the **client**) request services from a more powerful shared machine (the **server**) — files, databases, email. This recombined the personal and the central: your own computer in front of you, talking to shared resources over a network.
- **The web.** In 1989, **Tim Berners-Lee** proposed the **World Wide Web** — a system of linked documents accessed over the internet through a browser. It turned the internet from a specialist tool into something anyone could use, and unleashed an explosion of connected software.

The web mattered enormously for software because it made *distribution* trivial. You no longer shipped programs on disks; you served them over the network, instantly, worldwide. That shift sets up everything in the later [packaging and distribution](/learn/intro-software-dev/packaging-and-distribution/) lesson.

## Cloud, mobile, and the edge

The last three waves came quickly and overlap today:

| Era | What it is | The shift |
|-----|-----------|-----------|
| **Cloud** | Renting computing in vast remote data centers, on demand | Back to centralized — but infinitely scalable and pay-as-you-go |
| **Mobile** | Powerful computers in everyone's pocket | A capable computer with every person, all the time |
| **Edge / embedded** | Small, cheap chips doing real work near the data source | Compute pushed back out to the physical world |

Notice the pendulum: the **cloud** swung back toward the centralized, shared model of the mainframe — only now the "mainframe" is a global data center you rent by the minute. **Mobile** put a capable computer in every pocket. And **edge computing** pushed processing back out to tiny, cheap chips embedded in sensors, appliances, cars — and radios. After decades of swinging between central and personal, today we have *all of it at once*: data centers, laptops, phones, and a flood of small embedded devices, all networked together.

## Open source: software as a shared commons

One force runs through all of this: **open source**. Open-source software publishes its code for anyone to read, use, and improve, usually for free. Instead of every team rebuilding the same foundations in secret, the community shares peer-reviewed building blocks that everyone can stand on.

The impact is hard to overstate. **Linux** (open source) runs most of the world's servers and underlies Android. Countless libraries that developers rely on every day are open source. Progress compounds because you start from everyone else's best work rather than from scratch. Open source turned software into a shared commons, and that commons is exactly what makes the next point possible.

## How cheap compute made the radio software

Pull the threads together and you arrive back at GopherTrunk's world. Three trends converged:

1. **Cheap, abundant compute.** Decades of Moore's Law mean any ordinary laptop has the raw power to process a stream of radio samples in real time — something that once required specialized digital signal processing hardware.
2. **Cheap digitizing hardware.** Inexpensive analog-to-digital devices — like a thirty-dollar USB dongle — turn the airwaves into the stream of numbers that software then works on.
3. **Open-source DSP libraries.** Free, community-built signal-processing software does the heavy mathematical lifting, so you don't have to write filters and demodulators from scratch.

Put those together and **software-defined radio** stops being a lab discipline and becomes a laptop hobby. The receiver that once meant a rack of dedicated equipment is now mostly *software*, running on the same personal computer you use for everything else, built on open libraries anyone can download. It is the exact endpoint of this whole module: cheap compute, networks, and open source democratized not just computing in general, but radio specifically. If you want to chase that thread further, the [RF & SDR path](/learn/rf-sdr/) picks it up from the radio side.

<div class="knowledge-check" data-quiz data-correct-msg="Right — cloud is centralized rented data centers; edge pushes compute out to small devices near the data." markdown="0">
  <p class="knowledge-check__q">Quick check: What best describes the difference between cloud and edge computing?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Cloud runs on your phone; edge runs in remote data centers</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Cloud is centralized rented data centers; edge pushes compute out to small devices near the data</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">They are two names for the exact same thing</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Mainframes** — computing began as a costly central resource everyone shared and queued for.
- **Minicomputers and PCs** — falling hardware costs moved computing first to departments, then to individual desks.
- **Networks** — client/server and the web connected those personal machines and made software distribution instant and worldwide.
- **Cloud, mobile, edge** — the pendulum swung back to centralized data centers while also putting compute in every pocket and tiny device.
- **Open source** — a shared, peer-reviewed software commons let progress compound instead of restarting from scratch.
- **The payoff** — cheap compute plus cheap digitizers plus open DSP libraries turned software-defined radio into something you run on a laptop.

Next up: this module told the story; the next one digs into the tools. We start by mapping how programming languages group into families with shared traits in [Language families](/learn/intro-software-dev/language-families/).
