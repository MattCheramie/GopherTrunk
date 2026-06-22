---
slug: choosing-a-dev-machine
title: Choosing your development machine
description: Pick a desktop, laptop, or both by matching it to how you work — and learn which specs (RAM, SSD, CPU, screen, GPU) actually matter for writing software.
keywords: choosing a development machine, developer laptop vs desktop, dev machine specs, how much RAM for development, remote development, thin client, WSL, SSD for coding
level: intermediate
status: full
faq:
  - q: "What spec matters most for a development machine?"
    a: "**Memory first.** Editors, browsers, containers, and virtual machines all eat RAM, and running out of it slows everything to a crawl in a way more processor speed can't fix. After RAM, a fast SSD and a solid multi-core CPU round out the essentials. A powerful GPU only matters for specific work like machine learning or game development."
  - q: "Should I get a desktop or a laptop for coding?"
    a: "Match it to how you work. If you code in one place and want the most power for the money, a **desktop** wins. If your work moves with you, a **laptop** is the default. Many developers end up with both — a portable laptop for daily use and a desktop or remote server for the heavy lifting."
  - q: "Can I develop on a cheap laptop and run heavy work elsewhere?"
    a: "Yes, and it's a popular pattern. You edit on a light, inexpensive laptop while the demanding work — builds, tests, large services — runs on a remote **server or VPS** you connect to. The laptop becomes a comfortable window onto a much more powerful machine, so you don't pay for raw power you only need occasionally."
---
# Choosing your development machine

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Match the machine to how you work** — mobility, budget, and workload decide between a desktop, a laptop, or both. **RAM matters most** — then a fast SSD and a decent multi-core CPU; a strong GPU only for specific jobs. **You don't have to own all the power** — a light laptop plus a remote server or VPS lets you develop anywhere and run heavy work elsewhere.
</div>

You've now seen both personal machines: the desktop's raw value and the laptop's portability. This lesson turns that into a decision. Choosing a development machine isn't about chasing the biggest numbers — it's about matching a machine to the way you actually work and spending your budget where it helps your code. By the end you'll have a framework for picking desktop, laptop, or both, and you'll know which specs are worth paying for.

## Start with how you work

Before any spec sheet, answer three questions about yourself:

- **Mobility** — do you work in one place, or does your work travel? If it moves, you need a laptop. If it never leaves the desk, a desktop gives you more for the money.
- **Budget** — for a fixed budget, a desktop buys more performance; a laptop spends part of that budget on portability.
- **Workload** — light editing and web work run on almost anything. Heavy compiling, virtual machines, video, or machine learning reward more memory, a faster CPU, and sometimes a GPU.

Those three usually point at one of three answers:

| Your situation | Best fit |
|----------------|----------|
| Work in one place, want maximum value | **Desktop** |
| Work moves with you, "good enough" power | **Laptop** |
| Want mobility *and* serious power | **Both** — a laptop plus a desktop or remote server |

This is the same matching exercise the whole path builds toward in Module 7's [decision framework](/learn/intro-hardware/decision-framework/) — here, scaled down to one practical choice.

## The specs that actually matter

It's easy to over-index on processor speed. For most development, these matter more, roughly in order:

- **Memory (RAM) — first priority.** Code editors, browsers with many tabs, containers, databases, and virtual machines all consume RAM at once. Run out and the machine slows to a crawl no faster CPU can rescue. 16 GB is a comfortable floor for development in 2026; 32 GB gives real breathing room.
- **A fast SSD.** Solid-state storage makes the whole machine feel quick — opening projects, building code, searching files. Avoid old-style spinning hard drives for your main disk, and favor capacity you won't outgrow.
- **A decent multi-core CPU.** Compiling and running tests benefit from several cores. You rarely need the absolute top chip; a solid mid-range modern CPU is plenty.
- **A good screen.** You stare at it all day. Enough resolution and size to keep code and a browser side by side is worth more to your comfort than most benchmark wins.
- **A GPU — only sometimes.** A powerful graphics card matters for machine learning, game development, or 3D, and barely at all for ordinary web or backend work. Don't pay for one you won't use.

Spend on memory and storage before raw processor speed, and you'll feel the difference every day.

## Choosing an operating system

The machine is only half the choice; the system on it shapes your daily work. All three are practical for development:

- **Linux** — free, scriptable, and the same family of system most servers run, so what you build locally behaves like production. A strong default for backend and systems work.
- **macOS** — a polished Unix-based system on Apple hardware, popular for general software and the only supported way to build iOS apps.
- **Windows + WSL** — Windows with the **Windows Subsystem for Linux** gives you a real Linux environment alongside everything Windows runs, a comfortable middle path for many.

There's no single right answer. Pick the one that fits your tools and your target platforms; the languages and editors you'll use are available on all three.

## Develop light, run heavy: the remote pattern

Here's the option that breaks the "buy all the power you need" assumption. You don't have to run demanding work *on* your everyday machine at all. In the **thin client + remote server** pattern, you:

- Work on a **light, inexpensive laptop** — comfortable to carry, cheap to replace.
- Run the heavy work on a **remote server or VPS** you connect to over the network — long builds, big test suites, always-on services, or model training.

Your laptop becomes a window onto a far more powerful machine. The benefits are real: you carry less, the heavy machine can be sized exactly for the job and shared across devices, and you stop paying a portability premium for raw power you only need occasionally. The trade is that you need a network connection and a little setup. This leans directly on the hosting tier you met earlier — a [VPS](/learn/intro-hardware/vps/) for a modest rented machine, or a [dedicated server](/learn/intro-hardware/dedicated-servers/) when you want the whole box.

## A wrinkle for hardware work like SDR

The remote pattern has one important limit, and **GopherTrunk** illustrates it nicely. Software-defined radio needs a physical SDR dongle plugged into a **USB** port, and that hardware has to be on the machine actually processing the signal. You can't easily move a real-time USB radio stream to a distant server.

So your dev machine and your radio host may sensibly be *different computers*. You might write and test GopherTrunk on a laptop wherever you are, then deploy it to a desktop or small always-on machine sitting right at the antenna with the SDR plugged in. The lesson generalizes: whenever a project depends on local hardware, that hardware pins part of the job to a specific machine, no matter how you arrange the rest. Module 7's [combining tiers](/learn/intro-hardware/combining-tiers/) lesson explores splitting a project across machines like this.

<div class="knowledge-check" data-quiz data-correct-msg="Right — RAM is the first thing to get right; running out of it slows everything down." markdown="0">
  <p class="knowledge-check__q">Quick check: Which spec should you prioritize first on a development machine?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The most powerful GPU you can afford</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Enough memory (RAM), then a fast SSD and a solid multi-core CPU</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The highest single-core clock speed available</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Match the machine to how you work** — mobility, budget, and workload decide desktop, laptop, or both.
- **RAM first** — then a fast SSD and a decent multi-core CPU; a GPU only for ML, games, or 3D.
- **A good screen counts** — daily comfort often beats marginal benchmark gains.
- **Any of the three operating systems works** — Linux, macOS, or Windows with WSL; pick by tools and target platform.
- **Develop light, run heavy** — a thin laptop plus a remote server or VPS gives you mobility and power without buying both in one box.
- **Local hardware pins the job** — SDR's USB dongle means your dev machine and your radio host may differ.

Next up: we leave the desk entirely for the computer almost everyone already owns — the one in your pocket. See [Smartphones](/learn/intro-hardware/smartphones/).
