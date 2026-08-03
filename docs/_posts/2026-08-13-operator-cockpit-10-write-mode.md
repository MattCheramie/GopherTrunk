---
title: "The Operator's Cockpit, Part 10: Write Mode — Mutating Config Safely"
description: How GopherTrunk lets a browser or terminal change a running daemon without lying about what took effect — the mutation-capability gate, pointer-field PATCH semantics that preserve leave-alone, the hot-reload vs restart-required split, an mtime conflict guard, and the validate-then-activate config swap.
category: deep-dives
keywords: hot reload config daemon, patch settings api, restart required badge, mutation capability gate, config activate reload restart, mtime conflict 409, leave-alone pointer fields, safe config edits sdr, live edits web ui, gophertrunk operator cockpit
tags: [operator-cockpit, config, rest, safety, http, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cockpit"
series_part: 10
---

*Part 10 of **The Operator's Cockpit**, the series on driving one GopherTrunk
daemon through one REST + SSE API from browser and terminal alike. Every panel so
far has been a renderer over read-only endpoints. This post crosses to the write
side — how the same API lets you retune, lock out a talkgroup, or hot-swap a whole
config file, and how it refuses to lie to you about what actually took effect.*

> **TL;DR:** Write-mode is built on three ideas. **A capability gate**:
> `GET /api/v1/mutations` reports `can_mutate` plus per-subsystem writability so a
> UI lights up write actions only when they'd succeed. **Honest PATCH semantics**:
> settings and talkgroup edits use pointer fields so JSON-omitted keys mean
> "leave alone", and the response splits fields into `applied` (took effect now)
> vs `restart_required` (written but needs a restart). **Validate → activate**:
> swapping the whole config file resolves the path against an allow-list, then
> either hot-reloads in place or re-execs the daemon — with an mtime guard that
> returns `409` rather than clobbering an external edit.

**Key takeaways**

- **The daemon tells the UI what it can do.** `can_mutate` and the
  `*_writable` flags mean write buttons appear only when the wiring and auth allow
  them — no probing for a `403`.
- **Pointer fields preserve intent.** A PATCH carries only the keys you set;
  omitted fields are never zeroed, because the wire types use `*T` and the daemon
  dispatches only non-nil ones.
- **"Applied" vs "restart required" is a first-class answer.** Every mutating
  response says which fields hot-applied and which were only written to disk, so
  the UI can render an honest "restart required" badge.
- **Config swaps validate before they act, and won't clobber.** The path is
  allow-listed and stat-checked; a `reload` reports what it could hot-apply, a
  `restart` re-execs; a stale-mtime write returns `409`, not a silent overwrite.

## Cheat sheet

| Endpoint | What it does | Where it lives |
|---|---|---|
| `GET /api/v1/mutations` | capability gate: `can_mutate` + `*_writable` | `internal/api/handlers_mutations.go` (`handleMutationStatus`) |
| `PATCH /api/v1/talkgroups/{id}` | pointer-field policy edit | `handlers_mutations.go` (`handleUpdateTalkgroup`) |
| `POST /api/v1/calls/{serial}/end` | force-release a call | `handlers_mutations.go` (`handleEndCall`) |
| `PATCH /api/v1/settings` | live edit → applied / restart_required | `internal/api/handlers_settings.go` (`handleSettingsPatch`) |
| `applyHotReload` | route each field to hot or cold | `handlers_settings.go` |
| `POST /api/v1/config/activate` | reload or restart into a config file | `internal/api/handlers_config_activate.go` |

## In this post

- **The capability gate** — how the UI knows what it may write.
- **Pointer-field PATCH** — leave-alone semantics on the wire.
- **Hot vs cold** — the applied / restart-required split.
- **The mtime guard** — why an external edit gets a `409`.
- **Validate → activate** — swapping the running config safely.

## The capability gate

Before a UI shows a single write control, it asks the daemon what it's allowed to
do. `GET /api/v1/mutations` always answers `200` with a capability snapshot — the
auth mode, whether *this* request would be accepted, and which subsystems are even
wired for mutation:

```go
// internal/api/handlers_mutations.go (shape)
func (s *Server) handleMutationStatus(w http.ResponseWriter, r *http.Request) {
    canMutate := s.auth.canMutate(r)
    writeJSON(w, http.StatusOK, map[string]any{
        "auth_mode":          s.auth.mode.String(),   // "auto" | "required" | "disabled"
        "can_mutate":         canMutate,              // would this request pass auth?
        "allow_mutations":    canMutate,              // legacy alias (deprecated)
        "engine_writable":    s.mutator != nil,       // can we end calls / retune?
        "retention_writable": s.retention != nil,     // can we sweep retention?
        "tones_writable":     s.tones != nil,         // can we reset tone-out?
    })
}
```

That single call is why the browser and TUI can *light up* write keybindings
instead of probing a real endpoint and catching a `401`/`403`. The `*_writable`
flags are granular enough that a panel can show the exact reason a button is
inert — "tone-out detector not wired" versus "mutations disabled at daemon" —
which is far kinder than a greyed-out button with no explanation. The TUI ANDs
this with its own `--write` flag; the browser gates on the same snapshot. The
daemon is the authority on capability, and it says so up front.

## Pointer-field PATCH: leave-alone on the wire

The mutating edits — talkgroup policy, settings — are all **partial** updates, and
GopherTrunk encodes "partial" the same way everywhere: pointer fields. A talkgroup
PATCH body is a struct of `*T`, so a JSON-omitted field decodes to `nil` and is
never applied, and only the keys you actually sent are written:

```go
// internal/api/handlers_mutations.go (shape)
type updateTalkgroupRequest struct {
    Priority *int    `json:"priority"`
    Lockout  *bool   `json:"lockout"`
    Scan     *bool   `json:"scan"`
    Stream   *bool   `json:"stream"`
    // …Record, Mute, Icon — all pointers, all optional
}

func (s *Server) handleUpdateTalkgroup(w http.ResponseWriter, r *http.Request) {
    // …parse id, decode req; reject an all-nil body as 400
    s.talkgroups.UpdateFields(uint32(id), func(tg *trunking.TalkGroup) {
        if req.Priority != nil { tg.Priority = *req.Priority }
        if req.Lockout != nil  { tg.Lockout  = *req.Lockout  }
        if req.Scan != nil     { tg.Scan     = *req.Scan     }
        // …only non-nil fields are applied
    })
    writeJSON(w, http.StatusOK, talkgroupToDTO(s.talkgroups.Lookup(uint32(id))))
}
```

An all-nil body is a `400` ("supply at least one of…"), a missing talkgroup is a
`404`, and success returns the full updated record so the UI re-renders from truth
rather than optimistically guessing. The pattern repeats on the simpler mutators:
`handleEndCall` force-releases the call on a device serial (with an optional
`reason`, defaulting to `manual`) and `404`s when no call holds that device;
`handleRetentionSweep` and `handleToneReset` each `503` when their subsystem isn't
wired. Every one of them is either a clean success with the new state, or a
specific, actionable status code.

## Hot vs cold: the applied / restart-required split

Settings are where "did it take effect?" gets interesting, because some fields the
daemon can change live and others it genuinely can't. `PATCH /api/v1/settings`
takes a pointer-field patch (same leave-alone rule), writes it to `config.yaml`,
then walks the patch and dispatches each field to a live applier when one exists —
returning **two lists** so the UI can tell the truth:

```go
// internal/api/handlers_settings.go (shape)
func (s *Server) applyHotReload(p config.Patch) (applied, restartRequired []string) {
    app := s.settings // may be nil — then every field falls back to cold
    if p.AudioVolume != nil {
        if app != nil { app.SetAudioVolume(*p.AudioVolume); applied = append(applied, "audio.volume") }
        else          { restartRequired = append(restartRequired, "audio.volume") }
    }
    if p.LogLevel != nil {
        if app != nil && app.SetLogLevel(*p.LogLevel) == nil { applied = append(applied, "log.level") }
        else                                                  { restartRequired = append(restartRequired, "log.level") }
    }
    // …audio.muted/enabled, scanner.scan_mode, recordings.write_raw/enhance — hot when applier present
    // …then a table of cold-only keys (api.http_addr, sdr.sample_rate, storage.path, …) → restartRequired
    return applied, restartRequired
}
```

Some fields — the listen address, the SDR sample rate, the storage path — simply
cannot change under a running daemon, so they're written to `config.yaml` and
listed in `restart_required`. The others hot-apply through a `SettingsApplier`
interface (volume, mute, log level, scan mode, the voice-enhance toggle) and land
in `applied`. The response carries both lists plus a fresh runtime snapshot, and
the UI renders a "restart required" badge on exactly the cold fields. Nothing is
silently ignored, and nothing pretends to have taken effect when it didn't.

<figure class="lab-figure">
<svg viewBox="0 0 660 210" width="660" height="210" role="img" aria-label="The settings PATCH flow. A pointer-field patch is written to config.yaml through the ConfigWriter, which mtime-guards against external edits and returns 409 on conflict. On success, applyHotReload splits the patch fields into two lists: applied fields are dispatched to the live SettingsApplier and take effect now; restart-required fields are only written to disk. The response carries both lists so the UI badges the cold fields.">
  <rect x="10" y="84" width="120" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="70" y="102" text-anchor="middle" fill="var(--accent)" font-size="10">PATCH settings</text>
  <text x="70" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="8">pointer fields</text>
  <line x1="130" y1="105" x2="164" y2="105" stroke="currentColor"/><polygon points="164,101 174,105 164,109" fill="currentColor"/>
  <rect x="174" y="84" width="128" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="238" y="100" text-anchor="middle" fill="currentColor" font-size="9">WritePatch</text>
  <text x="238" y="114" text-anchor="middle" fill="var(--fg-muted)" font-size="8">mtime guard → 409</text>
  <line x1="302" y1="105" x2="336" y2="105" stroke="currentColor"/><polygon points="336,101 346,105 336,109" fill="currentColor"/>
  <rect x="346" y="84" width="128" height="42" rx="6" fill="none" stroke="currentColor"/>
  <text x="410" y="102" text-anchor="middle" fill="currentColor" font-size="9">applyHotReload</text>
  <text x="410" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="8">split per field</text>
  <line x1="410" y1="84" x2="410" y2="60" stroke="currentColor"/><polygon points="406,60 410,50 414,60" fill="currentColor"/>
  <line x1="474" y1="96" x2="520" y2="70" stroke="var(--accent)"/><polygon points="520,66 530,70 520,74" fill="var(--accent)"/>
  <line x1="474" y1="114" x2="520" y2="140" stroke="var(--fg-muted)" stroke-dasharray="4 3"/><polygon points="520,136 530,140 519,144" fill="var(--fg-muted)"/>
  <rect x="530" y="48" width="122" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="591" y="65" text-anchor="middle" fill="var(--accent)" font-size="9">applied</text>
  <text x="591" y="79" text-anchor="middle" fill="var(--fg-muted)" font-size="8">live now</text>
  <rect x="530" y="120" width="122" height="40" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="591" y="137" text-anchor="middle" fill="var(--fg-muted)" font-size="9">restart_required</text>
  <text x="591" y="151" text-anchor="middle" fill="var(--fg-muted)" font-size="8">on disk only</text>
  <text x="330" y="192" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the response carries both lists — the UI badges exactly the cold fields</text>
</svg>
<figcaption>Every settings write returns which fields took effect now and which need a restart. Honesty about effect is the whole design.</figcaption>
</figure>

### How that principle shaped the Go code

- **The applier is optional.** `s.settings` can be nil; when it is, *every* field
  falls to `restart_required`. The code degrades to "written but not applied"
  rather than crashing or silently dropping.
- **The writer is an interface.** `ConfigWriter` and `SettingsApplier` are seams,
  so the API package doesn't pull in the OS file machinery and tests can fake both.
- **`Path()` empty means read-only.** A daemon started without `-config` has no
  file to write; the endpoint `503`s and the UI renders Settings read-only rather
  than offering edits that can't persist.

## The mtime guard: a 409, not a clobber

Config edits can race. You might be editing `config.yaml` in a text editor while
the browser tries to PATCH it. GopherTrunk refuses to be the one that loses your
work: the writer stamps and checks the file's mtime, and a mismatch surfaces as a
distinct **conflict** rather than a generic failure:

```go
// internal/api/handlers_settings.go (shape)
if _, err := s.configWriter.WritePatch(patch); err != nil {
    if isExternalEditConflict(err) {                 // "modified externally"
        s.writeError(w, http.StatusConflict, err.Error()) // 409 → the UI can say
        return                                              //   "reload, it changed under you"
    }
    s.writeError(w, http.StatusBadRequest, err.Error())
    return
}
```

The `409` is deliberate: it lets the UI present a clear "the config changed under
you — reload before saving" toast instead of a scary generic error, and it means
the daemon never overwrites an external edit it didn't see. Safe write is as much
about *not* writing at the wrong moment as it is about the write itself.

## Validate → activate: swapping the whole config

The heaviest write is repointing the daemon at a different config file. That's
`POST /api/v1/config/activate`, and it's built as two clean phases — **validate**,
then **activate** — with the activation itself offering two modes:

```go
// internal/api/handlers_config_activate.go (shape)
func (s *Server) handleConfigActivate(w http.ResponseWriter, r *http.Request) {
    if s.configActiv == nil { s.writeError(w, 503, "activation not supported"); return }
    // — validate —
    path, err := s.configBuilder.resolvePath(req.Path) // allow-list + .yaml/.yml
    if err != nil { s.writeError(w, 400, "config: "+err.Error()); return }
    if _, err := os.Stat(path); err != nil { s.writeError(w, 404, "config: "+err.Error()); return }

    switch req.Mode {
    case "", "reload": // hot-apply what it can, report the rest
        applied, restartRequired, aerr := s.configActiv.ActivateReload(path)
        // …400 on load error, else 200 with { applied, restart_required }
    case "restart":    // re-exec the daemon so every field takes effect
        // …202 Accepted; the process tears down and re-execs — the SPA expects a reconnect
    }
}
```

The `resolvePath` step is the safety fence: the requested path is checked against
the config builder's allow-list of directories and must carry a `.yaml`/`.yml`
extension before anything touches it (the same resolver we'll meet again driving
the config builder in Part 13). A `reload` returns the familiar
`applied` / `restart_required` split — it hot-applies what the daemon can and
tells you what still needs a restart. A `restart` re-execs the process and returns
`202`, a signal to the SPA that a disconnect-and-reconnect is coming. Both modes
share the same validation front door; they differ only in how completely they take
effect. The finished experience — editing, staging, and activating live — is the
[Live edits]({{ '/live-edits.html' | relative_url }}) operator surface.

## Where this goes next

[Part 11]({{ '/blog/deep-dives/operator-cockpit-11-mobile-shell-pwa/' | relative_url }})
leaves the API and looks at the frame around it: the mobile shell, the command
palette, and installing the console as a PWA on a phone — how one React layout
serves a desktop sidebar and a phone's bottom nav from a single nav registry, and
how the whole SPA precaches so it opens instantly against a Raspberry Pi in the
field.

## FAQ

**How does the UI know whether it can write at all?**
It reads `GET /api/v1/mutations` — `can_mutate` says whether the current request
would pass auth, and the `*_writable` flags say which subsystems are wired. The
browser and TUI light up write controls from that snapshot instead of probing for
a rejection.

**What happens to fields I don't include in a PATCH?**
Nothing — they're left alone. The wire types use pointer fields, so a JSON-omitted
key decodes to `nil` and is never applied. Only the keys you send are written.

**Why does a settings change sometimes say "restart required"?**
Because some fields can't change under a running daemon (the listen address, the
SDR sample rate, the storage path). Those are written to `config.yaml` and returned
in `restart_required`; the hot-reloadable ones (volume, mute, log level, scan mode,
voice-enhance) are applied immediately and returned in `applied`.

**What's the 409 on a settings save?**
The config file's mtime changed since the daemon last read it — usually because
you edited it externally. Rather than clobber that edit, the daemon returns `409`
so the UI can prompt you to reload before saving.

**What's the difference between reload and restart on config activate?**
`reload` hot-applies what it can and reports what still needs a restart; `restart`
re-execs the daemon so *every* field takes effect and returns `202`, warning the
SPA to expect a reconnect. Both validate the path against an allow-list first.

## Series navigation

**Part 10 of 14** · ←
[Part 9: The Map — Plotting Sites & Emitters]({{ '/blog/deep-dives/operator-cockpit-09-the-map/' | relative_url }})
· Next →
[Part 11: The Mobile Shell, Command Palette & PWA]({{ '/blog/deep-dives/operator-cockpit-11-mobile-shell-pwa/' | relative_url }})
