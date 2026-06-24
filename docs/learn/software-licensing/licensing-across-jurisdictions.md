---
slug: licensing-across-jurisdictions
title: Licensing across jurisdictions
description: Software licensing law is not uniform worldwide. Learn how moral rights, registration, warranty disclaimers, the license-vs-contract question, governing law, and data-protection regimes differ across the US, EU, and beyond.
keywords: software licensing jurisdictions, moral rights, copyright registration international, warranty disclaimer enforceability, governing law clause, choice of law, GDPR CCPA, license vs contract international, Berne Convention, shipping software worldwide
level: intermediate
status: full
faq:
  - q: "Can I just disclaim all warranties and liability the way open-source licenses do?"
    a: "In the US, broad **\"AS IS\" disclaimers** are widely upheld, which is why MIT, Apache, and GPL all shout them in capitals. But many consumer-protection regimes — notably in the **EU and UK** — limit how far you can disclaim warranties or liability *to consumers*, and some such clauses are void no matter what the license says. Disclaimers are strong defaults, not a guarantee everywhere."
  - q: "What does a 'governing law' clause actually do?"
    a: "A **governing-law** (choice-of-law) clause states which jurisdiction's law interprets the agreement, and a paired **forum** clause says where disputes are heard. They add predictability, but they're not absolute — courts can override them, especially when **mandatory consumer-protection or data-protection law** in the user's country applies regardless of what the contract picked."
  - q: "Are moral rights a real concern for software?"
    a: "They can be. **Moral rights** — like the right to be credited as author and to object to derogatory treatment of the work — are strong in much of the **EU** (France and Germany especially) and often **can't be fully waived**. In the **US** they're weak for software. For most code it rarely bites, but it's a reason a blanket 'you waive all rights' clause may not hold everywhere."
---
# Licensing across jurisdictions

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**US law is our baseline, not the world's** — core copyright is global via treaty, but the details diverge. **Moral rights differ sharply** — strong and often unwaivable in much of the EU, weak in the US. **Disclaimers aren't bulletproof** — consumer-protection regimes can limit warranty and liability waivers. **Governing-law clauses help but don't override mandatory local law** — especially data protection like GDPR and CCPA.
</div>

Throughout this path we've used the **United States as the reference jurisdiction** and flagged differences in passing. This is the lesson where we slow down and treat those differences properly, because software ships across borders by default — a license written in California binds a user in Berlin, São Paulo, or Tokyo, and not every clause survives the trip. By the end you'll know the main axes on which licensing law varies and how to ship software worldwide without nasty surprises.

> **This is educational material, not legal advice.** For decisions that carry real risk, consult a qualified attorney.

## The shared baseline: copyright is global

Start with the good news, because it's substantial. Thanks to the **Berne Convention** and related treaties, the *core* of copyright is remarkably consistent worldwide:

- Copyright is **automatic on creation** in essentially every country — no registration required (the rule from [the copyright lesson](/learn/software-licensing/copyright-and-software-ownership/)).
- Foreign works get **national treatment** — a US program is protected in France roughly as a French one would be, and vice versa.
- The basic exclusive rights (copy, modify, distribute) exist broadly.

So your copyright follows your code abroad without you doing anything special. The divergence is in the *details* of ownership, what you can disclaim, how agreements are enforced, and — increasingly the biggest axis of all — what you must do with users' data. The rest of this lesson walks those axes.

## Moral rights: strong in Europe, weak in the US

This is one of the sharpest divides. **Moral rights** are rights that stay with the *human author* personally, separate from the economic copyright that can be sold or licensed. The main ones are:

- **Attribution** — the right to be named as the author.
- **Integrity** — the right to object to distortion or derogatory treatment of the work.

In much of the **EU — France and Germany especially** — moral rights are strong and, importantly, often **cannot be fully waived or assigned**, even by contract. In the **US**, moral rights are narrow (largely confined to certain visual art) and effectively weak for software; you can generally waive them.

The practical fallout: a "you irrevocably waive all rights, including moral rights" clause that's routine in a US contributor agreement may be **partly unenforceable** in a country where moral rights can't be waived. For most software this rarely surfaces, but it's a real reason blanket waivers don't always travel, and a consideration when you take contributions or contractor work from authors in moral-rights jurisdictions.

## Copyright registration: a US norm, not a global one

We noted this in [lesson two](/learn/software-licensing/copyright-and-software-ownership/); it's worth restating as a jurisdiction axis. In the **US**, registration is optional but valuable — a prerequisite to suing and the gateway to **statutory damages** and attorney's fees. Many other countries have **no registration system at all**; rights are automatic and that's it.

So "register your copyright" is sound US-specific advice and a non-sequitur elsewhere. If you operate internationally, treat registration as a US enforcement tactic, not a universal step.

## Warranty and liability disclaimers: how far they hold

Look at almost any open-source license and you'll find SHOUTING CAPITALS like:

```text
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND ...
IN NO EVENT SHALL THE AUTHORS BE LIABLE FOR ANY CLAIM, DAMAGES ...
```

These **warranty disclaimers** and **liability limitations** are central to how licensors protect themselves. In the **US**, they're broadly enforceable (the caps-lock convention exists because US commercial law historically wanted disclaimers to be "conspicuous").

But their reach varies a lot:

- In the **EU and UK**, **consumer-protection law** limits how far you can disclaim warranties or exclude liability *to consumers*. Some exclusions — for instance, excluding liability for death or personal injury caused by negligence, or stripping a consumer's statutory rights — are **void regardless of what the agreement says**.
- The **business-to-business vs business-to-consumer** distinction is critical: a disclaimer fully effective between two companies may be unenforceable against an individual consumer in the same jurisdiction.

So treat "AS IS" as a strong default that protects you well in the US and in B2B contexts, but *not* as an absolute shield against consumer claims everywhere.

## "License vs contract" varies by country

We met this question in [License vs contract](/learn/software-licensing/license-vs-contract/). Its answer is itself jurisdiction-dependent. Whether an open-source license is treated as a **bare license** (violation = copyright infringement) or as a **contract** (violation = breach), and how things like consideration and assent are evaluated, differs across legal traditions:

- **Common-law systems** (US, UK) lean on doctrines like consideration and have generated the case law treating open-source conditions as enforceable, sometimes as both license and contract.
- **Civil-law systems** (much of Europe) frame contract formation differently and may analyze the same license through a different lens.

The upshot: the *enforceability mechanics* of the very same license text can differ depending on where a dispute lands — another reason the next axis matters.

## Governing law and forum-selection clauses

Because the same words can be read differently in different places, well-drafted agreements try to pin down *where* and *under whose law* they'll be interpreted:

- A **governing-law (choice-of-law)** clause: "This agreement is governed by the laws of [State/Country]."
- A **forum-selection (jurisdiction)** clause: "Disputes shall be resolved in the courts of [place]."

These add valuable predictability. But they are **not absolute**:

- Courts can **decline to enforce** them, particularly against consumers, or where the chosen forum is wildly unfair.
- **Mandatory local law** can apply *regardless* of the clause. A consumer's home country's consumer-protection rules — and, crucially, its data-protection law — often apply no matter which law the contract picked. You can't contract your way out of GDPR by choosing US law.

| Axis | United States | Much of EU / UK |
|---|---|---|
| **Moral rights** | Weak for software; waivable | Strong (FR, DE); often not fully waivable |
| **Copyright registration** | Optional but unlocks statutory damages | Often no system; rights automatic |
| **Consumer warranty/liability disclaimers** | Broadly enforceable if conspicuous | Limited by consumer-protection law; some void |
| **Governing-law clause respected?** | Generally, with limits | Yes, but mandatory consumer/data law overrides |

## Data-protection law: a separate axis entirely

Here's the axis that has grown from a footnote into a headline. Even if your *licensing* is perfectly sorted, if your software **collects or processes personal data**, you face a whole separate body of law that has nothing to do with copyright:

- **GDPR** (EU) — sweeping rules on processing personal data, with extraterritorial reach (it can apply to you because of *where your users are*, not where *you* are) and large fines.
- **CCPA/CPRA** (California) and a growing patchwork of US **state** privacy laws.
- Many others — the UK, Brazil (LGPD), Canada, and more, each with their own requirements.

Two things make this dangerous to overlook. First, these laws often apply based on the **user's location**, so a small team shipping worldwide can be on the hook for GDPR. Second, they are largely **mandatory** — a governing-law clause won't opt you out. This deserves real attention of its own; we cover it in [Privacy policies & data agreements](/learn/software-licensing/privacy-and-data-agreements/).

## Practical advice for shipping worldwide

You don't need to master every country's law to ship responsibly. A few pragmatic habits cover most of the ground:

- **Use well-known, vetted licenses.** Standard open-source licenses (MIT, Apache, GPL) were drafted with international use in mind and behave predictably across borders. A bespoke license is a bespoke problem in every country.
- **Don't rely on a single clause to do impossible work.** Assume your most aggressive disclaimers and waivers may be trimmed for consumers in some places, and don't build your business on them holding everywhere.
- **Include sensible governing-law and forum clauses** — they help — but know they bow to mandatory local consumer and data law.
- **Treat data protection as a first-class, separate workstream.** If you touch personal data, scope your obligations by where your *users* are, not just where you are.
- **Get local advice for real exposure.** When you're selling at scale into a specific country or taking on real risk, a local attorney is cheap insurance — exactly the situation the disclaimer at the top of this lesson is about.

<div class="knowledge-check" data-quiz data-correct-msg="Right — mandatory local law, especially consumer-protection and data-protection rules, can apply regardless of a governing-law clause." markdown="0">
  <p class="knowledge-check__q">Quick check: your EULA says it's governed by US law. Can that clause keep EU data-protection rules from applying to your EU users?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Yes — a governing-law clause overrides any other country's law for everyone</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">No — mandatory local law like GDPR can apply based on where users are, regardless of the clause</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Only if the user explicitly opts out of US law in writing first</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Copyright's core is global** — automatic protection and national treatment hold almost everywhere via the Berne Convention; the *details* diverge.
- **Moral rights differ sharply** — strong and often unwaivable in much of the EU, weak for software in the US, so blanket waivers don't always travel.
- **Registration is a US norm** — valuable there for statutory damages, but absent in many countries.
- **Disclaimers aren't absolute** — broadly enforceable in the US and B2B, but consumer-protection law (EU/UK) can limit or void them.
- **Governing-law clauses help but bow to mandatory local law** — especially consumer-protection and data-protection rules.
- **Data protection is a separate axis** — GDPR, CCPA, and others apply by user location and can't be contracted away; treat it as its own workstream.

Next up: we leave the foundations and dig into what "open source" actually requires. See [What open source really means](/learn/software-licensing/what-is-open-source/).
