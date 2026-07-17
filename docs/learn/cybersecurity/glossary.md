---
slug: glossary
title: Glossary of cybersecurity terms
description: Plain-language definitions of cybersecurity terms — CIA triad, threat, vulnerability, encryption, hash, MFA, least privilege, phishing, injection, malware, defense in depth, and more — each cross-linked to the lesson that explains it.
keywords: cybersecurity glossary, security terms, CIA triad, encryption, hash, MFA, least privilege, phishing, malware, injection, defense in depth, PII
level: beginner
status: full
lesson_standalone: true
---

# Glossary of cybersecurity terms

Every term used across the [Cybersecurity Fundamentals](/learn/cybersecurity/) path,
defined in plain language and linked to the lesson where it's explained in full.
Skim it as a refresher, or use your browser's find (Ctrl/Cmd-F) to jump to a word.
Terms are grouped by theme, roughly in the order the path introduces them.

## Foundations

**Cybersecurity** — Protecting systems, data, and the people who depend on them from
digital harm; an ongoing practice, not a product. See
[What is cybersecurity?](/learn/cybersecurity/what-is-cybersecurity/)

**CIA triad** — The three goals that define "secure": confidentiality, integrity, and
availability. See [The CIA triad](/learn/cybersecurity/cia-triad/)

**Confidentiality / integrity / availability** — Keeping data secret, unaltered, and
accessible to legitimate users, respectively. See [The CIA triad](/learn/cybersecurity/cia-triad/)

**Threat** — A potential harmful event, or the actor that could cause it. See
[Threats, vulnerabilities & risk](/learn/cybersecurity/threats-vulnerabilities-risk/)

**Vulnerability** — A weakness that could be abused to cause harm. See
[Threats, vulnerabilities & risk](/learn/cybersecurity/threats-vulnerabilities-risk/)

**Exploit** — The means of taking advantage of a vulnerability. See
[Threats, vulnerabilities & risk](/learn/cybersecurity/threats-vulnerabilities-risk/)

**Risk** — The chance of harm and how bad it would be — roughly likelihood times
impact. See [Threats, vulnerabilities & risk](/learn/cybersecurity/threats-vulnerabilities-risk/)

**Attack surface** — Everything an attacker could interact with — ports, inputs,
dependencies, people; shrinking it is a cheap win. See
[Threats, vulnerabilities & risk](/learn/cybersecurity/threats-vulnerabilities-risk/)

**Threat actor** — Whoever might attack — opportunists, criminals, insiders,
nation-states — each with different motives. See [Who attacks, and why](/learn/cybersecurity/threat-actors/)

**Least privilege** — Granting each user or process only the access it needs, and no
more. See [The security mindset](/learn/cybersecurity/security-mindset/)

**Defense in depth** — Layering multiple, overlapping controls so a single failure
isn't a breach. See [Defense in depth](/learn/cybersecurity/defense-in-depth/)

**Assume breach** — Designing as if something is already compromised, so one failure
is contained. See [The security mindset](/learn/cybersecurity/security-mindset/)

## Cryptography

**Cryptography / encryption** — Scrambling information with a key so only the key
holder can read it. See [What is cryptography?](/learn/cybersecurity/what-is-cryptography/)

**Plaintext / ciphertext** — The readable original and its encrypted, unreadable
form. See [What is cryptography?](/learn/cybersecurity/what-is-cryptography/)

**Key** — The secret that encrypts and decrypts; in good crypto, only the key is
secret, not the algorithm. See [What is cryptography?](/learn/cybersecurity/what-is-cryptography/)

**Symmetric encryption** — One shared secret key encrypts and decrypts (e.g. AES);
fast, but the key must be shared safely. See
[Symmetric & public-key encryption](/learn/cybersecurity/symmetric-and-asymmetric/)

**Public-key (asymmetric) encryption** — A public/private key pair; anyone can
encrypt with the public key, only the private key decrypts. See
[Symmetric & public-key encryption](/learn/cybersecurity/symmetric-and-asymmetric/)

**Hash** — A one-way function turning any input into a fixed fingerprint; used for
integrity and password storage. See [Hashing & integrity](/learn/cybersecurity/hashing-and-integrity/)

**Salt** — Random data added before hashing a password so identical passwords don't
share a hash. See [Hashing & integrity](/learn/cybersecurity/hashing-and-integrity/)

**Digital signature** — Signing with a private key so anyone can verify who sent
something and that it wasn't changed. See
[Digital signatures & certificates](/learn/cybersecurity/signatures-and-certificates/)

**Certificate / PKI / CA** — A certificate binds an identity to a public key, signed
by a Certificate Authority within a public-key infrastructure. See
[Digital signatures & certificates](/learn/cybersecurity/signatures-and-certificates/)

**End-to-end encryption** — Encryption where only the endpoints hold the keys, so
even the service can't read the content. See [Cryptography in practice](/learn/cybersecurity/crypto-in-practice/)

## Identity & access

**Authentication** — Verifying who someone is, using something they know, have, or
are. See [Authentication basics](/learn/cybersecurity/authentication-basics/)

**Authorization** — Deciding what an authenticated user is allowed to do. See
[Authorization & access control](/learn/cybersecurity/authorization-and-access/)

**MFA (multi-factor authentication)** — Requiring two or more different factors, so a
stolen password alone isn't enough. See [Passwords, MFA & passkeys](/learn/cybersecurity/passwords-and-mfa/)

**Passkey** — A device-held key pair that replaces a password and resists phishing.
See [Passwords, MFA & passkeys](/learn/cybersecurity/passwords-and-mfa/)

**RBAC (role-based access control)** — Assigning permissions to roles and roles to
users, rather than to individuals directly. See
[Authorization & access control](/learn/cybersecurity/authorization-and-access/)

**Secret** — An API key, token, password, or private key that grants access if
known; keep it out of code and rotate it. See [Managing secrets & keys](/learn/cybersecurity/secrets-management/)

**Social engineering** — Manipulating people (via phishing, pretexting, and the like)
instead of attacking technology. See [Social engineering & the human layer](/learn/cybersecurity/social-engineering/)

## Attacks

**Malware** — Malicious software — viruses, worms, trojans, spyware. See
[Malware & endpoint security](/learn/cybersecurity/malware-and-endpoints/)

**Ransomware** — Malware that encrypts data and demands payment; defended against
mainly with tested backups. See [Malware & endpoint security](/learn/cybersecurity/malware-and-endpoints/)

**Phishing** — Fraudulent messages that trick people into credentials or actions. See
[Social engineering & the human layer](/learn/cybersecurity/social-engineering/)

**Credential stuffing** — Reusing passwords leaked from one breach to break into
other accounts. See [Common attacks, in plain terms](/learn/cybersecurity/common-attacks/)

**Injection (e.g. SQL injection)** — Untrusted input changing the meaning of a query
or command; prevented with parameterized queries. See
[Web application attacks](/learn/cybersecurity/web-application-attacks/)

**XSS (cross-site scripting)** — Untrusted input running as script in a victim's
browser; prevented with output encoding. See [Web application attacks](/learn/cybersecurity/web-application-attacks/)

**CSRF (cross-site request forgery)** — Tricking a browser into making an
authenticated request; prevented with tokens and SameSite cookies. See
[Web application attacks](/learn/cybersecurity/web-application-attacks/)

**Man-in-the-middle (MITM)** — Intercepting or altering traffic between two parties;
defended with encryption in transit. See [Network attacks](/learn/cybersecurity/network-attacks/)

**Spoofing** — Forging an identity or address to impersonate or redirect. See
[Network attacks](/learn/cybersecurity/network-attacks/)

**DoS / DDoS** — Overwhelming a service to make it unavailable. See
[Network attacks](/learn/cybersecurity/network-attacks/)

## Defenses & operations

**Hardening** — Shrinking a system's attack surface and locking down its
configuration. See [Hardening systems](/learn/cybersecurity/hardening-systems/)

**Patching** — Keeping software updated so known, fixed flaws can't be abused. See
[Hardening systems](/learn/cybersecurity/hardening-systems/)

**Secure coding** — Building software that resists attack — validating input,
avoiding injection, managing dependencies. See [Secure coding](/learn/cybersecurity/secure-coding/)

**Segmentation** — Separating systems so a breach in one zone can't reach everything.
See [Securing networks & services](/learn/cybersecurity/securing-networks-and-services/)

**Logging & monitoring** — Recording and watching activity so you can detect and
investigate trouble. See [Monitoring & incident response](/learn/cybersecurity/monitoring-and-incident-response/)

**Incident response** — The plan to identify, contain, eradicate, recover from, and
learn from a security incident. See [Monitoring & incident response](/learn/cybersecurity/monitoring-and-incident-response/)

**PII (personally identifiable information)** — Data that identifies a person;
minimize, protect, and retain it carefully. See [Privacy & data protection](/learn/cybersecurity/privacy-and-data-protection/)

**Data minimization** — Collecting and keeping only the data you need, so a breach
exposes less. See [Privacy & data protection](/learn/cybersecurity/privacy-and-data-protection/)

## Practice & ethics

**Shift left** — Building security in from the start of development rather than
bolting it on at the end. See [Security for developers](/learn/cybersecurity/security-for-developers/)

**CTF (capture the flag)** — A legal, purpose-built competition for practising
security skills with permission built in. See [Where to go next](/learn/cybersecurity/keep-learning-security/)

**Responsible disclosure** — Reporting a discovered flaw ethically and giving the
owner time to fix it. See [Where to go next](/learn/cybersecurity/keep-learning-security/)

**Metadata (on the air)** — The patterns that remain even when content is encrypted —
that a transmission happened, its timing, and often IDs. See
[Wireless & RF security](/learn/cybersecurity/rf-and-radio-security/)
