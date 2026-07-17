---
slug: privacy-data-and-compliance
title: Privacy, data & compliance
description: Sending a prompt to a hosted model shares your data with a third party. Learn where that data goes, how to read a provider's retention and training terms, how to minimize and redact PII, and the compliance basics that apply.
keywords: AI privacy, data handling, PII, data retention, training on your data, GDPR, CCPA, DPA, data residency, zero retention, data processing agreement, self-hosting
level: intermediate
status: full
prereq:
  - building-vs-using-ai
faq:
  - q: "Does sending a prompt to a model API share my data?"
    a: "Yes. A hosted model runs on the provider's servers, so anything in the prompt — user text, records, code — leaves your systems and reaches a third party to be processed. Treat it as a data-sharing decision: know the provider's terms and send only what the feature needs."
  - q: "Will my API inputs be used to train the model?"
    a: "It depends on the provider and tier. On business and API tiers, training on your inputs is often off by default — but you must verify it in the current terms rather than assume. Consumer and free tiers are more likely to use inputs for training. When it matters, choose a tier with a written no-training or zero-retention guarantee."
  - q: "What is a Data Processing Agreement (DPA)?"
    a: "A DPA is a contract between you and a provider that processes personal data on your behalf. It sets out what they may do with the data, security obligations, sub-processors, and deletion — and it's usually required to use a provider compliantly under laws like GDPR. Many providers publish a standard DPA you can accept."
  - q: "Is running a model myself more private?"
    a: "It can be. Self-hosting an open-weight model keeps data inside your own infrastructure, so prompts never reach a third party — a strong fit for sensitive data. The trade is real: you take on the capability gap, the operations, and the hardware cost yourself."
---
# Privacy, data & compliance

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Know where your data goes** — sending a prompt to a hosted model API shares that
data with a third party, so treat it as a data-sharing decision, not a detail.
**Check retention and training** — read the provider's terms for how long inputs are
kept and whether they train on them (often off by default on business tiers, but
verify). **Minimize and redact** — send the least data the feature needs and strip
identifiers before they leave your systems. If you're new to the shape of an AI
feature, start with
[what it means to build AI into software](/learn/building-ai/building-vs-using-ai/);
the sibling path's
[security, privacy & ethics](/learn/ai-software-dev/security-privacy-ethics/) covers
the developer-at-the-keyboard side of the same concerns.
</div>

The moment you move a model call inside your software, you are routing your users'
data through it. That's a privacy and compliance responsibility, and it lands on you,
not the model. This lesson is the practical version of that responsibility: where the
data goes, what to read before you ship, what to strip out, and the rules that may
apply. It's educational, not legal advice — when real obligations are on the line,
involve someone qualified.

## Where does the data go?

A **hosted model API** runs on the provider's servers. So when your code sends a
prompt — a support message, a customer record, a decoded packet, a chunk of a document
— that data leaves your systems and travels to the provider to be processed. There is
no way around it for a hosted model: the computation happens on their side, so the
input has to get there.

The important mental shift is to treat this as a **data-sharing decision**, not an
implementation detail. Every field you put in a prompt is a field you have chosen to
disclose to a third party. Decide deliberately what belongs there and what doesn't.

## Read the provider's data terms

Before you route user data to any provider, read how they handle it. The questions
that matter:

- **Do they train on your inputs?** Whether your prompts may be used to train or
  improve future models. On API and business tiers this is **often off by default** —
  but verify it in the provider's current terms rather than assume, because policies
  differ and change.
- **How long is data retained?** How long inputs and outputs are stored, and who can
  access them in that window.
- **Who are the sub-processors?** The other companies the provider may pass your data
  to (hosting, logging, abuse monitoring). These are usually listed.
- **Where is it hosted?** Which region or country your data is processed and stored in
  — this feeds directly into the compliance questions below.

Many providers offer **enterprise options** such as **zero-retention** processing,
where inputs aren't stored after the response is returned, and contractual no-training
guarantees. For sensitive workloads, those options are often the difference between a
provider you can use and one you can't.

The [data-agreement lesson](/learn/software-licensing/privacy-and-data-agreements/) in
the Software Licensing path goes deeper on reading these terms as a contract.

## Minimize and protect PII

The safest data is the data you never send. Work from that principle:

- **Send the least data needed.** If the feature only needs the body of a message,
  don't include the sender's name, account ID, and email too. Trim the prompt to what
  the task actually requires.
- **Redact or anonymize identifiers.** Strip or mask **PII** — names, emails, phone
  numbers, addresses, account numbers — before it leaves your systems, or replace it
  with placeholders you can restore on your side after the response.
- **Get consent and honor deletion.** Where users' personal data is involved, make
  sure you have a basis to process it, and be able to delete it (including from the
  provider) when asked.
- **Take extra care with special categories.** Health, financial, biometric, and
  similar sensitive data carry stricter rules and higher stakes. Think hard before any
  of it enters a prompt, and prefer stronger guarantees (zero-retention, or
  self-hosting) when it must.

## Compliance

Once personal data is involved, laws may apply regardless of how the feature is built.
The basics worth knowing:

- **GDPR / CCPA.** Privacy laws such as the EU's **GDPR** and California's **CCPA**
  give people rights over their personal data — to know what's collected, to access
  it, and to have it deleted — and put obligations on you as the one handling it. Using
  a model API doesn't exempt you from them.
- **Data Processing Agreements (DPAs).** When a provider processes personal data on
  your behalf, you typically need a **DPA** in place — a contract setting out what they
  may do with the data and their security and deletion obligations.
- **Data residency and localization.** Some rules require personal data to stay within
  a specific region or country. That constrains which providers and hosting regions you
  can use, which is why "where is it hosted?" matters.
- **Sector rules.** Health, finance, education, and government data often carry their
  own regimes on top of general privacy law.

You don't need to be a lawyer, but you do need to know when you're out of your depth.
**When in doubt, involve legal** before you ship.

## Secrets never go in prompts

Separate from personal data, one category is simply off-limits: **secrets**. API keys,
passwords, access tokens, and credentials do not belong in a prompt or in an agent's
context. Once sent to a provider you've lost control of where that value lives and how
long it's retained. And there's a sharper edge for agents: an agent that reads
untrusted content can be hijacked by
[prompt injection](/learn/building-ai/prompt-injection-and-security/), and a secret in
its context is exactly what an injected instruction will try to exfiltrate. Keep secrets
in environment variables or a secrets manager, and redact them with placeholders before
any code that contains them goes into a prompt.

## Self-hosting as an option

If data truly cannot leave your control, the strongest answer is not to send it out at
all. Running an **open-weight model** yourself keeps prompts inside your own
infrastructure — nothing reaches a third-party provider, which sidesteps the retention,
training, and residency questions in one move. The
[sibling lesson on local and open models](/learn/ai-software-dev/local-and-open-models/)
covers what that involves.

It's a genuine trade, not a free win. Self-hosted open-weight models can trail the best
hosted models in capability, and you take on the operational burden and hardware cost.
Weigh that against how sensitive the data is: for the most sensitive workloads, keeping
data in-house is worth the effort; for the rest, a zero-retention hosted tier may be the
better balance.

## Be transparent

Privacy isn't only about mechanics — it's also about trust. Tell users when **AI is
involved** in a feature and, where it matters, what data it touches. People have a
reasonable expectation to know when their information is being sent to a model, and
being upfront about it is both good ethics and, increasingly, a legal expectation.

<div class="knowledge-check" data-quiz data-correct-msg="Right — understand the terms and minimize what you send before any user data leaves your systems." markdown="0">
  <p class="knowledge-check__q">Quick check: before sending user data to a hosted model API, you should first...?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Assume the API tier keeps everything private and send it as-is</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Understand the provider's data/retention terms and minimize/redact what you send</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Encrypt the prompt so the provider can't read it</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **hosted model API** means user data leaves your systems for a third party — treat
  it as a data-sharing decision, not a detail.
- **Read the provider's terms**: training on inputs (often off by default on business
  tiers, but verify), retention, sub-processors, hosting region, and zero-retention
  options.
- **Minimize and redact** — send the least data needed, strip PII, get consent, honor
  deletion, and take extra care with health, financial, and biometric data.
- **Compliance basics**: GDPR/CCPA rights, DPAs, data residency, and sector rules — and
  involve legal when in doubt.
- **Secrets never go in prompts**; **self-hosting** an open-weight model keeps sensitive
  data in-house, at a cost in capability and effort.
- Be **transparent** with users about when AI is involved and what data it touches.

Next up: [observability & monitoring](/learn/building-ai/observability-and-monitoring/).
