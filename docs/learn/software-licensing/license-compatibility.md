---
slug: license-compatibility
title: License compatibility
description: License compatibility is whether code under two different licenses can be legally combined into one distributable work. Learn why permissive licenses combine freely, how copyleft constrains the result, and how to check before adding a dependency.
keywords: license compatibility, compatible licenses, combining open source licenses, GPL compatibility, Apache GPLv3, GPLv2 GPLv3 incompatible, one-way compatibility, most restrictive license, copyleft compatibility, license of the combined work, permissive licenses
level: intermediate
status: full
faq:
  - q: "What does it mean for two licenses to be 'compatible'?"
    a: "It means code under license A and code under license B can be legally combined into a single distributable work without violating either license. Compatibility is about *combining and shipping*, not about whether you personally like the terms. Two perfectly reasonable licenses can still be incompatible because their conditions clash."
  - q: "Can I put a GPL library into my MIT-licensed proprietary app?"
    a: "Not while keeping the app proprietary. The GPL is **copyleft**: combining GPL code into your work generally requires you to license the *whole* combined work under the GPL and share its source. You can pull MIT code into a GPL project, but not the other way into closed source — that's the **one-way** nature of copyleft."
  - q: "Are GPLv2 and GPLv3 compatible with each other?"
    a: "Generally **no**, unless the GPLv2 code says 'version 2 or later' (`GPL-2.0-or-later`). Plain GPLv2-only code and GPLv3 code cannot be combined, because each license forbids adding the other's extra conditions. This is one of the most famous incompatibilities in open source."
---
# License compatibility

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Compatible = combinable** — two licenses are compatible if their code can be merged into one work you can legally distribute. **Permissive combines freely** — MIT, BSD, and Apache impose few conditions, so they slot into almost anything. **Copyleft is the constraint** — the GPL family flows one way: you can pull permissive code *into* the GPL, but not GPL code into a permissive or proprietary work. **The result takes the strictest terms** — the combined work must satisfy every license at once, which usually means the most restrictive one wins.
</div>

By now you know the difference between [permissive and copyleft](/learn/software-licensing/permissive-vs-copyleft/) licenses. This lesson is about what happens when you mix them. The moment you pull a dependency into your project, you're combining your code's license with theirs — and not every pair of licenses can legally be combined. By the end you'll know exactly what "compatible" means, why the direction of the combination matters, the specific pairings that trip people up, and how to check a license before you add the dependency.

## What "compatible" actually means

Two licenses are **compatible** when you can take code released under one and code released under the other, combine them into a single work, and distribute that work without breaking either license's terms. That's the whole definition — and notice what it's *not* about. It is not about whether the licenses are similar, whether you approve of them, or whether they're both "open source." It's strictly about whether their conditions can all be satisfied at the same time in one shipped artifact.

Conditions clash when one license *requires* something another license *forbids*. If license A says "you may add no further restrictions" and license B says "you must add this restriction," there is no way to obey both in the same combined work. They're incompatible — not because either is bad, but because their rules contradict each other.

A few clarifications worth nailing down:

- Compatibility matters most when you **combine and distribute** a work — linking a library, copying a file, bundling source. Merely *using* a tool, or running unmodified copies side by side, is a much lighter question.
- "Distribute" includes shipping a binary, a container image, or a downloadable app. (Whether putting software on a *network* counts is the special twist the [AGPL](/learn/software-licensing/agpl-network-copyleft/) adds.)
- Compatibility is asymmetric for copyleft licenses. The order you combine things in changes the answer, which is the single most counterintuitive part — so it gets its own section.

## Permissive licenses combine freely

[Permissive licenses](/learn/software-licensing/mit-and-bsd-licenses/) — MIT, BSD, and [Apache 2.0](/learn/software-licensing/apache-license/) — ask for very little: keep the copyright notice and license text, and (for Apache) respect the patent and trademark terms. They do **not** demand that the larger work adopt their license. Because they impose almost no conditions on the combined work, they slot into nearly anything: another permissive project, a copyleft project, or closed-source proprietary software.

This is why permissive code is everywhere. You can take an MIT-licensed utility and drop it into a commercial product, a GPL application, or another MIT library, and in each case you satisfy the license simply by preserving its notice. GopherTrunk itself is [Apache-2.0](/learn/software-licensing/apache-license/) licensed and depends on a stack of permissively licensed Go modules; that breadth of compatibility is a big part of why permissive licensing dominates the dependency ecosystem.

The one wrinkle inside the permissive family is Apache 2.0's explicit patent and notice clauses. Older permissive licenses (4-clause BSD, and Apache 2.0 against GPL**v2**) have specific incompatibilities — we'll get to the Apache case below — but as a rule, permissive-to-permissive combinations just work.

## Copyleft is the constraint — and it flows one way

[Copyleft licenses](/learn/software-licensing/gpl-strong-copyleft/) like the GPL flip the deal. They say: you may use and modify this code, but if you distribute a work built from it, that whole work must be released under the *same* copyleft license, with source available. That requirement is what creates the compatibility problem, because it imposes a condition on the *combined* work, not just on the original code.

Here is the key mental model — **one-way compatibility**:

- **Permissive → copyleft: allowed.** You can take MIT or BSD code and pull it *into* a GPL project. The permissive license doesn't object to the result being GPL'd, and the GPL is happy to absorb code that came with looser terms. The combined work ships as GPL.
- **Copyleft → permissive or proprietary: not allowed.** You cannot take GPL code and fold it into an MIT-licensed library or a closed-source product while keeping that looser/closed license. The GPL would require the whole combined work to *become* GPL, which contradicts the permissive or proprietary licensing you wanted.

So compatibility has a *direction*. Code flows "uphill" from permissive into copyleft, but copyleft code can't flow "downhill" into something more permissive. People get burned when they assume "it's all open source, so it must mix" — adding one GPL dependency can force your entire distributable work to become GPL, or block the combination entirely if you can't accept that.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 186" role="img" aria-label="Three license boxes in a row: permissive licenses like MIT, BSD, and Apache on the left, a copyleft GPL project in the middle, and a proprietary or permissive work on the right. A solid arrow marked allowed flows from permissive into the GPL, while a barred, crossed-out arrow marked blocked shows GPL code cannot flow out into the proprietary or permissive work." xmlns="http://www.w3.org/2000/svg">
  <text x="230" y="26" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">One-way compatibility</text>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="16" y="64" width="118" height="56" rx="6" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.3"/>
    <text x="75" y="88" font-weight="600">Permissive</text><text x="75" y="102" font-size="7.5">MIT · BSD · Apache</text>
    <rect x="172" y="64" width="112" height="56" rx="6" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/>
    <text x="228" y="88" font-weight="600">Copyleft</text><text x="228" y="102" font-size="7.5">GPL</text>
    <rect x="326" y="64" width="118" height="56" rx="6" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.3"/>
    <text x="385" y="88" font-weight="600">Proprietary</text><text x="385" y="102" font-size="7.5">or permissive</text>
  </g>
  <line x1="134" y1="92" x2="172" y2="92" stroke="currentColor" stroke-width="1.8" marker-end="url(#lc_ar)"/>
  <text x="153" y="82" text-anchor="middle" font-size="8" fill="currentColor" font-weight="600">✓</text>
  <text x="153" y="138" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.9">flows in</text>
  <g stroke="currentColor" stroke-width="1.8">
    <line x1="284" y1="92" x2="316" y2="92" stroke-dasharray="4 3" stroke-opacity="0.6"/>
    <line x1="320" y1="84" x2="320" y2="100"/>
    <line x1="300" y1="84" x2="308" y2="100"/>
    <line x1="308" y1="84" x2="300" y2="100"/>
  </g>
  <text x="305" y="138" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.9">can't flow out</text>
  <text x="230" y="166" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.8">combined result ships as GPL</text>
  <defs><marker id="lc_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Copyleft compatibility runs one way. Permissive code (MIT, BSD, Apache) can be pulled <em>into</em> a GPL project and the combined work ships as GPL — but GPL code can't be folded back out into a proprietary or permissive work without forcing that work to become GPL too. The blocked arrow is where most people get burned.</figcaption>
</figure>

## The specific pairings that trip people up

Beyond the broad rule, a handful of named incompatibilities cause most real-world headaches.

### GPLv2 vs GPLv3

The GPL comes in versions, and **GPLv2 and GPLv3 are generally incompatible with each other.** GPLv3 added new conditions (an explicit patent grant, anti-tivoization terms) that GPLv2 doesn't permit you to add, and GPLv2 forbids adding terms beyond its own. So plain `GPL-2.0-only` code and `GPL-3.0` code can't be combined.

The escape hatch is the "or later" clause. Much GPL code is released as **"version 2 or later"** (`GPL-2.0-or-later`), which lets a downstream user choose to apply it under GPLv3 — making it compatible with other GPLv3 code. Always check whether a project is "v2 only" or "v2 or later"; the difference decides the whole question.

### Apache 2.0 → GPLv3 is fine; → GPLv2 is not

Apache 2.0's patent and indemnity provisions are considered *additional restrictions* under GPLv2, so **Apache-2.0 code cannot be combined into a GPLv2 work.** GPLv3, however, was deliberately written to accept Apache 2.0's patent terms, so **Apache-2.0 → GPLv3 is compatible.** This is a common surprise: the same Apache library is fine in one GPL version and forbidden in the other.

### A compatibility-direction table

| From (the code you're adding) | Into (the project) | Compatible? | Why |
|---|---|---|---|
| MIT / BSD-2/3 / Apache-2.0 | Another permissive | Yes | Few conditions; notices preserved |
| MIT / BSD | Proprietary / closed | Yes | Permissive allows closed combination |
| MIT / BSD / Apache-2.0 | GPLv3 | Yes | Permissive flows up into copyleft |
| Apache-2.0 | **GPLv2-only** | **No** | Apache patent terms = added restriction |
| GPL (any) | MIT / proprietary | **No** | Copyleft would force the whole work to be GPL |
| GPLv2-only | GPLv3 | **No** | Each forbids the other's added terms |
| GPLv2-**or-later** | GPLv3 | Yes | Downstream may upgrade to v3 |

(Read the table as direction-sensitive: "From → Into." Reverse the arrow and the answer often changes.)

## The combined work takes the most restrictive terms

When compatible licenses *do* combine, what license governs the result? The practical rule: **the combined work must satisfy every component license simultaneously, which means the most restrictive set of conditions wins.**

Mix a pile of MIT code with one GPL component and ship them as one program, and the *whole distributable work* must be offered under the GPL — because that's the only way to satisfy the GPL's requirement, and the MIT pieces don't object. The MIT files keep their own license notices internally (you must still preserve them), but the work as a whole is bound by the GPL's copyleft. The strictest license effectively sets the floor for the entire shipped artifact.

This is why a single copyleft dependency can quietly change the license of your product. It doesn't relicense the dependency's original files, but it constrains how you may distribute the *combination*.

## How to check before you add a dependency

The cheapest time to catch an incompatibility is *before* `go get`, `npm install`, or copying a file. A quick routine:

1. **Find the dependency's license.** Look for a `LICENSE` file, the SPDX identifier in package metadata, or the license field in `go.mod`/`package.json`. If you can't tell what license it's under, treat that as a red flag, not a green light.
2. **Identify your project's license and your intent.** Are you shipping closed-source? Permissive? GPL? The same dependency can be fine or fatal depending on what *you* ship.
3. **Check the direction against the table above.** Permissive-into-anything is almost always fine. Copyleft-into-permissive-or-proprietary almost never is. For GPL, check the exact version and the "or later" clause.
4. **Mind transitive dependencies.** Your direct dependency may pull in others under different licenses. The combination is only as compatible as its strictest link.
5. **When in doubt, use a tool.** Automated scanners (and SPDX-based tooling like GopherTrunk's `make licenses` job) inventory every license in the tree so a human can review it.

```bash
# GopherTrunk inventories every dependency license via this CI step,
# so an incompatible license shows up before it ships.
make licenses
```

That auditing process — scanning the whole dependency tree, producing an attribution file, and catching surprises — is a lesson of its own. We cover it in depth in [Auditing dependencies & SBOMs](/learn/software-licensing/auditing-dependencies/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — permissive code flows up into the GPL, but GPL code can't flow down into a permissive or proprietary work without forcing the whole thing to become GPL." markdown="0">
  <p class="knowledge-check__q">Quick check: which combination is allowed?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Pulling MIT-licensed code into a GPLv3 project and shipping the result as GPLv3</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Folding a GPL library into your closed-source product while keeping it closed</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Combining GPLv2-only code with GPLv3 code into one program</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Compatible means combinable** — two licenses are compatible if their code can be merged into one work you can legally distribute, satisfying both at once.
- **Permissive combines freely** — MIT, BSD, and Apache impose so few conditions they slot into almost any project, open or closed.
- **Copyleft flows one way** — you can pull permissive code into the GPL, but not GPL code into a permissive or proprietary work.
- **Versions and patents matter** — GPLv2-only and GPLv3 are incompatible unless "or later"; Apache-2.0 works with GPLv3 but not GPLv2.
- **The strictest license wins** — a compatible combined work must satisfy every component license, so its terms default to the most restrictive one.
- **Check before you add** — confirm the dependency's license, your intent, the direction, and the transitive tree before pulling anything in.

Next up: now that you can tell which licenses combine, how do you pick the right one for your *own* project? See [Choosing an open-source license](/learn/software-licensing/choosing-an-oss-license/).
