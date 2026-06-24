---
slug: risk-clauses
title: "The risk clauses: who pays when it breaks"
description: The clauses that decide liability — warranties versus AS-IS disclaimers, limitation-of-liability caps, exclusion of indirect and consequential damages, indemnification, insurance, and force majeure. Where the real money risk lives and what gets negotiated.
keywords: limitation of liability, liability cap, AS IS disclaimer, disclaimer of warranties, implied warranties, indemnification, indemnity, consequential damages, indirect damages, lost profits, force majeure, insurance clause, third-party IP claims, risk allocation
level: intermediate
status: full
faq:
  - q: "What does a liability cap actually limit?"
    a: "It sets a **maximum total amount** one party can be made to pay the other if things go wrong. A very common cap is the **fees you paid in the prior 12 months** — so if you paid $10,000, that's roughly the most you could recover even from a serious failure. Caps are usually paired with an **exclusion of indirect damages**, which removes lost profits and consequential losses on top. Read both together to see your true exposure."
  - q: "What's the difference between a liability cap and an indemnity?"
    a: "A **limitation of liability** caps what the parties owe *each other* under the contract. An **indemnity** is a promise to cover a *third party's* claim — for example, the vendor agreeing to defend you if someone sues claiming the software infringes their patent. Indemnities are often **carved out of the cap**, because the whole point is to make that protection meaningful, so they can be the largest real exposure in the deal."
  - q: "Can a vendor just disclaim all liability with an 'AS IS' clause?"
    a: "Largely in the US for business deals, yes — courts broadly uphold conspicuous warranty disclaimers and liability limits between sophisticated parties. But not without limits: many jurisdictions **void overly broad disclaimers**, especially against consumers. EU and UK consumer law, for instance, won't let a seller disclaim core protections, and most systems won't allow you to disclaim liability for things like gross negligence, fraud, or death and personal injury."
---
# The risk clauses: who pays when it breaks

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Warranties vs "AS IS"** — a warranty is a promise about the software; "AS IS" disclaims those promises. **Liability caps limit the total payout** — often capped at fees paid, with indirect damages excluded entirely. **Indemnity covers third-party claims** — crucial when you embed others' code. **This is where the real money risk lives** — and why these clauses are the most negotiated in B2B deals. **Disclaimers have legal limits** — consumer and other law can void overly broad ones.
</div>

The clauses we covered last lesson define the deal; the **risk clauses** decide what happens when the deal goes wrong. They allocate who bears the cost of a bug, an outage, a security breach, or a lawsuit — and they're written, almost always, to push that cost away from the vendor and toward you. This lesson decodes the major risk-allocation clauses so you can see your real exposure, understand why these are the most fought-over terms in any serious contract, and know where the law won't let a clause go too far. By the end you'll read the back half of a contract — the part most people's eyes glaze over — as the most important part.

> **This is educational material, not legal advice.** For decisions that carry real risk, consult a qualified attorney.

## Warranties vs "AS IS"

A **warranty** is a promise about the software — that it will perform as documented, that the vendor has the right to license it, that it's free of known malware. Warranties give you something to point to when the product fails.

The flip side is the **disclaimer of warranties**, usually written in shouting capitals:

```text
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE IMPLIED WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, AND NON-INFRINGEMENT.
```

That paragraph does real work. The law would otherwise read certain **implied warranties** into the deal automatically — notably **merchantability** (the goods are fit for ordinary use) and **fitness for a particular purpose** (suitable for the use the seller knew about). The disclaimer strips those out, leaving you with only whatever *express* warranties the contract explicitly grants — often none. The all-caps formatting isn't decoration; the law in many places requires such disclaimers to be **conspicuous** to be effective.

You'll recognize "AS IS" from open-source licenses too — Apache 2.0, MIT, and the GPL all disclaim warranties this way, because the code is given for free with no promises. In a paid commercial deal, by contrast, a blanket "AS IS" is something to push back on.

## Limitation of liability

If a warranty governs *whether* the vendor owes you anything, the **limitation of liability** governs *how much*. It has two moving parts, and you must read both:

### The cap

A **liability cap** sets the maximum total a party can be required to pay. The single most common formula is **the fees paid in the prior 12 months**:

```text
IN NO EVENT SHALL EITHER PARTY'S TOTAL AGGREGATE LIABILITY EXCEED THE FEES
PAID BY CUSTOMER IN THE TWELVE (12) MONTHS PRECEDING THE CLAIM.
```

So if you paid $10,000 last year, $10,000 is roughly the ceiling on what you could recover — even if a vendor failure cost you far more. Caps can also be a fixed dollar figure or a multiple of fees.

### The exclusion of indirect damages

Stacked on top of the cap is an **exclusion of consequential, indirect, and special damages** — and the big one, **lost profits**. Direct damages are the immediate, obvious losses; **consequential** damages are the knock-on losses (the revenue you lost while the service was down, the customers who churned). Excluding them is standard and often removes the *largest* real-world harm from the table.

Read the cap and the exclusion together: a low cap plus a broad exclusion of indirect damages means that even a catastrophic failure may leave you recovering only a fraction of your actual loss.

## Indemnification

**Indemnification** is a promise by one party to **cover a third party's claim** against the other — to defend the lawsuit and pay the resulting costs or damages. The most important kind in software is the **IP indemnity**: the vendor agrees that if someone sues *you* claiming the software infringes their patent or copyright, the vendor will step in and handle it.

This matters enormously when you **embed others' code** in something you ship. If you build a product on third-party components and a patent holder comes after your customers, an indemnity from your suppliers is what stands between you and the bill. (This is one reason the [Apache 2.0](/learn/software-licensing/apache-license/) patent grant is valued, and why [auditing your dependencies](/learn/software-licensing/auditing-dependencies/) matters.)

Two things to check on any indemnity: **what it covers** (IP claims only, or also data breaches and personal injury?) and **whether it's carved out of the liability cap**. A good IP indemnity is usually *uncapped* or has a much higher cap — otherwise the protection is hollow.

## Insurance and force majeure

Two supporting risk clauses round out the picture:

- **Insurance** — larger B2B contracts require each side to carry certain coverage (commercial general liability, cyber, errors-and-omissions) at stated limits. Insurance is what makes an indemnity *collectible* — a promise to pay is only as good as the payer's ability to pay.
- **Force majeure** — excuses a party from performing when something genuinely outside its control intervenes (natural disaster, war, sometimes large-scale infrastructure failure). It pauses obligations rather than ending them. Read what it covers: a clause that sweeps in "any event beyond reasonable control" can be used to excuse more than you'd expect. Note it usually does *not* excuse paying money you already owe.

## Why these clauses are the most negotiated

In B2B deals, the risk clauses are where the real negotiation happens — far more than the price. The reason is simple: **this is where the money risk actually lives.** A price is a known, bounded number. An uncapped liability or a missing indemnity is an *unbounded* number, and that's what keeps lawyers up at night.

Expect a serious negotiation to focus on raising the cap, narrowing the indirect-damages exclusion, getting (or giving) an uncapped IP indemnity, and adding carve-outs from the cap for the scariest scenarios (data breach, confidentiality violations, IP infringement). When you read a contract and the risk clauses are aggressively one-sided, that's not a detail — it's the heart of the deal.

## Jurisdictional limits on disclaimers

Risk clauses are not infinitely enforceable. The law sets floors that a contract can't disclaim below:

- **Consumer protection** — the EU and UK won't let sellers disclaim core consumer warranties or unfairly limit liability to consumers; a sweeping "AS IS" that's routine in a US B2B deal can be **void** against an EU consumer.
- **Unconscionability** — US courts can strike a liability limit that's grossly one-sided or hidden.
- **Carve-outs the law requires** — most systems forbid disclaiming liability for **gross negligence, willful misconduct, fraud, and death or personal injury**, no matter what the contract says.

So a clause that looks ironclad may not hold, especially across borders. The dedicated [cross-border lesson](/learn/software-licensing/licensing-across-jurisdictions/) goes deeper.

## A risk-clause reference

| Clause | What it does | Where the risk lands |
|---|---|---|
| **Warranty** | Promises the software performs as stated | More on vendor if granted; on you if absent |
| **"AS IS" / disclaimer** | Strips implied warranties (merchantability, fitness) | On you — you take the product as it is |
| **Liability cap** | Limits total payout, often to fees paid | On you, above the cap |
| **Indirect-damages exclusion** | Removes lost profits and consequential loss | On you — often the largest real loss |
| **Indemnification** | Covers third-party claims (esp. IP) | On the indemnifier — vital when embedding others' code |
| **Insurance** | Requires coverage at stated limits | Makes indemnities actually collectible |
| **Force majeure** | Excuses performance for uncontrollable events | Pauses obligations; rarely excuses paying what's owed |

<div class="knowledge-check" data-quiz data-correct-msg="Right — a very common limitation-of-liability cap is the total fees you paid in the previous 12 months, often paired with an exclusion of indirect damages." markdown="0">
  <p class="knowledge-check__q">Quick check: what is a very common way a limitation-of-liability clause caps the vendor's total liability?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">At the full amount of any losses you can prove, with no ceiling</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">At the total fees you paid in the prior 12 months</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">At a fixed statutory minimum set by federal law</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Warranties vs "AS IS"** — a warranty promises performance; an "AS IS" disclaimer strips the implied warranties the law would otherwise add.
- **Liability caps limit the payout** — commonly to fees paid, and almost always paired with an exclusion of indirect and consequential damages.
- **Indemnity covers third-party claims** — the IP indemnity is critical when you ship products built on others' code; check whether it's carved out of the cap.
- **Insurance and force majeure** — insurance makes indemnities collectible; force majeure pauses (not erases) obligations and rarely excuses payment.
- **This is where the money risk lives** — which is exactly why these are the most negotiated clauses in B2B deals.
- **Disclaimers have legal limits** — consumer law and rules against disclaiming gross negligence, fraud, or personal injury can void overly broad clauses, especially abroad.

Next up: turn all of this into a pre-signature checklist and a list of red flags to catch before you commit. See [What to check before you agree](/learn/software-licensing/before-you-agree/).
