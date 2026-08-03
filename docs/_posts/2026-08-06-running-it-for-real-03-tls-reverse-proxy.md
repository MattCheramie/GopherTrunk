---
title: "Running It For Real, Part 3: TLS & Sitting Behind a Reverse Proxy"
description: How GopherTrunk terminates TLS on its HTTP and gRPC listeners, why plain TCP stays the default, the both-or-neither cert/key rule, the permissive-by-default CORS allow-list, and the loopback-upstream reverse-proxy pattern with nginx or Caddy.
category: deep-dives
keywords: tls termination, reverse proxy nginx caddy, grpc tls, cors allow-list, servetls, self-signed cert, sse websocket streaming, loopback upstream, gophertrunk running it for real
tags: [running-it-for-real, tls, security, deployment, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Running It For Real"
series_part: 3
---

*Part 3 of **Running It For Real**, the series taking one GopherTrunk daemon from
a laptop demo to a hardened 24/7 service. Part 2 decided who may change things;
this post decides who can *read the wire* and where the daemon actually sits in
your network. There are two honest answers to "how do I put this on the internet"
— terminate TLS in the daemon itself, or hide the daemon on loopback and let a
real reverse proxy do it — and GopherTrunk is built to do either well.*

> **TL;DR:** Both the HTTP API (REST + SSE + WebSocket) and the gRPC server take
> optional TLS via one cert/key pair in `api` config. Plain TCP stays the default
> — bearer-token auth is enough for loopback and trusted-LAN, and TLS adds cert
> rotation and CA overhead. The daemon enforces **both-or-neither**: setting one
> of `tls_cert`/`tls_key` without the other is a refuse-to-start config error, not
> a silent fallback to plain HTTP. CORS is permissive by default (empty allow-list
> = any origin, so the SPA loads with zero config) and clamps down when you list
> origins. The recommended production shape is neither of those alone: run the
> daemon on **loopback**, front it with **nginx/Caddy** for TLS + auth, and let
> `auth.mode: auto` bypass on the loopback hop.

**Key takeaways**

- **TLS is opt-in, symmetric, and all-or-nothing.** One PEM cert/key pair serves
  both the HTTP and gRPC listeners; providing one path without the other fails
  startup rather than quietly serving plaintext.
- **CORS defaults open so the SPA just works.** An empty `allowed_origins` echoes
  any origin back — closed-LAN operators load the web UI from `file://` or a
  sibling static host with no CORS config; hostile-network operators list exact
  origins to clamp it.
- **Streaming endpoints opt out of write timeouts, not out of TLS.** SSE,
  WebSocket, and per-call audio streams disable the per-request `WriteTimeout` so
  they aren't torn down mid-frame — but they ride the same TLS listener as
  everything else.
- **The best posture is a reverse proxy over loopback.** Bind the daemon to
  `127.0.0.1`, let nginx/Caddy terminate TLS and authenticate, and Part 2's
  loopback bypass makes the daemon-side auth a no-op on the trusted hop.

## Cheat sheet

| Concern | Behaviour | Where it lives |
|---|---|---|
| TLS enable | both `tls_cert` + `tls_key` set → `ServeTLS` | `internal/api/server.go` (`Server.Run`) |
| Half-TLS guard | one without the other → construction error | `internal/api/server.go` (`NewServer`, XOR check) |
| CORS default | empty `allowed_origins` → echo any origin | `internal/api/cors.go` (`effectiveOrigins`) |
| CORS middleware | preflight + per-request headers | `internal/api/cors.go` (`corsMiddleware`) |
| Cert rotation | read once at `ServeTLS` → restart to rotate | `internal/api/server.go` |
| Proxy pattern | loopback bind + `auth.mode: auto` | [Hardening]({{ '/hardening.html' | relative_url }}) |

## In this post

- **Why plain TCP is still the default** — and when to leave it.
- **The both-or-neither cert/key rule** — one pair, two listeners, no silent fallback.
- **CORS, permissive by default** — how the SPA loads with zero config.
- **The reverse-proxy pattern** — loopback upstream, proxy-side TLS and auth.

## Why plain TCP is still the default

It would be easy to make TLS mandatory and feel virtuous about it. GopherTrunk
doesn't, because for the dominant deployment it would be security theatre with a
real operational cost. On a closed LAN the bearer-token posture from Part 2 is
sufficient, and TLS drags in cert generation, a CA decision, and — because the
daemon reads its cert at startup — a restart every rotation. So plain TCP is the
default and TLS is two lines of config away when you actually need it:

```yaml
api:
  http_addr: ":8080"
  grpc_addr: ":50051"
  tls_cert: "/etc/gophertrunk/tls/cert.pem"
  tls_key:  "/etc/gophertrunk/tls/key.pem"
```

The guidance is simple: loopback or trusted-LAN, leave it plain and lean on auth;
a public bind, enable TLS (or, better, put a proxy in front — more below). When
TLS is on, the *same* cert/key pair serves both listeners, so clients switch to
`https://` and `grpc+tls://` and everything — REST, SSE, WebSocket, gRPC
`StreamAudio` — is encrypted with one set of files to manage.

## The both-or-neither cert/key rule

The single most valuable thing the TLS code does is refuse to be half-configured.
A cert path with no key (a typo, a half-finished edit) is exactly the kind of
mistake that, under a "best-effort" design, silently falls back to plaintext — and
you don't find out until someone sniffs the wire. GopherTrunk makes it a
construction-time error instead:

```go
// internal/api/server.go (shape) — NewServer + Run
// Both files must be set to enable TLS; one without the other is a config error.
if (opts.TLSCert == "") != (opts.TLSKey == "") {
    return nil, errors.New("api: tls_cert and tls_key must both be set (or both empty)")
}
// …later, in Run:
tlsEnabled := s.tlsCert != "" && s.tlsKey != ""
if tlsEnabled {
    // ServeTLS reads cert/key off disk at start; rotation requires a restart.
    err = s.srv.ServeTLS(listener, s.tlsCert, s.tlsKey)
} else {
    err = s.srv.Serve(listener)
}
```

The `!=` on two emptiness checks is a compact XOR: it's true exactly when one path
is set and the other isn't, which is the invalid state. And the daemon layers a
second check on top — `preflight` (Part 7) calls `tls.LoadX509KeyPair` before the
listener ever binds, so a cert that exists but doesn't parse, or a key file owned
by the wrong user, surfaces as a clear `preflight: tls cert/key …` message rather
than an opaque goroutine error after the port is already open. Two independent
guards, both firing before any request is served.

One honest limitation lives here: `ServeTLS` reads the files once at startup, so
rotating a cert means restarting the daemon. That's a deliberate simplicity
trade-off — SIGHUP-driven reload is future work — and it's another reason the
proxy pattern below is often the better answer for anyone rotating certs on a
schedule.

### How that principle shaped the Go code

- **One server object, two transports.** `Server.Run` builds the listener, then
  branches to `ServeTLS` or `Serve` — TLS is a property of how the same handler
  chain is served, not a separate code path with its own routes.
- **Timeouts are set once, at the server.** `ReadHeaderTimeout` (10 s, Slowloris
  guard), `ReadTimeout`/`WriteTimeout` (30 s), and `IdleTimeout` (120 s) bound the
  standard REST handlers regardless of TLS.
- **Streaming endpoints opt out per-request.** SSE (`/api/v1/events`), the
  WebSocket upgrade, and the per-call audio stream disable `WriteTimeout` via
  `http.ResponseController` so a long-lived connection isn't guillotined at 30 s —
  they still ride the TLS listener, they just don't inherit the write deadline.
- **gRPC shares the pair.** The gRPC server loads the same cert/key, and its
  keep-alive params (`PermitWithoutStream: true`) keep idle `StreamAudio`
  subscribers alive through pings, encrypted or not.

## CORS, permissive by default

The browser SPA talks to the API cross-origin — served from a static host or even
`file://`, hitting the daemon on another port — so CORS is unavoidable, and here
too GopherTrunk defaults to "just works" and lets you clamp down:

```go
// internal/api/cors.go (shape)
// Empty AllowedOrigins → permissive default (any origin). Closed-LAN setups
// load the SPA without opting into CORS; hostile-network operators list origins.
func (c CORSConfig) effectiveOrigins() []string {
    if len(c.AllowedOrigins) == 0 {
        return []string{"*"}
    }
    return c.AllowedOrigins
}

func (c CORSConfig) originAllowed(origin string) (string, bool) {
    for _, allowed := range c.effectiveOrigins() {
        if allowed == "*" {
            return origin, true // echo the actual origin back so credentialed requests work
        }
        if strings.EqualFold(allowed, origin) {
            return origin, true
        }
    }
    return "", false
}
```

The wildcard case doesn't literally send `Access-Control-Allow-Origin: *` — it
echoes the caller's actual `Origin` back, because browsers reject `*` combined
with `Access-Control-Allow-Credentials`, and the SPA sends an `Authorization`
header. The middleware handles preflight `OPTIONS` with a 204 and a 10-minute
cache, and it advertises exactly the small surface the API uses: `GET/POST/PATCH/
DELETE/OPTIONS`, and `Authorization, Content-Type, Last-Event-ID` request headers
(that last one is what lets an SSE client resume a stream). Like the auth default,
the open default is paired with a startup warning: `IsDefaultPermissive` lets the
daemon flag an open CORS policy on a non-loopback bind, so it's visible, not
silent. An operator on a hostile network lists their one SPA origin and the
wildcard is gone.

## The reverse-proxy pattern

For a real public deployment, the recommended shape isn't "turn on the daemon's
TLS" — it's "hide the daemon and put a grown-up proxy in front." Bind the daemon
to loopback, terminate TLS and do auth at nginx or Caddy, and point the proxy's
upstream back at the daemon over `127.0.0.1`:

<figure class="lab-figure">
<svg viewBox="0 0 660 168" width="660" height="168" role="img" aria-label="The internet reaches a reverse proxy over HTTPS; the proxy terminates TLS and authenticates, then forwards to the GopherTrunk daemon bound to loopback over plain HTTP, where auth.mode auto bypasses on the loopback hop">
  <rect x="8" y="64" width="110" height="46" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="63" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="11">internet</text>
  <text x="63" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="9">clients</text>
  <line x1="118" y1="87" x2="150" y2="87" stroke="currentColor"/><polygon points="150,83 160,87 150,91" fill="currentColor"/>
  <text x="135" y="78" text-anchor="middle" fill="var(--accent)" font-size="9">https</text>
  <rect x="160" y="54" width="150" height="66" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="235" y="78" text-anchor="middle" fill="var(--accent)" font-size="11">nginx / Caddy</text>
  <text x="235" y="93" text-anchor="middle" fill="var(--fg-muted)" font-size="9">TLS terminate</text>
  <text x="235" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="9">+ auth</text>
  <line x1="310" y1="87" x2="342" y2="87" stroke="currentColor"/><polygon points="342,83 352,87 342,91" fill="currentColor"/>
  <text x="326" y="78" text-anchor="middle" fill="var(--fg-muted)" font-size="9">http · 127.0.0.1</text>
  <rect x="352" y="54" width="160" height="66" rx="6" fill="none" stroke="currentColor"/>
  <text x="432" y="78" text-anchor="middle" fill="currentColor" font-size="11">gophertrunk</text>
  <text x="432" y="93" text-anchor="middle" fill="var(--fg-muted)" font-size="9">loopback bind</text>
  <text x="432" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="9">auth: auto → bypass</text>
  <text x="330" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the daemon sees a loopback peer, so it trusts the hop; the proxy owns certs, auth, and rate limits</text>
</svg>
<figcaption>Loopback upstream. The proxy does the internet-facing hard parts; the daemon stays plain-TCP on loopback and lets auth.mode auto bypass the trusted hop.</figcaption>
</figure>

This is exactly what Part 2's `RemoteAddr`-only trust check was built to
cooperate with. The daemon sees the proxy connecting from `127.0.0.1`, so under
`auth.mode: auto` the token check is bypassed on that hop, and the proxy is the
one enforcing TLS, client auth, rate limits, and access logs — all things a
dedicated proxy does far better than a scanner daemon should try to. Because the
daemon deliberately ignores `X-Forwarded-For`, there's no way for a request that
*didn't* come through the proxy to inherit that loopback trust. The daemon's own
`tls_cert`/`tls_key` still exist for the standalone case (a single box with no
proxy, generate a cert with `certbot` or a self-signed pair for testing), but the
proxy pattern is what most 24/7 public deployments should reach for.

## Where this goes next

Auth and TLS answer "who and how" at the edge. The next question is "is it
healthy?" — and that's observability. [Part 4]({{ '/blog/deep-dives/running-it-for-real-04-metrics-that-matter/' | relative_url }})
opens the Prometheus registry: the counters and gauges the daemon exposes on
`/metrics`, and specifically the SDR-snapshot tiles — IQ power, clip ratio, gain,
lock state — that are worth an alert because they catch a front-end going deaf
before it costs you a call. The [Hardening]({{ '/hardening.html' | relative_url }})
doc carries the cert-generation and proxy recipes.

## FAQ

**Do I need TLS if I already have a bearer token?**
On loopback or a trusted LAN, no — auth protects mutations and there's nothing
private to sniff that isn't already on the air. On a public bind, yes: either
enable the daemon's TLS or, preferably, front it with a proxy that terminates TLS
for you.

**Why does setting only `tls_cert` fail instead of serving plain HTTP?**
Because a silent downgrade to plaintext is a security surprise. Half-configured
TLS is treated as a config error at construction, and `preflight` additionally
verifies the pair parses before the listener binds.

**Does the same cert cover gRPC?**
Yes. One cert/key pair serves both the HTTP and gRPC listeners; there's no
separate gRPC cert to manage.

**Will TLS break my SSE or audio streams?**
No. Streaming endpoints opt out of the per-request `WriteTimeout` so they aren't
cut off mid-frame, but they're served over the same TLS listener as every other
route.

**Should I use the daemon's TLS or a reverse proxy?**
A proxy over a loopback-bound daemon is the recommended production shape — it
handles cert rotation, auth, and rate limiting better, and the daemon's loopback
bypass makes it seamless. The built-in TLS is there for the standalone single-box
case.

## Series navigation

**Part 3 of 14** · ←
[Part 2: Auth Posture — Closed-LAN, Auto, Required]({{ '/blog/deep-dives/running-it-for-real-02-auth-posture/' | relative_url }})
· Next →
[Part 4: Metrics That Matter — Prometheus & SDR Tiles]({{ '/blog/deep-dives/running-it-for-real-04-metrics-that-matter/' | relative_url }})
