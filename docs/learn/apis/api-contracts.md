---
slug: api-contracts
title: API contracts
description: The shape of a request and response is a promise. Learn what an API contract covers, what a breaking change is, why contracts outlive implementations, and the discipline of additive-only change.
keywords: API contract, breaking change, backward compatibility, API stability, interface contract, additive change, API promise
level: beginner
status: full
prereq:
  - what-is-an-api
  - data-formats
---

# API contracts

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **API contract** is everything a consumer is entitled to rely on: the endpoints,
the request and response shapes, the meanings, the error behaviour. A **breaking
change** is any change that can make a correct existing client stop working —
removing or renaming a field, changing a type or a meaning. The safe direction is
**additive**: you may add, you may not take away or repurpose. Contracts routinely
**outlive their implementations**, which is why the contract deserves more care
than the code.
</div>

Unit 1 ends with its most important idea. You've seen that an API is a promise;
this lesson is about what, exactly, the promise covers — and what it costs to break
it. Everything in Unit 5 about designing and versioning APIs stands on this ground.

## What exactly is in the contract?

More than the documentation usually says. A consumer may reasonably rely on:

- **Endpoints and operations** — the paths, methods, or procedures that exist.
- **Request shape** — required and optional fields, their types, valid ranges.
- **Response shape** — which fields come back, their types, when they're present.
- **Meanings** — that `duration_seconds` is seconds, that `encrypted: true` means
  the audio is unrecoverable, that the list is sorted newest-first.
- **Error behaviour** — which failures produce which errors, and what they look like.

Notice the last two: the contract includes *semantics*, not just shapes. If a
scanner API always returned calls newest-first, and clients built paging on that,
silently switching to oldest-first breaks them just as surely as deleting a field —
even though every field still parses.

> Rule of thumb: anything a reasonable client could have depended on *is* the
> contract, whether or not you meant to promise it.

## What counts as a breaking change?

The test is behavioural: **could a correct client that worked yesterday fail
today?**

| Change | Breaking? | Why |
|--------|-----------|-----|
| Add a new optional response field | No | Well-behaved clients ignore unknown fields |
| Add a new endpoint | No | Nobody was calling it |
| Remove or rename a field | **Yes** | Clients reading it now get nothing |
| Change a field's type (`1201` → `"1201"`) | **Yes** | Parsers expecting a number choke |
| Make an optional request field required | **Yes** | Old requests are now rejected |
| Change a field's meaning or units | **Yes — and worst** | Everything parses; the answers are silently wrong |
| Tighten validation on inputs you used to accept | **Yes** | Previously valid requests now fail |

That last row is sneaky, and the meaning-change row is the most dangerous of all:
a renamed field fails loudly at parse time, but seconds-reinterpreted-as-
milliseconds fails *quietly*, producing plausible wrong numbers downstream.

## The additive discipline

The safe evolution rule is one sentence: **add, never remove or repurpose.** New
optional fields, new endpoints, new enum values (documented as "more may appear")
are all fine. This works because of its mirror-image rule for clients: **a client
must ignore fields it doesn't recognise** — a norm so useful it has a name,
tolerant reading, and JSON's flexibility makes it nearly free. Between the two
rules, an API can grow for years without a single coordinated upgrade.

Sometimes a break is genuinely necessary — a design mistake too costly to keep.
That's what [versioning](/learn/apis/api-versioning/) is for: run the old contract
and the new one side by side, and retire the old one politely. The `v1` in
GopherTrunk's `/api/v1/` paths is this discipline made visible: it's a namespace
for one contract, so a future incompatible contract can live at `v2` without
touching anyone still on `v1`.

## Why contracts outlive implementations

Here's the asymmetry that makes contracts precious: the provider can redeploy its
implementation in an afternoon, but the *consumers* are not yours to redeploy. They
are scripts on other people's machines, apps shipped to phones, integrations
written by someone who left the company. Every one of them encodes the contract as
it stood the day it was written.

So implementations churn — new frameworks, rewrites, whole languages — while the
contract accretes dependents and becomes the *hardest* thing to change. Seasoned
API designers therefore invert the beginner's instinct: sweat the interface, and
treat the implementation as replaceable. A day of thought before publishing an
endpoint is cheap; unpublishing one is nearly impossible.

## Contracts in the wild

You now have the whole Unit 1 toolkit: an [API](/learn/apis/what-is-an-api/) is a
promise, a [protocol](/learn/apis/what-is-a-protocol/) carries it,
[clients and servers](/learn/apis/clients-and-servers/) play the roles, a
[data format](/learn/apis/data-formats/) shapes the payloads, and the contract
binds it together. Unit 2 makes it concrete with the protocol that carries nearly
every contract you'll meet: HTTP.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a meaning change breaks clients silently, which makes it the most dangerous kind of break." markdown="0">
  <p class="knowledge-check__q">Quick check: an API keeps the field name and type of <code>duration</code> but starts sending milliseconds instead of seconds. Is that a breaking change?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">No — the response still parses, so clients are fine</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Yes — meanings are part of the contract, and this breaks clients silently</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Only if the API's documentation mentioned units explicitly</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The **contract** covers endpoints, shapes, **meanings**, and error behaviour —
  anything a reasonable client could depend on.
- A **breaking change** is anything that can make a correct existing client fail;
  meaning changes are the worst because they fail **silently**.
- Evolve **additively**: add optional fields and new endpoints; never remove or
  repurpose. Clients, in turn, must **ignore unknown fields**.
- When a break is unavoidable, **version** — run old and new contracts side by
  side.
- Contracts **outlive implementations** because consumers can't be redeployed —
  so design the interface with more care than the code.

Next up: [Anatomy of an HTTP request](/learn/apis/anatomy-of-http/).
