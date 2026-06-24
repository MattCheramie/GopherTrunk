---
slug: mit-and-bsd-licenses
title: "MIT & BSD: minimal permissive"
description: A walkthrough of the MIT and BSD licenses — the shortest, most permissive open-source licenses. See the grant, the keep-this-notice condition, the warranty disclaimer, the BSD non-endorsement clause, and where their minimalism helps and hurts.
keywords: MIT license, BSD license, BSD 2-clause, BSD 3-clause, permissive license, warranty disclaimer, copyright notice, non-endorsement clause, open source, patent grant, license compatibility, closed-source fork
level: beginner
status: full
faq:
  - q: "What's the actual difference between MIT and BSD?"
    a: "They're almost identical in effect — both are short permissive licenses that ask you to keep the copyright and license notice and disclaim warranty. The **3-clause BSD** adds a non-endorsement clause (you can't use the author's name to promote your product without permission); the **2-clause BSD** drops that and is essentially equivalent to MIT. Pick by taste; all three are maximally compatible."
  - q: "Does the MIT license give me a patent license?"
    a: "Not explicitly, and that's its main gap. MIT and BSD grant **copyright** permissions in plain language but say nothing clear about **patents**. If patent exposure matters to you, prefer [Apache 2.0](/learn/software-licensing/apache-license/), which includes an express patent grant and a retaliation clause."
  - q: "Can someone take my MIT-licensed code and make it closed source?"
    a: "Yes. Permissive licenses allow closed-source forks — anyone can build a proprietary product on your MIT or BSD code, as long as they keep your notice. If you want to prevent that and require changes to stay open, you want a [copyleft license](/learn/software-licensing/permissive-vs-copyleft/) instead."
---
# MIT & BSD: minimal permissive

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The minimalist permissive licenses** — MIT and BSD are among the shortest licenses in wide use. **Three parts** — a broad grant, one condition (keep the notice), and a warranty disclaimer. **BSD adds clauses** — 2-clause is MIT-like; 3-clause adds a non-endorsement clause. **Maximally compatible, with gaps** — simple and business-friendly, but no patent grant and no copyleft protection.
</div>

If you've used open-source software, you've used MIT- and BSD-licensed code — they're the most common permissive licenses on the planet. Their appeal is their brevity: you can read the entire MIT license in under a minute, and once you understand its three moving parts you understand most permissive licensing. By the end of this lesson you'll be able to read the MIT and BSD texts and know exactly what they grant and require, tell the BSD 2-clause and 3-clause variants apart, and weigh their strengths (simplicity, compatibility, adoption) against their real gaps (no patent grant, no copyleft protection, closed-source forks allowed).

## The MIT license, walked through

The MIT license is short enough to read in full. Here it is, with the placeholder fields a project fills in:

```text
MIT License

Copyright (c) <year> <copyright holders>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following condition:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. ...
```

It has exactly three parts:

### 1. The grant

The middle paragraph is the **grant** — and it's about as broad as a license can be. You may "use, copy, modify, merge, publish, distribute, sublicense, and/or sell" the software, and let others do the same. Note that **sell** is right there in the list: MIT-licensed code can be used in commercial products without restriction. "Sublicense" means you can pass the code along under different terms as part of your own work.

### 2. The one condition

There's a single string attached: **"The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software."** In other words — keep the notice. When you ship MIT-licensed code (or a substantial chunk of it), you must include the original copyright line and the license text. That's the entire compliance obligation. It's why a complete product can contain hundreds of MIT notices bundled in an attribution file. We cover doing this properly in [Meeting permissive obligations](/learn/software-licensing/permissive-compliance/).

### 3. The warranty disclaimer

The shouting all-caps paragraph at the end is the **warranty disclaimer**. It says the software is provided "AS IS," with no warranty of any kind — not of merchantability, not of fitness for a purpose, not of non-infringement. The capitals are a legal convention for "conspicuous" disclaimers. The point is to protect the author: if the code breaks something, you can't come after them. Almost every open-source license has a clause like this; we look at why in [The risk clauses](/learn/software-licensing/risk-clauses/).

## The BSD licenses

The **BSD** licenses (from the Berkeley Software Distribution) are the other minimalist family. They come in numbered variants for the number of clauses they carry.

### BSD 2-clause

The 2-clause BSD ("Simplified BSD") has the same shape as MIT: a grant, a keep-the-notice condition (it spells out that the notice must appear in both source and binary distributions), and a warranty disclaimer. For practical purposes it is equivalent to MIT.

### BSD 3-clause

The 3-clause BSD ("New" or "Modified" BSD) adds one extra clause — the **non-endorsement clause**:

```text
3. Neither the name of the copyright holder nor the names of its
   contributors may be used to endorse or promote products derived from this
   software without specific prior written permission.
```

This says you can't use the original author's or organization's name to advertise your product. Build whatever you like on the code, but don't imply they endorse it. That's the only meaningful difference from the 2-clause version, and it's why many companies and projects prefer 3-clause BSD — it adds a touch of name protection at no cost to permissiveness. (An older "4-clause" BSD existed with an advertising requirement; it caused compatibility headaches and is effectively retired — avoid it.)

## How they compare

| | **MIT** | **BSD 2-clause** | **BSD 3-clause** |
|---|---|---|---|
| Grant breadth | Very broad (use, modify, sell, sublicense) | Very broad | Very broad |
| Keep copyright/license notice | Yes | Yes (source and binary) | Yes (source and binary) |
| Warranty disclaimer | Yes | Yes | Yes |
| Non-endorsement clause | No | No | **Yes** |
| Explicit patent grant | No | No | No |
| Copyleft protection | No | No | No |
| Practical equivalence | ≈ BSD 2-clause | ≈ MIT | MIT + name protection |

The honest summary: all three are nearly the same license, and choosing among them is mostly a matter of taste and whether you want the non-endorsement clause.

## Strengths

These licenses are popular for good reasons:

- **Simple and readable.** You can understand the whole thing in one sitting — no lawyer required to grasp the basics.
- **Maximally compatible.** Because they impose so few conditions, MIT and BSD code can be combined with almost anything, including copyleft and proprietary code. This makes them the safe default for libraries meant to be widely reused. We'll see why compatibility matters in [License compatibility](/learn/software-licensing/license-compatibility/).
- **Business-friendly.** No obligation to open your own code means companies adopt them without legal hand-wringing.
- **Huge adoption.** A vast amount of the modern software ecosystem — front-end frameworks, utility libraries, language runtimes — is MIT or BSD, so they're familiar to everyone.

## Weaknesses

The same minimalism is also where they fall short:

- **No patent grant.** MIT and BSD speak the language of copyright and say nothing clear about patents. A contributor could, in theory, contribute code and still assert a patent against users of it. If that risk matters to you, [Apache 2.0](/learn/software-licensing/apache-license/) was designed to close exactly this gap with an explicit patent license.
- **No copyleft protection.** Nothing requires anyone to share their improvements. A company can take your MIT library, improve it substantially, and ship the result as closed source, giving nothing back.
- **Closed-source forks allowed.** This follows directly — if you want to *prevent* proprietary forks and force improvements to stay open, a permissive license is the wrong tool; you want [copyleft](/learn/software-licensing/permissive-vs-copyleft/).

None of these are bugs — they're the deliberate consequences of choosing "make it easy to use" over "make it stay free." Whether they're acceptable depends entirely on your goals, which is the subject of [Choosing an open-source license](/learn/software-licensing/choosing-an-oss-license/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — the 3-clause BSD's extra clause is the non-endorsement clause; the rest matches the 2-clause and MIT." markdown="0">
  <p class="knowledge-check__q">Quick check: what does the BSD 3-clause license add over the 2-clause version?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A requirement to release your changes under the same license</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">An explicit patent grant from contributors</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A non-endorsement clause barring use of the author's name to promote your product</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Minimal permissive licenses** — MIT and BSD are among the shortest and most widely used open-source licenses.
- **Three parts** — a broad grant (including the right to sell), one condition (keep the copyright and license notice), and a warranty disclaimer.
- **BSD variants** — 2-clause is essentially MIT; 3-clause adds a non-endorsement clause protecting the author's name.
- **Strengths** — simple, maximally compatible, business-friendly, and hugely adopted.
- **Weaknesses** — no explicit patent grant, no copyleft protection, and closed-source forks are allowed.

Next up: a permissive license that keeps the freedom but closes the patent gap. See [Apache 2.0: permissive with a patent grant](/learn/software-licensing/apache-license/).
