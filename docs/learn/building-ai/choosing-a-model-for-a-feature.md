---
slug: choosing-a-model-for-a-feature
title: Choosing a model for a feature
description: How to pick the right model for an AI feature — match the model to the job, run the dimensions that matter (capability, context, latency, cost, privacy), right-size to the smallest model that passes your evals, and abstract the provider so you can swap without pain.
keywords: choosing a model, model selection, right-sizing, model routing, context window, latency, cost, open weight, provider lock-in, model evals, self-hosted models, capability class
level: intermediate
status: full
prereq:
  - cost-latency-and-limits
faq:
  - q: How do I choose a model for an AI feature?
    a: Start from the feature's requirements, not from whichever model is famous this month. Write down what the job actually needs — capability, context size, latency, cost ceiling, and any privacy rules — then run your eval set against a shortlist and pick the smallest, cheapest model that passes. Revisit the choice on a schedule, because the options change every few months.
  - q: "Should I just use the biggest, most capable model everywhere?"
    a: "No. The largest model is usually slower and far more expensive than a simple, high-volume step needs, and it rarely scores better on easy work. Right-size instead: reserve a strong model for the genuinely hard reasoning steps and route simple, frequent steps to a small, fast, cheap one. Most real products end up using more than one."
  - q: "When does an open-weight or self-hosted model make sense?"
    a: "When data cannot leave your environment, when you need full control over versioning and uptime, or when predictable cost at high volume matters more than peak capability. The trade-off is that you take on the hosting, scaling, and maintenance, and open-weight models can trail the strongest proprietary ones. If a privacy or compliance rule is hard, it outranks raw capability."
  - q: "How do I avoid getting locked into one provider?"
    a: "Treat the model as a swappable component. Put every call behind your own interface so the rest of the code depends on your abstraction, not a vendor SDK. Choose by capability class rather than a specific model name, keep your prompts and eval set portable, and re-evaluate on a schedule so switching is a routine swap instead of a rewrite."
---
# Choosing a model for a feature

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Match the model to the job** — a hard reasoning step and a simple, high-volume
step have different needs. **Right-size** — start from requirements and pick the
smallest model that passes your evals, not whichever model is famous this month.
**Use more than one** — most products route different steps to different models,
and abstract the provider so swapping is cheap. Pick per feature, by requirements
and [evals](/learn/building-ai/evaluating-ai-features/), and revisit on a schedule.
See [cost, latency & limits](/learn/building-ai/cost-latency-and-limits/) for the
constraints, and the sibling
[Choosing your model & provider](/learn/ai-software-dev/choosing-model-and-provider/)
for the coding-tools angle.
</div>

"Which model should I use?" has no single answer, because your feature is not one
job. Picking well is ordinary engineering: name what the work demands, then choose
the cheapest option that meets it — and expect the answer to change as the models do.

## Match the model to the job

A feature is usually several steps, and they don't all need the same horsepower.
Classifying an incoming message, extracting a field, or drafting a one-line summary
is easy, frequent work. Reasoning across a long document, planning a multi-step
change, or debugging subtle logic is hard, occasional work.

Sending every step to one big model overpays on the easy ones and can still be
overkill; sending everything to one small model saves money right up until it fails
the hard step and the whole feature feels unreliable. That's why most real products
use several models: a **strong model** for the genuinely hard reasoning steps, and a
**small, fast, cheap model** for the simple, high-volume ones. Deciding which step
goes where is the heart of the choice — the sibling
[One model vs. many](/learn/ai-software-dev/one-model-vs-many/) covers the routing
pattern in depth.

## The dimensions that matter

For each step, run it through the dimensions below and ask "what does *this* step
actually need?" — not "what's the maximum available?"

| Dimension | The question to ask |
|---|---|
| **Capability** | Routine extraction, or hard multi-step reasoning? |
| **Context window** | How much text must the model hold at once — a snippet, or a whole document? |
| **Latency / speed** | Does a user wait on this reply, or does it run in the background? |
| **Cost** | What does one call cost, multiplied by how often the step runs? |
| **Modality** | Text only, or does it need images, audio, or other [multimodal input](/learn/building-ai/multimodal-inputs/)? |
| **Structured output & tools** | Does it need reliable [structured output](/learn/building-ai/structured-output/) or [tool use](/learn/building-ai/tool-use-and-function-calling/)? |
| **Privacy / hosting** | Can this data leave your environment, and where may it be sent? |

Two rules make the table usable. A dimension only matters if it touches the step — a
tiny context window is fatal for whole-document work and irrelevant for a one-line
classification. And hard requirements come first: if a
[privacy or compliance](/learn/building-ai/privacy-data-and-compliance/) rule says
the data can't leave your machine, that decides the question before capability gets a
vote, and points you toward
[local and open models](/learn/ai-software-dev/local-and-open-models/).

## Right-size it

The reflex is to default to the biggest, most capable model "to be safe." Resist it.
The largest model is usually the slowest and most expensive, and on easy work it
rarely scores better than a small one — you pay more for the same result and a worse
latency.

Right-sizing runs the other way. Start from the step's requirements, then pick the
**smallest, cheapest model that passes your evals**. That means you need an
[eval set](/learn/building-ai/evaluating-ai-features/) before you can choose well: a
handful of real examples with known-good answers, run against each candidate. Promote
to a bigger model only when the smaller one measurably fails — not on a hunch. This
is also the first lever in
[optimizing cost & latency](/learn/building-ai/optimizing-cost-and-latency/): the
cheapest call that still passes is almost always a smaller model.

## Proprietary vs. open-weight / self-hosted

A related axis is who runs the model. A **proprietary, hosted** model is called
through a provider's API — you get peak capability and zero operational burden, but
your data goes to the provider and you don't control versioning or uptime. An
**open-weight, self-hosted** model runs in your own environment — you gain control,
privacy, and predictable cost at volume, but you take on the hardware, scaling, and
maintenance, and open weights can trail the strongest hosted models.

Neither is universally right. Weigh capability, cost at your volume, privacy needs,
and how much control the feature actually requires. The siblings
[Local & open models](/learn/ai-software-dev/local-and-open-models/) and
[The provider landscape](/learn/ai-software-dev/the-provider-landscape/) map the
options in more detail.

## Avoid lock-in

Whatever you pick today will be superseded — new models and tiers arrive every few
months, and the best option this quarter rarely holds the title next quarter. Design
for that instead of fighting it.

Put every model call behind **your own interface**, so the rest of your code depends
on your abstraction rather than a vendor's SDK. Choose by **capability class** ("a
strong reasoning model," "a fast cheap one") rather than hard-coding a single model
name. Keep your prompts and eval set portable. Then re-evaluate on a schedule: run the
same evals against new candidates and swap the one behind the interface if a better
option wins. Done this way, switching is a routine config change, not a rewrite.

## A selection routine

1. **Write the requirements** for each step — capability, context, latency, cost
   ceiling, modality, and any hard privacy rule.
2. **Shortlist** two or three candidates in the right capability class.
3. **Run your eval set** against each on real examples.
4. **Weigh cost, latency, and privacy** among the ones that pass.
5. **Choose** the smallest, cheapest option that clears the bar — and note when to
   revisit it.

<div class="knowledge-check" data-quiz data-correct-msg="Right — define requirements first, then pick the smallest model that passes your evals, not the biggest by default." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the best way to choose a model for a feature?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Always use the largest, most capable model so it can handle anything</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Define requirements, then pick the smallest/cheapest model that passes your evals — not "always the biggest"</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Pick whichever model is topping the leaderboard this month</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Match the model to the job** — hard reasoning steps and simple, high-volume steps
  have different needs.
- **Run the dimensions** — capability, context window, latency, cost, modality,
  structured-output/tool support, and privacy each shape the choice.
- **Right-size** — start from requirements and pick the smallest model that passes
  your evals, not the biggest by default.
- **Proprietary vs. open-weight** — trade peak capability and zero ops against
  control, privacy, and predictable cost.
- **Avoid lock-in** — abstract the provider behind an interface, choose by capability
  class, and re-evaluate on a schedule.
- **Most products use more than one model**, routed per step.

Next up: [optimizing cost & latency](/learn/building-ai/optimizing-cost-and-latency/).
