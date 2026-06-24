---
slug: key-commercial-clauses
title: Key clauses in a commercial license
description: A clause-by-clause guide to commercial software licenses — grant, restrictions, fees, term and termination, warranty, limitation of liability, indemnification, IP, confidentiality, audit, and governing law — with what each does and what to watch for, whether you're writing or signing.
keywords: commercial license clauses, EULA clauses, license grant, limitation of liability, indemnification, warranty disclaimer, term and termination, audit rights, governing law, confidentiality, IP ownership, consequential damages, liability cap, software contract
level: advanced
status: full
faq:
  - q: "What's the single most important clause to read in a commercial license?"
    a: "For most buyers it's the **limitation of liability** clause, because it decides who absorbs the loss when the software causes real damage. Vendors cap their liability (often at fees paid) and exclude consequential damages, which can leave you holding far more than the cap if something goes badly wrong. For vendors, the **license grant and restrictions** matter most, since they define the product being sold. Read both carefully."
  - q: "Why are warranty disclaimers written in ALL CAPS?"
    a: "In the US, the Uniform Commercial Code requires that a disclaimer of implied warranties be **conspicuous** to be enforceable. All-caps (or bold) text is the traditional way to meet that bar. It's not shouting for emphasis — it's a legal formality to make sure the disclaimer can't be dismissed as buried fine print."
  - q: "Can a vendor disclaim *all* liability?"
    a: "Not entirely, and not everywhere. Most jurisdictions don't let you disclaim liability for things like gross negligence, willful misconduct, or death/personal injury, and **consumer-protection law (especially in the EU and UK)** limits how far you can push disclaimers against consumers. A clause that says \"we are never liable for anything\" is often partly unenforceable. The enforceable version uses a **cap** plus targeted **exclusions** — see [Licensing across jurisdictions](/learn/software-licensing/licensing-across-jurisdictions/)."
---
# Key clauses in a commercial license

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Grant + restrictions define the product** — what you may do, and what you may not. **The risk clauses decide who pays** — warranty disclaimer, limitation of liability, and indemnification matter most when things go wrong. **Term, termination, and survival** — what ends the deal and what outlives it. **The operational clauses** — fees, IP, confidentiality, audit rights, and governing law round out the agreement.
</div>

> **This is educational material, not legal advice.** For decisions that carry real risk, consult a qualified attorney.

This is the detailed companion to [Writing a license to sell software](/learn/software-licensing/writing-a-commercial-license/). Here we go clause by clause through a commercial software license: what each clause is for, and what to watch for. It's written to be useful from *both* sides — if you're drafting a license to sell, these are the clauses you'll write; if you're a buyer being handed an agreement, these are the clauses to read before you sign. Readers focused on the *signing* side will also want the [reading-agreements module](/learn/software-licensing/how-to-read-an-agreement/), which covers this from the customer's perspective. By the end you'll recognize every major clause and know its traps.

## The clauses that define the product

### License grant & scope

The **grant** is the heart of the license — the rights the vendor actually gives. A typical grant is "non-exclusive, non-transferable" and bounded by scope (users, devices, installs, territory, purpose). Everything not granted is reserved.

*Watch for:* vague scope ("reasonable use"), and whether the grant is *perpetual* or tied to a paid term. As a buyer, make sure the grant actually covers how you intend to use it (e.g., affiliates, contractors, cloud deployment). As a vendor, make the boundaries explicit so overuse is a clear breach.

### Restrictions

The flip side of the grant: the explicit list of prohibitions — no redistribution, no modification, no reverse engineering (subject to law), no benchmarking/publishing performance results, no competing use.

*Watch for:* the **reverse-engineering** ban, which in the EU is partly overridden by a legal right to decompile for interoperability; and "no benchmarking" clauses, which are common but controversial. Restrictions that contradict mandatory law are unenforceable to that extent.

## The money and the timeline

### Fees & payment

What's owed, when, in what currency, and the consequences of late payment (interest, suspension, termination). For subscriptions, this is also where **auto-renewal** and **price-increase** terms live.

*Watch for:* automatic renewal with a short cancellation window, and uncapped annual price increases. As a buyer, negotiate a renewal cap; as a vendor, be clear and conspicuous about renewals (some jurisdictions require it).

### Term & termination — and what survives

The **term** is how long the agreement lasts; **termination** covers how it ends — for convenience, for breach (often with a cure period), or automatically on non-payment. Critically, a **survival** clause lists which obligations continue *after* termination — typically confidentiality, limitation of liability, IP ownership, and accrued fees.

*Watch for:* what happens to your access and **your data** on termination (especially SaaS — is there an export window?), and whether the vendor can terminate "for convenience" on short notice. The survival list is easy to overlook and very important: a license with no survival clause may let obligations evaporate at the worst moment.

## The risk clauses: who pays when it breaks

These three clauses, together, allocate the financial risk of the deal. They are the most negotiated part of any serious contract and the most important to read.

### Warranty — and the all-caps disclaimer

A **warranty** is a promise about the software (e.g., "will perform substantially per the documentation for 90 days"). Almost every commercial license then **disclaims all other warranties** — the implied warranties of *merchantability* and *fitness for a particular purpose* that the law would otherwise read in. This disclaimer is written in ALL CAPS or bold because, under the US Uniform Commercial Code, a disclaimer of implied warranties must be **conspicuous** to be enforceable.

```text
EXCEPT AS EXPRESSLY SET FORTH ABOVE, THE SOFTWARE IS PROVIDED "AS IS"
WITHOUT WARRANTY OF ANY KIND, AND VENDOR DISCLAIMS ALL IMPLIED WARRANTIES,
INCLUDING THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A
PARTICULAR PURPOSE.
```

*Watch for:* "AS IS" with essentially no express warranty at all. As a buyer of critical software, push for at least a basic performance warranty.

### Limitation of liability

The clause that decides **how much** a party can be made to pay if the other suffers a loss. It has two standard parts: a **cap** (often "fees paid in the prior 12 months") and an **exclusion of consequential damages** (lost profits, lost data, business interruption).

*Watch for:* a cap far below your potential exposure, and remember the important limit — **some jurisdictions restrict how far you can disclaim liability**. You generally cannot exclude liability for gross negligence, willful misconduct, or death/personal injury, and **consumer and EU/UK law** further constrains disclaimers against consumers. A clause claiming zero liability for everything is often partly void. Common carve-outs sit *above* the cap: IP indemnity, breach of confidentiality, and the customer's payment obligations.

### Indemnification

An **indemnity** is a promise to defend and cover the other party against certain third-party claims. The classic one runs from vendor to customer: if a third party sues the customer claiming the software infringes their patent or copyright, the **vendor defends and pays**. Customers often give a narrower indemnity back (e.g., for misuse of the software).

*Watch for:* whether the **IP indemnity** exists at all (some vendors omit it), its exceptions (modifications, combinations, open-source components), and the vendor's remedies if a claim succeeds (repair, replace, or refund). For software built on open source, the indemnity interacts with your dependency hygiene — see [Auditing dependencies & SBOMs](/learn/software-licensing/auditing-dependencies/).

## The operational and legal clauses

### IP ownership

A plain statement that the vendor **retains all intellectual property** in the software; the customer gets only the license granted. For SaaS, this clause should also address ownership of **customer data** (the customer keeps it) and any **feedback** (often assigned to the vendor).

*Watch for:* overbroad assignment of customer data or work product to the vendor.

### Confidentiality

Defines what each side must keep secret (pricing, the software's non-public details, business information), with standard carve-outs (already public, independently developed, legally compelled). Often mirrors a separate [NDA](/learn/software-licensing/nda-and-other-agreements/).

*Watch for:* the duration of the obligation and whether trade secrets are protected indefinitely.

### Audit rights

A vendor right to **inspect** the customer's usage to confirm compliance with the license counts (seats, cores, instances). Common in enterprise software and a frequent source of friction.

*Watch for:* the notice period, frequency, and who pays for the audit. As a buyer, bound it (reasonable notice, once a year, vendor pays unless material under-licensing is found).

### Governing law & dispute resolution

Whose law applies (the **governing law**) and how/where disputes are resolved — courts in a named jurisdiction, or **arbitration**. This clause quietly decides how expensive and how convenient any future fight will be.

*Watch for:* an inconvenient forum (the other side's home turf far from you) and mandatory arbitration that waives your right to court or class actions. This is heavily jurisdiction-dependent — see [Licensing across jurisdictions](/learn/software-licensing/licensing-across-jurisdictions/).

## Quick reference

| Clause | What it does | Watch for |
|---|---|---|
| **License grant & scope** | The rights you get | Vague scope; perpetual vs term |
| **Restrictions** | What you may not do | Reverse-engineering / benchmarking bans vs mandatory law |
| **Fees & payment** | What's owed and when | Auto-renewal; uncapped price hikes |
| **Term & termination + survival** | How it ends; what continues | Data export on termination; "for convenience" exits |
| **Warranty + disclaimer** | Promises, and the all-caps "AS IS" | No real warranty at all |
| **Limitation of liability** | How much can be owed | Low cap; over-broad disclaimer that law won't enforce |
| **Indemnification** | Who defends third-party claims | Missing or narrow IP indemnity; exclusions |
| **IP ownership** | Vendor keeps the IP | Grabs on customer data/feedback |
| **Confidentiality** | What stays secret | Duration; trade-secret handling |
| **Audit rights** | Vendor can verify usage | Notice, frequency, who pays |
| **Governing law & disputes** | Whose law, where, how | Inconvenient forum; forced arbitration |

<div class="knowledge-check" data-quiz data-correct-msg="Right — warranty disclaimers are in all caps because US law requires the disclaimer of implied warranties to be conspicuous to be enforceable." markdown="0">
  <p class="knowledge-check__q">Quick check: why is the warranty disclaimer typically written in ALL CAPS?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It's a formatting convention with no legal meaning</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It makes the clause apply in more countries automatically</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">US law requires a disclaimer of implied warranties to be conspicuous to be enforceable</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Grant & restrictions define the product** — a scoped grant plus an explicit list of prohibitions; check the scope actually fits your use.
- **The risk clauses allocate loss** — the warranty disclaimer (all-caps for conspicuousness), the limitation of liability (cap + consequential-damages exclusion), and indemnification (who defends IP claims) are the most important to read.
- **You can't disclaim everything** — gross negligence, willful misconduct, and consumer/EU protections limit how far liability can be excluded.
- **Term, termination, and survival** — know what ends the deal, what happens to your data, and which obligations survive.
- **Operational clauses round it out** — fees and renewal, IP ownership, confidentiality, audit rights, and governing law/dispute resolution.

Next up: when DIY is fine, when you genuinely need a lawyer, and the reputable resources for both. See [Where to get licensing help](/learn/software-licensing/where-to-get-licensing-help/).
