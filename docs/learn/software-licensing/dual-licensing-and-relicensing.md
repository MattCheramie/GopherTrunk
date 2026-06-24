---
slug: dual-licensing-and-relicensing
title: Dual licensing & relicensing
description: Offering the same code under two licenses at once — GPL for the community and a paid commercial license for everyone else — is the classic open-core business model. Learn how it works, why you must own all the rights, and why relicensing an existing project is so hard.
keywords: dual licensing, relicensing, open core, GPL commercial license, MySQL dual license, Qt dual license, copyright assignment relicensing, license proliferation, dual license business model, contributor rights relicensing, changing a project license
level: advanced
status: full
faq:
  - q: "How can the same code be GPL and also sold under a commercial license?"
    a: "The **copyright holder** can offer their code to different people under different terms. They release it under the GPL for anyone who accepts copyleft, and sell a separate **commercial license** to companies that can't or won't comply with the GPL. Only the rights owner can do this, because only they can grant licenses other than the one already published."
  - q: "Why do I need a CLA to dual-license my project?"
    a: "To offer the code under a *second* license, you must hold the rights to **all** of it. The moment outside contributors add GPL'd code, you no longer own the whole work — so you can't relicense it commercially unless every contributor granted you that right, which is exactly what a broad [CLA](/learn/software-licensing/contributor-agreements/) collects."
  - q: "Can I just change my open-source project's license whenever I want?"
    a: "Only if you own all the copyright. If others have contributed under the old license without a CLA, you need **every** contributor's agreement to relicense — which is often impractical for a large, old project. This is why relicensing campaigns are rare, slow, and sometimes impossible."
---
# Dual licensing & relicensing

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Dual licensing = one codebase, two licenses** — typically GPL for the community plus a paid commercial license for those who can't accept copyleft. **It powers open core** — the classic MySQL/Qt model that funds open-source development from commercial sales. **You must own all the rights** — only the copyright holder can offer extra licenses, so you need to own the code or have CLA grants from every contributor. **Relicensing is hard** — changing an existing project's license needs every contributor's agreement, which is often impractical.
</div>

The previous lesson showed how a [CLA](/learn/software-licensing/contributor-agreements/) collects broad rights from contributors. This lesson shows the most powerful thing those rights unlock: offering the very same code under *two* licenses at once, and the related — and much harder — act of changing a project's license after the fact. By the end you'll understand the dual-licensing business model, the strict ownership requirement behind it, why relicensing an established project is so difficult, and how all of this connects to the open-core companies you've heard of.

## What dual licensing is

**Dual licensing** (sometimes multi-licensing) means the copyright holder releases the *same* code under more than one license, and each user picks the one that fits them. The classic pattern pairs a strong copyleft license with a commercial one:

- **A copyleft license — usually the [GPL](/learn/software-licensing/gpl-strong-copyleft/) — for the community.** Anyone can use the code for free, as long as they accept the GPL's terms: derivatives they distribute must also be GPL, with source.
- **A paid commercial license for everyone who can't accept that.** A company that wants to embed the code in a closed-source product *cannot* comply with the GPL without opening their own product. So they buy a commercial license from the copyright holder that grants the same code under proprietary-friendly terms — no copyleft obligations.

The clever part is that the GPL does the selling for you. Because the GPL would force a commercial user to open their product, it creates demand for the paid escape hatch. The community gets genuinely free software; the vendor gets paying customers among those who need to stay closed. The two licenses aren't in conflict — they're offered to *different audiences* by the one party allowed to offer them.

This is exactly the **open-core / commercial open-source** model behind famous examples: **MySQL** was dual-licensed (GPL plus a commercial license for embedding in closed products), and **Qt** has long offered both an open-source license and a paid commercial one. The community edition drives adoption and contributions; the commercial license funds the company.

## Why you must own (or have rights to) all the code

Here is the iron rule of dual licensing: **only the copyright holder can grant a license.** To offer the code under a *second*, commercial license, you must hold the rights to **every line** you're relicensing. If you don't own a part, you can't grant a non-GPL license to it — only its actual owner can.

This is fine when you're the sole author. It breaks the instant outside contributors add code:

- A contributor sends a GPL'd patch under plain inbound=outbound. You now have GPL'd code in your tree that *you don't own.*
- You can still ship the combined work under the GPL — that's allowed. But you **cannot** offer that contributor's code under your commercial license, because they never granted you that right.
- One un-owned line in a file is enough to poison your ability to commercially license that file.

The solution is the [CLA](/learn/software-licensing/contributor-agreements/) from the previous lesson. A broad license-grant CLA (or a copyright-assignment CLA) collects, from every contributor, the right for you to license their contribution under *other* terms — including commercially. That's why companies running a dual-licensed project almost always require a CLA: without it, every external contribution would quietly erode the very thing the business model depends on. This is also a concrete example of the contributor-side caution from the last lesson — by signing that CLA, contributors are enabling the company to sell their volunteer work under closed terms.

## Relicensing an existing project

**Relicensing** is changing the license a project is released under. It's governed by the same iron rule, which makes it one of the hardest things to do in open source.

To relicense, you need permission from **every copyright holder** in the codebase:

- **If you own all the code** (sole author, or everyone signed a CLA/assignment), you can relicense at will — you're just choosing different terms for code you own.
- **If contributors hold rights** under the old license with no CLA, you must get **each one** to agree to the new license. For a project with hundreds of contributors over many years — some unreachable, some deceased, some who refuse — this can be slow, expensive, or simply impossible.

| Situation | Can you relicense? |
|---|---|
| You are the sole author | Yes — you own everything |
| All contributors signed a broad CLA / assignment | Yes — they granted you the right |
| Contributors used "or later" version clauses | Partially — you may move to a newer version of the same license |
| Many contributors, plain inbound=outbound, no CLA | Only with every contributor's explicit agreement — often impractical |

Real campaigns show how hard this is. Mozilla spent years and tracked down contributors to relicense Firefox's code to its tri-license. Some projects have rewritten contested code from scratch rather than chase down an unreachable contributor. The practical lesson: **decide your licensing strategy early**, because reversing it later may be off the table.

A gentler form is moving between *versions* of the same license. Code released as "GPLv2 **or later**" can be used under GPLv3 by downstream users without re-collecting permission — the "or later" clause grants that flexibility up front. This is one reason license version choices matter, as we saw in [license compatibility](/learn/software-licensing/license-compatibility/).

## License proliferation, the cost side

Dual licensing and bespoke commercial terms feed a broader problem the open-source community calls **license proliferation**: too many slightly different licenses. Every non-standard license a project introduces adds [compatibility](/learn/software-licensing/license-compatibility/) uncertainty, more work for legal reviewers, and more confusion for users trying to combine software.

This is one more argument for using **standard** licenses for the open side of a dual-licensing scheme (a well-known copyleft license like the GPL or AGPL), keeping only the *commercial* license custom — because the commercial license is a private contract between you and one buyer, not something that has to compose with the wider ecosystem. Reusing a standard copyleft license for the public edition keeps your project compatible and recognizable while the paid license does the bespoke work.

## The open-core model, previewed

Dual licensing is one engine of a bigger pattern: **open core**, where a company builds an open-source core and sells something extra around it — a commercial license, proprietary add-ons, hosting, or support. The dual-license version we covered here is the "sell a commercial license to the same code" flavor, but open core has several shapes, and it shades into the source-available licensing that newer companies have adopted.

That whole business landscape — how companies actually make money from software they also give away, and the trade-offs of each model — is the subject of [Commercial license models](/learn/software-licensing/commercial-license-models/) in the next module. For now, hold onto the core mechanic: dual licensing works because *one party owns the rights and can therefore offer the same code on more than one set of terms.*

<div class="knowledge-check" data-quiz data-correct-msg="Right — only the copyright holder can grant additional licenses, so dual licensing requires owning all the code or holding CLA rights to every contribution." markdown="0">
  <p class="knowledge-check__q">Quick check: why can a company dual-license its project but a random fork usually can't?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Only the copyright holder can offer code under additional licenses, and the company owns or has CLA rights to all of it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The GPL specifically allows the original author to add commercial terms but forbids forks from doing anything</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Dual licensing only requires registering the project with the OSI first</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Dual licensing offers one codebase under two licenses** — typically GPL for the community plus a paid commercial license for those who can't accept copyleft.
- **The GPL drives the sale** — because copyleft would force closed users to open their product, they buy the commercial escape hatch.
- **It powers open core** — MySQL and Qt are the classic examples of funding open-source work through commercial licensing.
- **You must own all the rights** — only the copyright holder can grant extra licenses, so dual licensing requires sole authorship or broad CLA grants from every contributor.
- **Relicensing is hard** — changing an existing project's license needs every contributor's agreement, which is often impractical, so decide early.
- **Standardize the open side** — reuse a well-known copyleft license publicly to avoid license proliferation; keep only the commercial license bespoke.

Next up: depending on open source carries risks beyond price — including the relicensing moves we just saw, turned against *users*. See [The risks of depending on open source](/learn/software-licensing/open-source-risks/).
