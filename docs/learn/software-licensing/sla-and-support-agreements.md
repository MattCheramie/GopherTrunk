---
slug: sla-and-support-agreements
title: SLAs & support agreements
description: An SLA promises how available a service will be and what you get when it falls short; a support agreement defines response times, tiers, and coverage. Learn the "nines," service credits, severity levels, and how to read both from each side.
keywords: service level agreement, SLA, uptime, availability, the nines, service credits, maintenance window, support agreement, support tiers, response time, resolution time, severity levels, end of life, EOL, SLA exclusions
level: intermediate
status: full
faq:
  - q: "What does '99.9% uptime' actually mean in real downtime?"
    a: "It's the fraction of time the service must be available. **99.9%** (\"three nines\") allows about **43 minutes** of downtime per month; **99.99%** (\"four nines\") allows about **4.3 minutes**. Each extra nine cuts allowed downtime roughly tenfold — and gets dramatically harder and more expensive to deliver. Always check the *measurement window* and what counts as downtime, because exclusions can make a high number mean less than it looks."
  - q: "Are service credits real compensation if a service goes down?"
    a: "Rarely in proportion to your loss. A **service credit** is usually a small percentage of *your fee* for the affected period, capped low, and often **the sole remedy** — meaning you can't claim your actual business losses. They're a accountability signal more than insurance. If uptime is mission-critical, negotiate higher credits, an exit right for repeated breaches, or carry your own resilience and insurance."
  - q: "What's the difference between response time and resolution time in support?"
    a: "**Response time** is how fast support *acknowledges* and starts working on your ticket. **Resolution time** is how fast it's actually *fixed*. Most support agreements commit firmly to response times by severity and are far vaguer about resolution — because a fix can depend on factors outside the vendor's control. Read which one the SLA actually guarantees; the gap matters."
---
# SLAs & support agreements

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**An SLA promises availability** — the "nines" of uptime, how it's measured, and the exclusions. **Service credits are the usual remedy** — a small, capped refund, often the *only* recourse. **A support agreement defines help** — tiers, severity levels, response vs resolution times, and hours of coverage. **Maintenance and EOL matter too** — updates, patches, and when support ends. **Read both from both sides** — as a buyer, is the promise meaningful? As a seller, can you actually keep it?
</div>

When you buy or sell a service rather than a one-off copy of software, two agreements describe the *quality* of the ongoing relationship: the **Service Level Agreement (SLA)**, which commits to how available and performant the service will be, and the **support agreement**, which defines what help you get when something goes wrong. They often sit inside a larger [subscription or service contract](/learn/software-licensing/eula-and-tos/). This lesson explains the "nines" of uptime, why service credits are weaker than they sound, how support tiers and severity levels work, and — crucially — how the same document reads very differently depending on whether you're buying or selling. By the end you'll be able to size up an SLA on either side of the table.

## The SLA: a promise about availability

An **SLA** is a commitment to a measurable level of service — most commonly **availability**, expressed as a percentage of uptime over a period. The famous shorthand is the **"nines."** More nines means less allowed downtime, and each nine is roughly ten times harder to achieve.

| Availability | Nickname | Downtime / month | Downtime / year |
|---|---|---|---|
| 99% | "two nines" | ~7.3 hours | ~3.65 days |
| 99.9% | "three nines" | ~43 minutes | ~8.8 hours |
| 99.95% | — | ~22 minutes | ~4.4 hours |
| 99.99% | "four nines" | ~4.3 minutes | ~52 minutes |
| 99.999% | "five nines" | ~26 seconds | ~5.3 minutes |

The headline number is the easy part. The real meaning lives in three details:

- **Measurement window.** Is uptime measured monthly or annually? A monthly window resets the budget every month; an annual window lets one bad outage be averaged away. Buyers prefer monthly; sellers prefer annual.
- **What counts as "down."** Total outage only, or also degraded performance and partial failures? A service that's technically "up" but unusably slow may not breach a narrowly written SLA.
- **Exclusions.** Scheduled **maintenance windows**, force majeure, problems caused by the customer or by third-party networks, and beta features are commonly carved out and don't count against uptime. Generous exclusions can hollow out an impressive percentage.

### Service credits: the typical remedy, and their limits

When an SLA is breached, the standard remedy is a **service credit** — a partial refund applied to a future bill, scaled to how badly the target was missed (for example, 10% credit for falling below 99.9%, more for worse). Three limits make credits weaker than they first appear:

1. **They're small and capped.** Credits are a fraction of *your fee* for the affected period, with an overall cap (often a month's fees). They are not tied to *your* losses from the outage.
2. **They're often the "sole and exclusive remedy."** That phrase, common in SLAs, means the credit is *all you get* — you waive the right to claim your actual business damages.
3. **You usually have to claim them.** Credits frequently aren't automatic; you must request them within a window, or you forfeit them.

So a service credit is best understood as an **accountability signal** — skin in the game for the provider — rather than insurance against your loss. If availability is genuinely critical to you, the credit alone won't make you whole; you negotiate stronger terms (higher credits, a **termination right** if the SLA is missed repeatedly) and you build your own resilience.

## The support agreement: getting help

Where the SLA covers whether the service is *up*, the **support agreement** covers what happens when you need *help* — a bug, a question, an incident. Its main moving parts:

- **Support tiers / packages.** Vendors usually sell levels — say Basic, Business, Premium/Enterprise — that buy faster response, more channels, and longer hours.
- **Hours of coverage.** "Business hours" (e.g., 9–5 in one time zone) versus **24/7**. For a global or always-on system, the difference is enormous.
- **Channels.** Email/ticket only, versus chat, phone, or a named technical contact.
- **Response vs resolution time.** This distinction is central. **Response time** is how fast support *acknowledges and engages* your ticket; **resolution time** is how fast it's actually *fixed*. Agreements commit firmly to response, vaguely (if at all) to resolution — because fixing can depend on reproducing the issue, third parties, or a release cycle. Read which one is actually guaranteed.

### Severity levels

Support commitments are almost always **tiered by severity** — how badly the problem hurts you — so a production outage isn't treated like a cosmetic glitch. A typical scheme:

| Severity | Meaning | Typical target response |
|---|---|---|
| **Sev 1 / Critical** | Production down or unusable; no workaround | Minutes to ~1 hour, 24/7 |
| **Sev 2 / High** | Major function impaired; workaround painful | Hours, business day |
| **Sev 3 / Medium** | Minor issue or limited impact; workaround exists | ~1 business day |
| **Sev 4 / Low** | Cosmetic, question, or feature request | Best effort / next release |

Note that *the customer often files at a severity, but the vendor may reclassify it.* Check who decides severity and whether you can escalate — a vendor that quietly downgrades your "critical" to "medium" has changed the deal.

## Maintenance, updates, and end of life

Both agreements interact with the **lifecycle** of the software:

- **Updates and patches.** Who delivers them, how often, and whether security patches are guaranteed for a defined period. For self-hosted software this is a core part of the value.
- **Maintenance windows.** Planned downtime for upgrades — usually excluded from the SLA, but a good agreement bounds them (advance notice, off-peak timing, a monthly cap).
- **End of life (EOL).** When a version stops receiving support and security fixes. An EOL policy tells you how long you can safely run a given release before you *must* upgrade — vital for planning and for security (an unsupported version that stops getting patches becomes a liability, a theme in [the risks of depending on open source](/learn/software-licensing/open-source-risks/) and in [auditing dependencies](/learn/software-licensing/auditing-dependencies/)).

## Reading from both sides

The same SLA or support agreement should be read very differently depending on which seat you're in — this is the most useful habit in this lesson.

**As a buyer, ask: is the promise meaningful?**

- Does the uptime number, *after exclusions*, actually cover what I depend on?
- Is the credit worth anything next to what an outage costs me — and is it the *only* remedy?
- Are response times guaranteed for the severities I'll actually hit, in my time zone?
- What's my exit if they keep missing the targets?

**As a seller, ask: can I actually meet this?**

- Can my infrastructure realistically sustain the nines I'm promising, every measurement window — not on a good month?
- Have I scoped exclusions and maintenance windows so a routine upgrade doesn't breach my own SLA?
- Can my support team meet the response times I've committed to, at the hours I've promised, including 24/7 if I sold it?
- Are my credits capped and structured so a bad incident doesn't sink the contract — without being so stingy they look like bad faith?

The discipline cuts both ways: **don't buy a number you can't verify, and don't sell a number you can't keep.** An SLA you can't meet is worse than a humbler one you can, because chronic breach erodes trust and invites the exit clauses you wrote.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a service credit is usually a small, capped refund of your own fees, and frequently the sole remedy, so it rarely covers your real losses." markdown="0">
  <p class="knowledge-check__q">Quick check: a 99.9% SLA is breached and your business loses far more than a month of fees. What does the typical service-credit clause give you?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Full compensation for your actual business losses from the outage</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A small, capped credit against your fees — often the sole remedy, not your real losses</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">An automatic right to terminate immediately with a full refund</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **An SLA commits to availability** — the "nines" of uptime, but the meaning lives in the measurement window, what counts as down, and the exclusions.
- **Service credits are weak remedies** — small, capped, often the sole recourse and not automatic; treat them as accountability, not insurance.
- **Support agreements define help** — tiers, hours of coverage, channels, and the key gap between guaranteed *response* time and looser *resolution* time.
- **Severity levels prioritize** — and watch who gets to set or downgrade the severity of your ticket.
- **Lifecycle matters** — updates, bounded maintenance windows, and an EOL policy that tells you how long a version stays supported.
- **Read from both sides** — buyers ask whether the promise is meaningful; sellers ask whether they can actually keep it.

Next up: the confidentiality agreements and the rest of the B2B contract landscape, and how all these documents fit together. See [NDAs & the rest of the landscape](/learn/software-licensing/nda-and-other-agreements/).
