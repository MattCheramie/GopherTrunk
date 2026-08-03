---
title: "The Hunt, Part 11: Naming the Unknown"
description: How GopherTrunk turns a blind discovery's bare numbers into names — a synthesized system handle from the protocol and identity, a Name/Service/Purpose triple from the frequency allocation and reference catalog, and the honest line between what a decoder can name and what only an operator can.
category: deep-dives
keywords: naming trunked system, talkgroup aliasing, signal reference catalog, frequency allocation lookup, discovered system displayname, blind discovery naming, sigref, gophertrunk the hunt
tags: [the-hunt, naming, trunking, survey, metadata, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Hunt"
series_part: 11
---

*Part 11 of **The Hunt**. Our carrier is fully mapped now — sites, control
channels, a band plan, a fistful of talkgroups — and it reproduces from a
recording. What it still lacks is a **name**. The system prints as
`Unknown-p25-BEE00-2A9`; every talkgroup is a bare decimal. This post is about
the last, most human step: turning those numbers into labels. It is also the
most honest one in the series, because naming a blind discovery is exactly as
much as the signal actually tells you — and not one word more.*

> **TL;DR:** A discovery arrives as numbers, and GopherTrunk names as much of it
> as the RF can justify. A system gets a **synthesized handle** from its protocol
> and identity (`Unknown-p25-<WACN>-<SYSID>`) until an operator assigns a real
> `Name`. Every classified carrier gets a **Name / Service / Purpose** triple —
> `Service`/`Purpose` from the frequency allocation table, `Name` from the
> decoded protocol, else a best-guess from the signal reference catalog, else the
> modulation class. But talkgroups stay **numeric**: a blind decode sees a
> talkgroup's id and activity, never its meaning, so those fields are left blank
> for the operator or RadioReference to fill. Naming is aliasing, and the honesty
> is knowing where aliasing stops.

**Key takeaways**

- **A system always has a stable handle.** `DisplayName()` returns the operator's
  `Name` when set, otherwise a synthesized `Unknown-<proto>-<WACN>-<SYSID>` — so
  an unnamed discovery still round-trips and diffs cleanly.
- **Every carrier is named three ways.** `nameSignal` fills `Name` (what it is),
  `Service` (what the frequency is allocated to), and `Purpose` (what that
  service does) — a consolidated "what is this" for decodable and undecodable
  signals alike.
- **Naming falls back, never guesses wildly.** Decoded protocol → reference-catalog
  best guess (above a score floor) → modulation class. An undecoded carrier only
  earns a catalog name when the match clears `nameRefFloor` (0.40).
- **Talkgroups are deliberately unnamed.** A blind discovery knows a talkgroup's
  decimal, hex, and activity count — never its alpha tag — so those fields ship
  blank for a human or RR to complete.

## Cheat sheet

| Piece | What it does | Where it lives |
|---|---|---|
| `nameSignal` | fill Name / Service / Purpose for any carrier | `internal/hunt/naming.go` |
| `DisplayName` | operator name, else a synthesized handle | `internal/hunt/system.go` |
| `DiscoveredTalkgroup` | numeric id + activity, descriptive fields blank | `internal/hunt/system.go` |
| `sigref.Lookup` / `.Rank` | frequency allocation + catalog best guess | `internal/siglab/sigref` |
| `NetworkReport` | flatten the named system for the shared renderer | `internal/hunt/report.go` |
| `modClassFor` | survey class → blind modulation class for ranking | `internal/hunt/naming.go` |

## In this post

- **The two things a discovery is missing** — a system name and talkgroup names.
- **Naming a system** — the synthesized handle and why it's shaped that way.
- **Naming a carrier** — the Name / Service / Purpose triple and its fallbacks.
- **Why talkgroups stay numeric** — the honest limit of a blind decode.
- **Rendering the named system** — flattening it for the report.

## The two things a discovery is missing

Walk back through the series and notice what naming is *not*. The
[sweep]({{ '/blog/deep-dives/the-hunt-02-wideband-sweep-engine/' | relative_url }})
gave us frequencies. The [identify]({{ '/blog/deep-dives/the-hunt-04-classifying-a-signal/' | relative_url }})
gave us a protocol. The [map]({{ '/blog/deep-dives/the-hunt-01-what-discovery-means/' | relative_url }})
gave us identity fields — WACN, System ID, NAC — and a talkgroup list. All of
that is *measured*: it came off the air as bits. Names don't come off the air.
There is no RF field that says "this is the county fire dispatch system" or "this
talkgroup is Engine 12." Naming is the layer where measured numbers meet a
**human catalog**, and the discipline is to name only what the catalog can
actually justify from what we measured.

Two things want names: the system as a whole, and each talkgroup. GopherTrunk
handles them very differently, and the difference is the whole lesson.

## Naming a system

A `DiscoveredSystem` always needs *a* label — golden tests diff on it, exports
name files after it, the cockpit lists it. So `DisplayName` never returns empty.
It returns the operator's `Name` if one was assigned, and otherwise synthesizes a
stable handle from the identity we did measure:

```go
// internal/hunt/system.go (shape)
func (s *DiscoveredSystem) DisplayName() string {
    if s.Name != "" {
        return s.Name
    }
    proto := s.Protocol
    if proto == "" {
        proto = "unknown"
    }
    switch {
    case s.WACN != 0 && s.SystemID != 0:
        return fmt.Sprintf("Unknown-%s-%05X-%03X", proto, s.WACN, s.SystemID)
    case s.SystemID != 0:
        return fmt.Sprintf("Unknown-%s-%03X", proto, s.SystemID)
    default:
        return fmt.Sprintf("Unknown-%s", proto)
    }
}
```

Three things are true about this handle and all three are deliberate. It is
**stable** — the same system produces the same string every run, so it diffs and
re-imports without churn. It is **honest** — the leading `Unknown-` says nobody
has confirmed a real name yet. And it **degrades** — with full P25 identity you
get `Unknown-p25-BEE00-2A9`; with only a system id you get `Unknown-p25-2A9`; with
neither, just `Unknown-p25`. It carries exactly as much identity as we have and
no more. The moment an operator (or a RadioReference match) supplies a real name,
`Name` is set and the synthesized handle disappears — but until then, the system
has a stable, self-describing identity.

## Naming a carrier: Name, Service, Purpose

Down at the individual-carrier level — the survey inventory, every detected
signal — naming is richer, because a frequency is not just a number: it sits in an
**allocation table**. `nameSignal` fills a three-part label on every
`DetectedSignal`, decodable or not:

```go
// internal/hunt/naming.go (shape)
// nameSignal fills the consolidated inventory fields — Name, Service, Purpose —
// for any signal. Service/Purpose come from the frequency allocation; Name
// prefers the decoded protocol, then a best-guess from the reference catalog,
// then the modulation class.
func nameSignal(ds *DetectedSignal) {
    if a, ok := sigref.Lookup(ds.FreqHz); ok {
        ds.Service = a.Service   // what the frequency is allocated to
        ds.Purpose = a.Purpose   // what that service is used for
    }
    switch {
    case ds.Trunking != nil && ds.Trunking.Protocol != "":
        // decoded → the protocol's display name (P25, DMR, …)
    case len(ds.Pages) > 0:
        ds.Name = ds.Pages[0].Protocol
    default:
        // undecoded → rank against the signal reference catalog on baud,
        // modulation, bandwidth, and centre; take the top match only if it
        // clears the keep floor, else fall back to the modulation class.
    }
}
```

The precedence is the interesting part. `Service` and `Purpose` come from *where
the carrier lives* — the frequency allocation, which is known for a band whether
or not we decoded anything. `Name` comes from *what the carrier is*, and it walks
a strict ladder: a decoded protocol names itself outright; a paging carrier is
named by its page protocol; and an **undecoded** carrier is ranked against the
signal reference catalog on its measured features (baud, modulation, bandwidth,
centre) and named only if the best match clears a floor:

```go
// internal/hunt/naming.go (shape)
const nameRefFloor = 0.40 // below this, name by modulation class instead
if m := sigref.Rank(obs, 1); len(m) > 0 && m[0].Score >= nameRefFloor {
    ds.Name = m[0].Entry.DisplayName
    if ds.Confidence == 0 { // fill confidence for an undecoded/wideband row
        ds.Confidence = m[0].Score
    }
} else if ds.Wideband {
    ds.Name = "wideband signal"
} else {
    ds.Name = string(ds.Class) // e.g. "nbfm" — the honest floor
}
```

That floor is the guard against confident nonsense. A weak, ambiguous carrier
whose best catalog match scores 0.2 is *not* named "P25 control" — it is named
`nbfm`, its modulation class, which is all we actually know. Naming here is a best
guess with a confidence attached, and below the floor the honest label is the
mechanism, not the meaning.

<figure class="lab-figure">
<svg viewBox="0 0 640 220" width="640" height="220" role="img" aria-label="The carrier naming ladder. A carrier's Service and Purpose come from a frequency allocation lookup. Its Name is chosen by precedence: a decoded protocol names itself; else a paging carrier is named by its page protocol; else the signal is ranked against the reference catalog and named by the top match only if its score clears the 0.40 floor; else it falls back to its modulation class.">
  <rect x="10" y="90" width="120" height="40" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="70" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="10">freq allocation</text>
  <text x="70" y="120" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Service · Purpose</text>
  <line x1="130" y1="110" x2="160" y2="110" stroke="currentColor"/><polygon points="160,106 170,110 160,114" fill="currentColor"/>
  <rect x="170" y="18" width="180" height="30" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="260" y="37" text-anchor="middle" fill="var(--accent)" font-size="10">decoded protocol → Name</text>
  <rect x="170" y="58" width="180" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="260" y="77" text-anchor="middle" fill="currentColor" font-size="10">paging → page protocol</text>
  <rect x="170" y="98" width="180" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="260" y="114" text-anchor="middle" fill="currentColor" font-size="10">catalog rank ≥ 0.40</text>
  <text x="260" y="125" text-anchor="middle" fill="var(--fg-muted)" font-size="8">best guess + confidence</text>
  <rect x="170" y="138" width="180" height="30" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="260" y="157" text-anchor="middle" fill="var(--fg-muted)" font-size="10">else → modulation class</text>
  <line x1="350" y1="33" x2="470" y2="86" stroke="currentColor"/><polygon points="466,83 476,87 467,92" fill="currentColor"/>
  <line x1="350" y1="73" x2="470" y2="90" stroke="currentColor"/><polygon points="466,87 476,91 466,95" fill="currentColor"/>
  <line x1="350" y1="113" x2="470" y2="98" stroke="currentColor"/><polygon points="466,94 476,98 466,102" fill="currentColor"/>
  <line x1="350" y1="153" x2="470" y2="106" stroke="currentColor"/><polygon points="467,101 476,106 466,110" fill="currentColor"/>
  <rect x="476" y="80" width="150" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="551" y="98" text-anchor="middle" fill="var(--accent)" font-size="11">Name</text>
  <text x="551" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">first rule that fires wins</text>
  <text x="320" y="200" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the ladder names as much as the measurement justifies, then stops at the modulation class</text>
</svg>
<figcaption>Every carrier gets a Service/Purpose from its frequency and a Name from a strict precedence ladder — decoded, paged, catalog-ranked above a floor, or the honest modulation class.</figcaption>
</figure>

## Why talkgroups stay numeric

Here is where naming refuses to overreach. A `DiscoveredTalkgroup` carries a
decimal, a hex, an encrypted flag, an activity count, and a first-seen time — and
nothing else:

```go
// internal/hunt/system.go (shape)
// DiscoveredTalkgroup is one talkgroup observed on the control channel. On a
// blind discovery only the numeric id and activity are known; the descriptive
// fields are left blank for the operator (or RR) to fill in.
type DiscoveredTalkgroup struct {
    Dec       uint32    `json:"dec"`
    Hex       string    `json:"hex"`
    Encrypted bool      `json:"encrypted,omitempty"`
    Count     int       `json:"count"`
    FirstSeen time.Time `json:"first_seen"`
}
```

There is no `AlphaTag`, no `Description`, no `Group` — because a talkgroup's
*name* is nowhere in the RF. The control channel broadcasts that talkgroup 101 was
granted a channel; it never broadcasts that 101 is "Fire Dispatch." That mapping
lives in a human database, and inventing it would be lying. So the exporter emits
those columns blank for the operator or RadioReference to fill — the same reason
the RR submission package says, right at the top, "a blind discovery cannot name
talkgroups." The system counts a talkgroup's activity so you know which ids matter
most; it will not pretend to know what they are.

This is also why the [Trunking Engine]({{ '/blog/series/trunking-engine/' | relative_url }})
and [RadioReference import]({{ '/import.html' | relative_url }}) exist on the other
side: naming is the seam where an operator's knowledge joins the machine's
measurements. GopherTrunk names what it can prove and leaves a clearly-marked
blank for what it can't.

### How that principle shaped the Go code

- **`nameSignal` never overrides.** It only fills fields the router left unset and
  is safe to call once per signal, so a decoded protocol's own name always wins
  over a catalog guess.
- **The floor is a constant, not a mood.** `nameRefFloor` (0.40) is the single
  knob separating "named best guess" from "modulation class," matching the
  wideband survey's keep floor so the two agree.
- **Missing identity degrades the handle, it doesn't crash it.** `DisplayName`'s
  switch means a system with partial identity still produces a usable, stable
  string — the exporters never see an empty name.

## Rendering the named system

Once the system carries whatever names it has earned, `NetworkReport` flattens it
into the protocol-neutral `trunking.NetworkReport` the shared renderer consumes —
one `ReportSite` per discovered site under a shared identity header and band plan,
with neighbour frequencies resolved from the accumulated band plan:

```go
// internal/hunt/report.go (shape)
func (s *DiscoveredSystem) NetworkReport() trunking.NetworkReport {
    r := trunking.NetworkReport{
        Name:     s.DisplayName(), // the named (or synthesized) handle
        Protocol: s.Protocol, WACN: s.WACN, SystemID: uint32(s.SystemID), NAC: s.NAC,
    }
    // …one ReportSite per site: primary CC, secondaries, neighbours (with
    //   uplink derived from the band's TX offset), then the band plan
    return r
}
```

This is the same `NetworkReport` the live daemon renders for a configured system,
so a discovered system's summary reads exactly like a known one's — just with an
`Unknown-` handle and blank talkgroup names where a human hasn't filled them in
yet. The report is honest about its gaps by construction.

## Where this goes next

Talkgroups have numbers but no names — and the units riding them are the same
story. But there is one place a real, human name *does* travel over the air: the
**talker alias**, the display name a radio broadcasts for itself. [Part 12]({{ '/blog/deep-dives/the-hunt-12-alias-harvesting/' | relative_url }})
is about harvesting those aliases off a system's traffic channels *without*
following the voice — the one naming the RF actually hands you.

## FAQ

**Why does an unnamed system print `Unknown-p25-…` instead of blank?**
Because everything downstream needs a stable handle — golden diffs, export
filenames, the cockpit list. `DisplayName` synthesizes one from the measured
identity (WACN/SystemID), prefixed `Unknown-` to say it's unconfirmed. Assign a
real `Name` and the synthesized handle vanishes.

**Where do a carrier's Service and Purpose come from?**
The frequency allocation table (`sigref.Lookup`). They describe what the
*frequency* is allocated to and what that service does — known for a band whether
or not the carrier decoded — so even an unidentified signal shows a meaningful
band context.

**Why won't GopherTrunk name my talkgroups?**
Because the name isn't in the RF. The control channel says talkgroup 101 was
granted a channel; it never says 101 is "Fire Dispatch." That mapping lives in a
human database, so the discovery ships the id, hex, and activity count and leaves
the descriptive fields blank for you or RadioReference to complete.

**What is `nameRefFloor` protecting against?**
Confident nonsense. An undecoded carrier is only named from the reference catalog
when its best match scores at least 0.40; below that it's named by its modulation
class (`nbfm`, `psk`, …), which is all we actually measured. The floor keeps a
weak guess from masquerading as a firm identification.

**Does naming change the decode at all?**
No. Naming is a pure labelling pass over already-measured results — `nameSignal`
runs once per signal before it's stored, and never touches the DSP or the
accumulation. It's the human-catalog layer on top of the measurements, not part
of them.

## Series navigation

**Part 11 of 14** · ←
[Part 10: Offline vs Live Surveys — Hunting a Recording]({{ '/blog/deep-dives/the-hunt-10-offline-vs-live-surveys/' | relative_url }})
· Next →
[Part 12: Alias Harvesting — Following Traffic for Talker Aliases]({{ '/blog/deep-dives/the-hunt-12-alias-harvesting/' | relative_url }})
