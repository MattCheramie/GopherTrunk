---
slug: curl-and-http-tools
title: Making requests with curl
description: A plain-language guide to curl — how to fetch a page, read status codes and headers with -I and -v, send data with -X, -d, and -H, follow redirects with -L, save files with -o and -O, and use wget, so you can test an API or a site straight from the shell.
keywords: curl, wget, HTTP request, headers, POST, download, API testing, verbose, status code, -X, follow redirects, command line
level: intermediate
status: full
prereq:
  - http
faq:
  - q: What does the curl command do?
    a: "curl sends an HTTP request from the command line and prints the response. Point it at a URL and it fetches the body; add flags to see headers, change the method, send data, or save the result to a file. It lets you talk to a website or API without a browser."
  - q: How do I see the response headers with curl?
    a: "Use curl -I to fetch just the headers with no body, or curl -i to print the headers followed by the body. For the full picture — the request you sent, the TLS handshake, and the response — add curl -v for verbose output."
  - q: How do I send a POST request with curl?
    a: "Add -X POST to set the method and -d to attach the body, like curl -X POST -d 'key=value' URL. For a JSON payload, send the JSON with -d and set -H 'Content-Type: application/json' so the server knows how to read it."
  - q: What is the difference between curl and wget?
    a: "Both fetch things over HTTP from the shell. curl is built for making and inspecting single requests — headers, methods, bodies — and prints to the screen by default. wget is built for downloading, saving files by default and handling recursive fetches, so it shines when you just want to grab a file or a whole site."
---

# Making requests with curl

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**curl** sends [HTTP](/learn/networking/http/) by hand from the shell, so you can test
an API or a site without a browser. Fetch a page with `curl URL`; read the response
with **headers (-I)** or verbose **-v**; and shape a request with
**-X** (method), **-d** (data), and **-H** (headers). Add **-L** to follow redirects
and **-o** to save. It turns any web request into something you can repeat, script,
and inspect.
</div>

A browser hides the request behind a click. curl does the opposite: it makes the exact
request visible and puts every part of it under your control. That is precisely what you
want when something is misbehaving and you need to know what the server actually said.

## The basic request

Give curl a URL and it fetches the response body and prints it:

```sh
curl https://example.com
```

That is the whole "hello world" of curl — it fires off a `GET` request and dumps the
returned HTML (or JSON, or whatever) to your terminal. When you pass a large file or hit
a slow server, curl draws a progress meter that clutters the output; add `-s` (silent) to
quiet it:

```sh
curl -s https://example.com/api/status
```

## Seeing the response

The body is only half the story. The [status code and headers](/learn/networking/http/)
tell you whether the request worked and how the server wants you to treat the answer.
curl exposes them with three flags:

- `-I` — fetch the **headers only**, no body. Perfect for a quick "is it up, and what
  status does it return?" check.
- `-i` — print the **headers and the body**, headers first.
- `-v` — **verbose**: show the whole exchange, including the request curl sent, the TLS
  handshake, and every response header.

```sh
curl -I https://example.com
```

```
HTTP/2 200
content-type: text/html; charset=UTF-8
content-length: 1256
```

Reach for `-v` when a request fails for no obvious reason — it reveals redirects, header
mistakes, and TLS problems that `-I` alone would hide.

## Sending data

To do more than read, change the method and attach a body. `-X` sets the method and `-d`
carries the data:

```sh
curl -X POST -d 'name=matt&system=P25' https://example.com/register
```

For a JSON API, send the JSON as the body and tell the server how to read it with a
`Content-Type` header via `-H`:

```sh
curl -X POST \
  -H 'Content-Type: application/json' \
  -d '{"name":"matt","system":"P25"}' \
  https://example.com/api/register
```

`-H` also carries authentication, such as an API token:

```sh
curl -H 'Authorization: Bearer TOKEN' https://example.com/api/private
```

Keep secrets in headers, **never in the URL** — URLs end up in logs, browser history, and
`Referer` headers, so a token in the query string leaks. This request/response shaping is
exactly how you exercise a [REST API](/learn/networking/web-apis-and-rest/) by hand.

## Following redirects & downloading

By default curl does **not** follow a `3xx` redirect — it prints the redirect response and
stops. Add `-L` to follow the chain to the final destination:

```sh
curl -L https://example.com/download/latest
```

To save the result instead of printing it, use `-o` to name the file yourself, or `-O` to
reuse the name from the URL:

```sh
curl -L -o gophertrunk.tar.gz https://example.com/releases/gt.tar.gz
curl -L -O https://example.com/releases/gophertrunk.tar.gz
```

When downloading is all you want, **wget** is the download-focused alternative: it saves
to a file by default, resumes interrupted transfers, and can fetch recursively. curl is
the request inspector; wget is the file grabber.

```sh
wget https://example.com/releases/gophertrunk.tar.gz
```

## Why it's a superpower

Because a curl command is just text, every request is **reproducible and scriptable**. You
can paste it into a bug report, drop it in a shell script, or run it in a loop to watch a
service. When an API returns the wrong thing, one curl line with `-v` shows you the exact
status, headers, and body — no browser guesswork. It is the natural next step after the
basic [connectivity checks](/learn/networking/testing-connectivity/): once you know the
host is reachable, curl tells you what it is actually saying.

<div class="knowledge-check" data-quiz data-correct-msg="Right — -I fetches just the headers, no body." markdown="0">
  <p class="knowledge-check__q">Quick check: which curl flag shows only the response headers?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">-d</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">-I</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">-L</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **curl** sends an HTTP request from the shell and prints the response; `-s` quiets the
  progress meter.
- Read the response with `-I` (headers only), `-i` (headers + body), or `-v` (the whole
  exchange, including TLS).
- Shape a request with `-X` (method), `-d` (body), and `-H` (headers) — send JSON with a
  `Content-Type` header, and keep secrets out of the URL.
- `-L` follows redirects; `-o` and `-O` save to a file; **wget** is the download-focused
  alternative.
- Because it is plain text, curl makes requests **reproducible and scriptable** — ideal
  for diagnosing an API or a service.

Next up: [SSH tunnels & transfers](/learn/networking/ssh-tunnels-and-transfers/)
