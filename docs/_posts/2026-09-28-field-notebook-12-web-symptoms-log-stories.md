---
title: "The Field Notebook, Part 12: Web Symptoms That Are Really Log Stories"
description: "Five GopherTrunk web-console complaints — History hours behind, an empty table, a reconnect storm, a permanent symbol: poor badge, a silent 0.1-second recording row — traced through the daemon log to the seam that broke: the UTC render, the r.rows key, the pre-upgrade 404, symbol_proto, and a voice chain that decoded nothing."
category: tutorials
keywords: gophertrunk web console troubleshooting, history panel behind by hours, sdr scanner web ui empty table, websocket reconnect storm 404, symbol poor decode clean, silent recording history row, utc local time web panel, reconnectingsocket backoff, gophertrunk field notebook
tags: [field-notebook, logs, web, react, troubleshooting, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Field Notebook"
series_part: 12
---

*Part 12 of **The Field Notebook**, a 14-part operator's tutorial that reads
GopherTrunk's `debug.log` one line family at a time — what each field
measures, what a healthy rig prints, which number is a noise meter and which
one means traffic, and which deep dive to open when a line goes wrong.
[Part 11]({{ '/blog/tutorials/field-notebook-11-voice-chain-lines/' | relative_url }})
read the composer's voice-chain lines down to the teardown reason. This part
leaves the log for the browser and comes straight back: five things reported
as **web bugs**, each of which the log named before anyone opened devtools.*

> **TL;DR:** The web console renders what `/api/v1/*` returns, so when a
> panel lies the referee is the daemon log — `recorder: call started` and
> `call ended reason=…` are the truth a table must agree with. Five field
> reports, five seams: History "stopped at 00:15" while the feed said 03:15
> (a UTC string slice — `lib/formatTime.ts` now owns every timestamp); an
> *empty* History for every query (`r.rows` read off a `{"calls":[…]}`
> envelope); a stale tab reconnecting 2–4×/s forever (the daemon upgraded the
> WebSocket **before** resolving the SDR — now a 404 via `api.ErrUnknownDevice`
> plus `reconnectingSocket.ts`); `symbol: poor` next to `decode: clean` on a
> TETRA rig (a P25 C4FM receiver on a π/4-DQPSK carrier — fixed by
> `symbol_proto`); and a silent 0.1 s WAV row (a DMO voice chain with
> `speech_frames=0` torn down by `reason=timeout`). Meta-lesson: `npm test`
> never typechecks, so a green `ci.yml` did not mean the console built.

**Key takeaways**

- **A panel renders an API response; the log records what happened.** When
  they disagree, `curl` the endpoint and read the daemon line for the same
  second before touching React.
- **Timestamps are the first suspect for "stale" data.** The API returns UTC
  RFC3339; a panel that slices the string shows GMT, and UTC+3 reads it
  as "History stopped updating".
- **Silent mismatches hide behind `?? []`.** A mistyped envelope key and an
  empty result look the same to a panel — only the API and the `recorder:`
  lines can prove rows exist.
- **A badge is only as right as the receiver behind it.** The symbol panels
  graded a TETRA carrier with a P25 receiver; the `decode:` chip, fed by the
  real counters, was right the whole time.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Local-time rendering | the ONLY way a panel renders a daemon timestamp | `web/src/lib/formatTime.ts` (`formatLocalDateTime`, `formatClock`) |
| History envelope | `/api/v1/calls/history` answers `{"calls":[…]}`; client reads `r.calls ?? []` | `internal/api/handlers.go`, `web/src/api/client.ts` |
| Unknown-device 404 | resolve the serial BEFORE the WebSocket upgrade | `internal/api/devicerrors.go` (`ErrUnknownDevice`, `rejectUnknownDevice`) |
| Reconnect discipline | backoff resets only after a frame arrives (or `OPEN_GRACE_MS` held) | `web/src/api/reconnectingSocket.ts` |
| Symbol receiver selector | per-protocol `symbol_proto` per spectrum device | `cmd/gophertrunk/spectrum_provider.go` (`symbolProtoFor`), `web/src/api/symbols.ts` |
| Segment keying | each over's `CallComplete` attaches to its call | `CallComplete.CallStartedAt`, `call_recordings` |

## In this post

- **What these symptoms are telling you** — a panel is downstream of the log.
- **What healthy looks like** — the recorder lines a History row must match.
- **Five stories, one table** — symptom → log evidence → seam → fix.
- **When the panel and the log disagree** — four checks, plus the CI lesson.

## What these symptoms are telling you

Every operator-facing surface is a client of the same daemon: the SPA reads
`/api/v1/*` over REST, the live feed over the WebSocket twin of the event
stream ([Cockpit Part 4]({{ '/blog/deep-dives/operator-cockpit-04-sse-to-react/' | relative_url }})),
the DSP scopes over `/api/v1/diag/symbols`
([Cockpit Part 8]({{ '/blog/deep-dives/operator-cockpit-08-constellation-eye-symbol/' | relative_url }})).
Nothing in the browser *decodes*. A web symptom is a claim about one of
three seams — **data**, **transport**, **render** — and the log sits upstream
of all three. The habit: find the daemon line for the same
second, ask which seam turns *that* line into *this* panel.

## What healthy looks like

A History row renders a call the recorder finished; healthy means it agrees
with these lines on time, talkgroup and duration:

```
INF recorder: call started device=cc:same-carrier:1 wav=../recordings/Metro/805/2026-09-10T12-17-27Z_805.wav tg=805 provoice=false vocoder=imbe
INF recorder: call ended device=cc:same-carrier:1 wav=../recordings/Metro/805/2026-09-10T12-17-27Z_805.wav duration=5.6s reason=normal
```

The `wav=` path carries the UTC start; History should show it in *your*
clock with `duration` matching. The unhealthy twin from the 20 Aug DMO run — same
lines, wrong numbers (that run printed the older `colour=` field names):

```
INF composer: tetra DMO voice follow started — DNB TCH/S decode + ACELP vocoder serial=cc:same-carrier:1 seed_hint=0x00000000 rate_hz=18000
INF recorder: call started device=cc:same-carrier:1 wav=../recordings/Site-DMO/0/2026-08-20T19-41-02Z_0.wav tg=0 provoice=false vocoder=tetra-acelp
INF composer: tetra DMO voice follow ended serial=cc:same-carrier:1 dnb_bursts=242 speech_frames=0 bfi_count=242 seed=0x00000000 seed_known=false
INF call ended device=cc:same-carrier:1 grant=tetra-dmo tg=0 438900000Hz reason=timeout
INF recorder: call ended device=cc:same-carrier:1 wav=../recordings/Site-DMO/0/2026-08-20T19-41-02Z_0.wav duration=7.0s reason=timeout
```

`speech_frames=0` with `bfi_count=242` is a call that decoded nothing;
`reason=timeout` follows — nothing refreshed its liveness, so hangtime tore
it down. The web row is the *consequence*.

## Five stories, one table

| Panel symptom | Log evidence | Root cause | Fix |
|---|---|---|---|
| History "stops at 00:15"; the feed shows 03:15 | `recorder: call started` keeps coming; the `wav=` UTC stamps match the *feed* | six panels sliced the RFC3339 string (`.replace("T"," ")`) — always GMT | `formatLocalDateTime` in `lib/formatTime.ts`; History refetches on `call.end` |
| "No calls in the daemon's call log for this filter", every query | `recorder: call ended` rows exist; `curl /api/v1/calls/history` returns `{"calls":[…]}` | the client read `r.rows` with `?? []` — wrong key and empty result look identical | `r.calls ?? []` in `client.ts`; `History.test.tsx` pins the empty state |
| A tab left open across a restart reconnects 2–4×/s forever | a daemon line per attempt; upgrade, then a 1011 close | a handshake that always succeeds reset the backoff in `onopen`; `MAX_BACKOFF` was dead code | `rejectUnknownDevice` → **404** before upgrade; `reconnectingSocket.ts` resets backoff only after a frame |
| `symbol: poor` next to `decode: clean` on a TETRA rig | `tetra: decode status … bsch_ok=147 bsch_fail=0` — the decode was fine | `demodModeToProto("")` fell back to `p25-c4fm`: a C4FM receiver on a π/4-DQPSK carrier, ~9 dB "MER" ⇒ `< 10 dB` poor | `symbolProtoFor` → `symbol_proto`; `autoProtoFor` prefers it |
| A ~0.1 s silent WAV row after every DMO PTT | `voice follow ended … speech_frames=0 bfi_count=242`, `call ended … reason=timeout` | the voice chain fell back to colour 0 before adopting the pipeline's; the recorder only drops `dataBytes==0` or zero-voice calls | the chain now adopts the pipeline's colour; the tiny-row side is **staged, not shipped**; DMO is on-air-gated ([#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003)) |

Three more from the same batch were really recorder or shell stories.
**"Duration 30 s, recording 4 s"** was segment rolls: the recorder publishes
one `CallComplete` *per over*, stamped with the over's own `StartedAt`, and
the call log keyed on it — only the first over matched. `CallComplete.CallStartedAt`/`Segment` plus the `call_recordings`
table fix the keying, and `/calls/{id}/audio` concatenates the segments
(`concatRecordingSegments`) — the seam
[Recording Part 9]({{ '/blog/deep-dives/recording-streaming-09-call-complete-seam/' | relative_url }})
describes. **Cyrillic folders as `_______`**: `sanitize` allowed
`[A-Za-z0-9._-]`; it now keeps Unicode letters, marks and digits. And the
**AudioPlayer docked over the pager** — `DataTable` now renders one top and
bottom.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="The path from a daemon log line to a browser panel drawn as four boxes joined by arrows — recorder and engine log lines, the REST and WebSocket API, the client fetch layer, and the rendered panel — with each of the five September field reports pinned to the seam that broke it: symbol_proto at the API's device list, the 404 before upgrade at the transport, the r.calls envelope key at the fetch layer, the UTC string slice at the render, and speech_frames zero sitting upstream in the log itself.">
  <rect x="8" y="70" width="140" height="60" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="78" y="90" text-anchor="middle" fill="var(--accent)" font-size="10">daemon log</text>
  <text x="78" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="8">recorder: call started</text>
  <text x="78" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="8">call ended reason=…</text>
  <line x1="148" y1="100" x2="176" y2="100" stroke="currentColor"/><polygon points="174,96 182,100 174,104" fill="currentColor"/>
  <rect x="182" y="70" width="140" height="60" rx="6" fill="none" stroke="currentColor"/>
  <text x="252" y="90" text-anchor="middle" fill="currentColor" font-size="10">REST · WebSocket</text>
  <text x="252" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="8">{"calls":[…]} UTC RFC3339</text>
  <text x="252" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="8">devices: symbol_proto</text>
  <line x1="322" y1="100" x2="350" y2="100" stroke="currentColor"/><polygon points="348,96 356,100 348,104" fill="currentColor"/>
  <rect x="356" y="70" width="140" height="60" rx="6" fill="none" stroke="currentColor"/>
  <text x="426" y="90" text-anchor="middle" fill="currentColor" font-size="10">client fetch layer</text>
  <text x="426" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="8">client.ts · reconnectingSocket.ts</text>
  <text x="426" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="8">r.calls ?? []</text>
  <line x1="496" y1="100" x2="524" y2="100" stroke="currentColor"/><polygon points="522,96 530,100 522,104" fill="currentColor"/>
  <rect x="530" y="70" width="142" height="60" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="601" y="90" text-anchor="middle" fill="var(--accent)" font-size="10">rendered panel</text>
  <text x="601" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="8">formatLocalDateTime</text>
  <text x="601" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="8">qualityVerdict pill</text>
  <line x1="78" y1="130" x2="78" y2="160" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="78" y="174" text-anchor="middle" fill="currentColor" font-size="8">speech_frames=0 →</text>
  <text x="78" y="185" text-anchor="middle" fill="currentColor" font-size="8">0.1 s silent row</text>
  <line x1="252" y1="130" x2="252" y2="160" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="252" y="174" text-anchor="middle" fill="currentColor" font-size="8">symbol_proto missing →</text>
  <text x="252" y="185" text-anchor="middle" fill="currentColor" font-size="8">P25 receiver on TETRA</text>
  <line x1="339" y1="100" x2="339" y2="30" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="339" y="20" text-anchor="middle" fill="currentColor" font-size="8">101 then 1011 → 404 before upgrade</text>
  <line x1="426" y1="130" x2="426" y2="160" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="426" y="174" text-anchor="middle" fill="currentColor" font-size="8">r.rows ?? [] →</text>
  <text x="426" y="185" text-anchor="middle" fill="currentColor" font-size="8">empty History forever</text>
  <line x1="601" y1="130" x2="601" y2="160" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="601" y="174" text-anchor="middle" fill="currentColor" font-size="8">.replace("T"," ") →</text>
  <text x="601" y="185" text-anchor="middle" fill="currentColor" font-size="8">History 3 h behind</text>
  <text x="340" y="228" text-anchor="middle" fill="var(--fg-muted)" font-size="9">every web symptom is a seam between log and screen — find the seam; the log already names the cause</text>
</svg>
<figcaption>Log → API → fetch → render: each September report broke one seam, and the daemon line for the same second was the tie-breaker.</figcaption>
</figure>

## When the panel and the log disagree

Four checks, in order, settle almost any web report without a debugger.

**1. Compare clocks.** The API returns UTC RFC3339. A panel behind by
*exactly* your UTC offset is a render bug, not staleness;
`formatLocalDateTime`/`formatClock` are the only way to render a timestamp.
Grep for `replace("T", " ")` before shipping a panel.

**2. `curl` the endpoint the panel reads.** `/api/v1/calls/history` with the
same filter shows the envelope key and the rows. Rows there, panel empty ⇒
the client (History had no test; it has one now). In reverse,
an empty API answer with `recorder: call ended` lines in the log points at
the call log or its retention window
([Recording Part 10]({{ '/blog/deep-dives/recording-streaming-10-call-log-sqlite/' | relative_url }})).

**3. Read the decode counters before trusting a badge.** The `decode:` and
`symbol:` chips are deliberately different axes — frame-error rate versus raw
constellation — but they must look at the same receiver. On the TETRA rig
they did not: the Histogram panel with Mode forced to TETRA showed
`SNR (MER) —` and `balance ±5.7%` (*clean*) while the auto-selected panels
sat at ~9 dB on a C4FM receiver. Two receivers on one carrier is the tell; the
`tetra: decode status` line from
[Part 3]({{ '/blog/tutorials/field-notebook-03-tetra-decode-status/' | relative_url }})
is the authority; a `poor` pill against `bsch_fail=0` is the pill's problem.
Every `/diag/symbols` subscriber also builds its own full-rate
`Downconverter`; nothing is pooled.

**4. Count the reconnects.** A stale tab reconnecting several times a second
is a line per attempt in the log, and the mechanism was a handshake that
could not fail: the handlers resolved the serial *after* `upgrader.Upgrade`,
so an unknown device got a 101 then a 1011 close, and every client reset its
backoff in `onopen`. Three fixes were all needed — resolve the device first
and return **404**, reset the backoff only once a frame arrives (or the
socket held `OPEN_GRACE_MS` = 2 s), and reconcile the selected serial
against a refreshed device list. `reconnectingSocket.ts` also fixes the
`onerror`+`onclose` double-bind that scheduled two timers per failure — the
storm's sibling from
[Issue Tracker Part 19]({{ '/blog/solution-postmortem/from-the-issue-tracker-19-one-render-loop/' | relative_url }}).
`GONE_AFTER_FAILURES` (6) frameless attempts ⇒ `gone`.

A fifth check is for whoever ships a fix. **A green `ci.yml` did not mean
the console built**: `npm test` (vitest) never typechecks, so three `TS6133`
errors in `reconnectingSocket.test.ts` landed on main and broke
`npm run build` (`tsc --noEmit && vite build`) for an operator; the one job
that caught them, "Windows installer (PR)", was not a required check.
`ci.yml` now runs `web-typecheck` per SPA, and any web diff runs
`cd web && npm run typecheck` before pushing —
[Cockpit Part 14]({{ '/blog/deep-dives/operator-cockpit-14-testing-uis/' | relative_url }})
covers the tests; this was the gap they left.

| Symptom | Likely cause | Fix / read |
|---|---|---|
| Tables behind by exactly your UTC offset | raw RFC3339 string rendered | `formatLocalDateTime`; the `wav=` stamp is the UTC truth |
| Empty table, `recorder:` lines flowing | envelope key mismatch | `curl` the endpoint; `client.ts` |
| Reconnect every ~500 ms | stale serial; old handshake-then-close | `rejectUnknownDevice`; `reconnectingSocket.ts` |
| `symbol: poor` on a decoding rig | wrong receiver behind the scope | `symbol_proto`; trust the decode counters ([Part 3]({{ '/blog/tutorials/field-notebook-03-tetra-decode-status/' | relative_url }})) |
| Recording far shorter than its duration | segment rolls keyed wrong | `call_recordings` ([Recording 9]({{ '/blog/deep-dives/recording-streaming-09-call-complete-seam/' | relative_url }})) |
| Tiny silent DMO rows | `speech_frames=0`, `reason=timeout` | a decode problem — [Part 11]({{ '/blog/tutorials/field-notebook-11-voice-chain-lines/' | relative_url }}), [Cookbook 5]({{ '/blog/tutorials/operator-cookbook-05-tetra-dmo/' | relative_url }}) |

### How this line shapes operator practice

- **Report the daemon line, not the screenshot.** A History row with its
  `recorder: call started` / `call ended reason=` pair is diagnosable; a
  screenshot alone is not.
- **Trust the decode counters over the badge.** `bsch_fail`, `fec_pass`,
  `speech_frames` come from the receiver doing the work; a quality pill from
  a *second* receiver may be the wrong one's.
- **A `?? []` is a place to look, not a safety net.** Where a client
  defaults a missing key to empty, only the log can prove data existed.
- **Typecheck what CI doesn't.** `npm run typecheck` separates a green PR
  from an operator who cannot build.

## Where this goes next

Every story here ended the same way: a log line and the API answer for the
same second, pasted beside the symptom.
[Part 13]({{ '/blog/tutorials/field-notebook-13-from-log-to-issue/' | relative_url }})
makes that the whole method — what to paste, how to align a capture to the
log, which capture answers which question, and the wording that
lets an issue close as *verified* rather than merely fixed.

## FAQ

**Why is the GopherTrunk History panel hours behind the activity feed?**
Because it rendered UTC. The daemon returns RFC3339 UTC strings, and several
panels sliced them as text — always GMT — while the feed formatted locally.
Tables lagging by exactly your UTC offset is that bug;
`web/src/lib/formatTime.ts` now renders every timestamp in local time.

**How do I tell a web bug from a decode bug?**
Find the daemon lines for the same second — `recorder: call started`,
`call ended reason=…`, the protocol's decode-status line — and `curl` the
endpoint the panel reads. Log and API agree, panel disagrees ⇒ a render or
client seam. Log shows `speech_frames=0` ⇒ the panel is honest.

**What does symbol: poor next to decode: clean mean?**
Two receivers. The `decode:` chip reads the real decoder's counters; the
`symbol:` chip grades a diagnostic receiver, which on TETRA and DMR rigs used
to be a P25 C4FM one. The daemon now publishes a per-device `symbol_proto` so
the scopes open the right receiver.

**Why did a browser tab hammer the daemon after I swapped SDRs?**
The diag WebSocket handlers completed the handshake before checking the
device, so a stale serial got an open and an immediate close, and the
client reset its backoff on every open. The daemon now answers 404 before
upgrading, and `reconnectingSocket.ts` resets backoff only after a real frame.

**Why does History show a 0.1-second silent recording after a DMO PTT?**
The voice chain decoded no speech (`speech_frames=0`) and hangtime ended the
call (`reason=timeout`); the recorder only discards zero-byte or
zero-voice calls, and ACELP is neither, so a tiny row lands. The cause is
upstream — colour adoption in the voice chain — and DMO remains on-air-gated
under #1003.

## Series navigation

**Part 12 of 14** · ←
[Part 11: Voice-Chain Lines — Frames, Colours & Teardown Reasons]({{ '/blog/tutorials/field-notebook-11-voice-chain-lines/' | relative_url }})
· Next →
[Part 13: From Log to Issue — What to Paste, What to Capture]({{ '/blog/tutorials/field-notebook-13-from-log-to-issue/' | relative_url }})
