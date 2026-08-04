---
slug: forms-and-user-input
title: Forms & user input
description: Forms are how the web takes input. This lesson covers how HTML forms collect and submit data, the difference between GET and POST submissions, client-side versus server-side validation, and the security rule that underlies all of it — never trust input from the browser.
keywords: forms, user input, form validation, GET POST, input fields, client-side validation, server-side validation, never trust user input, form submission, HTML forms
level: intermediate
status: full
prereq:
  - the-dom
faq:
  - q: How does an HTML form send data to the server?
    a: "A form has a set of input fields and a submit action. When submitted, the browser gathers the fields' values and sends them to the URL in the form's `action`, using the method in its `method` — usually **GET** (values in the URL) or **POST** (values in the request body). The server receives them as a request and processes them."
  - q: What's the difference between client-side and server-side validation?
    a: "**Client-side** validation runs in the browser for fast, friendly feedback before submitting. **Server-side** validation runs on the server after the data arrives. You need both — client-side for experience, server-side for safety — because the browser's checks can be bypassed and the server is the only place you actually control."
  - q: Why should I never trust user input?
    a: "Because anything sent from a browser can be altered, faked, or crafted maliciously — the user controls their own device and the requests it sends. Treating input as untrusted, and validating and sanitising it on the server, is the foundation that prevents whole classes of attacks like injection and cross-site scripting."
---

# Forms & user input

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Forms** are how the web collects input — fields the user fills in and submits.
The browser gathers the values and sends them to the server, usually by **GET**
(values in the URL) or **POST** (values in the body). **Validation** happens in two
places: **client-side** in the browser for fast, friendly feedback, and
**server-side** for safety — and you need **both**, because the golden rule is
**never trust input from the browser**. This is where user data crosses from the
front end into your back end, so it's where security begins. Builds on
[the DOM](/learn/web-dev/the-dom/) and the client/server split from earlier.
</div>

So far the user has only *looked* at pages. Forms are how they *talk back* —
typing a search, logging in, submitting a report. A form is the main doorway
through which data crosses from the browser into your system, which makes it two
things at once: the primary tool of interactivity, and the primary place security
has to start. This lesson covers both faces.

## The anatomy of a form

An HTML **form** groups input controls and defines what happens when they're
submitted. The `<form>` element wraps the fields; each `<input>` (or `<textarea>`,
`<select>`) collects one value; a submit control sends the lot:

```html
<form action="/api/report" method="post">
  <label for="tg">Talkgroup</label>
  <input id="tg" name="talkgroup" type="text" required>

  <label for="note">Note</label>
  <textarea id="note" name="note"></textarea>

  <button type="submit">Submit report</button>
</form>
```

Two things do the real work. The **`name`** on each field becomes the key the
server receives the value under (`talkgroup=...`). And the pairing of `<label>`
with an input's `id` (via `for`) ties the label to the field — which matters for
usability and, as [accessibility basics](/learn/web-dev/accessibility-basics/)
will stress, is essential for screen-reader users. When submitted, the browser
bundles every named field and sends it to the form's `action`.

## GET and POST

A form's `method` decides *how* the data is sent, and the choice mirrors the HTTP
methods from [HTTP — the web's protocol](/learn/networking/http/):

- **GET** appends the values to the URL as a **query string** (`/search?q=fire`).
  Use it for requests that only **read** — searches, filters — where it's fine for
  the values to show in the URL, be bookmarked, and be repeated safely.
- **POST** puts the values in the **request body**, out of the URL. Use it for
  anything that **changes state** — creating a record, logging in, submitting a
  report — and for anything sensitive, since body data isn't sitting in the address
  bar or browser history.

The rule of thumb: **GET to read, POST to change**. A login form is always POST; a
search box is usually GET.

## Reading input with JavaScript

Forms work with no JavaScript at all — the browser can submit them directly. But
JavaScript lets you read and react to input *as it happens*, using the DOM skills
from the last lesson, for things like live feedback or updating the page without a
full submit:

```js
const form = document.querySelector("form");

form.addEventListener("submit", (event) => {
  event.preventDefault();               // stop the default page reload
  const tg = form.talkgroup.value;      // read a field by name
  console.log("Submitting talkgroup:", tg);
  // ...validate, then send with fetch()...
});
```

Calling `preventDefault()` stops the browser's default submit-and-reload, letting
you handle the data yourself — validate it, then send it in the background with
`fetch` ([fetching data](/learn/web-dev/fetching-data/)). This is how modern apps
submit forms without the page flashing and reloading.

## Validation: two places, two jobs

**Validation** means checking that input is well-formed and acceptable — a real
email, a required field filled, a number in range. It happens in two places, and
they do different jobs:

- **Client-side** (in the browser) is about **experience**. HTML attributes like
  `required`, `type="email"`, and `min`/`max` — plus JavaScript checks — give
  instant feedback *before* the user submits, so they fix mistakes without a round
  trip to the server. It's fast and friendly.
- **Server-side** (on the back end) is about **safety and truth**. After the data
  arrives, the server checks it again before acting on it or storing it.

You need **both**, and it's vital to understand *why*: client-side validation is a
convenience that can be **completely bypassed**. A user can edit the page, disable
JavaScript, or send a request straight to your server without ever loading your
form. So the browser's checks improve the experience but guarantee nothing.

## Never trust the browser

This is the security principle the whole lesson has been building toward, and it
outlives any specific technique: **never trust input from the browser.**

Everything the browser sends — form fields, URL parameters, headers, the request
itself — comes from a device the **user controls** and can be altered or forged.
Recall from [JavaScript in the browser](/learn/web-dev/javascript-in-the-browser/)
that client-side code is untrusted; the data it sends is untrusted for the same
reason. So on the server you always:

- **Validate** — reject anything that isn't the shape and range you expect, even if
  the form "should" have prevented it.
- **Sanitise / escape** — treat input as data, never as code. Input pasted straight
  into a database query or into a page is how **SQL injection** and
  **cross-site scripting (XSS)** happen, both covered in
  [web application attacks](/learn/cybersecurity/web-application-attacks/) and
  [web security essentials](/learn/web-dev/web-security-essentials/).
- **Authorise** — confirm the user is actually allowed to do what the request asks,
  no matter what the front end appeared to permit.

Internalise this and a huge swath of web vulnerabilities simply never opens up.
Forms are the friendly face of user input; "never trust the browser" is the
discipline behind that face.

<div class="knowledge-check" data-quiz data-correct-msg="Right — client-side validation can be bypassed, so the server must validate again; you need both." markdown="0">
  <p class="knowledge-check__q">Quick check: your form validates input in the browser. Is server-side validation still needed?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">No — if the browser checks it, the data is guaranteed clean</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Yes — browser checks can be bypassed, so the server must validate too</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">No — server-side validation is only for GET requests</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Forms** collect input in named fields and submit it to the server at the form's
  `action`.
- The **`name`** attribute becomes the key the server reads; pairing `<label>` with
  an input's `id` is essential for usability and accessibility.
- **GET** sends values in the URL (for reads); **POST** sends them in the body (for
  changes and sensitive data) — **GET to read, POST to change**.
- JavaScript can intercept submission with `preventDefault()` to validate and send
  data in the background with `fetch`.
- **Validate in both places**: client-side for **experience**, server-side for
  **safety** — client checks can be bypassed.
- **Never trust the browser** — validate, sanitise, and authorise all input on the
  server, which shuts down whole classes of attacks.

Next up: [Accessibility basics](/learn/web-dev/accessibility-basics/).
