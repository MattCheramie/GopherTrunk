---
slug: authentication-and-sessions
title: Authentication & sessions
description: How a web app knows who a user is across stateless HTTP requests — logging in, keeping a session, and the crucial difference between authentication (who you are) and authorization (what you're allowed to do).
keywords: authentication, authorization, login, session, session ID, stateless HTTP, password hashing, credentials, sign in, access control
level: intermediate
status: full
prereq:
  - building-a-rest-api
faq:
  - q: "What's the difference between authentication and authorization?"
    a: "**Authentication** answers *who are you* — it verifies identity, usually with a password or token. **Authorization** answers *what are you allowed to do* — it checks whether the now-known user may perform this action. You authenticate once, then authorize every request. A logged-in user is authenticated; whether they can delete another user's data is authorization."
  - q: "Why can't the server just remember I logged in?"
    a: "Because HTTP is **stateless** — each request arrives with no memory of the ones before it. The server holds no per-user connection open. So after you log in, the app hands the browser a **session identifier** that rides along on every later request, and the server uses it to look you up again."
  - q: "Should I ever store passwords in plain text?"
    a: "Never. Store only a **salted hash** made with a slow, purpose-built algorithm (bcrypt, scrypt, or Argon2). When someone logs in you hash what they typed and compare hashes — the real password is never kept, so a database leak doesn't hand attackers the passwords themselves."
---

# Authentication & sessions

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The web is **stateless** — every request stands alone — so an app has to re-learn
who you are on each one. **Authentication** proves *who you are* (logging in);
**authorization** decides *what you may do*. After a successful login the server
issues a **session** identified by a token the browser sends back on every request,
turning a string of anonymous requests into a recognised user. Passwords are never
stored raw — only a **salted hash**. The next lesson,
[cookies, tokens & JWTs](/learn/web-dev/cookies-tokens-jwt/), covers exactly how
that session token travels.
</div>

Once your app has a [REST API](/learn/web-dev/building-a-rest-api/) and a
[database](/learn/web-dev/backend-and-database/), the next thing almost every real
app needs is to know *who is asking*. That sounds simple until you remember how the
web actually works: the server answers one request and forgets everything. This
lesson is about bridging that gap — turning a stream of anonymous, unrelated
requests into a recognised, logged-in user.

## The stateless problem

HTTP has no memory. As the [client-server](/learn/web-dev/client-server-web/)
lesson showed, each request is independent: the server reads it, sends a response,
and moves on holding nothing about you. There is no open phone line between your
browser and the server that stays "you" between clicks.

That is great for scaling — any server can handle any request — but it means the
question *"is this the person who just logged in?"* has no built-in answer. The web
solves it by making the **client carry proof of identity on every request**. Log in
once, receive a token, and send that token back each time. The server checks the
token and knows you again.

## Authentication vs. authorization

These two words look alike and are constantly confused, but they are different jobs:

- **Authentication** — *who are you?* Verifying identity. The user proves they are
  who they claim, typically with a username and password, and possibly a second
  factor.
- **Authorization** — *what are you allowed to do?* Given a known identity, deciding
  whether this particular action is permitted — can this user see this record, edit
  this setting, reach this admin page?

You authenticate **once** (at login) and authorize **every request** thereafter. A
logged-in user is authenticated, but that alone doesn't let them delete another
person's account — that's an authorization check. Conflating the two is a classic
source of security holes: an app that checks *are you logged in?* but forgets to
check *is this yours?* leaks data between users. The sibling
[authentication basics](/learn/cybersecurity/authentication-basics/) lesson digs
into the identity side in more depth.

## The login flow

A typical username-and-password login looks like this:

1. The user submits a [form](/learn/web-dev/forms-and-user-input/) with their
   credentials over HTTPS.
2. The server looks up the account and **verifies the password** against the stored
   hash.
3. On success, it creates a **session** and hands the browser a session identifier.
4. Every later request includes that identifier; the server uses it to reload the
   user and treat the request as theirs.
5. On logout (or expiry), the session is destroyed and the identifier stops working.

The pseudocode is short — the care is in the details around it:

```python
def login(request):
    user = db.find_user(request.form["email"])
    if user is None or not verify_hash(request.form["password"], user.pw_hash):
        return error(401, "Invalid credentials")   # don't say which was wrong
    session_id = create_session(user.id)           # store server-side
    return set_session_cookie(session_id)          # browser sends it back
```

## Storing passwords safely

The single rule that matters most: **never store a password you can read.** If your
database keeps plain-text passwords, one leak exposes every user — and because
people reuse passwords, it exposes their other accounts too.

Instead, store a **hash**: a one-way transformation you can't reverse. When a user
logs in, hash what they typed and compare it to the stored hash. Two extra
requirements make this safe:

- **Salt** — a unique random value mixed into each hash, so identical passwords
  don't produce identical hashes and precomputed "rainbow table" attacks fail.
- **A slow, purpose-built algorithm** — **bcrypt**, **scrypt**, or **Argon2**.
  Ordinary hashes like SHA-256 are *too fast*: an attacker can try billions per
  second. Password hashes are deliberately slow to make guessing expensive.

Never invent your own scheme, and never use a plain fast hash for passwords.

## What a session actually is

A **session** is server-side state about one logged-in user, referenced by an
opaque **session ID**. Classically the server keeps a table — session ID → user,
plus when it was created and when it expires — and the browser holds only the ID.
Because the ID is meaningless on its own, stealing it is the whole game, which is
why it travels in a secure [cookie](/learn/web-dev/cookies-tokens-jwt/) and why
sessions expire.

There are two broad styles, both covered next lesson: **server-side sessions**,
where the server stores the state and the client holds only a reference, and
**token-based** approaches like JWTs, where the client holds signed claims and the
server stores nothing. Each has tradeoffs around revocation, scale, and where trust
lives — but both solve the same stateless problem you met above.

<div class="knowledge-check" data-quiz data-correct-msg="Right — authentication verifies who you are; authorization decides what that known user may do." markdown="0">
  <p class="knowledge-check__q">Quick check: a logged-in user tries to delete another user's post. Which check should stop them?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Authentication — they must not really be logged in</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Authorization — they're authenticated but not permitted to do this</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Neither — any logged-in user may do anything</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- HTTP is **stateless**, so an app must re-establish identity on every request; it
  does this by having the client carry proof of login each time.
- **Authentication** verifies *who you are*; **authorization** decides *what you may
  do*. Authenticate once, authorize every request.
- The **login flow** verifies credentials, creates a **session**, and hands the
  browser a session identifier to send back thereafter.
- **Never store plain-text passwords** — keep a **salted hash** made with bcrypt,
  scrypt, or Argon2, which are deliberately slow.
- A **session** is server-side state keyed by an opaque **session ID**; stealing the
  ID is the main risk, so it's kept secret and expires.
- Missing authorization checks — logged in but *is this yours?* — are a common,
  serious bug class.

Next up: [cookies, tokens & JWTs](/learn/web-dev/cookies-tokens-jwt/).
