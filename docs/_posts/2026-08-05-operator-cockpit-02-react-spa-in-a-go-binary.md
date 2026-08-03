---
title: "The Operator's Cockpit, Part 2: A React SPA Inside a Go Binary"
description: How GopherTrunk embeds its built Vite/React bundle into the daemon with go:embed all:dist, serves it at the root with a client-side-routing SPA fallback, degrades to a helpful message when the embed is empty, and stays a plain API client so browser and standalone deployments are identical.
category: deep-dives
keywords: go embed react spa, vite bundle go binary, spa fallback index html, single binary web console, embed vs standalone, client-side routing fallback, http fileserverfs, gophertrunk operator cockpit
tags: [operator-cockpit, web, go, react, api]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cockpit"
series_part: 2
---

*Part 2 of **The Operator's Cockpit**. One daemon, one REST + SSE contract, two
renderers — that was Part 1. This post is about the first renderer's delivery
mechanism: how a React single-page app becomes bytes inside the Go binary, gets
served at `/`, and behaves like a real client-side-routed app while still being
nothing more than a client of the same API the terminal uses.*

> **TL;DR:** GopherTrunk `//go:embed all:dist`-es the built Vite bundle into the
> `web` package, and `internal/api` serves it at `/` through `http.FileServerFS`
> with an **SPA fallback**: a real asset is served directly, and any *other*
> non-`/api` path is rewritten to `index.html` so React-Router resolves it on the
> client. When the embed is empty — a fresh checkout built without `make
> web-build` — `HasAssets()` is false and the daemon serves a **helpful 404** at
> `/` telling you to run `make dist`, instead of a blank "page not found." The
> SPA is a pure API client, so serving it embedded from the daemon or standalone
> from a file bundle changes nothing about how it talks to the engine.

**Key takeaways**

- **The bundle ships in the binary.** `//go:embed all:dist` snapshots `web/dist/`
  at compile time; there is no sibling directory to deploy, no static file server
  to run.
- **SPA fallback is one rule.** If the requested path is a real embedded file,
  serve it; otherwise serve `index.html` so the client router owns `/scanner`,
  `/settings`, `/import`, and friends — but never `/api/*`.
- **Empty embed fails loudly, not blankly.** A build without a prior SPA build
  contains only a `.gitkeep` sentinel; `HasAssets()` returns false and the daemon
  answers `/` with an HTML 404 that names the fix (`make dist`).
- **Embedded and standalone are the same app.** The SPA never assumes it's served
  by the daemon; it targets a base URL, so the identical bundle runs from the
  daemon's `/`, from a `file://` bundle, or behind a reverse proxy.

## Cheat sheet

| Piece | What it does | Where it lives |
|---|---|---|
| Embed | snapshot `web/dist/` into the binary | `web/embed.go` (`//go:embed all:dist`) |
| Asset access | expose the `dist` sub-tree as an `fs.FS` | `web/embed.go` (`Assets`, `HasAssets`) |
| Mount decision | serve SPA vs. missing-embed message | `internal/api/server.go` (`routes`) |
| SPA handler | file-or-fallback-to-index routing | `internal/api/server.go` (`spaHandler`) |
| Missing-embed | HTML 404 with build instructions | `internal/api/server.go` (`spaMissingHandler`) |
| Contract tests | root, asset, client-route, no-shadow | `internal/api/spa_test.go` |

## In this post

- **Embedding the bundle** — what `//go:embed all:dist` actually captures.
- **The SPA fallback rule** — the three-way branch that makes client routing work.
- **The empty-embed path** — why a fresh checkout gets a helpful 404, not a blank.
- **Not shadowing the API** — how `/api/*` stays owned by real handlers.
- **Embed vs. standalone** — one bundle, many ways to serve it.

## Embedding the bundle

A React app is, after `vite build`, a directory of static files: one
`index.html`, a hashed `assets/` folder of JS and CSS, some icons. The classic
way to ship that is to deploy the directory next to the binary and point a web
server at it. GopherTrunk doesn't. It puts the directory *inside* the binary:

```go
// web/embed.go
//go:embed all:dist
var rawDist embed.FS

// Assets returns the embed.FS sub-tree rooted at `dist`. Callers
// should treat it as a read-only fs.FS — internal/api wires it
// through http.FileServerFS.
func Assets() fs.FS {
    sub, err := fs.Sub(rawDist, "dist")
    if err != nil {
        return rawDist
    }
    return sub
}
```

The `all:` prefix matters: it tells `embed` to include files that start with `_`
or `.` (Vite emits some), which a plain `//go:embed dist` would silently skip. The
result is that a single `gophertrunk` binary *is* the web console — copy it to a
Raspberry Pi and the operator UI comes with it. No `nginx`, no sibling
`gophertrunk-web/`, no version skew between the API and the UI that talks to it,
because they were compiled together.

There is one wrinkle the package doc calls out. `web/dist/` is a build artifact,
not checked-in source, so a fresh clone has an empty `dist/` holding only a
`.gitkeep` sentinel. `//go:embed` still succeeds — it just captures the sentinel —
and the binary compiles. So "did the SPA actually get built?" is a runtime
question, which is what `HasAssets` answers:

```go
// web/embed.go
// HasAssets returns true when the embed contains real build output
// (index.html in particular).
func HasAssets() bool {
    _, err := fs.Stat(Assets(), "index.html")
    return err == nil
}
```

## The SPA fallback rule

A single-page app has client-side routes — `/scanner`, `/settings`, `/import` —
that exist only in React-Router, not on disk. If you deep-link to `/scanner` or
hit reload there, the browser asks the *server* for `/scanner`, and the server has
no such file. The fix is the **SPA fallback**: serve real files as themselves, and
serve everything else as `index.html` so the client router takes over. In
GopherTrunk that's one handler:

```go
// internal/api/server.go (shape) — spaHandler
func (s *Server) spaHandler() http.Handler {
    fileSrv := http.FileServerFS(s.webAssets)
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Defence in depth: never let the SPA answer an API path.
        if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/metrics" {
            http.NotFound(w, r)
            return
        }
        clean := strings.TrimPrefix(r.URL.Path, "/")
        if clean == "" {
            fileSrv.ServeHTTP(w, r) // "/" → index.html
            return
        }
        if _, err := fs.Stat(s.webAssets, clean); err == nil {
            fileSrv.ServeHTTP(w, r) // a real asset (assets/app-abc123.js)
            return
        }
        // Unknown path → serve index.html so React-Router resolves it.
        r2 := r.Clone(r.Context())
        r2.URL.Path = "/"
        fileSrv.ServeHTTP(w, r2)
    })
}
```

Three branches, in order: the API guard, the "is this a real file?" `fs.Stat`
check, and the fallback to `index.html`. The contract is pinned by tests that read
like a spec — `TestSPA_RootServesIndex`, `TestSPA_AssetServesDirectly`, and the
one that proves deep-linking works:

```go
// internal/api/spa_test.go (shape)
func TestSPA_ClientRouteFallsBackToIndex(t *testing.T) {
    base, teardown := mkServer(t, ServerOptions{Bus: bus, WebAssets: fakeSPAFS()})
    defer teardown()
    resp, _ := http.Get(base + "/scanner")   // no such file
    body, _ := io.ReadAll(resp.Body)
    if !bytesContains(body, "spa-root") {     // got index.html
        t.Errorf("client route should fall back to index.html")
    }
}
```

### How that principle shaped the code

- **`fs.Stat` is the branch key.** The handler decides "asset or route" by asking
  the embedded filesystem whether the path exists. Hashed asset URLs stat true and
  are served verbatim (with the file server's caching headers); route URLs stat
  false and fall through to `index.html`.
- **The API guard is belt-and-suspenders.** The Go 1.22 mux already routes
  `/api/v1/*` to their specific handlers before `/` ever matches, so the SPA
  handler never sees an API path in practice. The explicit `strings.HasPrefix`
  guard exists so that if an embed override ever *did* shadow an API route, a test
  would catch it loudly instead of the SPA quietly serving HTML to a JSON client.
- **The clone avoids mutating the request.** The fallback rewrites `URL.Path` on a
  `r.Clone`, not the original, so nothing downstream sees a surprise path.

## The empty-embed path

Now the fresh-checkout case. If `HasAssets()` is false, mounting `spaHandler`
would serve the `.gitkeep` sentinel as the whole app — a broken, confusing
console. So `routes()` branches on it:

```go
// internal/api/server.go (shape) — inside routes()
embeddedSPA := false
if s.webAssets != nil {
    if _, err := fs.Stat(s.webAssets, "index.html"); err == nil {
        mux.Handle("GET /", s.spaHandler())
        embeddedSPA = true
    }
}
if !embeddedSPA {
    mux.Handle("GET /{$}", s.spaMissingHandler())
}
```

The `spaMissingHandler` answers *exactly* `GET /` (the `{$}` pattern anchors it to
the root, so `/scanner` still 404s normally) with an HTML page that keeps the
`404` status — proxies and health checks still see "missing" — but replaces
stdlib's blank body with something an operator can act on:

```go
// internal/api/server.go (shape) — spaMissingHandler
func (s *Server) spaMissingHandler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        w.WriteHeader(http.StatusNotFound)
        _, _ = w.Write([]byte(spaMissingBody)) // "…run `make dist` …try /api/v1/health"
    })
}
```

`TestSPA_NoEmbedReturnsHelpfulMessage` asserts the body mentions `make dist`, "web
console," and `/api/v1/health` — so the failure mode is a signpost, not a dead
end. The API itself is completely healthy in this state; only the UI is absent,
and the page says exactly that.

<figure class="lab-figure">
<svg viewBox="0 0 660 220" width="660" height="220" role="img" aria-label="A request to the daemon root: if the go:embed snapshot has a real index.html it enters the SPA handler which serves a real asset directly or falls back to index.html for client routes; if the embed is empty it serves an HTML 404 telling the operator to run make dist">
  <rect x="8" y="90" width="110" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="63" y="110" text-anchor="middle" fill="currentColor" font-size="11">GET /path</text>
  <text x="63" y="125" text-anchor="middle" fill="var(--fg-muted)" font-size="9">browser</text>
  <line x1="118" y1="112" x2="150" y2="112" stroke="currentColor"/><polygon points="150,108 160,112 150,116" fill="currentColor"/>
  <rect x="160" y="86" width="130" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="225" y="106" text-anchor="middle" fill="var(--accent)" font-size="11">HasAssets()?</text>
  <text x="225" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="9">index.html embedded</text>
  <line x1="290" y1="100" x2="326" y2="60" stroke="currentColor"/><polygon points="321,58 332,55 326,66" fill="currentColor"/>
  <text x="308" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="9">yes</text>
  <line x1="290" y1="126" x2="326" y2="170" stroke="var(--fg-muted)"/><polygon points="326,164 332,175 321,172" fill="var(--fg-muted)"/>
  <text x="308" y="158" text-anchor="middle" fill="var(--fg-muted)" font-size="9">no</text>
  <rect x="332" y="30" width="150" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="407" y="50" text-anchor="middle" fill="var(--accent)" font-size="11">spaHandler</text>
  <text x="407" y="65" text-anchor="middle" fill="var(--fg-muted)" font-size="9">fs.Stat: file? serve it</text>
  <text x="407" y="77" text-anchor="middle" fill="var(--fg-muted)" font-size="9">else → index.html</text>
  <rect x="332" y="150" width="150" height="46" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="407" y="170" text-anchor="middle" fill="var(--fg-muted)" font-size="11">spaMissingHandler</text>
  <text x="407" y="185" text-anchor="middle" fill="var(--fg-muted)" font-size="9">HTML 404 · "make dist"</text>
  <line x1="482" y1="56" x2="514" y2="56" stroke="currentColor"/><polygon points="514,52 524,56 514,60" fill="currentColor"/>
  <rect x="524" y="34" width="120" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="584" y="54" text-anchor="middle" fill="var(--accent)" font-size="11">React app</text>
  <text x="584" y="69" text-anchor="middle" fill="var(--fg-muted)" font-size="9">client router</text>
  <text x="330" y="212" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the `/api/*` routes are matched first by the mux, so the SPA handler never shadows them</text>
</svg>
<figcaption>Serving the root: a populated embed enters the SPA fallback; an empty one returns a signposted 404. Either way the API is unaffected.</figcaption>
</figure>

## Embed vs. standalone

The same `spaHandler` machinery is reused for the daemon's *other* embedded
consoles — the Config Builder at `/config/`, Signal Lab at `/siglab/`, RF Scope at
`/rfscope/` — each a `FileServerFS` over its own `fs.FS` with the identical
file-or-fallback rule, just rooted one level down. And the same React bundle that
the daemon serves at `/` also runs **standalone**: the `gophertrunk config serve`
command serves the Config Builder SPA at its own `/`, and a `file://` bundle can
open the app with no daemon behind it at all.

That works because — as Part 3 will show — the SPA never assumes it lives at the
daemon's origin. It targets a **base URL** you give it and attaches an optional
bearer token, so the exact same compiled bytes talk to a daemon on `localhost`, a
Pi across the room, or a laptop on the far side of a reverse proxy. Embedding is a
*delivery* choice; it doesn't change what the app is. The app is a client of the
contract from Part 1, and it stays one no matter who hands it to the browser.

## Where this goes next

[Part 3]({{ '/blog/deep-dives/operator-cockpit-03-connect-screen-auth/' | relative_url }})
picks up exactly there: the first screen the SPA shows. Because the bundle can be
served from anywhere, its first job is to learn *which* daemon to talk to — the
connect screen takes a server URL and an optional token, validates them against
`GET /api/v1/health` before saving, and stores them so a phone remembers its Pi.
That's the browser end of the auth handshake whose server half — `s.gate` and
`authorize` — we met in Part 1.

## FAQ

**Why embed the SPA instead of serving a directory?**
So the console ships *with* the daemon as one file, with no version skew: the UI
and the API it calls were compiled together. Copy the binary and you've copied the
web console.

**What does `all:dist` add over `dist`?**
The `all:` prefix includes files whose names start with `_` or `.`, which Vite
emits and a plain `//go:embed dist` would skip. Without it, the bundle would be
subtly incomplete.

**Why does a fresh build show a 404 at `/`?**
`web/dist/` is a build artifact, not committed source. A binary built before
`make web-build` embeds only a `.gitkeep` sentinel, so `HasAssets()` is false and
the daemon serves a helpful HTML 404 naming the fix (`make dist`) rather than a
blank one.

**How does reloading `/scanner` not 404?**
The SPA fallback: `spaHandler` stats the path, finds no such file, and serves
`index.html` instead, so React-Router resolves `/scanner` on the client. Real
assets (hashed JS/CSS) stat true and are served directly.

**Can the SPA shadow an API route?**
No. The mux routes `/api/v1/*` and `/metrics` to their specific handlers before
`/` matches, and `spaHandler` additionally refuses any `/api/` path as defence in
depth — pinned by `TestSPA_APIRoutesNotShadowed`.

## Series navigation

**Part 2 of 14** · ←
[Part 1: One Engine, Many Front-Ends]({{ '/blog/deep-dives/operator-cockpit-01-one-engine-many-frontends/' | relative_url }})
· Next →
[Part 3: The Connect Screen & Auth Handshake]({{ '/blog/deep-dives/operator-cockpit-03-connect-screen-auth/' | relative_url }})
</content>
