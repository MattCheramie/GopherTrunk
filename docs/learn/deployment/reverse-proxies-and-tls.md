---
slug: reverse-proxies-and-tls
title: Reverse proxies & TLS
description: "Putting a service behind nginx or Caddy — terminating HTTPS, routing by host, and why the proxy, not the app, usually handles certificates."
keywords: reverse proxy, nginx, caddy, tls termination, https, lets encrypt, acme, certificate, host routing, proxy pass
level: intermediate
status: full
prereq:
  - services-and-systemd
faq:
  - q: What is a reverse proxy?
    a: "A reverse proxy is a server that sits in front of one or more backend applications and forwards client requests to them. To the outside world it looks like the whole site; behind it, it routes each request to the right backend by hostname or path. It's where you centralise HTTPS, compression, rate limiting, and routing so each app doesn't reimplement them."
  - q: Why does the proxy handle TLS instead of the app?
    a: "Terminating TLS at the proxy means one place holds the certificates, one place renews them, and one place gets the ciphers and protocol versions right. The app behind it can speak plain HTTP on localhost, staying simpler. GopherTrunk binds its API to 127.0.0.1 and lets a proxy add HTTPS in front — the app never has to know about certificates."
---

# Reverse proxies & TLS

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **reverse proxy** (nginx or Caddy) sits in front of your app, taking public traffic and
forwarding it to the app on localhost. It **terminates TLS** — holding the certificate
and speaking HTTPS to the world — so the app behind it stays plain HTTP. It **routes by
host** so one server can front many apps. Tools like Caddy fetch and renew Let's Encrypt
certificates automatically over **ACME**.
</div>

The [Compose lesson](/learn/deployment/docker-compose/) bound GopherTrunk's API to
`127.0.0.1:8080` — reachable only from the host. A **reverse proxy** is how you safely
expose that to the world with HTTPS, without the app itself dealing in certificates. This
builds on [TLS & HTTPS](/learn/networking/tls-and-https/) and
[proxies & load balancers](/learn/networking/proxies-and-load-balancers/) from the
networking module.

## What a reverse proxy does

A reverse proxy accepts every incoming connection and forwards it to a backend. That one
choke point becomes the natural home for the cross-cutting concerns you don't want each
app to reinvent:

- **TLS termination** — decrypt HTTPS here, forward plain HTTP inward.
- **Host / path routing** — send `scanner.example.com` to one app, `blog.example.com` to
  another.
- **Rate limiting, compression, caching, auth** — applied once, in front of everything.

<figure class="figure" markdown="0">
<svg viewBox="0 0 480 120" role="img" aria-label="Internet traffic over HTTPS hits a reverse proxy, which forwards plain HTTP to two backend apps on localhost." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
  <text x="45" y="64" font-size="9">internet</text>
  <rect x="120" y="42" width="100" height="36" rx="4" fill="none" stroke="currentColor" stroke-width="1.7"/><text x="170" y="58">reverse proxy</text><text x="170" y="70" font-size="7.5">TLS ends here</text>
  <rect x="320" y="12" width="140" height="30" rx="4" fill="none" stroke="currentColor"/><text x="390" y="31" font-size="8">127.0.0.1:8080 (app)</text>
  <rect x="320" y="78" width="140" height="30" rx="4" fill="none" stroke="currentColor"/><text x="390" y="97" font-size="8">127.0.0.1:3000 (blog)</text>
  </g>
  <g stroke="currentColor" fill="none"><line x1="70" y1="60" x2="120" y2="60"/><line x1="220" y1="55" x2="320" y2="27"/><line x1="220" y1="65" x2="320" y2="93"/></g>
  <text x="95" y="52" font-size="7.5" fill="currentColor">HTTPS</text>
  <text x="275" y="30" font-size="7.5" fill="currentColor">HTTP</text>
</svg>
<figcaption>HTTPS ends at the proxy; it forwards plain HTTP to each backend on localhost, routing by hostname.</figcaption>
</figure>

## Why the app stays plain HTTP

Terminating TLS at the proxy means the **certificate lives in exactly one place**. One
process holds the private key, renews it before it expires, and keeps the TLS protocol
versions and ciphers current. The app behind it just serves plain HTTP on localhost and
never touches a certificate — simpler app, one thing to get right instead of many. This
is exactly why GopherTrunk binds to `127.0.0.1`: the loopback hop is unencrypted but
never leaves the host, and the proxy adds HTTPS on the public side.

## nginx: explicit and everywhere

nginx is the classic choice — a config block forwards a hostname to the backend:

```text
server {
    listen 443 ssl;
    server_name scanner.example.com;

    ssl_certificate     /etc/letsencrypt/live/scanner.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/scanner.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;   # GopherTrunk's local API
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

`proxy_pass` is the forward; the `X-Forwarded-*` headers tell the app the *original*
client's address and that the outside hop was HTTPS. You get the certificate from
[Let's Encrypt](https://letsencrypt.org/) with `certbot`, which also installs a renewal
timer.

## Caddy: automatic HTTPS

Caddy does the same job but fetches and renews certificates for you automatically over
**ACME** — the protocol Let's Encrypt uses to prove you control a domain. The entire
config is often two lines:

```text
scanner.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

Point that hostname's DNS at your server, start Caddy, and it obtains a valid
certificate on first request and renews it forever — no `certbot`, no cron job. For a
new deployment that's the lowest-friction path to real HTTPS.

## The key idea

The proxy owns the edge; the app owns its logic. TLS, routing, and rate limiting live in
front, in one place, on one team's checklist. Add a second app later and you route it
with a few more lines — the pattern scales into the
[load balancer](/learn/networking/proxies-and-load-balancers/) you'll meet in
[scaling](/learn/deployment/scaling-and-load-balancing/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — terminating TLS at the proxy keeps the certificate in one place and the app plain HTTP." markdown="0">
  <p class="knowledge-check__q">Quick check: why let the reverse proxy handle TLS instead of the app?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The app can't serve HTTP at all</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">One place holds and renews the certificate; the app stays simple plain HTTP</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Proxies encrypt faster than any application can</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **reverse proxy** (nginx, Caddy) fronts your app and forwards requests to it.
- It **terminates TLS** — the certificate lives in one place, and the app behind speaks
  plain HTTP on localhost.
- It **routes by host/path** so one server can front many apps, with `X-Forwarded-*`
  headers passing the real client info through.
- **Caddy** automates Let's Encrypt certificates over **ACME**; nginx + certbot does the
  same with a bit more config.

Next up: how a running service tells you it's healthy — logging and health checks.
