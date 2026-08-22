---
slug: urls-queries-and-bodies
title: URLs, query strings, and bodies
description: Where data rides in an HTTP request — path segments for identity, query parameters for options and filters, JSON bodies for payloads — and how to choose among them, plus URL encoding.
keywords: query string, URL parameters, path parameters, request body, query vs body, URL encoding, percent encoding, API filtering pagination
level: beginner
status: full
prereq:
  - anatomy-of-http
  - rest-fundamentals
---

# URLs, query strings, and bodies

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A request has three places to carry data, each with a job: **path segments** say
*which resource* ("identity"), **query parameters** say *how* — filters, sorting,
paging ("options"), and the **body** carries *the payload itself* — the data being
created or changed. Special characters must be **percent-encoded** to travel in a
URL. Put identity in the path, options in the query, payloads in the body, and
**secrets in none of them** — URLs get logged everywhere.
</div>

You know a request's skeleton from
[anatomy of an HTTP request](/learn/apis/anatomy-of-http/). This lesson is about
choosing where each piece of data rides — a small decision you'll make constantly,
with real consequences for caching, logging, and clarity.

## The three carriages

Take a realistic query against a scanner daemon — "the last 50 calls on talkgroup
1201 of the county system, newest first":

```text
GET /api/v1/calls?system=county-p25&talkgroup=1201&limit=50&sort=desc HTTP/1.1
Host: scanner.local:8080
```

- **Path** — `/api/v1/calls` — names the resource: the call history. If the request
  were about *one specific call*, its ID would be a path segment too:
  `/api/v1/calls/48213`.
- **Query string** — everything after the `?`, as `key=value` pairs joined by `&` —
  refines the request: filter by system and talkgroup, cap at 50, sort descending.
  Omitting any of them should still make sense (defaults apply).
- **Body** — absent here, as it should be for a GET. When creating or updating, the
  payload goes in the body as JSON:

```text
PATCH /api/v1/talkgroups/1201 HTTP/1.1
Host: scanner.local:8080
Content-Type: application/json
Content-Length: 34

{"label": "County Fire Dispatch"}
```

## How to choose: identity, options, payload

> Rule of thumb: **path = which thing. Query = how to fetch it. Body = the thing's
> data.** If removing it changes *which resource* you mean, it's path. If removing
> it just changes the view — fewer results, a different order — it's query. If
> it's the content being stored, it's body.

Two useful consequences follow. First, path + query together form the full URL, so
everything there is **shareable and cacheable**: two clients sending the same URL
ask the same question, which is what lets caches and bookmarks work. Second,
bodies are neither — which is why *reads* favour query parameters (a filtered GET
can be repeated, cached, pasted into `curl`) and *writes* favour bodies (payloads
can be large, deeply structured, and aren't meant to be a name for anything).

| Data | Where | Example |
|------|-------|---------|
| Which call | Path | `/api/v1/calls/48213` |
| Filter, page, sort | Query | `?talkgroup=1201&limit=50` |
| New label to store | Body | `{"label": "County Fire Dispatch"}` |
| Credentials | **Header** | `Authorization: Bearer …` — never the URL |

That last row is a security rule, not a style choice: URLs are written to server
logs, proxy logs, and browser history as a matter of course, so a key in a query
string is a key copied everywhere. The [authentication lesson](/learn/apis/authentication-basics/)
returns to this.

## Percent-encoding: making data URL-safe

URLs have structural characters — `/`, `?`, `&`, `=`, spaces aren't allowed at
all — so data containing them must be escaped: each risky byte becomes `%` plus
two hex digits. A search for the label `Fire & Rescue` travels as:

```text
GET /api/v1/talkgroups?q=Fire%20%26%20Rescue
```

(`%20` is the space, `%26` the `&`.) Every HTTP library encodes and decodes this
for you *if you pass parameters properly* — the classic bug is pasting values into
a URL string by hand and shipping an unencoded `&` that silently splits your value
into two parameters. Let the library build the query string.

## Pagination: query strings earning their keep

A call history grows without bound, and no one wants a million-row response. The
standard pattern is paging through the collection with query parameters — either
**offset** style (`?limit=50&offset=100`) or **cursor** style
(`?limit=50&after=48213`), where each response hands you the cursor for the next
page. Cursor paging holds up better while data is being inserted (an offset
shifts under you when new calls land — and on a live scanner they always are).
Whichever style an API chooses, it belongs in the query string: it changes the
view, not the resource. How a *designer* picks defaults and caps here is a
[Unit 5 topic](/learn/apis/designing-a-good-api/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — a filter changes the view of the collection, so it rides in the query string." markdown="0">
  <p class="knowledge-check__q">Quick check: you want only encrypted calls in the history listing. Where does <code>encrypted=true</code> belong?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">In the path: <code>/api/v1/calls/encrypted/true</code></button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">In a JSON body attached to the GET</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">In the query string: <code>/api/v1/calls?encrypted=true</code></button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Path = identity** (which resource), **query = options** (filter, page, sort),
  **body = payload** (the data being stored or changed).
- Path + query form the URL, making them **shareable and cacheable**; bodies are
  neither — so reads favour query, writes favour body.
- **Percent-encoding** escapes structural characters; always let your HTTP
  library build query strings.
- **Pagination** (offset or cursor) lives in the query string; cursors survive
  live-growing data better.
- **Secrets never ride in URLs** — they land in logs; credentials belong in
  headers.

Next up: [API authentication](/learn/apis/authentication-basics/).
