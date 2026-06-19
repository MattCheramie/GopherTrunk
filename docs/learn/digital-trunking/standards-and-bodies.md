---
slug: standards-and-bodies
title: "Standards & who sets them: APCO, ETSI, TIA & the DMRA"
description: Who writes radio standards — APCO, the TIA-102 suite for P25, ETSI for TETRA and DMR, the DMR Association, and the ITU — plus open vs proprietary systems and why it matters.
keywords: APCO, TIA-102, ETSI, DMR Association, DMRA, ITU, P25 standard, TETRA standard, DMR standard, open vs proprietary, interoperability, mutual aid
level: beginner
status: full
prereq:
  - the-digital-leap
faq:
  - q: Who writes the P25 standard?
    a: P25 was driven by APCO (the Association of Public-Safety Communications Officials) and standardized by the TIA (Telecommunications Industry Association) as the TIA-102 suite of documents. That suite defines everything from the air interface to encryption, which is what makes P25 a true open, multi-vendor standard.
  - q: What is the difference between an open and a proprietary system?
    a: An open standard is published so any vendor can build compatible equipment, which enables interoperability and competition — P25 is the classic example. A proprietary system is controlled by one company and only works with its gear or licensees, such as Motorola Type II or Connect Plus. Open standards make multi-vendor and mutual-aid operation far easier.
  - q: What does ETSI standardize?
    a: ETSI (the European Telecommunications Standards Institute) publishes the standards for both TETRA and DMR. The DMR Association then promotes interoperability among DMR vendors on top of the ETSI specification. So TETRA and DMR are European-rooted open standards, even though they are used worldwide.
  - q: Why do open standards matter for public safety?
    a: Because emergencies cross agency and jurisdiction lines. When everyone's equipment follows the same open standard, radios from different vendors and agencies can talk to each other directly, which is essential for mutual aid. Proprietary systems can lock an agency to a single vendor and complicate cross-agency communication.
gophertrunk_links:
  - title: Status
    url: /status.html
    note: which standardized protocols GopherTrunk decodes.
---

# Standards & who sets them: APCO, ETSI, TIA & the DMRA

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Behind every protocol is a body that writes the rules. **APCO** drove **P25**, which the
**TIA** publishes as the **TIA-102** suite. **ETSI** standardizes both **TETRA** and
**DMR**, with the **DMR Association (DMRA)** promoting interoperability on top. The **ITU**
sets the global frame above them all. The key split is **open multi-vendor standards** like
P25 versus **proprietary systems** like Motorola Type II or Connect Plus. Open standards
matter because they enable **interoperability** and **mutual aid** — radios from different
vendors and agencies that can actually talk to each other.
</div>

You've met the protocols; this short lesson steps back to ask who *makes* them and why the
answer affects whether two agencies can talk during an emergency. It's also where you learn
to read a spec number when one turns up in the wild.

## The bodies that write the rules

A handful of organizations stand behind the standards in this path:

- **APCO** — the Association of Public-Safety Communications Officials. APCO drove **Project
  25 (P25)** to give North-American public safety an *open* standard, born of frustration
  with proprietary analog systems that couldn't interoperate.
- **TIA** — the Telecommunications Industry Association. The TIA turned the P25 requirements
  into the **TIA-102** suite of published documents, which defines the air interface,
  vocoder, encryption, and more.
- **ETSI** — the European Telecommunications Standards Institute. ETSI publishes the
  standards for both **TETRA** and **DMR**.
- **DMR Association (DMRA)** — an industry group that promotes **interoperability** between
  DMR vendors on top of the ETSI specification, so that equipment from different makers
  actually works together.
- **ITU** — the International Telecommunication Union, the United Nations body that
  coordinates spectrum and the broad global framework all of the above operate within.

| Body | Scope | Standards |
|------|-------|-----------|
| APCO | North-American public safety requirements | P25 (Project 25) |
| TIA | Publishes the P25 documents | **TIA-102** suite |
| ETSI | European standards (used worldwide) | TETRA, DMR |
| DMR Association | DMR interoperability | DMR profiles |
| ITU | Global spectrum & framework | Recommendations |

## Open vs proprietary

The most consequential distinction here isn't *which* body, but *how open* the result is.

An **open standard** is published in full, so **any vendor** can build compatible
equipment. **P25** is the textbook example: because TIA-102 is public, agencies can buy
radios and infrastructure from multiple manufacturers and have them interoperate. That
breeds competition, lower prices, and — crucially — the ability for different agencies'
radios to talk to each other.

A **proprietary system** is controlled by a single company. **Motorola Type II** and
**Connect Plus** are examples: they work with that vendor's gear (or licensees), and the
details aren't openly published. Proprietary systems can be excellent and feature-rich, but
they can lock an agency to one supplier and complicate cross-agency communication.

This is why a protocol's openness shows up everywhere downstream, including in how
[GopherTrunk](/status.html) and other software can support it: open specifications are far
easier to implement against than reverse-engineered proprietary ones.

## Why open standards matter: interoperability and mutual aid

Emergencies don't respect boundaries. A wildfire pulls in county, state, and federal crews;
a highway crash brings police, fire, and EMS from neighbouring jurisdictions. If each agency
runs an incompatible system, they can't coordinate on the radio when it matters most. **Open
standards** solve this: when everyone's equipment follows the same published specification,
radios from different vendors and different agencies can join the same talkgroup directly.
That's **mutual aid**, and it's the central reason North-American public safety pushed so
hard for an open P25.

## Reading a spec number

When you meet a standard reference, you can usually decode it at a glance. A number like
**TIA-102.xxxx** belongs to the P25 family — the TIA-102 suite is organized into documents,
each covering a slice such as the air interface or encryption. ETSI numbers (often beginning
**ETSI TS** or **ETSI EN**) identify TETRA and DMR specifications similarly. You rarely need
to *read* the document itself, but recognizing the prefix tells you instantly which standard
family someone is talking about, and that's enough to orient yourself.

## Recap

- **APCO** drove **P25**; the **TIA** publishes it as the **TIA-102** suite.
- **ETSI** standardizes **TETRA** and **DMR**; the **DMR Association** adds interoperability profiles; the **ITU** sets the global frame.
- The key split is **open multi-vendor standards** (P25) vs **proprietary systems** (Motorola Type II, Connect Plus).
- Open standards enable **interoperability** and **mutual aid** — different vendors' and agencies' radios talking directly.
- A spec's prefix (TIA-102, ETSI TS/EN) tells you which standard family it belongs to.

That completes the history module. The next module turns to how digital voice itself
works, starting with [Analog vs digital
voice](/learn/digital-trunking/analog-vs-digital-voice/).
