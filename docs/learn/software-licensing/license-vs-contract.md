---
slug: license-vs-contract
title: License vs contract
description: A bare license is one-sided permission; a contract is a mutual agreement with obligations and consideration. Learn why the distinction changes your legal remedies and how click-wrap, browse-wrap, and EULAs fit in.
keywords: license vs contract, bare license, unilateral permission, contract consideration, copyright infringement vs breach of contract, click-wrap, browse-wrap, EULA enforceability, open source license contract debate, affirmative assent
level: intermediate
status: full
faq:
  - q: "Is an open-source license a contract or just a license?"
    a: "It's genuinely **debated**, and the answer can depend on the jurisdiction and the facts. The traditional open-source view is that these are **bare licenses** — unilateral grants of permission whose violation is copyright infringement. But courts have increasingly been willing to treat their conditions as **contract** terms too, which can give *additional* remedies. In practice a violation may be both at once."
  - q: "What's the difference between click-wrap and browse-wrap agreements?"
    a: "**Click-wrap** requires an affirmative action — you check a box or click \"I agree\" before proceeding — so it's generally **enforceable** because you clearly assented. **Browse-wrap** just posts terms via a link and claims that using the site means you accept them, with no click; courts treat it as **much weaker** and often unenforceable, since the user may never have seen or agreed to the terms."
  - q: "Why does it matter whether something is a license or a contract?"
    a: "Because the **remedies differ**. Breaking the conditions of a bare license can be **copyright infringement**, which carries copyright remedies (potentially including statutory damages in the US). Breaking a contract is **breach of contract**, with its own remedies. The same act may trigger one, the other, or both — and which applies affects what the wronged party can claim."
---
# License vs contract

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**A bare license is one-sided** — it's just permission the owner grants, no promises required from you. **A contract is mutual** — it needs agreement, obligations on both sides, and consideration (an exchange of value). **The distinction changes remedies** — violating a license can be copyright infringement; breaking a contract is breach of contract. **How you "agree" matters** — click-wrap (you click "I agree") is far more enforceable than browse-wrap (terms just linked).
</div>

So far we've treated a license as simply "permission granted." But many software agreements are also full-blown **contracts**, and the two are not the same legal animal. This lesson draws the line between a bare license and a contract, explains why the difference changes what happens when something goes wrong, and walks through how everyday agreements — click-wrap boxes, browse-wrap links, and EULAs — fit the picture. By the end you'll understand why the same broken term can be infringement, breach, or both.

> **This is educational material, not legal advice.** For decisions that carry real risk, consult a qualified attorney.

## A bare license: one-sided permission

A **bare license** (sometimes "naked license") is the simplest case, and it's exactly the picture from [lesson one](/learn/software-licensing/what-is-a-software-license/): the copyright owner *unilaterally grants permission*. It's one-directional. The owner says "you may do X with my code," and that's the whole transaction. You aren't required to promise anything back; you just have permission you didn't have before.

A bare license typically comes with **conditions** — "you may use this *provided that* you keep the copyright notice." But those conditions are framed as the *boundaries of the permission*, not as promises you owe the owner. Stay inside the conditions and you're licensed; step outside them and you simply lose the permission, falling back to "all rights reserved."

## A contract: a mutual agreement

A **contract** is a different structure: a *mutual* agreement that binds both sides. To form one (in common-law systems like the US and UK), you generally need:

- **Offer and acceptance** — one side proposes terms, the other agrees to them.
- **Consideration** — each side gives or promises something of value. This is the classic "bargained-for exchange" — money for software, your data for a free service, mutual promises.
- **Intent to be bound** — both sides mean to create legal obligations.

The defining feature is **mutual obligation**. In a contract, *both* parties owe each other something, and either can be held to their promises. That's a richer relationship than a bare license's one-way grant.

| | Bare license | Contract |
|---|---|---|
| **Direction** | One-sided (permission granted) | Two-sided (mutual obligations) |
| **Needs your agreement?** | Not necessarily | Yes — offer, acceptance |
| **Needs consideration?** | No | Yes (exchange of value) |
| **Conditions are…** | Limits on the permission | Promises you've made |
| **Violation is…** | Often copyright infringement | Breach of contract |

## Why the distinction changes your remedies

This isn't an academic hair-split — it determines what the wronged party can actually claim when things go wrong.

If a grant is a **bare license** and you act outside its conditions, you no longer have permission, so your use is **copyright infringement**. That routes you into copyright remedies — injunctions, actual damages, and in the US potentially **statutory damages** and attorney's fees if the work was registered (see [the copyright lesson](/learn/software-licensing/copyright-and-software-ownership/)).

If the same arrangement is a **contract** and you violate a term, that's **breach of contract**, which carries its own, often different, remedies — typically the damages needed to put the other side where they'd have been had you performed, and whatever the contract itself specifies.

The amounts, the available remedies, and even *who can sue and where* can differ between the two paths. And critically, a single act can sometimes be **both** at once — infringement *and* breach — giving the owner a choice of theories. That's why lawyers care a great deal about which one a given clause is.

## The open-source debate: license or contract?

Here's where it gets genuinely unsettled. Are open-source licenses bare licenses or contracts?

The **traditional open-source position** is that they're **bare licenses**: a unilateral grant whose conditions (like copyleft's "share your changes under the same terms") are conditions *on the copyright permission*. Violate them and you've lost your license, so you're infringing copyright — and infringement remedies are what enforce the license. This framing has been important historically, partly because copyright remedies can be stronger.

But courts have increasingly been willing to treat those same conditions as **contractual** as well, especially as open-source agreements get used in commercial relationships. A notable line of US cases has recognized that open-source conditions can be enforced as contract terms, potentially giving licensors *both* infringement and breach remedies. The upshot for you: don't assume an open-source license is "just a license with no teeth." Depending on the jurisdiction and facts, violating one can expose you to copyright *and* contract liability.

## How you agree: click-wrap vs browse-wrap

For something to be a *contract*, you generally have to **assent** to it — and *how* a piece of software asks for your agreement strongly affects whether a court will enforce it.

### Click-wrap: affirmative assent, generally enforceable

A **click-wrap** agreement requires you to take a clear, affirmative action — check a box, click "I Agree," type "ACCEPT" — *before* you can proceed. Because you visibly and deliberately assented, courts generally find click-wrap **enforceable**: there's strong evidence you had the chance to read the terms and agreed to them.

```text
☐ I have read and agree to the Terms of Service and Privacy Policy.
            [ Continue ]   ← disabled until the box is checked
```

That pattern — proceeding *blocked* until you act — is the gold standard for forming an enforceable agreement.

### Browse-wrap: weak and often unenforceable

A **browse-wrap** agreement just posts the terms somewhere (often a footer link) and *declares* that by using the site or software you accept them — with **no click, no box, no required action**. Courts treat browse-wrap as **much weaker**. If a user could plausibly never have seen the terms, there's no real assent, and the agreement may be unenforceable. The more buried the link and the less a user has to do, the worse it holds up.

The lesson for builders: if you want your terms to bind, make people *act* to accept them. The lesson for users: a click-wrap "I Agree" is a real legal act — clicking it can form a binding contract, so it's worth a moment's attention ([How to read a software agreement](/learn/software-licensing/how-to-read-an-agreement/) covers what to look for).

## EULAs and their enforceability

A **EULA (End User License Agreement)** is the agreement presented when you install or first run software — the long text behind the "I Agree." Most EULAs are a *blend*: they grant a license (permission to use the software) *and* impose contract terms (restrictions on use, disclaimers of warranty, limits on liability). We give EULAs and terms of service their own treatment later in [EULAs & terms of service](/learn/software-licensing/eula-and-tos/).

Are EULAs enforceable? Largely **yes**, especially when delivered click-wrap style with affirmative assent — courts in the US have broadly upheld them. But enforceability isn't unlimited. Specific terms can be struck down if they're unconscionable or violate other law, and *how* the EULA was presented matters (the click-wrap vs browse-wrap distinction again).

### A jurisdiction note

This is heavily US-flavored. Other legal systems can treat consumer agreements quite differently — many **EU** countries and the UK have **unfair contract terms** rules and strong consumer-protection law that can void one-sided clauses (for example, sweeping disclaimers of liability to consumers) no matter what the EULA says, and they regulate how terms must be presented to bind a consumer. So a clause that's routinely enforced in the US might be unenforceable against a consumer in the EU. The dedicated [cross-border lesson](/learn/software-licensing/licensing-across-jurisdictions/) digs into this.

<div class="knowledge-check" data-quiz data-correct-msg="Right — click-wrap requires an affirmative action ('I Agree'), giving clear evidence of assent, so it's generally enforceable; browse-wrap requires no action and is much weaker." markdown="0">
  <p class="knowledge-check__q">Quick check: which kind of online agreement is generally the most enforceable?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Browse-wrap — a terms link in the page footer that applies just by using the site</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Click-wrap — you must check a box or click "I Agree" before proceeding</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Both are equally enforceable, since posting the terms is all that legally matters</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **A bare license is one-sided** — unilateral permission with conditions that bound the grant, requiring nothing back from you.
- **A contract is mutual** — it needs offer, acceptance, and consideration, and binds both parties to obligations.
- **The distinction changes remedies** — violating a bare license can be copyright infringement; breaking a contract is breach of contract, and one act can be both.
- **Open source is debated** — traditionally bare licenses, but courts increasingly enforce their conditions as contract terms too, so they can carry both kinds of liability.
- **How you agree matters** — click-wrap with affirmative assent is generally enforceable; browse-wrap with no required action is much weaker.
- **EULAs are mostly enforceable, but not unlimited** — and consumer-protection regimes (notably in the EU) can void one-sided terms the US would uphold.

Next up: zoom out to the full range of how software gets licensed, from fully closed to public domain. See [The licensing spectrum](/learn/software-licensing/the-licensing-spectrum/).
