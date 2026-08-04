---
slug: fetching-data
title: Fetching data from the front end
description: How a page pulls fresh data without reloading — the browser's fetch API, sending and parsing JSON, async/await, and the loading-error-success states every real request has to handle when the front end talks to a back-end API.
keywords: fetch, fetch API, JSON, async await, promise, HTTP request from browser, loading state, error handling, calling an API, XHR, fetching data
level: intermediate
status: full
prereq:
  - components-and-state
faq:
  - q: What is fetch?
    a: "fetch is the browser's built-in function for making HTTP requests from JavaScript. You give it a URL (and options like method and body), and it returns a promise that resolves to a response object. It's how a page asks a back-end API for data without reloading — the modern replacement for the older XMLHttpRequest."
  - q: Why do I have to call response.json()?
    a: "Because the response arrives as a stream of bytes, and reading the body is itself asynchronous. fetch resolves as soon as the headers are in; response.json() reads the rest of the body and parses it from JSON text into a JavaScript object. It returns a promise, so you await it too."
  - q: Does a 404 or 500 make fetch throw an error?
    a: "No — and this trips up almost everyone. fetch only rejects on a network failure (no connection, DNS error). An HTTP error like 404 or 500 is a successful round-trip as far as fetch is concerned, so you must check response.ok yourself and handle the bad status."
---

# Fetching data from the front end

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A modern page pulls fresh data **without reloading** by making HTTP requests from
JavaScript with the browser's **fetch** API. Requests are **asynchronous** — they take
time and return a **promise** — so you handle them with **async/await**. Data usually
travels as **JSON**, which you parse from the response. Every real fetch has three
states you must design for — **loading**, **error**, and **success** — and, crucially,
**fetch does not throw on a 404 or 500**, so you check the [status](/learn/networking/http/)
yourself. The result flows into component [state](/learn/web-dev/components-and-state/)
and the UI re-renders.
</div>

So far the data driving a component came from props or a click. In a real app most of
it comes from a **server** — a list of decoded calls, a user profile, search results.
This lesson is how the browser goes and gets that data over
[HTTP](/learn/networking/http/) and folds it into the UI, all without the full-page
reload that older sites needed. It is the bridge between the front-end mental model and
the back-end half of this module.

## fetch: an HTTP request from JavaScript

The browser gives you **fetch**, a function that makes an HTTP request from your code.
You pass a URL; it returns a **promise** that resolves to a **response** object. Reading
the body — here, parsing it as JSON — is a second asynchronous step. The cleanest way to
write it is **async/await**, which lets asynchronous code read top to bottom:

```js
async function getCalls() {
  const response = await fetch("/api/calls");   // send the request, wait for headers
  const data = await response.json();           // read + parse the JSON body
  return data;                                  // a normal JavaScript array/object
}
```

`fetch` is asynchronous because a network request takes real time — often much longer
than anything happening locally, as [treating a remote call](/learn/networking/http/)
taught. `await` pauses this function until the response is ready **without freezing the
page**; the browser stays responsive while the request is in flight. Under the hood
`await` is just friendlier syntax over promises and their `.then()` chains.

## JSON: the data format on the wire

Data almost always crosses the network as **JSON** — JavaScript Object Notation — a
plain-text format of objects, arrays, strings, numbers, and booleans that both sides
understand. It maps so cleanly onto JavaScript values that `response.json()` turns the
text straight into usable objects.

```json
{
  "calls": [
    { "id": 1, "talkgroup": "Fire-1", "seconds": 12 },
    { "id": 2, "talkgroup": "EMS-3",  "seconds": 4  }
  ]
}
```

To *send* data — say, saving a setting — you go the other way: set the method and
headers, and put a JSON string in the body.

```js
await fetch("/api/settings", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ theme: "dark" }),
});
```

The `Content-Type` header tells the server the body is JSON, and `JSON.stringify` turns
your object into the text that rides in the request. This is the same request shape the
[REST API](/learn/web-dev/building-a-rest-api/) on the back end is built to receive.

## fetch doesn't throw on HTTP errors

Here is the gotcha that catches everyone. `fetch` **only rejects on a network failure** —
no connection, DNS error, request never completes. An HTTP error status like **404 Not
Found** or **500 Internal Server Error** is, to `fetch`, a perfectly successful
round-trip: the server answered. So you must check the status yourself with
`response.ok` (true for 2xx) and handle a bad one:

```js
async function getCalls() {
  const response = await fetch("/api/calls");
  if (!response.ok) {
    throw new Error(`Request failed: ${response.status}`);   // 404, 500, ...
  }
  return response.json();
}
```

Skip this check and a 500 sails silently past `try/catch`, and your code tries to read
JSON out of an error page. Reading the [status code](/learn/networking/http/) — 2xx
worked, 4xx your request, 5xx the server — is the whole job here.

## Loading, error, success: three states

A request is not instant, and it can fail, so a fetch is never just "the data." It is
**three states** the UI has to represent: **loading** while it's in flight, **error** if
it fails, and **success** with the data. Design for all three or the page looks broken
the moment the network is slow. In a [component](/learn/web-dev/components-and-state/),
each is a piece of state:

```jsx
function Calls() {
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    getCalls().then(setData).catch(setError);   // runs after first render
  }, []);

  if (error) return <p>Couldn't load calls.</p>;   // error state
  if (!data) return <p>Loading…</p>;               // loading state
  return <ul>{data.calls.map((c) => <li key={c.id}>{c.talkgroup}</li>)}</ul>;
}
```

The `useEffect` hook runs the fetch **after** the component first renders — you don't
fetch during render itself — and each resolved result drops into state, re-rendering the
UI. This loading-error-success shape is so universal that libraries like React Query and
SWR exist to manage it, plus caching and retries, so you don't hand-write it every time.

## Where fetch fits in the bigger picture

Fetching is the front end's half of a conversation whose other half is the back end. The
browser calls an endpoint like `/api/calls`; a [HTTP server](/learn/web-dev/http-servers-and-routing/)
routes it to a handler, which reads a [database](/learn/web-dev/backend-and-database/) and
returns JSON. Everything you send and receive rides on
[HTTP](/learn/networking/http/) and, in production, over
[HTTPS](/learn/networking/tls-and-https/) so the data is encrypted in transit. Fetching
is also what makes a **single-page app** feel live — swapping data in place instead of
reloading, a rendering style we compare in
[SSR vs. SPA vs. static](/learn/web-dev/ssr-spa-static/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — fetch only rejects on a network failure, so you must check response.ok (or the status) to catch a 404 or 500." markdown="0">
  <p class="knowledge-check__q">Quick check: your fetch gets a 500 response but your try/catch never fires. Why?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A 500 is impossible from a real server, so nothing happened</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">fetch only rejects on network failures — you must check response.ok for HTTP errors</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">async/await swallows all errors, so catch never runs</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A page pulls fresh data **without reloading** using the browser's **fetch** API,
  which returns a **promise** you handle with **async/await**.
- Requests are **asynchronous** because the network takes time; `await` pauses your
  function without freezing the page.
- Data travels as **JSON** — parse it with `response.json()`, and send it with
  `JSON.stringify` plus a `Content-Type: application/json` header.
- **fetch does not throw on 404 or 500** — check `response.ok` (or the status code)
  and handle HTTP errors yourself.
- Every real fetch has **loading, error, and success** states; each is a piece of
  component [state](/learn/web-dev/components-and-state/) that re-renders the UI.
- Fetching is the front-end half of a conversation with a
  [back-end API](/learn/web-dev/building-a-rest-api/) over
  [HTTP](/learn/networking/http/).

Next up: [Build tools &amp; bundlers](/learn/web-dev/build-tools-and-bundlers/).
