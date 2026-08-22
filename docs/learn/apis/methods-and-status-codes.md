---
slug: methods-and-status-codes
title: Methods & status codes
description: GET, POST, PUT, PATCH, DELETE and the five status-code families — what each method promises, what safety and idempotence mean, and how to read a server's three-digit verdicts.
keywords: HTTP methods, GET vs POST, PUT vs PATCH, HTTP status codes, 404, 500, idempotent, safe methods, 2xx 4xx 5xx
level: beginner
status: full
prereq:
  - anatomy-of-http
---

# Methods & status codes

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The five everyday HTTP methods each make a promise: **GET** reads without changing
anything (**safe**), **PUT** and **DELETE** can be repeated without changing the
outcome (**idempotent**), **POST** creates or acts (neither guarantee), **PATCH**
partially updates. Status codes come in families by first digit: **2xx** success,
**3xx** redirection, **4xx** *your* request is at fault, **5xx** the *server*
failed. The 4xx/5xx split — whose fault is it? — is the most useful single bit in
any API response.
</div>

The request line's method and the response's status code are the two most
information-dense fields in HTTP. This lesson gives you both vocabularies and the
two properties — safety and idempotence — that make the method table more than
rote memorisation.

## The five methods and their promises

| Method | Meaning | Safe? | Idempotent? | Typical use |
|--------|---------|-------|-------------|-------------|
| **GET** | Read a resource | Yes | Yes | Fetch call history |
| **POST** | Create / act | No | No | Add a new system to scan |
| **PUT** | Replace entirely | No | Yes | Overwrite a talkgroup record |
| **PATCH** | Update part | No | Usually | Rename talkgroup 1201 |
| **DELETE** | Remove | No | Yes | Delete a recording |

**Safe** means the request doesn't change server state — a GET is a question, and
asking twice is merely redundant. Browsers, caches, and crawlers *rely* on this:
they prefetch and retry GETs freely, which is why a `GET /api/v1/delete-all` link
is a career-limiting design (yes, real caches have really followed such links).

**Idempotent** means repeating the request leaves the world as if you'd sent it
once. `DELETE /calls/48213` twice: the second returns "already gone," but nothing
*extra* happens. `PUT` replaces with the same content — same result. `POST` is the
odd one out: `POST /systems` twice may create two systems. Idempotence is what
makes **retries safe** — if the network eats a response, a client can resend an
idempotent request without fear, but must think hard before resending a POST. That
thought becomes a whole design topic in
[rate limiting & quotas](/learn/apis/rate-limiting-and-quotas/) and
[designing a good API](/learn/apis/designing-a-good-api/).

> Rule of thumb: if a request might be retried by anything — a flaky network, a
> proxy, a hasty user — you want it to be idempotent. Design toward PUT-shaped
> updates when you can.

## Status codes: three digits, five families

The first digit tells you which conversation you're in:

| Family | Verdict | The ones you'll actually meet |
|--------|---------|-------------------------------|
| **2xx** | Success | `200 OK` · `201 Created` · `204 No Content` (success, empty body) |
| **3xx** | Go elsewhere | `301`/`308` moved permanently · `304 Not Modified` (cache is fresh) |
| **4xx** | Client's fault | `400 Bad Request` · `401 Unauthorized` · `403 Forbidden` · `404 Not Found` · `429 Too Many Requests` |
| **5xx** | Server's fault | `500 Internal Server Error` · `502 Bad Gateway` · `503 Service Unavailable` |

The load-bearing distinction is **4xx vs 5xx**. A 4xx says: your request, as sent,
cannot succeed — fix the request before retrying, because retrying the same bytes
will fail the same way. A 5xx says: your request may be fine, the server broke —
retrying later (politely, with backoff) is reasonable. Clients that treat these
identically end up hammering servers with unfixable requests or giving up on
transient failures.

A few fine points worth knowing early: **401 vs 403** — 401 means "I don't know
who you are" (missing/invalid credentials), 403 means "I know who you are, and
no" (a distinction [API authentication](/learn/apis/authentication-basics/)
sharpens). **404** is also the polite way to hide a resource's existence from
those not allowed to see it. **429** is a server defending itself, and comes with
its own etiquette in [rate limiting](/learn/apis/rate-limiting-and-quotas/).

## Reading an exchange like a sentence

Put method and status together and every API exchange reads as a grammatical
sentence: `PATCH /api/v1/talkgroups/1201` → `200 OK` is "update this talkgroup —
done." `GET /api/v1/calls/999999` → `404 Not Found` is "fetch this call — no such
call." `POST /api/v1/systems` → `201 Created` plus a `Location: /api/v1/systems/4`
header is "make a new system — made, and here's its new name." Notice that last
pattern: creation answers with the URL of the thing created, so the client can
address it from then on.

## The codes are the contract too

Which codes an endpoint returns, and when, is part of the
[API contract](/learn/apis/api-contracts/). If clients learned that a missing
talkgroup yields 404, switching to `200` with an empty body breaks them as surely
as renaming a field. And a status code alone is rarely enough for a good error —
*what* was bad about the request belongs in a machine-readable body, which is the
subject of [error handling](/learn/apis/error-handling/) in Unit 5.

<div class="knowledge-check" data-quiz data-correct-msg="Right — 4xx means the request itself is at fault, so the same request will fail again until it's fixed." markdown="0">
  <p class="knowledge-check__q">Quick check: your client gets a <code>400 Bad Request</code>. What's the correct move?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Retry the identical request with backoff until it succeeds</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Fix the request — as sent, it will keep failing no matter how often it's retried</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Treat it like a 500 and wait for the server to recover</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **GET reads, POST creates/acts, PUT replaces, PATCH updates, DELETE removes** —
  and each carries a promise, not just a name.
- **Safe** = changes nothing (GET); **idempotent** = repeatable without extra
  effect (GET, PUT, DELETE) — idempotence is what makes retries safe.
- Status families: **2xx success, 3xx redirection, 4xx client fault, 5xx server
  fault** — the 4xx/5xx split decides whether retrying can ever help.
- **401** = who are you? **403** = no, you specifically. **201 + Location** =
  created, here's its URL.
- The status behaviour of an endpoint is **part of the contract**.

Next up: [URLs, query strings, and bodies](/learn/apis/urls-queries-and-bodies/).
