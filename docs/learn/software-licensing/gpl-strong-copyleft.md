---
slug: gpl-strong-copyleft
title: "The GPL: strong copyleft"
description: The GNU General Public License is the original copyleft license — it gives users broad freedoms and requires that anyone distributing the software pass those same freedoms on, in source form, under the same license.
keywords: GPL, GNU General Public License, strong copyleft, GPLv2, GPLv3, copyleft, derivative work, distribution trigger, corresponding source, tivoization, license compatibility, free software, viral license
level: intermediate
status: full
faq:
  - q: "If I use GPL software inside my company, do I have to publish my source code?"
    a: "No. The GPL's obligations are triggered by **distribution** (GPLv3 calls it *conveying*), not by use. Running GPL software on your own servers, or modifying it for purely internal use, does not require you to give source to anyone. The duty to share source only arises when you hand the binary to someone outside your organization."
  - q: "What's the practical difference between GPLv2 and GPLv3?"
    a: "GPLv3 adds three big things: an **anti-tivoization** clause (if you ship GPLv3 code in a device, you must also provide the keys or info users need to install modified versions), an **explicit patent grant and retaliation** terms, and better **compatibility** with some other licenses (like Apache 2.0). GPLv2 has none of these and is **not** automatically upgradeable unless the project said \"v2 or later.\""
  - q: "Does linking my proprietary program to a GPL library make my program GPL?"
    a: "Often, yes — that's the crux of the \"derivative work\" debate. The Free Software Foundation's position is that linking (static *or* dynamic) generally creates a combined work that must be GPL when distributed. The legal boundary is genuinely contested, which is exactly why the [LGPL and other weak-copyleft licenses](/learn/software-licensing/weak-copyleft-licenses/) exist to carve out a safe path for linking."
---
# The GPL: strong copyleft

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The original copyleft** — the GPL uses copyright to *guarantee* freedom rather than restrict it. **The bargain is reciprocity** — you get the four freedoms, and in return you must pass them on whenever you distribute. **The trigger is distribution, not use** — internal use and modification carry no source-sharing duty; *conveying* the software does. **Same license, complete source** — recipients must get the complete corresponding source under the same GPL terms.
</div>

In the [permissive vs copyleft](/learn/software-licensing/permissive-vs-copyleft/) lesson you met the two great families of open-source license. This lesson goes deep on the most influential copyleft license of all: the GNU General Public License, or **GPL**. By the end you'll understand the bargain the GPL strikes, what specifically triggers its obligations (and what does *not*), what "complete corresponding source" really means, how GPLv2 and GPLv3 differ, why "derivative work" and linking are the hardest questions in the whole field, and the honest strengths and weaknesses that make the GPL both beloved and avoided.

## The idea: copyright turned inside out

Most licenses use copyright to *withhold* permissions. The GPL, written by Richard Stallman and the Free Software Foundation (FSF) in 1989, does the opposite: it uses the author's copyright to **guarantee** that the software — and every version derived from it — stays free for users. This judo move is called **copyleft**.

The GPL grants four freedoms: to **run** the program for any purpose, to **study and modify** it (which requires source code), to **redistribute** copies, and to **distribute your modified versions**. To keep those freedoms from being stripped away by a later distributor, the license attaches one condition: anyone who passes the software on must pass on the same freedoms, under the same license. Because the freedoms are protected even after the code changes hands, the GPL is called **strong copyleft** — its reach extends across the whole combined program, not just the original files.

This is the difference from permissive licenses like [MIT and BSD](/learn/software-licensing/mit-and-bsd-licenses/). MIT lets a recipient take your code and ship it inside a closed product. The GPL forbids that closing-off: the freedoms must survive every hop.

## The bargain: you get the freedoms, you pass them on

It helps to think of the GPL as a deal. The author offers you something valuable — the right to use, read, change, and share working software — for free. In exchange, you accept one obligation that only matters if you choose to distribute: you must extend the same deal to whoever you give it to.

If you never distribute, you owe nothing. You can run a modified GPL program for years, on a thousand machines, and never publish a line. The obligation is dormant until you **convey** the software to someone else. At that moment the bargain activates: your recipients get the four freedoms too, including the source code they'd need to exercise them.

This reciprocity is the whole point. It's why the GPL is sometimes called "viral" — a loaded word we'll handle carefully in [Copyleft in products: the viral question](/learn/software-licensing/copyleft-in-products/). The freedoms spread because the license requires them to.

## What triggers the obligations: distribution, not use

This is the single most misunderstood part of the GPL, so be precise about it.

**Internal use does not trigger anything.** Running GPL software, deploying it on your own servers, or modifying it for your own purposes are all unconditionally allowed and impose no duty to share. The GPL is not a "use" license; it is a **distribution** license.

The obligation fires when you **distribute** (GPLv2's word) or **convey** (GPLv3's broader, more carefully defined word) the software to a third party. Conveying means making the software available to others in a way that lets them get copies. Handing someone a binary, shipping a device with the software inside, putting an installer on a download page — those convey. Letting users *interact* with the software over a network, without giving them a copy, generally does **not** count as conveying under the plain GPL.

That last point is a famous gap. A company can run modified GPL software as a web service, never distribute the binary, and owe its users no source at all. Closing that gap is the entire reason the [AGPL](/learn/software-licensing/agpl-network-copyleft/) exists — that's the next lesson.

| Activity | Triggers GPL source obligation? |
|---|---|
| Running the program | No |
| Modifying it for internal use | No |
| Offering it as a network service (plain GPL) | No — see [AGPL](/learn/software-licensing/agpl-network-copyleft/) |
| Shipping the binary to a customer | **Yes** |
| Embedding it in a device you sell | **Yes** |
| Publishing it for download | **Yes** |

## What you must provide: complete corresponding source

When the trigger fires, the core obligation is to provide the **complete corresponding source code** under the same GPL license. "Complete corresponding source" is a deliberately demanding phrase. It means everything a recipient needs to **build, install, and run** the exact version you distributed — not just the files you happened to edit.

In practice that includes the source for the whole program, the **build scripts and makefiles**, and the definitions of interfaces it uses to build. It does **not** require you to include the standard system libraries or the compiler — the so-called "system library" exception keeps you from having to ship the operating system. GPLv3 also requires the **Installation Information** (sometimes "installation scripts and keys") needed to install a modified version on the same device, when the software ships inside consumer hardware.

You can deliver the source two main ways: **alongside** the binary (the simplest), or with a **written offer** valid for three years to supply the source on request. You must also keep all license and copyright notices intact and state any changes you made.

```text
A typical GPL distribution includes, at minimum:
  - the complete source for the program as shipped
  - the Makefile / build scripts to compile it
  - the unmodified GPL license text (COPYING)
  - notices of what you changed and when
  - (GPLv3, consumer device) installation info / keys
```

## GPLv2 vs GPLv3: what changed and why it matters

There are two GPL versions in wide use, and they are **not** interchangeable. Many famous projects (the Linux kernel, for one) are GPLv2-only and have deliberately not moved to v3. Choosing or combining GPL code means knowing which version you're dealing with.

| Topic | GPLv2 (1991) | GPLv3 (2007) |
|---|---|---|
| **Anti-tivoization** | None — you can lock down hardware so users can't run modified code | Requires you to provide keys/info so users *can* install modified versions on consumer devices |
| **Patent grant** | Implicit at best; weak | Explicit patent license from contributors, plus retaliation terms |
| **Compatibility** | Incompatible with Apache 2.0 and several others | Compatible with Apache 2.0 and a wider set |
| **Anti-DRM language** | None | Clarifies the software isn't part of an "effective technological measure" |
| **Internationalization** | US-centric wording | Cleaner, more jurisdiction-neutral wording |
| **Cure period** | A single violation can permanently terminate | Offers a way to cure a first violation and be reinstated |

**Tivoization** — the headline GPLv3 concern — is named after TiVo, which shipped GPLv2 Linux but used hardware signature checks so a modified kernel wouldn't boot. Technically the source was provided; practically, the freedom to run modified versions was defeated. GPLv3 closes that loophole for consumer devices.

The **patent** changes matter too: like [Apache 2.0](/learn/software-licensing/apache-license/), GPLv3 gives recipients an explicit license to the patents a contributor holds over their contribution, and it discourages patent aggression. GPLv2 predates the era when this was a pressing worry.

One subtlety: a project licensed "GPLv2 **or later**" lets you choose v3, but "GPLv2 **only**" does not. Always check the exact grant.

## The hard part: "derivative work" and linking

The GPL's reach extends to anything that counts as a **derivative work** (or, in GPLv3 terms, a work "based on" the program) of the GPL code. Pinning down that boundary is the single thorniest question in open-source licensing.

Clear cases are easy. If you edit GPL source and ship the result, that's covered. If you fork a GPL project, the fork is GPL. The hard case is **linking**: if your otherwise-proprietary program calls into a GPL library, is the combination a derivative work that must be released under the GPL?

The FSF's position is that linking — whether **static** (the library is compiled into your binary) or **dynamic** (loaded at runtime) — generally creates a single combined work, so distributing it triggers the GPL on the whole thing. Many lawyers agree for static linking; the dynamic-linking case is more contested and has rarely been fully litigated, so it remains a genuine legal gray area rather than a settled rule. Communicating with a separate GPL program at arm's length — say, over a pipe, a socket, or as a separate process — is widely considered *not* to create a derivative work.

Because this uncertainty scares off legitimate use, the FSF created the **Lesser GPL (LGPL)** precisely to permit linking from non-GPL programs. That, along with file-level options like the MPL, is the subject of [Weak copyleft: LGPL, MPL, EPL](/learn/software-licensing/weak-copyleft-licenses/). And the question of how all this plays out in shipping products gets its own deep treatment in [Copyleft in products](/learn/software-licensing/copyleft-in-products/).

## Strengths and weaknesses, honestly

The GPL's **strengths** are real and have shaped modern computing. It keeps software free in a way permissive licenses cannot: a GPL project can't be quietly absorbed into a closed product, so improvements tend to flow back to everyone. It prevents **proprietary capture**, where a company takes a community's work and gives nothing back. For projects whose authors care most about user freedom, it is the strongest tool available, and it built ecosystems (GNU, Linux, much of the toolchain you compile with) on exactly that promise.

The **weaknesses** are equally real. The strong reach creates **compatibility headaches**: GPL code can't be freely combined with code under licenses whose terms conflict, and even GPLv2 and GPLv3 code can't be mixed unless the v2 side said "or later." This is the heart of [license compatibility](/learn/software-licensing/license-compatibility/). The same reach produces **commercial hesitancy**: companies wary of accidentally GPL-ing their proprietary code often ban GPL dependencies outright, which can shrink a project's adoption. And the linking ambiguity above means even careful engineers sometimes can't be sure where the line is without legal help.

There's no universally "right" answer — it's a values choice. Maximize freedom-for-users and accept narrower adoption (GPL), or maximize adoption and accept proprietary forks (permissive). [Choosing an open-source license](/learn/software-licensing/choosing-an-oss-license/) walks through making that call deliberately.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the GPL's source-sharing duty is triggered by distributing (conveying) the software, not by merely using or modifying it internally." markdown="0">
  <p class="knowledge-check__q">Quick check: what triggers the GPL's obligation to provide source code?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Running or modifying the software for your own internal use</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Distributing (conveying) the software to a third party</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Simply downloading a copy of the GPL software</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **The GPL is the original copyleft** — it uses copyright to guarantee user freedom rather than to withhold it, and its reach extends across the whole combined program (strong copyleft).
- **The bargain is reciprocity** — you receive the four freedoms and must pass them on, in source form and under the same license, whenever you distribute.
- **Distribution is the trigger, not use** — internal use and modification impose no source duty; conveying the software to others does. Plain GPL doesn't cover network-only access (that's the AGPL).
- **You must provide complete corresponding source** — everything needed to build, install, and run the version you shipped, with notices intact.
- **GPLv2 and GPLv3 differ and aren't interchangeable** — v3 adds anti-tivoization, explicit patent terms, and broader compatibility; check whether a project is "v2 only" or "v2 or later."
- **Linking and "derivative work" are genuinely contested** — which is why weak-copyleft licenses exist and why the viral question gets its own lesson.

Next up: how copyleft adapts to the cloud, where software is used over a network but never handed over. See [The AGPL: copyleft over the network](/learn/software-licensing/agpl-network-copyleft/).
