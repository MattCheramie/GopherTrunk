# GT-RF-01 — asset & license log

Mandatory log per playbook §1.4. Every third-party image, music bed, capture,
and font used in the pillar gets a row BEFORE it enters an edit.

## Rules
- Music: licensed library only (Epidemic/Artlist) — log track, license ID.
  Never platform-restricted tracks (clips go multi-platform).
- Images: own renders/photos preferred; anything borrowed needs source URL +
  license + attribution string here, and an on-screen credit where required.
- IQ captures / off-air audio: log provenance; only recordings legal to
  record/rebroadcast; never encrypted-traffic content; blur/omit identifying
  talkgroup metadata where sensible (playbook §2.4 legal note).

## Log

| Asset | Used in | Source | License | Attribution / notes |
|---|---|---|---|---|
| `docs/assets/gophertrunk-logo.png` | title cards, corner bug, thumbnails | this repo | project asset | — |
| Liberation Sans / DejaVu Sans Mono | all generated cards (render-env stand-in for the site's system font stack) | OS fonts | SIL OFL / free | swap for platform system fonts at final render if desired |
| espeak-ng scratch VO | preview exports only | generated | GPL (tool) | NOT for publication — timing reference only |
| Generated scenes (`pipeline/scenes.py`) | all segments | this kit | project asset | — |
| Music beds | (none selected yet) | — | — | pick 4–6 beds, tag calm/technical/upbeat |
| GopherTrunk UI captures | tie-in beats (.01–.22), demos (.12, .13, .21) | (to capture: OBS 4K dark theme) | project asset | note GT version in YouTube description |
| Off-air IQ/audio for demo beats | .13 SNR demo, cold open | `samples/` in repo or fresh capture | — | log exact file + jurisdiction check before edit |
