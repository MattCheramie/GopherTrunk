---
slug: which-agreements-do-you-need
title: Which agreements does your software need?
description: Beyond the license, the full set of agreements your software needs is determined by how it's delivered and what data it touches — a decision table mapping scenarios to EULA, ToS, privacy policy, DPA, SLA, CLA, NDA, and API terms.
keywords: software agreements needed, EULA vs terms of service, privacy policy required, data processing agreement, SLA, CLA DCO, NDA, API terms of use, SaaS legal documents, which agreements does my software need, delivery model agreements
level: intermediate
status: full
faq:
  - q: "Isn't the license enough? Why do I need other agreements?"
    a: "The license governs *who may use and copy your code*. Everything else — the terms of the service, what you do with users' data, uptime promises, contributions, confidential discussions — needs its own agreement. A downloaded app needs a EULA; a hosted service needs terms of service, a privacy policy, and often more. The license is one document in a set."
  - q: "Do I need a privacy policy?"
    a: "If your software collects or processes **personal data** — names, emails, locations, accounts, even IP addresses in some jurisdictions — then yes, you almost certainly do, and in the EU (GDPR) and California (CCPA) it's a legal requirement. You'll also need a [data processing agreement](/learn/software-licensing/privacy-and-data-agreements/) with any vendor that handles that data on your behalf."
  - q: "What's the difference between a EULA and terms of service?"
    a: "A **EULA** licenses a copy of software the user installs and runs on their own machine. **Terms of service** govern access to a hosted service the user doesn't possess. Downloaded software → EULA; web app or SaaS → ToS. Some products need both. See [EULAs & terms of service](/learn/software-licensing/eula-and-tos/)."
---
# Which agreements does your software need?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The license is only one document** — it covers your code; other agreements cover delivery, data, uptime, contributions, and confidentiality. **Delivery decides the core agreement** — downloaded → EULA; hosted → terms of service. **Data decides the privacy stack** — personal data means a privacy policy, and often a DPA with vendors and GDPR/CCPA duties. **Add-ons follow features** — uptime promises → SLA; contributions → CLA/DCO; an API → developer terms; confidential talks → NDA.
</div>

You've chosen a [license](/learn/software-licensing/choose-your-license/) — but a license only answers *who may use and copy your code*. The rest of your legal footprint is decided by two things: **how the software is delivered** and **what data it touches**. This lesson turns those into a concrete agreement set. We'll work through each agreement, give you a decision table that maps scenarios to the documents they require, and point back to the earlier lessons that cover each one in depth. By the end you'll be able to list exactly which agreements your specific product needs — no more, no less.

## The two questions that drive the set

Just as the [decision framework](/learn/software-licensing/the-decision-framework/) used delivery and data as constraints, the agreement set falls out of the same two questions:

1. **How does the software reach users?** Downloaded and installed, hosted as a service, embedded in something else, or exposed as an API. This decides your *core* agreement.
2. **What does it touch and promise?** Personal data, confidential information, uptime commitments, outside contributions. Each of these adds a specific document.

Answer both and you have your list. Let's walk the agreements one at a time, then assemble the table.

## Delivery-driven agreements

These are decided by *how* the software is delivered — the core of your set.

### Downloaded or installed software → EULA

If users download and run a copy on their own machine, you grant them permission to use that copy through an **End-User License Agreement (EULA)**. It sets what they may and may not do (install on N machines, no reverse-engineering), disclaims warranties, and limits your liability. This is the natural fit for a paid desktop app or any installed binary. Full treatment in [EULAs & terms of service](/learn/software-licensing/eula-and-tos/).

> **This is educational material, not legal advice.** For decisions that carry real risk, consult a qualified attorney.

### Web app or SaaS → Terms of service (+ more)

If users access a hosted service they don't possess, a EULA is the wrong instrument — there's no copy to license. Instead you publish **Terms of Service (ToS)** governing acceptable use, accounts, payment, and termination of the service relationship. A hosted service almost always also needs a **privacy policy** (see below), and if you're selling to businesses, a **subscription agreement** or **Master Services Agreement (MSA)** spelling out the commercial terms.

### Embedded or bundled → passthrough and component compliance

If your software is embedded in a larger product or bundled with others' components, two things apply: you must comply with the licenses of the components you include (attribution, source offers — see [permissive compliance](/learn/software-licensing/permissive-compliance/)), and you often need *passthrough* terms so the obligations flow correctly to your downstream users.

### An API → developer / API terms

If you expose an API, you need **developer terms** (sometimes called API terms of use) covering rate limits, acceptable use, data ownership, and the right to change or deprecate endpoints. These are distinct from your end-user ToS because the audience — developers building on you — has different rights and risks.

## Data- and feature-driven agreements

These are decided by *what the software touches or promises*, and they stack on top of the core agreement.

### Handles personal data → privacy policy (+ DPA, + GDPR/CCPA)

If your software collects or processes personal data, you need a **privacy policy** that discloses what you collect, why, and who you share it with. In the EU this is mandatory under the **GDPR**; in California, under the **CCPA/CPRA**. And whenever a third-party vendor processes that data on your behalf (your cloud host, analytics, email sender), you need a **Data Processing Agreement (DPA)** with them. All of this is covered in [privacy policies & data agreements](/learn/software-licensing/privacy-and-data-agreements/).

### Makes uptime promises → SLA

If you commit to a level of availability or support — "99.9% uptime," "responses within 4 hours" — that promise belongs in a **Service Level Agreement (SLA)**, with the remedies (usually service credits) if you miss it. Enterprise buyers expect one; consumer products often skip it. See [SLAs & support agreements](/learn/software-licensing/sla-and-support-agreements/).

### Accepts outside contributions → CLA or DCO

If people contribute code to your project, you need clarity on the rights to that code. A **Contributor License Agreement (CLA)** has contributors grant you explicit rights; a lighter-weight **Developer Certificate of Origin (DCO)** has them certify they wrote it and may submit it. Either prevents nasty ownership surprises later — especially if you ever want to relicense. See [contributor agreements](/learn/software-licensing/contributor-agreements/).

### Shares confidential information → NDA

If you'll exchange confidential information — with a contractor, partner, or beta customer — before or alongside other agreements, a **Non-Disclosure Agreement (NDA)** protects it. NDAs sit beside the rest of your stack rather than replacing any of it. See [NDAs & the rest of the landscape](/learn/software-licensing/nda-and-other-agreements/).

## The decision table

Here's the whole picture as a lookup. Find the rows that describe your product and union their agreements — most real products match several rows at once.

| If your software… | You need |
|---|---|
| Is **downloaded and installed** | EULA |
| Is a **web app / SaaS** | Terms of service + privacy policy |
| Is **SaaS sold to businesses** | + subscription agreement / MSA + DPA |
| **Handles personal data** | Privacy policy (+ GDPR/CCPA duties, + DPA with vendors) |
| **Makes uptime/support promises** | SLA |
| **Accepts outside contributions** | CLA or DCO |
| **Shares confidential info** | NDA |
| **Exposes an API** | Developer / API terms |
| Is **embedded / bundled** | Component-license compliance + passthrough terms |

Two reminders as you assemble. First, these *combine*: a business SaaS that handles personal data, promises uptime, accepts contributions, and has an API needs a ToS, privacy policy, DPA, subscription/MSA, SLA, CLA/DCO, and API terms — all of it. Second, the license you chose still sits underneath the whole stack; these agreements layer *on top of* it, they don't replace it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a hosted service the user never installs is governed by terms of service, not a EULA, and it almost always needs a privacy policy too." markdown="0">
  <p class="knowledge-check__q">Quick check: your product is a hosted web app users access in a browser. Which core agreement governs it?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A EULA, the same as a downloaded desktop app</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Terms of service, because the user never installs a copy</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Only an open-source license — no other agreement is needed</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **The license is one document in a set** — it covers your code; delivery and data decide the rest.
- **Delivery picks the core agreement** — downloaded → EULA; hosted → terms of service; embedded → component compliance; API → developer terms.
- **Personal data triggers the privacy stack** — a privacy policy, GDPR/CCPA obligations, and a DPA with any vendor that processes data for you.
- **Features add specific documents** — uptime promises → SLA; contributions → CLA/DCO; confidential sharing → NDA.
- **Agreements combine** — most real products match several rows of the table at once, and the stack layers on top of the license.

Next up: see the whole framework, license choice, and agreement set applied end-to-end to a realistic product in [Putting it all together](/learn/software-licensing/putting-it-all-together/).
