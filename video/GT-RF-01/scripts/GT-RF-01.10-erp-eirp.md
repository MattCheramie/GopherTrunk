# GT-RF-01.10 — ERP & EIRP
slug: erp-eirp
also_slugs: [field-strength]
type: term
target: ~3:50
note: covers ERP/EIRP plus field strength (what the radiated power becomes at a distant point); both article figures animated.

**[V: title | "ERP & EIRP" | GT-RF-01.10]**

**[V: two-towers — two transmitter sites: a big amplifier into a modest antenna, and a modest amplifier into a high-gain antenna; both stamp the same radiated-power figure]** [CLIP c1 start]
ERP and EIRP answer a deceptively simple question: how much power does this station *really* put out? Not the number on the amplifier — the power the system appears to radiate in its strongest direction, once you fold in the transmitter output, the feedline loss, and the antenna's gain. Two stations with completely different hardware can radiate identically: a big amplifier into a modest antenna, or a modest amplifier into a high-gain one. Effective radiated power is the single number that makes them comparable.

**[V: definition-card | "EIRP(dBm) = P_tx − L_feed + G(dBi) · ERP → dipole reference (dBd) · EIRP = ERP + 2.15 dB"]**
The two flavors differ only in their reference antenna. ERP — effective radiated power — is referenced to a half-wave dipole, with gain in d-B-d. EIRP — effective *isotropic* radiated power — is referenced to an ideal isotropic radiator, with gain in d-B-i. A lossless dipole already has two point one five decibels of gain over isotropic, so EIRP is always ERP plus two point one five. [CLIP c1 end]

**[V: chain — the article's SVG animated: TX +40 dBm → −2 dB cable → antenna +9 dBi → EIRP = +47 dBm; the sum writes itself out beneath]**
The arithmetic is pure decibel addition along the transmit chain. Say the transmitter delivers plus forty d-B-m. The cable eats two decibels. The antenna adds nine d-B-i of gain. Forty, minus two, plus nine: an EIRP of plus forty-seven d-B-m — the power an ideal isotropic source would need in order to match this system's peak radiated field.

**[V: directivity — an omnidirectional "light bulb" pattern squeezes into a flashlight beam; total energy stays constant; the off-axis field dims as the main lobe brightens]** [CLIP c2 start]
Here's the key idea hiding inside the word "gain": antenna gain is not amplification. A passive antenna radiates no more total power than it's fed. Gain describes *directivity* — the same energy concentrated into a narrower beam, a light bulb becoming a flashlight. Along the main lobe, the field is stronger than an omnidirectional reference would produce; away from it, weaker. That's why a high-EIRP directional link can be invisible off to the side. [CLIP c2 end]

**[V: field-strength — the field-strength article's SVG animated: the E-field sine with its "E (V/m)" amplitude arrow; the card S = E²/377 Ω appears; distance doubles and the amplitude halves]**
Out in space, what all that radiated power creates at your location is field strength — the amplitude of the wave's electric field at a point, in volts per meter, or for the weak fields of received signals, microvolts per meter. It's a property of the wave itself, independent of any receiving antenna. In the far field, field strength and power density are two views of the same thing, linked through the impedance of free space — three hundred seventy-seven ohms. And since power density falls as the inverse square of distance, field strength falls as one over distance: every doubling of range halves the volts per meter — minus six decibels.

**[V: regulator — a license card reading "50 kW ERP"; a Wi-Fi rule "36 dBm EIRP" with a slider trading amplifier power against antenna gain; a coverage-contour map drawn at a fixed µV/m level]**
This is the language regulators actually speak. They cap EIRP or ERP — not raw transmitter power — because what matters for interference and exposure is the field actually radiated. A Wi-Fi rule of thirty-six d-B-m EIRP lets you trade a bigger antenna against a smaller amplifier, as long as the product stays under the limit. A broadcast license of fifty kilowatts ERP fixes the effective coverage regardless of how it's produced. And coverage itself is drawn in field strength: broadcast contours are set at specific volts-per-meter levels, and RF-exposure limits are stated as maximum permitted fields.

**[V: gt-tie-in — GopherTrunk planning view: a distant trunking site with its published ERP; an arrow runs through path loss to an expected receive level at the SDR input]**
GopherTrunk is a receiver — it never radiates an ERP of its own — but these numbers govern every signal it hears. The EIRP of a trunking control channel, a broadcast tower, or a satellite downlink sets how strong that signal arrives after path loss, and published ERP figures for licensed land-mobile and broadcast transmitters are often on public record. Estimating a site's EIRP and distance lets you sanity-check expected receive levels — and a site whose numbers imply a weak arrival will need a better antenna or a low-noise amplifier on your end to decode reliably.

**[V: recap | "ERP & EIRP" | ① One number: P_tx − feedline + antenna gain | ② EIRP = ERP + 2.15 dB (isotropic vs dipole) | ③ Field strength: V/m at your location — −6 dB per doubling of range]**
So: ERP and EIRP fold a whole transmit chain into one radiated-power figure, the two differing only by the fixed two-point-one-five-decibel dipole offset — and field strength, in volts per meter, is what that power becomes by the time it reaches your antenna. Full write-ups on both are linked below.
