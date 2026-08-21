---
slug: api-documentation
title: API documentation
description: An undocumented API might as well not exist — the reference, examples, and guides an API needs, why copy-paste-runnable examples matter most, and how docs stay true as code changes.
keywords: API documentation, API reference, curl examples, OpenAPI docs, developer experience, documentation drift, getting started guide
level: beginner
status: full
prereq:
  - designing-a-good-api
---

# API documentation

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
To a consumer, **the documentation *is* the API** — capability they can't
discover doesn't exist, and where docs and behaviour disagree, trust dies. A
complete set is three layers: a **getting-started guide** (zero to first
successful call, fast), a **reference** (every endpoint, field, and error,
exhaustively), and **examples** — the layer users actually live in, which must
be **copy-paste runnable**. The enemy is **drift**; the defences are generating
reference from a [schema](/learn/apis/schemas-and-codegen/) and testing your
examples.
</div>

You can design a beautiful contract and implement it flawlessly, and none of it
matters if nobody can find out. This lesson covers what API docs must contain,
which parts earn the most return, and the discipline that keeps them honest
over years of change.

## Three layers, three questions

Different moments in a consumer's life need different documents:

| Layer | Answers | Shape |
|-------|---------|-------|
| **Getting started** | "How do I make *one* successful call?" | A short guided path: base URL, auth setup, first request, expected response |
| **Reference** | "Exactly what does this endpoint accept and return?" | Exhaustive per-endpoint tables: parameters, shapes, defaults, limits, **errors** |
| **Guides & examples** | "How do I accomplish my task?" | Task-shaped walkthroughs: "page through history", "subscribe to live events" |

The getting-started layer is disproportionately important: a consumer's first
ten minutes decide whether there's a second ten. Aim them at one concrete
victory —

```bash
curl http://scanner.local:8080/api/v1/talkgroups
```

— with the expected output shown, so success is *recognisable*. The reference
layer, meanwhile, is where completeness lives: every field's type and meaning
(is `duration_seconds` seconds? — [say so](/learn/apis/api-contracts/)),
defaults, pagination behaviour, and — most-skipped, most-needed — the
[error catalogue](/learn/apis/error-handling/): which codes, when, and which
are retryable.

## Examples are the real interface

Watch a developer meet a new API: they skim the prose, find the nearest
example, paste it, and mutate it toward their goal. Examples aren't decoration
on the docs — **they're the primary interface**, so treat them with code-level
care:

- **Runnable as pasted.** Real paths, plausible values, working syntax — with
  placeholders clearly marked (`$API_KEY`). An example that errors on paste
  costs more trust than no example.
- **Show the response too.** Half the value is knowing what success looks like.
- **In the consumer's tools** — `curl` at minimum, since it's the lingua franca
  ([curl & HTTP tools](/learn/networking/curl-and-http-tools/)); snippets in
  your main client languages are a bonus.

## The enemy is drift

Documentation is a copy of the contract, and copies rot: the API adds a field,
tightens a default, renames nothing but *behaves* differently — and the docs
silently describe last year's API. Drifted docs are worse than missing ones,
because consumers *trust* them into bugs. Two structural defences beat any
amount of diligence:

1. **Generate the reference from the schema.** If the
   [OpenAPI/`.proto` file](/learn/apis/schemas-and-codegen/) is the contract's
   single source of truth, reference docs rendered from it are current by
   construction — one more compounding return on schema-first.
2. **Test the examples.** Your examples are claims about behaviour; your
   [test suite](/learn/apis/testing-an-api/) can execute those claims. A CI job
   that runs each documented request against a real server turns "the docs
   lie" from a user bug report into a failing build.

> Rule of thumb: docs that no machine checks will drift. Wire the reference to
> the schema and the examples to the tests, and let humans spend their care on
> the guides.

## Write it down even when the audience is you

A private API — your daemon on your LAN, consumed by your own scripts — feels
exempt. It isn't: the consumer is *you in eighteen months*, who remembers
nothing. The minimal honest artifact is small — a README listing endpoints,
one example request/response apiece, auth if any, and the error shape. An
afternoon's work; its absence costs a re-derivation of the contract from source
every time you return. (GopherTrunk practices this at project scale — its
daemon's event interfaces are documented as public pages, which is exactly what
makes [Unit 6](/learn/apis/the-daemon-rest-api/) possible to teach.)

<div class="knowledge-check" data-quiz data-correct-msg="Right — generation and testing make drift a build failure instead of a slow trust leak." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the strongest defence against documentation drifting from real API behaviour?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A quarterly calendar reminder to re-read the docs</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Structural coupling — generate the reference from the schema and run the documented examples in CI</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Writing less documentation so there's less to drift</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- To consumers **the docs are the API**: undiscoverable capability doesn't
  exist, and doc/behaviour disagreement destroys trust.
- Three layers: **getting started** (first success, fast), **reference**
  (exhaustive, errors included), **guides/examples** (task-shaped).
- **Examples are the primary interface** — copy-paste runnable, responses
  shown, `curl` first.
- **Drift** is the enemy; **generate the reference** from the schema and
  **test the examples** in CI.
- Document even a **private API** — the future consumer who needs it most is
  you.

Next up: [API security](/learn/apis/api-security/).
