---
slug: designing-a-good-api
title: Designing a good API
description: Consistency, predictability, and small surface area — the qualities that make an API pleasant to use, and the concrete habits that produce them, from naming to pagination defaults.
keywords: API design, good API design principles, consistent API, API naming conventions, small surface area, principle of least surprise, API usability
level: intermediate
status: full
faq:
  - q: What makes an API good?
    a: "Predictability above all: a good API lets a user guess the second endpoint after learning the first, because naming, shapes, errors, and conventions are consistent everywhere. Add a small, deliberate surface area (few concepts, no redundant ways to do one thing) and honest ergonomics for the common case, and you have the qualities users describe as 'pleasant'."
  - q: Should I design the API before writing the implementation?
    a: Yes — the interface outlives and out-constrains the implementation. Sketch the endpoints, shapes, and error behaviour first (ideally as a schema), review them as design decisions, and let the implementation follow. Retrofitting a contract onto whatever the code grew produces APIs that leak internal structure and are painful to keep stable.
  - q: How big should an API be?
    a: As small as covers the real use cases. Every endpoint and field you publish is a promise you must keep for years, so publish deliberately — you can always add later (additive change is cheap), but you can almost never remove. When in doubt, leave it out and wait for a real consumer to need it.
---

# Designing a good API

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Good APIs share three qualities. **Consistency**: one naming style, one shape
vocabulary, one error format — everywhere, without exception. **Predictability**:
a user who has seen one endpoint can *guess* the next (the principle of least
surprise). **Small surface area**: few concepts, no duplicate ways to do one
thing, because every published element is a **forever promise**. Design the
interface before the implementation, for the consumer's mental model rather than
your internal one — and make the common case easy.
</div>

Units 1–4 taught you to read and consume interfaces; Unit 5 turns you around to
face the people consuming *yours*. This lesson is the aesthetic core: what
separates APIs people describe as "a joy" from the ones they curse in commit
messages — and the concrete habits behind the difference.

## Consistency: decide once, apply everywhere

An API is a small language, and languages are learnable in proportion to their
regularity. Decide each convention **once**, write it down, and never deviate:

- **Naming**: `snake_case` or `camelCase` for fields — either, but one.
  Plural collection names (`/talkgroups`). The same word for the same concept —
  if it's `talkgroup` in one response, it isn't `tg` or `group_id` elsewhere.
- **Shapes**: every list endpoint pages the same way, wraps results in the same
  envelope, sorts by the same convention. Every timestamp is the same format
  (ISO 8601, UTC — and *say so*).
- **Errors**: one error body shape everywhere — the
  [next lesson's](/learn/apis/error-handling/) entire subject.

Inconsistency is how APIs betray their history — three field styles marking
three contributors' eras. Users pay for it forever: every inconsistency is one
more thing that must be *looked up* instead of guessed. This is also, quietly,
the strongest everyday argument for [schemas](/learn/apis/schemas-and-codegen/):
conventions checked by tooling stay conventions.

> The test of predictability: show a user `GET /api/v1/talkgroups` and its
> response, then ask them to write the call fetching one radio ID by number. If
> the API is consistent, they'll write `GET /api/v1/radios/70233` — correctly —
> without opening the docs. Design so that guess is right.

## Small surface area: every element is a promise

The [contracts lesson](/learn/apis/api-contracts/) established the asymmetry:
adding is cheap, removing is nearly impossible. The design consequence is
restraint. Each endpoint, field, and option you publish is something you must
keep correct, documented, secure, and stable indefinitely — so publish only what
real use cases demand, and resist:

- **Speculative endpoints** ("someone might want to bulk-rename talkgroups") —
  wait for the someone.
- **Duplicate paths to one result** — two ways to filter calls means two
  behaviours to keep identical forever, and a user decision that shouldn't
  exist.
- **Leaked internals** — if the response mirrors your database tables, renaming
  a column just became an API question. Design shapes around the *consumer's*
  mental model (a "call" with a talkgroup label attached), not your storage.

## Ergonomics: make the common case easy

Consistency makes an API learnable; ergonomics make it pleasant. The moves are
unglamorous and compounding: **sane defaults** (a bare `GET /api/v1/calls`
returns something sensible — recent calls, newest first, reasonably limited —
with filters as opt-ins); **complete answers** (include the talkgroup label
*with* the call record; don't force N+1 follow-up requests for the obvious
join); **input flexibility, output strictness** (accept missing optionals
gracefully, but emit shapes rigidly — consumers parse what you emit); and
**design for the reader of code**, because each API call is written once and
read for years — `?since=2026-08-01` self-documents where `?s=1754006400`
doesn't.

## A worked sketch

The shape of a well-mannered daemon API, in miniature:

```text
GET  /api/v1/systems                     list configured systems
GET  /api/v1/systems/{id}                one system
GET  /api/v1/talkgroups?system={id}      talkgroups, filterable
PATCH /api/v1/talkgroups/{tgid}          update label/priority
GET  /api/v1/calls?talkgroup={tgid}&limit=50   history, paged
GET  /api/v1/events                      live SSE stream
```

Notice what makes it guessable: uniform plural nouns, IDs as path segments,
filters as query params, one verb vocabulary — every convention from
[Unit 2](/learn/apis/rest-fundamentals/), applied without exception. Boring, in
the best possible way. (For the builder's-side mechanics of routing and
handlers, the web-dev module's
[building a REST API](/learn/web-dev/building-a-rest-api/) picks up this
thread.)

<div class="knowledge-check" data-quiz data-correct-msg="Right — published surface is a forever promise, so speculative additions are liabilities, not generosity." markdown="0">
  <p class="knowledge-check__q">Quick check: why is "when in doubt, leave it out" good API design advice?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Smaller APIs respond faster to requests</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Everything published must be supported forever — adding later is cheap, removing is nearly impossible</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Users prefer APIs with fewer features to choose between</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Consistency** — one naming style, one shape vocabulary, one error format,
  decided once and never violated — is what makes an API learnable.
- **Predictability** is the test: a user should correctly *guess* endpoint two
  after seeing endpoint one.
- **Small surface area**: every element is a forever promise — no speculative
  endpoints, no duplicate paths, no leaked internals.
- Model shapes on the **consumer's mental model**, not your storage.
- **Ergonomics**: sane defaults, complete answers, readable parameters — easy
  common case, possible everything else.

Next up: [Error handling](/learn/apis/error-handling/).
