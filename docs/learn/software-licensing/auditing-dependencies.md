---
slug: auditing-dependencies
title: Auditing dependencies & SBOMs
description: You can't comply with licenses you don't know you're shipping. Learn how to inventory direct and transitive dependency licenses, what an SBOM is (SPDX and CycloneDX), the scanning tools to use including go-licenses for Go projects, and how to make license auditing continuous in CI.
keywords: auditing dependencies, SBOM, software bill of materials, SPDX, CycloneDX, transitive dependencies, license scanning, FOSSA, ScanCode, Syft, go-licenses, license policy CI, dependency inventory, license compliance automation
level: intermediate
status: full
faq:
  - q: "What is an SBOM and why do people suddenly care?"
    a: "An **SBOM** (Software Bill of Materials) is a machine-readable inventory of every component in your software — names, versions, and licenses — like an ingredients label. Interest spiked because supply-chain attacks and government rules (notably the US Executive Order 14028) now require SBOMs for software sold to many agencies, and because they make license and vulnerability auditing tractable. The two main formats are **SPDX** and **CycloneDX**."
  - q: "Why do I need to scan transitive dependencies — I only added a few libraries?"
    a: "Because the few libraries you added pull in *their* dependencies, which pull in more. A handful of direct dependencies routinely expands to dozens or hundreds of **transitive** ones, each with its own license. A copyleft or AGPL license can ride in three levels deep where you'd never notice it by hand — which is exactly why automated inventory exists."
  - q: "How do I keep license problems from sneaking back in?"
    a: "Make the check **continuous**, not a one-time audit. Run a license scanner in CI with a policy (allowed licenses, forbidden ones like AGPL) so a pull request that introduces a banned or unknown license **fails the build**. GopherTrunk does this — its `make licenses` target and CI `licenses` job run `go-licenses` over the whole module graph."
---
# Auditing dependencies & SBOMs

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**You can't comply blind** — compliance starts with knowing every component you ship. **Transitive is the trap** — your few direct dependencies expand into dozens of indirect ones, each licensed separately. **An SBOM is your ingredients label** — a machine-readable inventory (SPDX or CycloneDX), now often legally required. **Make it continuous** — a license scanner with a policy in CI catches problems on every pull request, not in a yearly fire drill.
</div>

Every previous lesson in this module assumed you *know* what's in your product. This one earns that assumption. The hard truth of license compliance is simple: **you cannot comply with a license you don't know you're shipping.** Modern software is assembled from hundreds of pieces you never personally chose, and any one of them can carry an obligation — or a copyleft trigger — you'd never spot by reading code. By the end of this lesson you'll know how to inventory the licenses in your dependency tree, what an SBOM is and why governments now demand one, the tools that automate the work (including the right one for a Go project like GopherTrunk), and how to turn a one-time audit into a continuous CI gate.

## The transitive dependency problem

When you add a library, you don't add one thing — you add a *tree*. Your **direct dependencies** are the ones you chose. Each of those has its own dependencies, and so on, several levels deep. These indirect ones are **transitive dependencies**, and they typically outnumber the direct ones by an order of magnitude. A project with a dozen direct dependencies can easily ship two hundred transitive ones.

Every node in that tree is separately licensed. So the real surface area of your compliance obligation is the whole tree, not the short list you typed into your manifest. A permissive MIT direct dependency might pull in an AGPL transitive dependency three levels down — and if you ship it, that obligation is yours whether or not you knew. Manual review doesn't scale to hundreds of components that change with every update. This is precisely why automated inventory exists.

## How to inventory the licenses

You don't read every LICENSE file by hand — you let the tooling do it, then review. The raw material is already there:

- **Package-manager metadata.** Every ecosystem records dependency licenses. `go.mod` plus module metadata for Go, `package.json` for npm, `pom.xml`/Maven for Java, `Cargo.toml` for Rust, `requirements`/wheel metadata for Python. Tools read this graph.
- **SPDX license identifiers.** [SPDX](https://spdx.org/licenses/) is a standardized list of short license IDs — `MIT`, `Apache-2.0`, `GPL-3.0-or-later`, `AGPL-3.0` — that make licenses machine-comparable. Most metadata and scanners speak SPDX, so you can write a policy in terms of these IDs rather than fuzzy names.
- **Source scanning** for the cases metadata misses — vendored code, files with embedded license headers, or dependencies that didn't declare a license cleanly. Deep scanners read the actual files.

The output of inventory is a list: every component, its version, and its detected license. That list is the basis for both your attribution file (the [previous lesson's](/learn/software-licensing/permissive-compliance/) `THIRD_PARTY_LICENSES`) and your SBOM.

## What an SBOM is

An **SBOM** — Software Bill of Materials — is a formal, machine-readable inventory of everything that goes into your software: components, versions, suppliers, and licenses. The analogy that sticks is the **ingredients label** on food: you can't make an informed choice about what you're consuming without knowing what's in it.

SBOMs went from nice-to-have to near-mandatory for two reasons. First, **supply-chain security**: when a vulnerability lands in a widely-used library, organizations need to answer "are we affected?" in minutes, and that's only possible if they have an inventory. Second, **regulation**: the US Executive Order 14028 (2021) directed federal agencies to require SBOMs from software vendors, and similar expectations are spreading internationally (the EU's Cyber Resilience Act pushes the same direction). If you sell software to governments or large enterprises, an SBOM is increasingly a contractual requirement, not a courtesy.

There are two dominant formats; you'll usually generate one or both:

| Format | Steward | Notes |
|--------|---------|-------|
| **SPDX** | Linux Foundation (ISO/IEC 5962 standard) | License-focused heritage, broad tool support, an international standard |
| **CycloneDX** | OWASP | Security-focused heritage, lightweight, strong vulnerability/VEX support |

Both can express components, versions, and licenses; pick based on what your customers or tools expect, or emit both since most generators can.

## License-scanning tools

A healthy ecosystem of tools turns inventory and SBOM generation into a command. A representative sampling:

| Tool | What it does | Notes |
|------|--------------|-------|
| **Syft** | Generates SBOMs (SPDX/CycloneDX) from images and source | Fast, pairs with Grype for vulnerabilities |
| **ScanCode Toolkit** | Deep source-level license & copyright detection | Open source, very thorough, catches embedded headers |
| **FOSSology** | License scanning + compliance workflow server | Open source, good for a review process |
| **FOSSA** | Hosted dependency & license compliance platform | Commercial, policy + CI integration |
| **license-checker** | Lists licenses of an npm dependency tree | Lightweight, JavaScript ecosystem |
| **go-licenses** | Reports licenses of a Go module's full dependency graph | The natural fit for Go projects |

The split worth understanding: **metadata-based tools** (license-checker, go-licenses) read the package graph and are fast and accurate when licenses are declared cleanly; **deep scanners** (ScanCode, FOSSology) read the actual files and catch what metadata misses, at the cost of speed. SBOM generators like **Syft** sit in between and produce the standardized artifact. Many teams use a fast metadata gate in CI and a deep scan periodically.

### go-licenses for a Go project like GopherTrunk

For a Go codebase, **`go-licenses`** (from Google) is the idiomatic choice: it walks the module graph and reports the license of every package the binary links in, transitive ones included. This isn't hypothetical for GopherTrunk — it's exactly what the project uses. The `make licenses` target runs `go-licenses` to regenerate the transitive inventory, and the **CI `licenses` job** runs the same target so a newly-introduced dependency with a non-permissive license fails the build. That machine-generated graph backstops the hand-curated [`/THIRD_PARTY_LICENSES.md`](/learn/software-licensing/permissive-compliance/), so the attribution file and the actual dependency tree can't silently drift apart.

```bash
# Inventory every package's license across the whole module graph
go-licenses csv ./... > THIRD_PARTY_LICENSES.csv

# Or fail loudly if a forbidden license type shows up
go-licenses check ./... --disallowed_types=forbidden,restricted
```

## Setting a policy and making it continuous

An inventory is only useful against a **policy** — a decision, made once, about which licenses are acceptable:

- **Allowed** — typically the permissive set: MIT, BSD, Apache-2.0.
- **Review required** — weak copyleft (LGPL, MPL) that's usable with care.
- **Forbidden** — most commonly **AGPL** (the [SaaS trigger](/learn/software-licensing/copyleft-in-products/)), sometimes strong GPL for closed products, and anything with an **unknown or missing** license (which is the most dangerous case — no license means no permission).

Writing the policy in terms of [SPDX identifiers](/learn/software-licensing/license-compatibility/) makes it enforceable by machine. And the key move is to make enforcement **continuous, not a fire drill.** A one-time audit is stale the moment someone updates a dependency. Instead, run the scanner in CI so that any pull request introducing a forbidden, unknown, or newly non-compliant license **fails the build** before it merges. That turns license compliance from an annual scramble into a property the codebase maintains automatically — which is exactly the posture GopherTrunk takes with its `licenses` CI job.

<div class="knowledge-check" data-quiz data-correct-msg="Right — running a license scanner with a policy in CI catches a forbidden or unknown license on the pull request that introduces it, before it ships." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the most reliable way to keep a banned license out of your product?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Audit the dependency list by hand once a year</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Run a license scanner with a policy in CI so non-compliant pull requests fail the build</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Only review the direct dependencies you added yourself</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Compliance starts with knowing** — you cannot meet obligations for components you don't know you ship.
- **Transitive dependencies dominate** — a few direct dependencies expand into dozens or hundreds of indirect ones, each separately licensed.
- **Inventory from metadata and SPDX** — package-manager data plus SPDX identifiers let tools build a machine-readable license list, with deep scanners for what metadata misses.
- **An SBOM is the standardized inventory** — SPDX or CycloneDX, now widely required by governments and enterprises for security and compliance.
- **Use the right tool** — Syft, ScanCode, FOSSology, FOSSA, license-checker; for Go, `go-licenses`, which GopherTrunk runs in CI.
- **Make it continuous** — a license policy enforced in CI fails non-compliant pull requests, turning a yearly fire drill into an automatic guardrail.

Next up: licenses aren't the only agreements you'll meet — now turn to reading the contracts themselves. See [How to read a software agreement](/learn/software-licensing/how-to-read-an-agreement/).
