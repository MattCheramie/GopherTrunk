---
slug: start-with-requirements
title: Start with your requirements
description: Good hardware choices begin with a clear list of requirements — compute, connectivity, power, budget, and where the device must physically live — not with a favorite board.
keywords: hardware requirements, project requirements checklist, choosing hardware, compute and memory needs, power source, connectivity needs, hardware budget, where hardware lives
level: beginner
status: full
faq:
  - q: "Why write requirements before choosing hardware?"
    a: "Because the hardware should fall out of the needs, not the other way around. If you pick a board first and then bend the project to fit it, you'll either overpay for capacity you never use or hit a wall the part can't clear. Writing the **requirements** down — compute, connectivity, power, budget, location — turns a guess into a short list of platforms that can actually do the job."
  - q: "What are the most important questions to answer first?"
    a: "Five carry most of the weight: **what does it do** (how much compute and memory), **does it need networking or the internet**, **does it touch the physical world** (sensors, motors, a radio dongle), **how is it powered** (battery, wall, always-on), and **where does it physically live**. Each answer rules large parts of the [hardware spectrum](/learn/intro-hardware/hardware-spectrum/) in or out before you ever compare prices."
  - q: "Does it matter whether I'm building one unit or thousands?"
    a: "Hugely. A $35 board is trivial for a one-off but ruinous at ten thousand units, where a $3 microcontroller wins. Scale flips which platform is sensible, so **how many units** belongs in your requirements from the start — it changes the answer as much as performance does."
  - q: "Who maintains it — why does that belong in requirements?"
    a: "Because a machine someone has to babysit costs more than its price tag. If it lives somewhere awkward, runs 24/7, or will be updated by a non-developer, that pushes you toward simpler, more reliable, more remotely-manageable options. Maintenance is a real constraint, not an afterthought."
---
# Start with your requirements

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Requirements first, hardware second** — write down what the project needs before you fall in love with a board. **A handful of questions decide most of it** — compute, connectivity, physical-world contact, power, budget, and location each rule parts of the spectrum in or out. **Scale and maintenance count too** — one unit versus thousands, and who looks after it, can flip the answer entirely.
</div>

This is the first lesson of Module 7 — the payoff module, where everything from the platform tour comes together into a single skill: choosing well. And good choosing starts long before you compare boards or hosting plans. It starts with writing down what the project actually needs. This lesson is about that discipline: the short list of questions that, once answered honestly, point you straight at the right part of the [hardware spectrum](/learn/intro-hardware/hardware-spectrum/).

## Why requirements come before parts

It's tempting to start from a part you already like — a Raspberry Pi you have in a drawer, a cloud provider you've used before — and shape the project around it. That's backwards. The hardware should fall out of the requirements, not drive them.

When you choose the part first, one of two things happens. Either you over-buy, paying for performance, power draw, and complexity you never use; or you under-buy and slam into a limit the part simply can't clear — no amount of clever code gives a cloud server a USB port in your living room. Writing requirements down first is the cheapest insurance against both. It also makes the decision *defensible*: when someone asks why you chose what you did, you can point at the need, not a hunch.

The good news is that requirements for a hardware project are short. A page is usually plenty. The art is asking the right questions.

## The questions that decide it

Work through these in order. Each answer eliminates options, which is exactly what you want — a smaller field is an easier decision (the [decision framework](/learn/intro-hardware/decision-framework/) later turns this into a repeatable method).

- **What does it do — and how much compute and memory does that take?** Blinking an LED and editing 4K video sit at opposite ends of the [spectrum](/learn/intro-hardware/hardware-spectrum/). Be concrete: a web dashboard for a handful of users is light; training a machine-learning model is not.
- **Does it need networking or the internet?** A public service needs to be reachable; an offline data logger needs none. Connectivity requirements pull you toward servers and away from isolated microcontrollers, or vice versa.
- **Does it touch the physical world?** Sensors, motors, buttons, a radio dongle — anything that must be *physically attached* rules out remote cloud machines immediately, and pulls you toward a [microcontroller](/learn/intro-hardware/what-is-a-microcontroller/) or a [single-board computer](/learn/intro-hardware/what-is-an-sbc/) sitting where the hardware is.
- **How is it powered?** Battery, wall outlet, or always-on mains changes everything. A device that must run for months on a coin cell is a microcontroller job; one that's fine drawing 60 watts around the clock opens up servers and desktops.
- **What's the budget — and at what quantity?** One unit or ten thousand? Cost that's trivial once can be fatal at scale. This is where a $3 part beats a $35 one even though the $35 one is "better."
- **Where does it physically live?** A clean office, a rack in a data center, a damp garden, the base of an antenna on a roof. Location dictates ruggedness, connectivity, and power, and often eliminates whole categories on its own.
- **Who maintains it?** A developer at a keyboard, or a non-technical person who just wants it to work? Hard-to-reach or 24/7 devices favor simple, reliable, remotely-managed options.

## A requirements checklist you can reuse

Capture the answers in a small table like this one. Filling it in *before* you shop is the whole point — the right column will already be hinting at platforms by the time you're done.

| Requirement | The question to answer | Which way it points |
|-------------|------------------------|---------------------|
| **Compute & memory** | How heavy is the work — trivial, moderate, or demanding? | Light → MCU/SBC; heavy → desktop/server/cloud |
| **Connectivity** | Offline, local network, or public internet? | Public → [VPS](/learn/intro-hardware/vps/)/server; offline → standalone device |
| **Physical world** | Does it need sensors, actuators, or attached hardware? | Yes → device on-site, never a remote cloud box |
| **Power** | Battery, wall, or always-on mains? | Battery → microcontroller; always-on → SBC/server |
| **Budget & quantity** | Cost ceiling, and one unit or many? | Many → cheapest part that works; one → convenience wins |
| **Location** | Clean indoors, harsh outdoors, data center, remote? | Harsh/remote → rugged, low-maintenance, remotely managed |
| **Maintenance** | Who updates and fixes it, how easily? | Non-technical/hard to reach → simplest reliable option |

## A worked read: where would GopherTrunk run?

Run the questions against **GopherTrunk**, the SDR scanner these docs are about, and watch the field narrow. *What does it do?* Captures and decodes radio — moderately demanding signal processing. *Networking?* Useful, so you can view recordings from elsewhere. *Physical world?* Yes — and this is decisive: the SDR dongle is a **USB device that must be physically plugged into the machine**. *Power and location?* It needs to sit near the antenna, often a roof or attic, ideally always-on. *Budget?* A hobby build, one unit.

Those answers eliminate a cloud [VPS](/learn/intro-hardware/vps/) outright — you can't plug a USB dongle into a server in someone else's data center. They point instead at a machine that lives *at the antenna*: a [Raspberry Pi](/learn/intro-hardware/raspberry-pi-and-family/) bolted up near the feedline, or a [laptop](/learn/intro-hardware/laptops/)/desktop with the dongle attached. We didn't compare specs to get there. We just wrote down the requirements, and the physical-world answer did most of the work.

<div class="knowledge-check" data-quiz data-correct-msg="Right — pin down the requirements first, and the suitable platforms fall out of them." markdown="0">
  <p class="knowledge-check__q">Quick check: What should you do before picking a board or a host?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Pick the most powerful platform you can afford, to be safe</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Write down what the project needs — compute, connectivity, physical contact, power, budget, location — and let that narrow the field</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Start with the part you already own and bend the project to fit it</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Requirements before hardware** — let the needs choose the part, not the other way around, to avoid over-buying or hitting a wall.
- **A short list of questions decides most of it** — compute and memory, connectivity, physical-world contact, power, budget, location, and maintenance.
- **Physical-world contact is decisive** — anything that must be plugged in or sensed rules out remote cloud machines immediately.
- **Scale flips the answer** — one unit versus thousands changes which platform is sensible as much as performance does.
- **Capture it in a checklist** — a one-page table filled in before you shop already points at the right region of the spectrum.

Next up: take those requirements and map common kinds of projects to the platforms they tend to reach for. See [Matching hardware to the project type](/learn/intro-hardware/matching-hardware-to-project/).
