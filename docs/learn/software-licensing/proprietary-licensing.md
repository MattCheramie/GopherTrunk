---
slug: proprietary-licensing
title: Proprietary & closed-source licensing
description: Proprietary software sits at the opposite end from open source — "all rights reserved" by default, with the license granting you a narrow, restricted right to use a copy. Learn what proprietary licenses typically forbid and why you license rather than buy software.
keywords: proprietary software, closed source, all rights reserved, EULA, software license vs purchase, no reverse engineering, no redistribution, seat license, freeware, shareware, trialware, license not sale
level: beginner
status: full
faq:
  - q: "If I pay for proprietary software, don't I own it?"
    a: "Almost never. You own the **copy** — the bits on your disk — but you don't own the software itself. What you buy is a **license**: a permission to use the software under the seller's terms. The copyright stays with the vendor, which is why the EULA can restrict copying, sharing, and modification even after you've paid."
  - q: "Is closed-source the same as proprietary?"
    a: "They overlap heavily but aren't identical. **Closed-source** describes the *delivery* — you get a binary, not the source code. **Proprietary** describes the *rights* — the vendor reserves them and licenses you a narrow set. Most proprietary software is also closed-source, but you can have proprietary licenses over visible source (that's [source-available](/learn/software-licensing/source-available-licenses/)) and closed binaries that are nonetheless freely redistributable."
  - q: "Are freeware and shareware open source?"
    a: "No. **Freeware** costs nothing but is still proprietary — you get a free copy under a restrictive license, not the freedoms open source grants. **Shareware** and **trialware** are the same idea with a payment or time gate. \"Free of charge\" is about price; **open source** is about freedom. They're unrelated axes."
---
# Proprietary & closed-source licensing

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Proprietary is the opposite end** — the vendor reserves the rights and licenses you a narrow slice of them. **"All rights reserved" by default** — copyright law gives the owner control, and a proprietary license hands back only what it explicitly says. **You license, you don't buy** — you own your copy, not the software. **Freeware, shareware, trialware** — all proprietary; "free of charge" is not "free as in freedom."
</div>

Open source defines itself by what it lets you do. Proprietary software defines itself by what it holds back. It's the far end of the [licensing spectrum](/learn/software-licensing/the-licensing-spectrum/) from the permissive licenses we've been studying: instead of granting broad freedoms, a proprietary license starts from total control and lends you a carefully limited right to use the product. By the end of this lesson you'll understand what "all rights reserved" means in practice, the restrictions a typical proprietary license imposes, the crucial idea that you license software rather than buy it, the clauses you'll see again and again, and where freeware, shareware, and trialware fit in.

## "All rights reserved" is the default

Here's the foundation everything else rests on, and it surprises people: under copyright law, the author of a piece of software automatically holds the exclusive rights to copy, modify, and distribute it the moment it's written. Nobody else may do those things **without permission**. You can read more about why in [Copyright & software ownership](/learn/software-licensing/copyright-and-software-ownership/), but the short version is that the default state of any code is *closed*.

This is the meaning of the old phrase **"all rights reserved."** It signals that the owner is keeping every right the law grants and is not waiving any of them. Open-source and permissive licenses are *generous exceptions* to this default — the author voluntarily hands rights to everyone. Proprietary licensing is what you get when the author keeps the default and grants you only a thin, specific set of permissions in return for payment (or for agreeing to terms).

So "proprietary" isn't a special legal mechanism you have to invent. It's what software *is* until someone chooses to open it up. A proprietary license is simply the document that says exactly which slivers of the reserved rights you're allowed to exercise.

## What a proprietary license typically restricts

Because the license starts from "you get nothing" and adds back only named permissions, the interesting content is usually the list of restrictions. The exact wording varies, but the same prohibitions show up across nearly every commercial product:

| Restriction | What it means in practice |
|---|---|
| **No source code** | You receive a binary (closed-source). The human-readable source is a [trade secret](/learn/software-licensing/other-ip-in-software/) the vendor keeps. |
| **No redistribution** | You may not give, sell, or share copies. The license is for *you* (or your organization), not the world. |
| **No modification** | You may not alter, adapt, or build derivative works from the software. |
| **No reverse engineering** | You may not decompile, disassemble, or otherwise try to recover the source or internal design. |
| **Seat / usage limits** | Use is capped — a number of users, devices, CPUs, or installs. Exceeding the cap is a breach. |
| **No transfer / assignment** | You often can't sell or give your license to someone else without the vendor's consent. |

Each of these is a right the vendor reserved and declined to grant. The "no reverse engineering" clause is worth a special note: it's extremely common, but its enforceability varies by jurisdiction. In the **EU**, for instance, the law explicitly permits some decompilation for *interoperability* purposes regardless of what the contract says, and a few US cases have limited overbroad bans. The clause is real and you should respect it, but "the EULA says so" doesn't make every restriction airtight everywhere — a theme we return to in [Licensing across jurisdictions](/learn/software-licensing/licensing-across-jurisdictions/).

## You license software; you don't buy it

This is the single idea most people get wrong, so it's worth stating bluntly: **when you "buy" proprietary software, you are not buying the software.** You are buying a **license** — a permission to use it on the vendor's terms — and a copy to run it from.

The slogan to remember is *"you own a copy, not the software."* The DVD, the download, the activation key — those are yours. The underlying program, and the copyright in it, remain the vendor's property. That's why:

- The vendor can dictate how you use *your* copy (one machine, no sharing, no tinkering).
- The license can be **revoked** or **expire** even though you paid, if you breach the terms or the subscription lapses.
- You can't legally resell a download the way you'd resell a used book — the "first sale" doctrine that lets you resell physical goods is murky and limited for licensed digital software.

This is a genuine difference from owning a physical object, and it's not a trick — it's how copyright-based licensing works. We dig into *why* a license isn't the same as a sale, and when it behaves more like a contract, in [License vs contract](/learn/software-licensing/license-vs-contract/).

## Typical clauses you'll meet

Open up almost any proprietary **EULA** (End User License Agreement — covered in depth in [EULAs & terms of service](/learn/software-licensing/eula-and-tos/)) and you'll find a recognizable skeleton. Here, briefly, is what to expect — the full clause-by-clause tour is in [Key clauses in a commercial license](/learn/software-licensing/key-commercial-clauses/):

- **Grant of license** — the narrow permission you *do* get ("a non-exclusive, non-transferable right to install and use one copy…").
- **Restrictions** — the prohibitions from the table above, spelled out.
- **Ownership / IP** — a reminder that the vendor owns everything and you own nothing but your copy.
- **Term & termination** — how long the license lasts and what ends it (and what happens to your right to use when it ends).
- **Warranty disclaimer** — usually an all-caps statement that the software is provided "AS IS."
- **Limitation of liability** — a cap on what the vendor will pay if things go wrong.

A short example of the grant-and-restriction core, the kind of language you'll see near the top of a EULA:

```text
Subject to your compliance with this Agreement, Vendor grants you a
limited, non-exclusive, non-transferable, revocable license to install
and use one (1) copy of the Software for your internal business use.

You may NOT: (a) copy or redistribute the Software; (b) modify or create
derivative works; (c) reverse engineer, decompile, or disassemble it;
or (d) transfer your rights to any third party.
```

Notice the shape: a tightly bounded grant, then a list of "you may not." That contrast *is* proprietary licensing in miniature.

## Freeware, shareware, and trialware

A common confusion is that software which costs nothing must be open or unrestricted. It isn't. These zero- or low-cost models are all **proprietary variants** — restrictive licenses that happen to adjust the price or the trial mechanics:

| Variant | Price model | Still proprietary? |
|---|---|---|
| **Freeware** | Free of charge, indefinitely | Yes — closed binary, restrictive license, no source |
| **Shareware** | Free to try, pay to keep using (honor system or nag screens) | Yes |
| **Trialware** | Free for a limited time or feature set, then locks | Yes |
| **Freemium** | Free tier, pay to unlock more (see [commercial models](/learn/software-licensing/commercial-license-models/)) | Yes |

The key insight: **"free of charge" and "free as in freedom" are different axes entirely.** Freeware can be downloaded at no cost yet forbid copying, modification, and reverse engineering just as strictly as a $500 product. You got a free *copy*; you did not get *freedom*. Plenty of well-known utilities are freeware — proprietary software the author chose not to charge for. That's a pricing decision, not a licensing philosophy.

These models lead naturally into how vendors actually structure paid software, which is the subject of the next lesson.

<div class="knowledge-check" data-quiz data-correct-msg="Right — you buy a license (permission to use) and a copy; the software itself and its copyright stay with the vendor." markdown="0">
  <p class="knowledge-check__q">Quick check: you pay for a copy of a closed-source app. What did you actually acquire?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">A license to use a copy under the vendor's terms — not ownership of the software</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Full ownership of the software, including the right to modify and resell it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The source code, since you paid for the product</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Proprietary is open source's opposite** — the vendor reserves the rights and grants you a narrow, named set of permissions.
- **"All rights reserved" is the default** — copyright closes all software automatically; open licenses are the exception, not the rule.
- **Typical restrictions** — no source, no redistribution, no modification, no reverse engineering, plus seat/usage caps and transfer limits.
- **You license, you don't buy** — you own your copy, not the software; the license can expire or be revoked even after payment.
- **Recognizable clauses** — grant, restrictions, ownership, term/termination, warranty disclaimer, liability cap.
- **Freeware, shareware, trialware are proprietary** — free of charge is not the same as free as in freedom.

Next up: how vendors actually package and charge for proprietary software — perpetual vs subscription, per-seat vs concurrent, and more. See [Commercial license models](/learn/software-licensing/commercial-license-models/).
