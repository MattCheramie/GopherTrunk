---
title: "Running It For Real, Part 2: Auth Posture — Closed-LAN, Auto, Required"
description: How GopherTrunk's mutation-endpoint auth works — the disabled/auto/required modes, why the default flipped to disabled for closed-LAN deployments, the loopback and trusted-network bypasses, constant-time token comparison, and hot token rotation without a restart.
category: deep-dives
keywords: api authentication, bearer token auth, closed-lan default, loopback bypass, trusted networks cidr, constant time compare, token rotation, sdr daemon security, gophertrunk running it for real
tags: [running-it-for-real, auth, security, ops, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Running It For Real"
series_part: 2
---

*Part 2 of **Running It For Real**, the series taking one GopherTrunk daemon from
a laptop demo to a hardened 24/7 service. Part 1 built the lifecycle skeleton;
this post is the very first decision you make before that daemon's HTTP API leaves
your LAN — who is allowed to *change* anything. It's a smaller surface than it
sounds, and the interesting part is not the token check but the **default**: why a
scanner daemon ships wide-open, and exactly what has to be true for that to be the
right call.*

> **TL;DR:** GopherTrunk gates every mutation endpoint (end-call, lockout, manual
> tune, settings PATCH, import commit, …) behind one middleware with three modes:
> **`disabled`** (wide open — the default, because the overwhelming deployment is a
> closed LAN where a bearer token is friction without a threat model),
> **`auto`** (require a token on public binds, bypass on loopback and listed
> trusted networks), and **`required`** (a valid token on every request, even
> loopback). The whole policy is `authState.authorize` in
> `internal/api/auth.go` — a three-case switch — and the safety rails are
> constant-time comparison, a `RemoteAddr`-only source check that ignores
> `X-Forwarded-For`, refusing to start in a mode that can never pass, and
> re-reading the token file on every request so rotation needs no restart.

**Key takeaways**

- **The default is `disabled`, on purpose.** An empty `api.auth.mode` resolves to
  wide-open mutations because GopherTrunk is overwhelmingly run on closed LANs;
  the daemon logs a loud warning when that default is combined with a non-loopback
  bind, so it's a choice, not an accident.
- **`auto` treats reachability as a trust proxy.** On a loopback-only bind every
  request is loopback-sourced by definition, so the token check is bypassed; on a
  public bind it's mandatory, and the daemon *refuses to start* without a token.
- **The source check is unforgeable.** Trust decisions read `RemoteAddr` only and
  deliberately ignore `X-Forwarded-For`, so a hostile upstream proxy can't spoof
  its way into the loopback/trusted-network bypass.
- **Rotation is a file write.** The token is re-read from disk on every mutation
  request, compared with `crypto/subtle.ConstantTimeCompare` — rotate with one
  `openssl rand` and a file write, no SIGHUP, no restart.

## Cheat sheet

| Mode | Loopback / trusted source | Public source | Fails to start when |
|---|---|---|---|
| `disabled` (default) | allowed | allowed | never (warns on non-loopback bind) |
| `auto` | allowed (bypass) | token required | non-loopback bind with no token/trusted net |
| `required` | token required | token required | no token configured at all |

| Piece | Where it lives |
|---|---|
| Mode enum + parse | `internal/api/auth.go` (`AuthMode`, `ParseAuthMode`) |
| Per-request decision | `internal/api/auth.go` (`authState.authorize`) |
| Startup validation | `internal/api/auth.go` (`newAuthState`) |
| Source-trust check | `internal/api/auth.go` (`sourceTrusted`, `remoteIP`) |
| Operator recipes | [Hardening]({{ '/hardening.html' | relative_url }}) → API authentication |

## In this post

- **Why a scanner ships wide-open** — the closed-LAN default and its warning.
- **The three modes** — disabled, auto, required, and where each one fits.
- **The bypass, done safely** — loopback, trusted networks, and the `X-Forwarded-For` trap.
- **The token path** — constant-time compare, hot rotation, refusing to start.

## Why a scanner ships wide-open

Most web services default to locked-down and make you opt into access. GopherTrunk
does the opposite for its *mutation* endpoints, and it's a deliberate read of who
actually runs the thing. The overwhelming deployment is a single box on a home or
club LAN, decoding a local system, with the operator being the only person who has
shell on it and the network being one they already trust. For that operator, a
bearer-token wall on "lock out this talkgroup" is pure friction — a threat model
with no threat.

So an empty `api.auth.mode` resolves to `disabled`:

```go
// internal/api/auth.go (shape) — ParseAuthMode
func ParseAuthMode(s string) (AuthMode, bool) {
    switch strings.ToLower(strings.TrimSpace(s)) {
    case "":                       // the new default
        return AuthModeDisabled, true
    case "auto":
        return AuthModeAuto, true
    case "required", "on", "true":
        return AuthModeRequired, true
    case "disabled", "off", "false":
        return AuthModeDisabled, true
    default:                       // unknown → disabled, ok=false so caller warns
        return AuthModeDisabled, false
    }
}
```

Two details make the default defensible rather than reckless. First, it is only
the *mutation* endpoints that are open — reads are always available, and there's
nothing here that reads private data the network can't already see off the air.
Second, the daemon is loud about it: when the `disabled` default (or the equally
permissive open-CORS default) is combined with a non-loopback bind, startup logs a
warning that names the exposure and points at the opt-in recipe. Reads stay open
regardless because liveness and readiness probes need `GET /api/v1/health` from
outside any auth boundary. The design goal is *no config needed for the safe
common case, one loud nudge when the case isn't safe.*

## The three modes

The per-request decision is a single switch, and it's worth reading in full
because it *is* the policy:

```go
// internal/api/auth.go (shape) — authState.authorize
func (s *authState) authorize(r *http.Request) (int, string) {
    switch s.mode {
    case AuthModeDisabled:
        return 0, "" // wide open
    case AuthModeRequired:
        return s.checkToken(r) // always, even loopback
    case AuthModeAuto:
        if s.sourceTrusted(r) {
            return 0, "" // loopback / trusted-network bypass
        }
        return s.checkToken(r)
    default:
        return http.StatusInternalServerError, "auth: invalid mode"
    }
}
```

**`disabled`** is the legacy `allow_mutations: true` behaviour — mutations are
open, full stop. It's for the closed-LAN single-host setup where the operator
already owns the box.

**`auto`** is the middle ground and the one most public-facing-but-single-operator
boxes should run. It treats *kernel-enforced reachability* as a reasonable
peer-cred proxy: if the daemon binds loopback-only, there's no network path for an
off-host request to reach the socket at all, so every request is trusted by
construction. If it binds a LAN or public address, a token is required — unless
the source IP falls inside `auth.trusted_networks`, a list of CIDRs you vouch for.
This is the mode to pair with an authenticating reverse proxy: point the proxy's
upstream at the daemon on loopback, let the proxy do auth, and `auto` bypasses on
the loopback hop.

**`required`** demands a valid Bearer token on every request regardless of source
— even loopback. Use it when the daemon shares a host with untrusted local users,
where "it came from 127.0.0.1" no longer means "I trust it."

### How that principle shaped the Go code

- **Invalid configs don't start.** `newAuthState` rejects `required` with no
  token and `auto` on a non-loopback bind with no token and no trusted networks —
  both are unwinnable postures (there's no way for a legitimate request to pass),
  so they become a refuse-to-start config error instead of a daemon that 403s
  every mutation forever.
- **The bind is inspected once, at construction.** `bindsToLoopback` resolves the
  listen address (`127.0.0.1` / `::1` / `localhost` → loopback; `:8080` /
  `0.0.0.0` / `[::]` → not) so the `auto` policy knows up front whether the
  loopback bypass even applies.
- **Capability is introspectable.** `GET /api/v1/mutations` reports `auth_mode`
  and `can_mutate` (via `canMutate`, which just runs `authorize` and checks for a
  zero status) so the TUI and scripts can light up write-side keybindings without
  probing a real endpoint and eating a 401.
- **Legacy configs still work.** `allow_mutations: true` maps to `disabled` with a
  deprecation warning — no existing deployment breaks on upgrade.

## The bypass, done safely

The subtle part of any "trust the local network" scheme is that trust decisions
based on the request's *claimed* source are forgeable. GopherTrunk sidesteps the
classic mistake by reading the real peer address and nothing else:

```go
// internal/api/auth.go (shape) — sourceTrusted / remoteIP
func (s *authState) sourceTrusted(r *http.Request) bool {
    if s.loopback {
        return true // loopback-only bind: no off-host path exists
    }
    ip := remoteIP(r) // RemoteAddr only — see below
    for _, n := range loopbackCIDRs { if n.Contains(ip) { return true } }
    for _, n := range s.trusted     { if n.Contains(ip) { return true } }
    return false
}

// remoteIP deliberately does NOT honour X-Forwarded-For: the loopback / trusted
// bypass must not be forgeable by a hostile upstream proxy that sets the header.
func remoteIP(r *http.Request) net.IP {
    host, _, _ := net.SplitHostPort(r.RemoteAddr)
    return net.ParseIP(host)
}
```

<figure class="lab-figure">
<svg viewBox="0 0 660 176" width="660" height="176" role="img" aria-label="A request enters the auth middleware, which switches on mode: disabled passes everything, auto checks whether the RemoteAddr source is loopback or in a trusted network and bypasses the token if so, and required always checks a constant-time-compared bearer token">
  <rect x="8" y="70" width="96" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="56" y="88" text-anchor="middle" fill="currentColor" font-size="11">request</text>
  <text x="56" y="103" text-anchor="middle" fill="var(--fg-muted)" font-size="9">RemoteAddr</text>
  <line x1="104" y1="92" x2="130" y2="92" stroke="currentColor"/><polygon points="130,88 140,92 130,96" fill="currentColor"/>
  <rect x="140" y="60" width="120" height="64" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="200" y="84" text-anchor="middle" fill="var(--accent)" font-size="11">authorize</text>
  <text x="200" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="9">switch on mode</text>
  <text x="200" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">XFF ignored</text>
  <line x1="260" y1="80" x2="300" y2="66" stroke="currentColor"/><polygon points="300,62 310,64 302,71" fill="currentColor"/>
  <line x1="260" y1="104" x2="300" y2="118" stroke="currentColor"/><polygon points="300,113 310,120 302,124" fill="currentColor"/>
  <rect x="310" y="42" width="150" height="40" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="385" y="60" text-anchor="middle" fill="var(--fg-muted)" font-size="10">source trusted?</text>
  <text x="385" y="74" text-anchor="middle" fill="var(--fg-muted)" font-size="9">loopback / CIDR → pass</text>
  <rect x="310" y="102" width="150" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="385" y="120" text-anchor="middle" fill="var(--accent)" font-size="10">checkToken</text>
  <text x="385" y="134" text-anchor="middle" fill="var(--fg-muted)" font-size="9">constant-time compare</text>
  <line x1="460" y1="122" x2="500" y2="122" stroke="currentColor"/><polygon points="500,118 510,122 500,126" fill="currentColor"/>
  <rect x="510" y="102" width="140" height="40" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="580" y="120" text-anchor="middle" fill="var(--fg-muted)" font-size="10">token_file</text>
  <text x="580" y="134" text-anchor="middle" fill="var(--fg-muted)" font-size="9">re-read per request</text>
  <text x="330" y="166" text-anchor="middle" fill="var(--fg-muted)" font-size="10">disabled short-circuits before any of this; required skips the trust check and always reaches checkToken</text>
</svg>
<figcaption>The auth decision. The bypass reads the real peer address only — an upstream proxy setting X-Forwarded-For can't forge trust — and the token is re-read from disk on every check.</figcaption>
</figure>

Loopback (`127.0.0.0/8` and `::1/128`) is implicitly trusted under `auto` and
doesn't need listing; `trusted_networks` adds your LAN prefix on top. And because
the check is `RemoteAddr`-only, fronting the daemon with nginx or Caddy is safe:
the proxy connects over loopback, the daemon sees a loopback peer, and the header
the proxy forwards is irrelevant to the trust decision. That's the seam Part 3
picks up.

## The token path

When a token *is* required, two properties matter — the comparison must not leak
timing, and rotation must not require downtime:

```go
// internal/api/auth.go (shape) — checkToken
func (s *authState) checkToken(r *http.Request) (int, string) {
    if s.tokFile != "" {
        // Re-read from disk on every request: rotation is a single file write,
        // no SIGHUP handler, no restart.
        if err := s.reloadTokenFile(); err != nil {
            return http.StatusInternalServerError, "auth: token_file unreadable"
        }
    }
    want := s.token.Load()
    if want == nil || *want == "" {
        return http.StatusForbidden, "auth: no token configured"
    }
    got, ok := bearerToken(r)
    if !ok {
        return http.StatusUnauthorized, "auth: missing Authorization: Bearer header"
    }
    if subtle.ConstantTimeCompare([]byte(got), []byte(*want)) != 1 {
        return http.StatusUnauthorized, "auth: invalid token"
    }
    return 0, ""
}
```

`crypto/subtle.ConstantTimeCompare` means a network attacker can't recover the
token byte-by-byte from response timing — a naive `==` would. The re-read on every
request sounds expensive but it's a single `os.ReadFile` on a tiny file, and it
buys real operational simplicity: to rotate, you `openssl rand -hex 32 > token`
and write the file; the very next request validates against the new value. The
token itself lives in a `token_file` (mode `0600`, owned by the daemon user)
rather than inline in `config.yaml`, so it never rides along in a config backup or
a settings-PATCH round-trip. The whole surface is small, and small is the point:
auth you can read end-to-end in one file is auth you can trust.

## Where this goes next

Auth answers "who may change things"; it says nothing about whether the bytes on
the wire are private. [Part 3]({{ '/blog/deep-dives/running-it-for-real-03-tls-reverse-proxy/' | relative_url }})
takes the daemon the rest of the way onto a hostile network: TLS termination for
the REST/SSE/WebSocket and gRPC listeners, the CORS allow-list and its
permissive-by-default posture, and the reverse-proxy pattern this post's loopback
bypass was built to cooperate with. The [Hardening]({{ '/hardening.html' | relative_url }})
doc has the copy-paste recipes; this series is why they're shaped the way they are.

## FAQ

**Isn't a wide-open default irresponsible?**
Only if it's silent — and it isn't. The default fits the dominant deployment (a
closed LAN the operator owns), reads stay open regardless, and the daemon logs a
warning the moment the open default meets a non-loopback bind. On a hostile
network you set `mode: required` (or `auto` behind a proxy); the docs lead with
that recipe.

**What's the difference between `auto` bypassing loopback and `required` not?**
`auto` treats "the kernel routed this from loopback" as sufficient trust — nobody
off-host can reach a loopback socket. `required` rejects that assumption, which
matters when untrusted users share the host and could originate loopback requests
themselves.

**Why ignore `X-Forwarded-For` when everyone behind a proxy sets it?**
Because trusting it would make the loopback/trusted-network bypass forgeable: any
client could add the header and claim to be loopback. The daemon uses only the
real TCP peer, and the correct proxy setup (loopback upstream + proxy-side auth)
works cleanly without XFF.

**Do I have to restart to change the token?**
No. The token file is re-read on every mutation request. Write a new value and the
next request uses it — rotation is a file write.

**How does the TUI know if it can mutate without trying?**
`GET /api/v1/mutations` runs the same `authorize` logic and returns `auth_mode`
plus `can_mutate`, so clients enable or grey out write actions up front instead of
discovering a 401 mid-action.

## Series navigation

**Part 2 of 14** · ←
[Part 1: From a Laptop Demo to a 24/7 Service]({{ '/blog/deep-dives/running-it-for-real-01-laptop-to-service/' | relative_url }})
· Next →
[Part 3: TLS & Sitting Behind a Reverse Proxy]({{ '/blog/deep-dives/running-it-for-real-03-tls-reverse-proxy/' | relative_url }})
