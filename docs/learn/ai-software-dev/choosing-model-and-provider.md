---
slug: choosing-model-and-provider
title: Choosing your model & provider
description: A repeatable way to pick an AI model and provider for coding — start from your requirements (complexity, budget, privacy, context, speed, ecosystem, open vs closed), map them to a choice or a combination, and switch cheaply when something better arrives.
keywords: choosing an AI model, choosing a provider, model selection framework, coding model requirements, privacy and AI, local models, open vs closed models, frontier model, budget, context window, ecosystem fit, switching models
level: intermediate
prereq:
  - coding-models-compared
  - one-model-vs-many
faq:
  - q: "How do I pick an AI model for coding without getting it wrong?"
    a: "Start from your **requirements**, not the latest headline: how hard is the task, what's your budget, can the code leave your machine, how much context do you need, how fast must it be, and does it fit your tools? Map those to a model or a deliberate mix, then try it on real work. You can't really get it 'wrong' because switching is cheap — the point is to pick something that fits and adjust as you learn."
  - q: "Should I just use whatever model is at the top of the leaderboard?"
    a: "No. The chart-topper changes constantly, and a leaderboard measures someone else's tasks, not yours. A frontier model is also overkill — and overpriced — for routine edits. Pick by requirements, and let the leaderboard be at most a tiebreaker between options that already fit your constraints."
  - q: "When does a local or open-weight model make sense?"
    a: "When your code or data **cannot leave your machine** — strict client confidentiality, regulated data, or an air-gapped environment — or when you want predictable cost and offline use. The trade-off is that open-weight models usually trail the frontier and need capable hardware. If privacy is a hard rule, that constraint outranks raw capability."
---
# Choosing your model & provider

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Start from requirements** — complexity, budget, privacy, context, speed, ecosystem and open-vs-closed, not the hype cycle. **Map requirements to a choice** — often a deliberate combination rather than one model. **Switch cheaply** — models change fast, so pick something that fits, try it on real work, and move when something better appears.
</div>

This is lesson 23 of the path, and it opens the final module — turning everything you've learned into decisions you can actually act on. [Coding models compared](/learn/ai-software-dev/coding-models-compared/) gave you the axes; [One model vs. many](/learn/ai-software-dev/one-model-vs-many/) showed that you don't have to commit to a single one. Now we make the choice. By the end of this lesson you'll have a repeatable framework that starts from *your* requirements and ends in a model, a provider, or a deliberate mix — one that survives the next release because it's built on how the decision works, not on which model is winning today.

## Start from requirements, not the hype cycle

The most common mistake is to choose backwards: read about the newest model, decide it's the best, and then look for reasons to use it. That gets you an expensive, possibly ill-fitting tool and a decision that's stale within weeks.

Choose forwards instead. Write down what the work actually demands, then find the cheapest, simplest option that meets every hard requirement. This is ordinary engineering — you size a part to the load, not to the catalogue's flashiest entry. The hype cycle is noise; your requirements are signal.

A **requirement** here is a concrete property the task needs: "the code can't leave my laptop," "I need a fast reply while I type," "it has to reason across a 4,000-line file." Some requirements are *hard* (privacy rules, a budget cap) and some are *soft* (a little faster would be nice). Hard requirements eliminate options; soft ones break ties.

## The factors that drive the choice

Here are the dimensions to run a task through. For each, ask "what does this task actually need?" — not "what's the maximum available?"

| Factor | The question to ask | Pushes you toward |
|---|---|---|
| **Task complexity** | Routine edits, or genuinely hard multi-step reasoning? | Simple work → a fast small model; hard work → a frontier model |
| **Budget** | Hobby spending, or a funded project? | Tight budget → cheap/small tiers, free apps, or local; generous → frontier when it earns its keep |
| **Privacy / data rules** | Can this code legally and contractually leave your machine? | A hard "no" → local / open-weight or a zero-retention enterprise tier |
| **Context size** | How much code must the model hold in view at once? | Whole-repo work → a large [context window](/learn/ai-software-dev/context-windows/); a single function → almost any model |
| **Speed / latency** | Do you need a reply in the flow of typing, or can you wait? | In-editor completion → fast/small; deep reasoning → slower is fine |
| **Ecosystem / tooling fit** | Does it work cleanly with your editor, agent, and language? | Whatever integrates with the tools you already use |
| **Open vs closed** | Do you need to inspect, host, or fully control the model? | Control / offline → open-weight; peak capability → closed/hosted |

Two rules make the table usable. First, **a factor only matters if it touches your task** — a tiny context window is fatal for refactoring a large file and irrelevant for completing one line. Second, **hard requirements come first.** If the code can't leave your machine, privacy decides the whole question before capability gets a vote.

### Privacy is special — it can outrank everything

Most factors are trade-offs you balance. Privacy is often a *gate*. When you use a hosted model, your prompts and code are sent to the provider (we cover exactly where they go in [Security, privacy & ethics](/learn/ai-software-dev/security-privacy-ethics/)). If a contract, a regulation, or plain caution says that's not allowed, then no amount of frontier capability rescues a model that breaks the rule. In that case you start from "must run locally" and choose the best option *within* that constraint. Decide privacy first, then optimise everything else inside the box it leaves you.

## Map requirements to a choice — often a combination

Once the factors are written down, the answer is usually not a single model. As [One model vs. many](/learn/ai-software-dev/one-model-vs-many/) argued, the strong move is a deliberate mix: route each kind of work to the option that fits it. A common shape:

- A **fast, cheap small model** as the default for routine edits, completions, and quick questions.
- A **frontier flagship** reserved for the genuinely hard problems — subtle bugs, tricky designs, unfamiliar code.
- An **open-weight local model** kept ready for anything that must stay on your machine.

You don't have to use all three. The point is that "which model?" can have a composite answer, and choosing the mix deliberately beats forcing every task through one tool.

## Three worked mini-examples

**A hobby project on a tight budget.** You're building a small tool on weekends and want to spend as close to nothing as possible. Complexity is moderate, privacy isn't a concern, speed is pleasant-to-have. Map it: budget is the hard constraint, everything else is soft. Choice — lean on a provider's free or low-cost app and a fast small-tier model for the bulk of the work, and only reach for a paid frontier model on the rare occasion you're genuinely stuck. You're optimising for cost, and the small tier handles most hobby coding fine.

**A privacy-sensitive codebase.** You maintain code under a strict confidentiality agreement: it cannot be sent to a third party. Here privacy is a gate, not a trade-off. No matter how capable a hosted frontier model is, it's disqualified. Map it: choose an **open-weight model you run locally** (or a vetted zero-retention enterprise tier if your policy explicitly allows it), accept that it trails the frontier, and provision hardware to run it. You optimise for capability *within* the privacy box — not across it.

**A complex agentic workflow.** You're handing an [agent](/learn/ai-software-dev/agentic-cli-tools/) a multi-file feature: read the repo, make coordinated changes across several packages, run the test suite, and iterate until green. This needs strong reasoning, [tool use / autonomy](/learn/ai-software-dev/agentic-and-vibe-coding/), and a large context window — and the value of getting it right is high. Map it: complexity and autonomy dominate, budget is secondary because the task is worth it. Choice — a **frontier flagship** with good agentic tool use and a big context window. The cost per task is real, but here it buys correctness on work that would take you hours by hand.

Notice the pattern: the same framework produces three different answers because the *requirements* differ. That's the whole idea.

## Picking the provider, not just the model

The model is only half the decision; the **provider** is the company or project that hosts or distributes it. Two models of similar capability can sit behind very different providers, and the provider often matters more than a few capability points. Weigh:

- **Data handling** — retention policy, whether your inputs may train future models, and zero-retention options. (Detailed in [Security, privacy & ethics](/learn/ai-software-dev/security-privacy-ethics/).)
- **Access and limits** — pricing model, rate limits, and regional availability (see [Usage limits & tiers](/learn/ai-software-dev/usage-limits-and-tiers/)).
- **Ecosystem fit** — does it plug into your editor, your agent, your language's tooling?
- **Stability and trust** — is the provider likely to be around, and does it document its policies clearly?

For an open-weight model, the "provider" is partly *you*: you host it, so retention and uptime are in your hands, and your obligation shifts to having the hardware and keeping it running.

## Models change fast — so optimise for cheap switching

Here's the meta-point that keeps this lesson evergreen: whatever you pick today will be superseded. New models, new tiers, and new providers arrive constantly, and the "best" option this quarter rarely holds the title next quarter. The losing strategy is to agonise over the perfect choice; the winning one is to **pick something that fits your requirements, try it on real work, and design your setup so switching is cheap.**

You make switching cheap by avoiding hard lock-in: keep your prompts and [config files](/learn/ai-software-dev/skills-and-config-files/) portable, prefer tools and standards that work across providers, and treat the model as a swappable component rather than a foundation you build on top of. Then, when something genuinely better appears, you run it against the same real tasks and move if it wins. The framework above doesn't expire — only the specific names that fill it in do.

For current prices, limits, and model lineups, always check each provider's live documentation rather than trusting a number you read months ago.

<div class="knowledge-check" data-quiz data-correct-msg="Right — choose from your requirements, and because models change fast, keep switching cheap rather than chasing the leaderboard." markdown="0">
  <p class="knowledge-check__q">Quick check: what should drive your choice of AI model and provider?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Whichever model currently sits at the top of the benchmark leaderboard</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Your task's requirements — complexity, budget, privacy, context, speed, fit — mapped to a fitting choice you can switch cheaply</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The most expensive frontier model, since it can handle anything</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Choose forwards** — write down what the task needs, then find the simplest option that meets every hard requirement.
- **Run the factors** — task complexity, budget, privacy, context size, speed, ecosystem fit, and open vs closed each push the choice.
- **Privacy can gate everything** — if code can't leave your machine, that rule outranks raw capability and points to local or zero-retention options.
- **Often a combination** — route routine work to a cheap small model, hard problems to a frontier model, and private work to a local one.
- **Provider matters too** — data handling, limits, ecosystem fit, and trust can outweigh a few capability points.
- **Switch cheaply** — models change fast, so pick something that fits, try it on real work, and keep your setup portable.

Next up: now that you've chosen a model, choose *how you reach it* — assembling a personal setup of app, IDE, and agent that matches how you work. See [Choosing your method](/learn/ai-software-dev/choosing-your-method/).
