---
slug: secure-coding
title: Secure coding
description: A defender's guide to writing safer software — validating untrusted input on the server, keeping data from becoming code with parameterized queries and output encoding, failing closed on errors, managing dependencies, applying least privilege, and catching the rest with tests and code review.
keywords: secure coding, input validation, parameterized queries, output encoding, error handling, dependency security, least privilege, code review, supply chain, secrets management
level: intermediate
status: full
prereq:
  - common-attacks
faq:
  - q: What is the single most important habit in secure coding?
    a: Never let untrusted input become executable code or part of a query. Almost every injection flaw — SQL injection, command injection, cross-site scripting — comes from mixing data and code. Use parameterized queries and safe APIs so input is always treated as data, and the whole class of bug largely disappears.
  - q: Isn't client-side validation enough to keep input safe?
    a: No. Client-side checks improve the user experience, but anyone can bypass them by talking to your server directly. Validation that protects you has to run on the server, where the attacker cannot reach in and turn it off. Treat client-side validation as a convenience, never as a security control.
  - q: How do I keep vulnerable dependencies out of my project?
    a: Most of the code you ship is written by other people, so keep dependencies updated, enable automated alerts like Dependabot to flag known vulnerabilities, and pay attention to the supply chain — who publishes a package and whether it is trustworthy. Update on a schedule rather than waiting for an incident.
  - q: Where should secrets like API keys and passwords live?
    a: Not in your source code. Keep secrets out of the repository and out of version history — use environment variables or a secrets manager, and grant each part of the system only the minimum permissions it needs. If a secret ever lands in a commit, rotate it, because history is forever.
---

# Secure coding

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Most software vulnerabilities come from a handful of missed habits, not exotic
attacks. **Validate input** — treat everything from outside as hostile until
checked. **Don't mix data and code** — the root of injection. **Manage
dependencies** — most of what you ship is other people's code. **Fail safely** —
close down when something goes wrong instead of leaking. Get these right and you
close off most of the [common attacks](/learn/cybersecurity/common-attacks/)
before they start.
</div>

Secure coding isn't a separate phase you bolt on at the end — it's a set of small
defaults you build into how you write software every day. The goal here is
educational: understand *why* each habit works so you reach for it automatically,
the way you already reach for a seatbelt.

## Never trust input

The first rule of defensive coding: everything that crosses into your program
from outside is **untrusted until you've validated it**. That means data from
users, from the network, from files, from other services — anything you didn't
generate yourself under controlled conditions. Assume it may be malformed,
oversized, or crafted specifically to break you.

Validation means checking that input matches what you expect *before* you act on
it: the right type, a sane length, an allowed set of values. Prefer an
**allow-list** ("only these characters/values are acceptable") over a deny-list
("block these bad ones"), because you can never list every bad input an attacker
might try.

The crucial part: **validate on the server, not just the client**. A check in the
browser is a nicety for honest users, but an attacker simply skips the browser and
talks to your server directly. Only server-side validation actually protects you —
a theme that runs through most
[web application attacks](/learn/cybersecurity/web-application-attacks/).

## Don't let data become code

If you remember one thing, make it this: **injection happens when untrusted data
gets treated as code**. A username becomes part of a database query; a filename
becomes part of a shell command; a comment becomes part of a web page. In each
case the attacker's data slips across the line and starts *executing*.

The fix is structural, not clever. Keep data as data:

- **Parameterized queries / prepared statements** send the query structure and the
  values separately, so user input can never change what the query *does* — only
  the values it operates on. This is the standard defense against SQL injection.
  (See [databases & persistence](/learn/intro-software-dev/databases-and-persistence/)
  for how queries are built.)
- **Safe APIs** that take arguments as a list, rather than building a command
  string, keep the same separation for shells and other interpreters.
- **Output encoding** protects the other direction: when you place data into HTML,
  a URL, or JSON, encode it for *that context* so it renders as text instead of
  being interpreted as markup. This is what stops cross-site scripting.

It's worth noticing this is the same lesson that shows up in AI systems as
[prompt injection](/learn/building-ai/prompt-injection-and-security/): once
untrusted content and trusted instructions share one channel, an attacker can
blur the boundary. Keeping data and code apart is a universal defense — see
[web application attacks](/learn/cybersecurity/web-application-attacks/) for the
attacker's side of it.

## Handle errors safely

Things will go wrong — the question is what your code does when they do. Secure
code **fails closed**: when a permission check errors out or a resource is missing,
the safe default is to deny, not to wave the request through.

Two more rules for error handling:

- **Don't leak in error messages.** A stack trace, a database error, or a file path
  handed to the user tells an attacker exactly how your system is built. Show a
  generic message to the user; keep the detail in your logs.
- **Log securely.** Logs are invaluable for spotting attacks, but never write
  passwords, tokens, or personal data into them — a leaked log file shouldn't be a
  second breach.

The broader craft of failing gracefully is covered in
[robustness & error handling](/learn/intro-software-dev/robustness-and-errors/);
here the security angle is simply: fail *safe*, and stay quiet about your internals.

## Manage your dependencies

Most of the code that runs in your application, you didn't write. Libraries,
frameworks, and packages make up the bulk of a modern project, and every one of
them is code you're responsible for — including its bugs.

- **Keep dependencies updated.** Known vulnerabilities get patched; running an old
  version leaves a published, well-documented hole open.
- **Watch for known vulnerabilities automatically.** Tools like **Dependabot** scan
  your dependencies and alert you when one has a reported flaw, so you're not
  checking by hand. (See [Git CLI, LFS & security](/learn/git/cli-lfs-and-security/)
  for enabling these on a repository.)
- **Mind the supply chain.** A package can be safe today and compromised tomorrow if
  its maintainer's account is hijacked or a malicious update slips in. Pay attention
  to who publishes what you depend on — the
  [language & ecosystem security](/learn/intro-software-dev/language-security/)
  lesson goes deeper on this.

## Least privilege & secrets

**Least privilege** means every part of your system runs with the *minimum*
permissions it needs and nothing more. A web app that only reads a table shouldn't
connect to the database as an admin; a service that never touches the filesystem
shouldn't run as root. When something does get compromised, least privilege limits
how far the damage spreads.

**Keep secrets out of code.** API keys, passwords, and tokens don't belong in your
source or its history — use environment variables or a dedicated secrets manager,
and rotate anything that leaks. The details of doing this well live in
[secrets management](/learn/cybersecurity/secrets-management/), and deciding who is
allowed to do what is the subject of
[authorization & access control](/learn/cybersecurity/authorization-and-access/).

## Test and review

Good habits catch most flaws, but not all — so build a net for the rest.

- **Automated tests** should include the security-relevant cases: what happens with
  empty input, oversized input, or input designed to break out of its context. See
  [testing](/learn/intro-software-dev/testing/) for how to structure them.
- **Code review** puts a second set of eyes on every change. A reviewer who asks
  "where does this input come from?" catches injection bugs the author didn't see.
- **Security scanning** — automated tools that read your code or dependencies for
  known-bad patterns — flags issues at scale, cheaply, before they ship. The
  [security for developers](/learn/cybersecurity/security-for-developers/) lesson
  ties these into a workflow.

None of these replaces the habits above; they exist to catch the day you forget one.

<div class="knowledge-check" data-quiz data-correct-msg="Right — parameterized queries and safe APIs keep input as data, so it can never be executed as code." markdown="0">
  <p class="knowledge-check__q">Quick check: the most reliable way to prevent injection in your code is to...</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Scan input for a list of known bad words and reject any that match</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Keep untrusted input as data (parameterized queries, safe APIs) — never build code/queries by string concatenation</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Validate the input in the browser before it's submitted</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Treat **all outside input as hostile** until validated — and validate on the
  **server**, not just the client.
- **Don't let data become code**: parameterized queries, safe APIs, and
  context-aware output encoding shut down injection.
- **Fail closed**, keep stack traces and secrets out of error messages, and log
  without leaking.
- **Manage dependencies** — update them, watch for known vulnerabilities, and mind
  the supply chain.
- Run with **least privilege**, keep secrets out of code, and let **tests, review,
  and scanning** catch what habits miss.

Next up: [securing networks & services](/learn/cybersecurity/securing-networks-and-services/)
