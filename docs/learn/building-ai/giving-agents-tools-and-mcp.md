---
slug: giving-agents-tools-and-mcp
title: Giving agents tools (and MCP)
description: Tools are how an agent reads data, calls APIs, and takes action — so designing them clearly and safely is most of agent engineering. Learn good tool design, tool safety and least privilege, and what the Model Context Protocol (MCP) standardises.
keywords: agent tools, tool design, MCP, Model Context Protocol, tool safety, least privilege, tool schema, integrations, sandboxing, human confirmation
level: advanced
status: full
prereq:
  - what-is-an-ai-agent
faq:
  - q: "What is MCP?"
    a: "**MCP**, the Model Context Protocol, is an open standard for exposing tools, data, and resources to models through a common interface. You write an MCP *server* once, and any MCP-compatible client or app can use its tools — instead of re-wiring the same integration separately for every app. It's provider-neutral: the idea, not one vendor's product."
  - q: "How do I make tools safe?"
    a: "Treat every tool-call request as untrusted input, because the model can be steered by content it reads. **Validate the arguments**, apply **least privilege** (scoped credentials, read-only where you can), **sandbox** any code execution, **rate-limit** calls, and **require human confirmation** for anything destructive or irreversible. The safety lives in your code, which runs the call — not in the model."
  - q: "Should I give an agent lots of tools?"
    a: "No — fewer, well-scoped tools beat many broad ones. Every tool is attack surface and another way for the model to pick wrong. A small, sharp tool set is easier for the model to choose from correctly and easier for you to reason about and secure."
  - q: "Do agent tools have to use MCP?"
    a: "No. Tools are just functions you describe to the model via [tool use](/learn/building-ai/tool-use-and-function-calling/); you can wire them directly in your own code. MCP is a *convention* for exposing them so they're reusable across apps. Use it when you want the same integration available to more than one client; skip it for a one-off, in-process tool."
---

# Giving agents tools (and MCP)

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Tools** are how an [agent](/learn/building-ai/what-is-an-ai-agent/) affects the
world — read data, call an API, write a change, run code. Without them an agent
can only talk. So designing tools well, and safely, is most of agent
engineering. Give each tool a clear name and a tight schema, treat its arguments
as untrusted, and apply **least privilege**. **MCP** — the Model Context Protocol
— is an open standard for exposing tools so one integration works across many
apps. Built on plain [tool use](/learn/building-ai/tool-use-and-function-calling/).
</div>

An agent is a loop: the model plans, calls a tool, sees the result, and plans
again. Everything it can actually *do* in that loop is defined by the tools you
give it. That makes tools the highest-leverage — and highest-risk — part of the
whole design. This lesson is about building them so the agent is both capable and
safe.

## Tools are the agent's hands

Strip the tools away and even the best model can only produce text. Tools are how
it reaches past the conversation into the real world. In practice they fall into a
few kinds:

- **Reading data** — look up a record, query a database, fetch a page.
- **Calling APIs** — hit an external service and bring back its answer.
- **Writing or updating** — create a file, change a row, post a message.
- **Running code** — execute a script or command and use its output.

The first two are how an agent *learns*; the last two are how it *acts*. An agent
with only read tools can investigate but never change anything — often exactly
what you want. The moment you add a write or execute tool, the stakes change, and
so does how carefully you design it.

## Designing good tools

The model chooses which tool to call, and with what arguments, from what you tell
it. The description and schema *are* the interface — so treat them like one:

- **A clear name and description.** The model reads these to decide when the tool
  fits. `search_session_logs` with a description of what it searches and when to
  use it beats a vague `query` the model has to guess about.
- **A tight parameter schema.** Constrain arguments with types, enums, and
  required fields — the same [structured output](/learn/building-ai/structured-output/)
  mechanism that carries tool arguments. Less room to fill them in wrongly.
- **One job per tool.** A tool that does a single, well-named thing is easier for
  the model to reach for than a Swiss-army function behind a mode flag.
- **Useful error messages.** When a call fails, return something the model can
  recover from. `"unknown talkgroup: 9999"` lets it correct itself on the next
  step; a bare `error` leaves it guessing.

Good tool design is mostly making the *right* call obvious and the *wrong* call
hard.

## Tool safety

Here is the rule that governs everything: **a tool-call request is untrusted
input.** The model decided to make it, but the model can be steered by whatever it
has read — a web page, a document, a user message. If any of that came from users
or the internet, the request could be doing someone else's bidding (see
[prompt injection & security](/learn/building-ai/prompt-injection-and-security/)).
Because *your* code runs the call, your code is where safety lives:

- **Validate the arguments.** The schema guarantees the shape, not the values. A
  requested id can be out of range, nonsensical, or hostile — check before acting.
- **Apply least privilege.** Give each tool the narrowest access that does its
  job: scoped credentials, read-only where possible, no blanket admin keys.
- **Sandbox code execution.** If a tool runs code or shell commands, run it in an
  isolated environment with no access to anything it doesn't need.
- **Rate-limit.** Cap how often and how fast tools can fire, so a runaway loop
  can't rack up cost or hammer a downstream service.
- **Require human confirmation** for anything destructive or irreversible —
  deleting, sending, spending, publishing. Ask a person before the agent acts.

None of this is about distrusting the model in particular; it's ordinary
defensive engineering against an input you didn't fully control.

## MCP — a standard for tools

Wiring tools directly into one program is fine, but you often want the *same*
tools available to several apps — a chat client, an IDE, a background agent. The
**Model Context Protocol (MCP)** is an open standard for exactly that: a common
interface for exposing tools, data, and resources to models.

The shape is simple. You write an MCP **server** that exposes a capability — a
data source, an API, a set of actions — once. Any MCP-compatible **client** or app
can then use it, without you re-implementing the integration per app. Instead of
N apps each needing custom glue for the same source, you build the server once and
plug it in everywhere.

MCP is provider-neutral — an open protocol, not one company's feature. It's the
same idea that lets a single assistant reach many capabilities: define the
connection once, reuse it across clients.

## An example tool set

Picture a read-only assistant for a scanner project. A small, sharp tool set might
be:

- `lookup_talkgroup(id)` — return the label and system for a talkgroup id.
- `search_session_logs(query)` — find matching lines in recorded session logs.
- `get_system_status()` — report what the scanner is currently doing.

Every one is read-only and tightly scoped — the assistant can *answer* questions
but can't change the scanner or delete anything. One schema, tight and typed:

```json
{
  "name": "lookup_talkgroup",
  "description": "Return the label and trunked system for a talkgroup id.",
  "parameters": {
    "type": "object",
    "properties": {
      "id": { "type": "integer", "minimum": 1 }
    },
    "required": ["id"]
  }
}
```

If you later add a tool that *changes* something — say `mute_talkgroup(id)` —
that's the one to guard with confirmation and a scoped credential, because it
crosses from reading into acting.

## Keep the surface small

It's tempting to hand an agent every tool you can think of. Resist it. **Fewer,
well-scoped tools beat many broad ones.** A big tool set makes the model's choice
harder — more chances to pick the wrong one or fill in the wrong arguments — and
every tool you add is more attack surface and another way for things to go wrong.
Prefer a handful of sharp, single-purpose tools over a sprawling toolbox, and add
a new one only when a real task proves you need it. A small surface is easier for
the model to use correctly and easier for you to secure.

<div class="knowledge-check" data-quiz data-correct-msg="Right — MCP standardises how tools and data are exposed, so one server works across any compatible app." markdown="0">
  <p class="knowledge-check__q">Quick check: what does MCP standardise?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The internal weights and architecture of the model</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">How tools and data are exposed to models, so one tool server works across compatible apps</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The exact wording of the system prompt every agent must use</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Tools are the agent's hands** — reading data, calling APIs, writing changes,
  running code. Without tools an agent can only talk.
- **Design for the model** — a clear name and description, a tight parameter
  schema, one job per tool, and error messages it can recover from.
- **A tool-call request is untrusted input** — validate arguments, apply least
  privilege, sandbox code, rate-limit, and confirm destructive actions.
- **MCP standardises tool exposure** — write an MCP server once and any compatible
  client can use it, instead of re-wiring integrations per app.
- **Keep the surface small** — fewer, well-scoped tools beat many broad ones;
  every tool is attack surface and a way to go wrong.

Next up: [evaluating AI features](/learn/building-ai/evaluating-ai-features/).
