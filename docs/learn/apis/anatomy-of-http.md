---
slug: anatomy-of-http
title: Anatomy of an HTTP request
description: Dissect one HTTP request and one response line by line — method, path, headers, blank line, body, status line — and you've seen the structure behind nearly every web API exchange.
keywords: HTTP request anatomy, HTTP request structure, HTTP headers, HTTP response, status line, request line, HTTP body, curl verbose
level: beginner
status: full
faq:
  - q: What are the parts of an HTTP request?
    a: "Four parts, in order: a request line (the method, the path, and the protocol version, like GET /api/v1/calls HTTP/1.1), then headers (one Name: value pair per line), then a blank line marking the end of the headers, then an optional body carrying data such as JSON. Responses mirror the shape, with a status line in place of the request line."
  - q: Is HTTP text I can actually read?
    a: "Yes — HTTP/1.1 is plain text, which is why you can type a request by hand into a raw connection or read one in a network capture. curl -v shows you the exact lines exchanged. Newer versions (HTTP/2 and HTTP/3) carry the same concepts in a binary framing for efficiency, but the model you learn from the text form transfers directly."
  - q: What is an HTTP header for?
    a: "Headers are labeled metadata about the request or response — what format the body is (Content-Type), how large it is (Content-Length), who's asking (User-Agent, Authorization), what answer formats the client accepts (Accept). They let the two sides negotiate and describe the exchange without mixing that bookkeeping into the data itself."
---

# Anatomy of an HTTP request

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An HTTP request is four parts in a fixed order: a **request line** (method + path +
version), **headers** (labeled metadata), a **blank line**, and an optional
**body**. A response mirrors it, opening with a **status line** (version + status
code + reason) instead. HTTP/1.1 is **plain text**, so you can read every byte of
an exchange — and once you've dissected one request and one response, you've seen
the structure of them all.
</div>

HTTP carries nearly every API in this module, and it's refreshingly literal: the
whole protocol is lines of text in a fixed order. This lesson takes one real
exchange apart, piece by piece, and gives you `curl -v` as the tool for seeing it
live. If you want the networking-side view of the same protocol, the
[Networking module's HTTP lesson](/learn/networking/http/) pairs well with this one.

## One complete exchange

Here is a client asking a GopherTrunk daemon for one talkgroup, and the answer —
every byte of the application-level conversation:

```text
GET /api/v1/talkgroups/1201 HTTP/1.1
Host: scanner.local:8080
Accept: application/json
User-Agent: curl/8.5.0

HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 62

{"id": 1201, "label": "County Fire Dispatch", "priority": 2}
```

Two blocks, one blank line inside each marking where headers end. Everything you
will ever see in an HTTP API is a variation of these few lines.

## The request line: method, path, version

`GET /api/v1/talkgroups/1201 HTTP/1.1` packs three fields:

- The **method** (`GET`) says what *kind* of action this is — here, "read, don't
  change anything." The method vocabulary gets its
  [own lesson](/learn/apis/methods-and-status-codes/).
- The **path** (`/api/v1/talkgroups/1201`) names *which resource* — this daemon's
  talkgroup 1201. Paths, query strings, and how data rides in them are
  [lesson 9](/learn/apis/urls-queries-and-bodies/).
- The **version** (`HTTP/1.1`) tells the server which dialect the client speaks.

Notice the path contains no server name — that travels separately, in the `Host`
header, because one server commonly serves several names.

## Headers: labeled metadata

Headers are `Name: value` lines, one per line, and they carry everything *about*
the exchange that isn't the data itself:

| Header | Direction | Says |
|--------|-----------|------|
| `Host` | request | Which site/server name this request is for |
| `Accept` | request | Body formats the client can handle |
| `Authorization` | request | Credentials proving who's asking ([Unit 2, lesson 10](/learn/apis/authentication-basics/)) |
| `Content-Type` | both | The format of *this message's* body (`application/json`) |
| `Content-Length` | both | Body size in bytes, so the receiver knows where it ends |

Header names are case-insensitive, order mostly doesn't matter, and unknown
headers are ignored — the same tolerant-reading norm you met in
[API contracts](/learn/apis/api-contracts/). `Content-Length` is quietly
load-bearing: it's how HTTP solves the message-boundary problem on a byte stream,
a theme that returns in [message framing](/learn/apis/message-framing/).

## The blank line and the body

The empty line is the fence: everything after it is **body**, raw payload bytes in
whatever format `Content-Type` declared. A `GET` request usually has no body; a
`POST` or `PATCH` usually carries JSON. The body is where the
[data formats](/learn/apis/data-formats/) lesson plugs in — HTTP frames the
message, JSON shapes the contents.

## The response: a status line, then the same shape

`HTTP/1.1 200 OK` is version + **status code** + human-readable reason. The
three-digit code is the machine-readable verdict — `200` success, `404` no such
resource, `500` server fault — and its families are half of the
[next-but-one lesson](/learn/apis/methods-and-status-codes/). After the status
line, a response is structurally identical to a request: headers, blank line,
body.

## See it live with curl

`curl -v` prints the raw exchange (`>` request lines, `<` response lines):

```bash
curl -v http://scanner.local:8080/api/v1/talkgroups/1201
```

Get comfortable reading that output now — it is the single most useful debugging
habit in this module, and the
[curl & HTTP tools lesson](/learn/networking/curl-and-http-tools/) goes deeper on
the tooling. When an API misbehaves, the raw request and response almost always
contain the answer: a missing header, a wrong `Content-Type`, a status code the
client ignored.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the blank line is the separator between the headers and the body." markdown="0">
  <p class="knowledge-check__q">Quick check: in an HTTP message, what marks the end of the headers?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A closing brace, because headers are JSON</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A blank line — everything after it is the body</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The Content-Type header, which is always last</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A request is **request line → headers → blank line → body**; a response is the
  same with a **status line** on top.
- The request line carries **method** (what kind of action), **path** (which
  resource), and **version**.
- **Headers** are labeled metadata — format, size, credentials, negotiation —
  kept separate from the data; unknown ones are ignored.
- The **blank line** fences off the **body**, whose format `Content-Type`
  declares and `Content-Length` measures.
- **`curl -v`** shows the raw exchange and is your first debugging tool for any
  HTTP API.

Next up: [REST fundamentals](/learn/apis/rest-fundamentals/).
