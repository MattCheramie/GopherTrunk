---
title: Importing systems from RadioReference PDFs
---

# `gophertrunk import-pdf`

The `import-pdf` subcommand parses RadioReference.com system PDF
exports and merges them into your `config.yaml`, generating per-system
Trunk-Recorder-style talkgroup CSVs as it goes. It is intended for
the initial bring-up of a new region — replacing the manual transcription
of 15+ sites and several hundred talkgroups per system.

## Quick start

1. Sign in to RadioReference and open the trunking system page (e.g.
   the "Maricopa County" or "Regional Wireless Cooperative" page).
2. Use the "Export PDF" option in the page footer and save the file.
3. Run:

   ```
   gophertrunk import-pdf \
     -pdf maricopa.pdf \
     -pdf rwc.pdf \
     -config /etc/gophertrunk/config.yaml
   ```

4. The TUI launches. Use the arrow keys to navigate systems, press
   <kbd>Enter</kbd> to drill in, <kbd>Tab</kbd> to switch between the
   Sites and Talkgroups tabs, and the edit keys below to tune the
   import.
5. Press <kbd>w</kbd> to write the merged config + CSVs, or <kbd>q</kbd>
   to discard.

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
```

Re-importing a system whose `name` already exists in `config.yaml`
requires `-force`:

```
gophertrunk import-pdf -pdf rwc.pdf -config config.yaml -no-tui -force
```

Without `-force` the importer aborts before touching anything on disk.

## What the importer writes

- **`config.yaml`** — the existing file is loaded, every comment and
  unrelated block (sdr, api, scanner, audio, tone_out…) is preserved
  verbatim, and a new entry is appended to `trunking.systems[]` per
  imported PDF. The control-channel list flattens the
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

## Supported PDFs

| Protocol | Status |
| --- | --- |
| Project 25 Phase 1 / Phase 2 | Supported |
| DMR / NXDN / TETRA / EDACS | Not yet — the PDF layouts differ |

The importer always sets `protocol: p25` for the parsed system, since
the RadioReference Phase 1 and Phase 2 PDFs share the same on-page
schema and the daemon's runtime distinguishes the two via the
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
