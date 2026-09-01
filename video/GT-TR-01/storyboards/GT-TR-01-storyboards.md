# GT-TR-01 — Storyboards (pilot, 5 segments + pillar elements)

Beat tables per segment. `visual` names map to components in
`video/brand/engine.js`; timings come from the measured narration (VO-first),
so beat times here are indicative, not authoritative — the timeline JSON in
`transcripts/` is the source of truth. All must-read content stays inside the
center 9:16 zone (x 690–1230 at 1080p); every visual has a `vert` re-layout in
the engine (stacked, no blind crops).

## GT-TR-01.01 — Trunked radio  (article SVG: channel pool + grant arrow)

| Beat | Visual | What happens on screen |
|---|---|---|
| Title | `title` | Card: TRUNKED RADIO, eyebrow, seg ID, domain stripe. Hard cut in/out. |
| Hook | `conv-vs-trunk intro` | Two-column compare builds: left "conventional" rows Police/Fire/Works each pinned to a channel; right side dark. |
| Idle waste | `conv-vs-trunk idle` | Left rows mostly dim (silent), brief random flickers of activity; "30 channels, ~4 busy" annotation. |
| Pool grant | `pool grant` | Article SVG animated: CC bar draws on top, request ping rises, grant arrow dashes down to voice 3, TG 101 lights. |
| Release | `pool release` | Voice 3 dims back to pool; "assigned per call, then released" caption line. |
| Erlang | `defcard erlang` | Definition card: grade of service. |
| Stats | `pool stats` | Pool serves stream of calls: channels light up/down rapidly, counter "dozens of talkgroups / 5 channels". |
| Variants | `defcard variants` | Card: FDMA · TDMA · single-site · multisite. |
| Tie-in | `gt-tiein` | GopherTrunk chip: CC decode → follow arrows to calls. |
| Recap | `recap` | 3 bullets staged; footer URL `gophertrunk.org/reference/trunked-radio/`. |

## GT-TR-01.02 — Control channel  (article SVG: always-on stripe + message ticks)

| Beat | Visual | On screen |
|---|---|---|
| Title | `title` | CONTROL CHANNEL card. |
| Hook | `ccstream intro` | Continuous data stripe draws across; waveform-like tick stream scrolls inside; "never voice" label. |
| Messages | `ccstream messages` | Ticks get labels as they pass: registration, grant → ch 3, affiliation. |
| Roles | `ccpair roles` | Split: control (continuous) vs voice channel (short-lived burst + hang time). |
| Grant flow | `ccstream grantflow` | Back-to-back TSBK/CSBK blocks; sysinfo + neighbor list blocks between grants. |
| Late entry | `ccstream lateentry` | Repeating "update" blocks; a radio joins mid-call. |
| Variants | `defcard variants` | dedicated · rotating rest channel · composite. |
| Tie-in | `gt-tiein` | GT camps on CC, tasks receivers per grant. |
| Recap | `recap` | 3 bullets + URL. |

## GT-TR-01.03 — Talkgroup  (article SVG: calls hopping across channels)

| Beat | Visual | On screen |
|---|---|---|
| Title | `title` | TALKGROUP card. |
| Hook | `tghop intro` | "Talkgroup 101 (one virtual channel)" header; call 1 box appears on ch 3. |
| Follow | `tghop follow` | Calls 2, 3 appear on ch 7, ch 2; a highlight ring follows the TG, not the row. |
| Membership | `tgprog membership` | Radio programming list; keyed TG broadcasts to affiliated radios together. |
| TGID | `defcard tgid` | Card: 16-bit, WACN/SysID scope, alias tables. |
| Radio ID | `rid pair` | Two chips: TG 101 (the many) + RID 4567 (the one). |
| Variants | `defcard variants` | dispatch · tactical · patched · announcement. |
| Tie-in | `gt-tiein` | GT call row: TG + alias + RID; allow/priority lists. |
| Recap | `recap` | 3 bullets + URL. |

## GT-TR-01.04 — Channel grant  (article SVG: CC → grant msg → radios retune)

| Beat | Visual | On screen |
|---|---|---|
| Title | `title` | CHANNEL GRANT card. |
| Hook | `grantflow intro` | Article SVG: CC box, grant message travels the arrow, radios box. |
| Retune | `grantflow retune` | Radio icons snap to ch 3 together; monitor icon follows. |
| Anatomy | `grantfields anatomy` | Message dissected into 3 fields: target / channel # (→ freq via plan) / slot. |
| Updates | `grantupdate repeat` | Timeline: grant, then repeating update blocks; late radio joins mid-call. |
| Variants | `defcard variants` | group · private · update · data. |
| Queue | `grantqueue busy` | All channels busy → request queued → delayed grant pops. |
| Tie-in | `gt-tiein` | GT: parse grant → channel plan → tune receiver + slot. |
| Recap | `recap` | 3 bullets + URL. |

## GT-TR-01.05 — FDMA & TDMA  (article SVGs: stacked channels; slot alternation)

| Beat | Visual | On screen |
|---|---|---|
| Title | `title` | FDMA & TDMA card. |
| FDMA | `fdmastack intro` | Frequency axis; 4 stacked channel bars each "call A–D", guard gaps. |
| Narrowband | `fdmastack narrowband` | Bars narrow: 25 kHz → 12.5 → 6.25 with protocol labels. |
| TDMA | `tdmaslots intro` | One channel bar splits into alternating slots 1/2 filling along time axis. |
| Timing | `tdmaslots timing` | Guard time gaps highlighted; sync pattern tick; DMR ×2 / TETRA ×4 labels. |
| Compare | `accesscompare table` | Side-by-side: grant names freq vs freq+slot. |
| Tie-in | `gt-tiein` | GT: DDC a carrier (FDMA) vs sync + slot extract (TDMA). |
| Recap | `recap` | 3 bullets + URL. |

## Pillar elements

- **Cold open** (`coldopen city`, `pool grant`, `ccstream intro`, `coldopen title`):
  dark radio-tower pulse scene → best-of pool grant → CC stripe → course title card
  "TRUNKED RADIO — the Field Guide course".
- **Intro** (`introcard`, `agenda list`): agenda list reveals 5 segment titles.
- **Transitions** (`transit "A → B"`): 10–20 s bridge card, A dims, B brightens.
- **Outro** (`outrosum`, `endslate`): five idea-chips interlock, CTA, 20 s end slate.

## Vertical re-layouts (flagged wide diagrams)

- `conv-vs-trunk`: columns stack vertically (conventional on top, trunked below).
- `ccpair`, `accesscompare`, `rid`: side-by-side → stacked.
- `pool`, `ccstream`, `tghop`, `grantflow`, `fdmastack`, `tdmaslots`: redrawn on a
  1080×1920 stage with larger type; no crops.
- Verticals open with a 2 s text hook over the segment's strongest visual and end
  with a 2 s end slate; captions burned inside safe zone (above bottom 25%).

## Shorts clip map

| Clip | Span | Hook (burned setup card) |
|---|---|---|
| 01c1 | conventional waste | "Why 30 radio channels sit silent" |
| 01c2 | pool grant/release | "Radio channels that exist for one call" |
| 02c1 | CC hook/messages | "The radio channel that never speaks" |
| 02c2 | grant flow/late entry | "How radios join a call they never heard start" |
| 03c1 | tghop | "This radio channel doesn't exist" |
| 03c2 | rid/variants | "Two numbers identify every radio call" |
| 04c1 | grant flow/retune | "One message moves 100 radios at once" |
| 04c2 | anatomy | "Inside the message that starts every call" |
| 05c1 | FDMA | "The simplest way radios share the air" |
| 05c2 | TDMA | "How 4 calls fit on one frequency" |
