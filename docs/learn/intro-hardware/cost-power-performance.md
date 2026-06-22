---
slug: cost-power-performance
title: Cost, power & performance trade-offs
description: Money, electricity, speed, control, and effort — five tensions that pull against each other in every hardware choice. Learn the axes so later decisions become deliberate, not guesswork.
keywords: hardware trade-offs, cost power performance, upfront vs ongoing cost, power draw battery vs always-on, control vs convenience, hardware maintenance effort, choosing hardware
level: beginner
status: full
faq:
  - q: "What are the main trade-offs in choosing hardware?"
    a: "Five recur in almost every decision: **cost** (upfront versus ongoing), **power draw** (battery, wall, or always-on), **performance**, **control** (how much is yours to manage versus handled for you), and **effort** to set up and maintain. They pull against each other — gaining on one axis usually costs you on another, which is why there's rarely a single best answer."
  - q: "What's the difference between upfront and ongoing cost?"
    a: "**Upfront cost** is what you pay once to acquire the hardware — buying a server or a Raspberry Pi. **Ongoing cost** is what it costs to keep running — monthly hosting fees, electricity, or replacement parts. A cheap-to-buy option can be expensive to run, and vice versa, so you have to weigh both against how long you'll use it."
  - q: "Why does power draw matter so much?"
    a: "It decides where and how a device can live. A microcontroller drawing milliwatts can run for years on a battery in a remote location; an always-on home server draws steady wall power and adds to your electricity bill 24/7. Power draw shapes battery life, running cost, and whether a device even needs cooling."
  - q: "Does more control always mean more work?"
    a: "Usually, yes. Renting cloud hosting gives you little control but almost no maintenance — the provider handles the hardware. Owning a home server gives you total control but makes every update, failure, and backup your responsibility. More control means more effort; more convenience means trusting someone else's choices."
---
# Cost, power & performance trade-offs

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Five recurring axes** — cost, power draw, performance, control, and effort show up in nearly every hardware decision. **They trade against each other** — gaining on one axis almost always costs you on another, so there's rarely a free win. **Knowing the axes makes choices deliberate** — once you can name the tensions, picking hardware becomes reasoning instead of guessing.
</div>

Across the last three lessons the same handful of tensions kept surfacing: things cost money, draw power, run fast or slow, hand you control or take it away, and demand more or less work. This lesson pulls those tensions out and names them directly. They're the axes every hardware decision turns on, and getting comfortable with them now is what lets Module 7 turn them into an actual method for choosing.

## Upfront vs ongoing cost

Cost has two faces, and confusing them leads to bad decisions. **Upfront cost** is what you pay once to get the hardware — buying a desktop, a server, or a Raspberry Pi. **Ongoing cost** is what it takes to keep it running — monthly hosting bills, electricity, replacement drives, your own time.

The two often trade against each other. Cloud hosting is near-free to start but bills you every month forever. A microcontroller costs a few dollars once and almost nothing to run. A home server is the opposite of the cloud: you pay real money upfront, then a smaller but constant electricity cost. The right balance depends on how long you'll use the thing — a few months favors low upfront cost; years of always-on use can favor owning hardware outright.

## Power draw: battery, wall, or always-on

How much electricity a device uses decides where it can live and what it costs to run. There are three rough regimes:

- **Battery** — phones, tablets, and low-power microcontrollers. Here every milliwatt matters, because power runs out. A microcontroller can run for years on a coin cell; a laptop lasts hours.
- **Wall power, on demand** — desktops and laptops plugged in when used. Power draw matters for the electricity bill but isn't a hard limit.
- **Always-on** — servers and home servers running 24/7. Even modest draw adds up over a year, and high draw means heat, fans, and cooling to manage.

Power draw and performance are linked: more capability generally means more watts. A microcontroller's tiny power budget is exactly what makes it suitable for a sensor left in a field for a year — and exactly what makes it useless for anything heavy.

## Performance: enough vs overkill

Performance is the obvious axis — how fast and how much the machine can do — but the useful question is rarely "what's fastest?" It's "what's *enough*?" A platform far more powerful than the job needs wastes money and electricity for nothing.

Reading a temperature once a minute needs almost no performance; a microcontroller does it perfectly. Serving a busy website needs a great deal, and a server delivers it. Matching performance to the job — neither starving it nor drowning it in overkill — is most of the skill. Buying the most powerful option "to be safe" is one of the most common and expensive mistakes.

## Control vs effort

The last two axes are tightly coupled, so it's worth taking them together. **Control** is how much of the machine is yours to decide; **effort** is how much work it takes to set up and keep running. They almost always move together: more control means more effort, and more convenience means handing both away.

Rent cloud hosting and you control little — but the provider handles the hardware, cooling, and failures, so your effort is minimal. Own a home server and you control everything — but every update, failed drive, and backup is now your job. Neither is better in the abstract. A beginner who just wants a website online wants the cloud's low effort; a tinkerer who enjoys the machine wants the home server's control. The question is which side of that trade you actually want.

## How the axes pull against each other

The reason hardware choice is interesting — and why there's no universal best answer — is that these axes fight one another. You rarely improve one without giving ground on another:

| Axis | Cheap end | Expensive / demanding end | The tension |
|------|-----------|----------------------------|-------------|
| **Cost** | Low upfront (cloud, MCU) | High upfront (owned server) | Pay now or pay forever |
| **Power draw** | Milliwatts (battery MCU) | Always-on watts (server) | Frugal and limited, or capable and hungry |
| **Performance** | Just enough (SBC, MCU) | Abundant (server, desktop) | Save money or buy headroom |
| **Control** | Managed for you (cloud) | All yours (home server) | Convenience or autonomy |
| **Effort** | Low (managed hosting) | High (self-hosted) | Let go or do it yourself |

GopherTrunk shows every one of these at once. Run it on a dedicated server and you get maximum performance and always-on uptime, at high cost, power draw, and effort. Run it on a laptop and you trade some of that for convenience and portability. Run it on a Raspberry Pi at the antenna and you get a cheap, low-power, set-and-forget node — giving up raw performance for it. Same software, three points on every axis, and no single "right" choice — just the one that fits your constraints.

That's the whole point of naming these axes. Once you can see a decision in terms of cost, power, performance, control, and effort, you stop guessing and start weighing. Module 7 builds directly on this, turning these five tensions into a repeatable way to match hardware to a project.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the axes trade against each other, so gaining on one (like control) usually costs you on another (like effort)." markdown="0">
  <p class="knowledge-check__q">Quick check: What is the key thing to understand about these five trade-off axes?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">One platform can max out all five at once if you spend enough</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">They pull against each other — gaining on one axis usually costs you on another, so there's rarely a free win</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Only cost and performance actually matter; the rest are minor</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Cost has two faces** — upfront (paid once) and ongoing (paid forever); a cheap-to-buy option can be expensive to run, so weigh both against how long you'll use it.
- **Power draw sets where a device can live** — battery, on-demand wall power, or always-on, and it climbs with performance.
- **Performance is about "enough," not "most"** — overkill wastes money and electricity; match the machine to the job.
- **Control and effort move together** — more control means more maintenance; more convenience means trusting someone else's choices.
- **The axes trade against each other** — there's rarely a free win, which is exactly why naming them turns hardware choice into reasoning.

Next up: the spectrum's high end in detail — renting space on someone else's machine to put something on the web. See [Web & shared hosting](/learn/intro-hardware/web-hosting/).
