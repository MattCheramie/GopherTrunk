---
slug: open-source-risks
title: The risks of depending on open source
description: Depending on open source carries real risks beyond "is it free" — license rug pulls, maintainer burnout and abandonment, supply-chain attacks, transitive license surprises, and audit exposure. Learn each risk and how to mitigate it.
keywords: open source risks, license rug pull, Redis SSPL, HashiCorp BSL, Elastic relicensing, maintainer burnout, supply chain attack, xz backdoor, typosquatting, transitive dependencies, SBOM, dependency pinning, vendoring, open source compliance risk
level: intermediate
status: full
faq:
  - q: "What is a license 'rug pull' in open source?"
    a: "It's when a project that built its user base as open source switches to a restrictive license, so existing users can no longer use new versions under the old terms. Redis moving to SSPL, HashiCorp's Terraform moving to the BSL, and Elastic's relicensing are the well-known examples. The old versions usually stay under the old license, but future development moves behind the new one."
  - q: "How can a free dependency become a security risk?"
    a: "Open-source packages are a prime **supply-chain** target. Attackers publish typosquatted package names, hijack abandoned packages, or — as in the **xz** backdoor — patiently gain maintainer trust and slip in malicious code. Because you pull dependencies automatically, a compromised package runs with your application's privileges."
  - q: "How do I reduce open-source dependency risk without avoiding open source?"
    a: "**Pin** exact versions so you control upgrades, consider **vendoring** critical dependencies so you're not at the mercy of a registry, generate an **SBOM** to know exactly what you ship, and audit licenses regularly. The goal isn't to avoid open source — it's to know what you depend on and stay able to comply."
---
# The risks of depending on open source

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The risk isn't only "is it free"** — license changes, abandonment, and security all matter as much as price. **Rug pulls happen** — vendors relicense projects to restrictive terms (Redis, HashiCorp, Elastic), stranding users. **Supply-chain risk is real** — typosquatting, hijacked packages, and the xz backdoor show how a free dependency can carry an attack. **Mitigate, don't avoid** — pin versions, vendor critical code, generate SBOMs, and audit licenses to stay in control and in compliance.
</div>

Open source is one of the best deals in software, but "free to use" is not the same as "free of risk." Depending on someone else's code means inheriting their licensing decisions, their maintenance energy, and their security. This lesson surveys the risks that actually bite teams — license rug pulls, abandonment, supply-chain attacks, transitive license surprises, and audit exposure — and the concrete habits that keep those risks manageable. By the end you'll be able to depend on open source with your eyes open rather than your fingers crossed.

## License changes and "rug pulls"

The first risk is the one this whole path equips you to see: the license can **change**. A project can be released under a permissive or copyleft license for years, build a huge user base, and then — usually when a company decides it needs to monetize or defend against cloud competitors — switch to far more restrictive terms.

Recent, high-profile examples:

- **Redis** moved from BSD to a dual SSPL/RSAL scheme (and the community forked it as Valkey).
- **HashiCorp** moved Terraform and its other tools from the Mozilla Public License to the **Business Source License (BSL)** (the community forked Terraform as OpenTofu).
- **Elastic** relicensed Elasticsearch from Apache-2.0 to SSPL/Elastic License (Amazon forked it as OpenSearch).

These are often called **rug pulls** because users who adopted the software as open source suddenly can't use new versions under the old terms. Typically the *old* releases stay under the *old* license — copyright already granted can't be revoked — but all future development moves behind the new, restrictive license. If you depend on the project, you're stuck choosing between an aging fork, accepting the new terms, or migrating.

The terms these vendors move *to* are usually **source-available** licenses — you can read the source, but you can't use it freely (often with a "no competing service" clause). That category is important enough to get its own lesson: [Source-available & "fair source"](/learn/software-licensing/source-available-licenses/). The mitigation here is awareness: prefer projects under standard licenses with broad governance (e.g. foundation-stewarded projects are far less likely to rug-pull than single-vendor ones), and know that any single-company open-source project *can* relicense — that's the [relicensing](/learn/software-licensing/dual-licensing-and-relicensing/) power we just studied, pointed at users.

## Abandonment and maintainer burnout

The second risk is quieter: the project just **stops**. A huge amount of critical infrastructure is maintained by one or a few unpaid people, and burnout, life changes, or loss of interest can end maintenance with no notice. The dependency doesn't disappear — it simply stops getting bug fixes, security patches, and compatibility updates.

An abandoned dependency is a slow-motion problem: it works until a security flaw is found, a platform changes underneath it, or a transitive dependency breaks — and now *you* own a fix for code you didn't write. Warning signs to check before you adopt: when was the last commit? How many maintainers? Is there a release cadence? A single-maintainer package with no activity in two years is a liability even if it works today.

Mitigation is partly about choosing well-supported dependencies and partly about being ready to step in — which is much easier if you've **vendored** the code (kept a copy in your own tree) so you can patch it yourself if you must.

## Supply-chain and provenance risk

The third risk turns a free dependency into an attack vector. Because modern projects pull packages automatically from public registries, the registry — and every package in it — is part of your **supply chain**. Attackers exploit this in several ways:

- **Typosquatting.** A malicious package is published under a name one keystroke away from a popular one (`reqeusts` for `requests`), hoping you'll fat-finger the install.
- **Hijacked / compromised packages.** An attacker gains control of an existing trusted package — through stolen credentials or by taking over an abandoned one — and pushes a malicious update that flows straight into everyone who upgrades.
- **The long con.** The **xz Utils backdoor** (2024) is the cautionary tale: an attacker spent *years* building maintainer trust on a widely used compression library, then slipped in a backdoor targeting SSH. It was caught almost by accident. It showed that even careful provenance can be subverted by patience.

Because dependencies run with your application's privileges, a compromised one can read your secrets, exfiltrate data, or open a backdoor. **Provenance** — knowing where a package came from, who published it, and that it hasn't been tampered with — is the defense. In practice that means pinning to known-good versions, verifying checksums/signatures where available, and not blindly auto-upgrading.

## Transitive dependency license surprises

The fourth risk is a licensing one. You carefully choose a permissive direct dependency — but *it* depends on something else, which depends on something else. Your real license exposure is the **whole transitive tree**, and a copyleft or source-available license can hide several layers down.

A classic surprise: you ship a closed-source product, your direct dependencies are all MIT/Apache, but one of them transitively pulls in a GPL library. Per [license compatibility](/learn/software-licensing/license-compatibility/), that GPL component can force obligations onto your distributable work that you never intended to take on. You can't manage what you can't see, which is the whole reason for systematic dependency auditing.

## Legal and audit exposure

The fifth risk is the consequence of all the above: if you ship software you can't legally comply with, you carry **audit and legal exposure.** Missing attribution, an undetected copyleft obligation, or a source-available license used outside its terms can surface during a security audit, an acquisition due-diligence review, or a customer's compliance check — exactly when it's most expensive. Acquirers routinely scan a target's dependency licenses; a single unmanaged copyleft component can reduce a company's valuation or sink a deal.

This isn't hypothetical or rare; it's a standard part of corporate due diligence. The cost of getting it wrong is paid later and at a bad moment, which is why getting it right is cheap insurance.

## Mitigation: stay in control of what you depend on

None of this is an argument against open source — it's an argument for *managing* it. The core habits:

| Practice | What it protects against |
|---|---|
| **Pin exact versions** | Surprise upgrades, hijacked-update supply-chain attacks |
| **Vendor critical dependencies** | Abandonment, registry outages, rug pulls (you keep a working copy) |
| **Generate an SBOM** (software bill of materials) | Transitive license/security surprises — you know exactly what you ship |
| **Audit licenses regularly** | Compliance and acquisition-time legal exposure |
| **Prefer well-governed projects** | Rug pulls and single-maintainer abandonment |
| **Verify provenance** (checksums, signatures) | Typosquatting and tampered packages |

```bash
# GopherTrunk keeps a license inventory and attribution file so
# transitive surprises and compliance gaps surface in CI, not in an audit.
make licenses
```

The throughline: know what you depend on, control when it changes, and keep enough of a copy and a record that a vendor's decision can't strand you. The mechanics of doing this systematically — generating SBOMs, scanning the dependency tree, producing an attribution file like GopherTrunk's `/THIRD_PARTY_LICENSES.md` — are covered in depth in [Auditing dependencies & SBOMs](/learn/software-licensing/auditing-dependencies/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — Redis, HashiCorp, and Elastic all relicensed established open-source projects to restrictive terms, stranding users on the old versions." markdown="0">
  <p class="knowledge-check__q">Quick check: what is a license "rug pull"?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A security flaw that lets an attacker revoke your license remotely</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">When a project relicenses to restrictive terms, so users can't use new versions under the old open-source license</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A bug where a dependency silently downgrades to an older, vulnerable version</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Look past "is it free"** — license changes, abandonment, and security are as real as price.
- **Rug pulls happen** — Redis, HashiCorp, and Elastic relicensed to restrictive terms; single-vendor projects are the ones most likely to do it.
- **Abandonment is a slow-motion risk** — an unmaintained dependency works until a security flaw or platform change makes it your problem.
- **Open source is a supply-chain target** — typosquatting, hijacked packages, and the xz backdoor show how a dependency can carry an attack.
- **Your exposure is the whole transitive tree** — copyleft or source-available licenses can hide several layers down and surface at audit time.
- **Mitigate, don't avoid** — pin versions, vendor critical code, generate SBOMs, audit licenses, and prefer well-governed projects.

Next up: we shift from open source to the other side of the spectrum — how closed, commercial software is licensed. See [Proprietary & closed-source licensing](/learn/software-licensing/proprietary-licensing/).
