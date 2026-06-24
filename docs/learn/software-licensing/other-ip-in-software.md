---
slug: other-ip-in-software
title: Patents, trademarks & trade secrets
description: Copyright is only one of four kinds of intellectual property around software. Learn how patents, trademarks, and trade secrets work, why license patent grants matter, and why open-source licenses don't give away trademarks.
keywords: software intellectual property, software patents, trademarks open source, trade secrets, patent grant license, project name trademark, NDA confidentiality, IP types software, patent troll, copyright vs patent
level: beginner
status: full
faq:
  - q: "Do open-source licenses let me use the project's name and logo too?"
    a: "Generally **no**. Open-source licenses grant copyright (and sometimes patent) rights to the *code*, but they almost always **exclude trademark rights** — the project's name and logo. You can fork and ship the code, but you usually can't call your version by the original name or use its logo in a way that implies endorsement. Apache 2.0, for instance, explicitly says it grants no trademark rights."
  - q: "Why do some licenses include a patent grant if the code is already copyrighted?"
    a: "Because copyright and patents are **different rights**. A copyright license lets you copy and modify the code, but a contributor could still hold a **patent** covering what the code does and sue you for using it. A patent grant — like the one in [Apache 2.0](/learn/software-licensing/apache-license/) — closes that gap by promising contributors won't assert their patents against users of the project."
  - q: "Is keeping my source code secret a form of legal protection?"
    a: "Yes — it's the basis of **trade secret** protection. Information that has value *because* it's secret, and that you take reasonable steps to keep secret, can be protected as a trade secret indefinitely. The catch is that protection evaporates the moment the secret leaks or is independently discovered, which is why companies pair secrecy with **NDAs** and access controls."
---
# Patents, trademarks & trade secrets

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Four kinds of IP surround software** — copyright, patents, trademarks, and trade secrets, each protecting something different. **Patents cover inventions, not expression** — they're why some licenses include a patent grant. **Trademarks protect names and logos** — and open-source licenses generally don't give them away. **Trade secrets protect what stays hidden** — valuable as long as it's kept secret, backstopped by NDAs.
</div>

The [last lesson](/learn/software-licensing/copyright-and-software-ownership/) showed that copyright protects the *expression* of your code but not the underlying ideas, algorithms, or functionality. So how do those get protected — and what about a project's name, or a secret algorithm you never publish? This lesson covers the other three forms of **intellectual property (IP)** that surround software, why they matter when you're reading a license, and how they stack together. By the end you'll be able to tell which kind of right is at play in any licensing situation.

## The four kinds of IP around software

Software is unusual in that all four major flavors of intellectual property can apply to a single product at once. They protect different things and have different rules:

| IP type | Protects | How you get it | Rough lifespan |
|---------|----------|----------------|----------------|
| **Copyright** | The *expression* — source and binary | Automatic on creation | Decades (life + ~70 yrs, varies) |
| **Patent** | A novel, non-obvious *invention* or method | Apply and be granted (expensive, slow) | ~20 years from filing |
| **Trademark** | *Names, logos, brands* that identify a source | Use in commerce; registration strengthens it | Indefinite, while in use |
| **Trade secret** | Valuable *secret* information | Keep it secret with reasonable measures | Indefinite, until it leaks |

Copyright we've covered. The other three each fill a gap copyright leaves.

## Patents: protecting the invention itself

Where copyright protects how you *wrote* something, a **patent** can protect the *invention behind it* — a novel and non-obvious method or process, regardless of the specific code used to implement it. If copyright is "you can't copy my words," a patent is closer to "you can't use my method, even in your own words."

Patents are a different beast from copyright in every practical way. They're **not automatic** — you must file an application, satisfy an examiner that the invention is new and non-obvious, and pay substantial fees, often over years. In exchange you get a strong but time-limited monopoly (roughly 20 years from filing).

### Software patents are controversial

Whether software *should* be patentable is genuinely contested. Critics argue many software patents are overly broad, cover "obvious" techniques, or describe abstract ideas dressed up as inventions, and that they're wielded by **patent trolls** (entities that hold patents mainly to extract licensing fees through litigation, not to build products). Different jurisdictions draw the line very differently — software patents are easier to obtain in some countries than others. You don't need to resolve the debate; you just need to know patents exist as a separate risk layer on top of copyright.

### Why patents matter for licenses

Here's the connection that makes patents relevant to *licensing*. Imagine you receive code under a license that grants you full copyright permission to use and modify it. You're safe on copyright — but a contributor to that project might *also* hold a **patent** covering what the code does. Nothing in a pure copyright license stops them from later suing you for patent infringement.

To close that gap, some licenses include an explicit **patent grant**: contributors promise not to assert their patents against users of the project. This is one of the headline differences between the bare MIT License (no explicit patent grant) and the [Apache 2.0 license](/learn/software-licensing/apache-license/) (explicit patent grant plus a defensive termination clause). The [GPL family](/learn/software-licensing/gpl-strong-copyleft/) also addresses patents. When you see "patent grant" in a license, this is the worry it's answering.

## Trademarks: protecting the name and brand

A **trademark** protects the things that identify *who a product comes from* — names, logos, slogans, distinctive branding. Its purpose is consumer protection: so that when you download "Firefox" you know it really came from Mozilla and not an impostor.

This creates a subtle but important rule for open source. **Open-source licenses generally do *not* grant trademark rights.** They let you copy, modify, and redistribute the *code*, but they specifically hold back the *name and logo*. The Apache 2.0 license is explicit about it:

```text
This License does not grant permission to use the trade names, trademarks,
service marks, or product names of the Licensor ...
```

The practical effect: you can legally fork an open-source project and ship your own modified version, but you usually **can't keep calling it by the original name** or use its logo in a way that suggests the original maintainers endorse your fork. This is why forks get renamed (think of the projects that rebranded after forking) and why you'll see notes like "this product includes software developed by X" rather than "this *is* X." Code is shareable; brand is not. The brand stays with its owner so users aren't misled.

## Trade secrets: protection through secrecy

The first three rights all involve *disclosing* something — published code, a published patent, a public brand. A **trade secret** is the opposite: it protects information precisely *because it's kept secret*.

To qualify, information generally must (1) derive value from not being publicly known and (2) be subject to reasonable efforts to keep it secret. A famous non-software example is the Coca-Cola formula. In software, a closed-source product's source code, an internal algorithm, or a clever optimization can all be trade secrets — protected indefinitely, for as long as they stay secret.

The trade-off is fragility. Trade secret protection gives you **no monopoly** over the idea — if a competitor independently invents the same thing, or lawfully reverse-engineers it, you have no claim against them. And the protection collapses entirely the moment the secret is genuinely public. That's why secrecy is rarely relied on alone. Companies reinforce it with:

- **NDAs (non-disclosure agreements)** — contracts that legally bind people who *do* see the secret to keep it confidential. (We cover NDAs in depth in [NDAs & the rest of the landscape](/learn/software-licensing/nda-and-other-agreements/).)
- **Access controls** — limiting who can see the source, and logging who does.

Keeping your source closed is itself a strategic choice on the [licensing spectrum](/learn/software-licensing/the-licensing-spectrum/): trade secret is the protection model that proprietary, closed-source software leans on most.

## How they layer together

The real power — and complexity — is that these rights stack on one product simultaneously. A single commercial application can be:

- **Copyrighted** — its source and binary can't be copied.
- **Patented** — a novel technique inside it is patent-protected.
- **Trademarked** — its name and logo identify the vendor.
- **Trade-secret-protected** — the unreleased internals stay confidential.

Even an open-source project layers them. GopherTrunk's code is licensed permissively under [Apache 2.0](/learn/software-licensing/apache-license/) (copyright *and* a patent grant given away), while — like virtually all open-source projects — its **name and any logo are not** licensed to you. Understanding which right is in play tells you what you can actually do: copy the code, yes; use a contributor's patent, yes (Apache grants it); call your fork "GopherTrunk," no.

<div class="knowledge-check" data-quiz data-correct-msg="Right — open-source licenses grant rights to the code but generally withhold trademark rights to the project's name and logo." markdown="0">
  <p class="knowledge-check__q">Quick check: you fork a project released under an open-source license. What does that license typically NOT give you?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The right to modify the source code</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The right to redistribute your modified version</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The right to use the project's name and logo for your fork</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Four IP types surround software** — copyright, patents, trademarks, and trade secrets, each protecting a different thing.
- **Patents protect inventions** — the method itself, not the code's expression; they're filed, expensive, time-limited, and controversial for software.
- **Patent grants close a gap** — a copyright license alone doesn't stop a patent claim, which is why licenses like Apache 2.0 add an explicit patent grant.
- **Trademarks protect names and logos** — and open-source licenses generally don't grant them, so forks get renamed.
- **Trade secrets protect secrets** — valuable while hidden, backed by NDAs and access controls, but lost the moment they leak or are independently discovered.
- **They layer together** — a single product can be covered by all four at once; knowing which right applies tells you what you're actually allowed to do.

Next up: a license grants permission, but is it also a contract? The distinction changes your remedies. See [License vs contract](/learn/software-licensing/license-vs-contract/).
