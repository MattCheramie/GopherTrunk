---
slug: agpl-network-copyleft
title: "The AGPL: copyleft over the network"
description: The GNU Affero GPL closes the "SaaS loophole" — it adds a trigger so that users who interact with the software over a network must be offered its complete source code, even when no binary is ever distributed.
keywords: AGPL, Affero GPL, GNU AGPL, SaaS loophole, network copyleft, section 13, copyleft over network, AGPL banned, hosted software, cloud provider protection, SSPL, source-available, GPL distribution gap
level: intermediate
status: full
faq:
  - q: "What exactly is the \"SaaS loophole\" the AGPL closes?"
    a: "The plain [GPL](/learn/software-licensing/gpl-strong-copyleft/) only triggers its source-sharing duty on **distribution**. A software-as-a-service company never distributes the binary — users just talk to it over the network — so the GPL's source obligation never fires and users never get the source. The AGPL adds a trigger for **network interaction**, so remote users must be offered the source too."
  - q: "Why do companies like Google ban AGPL dependencies?"
    a: "Because the network trigger is broad and the boundary of what counts as a combined work is uncertain, a company worries that linking AGPL code into an internal service could obligate it to publish source for proprietary systems. Rather than analyze every case, large firms (Google being the well-known example) often **ban AGPL outright** in their codebases as a blanket risk-avoidance policy."
  - q: "Is the AGPL the same as a source-available license like the SSPL?"
    a: "No. The AGPL is a true open-source license approved by the OSI; it just extends copyleft to network use. Licenses like the **SSPL** go further and impose conditions that the OSI considers to cross the line out of open source. The AGPL protects user freedom; the SSPL is mainly aimed at stopping cloud providers from reselling the software — see [source-available licenses](/learn/software-licensing/source-available-licenses/)."
---
# The AGPL: copyleft over the network

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**It closes the SaaS loophole** — plain GPL triggers on distribution, but SaaS never distributes a binary, so users never see source; the AGPL fixes that. **Section 13 is the heart** — it adds a trigger on *interacting with the software over a network*. **Remote users get an offer of source** — if people use your modified AGPL service, you must make its complete source available to them. **Powerful but feared** — it protects against cloud capture, which is exactly why many companies ban it.
</div>

The [GPL lesson](/learn/software-licensing/gpl-strong-copyleft/) ended on a gap: the GPL's obligations only fire on *distribution*, and a network service never distributes anything. The GNU Affero General Public License — **AGPL** — exists to close that gap. By the end of this lesson you'll understand the "SaaS loophole" precisely, what section 13 actually requires, why this matters so much in the cloud era, why a number of large companies ban AGPL code, where it's genuinely the right choice, and how it relates to the more aggressive source-available licenses that came after it.

## The SaaS loophole

Recall the GPL bargain: you get the four freedoms, and in exchange you must pass them on **when you distribute** the software. That worked beautifully when software was something you shipped — a binary on a disk, a download, a device. The duty to provide source was tied to the moment a copy changed hands.

Then the world moved to **software as a service (SaaS)**. Instead of shipping you a program, a company runs the program on its own servers and lets you interact with it over the network — your browser, an API call, a mobile app talking to a backend. You use the software fully, but **no copy is ever handed to you**.

Look at what that does to the GPL. The trigger is distribution. SaaS never distributes the binary. So a company can take a GPL program, modify it heavily, run it as a public service used by millions, and owe **no one any source code at all**. The users get all the benefit of the software and none of the freedoms the GPL was written to protect. This is the **SaaS loophole** (sometimes the "ASP loophole," from the older term "application service provider").

It isn't a bug in the GPL so much as a consequence of how it was scoped. But to the people who care most about user freedom, it was a hole big enough to drive the entire cloud industry through.

## Section 13: the network trigger

The AGPL is, line for line, almost identical to GPLv3 — it grants the same freedoms and imposes the same distribution obligations. The difference is one added clause: **section 13**, the "Remote Network Interaction" provision.

Section 13 adds a second trigger alongside distribution. In plain terms: **if you modify the program and let users interact with it remotely over a network, you must give those remote users access to the complete corresponding source** of your modified version. The license requires that the software offer remote users a way to obtain that source — typically a prominent link in the interface to download it.

So the AGPL has *two* ways its source obligation can fire:

| Trigger | Plain GPL | AGPL |
|---|---|---|
| You distribute the binary | Yes | Yes |
| Users interact with it over a network (no binary handed over) | **No** | **Yes (section 13)** |

The same scoping rules carry over from the GPL: **internal use still doesn't count**. Running AGPL software for yourself, or modifying it on a machine no one else can reach, triggers nothing. The duty arises only when *other people* interact with your modified version over the network. And as before, the obligation is to provide **complete corresponding source** — everything needed to build and run the version you're operating — under the AGPL.

```text
Plain GPL:  source duty fires only when a copy is handed over.
AGPL:       source duty also fires when remote users use the running service.
            -> a "Download source" link in the UI is the usual way to comply.
```

## Why this matters for hosted software

The AGPL flips the economics for hosted software. Under the GPL, a SaaS provider can keep its modifications secret forever. Under the AGPL, the moment it lets the public use a modified version, it must share those modifications back. That's a strong guarantee that improvements to the software remain available to the community, even when the software is never "shipped" in the traditional sense.

For an open-source project that expects to be **run as a service** — a database, a monitoring tool, a collaboration platform — the AGPL is the only mainstream license that preserves copyleft's reciprocity in that delivery model. Without it, the most common way the software gets used (hosted by someone else) would be the one way copyleft didn't reach.

## Why many companies ban the AGPL

Despite being a legitimate, OSI-approved open-source license, the AGPL is **banned outright in the codebases of many large companies** — Google's published open-source policy forbidding it is the famous example, and plenty of others quietly follow suit.

The reason is risk, not ideology. Two worries drive it:

- **Breadth of the trigger.** "Interacting over a network" is broad, and the boundary of what counts as part of the combined work — much like the [linking question for the GPL](/learn/software-licensing/gpl-strong-copyleft/) — is not crisply settled in case law. A company fears that pulling an AGPL library into one internal service could, in a worst-case reading, obligate it to publish source for adjacent proprietary systems.
- **Cost of analysis.** Even if most uses would be fine, evaluating each one is expensive. A blanket ban is cheaper than per-case legal review across thousands of engineers.

So the ban is a coarse but rational policy: it trades away access to some good AGPL software in exchange for never having to reason about the edge cases. The practical upshot for *you*: if you license your project AGPL, you should expect that a meaningful slice of corporate users won't touch it. That's a deliberate trade-off, covered more in [the risks of depending on open source](/learn/software-licensing/open-source-risks/) and in [choosing an OSS license](/learn/software-licensing/choosing-an-oss-license/).

## Legitimate uses: protecting against cloud capture

The AGPL isn't only a thorn — it's a genuinely good fit for some goals, especially **defending a project against cloud providers**.

Imagine you build an open-source database and a large cloud company offers it as a managed service, makes valuable proprietary improvements, captures most of the revenue, and contributes nothing back. Under permissive or even GPL terms, there's little you can do. Under the **AGPL**, that provider must publish its server-side modifications under the AGPL — so its enhancements flow back to you and everyone else. The license turns "host it and keep the improvements secret" into "host it and share the improvements."

This is why a number of infrastructure projects adopted the AGPL: it keeps the reciprocity promise alive against exactly the actor most able to exploit the SaaS loophole. When your worry is *cloud capture*, the AGPL is the strongest mainstream open-source tool for the job.

## Strengths and weaknesses

**Strengths.** The AGPL preserves copyleft's core promise in the cloud era, where the GPL alone falls short. It's a real open-source license, so the software stays free and community improvements keep coming. And it's a credible deterrent against proprietary cloud capture, protecting both the project and its users' freedoms.

**Weaknesses.** Its breadth scares corporate users and triggers outright bans, shrinking adoption. The section 13 boundary is legally less-tested than one would like, so even well-intentioned users face uncertainty. And operational compliance — making sure your running service actually offers source to users — is an ongoing obligation, not a one-time packaging step.

A final connection: some projects decided even the AGPL didn't go far enough to stop cloud resale, and moved to **source-available** licenses such as the **SSPL** that impose conditions the open-source community rejects as crossing the line out of open source. That's a different category — not open source at all — and we treat it in [source-available and "fair source"](/learn/software-licensing/source-available-licenses/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — the AGPL's section 13 adds a trigger for network interaction, so remote users of a modified service must be offered its source." markdown="0">
  <p class="knowledge-check__q">Quick check: what does the AGPL add that the plain GPL lacks?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">A trigger requiring you to offer source to users who interact with the software over a network</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Permission to relicense the software as proprietary after five years</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A clause that removes all source-sharing obligations for SaaS providers</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **The SaaS loophole is the problem** — plain GPL triggers only on distribution, but SaaS never distributes a binary, so users never get the source.
- **Section 13 is the fix** — the AGPL adds a trigger for interacting with the software over a network, requiring that remote users be offered the complete corresponding source.
- **Internal use is still safe** — like the GPL, the AGPL only fires when others use your modified version; running it for yourself triggers nothing.
- **Many companies ban it** — the broad trigger and uncertain boundary lead firms like Google to forbid AGPL dependencies as blanket risk-avoidance, so expect reduced corporate adoption.
- **It's the strongest defense against cloud capture** — when your concern is a provider hosting your project and keeping improvements secret, the AGPL forces those improvements back into the open.
- **It's still real open source** — unlike source-available licenses such as the SSPL, the AGPL is OSI-approved; it extends copyleft without leaving the open-source category.

Next up: the middle ground between permissive and strong copyleft — licenses that keep one component open while letting proprietary code link to it. See [Weak copyleft: LGPL, MPL, EPL](/learn/software-licensing/weak-copyleft-licenses/).
