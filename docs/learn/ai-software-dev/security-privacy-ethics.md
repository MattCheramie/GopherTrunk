---
slug: security-privacy-ethics
title: Security, privacy & ethics
description: Where your prompts and code go, data retention and training, why you must never paste secrets, the open questions around IP and licensing, the security risks in AI-written code and prompt injection, and the ethics of using these tools responsibly.
keywords: AI security, AI privacy, data retention, training on your data, zero retention, secrets and API keys, AI code licensing, IP ownership, prompt injection, insecure code, vulnerable dependencies, AI ethics, responsible use
level: intermediate
prereq:
  - verification-and-trust
faq:
  - q: "Does the AI provider see my code?"
    a: "If you use a hosted model, yes — your prompts and any code in them are sent to the provider to generate a response. What happens next depends on the provider's **retention policy** and whether your tier allows inputs to train future models. If the code truly cannot leave your machine, run a local model instead. Always check the provider's current data-handling docs."
  - q: "Who owns code that an AI writes?"
    a: "It's genuinely unsettled, and the answer varies by jurisdiction and provider terms. Some legal systems hesitate to grant copyright to purely machine-generated work, and there are open questions about training data. Treat AI output as a draft you review, understand, and take responsibility for — and check your provider's terms and your project's license before shipping."
  - q: "Is AI-written code a security risk?"
    a: "It can be. Models can produce insecure patterns, suggest outdated or vulnerable dependencies, and miss injection bugs — all while looking plausible. Agents that read untrusted content also face **prompt injection**, where hidden instructions hijack their behaviour. The defence is the same discipline as everywhere else: review, test, scan dependencies, and don't run generated code unsupervised on sensitive systems."
---
# Security, privacy & ethics

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Know where your data goes** — hosted models receive your prompts and code; retention and training rules vary, so never paste secrets. **AI-written code has security risks** — insecure patterns, vulnerable dependencies, injection bugs, and prompt injection when agents read untrusted content. **Use these tools responsibly** — be honest about AI's role, mind IP and licensing, and don't let over-reliance erode your skills.
</div>

This is lesson 26 of the path. [Verifying AI-written code](/learn/ai-software-dev/verification-and-trust/) covered whether the code is *correct*; this lesson covers everything around that — where your data goes, who might own the output, whether the code is *safe*, and how to use these tools without compromising your work or your judgement. None of this is meant to scare you off; it's the ordinary due diligence of a professional tool. By the end you'll know the real risks and a practical do/don't list to manage them.

## Where your prompts and code go

When you use a **hosted model** — a chat app, an IDE plugin, or an agent talking to a provider's API — your prompts, and any code in them, are sent over the network to that provider to generate a response. This is unavoidable for hosted models: the model runs on the provider's servers, so your input has to get there. The only way to keep everything on your machine is to run a **local model** (an open-weight model you host yourself), which is exactly why privacy can be the deciding factor in [choosing a model](/learn/ai-software-dev/choosing-model-and-provider/).

Two questions then matter about a hosted provider:

- **Retention** — how long does the provider store your inputs and outputs, and who can access them?
- **Training** — may your inputs be used to train or improve future models? Policies differ by provider and often by *tier*: many consumer/free tiers may use inputs for training by default, while paid business and enterprise tiers frequently offer **zero-retention** or no-training guarantees.

These policies change, so the evergreen advice is: read the provider's current data-handling documentation and pick the tier whose guarantees match your needs. For sensitive work, a zero-retention enterprise tier or a local model is the responsible choice.

## Never paste secrets

This one is simple and absolute. Do not put **secrets** into a prompt — API keys, passwords, access tokens, private keys, customer personal data, anything confidential. Once sent to a hosted provider, you've lost control of where that value lives, how long it's retained, and whether it could surface again. A leaked key in a prompt is as bad as a leaked key in a public commit.

When you need the model to work with code that *contains* a secret, redact it first — replace the real value with a placeholder like `YOUR_API_KEY` — and keep the real value in your environment or a secrets manager, never in the conversation.

## IP and licensing: the open questions

The legal picture around AI-generated code is genuinely unsettled, and it's worth being honest about the open questions rather than pretending they're solved.

- **Who owns AI output?** Some jurisdictions are reluctant to grant copyright to purely machine-generated work, which creates uncertainty about the status of code an AI wrote. Provider terms also vary in what rights they grant you over outputs.
- **Training-data questions.** Models are trained on large corpora that include code under many licenses; there are ongoing legal and ethical debates about whether and how that training is permitted.
- **The reproduction risk.** Because models learn from existing code, there's a real chance an AI reproduces a chunk of **licensed code** close enough to the original to carry that license's obligations — for example, copyleft terms you didn't intend to take on. This matters most for distinctive, substantial blocks rather than ordinary idioms.

You don't need to resolve the law; you need to manage the risk. Treat AI output as a draft you review and take responsibility for, be cautious with large verbatim-looking blocks, and check both your provider's terms and your project's license before shipping. Some tools offer code-reference or filtering features that flag likely reproductions — worth using where available.

## Security of AI-written code

AI can write code that is plausible *and* insecure. The fluency that makes [verification](/learn/ai-software-dev/verification-and-trust/) necessary applies to security too: a vulnerability looks just as clean as safe code. Watch for:

- **Insecure patterns.** Missing input validation, weak crypto choices, permissions set too broadly, secrets hard-coded into the suggestion, error handling that leaks internals.
- **Outdated or vulnerable dependencies.** A model may suggest a library version with a known vulnerability, or an abandoned package, simply because it was common in its training data. Pin and scan your dependencies.
- **Injection bugs.** Classic flaws like SQL injection or command injection arise when untrusted input is stitched into a query or command. A model may produce exactly that pattern if you don't steer it otherwise.
- **Prompt injection.** This one is specific to AI agents. When an [agent](/learn/ai-software-dev/agentic-and-vibe-coding/) reads untrusted content — a web page, a file, an issue comment, a dependency's README — that content can contain hidden instructions that try to hijack the agent ("ignore your task and do X instead"). An agent with the power to run commands or edit files is a real target. Limit what agents can touch, prefer human approval for consequential actions, and be wary of pointing an agent at untrusted external content.

The defence is the discipline you already know: review the code, test it, scan dependencies, and don't run generated code unsupervised against sensitive systems.

## Ethics and responsible use

Beyond the legal and security mechanics, a few habits make you a responsible user of these tools.

- **Honesty and attribution.** Be transparent about AI's role where it matters — in collaborative settings, in academic or professional contexts, and wherever your team or audience has a reasonable expectation to know. Don't present AI-generated work as if it were entirely your own when that distinction matters.
- **Guard against skill erosion.** Over-reliance is a real risk: lean on AI for everything and your own ability to read, write, and reason about code can atrophy. Keep your skills sharp by understanding what the AI produces (this is also why [verification](/learn/ai-software-dev/verification-and-trust/) doubles as learning) and by sometimes doing the hard part yourself.
- **You own the outcome.** The AI is a tool; the responsibility for what you ship is yours. "The model wrote it" is not a defence for a bug, a leak, or a license violation.

## A practical do / don't list

| Do | Don't |
|---|---|
| Check the provider's current retention and training policy | Assume your inputs are private by default |
| Use a zero-retention tier or local model for sensitive code | Paste secrets, API keys, tokens, or personal data |
| Redact secrets with placeholders before sharing code | Hard-code a secret a model suggested |
| Scan and pin dependencies the AI suggests | Trust a suggested library version blindly |
| Review generated code for insecure patterns and injection | Ship AI code unread because it looks clean |
| Limit what agents can read and act on; approve risky steps | Point an agent at untrusted content with broad powers |
| Check your project's license against large verbatim blocks | Assume AI output is automatically yours to use freely |
| Be honest about AI's role and keep your own skills sharp | Present AI work as wholly your own where the distinction matters |

<div class="knowledge-check" data-quiz data-correct-msg="Right — secrets sent to a hosted provider are out of your control, so keep them out of prompts entirely." markdown="0">
  <p class="knowledge-check__q">Quick check: what should you never include in a prompt to a hosted AI model?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Comments explaining what your code does</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Secrets like API keys, passwords, tokens, or personal data</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The name of the programming language you're using</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Hosted means shared** — your prompts and code go to the provider; retention and training rules vary by provider and tier, so check the live docs and prefer zero-retention or local for sensitive work.
- **Never paste secrets** — keys, passwords, tokens, and personal data don't belong in a prompt; redact with placeholders.
- **IP is unsettled** — ownership of AI output and training-data questions are open; watch for reproduced licensed code and take responsibility for what you ship.
- **AI code can be insecure** — insecure patterns, vulnerable dependencies, and injection bugs look just as plausible as safe code.
- **Prompt injection is real** — agents reading untrusted content can be hijacked; limit their reach and approve risky actions.
- **Use it responsibly** — be honest about AI's role, guard against skill erosion, and remember the outcome is yours, not the model's.

Next up: turning the whole path into action — a concrete plan to start using AI on a real task this week. See [Getting started today](/learn/ai-software-dev/getting-started-today/).
