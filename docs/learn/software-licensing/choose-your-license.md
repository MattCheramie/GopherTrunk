---
slug: choose-your-license
title: Choosing your license, step by step
description: A guided, branching walkthrough that lands you on a specific license — open vs proprietary vs source-available vs dual, then permissive vs copyleft — with a decision table and worked mini-scenarios.
keywords: choose a software license, which license should I use, MIT vs Apache vs GPL vs AGPL, permissive vs copyleft, source available license, dual licensing, BSL, AGPL for SaaS, proprietary EULA, license decision tree, pick a license
level: intermediate
status: full
faq:
  - q: "I just want people to use my library — which license?"
    a: "A **permissive** license. Pick **MIT** for the shortest, simplest terms, or **Apache 2.0** if you also want an explicit patent grant (worth it for anything touching patentable techniques or aimed at corporate users). Both let companies adopt your code in closed products, which removes the biggest barrier to adoption."
  - q: "How do I stop a cloud provider from reselling my project as a service?"
    a: "Two paths. The **AGPL** keeps the project open source but forces anyone offering it as a network service to release their modifications. A **source-available** license like the BSL goes further by restricting commercial hosting outright, at the cost of no longer being open source. Pick based on whether staying OSI-open matters to you."
  - q: "What if I want to sell my software?"
    a: "Then you're past open source and into a **proprietary** license — typically a EULA for downloaded software or terms of service for a hosted product. Some teams keep an open or copyleft core and sell a proprietary edition (open core or dual licensing). Writing the actual license is its own job — see [writing a commercial license](/learn/software-licensing/writing-a-commercial-license/)."
---
# Choosing your license, step by step

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**One branching walkthrough** — answer a short series of questions and you land on a specific license, not a family. **First fork: open vs proprietary vs source-available vs dual** — this top-level choice determines everything downstream. **Within open: permissive vs copyleft** — MIT/Apache for reach, GPL/AGPL for sharing-back, LGPL/MPL for the middle. **Match the worry to the tool** — patent comfort → Apache; cloud resellers → AGPL or source-available; selling → proprietary.
</div>

The [decision framework](/learn/software-licensing/the-decision-framework/) gave you the inputs and a high-level profile. This lesson is the concrete follow-through: a guided walkthrough that takes you from "I have some software" to a single named license. We'll go branch by branch — the top-level fork first, then the sub-choices within open source — back it with a decision table, and finish with worked mini-scenarios so you can see the whole path in action. By the end you'll be able to name the license that fits your project and know exactly why.

## Step 1 — The top-level fork

Before any nuance, every project sits in one of four buckets. Answer this first; it determines which branch you walk next.

- **Open source** — anyone can use, modify, and redistribute, subject to a standard [open-source license](/learn/software-licensing/what-is-open-source/). Choose this if your goal is adoption or community, and the software isn't *directly* how you make money.
- **Proprietary** — you keep the rights and grant only limited, usually paid, permission to use. Choose this if selling the software (or restricting it) is the point.
- **Source-available** — the source is public to read, but the license restricts use (commonly: no competing commercial hosting). Choose this when you want transparency *and* protection from cloud resellers, and you're willing to give up the "open source" label.
- **Dual** — two licenses at once: typically a free copyleft license plus a paid proprietary one for those who can't accept copyleft. Choose this to monetize while staying open, if you own all the rights.

If you picked **proprietary**, jump to [writing a commercial license](/learn/software-licensing/writing-a-commercial-license/) — that's a whole craft of its own, and the rest of this lesson is about the open branch. If you picked **dual**, you'll still choose an open license for the free side using the steps below, then pair it with a commercial license. If you picked **source-available**, see Step 4. If you picked **open**, continue to Step 2.

## Step 2 — Within open source: permissive or copyleft?

This is the central question of open-source licensing, and it comes straight from your goal.

- **Permissive** (MIT, BSD, Apache) — "do almost anything, just keep my notice." Maximizes adoption by *allowing* closed forks. Pick this if reach matters more than forcing anyone to share back.
- **Copyleft** (GPL, AGPL) — "you may use and modify, but derivatives must stay open under the same terms." Prevents closed forks at some cost to adoption. Pick this if you want improvements to flow back.
- **Weak copyleft** (LGPL, MPL) — a middle path: the *file or library* stays open, but a larger work that merely links to it can remain closed. Pick this for a library you want shared back without forcing every user's whole app open.

The deeper trade-off is laid out in [permissive vs copyleft](/learn/software-licensing/permissive-vs-copyleft/) and the [weak-copyleft](/learn/software-licensing/weak-copyleft-licenses/) lesson. The rule of thumb: the more you value adoption, the more permissive you go; the more you value forced sharing-back, the stronger the copyleft.

## Step 3 — Within permissive: MIT or Apache?

If Step 2 sent you permissive, the practical choice is almost always MIT vs Apache 2.0.

- **MIT** (and the similar BSD) — the shortest, most familiar permissive license. Use it when you want minimal text and maximum simplicity, and patents aren't a concern. Covered in [MIT & BSD](/learn/software-licensing/mit-and-bsd-licenses/).
- **Apache 2.0** — permissive *plus* an explicit patent grant and patent-retaliation clause. Use it when your code might touch patentable techniques, or you want corporate users and legal teams to feel safe adopting it. Covered in [Apache 2.0](/learn/software-licensing/apache-license/).

A simple way to decide: **if you want a patent grant, choose Apache; otherwise MIT is fine.** GopherTrunk itself is licensed Apache-2.0 for exactly this reason — a signal-processing project benefits from the explicit patent grant.

## Step 4 — Within copyleft: GPL, AGPL, or source-available?

If Step 2 sent you copyleft, the deciding factor is *how your software is delivered* — distributed or hosted.

- **GPL** — strong copyleft triggered by **distribution**. If you ship binaries or source, recipients get the source and the same freedoms. But it has a famous gap: running modified code as a network service (without distributing it) doesn't trigger it. See [the GPL](/learn/software-licensing/gpl-strong-copyleft/).
- **AGPL** — closes that gap. Copyleft extends over the **network**, so a provider offering your software as a hosted service must release their modifications to users of the service. This is the open-source answer to cloud resellers. See [the AGPL](/learn/software-licensing/agpl-network-copyleft/).
- **Source-available (e.g., BSL)** — if the AGPL still isn't enough — you want to *prohibit* commercial hosting outright, not just require source-sharing — a [source-available license](/learn/software-licensing/source-available-licenses/) does that, but it is **not** open source and will narrow your contributor and adopter pool.

## The whole walkthrough as a table

Here's the branching logic collapsed into a single lookup. Find the row that matches your strongest need; the license is the answer.

| Your strongest need | License | Why |
|---|---|---|
| Maximum adoption, simplest terms | **MIT** | Permissive, minimal, universally understood |
| Adoption + a patent grant | **Apache 2.0** | Permissive with explicit patent protection |
| A library shared back, app stays closed-able | **LGPL / MPL** | Weak copyleft scoped to the component |
| Improvements to distributed software stay open | **GPL** | Strong copyleft on distribution |
| Block closed *and* cloud-only forks, stay open source | **AGPL** | Copyleft over the network |
| Prohibit competing commercial hosting (giving up "open") | **Source-available (BSL)** | Use restriction, not just share-back |
| Sell the software | **Proprietary EULA** | You keep rights; users buy limited permission |
| Monetize while staying open, you own all rights | **Dual (copyleft + commercial)** | Free copyleft side + paid proprietary side |

## Worked mini-scenarios

Seeing the walkthrough applied makes it stick. Four common cases:

**1. A library you want everyone to adopt.** Goal: reach. Step 1 → open. Step 2 → permissive (you *want* companies to build on it). Step 3 → MIT if patents are irrelevant, **Apache 2.0** if you want the patent grant and corporate comfort. → **MIT or Apache 2.0.**

**2. A SaaS backend you want to keep others from reselling.** Goal: control, specifically against cloud capture. Step 1 → open (you still want the openness benefit) or source-available (if you want a harder block). Step 2 → copyleft. Step 4 → **AGPL** to force resellers to publish modifications, or **BSL** if you'd rather forbid commercial hosting outright. → **AGPL or BSL.**

**3. A paid desktop app.** Goal: revenue, downloaded delivery. Step 1 → proprietary. There's no open branch to walk; you grant limited use under a **EULA** and head to [writing a commercial license](/learn/software-licensing/writing-a-commercial-license/). → **Proprietary EULA.**

**4. A copyleft tool you also want to sell to companies that can't accept copyleft.** Goal: revenue *and* openness. Step 1 → dual. You pick **GPL or AGPL** for the free side (Steps 2/4) and pair it with a **paid commercial license** for customers who need proprietary terms. This requires owning all the rights — see [dual licensing](/learn/software-licensing/dual-licensing-and-relicensing/). → **Dual: copyleft + commercial.**

One closing rule that applies to every branch: **pick a standard, well-known license and don't write your own.** Custom licenses create compatibility headaches, scare off contributors and corporate legal teams, and may not even qualify as open source. Use an OSI-approved license via [choosealicense.com](https://choosealicense.com) or the [SPDX list](https://spdx.org/licenses/), then apply it correctly — a `LICENSE` file plus per-file notices — so the choice actually takes effect.

<div class="knowledge-check" data-quiz data-correct-msg="Right — Apache 2.0 is the permissive choice when you want an explicit patent grant; MIT is the choice when you don't need one." markdown="0">
  <p class="knowledge-check__q">Quick check: you've decided on a permissive license and you want an explicit patent grant. Which do you pick?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">MIT, because it's the shortest license</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">AGPL, because it has the strongest protections</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Apache 2.0, because it adds an explicit patent grant to permissive terms</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Start at the top fork** — open vs proprietary vs source-available vs dual decides which branch you walk.
- **Within open, permissive vs copyleft** — reach (MIT/Apache) vs forced sharing-back (GPL/AGPL), with LGPL/MPL as the middle.
- **Within permissive, patents decide** — Apache 2.0 if you want a patent grant, MIT if you don't.
- **Within copyleft, delivery decides** — GPL for distribution, AGPL to also cover network hosting, source-available to block commercial hosting outright.
- **Selling means proprietary** — a EULA or hosted terms, written as its own document.
- **Always pick a standard license** — use an OSI-approved license and apply it correctly; never invent your own.

Next up: the license covers your code, but how you deliver it decides what else you need. See [Which agreements does your software need?](/learn/software-licensing/which-agreements-do-you-need/).
