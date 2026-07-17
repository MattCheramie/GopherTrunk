---
slug: structured-output
title: Structured output & JSON
description: Structured output makes a model return JSON that matches a schema you define, turning a text generator into a programmable component. Learn JSON mode, schema-constrained decoding, and what the guarantee does and doesn't cover.
keywords: structured output, JSON mode, JSON schema, function calling schema, machine-readable output, data extraction, constrained decoding, schema validation, enums, data types
level: intermediate
status: full
prereq:
  - parsing-and-validating-output
faq:
  - q: "What is structured output?"
    a: "It's a mode where the model returns **machine-readable** data — almost always JSON — that conforms to a **schema** you define, instead of free-form prose. Your code gets predictable fields and types it can use directly, rather than a paragraph it has to parse."
  - q: "What's the difference between JSON mode and schema-constrained output?"
    a: "JSON mode only promises the reply is *valid JSON* — some object, any shape. Schema-constrained output goes further: the model is forced to match a specific JSON Schema you supply, so you also get the *right fields, types, and enums*. Prefer schema-constrained when your provider offers it."
  - q: "If the JSON matches my schema, are the values correct?"
    a: "No. A schema only guarantees the **shape** — valid JSON with the right fields and types. The model can still put a wrong summary or a mislabelled category into a perfectly-shaped object. You still validate the values, exactly as in [parsing and validating output](/learn/building-ai/parsing-and-validating-output/)."
  - q: "How do I stop the model inventing a value when it doesn't know?"
    a: "Give it an escape hatch in the schema — an enum value like \"unknown\" or a nullable field — and say in the prompt to use it when the answer isn't present. Without that, a required field forces the model to guess, and it will."
---

# Structured output & JSON

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Structured output** makes the model return data that matches a **JSON schema**
you define, instead of prose you have to pick apart. That's the bridge from
"text generator" to programmable component: your code gets predictable fields
and types it can consume directly. It's the robust successor to hand-parsing
text — but it guarantees the *shape*, not that the *values* are right, so you
still [parse and validate](/learn/building-ai/parsing-and-validating-output/).
</div>

In the previous lesson you coaxed data out of free-form text by parsing it. That
works, but it's fragile — the model rewords things, adds a preamble, or drops a
field, and your parser breaks. Structured output flips the problem around: instead
of parsing whatever the model happens to say, you tell it up front exactly what
shape the answer must take, and the provider makes it comply.

## Why structured output

Parsing free text is brittle because the model's wording is
[probabilistic](/learn/building-ai/llm-as-a-component/) — the same request can
come back as "The priority is high" one call and "This looks fairly urgent" the
next. Any code that scrapes those sentences is one rephrasing away from failing.

Structured output removes that whole class of problem. You define a schema, the
model emits JSON that conforms to it, and your code receives predictable data:
the same fields, the same types, every time. There's nothing to scrape — you just
read the fields you asked for.

## How providers offer it

Providers expose this in a few flavours, from loosest to strictest:

- **JSON mode** — the model is told to reply with valid JSON. You get well-formed
  JSON, but its *shape* is up to the model unless your prompt pins it down.
- **Schema-constrained structured output** — you supply a JSON Schema and the
  model is *forced* to produce output matching it. The decoder is constrained so
  the fields, types, and allowed values come back as specified. This is the
  strongest guarantee and the one to prefer when available.
- **Tool / function-calling schemas** — when you give the model a tool, you
  describe that tool's parameters with a schema, and the model fills them in to
  "call" it. That parameter object is structured output by another name. We build
  on this in [tool use & function calling](/learn/building-ai/tool-use-and-function-calling/).

The exact names and knobs differ between providers, but the idea is the same
everywhere: describe the shape you want, get that shape back.

## Define a schema

A **JSON Schema** is just a description of the data: the fields, each field's
type, which are required, and any allowed values (enums). You hand it over; the
model fills it in.

```json
{
  "type": "object",
  "properties": {
    "summary":     { "type": "string" },
    "priority":    { "type": "string", "enum": ["low", "medium", "high"] },
    "needs_human": { "type": "boolean" }
  },
  "required": ["summary", "priority", "needs_human"]
}
```

A conforming response looks like this — and nothing else:

```json
{
  "summary": "Customer can't reset their password",
  "priority": "high",
  "needs_human": true
}
```

No greeting, no markdown fence, no trailing explanation — just the object your
code expects.

## What it does and doesn't guarantee

This is the part that trips people up. Schema-constrained structured output
guarantees the **shape**: you get valid JSON, with the fields you named, of the
types you specified, using only the enum values you allowed. That alone kills the
parsing fragility.

It does **not** guarantee the **values are correct**. The model can hand back a
perfectly-shaped object whose `summary` misreads the message or whose `priority`
is wrong. The schema constrains *form*, not *truth*. So structured output doesn't
replace validation — it makes parsing reliable and lets you spend your checking
budget on the values instead. Everything from
[parsing and validating output](/learn/building-ai/parsing-and-validating-output/)
still applies, and the underlying reason is why we
[treat the model as a probabilistic component](/learn/building-ai/llm-as-a-component/)
whose work you always review.

## Designing good schemas

A schema is a prompt in disguise — a clearer one makes better output:

- **Clear field names.** `needs_human` beats `flag`. The name tells the model
  what you want in it.
- **Enums over free strings.** If a field has a fixed set of values, list them.
  `"enum": ["low", "medium", "high"]` prevents "kinda urgent" and gives your code
  values it can branch on.
- **An escape hatch.** Add an `"unknown"` or `"not_found"` option (or make the
  field nullable) so the model isn't *forced* to guess when the answer isn't in
  the input. A required field with no out invites invention.
- **Keep it small.** Ask for exactly the fields the task needs. Every extra field
  is another thing to get wrong and another thing to validate.

## A worked example

Say you want to triage an incoming support message into structured fields your
system can route on: a short summary, a priority, a category, and whether a human
needs to step in.

```json
{
  "type": "object",
  "properties": {
    "summary":     { "type": "string" },
    "priority":    { "type": "string", "enum": ["low", "medium", "high"] },
    "category":    { "type": "string",
                     "enum": ["billing", "bug", "account", "other"] },
    "needs_human": { "type": "boolean" }
  },
  "required": ["summary", "priority", "category", "needs_human"]
}
```

Feed in the message "I was charged twice this month and can't reach anyone!" and
a conforming reply might be:

```json
{
  "summary": "Customer double-charged and unable to get a response",
  "priority": "high",
  "category": "billing",
  "needs_human": true
}
```

The same shape works far from support tickets. Imagine pulling structured fields
out of a freeform note a user typed about a system they want to monitor — "Metro
police, control channel around 851.0125, seems to be P25" — into something your
config can use:

```json
{
  "system_name": "Metro police",
  "frequency_mhz": 851.0125,
  "modulation":    "p25",
  "confidence":    "medium"
}
```

The note is prose; the output is data your code can act on. Note `confidence`
gives the model a graceful way to hedge instead of pretending the frequency is
exact — and `851.0125` came back as a *number*, not a string, because the schema
said so.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a schema constrains the shape (valid JSON, correct fields and types), not whether the values are actually correct." markdown="0">
  <p class="knowledge-check__q">Quick check: schema-constrained structured output guarantees what?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">That every value in the output is factually correct</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The output's shape/format — not that the values are correct</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">That the model will run faster and cost less</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Structured output** makes the model return JSON matching a **schema** you
  define — the bridge from text generator to programmable component.
- Providers offer it as **JSON mode** (valid JSON), **schema-constrained** output
  (forced to match a JSON Schema), and **tool/function-calling** parameter schemas.
- A schema lists the **fields, types, required-ness, and enums**; the model fills
  it in and your code reads it directly.
- It guarantees the **shape**, not the **values** — still validate what comes back.
- Good schemas use **clear names, enums, an "unknown" escape hatch**, and stay as
  small as the task needs.

Next up: [tool use & function calling](/learn/building-ai/tool-use-and-function-calling/).
