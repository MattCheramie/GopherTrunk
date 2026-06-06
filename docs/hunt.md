# Hunting & mapping unknown systems

Many states and counties run trunked radio systems that are **not documented
on RadioReference.com**. GopherTrunk can already follow and decode a system
once you know its control-channel frequencies and identity; `gophertrunk hunt`
closes the loop for the *undocumented* case — it turns one or more IQ captures
of a suspected control channel into a structured **DiscoveredSystem** map and
exports it to standardized files plus a ready-to-paste RadioReference
submission package.

## What it does

For each capture you supply, `hunt`:

1. **Identifies** the protocol (auto, across all 13 GopherTrunk decoders) — or
   uses the protocol you pass with `-protocol`.
2. **Decodes** the control channel and **accumulates** what it observes into a
   single system map: identity (P25 NAC, plus WACN/SYSID/RFSS/Site and other
   per-protocol identifiers when the decoder surfaces them), the site's control
   channel, and every talkgroup seen in a grant.
3. **De-duplicates** across captures — fold several sites or several captures
   of the same system into one map (sites keyed by RFSS/Site, channels by
   frequency, talkgroups by id).
4. **Exports** to the formats you choose and, optionally, **merges straight
   into `config.yaml`** so you can start scanning the new system immediately.

## Quick start

```sh
# Capture a suspected control channel off a live SDR (see `gophertrunk capture`)
gophertrunk capture -serial 00000001 -freq 851012500 -seconds 60 -out cc.cfile

# Map it and export everything (bundle + trunk-recorder + RR package)
gophertrunk hunt -in cc.cfile -freq 851012500 -format f32 -sample-rate 2400000 \
  -name "New County P25" -state AZ -county Maricopa -out ./hunt-newcounty
```

Fold multiple sites of the same system into one map:

```sh
gophertrunk hunt \
  -in site1.cfile -freq 851012500 \
  -in site2.cfile -freq 853512500 \
  -format f32 -sample-rate 2400000 -name "New County P25"
```

Discover and merge straight into `config.yaml` (same writer the PDF/CSV
importer uses):

```sh
gophertrunk hunt -in cc.cfile -freq 851012500 -sample-rate 2400000 \
  -commit -config ./config.yaml
```

> **Tip:** pass `-freq` (the capture's center frequency in Hz) for every
> `-in` so the recorded control channel carries an absolute frequency.
> Without it, a baseband capture locks at 0 Hz and the channel can't be
> exported.

## Live mode (spectrum sweep)

Instead of supplying captures, point `hunt` at an SDR and let it find the
control channels on the air. It sweeps the band(s) you give with an FFT,
detects candidate carriers above the noise floor, then identifies, decodes,
and maps each one through the same pipeline as the offline path.

```sh
# Sweep 851-869 MHz on an SDR and map whatever trunked systems it finds
gophertrunk hunt -serial 00000001 -sample-rate 2400000 -band 851:869 \
  -state AZ -county Maricopa -out ./hunt-live

# Probe a known control-channel list directly (skip the sweep)
gophertrunk hunt -serial 00000001 -sample-rate 2400000 \
  -no-sweep -candidates 851.0125,853.5125
```

Live sweep tuning flags: `-band low:high` (MHz, repeatable), `-sweep-dwell`
(accumulation per step), `-peak-threshold-db` (carrier detection threshold over
the noise floor), `-min-spacing` (Hz between carriers), `-fft-size`,
`-dwell-seconds` (IQ captured per candidate for identify+decode), plus
`-gain`/`-ppm` for the SDR. This standalone CLI path opens the SDR directly; a
daemon-integrated live hunt (spare-SDR-else-borrow acquisition, with a
REST/TUI/web cockpit) is the next phase.

## Export formats

Select with `-formats` (comma-separated; default is all three):

| Value | File | Contents |
|---|---|---|
| `bundle` | `<name>.csv` | GopherTrunk multi-section import bundle — round-trips back in via `import-pdf -csv` (and is what `-commit` writes). |
| `trunk-recorder` | `<name>.json` | A trunk-recorder system config stanza (control channels + type + modulation). |
| `rr` | `<name>-radioreference.md` | A human-readable RadioReference submission package (identity, sites, control channels, observed talkgroups) with the Submit link. |

## RadioReference duplicate check (optional, read-only)

RadioReference has **no public write API** — new systems are added through a
web Submit form reviewed by administrators. GopherTrunk never posts anything.
What it *can* do is use RadioReference's read-only SOAP web service to check
whether your discovery already exists, so you don't submit a duplicate. The
result is folded into the `rr` submission package as a warning banner.

Provide an API key (request one at <https://www.radioreference.com/account/api>)
via `-rr-key`, the `radioreference.api_key` config block, or the
`GOPHERTRUNK_RR_KEY` environment variable (username/password via
`GOPHERTRUNK_RR_USER` / `GOPHERTRUNK_RR_PASS`), then point the check at
candidate systems:

```sh
# Compare against every system registered in a RadioReference county (ctid)
gophertrunk hunt -in cc.cfile -freq 851012500 -sample-rate 2400000 \
  -rr-key "$GOPHERTRUNK_RR_KEY" -rr-county-id 261

# …or against specific suspected system ids
gophertrunk hunt -in cc.cfile -freq 851012500 -sample-rate 2400000 \
  -rr-check-sid 7715 -rr-check-sid 8123
```

With no key the check is skipped and the export still happens. Use `-no-rr`
to force it off even when a key is configured.

## Key flags

| Flag | Meaning |
|---|---|
| `-in` (repeatable) | IQ capture of a suspected control channel |
| `-freq` (repeatable) | Nominal center frequency in Hz for the matching `-in` |
| `-format` / `-sample-rate` | IQ encoding (`u8`/`f32`) and sample rate |
| `-protocol` | Force a decoder; default auto-identifies each capture |
| `-min-confidence` | Skip auto-identified captures below this (default 0.40) |
| `-name` / `-state` / `-county` / `-location` | System metadata for the exports |
| `-out` / `-formats` | Output directory and which files to write |
| `-commit` / `-config` / `-dry-run` / `-force` | Merge the discovery into `config.yaml` |
| `-rr-key` / `-rr-county-id` / `-rr-check-sid` / `-no-rr` | RadioReference duplicate check |

## Limitations & roadmap

- **Live cockpit.** Besides the standalone CLI live sweep (`-serial`), the
  daemon exposes a live hunt over REST that shares the running radio via
  spare-SDR-else-borrow acquisition:

  | Method & path | Purpose |
  |---|---|
  | `GET /api/v1/hunt` | Run status + the discovered system map |
  | `POST /api/v1/hunt/start` | Start a run (`{bands, candidates, no_sweep, protocol, name, …}`) |
  | `POST /api/v1/hunt/stop` | Cancel the active run |
  | `GET /api/v1/hunt/export?format=bundle\|trunk-recorder\|rr` | Download the discovery |
  | `POST /api/v1/hunt/commit` | Merge the discovery into `config.yaml` (`{force, dry_run}`) |

  Progress streams over the event bus (`hunt.progress` / `hunt.candidate` /
  `hunt.done`) via `GET /api/v1/events`. The **web console** has a *Hunt* tab
  (start a run, watch progress, download/commit the result) and the **TUI** has
  a *Hunt* panel that monitors the run and can stop it.
- **Topology depth.** For **P25** the hunt now recovers full topology —
  WACN/SYSID, the camped RFSS/Site, advertised neighbor (adjacent) sites, and
  the over-the-air band plan (IDEN_UP) — all surfaced in the exports. For other
  protocols the map carries the identity the decoder surfaces (DMR ColorCode/
  SystemID, NXDN Site/System, TETRA MCC/MNC/LA, …) plus the observed site and
  talkgroups; neighbor accumulation for those protocols lands incrementally.
  Band-plan-from-air is P25-only (others resolve channels via configured plans).
- **Talkgroup names.** A blind discovery can only record talkgroup *numbers*
  and activity; names/descriptions are filled in during RadioReference
  submission.
