---
slug: web-security-essentials
title: Web security essentials
description: The vulnerabilities every web developer must understand — cross-site scripting (XSS), cross-site request forgery (CSRF), and the cross-origin resource sharing (CORS) rules — how each attack works and the concrete defenses that stop it.
keywords: web security, XSS, cross-site scripting, CSRF, cross-site request forgery, CORS, same-origin policy, input validation, output encoding, content security policy
level: advanced
status: full
prereq:
  - cookies-tokens-jwt
faq:
  - q: "What's the single most important habit for web security?"
    a: "**Never trust input, and never trust the client.** Every value that reaches your server — form fields, URLs, headers, uploaded files, even data from your own JavaScript — could be crafted by an attacker. Validate and encode it on the server. Most web vulnerabilities, XSS included, come down to treating attacker-controlled data as if it were safe."
  - q: "Isn't HTTPS enough to make my site secure?"
    a: "No. **HTTPS protects data *in transit*** — it stops eavesdropping and tampering on the network — but it does nothing about XSS, CSRF, broken authorization, or SQL injection, which attack the application itself. HTTPS is necessary and non-negotiable, but it's the floor, not the ceiling."
  - q: "Does CORS make my API more secure?"
    a: "Not exactly — **CORS relaxes** the browser's same-origin restriction in a controlled way; it doesn't add protection to your server. It decides which *other* origins a browser will let read your API's responses. A permissive CORS policy can expose data, but CORS is enforced by the browser, so it's never a substitute for real server-side authentication and authorization."
---

# Web security essentials

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Three vulnerabilities belong in every web developer's head. **XSS (cross-site
scripting)** runs an attacker's JavaScript in your users' browsers — defended by
**encoding output** and a content security policy. **CSRF (cross-site request
forgery)** tricks a logged-in browser into making a request it didn't intend —
defended by **anti-CSRF tokens** and the [`SameSite`](/learn/web-dev/cookies-tokens-jwt/)
cookie flag. **CORS** is the browser rule governing which *other* origins may read
your responses. The through-line: **never trust input, never trust the client.** The
cybersecurity path covers the attacker's view in
[web application attacks](/learn/cybersecurity/web-application-attacks/).
</div>

Everything you've built so far — [forms](/learn/web-dev/forms-and-user-input/),
[auth](/learn/web-dev/authentication-and-sessions/), a
[REST API](/learn/web-dev/building-a-rest-api/) — is also an attack surface. This
lesson covers the three vulnerabilities you'll meet first and most often. It's a
working developer's introduction, not a complete security course; for the offensive
perspective and more attack classes, see
[web application attacks](/learn/cybersecurity/web-application-attacks/).

## The core mindset: trust nothing from the client

Before the specific attacks, the mindset behind all of them: **any data that comes
from the client can be hostile.** Form fields, query strings, headers, cookies,
uploaded files, and even requests from your own front-end code can all be forged by
someone who bypasses your UI entirely. So every defense in this lesson is a variation
on one rule — **validate input, encode output, and re-check permissions on the
server.** The browser is a convenience for real users and no obstacle at all to an
attacker.

## XSS — cross-site scripting

**XSS** is when an attacker gets their **JavaScript to run in another user's
browser**, inside your site's origin. Once their script runs on your page it can do
anything the user can: read the [DOM](/learn/web-dev/the-dom/), steal data, make
authenticated requests, or lift a token that JavaScript can reach.

The classic path is unescaped user content. Suppose a comment field is written
straight into the page:

```javascript
// DANGEROUS — attacker's markup becomes live HTML
element.innerHTML = "Comment: " + userComment;
// if userComment is  <script>steal(document.cookie)</script>  it runs
```

The defenses:

- **Encode output for its context.** When user data lands in HTML, escape it so
  `<script>` becomes inert text, not a tag. Use your framework's safe rendering
  (React's `{value}`, templating auto-escaping) rather than building HTML by hand.
- **A Content Security Policy (CSP).** An HTTP header that restricts which scripts
  the browser will run, so injected inline script is blocked even if it slips in.
- **HttpOnly cookies.** As covered in
  [cookies & tokens](/learn/web-dev/cookies-tokens-jwt/), an HttpOnly session cookie
  is invisible to JavaScript, so XSS can't read it directly.

The single rule: **treat all user content as data, never as code**, and let a
vetted encoder decide how it's safely displayed.

## CSRF — cross-site request forgery

**CSRF** abuses the browser's helpfulness. Recall that cookies are sent
**automatically** on every request to a site. So if you're logged in to your bank
and then visit a malicious page, that page can quietly submit a request to the
bank — and the browser attaches your session cookie, making it look like *you*:

```html
<!-- On evil.com — fires a request to your bank, with your cookies attached -->
<img src="https://bank.example/transfer?to=attacker&amount=1000">
```

The server can't tell this forged request from a real one, because it *is* your
authenticated session. The defenses close that gap:

- **Anti-CSRF tokens.** The server embeds a secret, per-session token in each form
  and requires it back on state-changing requests. The attacker's page can't read
  it (that's blocked by the same-origin policy), so the forged request fails.
- **`SameSite` cookies.** Setting `SameSite=Lax` or `Strict` tells the browser *not*
  to send the cookie on cross-site requests, cutting off the mechanism CSRF relies
  on. This is now a strong first-line default.

CSRF targets **cookie-based** auth specifically; a [bearer
token](/learn/web-dev/cookies-tokens-jwt/) sent by hand isn't attached
automatically, which is why token APIs are largely immune to it.

## CORS & the same-origin policy

Browsers enforce the **same-origin policy**: by default, JavaScript on one origin
(scheme + host + port) can't *read* responses from a different origin. This quietly
stops a lot of cross-site mischief — a random page can't just read your logged-in
API's data.

**CORS (cross-origin resource sharing)** is the controlled way to *relax* that rule.
When your API needs to be called from a front end on a different origin, it sends
headers naming who's allowed:

```http
Access-Control-Allow-Origin: https://app.example.org
Access-Control-Allow-Credentials: true
```

Two things to keep straight. First, **CORS is enforced by the browser, not your
server** — it governs whether the *browser* hands the response back to JavaScript;
it is not a server-side access control. Second, a wildcard policy
(`Access-Control-Allow-Origin: *`) on an authenticated API can expose data broadly,
so name specific origins. CORS decides who may *read* your responses in a browser;
real protection still comes from
[authentication and authorization](/learn/web-dev/authentication-and-sessions/) on
the server.

## Beyond the big three

XSS, CSRF, and CORS are the essentials, but the same mindset guards the rest:
**parameterised queries** stop SQL injection (never build queries by string
concatenation — see [the database lesson](/learn/web-dev/backend-and-database/)),
**HTTPS everywhere** protects data in transit, and **rigorous authorization checks**
stop one user reaching another's data. Security is a habit applied on every request,
not a feature you add once.

<div class="knowledge-check" data-quiz data-correct-msg="Right — encoding user content so it renders as text, not markup, is the core XSS defense." markdown="0">
  <p class="knowledge-check__q">Quick check: which best prevents XSS when showing a user's comment on a page?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Serving the page over HTTPS instead of HTTP</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Encoding the comment so any markup renders as text, not live HTML</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Trusting it because it came from your own front-end form</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The core mindset is **trust nothing from the client**: validate input, encode
  output, and re-check permissions on the server.
- **XSS** runs an attacker's JavaScript in your users' browsers; defend by
  **encoding output**, using a **Content Security Policy**, and keeping session
  cookies **HttpOnly**.
- **CSRF** tricks a logged-in browser into sending an unintended request; defend with
  **anti-CSRF tokens** and **`SameSite`** cookies — it targets cookie-based auth.
- **CORS** is the browser rule that controls which *other* origins may read your
  responses; it relaxes the same-origin policy and is **not** a server-side access
  control.
- **HTTPS protects transit only** — it doesn't stop XSS, CSRF, or broken
  authorization; those need application-level defenses.

Next up: [caching & CDNs](/learn/web-dev/caching-and-cdns/).
