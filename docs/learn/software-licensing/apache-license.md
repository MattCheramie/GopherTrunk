---
slug: apache-license
title: "Apache 2.0: permissive with a patent grant"
description: Apache 2.0 is permissive like MIT but adds an explicit patent license, a patent-retaliation clause, a NOTICE file, and a duty to state your changes. See why it's the corporate default — using GopherTrunk, which is Apache-2.0, as the example.
keywords: Apache License 2.0, Apache 2.0, patent grant, patent retaliation, NOTICE file, permissive license, trademark, license compatibility, GPLv3, contributor patent, open source, attribution, GopherTrunk license
level: intermediate
status: full
faq:
  - q: "Why pick Apache 2.0 over MIT if both are permissive?"
    a: "The headline reason is **patents**. Apache 2.0 includes an explicit patent license from every contributor and a retaliation clause that revokes it if you sue over patents — protection MIT and BSD don't spell out. It also formalizes attribution via the NOTICE file and requires you to state your changes. That extra structure is why companies and foundations prefer it."
  - q: "What is the NOTICE file and do I have to keep it?"
    a: "The **NOTICE file** is a text file an Apache-2.0 project can ship containing required attribution notices. If a work you redistribute includes a NOTICE file, you must pass along the attribution notices it contains. It's a defined, structured way to carry credit — distinct from the license text itself, which you must also include."
  - q: "Can I combine Apache-2.0 code with GPL code?"
    a: "Carefully. Apache 2.0 is **one-way compatible with GPLv3** — you can include Apache-2.0 code in a GPLv3 project (the result is GPLv3), but not the reverse. It is **not compatible with GPLv2**, because of GPLv2's stance on the patent and termination terms. We cover the details in [License compatibility](/learn/software-licensing/license-compatibility/)."
---
# Apache 2.0: permissive with a patent grant

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Permissive, with more structure** — Apache 2.0 grants the same broad freedoms as MIT but spells out far more. **An explicit patent grant** — every contributor licenses their patents, and a retaliation clause revokes that grant if you sue. **NOTICE file and stated changes** — formal attribution plus a duty to mark what you changed. **The corporate/foundation default** — including GopherTrunk itself, which is Apache-2.0.
</div>

The **Apache License 2.0** is what you reach for when you want permissive freedom but with the legal loose ends tied up — especially around patents. It's the default for the Apache Software Foundation, the Cloud Native Computing Foundation, and a huge share of corporate open source. By the end of this lesson you'll understand what Apache 2.0 adds over MIT and BSD (an explicit patent license, a retaliation clause, a NOTICE file, a stated-changes duty, and a trademark carve-out), why those additions make it the corporate default, and how it fits with the GPL. We'll use GopherTrunk as the running example, because GopherTrunk itself is Apache-2.0.

## Permissive, but with the gaps filled in

At its core, Apache 2.0 is a permissive license: you may use, modify, and distribute the software, including in closed-source and commercial products, and there's no copyleft obligation to publish your changes. So far, this is MIT's bargain.

The difference is everything Apache 2.0 says *around* that grant. Where MIT is three short paragraphs, Apache 2.0 is several pages — and that length buys precision on the questions MIT leaves silent: patents, attribution, what counts as a contribution, and trademarks. If MIT is a handshake, Apache 2.0 is the same handshake written down by lawyers so there's nothing to argue about later.

## The explicit patent grant

This is the headline feature. Section 3 of Apache 2.0 grants you a patent license from **each contributor** covering their contributions:

```text
3. Grant of Patent License. ... each Contributor hereby grants to You a
   perpetual, worldwide, non-exclusive, no-charge, royalty-free, irrevocable
   ... patent license to make, have made, use, offer to sell, sell, import,
   and otherwise transfer the Work ...
```

Why this matters: code can be covered by patents, and a copyright license alone doesn't give you the right to practice those patents. With MIT or BSD, a contributor could in principle hand you code and still hold a patent over the technique, leaving downstream users exposed. Apache 2.0 closes that hole — by contributing, you grant everyone a patent license to use what you contributed.

### The retaliation (termination) clause

Paired with the grant is a **patent-retaliation clause**: if you initiate patent litigation alleging that the software infringes a patent, your patent license under Apache 2.0 terminates. It's a defensive mechanism — you keep your patent license only as long as you don't weaponize patents against the project. This discourages "use the code, then sue everyone" behavior and is a big part of why patent-conscious organizations favor Apache 2.0.

## The NOTICE file and stating changes

Apache 2.0 makes attribution more structured than MIT's "keep the notice." When you redistribute the work, Section 4 requires you to:

- **Include a copy of the license** and retain copyright, patent, trademark, and attribution notices.
- **Carry the NOTICE file.** If the work ships a file called `NOTICE` with attribution text, you must include those notices in your redistribution (in the source, the docs, or a display the software produces). The NOTICE file is the defined channel for required credit.
- **State your changes.** Modified files must carry prominent notices that you changed them. This is a duty MIT and BSD don't impose, and it helps downstream users see what diverged from upstream.

These obligations are light, but they're real, and meeting them is the heart of [permissive compliance](/learn/software-licensing/permissive-compliance/).

## Trademark non-grant

Apache 2.0 is explicit that it does **not** grant any trademark rights (Section 6). You get copyright and patent permissions, but the project's *name and logo* are not yours to use freely. You can build on the code, but you can't pass your fork off as the original project. This separation — open code, protected name — is common and sensible; we touch on the broader topic in [Patents, trademarks & trade secrets](/learn/software-licensing/other-ip-in-software/).

## GopherTrunk as the concrete example

You don't have to go far for a real Apache-2.0 project: **GopherTrunk is licensed under Apache 2.0**, and the full text lives in [`/LICENSE`](https://github.com/MattCheramie/GopherTrunk/blob/main/LICENSE) at the repo root. Everything above applies to it directly:

- Anyone may use, modify, and ship GopherTrunk, including in a commercial product, as long as they keep the license and notices.
- Every contributor to GopherTrunk grants the Apache 2.0 patent license over their contributions, and the retaliation clause applies.
- A fork must state its changes and can't use the GopherTrunk name to imply endorsement.

GopherTrunk also ships a [`/THIRD_PARTY_LICENSES.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/THIRD_PARTY_LICENSES.md) file inventorying the licenses of the code it depends on — a mix of MIT, BSD, and Apache-2.0 modules. That file is a preview of the *attribution* side of using open source: when you build on other people's code, you have to track and reproduce their notices. We'll dig into doing this properly in [Meeting permissive obligations](/learn/software-licensing/permissive-compliance/) and [Auditing dependencies & SBOMs](/learn/software-licensing/auditing-dependencies/).

## Apache 2.0 vs MIT

| | **MIT** | **Apache 2.0** |
|---|---|---|
| Permissive (no copyleft) | Yes | Yes |
| Keep copyright/license notice | Yes | Yes |
| Warranty disclaimer | Yes | Yes |
| Explicit patent grant | **No** | **Yes** |
| Patent-retaliation clause | No | **Yes** |
| NOTICE-file attribution | No | **Yes** |
| Must state your changes | No | **Yes** |
| Trademark explicitly not granted | Silent | **Yes** |
| Length / complexity | Tiny | Several pages |
| Typical user | Small libraries, broad reuse | Companies, foundations, patent-sensitive projects |

The trade is straightforward: Apache 2.0 asks a little more of you (carry the NOTICE, state changes) and gives a lot more certainty in return (patents, attribution, trademarks). For a one-file utility, MIT's brevity often wins; for anything with corporate involvement or patent exposure, Apache 2.0's structure is usually worth it.

## Compatibility with the GPL

One practical wrinkle to remember: Apache 2.0 is **one-way compatible with GPLv3**. You can include Apache-2.0 code in a GPLv3 project — the combined result is GPLv3 — but you can't take GPLv3 code into an Apache-2.0 project, because GPLv3's copyleft would govern the result.

Apache 2.0 is **not compatible with GPLv2**. The Free Software Foundation considers Apache 2.0's patent-termination and other terms to impose conditions GPLv2 doesn't permit, so the two can't be legally combined. GPLv3 was deliberately written to be compatible with Apache 2.0, which is one reason the GPLv2-vs-GPLv3 distinction matters. This is exactly the kind of trap [License compatibility](/learn/software-licensing/license-compatibility/) exists to help you avoid — for now, just remember "Apache 2.0 → GPLv3 yes, GPLv2 no."

<div class="knowledge-check" data-quiz data-correct-msg="Right — the explicit patent license (plus a retaliation clause) is Apache 2.0's signature addition over MIT and BSD." markdown="0">
  <p class="knowledge-check__q">Quick check: what does Apache 2.0 add that MIT and BSD do not clearly provide?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A copyleft requirement to publish your modifications</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A ban on using the code in commercial products</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">An explicit patent grant from contributors plus a patent-retaliation clause</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Permissive with structure** — Apache 2.0 grants MIT-like freedoms but spells out patents, attribution, and trademarks in detail.
- **Explicit patent grant + retaliation** — contributors license their patents, and that license ends if you sue over patents.
- **NOTICE file and stated changes** — formal, structured attribution plus a duty to mark modified files.
- **Trademark not granted** — open code, protected name.
- **GopherTrunk's own license** — Apache-2.0 (`/LICENSE`), with `/THIRD_PARTY_LICENSES.md` previewing dependency attribution.
- **GPL compatibility** — one-way compatible with GPLv3, but not compatible with GPLv2.

Next up: the other side of the divide — the license that makes freedom stick. See [The GPL: strong copyleft](/learn/software-licensing/gpl-strong-copyleft/).
