---
slug: main-clauses-decoded
title: The main clauses, decoded
description: A plain-English tour of the clauses in nearly every software agreement — license grant, permitted and prohibited use, fees, term and auto-renewal, termination and survival, confidentiality, IP ownership of your data and feedback, assignment, and the entire-agreement clause.
keywords: software contract clauses explained, license grant, permitted use, prohibited use, fees and taxes, price changes, term and renewal, auto-renewal, termination, survival clause, confidentiality, IP ownership, feedback clause, assignment, change of control, entire agreement, amendment
level: intermediate
status: full
faq:
  - q: "What's the difference between the license grant and the restrictions clause?"
    a: "The **grant** says what you *may* do — the scope of your permission (who can use it, for what, how many seats, perpetual or for the term). The **restrictions** list what you *may not* do — reverse-engineer, resell, exceed your seat count, benchmark, and so on. Read them together: your real rights are the grant minus the restrictions, and stepping outside either can end the license."
  - q: "Who owns the data I put into a service, or the feedback I send the vendor?"
    a: "Usually **you keep ownership of your data**, but you grant the provider a license to process it to run the service — read that license's scope carefully. **Feedback** is different and often overlooked: many agreements say any suggestions you send become the vendor's to use freely and forever. That's common, but worth knowing you're giving away."
  - q: "What does 'survival' mean in a termination clause?"
    a: "**Survival** lists the clauses that stay in force *after* the agreement ends — typically confidentiality, payment of amounts already owed, liability limits, and IP ownership. Termination doesn't wipe the slate clean; the surviving clauses keep binding you, sometimes indefinitely, so always check what survives before assuming an exit is a clean break."
---
# The main clauses, decoded

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Grant and restrictions define your real rights** — what you may do, minus what you may not. **Money clauses change over time** — watch fees, taxes, and unilateral price increases. **Term and renewal can auto-roll** — auto-renewal plus a notice window is the classic trap. **Termination has a survival list** — some clauses outlive the agreement. **IP clauses can reach your data and feedback** — know what you keep and what you give away.
</div>

You've learned [a method for reading any agreement](/learn/software-licensing/how-to-read-an-agreement/). Now we'll walk through the actual clauses you'll meet, one by one, and translate each into plain English: what it's for, what it really means, and why you should care. These same clauses appear, in some form, in almost every software agreement — a EULA, a SaaS contract, a commercial license. Learn them once and the next contract reads far faster. This lesson takes the *buyer/user* point of view; the [key commercial clauses](/learn/software-licensing/key-commercial-clauses/) lesson covers the same ground from the seller's side.

> **This is educational material, not legal advice.** For decisions that carry real risk, consult a qualified attorney.

## The license grant and its scope

The **grant** is the heart of the agreement — it's the permission itself. A well-drafted grant spells out its *scope* along several axes, and the scope is where the real meaning lives:

- **Exclusive or non-exclusive** — almost always non-exclusive (others get the same software).
- **Perpetual or for the term** — does the right last forever, or only while you keep paying?
- **Revocable or irrevocable** — can they pull it back?
- **Transferable or not** — can you pass it on, or is it locked to you?
- **The permitted users and uses** — how many seats, which entities (you, your Affiliates?), internal use only or also for serving customers.

A grant that says "non-exclusive, non-transferable, revocable license for internal business use by up to ten Authorized Users for the Subscription Term" is a very different thing from "perpetual, irrevocable license." Read every adjective; each one narrows what you actually get.

## Permitted and prohibited use

Right after the grant, agreements list what you **may not** do. Common restrictions: no reverse engineering, no reselling or sublicensing, no removing notices, no exceeding your seat or usage limits, no using the software to build a competing product, and sometimes no publishing benchmarks. 

Your real rights are **the grant minus the restrictions**. A generous-sounding grant can be hollowed out by a long restrictions list, so read them as a pair. And note: violating a restriction often doesn't just breach the contract — it can put you *outside* the license entirely, which (as the [license-vs-contract](/learn/software-licensing/license-vs-contract/) lesson explained) can make your use copyright infringement.

## Fees, taxes, and price changes

The money clause is more than a number. Watch for:

- **What the fee covers and how it's metered** — per seat, per usage, flat. Does it scale in a way you control?
- **Taxes** — most agreements make *you* responsible for applicable taxes on top of the listed price.
- **Payment terms** — net 30, late fees, suspension for non-payment.
- **Price changes** — the clause that lets the vendor raise prices, often "upon renewal" or sometimes "upon notice." A **unilateral price-increase** clause means the cost you signed up for is not the cost you're locked into.

The headline price is what gets advertised; the price-change clause is what you'll actually pay over the years.

## Term, renewal, and the auto-renewal trap

The **term** clause sets how long the agreement lasts. The part to slow down on is **renewal** — specifically **automatic renewal**.

```text
This Agreement renews automatically for successive one-year terms unless
either party gives written notice of non-renewal at least sixty (60) days
before the end of the then-current term.
```

Read that carefully. It renews *by default*. To stop it, you must act, in writing, inside a **notice window** (here, 60 days before the end). Miss the window by a day and you're locked in — and billed — for another full year. Auto-renewal isn't inherently abusive; it's everywhere. But the combination of auto-renewal *and* a long notice window is a classic way to keep customers paying past the point they meant to leave. Calendar the deadline the day you sign.

## Termination and survival

The **termination** clause says how the agreement ends: at the end of the term, for convenience (either side, with notice), or for cause (a breach the other side didn't cure). Read it for symmetry — can *you* exit as easily as they can?

Then read what **survives**. A **survival clause** lists the provisions that remain in force after termination — typically:

- **Confidentiality** (often for years afterward)
- **Payment** of amounts already owed
- **Limitation of liability** and **disclaimers**
- **IP ownership**

Termination is not a reset button. The surviving clauses keep binding you, so "we can cancel anytime" is only half the story — check what stays alive after you do.

## Confidentiality

The **confidentiality** clause defines what information each side must keep secret, for how long, and the carve-outs (information already public, independently developed, or legally compelled). If the agreement is mutual, it binds both ways; sometimes it's one-sided. Note the duration — confidentiality obligations commonly survive termination for a fixed number of years, occasionally forever for trade secrets. NDAs get their own treatment in [NDAs & the rest of the landscape](/learn/software-licensing/nda-and-other-agreements/).

## Intellectual-property ownership — including yours

This clause settles who owns what. Three pieces matter to you:

1. **The vendor's IP stays the vendor's.** Standard — you're licensed, not buying ownership.
2. **Your data.** Usually *you* keep ownership of the data you put into a service, but you grant the provider a **license** to process it to deliver the service. Read that license's scope: does it allow only running the service, or also "improving our products," training, or analytics? (We go deeper in [Privacy policies & data agreements](/learn/software-licensing/privacy-and-data-agreements/).)
3. **Feedback.** Easy to miss: many agreements say any **feedback, suggestions, or ideas** you send the vendor become *theirs* to use freely, forever, with no obligation to you. That's common and usually fine — but you should know you're handing it over.

## Assignment and change of control

**Assignment** governs whether either party can transfer the agreement to someone else. The clause to watch: can the vendor **assign on a change of control** — say, when they're acquired — so your contract ends up with a company you'd never have chosen? Many agreements let the vendor assign freely while restricting *you* from doing the same. Note the asymmetry.

## Entire agreement and amendment

Two short clauses with outsized effect:

- **Entire agreement (merger clause)** — states that the written agreement is the *complete* deal and supersedes all prior discussions. Translation: that promise the salesperson made in an email **doesn't count** unless it's in the document. If a commitment matters, get it written into the agreement.
- **Amendment** — how the terms can change. The safe version requires a signed writing from both sides. The risky version lets the vendor **change the terms unilaterally** (common in consumer terms of service: "we may update these terms; continued use is acceptance").

## A clause-by-clause reference

| Clause | Plain meaning | Why you care |
|---|---|---|
| **License grant** | What you may do, and the scope of it | Defines your core rights — read every adjective |
| **Restrictions** | What you may not do | Hollows out the grant; breaking one can void the license |
| **Fees & taxes** | What you pay, plus tax | Taxes and metering can exceed the headline price |
| **Price changes** | When the vendor can raise prices | The signed price may not be the locked-in price |
| **Term & renewal** | How long, and auto-renewal | Miss the notice window, get billed another term |
| **Termination** | How it ends, by whom | Check whether *you* can exit as easily as they can |
| **Survival** | Which clauses outlive the agreement | Ending the deal doesn't end every obligation |
| **Confidentiality** | What stays secret, for how long | Can bind you for years after termination |
| **IP ownership** | Who owns the software, your data, your feedback | You may be licensing your data and giving away feedback |
| **Assignment** | Who can transfer the deal | You could end up bound to a company you didn't choose |
| **Entire agreement** | Only the written terms count | Side promises don't bind unless they're in the document |
| **Amendment** | How terms change | A unilateral-change clause means the deal can shift under you |

<div class="knowledge-check" data-quiz data-correct-msg="Right — a survival clause lists the provisions (like confidentiality and liability limits) that stay binding even after the agreement ends." markdown="0">
  <p class="knowledge-check__q">Quick check: what does a "survival" clause do in a software agreement?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It guarantees the software keeps running if the vendor goes out of business</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It lets you survive a price increase by locking in your original rate</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It lists the clauses that stay in force after the agreement is terminated</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Grant minus restrictions equals your real rights** — read the scope adjectives and the prohibited-use list together.
- **Money clauses evolve** — taxes, metering, and unilateral price increases can push the cost past the headline number.
- **Auto-renewal plus a notice window is the classic trap** — calendar the cancellation deadline the day you sign.
- **Termination isn't a clean break** — the survival clause keeps confidentiality, payment, and liability terms alive afterward.
- **IP clauses can reach your data and feedback** — you usually keep your data but license it, and you may be giving feedback away outright.
- **Boilerplate bites** — the entire-agreement clause kills side promises, and an amendment clause can let terms change unilaterally.

Next up: the clauses that decide who actually pays when something goes wrong — warranties, liability caps, and indemnity. See [The risk clauses](/learn/software-licensing/risk-clauses/).
