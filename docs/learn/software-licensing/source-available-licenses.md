---
slug: source-available-licenses
title: Source-available & "fair source"
description: The middle ground that is not open source — code you can read but can't freely use. Learn the Business Source License (BSL), SSPL, the Elastic License, the Fair Source movement, why companies adopt them, and why the OSI doesn't call them open source.
keywords: source available, fair source, Business Source License, BSL, SSPL, Server Side Public License, Elastic License, MongoDB, HashiCorp, open source rug pull, cloud provider monetization, copyleft, OSI, license restrictions
level: intermediate
status: full
faq:
  - q: "Why isn't source-available code considered open source?"
    a: "Because it fails the [Open Source Definition](/learn/software-licensing/what-is-open-source/). Open source requires that *anyone* may use, modify, and redistribute the code *for any purpose*, including commercially and including competing with the vendor. Source-available licenses keep the code visible but restrict exactly those things — most often a ban on production use, competing use, or offering it as a service — so the OSI doesn't recognize them as open source."
  - q: "What is the Business Source License (BSL) and does it ever become open source?"
    a: "The **BSL** lets you read, modify, and use the code with one main restriction — typically no production or competing commercial use — and then **automatically converts** to a true open-source license (often Apache 2.0 or GPL) after a set delay, commonly four years. So today's release is source-available; the same version becomes open source on a timer. It's a compromise between monetizing now and committing to openness later."
  - q: "Why did several big projects relicense away from open source?"
    a: "Mostly because **cloud providers were taking the open-source software, running it as a paid managed service, and capturing the revenue** without contributing much back. Companies like MongoDB, Elastic, and HashiCorp responded by moving to source-available licenses (SSPL, the Elastic License, BSL) that specifically restrict offering the software as a competing service. This is the commercial flip side of the dependency \"rug pull\" risk covered in [The risks of depending on open source](/learn/software-licensing/open-source-risks/)."
---
# Source-available & "fair source"

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Source-available ≠ open source** — the code is visible, but the license restricts use, so it fails the Open Source Definition. **The main licenses** — BSL (converts to open source on a timer), SSPL (copyleft aimed at cloud providers), the Elastic License. **"Fair source"** — a newer label for source-available-with-eventual-openness. **Why companies move here** — to stop cloud providers from monetizing their work without giving back.
</div>

Between fully open source and fully proprietary lies a fast-growing middle ground: software whose source code is published and readable, but whose license withholds some of the freedoms that would make it open source. This is **source-available** code, and over the last few years several major projects have moved into it — a shift worth understanding whether you're choosing a dependency or a license. By the end of this lesson you'll know the major source-available licenses, what each one restricts, what "fair source" means, why companies adopt these terms, and why the Open Source Initiative declines to call them open source.

## The middle of the spectrum

Recall the [licensing spectrum](/learn/software-licensing/the-licensing-spectrum/): permissive open source on one end, proprietary closed-source on the other. Source-available sits squarely in the middle. You get something proprietary software almost never gives you — the actual source code, to read, audit, and often to modify and self-host — but you *don't* get the full bundle of open-source freedoms. There's a string attached, usually one of:

- **No competing use** — you can't offer the software as a service that competes with the vendor.
- **No production use** (for a time) — you can use it for development, testing, and evaluation, but not in production, until a delay passes.
- **No redistribution as a service** — you can't run it as a managed offering for others.

The mental test from [What open source really means](/learn/software-licensing/what-is-open-source/) applies cleanly: *can anyone use this for any purpose, including commercially and including competing with the author?* If the answer is "no, not for X," it's source-available, not open source — no matter how much code is on display.

## The Business Source License (BSL)

The **Business Source License (BSL)**, originally from MariaDB and adopted by HashiCorp, CockroachDB, Sentry, and others, is the most influential source-available license because of one clever twist: it has an **expiration date**.

Under the BSL you may freely read, copy, modify, and use the code, subject to a single **Additional Use Grant** restriction the vendor specifies — almost always *no production or competing commercial use*. Then, after a **Change Date** (commonly four years after release), each version **automatically converts** to a stated open-source license — typically Apache 2.0 or GPL.

```text
Business Source License — typical shape

  Additional Use Grant: You may use the Licensed Work in production,
  except to offer a commercial service that competes with the Licensor.

  Change Date:    Four years from the date of each release.
  Change License: Apache License, Version 2.0.
```

So a given version is source-available today and becomes genuinely open source on a timer. The BSL is best understood as a compromise: monetize the newest releases now, while guaranteeing the community gets full openness eventually. Note that "BSL" is a template — the real restriction depends on the Additional Use Grant the vendor fills in, so always read *that* line.

## SSPL — copyleft turned against cloud providers

The **Server Side Public License (SSPL)** was created by **MongoDB** in 2018. It looks like an aggressive form of [copyleft](/learn/software-licensing/permissive-vs-copyleft/): it builds on the AGPL idea that offering software over a network triggers source-sharing, then goes much further.

The AGPL says that if you modify the software and offer it as a network service, you must share *your modifications*. The SSPL says that if you offer the software *as a service*, you must release the source code of **the entire service stack used to provide it** — the management software, automation, monitoring, the works. In practice that demand is so sweeping that no cloud provider will comply, which is precisely the point: the SSPL is designed to make it commercially impossible to offer MongoDB-as-a-service without a commercial license. It's copyleft weaponized against a specific business behavior.

The OSI explicitly declined to approve the SSPL as open source. We compare it directly to the AGPL in [The AGPL: copyleft over the network](/learn/software-licensing/agpl-network-copyleft/).

## The Elastic License and friends

When Elastic relicensed Elasticsearch and Kibana in 2021 (it offered both the SSPL and its own license), it popularized the **Elastic License v2 (ELv2)** — a short, plain-language source-available license that grants broad rights with three carve-outs: you can't provide the software as a hosted/managed service, you can't circumvent license-key functionality, and you can't remove licensing notices. Other vendors have shipped similar bespoke "you may do anything except compete with us as a service" licenses (Confluent, Redis, and others have used variants of this idea).

The common thread across BSL, SSPL, and ELv2 is the **anti-cloud-provider** clause: read it, run it, modify it, self-host it — just don't resell it as a competing managed service.

## The "Fair Source" movement

**Fair Source** is a newer umbrella label (with its own site, fair.io) for software that is publicly readable, allows use, modification, and redistribution with *minimal* restrictions to protect the maker's business, and — crucially — includes a **delayed open-source** provision so the code eventually becomes truly open. The BSL is the canonical Fair Source license. The movement is an attempt to give this middle ground a positive, honest name rather than letting it be confused with open source. The important word is **honest**: Fair Source advocates are clear that it is *not* open source, which is more than can be said for some vendors who blur the line.

## Why companies move here — and the "rug pull"

Why would a company that built its name on open source relicense away from it? The dominant reason is **cloud providers monetizing OSS**. A hyperscaler can take a popular open-source database, wrap it in a slick managed service, and earn enormous revenue from it — while contributing little back to the project that made it possible. The original company foots the development bill; the cloud provider captures the margin.

Source-available licenses are the counter-move: keep the code open enough for users and self-hosters, but block competitors from running it as a service. From the vendor's side it's survival. From a *user's* side, a relicensing can feel like a **rug pull** — you adopted something as open source, built on it, and then the terms changed under you. That's the commercial face of the dependency risk we cover in [The risks of depending on open source](/learn/software-licensing/open-source-risks/): the license you depend on is only as stable as the business behind it (and as the contributor agreements that let them relicense — see [Contributor agreements](/learn/software-licensing/contributor-agreements/)).

## Why the OSI says these aren't open source

| License | Source visible? | Main restriction | Becomes open source? | OSI-approved? |
|---|---|---|---|---|
| **BSL** | Yes | No production/competing use (per Additional Use Grant) | Yes — after the Change Date | No |
| **SSPL** | Yes | Must open-source your *entire* service stack to offer it as a service | No | No |
| **Elastic License v2** | Yes | No hosted/managed-service offering; no key circumvention | No | No |
| **A true OSS license (e.g. Apache 2.0)** | Yes | None on field of use | n/a — already open | Yes |

The reason all the source-available licenses fail the [Open Source Definition](/learn/software-licensing/what-is-open-source/) is the same: they **discriminate against a field of endeavor** (offering a competing service) or otherwise restrict use. The OSD forbids that flatly — no "free for everything except commercial hosting." So however generous these licenses are, and however good their reasons, they are not open source, and the OSI is firm about not letting the term be stretched to cover them. Calling them "source-available" or "fair source" keeps the vocabulary honest.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the BSL restricts use now but automatically converts each version to a true open-source license after the Change Date." markdown="0">
  <p class="knowledge-check__q">Quick check: which source-available license is designed to automatically become open source after a set delay?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The SSPL (Server Side Public License)</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The Business Source License (BSL)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The Elastic License v2</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Source-available is the middle ground** — the code is visible and often modifiable and self-hostable, but the license restricts use, so it is *not* open source.
- **BSL** — usable now with a no-production/no-competing restriction, automatically converting to a true open-source license after a Change Date (often four years).
- **SSPL** — copyleft taken to an extreme, demanding the source of your entire service stack, designed to block cloud-provider services; OSI rejected it.
- **Elastic License v2 and friends** — short licenses granting broad rights minus a managed-service carve-out.
- **Fair Source** — an honest umbrella label for source-available code with a delayed-openness provision; explicitly not open source.
- **Why companies move here** — to stop cloud providers from monetizing their OSS without giving back; for users it can feel like a rug pull.
- **Why the OSI says no** — these licenses discriminate against a field of use, which the Open Source Definition forbids.

Next up: from analyzing other people's licenses to drafting your own — what to decide before you write a license to sell software. See [Writing a license to sell software](/learn/software-licensing/writing-a-commercial-license/).
