---
slug: eula-and-tos
title: EULAs & terms of service
description: A EULA licenses software you install; terms of service set the rules for using an online service. Learn the difference, when you need which, how click-wrap acceptance works, and the key terms each one contains.
keywords: EULA, end user license agreement, terms of service, terms of use, ToS, click-wrap, acceptable use, SaaS subscription agreement, software license grant, user accounts, account termination, user content license
level: beginner
status: full
faq:
  - q: "What's the actual difference between a EULA and terms of service?"
    a: "A **EULA** licenses a piece of software you install or download — its heart is the license grant and the restrictions on what you may do with the copy. **Terms of service** set the rules for using an ongoing service, site, or app you access — accounts, acceptable use, your content, and when access can be cut off. One licenses a thing; the other governs a relationship."
  - q: "Does a SaaS product need a EULA?"
    a: "Usually not a classic one. With **software as a service**, nobody installs your code — they use it over the network — so there's little to license to the user. SaaS typically pairs **terms of service** with a **subscription agreement** (the commercial terms: fees, term, SLA). You only need a EULA-style grant when the customer actually runs your binary on their own machine."
  - q: "Is a EULA legally binding if I just clicked 'I agree'?"
    a: "Generally yes. **Click-wrap** — where you take an affirmative action like ticking a box or clicking \"I agree\" — is widely enforced because you clearly assented. The weak form is **browse-wrap**, where terms are merely linked and using the site supposedly means acceptance; courts treat that as much shakier. See [license vs contract](/learn/software-licensing/license-vs-contract/)."
---
# EULAs & terms of service

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**EULA = license for software you run** — its core is the grant of permission to use the copy, plus restrictions. **ToS = rules for using a service** — accounts, acceptable use, your content, and termination. **SaaS usually skips the EULA** — it pairs terms of service with a subscription agreement instead. **Acceptance is by click-wrap** — clicking "I agree" forms a binding agreement; merely linking terms (browse-wrap) is far weaker.
</div>

Up to now this path has mostly looked at agreements between developers — open-source licenses, contributor terms, commercial licenses. This lesson turns to the agreements your software shows ordinary users: the **End User License Agreement (EULA)** and the **terms of service**. They sound interchangeable and are often confused, but they do different jobs. By the end you'll know which one fits installed software versus an online service, how acceptance actually works, and the key clauses each document carries.

## Two different jobs

The cleanest way to tell them apart is to ask *what is being agreed to*.

A **EULA** is a license. It is the agreement that comes with a piece of software you **install or download** — a desktop app, a mobile app, a game, a driver. Its center of gravity is the [license grant](/learn/software-licensing/what-is-a-software-license/): the publisher owns the copyright, and the EULA is how they give *you*, the end user, permission to run a copy. Around that grant sit the restrictions — what you may not do with the software.

**Terms of service** (also called **terms of use**) govern an ongoing **service** you access rather than a copy you possess — a website, a web app, a social platform, an API. There may be no software for you to license at all; you're using software that runs on someone else's servers. So the document's job shifts from "here's permission to use this copy" to "here are the rules of the relationship": who may have an account, what you're allowed to do on the service, what happens to content you upload, and when your access can be ended.

A rough rule of thumb: **a EULA licenses a thing; terms of service govern a relationship.**

## When you need which

Which document you need follows directly from how your software reaches the user.

- **You ship software the user installs and runs** — a downloaded app or on-premises product. The user has a copy, so you need a **EULA** to license that copy.
- **You run software the user accesses over the network** — software as a service (SaaS), a website, a web or mobile app backed by your servers. The user has no copy to license, so you need **terms of service** for the rules, and for a paid product a **subscription agreement** for the commercial terms.
- **You do both** — say a desktop client that talks to your cloud backend. It's common to use *both*: a EULA (or an embedded license grant) for the installed client and terms of service for the hosted side.

The SaaS case trips people up, so it's worth stating plainly: **most SaaS products do not use a classic EULA.** Because nothing is installed, the old "license to use this copy" framing barely applies. Instead you'll typically see terms of service paired with a subscription (or "master subscription") agreement that handles fees, the subscription term, renewal, and any [service-level commitments](/learn/software-licensing/sla-and-support-agreements/). The license grant that remains is small — usually a limited right to *access and use* the service during the subscription — rather than a right to possess and run a binary.

| | **EULA** | **Terms of service / use** |
|---|---|---|
| **Governs** | A copy of installed/downloaded software | Use of an online service, site, or app |
| **Core clause** | The software license grant + restrictions | Acceptable use, accounts, content, termination |
| **Typical product** | Desktop app, mobile app, game, driver | Website, web app, SaaS, social platform, API |
| **Is there a "copy"?** | Yes — the user runs it | Often no — it runs on the provider's servers |
| **Paired with** | Sometimes a separate sales contract | Often a subscription agreement (for paid SaaS) |
| **Main concern** | Protecting and bounding the licensed copy | Running the service and the user community safely |

## How acceptance works

Neither document does anything unless the user has actually **agreed** to it, and how that agreement is captured matters a great deal. This ties straight back to [license vs contract](/learn/software-licensing/license-vs-contract/): a EULA or ToS is typically a *contract*, and a contract needs assent.

The reliable mechanism is **click-wrap**: the user must take an affirmative action — tick a box, click "I agree," tap "Accept" — before they can proceed. Because the user clearly signaled agreement, courts in the US (and broadly elsewhere) generally enforce click-wrap terms. The weak mechanism is **browse-wrap**, where the terms are simply linked somewhere — often a small "Terms" link in the footer — and the document claims that merely using the site means you accept them. With no affirmative click, a user may never have seen the terms, and courts frequently find browse-wrap **unenforceable**, especially against consumers.

```text
Strong (click-wrap):
  [x] I have read and agree to the Terms of Service and EULA
  [ Continue ]   <- blocked until the box is checked

Weak (browse-wrap):
  ...site footer... | Privacy | Terms |   <- no action required, easy to miss
```

The practical lesson for anyone publishing software: **make acceptance affirmative and conspicuous.** Put the agreement (or a clear link to it) right at the point of installation, sign-up, or first use, and require a deliberate click. Burying it in a footer and hoping is a recipe for terms that don't hold up.

> Acceptance rules vary by jurisdiction, and consumer-protection law (for example in the EU and UK) adds its own requirements around clarity and unfair terms. The dedicated lesson on [licensing across jurisdictions](/learn/software-licensing/licensing-across-jurisdictions/) goes deeper; here, just remember that *how* you obtain consent is as important as *what* the terms say.

## What a EULA typically contains

A EULA is organized around the licensed copy. The usual building blocks:

- **License grant** — the permission itself: typically a non-exclusive, non-transferable right to install and use the software, often limited by number of devices, seats, or a personal-vs-commercial distinction.
- **Restrictions** — the "thou shalt nots": no reverse-engineering (subject to legal limits in some places), no redistribution, no renting or sublicensing, no removing notices.
- **Ownership / IP reservation** — a reminder that you are licensed, not sold, the software, and the publisher keeps all [intellectual-property rights](/learn/software-licensing/other-ip-in-software/).
- **Updates and termination** — how updates are delivered and when the license ends (for example, if you breach the restrictions).
- **Warranty disclaimer and limitation of liability** — the risk-shifting clauses we cover in the reading-agreements module; almost every EULA disclaims warranties and caps liability.
- **Third-party / open-source notices** — attribution for bundled open-source components (GopherTrunk's own [`THIRD_PARTY_LICENSES.md`](/learn/software-licensing/permissive-compliance/) is exactly this kind of notice file).

## What terms of service typically contain

Terms of service are organized around the *relationship and the service*. Common building blocks:

- **Eligibility and accounts** — who may use the service (age, region), and the user's duties to keep their account secure and their details accurate.
- **Acceptable use** — what users may and may not do on the service: no illegal activity, no abuse, no scraping, no interfering with others. Larger services often split this into a separate [Acceptable Use Policy](/learn/software-licensing/nda-and-other-agreements/).
- **Your content** — for any service where users upload or post, a clause covering ownership (usually *you* keep ownership) and the **license you grant the provider** to host, display, and process your content so the service can function.
- **Fees and subscription** — for paid services, how billing, renewal, and cancellation work (sometimes split into the subscription agreement).
- **Termination and suspension** — when and how either side can end the relationship, and what happens to your data and content afterward.
- **Disclaimers, liability limits, dispute resolution** — the same risk clauses as a EULA, plus often a governing-law and arbitration clause.
- **Changes to the terms** — how the provider may update the ToS and how you'll be notified.

### A note on consumer-facing examples

You agree to these constantly. The "I accept the License Agreement" screen when installing a desktop app or a game is a **EULA**. The "By creating an account you agree to our Terms" box on a social network, streaming service, or web app is **terms of service**. A paid cloud tool's "Terms" plus "Subscription Agreement" plus "Privacy Policy" trio is the SaaS pattern in the wild — terms for the rules, subscription for the money, and a [privacy policy](/learn/software-licensing/privacy-and-data-agreements/) for the data. Recognizing which is which tells you immediately where to look for the clause you care about.

<div class="knowledge-check" data-quiz data-correct-msg="Right — SaaS users don't install a copy, so it leans on terms of service plus a subscription agreement rather than a classic EULA." markdown="0">
  <p class="knowledge-check__q">Quick check: a company offers a cloud-only product nobody installs. Which set of agreements fits best?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Terms of service plus a subscription agreement, rather than a classic EULA</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A classic installed-software EULA, because all software needs a EULA</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">No agreement at all, since the user never downloads anything</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **A EULA licenses installed/downloaded software** — its core is the license grant to use the copy, plus the restrictions on what you may do with it.
- **Terms of service govern an online service** — accounts, acceptable use, your uploaded content, and termination of access.
- **SaaS usually skips the classic EULA** — because nothing is installed, it leans on terms of service plus a subscription agreement.
- **Acceptance must be affirmative** — click-wrap ("I agree") is enforceable; browse-wrap (terms merely linked) is weak and often fails.
- **Each document has a predictable shape** — know the standard clauses and you'll find what you need fast, whether you're agreeing or drafting.

Next up: the document that governs the personal data your software touches — what it must disclose, and the rules that may legally require it. See [Privacy policies & data agreements](/learn/software-licensing/privacy-and-data-agreements/).
