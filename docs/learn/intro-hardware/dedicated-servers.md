---
slug: dedicated-servers
title: Dedicated servers
description: A dedicated server is a whole physical machine that's yours alone — no virtualization, no neighbors. Learn when uncontended performance is worth the price.
keywords: dedicated server, bare metal server, physical server, no virtualization, uncontended performance, high traffic hosting, GPU server, dedicated vs VPS, server hardware
level: intermediate
status: full
faq:
  - q: "What is a dedicated server?"
    a: "A **dedicated server** is an entire physical machine reserved for one customer — no hypervisor splitting it, no other tenants sharing its CPU, memory, or disk. You rent (or own) the whole box and get all of its hardware, all the time. It sits one step beyond a VPS: instead of a virtual slice of a machine, you have the machine itself, sometimes called **bare metal**."
  - q: "When is a dedicated server worth it over a VPS?"
    a: "When you need **consistent, uncontended performance** or direct hardware access. High-traffic sites, large databases, busy game servers, heavy compute, and GPU or machine-learning workloads all benefit because no noisy neighbor can steal cycles. It's also chosen for predictable performance and certain compliance needs. The cost is real money and less elasticity, so it's overkill until your workload actually demands it."
  - q: "What are the downsides of a dedicated server?"
    a: "It's **expensive** compared to a VPS, **slower to provision** (a physical box may take hours instead of minutes), and **less elastic** — you can't resize it with a click. If hardware fails you either handle it yourself or pay for a managed plan where the provider does. You're paying for raw, reserved performance, and you give up the cloud's instant flexibility to get it."
---
# Dedicated servers

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**A dedicated server is a whole physical machine, yours alone** — no virtualization, no neighbors. **You get maximum, consistent performance** and full hardware access. **It costs more and flexes less** — expensive, slower to provision, and not elastic like the cloud. **Choose it when uncontended performance is the requirement**, not before.
</div>

A VPS shares its physical hardware with other tenants, and most of the time that's fine. But some workloads can't tolerate sharing — they need every cycle, every gigabyte, and predictable performance that no neighbor can disturb. That's where a dedicated server comes in: a complete physical machine reserved for you. This lesson covers what changes when nothing is shared, the jobs that justify the price, and how it stacks up against a VPS.

## What a dedicated server is

A **dedicated server** is an entire physical machine devoted to a single customer. There's no hypervisor carving it into slices and no other tenants competing for its resources. When you rent one, you get the whole box — every CPU core, all the memory, the full disk, the network card — and it's all yours, all the time. This is why it's often called **bare metal**: you're running directly on the hardware, with no virtualization layer skimming a share for someone else.

You can rent dedicated servers from hosting providers month to month, or — for an on-premises setup — own the physical machine outright. Either way, the defining feature is exclusivity: one machine, one tenant.

## What it's for

A dedicated server earns its keep wherever performance must be high *and* consistent:

- **High-traffic websites** — sites whose load would crowd a shared machine.
- **Large or busy databases** — where steady disk and memory performance matters.
- **Game servers** — latency-sensitive workloads that hate jitter from noisy neighbors.
- **Heavy compute** — video encoding, simulations, large builds, batch processing.
- **GPU and machine-learning workloads** — training and inference that need specific, powerful hardware you can't get from a generic VPS.
- **Predictable performance and compliance** — when you must guarantee resources or keep data on an isolated, known machine.

The common thread is that *sharing is the problem*. If a VPS's variable performance or lack of direct hardware access is hurting you, a dedicated server removes that ceiling.

## The languages and stacks it runs

As with a VPS, the answer is **anything**. You control the whole operating system and the whole machine, so any language, runtime, database, or specialized framework is available — including GPU stacks like CUDA for machine learning, which a generic VPS often can't offer. The difference from a VPS isn't *what* you can run but *how much* and *how consistently* it runs, because the hardware underneath is yours alone.

## Dedicated server vs VPS

Most of the decision comes down to a few trade-offs against a VPS:

| Aspect | VPS | Dedicated server |
|--------|-----|------------------|
| **Hardware** | Shared via a hypervisor | An entire physical machine, yours alone |
| **Performance** | Good, but can vary with neighbors | Maximum and consistent |
| **Cost** | Cheap to moderate | Expensive |
| **Provisioning** | Ready in minutes | Slower — often hours for a physical box |
| **Elasticity** | Resize and scale on demand | Fixed; hard to scale quickly |
| **Hardware access** | None (virtual) | Full — GPUs, specific cards, raw disks |
| **Hardware failures** | Provider's problem | Yours, or pay for a managed plan |

The pattern is clear: a dedicated server buys you raw, reserved, predictable power, and the price is money and flexibility. A VPS is the right default; a dedicated server is what you reach for when the VPS can no longer keep up or can't give you the hardware you need.

## Strengths and drawbacks

To put the same balance plainly:

| Strengths | Drawbacks |
|-----------|-----------|
| **Maximum performance** — all the hardware, all the time | **Expensive** — you pay for the whole machine |
| **Consistent** — no noisy neighbors, no contention | **Less elastic** — can't resize with a click |
| **Full hardware access** — GPUs, custom configurations | **Slower to provision** — physical setup takes time |
| **Isolation** — useful for compliance and security | **Hardware failures** — your problem unless managed |

For something like **GopherTrunk**, a dedicated server is usually more machine than the job needs — the radio's data stream is modest. But if you were running a large multi-receiver decoding operation crunching many streams at once, the steady, uncontended compute of a dedicated box could be exactly the right fit.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a dedicated server is a whole physical machine with no virtualization and no shared tenants." markdown="0">
  <p class="knowledge-check__q">Quick check: What most distinguishes a dedicated server from a VPS?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It's cheaper and can be resized instantly</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It's an entire physical machine with no virtualization and no other tenants sharing it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It's fully managed, so you never handle the operating system</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **A whole physical machine, yours alone** — no hypervisor, no shared tenants, often called bare metal.
- **Built for uncontended performance** — high-traffic sites, big databases, game servers, heavy compute, GPU/ML.
- **Runs anything** — full control of the OS and direct hardware access, including specialized cards.
- **More power, less flexibility** — maximum consistent performance, but expensive, slow to provision, and not elastic.
- **VPS first, dedicated when needed** — reach for bare metal only when sharing becomes the bottleneck.

Next up: bringing the server home — running your own machine on your own internet, with all the freedom and hidden costs that come with it. See [Home servers & self-hosting](/learn/intro-hardware/home-servers/).
