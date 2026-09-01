# GT-RF-01 — "Radio Fundamentals: How Signals Actually Work"

Pillar plan per `gophertrunk-video-strategy.md` §6 (Wave 1) and the production
playbook Part 4. Source category: `rf-fundamentals` (38 entries, curated to 22
segments / 24 slugs — `decibel` groups `dbm` + `dbfs` via `also_slugs`).

**Arc question:** *"When you turn a dial to 162.550 MHz and a voice comes out —
what actually happened?"* By the end the viewer can read a waterfall, follow a
signal from transmitter to receiver in dB, and knows the one hard limit
(Shannon) every radio lives under.

**Target length:** ~100 min (22 segments × ~4 min + connective tissue).

## Locked running order (dependency-sorted)

| # | seg_id | slug | Title | Treatment | Notes |
|---|---|---|---|---|---|
| — | — | — | Cold open (45–60 s) | teased clips | best moments from .13 / .21 |
| — | — | — | Course intro (2–3 min) | VO + map card | |
| **Act I — What a radio wave is** ||||||
| 1 | GT-RF-01.01 | electromagnetic-spectrum | Electromagnetic spectrum | concept | the map everything sits on |
| 2 | GT-RF-01.02 | radio-wave | Radio wave | term | |
| 3 | GT-RF-01.03 | frequency | Frequency | term | animate the article's 2-sine SVG |
| 4 | GT-RF-01.04 | wavelength | Wavelength | term | c = f·λ; antenna-size payoff |
| 5 | GT-RF-01.05 | amplitude | Amplitude | term | |
| 6 | GT-RF-01.06 | phase | Phase | term | clock-face analogy |
| 7 | GT-RF-01.07 | frequency-bands | Frequency bands (HF/VHF/UHF) | term | GT tie-in: band presets |
| **Act II — Carrying information & measuring power** ||||||
| 8 | GT-RF-01.08 | carrier-wave | Carrier wave | term | |
| 9 | GT-RF-01.09 | modulation | Modulation | **concept** | load-bearing chapter |
| 10 | GT-RF-01.10 | bandwidth | Bandwidth | term | |
| 11 | GT-RF-01.11 | decibel | Decibel (dB) | term | playbook §2.1 sample script; `also_slugs: [dbm, dbfs]` |
| 12 | GT-RF-01.12 | noise-floor | Noise floor | term | GT waterfall demo beat |
| 13 | GT-RF-01.13 | signal-to-noise-ratio | Signal-to-noise ratio (SNR) | **concept + demo** | ~50 % mark: strongest demo (GT SNR readout on live decode) |
| 14 | GT-RF-01.14 | attenuation | Attenuation | term | |
| 15 | GT-RF-01.15 | path-loss | Path loss | term | inverse-square picture |
| — | — | — | Midpoint recap (60–90 s) | map card | after .15 |
| **Act III — Real links, real limits** ||||||
| 16 | GT-RF-01.16 | impedance | Impedance | term | 50 Ω story |
| 17 | GT-RF-01.17 | resonance | Resonance | term | swing analogy; why antennas have a size |
| 18 | GT-RF-01.18 | harmonics | Harmonics | term | |
| 19 | GT-RF-01.19 | phase-noise | Phase noise | term | GT tie-in: the #764 Airspy story (carrier-clean, modulation-degraded) |
| 20 | GT-RF-01.20 | erp-eirp | ERP & EIRP | term | |
| 21 | GT-RF-01.21 | link-budget | Link budget | **concept + demo** | ~90 % mark: the payoff — whole chain in dB, GT receiving the result |
| 22 | GT-RF-01.22 | shannon-capacity | Shannon capacity | concept | finale; bridges to error-correction / next pillars |
| — | — | — | Outro + end screen (60–90 s + 20 s) | | next pillar: GT-RF-02 "SDR from Zero" |

Running-order rules honored: concepts early per act; demos at ~50 % (.13) and
~90 % (.21); treatment texture varied; no segment references another — all
connective tissue lives in the 21 transition beats (`scripts/GT-RF-01.00-pillar-elements.md`).

## Curation notes (16 entries deferred, Tier-C for this pillar)

`s-parameters`, `reflection-coefficient`, `return-loss`, `q-factor`,
`intermodulation`, `spurious-emissions`, `duty-cycle`, `crest-factor-papr`,
`group-delay`, `frequency-stability`, `field-strength`, `guard-band`,
`occupied-bandwidth`, `friis-transmission-equation` (folded into path-loss/link-budget
as one restated sentence each), `attenuation` siblings. They fit better in
GT-RF-04 (antennas) and GT-RF-05 (DSP) or as later single-segment drops.

## Assets shared across the pillar

- **Map card** (`graphics/map-card/`): the course diagram with per-node
  highlight states — used by intro, all 21 transition beats, midpoint recap.
- Animation library (`../pipeline/scenes.py`): sine/spectrum/dB-ladder/chain
  scenes reused across segments.
