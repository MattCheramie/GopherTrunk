# GT-RF-01.16 — Intermodulation Distortion (IMD)
slug: intermodulation
type: term
target: ~3:50

**[V: title | "Intermodulation Distortion (IMD)" | GT-RF-01.16]**

**[V: phantom-carriers — a spectrum with two strong signals; ghostly new carriers materialize beside them where nothing is transmitting]** [CLIP c1 start]
Intermodulation distortion is where phantom signals come from. Put two strong signals through any device that isn't perfectly linear and it doesn't just pass them — it multiplies them, breeding brand-new frequencies that nobody ever transmitted. Unlike harmonics, which are integer multiples of a single tone, intermod products are sums and differences of *multiples* of two or more input tones. And the worst of them land right next to the originals, where no filter can touch them.

**[V: definition-card | "3rd-order products: 2f₁−f₂ and 2f₂−f₁ — right beside the originals"]**
The most troublesome are the third-order products, at two f-one minus f-two and two f-two minus f-one. Intermodulation is the fundamental reason a receiver can only handle a limited range of signal strengths at once. [CLIP c1 end]

**[V: cheap-speaker — two pure notes feed an overdriven speaker; the output spectrum grows extra tones neither instrument played]**
The everyday version is a cheap speaker pushed too loud: play two pure notes through it and you hear ghost tones that neither instrument played — the speaker's own distortion mixing the two together. RF stages do exactly that to radio signals.

**[V: two-tone — the article's figure animated: tall tones f₁ and f₂ close together; smaller spurs at 2f₁−f₂ and 2f₂−f₁ rise just outside the pair, inside the highlighted passband]** [CLIP c2 start]
Here's the mechanism. A perfectly linear device only scales and delays its input — no new frequencies, ever. Real amplifiers and mixers have a slightly curved transfer characteristic that expands as a polynomial, and feeding two tones into it lets each nonlinear term multiply them together, producing sums and differences of their harmonics. The order of a product is the sum of its multiplier coefficients. Second-order products — the plain sum and difference of the two tones — usually fall far from the originals and are easy to filter. The dangerous ones come from the cubic term: when the two tones are close, its third-order products sit just outside the pair, squarely inside the receiver's passband, masquerading as real signals. [CLIP c2 end]

**[V: slope-chart — input level versus output level: the wanted signal climbs at one-to-one, the third-order product at three-to-one; extending both lines to their crossing marks IP3]** [CLIP c3 start]
The cubic origin gives third-order products a defining behaviour: they grow three decibels for every one decibel the input tones rise. That three-to-one slope is why intermod explodes as signal levels climb — and it's the basis of the third-order intercept point, IP3: the extrapolated level where the third-order product would notionally equal the wanted signal. It's a single figure of merit for a stage's linearity, and it typically sits about ten to fifteen decibels above the one-decibel compression point. The usable window between the noise floor and the level where third-order products emerge has its own name: the spurious-free dynamic range. [CLIP c3 end]

**[V: rusty-bolt — an antenna site: a corroded tower joint glows as two transmitters' signals mix in it and radiate interference on a third frequency]**
Intermod isn't confined to your receiver. Nearby transmitters can mix in any shared nonlinearity — a corroded connector, a rusty tower joint, even a diode-like oxide layer: the "rusty bolt" effect. That passive intermodulation radiates and interferes on frequencies where nothing is actually transmitting, and it's a recurring, hard-to-trace source of interference at antenna sites. In your own station, a busy RF environment — pagers, broadcast FM, cellular — can drive the low-noise amplifier, mixer, and converter into producing third-order products that appear as phantom carriers across the tuned band, or that desensitize the receiver by raising its effective noise. Wideband direct-sampling SDRs are especially exposed, because they present the whole spectrum to the converter at once; front-end filtering and attenuation are the usual defences, and IP3 and the compression point are the datasheet numbers that predict how a given SDR will cope.

**[V: gt-tie-in — GopherTrunk waterfall with a phantom control channel; an attenuator clicks in and the phantom vanishes while real signals merely shrink]**
GopherTrunk decodes the IQ samples *after* that front end, so intermod that happened in the analog chain is baked into the samples — software cannot remove a spur that looks like a legitimate signal. The project's own DSP notes turn this into a diagnostic rule: a symptom that appears only at higher capture rates and still reproduces in offline replay points at the captured data — front-end overload or intermod — not at the decoder. And the classic field signature: phantom control channels or elevated noise that vanish the moment you add an attenuator. If attenuation makes a signal disappear, it was never real.

**[V: recap | "Intermodulation distortion" | ① Nonlinear mixing of ≥2 signals breeds new frequencies | ② 3rd-order lands beside the originals, growing 3 dB per 1 dB | ③ IP3 predicts it — and an attenuator exposes it]**
So: intermodulation breeds phantom signals right next to real ones, third-order products grow three times faster than the tones that cause them, and the attenuator test tells you which carriers are lies. The full write-up is linked below.
