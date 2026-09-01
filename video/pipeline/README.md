# GopherTrunk video pipeline — code

Implements `video/gophertrunk-video-pipeline.md`. All media output goes to the
git-ignored `video/_render/`; only text/graphics/pipeline sources are committed.

Run order for a pillar (repo root = `video/`):

```bash
SP=<scratchpad with en-us-ryan-high.onnx>
# 1. VO-first narration + timelines + SRT + transcripts
for s in GT-TR-01/scripts/*.md; do
  python3 pipeline/tts.py "$s" $SP/en-us-ryan-high.onnx _render/GT-TR-01/tts
done
# 2. music bed
python3 pipeline/music.py _render/GT-TR-01/music/bed.wav 96
# 3. render each piece (wide)
python3 pipeline/render.py _render/GT-TR-01/tts/GT-TR-01.01.timeline.json \
  _render/GT-TR-01/tts/GT-TR-01.01.narration.wav _render/GT-TR-01/pieces/GT-TR-01.01.mp4
# 4. mix bed under each piece, then assemble pillar (concat, loudnorm, chapters)
python3 pipeline/assemble.py mix   _render/.../piece.mp4 _render/.../bed.wav _render/.../piece.mix.mp4
python3 pipeline/assemble.py pillar _render/GT-TR-01/pieces _render/GT-TR-01/final GT-TR-01/transcripts
# 5. standalone segment cuts: loudnorm each mixed segment piece to -14 LUFS
python3 pipeline/assemble.py loudnorm piece.mix.mp4 GT-TR-01.01.mp4 -14
# 6. verticals (re-layout + burned captions + hook + end slate), then shorts
python3 pipeline/render.py <timeline> <narration> out.vertical.mp4 --vert --hook "..."
python3 pipeline/shorts.py GT-TR-01/transcripts _render/GT-TR-01/verticals _render/GT-TR-01/shorts
```

Files:
- `tts.py` — script parser, per-sentence Piper synthesis, timeline JSON, narration WAV, SRT, transcript (doc §5–§6).
- `music.py` — procedural license-free ambient bed, loop-safe, ≈ −38 dBFS RMS (§7).
- `render.py` — Chromium `window.seek(t)` → PNG frames piped into ffmpeg (§8); `--vert` does the 9:16 re-layout with burned captions.
- `assemble.py` — bed mix (video stream copied), concat, two-pass loudnorm, chapters/videos.yml.stub/pillar SRT (§9).
- `shorts.py` — clip cuts at `[CLIP]` bounds with burned hook cards (§11).

The scene engine and brand tokens live in `video/brand/` (engine.js, brand.css).
