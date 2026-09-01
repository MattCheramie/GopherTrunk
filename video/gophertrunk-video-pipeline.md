# GopherTrunk Video Pipeline — Reproducible Production Recipe

**What this is:** A self-contained recipe that lets a future Claude session (or a human)
reproduce a complete GopherTrunk Field Guide video — motion graphics, neural TTS
narration, captions, music, 16:9 pillar + 9:16 verticals + Shorts clips — for **any new
topic**, end to end, inside a Claude Code session with no other tools than a shell,
Python, and the Playwright Chromium that ships with the environment.

**How to use it:** Upload this file to a session (or point at it in the repo:
`video/gophertrunk-video-pipeline.md`) and say:

> "Using video/gophertrunk-video-pipeline.md, produce the video for <topic>."

where `<topic>` is a Field Guide category (`trunked-radio`), a pillar ID (`GT-TR-01`),
or a list of entry slugs. Everything else — segment selection, scripts, narration,
graphics, rendering, packaging — follows from this document.

It is the automation-level companion to two planning docs (upload them too if you have
them, but this file restates every constraint needed to work without them):
`gophertrunk-video-strategy.md` (three-tier content model) and
`gophertrunk-video-production-playbook.md` (per-entry-type outlines + sample scripts).

**Provenance:** distilled from the GT-RF-01 "Radio Fundamentals" pilot production
(2026-08-31, Claude Code remote session). Environment facts below were verified in that
session, not assumed. §14 logs lessons learned; append to it every run.

---

## 1. The contract (what a run produces)

| Deliverable | Spec | Goes where |
|---|---|---|
| Pilot/pillar video | 16:9 1920×1080 @ 30 fps, H.264 High CRF 19–21, AAC 192k 48 kHz stereo, −14 LUFS int., TP ≤ −1 dB | **sent as file, never committed** |
| Per-segment 16:9 cuts | same specs, title card → recap card | sent as files |
| Vertical re-edits | 9:16 1080×1920, re-hooked ≤2 s open, burned captions, end slate, −13 LUFS | sent as files |
| Shorts clips | 30–60 s 9:16 cut at `[CLIP]` bounds, own hook | sent as files |
| Scripts | one `.md` per segment, 550–750 words, `[V: …]` + `[CLIP]` marks | committed: `video/<PILLAR>/scripts/` |
| Storyboards | per-segment beat table | committed: `video/<PILLAR>/storyboards/` |
| Transcripts + SRT | full transcript + per-segment and pillar SRT | committed: `video/<PILLAR>/transcripts/` |
| Graphics sources | brand kit + animation scenes (HTML/JS/SVG) | committed: `video/brand/`, `video/<PILLAR>/graphics/` |
| Pipeline code | the scripts in §5–§11, checked in once, reused | committed: `video/pipeline/` |
| Tracker + YAML stub | `<PILLAR>-tracker.csv`, `videos.yml.stub` (start/end per slug) | committed: `video/<PILLAR>/` |

**Repo policy (owner decision, standing):** NO rendered media (video or audio) is ever
committed to the repository. Renders live in a git-ignored `video/_render/` (or the
session scratchpad) and are delivered by sending files in the conversation. Text,
graphics sources, and pipeline code ARE committed, on the session's designated branch —
never push to a different branch.

**IDs** (filenames, YAML, descriptions): pillar `GT-<domain>-<NN>`, segment
`GT-<domain>-<NN>.<SS>`, clip `GT-<domain>-<NN>.<SS>c<N>`.

## 2. Segment selection from a topic

1. Open `docs/_data/reference.yml`; find the category (or slugs) named by the topic.
2. Pick 5–7 entries for a pilot (or ~20–24 for a full pillar), **dependency-sorted**:
   concepts/foundations first; each segment may assume only earlier segments — but only
   in transition beats, never inside a segment body.
3. Group tightly-coupled entries into one segment (e.g. decibel + dbm + dbfs; radio-wave
   + electromagnetic-spectrum). Record extras as `also_slugs`.
4. For each chosen entry read `docs/reference/<slug>.md`: front-matter `description`
   (becomes the hook/definition), section headings (become body beats), the inline
   `<figure class="figure">` SVG (becomes the storyboard for the core animation — most
   Field Guide articles have one), `see_also` (transition-beat fodder).
5. Pillar-only elements to script: cold open (45–60 s, teases the 2–3 best moments +
   the arc question), course intro (~2 min), one transition beat per boundary (10–20 s),
   outro (~60 s) + 20 s end slate.

## 3. Environment setup (verified 2026-08-31; re-verify each run)

```bash
pip3 install --user imageio-ffmpeg piper-tts playwright numpy
FF=$(python3 -c "import imageio_ffmpeg; print(imageio_ffmpeg.get_ffmpeg_exe())")
CHROME=$(ls -d /opt/pw-browsers/chromium-*/chrome-linux/chrome 2>/dev/null | head -1)
```

- `imageio-ffmpeg` ships a **full static ffmpeg** (libx264, aac, loudnorm, drawtext with
  fontconfig). The Playwright-bundled ffmpeg is VP8/WebM-only — do not use it.
- **Chromium**: use the pre-installed browser via `executable_path=$CHROME`
  (do NOT `playwright install`; `PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1` is already set).
  Launch args: `--no-sandbox --force-color-profile=srgb --hide-scrollbars`.
- **TTS decision tree** (try in order):
  1. `edge-tts` (best quality, needs network to `speech.platform.bing.com`) — in the
     remote sandbox this host is **policy-blocked** (CONNECT 403); on a local machine it
     usually works: `python3 -m edge_tts --voice en-US-AndrewMultilingualNeural ...`.
     Behind the agent proxy also append `/root/.ccr/ca-bundle.crt` to certifi's bundle
     and pass `--proxy "$HTTPS_PROXY"`.
  2. `piper-tts` with the **en-us-ryan-high** voice (22.05 kHz neural, local inference —
     what the pilot used). Hugging Face is blocked too; fetch the voice from GitHub
     releases (allowed):
     ```bash
     curl -sSL -O https://github.com/rhasspy/piper/releases/download/v0.0.2/voice-en-us-ryan-high.tar.gz
     tar xzf voice-en-us-ryan-high.tar.gz   # → en-us-ryan-high.onnx + .json
     echo "text" | python3 -m piper --model en-us-ryan-high.onnx --output-file line.wav
     ```
- Disk is a fixed allowance: render at 1080p, stream PNG frames straight into ffmpeg
  (never write frame files), delete intermediates after each segment.

**Smoke test before writing any content** (must pass): synthesize one TTS line → launch
Chromium on a trivial `window.seek(t)` page → pipe 60 PNG screenshots into
`$FF -f image2pipe -vcodec png -r 30 -i - -i line.wav -c:v libx264 -pix_fmt yuv420p -c:a aac out.mp4`
→ ffprobe shows H.264 + AAC. Render throughput ≈ 15–20 fps at 1080p (≈ 2× realtime).

## 4. Brand kit (build once — reuse: `video/brand/`)

Exact site tokens (from `docs/assets/css/style.scss`): dark bg `#0d1117`, elevated
`#161b22`, text `#e6edf3`, muted `#8b96a3`, accent (dark) `#58a6ff`, accent (light)
`#155799`, accent-soft `#e8f0fa`, border `#30363d`-family, radius `0.5rem`, system
sans + mono stacks. Videos skew dark-theme. Logo: `docs/assets/gophertrunk-logo.png`.

Templates (HTML/CSS, rendered by the same engine):
- **Title card** (2 s): eyebrow "GOPHERTRUNK FIELD GUIDE" in muted caps, term in ~96 px
  bold, segment ID small bottom-right, 6 px domain-color stripe, corner gopher mark.
  Hard cut in/out — it is the chapter boundary marker.
- **Recap card** (10–15 s): term small on top, 2–3 numbered bullets revealed one-by-one,
  footer line "Full write-up → gophertrunk.org/reference/<slug>/".
- **Lower-third / definition & formula cards**: elevated bg `#161b22`, 1 px border,
  radius 8 px, accent underline; ONE formula max on screen.
- **Corner bug**: logo at low opacity top-right, OUTSIDE the 9:16 center zone (so
  verticals are bug-free automatically).
- **Caption style** (burned verticals): white on `rgba(13,17,23,.85)` pill, bottom ~30%
  but above platform UI (avoid bottom 25%), ≤ 2 lines, keywords in accent.
- **End slate** (20 s pillar / 2 s vertical): "Full course on YouTube — GopherTrunk" +
  site URL, calm.
- **Thumbnails**: 1280×720 dark bg, big term left, one figure right, domain stripe;
  vertical variant hook-text top-third.
- **Layout rule for everything:** all must-read content inside the **center 9:16 zone**
  (x 690–1230 at 1080p 16:9). Wide diagrams that can't fit get a planned tall re-layout
  for the vertical (flag them in the storyboard) — never a blind crop.

## 5. Scripts (per segment)

Follow the playbook skeleton — hook-first, self-contained, 550–750 words ≈ 3:30–5:00
at ~150 wpm:

| Beat | Time | Rule |
|---|---|---|
| Title card | 2 s | no VO |
| Hook + definition | 15–25 s | first spoken words = the exact Field Guide term; paraphrase the page `description`; no "welcome back" |
| Body (type-specific) | 2.5–4 min | term: analogy → mechanism (animate the page SVG) → numbers → why-you-care. concept: problem-story → reveal. See playbook Part 2 for algorithm/protocol/hardware/person/org/language outlines |
| GopherTrunk tie-in | 20–40 s | where it shows up in the product/UI |
| Recap card | 10–15 s | 2–3 bullets + "full write-up linked below"; never "up next" (that lives in the transition beat) |

Script file format (drives everything downstream — keep it machine-parsable):

```markdown
# GT-RF-01.05 — Decibel (dB)   <!-- seg id + exact page title -->
slug: decibel
also_slugs: [dbm, dbfs]

**[V: title]**                 <!-- visual directions in [V: ...] lines -->

**[V: numbers-flying]** [CLIP c1 start]
Decibels exist because radio numbers are ridiculous. ...   <!-- VO paragraphs -->

**[V: definition-card "dB = 10·log₁₀(P₂/P₁)"]**
A decibel is a logarithmic way of writing a ratio... [CLIP c1 end]
...
**[V: recap "① ratio, log scale ② +3=×2 · +10=×10 ③ suffix = absolute"]**
So: decibels turn absurd ratios into small numbers...
```

Rules: no cross-segment references in the body; mark 2–3 `[CLIP]` spans at script time;
every fact verified against the article; keep the article's exact terminology (SRT and
chapter search depend on it). VO paragraphs = the transcript = the SRT source.

## 6. Narration (VO-FIRST — this ordering is the whole trick)

Record narration BEFORE animating; the measured audio drives the timeline (animating to
audio is easy; the reverse is misery).

1. Parse each script: split VO paragraphs into sentences (piper sounds best per
   sentence; keep numbers/units as speakable text — write "ten decibels", "d B F S" →
   test pronunciations; use "dee-bee" spellings only if a listen-check demands it).
2. Synthesize per line → `line_NNN.wav`; measure duration via ffprobe.
3. Build the segment **timeline JSON** — the single source of truth for audio, video
   and captions:

```json
{ "seg": "GT-RF-01.05", "fps": 30,
  "events": [
    {"t": 0.0,  "dur": 2.0,  "visual": "title", "vo": null},
    {"t": 2.0,  "dur": 8.13, "visual": "numbers-flying", "vo": "line_001.wav",
     "text": "Decibels exist because radio numbers are ridiculous.", "clip": "c1"},
    ...
  ]}
```

   Layout rule: each `[V:]` block starts when its first VO line starts; add 0.35 s gap
   between paragraphs, 0.8 s beat before recap; title card silent 2 s.
4. Concatenate line WAVs with the gaps (ffmpeg `aevalsrc=0:d=<gap>` + concat filter or
   pad with `adelay`/`concat` demuxer on padded lines) → one narration WAV per segment
   at 48 kHz.
5. SRT falls straight out of the same JSON (text + t/dur), one cue per line, wrapped
   ≤ 42 chars/line, 2 lines max.

## 7. Music bed (license-free, procedural)

No stock library is available (and platform-restricted tracks are forbidden anyway), so
synthesize a calm ambient bed with numpy: slow chord pad (root drone + fifth + octave,
sine/triangle partials with slow LFO amplitude), lowpassed noise swell at section
boundaries, ~70 BPM pulse at −30 dB or none. Bounce to 48 kHz stereo WAV, loop-safe.
Mix at ≥ 20 dB below VO (bed ≈ −38 dBFS RMS vs VO ≈ −16), duck 3 dB further under
speech via sidechaincompress if the bed is busy. **Beds start/stop at segment
boundaries — never straddle a cut.**

## 8. Motion-graphics engine (deterministic HTML/JS)

One HTML page per segment. Contract:
- Page sizes itself to the render resolution; everything absolute-positioned on a
  1920×1080 stage (or 1080×1920 for vertical layouts).
- Exposes `window.seek(t_seconds)` → synchronously sets ALL visual state for time `t`
  (no CSS animations/transitions, no rAF, no wall-clock — pure function of `t`).
- Scene = an array of the timeline JSON's events; each `visual:` name maps to a JS
  "component" (card build-ons, SVG figure animations). Ease with
  `p=min(1,(t-t0)/0.4); e=1-Math.pow(1-p,3)` style curves; 0.3–0.5 s ease-ins; sober
  motion (site tone).
- **Core animation per segment = the article's own SVG figure, animated**: copy the
  `<figure class="figure">` SVG from `docs/reference/<slug>.md`, restyle
  `currentColor` → theme tokens, then reveal it element-by-element synced to VO beats
  (stroke-dashoffset draws, staged opacity, morphs).
- Keep every card's text inside the center 9:16 zone; decorative spill may use the
  outer thirds (pillar-only).

Renderer (Python + Playwright — the validated loop):

```python
pg.goto(f"file://{scene}.html"); pg.evaluate("window.ready")   # fonts/layout settled
enc = subprocess.Popen([FF,"-y","-f","image2pipe","-vcodec","png","-r","30","-i","-",
  "-i",narration_wav,"-c:v","libx264","-preset","medium","-crf","20",
  "-pix_fmt","yuv420p","-c:a","aac","-b:a","192k","-ar","48000","-shortest",out_mp4],
  stdin=subprocess.PIPE)
for i in range(total_frames):
    pg.evaluate(f"window.seek({i/30})")
    enc.stdin.write(pg.screenshot(type="png"))
```

## 9. Assembly (pillar)

1. Render each piece (cold open, intro, segments, transition beats, outro+end slate)
   as its own MP4 with identical codec params.
2. Mix music: per-piece `amix` of narration + bed (bed trimmed to piece length,
   3 s fade-out at boundary).
3. Concat with the demuxer (`-f concat -safe 0 -i list.txt -c copy`) — identical params
   make stream-copy concat safe; hard cuts at title cards.
4. **Loudness**: two-pass `loudnorm` on the final master
   (`-af loudnorm=I=-14:TP=-1.5:LRA=11:measured_...` from pass-1 JSON), −13 for
   verticals.
5. Chapters: running-sum of piece durations → `00:00 Intro` + one line per segment
   using the exact Field Guide term; also emit `videos.yml.stub` with per-slug
   `start_s`/`end_s` (title-card frame → last recap frame) matching
   `docs/_data/videos.yml` schema in the strategy doc.
6. Pillar SRT = per-segment SRTs re-offset by chapter start.

## 10. Verticals (per segment)

- Duplicate scene with a 1080×1920 stage: stacked/tall re-layouts of wide visuals
  (this is real layout work for diagram-heavy segments — budgeted, not a crop).
- Replace the 2 s title card with a **1.5–2 s burned text hook** over the strongest
  visual (verticals die on slow opens).
- Burn captions from the SRT (drawtext or pre-rendered caption divs in the scene —
  the scene approach is easier and pixel-consistent; keep inside safe zones).
- End slate 2 s ("Full course on YouTube — GopherTrunk"). Loudness −13 LUFS.

## 11. Shorts clips (2–3 per segment)

Cut the vertical at the scripted `[CLIP]` bounds (`-ss/-to` on the re-encode, or render
the span directly). Each clip: ONE idea; if its first line assumes context, burn a
1-line setup card; end resolved (loopable) or on a curiosity gap. Name
`GT-XX-NN.SSc<N>.mp4`. Log clip→segment mapping in the tracker.

## 12. QC checklist (every export)

- [ ] ffprobe: H.264 High yuv420p, 30 fps exactly, AAC 48 kHz, expected duration ±0.5 s
- [ ] loudnorm print: −14 (16:9) / −13 (9:16) ±1 LU, TP ≤ −1 dBTP
- [ ] Frame spot-checks at each beat boundary (screenshot → visual review): no
      clipped text, correct card content, 9:16 zone respected
- [ ] SRT first/last cue aligns with audio; protocol names/units spelled exactly
- [ ] Standalone sanity: segment makes sense cold, no "as we saw" inside the body
- [ ] `git status`: no `.mp4/.wav/.mp3/.onnx` staged; render dirs ignored
- [ ] Worked-example numbers verified by hand (commenters will check)

## 13. Packaging & delivery

```
video/
  gophertrunk-video-pipeline.md     # this file (update §14 each run!)
  brand/                            # kit sources + rendered PNG cards/thumbnails
  pipeline/                         # tts.py, timeline.py, engine.js, render.py, assemble.py, music.py
  <PILLAR>/
    scripts/  storyboards/  transcripts/  graphics/
    <PILLAR>-tracker.csv  videos.yml.stub
  _render/                          # GIT-IGNORED — all media output
```

Commit text+graphics to the session's designated branch (`git push -u origin <branch>`,
retry w/ backoff on network errors). Send each rendered video via the conversation's
file-send as it's finished. Do NOT open a PR unless asked.

## 14. Lessons learned (append per run)

**2026-08-31 · GT-RF-01 pilot (remote sandbox):**
- `speech.platform.bing.com` and `huggingface.co` are policy-blocked in the remote
  environment; GitHub releases are open → Piper voices via
  `rhasspy/piper/releases/download/v0.0.2/…`. edge-tts remains preferable when the
  network allows.
- The agent proxy MITMs TLS: append `/root/.ccr/ca-bundle.crt` to certifi's
  `cacert.pem` for any Python HTTPS client that doesn't honor `SSL_CERT_FILE`.
- Playwright-bundled ffmpeg is useless for delivery (VP8/WebM only, no audio);
  imageio-ffmpeg's static build has everything (x264, aac, loudnorm, drawtext, soxr).
- Chromium lives at `/opt/pw-browsers/chromium-<rev>/chrome-linux/chrome` — glob it,
  the plain `chromium` path is a directory stub.
- 1080p30 screenshot-render throughput ≈ 16 fps with `preset fast` encode in the same
  pipe — budget ~2× realtime for renders.

**2026-09-01 · GT-TR-01 pilot (trunked radio, remote sandbox):**
- edge-tts still blocked, and the failure mode changed dress: after appending the proxy
  CA to certifi the CONNECT 403 surfaces as edge-tts `SkewAdjustmentError: No server
  date in headers` (the proxy's block page has no Date header) — recognize it as the
  policy block, not a clock problem. Piper ryan-high from GitHub releases remains the
  fallback; `PiperVoice.load()` once + `synthesize_wav()` per sentence avoids the
  per-line model-load cost of the CLI.
- Piper at `length_scale 1.04` reads ~200 wpm, not the 150 wpm the word-count table
  assumes — a 600-word segment lands at ~2:50, not 4:00. Write toward the upper word
  band (650–750) if you want 3:30+ segments.
- One shared deterministic engine (`brand/engine.js`) with per-visual components +
  variant timelines beats one HTML page per segment: consecutive `[V:]` blocks with the
  same visual name share a component instance, so multi-beat diagrams animate
  continuously across narration beats.
- Piper VO measured: peak −1.1 dBFS, active RMS ≈ −16 dBFS, zero clipping — mix math
  from §7 holds unchanged.
- 4-core box: two parallel render lanes (browser+x264 each) ≈ 1.6× throughput; per-lane
  ~9 fps at 1080p30 with `preset medium` (not 16 — budget ~3.5× realtime per lane).
- Burned-caption pill: don't pre-break lines with `<br>` — hand the renderer ≤76-char
  chunks and let the browser wrap inside a fixed-width pill (pre-broken 38-char lines
  double-wrap in DejaVu Sans at 40 px).
- ffmpeg loudnorm one-pass drifts on short clips; two-pass with `linear=true` +
  `alimiter` at TP target hits −14/−13 LUFS ±0.5 LU on every export. loudnorm/mix are
  audio-only re-encodes (`-c:v copy`) — video is encoded exactly once per piece.
- Clip spans need checking at TTS time, not render time: one `[CLIP]` span measured
  23 s (< the 30 s floor) and was fixed by moving the end mark one beat later in the
  script, then re-running that segment's TTS only.
- **imageio-ffmpeg 7.0.2 has NO drawtext filter** (contradicts the 2026-08-31 note —
  that build differed). Burn text via a Chromium-rendered transparent PNG +
  `overlay=enable='lt(t,2.2)'` instead; shorts.py does this now.
- ffmpeg concat-demuxer list files resolve relative paths against the LIST file's
  directory — write absolute paths into concat.txt.
- AAC encoding overshoots true peak by up to ~0.6 dB after a limiter; set the
  alimiter ceiling 0.6 dB below the TP target or the QC gate fails by 0.1–0.5 dB.
- Conversation file-send caps at 30 MiB: a full pillar exceeds it. Split at chapter
  boundaries by re-concatenating the per-piece mixes (lossless) and loudnorm each
  part — do not re-encode the whole master to squeeze under the cap.
