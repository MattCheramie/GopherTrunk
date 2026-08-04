---
slug: building-a-rest-api
title: Building a REST API
description: Turning routes and handlers into a coherent API a front end can consume — modelling resources as URLs, mapping HTTP verbs to actions, returning the right status codes, and speaking JSON, with the design conventions that make an API predictable.
keywords: REST API, resource, endpoint, HTTP verbs, GET POST PUT DELETE, status codes, JSON API, API design, RESTful, CRUD, request body, API contract
level: intermediate
status: full
prereq:
  - http-servers-and-routing
faq:
  - q: What makes an API RESTful?
    a: "A REST API models your data as resources, each with a URL, and uses HTTP's own verbs to act on them — GET to read, POST to create, PUT/PATCH to update, DELETE to remove — returning standard status codes and usually JSON. The style is 'RESTful' when it uses HTTP the way HTTP was designed rather than inventing its own scheme on top."
  - q: What's the difference between PUT and PATCH?
    a: "Both update an existing resource. PUT replaces the whole resource with the body you send, so it should contain the complete new state. PATCH applies a partial change — just the fields you include. PUT is idempotent (repeating it lands the same state); a careless PATCH may not be."
  - q: Why does the status code matter if the body already has the data?
    a: "Because the status code is the machine-readable summary the client checks first — 200/201 for success, 404 for missing, 400 for a bad request, 401/403 for auth. A front end (and every tool and proxy in between) relies on it to know what happened without parsing the body, which is exactly why fetch makes you check it."
---

# Building a REST API

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **REST API** is the contract your [back end](/learn/web-dev/what-a-backend-does/) offers
the front end. You model your data as **resources**, each with a **URL** (`/api/calls`,
`/api/calls/42`), and act on them with **HTTP verbs** — **GET** read, **POST** create,
**PUT/PATCH** update, **DELETE** remove. Every response carries a meaningful
[**status code**](/learn/networking/http/) and usually a **JSON** body. Done consistently,
the API becomes **predictable** — a front end can guess the next endpoint from the
pattern. The deeper reference is [web APIs &amp; REST](/learn/networking/web-apis-and-rest/).
</div>

The [routing lesson](/learn/web-dev/http-servers-and-routing/) gave you routes and
handlers. A **REST API** is what you get when you organise those routes into a coherent,
predictable **contract** — a set of conventions so consistent that a front-end developer
can consume it without reading much documentation. This lesson is those conventions:
resources, verbs, status codes, and JSON, assembled into an API a
[fetch](/learn/web-dev/fetching-data/) call can rely on. For the full protocol-level
treatment, the [networking module's REST lesson](/learn/networking/web-apis-and-rest/) is
the companion reference.

## Resources: nouns as URLs

The central idea of REST is to model your app as **resources** — the *things* it manages —
and give each a **URL**. Resources are **nouns**, not actions, and collections are plural:

- `/api/calls` — the collection of all calls.
- `/api/calls/42` — a single call, identified by id.
- `/api/talkgroups` — the collection of talkgroups.

The URL names *what*; the HTTP method (next section) says *what to do* with it. So you
never put a verb in the path — it's `POST /api/calls`, **not** `/api/createCall`. That one
discipline is most of what makes an API feel RESTful, because it means the same small set
of verbs works across every resource in a uniform way.

## Verbs: the same actions on every resource

REST reuses [HTTP's methods](/learn/networking/http/) as the verbs of the API, so the
*actions* are identical no matter which resource you're touching:

| Method | On `/api/calls` | On `/api/calls/42` |
|--------|-----------------|--------------------|
| **GET** | list all calls | fetch call 42 |
| **POST** | create a new call | — |
| **PUT / PATCH** | — | replace / update call 42 |
| **DELETE** | — | delete call 42 |

This is the familiar **CRUD** set — create, read, update, delete — mapped onto four verbs.
Because the mapping is uniform, learning one resource teaches you all of them: if you know
`GET /api/calls` and `POST /api/calls`, you already know `GET /api/talkgroups` and
`POST /api/talkgroups`. Remember the [method semantics](/learn/networking/http/) too — GET
is safe (never changes anything), and GET, PUT, and DELETE are idempotent while POST
generally isn't.

## Status codes: say what happened

Every response must return a meaningful [status code](/learn/networking/http/) — it's the
machine-readable summary the client checks before anything else. Pick the code that
actually describes the outcome:

- **200 OK** — a successful GET, PUT, or DELETE.
- **201 Created** — a successful POST that made a new resource.
- **400 Bad Request** — the client sent something malformed or invalid.
- **401 Unauthorized / 403 Forbidden** — not logged in / not allowed.
- **404 Not Found** — no such resource.
- **500 Internal Server Error** — the server failed.

Returning `200 OK` on an error — with the real problem buried in the body — is a common
mistake that breaks clients, because the [fetch](/learn/web-dev/fetching-data/) call sees
success and marches on. The status code is a promise: get it right and every client, proxy,
and tool downstream can trust it.

## JSON: the body format

REST APIs speak **JSON** in both directions. A GET returns a resource (or a list) as JSON;
a POST or PUT sends the new state in a JSON request body with a
`Content-Type: application/json` header.

```json
POST /api/calls
{ "talkgroup": "Fire-1", "seconds": 12 }
```

```json
HTTP/1.1 201 Created
Location: /api/calls/42
{ "id": 42, "talkgroup": "Fire-1", "seconds": 12 }
```

Notice the response echoes the created resource **with its new id**, and a `Location`
header points at its URL — small touches that make the API pleasant to consume. Keep field
names and shapes **consistent across endpoints** (same casing, same date format, the same
error shape everywhere) so a client learns your conventions once. And because this JSON
crosses the network, serve it over [HTTPS](/learn/networking/tls-and-https/) so it's
encrypted in transit.

## Design for predictability

The thread through all of this is **predictability**. A well-designed REST API is one a
developer can *guess*: see `GET /api/calls` and `GET /api/calls/42`, and they'll correctly
expect `DELETE /api/calls/42` to remove it and `POST /api/calls` to create one. Consistency
is the whole feature. A few conventions that pay off:

- **Version the API** (`/api/v1/…`) so you can change it later without breaking clients.
- **Return a consistent error shape** — same JSON structure for every error, with a
  message and maybe a code — so clients handle failures uniformly.
- **Validate every request on the server** — never trust the body — as
  [what a back end does](/learn/web-dev/what-a-backend-does/) insisted.
- **Paginate large collections** rather than returning ten thousand items at once.

Get these right and the front end's job — the [fetch](/learn/web-dev/fetching-data/) layer
— becomes almost mechanical, because the API behaves exactly as its shape implies. That
predictable contract is the real product of REST.

<div class="knowledge-check" data-quiz data-correct-msg="Right — in REST the URL names the resource (a noun) and the HTTP method says what to do, so it's POST /api/calls, not /api/createCall." markdown="0">
  <p class="knowledge-check__q">Quick check: in a REST API, how should you create a new call?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">GET /api/createCall?talkgroup=Fire-1</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">POST /api/calls with the new call in a JSON body</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">POST /api/calls/create with the verb in the path</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **REST API** is a **contract** built from consistent conventions the front end can
  rely on — see [web APIs &amp; REST](/learn/networking/web-apis-and-rest/) for the full
  reference.
- Model data as **resources** with **URLs** (nouns, plural collections); never put a verb
  in the path.
- Reuse **HTTP verbs** — GET/POST/PUT/PATCH/DELETE — so the same **CRUD** actions apply
  uniformly to every resource.
- Return a **meaningful status code** on every response — 201 on create, 404 on missing,
  400 on bad input — never 200 on an error.
- Speak **JSON** both ways with consistent field names and shapes, over
  [HTTPS](/learn/networking/tls-and-https/).
- Design for **predictability**: versioning, consistent errors, server-side validation,
  and pagination make an API a developer can guess.

Next up: [The back end &amp; its database](/learn/web-dev/backend-and-database/).
