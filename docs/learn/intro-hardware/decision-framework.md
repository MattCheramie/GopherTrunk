---
slug: decision-framework
title: A practical decision framework
description: A repeatable five-step method for turning requirements into a hardware choice — state needs, eliminate on hard constraints, weigh trade-offs, pick the simplest fit, then plan to scale.
keywords: hardware decision framework, choosing hardware method, hard constraints, hardware trade-offs, simplest thing that works, scaling hardware, eliminate platforms, hardware checklist
level: advanced
status: full
faq:
  - q: "What's the single most useful step in choosing hardware?"
    a: "Applying **hard constraints** to eliminate options. A few pass/fail facts — needs attached USB hardware, must run on a battery, must be a public 24/7 service — cut the field from everything down to two or three before you weigh anything. It's the cheapest, highest-leverage move, because elimination is faster and more certain than comparison."
  - q: "How do hard constraints differ from trade-offs?"
    a: "A **hard constraint** is pass/fail: a platform either can do the thing or it can't, so it's in or out (a cloud VPS *cannot* have a USB dongle attached, full stop). A **trade-off** is a matter of degree — cost, power, performance, effort — where every surviving option is viable and you're weighing which balance you prefer. Eliminate on the first, then weigh the second."
  - q: "Why prefer the simplest thing that meets the needs?"
    a: "Because every bit of extra capability is cost, power draw, and complexity you pay for forever, usually for nothing. The discipline is to **buy or rent the smallest thing that does the job** — it's cheaper, more reliable, and easier to maintain. You can always step up later; you rarely regret starting small."
  - q: "Should I plan for scale from the start?"
    a: "Plan for it, don't build for it prematurely. Know your likely growth path — a [VPS](/learn/intro-hardware/vps/) you can resize, a design that moves from one unit to many — so a change isn't a rewrite. But don't pay today for capacity you may never need; pick the simple option now and keep the upgrade path open."
---
# A practical decision framework

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Eliminate, then weigh, then keep it simple** — hard constraints cut the field, trade-offs rank what's left, and you pick the smallest thing that does the job. **Hard constraints do the heavy lifting** — a few pass/fail facts often decide the choice on their own. **Plan to change** — choose with an upgrade path in mind, but don't pay today for scale you may never need.
</div>

You've collected requirements, learned which project types reach for which platforms, and seen how tiers combine. This lesson assembles all of it into a method you can run the same way every time — five steps that move from facts (what's even possible?) to judgement (what's the best fit?). It's deliberately simple, because a method you'll actually use beats a perfect one you won't.

## The five steps

1. **State the requirements.** Write them down using the checklist from [Start with your requirements](/learn/intro-hardware/start-with-requirements/) — compute, connectivity, physical-world contact, power, budget, quantity, location, maintenance. You can't choose well against needs you haven't named.
2. **Apply hard constraints to eliminate.** These are pass/fail. Walk each requirement and strike out every platform that simply can't satisfy it. This is the cheapest, highest-leverage step — it usually cuts the field from "everything" to two or three before you compare anything.
3. **Weigh the soft trade-offs.** For what survives, balance the recurring four — cost, power, performance, and effort to build and maintain. The [cost, power & performance](/learn/intro-hardware/cost-power-performance/) lesson is the toolkit here; the point is that every survivor is *viable*, so now you're choosing a balance, not a winner.
4. **Prefer the simplest thing that meets the needs.** Among the viable options, pick the smallest, cheapest, lowest-power one that clears the bar. Extra capability you don't need is cost and complexity you pay for forever.
5. **Plan to scale or change.** Choose with an upgrade path in mind — a [VPS](/learn/intro-hardware/vps/) you can resize, a design that survives going from one unit to a hundred — but don't over-build today for a future that may not arrive.

```text
requirements → eliminate (hard constraints) → weigh trade-offs → pick simplest → plan to scale
   (facts)                                                                        (judgement)
```

## Hard constraints eliminate fast

Step two deserves its own focus, because it does most of the work. A hard constraint is a fact that makes a platform impossible, not merely worse. Internalize a few and most decisions collapse:

| If the project... | ...then rule out | ...and lean toward |
|-------------------|------------------|--------------------|
| Needs attached physical/USB hardware | Any remote cloud VPS or shared host | A device on-site: [SBC](/learn/intro-hardware/what-is-an-sbc/), PC, or [microcontroller](/learn/intro-hardware/what-is-a-microcontroller/) |
| Must run for months on a battery | Full computers and SBCs | A [microcontroller](/learn/intro-hardware/what-is-a-microcontroller/) (e.g. [ESP32](/learn/intro-hardware/esp32/)) with deep sleep |
| Needs a full operating system & general software | Microcontrollers | An [SBC](/learn/intro-hardware/what-is-an-sbc/), [desktop](/learn/intro-hardware/desktop-computers/), or server |
| Must be a public, always-on internet service | Battery devices, home machines on a flaky link | A [VPS](/learn/intro-hardware/vps/) or [dedicated server](/learn/intro-hardware/dedicated-servers/) |
| Ships in the thousands at low unit cost | Expensive boards bought one at a time | The cheapest part that meets the spec at volume |
| Needs serious GPU compute | Tiny and low-power platforms | A strong [desktop](/learn/intro-hardware/desktop-computers/) or rented cloud GPU |

Notice these are *in-or-out* decisions. You're not comparing a VPS against an SBC on price when the project needs a USB dongle attached — the VPS is simply impossible, and saying so out loud ends the debate. Elimination is faster and more certain than comparison, which is exactly why it goes first.

## Weighing what survives

Once hard constraints have trimmed the list, every remaining option *can* do the job — so the decision shifts to trade-offs of degree. Lay them side by side on the four dimensions that recur throughout this path:

- **Cost** — to buy and to keep running, at the quantity you actually need.
- **Power** — draw matters most for battery devices and anything always-on, where electricity adds up over a year.
- **Performance** — only as much as the job needs; headroom beyond that is waste.
- **Effort** — to set up, program, and maintain, including who has to look after it.

There's rarely a clean winner, and that's fine. The goal is to make the trade-offs *visible* so the choice is deliberate. If two options are genuinely close, the next principle breaks the tie.

## Prefer the simplest fit, then plan to change

Among viable options, **buy or rent the smallest thing that does the job.** A $35 Pi over a $1,000 server when both work; a single resizable VPS over a dedicated machine until traffic demands more; a $4 microcontroller over an SBC when all you need is a sensor. Simpler means cheaper, more reliable, lower-power, and easier to maintain — and you can almost always step up later.

That last clause is step five. Pick simple *now*, but with eyes open about how you'd grow: a VPS you can scale up with a click, a sensor design that moves from one to many without a redesign, a home server you could later mirror to the cloud. Planning the path is cheap; building it before you need it is not. The mistake to avoid is paying today for scale that may never arrive — over-provisioning is just a slower way to over-buy.

<div class="knowledge-check" data-quiz data-correct-msg="Right — applying hard, pass/fail constraints eliminates impossible options and cheaply cuts the field before you weigh anything." markdown="0">
  <p class="knowledge-check__q">Quick check: What's the highest-leverage step after stating your requirements?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Score every platform on cost and performance in detail</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Apply hard, pass/fail constraints to eliminate platforms that simply can't do the job</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Buy the most powerful option so you never hit a limit</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Five steps** — state requirements, eliminate on hard constraints, weigh the trade-offs, prefer the simplest fit, plan to scale.
- **Hard constraints do the heavy lifting** — pass/fail facts cut the field to a handful before any comparison.
- **Constraints rule out, trade-offs rank** — a platform is in or out on a constraint; among survivors you weigh cost, power, performance, and effort.
- **Smallest thing that works** — buy or rent the least capability that meets the need; headroom is cost you pay forever.
- **Plan the path, don't pre-build it** — choose with an upgrade route in mind, but don't pay today for scale you may never reach.

Next up: run this framework end to end on four real projects to make it concrete. See [Worked examples: picking hardware for real projects](/learn/intro-hardware/worked-examples/).
