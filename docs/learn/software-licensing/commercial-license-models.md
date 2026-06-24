---
slug: commercial-license-models
title: Commercial license models
description: How paid software is structured — perpetual licenses with maintenance vs subscriptions, per-seat vs concurrent, per-core, usage-based, site licenses, freemium, open-core, and OEM. A clear comparison of who each model fits and where the traps are.
keywords: commercial license models, perpetual license, subscription, SaaS, per-seat license, named user, concurrent license, per-core licensing, usage-based pricing, site license, enterprise license, freemium, open core, OEM license, software pricing
level: intermediate
status: full
faq:
  - q: "What's the difference between a perpetual license and a subscription?"
    a: "A **perpetual** license is bought once and lets you use that version forever, usually with an optional yearly **maintenance** fee for updates and support. A **subscription** is a recurring payment (monthly or yearly) that grants the right to use the software *only while you keep paying* — stop paying and access ends. Perpetual front-loads the cost; subscription spreads it out but never finishes."
  - q: "What's the difference between per-seat and concurrent licensing?"
    a: "**Per-seat** (or named-user) ties a license to specific people or devices — if you have 50 named users, only those 50 can use it. **Concurrent** licensing caps how many people use it *at the same time* — 50 concurrent licenses can serve 200 occasional users as long as no more than 50 are active at once. Concurrent suits tools used intermittently by many people; per-seat suits daily, dedicated users."
  - q: "Is open-core the same as open source?"
    a: "No. **Open-core** means a company publishes a genuinely open-source core and sells proprietary add-ons (advanced features, support, hosting) on top. The core is open source; the paid layer is not. It's a business model built *around* open source, not an open-source license. See [source-available licenses](/learn/software-licensing/source-available-licenses/) for the related but distinct strategies vendors use."
---
# Commercial license models

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Perpetual vs subscription** — buy-once-plus-maintenance versus pay-to-keep-using. **How you're counted** — per-seat/named-user, concurrent, per-core/CPU/instance, or by usage. **Scope tiers** — site, enterprise, and unlimited deals trade per-unit price for coverage. **Hybrid models** — freemium and open-core layer free and paid; OEM/redistribution licenses let others embed your software.
</div>

Proprietary software has to be paid for somehow, and over the years the industry has settled into a handful of well-worn structures for charging. Understanding them helps whether you're buying software (so you don't overpay or get trapped) or selling it (so you pick a model that fits your product and customers). By the end of this lesson you'll be able to tell perpetual from subscription, understand the main ways vendors count usage, recognize the scope-based and hybrid models, and spot the common pitfalls of each.

## Perpetual licenses vs subscriptions

The most fundamental split is *how long your right to use lasts*.

A **perpetual license** is a one-time purchase that grants the right to use a specific version of the software forever. Because software needs fixing and updating, perpetual licenses are almost always paired with an optional **maintenance** (or "software assurance") contract — a yearly fee, often 15–25% of the purchase price, that buys you bug fixes, updates, and support. Let the maintenance lapse and you keep the software you have, but you stop getting updates.

A **subscription** flips the model: you pay a recurring fee (monthly or annual), and your right to use the software lasts *only as long as you keep paying*. Stop, and access ends. Most cloud-delivered software (**SaaS** — Software as a Service) works this way, because the vendor is also hosting and running the software for you.

The industry has shifted hard toward subscriptions over the last decade. Vendors prefer the predictable recurring revenue; buyers get lower upfront cost and continuous updates but lose the "I paid once and own this version" finality.

| Model | How you pay | You stop paying… | Watch out for |
|---|---|---|---|
| **Perpetual + maintenance** | Large one-time fee, then optional yearly maintenance | …you keep the version you have, lose updates | Maintenance creep; being stranded on an old version |
| **Subscription / SaaS** | Recurring monthly or yearly | …access ends entirely | Lock-in; price hikes; no "owned" fallback |

## How vendors count you

Within either model, the vendor needs a *unit* to charge per. The unit shapes the whole deal.

### Per-seat and named-user

The most familiar unit is the **seat**. A **per-seat** or **named-user** license is assigned to a specific person (or sometimes a specific device). Twenty named users means twenty people may use it; a twenty-first needs another license. This is simple and predictable, and it fits software people use every day as part of their job.

### Concurrent (floating)

A **concurrent** or **floating** license caps the number of *simultaneous* users rather than total users. If you buy ten concurrent licenses, any of your hundred employees can use the software, but only ten at a time; the eleventh waits for a seat to free up. This is efficient for tools used intermittently by a large, rotating group — think of a specialized analysis app that each engineer needs only occasionally.

### Per-core, per-CPU, per-instance

For server and infrastructure software, charging per human makes no sense — the software serves machines, not individuals. These products license by **hardware or deployment unit**: per CPU socket, per **core**, per virtual machine, or per running **instance**. Databases and middleware are classic examples. The pitfall here is that modern multi-core CPUs and virtualized/cloud environments can make the count explode — a single beefy server can carry dozens of billable cores, and "do you count physical or virtual cores?" has been the source of many painful audit disputes.

### Usage / consumption-based

The newest common model charges for **what you actually use**: API calls, data processed, messages sent, compute-hours, gigabytes stored. It scales naturally with value and lets small users pay little — but it can be unpredictable, and a runaway job or a traffic spike can produce a shocking bill. Metered pricing is the default for most cloud platform services.

## Scope-based licensing: site, enterprise, unlimited

Counting per unit gets expensive and tedious at scale, so for large customers vendors offer **scope** deals that drop the per-unit math:

- **Site license** — covers everyone at a particular location or facility, no per-seat counting.
- **Enterprise license / Enterprise License Agreement (ELA)** — covers an entire organization, often across locations, usually as a negotiated lump sum.
- **Unlimited license** — use as much as you want, typically the most expensive and most heavily negotiated.

These trade a higher total price for simplicity and predictability. The pitfall is over-buying: an "unlimited" deal sold on projected growth that never materializes can cost far more than per-seat would have. Read the renewal terms — the easy first-year price sometimes resets sharply later.

## Hybrid and channel models

Several models mix free and paid, or license software for *others* to build on.

- **Freemium** — a free tier plus paid upgrades. The free tier is real software (still proprietary — see [Proprietary & closed-source licensing](/learn/software-licensing/proprietary-licensing/)) used as a funnel; you pay to unlock capacity or features.
- **Open-core** — a genuinely [open-source](/learn/software-licensing/what-is-open-source/) core with proprietary paid add-ons, support, or hosting on top. The core is open source; the surrounding layer is commercial. This is a business model around open source, not a license type.
- **OEM / redistribution / embedding** — licenses that let *another company* bundle your software inside their product (an "Original Equipment Manufacturer" deal), redistribute it to their customers, or embed it in a device. These are negotiated separately and priced by volume or royalty, because the licensee is reselling your work, not just using it.

## Putting it together

| Model | How you pay | Fits | Main pitfall |
|---|---|---|---|
| **Perpetual + maintenance** | Once, plus yearly support | Buyers who want to own a version | Stranded on old versions if maintenance lapses |
| **Subscription / SaaS** | Recurring | Continuously updated, hosted tools | Lock-in and price increases |
| **Per-seat / named-user** | Per person/device | Daily, dedicated users | Paying for idle seats |
| **Concurrent / floating** | Per simultaneous user | Intermittent use by many people | Bottlenecks at peak demand |
| **Per-core / CPU / instance** | Per hardware/deployment unit | Server & infrastructure software | Core/VM counts exploding in the cloud |
| **Usage / consumption** | Per call, GB, hour, etc. | Variable workloads, platform services | Unpredictable, spiky bills |
| **Site / enterprise / unlimited** | Negotiated lump sum | Large organizations | Over-buying; renewal price resets |
| **Freemium / open-core** | Free tier + paid upgrades | Bottom-up adoption | Free tier cannibalizing paid |
| **OEM / redistribution** | Volume or royalty | Embedding in another product | Mispriced royalties at scale |

No model is "best." The right one depends on the product, the buyer, and how value scales with use — which is exactly the kind of decision we work through in [A framework for choosing](/learn/software-licensing/the-decision-framework/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — concurrent licensing caps simultaneous users, so a large rotating pool can share a small number of seats." markdown="0">
  <p class="knowledge-check__q">Quick check: a tool is used briefly and occasionally by 200 employees, rarely more than 8 at once. Which model is most cost-efficient?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Per-seat / named-user licensing for all 200 people</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A per-core license sized to the server it runs on</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A handful of concurrent (floating) licenses sized to peak simultaneous use</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Perpetual vs subscription** — own a version forever (with optional maintenance) versus pay-to-keep-using; the industry has shifted toward subscriptions and SaaS.
- **Counting units** — per-seat/named-user (per person), concurrent (per simultaneous user), per-core/CPU/instance (per machine), and usage-based (per call/GB/hour).
- **Scope tiers** — site, enterprise, and unlimited licenses trade per-unit price for coverage and simplicity, with over-buying as the risk.
- **Hybrid models** — freemium and open-core layer free and paid offerings; open-core is a model around open source, not an open-source license.
- **Channel licenses** — OEM, redistribution, and embedding licenses let another company ship your software, priced by volume or royalty.

Next up: the gray zone between open source and proprietary — code you can see but not freely use, and why companies are moving there. See [Source-available & "fair source"](/learn/software-licensing/source-available-licenses/).
