---
slug: copyright-and-software-ownership
title: Copyright & software ownership
description: Copyright attaches to code automatically the moment it's written. Learn what it protects, the difference between authorship and ownership, and why contractors keep their copyright unless it's assigned in writing.
keywords: software copyright, copyright ownership, automatic copyright, copyright registration, work made for hire, contractor copyright assignment, joint authorship, expression vs ideas, who owns code, source code copyright
level: beginner
status: full
faq:
  - q: "Do I need to register my software copyright for it to count?"
    a: "No. Copyright attaches **automatically** on creation in essentially every country, with no registration required. In the **US specifically**, registering with the Copyright Office is optional but valuable — it's a prerequisite to suing and unlocks statutory damages and attorney's fees. Most other countries have no registration system at all, so this is a US-specific tactic, not a global requirement."
  - q: "If I pay a freelancer to build my app, do I own the code?"
    a: "Not automatically. By default, an independent **contractor keeps the copyright** in what they create, and you only get whatever license the contract grants. To actually *own* the code you need a written **copyright assignment** in the contract. Skipping this is one of the most common and expensive mistakes in software — fix it before work starts, in writing."
  - q: "Can someone copyright an algorithm or an idea?"
    a: "No. Copyright protects the specific **expression** — your actual source and binary — not the underlying **idea, algorithm, or functionality**. Someone can study what your program does and write their own independent implementation without infringing your copyright. Protecting the idea itself, if it's even possible, is the realm of patents, which we cover in the [next lesson](/learn/software-licensing/other-ip-in-software/)."
---
# Copyright & software ownership

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Copyright is automatic** — it attaches the instant original code is written, with no registration needed. **It protects expression, not ideas** — your source and binary are covered; the underlying algorithm and functionality are not. **Authorship and ownership can differ** — who wrote it isn't always who owns it. **Get assignments in writing** — employees' work is the employer's by default in the US, but contractors keep their copyright unless they assign it.
</div>

In the [first lesson](/learn/software-licensing/what-is-a-software-license/) we said code is owned the moment it's written and that a license grants permission from that owner. This lesson unpacks the "owned" half: where copyright comes from, what it does and doesn't cover, and — crucially — *who* the owner actually is, which is less obvious than it sounds. By the end you'll know why registration is optional, what copyright protects, and the written-assignment rule that trips up so many people who hire others to write code.

## Copyright attaches automatically

The foundational rule from lesson one bears repeating because so much follows from it: **copyright protection is automatic.** The instant you create an original work and fix it in a tangible medium — and saving a source file counts — copyright exists. You don't apply for it, you don't pay for it, and you don't need to add a `©` notice (though a notice is still a useful signal to others).

This isn't a quirk of one country. An international treaty, the **Berne Convention**, commits its many member states to automatic, no-formalities copyright, which is why the rule looks the same across most of the world.

### Registration: useful in the US, mostly absent elsewhere

If copyright is automatic, what is "registering a copyright"? This is where countries diverge, so be careful.

In the **United States**, registration with the Copyright Office is *optional* but carries real teeth. You generally must register before you can file an infringement lawsuit, and timely registration unlocks **statutory damages** (a fixed range of damages you can claim without proving exact financial harm) plus attorney's fees. That combination makes US registration a serious deterrent and a practical prerequisite to enforcement.

Here's the part people miss: **this is largely US-specific.** Many countries have no copyright registration system at all — your rights are simply automatic and that's the end of it. So "you should register your copyright" is good advice in the US and meaningless in much of the rest of the world. The [cross-border lesson](/learn/software-licensing/licensing-across-jurisdictions/) returns to this.

| | Automatic protection | Registration | Why register (where it exists) |
|---|---|---|---|
| **United States** | Yes, on creation | Optional, via Copyright Office | Required to sue; unlocks statutory damages + attorney's fees |
| **Most of the world** | Yes, on creation (Berne) | Often no system at all | N/A — rights are automatic |

## What copyright protects — and what it doesn't

Copyright is powerful but narrow. It protects **expression**, not **ideas**. For software, the practical line looks like this:

**Protected (expression):**

- Your **source code** — the specific way you wrote it.
- Your **compiled binary** — copying it bit-for-bit is copying the expression.
- Other creative expression bundled with the software, like documentation, UI art, and original text.

**Not protected by copyright (ideas and function):**

- The **idea** behind the program.
- The **algorithm** or method it uses.
- The **functionality** — what the program does.
- Facts and data themselves.

This distinction has big consequences. A competitor is free to look at what your program does, understand the idea, and write their *own independent* implementation in their own code. As long as they didn't copy your expression, they haven't infringed your copyright — even if their product competes directly with yours. If you want to protect the underlying invention itself, that's the job of **patents**, not copyright, and it's a much harder and more limited road (covered in [the next lesson](/learn/software-licensing/other-ip-in-software/)).

## Authorship is not the same as ownership

It's natural to assume the person who *wrote* the code owns it. Often they do — but not always. The law distinguishes two ideas:

- **Authorship** — who actually created the work.
- **Ownership** — who holds the copyright (the exclusive rights).

These usually start out together: the author is the first owner. But ownership can shift — by operation of law, by employment status, or by a written transfer — so that the author and the owner are different people or entities. The next two sections are the most important cases of exactly that.

## Work made for hire: employees' code belongs to the employer

In the **US**, code an employee writes *within the scope of their employment* is a **work made for hire**, and the **employer is the author and owner** from the start — not the employee. The developer who typed it never held the copyright in the first place.

This is why your day-job code belongs to your company, and it's why companies can confidently build and ship products without getting an assignment from each engineer for every commit. The employment relationship does the work automatically (for code created in the scope of the job — a genuinely personal side project on your own time and equipment is a different, sometimes contested, question).

A jurisdiction flag: the *result* — employer owns employee work — is common across many countries, but the legal machinery and the precise boundaries differ, and some systems (especially in Europe) keep certain **moral rights** with the human author regardless. More on that in [Licensing across jurisdictions](/learn/software-licensing/licensing-across-jurisdictions/).

## Contractors keep their copyright unless they assign it

Now the rule that costs people real money. An **independent contractor** is *not* an employee, so the work-made-for-hire default does **not** automatically apply. By default, **the contractor owns the copyright in what they create**, and the client who paid for it gets only whatever rights the contract grants — sometimes just an implied license to use the deliverable.

Read that again, because it's counterintuitive: *you can pay someone to build your software and not own it.* The money changing hands does not transfer the copyright. Only a written transfer does.

The fix is a **copyright assignment** — an explicit clause in the contract, in writing, saying the contractor assigns all copyright in the work to the client. With it, you own the code. Without it, you're relying on a license you may not even have nailed down, and you can be blocked from things as basic as relicensing, selling the business, or stopping the contractor from reusing your code elsewhere.

```text
Contractor hereby assigns to Client all right, title, and interest,
including all copyright, in and to the Work Product.
```

A single sentence like that, signed before work begins, is the difference between owning your product and merely being allowed to use it. Putting "make it a work made for hire" in a contract is not always enough on its own — for many works the law won't treat contractor output as work-for-hire even if you say so — which is why a belt-and-suspenders **assignment** clause is the standard, safe move. (This is a great example of where a real situation warrants a lawyer.)

## Joint authorship: when two people own one work

What if two developers genuinely co-create a single, merged work, each contributing copyrightable expression with the intent to combine it? They may be **joint authors**, co-owning the copyright together.

Joint ownership has surprising defaults, and they vary by country. In the **US**, each joint owner can generally license the *whole* work to others without the other's permission — but typically must **account** to the co-owner for their share of any profits. In some other countries, each co-owner can *block* a license, requiring unanimous consent. Either way, ambiguous joint ownership is a recipe for disputes, which is why serious projects with multiple contributors use [contributor agreements](/learn/software-licensing/contributor-agreements/) to make ownership and licensing explicit rather than leaving it to these defaults.

<div class="knowledge-check" data-quiz data-correct-msg="Right — without a written copyright assignment, an independent contractor keeps the copyright, and paying them doesn't transfer ownership." markdown="0">
  <p class="knowledge-check__q">Quick check: you hire an independent freelancer to write your app and the contract says nothing about copyright. Who owns the code?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">The contractor — they keep the copyright unless they assign it to you in writing</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">You — paying for the work automatically transfers ownership to the client</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Nobody — unregistered code has no owner until it's filed with the Copyright Office</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Copyright is automatic** — it attaches on creation everywhere under the Berne Convention, with no registration required.
- **Registration is a US-specific tactic** — optional but it's a prerequisite to suing and unlocks statutory damages; most countries have no registration system at all.
- **It protects expression, not ideas** — your source and binary are covered; the algorithm, functionality, and underlying idea are not (that's patent territory).
- **Authorship ≠ ownership** — the person who wrote the code isn't always the one who holds the copyright.
- **Employees' work is the employer's** — in the US, in-scope employee code is a work made for hire owned by the employer from the start.
- **Contractors keep their copyright** — you only own a freelancer's work if they assign it to you in writing; this is a common, expensive mistake to avoid.

Next up: copyright is only one of four kinds of intellectual property around software. See [Patents, trademarks & trade secrets](/learn/software-licensing/other-ip-in-software/).
