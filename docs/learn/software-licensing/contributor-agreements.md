---
slug: contributor-agreements
title: "Contributor agreements: CLAs & DCO"
description: When others contribute code, who owns and licenses it? Learn the inbound=outbound default, what a CLA grants a project, how the DCO sign-off works, and what contributors should understand before signing either.
keywords: contributor license agreement, CLA, DCO, developer certificate of origin, inbound equals outbound, copyright assignment, signed-off-by, contributor agreement, contributing to open source, relicensing rights, who owns contributions
level: intermediate
status: full
faq:
  - q: "If a project has no CLA, who owns the code I contribute?"
    a: "You keep your copyright, and by contributing you license your work to the project under the project's own license — the **inbound=outbound** default. Your code goes in under the same terms it goes out under (e.g. Apache-2.0 in, Apache-2.0 out). No rights transfer beyond that license unless a CLA says so."
  - q: "What's the difference between a CLA and a DCO?"
    a: "A **CLA** is a legal agreement where you grant the project specific rights (and sometimes assign copyright) — it's heavyweight and enables things like relicensing. A **DCO** is a lightweight *certification*: you add a `Signed-off-by` line attesting you have the right to submit the code under the project's license. The DCO grants no extra rights; it just records a promise."
  - q: "Should I worry about signing a company's CLA?"
    a: "Read it. A broad CLA can let the company **relicense or dual-license** your contribution however it wants, including in closed commercial products — you're granting that right. That may be perfectly fine, but understand you might be handing a company broad rights to code you wrote for free before you click 'agree.'"
---
# Contributor agreements: CLAs & DCO

> **This is educational material, not legal advice.** For decisions that carry real risk, consult a qualified attorney.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Inbound=outbound is the default** — without any extra agreement, contributions come in under the project's own license. **A CLA grants the project rights** — sometimes broad, sometimes copyright assignment; it's what enables relicensing and dual-licensing. **A DCO is a lightweight sign-off** — a `Signed-off-by` line certifying you have the right to contribute, granting no extra rights. **Contributors should read before signing** — a CLA can hand a company broad rights to code you wrote for free.
</div>

The moment someone else sends a patch to your project, a question appears that didn't exist when you were the only author: who owns that code, and under what terms can the project use it? This lesson covers the three answers in practice — the silent default, the Contributor License Agreement (CLA), and the Developer Certificate of Origin (DCO) — from both sides of the table. By the end you'll know what each mechanism does, why a maintainer might want one, and — just as important — what you as a contributor are agreeing to when you sign.

## The default: inbound=outbound

Suppose your project is [Apache-2.0](/learn/software-licensing/apache-license/) and a stranger opens a pull request. If you have *no* separate agreement in place, what governs their contribution?

The widely accepted default is **inbound=outbound**: contributions come *in* under the same license the project goes *out* under. Phrased by the Apache and open-source community as a norm, it means a contributor who submits code to your Apache-2.0 project is — by the act of contributing — licensing that code to the project under Apache-2.0. The contributor keeps their copyright; they simply grant the project (and everyone downstream) the same license everyone else already has.

GitHub's own terms of service codify this for public repos: absent a different agreement, contributions are licensed under the repository's existing license. Many projects also state it explicitly in `CONTRIBUTING.md`. The Apache-2.0 license even has section 5, which says contributions are deemed to be under the license's terms unless stated otherwise — so an Apache project gets a clean inbound grant essentially for free.

The thing to understand about inbound=outbound: it grants the project *exactly* the project's license and **no more**. That's usually all a healthy open-source project needs. But it means the project can't later relicense the code, because each contributor only granted the existing license — not the right to change it. That limitation is the reason CLAs exist.

## The CLA: a Contributor License Agreement

A **CLA** is a separate legal agreement a contributor signs (often once, electronically, before their first merge) that grants the project specific rights to their contributions. CLAs come in two main flavors:

- **License-grant CLA.** The contributor keeps copyright but grants the project a broad, irrevocable license — including, critically, the right to *relicense* and to grant patent rights. The Apache Software Foundation's Individual CLA is the model here. This lets the project sublicense and combine contributions flexibly.
- **Copyright-assignment CLA.** The contributor *transfers ownership* of the contribution's copyright to the project or its sponsoring entity. The strongest version — the entity now owns the code outright. The Free Software Foundation historically required this for GNU projects.

Why would a maintainer or company want a CLA? A few reasons:

- **Relicensing flexibility.** With broad rights from every contributor, the project can move to a new license version, or **dual-license** the code — for example, offering it as both GPL and a paid commercial license. (That model is the subject of the next lesson, [Dual licensing & relicensing](/learn/software-licensing/dual-licensing-and-relicensing/).)
- **Legal certainty / enforcement.** A signed CLA gives the project a clear, documented chain of rights — useful if it ever has to defend the license or prove it had permission to use a contribution.
- **Patent clarity.** Many CLAs include an explicit patent grant, so a contributor can't later claim a patent over code they donated.

The cost is real friction: contributors must sign before they can help, which deters drive-by fixes, and a corporate CLA can feel — and be — like asking volunteers to sign over rights to a company.

## The DCO: Developer Certificate of Origin

The **DCO** is the lightweight alternative, created for the Linux kernel after the SCO lawsuits. It is **not** a rights grant — it's a *certification*. By adding a one-line `Signed-off-by` to each commit, the contributor attests to a short, fixed statement: that they wrote the code (or have the right to submit it) and that they're contributing it under the project's license.

```text
Signed-off-by: Jane Developer <jane@example.com>
```

That line, added with `git commit -s`, is the entire mechanism. It points to the published DCO text (version 1.1) and records the contributor's promise that the code is theirs to give. There's no separate document to sign, no account to register, no copyright transfer. The contribution still flows in under the project's license via inbound=outbound — the DCO just adds an auditable assertion of provenance on top.

The DCO grants the project **no extra rights** — no relicensing power, no copyright assignment. That's the point. It gives a maintainer a record of who certified what, without asking contributors to give up anything beyond the normal inbound license.

## CLA vs DCO: a side-by-side

| Aspect | CLA | DCO |
|---|---|---|
| What it is | A signed legal agreement | A per-commit certification line |
| Rights granted to project | Broad license, possibly copyright assignment | None beyond inbound=outbound |
| Enables relicensing / dual-licensing | Yes (that's a main purpose) | No |
| Contributor effort | Sign a document, often via a bot | `git commit -s` adds one line |
| Friction for casual contributors | Higher — deters one-off fixes | Low |
| Who uses it | Google, Apache (ASF projects), many corporate-backed projects | Linux kernel, GitLab, many CNCF projects |
| Main downside | Can grant a company broad rights to volunteer work | Doesn't give the project relicensing flexibility |

Neither is "better" in the abstract. A single-company project that may want to dual-license later leans CLA. A community project that values low contribution friction and doesn't plan to relicense leans DCO — or just relies on inbound=outbound with nothing extra at all.

## What contributors should understand before signing

Most lessons on this topic are written for maintainers. Here's the contributor's side, because it matters:

- **A DCO costs you nothing extra.** You're certifying you have the right to submit the code under the project's license. If that's true, sign off freely.
- **A CLA can grant a lot.** Read it before agreeing. A broad license-grant CLA may let the project — often a company — **relicense your contribution into closed, paid products**. A copyright-assignment CLA means you no longer own the code you wrote. That can be entirely reasonable (it's how many sustainable projects fund themselves), but go in with eyes open.
- **Watch for asymmetry.** Some CLAs let the company do things with your code that the open-source license wouldn't let *you* do with theirs. You're contributing to a commons; under a broad CLA the company may be contributing to its own commercial product.
- **Check what you're allowed to contribute.** If your employer owns your code (common under employment agreements), you may need their permission before you can truthfully sign either a DCO or a CLA. This intersects with [copyright and ownership](/learn/software-licensing/copyright-and-software-ownership/).

None of this means "don't sign." It means *read the specific agreement*, understand whether you're merely certifying provenance (DCO) or granting broad rights (CLA), and decide knowingly.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a DCO is a per-commit certification of provenance and grants the project no rights beyond the normal inbound=outbound license." markdown="0">
  <p class="knowledge-check__q">Quick check: what does signing a DCO with `Signed-off-by` actually do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Transfers the copyright of your contribution to the project</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Grants the project the right to relicense your code commercially</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Certifies that you have the right to submit the code under the project's license</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Inbound=outbound is the silent default** — without an agreement, contributions come in under the project's own license, and the contributor keeps copyright.
- **A CLA grants the project rights** — a broad license or even copyright assignment, which is what enables relicensing and dual-licensing.
- **A DCO is a sign-off, not a grant** — a `Signed-off-by` line certifying provenance, granting no extra rights and adding minimal friction.
- **Maintainers choose based on intent** — CLA for relicensing flexibility and legal certainty; DCO (or nothing extra) for low-friction community contribution.
- **Contributors should read before signing** — a DCO is cheap to grant, but a broad CLA can hand a company sweeping rights to code you wrote for free.
- **Employment can complicate it** — if your employer owns your work, you may need permission before you can honestly sign either.

Next up: the relicensing flexibility a CLA unlocks has a powerful business use — offering the same code under two licenses at once. See [Dual licensing & relicensing](/learn/software-licensing/dual-licensing-and-relicensing/).
