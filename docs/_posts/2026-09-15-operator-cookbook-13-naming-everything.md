---
title: "The Operator's Cookbook, Part 13: Naming Everything — Aliases, Labels & Exports"
description: Turn every talkgroup and radio ID on your GopherTrunk rig into a name — the real talkgroup_file and rid_alias_file CSV columns, naming things live from the web console with the labels layer, harvesting talker aliases off the air, and exporting the merged catalogue back to CSV.
category: tutorials
keywords: talkgroup csv format, radioreference talkgroup import, radio id alias file, scanner talkgroup names, talker alias decode, trunk recorder talkgroup file, gophertrunk talkgroup_file, sdr scanner alias csv, gophertrunk cookbook
tags: [operator-cookbook, talkgroups, aliases, csv, web-console, metadata]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook"
series_part: 13
---

*Part 13 of **The Operator's Cookbook**, a 14-part series of complete,
copy-paste GopherTrunk builds — one working rig per part, antenna to browser.
[Part 12]({{ '/blog/tutorials/operator-cookbook-12-diversity-mrc/' | relative_url }})
finished the hardware story with the two-antenna diversity build. This part
buys nothing and changes everything you look at: it turns `tg=9001` into
`FIRE-DISP` everywhere — log lines, recordings tree, History searches, the
live panels. The recipe is three layers deep: CSV files you own, a labels
layer you edit from the browser, and names the air itself hands you. The
running rule: **your most recent explicit act wins, and nothing you type gets
silently lost.***

> **TL;DR:** Per system, `talkgroup_file` and `rid_alias_file` load CSVs
> keyed on a required **`Decimal`/`DEC`** column (RIDs also accept `ID`),
> with optional case-insensitive columns — `Alpha Tag`, `Description`, `Tag`,
> `Group`, `Priority` (a literal `L` means lockout), `Lockout`, `Scan`,
> `Stream`, `Record`, `Mute`, `Icon` for talkgroups; `Alias`, `Owner`,
> `Watch` for radios. Healthy startup logs
> `daemon: talkgroups loaded system=… count=…`. Names applied live via the
> web (behind `PATCH /api/v1/talkgroups/{id}` and `/api/v1/rids/{id}`)
> persist in a SQLite **labels** table layered *over* the files — the label
> wins, with a WARN when they disagree — and
> `GET /api/v1/labels/export` folds the merged result back into a CSV that
> loads as your next `talkgroup_file`.

**Key takeaways**

- **One column is mandatory, everything else is optional.** A CSV with just
  `Decimal` and `Alpha Tag` headers works, and RadioReference-style exports
  drop straight in. A missing `Decimal`/`DEC` header is the one hard error.
- **The browser is a first-class editor, not a viewer.** Naming a radio that
  just showed up live *creates* its catalogue entry — the daemon stopped
  404-ing on unknown IDs precisely because the radios worth naming are the
  ones not in any file yet.
- **Labels layer over files; the label wins.** Files load first, persisted
  labels apply on top, and a disagreement logs a WARN instead of a silent
  overwrite — then the export endpoint lets you fold labels back into the
  file and retire them.
- **Some names come off the air for free.** Talker aliases, discovered
  talkgroups and the affiliation tracker populate the rosters before you've
  typed anything — [The Hunt Part 12]({{ '/blog/deep-dives/the-hunt-12-alias-harvesting/' | relative_url }})
  is that story.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Talkgroup names | per-system CSV/JSON catalogue | `trunking.systems[].talkgroup_file` |
| Radio-ID names | the per-RID equivalent | `trunking.systems[].rid_alias_file` |
| Live naming | create/update names from web or API | `PATCH /api/v1/talkgroups/{id}`, `PATCH /api/v1/rids/{id}` |
| Persistence | operator names survive restarts | labels table in `storage.path` (SQLite) |
| Getting names out | merged catalogue back as loadable CSV | `GET /api/v1/labels/export?kind=…&scope=…` |
| Names off the air | talker aliases on traffic channels | `signalling_taps` ([P25 talker alias]({{ '/reference/p25-talker-alias/' | relative_url }})) |
| Whole-system exports | trunk-recorder / RadioReference / SigMF bundles | [The Hunt Part 13]({{ '/blog/deep-dives/the-hunt-13-exporting-your-finds/' | relative_url }}) |

## In this post

- **What you're building** — a naming layer over every number the rig shows.
- **The files — real columns, verified** — both CSV formats from the actual loaders.
- **Naming live from the browser** — the labels layer and its precedence rule.
- **Names off the air** — talker aliases, discovered talkgroups, and one filing fix.
- **First run — what healthy looks like** — the load counts and the divergence WARN.
- **When it doesn't work** — symptom → cause → fix, then variations.

## What you're building

Nothing on the antenna side changes. What changes is every surface you read:
recordings file under `<system>/<talkgroup>/` with the `{alpha}` token
available in `filename_template`, the History panel becomes searchable by
name, the Talkgroups and Radio IDs panels become rosters instead of number
columns, and broadcast feeds carry proper labels. A
[talkgroup]({{ '/reference/talkgroup/' | relative_url }}) or
[radio ID]({{ '/reference/radio-id/' | relative_url }}) is just a number on
the air — everything human about it is this recipe.

Three sources feed one merged catalogue, in a strict order: **files** (loaded
at startup, per system), then **labels** (your live edits, persisted to
SQLite, applied over the files), with **on-air harvest** — talker aliases,
discovered talkgroups — filling gaps around both. The precedence is the whole
design: a label is your most recent explicit act, so it beats the file; the
air never overwrites either.

<figure class="lab-figure">
<svg viewBox="0 0 680 240" width="680" height="240" role="img" aria-label="Three naming layers feed one merged catalogue: talkgroup and RID alias files load at startup, the SQLite labels table applies operator edits over them and wins disagreements, and on-air harvest fills gaps; the catalogue feeds the web panels, recordings tree, history search and broadcast feeds, with an export loop back to CSV.">
  <rect x="20" y="176" width="300" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="170" y="193" text-anchor="middle" fill="currentColor" font-size="10">files: talkgroup_file · rid_alias_file (CSV/JSON)</text>
  <text x="170" y="207" text-anchor="middle" fill="var(--fg-muted)" font-size="9">loaded at startup, per system</text>
  <rect x="20" y="118" width="300" height="40" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="170" y="135" text-anchor="middle" fill="var(--accent)" font-size="10">labels table (SQLite, storage.path)</text>
  <text x="170" y="149" text-anchor="middle" fill="var(--fg-muted)" font-size="9">live web/API edits — label wins, WARN on disagreement</text>
  <rect x="20" y="60" width="300" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="170" y="77" text-anchor="middle" fill="currentColor" font-size="10">on-air harvest: talker aliases · discovered TGs</text>
  <text x="170" y="91" text-anchor="middle" fill="var(--fg-muted)" font-size="9">fills gaps; never overwrites your names</text>
  <line x1="170" y1="176" x2="170" y2="158" stroke="currentColor"/><polygon points="166,164 170,156 174,164" fill="currentColor"/>
  <line x1="170" y1="118" x2="170" y2="100" stroke="currentColor"/><polygon points="166,106 170,98 174,106" fill="currentColor"/>
  <line x1="320" y1="90" x2="392" y2="110" stroke="var(--accent)"/><polygon points="384,108 396,112 386,116" fill="var(--accent)"/>
  <rect x="396" y="88" width="150" height="48" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="471" y="108" text-anchor="middle" fill="var(--accent)" font-size="10">merged catalogue</text>
  <text x="471" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="9">one roster per kind</text>
  <line x1="546" y1="112" x2="586" y2="112" stroke="currentColor"/><polygon points="580,108 588,112 580,116" fill="currentColor"/>
  <rect x="588" y="66" width="84" height="92" rx="5" fill="none" stroke="currentColor"/>
  <text x="630" y="84" text-anchor="middle" fill="currentColor" font-size="9">web panels</text>
  <text x="630" y="100" text-anchor="middle" fill="currentColor" font-size="9">recordings tree</text>
  <text x="630" y="116" text-anchor="middle" fill="currentColor" font-size="9">history search</text>
  <text x="630" y="132" text-anchor="middle" fill="currentColor" font-size="9">broadcast feeds</text>
  <path d="M 460 136 C 440 210 240 232 174 220" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <polygon points="182,216 172,220 180,226" fill="var(--fg-muted)"/>
  <text x="440" y="212" text-anchor="middle" fill="var(--fg-muted)" font-size="9">/api/v1/labels/export → CSV that loads as the file</text>
</svg>
<figcaption>Files load first, labels apply on top (and win), the air fills the gaps — and the export loop means the labels store is a convenience, never a lock-in.</figcaption>
</figure>

## The files — real columns, verified

Both keys are per-system, and both loaders dispatch on extension (`.csv` or
`.json`). Paths resolve relative to the folder containing `config.yaml`:

```yaml
trunking:
  systems:
    - name: "Metro-P25"
      protocol: p25
      control_channels:
        - 857_262_500
      talkgroup_file: "../config/talkgroups-p25.csv"
      rid_alias_file: "../config/rids-p25.csv"
```

**The talkgroup CSV** requires one numeric `Decimal` (or `DEC`) column;
everything else is optional and matched by header, case-insensitively:
`Alpha Tag`, `Description`, `Mode`, `Tag`, `Group` (or `Category`),
`Priority`, `Lockout`, `Scan` (or `Active`), `Stream`, `Record`, `Mute`,
`Icon`. Two conventions worth knowing: a Priority of literal **`L`** sets
lockout (matching common community CSVs), and the boolean columns default
*permissive* — `Scan`, `Stream` and `Record` are on unless the cell says
`n`/`no`/`false`/`0`/`off`, while `Lockout` and `Mute` are off unless it says
`y`/`yes`/`true`/`1`:

```
Decimal,Alpha Tag,Description,Tag,Group,Priority,Stream
9001,FIRE-DISP,Fire dispatch,Fire Dispatch,Fire,3,
9002,PD-TAC2,Police tactical 2,Law Tac,Police,5,
9060,PW-ROADS,Public works roads,Public Works,Services,,no
9091,PD-ADMIN,Records desk,Law Talk,Police,L,
```

Row 3 stays recorded and scannable but never reaches a
[broadcast feed]({{ '/blog/tutorials/operator-cookbook-09-sharing-the-feed/' | relative_url }});
row 4's `L` locks it out entirely. What each policy flag does at grant time —
scan lists, priorities, lockout — is
[Trunking Engine Part 6]({{ '/blog/deep-dives/trunking-engine-06-talkgroups-scan-modes/' | relative_url }})'s
territory; this part is only about the file that feeds it.

**The RID CSV** requires `Decimal`/`DEC`/`ID`, and accepts `Alias` (or
`Alpha Tag`), `Description`, `Tag`, `Group` (or `Category`/`Agency`), `Owner`
(or `User`/`Operator`), `Priority`, `Lockout`, `Watch` (or `Active`, default
on), `Icon`:

```
Decimal,Alias,Description,Group,Owner
7001234,ENG-51,Engine 51 mobile,Fire,Metro FD
7005678,CAR-12,Patrol unit 12,Police,Metro PD
```

Where do the numbers come from? RadioReference for talkgroups (its CSV
export's `Decimal`/`Alpha Tag`/`Description` headers load unmodified), your
own History panel for radios — and if the system isn't catalogued anywhere,
[`gophertrunk hunt`]({{ '/blog/deep-dives/the-hunt-11-naming-the-unknown/' | relative_url }})
plus the import pipeline (`POST /api/v1/import` takes RadioReference PDFs and
CSV bundles, previews, then commits) get you a starting file.

## Naming live from the browser

Files are for bulk; the browser is for the radio that keyed up *just now*.
The Talkgroups and Radio IDs panels edit names in place (the TUI too), backed
by two endpoints:

```sh
curl -X PATCH http://127.0.0.1:8080/api/v1/rids/7001234 \
  -H 'Content-Type: application/json' \
  -d '{"alias":"ENG-51","owner":"Metro FD"}'
```

Three behaviors make this layer trustworthy:

- **Unknown IDs are created, not 404'd.** Requiring you to edit
  `rid_alias_file` and restart before naming a radio defeats the point — the
  radios worth naming are the ones showing up live, which by definition
  aren't in the file yet. A synthesized entry behaves exactly like a loaded
  one (and is deliberately *not* tagged as auto-discovered, so cleanup sweeps
  of discovered entries can never delete your names).
- **Names persist; policy doesn't.** The name fields
  (alias/description/tag/group/owner/icon) write to the **labels** table in
  `storage.path` and re-apply over the files at every startup. The policy
  fields — priority, lockout, scan, watch — stay in-memory, exactly as
  before; making those durable is a separate decision the daemon doesn't
  take for you.
- **The label wins, loudly.** At startup, files load first, then labels apply
  on top. If your label and the file disagree, the label is your most recent
  explicit act — it wins, and the daemon logs a WARN naming both so you can
  reconcile.

Reconciling is one request — the export endpoint emits the *merged* catalogue
as a CSV whose columns are pinned by round-trip tests against the loader:

```sh
curl -o talkgroups-merged.csv \
  "http://127.0.0.1:8080/api/v1/labels/export?kind=talkgroup&system=Metro-P25&scope=all"
```

Point `talkgroup_file` at the result, delete the labels
(`DELETE /api/v1/labels/{kind}/{id}`), and you're back to one hand-edited
file — the store is a convenience, never a lock-in. (`scope=labels`, the
default, exports only the ids you've actually labelled; `kind=rid` does the
same for radios, headers `Decimal,Alias,…,Watch,Icon`.) These are mutation
routes, so off-loopback they need the
[auth token]({{ '/blog/deep-dives/running-it-for-real-02-auth-posture/' | relative_url }})
like every other write.

## Names off the air

Some names arrive without you typing anything:

- **Talker aliases.** Many radios transmit their own name on the traffic
  channel; GopherTrunk decodes it and shows it in the Radio IDs roster
  (`talker_alias` in the API) alongside your catalogue alias. On busy P25
  Phase 2 systems most grants never get a voice tap, so the alias decode in
  the voice chain rarely runs — set `signalling_taps: 2` on the wideband
  device to harvest aliases from the signalling stream instead (issue
  [#376](https://github.com/MattCheramie/GopherTrunk/issues/376); the
  full story, including the encrypted-alias cryptanalysis, is
  [The Hunt Parts 11–12]({{ '/blog/deep-dives/the-hunt-12-alias-harvesting/' | relative_url }})).
- **Discovered talkgroups.** Talkgroups seen on grants but absent from your
  file appear in the roster tagged as discovered, so the panel is a live
  census of what actually talks — usually the fastest way to find what your
  CSV is missing.
- **Affiliations.** The
  [affiliation tracker]({{ '/blog/deep-dives/trunking-engine-08-affiliation-tracking/' | relative_url }})
  surfaces radios as they register, populating the RID roster with sightings
  even with no `rid_alias_file` configured at all.

One filing fix worth knowing if you run DMR: an earlier build reclassified a
group call as *individual* whenever the talkgroup number happened to equal a
known radio ID — valid logic on TETRA, where group and individual IDs never
overlap, but DMR shares one 24-bit space for both, so group calls got
misfiled under `individual/<TG>/` and corrupted the roster's Last Talkgroup
column. That mechanism is now scoped to TETRA only; DMR's own call-type
signalling is authoritative, pinned by a regression test.

## First run — what healthy looks like

Restart after adding the files and watch startup:

```
INF daemon: talkgroups loaded system=Metro-P25 count=214
INF daemon: rids loaded system=Metro-P25 count=87
INF daemon: operator labels applied updated=12 created=3
```

The counts are your first sanity check — a `count=0` on a file you know has
rows means the header didn't parse (see the table below). If you've labelled
things that also live in the file, you may also see the divergence WARN,
which is informational by design:

```
WRN daemon: operator label overrides the alias file's name kind=talkgroup id=9001 file=FD-DISP label=FIRE-DISP
```

Then let a call come in: the recorder line now shows the folder tree using
your talkgroup number with the alias visible everywhere the UI renders it,
History filters accept the name, and a search by `FIRE-DISP` in the
Talkgroups panel lands on 9001. The names also flow outward — call webhooks
and [broadcast uploads]({{ '/blog/tutorials/operator-cookbook-09-sharing-the-feed/' | relative_url }})
carry the metadata, and a talkgroup with `stream: no` never leaves the box.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| `trunking: csv missing required Decimal/DEC column` | the header row lacks a `Decimal`/`DEC` column (RIDs also take `ID`) | rename the ID column; everything else is optional |
| WARN `talkgroup_file … failed to load … no alpha tags` | wrong path — paths resolve relative to the folder containing `config.yaml` | fix the relative path or use an absolute one |
| `talkgroups loaded count=0` on a non-empty file | rows whose Decimal cell is blank/non-numeric are skipped silently | check for stray header rows or hex-only IDs; `Decimal` must be base-10 |
| Names applied in the web vanish after restart | no `storage.path` configured — the labels table needs SQLite | set `storage.path` (Part 1's config has it); export with `scope=all` still works meanwhile |
| PATCH returns 403 | mutations gated by `api.auth` off loopback | use the token, or a `trusted_networks` entry — [auth posture]({{ '/blog/deep-dives/running-it-for-real-02-auth-posture/' | relative_url }}) |
| WARN `operator label overrides the alias file's name` every startup | label and file disagree — deliberate, not an error | fold labels into the file via `/api/v1/labels/export`, then delete the label |
| DMR group calls filed under `individual/` | the old cross-protocol reclassification bug | fixed — classification is TETRA-scoped now; update your build |
| Radio roster shows no Last Talkgroup for some radios | those radios only made individual calls, which the column excludes by design | nothing to fix — check History with the RID filter instead |

### Variations

- **JSON instead of CSV** — both loaders take a `.json` array of objects
  (`{"id": 9001, "alpha_tag": "FIRE-DISP", …}`); same fields, same defaults.
- **Per-system files** — every system gets its own `talkgroup_file` /
  `rid_alias_file`, and the label store keys on `(kind, system, target_id)`
  so the CSV export can emit one file per system.
- **Conventional channels too** — a scanner-list channel can pin an explicit
  `talkgroup_id` so its roster row (and any names you attach) survive
  re-ordering of the channel list.
- **Templates that use the names** — `filename_template` accepts `{alpha}`,
  so recordings can carry the alias in the filename itself
  ([naming & sidecars]({{ '/blog/deep-dives/recording-streaming-06-segmentation-naming-sidecars/' | relative_url }})).

## Where this goes next

That's the last recipe with a job of its own. [Part
14]({{ '/blog/tutorials/operator-cookbook-14-kitchen-sink-config/' | relative_url }})
is the finale: the kitchen-sink `config.yaml`, walked section by section and
annotated with which part of this series owns each block — plus the decision
table that turns "what do I want?" into "which part do I read?".

## FAQ

**Can I use a RadioReference talkgroup export directly?**
Yes — the loader keys on the `Decimal` column and matches `Alpha Tag`,
`Description`, `Tag` and `Category` headers case-insensitively, which covers
the RadioReference CSV shape. Trunk-recorder-style files load too; the
`Priority`-of-`L` lockout convention is supported for community files.

**Why did the name I set in the web UI disappear after a restart?**
Almost always: no `storage.path`, so there was no SQLite database for the
labels table to persist into. Set it, re-apply the name, and look for
`daemon: operator labels applied` on the next startup. If storage *is* set,
check the startup log for `operator labels not applied` with an error.

**What's the difference between the labels table and the CSV files?**
Scope and authorship. The files are your bulk, hand-maintained catalogue,
loaded at startup; labels are individual edits made live, persisted
separately, and applied over the files — winning any disagreement because
they're your most recent explicit act. The export endpoint merges the two
back into a file whenever you want a single source of truth again.

**Does GopherTrunk name talkgroups automatically?**
Partially. Radios that transmit talker aliases get named from the air, and
unknown talkgroups appear in the roster as discovered — a census, not a
naming. Actual alpha tags for talkgroups still come from you or a database
like RadioReference; the roster just makes the gaps obvious.

**Why does the Radio IDs panel show two names for one radio?**
They're different columns: your catalogue alias (file or label) and the
talker alias the radio itself transmitted. They often disagree —
fleet-programmed aliases can be stale or cryptic — and GopherTrunk shows
both rather than guessing which you trust.

## Series navigation

**Part 13 of 14** · ←
[Part 12: Two Antennas, One Signal — A Diversity Build]({{ '/blog/tutorials/operator-cookbook-12-diversity-mrc/' | relative_url }})
· Next →
[Part 14: The Kitchen-Sink Config, Annotated]({{ '/blog/tutorials/operator-cookbook-14-kitchen-sink-config/' | relative_url }})
