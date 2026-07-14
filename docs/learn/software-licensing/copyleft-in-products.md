---
slug: copyleft-in-products
title: "Copyleft in products: the viral question"
description: Will the GPL infect your product? Learn what actually triggers copyleft obligations — distribution, not internal use or SaaS (except AGPL) — the contested linking boundary, and the safe architectural patterns that let you use copyleft code with discipline.
keywords: GPL infect product, copyleft viral, GPL commercial product, copyleft trigger distribution, derivative work linking, static vs dynamic linking, LGPL relink, AGPL SaaS, mere aggregation, copyleft compliance, copyleft architecture
level: advanced
status: full
faq:
  - q: "Will using a GPL library force me to open-source my entire product?"
    a: "Only if you **distribute** a product that is a *derivative work* of that GPL code. Using GPL code privately inside your company, or offering it as a hosted service, does **not** trigger the GPL's source-sharing obligation (the AGPL is the exception — it triggers on network use). And whether your code is a derivative work depends on how tightly it's bound to the GPL code, which is exactly what good architecture controls."
  - q: "Is dynamic linking a safe way around the GPL?"
    a: "It's safer than static linking, but it is **not a settled legal guarantee.** The Free Software Foundation considers most linking — static or dynamic — to create a derivative work. Many lawyers treat a clean, arm's-length boundary (separate process, stable interface) as far stronger evidence of independence than dynamic linking alone. For the **LGPL** specifically, dynamic linking plus a relink mechanism is explicitly permitted."
  - q: "Does running GPL software on my servers (SaaS) trigger anything?"
    a: "Under the plain **GPL**, no — making software available over a network is not 'distribution' in the GPL's sense, so there's no obligation to share source with users. The **AGPL** was written precisely to close that gap: if users interact with AGPL software over a network, you must offer them the corresponding source. So the answer is license-specific."
---
# Copyleft in products: the viral question

> **This is educational material, not legal advice.** For decisions that carry real risk, consult a qualified attorney.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Distribution is the trigger** — copyleft obligations fire when you *ship* a derivative to users, not from internal use. **SaaS is mostly safe — except AGPL** — the GPL doesn't trigger on network use; the AGPL does. **The linking boundary is contested** — static vs dynamic matters, but a clean separate-process boundary is the stronger defense. **Usable with discipline** — copyleft is fine in products if you architect deliberately, and dangerous only if you ignore it.
</div>

This is the lesson everyone is secretly worried about. "Copyleft" — the family of licenses led by the [GPL](/learn/software-licensing/gpl-strong-copyleft/) — has a reputation for being *viral*: the fear that touching one GPL file forces you to give away your entire commercial product. That reputation is half-true and half-myth, and the difference is worth real money. By the end of this lesson you'll know exactly what triggers copyleft obligations, why internal use and most SaaS don't, where the genuinely contested "derivative work" line sits, and the architectural patterns that let you use copyleft code in a product safely — plus when to just keep it at arm's length.

## What actually triggers an obligation: distribution

The single most important word in copyleft is **distribution** (the GPLv3 calls it *conveying*). Copyleft's source-sharing obligation attaches when you **convey the software to someone else** — ship a binary, hand over an installer, sell a device with the code on it. Until you distribute, you can do anything you like in private: modify GPL code, link it into your closed product, run it across your whole company. The GPL grants you those rights with **no obligation** as long as the result stays inside your walls.

This is why the "viral" framing misleads. The GPL doesn't spread by contact; it attaches a condition to one specific act — distribution. No distribution, no obligation. Three consequences fall out of this:

- **Internal use is free.** A company can use, modify, and deploy GPL software internally with zero source-sharing duty. Nothing is conveyed.
- **Most SaaS is free of GPL obligations.** Letting users *interact with* software over a network is not conveying a copy to them. So a hosted product built on GPL code does not, under the plain GPL, owe users source. (This famous gap is the "ASP loophole.")
- **The AGPL closes the SaaS gap.** The [AGPL](/learn/software-licensing/agpl-network-copyleft/) adds a clause that treats network interaction *as if* it were distribution: if users interact with AGPL software over a network, you must offer them the corresponding source. We return to this below.

## "Derivative work" and the linking boundary

When you *do* distribute, the next question is scope: **how much** must you release? Copyleft obligations cover the derivative work — the GPL code plus whatever is so tightly combined with it that it counts as one work. Code that is genuinely separate and independent is *not* covered, even if you ship it in the same package (more on that under "mere aggregation").

The hard part is that "derivative work" is a copyright concept, not a precise technical one, and the **linking boundary is genuinely contested.** Here's the honest state of it:

- The **Free Software Foundation's position** is that linking GPL code into your program — *static or dynamic* — generally creates a derivative work, so the combined program falls under the GPL.
- **Many practitioners and some courts** focus less on the linking mechanism and more on *how intertwined* the code is: shared data structures, intimate function calls, and tight coupling point toward a derivative; communication across a clean, stable, arm's-length interface points toward independence.
- **There is no single binding US precedent** that resolves dynamic linking cleanly. So treat dynamic linking as *safer* but not as a magic exemption.

| Combination pattern | Typical risk of being a derivative |
|---------------------|-------------------------------------|
| Static linking (GPL compiled into your binary) | High — widely treated as derivative |
| Dynamic linking (GPL shared library) | Contested — FSF says derivative; weaker boundary than separation |
| LGPL dynamic linking with relink ability | Allowed — the LGPL explicitly permits this for your closed code |
| Separate process, talking over IPC/CLI/socket | Low — strong evidence of independence |
| Two independent programs shipped together | Mere aggregation — not derivative |

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 178" role="img" aria-label="A coupling gradient of four ways to combine with copyleft code — static link, dynamic link, separate process over IPC, and mere aggregation — ordered from tightest coupling on the left to loosest on the right. A shaded band under the left half marks where the copyleft obligation likely reaches, and a dashed contested boundary sits between dynamic linking and a separate process, past which the code is likely independent." xmlns="http://www.w3.org/2000/svg">
  <text x="230" y="22" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">Coupling gradient: where copyleft reach fades</text>
  <text x="44" y="46" text-anchor="start" font-size="8" fill="currentColor" fill-opacity="0.85">◄ tighter coupling</text>
  <text x="416" y="46" text-anchor="end" font-size="8" fill="currentColor" fill-opacity="0.85">looser coupling ►</text>
  <line x1="40" y1="72" x2="430" y2="72" stroke="currentColor" stroke-width="1" stroke-opacity="0.35"/>
  <g text-anchor="middle" fill="currentColor">
    <circle cx="66" cy="72" r="6" fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="1.3"/>
    <text x="66" y="90" font-size="8" font-weight="600">static link</text><text x="66" y="101" font-size="7">compiled in</text>
    <circle cx="176" cy="72" r="6" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.2"/>
    <text x="176" y="90" font-size="8" font-weight="600">dynamic link</text><text x="176" y="101" font-size="7">shared library</text>
    <circle cx="296" cy="72" r="6" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/>
    <text x="296" y="90" font-size="8" font-weight="600">separate process</text><text x="296" y="101" font-size="7">IPC · CLI · socket</text>
    <circle cx="410" cy="72" r="6" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.2"/>
    <text x="410" y="90" font-size="8" font-weight="600">aggregation</text><text x="410" y="101" font-size="7">bundled only</text>
  </g>
  <rect x="40" y="118" width="196" height="20" rx="3" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="0.8"/>
  <text x="138" y="131" text-anchor="middle" font-size="7.5" fill="currentColor">obligation likely reaches</text>
  <rect x="236" y="118" width="194" height="20" rx="3" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.5" stroke-dasharray="4 3"/>
  <text x="333" y="131" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">likely independent</text>
  <line x1="236" y1="56" x2="236" y2="142" stroke="currentColor" stroke-width="1" stroke-opacity="0.5" stroke-dasharray="4 3"/>
  <text x="236" y="158" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">contested boundary</text>
</svg>
<figcaption>How tightly you bind to copyleft code is the lever. Static linking fuses it into your binary (high reach); a separate process talking over IPC keeps it at arm's length (low reach). The obligation fades as coupling loosens, and the exact stopping point — somewhere around the dynamic-link boundary — is contested, which is why a clean separation is the strongest defense.</figcaption>
</figure>

### LGPL's relink allowance

The [LGPL](/learn/software-licensing/weak-copyleft-licenses/) — "Lesser GPL," a weak copyleft license — was designed for exactly the linking case. It lets your proprietary program link against an LGPL library while keeping your own code closed, **on the condition** that the user can replace the library with a modified version. In practice that means dynamic linking (so the user can swap the `.so`/`.dll`), or shipping object files for relinking. The library itself stays copyleft — changes to *it* must be shared — but your application doesn't.

## Safe architectural patterns

Architecture is the lever. The same copyleft component can be a compliance non-event or a problem depending on how you build around it. The guiding principle: **the cleaner the boundary, the weaker the case that your code is a derivative.**

- **Run it as a separate process.** Invoke the copyleft tool as its own program — a CLI you shell out to, or a service you talk to over a socket or pipe. Two programs communicating at arm's length are strong evidence of independence (this is *mere aggregation* territory).
- **Define a clear, stable interface.** Talk through a documented protocol, file format, or standard system call, not by reaching into the copyleft code's internals. Intimacy is what creates derivation; a narrow interface limits it.
- **For LGPL, dynamic-link and preserve relink.** Use the library as a swappable shared object and document how a user can replace it.
- **Keep GPL *tools* at arm's length from your *product*.** Using a GPL compiler, build tool, or test harness to *produce* your software does **not** make your software GPL — the GPL covers the tool, not its output (unless the output embeds GPL code). Build-time use is generally fine.
- **Isolate, don't intertwine.** If you need a GPL component's functionality, wrap it behind a boundary rather than weaving it through your codebase.

### Mere aggregation vs derivative work

The GPL itself draws this line. **Mere aggregation** — putting independent programs together on the same disk, image, or installer — does *not* make the aggregate a single work, so your independent program keeps its own license. A Linux distribution is the canonical example: it ships thousands of separately-licensed programs together, and that bundling alone doesn't relicense any of them. The test is whether the parts are genuinely independent works that merely sit together, versus components fused into one program.

## AGPL in a hosted product

The [AGPL](/learn/software-licensing/agpl-network-copyleft/) deserves its own caution because it defeats the usual "SaaS is safe" reasoning. If your hosted product includes AGPL code that users interact with over the network — even unmodified — you must make the **corresponding source** (including your modifications) available to those users. Many companies respond by **banning AGPL outright** in their dependency policy, precisely because it's easy to pull in accidentally and its trigger reaches the hosted model that the plain GPL doesn't. If you do use it, isolate it behind a clear network boundary and be ready to publish your changes to it.

## The bottom line

Copyleft is **usable with discipline and fatal if ignored.** None of these licenses forbid commercial use, and large products ship copyleft components every day. What they punish is carelessness — statically linking GPL into a closed binary you then sell, or pulling in AGPL without realizing the SaaS trigger. The defense is the same defense as for all compliance: know what you're shipping (the [next lesson](/learn/software-licensing/auditing-dependencies/)), decide deliberately how each copyleft component sits in your architecture, and keep clean boundaries. Do that, and "will it infect my product?" stops being a fear and becomes a design choice.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the plain GPL's source-sharing obligation triggers on distribution, not on internal use or network/SaaS access; the AGPL is what extends it to network use." markdown="0">
  <p class="knowledge-check__q">Quick check: which act triggers the plain GPL's obligation to offer source?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Using the GPL software internally inside your company</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Letting users access it over a network as a hosted service</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Distributing (conveying) a derivative work to other people</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Distribution is the trigger** — copyleft obligations attach when you convey a derivative to users; private internal use carries no source-sharing duty.
- **SaaS is mostly safe — except AGPL** — the plain GPL doesn't trigger on network access, but the AGPL was written to close that exact gap.
- **The linking boundary is contested** — static linking is high-risk, dynamic linking is safer but not settled, and a separate-process boundary is the strongest defense.
- **Architecture controls scope** — separate processes, clean interfaces, LGPL relinking, and keeping GPL tools at build-time arm's length all limit derivation.
- **Mere aggregation isn't derivation** — independent programs bundled together keep their own licenses.
- **Discipline, not fear** — copyleft is fully usable in commercial products when you choose deliberately and keep boundaries clean.

Next up: you can only comply with licenses you know you're shipping. See [Auditing dependencies & SBOMs](/learn/software-licensing/auditing-dependencies/).
