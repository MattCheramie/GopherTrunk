---
slug: error-handling
title: Error handling
description: Errors are part of the contract — pairing the right status code with a machine-readable error body, stable error codes, messages that help, and what never to leak in an error.
keywords: API error handling, error response format, machine-readable errors, error codes, problem details, 4xx errors, helpful error messages
level: intermediate
status: full
prereq:
  - methods-and-status-codes
  - designing-a-good-api
---

# Error handling

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Errors are **part of the contract**, not an afterthought: consumers write code
against your failures as much as your successes. A good error is three layers —
the right **status code** (coarse, machine routing), a **stable error code**
(fine, machine branching), and a **human message** that says *what was wrong and
how to fix it*. Use **one error body shape everywhere**, never leak internals,
and never return errors as 200s.
</div>

Every API spends most of its documentation on the happy path and most of its
consumers' debugging time on the sad one. This lesson is about designing
failure well — the skill that most reliably separates professional APIs from
hobby ones.

## Errors are read by two audiences at once

When a request fails, two very different readers need answers. A **program**
needs to branch: retry or not? re-authenticate? report which field? It needs
*stable, machine-comparable* values. A **person** — the developer at 1 a.m. —
needs to understand: what exactly was wrong with what I sent, and what do I
change? They need *specific, helpful prose*. A design that serves only one
audience fails half its readers; the standard answer is to serve both in
layers:

```text
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": {
    "code": "invalid_parameter",
    "message": "Parameter 'talkgroup' must be a positive integer; got \"fire-dispatch\". Did you mean to filter by label? Use 'q=fire-dispatch'.",
    "param": "talkgroup"
  }
}
```

- **Status `400`** routes coarse behaviour: the
  [4xx/5xx logic](/learn/apis/methods-and-status-codes/) — don't retry
  unchanged.
- **`code: "invalid_parameter"`** is for `if` statements — a short, documented,
  **stable** string. Programs must never parse the prose message (it will be
  reworded, and every rewording would break them); the code is the
  machine-facing contract, and changing one is a
  [breaking change](/learn/apis/api-contracts/).
- **`message`** is for humans — and note what makes it good: it names the
  offending parameter, shows the rejected value, states the expectation, and
  suggests the likely fix. "Bad request" alone is a taunt; this is help.
- **`param`** adds structured detail where it exists — validation errors
  especially benefit from pointing at fields precisely.

There's even a standard shape if you'd rather adopt than invent: RFC 9457
"Problem Details" (`application/problem+json`) defines this same layering.
Whether you use it or your own — **use exactly one shape for every error on the
API**. A client should parse failures with one function, not per-endpoint
archaeology.

## The classic failures of failure

| Anti-pattern | Why it hurts |
|--------------|--------------|
| `200 OK` with `{"success": false}` | Invisible to every status-aware tool — monitoring, caches, retry logic, `curl -f` all see success |
| Different error shapes per endpoint | Clients need N parsers for one API |
| Prose-only errors | Programs resort to string-matching messages — which then break on rewording |
| Stack traces / SQL in responses | Leaks internals to attackers; see below |
| `500` for bad input | Miscategorises fault — clients retry the unfixable; your error monitoring drowns |

That first row deserves its emphasis: tunneling failures inside `200` opts you
out of the entire HTTP error ecosystem. The status code is not decoration — 
it's the field every generic tool reads.

## Say enough, but not too much

Error detail has a security dimension. To a *developer*, "password must be at
least 12 characters" is helpful; to an *attacker probing your login*, "no such
user" vs "wrong password" is reconnaissance (it confirms which usernames
exist). The balance: be **maximally specific about the client's input** (their
data — they sent it) and **minimally specific about your internals**. Stack
traces, file paths, SQL fragments, and dependency versions belong in your
server logs, correlated by an opaque `request_id` you *can* safely include —
letting a user report exactly which failure they hit without the error saying
anything about your insides. The
[API security lesson](/learn/apis/api-security/) returns to this boundary.

## Document the sad paths

Finish the contract: for each endpoint, document which status codes and error
codes it can produce and when — the
[testing lesson](/learn/apis/testing-an-api/) will turn those statements into
assertions. Good failure documentation is also where consumers learn your
*retry semantics*: which errors are transient (`503`, `429` — wait and retry,
per the [rate-limiting lesson](/learn/apis/rate-limiting-and-quotas/)) versus
permanent (`400`, `404` — fix or give up). An API that documents only success
has documented half of itself — the half that needs it less.

<div class="knowledge-check" data-quiz data-correct-msg="Right — status codes are what every generic tool keys on; errors inside 200 are invisible to all of them." markdown="0">
  <p class="knowledge-check__q">Quick check: what's wrong with returning <code>200 OK</code> with <code>{"success": false, "message": "not found"}</code>?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Nothing — the client can read the success field</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Every status-aware tool — monitoring, retries, caches, curl — sees a success, so the failure is invisible outside custom client code</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">JSON bodies aren't allowed on error responses</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Errors serve **two readers**: programs need the **status code** and a
  **stable error code**; humans need a **message** naming the problem and the
  fix.
- **One error shape everywhere** — clients should parse failure with a single
  function (RFC 9457 exists if you'd rather adopt than invent).
- Never tunnel errors through **200**, and never make programs parse prose.
- Be specific about **their input**, silent about **your internals** — an
  opaque `request_id` bridges to your logs.
- **Document the sad paths**, including which errors are retryable — error
  behaviour is contract.

Next up: [Rate limiting & quotas](/learn/apis/rate-limiting-and-quotas/).
