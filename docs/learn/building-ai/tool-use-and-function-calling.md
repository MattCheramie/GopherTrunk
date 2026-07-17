---
slug: tool-use-and-function-calling
title: Tool use & function calling
description: Tool use (function calling) lets you describe functions a model can ask to call, so it can reach live data, do exact math, or take actions it otherwise can't. Learn the request/execute/return flow and why your code always runs the call.
keywords: tool use, function calling, tools, model calling functions, JSON arguments, agent tools, tool schema, structured arguments, request execute return, least privilege
level: intermediate
status: full
prereq:
  - structured-output
faq:
  - q: "What is tool use, or function calling?"
    a: "It's a mode where you describe some functions — each with a name, a description, and a parameter schema — and the model can respond by **requesting a call** to one, with arguments, instead of replying in prose. It's how a model reaches abilities it lacks on its own: live data, exact math, or taking an action."
  - q: "Does the model actually run my function?"
    a: "No. The model only **requests** a call and supplies arguments; your code decides whether and how to run it, then returns the result. That separation is the whole safety story — validate the arguments, apply least privilege, and guard anything destructive before you execute."
  - q: "How do the arguments come back?"
    a: "As schema-constrained JSON — the same mechanism as [structured output](/learn/building-ai/structured-output/). The model fills in the parameter object you described. You still validate it: the shape is guaranteed, the values are not."
  - q: "What makes a tool easy for the model to use well?"
    a: "A clear description (the model reads it to decide when to call), a tight parameter schema, and one job per tool. Return helpful errors too — a message like \"unknown talkgroup\" lets the model correct itself on the next step."
---

# Tool use & function calling

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Tool use** (also called **function calling**) lets you describe functions the
model is allowed to ask for — and then **the model requests, you execute**. Given
a list of tools, instead of answering in prose the model can emit a request to
call one with arguments, gaining abilities it otherwise lacks: live data, exact
math, or taking an action in the world. The arguments come back as
schema-constrained JSON — [structured output](/learn/building-ai/structured-output/)
under the hood. Wire the request/execute/return cycle into a loop and you have the
seed of an [agent](/learn/building-ai/what-is-an-ai-agent/).
</div>

A model on its own only produces text. It doesn't know today's date, can't look
up a record in your database, can't reliably multiply two large numbers, and
can't send an email. Tool use is how you hand it those abilities — safely, and on
your terms. You describe what functions exist; the model decides when one would
help and asks you to run it.

## What tool use is

You provide the model with a **list of tools**. Each tool is three things: a
**name**, a **description** of what it does, and a **parameter schema** — a JSON
Schema describing the arguments it takes. That's the whole contract.

With tools available, the model has a new option. Instead of replying in prose,
it can emit a **request to call a tool**, naming which one and supplying the
arguments. It's saying, in effect, "to answer this, I need you to run
`lookup_frequency` with these values." Nothing has happened yet — that's a
request, not an action.

## The flow

Tool use is a short cycle, not a single reply:

1. **Define tools.** You send the model the list of tools it may use.
2. **The model chooses.** Given the conversation, it either answers directly or
   requests a tool call with arguments.
3. **Your code executes.** *You* run the function — query the database, call the
   clock, do the math — with whatever safeguards you decide.
4. **You return the result.** You hand the tool's output back to the model.
5. **The model continues.** It uses that result to answer — or requests another
   tool call, and the cycle repeats.

That **request → execute → return** cycle is the whole mechanism. Run it in a
loop — where the model can keep calling tools until it's done — and you've built
the seed of an [AI agent](/learn/building-ai/what-is-an-ai-agent/). An agent is
little more than this loop pointed at a goal.

## The model doesn't run your code

This is the point to internalise: **the model never executes anything.** It only
*requests* a call. Your code sits in the middle and decides whether to honour that
request, and how.

That gap is where all your safety lives. Because you run the call, you can:

- **Validate the arguments** before acting on them — a requested value can be
  wrong, out of range, or nonsensical.
- **Apply least privilege** — give each tool the narrowest access it needs and
  nothing more.
- **Guard destructive actions** — require confirmation, or refuse outright, for
  anything that deletes, sends, or spends.

Treat a tool-call request the way you'd treat input from an untrusted source,
because in effect it is — the model can be steered by the content it reads (see
[prompt injection & security](/learn/building-ai/prompt-injection-and-security/)),
and it remains a [probabilistic component](/learn/building-ai/llm-as-a-component/)
whose requests you check rather than trust.

## Structured arguments

How do the arguments arrive? As **schema-constrained JSON** matching the parameter
schema you defined. If your tool takes a `system` string and a `talkgroup`
integer, that's what the model fills in.

This is exactly [structured output](/learn/building-ai/structured-output/) wearing
a different hat: the parameter schema constrains the *shape* of the arguments, so
you get the right fields and types. But the same caveat carries over — the shape
is guaranteed, the *values* are not. The model can ask for a `talkgroup` that
doesn't exist. Validate before you act.

## Designing tools

The model decides when and how to call a tool from what you tell it, so the
description and schema *are* the interface:

- **Clear descriptions.** The model reads the description to decide whether the
  tool fits. Say what it does and when to use it, in plain language.
- **Tight schemas.** Constrain the parameters — types, enums, required fields — so
  there's less room to fill them in wrongly.
- **One job per tool.** A tool that does a single, well-named thing is easier for
  the model to reach for than a Swiss-army function with a mode flag.
- **Helpful error returns.** When a call fails, return a message the model can
  act on — `"unknown talkgroup: 9999"` lets it correct course on the next step,
  where a bare `error` leaves it guessing.

## A worked example

Suppose you want the model to answer questions about a scanner system it can't
know from its training. You expose a tool:

```json
{
  "name": "lookup_frequency",
  "description": "Look up the RF frequency for a talkgroup on a named trunked system.",
  "parameters": {
    "type": "object",
    "properties": {
      "system":    { "type": "string" },
      "talkgroup": { "type": "integer" }
    },
    "required": ["system", "talkgroup"]
  }
}
```

A user asks: "What frequency is Metro dispatch on?" The exchange runs like this:

```text
model  → requests call: lookup_frequency({ "system": "Metro", "talkgroup": 101 })
you    → validate args, run the lookup → { "frequency_mhz": 851.0125 }
you    → return that result to the model
model  → "Metro dispatch (talkgroup 101) is on 851.0125 MHz."
```

The model supplied structured arguments; your code did the actual lookup and
handed back a fact; the model wove it into a plain-language answer. The same
pattern powers a `get_current_time()` tool (a model can't read a clock) or a
`calculate()` tool (a model shouldn't be trusted with exact arithmetic) — anything
the model can't do itself but your code can.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the model only requests a call; your code decides whether and how to run it." markdown="0">
  <p class="knowledge-check__q">Quick check: when the model "calls a tool", what actually happens?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The model runs your function itself and returns its output</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It REQUESTS a call with arguments; your code decides whether and how to run it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The provider runs the function on its own servers automatically</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Tool use / function calling** lets you describe functions the model can ask to
  call, giving it abilities it lacks: live data, exact math, real-world actions.
- The cycle is **define tools → model requests a call → your code executes → you
  return the result → the model continues** — loop it and you have an agent.
- **The model only requests**; your code runs the call. That gap is where you
  validate arguments, apply least privilege, and guard destructive actions.
- Arguments arrive as **schema-constrained JSON** — structured output under the
  hood — so the shape is guaranteed but the values still need validating.
- Design tools with **clear descriptions, tight schemas, one job each, and helpful
  error returns** — the description is how the model knows when to call it.

Next up: [streaming responses](/learn/building-ai/streaming/).
