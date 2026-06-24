---
slug: choosing-an-oss-license
title: Choosing an open-source license
description: Turn what you know about license families into a decision. Learn the key questions — adoption, preventing proprietary forks, SaaS protection, patent grants — and a goal-to-license decision table for MIT, Apache, MPL, GPL, and AGPL.
keywords: choosing an open source license, which license to pick, MIT vs Apache vs GPL, open source license decision, choosealicense, SPDX, OSI approved licenses, AGPL SaaS, patent grant license, prevent proprietary forks, license families
level: intermediate
status: full
faq:
  - q: "What license should I pick if I just want as many people as possible to use my code?"
    a: "Reach for a **permissive** license — **MIT** if you want minimal text and maximum simplicity, or **Apache 2.0** if you also want an explicit patent grant. Permissive licenses let anyone, including companies shipping closed products, adopt your code, which removes the biggest barrier to adoption."
  - q: "How do I stop a company from taking my open-source project, running it as a paid service, and giving nothing back?"
    a: "That's exactly what the **AGPL** is designed for. It extends copyleft to network use, so a provider who offers your code as a hosted service must release their modified source. The MPL and GPL don't reach network-only use, so they won't close that gap."
  - q: "Should I ever write my own custom open-source license?"
    a: "Almost never. A custom license creates compatibility headaches, scares off contributors and corporate legal teams, and may not even qualify as open source. Pick a standard **OSI-approved** license from [choosealicense.com](https://choosealicense.com) or the [SPDX list](https://spdx.org/licenses/) — they're vetted, understood, and tooling-friendly."
---
# Choosing an open-source license

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Start from your goal** — adoption, preventing proprietary forks, SaaS protection, or a patent grant each point to a different license. **Map goals to families** — permissive (MIT/Apache) for reach, weak copyleft (MPL) for a middle path, strong copyleft (GPL/AGPL) for sharing-back. **Pick a standard license** — choose an OSI-approved license via choosealicense.com or SPDX; never write your own. **Apply it correctly** — add the license file and per-file notices so the choice actually takes effect.
</div>

You've toured the license families — [permissive](/learn/software-licensing/mit-and-bsd-licenses/), [Apache](/learn/software-licensing/apache-license/), [GPL](/learn/software-licensing/gpl-strong-copyleft/), [AGPL](/learn/software-licensing/agpl-network-copyleft/), and the [weak-copyleft](/learn/software-licensing/weak-copyleft-licenses/) middle — and you understand [compatibility](/learn/software-licensing/license-compatibility/). This lesson turns all of that into a single decision: which license to put on *your* project. By the end you'll have a short list of questions to ask yourself, a goal-to-license table, the tools that make the choice easy, and a clear reason to pick a standard license rather than invent one.

## The decision is about your goals, not the licenses

There's no "best" open-source license — there's the one that matches what you want to happen to your code. The whole choice collapses to a few honest questions about your intent. Answer these and the license almost picks itself.

- **Do you want adoption above all else?** If your priority is the widest possible use — including by companies building closed products on top of you — you want the fewest possible conditions. That points permissive.
- **Do you want to prevent proprietary forks?** If you'd be unhappy seeing someone take your code, improve it, and ship those improvements without sharing back, you want copyleft to require that they share.
- **Do you want to protect against SaaS capture?** If your worry is specifically a cloud provider running your code as a *service* and never distributing a binary (so ordinary copyleft never triggers), you need network copyleft.
- **Do you need an explicit patent grant?** If your code might touch patentable techniques, or you want corporate users to feel safe, you want a license that grants patent rights and includes a patent-retaliation clause.
- **Do you care about contributor and corporate comfort?** Some licenses (notably the AGPL) make corporate legal teams nervous and can shrink your contributor pool; permissive licenses are the easiest sell.

Notice these can conflict. Maximum adoption and strong sharing-back pull in opposite directions — permissive maximizes reach by *allowing* closed forks, while copyleft prevents closed forks at the cost of some adoption. Choosing a license is choosing which trade-off you want.

## A goal-to-license decision table

Here's the mapping from the question you answered "yes" to most strongly, to the license family that serves it.

| Your main goal | License to reach for | What it gives you | The trade-off |
|---|---|---|---|
| Maximum adoption, minimal fuss | **MIT** | Shortest, most familiar permissive license | No patent grant; closed forks allowed |
| Adoption **plus** patent protection | **Apache 2.0** | Permissive + explicit patent grant + notice rules | Slightly longer; incompatible with GPLv2 |
| A middle path — share file-level changes but allow linking into closed code | **MPL 2.0** | File-level (weak) copyleft | Won't force the *whole* product open |
| Prevent proprietary forks of a distributed app/library | **GPLv3** | Strong copyleft on distribution | Many companies avoid GPL dependencies |
| Prevent SaaS capture — share-back even when only offered as a service | **AGPLv3** | Copyleft extended to network use | Strongest deterrent to corporate adoption |

If you genuinely don't know, the safe default for most new projects that just want to be used is **MIT or Apache 2.0** — Apache if patents or corporate trust matter, MIT if you want the absolute minimum. If your project's whole point is keeping the ecosystem open, **GPLv3** (distributed software) or **AGPLv3** (network services) is the deliberate copyleft choice.

## Why a standard, OSI-approved license beats a custom one

It is tempting to write your own license, or to bolt a custom clause onto a standard one. Resist it. A non-standard license causes real, lasting problems:

- **Compatibility unknowns.** Standard licenses have well-understood [compatibility](/learn/software-licensing/license-compatibility/) relationships. A custom license is a question mark — nobody knows if it can be combined with anything.
- **Contributor and corporate friction.** Developers and company legal teams recognize MIT, Apache, and GPL instantly and can clear them in minutes. A bespoke license triggers an expensive, slow legal review — or an outright "no."
- **It may not even be open source.** "Open source" has a specific definition (the OSI's Open Source Definition). A homemade license with, say, a non-commercial clause is *source-available*, not open source — a distinction we cover in [Source-available & "fair source"](/learn/software-licensing/source-available-licenses/).
- **Tooling won't recognize it.** License scanners, SBOM generators, and package registries key off standard SPDX identifiers. A custom license is invisible or flagged as "unknown" to all of them.

The **OSI** (Open Source Initiative) maintains the canonical list of licenses that meet the Open Source Definition. **SPDX** assigns each a short standard identifier (`MIT`, `Apache-2.0`, `GPL-3.0-only`, `AGPL-3.0-only`) that tooling everywhere relies on. Sticking to that vetted set is almost always the right call.

## Tools that make the choice (and the application) easy

You don't have to reason from scratch:

- **[choosealicense.com](https://choosealicense.com)** — GitHub's plain-language guide. Answer a couple of questions about what you want and it recommends a license, shows the full text, and even helps you add it. The best starting point for most people.
- **The [OSI license list](https://opensource.org/licenses)** — the authoritative set of licenses that meet the Open Source Definition. If a license isn't here (or in the SPDX list), be skeptical of calling it open source.
- **The [SPDX license list](https://spdx.org/licenses/)** — standard identifiers for every common license, the strings you put in metadata and SBOMs.

Choosing isn't enough — you have to *apply* the license so it legally takes effect. In practice that means a `LICENSE` file at the repository root and, for many licenses, a short per-file header. GopherTrunk does exactly this: its [Apache-2.0](/learn/software-licensing/apache-license/) terms live in `/LICENSE`, and it ships a `/THIRD_PARTY_LICENSES.md` attribution file for its dependencies.

```text
# A typical short license header for an Apache-2.0 source file
# SPDX-License-Identifier: Apache-2.0
# Copyright 2026 The GopherTrunk Authors
```

The `SPDX-License-Identifier` line is the machine-readable way to declare a file's license — short, unambiguous, and recognized by every scanning tool.

## A quick recap of the family strengths and weaknesses

To consolidate the whole module so far:

| Family | Strength | Weakness |
|---|---|---|
| **Permissive (MIT/BSD)** | Maximum adoption; trivial to comply with | Allows closed forks; no patent grant |
| **Apache 2.0** | Permissive *and* an explicit patent grant; corporate-friendly | Patent terms make it incompatible with GPLv2 |
| **Weak copyleft (MPL/LGPL/EPL)** | Share file/library changes, still linkable into closed code | Boundary of what must be shared can be subtle |
| **Strong copyleft (GPL)** | Forces distributed derivatives to stay open | Many companies refuse GPL dependencies |
| **Network copyleft (AGPL)** | Closes the SaaS loophole; strongest share-back | Biggest deterrent to corporate adoption |

There's no winner in this table — only fits. The right license is the one whose strength matches your goal and whose weakness you can live with.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the AGPL extends copyleft to network use, so a provider offering your code as a hosted service must release their source." markdown="0">
  <p class="knowledge-check__q">Quick check: which license is designed to stop a cloud provider from running your code as a paid service without sharing their changes?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">MIT, because it has the fewest conditions</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Apache 2.0, because of its patent grant</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">AGPLv3, because it extends copyleft to network use</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Start from your goal** — adoption, preventing proprietary forks, SaaS protection, or a patent grant each lead to a different license.
- **Map goal to family** — MIT/Apache for reach, MPL for a middle path, GPL to keep distributed derivatives open, AGPL to close the SaaS loophole.
- **Default to MIT or Apache** if you mainly want to be used; choose copyleft deliberately when keeping the ecosystem open is the point.
- **Pick a standard license** — use an OSI-approved, SPDX-identified license; never write your own.
- **Use the tools** — choosealicense.com, the OSI list, and SPDX make both the choice and the metadata easy.
- **Apply it correctly** — a `LICENSE` file plus per-file SPDX headers are what make your choice actually bind.

Next up: once others start contributing to your project, who owns and licenses *their* code? See [Contributor agreements: CLAs & DCO](/learn/software-licensing/contributor-agreements/).
