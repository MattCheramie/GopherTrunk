---
slug: using-oss-in-products
title: Can you sell software built on open source?
description: Yes — you can absolutely build and sell commercial software on open source. The catch is obligations, not prohibition. Learn what permissive and copyleft licenses require, what "selling" open source means, and the few things you can never do.
keywords: sell software built on open source, commercial open source, can you sell open source, open source in products, open source obligations, permissive vs copyleft selling, monetize open source, open source compliance, relicensing, attribution
level: beginner
status: full
faq:
  - q: "Is it legal to charge money for software that uses open source?"
    a: "Yes. Open-source licenses grant the right to use, modify, and **distribute** the code, including commercially — none of the mainstream ones forbid charging money. The Open Source Definition explicitly bars any license that restricts selling. What you owe is **compliance** with the license's conditions (like attribution), not a fee or permission."
  - q: "Can I keep my own code closed if it's built on open source?"
    a: "It depends on the license. With **permissive** licenses (MIT, BSD, Apache) you can keep your additions closed as long as you preserve notices. With **copyleft** licenses (GPL, AGPL), distributing a derivative can require you to release your changes under the same terms. Knowing which bucket each dependency falls in is the whole game — covered in the next lessons."
  - q: "Do I need permission from the project to use it commercially?"
    a: "No. The license *is* the permission — it's a standing grant to everyone, so you don't need to ask or sign anything. You just have to follow its terms. The exceptions are dual-licensed projects offering a separate commercial license, and trademark use, which licenses usually don't grant."
---
# Can you sell software built on open source?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Yes, you can sell it** — commercial software built on open source is normal, legal, and everywhere. **The catch is obligations, not prohibition** — licenses say *what you must do* (attribution, sometimes sharing changes), not *that you can't*. **Effort is a spectrum** — permissive licenses ask for little; copyleft asks for real discipline. **A few hard lines** — never strip notices or pass off others' code as your own.
</div>

This is the first lesson of the module on shipping commercial software built on open source. There's a stubborn myth that "open source" and "selling software" are opposites — that touching an open-source library somehow forces your whole product to be free, or that you need special permission to make money. None of that is true. By the end of this lesson you'll understand why you can absolutely build a paid product on open source, what the licenses actually ask of you in return, what "selling" open source even means, and the handful of things you can never do. The next three lessons then make compliance practical.

## Busting the myth: open source is built for commercial use

Start with the single most important fact: **open-source licenses permit commercial use.** This isn't a loophole or a gray area — it's the design. The [Open Source Definition](/learn/software-licensing/what-is-open-source/), the criteria a license must meet to be called open source, explicitly requires that the license **not restrict anyone from selling the software** and **not discriminate against fields of endeavor**, including commercial ones. A license that said "free for hobbyists, but not for companies" would, by definition, not be open source.

So the question "can I sell software built on open source?" has a clear answer: **yes.** Some of the largest software businesses in the world run on open-source foundations. Your phone, your bank's servers, and the website you bought something from this morning are all stacked on open-source code that someone is very much making money around.

What trips people up is conflating two different things:

| The myth | The reality |
|----------|-------------|
| "Using open source means my product must be free" | You can charge whatever you like; price is yours to set |
| "I need the project's permission to use it commercially" | The license is the permission — it's already granted to everyone |
| "Open source forbids making a profit" | It forbids *restricting others*, not *you profiting* |
| "Any open source 'infects' my code" | Only some copyleft licenses create sharing obligations, and only on distribution |

The thing that *is* real — and that the rest of this module is about — is **obligations**. Open source gives you broad rights up front, and in exchange attaches conditions. Meeting those conditions is the job.

## Permission is granted; obligations are the price

A useful mental model: an open-source license is a **standing offer** from the author to everyone. It says, in effect, "here are rights — to use, copy, modify, and distribute this — and here is what you must do if you accept them." You accept the offer by using the code under its terms. There's no contract to sign, no invoice, no gatekeeper. (Whether a license is technically a [license or a contract](/learn/software-licensing/license-vs-contract/) is a subtle legal question, but the practical effect is the same: rights with conditions.)

Because the rights are already granted, the only way to *lose* them is to break the conditions. That reframes the whole topic. You're not asking "am I allowed?" — you are. You're asking "**what do I owe in return, and can I meet it?**" For the vast majority of dependencies, the answer is "very little, and easily."

## The spectrum of effort

Not all obligations are equal. They run along a spectrum from trivial to substantial, and the license family tells you roughly where a dependency sits. This is the [permissive vs copyleft](/learn/software-licensing/permissive-vs-copyleft/) distinction, viewed through the lens of "how much work is this going to be?"

| License family | Examples | What you typically owe | Effort |
|----------------|----------|------------------------|--------|
| **Permissive** | [MIT, BSD](/learn/software-licensing/mit-and-bsd-licenses/), [Apache 2.0](/learn/software-licensing/apache-license/) | Preserve copyright and license notices; Apache adds a NOTICE file and patent terms | Low — mostly bookkeeping |
| **Weak copyleft** | LGPL, MPL, EPL | Share changes *to the open-source files themselves*; your own separate code can stay closed | Moderate — keep boundaries clean |
| **Strong copyleft** | [GPL](/learn/software-licensing/gpl-strong-copyleft/) | Distribute the derivative work's source under the same license | High — architectural discipline |
| **Network copyleft** | [AGPL](/learn/software-licensing/agpl-network-copyleft/) | Strong copyleft, *and* it triggers even when users only reach the software over a network | Highest — affects SaaS too |

Permissive licenses are the easy case: you ship the product, include the right notices, done. Copyleft is where the real constraints live — and where the fear of "infection" comes from. The honest summary is that **copyleft is entirely usable in commercial products with discipline, and genuinely risky if ignored.** We'll dedicate a full lesson to making that practical.

## What "selling" open source actually means

Here's a point that confuses newcomers: if anyone can get the code for free, how can you *sell* it? The answer is that open source doesn't give you exclusivity, and selling open source is not selling exclusivity — it's selling something else:

- **Copies and convenience.** You can charge for a packaged, ready-to-run build, an installer, an app-store listing, or a supported distribution. The license lets others redistribute too, but plenty of customers happily pay for the convenient version.
- **Support, warranties, and SLAs.** Open-source code ships "as is" with no warranty. Businesses pay real money for someone to stand behind it — support contracts, [SLAs](/learn/software-licensing/sla-and-support-agreements/), security patches, and indemnity.
- **Hosting and operations.** Running software well is a service. Selling a managed, hosted version of open-source software is a huge and legitimate business model.
- **Proprietary additions on top.** With permissive (and carefully with weak-copyleft) dependencies, you can build closed features around an open core and sell those.

What you are *not* selling is the right to be the only one who has the code. Open source trades exclusivity for reach. That's the bargain — and it's a workable, profitable one.

## What you can never do

The rights are broad, but a few lines are bright and absolute. Crossing them isn't a compliance slip; it's infringement.

- **Never strip or hide the notices.** Practically every open-source license requires you to preserve the copyright statement and a copy of the license text when you distribute. Deleting them to make your product look wholly yours breaks the license — which means you no longer have permission to use the code at all. We cover exactly how to keep them in the [next lesson](/learn/software-licensing/permissive-compliance/).
- **Never relicense others' code as your own.** You can't take someone's MIT-licensed library, slap your name and a proprietary license on it, and claim authorship. You may license *your* contributions, but the original code keeps its license and its [copyright](/learn/software-licensing/copyright-and-software-ownership/).
- **Never ignore copyleft on distribution.** If you ship a product that includes GPL code as part of a derivative work, "I didn't realize" is not a defense. The obligation to offer source attaches whether or not you read the license.
- **Never assume the license covers trademarks.** A license to the *code* is not a license to the project's *name or logo*. Using those to imply endorsement is a separate problem — see [patents, trademarks & trade secrets](/learn/software-licensing/other-ip-in-software/).

Stay inside those lines and follow each dependency's conditions, and you can build and sell with confidence.

## Where this module goes next

This lesson set the frame: yes, you can sell, and the work is meeting obligations. The next three lessons make it concrete:

- **[Permissive compliance](/learn/software-licensing/permissive-compliance/)** — the easy-but-mandatory case: exactly what MIT, BSD, and Apache require when you ship, and where to put the attributions.
- **[Copyleft in products](/learn/software-licensing/copyleft-in-products/)** — the "will GPL infect my product?" question, answered with what actually triggers obligations and the safe architectural patterns.
- **[Auditing dependencies & SBOMs](/learn/software-licensing/auditing-dependencies/)** — you can't comply with licenses you don't know you're shipping, so this is how to find and track them all.

<div class="knowledge-check" data-quiz data-correct-msg="Right — open source grants commercial rights up front; the work is meeting the license's obligations, not asking permission." markdown="0">
  <p class="knowledge-check__q">Quick check: what is the real catch when selling software built on open source?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">You must meet the license's obligations, such as preserving notices and sometimes sharing changes</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">You must get written permission from each project before charging money</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">You are not allowed to make a profit from open-source code</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Selling is allowed** — open-source licenses permit commercial use by design; the Open Source Definition forbids restricting sale.
- **Obligations, not prohibition** — the license is a standing grant of rights with conditions; your job is to meet the conditions, not to ask permission.
- **Effort is a spectrum** — permissive licenses ask for little more than attribution; copyleft asks for real architectural discipline.
- **"Selling" means copies, support, hosting, and add-ons** — you trade exclusivity for reach and monetize everything around the code.
- **A few hard lines** — never strip notices, relicense others' code as your own, ignore copyleft on distribution, or assume you got the trademark.

Next up: the easy-but-mandatory case in detail. See [Meeting permissive obligations](/learn/software-licensing/permissive-compliance/).
