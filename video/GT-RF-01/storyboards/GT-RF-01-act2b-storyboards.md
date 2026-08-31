# GT-RF-01 act 2b — storyboards (segments .13–.17)

One row per visual beat. "Vert" = how the visual survives 9:16: `safe` = built inside
the center 9:16 zone, crops clean; `relayout` = dedicated tall arrangement in the
vertical scene. Clips list the planned Tier-3 cuts (bounds are the `[CLIP]` marks in
the scripts; timing measured after VO).

## GT-RF-01.13 S-parameters
| Beat | Visual | Vert | Notes |
|---|---|---|---|
| title | title card | safe | |
| black-box | sealed box in a feedline; arriving wave splits into bounce-back + pass-through | safe | arrows kept near center |
| definition-card | Sij = out-of-i per unit into-j, "magnitude AND phase" card | safe | |
| window | sunlight on glass: glint back vs pass through; tint changes the split | safe | |
| two-port | article SVG animated: a1 in → S11 back, S21 out; drive swaps for S22/S12 | relayout | horizontal port-to-port flow → vertical stack in 9:16 |
| why-waves | probe fails to hold open/short at RF; 50 Ω terminations calm everything | safe | |
| derived-card | \|S11\|→return loss · \|S21\|→gain/loss · \|S12\|→isolation · .s2p card | safe | |
| vna | VNA sweep, S11/S21 traces drawing vs frequency | relayout | wide trace → tall stacked traces |
| gt-tie-in | antenna→filter→LNA→SDR→decoder chain, S11/S21 datasheet callouts | relayout | chain stacks vertically |
| recap | recap card | safe | |

Clips: c1 hook "where does the energy go" + complex-ratio definition (~50 s) · c2 four numbers of a two-port (~48 s) · c3 datasheet readings + .s2p caveat (~46 s).

## GT-RF-01.14 Resonance
| Beat | Visual | Vert | Notes |
|---|---|---|---|
| title | title card | safe | |
| bell-strike | struck bell rings; on-note taps build amplitude, off-note taps do nothing | safe | |
| definition-card | f₀ = 1/(2π√(LC)) card | safe | |
| energy-slosh | energy swaps L-field ↔ C-field, beside plucked string kinetic ↔ potential | relayout | side-by-side pair → stacked |
| tank-curve | article SVG animated: LC tank + XL/XC crossover erupting into the f₀ peak | relayout | tank + curve side-by-side → stacked |
| series-parallel | series LC impedance dips (passes f₀) vs parallel tank towers (blocks f₀) | relayout | two mini-circuits stack |
| q-curves | Q-factor article SVG: narrow high-Q vs broad low-Q peak, Δf bracket, Q = f₀/Δf | safe | the second core animation |
| q-ladder | component Q ladder: inductor tens → LC hundreds–low k → cavity tens of k → crystal hundreds of k | relayout | ladder runs vertical anyway |
| loaded-q | unloaded peak broadens when source + load couple in | safe | |
| gt-tie-in | antenna→preselector→high-Q reference→ADC→digital channelizer | relayout | chain stacks vertically |
| recap | recap card | safe | |

Clips: c1 bell hook + f₀ definition (~49 s) · c2 XL/XC cancellation mechanism (~38 s) · c3 Q definition, ringing, R-lever (~45 s).

## GT-RF-01.15 Harmonics
| Beat | Visual | Vert | Notes |
|---|---|---|---|
| title | title card | safe | |
| overtones | guitar string + overtone stack morphs into a transmitter spectrum | safe | spectrum lines centered |
| definition-card | fₙ = n·f₀ card | safe | |
| bent-curve | straight transfer line passes sine clean; bent line births 2f₀/3f₀ lines | relayout | curve + spectrum pair → stacked |
| spectrum | article SVG animated: f₀ tall, shrinking 2f₀/3f₀/4f₀; low-pass filter sweeps them away | safe | the core animation |
| spur-mask | spurious-emissions article SVG: wanted channel, scattered spurs, dashed regulatory limit | safe | |
| rx-side | FM tower's harmonic lands in monitored band; band-pass filter drops in, spur collapses | relayout | tower→band→dongle flow → vertical descent |
| gt-tie-in | waterfall: retune test (spur stays put) + "2× FM broadcast" annotation | relayout | waterfall crops tall natively |
| recap | recap card | safe | |

Clips: c1 overtones hook + integer-multiple definition (~49 s) · c2 filter defence + THD (~39 s) · c3 receive-side overload + band-pass cure (~38 s).

## GT-RF-01.16 Intermodulation distortion
| Beat | Visual | Vert | Notes |
|---|---|---|---|
| title | title card | safe | |
| phantom-carriers | two strong signals; ghost carriers materialize beside them | safe | |
| definition-card | 2f₁−f₂ / 2f₂−f₁ card | safe | |
| cheap-speaker | two pure notes through an overdriven speaker grow ghost tones | safe | |
| two-tone | article SVG animated: f₁/f₂ tall, IM3 flanks rising just outside the pair, passband highlighted | safe | the core animation |
| slope-chart | 1:1 wanted line vs 3:1 IM3 line, extrapolated crossing = IP3, P1dB below | relayout | axes re-proportioned tall |
| rusty-bolt | corroded tower joint mixes two transmitters, radiates on a third frequency | relayout | site scene → vertical tower |
| gt-tie-in | waterfall phantom control channel; attenuator clicks in, phantom vanishes | relayout | |
| recap | recap card | safe | |

Clips: c1 phantom-signals hook + third-order definition (~45 s) · c2 polynomial mixing mechanism (~48 s) · c3 3:1 slope, IP3, SFDR (~41 s).

## GT-RF-01.17 Occupied bandwidth
| Beat | Visual | Vert | Notes |
|---|---|---|---|
| title | title card | safe | |
| endless-skirts | waterfall zoom; edges never end; cursor hunts for "the edge" | safe | |
| definition-card | article SVG animated: PSD area shades, 0.5% slivers shave off, 99% bracket spans | safe | the core animation |
| parking-lot | cars in bays, mirrors overhang, painted gaps between bays | relayout | lot rotates to top-down vertical |
| guard-band | guard-band article SVG animated: two masks' skirts descend; vacant strip highlights; spacing = OBW + guard | safe | |
| designator-card | 16K0F3E decode card; 25 kHz channel splits into two 12.5 kHz | safe | |
| scales | guard bands at every scale: land-mobile, broadcast, cellular/Wi-Fi edges, OFDM edge subcarriers | relayout | scale examples stack |
| gt-tie-in | channelizer hugs a ~12.5 kHz P25 footprint; too-narrow clips, too-wide drags neighbour | relayout | filter + spectrum stack |
| recap | recap card | safe | |

Clips: c1 "how wide, really" hook + 99% definition (~50 s) · c2 guard band + spacing arithmetic (~42 s) · c3 emission designator + narrowbanding (~38 s).
