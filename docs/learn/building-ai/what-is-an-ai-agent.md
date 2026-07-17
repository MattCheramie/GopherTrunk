---
slug: what-is-an-ai-agent
title: Agents — letting the model act in a loop
description: An AI agent is a model running in a loop with tools — it plans a step, acts by calling a tool, observes the result, and decides what to do next until the task is done. Learn what agents unlock, where the risk climbs, and how to keep them contained.
keywords: AI agent, agent loop, plan act observe, autonomous, tool use, multi-step, guardrails, agentic, max steps, human approval
level: advanced
status: full
prereq:
  - tool-use-and-function-calling
faq:
  - q: "What makes something an agent?"
    a: "It runs a **loop**. A plain feature makes one request and takes one answer; an agent lets the model **plan a step, act by calling a tool, observe the result, and decide what to do next** — repeating until the task is done. The loop and the tools are what turn a chatbot into an agent."
  - q: "Does every AI feature need an agent?"
    a: "No — and most don't. If you can lay out the steps in advance, a fixed sequence of calls (or one [structured call](/learn/building-ai/structured-output/)) is simpler, cheaper, and easier to trust. Reach for a full agent loop only when the steps genuinely can't be predetermined."
  - q: "Why are agents riskier than a single call?"
    a: "Autonomy multiplies failure modes. A single call gives one wrong answer; a loop can take a wrong **action**, spin forever, run up cost, or be steered by [malicious input](/learn/building-ai/prompt-injection-and-security/) — each iteration compounding the last. That's why agents need guardrails a one-shot call doesn't."
  - q: "How do I stop an agent from running away?"
    a: "Cap it: a maximum number of steps, a budget, an allow-list of tools it may call, and human approval before anything risky or irreversible. Give it a clear stopping condition so it knows when it's done rather than looping on."
---

# Agents — letting the model act in a loop

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **agent** is the model running in a **loop** with [tools](/learn/building-ai/tool-use-and-function-calling/),
driving a multi-step task itself rather than answering once. Each turn it runs a
**plan–act–observe loop**: think about the next step, act by calling a tool,
observe the result, decide what to do next — until the goal is met. That autonomy
is the power *and* the danger: **autonomy = risk**, because a loop can take wrong
actions, spin forever, or run up cost. Give it guardrails, and reach for a full
agent only when the steps can't be laid out in advance. You've met the hands-off
version of this while [agentic coding](/learn/ai-software-dev/agentic-and-vibe-coding/).
</div>

You've already seen a model *request* a tool call and hand control back to your
code. An agent is what you get when you put that exchange in a loop and point it
at a goal. Instead of you deciding each next step, the model does — and keeps
going until the task is finished. This lesson is about that shift: what it
unlocks, where it gets dangerous, and how to keep it contained.

## From one call to a loop

A plain AI feature is one round trip: a request goes out, an answer comes back,
you're done. Summarise this, classify that, extract these fields — one call, one
result.

An **agent** replaces that single exchange with a repeating cycle:

1. **Plan** — the model reasons about what to do next given the goal so far.
2. **Act** — it calls a [tool](/learn/building-ai/tool-use-and-function-calling/):
   a search, a file read, a calculation, an action in the world.
3. **Observe** — your code runs the tool and hands the result back.
4. **Decide** — the model reads that result and either finishes or loops to step 1
   for the next move.

This builds directly on [tool use](/learn/building-ai/tool-use-and-function-calling/):
the request → execute → return cycle is a single turn of the loop. Wrap that
cycle so the model can keep calling tools until it decides it's done, and you have
an agent. Nothing more exotic is going on — an agent is little more than this loop
pointed at a goal.

```text
goal ─► [ plan ─► act (call a tool) ─► observe result ─► decide ] ─► finish
              ▲                                            │
              └──────────────── loop ──────────────────────┘
```

## What agents unlock

The loop lets the model handle tasks it can't do in a single reply — ones where
the *right* next step depends on what the last step turned up. The model sequences
those steps itself:

- **Research** — search, read what comes back, search again to fill the gaps, then
  synthesise an answer.
- **Triage** — read an incoming report, pull related records, and route or label
  it based on what it finds.
- **Fix-and-verify** — make a change, run the check, read the failure, and try
  again until it passes.
- **Orchestration** — drive several tools in sequence to complete a workflow no
  single call could.

You've already used agents like this from the outside: the [agentic CLI tools](/learn/ai-software-dev/agentic-cli-tools/)
and [agentic coding](/learn/ai-software-dev/agentic-and-vibe-coding/) covered
earlier are exactly this loop — plan, edit, run the tests, read the errors,
iterate — wrapped around a coding goal. Now you're looking at the machinery that
makes that behaviour work, so you can build it yourself.

## Where the risk climbs

Autonomy is the payoff, and it's also the hazard. A single call can only give you
a wrong *answer*; a loop can take a wrong *action* and then build on it. The
failure modes multiply:

- **Wrong actions.** The model acts on a mistaken conclusion — deletes the wrong
  record, sends the wrong message — and the next step compounds it.
- **Infinite loops.** It gets stuck retrying the same failing step, never deciding
  it's done.
- **Runaway cost.** Every iteration is more calls and more tokens; an unbounded
  loop is an unbounded bill (see [cost, latency & limits](/learn/building-ai/cost-latency-and-limits/)).
- **Acting on malicious input.** Content the agent reads can carry instructions
  that hijack its next action — [prompt injection](/learn/building-ai/prompt-injection-and-security/)
  turns "read this page" into "and now do what the page says."

And because the model stays a probabilistic component, its mistakes don't stop at
the first step — the loop carries them forward (see [handling hallucinations & failure](/learn/building-ai/handling-hallucinations-and-failure/)).

## Guardrails

You contain an agent by bounding what its loop can do. The essentials:

- **Step / iteration limits.** A hard cap on how many times the loop may run, so a
  stuck agent stops instead of spinning.
- **Budgets.** A ceiling on tokens, calls, or spend — the loop halts when it's
  reached.
- **Allow-listed tools.** The agent may only call tools you've explicitly granted,
  each with the narrowest access it needs.
- **Human approval.** For anything risky or irreversible — spending, sending,
  deleting — require a person to confirm before the action runs.
- **Clear stopping conditions.** Tell the agent what "done" looks like, so it ends
  on success rather than looping for more.

We'll get concrete about wiring tools to an agent safely — and the protocols for
it — in [giving agents tools (and MCP)](/learn/building-ai/giving-agents-tools-and-mcp/).

## Keep it as simple as the task allows

An agent loop is the most powerful option and the hardest to control, so it should
be your *last* resort, not your first. Many problems that sound like they need an
agent don't:

- If you already know the steps, write them as a **fixed sequence of calls**. It's
  cheaper, predictable, and easy to debug.
- If you need one structured result, make a **single [structured call](/learn/building-ai/structured-output/)**.
- Only when the steps genuinely **can't be predetermined** — when the right next
  move depends on what earlier moves reveal — does a full agent loop earn its
  keep.

The loop buys you flexibility, and you pay for it in cost, latency, and
unpredictability. Don't pay unless the task needs what you're buying.

## A small example

Picture a small research agent that answers a question it can't answer from
memory. It has two tools — `search(query)` and `read(url)` — and a **max of five
steps**:

```text
goal: "Which GopherTrunk lesson covers setting SDR gain?"

step 1  plan  → "I need to find the lesson."
        act   → search("GopherTrunk gain lesson")
        obs   → [list of candidate pages]
step 2  plan  → "The gain-and-agc page looks right; read it."
        act   → read("/learn/rf-sdr/gain-and-agc/")
        obs   → "…highest gain that keeps the strongest signal below 0 dBFS…"
step 3  plan  → "I have enough to answer."
        finish → "The 'Gain, AGC & avoiding overload' lesson covers it."
```

The model chose each step from what the previous one returned — search, then read
the best hit, then decide it had enough. The **step cap** is the safety net: if it
never found a good answer, the loop would stop at five rather than searching
forever. Swap in real tools and a real goal and every agent is a variation on this
shape.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an agent runs a plan–act–observe loop with tools instead of a single request/response." markdown="0">
  <p class="knowledge-check__q">Quick check: what distinguishes an agent from a plain model call?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It uses a bigger, more capable model</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It runs a plan–act–observe loop with tools, not a single request/response</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It never needs a human to approve anything</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **agent** is the model running in a **loop** with tools, driving a multi-step
  task itself instead of answering once.
- Each turn is a **plan–act–observe loop**: plan the next step, act by calling a
  tool, observe the result, decide what's next — until it's done.
- Agents unlock tasks the model **sequences itself** — research, triage,
  fix-and-verify, orchestration — the same loop behind agentic coding tools.
- **Autonomy = risk**: wrong actions, infinite loops, runaway cost, and acting on
  malicious input all get worse across iterations.
- **Guardrails** contain it — step limits, budgets, allow-listed tools, human
  approval for risky actions, and clear stopping conditions.
- **Keep it simple**: prefer a fixed sequence or a single structured call; reach
  for a full agent only when the steps can't be predetermined.

Next up: [giving agents tools (and MCP)](/learn/building-ai/giving-agents-tools-and-mcp/).
