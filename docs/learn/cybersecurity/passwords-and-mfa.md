---
slug: passwords-and-mfa
title: Passwords, MFA & passkeys
description: Why passwords fail on their own and how to make accounts genuinely hard to break into — long unique passwords from a manager, how servers should store them as salted slow hashes, adding multi-factor authentication, and how passkeys and FIDO2 resist phishing.
keywords: password, password manager, MFA, two-factor, TOTP, passkey, FIDO2, credential stuffing, salted hash, phishing-resistant, account takeover
level: beginner
status: full
prereq:
  - authentication-basics
faq:
  - q: Are passwords enough to protect an account?
    a: "On their own, no. People reuse passwords across sites, pick weak ones, and get tricked into typing them into fake pages, so a single leaked or guessed password can open an account. The fix isn't heroic memory — it's a password manager generating a long unique password per site, plus multi-factor authentication so a stolen password alone isn't enough to log in."
  - q: Why should I use a password manager?
    a: "Because you can't remember dozens of long, unique, random passwords, and any shortcut you take — reuse, simple patterns — is exactly what attackers exploit. A password manager generates and stores a different strong password for every site, so a breach at one never spreads. You memorise one strong master password and protect the manager itself with multi-factor authentication."
  - q: What is MFA and does it really help?
    a: "Multi-factor authentication asks for a second proof beyond your password — an app code (TOTP), a push approval, or a hardware key — so a stolen password alone can't log in. It stops the large majority of account takeovers, because attackers who phish or buy a password still lack the second factor. Phishing-resistant factors like hardware keys and passkeys are the strongest."
  - q: What is a passkey?
    a: "A passkey is a login credential based on a device-held key pair instead of a shared secret. Your device keeps a private key and proves possession of it to the site, so there's no password to phish, guess, or reuse across sites. Passkeys build on the FIDO2 standard, are tied to the real site's identity, and are where account security is heading."
---

# Passwords, MFA & passkeys

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A password on its own is weak, but you can make an account genuinely hard to
break into. Use **long, unique passwords** kept in a **password manager**, turn
on **MFA** so a stolen password isn't enough, and prefer **passkeys** where you
can. On the server side, passwords should be **stored salted and slow** so a leak
resists cracking. None of this relies on you memorising secrets — the tools do
the remembering. See [authentication basics](/learn/cybersecurity/authentication-basics/)
for how proving identity fits together.
</div>

Passwords are the front door to almost every account you own, and they're the
part attackers work on hardest. The good news: a handful of habits — a manager, a
second factor, and soon passkeys — turn a fragile login into one that shrugs off
the common attacks. This lesson is defensive: how to protect *your* accounts and
build systems that protect *your users*.

## Why passwords fail

A password is a **shared secret**: you know it, the site stores something to check
it, and anyone else who learns it can log in as you. That model breaks down in
predictable ways.

- **Reuse.** People use the same password across many sites. One breach then opens
  all of them.
- **Weak choices.** Short or guessable passwords fall to automated guessing.
- **Phishing.** A convincing fake login page collects the password directly — no
  guessing required.
- **Breach reuse (credential stuffing).** Attackers take username-and-password
  pairs leaked from one site and replay them across thousands of others, betting
  on reuse. It works often enough to be a whole industry.

Underneath all of these is one human limit: **nobody can remember dozens of long,
unique secrets**. So people reuse and simplify — exactly the behaviour attackers
count on. The answer isn't a better memory; it's not relying on memory at all.

## Good password practice

The modern rule is short to state: **long, unique passwords, one per site,
generated and stored by a password manager.**

- **Long and unique.** Length beats complexity tricks, and a *different* password
  per site means one leak never spreads.
- **Let software remember.** A **password manager** generates random passwords and
  fills them in for you, so you never see or type most of them.
- **One master password.** You memorise a single strong password (or passphrase)
  that unlocks the manager — and you put **MFA on the manager itself**, because it
  now holds the keys to everything.

Done this way, the password you actually remember is one, not fifty, and every
account gets a strong secret you'd never be able to memorise.

## How passwords should be stored (server side)

If you build a login, how you store passwords decides how bad a breach is. The
rules are strict.

- **Never in plaintext**, and never in any **reversible** form. If the database
  can turn stored data back into the real password, so can whoever steals it.
- **Store a salted, slow hash.** Keep only a one-way [hash](/learn/cybersecurity/hashing-and-integrity/)
  of the password, add a unique **salt** per user so identical passwords don't
  produce identical hashes, and use a deliberately **slow** algorithm — **bcrypt**,
  **scrypt**, or **argon2** — so guessing at scale is expensive.

The goal is that even a full database leak forces attackers into slow, costly
cracking rather than instant access. See
[hashing & integrity](/learn/cybersecurity/hashing-and-integrity/) for how salted
slow hashes work.

## Multi-factor authentication (MFA)

**Multi-factor authentication** adds a second, independent proof on top of the
password — so a stolen password alone can't get in. It draws on the factors from
[authentication basics](/learn/cybersecurity/authentication-basics/): something you
know, something you have, something you are.

- **App codes (TOTP).** An authenticator app shows a six-digit code that changes
  every 30 seconds. Simple and widely supported.
- **Push approvals.** The site sends a prompt to your phone and you approve or
  deny. Convenient, but approve carefully — attackers sometimes spam prompts hoping
  you tap yes.
- **Hardware keys.** A small physical key you tap or plug in. These are the
  strongest and are **phishing-resistant** — they won't hand a secret to a fake
  site.

MFA stops the large majority of account takeovers, because an attacker who phishes
or buys your password still lacks the second factor. If you turn on one thing after
reading this, make it MFA.

## Passkeys & phishing resistance

**Passkeys** go a step further: they **replace the password entirely** with a
device-held key pair, built on the **FIDO2** standard. Your device keeps a private
key and only ever proves it possesses it — the secret never leaves the device and
is never sent to the site.

That design removes whole classes of attack:

- **Can't be phished.** A passkey is bound to the real site's identity, so a fake
  page can't trigger it.
- **Can't be reused.** There's no shared password to leak, guess, or replay across
  sites.
- **Nothing to store badly.** The site holds only a public key, which is useless to
  a thief.

Passkeys lean on the same public-key ideas behind
[signatures & certificates](/learn/cybersecurity/signatures-and-certificates/), and
they're the direction account security is heading — logins that are strong by
design instead of by discipline.

<div class="knowledge-check" data-quiz data-correct-msg="Right — MFA means a stolen password alone isn't enough to get in." markdown="0">
  <p class="knowledge-check__q">Quick check: the single biggest upgrade to an account's security is usually...</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Changing the password every 30 days</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Turning on multi-factor authentication (ideally phishing-resistant)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Adding numbers and symbols to a short password</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A password alone is weak — **reuse, weak choices, phishing, and credential
  stuffing** all defeat it.
- Use **long, unique passwords from a password manager**, protected by one strong
  master password and MFA.
- Servers must store passwords as a **salted, slow hash** (bcrypt/scrypt/argon2),
  never in plaintext or reversible form.
- **MFA** adds a second factor and stops most account takeovers even if the
  password leaks.
- **Passkeys / FIDO2** replace passwords with a device-held key pair that can't be
  phished or reused — the direction things are heading.

Next up: [authorization & access control](/learn/cybersecurity/authorization-and-access/)
