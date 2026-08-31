# GopherTrunk video production package

Production system for the Field Guide video program (strategy: three-tier
pillar → segment → clip model). **No rendered media lives in this repository**
— `video/_render/` is git-ignored; videos are delivered out-of-band. Everything
here is the source: scripts, storyboards, graphics, captions, and the pipeline
that turns them into finished video.

**Start here: [`gophertrunk-video-pipeline.md`](gophertrunk-video-pipeline.md)** —
the self-contained recipe. Give that file plus a topic to a future Claude session
(or follow it by hand) and it reproduces the whole production for a new subject.

## Layout

```
gophertrunk-video-pipeline.md   the reproducible recipe (read first)
brand/                          brand kit: tokens (brand.css) + card templates (cards.html)
pipeline/                       the toolchain (see below)
GT-RF-01/                       "Radio Fundamentals" pillar
  scripts/                      all segment + pillar-element scripts (VO = transcript)
  storyboards/                  per-segment beat tables, vertical plans, clip picks
  transcripts/                  SRT captions (per segment + full pilot)
  graphics/                     animation scenes (one HTML per segment; deterministic)
  GT-RF-01-tracker.csv          per-segment production status
_render/                        (git-ignored) all rendered media + working files
```

## Pipeline (video/pipeline/)

| Script | Does |
|---|---|
| `common.py` | shared paths: ffmpeg (imageio-ffmpeg), Chromium, fps |
| `script_parse.py` | script .md → visual events + VO paragraphs + [CLIP] marks |
| `tts.py` | VO-first: Piper TTS per sentence → narration.wav + timeline.json + SRT |
| `pillar_tts.py` | splits the pillar-elements script into pieces and runs tts on each |
| `engine.js` / `fig.js` / `common_visuals.js` | deterministic clock-driven motion-graphics engine + shared figure/card components |
| `render.py` | scene + timeline → H.264 frames via headless Chromium screenshots |
| `render_all.py` | batch: every pilot piece, video-only |
| `snap.py` | QC stills of a scene at chosen times |
| `render_still.py` | brand cards/thumbnails → PNG |
| `music.py` | procedural ambient bed (license-free), −34 dBFS under VO |
| `assemble.py` | mix VO+bed per piece → concat → 2-pass loudnorm (−14 LUFS) → chapters, videos.yml stub, full SRT, per-segment 16:9 exports |
| `vertical.html` / `vertical_all.py` | 9:16 re-edits (re-hooked open, burned captions, end slate, −13 LUFS) + Shorts clips cut at [CLIP] spans |

## Re-rendering from scratch

```bash
pip3 install --user imageio-ffmpeg piper-tts playwright numpy
# fonts + voice model (not committed): see gophertrunk-video-pipeline.md §3–§4
cd video/pipeline
python3 tts.py ../GT-RF-01/scripts/GT-RF-01.0*.md   # narration + timelines + SRT
python3 pillar_tts.py                                # cold open, intro, transitions, outro
python3 render_all.py                                # all 16:9 pieces (slow: ~2× realtime)
python3 assemble.py                                  # pilot + segment exports + chapters
python3 vertical_all.py                              # verticals + clips
```

Renders land in `video/_render/exports/`.

## Publishing checklist (per playbook §4.4)

1. Upload pilot UNLISTED → verify chapters from `exports/chapters.txt` → public.
2. Upload `exports/GT-RF-01-pilot.srt` as the caption track.
3. Fill `exports/videos.yml.stub` with the real `youtube_id`/date → paste into
   `docs/_data/videos.yml` (site embed wiring is a separate PR per the strategy doc).
4. Drip verticals (2–3/week) and Shorts (2–3/week), each linking back to the pillar
   at the segment timestamp; keep segment IDs in every description.
