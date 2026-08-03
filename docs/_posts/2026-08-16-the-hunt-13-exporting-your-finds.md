---
title: "The Hunt, Part 13: Exporting Your Finds — RadioReference, TrunkRecorder, SigMF"
description: How GopherTrunk exports a discovered system in every format that matters — a round-tripping CSV import bundle, a TrunkRecorder JSON stanza, a RadioReference submission package with a live cross-reference diff, and a SigMF-tagged capture bundle — one DiscoveredSystem, one Write call, many destinations.
category: deep-dives
keywords: export trunked system, radioreference submission, trunkrecorder config, sigmf capture bundle, csv import bundle, rr diff cross reference, discovered system export, gtbundle, gophertrunk the hunt
tags: [the-hunt, export, radioreference, sigmf, interop, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Hunt"
series_part: 13
---

*Part 13 of **The Hunt**. Our 851–869 MHz carrier has been swept up, classified,
locked, mapped across sites, named as far as the RF allows, and mined for aliases.
The find is complete — and completely stuck on one machine. This post is about
getting it *out*: turning the `DiscoveredSystem` into files the rest of the world
reads. A RadioReference submission so the system gets documented. A TrunkRecorder
config so another tool can record it. GopherTrunk's own bundle so it re-imports
without loss. And a SigMF-tagged capture bundle so the raw IQ is portable too.*

> **TL;DR:** One `DiscoveredSystem`, one `Write(w, sys, format, hints)` call, four
> encodings. **Bundle** is a multi-section CSV that is the exact reverse of
> GopherTrunk's importer — it round-trips straight back into `config.yaml`.
> **TrunkRecorder** is a JSON system stanza with the control channels flattened.
> **RR** is a Markdown submission package the operator pastes into RadioReference's
> web form (there is no write API), optionally carrying a **live cross-reference
> diff** that upgrades "is this a duplicate?" to "what's new or off here?" A fifth
> path exports the raw capture as a **SigMF**-sidecar'd `.gtb` bundle so the IQ is
> interoperable with inspectrum and GNU Radio. Every format is honest about what a
> blind discovery can't know.

**Key takeaways**

- **One writer, four formats.** `hunt.Write` dispatches a `Format` — bundle,
  trunk-recorder, rr, summary — so the CLI, daemon, and cockpit all export through
  one seam.
- **The bundle round-trips.** Its section markers and column names match the
  importer exactly, so a discovery exported as CSV re-imports into `config.yaml`
  without loss — and site names with commas survive via real `encoding/csv`.
- **RR export can diff against reality.** `DiffAgainstRR` pairs discovered
  frequencies against an existing RadioReference system — exact matches are
  silent, near matches are flagged as tuning offsets, the rest are "new."
- **Two artifacts leave the machine.** The system *map* (bundle/TR/RR) and the raw
  *capture* (a SigMF-tagged `.gtb` bundle) — so both what you concluded and what
  you measured are portable.

## Cheat sheet

| Format | What it is | Where it lives |
|---|---|---|
| `FormatBundle` | multi-section CSV, reverses the importer | `internal/hunt/export_bundle.go` |
| `FormatTrunkRecorder` | JSON system stanza | `internal/hunt/export_trunkrecorder.go` |
| `FormatRR` | Markdown RadioReference submission | `internal/hunt/export_rr.go` |
| `RRDiff` / `DiffAgainstRR` | cross-reference vs an existing RR system | `internal/hunt/export.go` |
| `SurveyJSON` / `SurveyCSV` | the full classified inventory | `internal/hunt/export_survey.go` |
| SigMF sidecar | portable capture metadata in a `.gtb` bundle | `internal/gtbundle/sigmf.go` |

## In this post

- **One writer, four formats** — the `Write` seam and the `Format` enum.
- **The bundle that round-trips** — CSV as the exact inverse of the importer.
- **TrunkRecorder** — flattening a multi-site map into one stanza.
- **The RadioReference package** — a submission, and a diff against reality.
- **SigMF** — making the raw capture portable too.

## One writer, four formats

Every export goes through a single function. `Write` (and its diff-carrying
sibling `WriteWithRRDiff`) sorts the system for deterministic output, then
dispatches on a `Format`:

```go
// internal/hunt/export.go (shape)
func WriteWithRRDiff(w io.Writer, sys *DiscoveredSystem, f Format, hints []DuplicateHint, diff *RRDiff) error {
    sys.sortAll()
    switch f {
    case FormatBundle:         return writeBundle(w, sys)
    case FormatTrunkRecorder:  return writeTrunkRecorder(w, sys)
    case FormatRR:             return writeRR(w, sys, hints, diff)
    case FormatSummary:        return writeSummary(w, sys)
    }
    return fmt.Errorf("hunt: unsupported format %d", f)
}
```

`ParseFormat` maps the strings operators type (`bundle`/`csv`, `trunk-recorder`/`tr`,
`rr`, `summary`) onto the enum, and `FileExtension` gives each its conventional
suffix (`.csv`, `.json`, `.md`, `.txt`). Because there is one seam, the offline
CLI, the daemon's export endpoint, and the web cockpit all produce byte-identical
files — the same "one engine, many drivers" discipline the whole series has leaned
on, now at the exit.

The reason for four formats rather than one is that a find has more than one
audience. RadioReference wants a human-reviewed submission; TrunkRecorder wants a
machine-readable config; GopherTrunk itself wants something it can re-ingest; and
sometimes you just want a readable network report to skim. Rather than pick a
lowest common denominator, the exporter speaks each destination's native dialect
— and `sortAll` runs first every time, so any two runs over the same system emit
identical bytes, which is what makes the golden tests and the round-trip possible
at all.

## The bundle that round-trips

The default format is the one that matters most for GopherTrunk itself. A
discovery isn't useful if it can only leave — it has to be able to come *back*, as
configuration you can scan. So `writeBundle` emits a multi-section CSV that is the
**exact inverse** of the importer:

```go
// internal/hunt/export_bundle.go (shape)
// writeBundle emits the multi-section CSV import bundle that
// cmd/gophertrunk/import_csv.go parseCSVStream reads. The section markers and
// column names match parseMetadataSection / parseSitesSection /
// parseTalkgroupsSection exactly so a discovery round-trips back into
// config.yaml without loss.
func writeBundle(w io.Writer, sys *DiscoveredSystem) error {
    bw := &bundleWriter{w: w}
    bw.section("metadata")
    bw.row("key", "value")
    bw.row("name", sys.DisplayName())
    // …protocol, sysid, wacn, location, county
    bw.section("sites")
    bw.row("rfss", "site_id", "site_name", "county", "frequencies")
    // …one row per site, control channels carrying a trailing 'c' flag
    bw.section("talkgroups")
    // …decimal, hex, mode=D, blank alpha_tag/description, scan=Y
    return bw.flush()
}
```

Two details make the round-trip real. Frequencies are rendered by `formatMHz`
into exactly the spelling the importer parses (`851012500` → `851.0125`), and the
whole thing is written through `encoding/csv`, whose dialect matches the
importer's line splitter — so a site name or location containing a comma survives
the trip intact. The talkgroup rows ship with `mode=D` and `scan=Y` defaults but
**blank** alpha tags and descriptions, faithfully carrying forward the
[naming post's]({{ '/blog/deep-dives/the-hunt-11-naming-the-unknown/' | relative_url }})
rule: a blind discovery knows the numbers, not the names, so it leaves the names
for a human. Export it, edit it, `import -csv` it, and it lands in
[config.yaml]({{ '/import.html' | relative_url }}) as a scannable system.

## TrunkRecorder: flattening a map into a stanza

TrunkRecorder models a system more simply than GopherTrunk does — one
`control_channels` array, not a per-site topology — so `writeTrunkRecorder`
flattens the discovery into the shape it expects and is honest in the comment
about what it collapsed:

```go
// internal/hunt/export_trunkrecorder.go (shape)
func writeTrunkRecorder(w io.Writer, sys *DiscoveredSystem) error {
    var ccs []int64
    for _, st := range sys.Sites {
        for _, ch := range st.ControlChannels {
            if ch.IsControl {
                ccs = append(ccs, int64(ch.FrequencyHz)) // all sites → one array
            }
        }
    }
    comment := "Discovered by GopherTrunk hunt. Verify control channels and add recorders/talkgroupsFile before use."
    if len(sys.Sites) > 1 {
        comment += " Multi-site system: control_channels flattens all sites; split per-site if desired."
    }
    // …emit {shortName, type, control_channels, modulation, talkgroupsFile, comment}
}
```

`trProtocol` maps GopherTrunk's spellings onto TrunkRecorder's `type` enum (both
P25 phases → `p25`, all DMR tiers → `dmr`), and `slug` produces a
filesystem-safe `shortName` and a matching `talkgroupsFile` name. A protocol
TrunkRecorder doesn't model at all falls back to the GopherTrunk name rather than
being dropped, so the stanza is always informative even when it needs a manual
edit. The stanza is deliberately a *starting point* — recorders and modulation
tuning are the operator's to add — and the comment says so, because a config that
pretends to be complete is worse than one that flags its own gaps.

## The RadioReference package — and a diff against reality

RadioReference has no public write API; new systems go through a reviewed web
form. So the RR "export" is not an upload — it's a **Markdown submission package**
the operator pastes into that form, with every field laid out and a standing
warning that a blind discovery can't name talkgroups or confirm geography. But the
genuinely clever part is optional and lives one layer up: a cross-reference diff.

```go
// internal/hunt/export.go (shape)
// DiffAgainstRR compares the system's discovered frequencies and talkgroups
// against a flattened RadioReference list. Frequencies are paired greedily by
// nearest unused RR frequency: an exact pair is silent, a pair within
// rrFreqToleranceHz is an offset, and anything farther (or unmatched) is "not in RR".
const rrFreqToleranceHz = 5_000 // below the 6.25 kHz raster, so adjacent channels don't pair
func DiffAgainstRR(sys *DiscoveredSystem, sid int, name string, rrFreqs, rrTGs []uint32) RRDiff
```

This upgrades the RR step from a yes/no duplicate check to a *what's-different*
report. An exact frequency match is silent (nothing to submit). A match within 5
kHz is flagged as a **frequency offset** — "your capture reads 851.0130, RR has
851.0125; check your tuning/PPM." Anything farther, or unmatched, is a frequency
**not in RR** — genuinely new. Talkgroups you've observed that RR hasn't get the
same treatment. The submission package renders all of it under a "Differences vs
RadioReference" header, alongside any `DuplicateHint`s from the read-only RR
lookup, so you contribute what's *new* and correct what's *off* instead of
re-submitting a system that already exists.

<figure class="lab-figure">
<svg viewBox="0 0 660 210" width="660" height="210" role="img" aria-label="One discovered system fanning out to export destinations. In the centre, the DiscoveredSystem passes through the hunt Write seam, which emits four formats: a round-tripping CSV bundle back into GopherTrunk config, a TrunkRecorder JSON stanza, and a RadioReference Markdown submission package carrying an optional cross-reference diff. Separately, the raw capture is exported as a SigMF-tagged gtb bundle for interoperability.">
  <rect x="250" y="86" width="160" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="330" y="104" text-anchor="middle" fill="var(--accent)" font-size="11">DiscoveredSystem</text>
  <text x="330" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="9">hunt.Write(format)</text>
  <line x1="250" y1="100" x2="196" y2="46" stroke="currentColor"/><polygon points="200,49 191,45 200,42" fill="currentColor"/>
  <rect x="40" y="26" width="150" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="115" y="42" text-anchor="middle" fill="currentColor" font-size="10">CSV bundle</text>
  <text x="115" y="54" text-anchor="middle" fill="var(--fg-muted)" font-size="9">→ config.yaml (round-trip)</text>
  <line x1="250" y1="108" x2="196" y2="108" stroke="currentColor"/><polygon points="196,104 187,108 196,112" fill="currentColor"/>
  <rect x="40" y="91" width="150" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="115" y="107" text-anchor="middle" fill="currentColor" font-size="10">TrunkRecorder JSON</text>
  <text x="115" y="119" text-anchor="middle" fill="var(--fg-muted)" font-size="9">flattened CCs</text>
  <line x1="250" y1="116" x2="196" y2="170" stroke="currentColor"/><polygon points="200,166 191,170 200,173" fill="currentColor"/>
  <rect x="40" y="156" width="150" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="115" y="172" text-anchor="middle" fill="var(--accent)" font-size="10">RR submission</text>
  <text x="115" y="185" text-anchor="middle" fill="var(--fg-muted)" font-size="9">+ diff vs reality</text>
  <line x1="410" y1="108" x2="470" y2="108" stroke="var(--fg-muted)" stroke-dasharray="4 3"/><polygon points="470,104 480,108 470,112" fill="var(--fg-muted)"/>
  <rect x="480" y="86" width="150" height="44" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="555" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="10">raw capture</text>
  <text x="555" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="9">SigMF .gtb bundle</text>
  <text x="330" y="204" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the map leaves as config/JSON/Markdown; the measurement leaves as portable IQ</text>
</svg>
<figcaption>One discovered system, four map encodings through the Write seam, plus a separate SigMF-tagged capture bundle so the raw IQ travels too.</figcaption>
</figure>

### How that principle shaped the Go code

- **Formats are data, dispatch is one switch.** Adding an encoding is a `Format`
  constant plus a `writeX` function and a `ParseFormat` alias — the CLI, daemon,
  and cockpit inherit it with no wiring.
- **The bundle is defined by its inverse.** `writeBundle`'s section names and
  columns are pinned to the importer's parser, and a golden test round-trips a
  discovery out and back — so the two can't drift apart silently.
- **The diff is kept out of the package.** `DiffAgainstRR` takes flattened
  frequency/talkgroup slices, not a RadioReference client, so `internal/hunt`
  never imports the RR API — the CLI flattens and passes them in.

## SigMF: making the capture portable

The system map is only half of a find. The other half is the raw IQ it came from —
and a capture is only as useful as it is portable. GopherTrunk's answer is to
package a survey capture into a `.gtb` bundle with a **SigMF** metadata sidecar:

```go
// internal/gtbundle/sigmf.go (shape)
// GopherTrunk does not read SigMF natively, but siglab.Metadata is a near-subset
// of the SigMF global object, so emitting one makes a bundled capture/slice
// interoperable with SigMF-aware tooling (inspectrum, GNU Radio, sigmf-python).
const sigmfVersion = "1.0.0"
// RoleCaptureSigMF → capture/<stem>.sigmf-meta — optional SigMF sidecar
```

The capture-bundle assembler (driven from
[`survey-capture`]({{ '/blog/deep-dives/the-hunt-10-offline-vs-live-surveys/' | relative_url }}))
writes the raw IQ, GopherTrunk's own metadata, a carved narrowband slice, and this
SigMF sidecar into one `.gtb.tar.gz`, tagged with a capture *intent* (cc-map for a
hunt, crypto for a cipher case, survey for a wideband grab). So the artifact that
leaves your machine isn't just "a system I think exists" — it's "a system I think
exists, *and* the exact samples that convinced me, in a format inspectrum can
open." Both the conclusion and the evidence travel. That is the payoff of the
whole [offline/live symmetry]({{ '/blog/deep-dives/the-hunt-10-offline-vs-live-surveys/' | relative_url }}):
a reviewer can re-derive your find from your bytes.

## Where this goes next

Everything so far has been the machinery of a find — engine parts described from
the outside. [Part 14]({{ '/blog/deep-dives/the-hunt-14-cockpit-and-testing/' | relative_url }}),
the finale, closes the loop two ways: the interactive **hunt cockpit** that drives
all of this live from a browser or terminal, and the thing that makes the whole
series trustworthy — testing the entire discovery pipeline with canned IQ sources
and captures, no radio required.

## FAQ

**Which export format should I use?**
Bundle (the default) if you want the find back inside GopherTrunk — it round-trips
into `config.yaml`. TrunkRecorder if another tool will record the system. RR to
document it publicly. Summary for a human-readable network report. They're all the
same `Write` call with a different `Format`.

**Does the CSV bundle really re-import losslessly?**
Yes — its section markers and column names are pinned to the importer's parser,
and it's written through `encoding/csv`, so commas in site names survive. A golden
test round-trips a discovery out and back to keep the two from drifting.

**Why is the RadioReference export a Markdown file, not an upload?**
RadioReference has no public write API; new systems are added through a reviewed
web Submit form. So the export is a submission package you paste in — with a
duplicate check and an optional diff against any existing RR system so you don't
re-submit or mis-tune.

**What does the RR diff actually tell me?**
It pairs your discovered frequencies against an existing RR system: exact matches
are silent, matches within 5 kHz are flagged as tuning/PPM offsets to verify, and
anything unmatched is genuinely new. Talkgroups get the same treatment. It turns
"is this a duplicate?" into "here's exactly what's new or off."

**What is the SigMF bundle for?**
Portability of the raw IQ. GopherTrunk doesn't read SigMF natively, but its
capture metadata is a near-subset of SigMF's, so emitting a `.sigmf-meta` sidecar
inside the `.gtb` bundle lets inspectrum, GNU Radio, and sigmf-python open your
capture. The find and the samples that prove it travel together.

## Series navigation

**Part 13 of 14** · ←
[Part 12: Alias Harvesting — Following Traffic for Talker Aliases]({{ '/blog/deep-dives/the-hunt-12-alias-harvesting/' | relative_url }})
· Next →
[Part 14: The Hunt Cockpit & Testing Discovery Without Radios]({{ '/blog/deep-dives/the-hunt-14-cockpit-and-testing/' | relative_url }})
