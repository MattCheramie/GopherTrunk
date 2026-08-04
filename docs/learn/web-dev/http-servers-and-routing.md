---
slug: http-servers-and-routing
title: HTTP servers & routing
description: How a server answers the web — a listening loop that accepts HTTP requests, a router that maps a URL path and method to the right handler function, and a handler that builds and returns the response, shown with a small Go example.
keywords: HTTP server, routing, router, handler, request handler, listen, port, URL path, HTTP method, Go http server, middleware, ServeMux
level: intermediate
status: full
prereq:
  - what-a-backend-does
faq:
  - q: What is a route?
    a: "A route is a rule that maps an incoming request — identified by its HTTP method and URL path, like GET /api/calls — to the handler function that should answer it. The router is the part of the server that looks at each request and picks the matching route, so the right code runs for each URL."
  - q: What does 'listening on a port' mean?
    a: "A server binds to a port number (like 8080) on the machine and waits for incoming connections there. The port is the address within the machine that clients connect to; a URL's host and port tell the network which machine and which listening program should receive the request."
  - q: What is a handler?
    a: "A handler is the function that runs for a matched route. It receives the request (path, method, headers, body) and a way to write the response, does whatever work is needed — read a database, check auth — and writes back a status code, headers, and a body. One handler per kind of request."
---

# HTTP servers & routing

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **HTTP server** is a program that **listens on a port**, accepts incoming
[HTTP requests](/learn/networking/http/), and returns responses — running the loop
forever. A **router** inspects each request's **method and path** (like `GET /api/calls`)
and dispatches it to the matching **handler** function. The **handler** does the work —
read a [database](/learn/web-dev/backend-and-database/), check auth — and writes back a
**status code, headers, and body**. **Middleware** wraps handlers to run shared work
(logging, auth) around every request. This is the machinery behind every
[back end](/learn/web-dev/what-a-backend-does/).
</div>

The [last lesson](/learn/web-dev/what-a-backend-does/) said the back end runs on a
server; this one is what that server actually *is* as a program. Strip away the framework
names and it's three things: something that **listens** for requests, something that
**routes** each one to the right code, and the **handler** code that produces a response.
Every web framework in every language — Express, Flask, Rails, Go's standard library — is
a variation on those three parts. We'll use Go, since it makes the shape especially clear.

## The listening loop

A server's outermost job is to **listen** on a **port** and accept connections. It binds
to a port number on the machine — say `8080` — and then waits, handling each request that
arrives and looping forever. A client reaches it via the host and port in a URL; the
network delivers the request to whichever program is listening there.

```go
package main

import (
    "fmt"
    "net/http"
)

func main() {
    mux := http.NewServeMux()          // the router
    mux.HandleFunc("GET /api/calls", listCalls)

    fmt.Println("listening on :8080")
    http.ListenAndServe(":8080", mux)  // bind the port and serve forever
}
```

`ListenAndServe` is the loop: it accepts each connection, parses the HTTP request, and
hands it to the router (`mux`). You write that line once; everything interesting happens
in the routes and handlers it dispatches to. Under the hood the server also runs each
request concurrently, so a slow one doesn't block the rest — but the mental model is
simply "accept a request, route it, respond, repeat."

## Routing: method + path to a handler

A real back end answers many different requests, so it needs to decide **which code**
handles each one. That's **routing**: matching a request's **HTTP method and URL path** to
a **handler**. `GET /api/calls` lists calls; `POST /api/calls` creates one;
`GET /api/calls/42` fetches one by id. The **router** holds a table of these rules and
picks the match.

```go
mux := http.NewServeMux()
mux.HandleFunc("GET  /api/calls",      listCalls)    // read the collection
mux.HandleFunc("POST /api/calls",      createCall)   // create in the collection
mux.HandleFunc("GET  /api/calls/{id}", getCall)      // read one item
```

Note that **the same path with a different method routes to different handlers** —
`GET /api/calls` and `POST /api/calls` are separate routes. That method-plus-path pairing
is exactly the structure a [REST API](/learn/web-dev/building-a-rest-api/) is built on, so
routing and REST design go hand in hand. Bigger apps use richer routers (path parameters,
groups, patterns), but the job never changes: request in, correct handler chosen.

## The handler: build the response

A **handler** is the function that answers a matched route. It receives the **request**
(method, path, headers, body) and a **response writer**, does the work, and writes back a
**status code, headers, and body** — the response shape from the
[HTTP lesson](/learn/networking/http/).

```go
func listCalls(w http.ResponseWriter, r *http.Request) {
    calls := readCallsFromDB()               // do the work (query the database)

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)             // 200
    json.NewEncoder(w).Encode(calls)         // write the JSON body
}
```

That's the whole contract: read what came in, do something trustworthy on the server, and
write a response out. A handler that creates something returns **201 Created**; one that
can't find the item returns **404**; one that hits an internal error returns **500** — the
[status codes](/learn/networking/http/) are how the handler tells the client what
happened. Keep handlers focused: parse and validate the input, call into your logic and
[database](/learn/web-dev/backend-and-database/), and format the response.

## Middleware: shared work around every request

Most requests need the same surrounding work — log the request, check the user is
authenticated, set common headers, recover from a panic. Rather than repeat that in every
handler, you wrap handlers in **middleware**: a function that runs **before and/or after**
the handler, forming a chain the request passes through.

```go
func withLogging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s", r.Method, r.URL.Path)  // before
        next.ServeHTTP(w, r)                        // run the real handler
    })
}
```

The request flows **listener → middleware chain → router → handler**, and the response
flows back out through the same chain. Cross-cutting concerns — authentication (a natural
fit, checking identity once for many routes), logging, rate limiting,
[CORS](/learn/web-dev/what-a-backend-does/) — live in middleware so handlers stay focused
on their one job. This layered shape is universal, whatever the language; learn it once
and every server framework reads the same. If you want to go deeper on Go itself, the
[Go module](/learn/programming-go/) covers the language behind these examples.

<div class="knowledge-check" data-quiz data-correct-msg="Right — routing maps a request's method and path (like GET /api/calls) to the handler function that should answer it." markdown="0">
  <p class="knowledge-check__q">Quick check: what does a router do in an HTTP server?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Encrypts the connection between the browser and the server</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Maps each request's method and path to the handler function that answers it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Stores the app's data so handlers don't need a database</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **HTTP server** binds to a **port**, **listens** for requests, and responds in a
  loop that runs forever.
- A **router** maps each request's **method and path** (like `GET /api/calls`) to the
  matching **handler**; the same path with a different method is a different route.
- A **handler** reads the request, does trusted server-side work, and writes a
  **status code, headers, and body** back.
- **Middleware** wraps handlers to run shared work — logging, auth, CORS — before and
  after the request, keeping handlers focused.
- The request flows **listener → middleware → router → handler** and the response back
  out — the same shape in every language and framework.

Next up: [Building a REST API](/learn/web-dev/building-a-rest-api/).
