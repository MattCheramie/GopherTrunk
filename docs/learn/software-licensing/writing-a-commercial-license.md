---
slug: writing-a-commercial-license
title: Writing a license to sell software
description: What to decide before you draft a commercial software license or EULA — the rights you're granting, the skeleton of the document, how a license relates to a purchase order or MSA, and why you should start from a vetted template or lawyer, not a blank page.
keywords: writing a commercial license, software EULA, license drafting, license grant, end user license agreement, MSA, master services agreement, purchase order, license template, SaaS terms, software contract structure, sell software license
level: advanced
status: full
faq:
  - q: "Can I just write my own EULA from scratch?"
    a: "You *can*, but for anything beyond a hobby project you shouldn't. License language has decades of case law behind specific phrasings, and a homemade clause can be unenforceable or accidentally give away more than you meant. Start from a **vetted template** or have a lawyer draft or review it — see [Where to get licensing help](/learn/software-licensing/where-to-get-licensing-help/). Writing from a blank page is where expensive mistakes get baked in."
  - q: "What's the difference between a license, a purchase order, and an MSA?"
    a: "The **license** (EULA) defines what the customer may *do* with the software. A **purchase order (PO)** is the commercial transaction — what's being bought, quantity, price. A **Master Services Agreement (MSA)** is an overarching contract governing an ongoing business relationship, with the license and specific deals as schedules under it. Big enterprise sales often use all three; a simple product may use only a EULA."
  - q: "Is this lesson legal advice?"
    a: "No. **This is educational material, not legal advice.** It explains the structure and decisions involved so you can have a productive conversation with a qualified attorney. For an actual license you intend to sell under, get professional review — the cost of a lawyer is small next to the cost of an unenforceable or overreaching agreement."
---
# Writing a license to sell software

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Decide before you draft** — what you're granting, to how many, whether they can modify or transfer, and what's prohibited. **Know the skeleton** — grant, definitions, restrictions, fees, term/termination, warranty, liability, IP, governing law. **License vs PO vs MSA** — the license says what they may *do*; other documents handle the *deal* and the *relationship*. **Don't start from a blank page** — use a vetted template or a lawyer.
</div>

> **This is educational material, not legal advice.** For decisions that carry real risk, consult a qualified attorney.

If you're going to sell proprietary software, at some point you need a document that says what your customers are allowed to do with it. This lesson is about *preparing* to create that document — the decisions you make before any drafting, the standard structure a commercial license follows, and how the license fits alongside the other paperwork of a sale. By the end you'll know what you need to settle up front, what a EULA's skeleton looks like, how the license relates to a purchase order and an MSA, and — most importantly — why you should not write one from scratch.

## Decide what you're granting before you draft

The biggest mistake in license drafting is opening a blank document and starting to write clauses. The clauses are downstream. First, settle the **business decisions** that the license merely *records*. Get these wrong and even perfect legal language ships the wrong product.

Work through these questions before anything else:

| Decision | The question to answer |
|---|---|
| **Duration** | Perpetual or subscription? (See [commercial license models](/learn/software-licensing/commercial-license-models/).) |
| **Scope of use** | How many users, devices, installs, cores, or calls? Counted how? |
| **Modification** | May the customer modify or configure the software, or not at all? |
| **Transfer** | Can they assign or resell the license, or is it locked to them? |
| **Redistribution** | May they embed or redistribute it, or is it for internal use only? |
| **Prohibited uses** | What's flatly banned — reverse engineering, benchmarking, competing use? |
| **Support & updates** | Are these included, separate, or excluded? |
| **Data & hosting** | For SaaS: who owns the data, where does it live, what happens at termination? |

Notice that every one of these is a *product and pricing* decision, not a legal one. The license is the place you write the answers down precisely. If you can't answer them, you're not ready to draft — and no template will answer them for you. The [decision framework](/learn/software-licensing/the-decision-framework/) lesson helps work through them systematically.

## The skeleton of a commercial license / EULA

Commercial licenses are remarkably consistent in structure, because the same things have to be covered every time. Here's the standard skeleton, in roughly the order you'll see it:

1. **Parties** — who's licensing to whom (the licensor and the licensee).
2. **Definitions** — the precise meaning of key terms ("Software," "Documentation," "Authorized Users," "Order"). Boring but load-bearing; ambiguity here causes most disputes.
3. **License grant** — the rights you *are* giving, with their scope and limits.
4. **Restrictions** — what the customer may *not* do.
5. **Fees & payment** — what's owed, when, and what happens if it isn't paid.
6. **Term & termination** — how long it lasts, what ends it, and what survives.
7. **Warranty / disclaimer** — any promises, and the disclaimer of everything else.
8. **Limitation of liability** — caps and exclusions on what you'll pay if things go wrong.
9. **Intellectual property** — a clear statement that you keep ownership.
10. **Governing law & dispute resolution** — whose law applies and where disputes are settled.

A minimal grant-and-restriction core looks like this — the heart of the document, even when everything around it grows:

```text
1. LICENSE GRANT. Subject to the terms of this Agreement and payment of
   the applicable Fees, Licensor grants Licensee a non-exclusive,
   non-transferable license to install and use the Software for up to the
   number of Authorized Users specified in the applicable Order, solely
   for Licensee's internal business purposes.

2. RESTRICTIONS. Licensee shall not (a) sublicense, sell, rent, or
   redistribute the Software; (b) modify or create derivative works;
   (c) reverse engineer, decompile, or disassemble it except to the
   extent that applicable law expressly permits; or (d) remove any
   proprietary notices.
```

We walk through each of these clauses — what it does, what to put in it, and what to watch for — in [Key clauses in a commercial license](/learn/software-licensing/key-commercial-clauses/). Treat that lesson as the detailed companion to this structural overview.

## How the license relates to the PO and the MSA

A common source of confusion: the license is usually *not the only document* in a real sale. Three pieces of paper do three different jobs, and keeping them straight matters.

- **The license / EULA** answers *"what may the customer do with the software?"* — the grant, restrictions, and the legal terms above.
- **The purchase order (PO) or order form** answers *"what exactly is being bought?"* — which product, how many seats, the price, the dates. It's the commercial transaction.
- **The Master Services Agreement (MSA)** answers *"what governs the overall relationship?"* — the umbrella contract for an ongoing engagement, under which individual orders and license terms hang as schedules or exhibits.

For a small product sold off a website, a single EULA accepted at install or checkout may be all you have. For an enterprise deal, you'll typically have an MSA that incorporates the license terms by reference, with one or more order forms specifying the actual purchase. The key drafting principle is to avoid **conflicts** between them: state clearly which document controls if they disagree (usually the MSA, with order-specific details winning on quantity and price).

| Document | Answers | When you need it |
|---|---|---|
| **License / EULA** | What can the customer do with the software? | Always |
| **Purchase order / order form** | What is being bought, how much, for how long? | Any paid transaction |
| **MSA** | What governs the ongoing relationship? | Larger / recurring B2B deals |

## Start from a template or a lawyer — not a blank page

This is the strongest practical advice in the whole lesson: **do not draft a commercial license from scratch.**

License language is dense with terms of art whose exact wording has been tested in court. A homegrown limitation-of-liability clause might be unenforceable; a sloppy grant might accidentally be broader than you intended; a missing "survives termination" line might let a customer keep using your IP after the deal ends. These aren't hypotheticals — they're the everyday failure modes of DIY legal drafting.

Two good starting points, in increasing order of safety:

- **A vetted template.** Reputable, professionally prepared templates encode the standard structure and battle-tested language. They still need *your* business decisions filled in, and ideally a lawyer's once-over, but they save you from inventing clauses. Be wary of random templates of unknown provenance — see the discussion of template risks in [Where to get licensing help](/learn/software-licensing/where-to-get-licensing-help/).
- **A qualified attorney.** For anything that carries real risk — selling at scale, enterprise deals, custom terms, raising money — have a lawyer draft or review the license. The cost is small compared to a dispute over an unenforceable agreement, and a good lawyer will also catch jurisdiction-specific issues you won't know to look for.

The judgment call of *when a template is fine versus when you genuinely need a lawyer* is exactly what the next module-mate lesson, [Where to get licensing help](/learn/software-licensing/where-to-get-licensing-help/), is built to answer.

<div class="knowledge-check" data-quiz data-correct-msg="Right — settle the business decisions (duration, scope, modification, transfer, prohibitions) first; the license just records them." markdown="0">
  <p class="knowledge-check__q">Quick check: what should you do *first* when creating a license to sell your software?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Open a blank document and start writing the legal clauses</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Decide the business terms — duration, scope, modification, transfer, and prohibitions</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Pick a governing-law jurisdiction before anything else</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Decide before you draft** — duration, scope, modification, transfer, redistribution, prohibited uses, support, and data are business decisions the license merely records.
- **Know the skeleton** — parties, definitions, grant, restrictions, fees, term/termination, warranty, liability, IP, and governing law, in roughly that order.
- **The grant and restrictions are the core** — a tightly scoped grant followed by an explicit list of prohibitions.
- **License vs PO vs MSA** — the license says what they may *do*, the order says what's *bought*, the MSA governs the *relationship*; state which controls in a conflict.
- **Don't start from a blank page** — use a vetted template or a lawyer; DIY drafting is where unenforceable and overbroad clauses get baked in.

Next up: the detailed companion — every major clause in a commercial license, what it does, and what to watch for. See [Key clauses in a commercial license](/learn/software-licensing/key-commercial-clauses/).
