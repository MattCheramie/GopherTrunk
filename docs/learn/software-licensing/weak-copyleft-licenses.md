---
slug: weak-copyleft-licenses
title: "Weak copyleft: LGPL, MPL, EPL"
description: Weak-copyleft licenses scope the share-alike duty to a single library or file instead of the whole program — so proprietary code can use the component as long as the component itself stays open.
keywords: weak copyleft, LGPL, MPL 2.0, EPL, CDDL, file-level copyleft, library copyleft, dynamic linking, static linking, relink, patent grant, business-friendly license, copyleft scope
level: intermediate
status: full
faq:
  - q: "How is weak copyleft different from the GPL?"
    a: "The [GPL is strong copyleft](/learn/software-licensing/gpl-strong-copyleft/): its share-alike duty extends to the whole combined program. Weak copyleft narrows that scope to just the licensed **component** — the library (LGPL) or the individual file (MPL). You can link or combine proprietary code with it as long as the component's own source stays open under its license."
  - q: "What's the deal with LGPL and linking?"
    a: "The LGPL lets a proprietary program link to an LGPL library without becoming LGPL, but it adds a condition: users must be able to **relink** your program against a modified version of the library. **Dynamic linking** satisfies this easily; **static linking** is allowed too but requires you to provide object files (or equivalent) so users can swap in a new library build."
  - q: "Why do so many people like MPL 2.0?"
    a: "Because its **file-level** copyleft is a clean, predictable boundary: files that were already under MPL stay open when modified, but you can add new proprietary files in the same project without them becoming MPL. It also includes an explicit **patent grant** and is written to be compatible with the GPL, which avoids many of the headaches of older copyleft licenses."
---
# Weak copyleft: LGPL, MPL, EPL

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The middle ground** — weak copyleft sits between permissive and strong copyleft. **Scope is the component, not the program** — only the licensed library or file must stay open; surrounding proprietary code can stay closed. **LGPL = library scope; MPL = file scope** — LGPL protects a whole library (with relink rights), MPL 2.0 protects modified files individually. **Business-friendly, but boundary-fussy** — easier to adopt commercially than the GPL, at the cost of having to track exactly where the copyleft line falls.
</div>

You've now seen the two ends of the spectrum: [permissive licenses](/learn/software-licensing/mit-and-bsd-licenses/) that ask almost nothing, and the [strong copyleft of the GPL](/learn/software-licensing/gpl-strong-copyleft/) that reaches across the whole combined program. This lesson covers the deliberate compromise in between: **weak copyleft**. By the end you'll understand the core idea — copyleft scoped to a component rather than the whole program — and the differences between the main weak-copyleft licenses: the LGPL, MPL 2.0, the EPL, and (briefly) the CDDL. You'll know when each fits and where their boundaries get tricky.

## The core idea: copyleft, but scoped

Strong copyleft's reach is its strength and its liability. It keeps the whole program free, but it also means proprietary software generally can't combine with GPL code without becoming GPL — which scares off a lot of legitimate use. Weak copyleft was invented to solve exactly that.

The trick is to **narrow the scope of the share-alike duty**. Instead of "any work that combines with this must be open," weak copyleft says "this *component* must stay open — but code that merely uses it can stay proprietary." The reciprocity still protects the original work and improvements to it, but it stops at the component boundary instead of spreading to the whole program.

There are two common ways to draw that boundary:

- **Library scope (LGPL).** The unit is a whole library. You may link proprietary code to the library; only the library itself (and changes to it) must stay open.
- **File scope (MPL, EPL, CDDL).** The unit is the individual source file. Files originally under the license stay open even after you modify them, but new files you write can be under any license you choose, including proprietary.

This makes weak copyleft a **business-friendly compromise**: companies can build closed products on top of these components and still contribute improvements back to the shared part.

## LGPL: copyleft for a library

The **GNU Lesser General Public License (LGPL)** was created by the FSF specifically to answer the [linking question](/learn/software-licensing/gpl-strong-copyleft/) that makes the GPL so cautious. "Lesser" signals that it asks less than the full GPL. Its bargain: a proprietary program may **link to** an LGPL library without itself becoming LGPL, **provided** the library stays open and users retain the ability to **relink** the program against a modified library.

That relink requirement is the heart of the LGPL, and it's where the static-vs-dynamic distinction matters:

| Linking style | Allowed under LGPL? | What you must do |
|---|---|---|
| **Dynamic linking** (library loaded at runtime) | Yes | Easiest case — users can already swap in a new `.so`/`.dll`, satisfying the relink right |
| **Static linking** (library compiled into your binary) | Yes | Also allowed, but you must provide object files (or source) so users can rebuild the binary against a modified library |

In both cases, any changes *to the LGPL library itself* must be released under the LGPL. What stays proprietary is your own application code that merely calls the library. Dynamic linking is the path of least resistance; static linking works but adds the obligation to ship enough (object files or the like) that a user could relink. The LGPL comes in versions tied to GPLv2 and GPLv3, inheriting the corresponding patent and anti-tivoization stances.

## MPL 2.0: copyleft per file

The **Mozilla Public License 2.0 (MPL)** — written for the Firefox/Mozilla ecosystem and now widely used well beyond it — takes the file-level approach, and it's the one many engineers find the cleanest.

The rule is simple to reason about: **any file that is under the MPL must stay under the MPL, including your modifications to it.** But files you *add* to the project can be under any license, including a proprietary one. So you can drop an MPL component into a closed-source product, modify the MPL files (those changes stay open), and write all your new code as proprietary — no relink mechanics to worry about, just "keep the MPL files open."

MPL 2.0 also includes two features that earn it goodwill:

- An **explicit patent grant** from contributors, similar in spirit to [Apache 2.0](/learn/software-licensing/apache-license/), with patent-retaliation terms.
- Deliberate **GPL compatibility** (unless a file is specifically marked "Incompatible With Secondary Licenses"), which sidesteps a class of [compatibility problems](/learn/software-licensing/license-compatibility/) that plagued earlier copyleft licenses.

The combination — predictable boundary, real patent protection, broad compatibility — is why MPL 2.0 is one of the best-liked copyleft licenses in modern practice.

## EPL and CDDL: the file-level cousins

Two more file-level weak-copyleft licenses come up often enough to know.

The **Eclipse Public License (EPL)** is the license of the Eclipse Foundation and much of the Java ecosystem (the Eclipse IDE, many Jakarta EE and tooling projects). Like the MPL it is **file-level** copyleft with an explicit patent grant. Its distinctive wrinkle is a **commercial contributor** indemnity provision: a contributor who distributes the work commercially takes on some responsibility to defend others against claims related to its commercial use. EPL 2.0 also added an option to designate GPL compatibility. Practically, EPL behaves much like MPL for the question that matters most — your own files can stay proprietary, the EPL files stay open.

The **Common Development and Distribution License (CDDL)**, written by Sun for projects like OpenSolaris and ZFS, is another file-level, MPL-derived license. It's worth knowing mainly for one famous friction point: the CDDL is widely considered **incompatible with the GPL**, which is the reason ZFS can't be shipped as part of the mainline GPLv2 Linux kernel. It's a clean illustration that even two reasonable open-source licenses can refuse to combine — the subject of the next module's [license-compatibility lesson](/learn/software-licensing/license-compatibility/).

## Comparing the scope and trigger

The single most useful way to hold these in your head is by **what the copyleft attaches to** and **what stays proprietary**:

| License | Copyleft scope | Proprietary code can… | Notable feature |
|---|---|---|---|
| **GPL** (for contrast) | Whole combined program | …generally not combine without becoming GPL | Strong copyleft |
| **LGPL** | The library | …link to the library if users can relink | Relink requirement; static-link conditions |
| **MPL 2.0** | Each individual file | …live in new files alongside MPL files | Patent grant; GPL-compatible |
| **EPL** | Each individual file | …live in new files alongside EPL files | Patent grant; commercial-contributor indemnity |
| **CDDL** | Each individual file | …live in new files alongside CDDL files | File-level; **GPL-incompatible** |
| **MIT/BSD** (for contrast) | Nothing (permissive) | …do almost anything | No share-alike at all |

Notice the trend: as you move from GPL down toward MIT, the unit the copyleft protects shrinks from "the whole program" to "a library" to "a file" to "nothing." Weak copyleft is precisely the part of that gradient where reciprocity is preserved for the shared component but released for everything you build around it.

## Strengths and weaknesses

**Strengths.** Weak copyleft is the **business-friendly compromise**: it lets proprietary products use open components while still guaranteeing the components — and improvements to them — stay open. That makes these licenses far easier to adopt than the GPL, so the components see wide use, and the original projects still get changes flowed back. MPL and EPL's patent grants add real protection, and their file-level scope is easy to reason about.

**Weaknesses.** The cost is **boundary complexity**. You have to actually track where the copyleft line falls — which files are MPL, what counts as "the library," whether your linking satisfies the LGPL's relink requirement. Get the boundary wrong and you can fail to meet the obligation without realizing it. Compatibility surprises also lurk (the CDDL/GPL clash being the classic), so combining weak-copyleft code with other licenses still needs care. And static linking under the LGPL, in particular, carries enough conditions that some teams avoid it.

For most teams the rule of thumb is comforting: weak copyleft means **you can build a proprietary product on it — just keep the component itself open.** How that plays out when you ship is covered in [copyleft in products](/learn/software-licensing/copyleft-in-products/), and how to meet the lighter permissive duties is in [meeting permissive obligations](/learn/software-licensing/permissive-compliance/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — weak copyleft scopes the share-alike duty to the component (the library or the file), so proprietary code can use it as long as that component stays open." markdown="0">
  <p class="knowledge-check__q">Quick check: what makes a license "weak" copyleft rather than "strong"?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It places no obligations on anyone, exactly like the MIT license</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It removes the copyleft duty once the software is offered over a network</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It scopes the share-alike duty to the component (a library or a file) instead of the whole program</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Weak copyleft is the middle ground** — it preserves reciprocity for one component while letting surrounding proprietary code stay closed.
- **Scope is the key difference from the GPL** — strong copyleft reaches the whole program; weak copyleft stops at the library (LGPL) or the file (MPL/EPL/CDDL).
- **LGPL is library-scoped with a relink right** — proprietary code may link to it; dynamic linking satisfies the relink requirement easily, static linking adds conditions.
- **MPL 2.0 is file-scoped and well-liked** — modified MPL files stay open, new files can be proprietary, plus a patent grant and GPL compatibility.
- **EPL and CDDL are file-level cousins** — EPL adds a commercial-contributor indemnity; CDDL is famously GPL-incompatible (the ZFS-on-Linux problem).
- **Strengths vs weaknesses** — business-friendly and widely adopted, but you must track the copyleft boundary carefully and watch for compatibility surprises.

Next up: the other extreme — giving up rights entirely, and why "public domain" is trickier than it sounds. See [Public domain, Unlicense & CC0](/learn/software-licensing/public-domain-and-cc0/).
