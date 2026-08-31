# GT-RF-01 act 2a — storyboards (segments .07–.12)

One row per visual beat. "Vert" = how the visual survives 9:16: `safe` = built inside
the center 9:16 zone, crops clean; `relayout` = dedicated tall arrangement in the
vertical scene. Clips list the planned Tier-3 cuts (bounds are the `[CLIP]` marks in
the scripts; timing measured after VO).

## GT-RF-01.07 Amplitude
| Beat | Visual | Vert | Notes |
|---|---|---|---|
| title | title card | safe | |
| two-signals | article SVG animated: tall + short sines, meter climbs | safe | already stacked on one centre line |
| definition-card | amplitude/phase double definition card | safe | |
| swing | playground swing, amplitude bracket + phase marker | safe | second swing out of step |
| power-square | amplitude ×2 → power ×4, dB scale flip | safe | |
| phase-offset | article SVG: sliding sines + phase bracket → IQ plane, rotating point | relayout | sines above, IQ plane below in 9:16 |
| psk | BPSK → QPSK constellation with bit labels | safe | |
| hazards | rotating constellation, phase-noise clouds, fading bars | relayout | three panels stack |
| gt-tie-in | GopherTrunk IQ view: magnitude → AGC/activity, angle → carrier recovery | relayout | split view stacks |
| recap | recap card | safe | |

Clips: c1 hook "height of the wave" (~40 s) · c2 IQ polar picture (~30 s) · c3 PSK bits-in-the-phase (~30 s).

## GT-RF-01.08 Attenuation
| Beat | Visual | Vert | Notes |
|---|---|---|---|
| title | title card | safe | |
| fading-wave | article SVG animated: shrinking sine TX→RX + dB stamps | safe | |
| definition-card | spreading + absorption / path-loss card | safe | |
| toll-booths | road with toll booths: coax, connectors, walls, rain | relayout | road runs vertical in 9:16 |
| sphere | expanding sphere → path-loss article SVG distance curve | safe | curve keeps axis labels inside centre zone |
| layers | distance curve + shadowing + multipath flicker + fade margin band | safe | |
| levers | few-dB lever vs geometry lever; mast rises, Fresnel clears | relayout | levers stack |
| gt-tie-in | dashboard channel near noise floor + troubleshooting checklist | relayout | |
| recap | recap card | safe | |

Clips: c1 hook "the toll every signal pays" (~40 s) · c2 inverse-square + path-loss exponent (~35 s).

## GT-RF-01.09 Link budget
| Beat | Visual | Vert | Notes |
|---|---|---|---|
| title | title card | safe | |
| ledger | bank-ledger deposits/withdrawals, running dBm balance | relayout | ledger column is natural 9:16 |
| definition-card | P_rx = EIRP − losses + G_rx / margin card | safe | |
| staircase | article SVG animated: steps up/down landing at P_rx, margin bracket | relayout | staircase re-plotted taller |
| friis | article SVG animated: TX→RX over distance d, formula assembles | safe | |
| ideal-vs-real | Friis ceiling line + stacked correction terms | safe | |
| two-directions | forward/backward budget runs, one spreadsheet row | safe | |
| gt-tie-in | receive-side budget card with "dB short" readout | relayout | chain flows downward |
| recap | recap card | safe | |

Clips: c1 hook "radio's bank statement" (~45 s) · c2 Friis consequences (~35 s) · c3 receive-side diagnosis (~35 s).

## GT-RF-01.10 ERP & EIRP
| Beat | Visual | Vert | Notes |
|---|---|---|---|
| title | title card | safe | |
| two-towers | two different TX chains, same radiated-power stamp | relayout | towers stack |
| definition-card | EIRP formula + dipole/isotropic reference + 2.15 dB card | safe | |
| chain | article SVG animated: 40 − 2 + 9 = 47 dBm sum | relayout | chain flows downward in 9:16 |
| directivity | light bulb pattern squeezes into flashlight beam | safe | |
| field-strength | field-strength article SVG: E (V/m) arrow + S = E²/377 Ω card | safe | |
| regulator | license "50 kW ERP", Wi-Fi "36 dBm EIRP" slider, µV/m contour map | relayout | three cards stack |
| gt-tie-in | published site ERP → path loss → expected receive level | relayout | |
| recap | recap card | safe | |

Clips: c1 hook "how much power does it really put out" (~40 s) · c2 gain = directivity, not amplification (~30 s).

## GT-RF-01.11 Impedance (Z)
| Beat | Visual | Vert | Notes |
|---|---|---|---|
| title | title card | safe | |
| fifty-ohm | gear parade, every port stamped 50 Ω | relayout | parade stacks |
| definition-card | article SVG: source/line/load + Z = R + jX phasor | relayout | circuit above, phasor below |
| spring-friction | pipe friction (R) + springy membrane (X) + frequency knob | safe | |
| line | coax cross-section, Z0 = √(L/C), "not an ohmmeter reading" stamp | safe | |
| matching | Zs/Z0/ZL row; mismatch → reflected wave + standing ripple | safe | |
| fifty-compromise | 30 Ω ↔ 77 Ω dial settling at 50; 75 Ω side note | safe | |
| gt-tie-in | SDR front-end chain, mismatch reflects weak signal pre-ADC | relayout | chain stacks |
| recap | recap card | safe | |

Clips: c1 hook "the number on everything: 50 Ω" (~40 s) · c2 characteristic impedance (~30 s).

## GT-RF-01.12 Reflection Coefficient (Γ)
| Beat | Visual | Vert | Notes |
|---|---|---|---|
| title | title card | safe | |
| bounce | article SVG animated: incident wave in, Γ·incident peels back | safe | |
| definition-card | Γ = (ZL − Z0)/(ZL + Z0) card, 0→1 range | safe | |
| rope | rope pulse: spliced / wall / free end | relayout | three rope panels stack |
| three-cases | matched / open / short cards + \|Γ\|² power stamp | safe | |
| rl-ladder | RL = −20·log₁₀\|Γ\| conversion + 0/10/20 dB ladder | safe | |
| sweep | return-loss article SVG animated: notch vs frequency + 10 dB line | safe | |
| gt-tie-in | antenna→feedline→SDR, reflected slice + "check return loss first" card | relayout | |
| recap | recap card | safe | |

Clips: c1 hook "the signal that bounces back" (~40 s) · c2 matched/open/short + |Γ|² (~40 s) · c3 return-loss ladder (~35 s).
