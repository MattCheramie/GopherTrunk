---
slug: privacy-and-data-agreements
title: Privacy policies & data agreements
description: If your software touches personal data you likely need a privacy policy and, with vendors, a data processing agreement. Learn what a privacy policy must disclose, GDPR and CCPA basics, controller vs processor, and cross-border data transfers.
keywords: privacy policy, data processing agreement, DPA, GDPR, CCPA, CPRA, personal data, data controller, data processor, sub-processor, lawful basis, data subject rights, standard contractual clauses, data residency, cross-border data transfer
level: intermediate
status: full
faq:
  - q: "Does my software actually need a privacy policy?"
    a: "If it collects or processes **personal data** — names, emails, IP addresses, device IDs, location, analytics tied to a person — then almost certainly yes, and often it's **legally required**. Laws like the EU's GDPR and California's CCPA/CPRA mandate disclosure, and app stores and ad networks require a policy to publish at all. A purely offline tool that touches no personal data may not need one, but that's rarer than people think."
  - q: "What's the difference between a data controller and a data processor?"
    a: "The **controller** decides *why and how* personal data is processed — it sets the purposes. The **processor** handles data *on the controller's behalf and instructions* — a vendor that stores or crunches data for you. If you run a service that collects user data, you're typically the **controller**; your cloud host or analytics vendor is a **processor**, which is exactly why you sign a **DPA** with them."
  - q: "What is a DPA and when do I need one?"
    a: "A **Data Processing Agreement** is a contract between a controller and a processor that locks down how the processor may handle personal data — purpose limits, security, breach notice, sub-processors, and deletion. Under GDPR it's **legally required** whenever a processor handles personal data for you. In practice you sign one with most data-handling vendors, and your own customers will ask you to sign one when *you* are their processor."
---
# Privacy policies & data agreements

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Touch personal data, need a privacy policy** — and it's often legally required, not just good manners. **A policy must disclose** what you collect, why, who you share it with, how long you keep it, and users' rights. **GDPR (EU) and CCPA/CPRA (California)** are the two reference regimes, and they differ. **Controller vs processor** decides who's responsible — and a **Data Processing Agreement (DPA)** binds the processor. **Cross-border transfers** need a legal mechanism like Standard Contractual Clauses.
</div>

The agreements in the last lesson set the rules for *using* your software. This one is about the **data your software handles about people**. If your product collects anything that can identify a user, a whole body of law and a set of standard agreements come into play. This lesson is concepts-first, using the United States (California's CCPA/CPRA) and the EU (GDPR) as the two reference points, because the cross-border angle is where this topic gets real. By the end you'll know when you need a privacy policy, what it must say, the difference between a data controller and a processor, and why you end up signing DPAs.

> **This is educational material, not legal advice.** Privacy law is fast-moving and jurisdiction-specific; for real compliance decisions, consult a qualified attorney or privacy professional.

## If you touch personal data, you probably need a policy

**Personal data** (the GDPR term; US laws say **personal information**) is any information relating to an identifiable person — obvious things like name and email, but also IP addresses, device identifiers, location, cookies, and analytics tied to an individual. The threshold is lower than people expect: a web app with logins and analytics is already processing personal data.

Once you are, a **privacy policy** is usually not optional. It's **legally required** under GDPR and the California laws (among many others), it's demanded by the Apple and Google app stores before you can publish, and ad and analytics networks require one in their terms. A privacy policy is a *disclosure* document — its job is to tell people, in plain terms, what you do with their data. It is distinct from the [terms of service](/learn/software-licensing/eula-and-tos/), which set the rules of use; the two usually travel together but do different jobs.

## What a privacy policy must disclose

The exact requirements vary by law, but a sound privacy policy answers a consistent set of questions:

- **What you collect** — the categories of personal data (account info, usage data, device data, location, etc.).
- **Why you collect it** — the purposes, and under GDPR the **lawful basis** for each (more below).
- **Who you share it with** — categories of third parties: vendors/processors, partners, advertisers, and any sale or "sharing" of data.
- **How long you keep it** — retention periods, or the criteria you use to decide them.
- **Users' rights and how to exercise them** — access, correction, deletion, opting out, and a contact channel.
- **Transfers** — whether data leaves the user's country/region, and the safeguard used.
- **Cookies/tracking and contact details** — your identity as the responsible party and how to reach you (under GDPR, sometimes a Data Protection Officer).

The throughline: a privacy policy should let a reasonable person understand what happens to their data *before* they hand it over. Vague boilerplate that says "we may use your data to improve our services" satisfies nobody and increasingly fails legal scrutiny.

## GDPR basics (EU)

The EU's **General Data Protection Regulation** is the most influential privacy law in the world; it applies to organizations in the EU and to those *outside* the EU that target or monitor people in the EU. A few core ideas:

- **Lawful basis.** You can't process personal data just because you'd like to. You need one of six lawful bases — commonly **consent**, **contract** (processing needed to deliver what the user signed up for), or **legitimate interests** (a balancing test). You must pick and document the basis for each purpose.
- **Consent has standards.** Where you rely on consent, it must be freely given, specific, informed, and unambiguous — an affirmative opt-in, not a pre-ticked box. And it must be as easy to withdraw as to give.
- **Data subject rights.** Individuals can request access to their data, correction, **erasure** ("right to be forgotten"), portability, and can object to certain processing. You must be able to respond, usually within a set time.
- **Accountability and breach notice.** You must be able to *demonstrate* compliance, and serious breaches generally must be reported to a regulator within 72 hours.

Penalties are large — up to the greater of €20 million or 4% of global annual turnover for the most serious violations — which is why GDPR drives so much global practice. The UK has its own post-Brexit equivalent (UK GDPR) that is closely aligned.

## CCPA / CPRA basics (California)

The United States has **no single federal privacy law**; instead it has a growing patchwork of state laws, of which California's is the bellwether. The **California Consumer Privacy Act (CCPA)**, strengthened by the **California Privacy Rights Act (CPRA)**, gives California residents rights over their personal information.

- **Rights.** To **know** what's collected, to **delete** it, to **correct** it, and to **opt out of the sale or "sharing"** of personal information (CPRA added the "sharing" concept, aimed at cross-context behavioral advertising).
- **Opt-out, not opt-in.** This is a key contrast with GDPR. CCPA generally works on an **opt-out** model — businesses can collect and even "sell" data unless the consumer opts out (hence "Do Not Sell or Share My Personal Information" links). GDPR leans toward needing a lawful basis up front.
- **Who it covers.** It applies to for-profit businesses that meet certain thresholds (revenue or volume of data), not to every tiny app.

Other US states (Virginia, Colorado, and more) have passed similar laws, so "comply with US privacy law" increasingly means handling a set of state regimes, not one rule.

| | **GDPR (EU)** | **CCPA / CPRA (California)** |
|---|---|---|
| **Default model** | Need a **lawful basis** to process at all | Collect unless consumer **opts out** |
| **Consent** | Affirmative opt-in, easy to withdraw | Opt-out of sale/sharing is the headline right |
| **Term for data** | Personal data | Personal information |
| **Key rights** | Access, erasure, portability, object | Know, delete, correct, opt out of sale/sharing |
| **Who's regulated** | Anyone targeting/monitoring EU people | For-profit businesses over set thresholds |
| **Headline penalty** | Up to €20M or 4% of global turnover | Per-violation fines; private action for breaches |

These two are the reference points, but they are not the whole world — Brazil's LGPD, Canada's PIPEDA, and many others exist. The [licensing-across-jurisdictions](/learn/software-licensing/licensing-across-jurisdictions/) lesson is where this path treats cross-border issues in depth.

## Controller vs processor, and the DPA

When more than one organization touches the data, the law assigns roles, and the roles decide who's responsible.

- A **controller** decides the *purposes and means* of processing — *why* and *how* data is handled. If you run a service that collects users' data for your own ends, you're the controller.
- A **processor** processes data **on the controller's behalf and under its instructions** — a vendor doing a job for you. Your cloud host, your email-sending service, your analytics provider, your error-tracking tool are typically processors.

Because a controller stays responsible for data even when a vendor handles it, GDPR **requires a contract** between them: the **Data Processing Agreement (DPA)**. A DPA pins down how the processor may act:

- Process only on the controller's documented instructions and only for the stated purpose.
- Keep the data secure and keep staff under confidentiality.
- Notify the controller of breaches without undue delay.
- Allow audits and help the controller meet data-subject requests.
- Delete or return the data when the engagement ends.

### Sub-processors

A processor often uses its own vendors — a SaaS app that runs on a cloud provider, for instance. Those downstream vendors are **sub-processors**. A DPA normally requires the processor to *flow down* the same obligations to its sub-processors, to disclose who they are, and often to give the controller a chance to object to new ones. This is why mature vendors publish a sub-processor list.

You will sit on **both sides** of this over time. When you use vendors, you're the controller signing DPAs with them. When your *customers* feed their users' data into your product, *you* are the processor and they'll ask you to sign a DPA — so you'll need one ready to offer.

## Data residency and cross-border transfers

Personal data often crosses borders — a European user's data landing on a US-based cloud server, say. GDPR restricts transferring personal data outside the EU/EEA unless there's an approved safeguard. The most common one for ordinary companies is the **Standard Contractual Clauses (SCCs)** — pre-approved contract terms the parties sign to commit to GDPR-level protection. (For the EU–US route specifically, the **Data Privacy Framework** is an additional adequacy-based mechanism for certified US companies.)

**Data residency** is the related, more concrete question of *where data physically lives* — which country's data centers store it. Some customers and some laws require data to stay within a particular region, which is why cloud providers offer region selection and why enterprise contracts often specify residency. The takeaway: if your data moves across borders, you need a transfer mechanism, and if customers care where it sits, you need to be able to answer.

<div class="knowledge-check" data-quiz data-correct-msg="Right — you set the purposes, so you're the controller; the analytics vendor processes on your behalf, making it your processor, which is why you sign a DPA." markdown="0">
  <p class="knowledge-check__q">Quick check: your app collects user data for your own purposes and sends some of it to a third-party analytics vendor. What are the roles?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">You're the processor and the analytics vendor is the controller</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">You're both controllers, so no agreement between you is needed</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">You're the controller, the vendor is your processor, and you sign a DPA with it</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Touch personal data, need a privacy policy** — it's commonly legally required and demanded by app stores and ad networks, not just polite.
- **Disclose the essentials** — what you collect, why, who you share with, how long you keep it, users' rights, and any transfers.
- **GDPR and CCPA/CPRA differ in default** — GDPR needs a lawful basis up front (opt-in for consent); CCPA leans on a consumer opt-out of sale/sharing.
- **Roles decide responsibility** — the controller sets the purposes; the processor acts on instructions; a DPA binds the processor (and flows down to sub-processors).
- **Cross-border data needs a mechanism** — Standard Contractual Clauses or similar for transfers, and data-residency choices when customers or law require it.

Next up: the agreements that promise how *well* a service will run, and what you get when it doesn't. See [SLAs & support agreements](/learn/software-licensing/sla-and-support-agreements/).
