---
slug: permissive-compliance
title: Meeting permissive obligations
description: MIT, BSD, and Apache are easy to comply with — but easy is not the same as nothing. Learn exactly what permissive licenses require when you ship, Apache's extra NOTICE and state-changes terms, and where to put attributions, with GopherTrunk's own THIRD_PARTY_LICENSES as a worked example.
keywords: permissive license compliance, MIT license attribution, BSD license notice, Apache 2.0 NOTICE file, third party licenses file, license attribution, open source notices, binary distribution attribution, THIRD_PARTY_LICENSES, software compliance
level: intermediate
status: full
faq:
  - q: "Do I really have to include the license text if I only ship a binary?"
    a: "Yes. MIT, BSD, and Apache all require the copyright notice and license text to travel with **the software**, whether you distribute source or a compiled binary. For a binary you can't put it in a code comment, so you collect the notices into a file or screen that ships with the product — an about box, a bundled `THIRD_PARTY_LICENSES` file, or in-app credits."
  - q: "What's the difference between Apache's NOTICE file and the LICENSE file?"
    a: "The **LICENSE** is the Apache-2.0 license text itself. The **NOTICE** is a separate file the original authors may include with required attributions; Apache-2.0 says you must pass along the relevant parts of any NOTICE you received. So Apache compliance is: include the license text, preserve copyright notices, propagate NOTICE contents, and state if you modified files."
  - q: "Is one big THIRD_PARTY_LICENSES file enough?"
    a: "For most products, yes — a single, accurate file (or about-box section) that lists each dependency with its copyright line and full license text satisfies MIT, BSD, and Apache attribution. The key is that it's **complete and reachable** by the user. GopherTrunk does exactly this with its `/THIRD_PARTY_LICENSES.md`."
---
# Meeting permissive obligations

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Easy isn't nothing** — permissive licenses ask little, but that little is mandatory. **Ship the text and the notice** — include the full license and the copyright line for every permissive dependency. **Apache adds extras** — propagate the NOTICE file and flag files you changed. **Put it somewhere reachable** — an about box, a `THIRD_PARTY_LICENSES` file, or docs that travel with the product.
</div>

[Permissive licenses](/learn/software-licensing/mit-and-bsd-licenses/) — MIT, the BSD family, and [Apache 2.0](/learn/software-licensing/apache-license/) — are the friendliest licenses to ship in a commercial product. They let you keep your own code closed, charge what you like, and combine almost anything. But "friendly" is not "free of obligations," and the single most common compliance failure in the industry is shipping permissive code while quietly dropping its attribution. By the end of this lesson you'll know precisely what these licenses require when you distribute, the extra terms Apache adds, where to place attributions for source and binary releases, and what a correct attribution looks like — using GopherTrunk's own dependency file as a real example.

## The core obligation: notices travel with the software

Strip away the legalese and every mainstream permissive license boils down to one duty when you distribute: **carry the copyright notice and the license text along with the software.** That's the whole deal for MIT and BSD. You may do nearly anything else — modify, relicense your combined work, sell it, keep your additions secret — provided those notices ride along.

Read the operative line from the MIT license itself:

```text
The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.
```

"All copies or substantial portions" is the key phrase. If your product contains a substantial portion of MIT-licensed code — which a linked library certainly is — the notice and license text must be included with what you ship. The BSD licenses say essentially the same thing, with the 3-clause variant adding that you can't use the authors' names to endorse your product without permission.

The mistake to avoid is treating attribution as optional politeness. It is a **condition of the license.** If you don't meet it, your permission to use the code lapses, and you're now distributing someone's copyrighted code without a license — the same legal position as if you'd never had one.

## Apache 2.0 adds three things

[Apache 2.0](/learn/software-licensing/apache-license/) is still permissive, but it spells out more than MIT or BSD. When you ship Apache-licensed code, you owe four things:

1. **Include the license.** Provide a copy of the Apache-2.0 license text.
2. **Preserve notices.** Keep all copyright, patent, trademark, and attribution notices from the source.
3. **Propagate the NOTICE file.** If the project shipped a `NOTICE` file, you must include the readable attribution notices it contains in your distribution (in your own NOTICE, the docs, or a displayed screen). The NOTICE is *separate* from the license — it's the authors' required-attribution text.
4. **State your changes.** If you modified any Apache-licensed files, the modified files must carry prominent notices that you changed them.

There's also the [patent grant](/learn/software-licensing/apache-license/): Apache-2.0 gives you a license to the contributors' patents that read on the code, and that grant terminates if you sue a contributor over patents in that code. You don't have to *do* anything for the patent grant — but it's a big reason Apache is preferred for commercial use.

| Requirement | MIT | BSD (2/3-clause) | Apache 2.0 |
|-------------|-----|------------------|------------|
| Include license text | Yes | Yes | Yes |
| Preserve copyright notice | Yes | Yes | Yes |
| Propagate a NOTICE file | n/a | n/a | Yes, if one exists |
| Mark modified files | No | No | Yes |
| No-endorsement clause | No | 3-clause only | Yes (trademark) |
| Express patent grant | No | No | Yes |

## Where to put the attributions

The license says the notices must accompany the software; it doesn't dictate exactly where. What matters is that the attribution is **complete** and **reachable by the person who receives your product.** Common, accepted placements:

- **An "About" or "Credits" screen** in a GUI or mobile app — often an "Open Source Licenses" sub-screen listing each dependency and its license.
- **A bundled file**, conventionally named `THIRD_PARTY_LICENSES`, `NOTICES`, or `licenses.txt`, shipped alongside the binary or inside the installer/package.
- **Documentation or a docs page** that travels with the product (printed manual, in-app help, or a docs site you point users to).
- **For source distributions**, the upstream `LICENSE` and `NOTICE` files preserved in the tree, which is usually automatic if you don't delete them.

### Binary vs source distribution

The obligation is the same either way — the *mechanics* differ:

- **Source distribution:** keep each dependency's `LICENSE`/`NOTICE` where they sit. If you vendor dependencies into your tree, don't strip their license files.
- **Binary distribution:** the source comments and license files don't ship with a compiled artifact, so you must **collect** the notices into something that does — that bundled file or about screen. A statically-linked binary still "contains" its dependencies, so their notices still must accompany it.

## A worked example: GopherTrunk

This is not hypothetical — GopherTrunk does exactly this. GopherTrunk is itself licensed **Apache-2.0** (see [`/LICENSE`](/learn/software-licensing/apache-license/)), and because it ships as a single statically-linked binary, every dependency it links in must have its notice carried along. GopherTrunk collects them in **`/THIRD_PARTY_LICENSES.md`** at the repository root — a single, reachable attribution file that lists each direct dependency with its version, SPDX license identifier, and a link upstream, plus the full text for in-tree code derived from other projects.

A correct attribution entry is simple. For an MIT dependency it looks like this:

```text
github.com/charmbracelet/bubbletea  v1.3.10  MIT
  Copyright (c) 2020-2023 Charmbracelet, Inc.

  Permission is hereby granted, free of charge, to any person
  obtaining a copy of this software ... [full MIT license text]
  ... THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND.
```

The essential parts are the **identity** (which component), the **copyright line** (who holds it), and the **full license text** (so the recipient has their own copy of the grant and the warranty disclaimer). Repeat that for every permissive dependency and you've met the obligation. The next lesson, [auditing dependencies](/learn/software-licensing/auditing-dependencies/), covers how to make sure that list is complete — including the transitive dependencies you didn't add by hand.

<div class="knowledge-check" data-quiz data-correct-msg="Right — Apache-2.0 additionally requires propagating the NOTICE file and marking files you modified, on top of including the license and preserving notices." markdown="0">
  <p class="knowledge-check__q">Quick check: what does Apache 2.0 require that MIT and BSD do not?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">That you release your own source code under Apache 2.0</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">That you pay a royalty for commercial use</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">That you propagate any NOTICE file and mark files you modified</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **The core duty is attribution** — include the full license text and the copyright notice for every permissive dependency you distribute, in source or binary form.
- **It's mandatory, not polite** — dropping the notice voids your permission to use the code, turning use into infringement.
- **Apache adds three extras** — propagate the NOTICE file, mark modified files, and benefit from its patent grant.
- **Put notices somewhere reachable** — an about box, a bundled `THIRD_PARTY_LICENSES`/`NOTICES` file, or docs that ship with the product.
- **GopherTrunk is a real example** — Apache-2.0 licensed, it collects every dependency's notice in `/THIRD_PARTY_LICENSES.md` because its single static binary carries them all.

Next up: the harder case, and the one people fear most. See [Copyleft in products: the viral question](/learn/software-licensing/copyleft-in-products/).
