---
slug: agentic-cli-tools
title: Agentic & command-line tools
description: Agentic CLI tools like Claude Code, Codex CLI, and Aider run in your terminal and operate on the whole project. Learn what "agentic" means — the plan, act, observe loop — what it makes easy, and the guardrails you need against destructive commands and runaway cost.
keywords: agentic tools, Claude Code, Codex CLI, Aider, command line AI, terminal AI, plan act observe, agent loop, multi-file changes, permission prompts, sandboxing, version control, tokens cost
level: intermediate
status: full
prereq:
  - ide-integration
faq:
  - q: "What does \"agentic\" actually mean?"
    a: "An **agent** is an AI that uses tools in a loop — it plans, takes an action (like reading a file or running a command), observes the result, and decides what to do next. Instead of producing one answer, it works toward a goal over many steps with little hand-holding. That tool-using loop is what separates an agent from a plain chat model."
  - q: "Can an agentic tool delete my files or break things?"
    a: "Yes, which is exactly why guardrails matter. An agent that can run shell commands can in principle run destructive ones. Protect yourself with permission prompts (the tool asks before acting), sandboxing (limiting what it can touch), and version control so any change can be reviewed as a diff and reverted. Never run one on important work without those nets in place."
  - q: "Why can agentic tools get expensive?"
    a: "They make many model calls as they loop — reading files, running tests, reading the output, trying again — and each call consumes tokens you pay for. A tight, well-scoped task is cheap; an agent stuck looping on a hard problem can burn a lot. Watch the cost, scope tasks narrowly, and stop a run that's spinning its wheels."
---
# Agentic & command-line tools

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Agentic means a loop** — the model plans, acts with tools, observes the result, and iterates toward a goal. **Whole-project reach** — these terminal tools read your repo, run commands and tests, and edit many files end to end. **Guardrails required** — they can run destructive commands and burn tokens looping, so you need permission prompts, sandboxing, version control, and a careful review of every diff.
</div>

This is lesson 13 of the path. The [previous lesson](/learn/ai-software-dev/ide-integration/) put AI into your editor, where it can see your files but still waits for you to accept each suggestion. This lesson goes a step further: tools that act on your whole project largely on their own. By the end you'll understand what "agentic" means, what these command-line tools make possible, and — just as important — the guardrails that keep that power from turning against you.

## What "agentic" means

Most of the tools so far have been *reactive*: you ask, the model answers once, and you decide what to do with the answer. An **agent** is different. It works in a loop:

1. **Plan** — decide what to do next toward the goal.
2. **Act** — use a *tool*: read a file, edit a file, run a shell command, run the test suite.
3. **Observe** — read the result of that action (the file's contents, the command's output, the test failures).

Then it loops back to plan again, using what it just observed. This **plan → act → observe** cycle is the heart of what "agentic" means. The model isn't producing one block of text; it's pursuing a goal over many steps, choosing its own next action each time, with little hand-holding from you.

The key enabler is **tools** — functions the model is allowed to call. Reading a file, writing a file, and running a command are tools. Given those, an agent can explore an unfamiliar codebase, make a change, check whether it worked, and fix it if it didn't — much like a developer would, just faster and tirelessly.

## Agentic tools live in your terminal

These tools typically run in your **terminal** (the command-line shell) and operate on the **whole project** — the entire repository, not just the file you have open. Common examples include Anthropic's **Claude Code**, OpenAI's **Codex CLI**, and **Aider**. They differ in details, but the shape is the same: you describe a goal in plain language, and the agent works through the repo to accomplish it.

Because it has the project in reach and can run commands, an agentic tool shines at work that the chat app and editor completion struggle with:

| Task | Why an agent fits |
|------|-------------------|
| **Multi-file changes** | It can edit several files in one coherent pass and keep them consistent |
| **Building a feature** | It can scaffold, wire up, and test new code across the project |
| **Fixing failing tests** | It can run the tests, read the failures, edit, and re-run until green |
| **Exploring an unfamiliar repo** | It can read its way around to answer "where does X happen?" |

## A GopherTrunk example: add a decoder feature and make it green

Picture pointing an agent at the GopherTrunk repository with a goal: "Add a small field to the control-channel decoder that records the last sync timestamp, and add a test for it."

An agent can carry that out as a loop. It reads the relevant decoder files to learn the existing structure. It edits the decoder to add the field. It writes a test. Then it runs the project's check command:

```bash
make vet test
```

It reads the output. If `vet` flags an unused variable or a test fails, it sees that in the observe step, edits the code, and runs `make vet test` again — repeating until the suite is green. That iterate-until-green ability is exactly what makes agents feel different: the feedback loop a developer would run by hand happens automatically. (GopherTrunk's own [CLAUDE.md](/learn/intro-software-dev/testing/) guidance leans on this — "must be green before any commit.")

What you did was state the goal and, crucially, *review the result*. What the agent did was the legwork in between.

## The drawbacks, and the guardrails you need

This is the most powerful tool in the module, and also the one that demands the most care. The same capabilities that make agents useful make them risky.

**It can run destructive commands.** An agent that can run shell commands can, in principle, run a command that deletes files, rewrites git history, or changes things outside your project. It won't usually *want* to, but a misunderstanding or a bad plan can cause real damage. Treat the ability to run commands with the seriousness it deserves.

**It can make sweeping changes.** Because it edits across the whole project, an agent can touch far more than you expected. A small request can balloon into edits in a dozen files if the goal was ambiguous or the agent over-interpreted it. Scope tasks narrowly and keep them reviewable.

**It can burn tokens looping (cost).** Every step is model calls, and model calls consume tokens you pay for (see [Understanding cost](/learn/ai-software-dev/understanding-cost/)). A clean task is cheap. But an agent stuck on a hard problem can loop — try, fail, try again — and the cost adds up quickly. Watch long runs and stop one that's clearly spinning its wheels.

**You must review the diff.** None of this is safe to merge unread. The agent produces a **diff** — the set of changes — and reviewing it is your job, the same way you'd review a colleague's pull request (see [Version control and collaboration](/learn/intro-software-dev/version-control-collab/)). The model predicts plausible code; plausible is not verified, and we'll come back to that hard rule in [Verification and trust](/learn/ai-software-dev/verification-and-trust/).

Given those risks, three guardrails are essentially mandatory:

- **Permission prompts.** Configure the tool to *ask before acting* — especially before running commands or editing files. A prompt before each risky action is your chance to say no.
- **Sandboxing.** Run the agent in an environment where it can only touch what it should — a container, a scoped working directory, restricted command access. If it can't reach your whole system, a mistake stays contained.
- **Version control.** Commit a clean baseline before you start. With git as a safety net, any change the agent makes can be inspected as a diff and reverted instantly. This is the single most important habit: never let an agent loose on uncommitted work you can't get back.

Used with those nets, an agent is a remarkably effective collaborator. Used without them, it's an automated way to make a large mess quickly.

## When it's the right tool

Reach for an agentic tool when the task is genuinely *project-scale*:

- A change that spans several files and needs to stay consistent.
- Building or scaffolding a feature end to end.
- Fixing a batch of failing tests by iterating until they pass.
- Exploring a large, unfamiliar codebase to answer a question.

And reach for something lighter when the task is small: a quick question goes to the [chat app](/learn/ai-software-dev/the-provider-app/); a single in-file edit fits [editor AI](/learn/ai-software-dev/ide-integration/). Bringing an agent to a one-line fix is overkill — and a needless place for it to do something unexpected.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an agent runs a plan, act, observe loop, using tools to read, run, and edit until the goal is met." markdown="0">
  <p class="knowledge-check__q">Quick check: What makes a tool "agentic" rather than a plain chat model?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It runs in the terminal instead of a browser</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It uses tools in a plan, act, observe loop — reading, running, and editing to pursue a goal over many steps</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It never needs you to review its changes because it tests its own work</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Agentic tools** — terminal tools like Claude Code, Codex CLI, and Aider that operate on the whole project.
- **The agent loop** — they plan, act with tools, observe the result, and iterate toward a goal with little hand-holding.
- **Whole-project reach** — they can read the repo, run commands and tests, and make consistent multi-file changes end to end.
- **Real risks** — they can run destructive commands, make sweeping edits, and burn tokens looping on hard problems.
- **Guardrails are mandatory** — permission prompts, sandboxing, and version control turn that power into something safe to use.
- **Always review the diff** — the agent's output is unverified until you read it, just like a colleague's pull request.

Next up: now that you've met the app, the editor, and the agent, how do you decide which to use — and is it really one or the other? See [App, IDE, agent — or all three?](/learn/ai-software-dev/app-vs-ide-vs-both/).
