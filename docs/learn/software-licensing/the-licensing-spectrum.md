---
slug: the-licensing-spectrum
title: The licensing spectrum
description: Software licensing runs on a spectrum from fully proprietary to public domain. Learn the five main bands — proprietary, source-available, permissive open source, copyleft, and public domain — and what each lets you do.
keywords: software licensing spectrum, proprietary vs open source, source-available, permissive license, copyleft, public domain, open source spectrum, license comparison, closed source, free software
level: beginner
status: full
faq:
  - q: "Is 'source-available' the same as open source?"
    a: "No, and conflating them is a common mistake. **Source-available** means you can *see* the source, but the license still restricts what you can do — you may not be allowed to use it commercially, compete, or redistribute freely. **Open source** requires the freedoms to use, modify, and redistribute for any purpose. Visible source is necessary for open source but nowhere near sufficient."
  - q: "If something is open source, can I sell it?"
    a: "Yes. A core requirement of open source is that the license can't forbid selling the software or charging for it — there's no 'non-commercial only' open-source license. What [copyleft](/learn/software-licensing/permissive-vs-copyleft/) licenses *can* require is that if you distribute it, you also share your source under the same terms. Selling is fine; locking it up may not be."
  - q: "Where does public domain sit on the spectrum?"
    a: "At the far 'most free' end. **Public domain** means no copyright applies at all — anyone can do anything with the work, with no conditions and no need for permission. It's even more permissive than a permissive license, which still imposes small conditions like keeping a notice. Reaching public domain deliberately is trickier than it sounds, which is why tools like [CC0](/learn/software-licensing/public-domain-and-cc0/) exist."
---
# The licensing spectrum

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Licensing is a spectrum, not two camps** — it runs from fully proprietary to public domain, with meaningful bands in between. **Source-available ≠ open source** — visible source can still come with heavy restrictions. **Open source splits into permissive and copyleft** — both grant the core freedoms, but copyleft adds share-alike conditions. **Public domain is the free extreme** — no copyright, no conditions at all.
</div>

People tend to talk about software as either "open" or "closed," but that misses most of the picture. Licensing is better understood as a **spectrum**, running from the most restrictive (locked-down proprietary) to the most free (public domain), with several distinct bands along the way. This lesson lays out that spectrum end to end and gives you one big comparison table to anchor it. By the end you'll have a mental map that the rest of the path fills in band by band.

## The spectrum, end to end

From most restrictive to least, the five main bands are:

1. **Proprietary / closed-source** — source hidden, rights tightly restricted.
2. **Source-available** — source visible, but use is restricted.
3. **Permissive open source** — full freedoms, minimal conditions.
4. **Copyleft open source** — full freedoms, but with share-alike conditions.
5. **Public domain** — no copyright, no conditions at all.

Notice the spectrum has two axes hiding inside it: *can you see the source?* and *what may you do with it?* They don't move in lockstep — source-available shows you the code without granting the freedoms, which is exactly why "I can see it" tells you almost nothing about "I can use it." Keep the four actions from [lesson one](/learn/software-licensing/what-is-a-software-license/) in mind — use, copy, modify, distribute — because each band answers them differently.

<figure class="figure" markdown="0">
<svg viewBox="0 0 480 176" role="img" aria-label="A horizontal licensing spectrum of five bands — proprietary, source-available, permissive, copyleft, and public domain — from left to right. A top arrow pointing right shows freedoms and rights growing toward public domain, while a bottom arrow pointing left shows restrictions and obligations growing back toward proprietary." xmlns="http://www.w3.org/2000/svg">
  <line x1="34" y1="44" x2="446" y2="44" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.7" marker-end="url(#spec_ar)"/>
  <text x="240" y="35" text-anchor="middle" font-size="9" fill="currentColor" fill-opacity="0.9">freedoms &amp; rights grow →</text>
  <line x1="40" y1="90" x2="440" y2="90" stroke="currentColor" stroke-width="1" stroke-opacity="0.35"/>
  <g text-anchor="middle" fill="currentColor">
    <circle cx="52" cy="90" r="6.5" fill="currentColor" fill-opacity="0.35" stroke="currentColor" stroke-width="1.4"/>
    <text x="52" y="110" font-size="8.5" font-weight="600">proprietary</text><text x="52" y="121" font-size="7.5" fill-opacity="0.85">source hidden</text>
    <circle cx="151" cy="90" r="6" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.2"/>
    <text x="151" y="110" font-size="8.5" font-weight="600">source-available</text><text x="151" y="121" font-size="7.5" fill-opacity="0.85">visible, restricted</text>
    <circle cx="250" cy="90" r="6" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.2"/>
    <text x="250" y="110" font-size="8.5" font-weight="600">permissive</text><text x="250" y="121" font-size="7.5" fill-opacity="0.85">few conditions</text>
    <circle cx="349" cy="90" r="6" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.2"/>
    <text x="349" y="110" font-size="8.5" font-weight="600">copyleft</text><text x="349" y="121" font-size="7.5" fill-opacity="0.85">share-alike</text>
    <circle cx="440" cy="90" r="6" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.2"/>
    <text x="440" y="110" font-size="8.5" font-weight="600">public domain</text><text x="440" y="121" font-size="7.5" fill-opacity="0.85">no conditions</text>
  </g>
  <line x1="446" y1="146" x2="34" y2="146" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.7" marker-end="url(#spec_ar)"/>
  <text x="240" y="164" text-anchor="middle" font-size="9" fill="currentColor" fill-opacity="0.9">← restrictions &amp; obligations grow</text>
  <defs><marker id="spec_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The five bands on one line. Moving right — proprietary to public domain — the rights you're granted grow while the obligations placed on you shrink. The two arrows pull in opposite directions, which is why placing a license on this map tells you most of what it lets you do before you read the fine print.</figcaption>
</figure>

## Band 1: Proprietary / closed-source

**Proprietary** software is the default-restrictive end. The source code is kept secret (protected as a [trade secret](/learn/software-licensing/other-ip-in-software/)), and the license grants you a narrow right — usually just to *use* the binary, often on specific terms, with no right to copy beyond your license, modify, or redistribute. Most commercial desktop and enterprise software lives here. This is the world of EULAs, seat licenses, and subscriptions, which we cover in [Proprietary & closed-source licensing](/learn/software-licensing/proprietary-licensing/).

## Band 2: Source-available

**Source-available** software lets you *read* the source — and sometimes modify it for your own use — but the license still imposes real restrictions that disqualify it from being open source. Common limits include "non-commercial use only," "you may not compete with us," or "you may not offer this as a hosted service."

This band has grown fast as companies look for a middle ground: transparency and the goodwill of visible code, without giving competitors the right to take it. But it is **not open source**, and calling it that is misleading. We give this band its own lesson, [Source-available & "fair source"](/learn/software-licensing/source-available-licenses/). The key takeaway now: *seeing the source is not the same as being free to use it.*

## Band 3: Permissive open source

Now we cross into genuine **open source**, which begins with **permissive** licenses. These grant the full open-source freedoms — use, copy, modify, distribute, for any purpose including commercial — with only **minimal conditions**, typically just "keep the copyright and license notice." The MIT, BSD, and [Apache 2.0](/learn/software-licensing/apache-license/) licenses are the classic examples.

The defining trait: you can take permissively licensed code, modify it, and ship it inside a **closed-source** product without being required to release your own source. The permission flows out and asks very little back. GopherTrunk uses one of these — Apache 2.0 — which is why others can build on it freely. We compare these in [MIT & BSD](/learn/software-licensing/mit-and-bsd-licenses/) and [Permissive vs copyleft](/learn/software-licensing/permissive-vs-copyleft/).

## Band 4: Copyleft open source

**Copyleft** licenses grant the same core freedoms as permissive ones — but attach a powerful condition: if you distribute the software (or a modified version), you must **release your source under the same license**. The freedoms are designed to *propagate*: everyone downstream gets the same rights you did.

This is sometimes called "viral," though that word overstates it — copyleft obligations trigger on **distribution**, not merely on use, and the scope depends on the specific license. **Strong copyleft** (the [GPL](/learn/software-licensing/gpl-strong-copyleft/)) reaches broadly; **weak copyleft** ([LGPL, MPL, EPL](/learn/software-licensing/weak-copyleft-licenses/)) is more contained, typically applying only to the licensed component itself; and the [AGPL](/learn/software-licensing/agpl-network-copyleft/) extends the obligation even to software offered over a network. Copyleft is still firmly open source — you can sell it — but it constrains *how* you can combine and ship it.

## Band 5: Public domain

At the far end, **public domain** means **no copyright applies at all**. There's no owner exercising exclusive rights, so anyone can do anything — use, copy, modify, sell, relicense — with no conditions and no need to credit anyone. It's even freer than a permissive license, which still asks you to preserve a notice.

The catch is that *getting* to public domain deliberately is harder than it sounds: in many countries you can't simply abandon your copyright, and there's no clean legal mechanism to do so. That's why dedicated tools exist — the **Unlicense** and Creative Commons **CC0** try to achieve a public-domain-like result, with a permissive license as a fallback for jurisdictions that don't allow true dedication. We cover them in [Public domain, Unlicense & CC0](/learn/software-licensing/public-domain-and-cc0/).

## The whole spectrum at a glance

Here is the spectrum across the questions that matter most:

| | See source? | Modify? | Redistribute? | Sell it? | Main obligation |
|---|---|---|---|---|---|
| **Proprietary** | No | No | No | No (only the vendor sells) | Pay; obey EULA; don't reverse-engineer |
| **Source-available** | Yes | Often, for own use | Restricted | Usually no | Obey the (non-open) restrictions |
| **Permissive OSS** | Yes | Yes | Yes | Yes | Keep the copyright/license notice |
| **Copyleft OSS** | Yes | Yes | Yes | Yes | Share your source under the same license when you distribute |
| **Public domain** | Yes | Yes | Yes | Yes | None |

Read down any column and the spectrum comes into focus: rights generally *expand* as you move from proprietary toward public domain, while *obligations* shift from "pay and obey" to "just keep a notice" to "nothing." The interesting wrinkle is copyleft, which grants broad freedoms yet adds an obligation back — its share-alike condition is the price of keeping those freedoms open for everyone downstream.

## Using the map

This spectrum is the scaffolding for the rest of the path. Most of the upcoming modules zoom into one band:

- The **open-source** module dives into permissive and copyleft licenses one by one.
- The **proprietary & commercial** module covers the closed and source-available bands and how to *sell* software.
- The **using open source in products** module is all about combining bands safely — what happens when permissive, copyleft, and proprietary code meet in one product.

Whenever you meet a new license, your first move is to place it on this spectrum. That single act — "which band is this?" — tells you most of what you need to know before you read a word of the fine print.

<div class="knowledge-check" data-quiz data-correct-msg="Right — source-available means you can see the code, but the license still restricts use, so it is not open source." markdown="0">
  <p class="knowledge-check__q">Quick check: a project's license lets you view and read the source but forbids commercial use. Where does it sit on the spectrum?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It's open source, because the source code is available to read</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It's source-available — visible source but restricted use, which is not open source</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It's public domain, because anyone can access the code</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Licensing is a spectrum** — from fully proprietary to public domain, with distinct bands rather than a simple open/closed split.
- **Two hidden axes** — "can you see the source?" and "what may you do with it?" move independently.
- **Source-available is not open source** — visible code can still carry heavy use restrictions.
- **Open source = permissive + copyleft** — both grant the core freedoms; permissive asks little back, copyleft adds share-alike conditions on distribution.
- **Public domain is the free extreme** — no copyright, no conditions, though reaching it deliberately needs tools like CC0.
- **Place every license on the map first** — identifying the band is the fastest way to understand a new license.

Next up: the same concepts shift across borders — moral rights, registration norms, and which clauses actually hold up. See [Licensing across jurisdictions](/learn/software-licensing/licensing-across-jurisdictions/).
