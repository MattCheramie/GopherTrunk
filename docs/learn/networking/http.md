---
slug: http
title: HTTP — the web's protocol
description: A plain-language guide to HTTP — the request-response exchange behind every website and API, the common methods (GET, POST, PUT, DELETE), status-code classes (2xx, 3xx, 4xx, 5xx), what headers carry, why HTTP is stateless, and how HTTP/1.1, HTTP/2, and HTTP/3 relate.
keywords: HTTP, GET, POST, status code, headers, request, response, stateless, HTTP methods, HTTP request response, 404, Content-Type
level: intermediate
status: full
prereq:
  - clients-and-servers
faq:
  - q: What is HTTP?
    a: "HTTP is the request-response protocol the web is built on. A client sends a text-based request naming a method and a path; the server sends back a response with a status code, headers, and usually a body. Nearly every website and API speaks it."
  - q: What is the difference between GET and POST?
    a: "GET asks the server to read and return something without changing it, so it's safe to repeat. POST sends data to create or submit something, which can change state on the server, so repeating it may repeat the effect."
  - q: What does an HTTP status code mean?
    a: "The status code is a three-digit number summarising the outcome. Its first digit gives the class: 2xx succeeded, 3xx redirects elsewhere, 4xx blames the request (like 404 not found), and 5xx means the server itself failed."
  - q: Why is HTTP called stateless?
    a: "Each HTTP request is handled on its own, with no memory of earlier ones. To carry identity or context across requests, the client resends it every time using cookies, tokens, or other headers."
---

# HTTP — the web's protocol

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
HTTP is the simple, text-based language nearly every website and API speaks. It
follows the **request/response** pattern from
[clients & servers](/learn/networking/clients-and-servers/): a client sends a
request, the server sends a response. The request names a **method** —
**GET** to read, **POST** to submit — and the response opens with a **status
code** (2xx worked, 4xx your fault, 5xx the server's). Both carry **headers**,
metadata describing the message. HTTP is **stateless**: each request stands
alone, so anything remembered gets resent every time.
</div>

If you've followed the [client-server](/learn/networking/clients-and-servers/)
rhythm, you already know the shape of HTTP — it's just the most important
concrete example of it. Learn how one HTTP exchange is put together and you've
learned the thing under nearly every website, app, and API.

## Request and response

An HTTP exchange is two messages. The **client sends a request**; the **server
returns a response**. Both are plain text and share the same three-part
structure: a **start line**, a set of **headers**, and an optional **body**.

The request's start line names a **method** and a **path**; the response's
start line reports a **status code**. Here's a minimal pair:

```http
GET /index.html HTTP/1.1
Host: example.com
User-Agent: curl/8.4.0
```

```http
HTTP/1.1 200 OK
Content-Type: text/html
Content-Length: 1256

<!doctype html>...
```

The request above has no body — it's just asking. The response's body is the
page itself, introduced by headers that describe it. That's the whole
conversation: one message out, one message back.

## Methods

The **method** tells the server what kind of action the request wants. A handful
cover almost everything:

- **GET** — read something. Fetch a page, an image, or some data. GET never
  sends a body and shouldn't change anything on the server.
- **POST** — create or submit. Send form data, upload a record, trigger an
  action. The data rides in the request body.
- **PUT** / **PATCH** — update an existing thing. PUT replaces it; PATCH changes
  part of it.
- **DELETE** — remove something.

Two words describe how methods behave. A method is **safe** if it only reads and
changes nothing — GET is safe. A method is **idempotent** if repeating it lands
you in the same place — GET, PUT, and DELETE are, but **POST usually isn't**,
which is why a browser warns before resubmitting a form.

## Status codes

Every response opens with a three-digit **status code**. The first digit is the
class, so you can read the outcome at a glance even without memorising the exact
number:

| Class | Meaning | Common examples |
|-------|---------|-----------------|
| **2xx** | Success | 200 OK, 201 Created |
| **3xx** | Redirect — look elsewhere | 301 Moved Permanently, 304 Not Modified |
| **4xx** | Client error — the request was wrong | 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found |
| **5xx** | Server error — the server failed | 500 Internal Server Error, 503 Service Unavailable |

The split between **4xx** and **5xx** is the useful part when something breaks.
A **4xx** points at the request — a bad path, missing login, no permission — so
you fix what you sent. A **5xx** points at the server — it received the request
fine but couldn't complete it — so the fix is on the other end.

## Headers

**Headers** are the metadata around a message: `Name: value` lines that sit
between the start line and the body. They describe the message without being
part of its content.

A few you'll meet constantly:

- **Content-Type** — what the body is: `text/html`, `application/json`, an
  image. It tells the receiver how to interpret the bytes.
- **Authorization** — credentials proving who's asking, such as a token or
  API key.
- **User-Agent** — identifies the client software making the request.

Headers appear on both requests and responses, and there are many more —
caching hints, cookies, content length. They're how the two sides negotiate the
details of an exchange without cluttering the body.

## Stateless

HTTP is **stateless**: the server handles each request on its own and remembers
nothing about the last one. Two requests from the same browser, a second apart,
arrive as strangers to each other.

That keeps servers simple and scalable, but it raises a question — how does a
site know you're logged in on the *next* click? The answer is that the client
resends the proof every time. **Cookies**, **tokens**, and other **headers**
carry that context along with each request, reconstructing "state" out of many
independent, stateless messages.

## Versions (brief)

You'll see **HTTP/1.1**, **HTTP/2**, and **HTTP/3** named in tools and configs.
The good news: they share the **same semantics** — the methods, status codes,
and headers above are identical across all three. What changed is the
**transport**: newer versions move the same messages faster and more
efficiently over the wire (multiplexing many requests, compressing headers, and
in HTTP/3 riding on a quicker connection setup). For everyday reasoning about
HTTP, you rarely need to think about which version you're on.

## Where you'll use it

HTTP is everywhere you touch the web: every site you load and every **web API**
you call runs on it. Because it's just text, you can even send an HTTP request
**by hand** with a tool like curl and read the raw response — we'll do exactly
that in [curl & HTTP tools](/learn/networking/curl-and-http-tools/). And when
APIs organise their HTTP methods and paths into a consistent style, you get
**REST**, which we cover in
[web APIs & REST](/learn/networking/web-apis-and-rest/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — 4xx means the request was at fault; 404 says the resource wasn't found." markdown="0">
  <p class="knowledge-check__q">Quick check: an HTTP response comes back 404. Which class of problem is that?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A server error (5xx) — the server crashed</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A client error (4xx) — the resource wasn't found</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A redirect (3xx) — look somewhere else</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- HTTP is a **text-based request/response** protocol — a client requests, a
  server responds — and it's what nearly every website and API speaks.
- Requests and responses share three parts: a **start line**, **headers**, and
  an optional **body**.
- **Methods** name the action: **GET** reads, **POST** submits, PUT/PATCH
  update, DELETE removes.
- **Status codes** report the outcome by class: **2xx** success, **3xx**
  redirect, **4xx** client error, **5xx** server error.
- **Headers** carry metadata like Content-Type, Authorization, and User-Agent.
- HTTP is **stateless** — each request stands alone, so cookies, tokens, and
  headers resend any state that must persist.

Next up: [TLS & HTTPS](/learn/networking/tls-and-https/)
