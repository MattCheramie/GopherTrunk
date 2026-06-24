---
slug: putting-it-all-together
title: Putting it all together
description: A full end-to-end worked example — turning an SDR scanner into an open-source core plus a paid hosted edition — applying the framework to choose a license, handle dependencies, and assemble the agreement set. The capstone of the path.
keywords: software licensing worked example, open core licensing, Apache vs AGPL, third party licenses, SBOM dependency scanning, SaaS agreements, DPA SLA subscription terms, GopherTrunk Apache 2.0, putting licensing together, end to end licensing decision, staying compliant
level: advanced
status: full
faq:
  - q: "How do I license an open-source core and a paid hosted version at the same time?"
    a: "Common pattern: license the **core** under a standard open-source license (Apache 2.0 for adoption, or AGPL to keep hosted competitors from going closed) and run the **paid hosted edition** under your own terms of service plus the commercial agreement stack. The open license governs the code; the hosted edition's agreements govern the service you sell on top of it."
  - q: "What's the minimum agreement set for a paid hosted product?"
    a: "At a realistic minimum: **terms of service** (the rules of the service), a **privacy policy** (what you collect and why), a **DPA** with any vendor that processes user data for you, and **subscription terms** (the commercial deal). Add an **SLA** if you promise uptime, and a **CLA/DCO** if the open core takes outside contributions."
  - q: "What is GopherTrunk actually licensed under?"
    a: "**Apache 2.0** — see the repository's [`/LICENSE`](https://github.com/MattCheramie/GopherTrunk/blob/main/LICENSE) file. It also ships a [`/THIRD_PARTY_LICENSES.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/THIRD_PARTY_LICENSES.md) attribution file and inventories transitive dependency licenses with `make licenses` in CI. It's a real, working example of permissive licensing done correctly, which is why this path keeps pointing at it."
---
# Putting it all together

> **This is educational material, not legal advice.** For decisions that carry real risk, consult a qualified attorney.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**One worked example, end to end** — we take a realistic product through the whole framework: license, dependencies, and agreements. **Open core + paid hosted** — Apache (or AGPL) for the core, a full agreement stack for the hosted edition. **Dependencies are real work** — attribution files and scanning keep you compliant as the tree changes. **You now have a process** — six inputs in, a license and an agreement set out, repeatable on any project.
</div>

This is the final lesson of the path. You've learned the foundations, the license families, how to combine open source, how proprietary and commercial licensing work, how to use open source in software you sell, how to read agreements, and how to choose your own. Now we put it all to work on one realistic product, start to finish, so the whole thing clicks into a single mental motion. Then we'll wrap up everything you can now do, note how to stay compliant as things change, and point you to where to get help. By the end you'll have seen the entire path applied at once — and be ready to do it for real.

## The scenario

Imagine you've built a capable SDR trunking scanner — much like [GopherTrunk](https://github.com/MattCheramie/GopherTrunk) — that decodes P25, DMR, and TETRA from raw radio captures. You want two things at once:

1. An **open-source core** so the community can use, audit, and improve the decoder.
2. A **paid hosted edition** — a web dashboard where customers point their receivers at your cloud, and you decode, store, and visualize their traffic for a subscription.

This is the classic **open-core / hosted** model, and it's an ideal capstone because it exercises every part of the framework: an open-source license *and* a commercial one, real dependencies to comply with, a hosted delivery model, and personal data. Let's run the [decision framework](/learn/software-licensing/the-decision-framework/) on it.

## Step 1 — Run the six inputs

Walking the framework's six inputs:

| Input | For this product |
|---|---|
| **Goal** | Both adoption (the core) *and* revenue (the hosted edition) |
| **Business model** | Open core: free core + paid hosted service |
| **Dependencies** | Go modules and in-tree code under various OSS licenses |
| **Delivery model** | Downloaded (core binary) **and** hosted (the web edition) |
| **Data handled** | Hosted edition stores user accounts, captures, locations |
| **Risk & jurisdiction** | Paying customers + personal data → moderate-to-high; assume US + EU users |

Notice the split goal. That's the signature of open core, and it means we'll produce *two* answers — a license for the core and a commercial stack for the hosted edition — rather than one.

## Step 2 — Choose the license for the core

Run the [license walkthrough](/learn/software-licensing/choose-your-license/). Step 1 (the top fork) is **open** for the core. Step 2 (permissive vs copyleft) is where the interesting decision lives, and it hinges on the one thing the open-core model has to worry about: a competitor taking your open core and running their *own* hosted edition against you.

- **Apache 2.0** — maximizes adoption and includes a patent grant, which matters for signal-processing code. The trade-off: nothing stops a competitor from hosting your core commercially. You accept that in exchange for the widest possible community. **This is what GopherTrunk itself chose** (`/LICENSE`).
- **AGPL** — keeps the core open but forces any competitor who hosts a modified version to publish their changes, blunting the "take it closed and resell it" move. The trade-off: AGPL makes some corporate users and contributors wary, shrinking adoption.

Both are defensible; the choice is a pure expression of the [permissive-vs-copyleft](/learn/software-licensing/permissive-vs-copyleft/) trade-off you learned early. For a project whose *first* priority is community trust and reach, **Apache 2.0** is the right call — and your paid edition's moat then comes from the hosting, support, and proprietary dashboard, not from the license. If protecting the hosted edition were the higher priority, **AGPL** (or even a [source-available](/learn/software-licensing/source-available-licenses/) license, giving up the "open" label) would be the move. We'll go with **Apache 2.0 for the core**, matching the real GopherTrunk.

## Step 3 — Handle the open-source dependencies

Your core isn't written from scratch — it links a tree of Go modules and includes some in-tree code derived from other projects. Before you ship *anything*, open or paid, you must comply with those licenses. This is the [auditing dependencies](/learn/software-licensing/auditing-dependencies/) and [permissive compliance](/learn/software-licensing/permissive-compliance/) work made concrete.

Two things to get right:

- **Attribution.** Permissive licenses (MIT, BSD, Apache) require you to carry their copyright notices and license text. GopherTrunk does this with a [`/THIRD_PARTY_LICENSES.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/THIRD_PARTY_LICENSES.md) file listing every direct dependency, its version, license, and upstream — plus the in-tree derived code. Your product needs the same: one canonical attribution file.
- **Scanning, continuously.** A dependency tree changes with every update, and a new transitive dependency can quietly introduce a license you don't want (a copyleft one that conflicts with your distribution). Automate it. GopherTrunk inventories transitive licenses with a `make licenses` target and a CI `licenses` job that fails the build on a surprise. This is the practical face of an **SBOM** (software bill of materials).

```text
# The pattern, illustrated
$ make licenses          # regenerate the dependency license inventory
$ git diff THIRD_PARTY_LICENSES.md   # review what changed
# CI runs the same check and fails if an unexpected/incompatible license appears
```

The lesson here, which the whole [using-OSS-in-products](/learn/software-licensing/using-oss-in-products/) module drove home: compliance isn't a one-time chore at launch — it's a check that runs forever, because the inputs keep moving.

## Step 4 — Assemble the agreement set for the hosted edition

The core is just an Apache-licensed binary — it needs the license and the attribution file, nothing more. The **paid hosted edition** is where the [agreement set](/learn/software-licensing/which-agreements-do-you-need/) comes in. Run that lesson's table against this product: it's a business SaaS, it handles personal data, you'll promise some uptime, and the open core takes contributions. That unions to:

| Agreement | Why this product needs it |
|---|---|
| **Terms of Service** | It's a hosted service users access, not a copy they install — ToS governs accounts, acceptable use, payment, termination ([EULA & ToS](/learn/software-licensing/eula-and-tos/)) |
| **Privacy policy** | It stores user accounts, captures, and locations — personal data, with GDPR (EU) and CCPA (US/CA) duties ([privacy & data](/learn/software-licensing/privacy-and-data-agreements/)) |
| **DPA** | Your cloud host and any analytics vendor process that personal data on your behalf ([privacy & data](/learn/software-licensing/privacy-and-data-agreements/)) |
| **Subscription terms** | The commercial deal — pricing, renewal, refunds — for the paid tier ([commercial models](/learn/software-licensing/commercial-license-models/)) |
| **SLA** | You promise uptime; the SLA states the level and the remedy (credits) if you miss ([SLAs & support](/learn/software-licensing/sla-and-support-agreements/)) |
| **CLA or DCO** | The open core accepts outside contributions; you need clear rights to that code ([contributor agreements](/learn/software-licensing/contributor-agreements/)) |

Note one thing worth flagging given the product: a scanner that decodes radio traffic may touch communications that are sensitive or regulated in some jurisdictions. That's a domain-specific legal question beyond licensing, and exactly the kind of thing to raise with a lawyer — your ToS should at minimum put acceptable-use responsibility on the customer.

That's the complete picture: **Apache 2.0 + attribution** for the core, and **ToS + privacy policy + DPA + subscription terms + SLA + CLA/DCO** for the hosted edition. Six inputs in, a license and an agreement set out — the framework delivered exactly what it promised.

## You now know how to…

Step back and look at what this path has given you. You can now:

- **Tell a license from a contract** and read either one — the [main clauses](/learn/software-licensing/main-clauses-decoded/), the [risk clauses](/learn/software-licensing/how-to-read-an-agreement/), and what to check [before you agree](/learn/software-licensing/before-you-agree/).
- **Place any license on the spectrum** — from [public domain](/learn/software-licensing/public-domain-and-cc0/) through [permissive](/learn/software-licensing/mit-and-bsd-licenses/) and [Apache](/learn/software-licensing/apache-license/) to [copyleft](/learn/software-licensing/gpl-strong-copyleft/) and [network copyleft](/learn/software-licensing/agpl-network-copyleft/), with the [weak-copyleft](/learn/software-licensing/weak-copyleft-licenses/) middle.
- **Combine open source safely** — [compatibility](/learn/software-licensing/license-compatibility/), [contributor agreements](/learn/software-licensing/contributor-agreements/), and the [risks](/learn/software-licensing/open-source-risks/) of depending on it.
- **Build a business on it** — [proprietary](/learn/software-licensing/proprietary-licensing/) and [commercial models](/learn/software-licensing/commercial-license-models/), [source-available](/learn/software-licensing/source-available-licenses/) options, and how to [sell software built on open source](/learn/software-licensing/using-oss-in-products/) while meeting its obligations.
- **Choose for your own project** — run the [framework](/learn/software-licensing/the-decision-framework/), pick a [license](/learn/software-licensing/choose-your-license/), and assemble the [agreement set](/learn/software-licensing/which-agreements-do-you-need/).

That's a complete working literacy in software licensing — enough to make confident, defensible decisions and to know precisely when a question has grown big enough to hand to a professional.

## Staying compliant as things change

A license-and-agreement decision is not a monument; it's a living thing. Three habits keep it healthy:

- **Re-scan dependencies on every change.** A new transitive dependency can introduce an incompatible license overnight — which is why GopherTrunk's `make licenses` check runs in CI, not once a year.
- **Revisit when the product changes.** Add an API, start handling a new kind of data, expand into a new region, or open a paid tier, and your agreement set changes with it. Re-run the [agreements table](/learn/software-licensing/which-agreements-do-you-need/).
- **Watch the landscape.** Licenses and data laws evolve; staying current is part of the job, and the [jurisdictions](/learn/software-licensing/licensing-across-jurisdictions/) lesson is your starting point for the cross-border picture.

<div class="knowledge-check" data-quiz data-correct-msg="Right — in open core you license the code with an open license and govern the paid service with its own agreement stack; they're separate, layered decisions." markdown="0">
  <p class="knowledge-check__q">Quick check: in an open-core product with a paid hosted edition, what governs the paid service itself?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The same open-source license as the core, with nothing added</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Only a EULA, since the customer is using your software</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A separate agreement stack — terms of service, privacy policy, DPA, subscription terms, and often an SLA</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **The framework delivers two outputs** — for our scanner: Apache 2.0 (with attribution) for the core, and a full agreement stack for the paid hosted edition.
- **License the code, then agreements layer on top** — the open license governs the core; ToS, privacy policy, DPA, subscription terms, SLA, and a CLA/DCO govern the hosted service and contributions.
- **GopherTrunk is the real example** — Apache-2.0 (`/LICENSE`), a `/THIRD_PARTY_LICENSES.md` attribution file, and a `make licenses` CI check show permissive licensing done correctly.
- **Compliance is continuous** — re-scan dependencies, revisit agreements when the product changes, and watch the legal landscape.
- **You have a repeatable process** — six inputs in, a license and an agreement set out, ready to apply to any project.

That's the whole path. You started not knowing what a software license *is*, and you can now take a real product end-to-end — choose its license, comply with its dependencies, and assemble its agreements — and recognize the moment a question deserves a professional's eyes. When that moment comes, [Where to get licensing help](/learn/software-licensing/where-to-get-licensing-help/) points you to the right resources.

Thank you for working through it. To consolidate every term you've met along the way, head to the [glossary](/learn/software-licensing/glossary/) — and go make something, license it well, and ship it with confidence.
