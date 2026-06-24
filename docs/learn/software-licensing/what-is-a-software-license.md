---
slug: what-is-a-software-license
title: What is a software license?
description: A software license is the grant of permission that lets you use, copy, modify, or share someone else's code. Learn why code is "all rights reserved" by default and why even free projects need an explicit license.
keywords: software license, what is a software license, all rights reserved, copyright default, permission to use software, no license means no permission, open source license, license grant, using copying modifying distributing, free software license
level: beginner
status: full
faq:
  - q: "If a project on the internet has no license at all, can I use it?"
    a: "Legally, no — not safely. With **no license**, the code is \"all rights reserved\" by default, so you have no permission to copy, modify, or distribute it, even if the source is public. You might get away with reading it, but building on it exposes you to a copyright claim. The fix is to ask the author to add a license."
  - q: "Isn't putting code on a public site the same as making it free to use?"
    a: "No. Posting source publicly makes it **visible**, not **licensed**. Visibility and permission are separate things. Unless the author attached a license granting rights, the default \"all rights reserved\" still applies, and the public copy is just a copy you're allowed to look at, not one you're allowed to reuse."
  - q: "Do I need a license for my own hobby project that nobody pays for?"
    a: "Yes, if you want anyone to legally use it. Being free of charge is about **price**; a license is about **permission**, and the two are unrelated. Without a license your free project is still \"all rights reserved,\" which usually defeats the point of sharing it. Adding even a one-line permissive license fixes that."
---
# What is a software license?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Code is copyrighted the moment it's written** — the default is "all rights reserved," meaning no one else has permission to use it. **A license is the grant of permission** — it's the document that says what others may do with your code. **No license means no permission** — public source with no license is something you can look at but legally can't reuse. **Free and licensed are different things** — price and permission are separate, so even free projects need a license.
</div>

This is the first lesson of the path. Before we get into open source, copyleft, EULAs, or how to choose a license, we need the one idea the whole subject rests on: software is *owned* the instant it's created, and a license is how that owner hands out permission to everyone else. By the end of this lesson you'll understand why code defaults to "all rights reserved," what a license actually grants, why even a free personal project needs one, and why "no license" is a trap rather than a generosity.

> **This is educational material, not legal advice.** For decisions that carry real risk, consult a qualified attorney.

## Software is owned the moment it's written

Here's the part that surprises most people: you don't have to register anything, file anything, or even add a copyright notice for your code to be protected. The moment you write an original program and fix it in some tangible form — saving the file is enough — it is automatically protected by **copyright**, and you are the owner. (We'll dig into copyright itself in the [next lesson](/learn/software-licensing/copyright-and-software-ownership/).)

Copyright gives the owner a set of exclusive rights: the right to copy the work, to modify it, to distribute it, and so on. "Exclusive" is the important word. By default, *only* the owner may do those things. Everyone else has no permission at all.

This is the legal default for every piece of code on Earth, and it has a name people use as shorthand: **"all rights reserved."** Unless the owner says otherwise, all of those rights stay with them, and you have none of them.

## "All rights reserved" is the starting point

It's worth sitting with how restrictive the default really is. If a friend emails you a useful Go file with no further word, the law's default position is:

- You may **read** it (you have a lawful copy in front of you).
- You may **not** legally copy it into your own project.
- You may **not** legally modify it and ship the result.
- You may **not** legally redistribute it to anyone else.

None of that depends on the author being unfriendly or wanting money. It's simply the off state. Copyright is "deny by default," and permission has to be *granted* — it doesn't appear on its own.

## A license is permission, granted

A **software license** is exactly that grant of permission. It is a statement from the copyright owner that says, in effect, "I still own this, but here is what I allow you to do with it, and on what conditions." It turns "all rights reserved" into "these specific rights, granted to you."

A license can be tiny or enormous. One of the most popular licenses in the world, the MIT License, is a single short paragraph that says you can do almost anything as long as you keep the copyright notice. A commercial enterprise license can run many pages of conditions, payments, and restrictions. Both are doing the same fundamental job: defining the permission the owner is handing out.

```text
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software ... to use, copy, modify, merge, publish, distribute ...
```

That snippet is the heart of the MIT License. Notice what it is: not a transfer of ownership, but a *grant of permission*. The author keeps the copyright; you get rights to use the code under the stated terms.

## The four things permission usually covers

When you read any license, you're really asking which of a small set of actions it allows, and under what conditions. These four come up again and again:

| Action | What it means | Why it matters |
|--------|---------------|----------------|
| **Use** | Running the software for its purpose | Even just *running* a program can require permission, especially for commercial software |
| **Copy** | Making additional copies of the code or binary | Installing on many machines, vendoring a dependency, forking a repo |
| **Modify** | Changing the source to suit your needs | Fixing a bug, adding a feature, adapting it to your project |
| **Distribute** | Giving copies to other people | Shipping a product, publishing a fork, including it in your release |

A license might grant all four freely (most permissive open-source licenses), grant them with conditions attached (copyleft licenses ask you to share your changes under the same terms), or grant only a narrow slice (a typical commercial license lets you *use* the software but not modify or redistribute it). Keeping these four actions straight is the single most useful habit when reading any license, and it's the lens we'll use throughout this path.

## Free of charge is not the same as licensed

A common and costly confusion: people treat "free" and "open" and "public" as if they all mean "I can use it." They don't.

- **Free of charge** is about *price*. You paid nothing.
- **Licensed** is about *permission*. You were granted rights.

These are independent. Expensive proprietary software is licensed (you bought permission). A free download might be fully licensed, partly licensed, or — surprisingly often — not licensed at all. The fact that something cost nothing tells you nothing about what you're allowed to do with it. We'll map this whole landscape in [The licensing spectrum](/learn/software-licensing/the-licensing-spectrum/).

## Why even a personal project needs a license

If you publish a hobby project to a public repository and add no license, here's the unhappy result: *nobody can legally use it.* The code is visible, but it's still "all rights reserved." A developer who wants to build on your project, a company that might adopt it, even a friend who wants to fix a typo and send it back — none of them have permission. Many of them, knowing the rules, will simply walk away rather than risk it.

Posting source code in public makes it **visible**. It does not make it **usable**. Those are two different acts, and only adding a license accomplishes the second.

So if your goal in sharing is for people to actually use your work, the license isn't bureaucratic overhead — it's the whole point. Choosing the right one is its own topic ([Choosing an open-source license](/learn/software-licensing/choosing-an-oss-license/)), but the decision to have *a* license is rarely the wrong one for code you want others to use.

> A quick jurisdiction note: this "automatic copyright, no registration required" rule holds in essentially every country, thanks to an international treaty (the Berne Convention). The details of *enforcement* differ — the [cross-border lesson](/learn/software-licensing/licensing-across-jurisdictions/) covers those — but the core idea that code is owned by default is close to universal.

## "No license" is a decision, even if you didn't mean it to be

Leaving off a license feels like *not deciding*. Legally, it's a very firm decision: you've chosen the most restrictive option available. "No license" means "all rights reserved" means "no one may use this." If that's genuinely what you want — say, code you're sharing only to show, not to share — then fine. But if you wanted people to use your work, silence has the opposite effect from what you intended.

The good news is the fix is trivial: add a `LICENSE` file. GopherTrunk itself does exactly this — its `/LICENSE` file is the Apache 2.0 license, which is what makes the whole project legally usable by others. One file converts "look but don't touch" into "here's what you may do."

<div class="knowledge-check" data-quiz data-correct-msg="Right — with no license, code stays 'all rights reserved' by default, so you have no permission to reuse it even when the source is public." markdown="0">
  <p class="knowledge-check__q">Quick check: a project is on a public site with no license file. What can you legally do with the code?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Anything you like, because publishing it publicly puts it in the public domain</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Use and modify it freely, since it was offered free of charge</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Essentially nothing beyond looking at it — with no license it stays "all rights reserved"</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Code is copyrighted on creation** — the moment you write original software, you own it, with no registration or notice required.
- **The default is "all rights reserved"** — copyright is deny-by-default, so by default no one else may use, copy, modify, or distribute your code.
- **A license is a grant of permission** — it keeps ownership with the author while spelling out what others may do, and on what conditions.
- **Think in four actions** — use, copy, modify, distribute; reading any license is mostly asking which of these it grants and how.
- **Free is not licensed** — price and permission are separate, so a no-cost project still needs an explicit license to be usable.
- **"No license" blocks everyone** — public but unlicensed code is look-but-don't-touch; adding a `LICENSE` file is what makes a project usable.

Next up: where that automatic protection comes from and who actually owns the code. See [Copyright & software ownership](/learn/software-licensing/copyright-and-software-ownership/).
