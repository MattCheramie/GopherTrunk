---
layout: page
title: Hunting & mapping unknown systems
description: Discover, identify, map, and export undocumented trunked radio systems from live SDRs or IQ captures with gophertrunk hunt
nav_group: Operate
---

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

### Off-centre control channels

`-auto-tune` no longer assumes the control channel is the loudest carrier in
the capture. It detects the ranked carrier candidates and tries each until one
identifies/locks, so a control channel that sits off-centre **and** below a
louder neighbour (e.g. a P25 CC at −625 kHz inside a 2 MSPS grab whose strongest
carrier is a voice channel) is still found — no manual `-tune-hz` needed. The
same multi-candidate auto-tune backs `gophertrunk replay`/`analyze -auto-tune`.

To inventory **every** carrier in one wideband recording (not just the control
channel), add `-detect-carriers` to an offline `-survey`: it FFT-peak-detects all
carriers, shifts each to baseband, and classifies/decodes them — the offline
equivalent of the live spectrum sweep. Pair it with `-survey-deep` so a digital
control channel the blind classifier reads as narrowband FM is still handed to
the trunking identify:

```sh
# Find the trunked system + map the whole band from one wideband capture
gophertrunk hunt -survey -survey-deep -detect-carriers \
  -in wideband.cfile -freq 450500000 -format f32 -sample-rate 2000000 \
  -out ./hunt-survey
```

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

### Whole-device sweep (the full-spectrum survey)

To inventory **everything the dongle can hear** without naming a band, pass
`-whole-device`: the sweep range is derived from the SDR's own tuning limits
(e.g. 24 MHz–1.766 GHz for an R820T, 1 MHz–4.294 GHz for a HackRF), so one run
walks the entire reachable spectrum. It prints the resolved span, the step
count, and a rough ETA before it starts. Pair it with `-survey -classify-only`
for a fast first pass that catalogues every carrier:

```sh
gophertrunk hunt -survey -classify-only -whole-device \
  -serial 00000001 -sample-rate 2400000 -survey-ndjson hunt.ndjson
```

`-survey-ndjson` streams each classified carrier to disk as it is found, and
`-resume` skips carriers already recorded — so a long full-spectrum sweep is
crash-safe and restartable. A device that can't report a tuning range (a
network or file source) tells you to pass an explicit `-band` instead.

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
  a *Hunt* panel that monitors the run and can stop it. The web *Hunt* tab also
  lists each candidate's **capture report** (control frequency, protocol, lock /
  skip reason) and the discovered **sites** with their control channels once a
  run completes.
- **DMR Tier III band-plan learning, in the web console.** When the LCN
  autoconfig learner fits a band plan, the web *CC Activity* tab shows the live
  `DMR grant` observations and the `Band plan learned` result (base frequency,
  channel spacing, confidence), and the *Systems* tab shows the active
  LCN→frequency plan per system (`GET /api/v1/systems` → `dmr_band_plan`),
  whether operator-configured or learned over the air. The control-channel hunt
  lifecycle (`cchunt.progress` / `cchunt.failed`) is surfaced there too.
- **Topology depth.** What the hunt recovers per protocol:

  | Protocol | Identity | Neighbor sites | Band plan |
  |---|---|---|---|
  | P25 | WACN/SYSID/RFSS/Site/LRA | yes (adjacent-site) | yes (IDEN_UP, over-the-air) |
  | DMR Tier III | SystemID/RFSS/Site/ColorCode | yes (adjacent-site CSBK) | configured plan |
  | EDACS | SystemID | yes (adjacent-site CCW) | configured plan |
  | Motorola Type II | SystemID | yes (adjacent-site OSW) | configured plan |
  | NXDN | SystemID/Site/Location | — | configured plan |
  | TETRA | MCC/MNC/LA/colour code | — | configured plan |
  | MPT1327 / dPMR | SystemID | — | configured plan |
  | LTR / YSF / D-STAR | — (no system identity broadcast) | — | — |

  Band-plan-from-air is P25-only; the others resolve channels via the
  configured band plan. Neighbor accumulation beyond the protocols above (e.g.
  TETRA neighbour cells) lands incrementally.
- **Talkgroup names.** A blind discovery can only record talkgroup *numbers*
  and activity; names/descriptions are filled in during RadioReference
  submission.
