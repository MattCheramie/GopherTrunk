---
layout: page
title: Import (PDF / CSV)
description: Importing trunked-system definitions from RadioReference PDFs and structured CSV bundles into config.yaml
nav_group: Operate
---

# `gophertrunk import-pdf`

The `import-pdf` subcommand parses trunked-system data from two sources
and merges it into your `config.yaml`, generating per-system Trunk-
Recorder-style talkgroup CSVs as it goes:

- **RadioReference.com PDF exports** — the page-footer "Export PDF"
  button on any P25 trunking-system page.
- **Structured CSV bundles** — a single multi-section CSV file per
  system, format documented below. Use this when your data comes from
  somewhere other than RadioReference (the radio wiki for your region,
  a hand-curated spreadsheet, an export from another scanner program,
  …).

Both sources flow through the same TUI, the same writer, and the
same atomic-merge pipeline, so the daemon-side outputs are identical
regardless of input format.

> **Why the name?** The subcommand started PDF-only; the `-pdf` name
> is now slightly misleading but kept for backwards compatibility. Pass
> any combination of `-pdf` and `-csv` flags — they may be mixed in a
> single invocation.

## Quick start — PDF (RadioReference)

1. Sign in to RadioReference and open the trunking-system page (e.g.
   the "Maricopa County" or "Regional Wireless Cooperative" page).
2. Click "Export PDF" in the page footer; save the file.
3. Run:

   ```
   gophertrunk import-pdf \
     -pdf maricopa.pdf \
     -pdf rwc.pdf \
     -config /etc/gophertrunk/config.yaml
   ```

4. The TUI launches. Review/prune sites, toggle Scan/Lockout/Priority
   on talkgroups, then press <kbd>w</kbd> to write or <kbd>q</kbd> to
   discard.

## Quick start — CSV

1. Build a CSV file in the format below (one file per system).
2. Run:

   ```
   gophertrunk import-pdf \
     -csv my-system.csv \
     -config /etc/gophertrunk/config.yaml
   ```

3. The TUI launches with the same review/edit flow as the PDF path.

## CSV format

A single CSV file contains **one system**. The file is split into named
sections; each section has its own header row followed by data rows.
Section order doesn't matter, but every system needs at minimum a
`metadata` section with a `name` and either a `sites` or `talkgroups`
section (or both — typical).

Sections are delimited by a comment line of the form

```
# Section: <section-name>
```

(case-insensitive, flexible whitespace). Any other line starting with
`#` is treated as a free-form comment and skipped. Empty lines are
also skipped.

Standard CSV quoting rules apply (double-quote fields that contain
commas or quote characters; escape an embedded `"` as `""`).

### `metadata` section

A simple key/value table. Required column header: `key,value`
(or `field,val` — both spellings accepted).

| Key | Required | Description |
| --- | --- | --- |
| `name` | yes | System display name (becomes `trunking.systems[].name`). |
| `protocol` | yes¹ | One of `p25`, `dmr`, `nxdn`. Defaults to `p25` if omitted. |
| `sysid` | no | System ID, used to build the talkgroup CSV filename suffix. |
| `wacn` | no | Wide-area network code. Informational. |
| `location` | no | Free-text location (e.g. `"Phoenix, AZ"`). |
| `county` | no | Free-text county. |
| `system_type` | no | Free-text type (e.g. `"Project 25 Phase II"`). |

¹ Validated by the daemon's `internal/config.Config.Validate`. The
importer rejects unknown protocols before touching the config.

### `sites` section

One row per site. Frequencies for a site live in a single
pipe-delimited cell so the row stays atomic — see the example.

| Column | Required | Description |
| --- | --- | --- |
| `rfss` | no | Integer RFSS ID (decimal). |
| `site_id` | no | Integer site ID (decimal). |
| `site_name` | yes | Human-readable site name. |
| `county` | no | County. |
| `frequencies` | yes | `\|`-delimited list of `MHz[c]` entries. Add a trailing `c` to mark a control-channel-capable frequency. Spaces, commas (in quoted fields), and semicolons are also accepted as separators. |

Frequencies are validated to fall within 25–1300 MHz (a wide trunking
band). Anything outside that range fails the import.

### `talkgroups` section

One row per talkgroup. Column names use Trunk-Recorder conventions so
files exported from that tool import without modification.

| Column | Required | Description |
| --- | --- | --- |
| `decimal` (or `DEC`) | yes | Decimal talkgroup ID. |
| `hex` | no | Hex form; auto-computed from `decimal` if absent. |
| `mode` | no | `D` (digital), `A` (analog), or `M` (mixed). Defaults to `D`. `T`, `TE`, `DE` are accepted as aliases for `D`. |
| `alpha_tag` (or `Alpha Tag`) | no | Short label. |
| `description` (or `desc`) | no | Long description. |
| `tag` (or `Category`) | no | Function tag (`Law Dispatch`, `Fire-Tac`, …). |
| `group` | no | Top-level group label. |
| `priority` | no | Integer 1–10 (1 = highest). Empty = unset. |
| `lockout` | no | `Y`/`N` (also `yes`/`no`, `true`/`false`, `1`/`0`). Default `N`. |
| `scan` (or `Active`) | no | `Y`/`N`. Default `Y`. |

### Annotated example

A complete example bundle lives at
[`samples/rr-import/example.csv`](../samples/rr-import/example.csv).

```csv
# Section: metadata
key,value
name,Example P25 System
protocol,p25
sysid,49A
wacn,BEE99
location,"Example City, AZ"

# Section: sites
rfss,site_id,site_name,county,frequencies
1,1,Tower Alpha,Example,851.0125c|851.2625c|852.0125|853.0125
1,2,Tower Bravo,Example,853.5125c|854.0125c|854.5125

# Section: talkgroups
decimal,hex,mode,alpha_tag,description,tag,group,priority,lockout,scan
1000,3e8,D,DISPATCH,Primary Dispatch,Law Dispatch,Police,1,,Y
1001,3e9,D,TAC1,Tactical 1,Law Tac,Police,,,Y
1002,3ea,D,FIRE-DSP,Fire Dispatch,Fire Dispatch,Fire,2,,Y
1003,3eb,D,FIRE-TAC,Fire Tactical,Fire-Tac,Fire,,,Y
1004,,D,EMS,EMS Operations,EMS Dispatch,EMS,2,,Y
1005,,A,Analog Repeat,Backup analog,Multi-Tac,Common,,Y,N
```

This bundle produces one entry in `trunking.systems[]` (with four
control-channel-capable frequencies flattened across both sites) and a
six-row talkgroup CSV. The last talkgroup is locked out (`Lockout=Y`)
and excluded from scan (`Scan=N`).

### Tips for hand-authored bundles

- Spreadsheet editors handle the format fine — open the file as CSV,
  edit, save. The `# Section: …` comment rows pass through Excel/Numbers
  as a single-column row.
- The `frequencies` cell allows pipes, spaces, semicolons, or commas
  as separators (use commas only inside a quoted cell — otherwise they
  collide with the CSV's own field separator).
- `hex`, `priority`, and `lockout` are all safe to leave empty; the
  importer fills sensible defaults.
- To round-trip data from an existing GopherTrunk talkgroup CSV (the
  per-system file the daemon reads), wrap it in a `# Section: talkgroups`
  marker and add a `# Section: metadata` block above it — the column
  headers are already compatible.

## TUI key bindings

| View | Key | Action |
| --- | --- | --- |
| Any | <kbd>w</kbd> | Write merged config + CSVs and exit |
| Any | <kbd>q</kbd> / <kbd>Ctrl+C</kbd> | Quit without writing |
| Systems list | <kbd>↑</kbd>/<kbd>↓</kbd> | Move cursor |
| Systems list | <kbd>Enter</kbd> | Open system |
| System (Sites tab) | <kbd>Space</kbd> | Toggle site Include flag |
| System (any tab) | <kbd>Tab</kbd> | Switch Sites ↔ Talkgroups |
| Talkgroups | <kbd>s</kbd> | Toggle Scan |
| Talkgroups | <kbd>L</kbd> | Toggle Lockout |
| Talkgroups | <kbd>0</kbd>–<kbd>9</kbd> | Set Priority (0 clears) |
| Talkgroups | <kbd>e</kbd> | Edit Alpha Tag (Enter saves, Esc cancels) |
| System view | <kbd>Esc</kbd> / <kbd>h</kbd> | Back to systems list |

## CLI / headless mode

Skip the TUI with `-no-tui` (useful for CI bring-up). Preview the
changes without writing using `-dry-run`:

```
gophertrunk import-pdf -pdf maricopa.pdf -config config.yaml -no-tui -dry-run
gophertrunk import-pdf -csv my-system.csv -config config.yaml -no-tui -dry-run
```

Re-importing a system whose `name` already exists in `config.yaml`
requires `-force`:

```
gophertrunk import-pdf -csv rwc.csv -config config.yaml -no-tui -force
```

Without `-force` the importer aborts before touching anything on disk.

### Flags

| Flag | Description |
| --- | --- |
| `-pdf <file.pdf>` | RadioReference PDF (repeatable). |
| `-csv <file.csv>` | Structured CSV bundle (repeatable). |
| `-config <path>` | Existing `config.yaml` (merged in place). Default `./config.yaml`. |
| `-csv-dir <dir>` | Where to write talkgroup CSVs. Default: directory of `-config`. |
| `-no-tui` | Skip the review TUI; merge straight from parsed defaults. |
| `-dry-run` | Print the planned changes and exit without writing. |
| `-force` | Overwrite an existing `trunking.systems[]` entry with the same name. |
| `-wizard` | Launch the interactive config-builder wizard (see below). |

## Config-file builder (`-wizard`)

`-wizard` is for first-time operators who don't yet have a
`config.yaml`. It launches an interactive walk-through that asks one
question per daemon-config section — log level, API bind addresses,
auth mode, CORS, storage paths, recordings directory, retention,
SDR devices, scanner cockpit, audio playback — and then writes a
fully-annotated `config.yaml` you can immediately run. Defaults at
every step match what `config.example.yaml` ships, so pressing
Enter through every screen still produces a valid file.

Three usage patterns:

```
# Build a fresh config from scratch (no PDF / CSV imports).
gophertrunk import-pdf -wizard

# Build a fresh config and immediately merge a RadioReference PDF
# on top of it — the wizard runs first, then the existing site-
# review TUI takes over.
gophertrunk import-pdf -wizard -pdf maricopa.pdf

# Write to a custom path.
gophertrunk import-pdf -wizard -config /etc/gophertrunk/config.yaml
```

### Wizard key bindings

| Key | Action |
| --- | --- |
| `Enter` | Save the current field and advance — to the next field within a step, or to the next step when on the last field. |
| `Tab` / `Shift+Tab` | Move between fields within a step (without advancing). |
| `↑` / `↓` | Same as Shift+Tab / Tab. |
| `←` / `→` | Cycle through values on a choice field (log level, auth mode, scan mode, …) **or** toggle a boolean field. |
| `y` / `n` / `Space` | Toggle a boolean field. |
| `Esc` | Back up one step. |
| `q` / `Ctrl+C` | Abort without writing. |

The CORS allow-list and SDR-device builder are list editors —
type a value and press Enter to append, Backspace to pop. The
review step (final screen) shows a preview of the rendered YAML;
press Enter to write or Esc to back up and edit.

**Config-file path field:** the welcome step's path accepts shell-
style env-var references — `%APPDATA%\GopherTrunk\config.yaml`
(Windows), `$HOME/.config/gophertrunk/config.yaml` (POSIX), and a
leading `~` for the home dir all expand at write time. The review
screen shows "resolves to: \<abs\>" beneath the typed path when
expansion changes it, so you see the actual destination before
pressing Enter. The default picked when the wizard starts is
whatever `$GOPHERTRUNK_CONFIG` points at (the Windows installer
sets this); otherwise `./config.yaml` when the current directory
is writable, or `<UserConfigDir>/GopherTrunk/config.yaml` when
it isn't (e.g. launched from `C:\Program Files\GopherTrunk\`).

## What the importer writes

- **`config.yaml`** — the existing file is loaded, every comment and
  unrelated block (sdr, api, scanner, audio, tone_out…) is preserved
  verbatim, and a new entry is appended to `trunking.systems[]` per
  imported source. The control-channel list flattens the
  control-channel-capable frequencies of every Include=true site.
- **`talkgroups-<slug>-<sysid>.csv`** — one file per system, written
  alongside `config.yaml` (override the directory with `-csv-dir`).
  Columns: `Decimal,Hex,Mode,Alpha Tag,Description,Tag,Group,Priority,Lockout,Scan`.
  This is the same format `internal/trunking.TalkgroupDB.LoadCSV`
  understands, so the daemon picks the file up on the next start
  without any extra wiring.

Writes are atomic: each CSV and the config are written to a temp file
in the destination directory and `rename(2)`-d into place after both
the struct-level and node-level YAML schema validations pass.

## Supported sources / protocols

| Source | Protocol | Status |
| --- | --- | --- |
| PDF | Project 25 Phase 1 / Phase 2 | Supported |
| PDF | DMR / NXDN / TETRA / EDACS | Not yet — the PDF layouts differ |
| CSV | P25 / DMR / NXDN | Supported (protocol declared in `metadata`) |

The PDF importer always sets `protocol: p25` for the parsed system,
since the RadioReference Phase 1 and Phase 2 PDFs share the same
on-page schema and the daemon's runtime distinguishes the two via the
[`p25_phase2_*` keys](../config.example.yaml). Operators on
pure-Phase-2 systems may want to hand-add
`p25_phase2_clock_mode: gardner` to the imported entry — defaults are
correct for Phase 1 captures.

## Known PDF format hazards

- **Custom font encoding.** RadioReference's PDF export uses a font
  subset where every glyph's encoded byte sits 27 below its real
  ASCII codepoint. The importer reverses the shift per-glyph during
  extraction. If RadioReference changes the encoding the importer
  will produce gibberish — open an issue with a sample PDF attached.
- **Ligature drops.** The font subset has no `ﬃ`/`ﬁ`/`ﬂ` glyphs, so
  words like "Office" arrive as "ONce". The importer applies a small
  fix-up table (`Office`, `Officers`, `Official`, …). If you see
  garbled text in the TUI's Group column, fix it in the CSV after
  write — the field is cosmetic and the daemon never parses it.
- **Continuation lines.** Sites with more than seven frequencies wrap
  to the next visual row. The importer rejoins continuation lines
  automatically via the positioned-text Y-coordinate.
- **Two-token counties.** "La Paz" and "Santa Cruz" are recognised
  as multi-token county names; anything else assumes the County is
  the last token before the first frequency.

## Re-importing

`-force` overwrites a same-name entry in `trunking.systems[]` and
truncates the matching talkgroup CSV. Operator edits made via the API
(Priority/Lockout mutations applied to `TalkgroupDB`) live only in
memory; if you have persistent edits in the CSV, back it up before
re-importing.

## See also

- [`config.example.yaml`](../config.example.yaml) — full schema for
  `trunking.systems[]`.
- [`internal/trunking/talkgroup.go`](../internal/trunking/talkgroup.go) —
  source of truth for the CSV format the importer writes.
- [`samples/rr-import/example.csv`](../samples/rr-import/example.csv) —
  worked example of the multi-section CSV bundle.
