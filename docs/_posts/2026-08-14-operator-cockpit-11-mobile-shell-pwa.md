---
title: "The Operator's Cockpit, Part 11: The Mobile Shell, Command Palette & PWA"
description: How one React shell serves a desktop sidebar and a phone's bottom nav from a single nav registry, adds a ⌘K command palette as the fastest path to any panel, and installs as an offline-first PWA on a phone — with a service-worker denylist so the console never shadows its sibling apps.
category: deep-dives
keywords: react app shell responsive, command palette cmd k, pwa install phone, vite plugin pwa manifest, service worker precache, offline first spa, bottom nav mobile drawer, nav registry single source, navigate fallback denylist, gophertrunk operator cockpit
tags: [operator-cockpit, pwa, react, responsive, service-worker, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cockpit"
series_part: 11
---

*Part 11 of **The Operator's Cockpit**, the series on driving one GopherTrunk
daemon through one REST + SSE API from browser and terminal alike. The last four
posts filled the console with panels. This post is about the *frame* around them:
the layout that turns the same SPA into a desktop console and a phone app, the
command palette that reaches any panel in two keystrokes, and the PWA packaging
that lets you install the whole thing on your phone and point it at a Raspberry Pi
in the field.*

> **TL;DR:** One `AppShell` renders four navigation surfaces — a desktop sidebar,
> a mobile top bar, a mobile bottom nav, and a full drawer — all driven by a
> single **nav registry**, so every panel is reachable on every device. A global
> **⌘K / Ctrl-K command palette** fuzzy-jumps to any panel and is the discovery
> safety net for the deep ones. The SPA ships as an **installable PWA**: Vite's
> PWA plugin emits a manifest and a Workbox service worker that precaches the
> whole bundle for offline-first loading — with a **navigate-fallback denylist**
> so the console's service worker never shadows the daemon's sibling apps (Config
> Builder, Signal Lab, RF Scope, Crypto Lab).

**Key takeaways**

- **One shell, four nav surfaces.** `AppShell` composes a sidebar, top bar, bottom
  nav, and drawer; all four read the same registry, so a hidden or new tab appears
  consistently everywhere.
- **The command palette is the universal shortcut.** ⌘K opens a fuzzy switcher
  over every nav item — the fastest path to any panel, and how the deep panels
  stay discoverable without cluttering the primary nav.
- **The console installs like an app.** Vite-Plugin-PWA emits the manifest and a
  Workbox SW that precaches the bundle, so it opens instantly and runs offline
  (the connect screen still needs a live daemon).
- **The service worker knows its lane.** A navigate-fallback denylist keeps the
  main console's SW from answering `/config/`, `/siglab/`, `/rfscope/`,
  `/cryptolab/`, or `/api/` with its own cached `index.html`.

## Cheat sheet

| Piece | What it does | Where it lives |
|---|---|---|
| App shell | composes all nav surfaces around the routed page | `web/src/components/shell/AppShell.tsx` |
| Bottom nav | mobile primary tabs + "More" | `web/src/components/shell/BottomNav.tsx` |
| Mobile drawer | full nav sheet behind "More" | `web/src/components/shell/MobileDrawer.tsx` |
| Top bar | title, live pill, palette trigger, hamburger | `web/src/components/shell/TopBar.tsx` |
| Command palette | ⌘K fuzzy quick-switcher | `web/src/components/CommandPalette.tsx` |
| PWA manifest + SW | installable, offline-first packaging | `web/vite.config.ts` (`VitePWA`) |
| SW denylist | don't shadow sibling consoles | `web/src/lib/swDenylist.ts` |

## In this post

- **One shell, four nav surfaces** — desktop and phone from one layout.
- **The registry as the single source** — why nav is data, not markup.
- **The command palette** — ⌘K, fuzzy ranking, keyboard-first.
- **Installing as a PWA** — manifest, precache, offline-first.
- **The denylist** — how the service worker avoids shadowing siblings.

## One shell, four nav surfaces

`AppShell` is pure chrome. It owns no connection or data state — it composes the
navigation surfaces around whatever page the router hands it as `children`, and
manages exactly three bits of local UI state: the drawer, the palette, and the
sidebar's collapsed flag.

```tsx
// web/src/components/shell/AppShell.tsx (shape)
export function AppShell({ children }: Props) {
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [paletteOpen, setPaletteOpen] = useState(false);
  const isDesktop = useIsDesktop();

  // Global ⌘K / Ctrl-K toggles the palette from anywhere.
  useEffect(() => { /* window keydown → setPaletteOpen(o => !o) */ }, []);
  // The drawer is mobile-only; close it if the viewport grows to desktop.
  useEffect(() => { if (isDesktop && drawerOpen) setDrawerOpen(false); }, [isDesktop, drawerOpen]);

  return (
    <div className="min-h-full flex">
      <Sidebar collapsed={collapsed} onToggleCollapse={toggleCollapse} /> {/* desktop */}
      <div className="flex-1 flex flex-col min-w-0">
        <TopBar onOpenMore={() => setDrawerOpen(true)} onOpenPalette={() => setPaletteOpen(true)} />
        <main id="main" className="flex-1 p-3 sm:p-4 pb-24 sm:pb-4">{children}</main>
      </div>
      <BottomNav onOpenMore={() => setDrawerOpen(true)} />       {/* mobile */}
      <MobileDrawer open={drawerOpen} onClose={() => setDrawerOpen(false)} />
      <CommandPalette open={paletteOpen} onClose={() => setPaletteOpen(false)} />
      <ToastViewport />
    </div>
  );
}
```

The responsiveness is entirely Tailwind breakpoints, not JS branching: the
`Sidebar` is hidden below `sm`, the `BottomNav` and `TopBar` hamburger are hidden
at `sm` and up. The same DOM serves both; CSS decides which surface shows. That's
why a phone rotating to landscape, or a browser window growing, just works — and
why the drawer-close effect exists at all, to tidy up the one case CSS can't (a
mobile-only overlay left open when the viewport crosses into desktop).

Note the accessibility bones baked into the frame: a "Skip to content" link that
appears on focus, a semantic `<main id="main">`, and `pb-24` bottom padding so the
fixed bottom nav never covers the last row of a scrolling panel. The shell is
small, but it carries the whole app's keyboard and screen-reader story.

## The registry as the single source

None of these four surfaces hard-code a list of panels. They all read a shared
**nav registry**, and each surface asks it a slightly different question. The
bottom nav takes the primary items and filters hidden ones:

```tsx
// web/src/components/shell/BottomNav.tsx (shape)
export function BottomNav({ onOpenMore }: Props) {
  const hidden = new Set(useShared((s) => s.hiddenTabs));
  const items = PRIMARY_ITEMS.filter((i) => !hidden.has(tabKey(i)));
  // …render four primaries + a fifth "More" button that opens the drawer
}
```

The drawer asks for the *grouped* view (`useVisibleNavGroups`), the palette asks
for the *flat* visible list (`useVisibleNavItems`), and the top bar looks up the
current route's label in `NAV_ITEMS`. One data source, four projections. The
payoff is the same one the whole series keeps hitting: a tab hidden via config
(`web.tabs`) disappears from *every* surface at once, and a new panel is a new
registry entry, not four edits. It's the exact discipline the TUI applies to its
own tab strip — the same `web.tabs` keys hide a panel in both UIs, a thread we
pick up in Part 12.

<figure class="lab-figure">
<svg viewBox="0 0 660 200" width="660" height="200" role="img" aria-label="One nav registry feeds four navigation surfaces. The registry is the single source; the desktop sidebar reads the grouped view, the mobile bottom nav reads the primary items, the mobile drawer reads the grouped view, and the command palette reads the flat visible list. A tab hidden by config disappears from all four at once.">
  <rect x="250" y="16" width="160" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="330" y="34" text-anchor="middle" fill="var(--accent)" font-size="11">nav registry</text>
  <text x="330" y="49" text-anchor="middle" fill="var(--fg-muted)" font-size="8">single source of nav</text>
  <line x1="290" y1="58" x2="120" y2="112" stroke="currentColor"/><polygon points="120,108 110,112 122,116" fill="currentColor"/>
  <line x1="315" y1="58" x2="270" y2="112" stroke="currentColor"/><polygon points="270,108 262,114 274,116" fill="currentColor"/>
  <line x1="345" y1="58" x2="400" y2="112" stroke="currentColor"/><polygon points="398,108 406,114 394,116" fill="currentColor"/>
  <line x1="370" y1="58" x2="540" y2="112" stroke="currentColor"/><polygon points="538,108 550,116 534,116" fill="currentColor"/>
  <rect x="42" y="114" width="140" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="112" y="131" text-anchor="middle" fill="currentColor" font-size="9">desktop sidebar</text>
  <text x="112" y="145" text-anchor="middle" fill="var(--fg-muted)" font-size="8">grouped view</text>
  <rect x="196" y="114" width="140" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="266" y="131" text-anchor="middle" fill="currentColor" font-size="9">bottom nav</text>
  <text x="266" y="145" text-anchor="middle" fill="var(--fg-muted)" font-size="8">primaries + More</text>
  <rect x="350" y="114" width="140" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="420" y="131" text-anchor="middle" fill="currentColor" font-size="9">mobile drawer</text>
  <text x="420" y="145" text-anchor="middle" fill="var(--fg-muted)" font-size="8">grouped view</text>
  <rect x="504" y="114" width="140" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="574" y="131" text-anchor="middle" fill="var(--accent)" font-size="9">command palette</text>
  <text x="574" y="145" text-anchor="middle" fill="var(--fg-muted)" font-size="8">flat visible list</text>
  <text x="330" y="184" text-anchor="middle" fill="var(--fg-muted)" font-size="10">hide a tab via web.tabs → it disappears from all four surfaces at once</text>
</svg>
<figcaption>Navigation is data, not markup. Four surfaces are four projections of one registry, so a hidden or new tab is consistent everywhere.</figcaption>
</figure>

## The command palette

The palette is the fastest path to any panel and the discovery net for the deep
ones — the panels that don't earn a slot in the primary nav still turn up the
moment you type. It's a ⌘K-triggered dialog over the flat visible nav list, with
arrow keys to move, Enter to activate, Escape to close, and a focus trap so the
keyboard stays inside it:

```tsx
// web/src/components/CommandPalette.tsx (shape)
export function CommandPalette({ open, onClose }: Props) {
  const items = useVisibleNavItems();
  const results = useMemo(() => filterItems(items, query), [items, query]);
  // …ArrowUp/Down move `active`; Enter → choose(results[active]); Esc closes
  function choose(item) {
    onClose();
    if (item.external) window.open(item.to, "_blank", "noopener");
    else navigate(item.to);
  }
}

// filterItems ranks: label startsWith (3) > label includes (2) > keyword hit (1).
function filterItems(items, query) {
  // …empty query returns everything in registry order; else score + sort desc
}
```

The ranking is intentionally simple and predictable: a label that *starts with*
your query beats one that merely *contains* it, which beats a keyword match. Empty
query lists everything in registry order, so ⌘K with no typing is a full index.
External items (the sibling consoles) open in a new tab; internal ones navigate in
place. It's a small component, but it's the single control that makes a
twenty-something-panel console feel small — you never hunt through nav, you type
two letters.

## Installing as a PWA

The console is a static bundle with no runtime Node dependency and no CDN fetches,
which is exactly what makes it installable. Vite-Plugin-PWA emits the manifest and
a Workbox service worker at build time:

```ts
// web/vite.config.ts (shape)
VitePWA({
  registerType: "autoUpdate",
  manifest: {
    name: "GopherTrunk", short_name: "GopherTrunk",
    display: "standalone", start_url: "./", scope: "./",
    theme_color: "#0f172a", background_color: "#0f172a",
    icons: [ /* favicon.svg, purpose any + maskable */ ],
  },
  workbox: {
    // Precache the whole SPA so it installs offline-first and loads instantly.
    globPatterns: ["**/*.{js,css,html,svg,png,ico,webmanifest}"],
    navigateFallbackDenylist: SW_NAVIGATE_FALLBACK_DENYLIST,
  },
});
```

`display: "standalone"` and the relative `start_url`/`scope` are what let a phone
"Add to Home Screen" the console and launch it chromeless, pointed at whatever
daemon it was installed from. The Workbox `globPatterns` precache the entire
bundle, so the UI itself loads instantly and works without a network round-trip
for its assets — the connect screen still needs a live daemon to do anything
useful, but the *app* is always there. The base path defaults to `./`, so the
same `dist/` works opened via `file://`, hosted at a static path, or rooted at
`/`. One build, every deployment shape.

### How that principle shaped the front-end

- **Everything is inlined at build.** No CDN scripts, no external fonts — the
  precache list is meaningful because the whole app is genuinely self-contained.
- **`autoUpdate` keeps installs fresh.** A new daemon build ships a new SW that
  updates the installed app on next load, so a field phone doesn't drift.
- **API responses are never cached.** The SW precaches assets only; `/api/*` and
  `/metrics` are always fetched live, so the console never shows stale daemon
  state from a cache.

## The denylist: knowing its lane

There's a sharp edge to a service worker scoped at `/`: by default its
`navigateFallback` answers *any* navigation with the cached `index.html`. But the
daemon mounts sibling consoles at subpaths — Config Builder at `/config/`, Signal
Lab at `/siglab/`, RF Scope at `/rfscope/`, Crypto Lab at `/cryptolab/`. Without a
guard, opening one of those in a new tab would get the *main* console's
`index.html`, and the tab would boot at a route React Router doesn't know — a
blank page. The denylist is the fix:

```ts
// web/src/lib/swDenylist.ts (shape)
export const SW_NAVIGATE_FALLBACK_DENYLIST: RegExp[] = [
  /^\/api\//, /^\/metrics/,
  /^\/config\//, /^\/siglab\//, /^\/rfscope\//, /^\/cryptolab\//,
];
```

Any navigation matching one of these is *not* served the main SPA's fallback — it
goes to the network, so the sibling app loads. This list is kept in sync with the
external console links in the nav registry, and a unit test (`swDenylist.test.ts`)
enforces that sync so a new sibling console can't be added without teaching the
service worker to stay out of its way. It's a tiny file that encodes an important
boundary: one console's offline cache must never eat another console's front door.

## Where this goes next

[Part 12]({{ '/blog/deep-dives/operator-cockpit-12-the-tui-cockpit/' | relative_url }})
leaves the browser entirely for the terminal — the Bubbletea TUI that drives the
*same* REST + SSE API from an SSH session. It's the other half of the series'
through-line made concrete: an Elm-model reducer, a polling fan plus an SSE pump,
and a tab strip that hides the same panels the web `web.tabs` config hides here.

## FAQ

**How does the same UI serve a desktop and a phone?**
`AppShell` renders all four navigation surfaces into one DOM; Tailwind breakpoints
show the sidebar on desktop and the top bar + bottom nav + drawer on mobile. There
is almost no JS branching — CSS decides which surface is visible, so resizing or
rotating just works.

**What powers the command palette?**
The shared nav registry. ⌘K opens a fuzzy quick-switcher over every visible nav
item, ranked by how the query matches the label (startsWith > includes > keyword).
It's the fastest route to any panel and how the deep panels stay discoverable.

**Does the PWA work offline?**
The app *shell* does — Workbox precaches the whole bundle, so the console loads
instantly and runs without a network round-trip for its assets. But it's an
operator console for a live daemon: the connect screen still needs a reachable
daemon to show real data. API responses are never cached.

**Why is there a service-worker denylist?**
Because the console's SW is scoped at `/`, and the daemon mounts sibling apps at
`/config/`, `/siglab/`, `/rfscope/`, `/cryptolab/`. The denylist stops the SW from
answering those navigations (and `/api/`, `/metrics`) with its own cached
`index.html`, which would otherwise open those tabs blank.

**Can I install it pointed at any daemon?**
Yes — `start_url`/`scope` are relative (`./`), so installing from a given daemon's
web console launches the app pointed back at that daemon. The bundle is fully
self-contained, so it also works served from any static path or even `file://`.

## Series navigation

**Part 11 of 14** · ←
[Part 10: Write Mode — Mutating Config Safely]({{ '/blog/deep-dives/operator-cockpit-10-write-mode/' | relative_url }})
· Next →
[Part 12: The TUI Cockpit — The Same API in Your Terminal]({{ '/blog/deep-dives/operator-cockpit-12-the-tui-cockpit/' | relative_url }})
