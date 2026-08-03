---
title: "The Operator's Cockpit, Part 3: The Connect Screen & Auth Handshake"
description: How a browser pairs with a GopherTrunk daemon — the connect screen that probes health before saving, the three-mode bearer-token policy with loopback trust, the mutation gate, and the capability probe that lets a front-end grey out write controls before it ever tries them.
category: deep-dives
keywords: bearer token auth, loopback trust sdr, connect screen react, health probe credentials, mutation gate middleware, can_mutate capability probe, token storage browser, auth mode auto required disabled, gophertrunk operator cockpit
tags: [operator-cockpit, api, auth, web, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cockpit"
series_part: 3
---

*Part 3 of **The Operator's Cockpit**. The bundle from Part 2 can be served from
anywhere, so the first thing it must learn is which daemon to talk to and whether
it may write. This post is the pairing handshake from both ends: the browser's
connect screen and the daemon's `authState`, meeting over the same
`/api/v1/health` and `/api/v1/mutations` reads that the terminal also uses.*

> **TL;DR:** A browser pairs with a daemon by entering a **server URL** and an
> optional **bearer token**; the SPA probes `GET /api/v1/health` first and only
> stores the credentials if the probe succeeds, so typos fail fast. On the daemon,
> `authState` runs one of three policies — `auto` (loopback and trusted CIDRs
> bypass, everything else needs a token), `required` (token always), `disabled`
> (open) — and every mutation route is wrapped in `s.gate(...)`. A separate
> capability probe, `GET /api/v1/mutations`, returns `can_mutate` for *this*
> request, so a front-end greys out write controls up front instead of surprising
> the operator with a 401.

**Key takeaways**

- **Validate before you save.** The connect screen calls `api.health` with the
  entered URL + token and stores nothing until it returns 200 — an unreachable
  host or wrong token surfaces as a human-readable banner, not a broken session.
- **Three policies, one gate.** `AuthModeAuto` / `Required` / `Disabled` all flow
  through `authorize`, and every write route is `s.gate(handler)`. Reads are never
  gated.
- **Loopback is trusted under `auto`.** A daemon bound to `127.0.0.1` treats every
  request as loopback-sourced; a token is only *required* on a non-loopback bind,
  so the common single-host setup needs no token at all.
- **Capability, not trial and error.** `GET /api/v1/mutations` reports
  `can_mutate` for the caller's own credentials, so both the SPA and the TUI light
  up write controls only when the daemon would accept them.

## Cheat sheet

| Piece | What it does | Where it lives |
|---|---|---|
| Connect screen | URL + token entry, health-probe gate | `web/src/components/ConnectScreen.tsx` |
| Health probe | validates credentials before saving | `web/src/api/client.ts` (`probe`, `api.health`) |
| Credential store | holds URL/token, persists per device | `web/src/store/shared.ts` (`setCredentials`) |
| Auth policy | parse + evaluate the three modes | `internal/api/auth.go` (`authState`, `authorize`) |
| Mutation gate | wrap every write route | `internal/api/server.go` (`gate`) |
| Capability probe | report `can_mutate` for this request | `internal/api/handlers_mutations.go` |

## In this post

- **The connect screen** — server URL, optional token, and validate-before-save.
- **The three auth modes** — and why loopback gets a bypass.
- **The gate** — the one middleware every mutation shares.
- **The capability probe** — how a client knows before it tries.
- **Where the token lives** — storage, quick-links, and the header, not the query.

## The connect screen

Because the same bundle runs against a daemon on `localhost`, a Pi across the
room, or a laptop behind a proxy (Part 2), the SPA's first screen is a form: a
**Server URL** and an optional **Bearer token**. The critical design choice is
that it *validates before it saves* — it probes the daemon's health with the
entered credentials, and only commits them to the store if the probe returns 200:

```tsx
// web/src/components/ConnectScreen.tsx (shape)
const submit = async (e: React.FormEvent) => {
  e.preventDefault();
  const trimmed = url.trim().replace(/\/+$/, "");
  if (!/^https?:\/\//i.test(trimmed)) {
    setErr("Server URL must start with http:// or https://");
    return;
  }
  try {
    await api.health({ baseURL: trimmed, token: token || null }); // probe first
    setCredentials(trimmed, token || null, remember);             // then save
    setConnected(true);
  } catch (e) {
    if (e instanceof HTTPError) {
      setErr(`Daemon refused the request (${e.status}). ${e.body || ""}`);
    } else if (e instanceof Error) {
      setErr(`Could not reach ${trimmed}: ${e.message}`);
    }
  }
};
```

The two failure paths are distinct on purpose. An `HTTPError` means the daemon
answered but *refused* — wrong token, most likely — and the banner shows the
status. A plain `Error` means the host was unreachable — typo, wrong port, daemon
down — and the banner says so. A first-run operator gets a specific reason instead
of a spinner that never resolves.

Two smaller touches earn their keep. When the page is served over http(s) by the
daemon itself, `suggestedServerURL()` pre-fills the field with
`window.location.origin` — the "open the daemon's own URL" case is one click. And
the URL hash can carry `#server=…&token=…` for a one-click bookmark; the screen
reads it, then immediately `history.replaceState`s the hash away so a screenshot
or shared bookmark doesn't leak the token.

## The three auth modes

On the daemon side, the policy the connect screen is negotiating with is
`AuthMode`, and there are exactly three:

```go
// internal/api/auth.go (shape)
const (
    AuthModeAuto     AuthMode = iota // loopback + trusted CIDRs bypass; else token
    AuthModeRequired                 // token always, even loopback
    AuthModeDisabled                 // open — the legacy allow_mutations behaviour
)

// authorize returns 0,"" when the request may proceed; a 401/403 otherwise.
func (s *authState) authorize(r *http.Request) (int, string) {
    switch s.mode {
    case AuthModeDisabled:
        return 0, ""
    case AuthModeRequired:
        return s.checkToken(r)
    case AuthModeAuto:
        if s.sourceTrusted(r) {
            return 0, ""
        }
        return s.checkToken(r)
    }
    return http.StatusInternalServerError, "auth: invalid mode"
}
```

The interesting mode is `auto`, and the interesting idea in it is **loopback
trust**. A GopherTrunk daemon is overwhelmingly deployed on a closed LAN where the
operator is the only one with shell access; requiring a bearer token there is
friction with no matching threat. So under `auto`, if the listener is bound to
loopback only, *every* request is loopback-sourced by definition — there is no
kernel path for an off-host request to reach the socket — and `sourceTrusted`
returns true without looking at a token:

```go
// internal/api/auth.go (shape)
func (s *authState) sourceTrusted(r *http.Request) bool {
    if s.loopback { // bound to 127.0.0.1 / ::1 only
        return true
    }
    ip := remoteIP(r) // from RemoteAddr — never X-Forwarded-For
    for _, n := range loopbackCIDRs {
        if n.Contains(ip) {
            return true
        }
    }
    for _, n := range s.trusted { // operator-configured CIDRs
        if n.Contains(ip) {
            return true
        }
    }
    return false
}
```

`remoteIP` reads `RemoteAddr` and *deliberately ignores* `X-Forwarded-For`,
because the loopback bypass must not be forgeable by a hostile upstream proxy
inserting a fake header. And the config is validated at startup, not at first
request: `newAuthState` refuses to construct an `auto` policy on a non-loopback
bind with no token and no trusted networks, so a wide-open public listener is a
*startup* error, not a silent hole.

Token comparison itself is `crypto/subtle.ConstantTimeCompare`, and a `token_file`
is re-read on every request so operators can rotate a token without a daemon
restart or a SIGHUP handler.

## The gate

Every mutation route wears the same one-line middleware — the same `s.gate` we met
in Part 1, here in its role as the enforcement point for all of the above:

```go
// internal/api/server.go (shape)
mux.HandleFunc("PATCH /api/v1/audio",        s.gate(s.handleAudioPatch))
mux.HandleFunc("POST  /api/v1/hunt/start",   s.gate(s.handleHuntStart))
mux.HandleFunc("PATCH /api/v1/settings",     s.gate(s.handleSettingsPatch))
mux.HandleFunc("POST  /api/v1/spectrum/devices/{serial}/tune", s.gate(s.handleSpectrumTune))
```

Reads — `GET /api/v1/systems`, `/calls/active`, `/scanner` — are never wrapped.
The rule is grep-able: if a route mutates state, it is `s.gate(...)`; if it only
reads, it isn't. That uniformity is what lets the browser's `client.ts` attach the
`Authorization: Bearer` header to *every* request unconditionally (it's harmless
on reads) and let the gate sort out whether a given write is allowed.

## The capability probe

Here is the piece that turns a security boundary into good UX. A front-end could
just *try* a mutation and handle the 401 — but then the write button looks live
until you press it and it fails. Instead there's a read that reports the answer in
advance:

```go
// internal/api/handlers_mutations.go (shape)
func (s *Server) handleMutationStatus(w http.ResponseWriter, r *http.Request) {
    canMutate := s.auth.canMutate(r) // would THIS request pass authorize()?
    writeJSON(w, http.StatusOK, map[string]any{
        "auth_mode":          s.auth.mode.String(), // "auto" | "required" | "disabled"
        "can_mutate":         canMutate,
        "allow_mutations":    canMutate, // legacy alias
        "engine_writable":    s.mutator != nil,
        "retention_writable": s.retention != nil,
        "tones_writable":     s.tones != nil,
    })
}
```

`canMutate` runs the *same* `authorize` logic against the current request without
performing any mutation, so it's a faithful preview. The browser store folds that
into a single gate that also honours the operator's own write-mode toggle:

```ts
// web/src/store/shared.ts (shape)
// Can-mutate gate combining write-mode toggle with daemon capability.
export function selectCanMutate(s: SharedState): boolean {
  return s.writeMode && (s.mutations?.allow_mutations ?? false);
}
```

So a control is live only when *both* the operator has opted into write mode *and*
the daemon says this request would pass. The TUI does the identical thing —
`cmdMutationStatus` in its `Init` batch, AND-ed with its `--write` flag — which is
why "can I change this?" resolves the same way in a terminal and a browser. It's
the capability contract, not two independent guesses.

<figure class="lab-figure">
<svg viewBox="0 0 660 210" width="660" height="210" role="img" aria-label="The connect handshake: the browser probes health with the entered URL and token; on success it saves credentials and fetches the mutations capability; write controls light up only when both the operator's write-mode toggle and the daemon's can_mutate are true">
  <rect x="8" y="20" width="150" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="83" y="40" text-anchor="middle" fill="var(--accent)" font-size="11">connect screen</text>
  <text x="83" y="56" text-anchor="middle" fill="var(--fg-muted)" font-size="9">URL + token</text>
  <line x1="158" y1="43" x2="196" y2="43" stroke="currentColor"/><polygon points="196,39 206,43 196,47" fill="currentColor"/>
  <rect x="206" y="20" width="150" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="281" y="40" text-anchor="middle" fill="currentColor" font-size="11">GET /health</text>
  <text x="281" y="56" text-anchor="middle" fill="var(--fg-muted)" font-size="9">probe before save</text>
  <line x1="356" y1="43" x2="394" y2="43" stroke="currentColor"/><polygon points="394,39 404,43 394,47" fill="currentColor"/>
  <rect x="404" y="20" width="150" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="479" y="40" text-anchor="middle" fill="var(--accent)" font-size="11">save creds</text>
  <text x="479" y="56" text-anchor="middle" fill="var(--fg-muted)" font-size="9">store + persist</text>
  <line x1="479" y1="66" x2="479" y2="92" stroke="currentColor"/><polygon points="475,92 479,102 483,92" fill="currentColor"/>
  <rect x="404" y="102" width="150" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="479" y="122" text-anchor="middle" fill="currentColor" font-size="11">GET /mutations</text>
  <text x="479" y="138" text-anchor="middle" fill="var(--fg-muted)" font-size="9">can_mutate?</text>
  <line x1="404" y1="125" x2="360" y2="125" stroke="currentColor"/><polygon points="360,121 350,125 360,129" fill="currentColor"/>
  <rect x="200" y="102" width="150" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="275" y="119" text-anchor="middle" fill="var(--accent)" font-size="11">write control</text>
  <text x="275" y="134" text-anchor="middle" fill="var(--fg-muted)" font-size="9">writeMode &amp;&amp; can_mutate</text>
  <text x="330" y="182" text-anchor="middle" fill="var(--fg-muted)" font-size="10">reads flow the moment health passes; writes light up only when both gates agree</text>
  <text x="330" y="198" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the token rides the Authorization header — never a query parameter, never a log line</text>
</svg>
<figcaption>Pairing is probe-then-save; write controls are a capability the client asks for, not a 401 it discovers.</figcaption>
</figure>

## Where the token lives

The token is the operator's, and it stays that way. `setCredentials(url, token,
persist)` writes it into the browser store (and, when "Remember on this device" is
checked, to local storage via `prefs`); the connect screen's fine print — "The
token stays in your browser. The page never phones home; every request is direct
to the daemon" — is literally true, because the SPA has no server of its own to
phone.

On the wire, the token is always an `Authorization: Bearer` header, never a query
parameter, so it never lands in an access log. That's `client.ts`'s job on every
request, and it's also why the live audio stream (Part 5) uses `fetch()` rather
than a bare `<audio src>` element — only `fetch` can set the header. The one place
that constraint bites is the browser `EventSource`, which *can't* set headers at
all; that's why the SPA reaches for the WebSocket event twin instead, which is
exactly where Part 4 begins.

## FAQ

**Do I need a token on my home LAN?**
Usually no. Under the default `auto` policy a loopback-bound daemon trusts every
request, and you can add your LAN's CIDR to `trusted_networks` to extend that to
other boxes. A token is only *required* when you bind to a public interface (or
pick `required` explicitly).

**Why probe health before saving credentials?**
So a bad URL or wrong token fails immediately with a specific banner — "daemon
refused (401)" vs. "could not reach host" — instead of saving broken credentials
and leaving every subsequent panel spinning.

**Why does `X-Forwarded-For` get ignored?**
Because the loopback/trusted-network bypass keys off the source IP, and honouring a
client-settable header would let a hostile upstream forge a trusted source.
`remoteIP` reads only `RemoteAddr`.

**What's the difference between `can_mutate` and `allow_mutations`?**
None functionally — `allow_mutations` is a legacy alias for the same boolean. Both
report whether *this* request would pass the gate, computed by running the real
`authorize` logic without side effects.

**Can I rotate the token without restarting?**
Yes, if you use `token_file`. The daemon re-reads the file on every request, so
writing a new token takes effect on the next call — no restart, no SIGHUP.

## Series navigation

**Part 3 of 14** · ←
[Part 2: A React SPA Inside a Go Binary]({{ '/blog/deep-dives/operator-cockpit-02-react-spa-in-a-go-binary/' | relative_url }})
· Next →
[Part 4: The Event Stream — SSE to React]({{ '/blog/deep-dives/operator-cockpit-04-sse-to-react/' | relative_url }})
</content>
