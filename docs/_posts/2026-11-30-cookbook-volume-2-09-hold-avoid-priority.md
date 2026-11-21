---
title: "The Operator's Cookbook Vol. 2, Part 9: Hold, Avoid and Priority — Driving It Like a Scanner"
description: "A copy-paste GopherTrunk recipe for the scanner front panel: talkgroup hold and timed avoid over the REST API, web console and TUI, scan-all versus scan-list, talkgroup priority and preemption, and a conventional scan list with priority channels and lockouts that persist by frequency. It also says which of these a restart clears."
category: tutorials
keywords: sdr scanner talkgroup hold, scanner avoid talkgroup temporary lockout, trunking scanner priority preemption, scan list mode sdr, priority scan conventional channels, scanner lockout persist, gophertrunk rest api curl, uniden hold avoid sdr, gophertrunk hold avoid priority
tags: [operator-cookbook, cookbook-volume-2, scanning, talkgroups, priority, rest-api]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook, Volume 2"
series_part: 9
---

*Part 9 of **The Operator's Cookbook, Volume 2**, a second 14-part run of
complete, copy-paste GopherTrunk builds — one working rig per part.
[Part 8]({{ '/blog/tutorials/cookbook-volume-2-08-your-own-keys/' | relative_url }})
gave the rig your own keys. This part gives it back the front panel of a
hardware scanner: **Hold** on the talkgroup working an incident, **Avoid**
on the one that will not stop talking, a priority channel that gets
checked more often than the rest, and a lockout that is still there
tomorrow. All of it is a curl command, a key in the TUI, or a button in
the web console.*

> **TL;DR:** `POST /api/v1/scanner/hold` with a `talkgroup` pins the
> trunking engine to one talkgroup until `DELETE /api/v1/scanner/hold`;
> `POST /api/v1/talkgroups/{id}/avoid` locks one out for a duration
> (default 30 minutes) and then resumes by itself. Both sit in the grant
> path after Lockout and before the scan-list gate, emergency grants pass
> all three, and a held talkgroup also bypasses list mode. Both are
> **session-only**: a restart clears them, like a scanner power cycle. On
> the conventional side, `scanner.priority_interleave: N` visits a
> priority channel after every N ordinary ones, and scan-list lockouts
> **persist by frequency** when `storage.path` is set. Talkgroup priority
> (1 highest to 10) decides who keeps a tuner only when calls outnumber
> them. Status: synthetic-verified by unit tests; no field report either
> way.

**Key takeaways**

- **Hold and avoid gate grants, not audio.** They decide which new calls
  get a voice tuner. A call already in progress on another talkgroup runs
  to its end.
- **Emergency always gets through.** Lockout, hold, avoid and the scan
  list all step aside for a grant flagged emergency.
- **Two things survive a restart; the rest do not.** Config and the
  talkgroup CSV are durable, and so are scan-list lockouts. Hold, avoid
  and API edits to priority, lockout and scan are memory only.
- **The scan list is a separate machine.** It runs on its own SDR and is
  not gated by talkgroup hold or avoid; it has priority channels,
  lockouts and its own pause.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Hold one talkgroup | drops every other grant until released | `POST` and `DELETE /api/v1/scanner/hold` |
| Timed avoid | locks a talkgroup out, then resumes by itself | `POST` and `DELETE /api/v1/talkgroups/{id}/avoid` |
| Scan-all or scan-list | follow everything, or only talkgroups with Scan set | `scanner.scan_mode`, `PATCH /api/v1/scanner` |
| Talkgroup priority | who keeps a tuner when calls outnumber tuners | the CSV's `Priority` column, `PATCH /api/v1/talkgroups/{id}` |
| Priority channels on the scan list | visits them every N ordinary channels | `scanner.priority_interleave` plus a channel's `priority` |
| Scan-list lockout | skips a channel; stored by frequency | `POST /api/v1/scanner/conventional/{index}/lockout` |
| What is in force now | hold, avoids, channel indexes and levels | `GET /api/v1/scanner` |

## In this post

- **What you're building** — the gates a grant passes, in order.
- **The config** — one trunked system, one scan list, one CSV.
- **First run — what healthy looks like** — a worked evening in curl, with
  the replies.
- **When it doesn't work** — symptom → cause → fix.
- **Variations** — the same controls in the web console and the TUI.

## What you're building

A rig you can steer while it runs. Volume 1's
[multi-system pool]({{ '/blog/tutorials/operator-cookbook-07-multi-system-pool/' | relative_url }})
set policy in files — `Priority`, `Scan` and `Lockout` columns in a CSV,
read at start-up. This part adds the controls you press mid-shift, and it
is worth knowing exactly where they act.

Every trunked grant walks the same short ladder inside the engine's
`HandleGrant`. First **Lockout**: a talkgroup flagged locked out is
dropped. Then **hold and avoid** (`internal/trunking/holdavoid.go`): with a
hold active, any grant for a different
[talkgroup]({{ '/reference/talkgroup/' | relative_url }}) is dropped; with
an unexpired avoid on its talkgroup, the grant is dropped. Then the
**scan-list gate**: in `list` mode a talkgroup whose Scan flag is off is
dropped — unless it is the held one, which skips this gate. Whatever is
left goes to the voice pool, where a free tuner takes it or, if none is
free, a strictly higher
[priority]({{ '/blog/deep-dives/trunking-engine-05-priority-preemption/' | relative_url }})
preempts the lowest-priority call. A grant flagged
[emergency]({{ '/reference/emergency-call/' | relative_url }}) passes all
three gates.

Two consequences follow from where the gates sit. They act on *grants*, so
holding talkgroup A does not hang up a call already running on talkgroup
B; B finishes, and then nothing but A is followed. And they act on the
*trunking engine*: the conventional scan list is a different loop on its
own voice SDR, opening calls from squelch rather than grants, so a
talkgroup hold never silences it. The scan list has its own controls —
[priority scan]({{ '/reference/priority-scan/' | relative_url }}),
lockouts, and a pause.

<figure class="lab-figure">
<svg viewBox="0 0 680 240" width="680" height="240" role="img" aria-label="Top: a trunked grant passes three gates in order, Lockout, then hold and avoid, then the scan-list gate, before reaching the voice pool; each gate can drop it, an emergency grant bypasses all three, and the held talkgroup skips the scan-list gate. Bottom: the conventional scan list on its own SDR with priority interleave two visits the priority channel after every two ordinary channels">
  <text x="316" y="20" text-anchor="middle" fill="var(--accent)" font-size="9">emergency grant: passes all three gates</text>
  <polyline points="43,56 43,28 589,28 589,56" fill="none" stroke="var(--accent)" stroke-dasharray="4 3"/>
  <rect x="8" y="56" width="70" height="44" rx="4" fill="none" stroke="currentColor"/>
  <text x="43" y="82" text-anchor="middle" fill="currentColor" font-size="10">grant</text>
  <line x1="78" y1="78" x2="106" y2="78" stroke="currentColor"/>
  <rect x="106" y="56" width="96" height="44" rx="4" fill="none" stroke="currentColor"/>
  <text x="154" y="82" text-anchor="middle" fill="currentColor" font-size="10">Lockout</text>
  <line x1="202" y1="78" x2="230" y2="78" stroke="currentColor"/>
  <rect x="230" y="56" width="110" height="44" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="285" y="74" text-anchor="middle" fill="var(--accent)" font-size="10">hold / avoid</text>
  <text x="285" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="9">session only</text>
  <line x1="340" y1="78" x2="368" y2="78" stroke="currentColor"/>
  <rect x="368" y="56" width="110" height="44" rx="4" fill="none" stroke="currentColor"/>
  <text x="423" y="74" text-anchor="middle" fill="currentColor" font-size="10">scan-list gate</text>
  <text x="423" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="9">held TG skips this</text>
  <line x1="478" y1="78" x2="506" y2="78" stroke="currentColor"/>
  <rect x="506" y="56" width="166" height="44" rx="4" fill="none" stroke="currentColor"/>
  <text x="589" y="74" text-anchor="middle" fill="currentColor" font-size="10">voice pool</text>
  <text x="589" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="9">free tuner, or strict preempt</text>
  <line x1="154" y1="100" x2="154" y2="118" stroke="var(--fg-muted)"/>
  <text x="154" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="9">TG locked out</text>
  <line x1="285" y1="100" x2="285" y2="118" stroke="var(--fg-muted)"/>
  <text x="285" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="9">another TG held, or avoided</text>
  <line x1="423" y1="100" x2="423" y2="118" stroke="var(--fg-muted)"/>
  <text x="423" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Scan off (list mode)</text>
  <text x="8" y="164" fill="currentColor" font-size="10">scan list, on its own voice SDR — priority_interleave: 2</text>
  <rect x="8" y="174" width="62" height="28" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="39" y="192" text-anchor="middle" fill="var(--accent)" font-size="9">PRI</text>
  <rect x="78" y="174" width="62" height="28" rx="4" fill="none" stroke="currentColor"/>
  <text x="109" y="192" text-anchor="middle" fill="currentColor" font-size="9">ch 2</text>
  <rect x="148" y="174" width="62" height="28" rx="4" fill="none" stroke="currentColor"/>
  <text x="179" y="192" text-anchor="middle" fill="currentColor" font-size="9">ch 3</text>
  <rect x="218" y="174" width="62" height="28" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="249" y="192" text-anchor="middle" fill="var(--accent)" font-size="9">PRI</text>
  <rect x="288" y="174" width="62" height="28" rx="4" fill="none" stroke="currentColor"/>
  <text x="319" y="192" text-anchor="middle" fill="currentColor" font-size="9">ch 4</text>
  <rect x="358" y="174" width="62" height="28" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="389" y="192" text-anchor="middle" fill="var(--accent)" font-size="9">PRI</text>
  <rect x="428" y="174" width="62" height="28" rx="4" fill="none" stroke="currentColor"/>
  <text x="459" y="192" text-anchor="middle" fill="currentColor" font-size="9">ch 2</text>
  <rect x="498" y="174" width="62" height="28" rx="4" fill="none" stroke="currentColor"/>
  <text x="529" y="192" text-anchor="middle" fill="currentColor" font-size="9">ch 3</text>
  <rect x="568" y="174" width="62" height="28" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="599" y="192" text-anchor="middle" fill="var(--accent)" font-size="9">PRI</text>
  <text x="8" y="224" fill="var(--fg-muted)" font-size="9">not gated by hold or avoid; a lockout removes a channel from this rotation and is stored by frequency</text>
</svg>
<figcaption>Hold and avoid are one gate among three on the trunked grant path; the scan list is a separate rotation with its own priority sampling and its own lockouts.</figcaption>
</figure>

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| A trunked rig | $0 if you built one | any Volume 1 system; this recipe uses a wideband P25 dongle with two voice taps |
| A second RTL-SDR | ~$35 | a voice-role dongle for the scan list; skip it and the conventional half of this part does not apply |
| `curl` | $0 | or the web console, or the TUI — all three call the same routes |

## The config

```yaml
log:
  level: info

storage:
  path: "../data/calls.db"          # scan-list lockouts are stored here

recordings:
  dir: "../recordings"

sdr:
  sample_rate: 2_400_000
  devices:
    - serial: "00000101"            # control channel + voice taps
      role: wideband
      gain: "auto"
      center_freq_hz: 858_000_000
      voice_taps: 2                 # two calls at once, then priority decides
      channels:
        - frequency_hz: 857_262_500
          system: "County-P25"
    - serial: "00000102"
      role: voice                   # the scan list runs on this one
      gain: "auto"

trunking:
  systems:
    - name: "County-P25"
      protocol: p25
      control_channels: [857_262_500, 858_487_500]
      talkgroup_file: "../config/talkgroups-p25.csv"

scanner:
  scan_mode: list                   # all | list
  priority_interleave: 2            # a priority channel every 2 ordinary ones
  conventional:
    - label: "Fire Dispatch (analog)"
      frequency_hz: 154_190_000
      mode: fm
      squelch_dbfs: -48
      priority: 1                   # above 0 = a priority channel
      talkgroup_id: 2147500001
    - label: "Public Works"
      frequency_hz: 155_715_000
      mode: nfm
      squelch_dbfs: -50
      talkgroup_id: 2147500002
    - label: "Rail Yard"
      frequency_hz: 160_800_000
      mode: nfm
      squelch_dbfs: -50
      talkgroup_id: 2147500003
    - label: "Marine VHF 16"
      frequency_hz: 156_800_000
      mode: nfm
      squelch_dbfs: -50
      talkgroup_id: 2147500004
```

And `talkgroups-p25.csv`, the durable half of the policy:

```text
Decimal,Alpha Tag,Priority,Scan,Lockout
9001,Fire Dispatch,1,Y,
9014,Fire Tac 2,2,Y,
9101,Schools Ops,,N,
9230,Water Dept,8,Y,
```

**`scan_mode`** is `all` (follow every grant that is not locked out) or
`list` (only talkgroups whose Scan flag is on; the CSV's `Scan` column,
default on). [Trunking Engine Part 6]({{ '/blog/deep-dives/trunking-engine-06-talkgroups-scan-modes/' | relative_url }})
has the semantics.

**`Priority`** in the CSV is 1 (highest) to 10; empty counts as lowest. It
matters only when both voice taps are busy: a new grant preempts the
lowest-priority call if it is *strictly* higher, and equal priority never
preempts. Emergency outranks everything.

**`priority_interleave`** and the channel's **`priority`** are a different
mechanism with a similar name. On the scan list any `priority` above zero
marks a priority channel; the number does not rank them, they take turns.
With `priority_interleave: 2` the rotation visits a priority channel
after every two ordinary ones. It is sampling in the rotation, not an
interrupt: a dwell on an ordinary channel is not cut short to go and look.
`0` (the default) is plain round robin.

**`storage.path`** is what makes a scan-list lockout survive a restart.
Lockouts are stored by frequency, never by list position, so editing the
list does not move them onto the wrong channel.

What a restart keeps, in one place:

| Control | After a restart |
|---|---|
| `scanner.scan_mode`, the CSV's `Priority`, `Scan` and `Lockout` columns | kept: they are files |
| Scan-list lockouts | kept, by frequency, when a call database is configured |
| Talkgroup hold and timed avoid | cleared, on purpose |
| API changes to scan mode, or to a talkgroup's priority, lockout or scan flag | cleared; only a talkgroup's name fields are stored |

## First run — what healthy looks like

```sh
gophertrunk run -config config.yaml
```

The API listens on `127.0.0.1:8080` by default, and on a loopback bind the
write routes need no token. A worked evening follows; `API` is set once.

```sh
API=http://127.0.0.1:8080
```

**19:40 — a structure fire, and you want Fire Tac 2 and nothing else.**

```sh
curl -s -X POST "$API/api/v1/scanner/hold" \
  -H 'Content-Type: application/json' \
  -d '{"system":"County-P25","talkgroup":9014}'
```

```text
{"ok":true,"hold":{"system":"County-P25","talkgroup":9014,"since":"..."}}
```

`system` is optional; without it the hold matches that talkgroup number on
any system, and with it grants from every other system are dropped too.
From here the engine drops each grant that is not talkgroup 9014 — at
debug level you would see `grant dropped by hold` — unless it is an
emergency. A patch that includes 9014 is still followed.

**20:55 — the incident winds down.**

```sh
curl -s -X DELETE "$API/api/v1/scanner/hold"
```

```text
{"ok":true,"released":true}
```

**21:00 — the Water Dept is reading meters aloud. An hour off.**

```sh
curl -s -X POST "$API/api/v1/talkgroups/9230/avoid" \
  -H 'Content-Type: application/json' -d '{"duration":"1h"}'
```

```text
{"ok":true,"avoid":{"talkgroup":9230,"until":"..."}}
```

The body takes `duration` (a Go duration such as `30m` or `2h`) or
`minutes`; with neither it is 30 minutes. At 22:00 the talkgroup is
followed again with no further action. To end it early,
`curl -s -X DELETE "$API/api/v1/talkgroups/9230/avoid"`.

**21:10 — a quiet night; open the list up, and promote Fire Dispatch.**

```sh
curl -s -X PATCH "$API/api/v1/scanner" \
  -H 'Content-Type: application/json' -d '{"scan_mode":"all"}'
curl -s -X PATCH "$API/api/v1/talkgroups/9001" \
  -H 'Content-Type: application/json' -d '{"priority":1}'
```

The first replies `{"ok":true,"scan_mode":"all","previous_mode":"list"}`.
If a third call arrives while both taps are busy and it outranks one of
them, the log shows the swap:

```text
INF call ended device=... grant=... reason=preempted
INF call started device=... grant=... priority=1
```

If it does not outrank anything, you get
`INF no voice device available for grant` and the call is not followed.

**21:30 — the Rail Yard channel has a stuck carrier.** Channel indexes
come from the status route, which also shows the hold and avoids in
force:

```sh
curl -s "$API/api/v1/scanner"
```

```text
{"scan_mode":"all", ...
 "conventional":{"enabled":true, ... "channels":[ ...
   {"index":2,"label":"Rail Yard","frequency_hz":160800000,"mode":"nfm","active":true, ...}, ...]},
 ... "avoids":[{"talkgroup":9230,"until":"..."}]}
```

```sh
curl -s -X POST "$API/api/v1/scanner/conventional/2/lockout"
```

```text
{"ok":true,"index":2,"locked_out":true}
INF synthetic call ended device=... reason=lockout
```

The dwell in progress on that channel ends at once (the second line is
the daemon's log, not the reply). After the next restart the lockout is
still there:

```text
INF conv: restored persisted lockouts count=1
```

Undo it with `.../conventional/2/unlockout`.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| Hold set, but another talkgroup is still playing | that call started before the hold; the gate acts on new grants | let it finish, or end it with `POST /api/v1/calls/{deviceSerial}/end` |
| Hold set, scan-list channels still open calls | the conventional scanner is not gated by talkgroup hold | pause it with `POST /api/v1/scanner/conventional/hold`, and `.../resume` later |
| Nothing at all is followed any more | a hold was left on, or names a system that is not the one granting | read `hold` in `GET /api/v1/scanner`, then release it |
| An avoid will not clear | it was set with a system name and cleared without one | repeat the system: `DELETE .../avoid?system=County-P25` |
| Hold, avoid or a priority change vanished | the daemon restarted; those are memory only | durable policy belongs in the CSV and in `scanner.scan_mode` |
| The route answers 401 or 403 | the API is bound off loopback, so write routes need the token | send `Authorization: Bearer <token>`; see [auth posture]({{ '/blog/deep-dives/running-it-for-real-02-auth-posture/' | relative_url }}) |
| The route answers 503 | no trunking engine or no conventional scanner is running | check the start-up log for the scanner's missing voice SDR WARN |
| A scan-list lockout did not survive a restart | no call database is configured | set `storage.path` |
| Priority channel is not checked during a long dwell elsewhere | the interleave samples between dwells, it does not interrupt one | shorten that channel's `hangtime_ms`, or lock it out for the night |

### How this recipe shapes operator practice

- **Put standing policy in files and tonight's policy in the API.** The
  CSV and `scan_mode` are what the rig wakes up with; hold and avoid are
  for the shift.
- **Use avoid before lockout.** An avoid expires by itself. A talkgroup
  lockout set through the API lasts until the restart, and one in the CSV
  until you edit it out.
- **Glance at the status route before you blame the radio.** A forgotten
  hold looks exactly like a dead system.

## Variations

- **The web console.** In the Talkgroups panel each row has `scan`,
  `lock`, `hold` and `avoid 30m` buttons; the Scanner panel shows a
  "Hold & avoid" card with `release` and `clear`. Write mode must be on.
- **The TUI.** In Talkgroups: `h` holds or releases the selected
  talkgroup, `a` avoids it for 30 minutes or clears the avoid, `l`
  toggles lockout, `S` the scan flag, `+` and `-` the priority. In
  Scanner: `m` cycles the scan mode, `L` locks a scan-list channel out or
  back in, `Enter` dwells on one. The
  [TUI doc]({{ '/tui.html' | relative_url }}) lists every key.
- **Go to a channel.** `POST /api/v1/scanner/conventional/{index}/dwell`
  sends the scan list to a channel on its next pick.
- **No talkgroup file yet.** `PATCH /api/v1/talkgroups/{id}` creates the
  talkgroup if it is not catalogued, so you can set `scan`, `lockout` or
  `priority` on something you only just heard.

Where these controls came from, and what is still missing next to a
hardware scanner, is in the
[competitive feature assessment]({{ '/competitive-feature-assessment.html' | relative_url }});
the internals of the scan-list side are
[Squelch Part 13]({{ '/blog/deep-dives/squelch-13-operator-controls/' | relative_url }}).

## Where this goes next

Everything so far has assumed a computer on a desk.
[Part 10]({{ '/blog/tutorials/cookbook-volume-2-10-phone-rig/' | relative_url }})
puts GopherTrunk in a pocket: an Android phone, Termux, an RTL-SDR on an
OTG cable — which build to download, why it is not the Linux one, and
what a phone can and cannot keep up with.

## FAQ

**How do I hold a talkgroup in GopherTrunk?**
Send `POST /api/v1/scanner/hold` with a JSON body naming the `talkgroup`
(and optionally the `system`), press `h` on it in the TUI's Talkgroups
panel, or click `hold` in the web console. Every other grant is dropped
until you release the hold with `DELETE /api/v1/scanner/hold`.

**What is the difference between avoid and lockout?**
An avoid is a lockout with an expiry: the talkgroup is skipped for the
duration you give (30 minutes by default) and then followed again by
itself. A lockout stays until you remove it. Neither blocks an emergency
grant.

**Do hold and avoid survive a restart?**
No, by design. Both are session state, cleared by a restart the way a
scanner's hold is cleared by a power cycle. Conventional scan-list
lockouts are the exception: they are stored by frequency when
`storage.path` is set.

**Why does my priority channel not interrupt another channel's call?**
`scanner.priority_interleave` changes the order of the rotation so a
priority channel is visited more often between dwells. It does not break
into a dwell in progress. Trunked talkgroup priority is different: it can
preempt a lower-priority call, but only when no tuner is free.

**Is hold and avoid verified on air?**
It is synthetic-verified: the grant gate and the priority interleave are
pinned by unit tests (`TestEngineHoldFollowsOnlyTheHeldTalkgroup`,
`TestEngineAvoidIsATimedLockout`, `TestConvScannerPriorityInterleave`).
The logic has no RF dependence, and the project's notes record no field
report for or against it.

## Series navigation

**Part 9 of 14** · ←
[Part 8: Your Own Keys — Decrypting What You're Authorised to Hear]({{ '/blog/tutorials/cookbook-volume-2-08-your-own-keys/' | relative_url }})
· Next →
[Part 10: The Phone Rig — GopherTrunk in Termux]({{ '/blog/tutorials/cookbook-volume-2-10-phone-rig/' | relative_url }})
