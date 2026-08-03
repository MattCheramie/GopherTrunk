---
title: "The Operator's Cockpit, Part 12: The TUI Cockpit — The Same API in Your Terminal"
description: How GopherTrunk's Bubbletea TUI drives the identical REST + SSE API as the browser — an Elm-model root reducer fanning out polling commands plus a long-lived SSE pump into one SharedState, thirteen panels as pure renderers over it, and a tab strip that hides the same web.tabs panels the web SPA does.
category: deep-dives
keywords: bubbletea tui sdr, elm architecture go, sse client terminal, rest polling fan, shared state reducer, terminal operator console, tui panels renderer, web tabs both uis, charmbracelet lipgloss, gophertrunk operator cockpit
tags: [operator-cockpit, tui, bubbletea, sse, terminal, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cockpit"
series_part: 12
---

*Part 12 of **The Operator's Cockpit**, the series on driving one GopherTrunk
daemon through one REST + SSE API from browser and terminal alike. Eleven posts
have built out a React console. This post proves the "and terminal" half of the
promise: a full-screen Bubbletea TUI that drives the **exact same** REST + SSE API
over SSH — no gRPC, no private socket, the same endpoints the browser hits.*

> **TL;DR:** The TUI is an Elm-architecture Bubbletea program. Its root `Model`
> holds a `SharedState` snapshot; `Init` kicks off a **fan of polling commands**
> (health, systems, talkgroups, active calls, metrics, devices, scanner, hunt, …)
> plus one **long-lived SSE pump**. `Update` is the reducer: root keys (quit,
> help, palette, panel nav) win first, then poll/SSE messages fold into
> `SharedState` and reschedule themselves, then the active panel gets the message.
> The thirteen panels are pure renderers over `SharedState`. It reconnects SSE
> with backoff, hides the **same `web.tabs` panels** the SPA hides, and gates
> writes on the same `/api/v1/mutations` capability. Same daemon, same contract,
> different renderer.

**Key takeaways**

- **It's the same API, not a parallel one.** The TUI calls the identical REST
  endpoints and consumes the identical SSE event stream the browser does — the
  through-line of the series, made literal.
- **One reducer, one SharedState.** `Update` folds every poll and event into a
  single snapshot; panels never fetch — they read `SharedState` and render.
- **Polls reschedule themselves; SSE is a pump.** Each poll message re-arms its
  own timer, and a single SSE goroutine feeds events into the reducer, reconnecting
  with backoff on drop.
- **Consistency with the web is enforced, not hoped for.** The same `web.tabs`
  keys hide a panel in both UIs, and the same `/api/v1/mutations` snapshot gates
  writes — the TUI and SPA can't drift on what's visible or writable.

## Cheat sheet

| Piece | What it does | Where it lives |
|---|---|---|
| Root model | Elm-model reducer + chrome | `internal/tui/app.go` (`Model`) |
| Polling fan | one poll Cmd per read endpoint | `app.go` (`Init`, `cmdPoll*`) |
| SSE pump | long-lived event stream + reconnect | `internal/tui/client/sse.go` (`Stream` / `parseSSE`) |
| SharedState | the snapshot all panels read | `internal/tui/state/state.go` |
| Panels | pure renderers over SharedState | `internal/tui/panels/` (e.g. `dashboard.go`) |
| Global keys | quit, help, palette, panel jumps | `internal/tui/keys.go` |
| Write gate | AND of `--write` and daemon capability | `state.go` (`WriteEnabled`) |

## In this post

- **The Elm model** — Init, Update, View, and why it fits a live console.
- **The polling fan** — one self-rescheduling Cmd per read endpoint.
- **The SSE pump** — the same event stream the browser gets, parsed in Go.
- **SharedState + panels** — snapshot in the middle, renderers on the edge.
- **Parity with the web** — same tabs hidden, same writes gated.

## The Elm model

Bubbletea is The Elm Architecture in Go: a `Model`, an `Init` that returns startup
commands, an `Update` reducer that folds messages into new state and returns more
commands, and a `View` that renders the model to a string. GopherTrunk's root
`Model` is exactly that — it holds the current panel selection, the thirteen
panels, and the `SharedState` every panel reads:

```go
// internal/tui/app.go (shape)
type Model struct {
    cli    *client.Client
    active state.PanelKind
    panels []panels.Panel
    shared *state.SharedState

    eventCh    <-chan client.Event // the live SSE feed
    sseCancel  func()
    sseRetries int
    // …palette, modals, tab hit-rects, toast
}

func (m *Model) View() string {
    tabs := m.renderTabs()
    status := m.renderStatusBar()
    body := m.panels[m.active].View(m.width, bodyH, true, m.shared)
    return lipgloss.JoinVertical(lipgloss.Left, tabs, body, status)
}
```

`View` composes chrome (a tab strip and a status bar) around the active panel's
own render, using Lipgloss for layout. The panel gets the width, a body height,
and a pointer to `SharedState` — that's the whole contract. Everything the panel
needs to draw is already in the snapshot; it never reaches for the network.

## The polling fan

`Init` is where the console comes alive. It returns a `tea.Batch` of one polling
command per read endpoint, plus the SSE connect — a fan of concurrent fetches
that mirrors, one-for-one, the set of things the browser polls:

```go
// internal/tui/app.go (shape)
func (m *Model) Init() tea.Cmd {
    return tea.Batch(
        cmdPollHealth(m.cli), cmdPollVersion(m.cli),
        cmdPollSystems(m.cli), cmdPollTalkgroups(m.cli),
        cmdPollActive(m.cli), cmdPollMetrics(m.cli),
        cmdPollHistory(m.cli, client.HistoryFilter{Limit: 100}),
        cmdPollDevices(m.cli), cmdPollScanner(m.cli), cmdPollHunt(m.cli),
        cmdPollAudio(m.cli), cmdPollRuntime(m.cli),
        cmdMutationStatus(m.cli),
        connectSSE(m.cli),
    )
}
```

Each poll's *result* message re-arms its own timer inside `Update`, so the fan is
self-sustaining — there's no central scheduler, just each stream keeping itself
alive:

```go
// internal/tui/app.go (shape) — inside Update
case pollActiveMsg:
    if msg.err == nil {
        m.shared.ActiveCalls = msg.calls
        m.shared.LastPoll = time.Now()
    } else {
        m.toast(fmt.Sprintf("active: %v", msg.err))
    }
    cmds = append(cmds, scheduleAfter(pollActiveEvery, cmdPollActive(m.cli)))
```

That "fold the result, then reschedule myself" shape repeats for every endpoint.
The active-calls list refreshes on its own cadence, the systems list on its own,
metrics on theirs — each an independent, self-healing loop. A poll error becomes a
toast, not a crash, and the next tick tries again.

## The SSE pump

Polling gives the console a steady heartbeat; the SSE stream gives it *reactions*.
The TUI subscribes to `GET /api/v1/events` — the identical Server-Sent-Events feed
the browser's `EventSource` consumes — and parses it in Go. The parser is a
faithful little SSE implementation: dispatch on a blank line, accumulate `event:`
and `data:` fields, ignore `:` comments:

```go
// internal/tui/client/sse.go (shape)
func (c *Client) Stream(ctx context.Context) (<-chan Event, <-chan error) {
    // …GET /api/v1/events with Accept: text/event-stream, no per-request timeout
    // parseSSE(resp.Body, out) → one Event per dispatched block
}

// The data line is itself a JSON envelope {kind, timestamp, payload};
// parseSSE decodes it and forwards the payload as Event.Raw so panels can
// type-decode it — the same envelope the web UI consumes.
```

Inside `Update`, a single event both lands in the ring buffers and triggers
*targeted* refreshes — a `call.start` re-polls active calls, an `sdr.attached`
re-polls devices — so the console reacts inside one SSE round-trip instead of
waiting for the next poll tick:

```go
// internal/tui/app.go (shape) — inside Update
case eventMsg:
    m.shared.EventLog.(*RingBuf[client.Event]).Push(msg.ev)
    switch msg.ev.Kind {
    case "call.start", "call.end":
        cmds = append(cmds, cmdPollActive(m.cli), cmdPollScanner(m.cli))
    case "sdr.attached", "sdr.detached":
        cmds = append(cmds, cmdPollDevices(m.cli))
    // …cchunt.*, cc.locked/lost, audio.state
    }
    cmds = append(cmds, listenSSE(m.eventCh)) // keep pumping

case sseDownMsg:
    m.sseRetries++
    backoff := min(1<<m.sseRetries * time.Second, 30*time.Second)
    m.toast("event stream disconnected — reconnecting in " + backoff.String())
    cmds = append(cmds, tea.Tick(backoff, func(time.Time) tea.Msg { return connectSSE(m.cli)() }))
```

The reconnection policy lives in the reducer, exactly as `sse.go`'s doc comment
promises ("reconnection is the caller's responsibility"): a dropped stream becomes
an `sseDownMsg`, the retry count grows an exponential backoff capped at 30 s, and
a toast tells the operator what's happening. It's the same jittered-backoff
instinct the browser's WebSocket clients use, expressed as Bubbletea messages.

<figure class="lab-figure">
<svg viewBox="0 0 660 210" width="660" height="210" role="img" aria-label="The TUI reducer. A fan of self-rescheduling polling commands and one long-lived SSE pump both emit messages into the Update reducer, which folds them into a single SharedState snapshot. The active panel renders from SharedState. Root keys are handled before messages reach the panel; a dropped SSE stream schedules a backoff reconnect.">
  <rect x="12" y="40" width="130" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="77" y="57" text-anchor="middle" fill="currentColor" font-size="10">polling fan</text>
  <text x="77" y="71" text-anchor="middle" fill="var(--fg-muted)" font-size="8">self-reschedules</text>
  <rect x="12" y="130" width="130" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="77" y="147" text-anchor="middle" fill="currentColor" font-size="10">SSE pump</text>
  <text x="77" y="161" text-anchor="middle" fill="var(--fg-muted)" font-size="8">reconnect w/ backoff</text>
  <line x1="142" y1="60" x2="214" y2="98" stroke="currentColor"/><polygon points="214,94 224,99 213,102" fill="currentColor"/>
  <line x1="142" y1="150" x2="214" y2="112" stroke="currentColor"/><polygon points="214,116 224,111 213,108" fill="currentColor"/>
  <rect x="224" y="84" width="140" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="294" y="101" text-anchor="middle" fill="var(--accent)" font-size="11">Update reducer</text>
  <text x="294" y="115" text-anchor="middle" fill="var(--fg-muted)" font-size="8">root keys first</text>
  <line x1="364" y1="105" x2="410" y2="105" stroke="currentColor"/><polygon points="410,101 420,105 410,109" fill="currentColor"/>
  <rect x="420" y="84" width="120" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="480" y="101" text-anchor="middle" fill="var(--accent)" font-size="10">SharedState</text>
  <text x="480" y="115" text-anchor="middle" fill="var(--fg-muted)" font-size="8">one snapshot</text>
  <line x1="540" y1="105" x2="586" y2="105" stroke="currentColor"/><polygon points="586,101 596,105 586,109" fill="currentColor"/>
  <rect x="596" y="84" width="56" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="624" y="101" text-anchor="middle" fill="currentColor" font-size="9">panel</text>
  <text x="624" y="115" text-anchor="middle" fill="var(--fg-muted)" font-size="8">View()</text>
  <text x="330" y="194" text-anchor="middle" fill="var(--fg-muted)" font-size="10">panels never fetch — they read the snapshot the reducer maintains</text>
</svg>
<figcaption>Polls and the SSE pump feed one reducer; the reducer maintains one SharedState; panels render from it. The same shape as a Redux store fed by fetch + EventSource — in a terminal.</figcaption>
</figure>

## SharedState and the panels

The reducer's whole job is to keep one struct current. `SharedState` is the
snapshot of daemon-derived data all panels read — the exact analogue of the
browser's shared store:

```go
// internal/tui/state/state.go (shape)
type SharedState struct {
    Health      client.Health
    Systems     []client.SystemDTO
    Talkgroups  []client.TalkgroupDTO
    ActiveCalls []client.ActiveCallDTO
    Scanner     client.ScannerStatusDTO
    Runtime     client.RuntimeDTO
    EventLog    RingReader[client.Event]
    ToneAlerts  RingReader[client.Event]
    // …Devices, Hunt, Audio, Metrics, History + their *Err fields
    WriteEnabled bool                   // --write AND daemon capability
    Mutations    client.MutationStatus  // per-subsystem writability
}
```

The panels are correspondingly dumb — in the best way. The Dashboard panel is a
*pure renderer*: it owns no local state and simply lays four cards over
`SharedState`, collapsing the 2×2 grid to a vertical stack below 80 columns:

```go
// internal/tui/panels/dashboard.go (shape)
type DashboardPanel struct{} // no fields — no state to get wrong

func (p *DashboardPanel) View(width, height int, _ bool, s *state.SharedState) string {
    if width < 80 { /* stack Health / Active / Events / Tones vertically */ }
    // …else a 2×2 grid of dashboardCard(...) over s
}
```

Because the snapshot is the single source and the panels don't fetch, the panels
compose trivially: `Update` forwards each message to the active panel *after* it
has folded the poll/event data, so a panel always renders the freshest state. The
root handles the cross-cutting concerns — window resize, quit, help, the command
palette (`ctrl+p`), theme toggle (`ctrl+t`), and the numeric/tab panel jumps in
`keys.go` — before a message ever reaches a panel.

## Parity with the web is enforced

The series' claim is "one API, two renderers," and the TUI backs it with two
concrete parity mechanisms rather than good intentions.

First, **the same tabs hide in both UIs.** The runtime snapshot carries the
`web.tabs` hidden set, and `visiblePanels` filters the tab strip, cycling, jump
keys, and mouse hit-testing off it — using the *same keys* the SPA uses (a
panel's `Key()` is its web route minus the leading slash):

```go
// internal/tui/app.go (shape)
func (m *Model) visiblePanels() []state.PanelKind {
    hidden := setOf(m.shared.Runtime.HiddenTabs)  // same web.tabs keys as the SPA
    // …keep every panel whose Key() isn't hidden; never strand the operator
}
```

Second, **the same capability gates writes.** `WriteEnabled` is the AND of the
TUI's `--write` flag and the daemon's `/api/v1/mutations` `allow_mutations` — the
identical snapshot the browser reads in Part 10 — so a write keybinding either
fires the same mutation the web console would, or shows a toast explaining exactly
why it can't. Neither UI can decide on its own what's visible or writable; the
daemon is the authority, and both renderers obey it. The finished terminal surface
is documented on the [TUI]({{ '/tui.html' | relative_url }}) operator page.

## Where this goes next

[Part 13]({{ '/blog/deep-dives/operator-cockpit-13-reflect-driven-config-form/' | relative_url }})
digs into the one place the browser and terminal share *more* than an API — a
single reflect-driven schema that generates the config editor for both the web
form and a Bubbletea form, so the two config builders can't drift from the struct
or from each other.

## FAQ

**Does the TUI use a different backend than the web console?**
No. It calls the identical REST endpoints and consumes the identical SSE event
stream (`GET /api/v1/events`) the browser uses. There's no TUI-only API — it's a
second renderer over the same contract.

**How does it stay live without a browser's event loop?**
Bubbletea's message loop plays that role. A fan of polling commands re-arm their
own timers, and one long-lived SSE goroutine pushes events in as messages; the
`Update` reducer folds both into `SharedState`. A dropped stream schedules a
backoff reconnect.

**Are the panels fetching data themselves?**
No — panels are pure renderers over `SharedState`. Only the root model fetches
(via commands) and folds results into the snapshot; the active panel reads that
snapshot to draw. It's the terminal version of components reading a shared store.

**Why do the same panels hide in the TUI and the web UI?**
Because both read the `web.tabs` hidden set using the same keys (a panel's key is
its web route). Hiding a tab in config removes it from the SPA nav *and* the TUI
tab strip, so the two surfaces can't disagree on what's visible.

**Can the TUI change the daemon, or is it read-only?**
It can write, but it's gated: `WriteEnabled` is the AND of the `--write` flag and
the daemon's `/api/v1/mutations` capability — the same snapshot the browser uses.
Without both, write keybindings show a "mutations disabled" toast instead of
firing.

## Series navigation

**Part 12 of 14** · ←
[Part 11: The Mobile Shell, Command Palette & PWA]({{ '/blog/deep-dives/operator-cockpit-11-mobile-shell-pwa/' | relative_url }})
· Next →
[Part 13: The Reflect-Driven Config Form]({{ '/blog/deep-dives/operator-cockpit-13-reflect-driven-config-form/' | relative_url }})
