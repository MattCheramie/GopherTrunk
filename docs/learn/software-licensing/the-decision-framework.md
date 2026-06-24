---
slug: the-decision-framework
title: A framework for choosing
description: A repeatable framework that turns six inputs — your goal, business model, dependencies, delivery model, data handled, and risk tolerance — into a clear license and agreement decision.
keywords: choosing a software license, license decision framework, license and agreement decision, open core, SaaS licensing, business model licensing, delivery model, license compatibility, risk tolerance, software licensing strategy, dual licensing decision
level: intermediate
status: full
faq:
  - q: "Where do I even start when picking a license and agreements?"
    a: "Start with your **goal**, not the licenses. Decide whether you most want adoption, control, or revenue — that single answer steers everything else. Then layer in your business model, dependencies, delivery model, data, and risk tolerance. The framework in this lesson walks those six inputs in order so the answer falls out at the end."
  - q: "Can my dependencies force my hand on which license I pick?"
    a: "Yes. If you build on **strong-copyleft** code like the GPL and distribute the result, you generally have to license your combined work under compatible terms. So before you fall in love with a license, audit what you already depend on — see [license compatibility](/learn/software-licensing/license-compatibility/) and [auditing dependencies](/learn/software-licensing/auditing-dependencies/)."
  - q: "Is the license the whole decision?"
    a: "No — the license governs your **code**, but how you *deliver* the software and what *data* it touches decide which **agreements** you also need (a EULA, terms of service, a privacy policy, an SLA, and so on). This lesson produces both halves of the decision; the next two lessons turn each half into specifics."
---
# A framework for choosing

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Six inputs, one decision** — your goal, business model, dependencies, delivery model, data handled, and risk tolerance together determine your license and agreements. **Goal first** — adoption vs control vs revenue is the master question that orients everything else. **Constraints before preferences** — your existing dependencies and the data you touch can rule options out before you ever pick a favorite. **Two outputs** — the framework produces a *license* for your code and an *agreement set* for how you ship it.
</div>

You've spent this path learning the parts — license families, open source, proprietary models, agreements, and how to read the fine print. This lesson assembles those parts into a single repeatable process: a short, ordered set of questions that turns facts about your project into a license-and-agreement decision you can defend. By the end you'll have a framework you can run on any project — yours, your employer's, or a side project — and a summary table that maps the inputs to outcomes. The next two lessons then turn the two halves of that decision into specifics: [which license](/learn/software-licensing/choose-your-license/) and [which agreements](/learn/software-licensing/which-agreements-do-you-need/).

## The six inputs

The decision feels overwhelming because people try to answer it all at once. Don't. The choice is driven by six inputs, and if you gather them in order, the answer mostly assembles itself. Two of them — your goal and your business model — are *preferences* that point you toward an option. The other four — dependencies, delivery model, data, and risk — are *constraints* that can rule options out no matter what you'd prefer.

| Input | The question it answers | What it drives |
|---|---|---|
| **Goal** | What do you want to *happen* to this software? | The broad license posture |
| **Business model** | How (if at all) does this make money? | License + commercial agreements |
| **Dependencies** | What licenses must you already comply with? | Hard limits on your license |
| **Delivery model** | How does the software reach users? | Which agreements you need |
| **Data handled** | What does it collect or process? | Privacy/data agreements |
| **Risk & jurisdiction** | How much risk can you carry, and where? | Strength of terms, where you operate |

Work top to bottom. The preferences set your direction; the constraints prune it. Let's take them one at a time as a set of questions with outcomes.

## Question 1 — What is your goal: adoption, control, or revenue?

This is the master question. Almost every later trade-off resolves once you know which of these three you weight most heavily.

- **Adoption** — you want the widest possible use, including by companies building closed products on top of you. Fewer conditions means more reach. → *Points toward permissive open source.*
- **Control** — you want to keep your code shared but stop others from taking it private, or from reselling it as a service. → *Points toward copyleft (or, if a cloud competitor is the worry, network copyleft or source-available).*
- **Revenue** — the software is, directly, how you make money, and you need to charge for it or restrict it. → *Points toward proprietary or a dual/open-core model.*

These aren't mutually exclusive — many projects want some of each — but ranking them honestly is the single most clarifying thing you can do. A project that says "I want maximum adoption *and* nobody can ever build a closed fork" is asking for two licenses at once, and the [permissive-vs-copyleft](/learn/software-licensing/permissive-vs-copyleft/) trade-off is exactly that tension.

## Question 2 — What is your business model?

Your goal is what you *want*; your business model is how the project sustains itself. The two usually agree, but naming the model sharpens the license choice and tells you whether you need *commercial* agreements at all.

| Business model | Typical license posture | Commercial agreements? |
|---|---|---|
| **Pure open / community** | Permissive or copyleft open source | No — just the open license |
| **Paid, downloaded product** | Proprietary EULA | Yes — a EULA, maybe support terms |
| **SaaS / hosted** | Often AGPL or source-available for the core; proprietary backend | Yes — ToS, subscription terms |
| **Open core** | Permissive/copyleft core + proprietary add-ons | Yes — for the paid tier only |
| **Dual-licensed** | Copyleft for free use + a paid proprietary license | Yes — the commercial license |

If you're considering open core or dual licensing, revisit [commercial license models](/learn/software-licensing/commercial-license-models/) and [dual licensing & relicensing](/learn/software-licensing/dual-licensing-and-relicensing/) — those models have real ownership prerequisites (you generally need to own or have CLA-backed rights to all the code you want to relicense).

## Question 3 — What dependencies must you already comply with?

Here the *constraints* begin, and this one can override your preferences entirely. You don't get a free hand: the licenses of the code you build on can dictate what license your own work may carry.

The rule that bites hardest is strong copyleft. If you incorporate GPL or AGPL code and then **distribute** (or, for AGPL, **network-serve**) the result, you generally must release the whole combined work under compatible copyleft terms. That can make a proprietary or even a permissive outcome impossible without removing the dependency. Permissive dependencies (MIT, BSD, Apache) impose far lighter obligations — mostly attribution — and rarely constrain your choice.

So before you commit to anything in Questions 1–2, inventory what you depend on. This is precisely the work covered in [license compatibility](/learn/software-licensing/license-compatibility/) and [auditing dependencies](/learn/software-licensing/auditing-dependencies/). If an audit turns up a copyleft dependency that conflicts with your intended outcome, you have three options: change the license you target, replace the dependency, or get a commercial license for it. Discovering this *after* you've shipped is far more expensive.

## Question 4 — How is the software delivered?

Delivery model is what switches the decision from "just a license" to "license **plus** agreements." How the software reaches its users determines which agreements you need on top of the license.

| Delivery model | License governs | You also typically need |
|---|---|---|
| **Downloaded / installed** | The copy they run | A EULA |
| **Hosted / SaaS** | (Often nothing the user holds) | Terms of service, privacy policy |
| **Embedded / bundled** | The component you ship | Compliance with the component's license; passthrough terms |

Two subtleties matter. First, for a hosted service the user never receives a copy, so a traditional EULA is the wrong instrument — a [terms of service](/learn/software-licensing/eula-and-tos/) governs the relationship instead. Second, delivery interacts with Question 3: network copyleft (AGPL) is triggered by *hosting*, not by *distributing*, which is exactly why it matters for SaaS and not for a desktop app.

## Question 5 — What data does it handle?

If your software touches personal data — names, emails, locations, anything tied to a person — you inherit legal obligations that exist independently of your license. At minimum you'll need a [privacy policy](/learn/software-licensing/privacy-and-data-agreements/); depending on jurisdiction you may owe GDPR or CCPA duties, and if vendors process data on your behalf you'll need a data processing agreement (DPA) with them.

This input is binary in a useful way: either you handle personal data or you don't, and the answer adds a fixed block of agreements. A GopherTrunk-style scanner that decodes radio signals locally and stores nothing personal sits at the easy end; a hosted dashboard that logs user accounts and locations sits at the demanding end.

## Question 6 — What is your risk tolerance and jurisdiction?

The final input tunes the *strength* of your terms rather than their kind. Higher stakes — paying customers, regulated data, enterprise buyers — justify stronger warranty disclaimers, limitation-of-liability caps, and clearer indemnification, the [risk clauses](/learn/software-licensing/main-clauses-decoded/) you met earlier. Lower stakes (a hobby project) need far less.

Jurisdiction matters too, but mostly as a flag: consumer-protection rules in the EU and UK limit how far you can disclaim liability, and data law varies widely. Keep it light here — the dedicated [licensing across jurisdictions](/learn/software-licensing/licensing-across-jurisdictions/) lesson carries the cross-border depth. The framework just asks: where do your users live, and how much risk can you afford to carry?

## The framework as one table

Run the six questions and you'll land in one of a handful of common profiles. This summary table collapses the framework into its typical outcomes — a starting point, not a verdict.

| Profile (goal + model) | Likely license | Likely agreement set |
|---|---|---|
| Library, want broad adoption | Permissive (MIT/Apache) | License only |
| App, want code shared but no closed forks | GPL | License only (+ EULA if you also sell builds) |
| SaaS, want to block cloud resellers | AGPL or source-available (BSL) | ToS, privacy policy, DPA, maybe SLA |
| Paid desktop app | Proprietary EULA | EULA, support terms |
| Open-core product | Permissive/copyleft core + proprietary tier | License (core) + commercial agreements (tier) |
| Dual-licensed component | Copyleft + paid proprietary | Copyleft license + commercial license |

Notice the table has two columns of output: a **license** for your code and an **agreement set** for how you deliver it. That split is the whole point of the framework, and it's why the next two lessons exist — one for each column.

<div class="knowledge-check" data-quiz data-correct-msg="Right — your goal (adoption vs control vs revenue) is the master input that orients the whole decision." markdown="0">
  <p class="knowledge-check__q">Quick check: in this framework, which input is the "master question" you answer first?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Your goal — adoption, control, or revenue</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Which programming language you wrote it in</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Whether you personally prefer MIT or GPL</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Six inputs drive the decision** — goal, business model, dependencies, delivery model, data handled, and risk/jurisdiction.
- **Goal first** — adoption, control, or revenue is the master question that orients everything else.
- **Constraints can override preferences** — your existing dependencies (especially copyleft) and the data you touch can rule options out before you pick a favorite.
- **Delivery and data add agreements** — how you ship and what data you handle decide which agreements you need on top of the license.
- **Two outputs** — the framework produces a license for your code *and* an agreement set for how you deliver it.
- **It's a starting point** — the summary table gives a typical answer; the next lessons turn each half into specifics.

Next up: turn the first output into a concrete pick with the guided, branching walkthrough in [Choosing your license, step by step](/learn/software-licensing/choose-your-license/).
