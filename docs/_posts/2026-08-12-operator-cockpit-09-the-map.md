---
title: "The Operator's Cockpit, Part 9: The Map — Plotting Sites & Emitters"
description: How GopherTrunk plots P25 sites and position-bearing emitters — APRS, AIS, ADS-B, DSC — on one shared Leaflet map, patching markers in place from the daemon's locations and sites REST endpoints as fixes arrive, with an auto-fitting camera and a merged discovered-plus-configured site table.
category: deep-dives
keywords: leaflet sdr map, aprs ais adsb map, p25 site map, live position markers, circlemarker patch in place, discovered sites merge, openstreetmap tiles operator, react map component, location fixes rest, gophertrunk operator cockpit
tags: [operator-cockpit, map, leaflet, react, rest, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cockpit"
series_part: 9
---

*Part 9 of **The Operator's Cockpit**, the series on driving one GopherTrunk
daemon through one REST + SSE API from a browser and a terminal alike. Parts 7
and 8 painted the RF — spectrum, waterfall, constellations. This post steps out
to the most literally *situational* panel in the cockpit: a map that plots where
things are, from P25 sites the control channel named to aircraft, vessels, and
weather-alert beacons the decoders heard.*

> **TL;DR:** One reusable `<PositionMap>` React component drives every
> position-bearing panel — APRS stations, AIS vessels, ADS-B aircraft, DSC
> distress alerts. It's a Leaflet map fed a flat `MapPoint[]`; on every update it
> **patches `CircleMarker`s in place** (keyed by a stable `id`) rather than
> tearing them down, colours them by kind, and auto-fits the camera to the live
> points. The data comes from plain REST: `GET /api/v1/locations` for recent
> fixes and `GET /api/v1/sites` for the union of *discovered* P25 sites (with
> live control-channel frequency) and *configured* site names. Same daemon, same
> API contract — the map is just another renderer over it.

**Key takeaways**

- **One map component, many panels.** `<PositionMap points=… />` is shared by
  every geographic panel; the points array is the entire contract, so a new
  emitter type is a new caller, not a new map.
- **Markers patch, they don't rebuild.** A per-row `id` keys a marker dictionary,
  so a poll updates position/colour/tooltip in place and only truly-gone markers
  are removed — smooth motion, no flicker.
- **Sites merge two sources of truth.** `GET /api/v1/sites` unions what the
  control channel *discovered* (live CC frequency, WACN/SystemID) with what the
  operator *named* in config, so a site's name always shows even before it's heard.
- **The daemon degrades gracefully.** Every location/affiliation/site handler
  returns a stable, empty JSON shape when its subsystem isn't wired, so the UI
  renders a consistent skeleton instead of erroring.

## Cheat sheet

| Piece | What it does | Where it lives |
|---|---|---|
| Map component | Leaflet map over a flat `MapPoint[]` | `web/src/components/PositionMap.tsx` |
| Marker patch | keyed `CircleMarker` update-in-place | `PositionMap.tsx` (points effect) |
| Locations API | recent geographic fixes | `internal/api/handlers_locations.go` (`handleLocations`) |
| Affiliations API | unit-on-talkgroup activity table | `handlers_locations.go` (`handleAffiliations`) |
| Sites API | discovered ∪ configured P25 sites | `internal/api/handlers_sites.go` (`handleListSites`) |
| Kind → style | colour + radius per emitter class | `PositionMap.tsx` (`KIND_COLOR` / `KIND_RADIUS`) |

## In this post

- **One map, every emitter** — the shared component and its `MapPoint` contract.
- **Patch, don't rebuild** — why markers are keyed and updated in place.
- **The locations endpoint** — recent fixes, and the empty-shape guarantee.
- **The sites endpoint** — merging discovered CC data with configured names.
- **Tiles and the single-operator assumption** — the OSM usage note.

## One map, every emitter

GopherTrunk decodes several protocols that carry positions — APRS station fixes,
AIS vessel tracks, ADS-B aircraft, DSC distress alerts. Rather than write a map
four times, there's one component and one contract. A panel hands it a flat array
of points; that's the whole interface:

```tsx
// web/src/components/PositionMap.tsx (shape)
export interface MapPoint {
  id: string;               // stable per-row identity → React key for the marker
  latitude: number;
  longitude: number;
  kind: "aprs" | "ais" | "adsb" | "dsc-distress" | "default"; // → colour + radius
  label: string;            // tooltip: bold first line
  detail?: string;          // optional second line
}

export function PositionMap({ points, heightPx = 360, fallbackCenter = [37.5, -122.0] }: PositionMapProps) {
  // …lazy-init Leaflet + OSM tiles once; sync markers on every `points` change
}
```

The `kind` field is the only styling channel: it maps to a fill colour and radius
so an operator can read the map at a glance — blue APRS stations, cyan vessels,
purple aircraft, and an emphatic red, larger dot for a DSC distress alert.

```tsx
// web/src/components/PositionMap.tsx (shape)
const KIND_COLOR = {
  aprs: "#3b82f6", ais: "#06b6d4", adsb: "#a855f7",
  "dsc-distress": "#ef4444", default: "#6b7280",
};
const KIND_RADIUS = { aprs: 5, ais: 5, adsb: 6, "dsc-distress": 8, default: 4 };
```

Because the points array *is* the contract, adding a new emitter type to the map
is adding a caller that produces `MapPoint`s — never a change to the map itself.
That's the same "one renderer, many drivers" discipline the DSP scopes followed
in Part 8, applied to geography.

## Patch, don't rebuild

The map polls. Left naive, that means throwing away every marker and re-creating
it several times a second — visible flicker, lost tooltips, jumpy motion. So the
component keeps a *dictionary* of live markers keyed by each row's `id`, and every
update diffs against it: existing markers move and restyle in place, new ones are
created, and only genuinely-absent ones are removed.

```tsx
// web/src/components/PositionMap.tsx (shape) — the points effect
const markers = markersRef.current; // Map<string, L.CircleMarker>
const seen = new Set<string>();
for (const p of points) {
  seen.add(p.id);
  const existing = markers.get(p.id);
  const latLng = [p.latitude, p.longitude];
  if (existing) {
    existing.setLatLng(latLng);                       // move in place
    existing.setStyle({ fillColor: KIND_COLOR[p.kind] });
    existing.bindTooltip(tooltipHTML(p), { direction: "top" });
  } else {
    const m = L.circleMarker(latLng, { radius: KIND_RADIUS[p.kind], /* …outline, opacity */ });
    m.bindTooltip(tooltipHTML(p), { direction: "top" }).addTo(map);
    markers.set(p.id, m);
  }
}
for (const [id, m] of markers.entries()) {
  if (!seen.has(id)) { m.remove(); markers.delete(id); } // reap the gone
}
```

After the diff it fits the camera to whatever's live, with a little padding so
markers don't pin to the edges:

```tsx
// web/src/components/PositionMap.tsx (shape)
if (points.length > 0) {
  const bounds = L.latLngBounds(points.map((p) => [p.latitude, p.longitude]));
  map.fitBounds(bounds, { padding: [40, 40], maxZoom: 12 });
}
```

The stable `id` is what makes all of this possible — it's why the same aircraft
keeps its marker (and its open tooltip) as its position updates, instead of
blinking as a new object every poll. Tooltip HTML is escaped, because labels can
carry arbitrary decoded text (a vessel name, a callsign) and a map marker is not a
place to trust input.

<figure class="lab-figure">
<svg viewBox="0 0 660 196" width="660" height="196" role="img" aria-label="The map's update-in-place loop. A poll produces a new MapPoint array. For each point, if a marker with that id already exists it is moved and restyled in place; otherwise a new CircleMarker is created and added. Markers whose id is not in the new set are removed. Finally the camera auto-fits to the live points.">
  <rect x="10" y="78" width="120" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="70" y="97" text-anchor="middle" fill="var(--accent)" font-size="10">poll → points[]</text>
  <text x="70" y="111" text-anchor="middle" fill="var(--fg-muted)" font-size="8">each has an id</text>
  <line x1="130" y1="99" x2="166" y2="99" stroke="currentColor"/><polygon points="166,95 176,99 166,103" fill="currentColor"/>
  <rect x="176" y="42" width="150" height="36" rx="6" fill="none" stroke="currentColor"/>
  <text x="251" y="60" text-anchor="middle" fill="currentColor" font-size="9">id in dict?</text>
  <text x="251" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="8">yes → move + restyle</text>
  <rect x="176" y="86" width="150" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="251" y="104" text-anchor="middle" fill="currentColor" font-size="9">no → create marker</text>
  <rect x="176" y="128" width="150" height="34" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="251" y="146" text-anchor="middle" fill="var(--fg-muted)" font-size="9">gone → remove</text>
  <line x1="326" y1="99" x2="372" y2="99" stroke="currentColor"/><polygon points="372,95 382,99 372,103" fill="currentColor"/>
  <rect x="382" y="80" width="140" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="452" y="98" text-anchor="middle" fill="var(--accent)" font-size="10">fitBounds()</text>
  <text x="452" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="8">camera auto-fit</text>
  <text x="540" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="9">smooth,</text>
  <text x="540" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">no flicker</text>
</svg>
<figcaption>Every poll diffs against a keyed marker dictionary — move, create, reap — then the camera fits the live set. Markers persist; only the map's contents change.</figcaption>
</figure>

## The locations endpoint

The fixes come from plain REST. `GET /api/v1/locations` returns recent geographic
fixes, and its most important property is that it *always* returns a stable shape
— an empty list when the subsystem isn't wired — so the UI never has to special-
case "the daemon can't do locations":

```go
// internal/api/handlers_locations.go (shape)
func (s *Server) handleLocations(w http.ResponseWriter, r *http.Request) {
    if s.locations == nil { // no storage → stable empty shape, still 200
        writeJSON(w, http.StatusOK, map[string]any{"locations": []LocationFix{}})
        return
    }
    limit := 500 // …overridable via ?limit=
    fixes, err := s.locations.RecentLocations(limit)
    if err != nil { s.writeError(w, http.StatusInternalServerError, "query locations: "+err.Error()); return }
    if fixes == nil { fixes = []LocationFix{} } // never null on the wire
    writeJSON(w, http.StatusOK, map[string]any{"locations": fixes})
}
```

The sibling `handleAffiliations` follows the same rule for the unit-on-talkgroup
activity table — always `200`, an empty slice when the tracker isn't wired. This
"empty, not absent" contract is what lets the map (and the panels around it) render
a consistent skeleton on a fresh daemon and simply fill in as data arrives.

## The sites endpoint: two truths, merged

P25 sites are a richer case, because there are two independent sources of truth
and the endpoint's whole job is to reconcile them. `GET /api/v1/sites`
returns the **union** of sites GopherTrunk *discovered* from the control channel
— each carrying the live control-channel frequency and network identity — and the
sites the operator *named* in config. Discovered rows get their name merged on;
configured-but-unheard sites are appended with the frequency omitted, so a name
always shows:

```go
// internal/api/handlers_sites.go (shape)
byKey := make(map[key]*SiteDTO) // key = (system, RFSS, site)

// 1) Discovered sites first — live CC frequency + WACN/SystemID.
for _, si := range s.sites.Sites() {
    byKey[key{si.System, si.RFSSID, si.SiteID}] = &SiteDTO{
        System: si.System, RFSSID: si.RFSSID, SiteID: si.SiteID,
        ControlChannelHz: si.ControlChannelHz, WACN: si.WACN, SystemID: si.SystemID,
        // …per-site TSBK decode quality, only once a TSBK has actually been tried
    }
}

// 2) Merge operator-supplied names; append configured-but-unseen sites.
for _, sys := range s.systems {
    for _, cs := range sys.Sites {
        k := key{sys.Name, cs.RFSS, cs.Site}
        if dto, ok := byKey[k]; ok { dto.Name = cs.Name; continue }
        byKey[k] = &SiteDTO{System: sys.Name, RFSSID: cs.RFSS, SiteID: cs.Site, Name: cs.Name}
    }
}
```

Two details reward a second look. Per-site decode quality (TSBK error rate) is
attached **only** once a TSBK has actually been attempted, so a fresh lock reports
*no* quality rather than a misleading 0%. And each system's advertised neighbour
sites are overlaid onto exactly the row that broadcast the adjacent-site list —
keyed by the topology snapshot's own RFSS/Site — not smeared across every site of
the system. The result is one sorted `SiteDTO[]`, each row a merge of everything
GopherTrunk knows about that site from either the air or the config.

## Tiles and the single-operator assumption

The map draws OpenStreetMap standard tiles from the public tile server, and the
component is honest about the tradeoff in its own comments: this is fine for a
single self-hosted operator console under the OSM tile usage policy, but power
users running large fleets of daemons should point it at their own tile cache. The
zoom is clamped (`minZoom: 2`, `maxZoom: 18`) precisely so a quick pan can't yank
thousands of tiles. It's the same design posture as the rest of GopherTrunk — a
single-user tool that assumes one operator, one radio, one console — expressed
here as a tile policy.

## Where this goes next

[Part 10]({{ '/blog/deep-dives/operator-cockpit-10-write-mode/' | relative_url }})
turns from *reading* the daemon to *changing* it. Every panel so far has been a
renderer over read-only endpoints; the next post is about the write side — the
validate → stage → activate discipline, the mutation-capability gate, and the
guardrails that let you retune, lock out a talkgroup, or hot-swap a config file
from the browser without the daemon lying to you about what took effect.

## FAQ

**Is the map fed by SSE or by polling?**
By polling the REST endpoints (`/api/v1/locations`, `/api/v1/sites`) — the
position feeds update on a cadence, and the marker-patching keeps that visually
smooth. SSE carries the bus events the *rest* of the cockpit reacts to; the map's
data is a periodic snapshot.

**Why patch markers instead of re-rendering them?**
A keyed marker dictionary keeps each object alive across polls, so a moving
emitter glides and keeps its open tooltip instead of blinking as a brand-new
marker. Only markers whose `id` disappears are removed. It's smoother and cheaper.

**What's the difference between a discovered and a configured site?**
Discovered sites are ones GopherTrunk heard on the control channel — they carry a
live CC frequency and the network identity (WACN/SystemID). Configured sites are
names you supplied in config. `GET /api/v1/sites` merges them so a named site
shows its name even before it's been heard, and a heard site shows live data even
before it's been named.

**Does the map need the location subsystem wired?**
No — every handler returns a stable empty shape (still `200`) when its subsystem
isn't present. The panel renders its skeleton and simply fills in when fixes or
sites start arriving.

**Can I use my own map tiles?**
Not through config yet — the component notes it as a follow-up. Today it uses OSM
standard tiles, which the usage policy permits for a single self-hosted operator;
large fleets should front their own tile cache.

## Series navigation

**Part 9 of 14** · ←
[Part 8: Constellation, Eye Diagram & Symbol Scope]({{ '/blog/deep-dives/operator-cockpit-08-constellation-eye-symbol/' | relative_url }})
· Next →
[Part 10: Write Mode — Mutating Config Safely]({{ '/blog/deep-dives/operator-cockpit-10-write-mode/' | relative_url }})
