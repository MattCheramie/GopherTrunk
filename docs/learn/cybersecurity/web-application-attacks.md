---
slug: web-application-attacks
title: Web application attacks
description: How the most common web vulnerabilities work and how to prevent them — why injection, cross-site scripting (XSS), and cross-site request forgery (CSRF) share one root cause, plus the framework-level defenses (parameterized queries, output encoding, anti-CSRF tokens) that stop them.
keywords: web security, SQL injection, XSS, CSRF, OWASP, input validation, parameterized queries, output encoding, content security policy, secure coding, web application attacks
level: advanced
status: full
prereq:
  - common-attacks
faq:
  - q: "What do most web application attacks have in common?"
    a: "Almost all of them mix untrusted input with code. Whether it's a database query, an HTML page, or a shell command, the vulnerability appears when data the attacker controls gets treated as instructions. Fix the mixing and the whole class of attack goes away."
  - q: "How is SQL injection prevented?"
    a: "By keeping the query structure and the user data separate — parameterized queries (prepared statements) or a well-used ORM send the query and the values on different channels, so input can never change what the query means. Trying to blocklist 'bad characters' is fragile and not the real fix."
  - q: "What stops cross-site scripting (XSS)?"
    a: "Encode output for the context it lands in so untrusted text is displayed, never executed. Modern frameworks auto-escape by default, and a Content Security Policy adds a second layer that limits what scripts a page is allowed to run."
  - q: "Do I need to memorise every web vulnerability?"
    a: "No. Understand the shared root cause — data being treated as code — and apply the standard defenses by design. The OWASP Top 10 is a useful prioritized checklist to make sure you haven't missed a common category."
---

# Web application attacks

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Most web vulnerabilities share **one root cause**: **untrusted input mixing with
code** — a query, a page, or a request that treats attacker-controlled data as
instructions. The big classes — **injection, XSS, and CSRF** — are all versions
of that same mistake. The reliable answer is to **prevent by design**: keep data
and code on separate channels and lean on framework defenses that do this for
you. Builds on [common attacks](/learn/cybersecurity/common-attacks/).
</div>

Web applications take input from strangers and turn it into database queries, HTML
pages, and outbound requests. That is exactly where things go wrong. The good news
is that the most damaging vulnerabilities are not exotic — they come from a handful
of predictable patterns, and each has a well-understood defense that frameworks
now provide by default.

This lesson is about understanding **why** each class exists and **how to prevent
it**. It deliberately shows no working attack strings — you don't need a weapon to
understand the wound, and the prevention is the useful part.

## The common root cause

Step back and the major web vulnerabilities rhyme. In each one, a program builds
something out of two ingredients — **trusted code the developer wrote** and
**untrusted data a user supplied** — and then loses track of which is which. When
the data is allowed to change the meaning of the code, an attacker who controls the
data controls the program.

That is the same "data treated as instructions" problem behind
[prompt injection](/learn/building-ai/prompt-injection-and-security/) in AI systems:
the model can't tell the developer's instructions apart from text that arrived in a
document or a web page. Different technology, identical failure mode. Once you see
the pattern, the defenses all look like variations on one idea — **keep data and
code separate, and never let input cross into the instruction channel**.

## Injection (e.g. SQL injection)

**The concept.** A database query is code. When an application builds that code by
pasting user input directly into the query text, the input can change what the query
*means* — not just the value it's searching for, but the structure of the query
itself. A field that was supposed to hold a username becomes a way to rewrite the
whole request. That is SQL injection, and the same idea applies to any interpreter:
operating-system commands, LDAP lookups, template engines.

**Why it exists.** String-building is the intuitive way to assemble a query, and it
works perfectly in testing where nobody is hostile. The gap only shows up when input
is chosen to break out of the value and into the code.

**Prevention.** Never build queries by gluing strings together. Use **parameterized
queries** (also called prepared statements), where the query structure is fixed up
front and the user's values are sent separately — the database treats them strictly
as data, so they can't alter the query's meaning. A well-used **ORM** does this for
you. Most database libraries make the safe path the easy path; the job is to always
take it. See [databases and persistence](/learn/intro-software-dev/databases-and-persistence/)
for how query parameters work.

## Cross-site scripting (XSS)

**The concept.** A web page is also code — HTML and JavaScript the browser executes.
When an application takes untrusted input and drops it into a page without treating
it as plain text, that input can carry markup or script that then **runs in the
victim's browser**, with the victim's session and permissions. The attacker's code
executes on a page the user trusts.

**Why it exists.** Displaying user content — comments, names, search terms — is core
to most web apps, and it's easy to insert that content into a page as-is. If the app
doesn't mark it as "display only," the browser has no way to know it wasn't meant to
run.

**Prevention.** **Encode (escape) output** for the context it lands in, so that
untrusted characters are shown literally rather than interpreted as markup. Modern
frameworks **auto-escape by default** in their templates — a big reason to use them
rather than assembling HTML by hand. Add a **Content Security Policy (CSP)** as a
second layer: it tells the browser which scripts are allowed to run, so even a slip
is contained. Validate input on the way in, but rely on output encoding as the real
defense — the same string is safe in one context and dangerous in another.

## Cross-site request forgery (CSRF)

**The concept.** Browsers automatically attach a user's cookies to requests aimed at
a site the user is logged into. CSRF abuses that: a malicious page gets the victim's
browser to **fire off an authenticated request** to another site — say, "change my
email" — without the user meaning to. The server sees a valid, logged-in request and
acts on it. Nothing was stolen; the victim's own browser was tricked into asking.

**Why it exists.** Ambient cookie authentication is convenient — you stay logged in —
but it means the browser proves *who you are* on every request, including ones a
different site provoked.

**Prevention.** Require proof that the request came from your own application, not
another site. **Anti-CSRF tokens** — an unpredictable value tied to the session and
checked on each state-changing request — do this, because an attacker's page can't
read or guess the token. Set **`SameSite` cookies** so the browser won't send session
cookies on cross-site requests in the first place. Frameworks bundle both; turn them
on for anything that changes state.

## The OWASP idea

You could try to memorise every vulnerability, but a better move is to work from a
**shared, prioritized list** of what actually goes wrong most often. That is the value
of the **OWASP Top 10**: a community-maintained ranking of the most common and
impactful web application security risks, updated as the landscape shifts.

Used well, it's a **checklist** — a way to confirm you've considered each major
category (injection, broken access control, misconfiguration, and the rest) before
you ship, rather than discovering a whole class of bug in production. It points you
at the risks and the defenses without handing anyone a set of exploits.

## Defend by design

None of these defenses are things you bolt on at the end. They're design choices:

- **Validate input** — check that data matches what you expect (type, length, range,
  format) and reject the rest. This is a useful filter, not a complete defense on its own.
- **Encode output** — escape data for the exact context it's rendered into, so it's
  displayed and never executed.
- **Use safe framework features** — auto-escaping templates, **prepared statements**,
  built-in CSRF protection. The safe path should be the default path.
- **Keep dependencies patched** — a known vulnerability in a library you use is a
  vulnerability in your app. Update regularly.
- **Test for it** — include security cases in your [testing](/learn/intro-software-dev/testing/),
  and review code for the patterns above. See [secure coding](/learn/cybersecurity/secure-coding/)
  for the habits that keep these bugs out from the start.

The theme throughout: assume every input is hostile, keep data out of the instruction
channel, and let well-worn framework defenses carry the load.

<div class="knowledge-check" data-quiz data-correct-msg="Right — separating the query structure from the data is what makes injection impossible, not filtering input." markdown="0">
  <p class="knowledge-check__q">Quick check: SQL injection is prevented mainly by...</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Blocklisting suspicious characters in input</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Parameterized queries / prepared statements — not by trying to blocklist bad input</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Hiding the database behind a firewall</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Most web vulnerabilities share **one root cause**: untrusted input mixing with code,
  so data is treated as instructions.
- **Injection** changes the meaning of a query; **prepared statements / parameterized
  queries** keep data and query structure separate.
- **XSS** runs attacker input in a victim's browser; **output encoding**, a **CSP**,
  and framework auto-escaping stop it.
- **CSRF** tricks a logged-in browser into a request; **anti-CSRF tokens** and
  **`SameSite` cookies** prevent it.
- The **OWASP Top 10** is a prioritized checklist of the common risks, not a set of exploits.
- **Defend by design**: validate input, encode output, use safe framework features,
  patch dependencies, and test.

Next up: [network attacks](/learn/cybersecurity/network-attacks/)
