# GT-TR-01 — posting strategy & launch calendar

Distribution plan for the 22 rendered deliverables of the Trunked Radio pillar
(pillar 17:02, five 16:9 segment cuts, five 9:16 verticals, ten Shorts clips).
Dates assume a Tue 2026-09-08 launch; shift the whole grid if launch moves.
The HTML version of this plan (shareable) is published as a session artifact.

## Platform roles

| Platform | Role | What goes there |
|---|---|---|
| YouTube (main) | Searchable evergreen home | Pillar (rejoined single file), 5 segment cuts in a course playlist, SRT + chapters + thumbnails |
| YouTube Shorts | Discovery on the same channel | 10 clips + the 5 full verticals (all ≤3:00, so they qualify) |
| TikTok | Top-of-funnel discovery | 10 clips, then full verticals as weekend "deep dives" |
| Instagram Reels | Same clips, maker/tech audience | 10 clips (verticals too where the 3-min cap applies) |
| Facebook (Reels + groups) | Scanner-hobby communities | Clips via Meta Business Suite cross-post; pillar link into relevant groups |
| X | Announcement + 2–3 best clips | Native uploads, not links |
| Reddit / forums | Community seeding, value-first | Pillar + article links (r/RTLSDR etc.; respect 9:1 self-promo norms) |
| gophertrunk.org | Owned home | Embed segments on their Field Guide pages via `videos.yml` (stub committed) |
| LinkedIn (optional) | Open-source project reach | One pillar announcement in week 4 |

Always upload the clean masters — never re-share a watermarked download.

Rejoin the pillar from the two delivered parts (stream copy, no quality loss):

```
printf "file 'GT-TR-01-pillar-part1.mp4'\nfile 'GT-TR-01-pillar-part2.mp4'\n" > join.txt
ffmpeg -f concat -safe 0 -i join.txt -c copy GT-TR-01-pillar.mp4
```

## Cadence logic

- Pillar first (it is the destination every clip funnels to), segments staggered
  behind it so they don't cannibalize its first week.
- Shorts 3×/week (Mon/Wed/Fri ~18:00 ET) posted natively to TikTok + YT Shorts +
  Reels the same day; full verticals take the weekend slot (Sun ~10:00 ET).
- Each segment's clips land in the same week its 16:9 cut goes public.
- Every recap card already burns the article URL; descriptions use UTM links:
  `https://gophertrunk.org/reference/<slug>/?utm_source=<platform>&utm_medium=video&utm_campaign=gt-tr-01`

## Calendar

**Week 0 — prep (Wed Sep 2 – Sun Sep 6)**
- Rejoin pillar; upload + schedule everything on YouTube (unlisted → scheduled).
- Attach `GT-TR-01-pillar.srt`, paste `GT-TR-01-chapters.txt` into the description,
  set thumbnails (in `video/brand/thumbs/`), build the "Trunked Radio — Field
  Guide" playlist, end screens + cards between videos.
- Merge `videos.yml.stub` into `docs/_data/videos.yml` so article pages embed the
  pillar at per-slug timestamps.
- Sun: YouTube community-post teaser.

**Week 1 — launch (Mon Sep 7 – Sun Sep 13)**
| Day | Asset | Where |
|---|---|---|
| Mon 9/7 18:00 ET | `02c1` "The radio channel that never speaks" (strongest hook) | TikTok + YT Shorts + Reels |
| Tue 9/8 18:00 ET | **Pillar premiere** + community post; site embeds live | YouTube |
| Wed 9/9 | `01c1` "Why 30 radio channels sit silent" | TikTok + YT Shorts + Reels |
| Fri 9/11 | `01c2`; native `02c1` upload | verticals; X |
| Sun 9/13 10:00 ET | `01-vertical` (full 2:52) | TikTok + YT Shorts |

**Week 2 (Sep 14 – 20)**
| Day | Asset | Where |
|---|---|---|
| Mon | `03c1` "This radio channel doesn't exist" | verticals |
| Tue 18:00 ET | Segment `01` public in playlist; value-first pillar post | YouTube; Reddit r/RTLSDR |
| Wed | `02c2` | verticals |
| Fri | `03c2` | verticals |
| Sun | `02-vertical` | TikTok + YT Shorts |

**Week 3 (Sep 21 – 27)**
| Day | Asset | Where |
|---|---|---|
| Mon | `04c1` "One message moves 100 radios at once" | verticals |
| Tue | Segment `02` public | YouTube |
| Wed | `04c2` | verticals |
| Thu | Segment `03` public | YouTube |
| Fri | `05c1` | verticals |
| Sun | `03-vertical` | TikTok + YT Shorts |

**Week 4 (Sep 28 – Oct 4)**
| Day | Asset | Where |
|---|---|---|
| Mon | `05c2` | verticals |
| Tue | Segment `04` public | YouTube |
| Wed | Re-post the top-2 clips by 3-s retention with fresh captions | TikTok + Reels |
| Thu | Segment `05` public → playlist complete; LinkedIn announcement | YouTube; LinkedIn |
| Fri | Community post: course complete + poll for the next pillar topic | YouTube |
| Sat | `04-vertical` | TikTok + YT Shorts |
| Sun | `05-vertical` | TikTok + YT Shorts |

**Steady state (from Oct 6)**: 2–3 Shorts/week (cut fresh `[CLIP]` spans from the
existing segments with the pipeline), one new pillar per month reusing this
four-week template, shifted to the new launch date.

## Per-platform metadata

- **YouTube pillar title**: "How Trunked Radio Works — the complete course (P25,
  DMR, TETRA)". Segments: question-form titles ("What Is a Talkgroup?…").
  Category Science & Technology, not made for kids, chapters + SRT attached.
- **Shorts/TikTok/Reels titles** = the burned hook lines. First comment links the
  pillar (UTM). Hashtags: `#SDR #RTLSDR #HamRadio #ScannerRadio #P25 #DMR #TETRA
  #SoftwareDefinedRadio` + 2–3 broad (`#engineering #radio #howitworks`).
- **Caveat**: if an account's Reels cap is 90 s, post only the ≤50 s clips there.

## KPIs (review every Friday)

| Metric | Target by week 4 |
|---|---|
| Pillar average view duration | ≥ 35 % (≈ 6 min) |
| Shorts 3-second retention | ≥ 70 % on at least 3 clips |
| Clip → pillar/site click-through (UTM) | ≥ 1.5 % |
| Playlist completion starts | growing week over week |

Double down on whichever two hooks win: cut more clips from those segments
before producing the next pillar.
