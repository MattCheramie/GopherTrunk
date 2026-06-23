---
slug: agentic-and-vibe-coding
title: Agentic coding & "vibe coding"
description: Agentic coding hands the model the wheel — you state a goal and an agent plans, edits files, runs tests, and iterates. Learn what "vibe coding" means, its real upside, and where it's reasonable versus dangerous.
keywords: agentic coding, vibe coding, AI agent, autonomous coding, Andrej Karpathy, AI productivity, code you don't understand, prototyping, technical debt, over-trust, agentic CLI
level: intermediate
prereq:
  - chat-assisted-coding
faq:
  - q: "What is agentic coding?"
    a: "You state a **goal** and an agent does the multi-step work to reach it: it plans, edits multiple files, runs commands and tests, reads the results, and iterates — with minimal intervention from you. It's the most hands-off mode, the opposite end from inline [completion](/learn/ai-software-dev/code-completion/)."
  - q: "What does \"vibe coding\" mean?"
    a: "**Vibe coding** is describing what you want in natural language and largely accepting what comes back *without closely reading the code*. The term was popularized by Andrej Karpathy in early 2025. It can be liberating for low-stakes work and reckless for anything you must trust or maintain — the difference is whether the lack of review matters."
  - q: "Is vibe coding bad?"
    a: "Not inherently — it's a tool with a place. It's reasonable for prototypes, throwaway scripts, learning, and other low-stakes work where being wrong is cheap. It's dangerous for production, security-sensitive code, or anything you'll have to maintain, because you end up owning code you don't understand. Match the method to the stakes."
---
# Agentic coding & "vibe coding"

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Agentic coding** — you state a goal and an agent plans, edits many files, runs tests, and iterates on its own. **Vibe coding** — describing what you want and largely accepting the result without closely reading it. **Match it to the stakes** — a higher productivity ceiling comes with real risks, so it suits prototypes far better than production.
</div>

This is lesson 17 of the path, and the third mode of AI-assisted coding. The previous two — [completion](/learn/ai-software-dev/code-completion/) and [chat](/learn/ai-software-dev/chat-assisted-coding/) — kept you firmly in the driver's seat. This one hands over the wheel. By the end you'll understand what agentic coding actually does, what "vibe coding" means and where the term came from, the genuine productivity it unlocks, and — stated honestly — the risks that come with it and when they're worth taking.

## Handing the model the wheel

In [chat-assisted coding](/learn/ai-software-dev/chat-assisted-coding/), you asked one question and got one answer to review. **Agentic coding** changes the unit of work from a question to a *goal*. You say what you want accomplished — "add NXDN support to the decoder," "make the failing tests pass," "build me a small dashboard for the scan log" — and an **agent** carries out the multi-step work to get there.

The word "agent" here means a model wrapped in a loop that can *act*, not just talk. We met these in [Agentic CLI tools](/learn/ai-software-dev/agentic-cli-tools/); the behaviour is what matters now. A coding agent typically:

1. **Plans** — breaks the goal into steps.
2. **Edits** — creates and changes files across the project, not just one snippet.
3. **Runs commands** — executes the build, the tests, a linter, a script.
4. **Reads the results** — sees the compiler errors or test failures.
5. **Iterates** — fixes what broke and tries again, looping until it believes the goal is met.

All of this happens with **minimal intervention**. You might approve a step or answer a question, but you're not writing or reviewing each line as it goes. The agent is doing the typing, the running, and the debugging.

This is the high-delegation end of the spectrum, and it's genuinely powerful. For a project like GopherTrunk, an agent given "scaffold a plugin that decodes mode X following the structure of the existing P25 plugin" can read the existing plugins, mirror their layout, wire up the registration, generate a first pass of the decoder, and run the test suite — work that would take you a chunk of an afternoon — and present you with something that compiles. The leverage is real.

## What "vibe coding" means

**Vibe coding** is a specific way of using agentic (or even chat) tools: you describe what you want in natural language and *largely accept what comes back without closely reading the code*. You're steering by results and feel — does the app do the thing? — rather than by understanding each change. As its originator put it, you "give in to the vibes" and "forget that the code even exists."

The term was popularized by **Andrej Karpathy** in early 2025, describing exactly this style: leaning fully into the model, accepting diffs without reading them, and pasting errors back in until things work. It named something many people were already doing and gave the industry a shared word for it.

It's worth being precise about what makes vibe coding distinct. It is not simply *using an agent* — you can run an agent and still carefully review everything it produces, which is the disciplined, human-in-the-loop approach. Vibe coding is specifically the choice to *not* closely review. That choice is the whole story: it's what makes vibe coding fast, and it's also what makes it risky.

## The upside: a higher ceiling

The honest case for these modes is that they raise your **productivity ceiling** dramatically. When you don't have to type every line or review every change, you can move at the speed of *describing* rather than the speed of *implementing*. Whole features, prototypes, and small tools that used to be a day's work can come together in an hour.

That speed is most transformative when:

- You're exploring an idea and want to *see* it running before deciding if it's worth doing properly.
- You're in unfamiliar territory and the agent can stitch together patterns faster than you'd learn them.
- The task is large but routine — lots of similar files, plumbing, scaffolding.
- Getting *something* working is more valuable right now than getting it perfect.

For prototyping especially, vibe coding can be liberating. The point of a prototype is to learn quickly whether an idea has legs; reading every line of code you might throw away tomorrow would defeat the purpose. Used this way, it's not sloppy — it's appropriately matched to the goal.

## The risks, stated honestly

The same property that gives vibe coding its speed — not reading the code — is the source of every risk, so it's worth naming them plainly rather than waving them away.

- **Code you don't understand.** When the agent finishes, you own a body of code nobody on your side has read. The moment it breaks or needs changing, you're debugging a stranger's work — and that stranger isn't available to explain it.
- **Subtle bugs.** Agents produce code that *runs*, which is not the same as code that's *correct*. An off-by-one, a mishandled edge case, a race condition — these pass the "does it work?" glance and surface later, often in production.
- **Security holes.** Unread code can quietly skip input validation, log secrets, build SQL by string concatenation, or pull in a dependency you never vetted. Security flaws rarely announce themselves by failing a demo. We cover this in depth in [Security, privacy & ethics](/learn/ai-software-dev/security-privacy-ethics/).
- **Brittle, "works until it doesn't" results.** Code assembled without a coherent design can hold together for the happy path and collapse the moment reality deviates — and because no one understands it, the collapse is hard to diagnose.
- **Harder maintenance.** Software is read far more often than it's written. Code generated fast and reviewed never tends to lack the structure, naming, and consistency ([clean code](/learn/intro-software-dev/clean-code/)) that makes future changes safe.
- **Over-trust.** Fluent output and a green test run breed confidence the code may not deserve. The most dangerous moment is when it works so smoothly that you stop questioning it.

None of these is a reason to never use agents. They're a reason to be clear-eyed: the cost of skipping review is paid *later*, and it's paid in understanding, reliability, and security.

## When it's reasonable, and when it's dangerous

The useful question is never "is vibe coding good or bad?" but "does the lack of review *matter here*?" That depends on the stakes.

| Setting | Vibe coding verdict | Why |
|---------|---------------------|-----|
| **Throwaway script** | Reasonable | If it's wrong you rerun it; nobody else depends on it. |
| **Prototype / spike** | Reasonable | The goal is to learn fast; the code may be discarded anyway. |
| **Learning / exploring** | Reasonable | You're seeing what's possible, not shipping it. |
| **Internal low-stakes tool** | Use with care | Fine if a failure is mildly annoying, not costly. |
| **Code you must maintain** | Dangerous | You'll inherit code you don't understand the moment it needs to change. |
| **Production systems** | Dangerous | Real users feel the subtle bugs and brittleness. |
| **Security-sensitive code** | Dangerous | Unreviewed code is exactly where quiet vulnerabilities live. |

A simple lens: the more **reversible** the change and the **lower the stakes**, the more delegation you can safely accept. A prototype you'll delete is fully reversible; a payment flow in production is not. For GopherTrunk, vibe coding a quick script to plot some capture statistics is fine. Vibe coding the core demodulation path — where a subtle error silently corrupts every decode and nobody can read the code to find out why — is asking for trouble.

The balanced position is this: agentic and vibe coding are real tools that genuinely expand what you can build, and refusing to use them costs you speed. But the speed comes from skipping review, and skipping review has a price that's invisible at first and steep later. Use the modes deliberately, scaled to the stakes — which is exactly the judgment the [next lesson](/learn/ai-software-dev/the-spectrum/) is about.

<div class="knowledge-check" data-quiz data-correct-msg="Right — vibe coding means accepting the model's output without closely reviewing the code, which suits low-stakes work." markdown="0">
  <p class="knowledge-check__q">Quick check: Which describes "vibe coding"?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Carefully reviewing every line an agent writes before accepting it</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Describing what you want and largely accepting the result without closely reading the code</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Writing all the code yourself and using AI only for autocomplete</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Agentic coding** — you state a goal and an agent plans, edits many files, runs commands and tests, and iterates with minimal intervention.
- **Vibe coding** — describing what you want and largely accepting the output without closely reading it; popularized by Andrej Karpathy in early 2025.
- **Higher ceiling** — the upside is real: features and prototypes built far faster than by hand.
- **Honest risks** — code you don't understand, subtle bugs, security holes, brittleness, harder maintenance, and over-trust all stem from skipping review.
- **Match the stakes** — reasonable for prototypes, throwaway scripts, and learning; dangerous for production, security-sensitive, or must-maintain code.
- **Reversibility is the lens** — the more reversible and low-stakes the change, the more delegation you can safely accept.

Next up: how completion, chat, agentic, and vibe coding fit together as one spectrum of delegation — and how to choose where to work for each task — see [Choosing where on the spectrum to work](/learn/ai-software-dev/the-spectrum/).
