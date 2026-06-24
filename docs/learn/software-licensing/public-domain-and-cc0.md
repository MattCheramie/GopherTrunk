---
slug: public-domain-and-cc0
title: "Public domain, Unlicense & CC0"
description: Giving up all rights sounds simple, but "public domain" is legally tricky in many countries. Learn how the Unlicense, CC0, and 0BSD try to dedicate work to everyone — and why no license at all is the opposite of free.
keywords: public domain, CC0, Unlicense, 0BSD, WTFPL, public domain dedication, moral rights, disclaim copyright, no license, all rights reserved, US federal works, fallback license, dedicate to public domain
level: beginner
status: full
faq:
  - q: "Can I just put my code in the public domain?"
    a: "In the US you can come close, but it's surprisingly hard to do cleanly — copyright attaches automatically and several countries don't even let you disclaim it. That's why the recommended approach is a tool like **CC0**, which is a public-domain *dedication* plus a fallback license that grants the same broad permissions in places where dedication doesn't work."
  - q: "If a repository has no license file, is it public domain?"
    a: "No — it's the **opposite**. With no license, default copyright applies and **all rights are reserved**: nobody has permission to copy, modify, or distribute it. \"No license\" is the most restrictive state, not the freest. See [what is a software license](/learn/software-licensing/what-is-a-software-license/)."
  - q: "What are \"moral rights\" and why do they complicate public domain?"
    a: "Moral rights are an author's personal rights (like attribution and integrity of the work) that, in much of the EU and elsewhere, **cannot be waived or transferred** even if the author wants to. So a pure \"I give up everything\" dedication can't fully take effect in those countries, which is why good dedications like CC0 include language to waive what's waivable and license the rest."
---
# Public domain, Unlicense & CC0

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Public domain means no rights reserved** — anyone can use the work for anything, with no conditions. **But it's legally tricky** — copyright is automatic, and several countries won't let you disclaim it (moral rights survive). **CC0 is the recommended tool** — it's a dedication *plus* a fallback license, so it works even where pure dedication doesn't. **No license is NOT public domain** — it's the most restrictive default of all: all rights reserved.
</div>

The licenses you've seen so far all keep *some* rights and attach *some* conditions, even the gentle [MIT and BSD](/learn/software-licensing/mit-and-bsd-licenses/) ones. This lesson covers the far end of the spectrum: giving up rights entirely, putting work in the **public domain**. It sounds like the simplest possible choice, but it's one of the most legally subtle. By the end you'll know what public domain really means, why a clean disclaimer of copyright is harder than expected, which tools (CC0, the Unlicense, 0BSD) actually do the job, why "no license at all" is the opposite of free, and when public domain genuinely makes sense.

## What "public domain" means

A work in the **public domain** belongs to everyone and no one. There is no copyright holder controlling it, so anyone may copy, modify, distribute, sell, or build on it for any purpose, with no permission required and no conditions to satisfy — not even attribution. It is the most permissive state a creative work can be in: even more open than MIT, because MIT still requires you keep its notice.

Works enter the public domain in a few ways: copyright **expires** (very old works), the law **never granted** copyright in the first place (see US federal works below), or the author deliberately **dedicates** the work to the public domain. That last route — a living author voluntarily releasing rights — is the one that concerns software, and it's where things get complicated.

## Why dedicating to the public domain is tricky

Here's the catch most people don't expect: **copyright is automatic and not always disposable.** The moment you write original code, copyright attaches to it whether you want it or not. To put it in the public domain you'd have to actively *give that copyright away* — and the law doesn't always provide a clean mechanism to do so.

Two problems in particular:

- **No clear disclaimer mechanism.** US law has no formal, reliable procedure for an author to abandon copyright. A bare statement like "I release this into the public domain" may or may not be fully effective, and its meaning is uncertain enough that careful lawyers don't rely on it alone.
- **Moral rights that can't be waived.** In much of the EU, and in many other jurisdictions, authors hold **moral rights** — personal rights such as attribution and the integrity of the work — that are **inalienable**: they cannot be waived or sold even if the author wants to. So a pure "I give up everything" dedication simply cannot take full effect in those countries.

The net result: a one-line "this is public domain" note is well-intentioned but legally shaky, especially across borders. The deep cross-border treatment lives in [licensing across jurisdictions](/learn/software-licensing/licensing-across-jurisdictions/); for now, just hold the idea that "public domain" doesn't mean the same thing everywhere.

## The tools that actually do the job

Because a bare dedication is fragile, several tools were created to dedicate work to the public domain *robustly*. The good ones share a clever two-part structure: **waive every right you can, and as a fallback, grant an extremely broad permissive license for anything that can't be waived.** That fallback is what makes them work internationally.

| Tool | What it is | Notes |
|---|---|---|
| **CC0** | Creative Commons "no rights reserved" dedication + fallback license | **Recommended.** Carefully drafted; handles moral-rights jurisdictions via the fallback. Used for data, docs, and code. |
| **Unlicense** | A public-domain dedication aimed at software | Popular and simple, but its legal drafting is more debated than CC0's. |
| **0BSD** ("Zero-clause BSD") | A normal *permissive license* with the attribution requirement removed | Not a dedication at all, but achieves a similar "do anything, no conditions" effect with rock-solid, familiar license mechanics. |
| **WTFPL** | "Do What The F*** You Want To Public License" | Mentioned for completeness; **discouraged** — its joke tone and thin drafting make it a poor choice for anything serious. |

A few practical points:

- **CC0** is the safest pick when you genuinely want public-domain-like behavior. Its fallback license means it still grants near-total freedom even in countries where you can't actually abandon copyright. (One caveat worth knowing: CC0 explicitly does *not* grant patent rights, so it differs from licenses with an express patent grant like [Apache 2.0](/learn/software-licensing/apache-license/).)
- **0BSD** is often the *better* choice for source code specifically. Because it's a plain permissive license rather than a dedication, it avoids the "can you really abandon copyright?" question entirely while giving users the same no-conditions experience. Many people who want "MIT but without even the attribution line" reach for 0BSD.
- **The Unlicense** works and is widely used, but if a lawyer is in the room, CC0 or 0BSD will usually be preferred for stronger drafting.
- **WTFPL** is a meme. Avoid it for real projects — the lack of a warranty disclaimer alone is a reason to skip it.

```text
The robust-dedication pattern (CC0, Unlicense):
  1. Waive all copyright and related rights, worldwide, to the extent allowed.
  2. As a fallback, grant a maximally broad license for any rights that
     cannot be waived (e.g., moral rights in the EU).
  -> result: "do anything," enforceable almost everywhere.
```

## The big trap: no license is NOT public domain

This is the most important practical takeaway in the lesson, and it catches people constantly. **A project with no license file is not in the public domain. It is the most restrictive state possible.**

Remember from [what is a software license](/learn/software-licensing/what-is-a-software-license/) that copyright is automatic and the default is **all rights reserved**. So when you find code on a public repository with no LICENSE file, the absence of a license doesn't mean "take it freely" — it means the author has granted *no permissions at all*. Legally, you may look at it, but you have no right to copy, modify, or redistribute it. Posting code publicly does not waive copyright.

So the spectrum has a surprising shape at its edges. Public domain (CC0/0BSD) is the *freest* state; "no license" is the *least* free. They look superficially similar — neither has a long license text — but they are opposites. If you mean "anyone can use this," you must *say so* with an actual dedication or license; silence does the reverse of what you intend.

## When public domain makes sense

Given the complications, when is public-domain dedication actually the right call? A few clear cases:

- **Specifications and standards.** When you want maximum, frictionless adoption of a format or protocol, you don't want attribution lines or license terms getting in the way. Dedicating the spec text to the public domain (or CC0) removes all barriers.
- **Tiny snippets and examples.** A ten-line example in documentation isn't worth a license header. CC0 or 0BSD lets people copy it into their own code without bookkeeping. (Below a threshold, very short snippets may not even be copyrightable, but a dedication removes the doubt.)
- **Government works.** In the United States, works created by **federal government employees** in the course of their duties are, by statute, **not subject to copyright** — they're in the public domain from creation. This is why a lot of US government software and data can be used freely. Note this is US-specific: other governments handle their works differently (the UK, for instance, uses Crown Copyright), another reason jurisdiction matters.
- **Maximizing reuse over recognition.** If you simply don't care about credit or control and want the lowest-friction reuse imaginable, a robust dedication is the way to express that.

For most ordinary software projects, a short permissive license like [MIT](/learn/software-licensing/mit-and-bsd-licenses/) is still the more common choice, because the one remaining condition — keep the notice — is trivial and the license is universally understood. But when you truly want *nothing* asked of users, reach for CC0 or 0BSD rather than a bare "public domain" note. How to weigh these options deliberately is covered in [choosing an open-source license](/learn/software-licensing/choosing-an-oss-license/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — with no license, default copyright reserves all rights, so it's the most restrictive state, not public domain." markdown="0">
  <p class="knowledge-check__q">Quick check: a public repository has code but no LICENSE file. What does that mean?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The code is in the public domain — anyone can use it freely</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">All rights are reserved by default — you have no permission to copy, modify, or redistribute it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It's automatically licensed under MIT because it was posted publicly</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Public domain is the freest state** — no rights reserved, no conditions, not even attribution; freer than MIT.
- **Clean dedication is legally tricky** — copyright is automatic, the US has no formal abandonment procedure, and moral rights can't be waived in many countries.
- **Use a real tool, not a bare note** — CC0 (a dedication plus a fallback license) is recommended; 0BSD is often the best choice for source code; the Unlicense works but is less crisply drafted; WTFPL is discouraged.
- **No license is NOT public domain** — it's the most restrictive default (all rights reserved); silence does the opposite of "use freely."
- **Public domain fits specific cases** — specs and standards, tiny snippets, and US federal government works (US-specific — other countries differ).
- **For most projects, permissive still wins** — a short MIT-style license is universally understood; reach for CC0/0BSD only when you want truly nothing asked of users.

Next up: what happens when you try to combine code under different licenses — and why some combinations simply aren't allowed. See [License compatibility](/learn/software-licensing/license-compatibility/).
