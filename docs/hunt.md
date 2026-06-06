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

- **Operator-supplied captures.** Today you supply the captures (one per
  suspected control channel). A live wideband **spectrum-sweep** front-end
  that finds the candidate control channels on the air automatically — using
  the existing FFT spectrum producer for peak detection — is the planned
  follow-on, along with a live cockpit/TUI/web panel.
- **Topology depth.** Full multi-site topology (WACN/SYSID/RFSS/Site,
  neighbor/adjacent sites, band plan) is richest on **P25**; for other
  protocols the map carries the identity the decoder surfaces plus the single
  observed site and its talkgroups — enough to export and submit.
- **Talkgroup names.** A blind discovery can only record talkgroup *numbers*
  and activity; names/descriptions are filled in during RadioReference
  submission.
