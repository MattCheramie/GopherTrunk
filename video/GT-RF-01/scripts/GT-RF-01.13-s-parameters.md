# GT-RF-01.13 — S-parameters (Scattering Parameters)
slug: s-parameters
type: term
target: ~3:50

**[V: title | "S-parameters (Scattering Parameters)" | GT-RF-01.13]**

**[V: black-box — a sealed box spliced into an antenna feedline; a wave arrives at it and splits: one arrow bounces back, one passes through resized]** [CLIP c1 start]
S-parameters answer the one question that matters about every box you ever put in an antenna line — a filter, an amplifier, a length of cable: when a wave arrives at it, where does the energy go? Some of it reflects straight back the way it came. Some of it passes through, bigger or smaller. Scattering parameters describe a linear RF network entirely by how it scatters incoming waves into reflected and transmitted ones — nothing about what's inside the box, only what it does to waves at its ports.

**[V: definition-card | "Sij = wave out of port i per unit wave into port j — complex: magnitude AND phase"]**
Each S-parameter is a complex number — a magnitude and a phase — giving the wave that emerges at one port per unit of wave injected at another, with every other port terminated in the reference impedance, usually fifty ohms. [CLIP c1 end]

**[V: window — sunlight strikes a pane of glass; part glints back at the viewer, part passes into the room; the glass tints and the split changes]**
The everyday picture is a window. Shine light at a pane of glass and part of it glints back at you — reflection — while the rest passes through into the room — transmission. Tint the glass and less gets through. An RF two-port is that pane of glass for radio waves, and S-parameters are the numbers printed on the tint.

**[V: two-port — the article's figure animated: wave a1 enters port 1; the S11·a1 arrow bounces back; the S21·a1 arrow exits port 2; then the drive swaps sides to reveal S22 and S12]** [CLIP c2 start]
For the ubiquitous two-port device — one connector in, one out — four numbers capture everything. Drive port one, and terminate port two so nothing comes back from that side. The fraction of the wave that reflects at the input is S-one-one: the input match. The fraction that makes it through to port two is S-two-one: the forward transmission — the gain of an amplifier, or the loss of a filter or cable. Now swap the drive to the other side: S-two-two is the output match, and S-one-two is the reverse transmission — how much leaks backward through the device. In matrix form the whole story is one line: the outgoing waves equal the S-matrix times the incoming ones. [CLIP c2 end]

**[V: why-waves — a probe tries to hold an open and a short at RF; the test leads' reactances squirm and dominate; the S-approach calmly terminates every port in 50 Ω]**
Why waves, and not plain voltage and current? Because at radio frequencies you can't cleanly hold a port open or shorted the way classic impedance or admittance measurements demand — the reactances of the test leads take over, and open and short standards misbehave. Working with travelling waves into a well-defined fifty-ohm termination sidesteps the whole mess.

**[V: derived-card | "|S11| → return loss · |S21| → gain / insertion loss · |S12| → isolation · shipped as .s2p Touchstone"]** [CLIP c3 start]
In a datasheet these arrive pre-digested. The magnitude of S-one-one in decibels is the return loss — how well the port is matched. The magnitude of S-two-one in decibels is the gain of an amplifier, or the insertion loss of a passive part. A large negative S-one-two means the device is well unilateral — almost nothing leaks backward. And because every parameter carries phase as well as magnitude, the same data lets you cascade stages, design matching networks, and build filters. It ships as a Touchstone dot-s-two-p file — magnitude and phase across frequency — that circuit simulators read directly. One caveat: S-parameters describe linear, small-signal behaviour; compression and intermodulation need their own measurements. [CLIP c3 end]

**[V: vna — a vector network analyzer sweeps; S11 and S21 traces draw across the screen versus frequency]**
The instrument behind all of this is the vector network analyzer: it sweeps a known incident wave across frequency and ratios the reflected and transmitted waves against it to recover each parameter, magnitude and phase.

**[V: gt-tie-in — GopherTrunk receive chain: antenna → filter → LNA → SDR → decoder; datasheet callouts "S11 / S21" pinned to the filter and LNA]**
GopherTrunk lives after the analog chain — software neither produces nor consumes S-parameters. But every part you choose in front of the SDR speaks this language: a low-noise amplifier is quoted by its S-two-one gain and its S-one-one input match, a filter by its S-two-one passband shape and stopband rejection, cables and connectors by their return and insertion loss. A good S-one-one and S-two-one profile at your frequency of interest is what ultimately hands the decoder clean samples.

**[V: recap | "S-parameters" | ① Sij = how a network scatters waves: reflected + transmitted | ② Two-port: S11 in-match · S21 gain · S12 isolation · S22 out-match | ③ Complex data, measured on a VNA, shipped as .s2p]**
So: S-parameters describe any RF box by what it reflects and what it transmits, four of them fully specify a two-port, and they're the language every front-end datasheet is written in. The full write-up is linked below.
