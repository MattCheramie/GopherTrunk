---
slug: nda-and-other-agreements
title: NDAs & the rest of the landscape
description: Round out the agreement landscape — NDAs and confidentiality, the MSA plus SOW structure for B2B deals, acceptable use policies, beta and trial agreements, partner/reseller terms, and developer/API terms, and how they all fit with the license.
keywords: NDA, non-disclosure agreement, confidentiality agreement, mutual NDA, one-way NDA, MSA, master service agreement, statement of work, SOW, order form, acceptable use policy, AUP, beta agreement, evaluation agreement, reseller agreement, API terms, developer terms
level: beginner
status: full
faq:
  - q: "What's the difference between a one-way and a mutual NDA?"
    a: "A **one-way (unilateral) NDA** protects one side's secrets — typical when only one party is sharing, like a company showing a contractor its plans. A **mutual NDA** protects both sides because both will share confidential information — common when two companies explore a partnership. Mutual is the default when the conversation is genuinely two-directional; the obligations are otherwise much the same."
  - q: "Why split a B2B deal into an MSA plus a SOW or order form?"
    a: "It separates the *stable legal terms* from the *deal-specific details*. The **Master Service Agreement (MSA)** is negotiated once and covers liability, IP, confidentiality, and the like; each new project or purchase is then a short **Statement of Work (SOW)** or **order form** referencing the MSA. You re-negotiate the heavy terms once, then move fast on each subsequent deal."
  - q: "Does an NDA make information confidential just because it says so?"
    a: "Not by itself. An NDA defines *what counts* as confidential and obliges the recipient to protect it, but it also includes **carve-outs** — information that's already public, independently developed, or rightfully obtained elsewhere isn't covered. And to keep trade-secret protection you generally must actually treat the information as secret, not just paper it. The agreement is one part of a broader practice."
---
# NDAs & the rest of the landscape

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**NDAs protect secrets** — one-way or mutual, with a defined scope, a term, and carve-outs. **MSA + SOW structures B2B deals** — negotiate the heavy terms once, then add short order forms. **AUPs set usage boundaries** — what users may not do on a service. **Beta, partner, and API terms** cover special relationships. **It all clips onto the license** — the license grants permission; these agreements govern the relationships around it.
</div>

This module has worked through the agreements your software shows users — [EULAs and terms of service](/learn/software-licensing/eula-and-tos/), [privacy and data agreements](/learn/software-licensing/privacy-and-data-agreements/), and [SLAs and support](/learn/software-licensing/sla-and-support-agreements/). This final lesson rounds out the landscape: the confidentiality agreements that protect what you share, the contract *structure* B2B vendors use, and the specialized terms for betas, partners, and developers. None of these replace the license — they sit *around* it. By the end you'll recognize each agreement, know what it governs, and understand when you reach for it.

> **This is educational material, not legal advice.** For agreements that carry real risk, have a qualified attorney review them before you sign.

## NDAs and confidentiality

A **Non-Disclosure Agreement (NDA)** — also called a confidentiality agreement — is a contract where one or both parties promise to protect information the other shares. You sign them constantly: before a partnership talk, when hiring a contractor, during due diligence, or when showing pre-release plans. NDAs are also one of the main ways you protect a [trade secret](/learn/software-licensing/other-ip-in-software/), which keeps its legal status only as long as you actually keep it secret.

**One-way vs mutual.** A **one-way (unilateral)** NDA protects a single side's information — appropriate when only one party is disclosing, such as a company briefing a freelancer. A **mutual (bilateral)** NDA protects both, which is the norm when two organizations will each reveal something, like two companies scoping a deal. The obligations look similar either way; the difference is *who is protected*.

Whatever the direction, the substance turns on a few clauses:

- **Definition of "Confidential Information."** What's actually covered — often anything marked confidential, plus anything a reasonable person would treat as such.
- **Carve-outs (exclusions).** Information that's *not* covered: already public, already known to the recipient, independently developed, or rightfully received from a third party. These keep an NDA from absurdly locking up public facts.
- **Permitted use and recipients.** The recipient may use the information only for the agreed purpose and may share it only with people who need it and are themselves bound.
- **Term.** How long the duty lasts — a fixed number of years after disclosure, though genuine trade secrets are sometimes protected for as long as they stay secret.
- **Return or destruction.** What happens to the information when the relationship ends.

A common misconception is that signing an NDA *makes* information confidential. It doesn't, on its own — the carve-outs still apply, and you must back the paper with actual secrecy practices for it to mean much.

## MSA + SOW: the structure of B2B deals

Business-to-business software and services rarely live in one big document. The standard pattern splits the stable legal terms from the deal-specific details:

- The **Master Service Agreement (MSA)** is the master contract, negotiated once. It carries the heavy, slow-to-agree terms: liability limits, indemnification, IP ownership, confidentiality, warranties, governing law, termination — the [risk clauses](/learn/software-licensing/before-you-agree/) you don't want to re-fight on every order.
- A **Statement of Work (SOW)** or **order form** then describes a single engagement *under* the MSA: the specific scope, deliverables, timeline, and price. It's short, because it inherits all the legal terms from the MSA by reference.

```text
  Master Service Agreement (MSA)        <- the heavy terms, signed once
    ├── SOW #1: build the data pipeline   (scope, dates, price)
    ├── Order form: 50 seats, 12 months   (quantity, term, fees)
    └── SOW #2: migration project         (scope, dates, price)
```

The payoff is speed without re-litigating fundamentals: agree the MSA once, then each new project or purchase is a quick SOW or order form. When you review a B2B deal, read the MSA *and* the SOW together — a friendly-looking order form may be carrying very serious MSA terms beneath it.

## Acceptable Use Policy (AUP)

An **Acceptable Use Policy** spells out what users may *not* do with a service — no illegal activity, no spam, no hacking or probing, no overloading the system, no infringing others' rights. It's often broken out of the [terms of service](/learn/software-licensing/eula-and-tos/) into its own document so the provider can update the rules of conduct without reopening the main contract. Violating the AUP is typically grounds for suspension or termination, which ties it back to the termination clause in the ToS.

## Beta, evaluation, and trial agreements

Before a product is generally available — or before a customer commits — special short-term agreements apply. **Beta (or evaluation/trial) agreements** cover pre-release or time-limited access, and they differ from normal terms in predictable ways: they grant a **limited, temporary** right to use; they disclaim warranties even more firmly ("provided as-is, may break"); they often add **feedback** and confidentiality clauses (the beta itself may be a secret); and they make clear the provider can change or pull the product at any time. If you're evaluating software, read these for what survives the trial — especially any clause about who owns feedback or what happens to your data when the trial ends.

## Partner, reseller, and developer/API terms

Two more relationship types round out the picture:

- **Partner and reseller agreements** govern others selling or building on your product — resellers, distributors, affiliates, integration partners. They cover who may sell to whom, in what territory, at what margin or referral fee, branding rules, and how the end customer's license is granted (directly by you, or sublicensed through the partner).
- **Developer / API terms** govern third parties that build on your **API** or platform. They set rate limits and acceptable use, restrict scraping and competing uses, address ownership of data passing through, and often reserve the right to change or deprecate the API. If your product exposes an API, these terms are what keep that ecosystem healthy and within bounds.

## How it all fits together with the license

Step back and the whole landscape clicks into place. The **license** is the foundation — it grants the permission to use the software at all, the subject of [what is a software license](/learn/software-licensing/what-is-a-software-license/). Everything in this module clips onto that foundation to govern a *relationship* around the licensed software: terms of service set the rules of use, the privacy policy and DPA govern data, the SLA and support agreement set service quality, the NDA protects what's shared, the MSA/SOW structures a B2B deal, and AUP, beta, partner, and API terms handle specific situations.

| Agreement | What it governs | When you need it |
|---|---|---|
| **License / EULA** | Permission to use the software/copy | Always — it's the foundation |
| **Terms of service** | Rules for using an online service | You run a service/site/app |
| **Privacy policy** | What you do with personal data | You collect personal data (often legally required) |
| **DPA** | A vendor processing data for you | A processor handles personal data on your behalf |
| **SLA** | Availability/performance promises | You sell an ongoing service |
| **Support agreement** | Help: tiers, response, coverage | You offer paid support |
| **NDA** | Protecting shared confidential info | Before sharing secrets with anyone |
| **MSA + SOW** | The legal frame + each deal's scope | Repeat B2B sales/services |
| **AUP** | What users may not do | A multi-user service needs conduct rules |
| **Beta / trial** | Limited pre-release/evaluation use | You offer betas, trials, or evaluations |
| **Partner / reseller** | Others selling/distributing your product | You sell through a channel |
| **Developer / API terms** | Third parties building on your API | You expose an API or platform |

You won't need all of these — most products use a handful. The skill is recognizing which relationships your software actually creates and reaching for the matching agreement. That's exactly what the final module turns into a decision you can make for your own project.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the MSA holds the heavy legal terms negotiated once, and each new project or purchase is a short SOW or order form referencing it." markdown="0">
  <p class="knowledge-check__q">Quick check: a vendor wants to sell you several projects over time without re-negotiating liability and IP each time. What structure fits?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A fresh standalone contract with full legal terms for every single project</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A one-way NDA that also covers pricing and deliverables</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A Master Service Agreement for the heavy terms, plus a short SOW or order form per project</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **NDAs protect confidential information** — one-way or mutual, with a defined scope, carve-outs for public/independent info, a term, and the actual-secrecy practices that back them.
- **MSA + SOW structures repeat B2B deals** — heavy legal terms negotiated once in the MSA, deal specifics in each short SOW or order form.
- **An AUP sets conduct rules** — what users may not do, often split out of the terms of service so it can be updated independently.
- **Beta, partner, and developer terms cover special relationships** — temporary trials, channel sales, and third parties building on your API.
- **Everything clips onto the license** — the license grants permission; these agreements govern the relationships built around the licensed software.

Next up: with the whole agreement landscape in view, the path turns to choosing what *your* project needs. See [A framework for choosing](/learn/software-licensing/the-decision-framework/).
