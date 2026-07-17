---
slug: how-a-model-api-works
title: How a model API call works
description: A model API call is an HTTP request carrying an ordered list of messages plus parameters, and it returns generated text with a usage report. Learn about roles, tokens, and why the API is stateless.
keywords: model API, messages, system prompt, roles, tokens, context window, request response, stateless, completion, usage, finish reason, temperature
level: beginner
status: full
prereq:
  - building-vs-using-ai
faq:
  - q: What do you actually send to a model API?
    a: "An ordered list of messages, each tagged with a role (system, user, or assistant), plus parameters like which model to use, a maximum output length, and temperature. The API returns generated text and a usage report of how many tokens were consumed."
  - q: How does the model remember earlier turns of a conversation?
    a: "It doesn't. The API is stateless — it keeps nothing between calls. To continue a conversation, your code resends the whole message history on every request. The model only ever sees what you send this time."
  - q: What is a token?
    a: "A token is a chunk of text — roughly a short word or part of one — that the model reads and writes in. Both your input and the model's output are measured in tokens, the context window caps how many fit in one call, and token counts drive cost and latency."
  - q: What does the finish reason tell me?
    a: "It says why the model stopped: \"stop\" means it finished naturally, \"length\" means it hit your maximum output limit, and \"tool_use\" means it wants to call a tool. Checking it tells you whether a short reply was complete or simply cut off."
---
# How a model API call works

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A model call is an ordinary **HTTP request**. You send an ordered list of
**messages**, each tagged with a **role** — **system** (standing instructions),
**user** (the input), **assistant** (prior replies) — plus a few parameters. Text is
counted in **tokens** on the way in and out. The reply is generated text plus a usage
report. And it's **stateless**: the API remembers nothing, so your code resends the
whole conversation every time.
</div>

You've seen [when to build on a model versus just use one](/learn/building-ai/building-vs-using-ai/).
This lesson opens the box: what actually goes over the wire when your program talks to
a model. It's simpler than it looks — a request with a list of messages, and a response
with generated text. Everything else is detail on top of that shape.

## The request: a list of messages

The core of every call is an **ordered list of messages**. Order matters: the model
reads them top to bottom as a conversation so far. Each message carries a **role** that
tells the model what that message *is*:

- **system** — standing instructions: the model's persona, the rules it should follow,
  the format you want back. This is set by you, the developer, not the end user, and it
  frames everything else.
- **user** — the input: a question, a command, a chunk of text to work on. In a chat app
  this is what the person typed.
- **assistant** — the model's *own* prior replies. You include these so the model can
  see what it already said and stay consistent across a multi-turn conversation.

Alongside the messages you send **parameters** that shape the call: which **model** to
use, the **maximum number of output tokens** to generate, the **temperature**, and
optionally **stop** sequences. More on those below.

## Tokens

The model doesn't read characters or whole words — it reads **tokens**. A token is a
chunk of text, often a short word or a piece of one. "Antenna" might be one token;
"downconverter" might be several. Both what you send (**input tokens**) and what the
model generates (**output tokens**) are counted this way.

Two consequences follow. First, every model has a **context window** — a hard cap on how
many tokens fit in a single call, input and output combined. Overflow it and the call
fails or older content must be dropped. (See the sibling lesson
[context windows](/learn/ai-software-dev/context-windows/).) Second, tokens are the unit
of both **cost** and **latency**: you pay per token, and more tokens take longer to
process. We'll dig into that in [cost, latency and limits](/learn/building-ai/cost-latency-and-limits/).

## The response

The reply comes back with three things worth knowing:

- **Generated content** — the text the model produced. (Depending on how you called it,
  this may instead be a request to call a tool, or a structured object like JSON.)
- **A finish reason** — why the model stopped. Common values: `stop` (finished
  naturally), `length` (hit your output-token limit), `tool_use` (wants to call a tool).
- **A usage block** — how many input and output tokens the call consumed, so you can
  track cost.

Here is a compact, provider-neutral sketch of a request and its response:

```json
// request
{
  "model": "some-model",
  "max_output_tokens": 200,
  "temperature": 0.7,
  "messages": [
    { "role": "system", "content": "You are a concise assistant." },
    { "role": "user",   "content": "Name three SDR sample rates." }
  ]
}
```

```json
// response
{
  "content": "Common SDR sample rates include 2.4, 2.5, and 10 MS/s.",
  "finish_reason": "stop",
  "usage": { "input_tokens": 24, "output_tokens": 18 }
}
```

Field names differ between providers, but the shape — messages in, content plus a finish
reason plus usage out — is the same everywhere.

## It's stateless — you carry the conversation

Here is the part that surprises people: the API is **stateless**. It remembers *nothing*
between calls. Each request is judged entirely on the messages you send in that request,
and the moment it responds, the server forgets you were ever there.

So how does a chatbot "remember" the last ten turns? Your code does the remembering. To
continue a conversation, you resend the **entire message history** — every prior user and
assistant message — appended with the new user message, on every single call. The model
re-reads the whole thing each time.

This is why context, and cost, **grow as a conversation gets longer**: turn twenty sends
all nineteen previous turns plus the new one. Nothing is stored for you; the running
conversation lives in your application.

## Parameters you'll actually set

A handful of parameters do most of the work:

- **Model** — which model to call. Bigger, more capable models cost more and run slower;
  choosing well is its own topic (see
  [choosing a model for a feature](/learn/building-ai/choosing-a-model-for-a-feature/)).
- **Maximum output tokens** — a ceiling on how much the model may generate. It caps cost
  and latency, but set it too low and replies get cut off (finish reason `length`).
- **Temperature** — how much randomness the model uses when picking each next token.
  Lower is more focused and repeatable; higher is more varied. The mechanics are covered
  in [how models decide](/learn/ai-software-dev/how-models-decide/).
- **Stop sequences** — strings that, when generated, make the model halt immediately.
  Handy for keeping output inside a boundary you define.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the API is stateless, so your code resends the prior messages on every call." markdown="0">
  <p class="knowledge-check__q">Quick check: how does the model "remember" earlier turns of a chat?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The server stores your conversation and looks it up by session</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It doesn't — your code resends the prior messages on each call (the API is stateless)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The model was trained on your previous messages</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A model call is an **HTTP request** carrying an ordered **list of messages** plus
  parameters, and it returns generated text with a usage report.
- Each message has a **role** — **system** (standing instructions), **user** (the input),
  **assistant** (prior replies) — and order matters.
- Text is measured in **tokens** on both input and output; the **context window** caps
  how many fit, and tokens drive cost and latency.
- The response carries **content**, a **finish reason**, and a **usage** block.
- The API is **stateless**: to continue a conversation your code resends the whole
  history every call, which is why context and cost grow.
- The parameters you'll set most are **model**, **max output tokens**, **temperature**,
  and **stop sequences**.

Next up: [your first API call](/learn/building-ai/your-first-api-call/).
