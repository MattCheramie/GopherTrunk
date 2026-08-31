# GT-RF-01 act 3 — storyboards (segments .18–.22)

One row per visual beat, same conventions as the pilot storyboards: "Vert" = how the
visual survives 9:16 (`safe` = built inside the center 9:16 zone, crops clean;
`relayout` = dedicated tall arrangement in the vertical scene). Clips list the planned
Tier-3 cuts (bounds are the `[CLIP]` marks in the scripts; timing measured after VO).

## GT-RF-01.18 Duty Cycle
| Beat | Visual | Vert | Notes |
|---|---|---|---|
| title | title card | safe | |
| keyed-mic | PTT mic + timeline painting on/off + % counter | safe | timeline kept short, centered |
| definition-card | D = t_on/(t_on+t_off) · avg = D × peak card | safe | |
| sprinter | sprinter vs marathoner split + shared heat gauge | relayout | split-screen → stacked |
| pulse-trains | article SVG animated: 25% vs 75% trains + avg-power bars | safe | already a vertical stack of 2 traces; the core animation |
| datasheet | dummy-load spec card + duty slider | safe | |
| boundary-card | duty cycle vs crest factor boundary card | safe | |
| gt-tie-in | spectrogram: slotted TDMA vs continuous CC + callouts | relayout | side-by-side spectrograms → stacked |
| recap | recap card | safe | |

Clips: c1 hook + "average = D × peak" (~55 s) · c2 pulse-trains range tour (~50 s) · c3 spectrogram field mark (~40 s).

## GT-RF-01.19 Crest Factor & PAPR
| Beat | Visual | Vert | Notes |
|---|---|---|---|
| title | title card | safe | |
| calm-then-spike | idling envelope, one peak hits red clipping ceiling | safe | |
| stadium | crowd clap wash → aligned crack pins meter | safe | meter vertical already |
| envelope-figure | article SVG animated: envelope + P_avg/P_peak lines + PAPR bracket | safe | the core animation |
| backoff | PA transfer curve, point slides 8 dB down, efficiency gauge sags | relayout | curve + gauge stack |
| boundary-card | PAPR vs duty cycle boundary card | safe | |
| air-interfaces | 3 cards: LMR constant-envelope · LTE DL/UL · TETRA π/4-DQPSK | relayout | card row → column |
| gt-tie-in | ADC headroom margin + dBFS gain-staging meter | safe | |
| recap | recap card | safe | |

Clips: c1 hook "sized for the peak" (~40 s) · c2 where peaks come from / OFDM 10–13 dB (~45 s) · c3 backoff cost + countermeasures (~50 s).

## GT-RF-01.20 Group Delay
| Beat | Visual | Vert | Notes |
|---|---|---|---|
| title | title card | safe | |
| stealth-smear | crisp pulse → filter box → smeared pulse, amplitude unchanged | relayout | left-right flow → top-down |
| marching-band | corridor ×2: equal delay intact vs trombones-late smeared | relayout | corridor runs vertical |
| delay-curve | article SVG animated: flat ideal vs edge-rising real τ_g curve | safe | the core animation |
| numbers-card | 10 µs constant harmless vs 2 µs variation card + eye pinching shut | safe | |
| fixes | RRC pair handshake + adaptive equalizer + EVM readout dropping | relayout | 3 blocks stack |
| gt-tie-in | FIR chain stamped "linear phase"; upstream filter/feedline flagged | relayout | chain stacks vertically |
| recap | recap card | safe | |

Clips: c1 hook "ruin without touching strength" (~45 s) · c2 flat vs dispersive / FIR half-length (~55 s) · c3 the 10 µs vs 2 µs numbers + fixes (~55 s).

## GT-RF-01.21 Phase Noise
| Beat | Visual | Vert | Notes |
|---|---|---|---|
| title | title card | safe | |
| metronome | metronome tick → spectrum line blurs into skirts | safe | |
| singer | held note + wobble overlay vs sagging-flat overlay | safe | pitch line horizontal, fits center zone |
| skirts-figure | article SVG animated: ideal line vs skirts; −110 dBc/Hz bracket; neighbor smears in | safe | the core animation; neighbor enters from side — keep inside center zone |
| numbers-ladder | card stack: 20·log₁₀(N) · ±2 ppm@450 MHz→±900 Hz · crystal→TCXO→OCXO→GPSDO | safe | already a vertical stack |
| crowded-band | waterfall: weak CC beside blazing pager, floor rises | relayout | waterfall rotated tall |
| gt-tie-in | front end "noise born here" + ppm-correction / tracking-loop boxes | relayout | chain stacks vertically |
| recap | recap card | safe | |

Clips: c1 metronome hook (~35 s) · c2 dBc/Hz + reciprocal mixing (~55 s) · c3 buried-by-the-pager fix (~40 s).

## GT-RF-01.22 Shannon capacity
| Beat | Visual | Vert | Notes |
|---|---|---|---|
| title | title card | safe | |
| speed-limit | road-sign formula stamp + rate meter hitting hard ceiling | safe | |
| noisy-room | party scene + "talk faster"/"talk louder" dials | relayout | dials move below scene |
| capacity-curve | article SVG animated: C-vs-SNR curve draws, formula rides it, C/B label peels | safe | the core animation |
| two-regimes | curve annotated: +3 dB steps flattening · low-SNR spread · −1.6 dB floor | safe | annotations layered on same curve |
| closing-gap | 1948→today timeline, codes climb toward fixed Shannon line | relayout | timeline runs vertical |
| gt-tie-in | waveform point on curve; SNR marker slides below/above; antenna/LNA push it up | safe | |
| recap | recap card (finale: closing line sweeps course arc) | safe | |

Clips: c1 "speed limit of the universe" hook (~40 s) · c2 two regimes + the −1.6 dB floor (~45 s) · c3 "moves you back inside the curve" GT finale (~40 s).

## Pillar elements
Act-3 outro: the course-map card lights all 22 nodes, then holds on .22's curve beat for
the finale line — no new graphics beyond the map comp already built for the pilot.
