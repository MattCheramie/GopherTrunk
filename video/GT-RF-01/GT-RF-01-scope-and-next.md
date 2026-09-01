# GT-RF-01 — Scope of the primary video, and the next video

## Primary video: "Radio Fundamentals — How Signals Actually Work (Act I)"

**What it is.** The pilot of the GopherTrunk Field Guide video program and Act I of the
GT-RF-01 pillar: a 21:24, six-chapter foundation course. One chapter ≈ one Field Guide
entry, each chapter self-contained (per the segment-first production model), stitched
with a cold open, course intro, map-card transition beats, outro, and end slate.

**Audience.** Scanner listeners, new hams, SDR tinkerers — anyone pointing a
software-defined radio at the sky who nods along to "megahertz" without being sure what
it measures. No math beyond arithmetic; no prerequisites.

**Arc question.** *How does an invisible ripple in a field carry a voice across a city —
and what decides whether it decodes?*

| Chapter (exact Field Guide term) | Slugs covered (`also_slugs`) | Pillar timestamps |
|---|---|---|
| Radio wave | radio-wave (+ electromagnetic-spectrum) | 1:42–4:45 |
| Frequency | frequency (+ wavelength, frequency-bands touch) | 4:54–7:53 |
| Modulation | modulation (+ carrier-wave, amplitude/phase touch) | 8:05–11:20 |
| Bandwidth | bandwidth | 11:32–14:14 |
| Decibel (dB) | decibel (+ dbm, dbfs) | 14:24–17:07 |
| Signal-to-noise ratio (SNR) | signal-to-noise-ratio (+ noise-floor) | 17:17–20:18 |

**Deliberately OUT of scope for Act I:** hardware (antennas, SDRs), propagation,
anything protocol-specific (trunking, P25/DMR/TETRA), DSP internals, and the deeper
RF metrics (impedance, reflections, phase noise…). Each chapter plants exactly the
hooks those later topics need — the three wave properties, λ = c/f, the symbol/
constellation idea, Shannon's C = B·log₂(1+SNR), decibel arithmetic, and the SNR cliff.

**Derivatives shipped with it:** six standalone 16:9 chapter cuts, six 9:16 verticals
(re-hooked, burned captions), 13 Shorts clips, chapters list, SRT, `videos.yml` stub.

## Next video: GT-RF-01 Act II — "The RF Toolbox" (recommended)

Finish the pillar before starting a new one: **eight segments, ~30–33 min**, from
scripts and storyboards that are ALREADY WRITTEN and in the repo
(`video/GT-RF-01/scripts/GT-RF-01.07…14`, storyboards in `storyboards/`):

| # | Segment | Slugs |
|---|---|---|
| .07 | Amplitude (& phase) | amplitude, phase |
| .08 | Attenuation & path loss | attenuation, path-loss |
| .09 | Link budget | link-budget, friis-transmission-equation |
| .10 | ERP & EIRP | erp-eirp, field-strength |
| .11 | Impedance (Z) | impedance |
| .12 | Reflection Coefficient (Γ) & return loss | reflection-coefficient, return-loss |
| .13 | S-parameters | s-parameters |
| .14 | Resonance & Q | resonance, q-factor |

**Arc question:** *you can describe a signal — now can you measure it, budget it, and
deliver it to the antenna without losing it?* Act II follows the signal's power: the
two remaining wave knobs (.07) → what the world takes away (.08) → the accounting
(.09–.10) → what the feedline and antenna do to it (.11–.14). It leans directly on
Act I's decibel and SNR chapters; transition beats reference them (never the segment
bodies).

**Production notes for Act II:** narration + timelines come straight from
`pipeline/tts.py` on the existing scripts; the animation engine, brand kit, and
vertical/clip pipeline are all reusable — the new work is 8 scene files (each
animating its article's SVG figure, exactly as Act I did). Act III (.15–.22:
harmonics/spurs, IMD, occupied bandwidth, duty cycle, PAPR, group delay, phase
noise, Shannon capacity as the finale) closes the pillar the same way, after which
the three Acts concatenate into the full 90-minute GT-RF-01 upload per the strategy's
pillar model.

**Alternative next video** (if audience-building beats pillar completion): Wave-1
roadmap item **GT-RF-02 "SDR from Zero — your first software-defined radio"**
(sdr-devices category, ~22 segments) — broader search appeal and buyer-guide traffic,
but it needs all-new scripts, and hardware segments want real bench footage the
current all-graphics pipeline can't synthesize. Recommendation stands: **Act II next.**
