---
slug: schemas-and-codegen
title: Schemas & code generation
description: Write the contract once, generate the client and server from it — machine-readable schemas like .proto and OpenAPI as the single source of truth, what generation buys, and where it bites.
keywords: API schema, code generation, OpenAPI, swagger, protobuf schema, single source of truth, contract-first design, generated client
level: intermediate
status: full
prereq:
  - api-contracts
  - grpc-and-protobuf
---

# Schemas & code generation

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **schema** is the API contract written in a **machine-readable** form — a
`.proto` file, an **OpenAPI** document — from which tools can **generate** typed
clients, server scaffolding, docs, and validators. The payoff is a **single
source of truth**: the contract can't silently drift from the code, and a
breaking change becomes a visible diff in one file. The discipline that makes it
work is **contract-first**: the schema leads, code follows — never the reverse
regenerated from whatever the code happens to do.
</div>

Unit 4 closes by generalising its best idea. gRPC's `.proto` workflow — define
once, generate everywhere — isn't a gRPC quirk; it's a philosophy that applies to
REST too, and it directly attacks the failure mode Unit 1 warned about: contracts
that live only in prose and habit, quietly rotting.

## The problem: contracts written in prose drift

Without a schema, your API's contract exists in several places at once: the
server's handler code, the docs page, each client's parsing code, and everyone's
memory. Nothing enforces agreement among them. The server adds a field, the docs
lag, a client typos `talkgorup` and reads `null` forever —
[remember](/learn/apis/api-contracts/) that a mistyped JSON key doesn't error,
it just yields nothing. Every one of those drifts is invisible until a bug
surfaces far away.

A schema collapses them into **one artifact**. For gRPC you've seen it — the
`.proto` file. For REST APIs the standard is **OpenAPI** (née Swagger): a
YAML/JSON document describing every path, method, parameter, and response shape:

```text
paths:
  /api/v1/talkgroups/{id}:
    get:
      parameters:
        - name: id
          in: path
          required: true
          schema: { type: integer }
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Talkgroup"
components:
  schemas:
    Talkgroup:
      type: object
      required: [id, label]
      properties:
        id:       { type: integer }
        label:    { type: string }
        priority: { type: integer }
```

## What generation buys

Feed that file to the toolchain and out comes, per language you care about:

- **Typed clients** — `GetTalkgroup(id int) (Talkgroup, error)` instead of
  hand-parsed maps; the typo bug becomes a compile error.
- **Server scaffolding** — handler interfaces the implementation must satisfy,
  so the server can't accidentally diverge from the contract either.
- **Validators** — request/response checking against the schema at runtime or
  in tests.
- **Documentation** — rendered reference pages that are *definitionally*
  current, the foundation the [API documentation lesson](/learn/apis/api-documentation/)
  builds on.

And one more thing, subtler and maybe most valuable: **reviewable evolution**.
When the contract is a file, every change to it is a diff in version control.
"Wait, this PR renames a response field — that's breaking" becomes a code-review
observation instead of a production incident. Tools can even lint schema diffs
for compatibility automatically, mechanising the
[additive-change discipline](/learn/apis/api-versioning/).

## Contract-first, not code-first

There are two ways to hold a schema, and they differ more than they look:

| | Contract-first | Code-first |
|---|----------------|------------|
| The schema is | **authored** — the design artifact | **extracted** — generated from code annotations |
| Design pressure | interface thought through before implementing | whatever shape the code grew |
| Drift risk | code can't drift (it's generated/validated) | schema faithfully documents drift as it happens |

Code-first ("annotate the handlers and export OpenAPI") feels convenient and is
better than nothing — but it inverts authority: the code leads, and the schema
becomes a *description* rather than a *contract*. Every accidental change of
shape gets dutifully exported as the new "contract." Contract-first keeps the
promise where [Unit 1](/learn/apis/api-contracts/) said it belongs: designed
deliberately, changed deliberately.

> Rule of thumb: the schema is the contract's home. Author it, review changes to
> it like the interface decisions they are, and generate everything else from
> it.

## The honest costs

Schema-first isn't free, and pretending otherwise breeds backlash. There's a
**toolchain** to install and version (`protoc`, generators, CI steps); generated
code can be **bulky or unidiomatic**, and you must never hand-edit it (edits die
at next generation — wrap it instead); and schema languages have **expressive
limits** — "field B is required when A is set" often lives in prose or
validators anyway, so the schema-plus-docs pairing remains a pairing. For a
three-endpoint hobby API, a well-written README and disciplined
[tests](/learn/apis/testing-an-api/) may honestly be the right size. The
technique earns its overhead as endpoints, clients, and contributors multiply —
which real projects tend to do.

<div class="knowledge-check" data-quiz data-correct-msg="Right — extracted schemas document whatever the code does, drift included; authored schemas are the contract." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the core weakness of generating your schema from code annotations?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The generated schema file is too large to keep in version control</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Code-first schemas can't describe JSON request bodies</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Authority inverts — accidental code changes get exported as the new "contract" instead of being caught against it</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **schema** (`.proto`, **OpenAPI**) is the contract in machine-readable form —
  one artifact where prose-and-habit contracts were many.
- Generation yields **typed clients, server scaffolding, validators, and
  ever-current docs** — and mistyped-field bugs become compile errors.
- A schema in version control makes evolution **reviewable and lintable**:
  breaking changes show up as diffs.
- **Contract-first** keeps the schema authoritative; code-first merely documents
  drift.
- Costs are real — toolchain, generated-code ergonomics, expressive limits —
  and scale with project size in reverse: the bigger the project, the better the
  deal.

Next up: [Designing a good API](/learn/apis/designing-a-good-api/).
