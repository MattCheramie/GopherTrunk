---
layout: page
title: Radio IDs panel
description: Per-radio (subscriber-unit) entity browser merging the operator alias catalogue with the live affiliation tracker
nav_group: Operate
---

# Radio IDs panel

GopherTrunk's **Radio IDs** panel (web `/rids`) gives every
subscriber unit (the "RID" or `source_id` in grant / call payloads)
the same first-class treatment talkgroups have always had: a
filterable list, a detail view, per-RID call history, and operator-
configurable aliases.

It exists because recurring radios are operationally interesting in
their own right. The dispatch officer who's on the air all morning,
the patrol unit that runs the same routes every day, the test radio
that helps you spot a misconfigured CC — naming them once and
seeing them as named entities in the live feed makes that workflow
far easier than scrolling raw `source_id` numbers.

## What you get

- A merged view of two RID sources, keyed by radio ID:
  - **Configured** rows from a per-system `rid_alias_file` (CSV or
    JSON) — operator-assigned `alias`, `description`, `tag`,
    `group`, `owner`, `priority`, `lockout`, `watch`, `icon`.
  - **Live** rows from the affiliation tracker — `last_seen`,
    `first_seen`, `last_talkgroup`, observed over-the-air
    `talker_alias`, and a `call_count` for "how recurring is this
    radio."
- Configured + live overlap is merged on `id`; either source can
  contribute a row by itself.
- Detail modal with the last 50 calls observed for the RID
  (queried from the persisted call log by `source_id`).
- Write-mode edits to the in-memory catalogue (alias / watch /
  lockout / priority / tag / group / owner / icon).

## Loading aliases

Each trunked system can point at its own RID catalogue with the
`rid_alias_file` key (mirrors `talkgroup_file`):

```yaml
trunking:
  systems:
    - name: "Example-P25"
      protocol: p25
      control_channels: [851_000_000]
      talkgroup_file: "/etc/gophertrunk/talkgroups-p25.csv"
      rid_alias_file:  "/etc/gophertrunk/rids-p25.csv"
```

The file is dispatched by extension: `*.json` goes through the
JSON loader, anything else through the CSV loader.

### CSV format

Required column (case-insensitive): `Decimal` / `DEC` / `ID`.
Optional columns: `Alias` (or `Alpha Tag` / `AlphaTag`),
`Description`, `Tag`, `Group`, `Owner`, `Priority`, `Lockout`,
`Watch`, `Icon`.

```csv
Decimal,Alias,Description,Tag,Group,Owner,Priority,Watch
207545,CPL-SMITH,Patrol corporal,Patrol,Bossier PD,Cpl. Smith,2,
207546,LOCKED,Decommissioned radio,,,L,,no
207547,ENG-12,Fire engine 12,Fire,Bossier Fire,Engine 12,1,
```

`Lockout` accepts `Y` / `yes` / `true` / `1`; the legacy `Priority:L`
sentinel from talkgroup CSVs also sets `Lockout`. `Watch` defaults
to true; explicit `no` / `false` / `0` / `n` opts a row out of the
watch list.

### JSON format

```json
[
  {"id": 207545, "alias": "CPL-SMITH", "owner": "Cpl. Smith", "priority": 2},
  {"id": 207546, "alias": "LOCKED", "lockout": true, "watch": false},
  {"id": 207547, "alias": "ENG-12", "tag": "Fire", "group": "Bossier Fire"}
]
```

The daemon preflights the file the same way it preflights
`talkgroup_file` — non-fatal warnings for missing / empty paths so
the daemon still starts and the affiliation tracker still surfaces
live RIDs.

## REST surface

| Method | Path | Auth | What it does |
| --- | --- | --- | --- |
| `GET`   | `/api/v1/rids`                  | open           | Merged list (configured ∪ live). |
| `GET`   | `/api/v1/rids/{id}`             | open           | Single merged row. |
| `GET`   | `/api/v1/rids/{id}/history`     | open           | `call_log` filtered by `source_id`. Same query params as `/api/v1/calls/history` (`limit`, `only_ended`, `system`). |
| `PATCH` | `/api/v1/rids/{id}`             | mutation-gated | Edit alias / description / tag / group / owner / priority / lockout / watch / icon. Only works on rows backed by the static catalogue — live-only rows return 404 with a hint to add the RID to `rid_alias_file` first. |

PATCH mutations live in memory only — the on-disk `rid_alias_file`
is not rewritten, matching the talkgroup behaviour. Restarting the
daemon reloads from disk.

## gRPC surface

`RIDService` in `proto/rid.proto` mirrors the HTTP surface:

- `ListRIDs(ListRIDsRequest) → ListRIDsResponse`
- `GetRID(GetRIDRequest) → GetRIDResponse`
- `ListRIDHistory(ListRIDHistoryRequest) → ListRIDHistoryResponse`

Read-only for this slice; mutations go through the HTTP PATCH.

## Talker aliases

The decoded over-the-air talker alias (the radio's display name)
shows up in two places:

- On the RID row as `talker_alias` once the daemon reassembles it,
  with `talker_alias_at` as the observation timestamp.
- As a `talker.alias` event on the bus / CC Activity feed at
  decode time.

Three paths feed it:

- **Motorola vendor TSBK** — control-channel `OpVendorTalkerAlias`
  0x15, reassembled by `phase1.TalkerAliasAssembler` (Phase 1)
  and the Phase 2 vendor MAC opcode.
- **Motorola voice-channel LCs** — P25 Phase 1 LDU1 Link Control
  opcodes 0x15 (header: talkgroup + variable block_count +
  sequence number) and 0x17 (data blocks, 44 bits each),
  reassembled per call by `phase1.MotorolaTalkerAliasBuf`. The
  reassembled message is run through a reverse-engineered
  Motorola substitution-table cipher
  (`phase1.decodeAliasBytes`) to recover the printable alias
  characters. (Earlier work assumed the TIA-102.AABF "standard"
  HEADER+BLOCK1+BLOCK2 layout; real Motorola systems don't emit
  that form, so the standard-form decoder was replaced with this
  vendor variant in a follow-up to PR #389.)
- **P25 Phase 2 FACCH-S signalling follower** — on Phase 2 systems
  the talker alias rides the traffic channel's FACCH-S MAC
  signalling during hangtime (Motorola header opcode 0x91 + data
  0x95), not the control channel. Decoding it used to require a
  voice tuner to follow the call, so on busy multi-site systems —
  where most grants never get a voice tap, and encrypted calls are
  torn down before hangtime — the alias was almost never decoded
  (issue #376). The signalling follower (`internal/sigfollow`)
  fixes this: it allocates lightweight signalling-only DDC taps on
  the wideband IQ stream and harvests the alias off the traffic
  channel independent of the voice pool, the way SDRTrunk does.
  Enable it with `signalling_taps: N` on a `role: wideband` device
  (see `config.example.yaml`); the decode runs the same shared MAC
  dispatch the voice chain uses, so the voice and follower paths
  never diverge.

## See also

- [CC Activity panel](cc-activity.md) — the live chatter feed
  whose RID chips link into this panel.
- [Web console](web.md) — overall web UI orientation.
- [Hardening](hardening.md) — the bearer-token auth gate that
  protects the PATCH endpoint.
