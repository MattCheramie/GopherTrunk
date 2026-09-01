# GopherTrunk video production — GT-RF-01

Production kit for the **"Radio Fundamentals: How Signals Actually Work"**
pillar video (the intro-to-RF/radio course), built to the two planning docs:
`gophertrunk-video-strategy.md` and `gophertrunk-video-production-playbook.md`.

> **Location note:** the strategy doc says production files live on a
> production drive, NOT the repo. This tree is committed to the
> `claude/rf-radio-intro-video-wr359f` branch purely so the work persists;
> move it to the production drive (`GT-Video/`) before merging anything, or
> keep the branch unmerged as the archive. Nothing under `video-production/`
> is referenced by the Jekyll site.

## What's here

```
_brand/                     One-time brand kit (playbook §1.2)
  brand.py                  Site design tokens for all generators
  gen_cards.py              Title/recap cards, lower-third, corner bug,
                            thumbnails, course map card (per-node states)
  renders/                  Sample renders of each template
pipeline/
  scenes.py                 Reusable motion-graphic scene library (sine,
                            two-frequency, spectrum+SNR, dB ladder,
                            link-budget chain, text cards, animatic slate)
  segment_plans.py          slug → per-paragraph scene plan + recap bullets
  render.py                 Script → scratch-VO preview video + SRT captions
GT-RF-01/
  GT-RF-01-plan.md          Pillar plan: arc question, locked running order
  GT-RF-01-tracker.csv      Per-segment production tracker (playbook Part 5)
  GT-RF-01-licenses.md      Asset/license log
  scripts/                  22 full segment scripts + pillar-only elements
  captions/                 SRT per segment (generated from scripts)
  audio/vo-scratch/         espeak scratch narration (timing reference ONLY)
  exports/                  Rendered preview segments (720p animatic/preview)
  videos-yml-draft.yml      Ready-to-fill block for docs/_data/videos.yml
  youtube-description.md    Upload description template w/ chapter list
```

## The workflow (solo producer)

1. **Scripts are final-draft**: each is 550–750 words, grounded in its
   `docs/reference/<slug>.md` article, with `[V:]` storyboard directions and
   `[CLIP]` marks already chosen (playbook §1.5 pre-production ✓).
2. **Preview any segment today** (no camera, no mic):
   ```
   cd video-production/pipeline
   python3 render.py ../GT-RF-01/scripts/GT-RF-01.11-decibel.md
   ```
   → `GT-RF-01/exports/<seg>.mp4` with scratch TTS narration, real motion
   graphics where `segment_plans.py` has a plan, storyboard slates elsewhere,
   plus a proofread-ready SRT. Use these to judge pacing before recording.
   `--srt-only` regenerates captions + scratch VO without video.
3. **Record real VO** against the scripts (48 kHz WAV, −12 to −6 dBFS peaks,
   150 wpm ≈ the scratch timing). Drop files next to the scratch WAVs; the
   Resolve edit replaces scratch with real VO 1:1.
4. **Upgrade graphics**: add scenes to `scenes.py` / plans to
   `segment_plans.py` (or rebuild the same boards in Resolve/Fusion/Motion
   Canvas at 4K). Cards regenerate at any size: `gen_cards.py`.
5. **Capture GopherTrunk UI** for the tie-in beats: OBS 4K/30, dark theme,
   ≥125 % zoom — spectrum/waterfall, dBFS meters, SNR readouts, live decode.
6. **Assemble the pillar** in Resolve per playbook §4.3: one timeline per
   segment, nested master timeline, transition beats from
   `scripts/GT-RF-01.00-pillar-elements.md`, chapter markers on title cards.
7. **Publish + wire the site** per playbook §4.4 using
   `GT-RF-01/videos-yml-draft.yml` and `youtube-description.md`.

## Scratch VO disclaimer

`audio/vo-scratch/` and the preview exports use espeak-ng. They are timing
and pacing references for the edit — **never publish them**. Same for the
storyboard-slate scenes: they mark paragraphs whose graphics aren't built yet.

## Renders at 4K

`render.py --size=3840x2160` renders the same timelines at delivery
resolution; cards and scenes are resolution-independent. Keep 720p for
iteration speed.
