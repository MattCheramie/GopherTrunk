---
slug: your-first-api-call
title: Your first API call
description: The smallest end-to-end example of using a hosted model — get an API key, send one message over an SDK or raw HTTP, and read the reply — while keeping the key out of your code and safe in an environment variable.
keywords: API key, SDK, first request, environment variable, secret, authentication, model call, hello world, raw HTTP, usage tokens
level: beginner
status: full
prereq:
  - how-a-model-api-works
faq:
  - q: "Do I need my own server to call a model?"
    a: "No. The model runs on the provider's servers; you just send an HTTP request with your API key and read the reply. Any script, laptop, or backend that can make a web request can call it. You only need your own server later, when you want to keep your key off the client and add your own logic in front of the model."
  - q: "Where do I keep the API key?"
    a: "Out of your code. Load it from an **environment variable** or a secret manager at runtime, and never commit it to git. A key hard-coded in a source file — or pasted into a repo — is a key someone else can find and spend."
  - q: "Should I use an SDK or raw HTTP?"
    a: "Either works. An official **SDK** wraps the provider's HTTP API in your language, so you write a few lines instead of building requests by hand. Raw HTTP has zero dependencies and works anywhere, but you handle the request, headers, and JSON yourself. For a first call, the SDK is usually the gentler path."
  - q: "Why is the reply different each time I run it?"
    a: "Model output is sampled, not looked up, so the same prompt can return different wording on each call. That is normal. If you need stable output for tests, most APIs let you lower the randomness, but you should still treat the response as variable."
---

# Your first API call

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The smallest end-to-end example: **get a key, send a message, read the reply.**
You authenticate every request with an **API key**, send it most easily through
an official **SDK** (or raw HTTP if you prefer), and — the one rule that matters
most — you keep that key **out of your code**, loaded from an
**environment variable** instead. If you know
[how a model API works](/learn/building-ai/how-a-model-api-works/), this is where
you actually make the call.
</div>

The previous lesson explained the request-and-response shape of a hosted model.
This one turns that into a working call you can run today. There is less to it
than you might expect: the model already lives on the provider's servers, so your
job is just to authenticate, send one message, and read what comes back. The only
part that needs real care is your key.

## Get a key

Start by signing up with a provider. OpenAI, Anthropic, Google, and others all
offer hosted model APIs; the sign-up and the console look different, but the flow
is the same everywhere. Once you have an account, create an **API key** from the
provider's dashboard.

That key does two jobs. It **authenticates** your requests — it proves the call is
yours — and it **bills** them to your account. Because it is both identity and
payment, treat it exactly like a password: copy it once when it is shown, store it
somewhere safe, and don't paste it into chats, screenshots, or code.

## Keep the key secret

This is the rule to internalise before you write a single line: **never hard-code
the key, and never commit it to git.** A key checked into a repository — even a
private one, even briefly — should be considered leaked. Bots scan public code for
keys within minutes, and a leaked key means **someone else's requests on your
bill**, plus access to whatever your account can do.

Instead, load the key at runtime from an **environment variable** or a secret
manager, so the value lives in your environment, not your source:

```bash
export PROVIDER_API_KEY="your-key-here"
```

Add any local secrets file (like `.env`) to your `.gitignore` so it can't be
committed by accident. Handling personal data and credentials responsibly is a
topic in its own right — we come back to it in
[privacy, data & compliance](/learn/building-ai/privacy-data-and-compliance/),
and the sibling path covers it from the tooling side in
[security, privacy & ethics](/learn/ai-software-dev/security-privacy-ethics/).

## Install an SDK (or use raw HTTP)

Most providers ship an official **SDK** — a small library that wraps their HTTP
API in your language (Python, JavaScript, and more). It handles the endpoint,
headers, and JSON for you, so a call is a few readable lines. Install it with your
language's package manager and point it at your key.

You are never *required* to use an SDK, though. The API is just HTTP, so you can
send the request yourself with any HTTP client. Raw HTTP has no dependencies and
works in any language, but you assemble the request and parse the response by
hand. For a first call, the SDK is the smoother road; for a tiny script or an
unusual language, raw HTTP is perfectly fine.

## The smallest call

Here is the whole thing in pseudocode. Read the key from the environment, send one
user message, and print the reply:

```python
import os
from provider_sdk import Client

# Load the key from the environment — never write it in the file.
client = Client(api_key=os.environ["PROVIDER_API_KEY"])

response = client.messages.create(
    model="a-model-name",
    messages=[
        {"role": "user", "content": "Say hello in one short sentence."},
    ],
)

print(response.text)
```

The shape is the same in JavaScript: read `process.env.PROVIDER_API_KEY`, call the
SDK's create method with a model name and a list of messages, and read the text off
the response. Different libraries name things slightly differently, but every one
follows this get-key, send-messages, read-reply pattern.

## Reading the response

The reply is more than just text. A typical response gives you:

- **The text** — the model's actual answer, which is what you usually print or use.
- **Usage / token counts** — how many tokens went in and came back. This is what
  you are billed on, so it is worth logging even on a first call.
- **A finish reason** — why the model stopped: because it finished naturally, or
  because it hit the length limit and was cut off mid-answer.

Checking the finish reason is a habit worth forming early. As the previous lesson
noted, a truncated reply usually means you hit a length cap, not that the model had
nothing more to say.

## First gotchas

A few things trip up almost everyone on their first attempt:

- **Auth errors.** A missing, mistyped, or wrong key gives an authentication
  error. Confirm the environment variable is actually set in the shell that runs
  your code.
- **Wrong model name.** Model identifiers are exact strings; a typo or an outdated
  name returns a "model not found" error. Copy the name from the provider's docs.
- **Rate limits and quota.** New accounts have caps on how fast and how much you
  can call. Hitting them returns a rate-limit error — not a bug, just a limit.
  Designing around these is the next lesson,
  [cost, latency & rate limits](/learn/building-ai/cost-latency-and-limits/).
- **The reply changes run to run.** Same prompt, different wording — that is normal
  sampling, not a fault. Building on top of a component that varies is exactly the
  mindset from [treating the model as a component](/learn/building-ai/llm-as-a-component/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — the key lives in an environment variable or secret store, never committed in your code." markdown="0">
  <p class="knowledge-check__q">Quick check: where should your API key live?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Hard-coded near the top of your main source file so it's easy to find</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">In an environment variable or secret store — never committed in your code</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">In a config file checked into git so your teammates can use it too</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A hosted model runs on the provider's servers — you need **no server of your
  own** to make a first call, just an API request.
- An **API key** both authenticates and bills your calls; treat it like a password.
- **Keep the key out of your code** — load it from an **environment variable** or
  secret manager, and never commit it to git.
- An official **SDK** wraps the HTTP API for you; raw HTTP works too, with more
  manual work.
- The smallest call is **get key, send one message, read the reply** — and read the
  text, the token usage, and the finish reason off the response.
- Expect auth errors, wrong model names, rate limits, and replies that vary run to
  run; none of those are bugs.

Next up: [cost, latency & rate limits by design](/learn/building-ai/cost-latency-and-limits/).
