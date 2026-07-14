---
slug: permissive-vs-copyleft
title: Permissive vs copyleft
description: The single biggest divide among open-source licenses is permissive vs copyleft — whether you can keep your changes to yourself or must share them back. Learn reciprocity, strong vs weak copyleft, and what each kind asks of you.
keywords: permissive license, copyleft, reciprocity, share alike, strong copyleft, weak copyleft, GPL, MIT, Apache, derivative works, open source obligations, license comparison, copyleft vs permissive
level: beginner
status: full
faq:
  - q: "What's the simplest way to tell permissive and copyleft apart?"
    a: "Ask: *if I modify this and ship it, what must I give back?* A **permissive** license asks only that you keep the copyright and license notice — you can otherwise do almost anything, including keeping your changes closed. A **copyleft** license adds reciprocity: derivative works you distribute must be released under the same (or a compatible) license, so the source stays open downstream."
  - q: "Is copyleft 'better' or 'worse' than permissive?"
    a: "Neither — they optimize for different goals. **Permissive** maximizes adoption and lets anyone, including closed-source vendors, build on the code. **Copyleft** maximizes the guarantee that the software and its descendants stay free. The right choice depends on what you want; we cover that in [Choosing an open-source license](/learn/software-licensing/choosing-an-oss-license/)."
  - q: "What's the difference between strong and weak copyleft?"
    a: "It's about *how far* the share-alike obligation reaches. **Strong copyleft** (like the GPL) extends to the whole combined work that includes the licensed code. **Weak copyleft** (like the LGPL or MPL) limits the obligation to the licensed component itself — typically the files or the library — so you can link it into a larger, differently-licensed program without that program becoming copyleft."
---
# Permissive vs copyleft

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The biggest divide in open source** — almost every open-source license is either permissive or copyleft. **Permissive = keep the notice** — do nearly anything, just preserve the copyright and license text. **Copyleft = share alike** — derivatives you distribute must stay under the same license; this is reciprocity. **Strong vs weak** — strong copyleft covers the whole combined work; weak copyleft covers only the licensed component.
</div>

Once you know that open source is a precise term, the next thing to understand is that open-source licenses are not all alike. They split along one dominant axis — **permissive vs copyleft** — and that single distinction shapes almost everything else: what you can build, what you must publish, and which licenses you can safely combine. By the end of this lesson you'll be able to place any open-source license on this axis, explain the idea of reciprocity, and tell strong copyleft from weak copyleft. This framing sets up the next several lessons, each of which zooms in on a specific license family.

## The one axis that matters most

There are dozens of open-source licenses, but the practical question they answer first is always the same: **when you take this code, change it, and give it to someone else, what do you owe?**

The answers cluster into two camps:

- **Permissive licenses** ask for very little. Keep the original copyright and license notice, and you're free to do almost anything else — including building a closed-source product on top and never showing anyone your changes.
- **Copyleft licenses** ask for reciprocity. If you distribute a modified or derived version, you must release it under the same kind of license, so the freedoms travel downstream with the code.

Everything else — patent grants, attribution files, network clauses — is detail layered on top of this core choice. Get this axis straight and the rest of the open-source landscape becomes navigable.

## Permissive: "do almost anything, just keep the notice"

A **permissive** license is the minimal-strings option. The canonical examples are MIT, the BSD licenses, and Apache 2.0, which get their own lessons next. The deal is roughly: *here is the code; use it however you like — commercially, in closed products, modified or not — as long as you preserve the copyright notice and a copy of the license, and don't pretend there's a warranty.*

What permissive licenses deliberately do **not** require:

- They do not require you to publish your changes.
- They do not require your larger product to be open source.
- They do not restrict who can use the code or for what.

This is why permissive licenses dominate in business settings and why they tend to get the widest adoption — a company can build on them without taking on obligations about its own proprietary code. The trade-off is that nothing stops someone from taking the code, improving it, and releasing only a closed-source fork. For the author, that's a feature or a flaw depending on what they wanted. We walk through the mechanics in [MIT & BSD](/learn/software-licensing/mit-and-bsd-licenses/) and [Apache 2.0](/learn/software-licensing/apache-license/).

## Copyleft: "share alike"

A **copyleft** license flips the permissive bargain. It uses copyright law not to restrict sharing but to *guarantee* it: anyone who receives the software gets the four freedoms, and — this is the key part — anyone who distributes a modified version must pass those same freedoms along. The name is a play on "copyright": instead of reserving rights, it reserves *freedoms*.

The mechanism is a license condition that says, in effect, *if you distribute a derivative work, you must license that work under this same license.* The GNU General Public License (GPL) is the archetype, covered in [The GPL: strong copyleft](/learn/software-licensing/gpl-strong-copyleft/).

The intuition behind copyleft is that permissive freedom can be a one-way door: a company can take freely-given code and lock its improvements away, and the next user never benefits. Copyleft closes that door by making the freedom *sticky*. The cost is that copyleft code is more constraining to combine with proprietary software, which is exactly the tension [Copyleft in products: the viral question](/learn/software-licensing/copyleft-in-products/) tackles.

## Reciprocity: the idea under copyleft

The concept that makes copyleft tick is **reciprocity**. The bargain is: *you may have all these freedoms with my code, on the condition that you extend the same freedoms to everyone you pass it on to.* You take freedom, you give freedom — a fair exchange that keeps the commons growing instead of being quietly drained.

Reciprocity is why copyleft is sometimes loosely called "viral" — but that word is misleading and often used pejoratively. Copyleft doesn't reach out and infect unrelated code on your machine; it only applies to works that actually incorporate the copyleft-licensed code, and only when you *distribute* them. We'll unpack exactly where the boundary falls, because "what counts as a derivative work" is the whole ballgame for compliance.

A useful reframing: permissive licenses optimize for **adoption** (make it as easy as possible to use), while copyleft licenses optimize for **preservation** (make sure the software and its descendants stay free). Both are legitimate goals.

## Strong vs weak copyleft (a preview)

Copyleft is not all-or-nothing. It comes in two strengths that differ in *how far* the share-alike obligation reaches:

- **Strong copyleft** extends the obligation to the entire combined work. If you link GPL code into your program and distribute the result, the whole program generally must be GPL. This is the GPL and, for network use, the AGPL ([copyleft over the network](/learn/software-licensing/agpl-network-copyleft/)).
- **Weak copyleft** confines the obligation to the licensed component itself — usually the specific files or the library. You can use a weak-copyleft library inside a larger proprietary program: you must share changes *to the library*, but the rest of your program can stay under whatever license you choose. The LGPL, MPL, and EPL are the main examples, covered in [Weak copyleft: LGPL, MPL, EPL](/learn/software-licensing/weak-copyleft-licenses/).

Weak copyleft is the middle ground between "anything goes" permissive and "the whole thing must be free" strong copyleft.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="Three products side by side showing the escalating share-alike obligation. Under a permissive license nothing in the product must stay open; under weak copyleft only the licensed component must stay open; under strong copyleft the entire combined work must stay open. A top arrow points right to show the obligation growing across the three." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="30" x2="420" y2="30" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.7" marker-end="url(#pc_ar)"/>
  <text x="230" y="22" text-anchor="middle" font-size="9" fill="currentColor" fill-opacity="0.9">share-alike obligation grows →</text>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="78" y="52" font-weight="600">Permissive</text>
    <rect x="18" y="58" width="120" height="88" rx="6" fill="currentColor" fill-opacity="0.04" stroke="currentColor" stroke-width="1.3"/>
    <text x="78" y="98" font-size="8" fill-opacity="0.85">your product</text>
    <text x="78" y="132" font-size="7.5" fill-opacity="0.7">nothing must open</text>
    <text x="78" y="162" font-size="7.5">MIT · BSD · Apache</text>

    <text x="230" y="52" font-weight="600">Weak copyleft</text>
    <rect x="170" y="58" width="120" height="88" rx="6" fill="currentColor" fill-opacity="0.04" stroke="currentColor" stroke-width="1.3"/>
    <text x="230" y="80" font-size="7.5" fill-opacity="0.7">your product</text>
    <rect x="182" y="100" width="96" height="36" rx="4" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1"/>
    <text x="230" y="122" font-size="7.5">the component</text>
    <text x="230" y="162" font-size="7.5">LGPL · MPL · EPL</text>

    <text x="382" y="52" font-weight="600">Strong copyleft</text>
    <rect x="322" y="58" width="120" height="88" rx="6" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1.3"/>
    <text x="382" y="106" font-size="7.5">entire combined</text>
    <text x="382" y="117" font-size="7.5">work</text>
    <text x="382" y="162" font-size="7.5">GPL · AGPL</text>
  </g>
  <g font-size="7.5" fill="currentColor">
    <rect x="150" y="180" width="12" height="10" rx="2" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="0.8"/>
    <text x="168" y="189" text-anchor="start">shaded = must stay open downstream</text>
  </g>
  <defs><marker id="pc_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The obligation escalates left to right. Permissive asks nothing to stay open; weak copyleft keeps just the licensed component open so the rest of your product can stay closed; strong copyleft pulls the whole combined work open. The shaded region — what must stay open downstream — widens at every step.</figcaption>
</figure>

## The comparison at a glance

This table is the one to keep in your head. It frames the next several lessons.

| | **Permissive** | **Weak copyleft** | **Strong copyleft** |
|---|---|---|---|
| Examples | MIT, BSD, Apache 2.0 | LGPL, MPL, EPL | GPL, AGPL |
| Must keep copyright/license notice | Yes | Yes | Yes |
| Must publish your changes when you distribute | No | Yes — to the licensed **component** only | Yes — to the **whole combined work** |
| Can you build a closed-source product on it | Yes | Yes (the larger product stays proprietary) | Generally no — the combined work must be copyleft |
| What stays open downstream | Only the original files (if at all) | The licensed component | The entire derivative work |
| Optimizes for | Adoption | A middle ground | Preserving freedom |

Read across the rows and you can see the obligation grow from left to right: every column keeps the notice, but the *share-alike* duty starts at "none," widens to "just this component," and finally reaches "the whole thing."

<div class="knowledge-check" data-quiz data-correct-msg="Right — reciprocity (share-alike) is the defining feature of copyleft; permissive licenses don't require you to publish your changes." markdown="0">
  <p class="knowledge-check__q">Quick check: what does a copyleft license require that a permissive one does not?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">That derivative works you distribute be released under the same (or a compatible) license</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">That you pay a royalty to the original author for commercial use</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">That you keep the copyright and license notice intact</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **One dominant axis** — open-source licenses split into permissive and copyleft, and that split shapes nearly everything else.
- **Permissive** — keep the copyright and license notice, then do almost anything, including shipping closed-source builds.
- **Copyleft** — adds reciprocity: distributed derivatives must stay under the same kind of license, keeping the source open downstream.
- **Reciprocity, not virality** — copyleft only applies to works that actually include the code, and only when you distribute them.
- **Strong vs weak** — strong copyleft covers the whole combined work; weak copyleft covers only the licensed component.

Next up: the simplest, most widely used permissive licenses, walked through line by line. See [MIT & BSD: minimal permissive](/learn/software-licensing/mit-and-bsd-licenses/).
