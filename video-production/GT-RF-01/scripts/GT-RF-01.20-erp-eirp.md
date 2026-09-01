# GT-RF-01.20 — ERP & EIRP
slug: erp-eirp · type: term · treatment: 2.1 term · target: 3:30–4:30 · words: 598

**[V: Title card — "ERP & EIRP"]**

**[V: a transmitter block labeled with one power; question mark over the antenna: "what actually leaves?"]** [CLIP c1 starts]
ERP and EIRP answer a deceptively simple question: how much power is that transmitter *really* putting out? Not what the amplifier makes — what actually leaves the antenna, in its strongest direction. Both are effective radiated power measures that fold the transmitter's output, the feedline's losses, and the antenna's gain into a single number. The only difference between them is the ruler they're measured against. [CLIP c1 ends]

**[V: a bare bulb glowing in all directions; the same bulb dropped into a flashlight reflector, throwing a bright beam one way]** [CLIP c2 starts]
Start with a flashlight. The bulb makes the same light either way, but the reflector squeezes it into a beam — and along that beam, it looks like a far brighter bulb. Antenna gain works exactly like the reflector. A passive antenna radiates no more total power than it's fed; gain is *directivity* — energy concentrated into a narrower beam. Along the main lobe the field is stronger than an omnidirectional antenna would produce, and off to the side it's weaker. That's why a high-power directional link can be nearly invisible from the wrong angle. [CLIP c2 ends]

**[V: article's chain figure animates: TX +40 dBm → −2 dB cable → +9 dBi antenna → "EIRP = +47 dBm"]**
The bookkeeping is a short chain of decibels. Take the transmitter's output — say plus forty dBm, ten watts. Subtract the feedline and connector losses — two dB lost up the tower. Add the antenna's gain — nine dBi. Forty minus two plus nine: forty-seven dBm of EIRP. That's the power an ideal antenna radiating equally in every direction would need to match this system's peak — even though the amplifier only ever made forty.

**[V: card: "EIRP → isotropic reference (dBi) · ERP → half-wave dipole (dBd) · EIRP = ERP + 2.15 dB"]**
So what separates the two names? Just the reference antenna. EIRP compares against an ideal *isotropic* radiator — a mathematical point source, equal in all directions. ERP compares against a half-wave *dipole* — a real, buildable antenna. A lossless dipole already has 2.15 dB of gain over isotropic, so the conversion is fixed forever: EIRP equals ERP plus 2.15 dB. Broadcasters and land-mobile regulators traditionally quote ERP; satellite, microwave, and radar engineers quote EIRP, because the isotropic reference is cleaner math.

**[V: two rule cards, shown in sequence: "Wi-Fi rule: 36 dBm EIRP" · "Broadcast license: 50 kW ERP"]** [CLIP c3 starts]
And here's why regulators cap these numbers instead of raw transmitter power: interference and exposure depend on what's actually radiated. A Wi-Fi rule of thirty-six dBm EIRP lets you trade a bigger antenna against a smaller amplifier, so long as the product stays under the limit. A broadcast license of fifty kilowatts ERP fixes the station's effective reach no matter how that power splits between the amplifier and the antenna. The cap is on the field, not the box. [CLIP c3 ends]

**[V: a distant tower with an EIRP label; a thin ray crossing terrain to a rooftop receive antenna]**
As a listener, you'll never radiate an ERP of your own — but every signal you hear was launched with one. A trunking control channel, a broadcast tower, an aircraft transponder: each one's EIRP sets how strong it arrives at your antenna after the path takes its toll, and hence whether it clears your noise floor at all. Better still, for licensed land-mobile and broadcast transmitters the ERP is often on public record — so you can look up how loudly a site shouts before you ever tune to it.

**[V: GopherTrunk web UI, dark theme: dBFS signal meter and SNR readout on a live control channel]**
That makes ERP a planning tool for GopherTrunk. If a site's published power and its distance imply a weak arriving signal, the dBFS level and SNR readout in GopherTrunk's interface will confirm it — and the fix lives on your side of the link: a better receive antenna, a low-noise amplifier, some gain of your own. The decoder can only work with what survives the trip.

**[V: Recap card: "① ERP/EIRP = TX power − feedline loss + antenna gain, one number ② EIRP = ERP + 2.15 dB (dipole vs isotropic) ③ gain is a reflector, not an amplifier"]**
So: ERP and EIRP fold transmitter power, cable loss, and antenna gain into one radiated-power figure; they differ only by reference antenna — dipole versus isotropic, exactly 2.15 dB apart; and antenna gain is a reflector, not an amplifier — concentration, never creation. Full write-up linked below.

## Clip picks
- **c1** (~25 s): the "what actually leaves the antenna?" hook + one-number definition.
- **c2** (~30 s): the flashlight-reflector beat — "gain is not amplification" is the shareable contrarian idea. Shorts title: "Antenna gain doesn't add power".
- **c3** (~25 s): "regulators cap the field, not the box" — the Wi-Fi / broadcast trade-off beat.

## Vertical plan
Direct center crop. The TX→cable→antenna chain is the one wide visual — vertical re-layout runs the chain top-to-bottom (TX at top, EIRP figure at bottom), which matches a real tower anyway. Flashlight, rule cards, and recap all center-safe. Re-hook: burned text "How much power does that tower REALLY make?" over the question-mark antenna.

## Assets
- Title + recap cards (templates)
- **Core animation:** rebuilt from the article's inline SVG — the TX +40 dBm → −2 dB cable → +9 dBi → 47 dBm EIRP chain
- Flashlight/bare-bulb directivity animation (reusable for the antenna-gain entry later)
- Cards: dipole-vs-isotropic reference card (with the fixed 2.15 dB) · Wi-Fi 36 dBm EIRP · 50 kW ERP license (one at a time)
- Distant-tower-to-receiver ray graphic
- GopherTrunk screen capture: dBFS meter + SNR readout on a control channel (10–20 s, dark theme)
- VO track; calm/technical music bed

## Checklist deltas
- [ ] Analogy check: the flashlight maps directivity exactly (same total light, brighter beam) — never say or imply the reflector "adds" light
- [ ] Formula shown matches article: EIRP(dBm) = P_tx − losses + G_ant(dBi); worked numbers are the article's own (40 − 2 + 9 = 47; ERP = 44.85 dBm if shown)
- [ ] 2.15 dB dipole-over-isotropic figure and EIRP = ERP + 2.15 dB stated exactly as in the article
- [ ] Regulatory examples verbatim from the article (36 dBm EIRP Wi-Fi, 50 kW ERP broadcast) — no invented limits
- [ ] GT tie-in stays receive-side: GopherTrunk has no ERP/EIRP of its own; framing is planning/diagnosis only
